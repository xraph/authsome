package extension

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	"github.com/xraph/forgeui/router"

	"github.com/xraph/authsome/dashboard"
	authclient "github.com/xraph/authsome/sdk/go"
)

// TestClientAuthPages_Register_SendsAppIDInBody pins the wire shape that
// fixes the "400 app context required" regression: when the dashboard runs
// in client mode against a remote authsome, register submissions must
// include the SDK-discovered platform AppID in the request body so the
// upstream's resolvePublicAppID accepts the call without an
// X-Publishable-Key header.
func TestClientAuthPages_Register_SendsAppIDInBody(t *testing.T) {
	t.Parallel()

	const platformAppID = "app_platform_test"
	var gotPath string
	var gotBody map[string]any

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user":{"id":"u_1","email":"new@example.com"},"session_token":"sess_1","refresh_token":"r_1","expires_at":"2099-01-01T00:00:00Z"}`))
	}))
	defer upstream.Close()

	client := authclient.NewClient(upstream.URL, authclient.WithAppID(platformAppID))
	pages := &clientAuthPages{client: client, basePath: ""}

	form := url.Values{}
	form.Set("email", "new@example.com")
	form.Set("password", "hunter22!")
	form.Set("first_name", "New")
	form.Set("last_name", "User")
	form.Set(dashboard.FormCSRFFormField, "csrf-tok")

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "https://dashboard.local/register", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: dashboard.FormCSRFCookieName, Value: "csrf-tok"})

	rec := httptest.NewRecorder()
	ctx := &router.PageContext{Request: req, ResponseWriter: rec}

	redirect, comp, err := pages.HandleAuthAction(ctx, dashauth.PageRegister)
	if err != nil {
		t.Fatalf("HandleAuthAction returned error: %v", err)
	}
	if comp != nil {
		t.Fatalf("expected nil component on success, got %#v", comp)
	}
	if redirect != "/" {
		t.Fatalf("redirect = %q, want %q (success → basePath root)", redirect, "/")
	}

	if gotPath != "/v1/signup" {
		t.Fatalf("upstream path = %q, want /v1/signup", gotPath)
	}
	if got, _ := gotBody["app_id"].(string); got != platformAppID {
		t.Fatalf(`request body "app_id" = %q, want %q (fix for "400 app context required")`, got, platformAppID)
	}
}

// TestClientAuthPages_ForgotPassword_SendsAppIDInBody pins the same wire
// contract for /forgot-password. The endpoint is enumeration-resistant
// (always 200), so this test asserts the call shape rather than the
// response — the dashboard's success page renders regardless.
func TestClientAuthPages_ForgotPassword_SendsAppIDInBody(t *testing.T) {
	t.Parallel()

	const platformAppID = "app_platform_test"
	var gotPath string
	var gotBody map[string]any

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer upstream.Close()

	client := authclient.NewClient(upstream.URL, authclient.WithAppID(platformAppID))
	pages := &clientAuthPages{client: client, basePath: ""}

	form := url.Values{}
	form.Set("email", "someone@example.com")
	form.Set(dashboard.FormCSRFFormField, "csrf-tok")

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "https://dashboard.local/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: dashboard.FormCSRFCookieName, Value: "csrf-tok"})

	rec := httptest.NewRecorder()
	ctx := &router.PageContext{Request: req, ResponseWriter: rec}

	if _, _, err := pages.HandleAuthAction(ctx, dashauth.PageForgotPassword); err != nil {
		t.Fatalf("HandleAuthAction returned error: %v", err)
	}

	if gotPath != "/v1/forgot-password" {
		t.Fatalf("upstream path = %q, want /v1/forgot-password", gotPath)
	}
	if got, _ := gotBody["app_id"].(string); got != platformAppID {
		t.Fatalf(`request body "app_id" = %q, want %q (fix for "400 app context required")`, got, platformAppID)
	}
}
