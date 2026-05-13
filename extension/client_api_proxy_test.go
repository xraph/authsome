package extension

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xraph/forge"
)

// TestBuildClientAPIProxy_ForwardsAuthCookieAsBearer pins the contract that
// the dashboard's auth_token cookie is promoted to an Authorization: Bearer
// header on the proxied request, and that the original Cookie header is
// stripped (cookies on the dashboard host's domain are not meaningful to
// the upstream authsome service).
func TestBuildClientAPIProxy_ForwardsAuthCookieAsBearer(t *testing.T) {
	t.Parallel()

	var (
		gotAuth   string
		gotCookie string
		gotPath   string
		gotMethod string
		gotBody   string
	)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCookie = r.Header.Get("Cookie")
		gotPath = r.URL.Path
		gotMethod = r.Method
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	e := newTestExtension(upstream.URL)

	mount, proxy, err := e.buildClientAPIProxy()
	if err != nil {
		t.Fatalf("buildClientAPIProxy: %v", err)
	}
	if mount != "/authsome/v1/*" {
		t.Fatalf("mount prefix = %q, want /authsome/v1/*", mount)
	}

	req := httptest.NewRequest(http.MethodPut,
		"http://dashboard.local/authsome/v1/admin/settings/values/auth.require_email_verification",
		strings.NewReader(`{"value":true,"scope":"global"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "tok-from-dashboard"})

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("proxied status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if gotMethod != http.MethodPut {
		t.Errorf("upstream method = %q, want PUT", gotMethod)
	}
	if gotPath != "/authsome/v1/admin/settings/values/auth.require_email_verification" {
		t.Errorf("upstream path = %q, want preserved /authsome/v1/admin/...", gotPath)
	}
	if gotAuth != "Bearer tok-from-dashboard" {
		t.Errorf("upstream Authorization = %q, want Bearer tok-from-dashboard", gotAuth)
	}
	if gotCookie != "" {
		t.Errorf("upstream Cookie = %q, want empty (must be stripped)", gotCookie)
	}
	if gotBody != `{"value":true,"scope":"global"}` {
		t.Errorf("upstream body = %q, want JSON payload echoed through", gotBody)
	}
}

// TestBuildClientAPIProxy_PreservesExplicitAuthorization ensures a caller
// that already supplied Authorization: Bearer (e.g. an SDK consumer hitting
// the proxy directly) wins over the cookie path — we only fall back to the
// cookie when no header is present.
func TestBuildClientAPIProxy_PreservesExplicitAuthorization(t *testing.T) {
	t.Parallel()

	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	e := &Extension{
		clientMode: true,
		config:     Config{BasePath: "/authsome", PortalURL: upstream.URL},
	}
	_, proxy, err := e.buildClientAPIProxy()
	if err != nil {
		t.Fatalf("buildClientAPIProxy: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"http://dashboard.local/authsome/v1/admin/settings/definitions", nil)
	req.Header.Set("Authorization", "Bearer explicit-sdk-token")
	req.AddCookie(&http.Cookie{Name: "auth_token", Value: "cookie-token"})

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("proxied status = %d, want 200", rec.Code)
	}
	if gotAuth != "Bearer explicit-sdk-token" {
		t.Errorf("upstream Authorization = %q, want Bearer explicit-sdk-token", gotAuth)
	}
}

// TestBuildClientAPIProxy_NoAuthWhenAbsent confirms an unauthenticated
// inbound request is forwarded without an Authorization header so upstream
// can return its canonical 401 instead of the proxy synthesising one.
func TestBuildClientAPIProxy_NoAuthWhenAbsent(t *testing.T) {
	t.Parallel()

	var gotAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer upstream.Close()

	e := &Extension{
		clientMode: true,
		config:     Config{BasePath: "/authsome", PortalURL: upstream.URL},
	}
	_, proxy, err := e.buildClientAPIProxy()
	if err != nil {
		t.Fatalf("buildClientAPIProxy: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet,
		"http://dashboard.local/authsome/v1/admin/settings/definitions", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("proxied status = %d, want 401 (passthrough)", rec.Code)
	}
	if gotAuth != "" {
		t.Errorf("upstream Authorization = %q, want empty", gotAuth)
	}
}

// TestBuildClientAPIProxy_RejectsBadPortalURL guards the boot-time
// validation so misconfigured PortalURLs surface as a clear error instead
// of a runtime panic on the first request.
func TestBuildClientAPIProxy_RejectsBadPortalURL(t *testing.T) {
	t.Parallel()

	e := newTestExtension("not-a-url")
	if _, _, err := e.buildClientAPIProxy(); err == nil {
		t.Fatal("expected error for PortalURL without scheme/host, got nil")
	}
}

// newTestExtension builds a minimal client-mode Extension wired with the
// embedded BaseExtension that production code constructs via New(). Tests
// that exercise the proxy directly need this so e.Logger() is safe to call.
func newTestExtension(portalURL string) *Extension {
	e := New()
	e.clientMode = true
	e.config = Config{BasePath: "/authsome", PortalURL: portalURL}
	return e
}

// TestRegisterClientAPIProxy_OnRealRouter pins the regression that was
// observed in the TwinOS portal: forge's BunRouterAdapter.Mount registers
// both the exact path and a "/*filepath" wildcard for each method when the
// supplied path lacks "/*", which makes bunrouter panic with
//
//	routes "/authsome/v1/" and "/authsome/v1/*filepath" can't both handle GET
//
// We pass "/authsome/v1/*" explicitly so the adapter's wildcard-only
// branch fires. This test would have caught that panic before it shipped.
func TestRegisterClientAPIProxy_OnRealRouter(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	e := newTestExtension(upstream.URL)

	router := forge.NewRouter()
	if err := e.registerClientAPIProxy(router); err != nil {
		t.Fatalf("registerClientAPIProxy: %v", err)
	}
}
