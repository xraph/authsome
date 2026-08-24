package extension

import (
	"context"
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

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut,
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

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
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

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
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

// TestBuildClientAPIProxy_ForwardedHeadersDescribeTheClient pins what upstream
// is told about the caller.
//
// The previous Director set X-Forwarded-Host from req.Host immediately after
// overwriting req.Host with the upstream host, so upstream was handed its own
// hostname instead of the client's. Upstream keys tenant resolution and CSRF on
// these headers, so that value has to be the host the client actually used.
func TestBuildClientAPIProxy_ForwardedHeadersDescribeTheClient(t *testing.T) {
	t.Parallel()

	var gotHost, gotProto, gotFor, gotHostHeader string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Header.Get("X-Forwarded-Host")
		gotProto = r.Header.Get("X-Forwarded-Proto")
		gotFor = r.Header.Get("X-Forwarded-For")
		gotHostHeader = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	e := newTestExtension(upstream.URL)
	_, proxy, err := e.buildClientAPIProxy()
	if err != nil {
		t.Fatalf("buildClientAPIProxy: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		"http://dashboard.local/authsome/v1/admin/settings", nil)
	req.RemoteAddr = "203.0.113.9:54321"

	proxy.ServeHTTP(httptest.NewRecorder(), req)

	if gotHost != "dashboard.local" {
		t.Errorf("X-Forwarded-Host = %q, want the client's host dashboard.local", gotHost)
	}
	if gotProto != "http" {
		t.Errorf("X-Forwarded-Proto = %q, want http", gotProto)
	}
	if gotFor != "203.0.113.9" {
		t.Errorf("X-Forwarded-For = %q, want the peer address 203.0.113.9", gotFor)
	}
	// The Host header itself still targets upstream, so upstream's own routing
	// and TLS/vhost matching keep working.
	upstreamHost := strings.TrimPrefix(upstream.URL, "http://")
	if gotHostHeader != upstreamHost {
		t.Errorf("Host = %q, want the upstream host %q", gotHostHeader, upstreamHost)
	}
}

// TestBuildClientAPIProxy_IgnoresClientSuppliedForwardingHeaders is the security
// half of the migration.
//
// Under Director, a client-supplied X-Forwarded-Host survived to upstream (the
// old code only filled it in when absent), which let a caller choose the host
// upstream resolves a tenant from. Rewrite makes ReverseProxy strip every
// inbound Forwarded and X-Forwarded-* header first, so the values upstream sees
// are always derived from the connection.
func TestBuildClientAPIProxy_IgnoresClientSuppliedForwardingHeaders(t *testing.T) {
	t.Parallel()

	var gotHost, gotProto, gotFor, gotForwarded string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Header.Get("X-Forwarded-Host")
		gotProto = r.Header.Get("X-Forwarded-Proto")
		gotFor = r.Header.Get("X-Forwarded-For")
		gotForwarded = r.Header.Get("Forwarded")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	e := newTestExtension(upstream.URL)
	_, proxy, err := e.buildClientAPIProxy()
	if err != nil {
		t.Fatalf("buildClientAPIProxy: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		"http://dashboard.local/authsome/v1/admin/settings", nil)
	req.RemoteAddr = "203.0.113.9:54321"
	// A caller trying to pick the tenant and hide its address.
	req.Header.Set("X-Forwarded-Host", "victim-tenant.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	req.Header.Set("Forwarded", "host=victim-tenant.example.com;proto=https")

	proxy.ServeHTTP(httptest.NewRecorder(), req)

	if gotHost == "victim-tenant.example.com" {
		t.Errorf("X-Forwarded-Host = %q: a client-supplied host reached upstream", gotHost)
	}
	if gotHost != "dashboard.local" {
		t.Errorf("X-Forwarded-Host = %q, want dashboard.local", gotHost)
	}
	if gotProto != "http" {
		t.Errorf("X-Forwarded-Proto = %q, want http derived from the connection", gotProto)
	}
	if strings.Contains(gotFor, "10.0.0.1") {
		t.Errorf("X-Forwarded-For = %q: a client-supplied address reached upstream", gotFor)
	}
	if gotFor != "203.0.113.9" {
		t.Errorf("X-Forwarded-For = %q, want only the peer address", gotFor)
	}
	if gotForwarded != "" {
		t.Errorf("Forwarded = %q, want it stripped", gotForwarded)
	}
}

// TestBuildClientAPIProxy_DoesNotJoinPortalPath pins why the Rewrite func sets
// the URL by hand instead of calling ProxyRequest.SetURL: SetURL joins the
// target's path onto the inbound one, which would duplicate the prefix upstream
// already expects.
func TestBuildClientAPIProxy_DoesNotJoinPortalPath(t *testing.T) {
	t.Parallel()

	var gotPath string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	// A PortalURL carrying a trailing path is the case that would double up.
	e := newTestExtension(upstream.URL + "/authsome")
	_, proxy, err := e.buildClientAPIProxy()
	if err != nil {
		t.Fatalf("buildClientAPIProxy: %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
		"http://dashboard.local/authsome/v1/admin/settings", nil)

	proxy.ServeHTTP(httptest.NewRecorder(), req)

	if gotPath != "/authsome/v1/admin/settings" {
		t.Errorf("upstream path = %q, want the inbound path unchanged", gotPath)
	}
}
