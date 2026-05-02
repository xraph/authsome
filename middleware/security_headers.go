package middleware

import (
	"net/http"
	"strings"

	"github.com/xraph/forge"
)

// SecurityHeadersOptions configures SecurityHeaders.
//
// Defaults are tuned for the dashboard's HTML routes. JSON-only API
// surfaces should pass APIRoutes:true to drop the HTML-shaped CSP.
type SecurityHeadersOptions struct {
	// HSTSMaxAgeSeconds is the max-age value emitted in the
	// Strict-Transport-Security header. Zero (the default) means
	// "do not emit HSTS" — appropriate for local development. Set
	// to 31536000 (1 year) for production deployments behind TLS.
	HSTSMaxAgeSeconds int

	// HSTSIncludeSubdomains adds includeSubDomains to the HSTS
	// directive. Only meaningful when HSTSMaxAgeSeconds > 0.
	HSTSIncludeSubdomains bool

	// HSTSPreload adds the preload directive (the operator must
	// also submit the domain to hstspreload.org). Only meaningful
	// when HSTSMaxAgeSeconds > 0 and HSTSIncludeSubdomains is true.
	HSTSPreload bool

	// CSP overrides the default Content-Security-Policy. When
	// empty, a conservative default is used: own-origin scripts
	// and styles, no framing, no inline scripts, no plugins.
	// Pass an empty value via DisableCSP to opt out entirely
	// (e.g. for JSON-only APIs where CSP has no effect).
	CSP string

	// DisableCSP omits the Content-Security-Policy header.
	// Useful for pure-JSON API routes where CSP is irrelevant
	// and a misconfigured policy can confuse responders.
	DisableCSP bool

	// FrameOptions is the value of X-Frame-Options. Defaults to
	// "DENY". Set to "SAMEORIGIN" if the dashboard ever needs to
	// embed itself; never set to ALLOW-FROM (deprecated).
	FrameOptions string

	// ReferrerPolicy is the value of Referrer-Policy. Defaults to
	// "strict-origin-when-cross-origin" — the OWASP-recommended
	// default that preserves analytics on same-origin while
	// stripping query strings on cross-origin navigations.
	ReferrerPolicy string

	// PermissionsPolicy is the value of Permissions-Policy.
	// Defaults to a conservative deny-list that disables sensors,
	// payment, and serial APIs the dashboard never uses.
	PermissionsPolicy string
}

const (
	defaultCSP = "default-src 'self'; " +
		"script-src 'self'; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src 'self' data: https:; " +
		"font-src 'self' data:; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none'; " +
		"form-action 'self'; " +
		"base-uri 'self'; " +
		"object-src 'none'"

	defaultFrameOptions      = "DENY"
	defaultReferrerPolicy    = "strict-origin-when-cross-origin"
	defaultPermissionsPolicy = "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=(), serial=()"
)

// SecurityHeaders is a forge.Middleware that emits a baseline set of
// security response headers on every served request.
//
// Headers (set unconditionally unless an option overrides them):
//   - Content-Security-Policy
//   - X-Content-Type-Options: nosniff
//   - X-Frame-Options: DENY
//   - Referrer-Policy: strict-origin-when-cross-origin
//   - Permissions-Policy: deny-list of sensor/payment APIs
//   - Strict-Transport-Security (only when HSTSMaxAgeSeconds > 0)
//
// The middleware is conservative by design: callers can widen the CSP
// (e.g. to allow a CDN) but the default closes XSS and clickjacking
// surfaces that the dashboard cannot exercise legitimately.
func SecurityHeaders(opts SecurityHeadersOptions) forge.Middleware {
	csp := opts.CSP
	if csp == "" {
		csp = defaultCSP
	}
	frameOptions := opts.FrameOptions
	if frameOptions == "" {
		frameOptions = defaultFrameOptions
	}
	referrerPolicy := opts.ReferrerPolicy
	if referrerPolicy == "" {
		referrerPolicy = defaultReferrerPolicy
	}
	permissionsPolicy := opts.PermissionsPolicy
	if permissionsPolicy == "" {
		permissionsPolicy = defaultPermissionsPolicy
	}
	hsts := buildHSTSValue(opts)

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			h := ctx.Response().Header()
			if !opts.DisableCSP {
				h.Set("Content-Security-Policy", csp)
			}
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", frameOptions)
			h.Set("Referrer-Policy", referrerPolicy)
			h.Set("Permissions-Policy", permissionsPolicy)
			if hsts != "" {
				h.Set("Strict-Transport-Security", hsts)
			}
			return next(ctx)
		}
	}
}

func buildHSTSValue(opts SecurityHeadersOptions) string {
	if opts.HSTSMaxAgeSeconds <= 0 {
		return ""
	}
	parts := []string{"max-age=" + itoaInt(opts.HSTSMaxAgeSeconds)}
	if opts.HSTSIncludeSubdomains {
		parts = append(parts, "includeSubDomains")
	}
	if opts.HSTSIncludeSubdomains && opts.HSTSPreload {
		parts = append(parts, "preload")
	}
	return strings.Join(parts, "; ")
}

// SecurityHeadersForAPI is a convenience preset that drops the CSP
// (irrelevant for JSON responses) but keeps the rest. Use this on
// routers that serve only JSON.
func SecurityHeadersForAPI(opts SecurityHeadersOptions) forge.Middleware {
	opts.DisableCSP = true
	return SecurityHeaders(opts)
}

// Compile-time assertion that the helper compiles against http.Header.
var _ = http.Header{}
