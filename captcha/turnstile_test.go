package captcha

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newMockServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func TestTurnstile_VerifyValidTokenSucceeds(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"challenge_ts":"2026-01-01T00:00:00Z","hostname":"example.com","action":"signup"}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	if err := v.Verify(context.Background(), "tok", "1.2.3.4", "signup"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestTurnstile_VerifyMissingTokenRejects(t *testing.T) {
	called := false
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = io.WriteString(w, `{"success":true}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	err := v.Verify(context.Background(), "", "", "")
	if !errors.Is(err, ErrMissingToken) {
		t.Fatalf("expected ErrMissingToken, got %v", err)
	}
	if called {
		t.Fatal("expected no HTTP call for empty token")
	}
}

func TestTurnstile_VerifyInvalidTokenRejects(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["invalid-input-response"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	err := v.Verify(context.Background(), "tok", "", "")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestTurnstile_VerifyDuplicateTokenRejects(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["timeout-or-duplicate"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	err := v.Verify(context.Background(), "tok", "", "")
	if !errors.Is(err, ErrDuplicateToken) {
		t.Fatalf("expected ErrDuplicateToken, got %v", err)
	}
}

func TestTurnstile_VerifyOtherErrorPreservesCodes(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["internal-error","custom-code"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	err := v.Verify(context.Background(), "tok", "", "")
	var ve *VerifyError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VerifyError, got %T %v", err, err)
	}
	if len(ve.Codes) != 2 || ve.Codes[0] != "internal-error" || ve.Codes[1] != "custom-code" {
		t.Fatalf("expected codes preserved, got %v", ve.Codes)
	}
}

func TestTurnstile_VerifyMissingInputSecretIsVerifyError(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["missing-input-secret"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("", nil)
	v.endpoint = srv.URL

	err := v.Verify(context.Background(), "tok", "", "")
	var ve *VerifyError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VerifyError, got %T %v", err, err)
	}
}

func TestTurnstile_VerifyBadRequestIsVerifyError(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["bad-request"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	err := v.Verify(context.Background(), "tok", "", "")
	var ve *VerifyError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VerifyError, got %T %v", err, err)
	}
}

func TestTurnstile_VerifyMissingInputResponseIsMissingToken(t *testing.T) {
	// If the server tells us the response was missing, surface as ErrMissingToken.
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["missing-input-response"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	err := v.Verify(context.Background(), "tok", "", "")
	if !errors.Is(err, ErrMissingToken) {
		t.Fatalf("expected ErrMissingToken, got %v", err)
	}
}

func TestTurnstile_VerifyHTTPErrorIsTransient(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	err := v.Verify(context.Background(), "tok", "", "")
	if !errors.Is(err, ErrTransientFailure) {
		t.Fatalf("expected ErrTransientFailure, got %v", err)
	}
}

func TestTurnstile_VerifyMalformedJSONIsTransient(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `not-json`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	err := v.Verify(context.Background(), "tok", "", "")
	if !errors.Is(err, ErrTransientFailure) {
		t.Fatalf("expected ErrTransientFailure, got %v", err)
	}
}

func TestTurnstile_VerifySendsExpectedFormFields(t *testing.T) {
	var (
		gotSecret   string
		gotResponse string
		gotRemoteIP string
		gotCT       string
	)
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		_ = r.ParseForm()
		gotSecret = r.PostForm.Get("secret")
		gotResponse = r.PostForm.Get("response")
		gotRemoteIP = r.PostForm.Get("remoteip")
		_, _ = io.WriteString(w, `{"success":true}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("the-secret", nil)
	v.endpoint = srv.URL

	if err := v.Verify(context.Background(), "the-token", "9.9.9.9", ""); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if gotSecret != "the-secret" {
		t.Errorf("secret: got %q", gotSecret)
	}
	if gotResponse != "the-token" {
		t.Errorf("response: got %q", gotResponse)
	}
	if gotRemoteIP != "9.9.9.9" {
		t.Errorf("remoteip: got %q", gotRemoteIP)
	}
	if !strings.HasPrefix(gotCT, "application/x-www-form-urlencoded") {
		t.Errorf("content-type: got %q", gotCT)
	}
}

func TestTurnstile_VerifyContextCancellationPropagates(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		_, _ = io.WriteString(w, `{"success":true}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := v.Verify(ctx, "tok", "", "")
	if err == nil {
		t.Fatal("expected error from canceled ctx")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected ctx.Canceled wrapped, got %v", err)
	}
}

func TestTurnstile_VerifyDefaultsTimeoutTo5s(t *testing.T) {
	v := NewTurnstileVerifier("s", nil)
	if v.httpClient == nil {
		t.Fatal("expected default http client")
	}
	if v.httpClient.Timeout != 5*time.Second {
		t.Fatalf("expected 5s timeout, got %v", v.httpClient.Timeout)
	}
}

func TestTurnstile_VerifyUsesCloudflareEndpointByDefault(t *testing.T) {
	v := NewTurnstileVerifier("s", nil)
	if v.endpoint != turnstileEndpoint {
		t.Fatalf("expected default cloudflare endpoint, got %q", v.endpoint)
	}
}

// Compile-time assertion that *TurnstileVerifier satisfies Verifier.
var _ Verifier = (*TurnstileVerifier)(nil)
