package authclient_test

// Tests for WithPublishableKey / SetPublishableKey: the SDK plumbing
// that gives the server enough context to route /v1/signup et al. into
// the correct app instead of silently falling back to the platform.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	authclient "github.com/xraph/authsome/sdk/go"
)

// TestWithPublishableKey_SendsHeaderOnEveryRequest pins the wire
// behavior the server fix relies on: when a publishable key is set,
// the SDK MUST stamp X-Publishable-Key on every request so the
// server-side middleware can resolve the app.
func TestWithPublishableKey_SendsHeaderOnEveryRequest(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Publishable-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"platform_app_id":"app_x"}`))
	}))
	defer srv.Close()

	c := authclient.NewClient(srv.URL, authclient.WithPublishableKey("pk_live_routing_test"))
	if _, err := c.GetManifest(context.Background()); err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	if got != "pk_live_routing_test" {
		t.Fatalf("X-Publishable-Key: got %q, want %q", got, "pk_live_routing_test")
	}
}

// TestSetPublishableKey_UpdatesAtRuntime mirrors the imperative setter
// path. Operators who switch keys mid-process (e.g. during a tenant
// switch) must see the new key on the very next request.
func TestSetPublishableKey_UpdatesAtRuntime(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Publishable-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := authclient.NewClient(srv.URL)
	c.SetPublishableKey("pk_test_runtime")
	if _, err := c.GetManifest(context.Background()); err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	if got != "pk_test_runtime" {
		t.Fatalf("X-Publishable-Key after SetPublishableKey: got %q, want %q", got, "pk_test_runtime")
	}
	if pk := c.PublishableKey(); pk != "pk_test_runtime" {
		t.Fatalf("PublishableKey getter: got %q, want %q", pk, "pk_test_runtime")
	}
}

// TestWithPublishableKey_RejectsSecretKeys ensures a misconfigured
// caller can't accidentally put a secret key on the wire as a
// publishable. Secret keys (sk_*, ask_*) should be silently dropped at
// configure time so the request lands without a pk header rather than
// leaking the secret to logs / proxies that grep for X-Publishable-Key.
func TestWithPublishableKey_RejectsSecretKeys(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Publishable-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	for _, secret := range []string{"sk_test_abc", "sk_stg_abc", "sk_live_abc", "ask_abc"} {
		got = ""
		c := authclient.NewClient(srv.URL, authclient.WithPublishableKey(secret))
		if _, err := c.GetManifest(context.Background()); err != nil {
			t.Fatalf("WithPublishableKey(%q): unexpected error %v", secret, err)
		}
		if got != "" {
			t.Fatalf("WithPublishableKey(%q): secret leaked as X-Publishable-Key header (%q)", secret, got)
		}
		if pk := c.PublishableKey(); pk != "" {
			t.Fatalf("WithPublishableKey(%q): secret stored on client (%q)", secret, pk)
		}
	}
}

// TestPublishableKey_OmittedWhenUnset confirms the absence of the
// header when the SDK was never configured with a pk — the new
// behavior must be additive, not breaking.
func TestPublishableKey_OmittedWhenUnset(t *testing.T) {
	gotHeaders := http.Header{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := authclient.NewClient(srv.URL)
	if _, err := c.GetManifest(context.Background()); err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	if v := gotHeaders.Get("X-Publishable-Key"); v != "" {
		t.Fatalf("expected no X-Publishable-Key when unset, got %q", v)
	}
}
