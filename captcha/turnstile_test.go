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
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"challenge_ts":"2026-01-01T00:00:00Z","hostname":"example.com","action":"signup"}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	res, err := v.Verify(context.Background(), "tok", "1.2.3.4", "signup")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestTurnstile_VerifyMissingTokenRejects(t *testing.T) {
	called := false
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		_, _ = io.WriteString(w, `{"success":true}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	res, err := v.Verify(context.Background(), "", "", "")
	if !errors.Is(err, ErrMissingToken) {
		t.Fatalf("expected ErrMissingToken, got %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result on failure, got %+v", res)
	}
	if called {
		t.Fatal("expected no HTTP call for empty token")
	}
}

func TestTurnstile_VerifyInvalidTokenRejects(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["invalid-input-response"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	res, err := v.Verify(context.Background(), "tok", "", "")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
	if res != nil {
		t.Fatalf("expected nil result on failure, got %+v", res)
	}
}

func TestTurnstile_VerifyDuplicateTokenRejects(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["timeout-or-duplicate"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	if _, err := v.Verify(context.Background(), "tok", "", ""); !errors.Is(err, ErrDuplicateToken) {
		t.Fatalf("expected ErrDuplicateToken, got %v", err)
	}
}

func TestTurnstile_VerifyOtherErrorPreservesCodes(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["internal-error","custom-code"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("secret", nil)
	v.endpoint = srv.URL

	_, err := v.Verify(context.Background(), "tok", "", "")
	var ve *VerifyError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VerifyError, got %T %v", err, err)
	}
	if len(ve.Codes) != 2 || ve.Codes[0] != "internal-error" || ve.Codes[1] != "custom-code" {
		t.Fatalf("expected codes preserved, got %v", ve.Codes)
	}
}

func TestTurnstile_VerifyMissingInputSecretIsVerifyError(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["missing-input-secret"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("", nil)
	v.endpoint = srv.URL

	_, err := v.Verify(context.Background(), "tok", "", "")
	var ve *VerifyError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VerifyError, got %T %v", err, err)
	}
}

func TestTurnstile_VerifyBadRequestIsVerifyError(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["bad-request"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	_, err := v.Verify(context.Background(), "tok", "", "")
	var ve *VerifyError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VerifyError, got %T %v", err, err)
	}
}

func TestTurnstile_VerifyMissingInputResponseIsMissingToken(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["missing-input-response"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	if _, err := v.Verify(context.Background(), "tok", "", ""); !errors.Is(err, ErrMissingToken) {
		t.Fatalf("expected ErrMissingToken, got %v", err)
	}
}

func TestTurnstile_VerifyHTTPErrorIsTransient(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	if _, err := v.Verify(context.Background(), "tok", "", ""); !errors.Is(err, ErrTransientFailure) {
		t.Fatalf("expected ErrTransientFailure, got %v", err)
	}
}

func TestTurnstile_VerifyMalformedJSONIsTransient(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `not-json`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	if _, err := v.Verify(context.Background(), "tok", "", ""); !errors.Is(err, ErrTransientFailure) {
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

	if _, err := v.Verify(context.Background(), "the-token", "9.9.9.9", ""); err != nil {
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
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second)
		_, _ = io.WriteString(w, `{"success":true}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := v.Verify(ctx, "tok", "", "")
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

// TestTurnstile_VerifyDoesNotSendActionFormField asserts that the verifier
// never includes an "action" form field, even when the caller supplies one.
// Cloudflare's siteverify does not accept action as a request parameter — it
// is bound at widget render time and echoed back in the response.
func TestTurnstile_VerifyDoesNotSendActionFormField(t *testing.T) {
	var hadActionField bool
	var rawBody string
	srv := newMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		rawBody = string(b)
		_ = r.ParseForm()
		// PostForm requires re-parsing since we drained the body; reparse by hand.
		// Easier: just check the captured raw body string.
		_, hadActionField = parseFormHas(rawBody, "action")
		_, _ = io.WriteString(w, `{"success":true,"action":"signup"}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	if _, err := v.Verify(context.Background(), "tok", "", "signup"); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if hadActionField {
		t.Fatalf("expected no 'action' form field in request body, got: %s", rawBody)
	}
}

// parseFormHas reports whether the urlencoded body contains the given key.
func parseFormHas(body, key string) (string, bool) {
	for _, kv := range strings.Split(body, "&") {
		if kv == "" {
			continue
		}
		eq := strings.IndexByte(kv, '=')
		var k string
		if eq < 0 {
			k = kv
		} else {
			k = kv[:eq]
		}
		if k == key {
			if eq < 0 {
				return "", true
			}
			return kv[eq+1:], true
		}
	}
	return "", false
}

// TestTurnstile_VerifyActionMismatchRejects asserts that when the response's
// action does not match the caller-supplied action, Verify returns
// *VerifyError with codes=[action-mismatch].
func TestTurnstile_VerifyActionMismatchRejects(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":true,"action":"login"}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	res, err := v.Verify(context.Background(), "tok", "", "signup")
	if res != nil {
		t.Fatalf("expected nil result on mismatch, got %+v", res)
	}
	var ve *VerifyError
	if !errors.As(err, &ve) {
		t.Fatalf("expected *VerifyError, got %T %v", err, err)
	}
	if len(ve.Codes) != 1 || ve.Codes[0] != "action-mismatch" {
		t.Fatalf("expected codes=[action-mismatch], got %v", ve.Codes)
	}
}

// TestTurnstile_VerifyActionMatchSucceeds asserts that when the response's
// action matches the caller-supplied action, Verify returns a non-nil Result
// and a nil error.
func TestTurnstile_VerifyActionMatchSucceeds(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":true,"action":"signup","hostname":"example.com","challenge_ts":"2026-01-01T00:00:00Z"}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	res, err := v.Verify(context.Background(), "tok", "", "signup")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Action != "signup" {
		t.Errorf("action: got %q", res.Action)
	}
}

// TestTurnstile_VerifyEmptyActionAccepts asserts that when the caller passes
// an empty action, the verifier does not enforce any action check (regardless
// of what the response contains).
func TestTurnstile_VerifyEmptyActionAccepts(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":true,"action":"login"}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	if _, err := v.Verify(context.Background(), "tok", "", ""); err != nil {
		t.Fatalf("expected nil with empty action arg, got %v", err)
	}
}

// TestTurnstile_VerifyReturnsResultOnSuccess asserts that on success the
// returned *Result has the parsed challenge_ts, hostname, and action fields
// populated from the provider response.
func TestTurnstile_VerifyReturnsResultOnSuccess(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":true,"challenge_ts":"2026-01-01T00:00:00Z","hostname":"example.com","action":"signup","cdata":"opaque-data"}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	res, err := v.Verify(context.Background(), "tok", "", "signup")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if !res.Success {
		t.Error("expected Success=true")
	}
	if res.ChallengeTS != "2026-01-01T00:00:00Z" {
		t.Errorf("challenge_ts: got %q", res.ChallengeTS)
	}
	if res.Hostname != "example.com" {
		t.Errorf("hostname: got %q", res.Hostname)
	}
	if res.Action != "signup" {
		t.Errorf("action: got %q", res.Action)
	}
}

// TestTurnstile_VerifyReturnsResultOnFailure asserts that on failure the
// returned Result is nil; codes are inspected via *VerifyError.
func TestTurnstile_VerifyReturnsResultOnFailure(t *testing.T) {
	srv := newMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"success":false,"error-codes":["invalid-input-response"]}`)
	})
	defer srv.Close()

	v := NewTurnstileVerifier("s", nil)
	v.endpoint = srv.URL

	res, err := v.Verify(context.Background(), "tok", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if res != nil {
		t.Fatalf("expected nil result on failure, got %+v", res)
	}
}

// Compile-time assertion that *TurnstileVerifier satisfies Verifier.
var _ Verifier = (*TurnstileVerifier)(nil)
