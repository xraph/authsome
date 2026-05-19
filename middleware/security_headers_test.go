package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
)

func newSecHeadersRouter(t *testing.T, opts middleware.SecurityHeadersOptions) http.Handler {
	t.Helper()
	r := forge.NewRouter()
	r.Use(middleware.SecurityHeaders(opts))
	r.GET("/x", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})
	return r
}

func TestSecurityHeaders_DefaultsApplied(t *testing.T) {
	t.Parallel()
	router := newSecHeadersRouter(t, middleware.SecurityHeadersOptions{})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	h := rec.Header()
	assert.Contains(t, h.Get("Content-Security-Policy"), "frame-ancestors 'none'")
	assert.Equal(t, "nosniff", h.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", h.Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", h.Get("Referrer-Policy"))
	assert.Contains(t, h.Get("Permissions-Policy"), "geolocation=()")
	assert.Empty(t, h.Get("Strict-Transport-Security"),
		"HSTS must be opt-in via HSTSMaxAgeSeconds; default-off avoids local-dev TLS pinning")
}

func TestSecurityHeaders_HSTSOptIn(t *testing.T) {
	t.Parallel()
	router := newSecHeadersRouter(t, middleware.SecurityHeadersOptions{
		HSTSMaxAgeSeconds:     31536000,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.Contains(t, hsts, "max-age=31536000")
	assert.Contains(t, hsts, "includeSubDomains")
	assert.Contains(t, hsts, "preload")
}

func TestSecurityHeaders_HSTSPreloadRequiresIncludeSubdomains(t *testing.T) {
	t.Parallel()
	router := newSecHeadersRouter(t, middleware.SecurityHeadersOptions{
		HSTSMaxAgeSeconds: 31536000,
		HSTSPreload:       true,
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	hsts := rec.Header().Get("Strict-Transport-Security")
	assert.NotContains(t, hsts, "preload",
		"preload directive must be ignored without includeSubDomains; submitting a domain to the preload list without subdomain coverage is a known footgun")
}

func TestSecurityHeaders_DisableCSP(t *testing.T) {
	t.Parallel()
	router := newSecHeadersRouter(t, middleware.SecurityHeadersOptions{
		DisableCSP: true,
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"),
		"non-CSP headers must still apply when CSP is disabled")
}

func TestSecurityHeaders_CustomOverrides(t *testing.T) {
	t.Parallel()
	router := newSecHeadersRouter(t, middleware.SecurityHeadersOptions{
		CSP:               "default-src https:",
		FrameOptions:      "SAMEORIGIN",
		ReferrerPolicy:    "no-referrer",
		PermissionsPolicy: "camera=()",
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	h := rec.Header()
	assert.Equal(t, "default-src https:", h.Get("Content-Security-Policy"))
	assert.Equal(t, "SAMEORIGIN", h.Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", h.Get("Referrer-Policy"))
	assert.Equal(t, "camera=()", h.Get("Permissions-Policy"))
}

func TestSecurityHeadersForAPI_DropsCSP(t *testing.T) {
	t.Parallel()
	r := forge.NewRouter()
	r.Use(middleware.SecurityHeadersForAPI(middleware.SecurityHeadersOptions{}))
	r.GET("/x", func(ctx forge.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
}

func TestSecurityHeaders_NoMutationOnNextError(t *testing.T) {
	t.Parallel()
	r := forge.NewRouter()
	r.Use(middleware.SecurityHeaders(middleware.SecurityHeadersOptions{}))
	r.GET("/boom", func(ctx forge.Context) error {
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "x"})
	})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"),
		"security headers must apply even on error responses")
	assert.True(t, strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src"))
}
