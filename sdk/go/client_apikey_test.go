package authclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authclient "github.com/xraph/authsome/sdk/go"
)

// TestWithAPIKey_RejectsPublishableKey pins that the SDK refuses
// publishable keys at configure time. The server-side API-key
// strategy explicitly rejects pk_* (plugins/apikey/plugin.go);
// without this guard the SDK would silently fire a doomed request
// and surface a confusing 401.
func TestWithAPIKey_RejectsPublishableKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		// Should never be hit — config error must short-circuit do().
		t.Fatalf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	for _, key := range []string{"pk_test_abc", "pk_stg_abc", "pk_live_abc"} {
		c := authclient.NewClient(srv.URL, authclient.WithAPIKey(key), authclient.WithAppID("app_x"))
		_, err := c.GetManifest(context.Background())
		if err == nil {
			t.Fatalf("WithAPIKey(%q): expected error, got nil", key)
		}
		if !strings.Contains(err.Error(), "publishable") {
			t.Fatalf("WithAPIKey(%q): expected error mentioning 'publishable', got %q", key, err.Error())
		}
	}
}

// TestSetAPIKey_RejectsPublishableKey mirrors the test above for
// the imperative setter — operators who switch keys at runtime
// shouldn't be able to silently downgrade to a publishable key.
func TestSetAPIKey_RejectsPublishableKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	c := authclient.NewClient(srv.URL, authclient.WithAppID("app_x"))
	c.SetAPIKey("pk_live_xyz")
	_, err := c.GetManifest(context.Background())
	if err == nil || !strings.Contains(err.Error(), "publishable") {
		t.Fatalf("SetAPIKey(pk_*): expected publishable-key error, got %v", err)
	}
}

// TestWithAPIKey_AcceptsSecretKeys ensures the guard does not
// over-reject — every legitimate secret prefix must be accepted.
func TestWithAPIKey_AcceptsSecretKeys(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"platform_app_id":"app_x"}`))
	}))
	defer srv.Close()

	for _, key := range []string{"ask_abc", "sk_test_abc", "sk_stg_abc", "sk_live_abc"} {
		c := authclient.NewClient(srv.URL, authclient.WithAPIKey(key), authclient.WithAppID("app_x"))
		if _, err := c.GetManifest(context.Background()); err != nil {
			t.Fatalf("WithAPIKey(%q): unexpected error %v", key, err)
		}
	}
}

// TestAdminCreateAppWithHint_Wraps403WithPermissionHint pins that
// a 403 from POST /v1/admin/apps surfaces the most-likely cause:
// the API-key-bound user lacks `manage:user` (typically because
// the platform_admin role was never assigned). Without this hint,
// operators routinely chase the wrong rabbit hole — wrong key,
// wrong app id, etc.
//
// The hint is opt-in via the hand-maintained AdminCreateAppWithHint
// wrapper; the auto-generated AdminCreateApp returns the raw
// ClientError unchanged so non-Go SDK consumers see uniform shapes.
func TestAdminCreateAppWithHint_Wraps403WithPermissionHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"insufficient permissions"}`))
	}))
	defer srv.Close()

	c := authclient.NewClient(srv.URL,
		authclient.WithAPIKey("sk_live_xyz"),
		authclient.WithAppID("app_x"),
	)
	_, err := c.AdminCreateAppWithHint(context.Background(), &authclient.AdminCreateAppRequest{
		Name: "x", Slug: "x",
	})
	if err == nil {
		t.Fatal("expected 403 error, got nil")
	}
	if !strings.Contains(err.Error(), "manage:user") || !strings.Contains(err.Error(), "platform_admin") {
		t.Fatalf("expected wrapped hint mentioning manage:user/platform_admin, got %q", err.Error())
	}
}

// TestDo_SendsBothTokenAndAPIKey pins the dual-credential contract:
// when both WithToken and WithAPIKey are configured, the SDK sends
// the session token in Authorization AND the API key in X-API-Key.
// This supports the "service-account API key + per-request user
// session token" pattern (see twinos's UserClient). The previous
// "token wins, drop X-API-Key" precedence broke this — every
// service-account call routed through the session validator and
// 401'd as "no strategy ran".
func TestDo_SendsBothTokenAndAPIKey(t *testing.T) {
	var (
		gotAuth   string
		gotAPIKey string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAPIKey = r.Header.Get("X-API-Key")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"platform_app_id":"app_x"}`))
	}))
	defer srv.Close()

	c := authclient.NewClient(srv.URL,
		authclient.WithToken("tok_abc"),
		authclient.WithAPIKey("sk_live_xyz"),
		authclient.WithAppID("app_x"),
	)
	if _, err := c.GetManifest(context.Background()); err != nil {
		t.Fatalf("GetManifest: %v", err)
	}
	if gotAuth != "Bearer tok_abc" {
		t.Errorf("Authorization header: got %q want %q", gotAuth, "Bearer tok_abc")
	}
	if gotAPIKey != "sk_live_xyz" {
		t.Errorf("X-API-Key header: got %q want %q", gotAPIKey, "sk_live_xyz")
	}
}
