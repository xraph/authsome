package authclient_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authclient "github.com/xraph/authsome/sdk/go"
)

// These tests pin the contract the twinos workspace provisioner
// (and other downstream services) depends on: ClientError must
// extract a useful Message from any of the common forge.HTTPError
// envelope shapes, and must preserve the raw response body so
// callers can log it when nothing decodes.

func runClientDoAgainst(t *testing.T, srv *httptest.Server) error {
	t.Helper()
	c := authclient.NewClient(srv.URL)
	// CreateOrganization is one of many "do"-driven calls; using
	// it exercises the same internal pipeline as every other
	// generated method.
	_, err := c.CreateOrganization(context.Background(), &authclient.CreateOrganizationRequest{
		AppID: "aapp_test",
		Name:  "x",
		Slug:  "x",
	})
	return err
}

func TestClientError_extractsErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"name required"}`))
	}))
	defer srv.Close()

	err := runClientDoAgainst(t, srv)
	var ce *authclient.ClientError
	if !errors.As(err, &ce) {
		t.Fatalf("err is not *ClientError: %T", err)
	}
	if ce.Message != "name required" {
		t.Errorf("Message = %q, want 'name required'", ce.Message)
	}
}

func TestClientError_extractsMessageField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"alt format"}`))
	}))
	defer srv.Close()

	var ce *authclient.ClientError
	if !errors.As(runClientDoAgainst(t, srv), &ce) {
		t.Fatal("ClientError expected")
	}
	if ce.Message != "alt format" {
		t.Errorf("Message = %q, want 'alt format'", ce.Message)
	}
}

func TestClientError_extractsDetailsField(t *testing.T) {
	// forge.HTTPError uses {code, error, details}. When error is
	// blank the SDK must fall through to details — this is exactly
	// the case that surfaced the duplicate-key error in twinos.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"code":500,"error":"","details":"organization: dup key: ford"}`))
	}))
	defer srv.Close()

	var ce *authclient.ClientError
	if !errors.As(runClientDoAgainst(t, srv), &ce) {
		t.Fatal("ClientError expected")
	}
	if !strings.Contains(ce.Message, "dup key: ford") {
		t.Errorf("Message = %q, want details substring", ce.Message)
	}
}

func TestClientError_preservesRawBodyForLogging(t *testing.T) {
	body := `{"code":500,"error":"","details":"deep cause"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", "req-abc")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	var ce *authclient.ClientError
	if !errors.As(runClientDoAgainst(t, srv), &ce) {
		t.Fatal("ClientError expected")
	}
	if string(ce.RawBody) != body {
		t.Errorf("RawBody = %q, want %q", ce.RawBody, body)
	}
	if ce.Headers["Content-Type"] != "application/json" {
		t.Errorf("Headers[Content-Type] = %q, want application/json", ce.Headers["Content-Type"])
	}
	if ce.Headers["X-Request-ID"] != "req-abc" {
		t.Errorf("Headers[X-Request-ID] = %q, want req-abc", ce.Headers["X-Request-ID"])
	}
}

func TestClientError_emptyBodyHasUsefulErrorString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := runClientDoAgainst(t, srv)
	if err == nil {
		t.Fatal("expected error")
	}
	// The previous regression surfaced as "authsome: 500 " (literal
	// trailing space, no signal). The new format must include a
	// hint that the body was empty.
	if !strings.Contains(err.Error(), "empty body") {
		t.Errorf("err.Error() = %q, want 'empty body' marker", err)
	}
}

func TestClientError_nonJSONBodyPreservedAsSnippet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>nginx says no</html>"))
	}))
	defer srv.Close()

	err := runClientDoAgainst(t, srv)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "nginx says no") {
		t.Errorf("err.Error() = %q, must surface the raw body", err)
	}
}

func TestClientError_largeBodyTruncatedInErrorString(t *testing.T) {
	huge := strings.Repeat("A", 1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(huge))
	}))
	defer srv.Close()

	err := runClientDoAgainst(t, srv)
	// Error() should truncate at 256 chars + ellipsis so logs stay
	// readable; RawBody preserves the full payload (capped at the
	// 8KiB read limit) for callers that want it.
	if !strings.Contains(err.Error(), "…") {
		t.Errorf("err.Error() = %q, want truncation marker", err)
	}
	if len(err.Error()) > 400 {
		t.Errorf("err.Error() length %d — should be bounded", len(err.Error()))
	}
}
