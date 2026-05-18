package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/captcha"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
)

// Captcha setting keys (mirror the definitions in captcha_settings.go to
// avoid a middleware → authsome import cycle).
const (
	captchaSettingRequired  = "auth.captcha_required"
	captchaSettingProvider  = "auth.captcha_provider"
	captchaSettingSecretKey = "auth.captcha_secret_key" //nolint:gosec // this is a setting *key name*, not a credential
)

// captchaTokenHeader is the canonical header for the captcha challenge token.
// Same name accepted by Cloudflare Turnstile and hCaptcha widget integrations.
const captchaTokenHeader = "X-Captcha-Token" //nolint:gosec // G101: header name, not a credential

// captchaTokenFormField is the form/query fallback for HTML form posts where
// adding a custom header isn't ergonomic.
const captchaTokenFormField = "captcha_token" //nolint:gosec // G101: form field name, not a credential

// CaptchaOptions configures CaptchaMiddleware.
type CaptchaOptions struct {
	// Settings is the settings manager used to resolve per-app captcha
	// configuration at request time. Required.
	Settings *settings.Manager

	// ResolveAppID returns the app id for a request. Defaults to a
	// resolver that reads forge.AppIDFrom(ctx). Returning ok=false skips
	// app-scoped settings (global cascade only).
	ResolveAppID func(forge.Context) (id.AppID, bool)

	// VerifierFor builds a captcha.Verifier from a (provider, secret) pair.
	// Defaults to defaultCaptchaVerifierFor which knows about Turnstile.
	VerifierFor func(provider, secret string) (captcha.Verifier, error)

	// Action is the optional captcha action binding (e.g. "signup",
	// "signin"). Empty disables the action check.
	Action string

	// Chronicle, if non-nil, records audit events with Action="captcha.verify".
	Chronicle bridge.Chronicle

	// Logger for transient verifier failures. May be nil.
	Logger log.Logger
}

// CaptchaResult is the outcome of an out-of-middleware captcha check.
// Allowed reports whether the request may proceed; when false, RejectCode is
// the same stable type code the middleware would have written
// ("captcha_required", "captcha_invalid", "captcha_unavailable",
// "captcha_misconfigured"). RejectStatus is the HTTP status code that
// matches.
type CaptchaResult struct {
	Allowed      bool
	RejectCode   string
	RejectStatus int
	Reason       string
}

// VerifyCaptchaForRequest is a non-forge variant of CaptchaMiddleware for
// callers that aren't part of forge's standard middleware chain (e.g.
// dashboard auth-pages handlers, which receive a *router.PageContext rather
// than a forge.Context). The semantics match the middleware exactly:
//
//   - If auth.captcha_required resolves to false: returns Allowed=true.
//   - Token missing: Allowed=false, RejectCode="captcha_required".
//   - Token rejected: Allowed=false, RejectCode="captcha_invalid".
//   - Verifier transient failure: Allowed=false, RejectCode="captcha_unavailable".
//   - Verifier construction failure: Allowed=false, RejectCode="captcha_misconfigured".
//
// The optional appID scopes setting resolution; pass id.AppID{} for global only.
// Audit recording is the caller's responsibility (the middleware records it
// internally; out-of-middleware callers may want different fields).
func VerifyCaptchaForRequest(ctx context.Context, opts CaptchaOptions, r *http.Request, appID id.AppID, action string) CaptchaResult {
	if opts.VerifierFor == nil {
		opts.VerifierFor = defaultCaptchaVerifierFor
	}
	resolveOpts := settings.ResolveOpts{}
	if !appID.IsNil() {
		resolveOpts.AppID = appID.String()
	}

	required, err := captchaResolveBool(ctx, opts.Settings, captchaSettingRequired, resolveOpts)
	if err != nil || !required {
		return CaptchaResult{Allowed: true}
	}

	token := captchaExtractToken(r)
	if token == "" {
		return CaptchaResult{
			Allowed: false, RejectCode: "captcha_required",
			RejectStatus: http.StatusForbidden, Reason: "missing-token",
		}
	}

	provider, _ := captchaResolveString(ctx, opts.Settings, captchaSettingProvider, resolveOpts) //nolint:errcheck // best-effort
	if provider == "" {
		provider = "turnstile"
	}
	secret, _ := captchaResolveString(ctx, opts.Settings, captchaSettingSecretKey, resolveOpts) //nolint:errcheck // best-effort

	verifier, err := opts.VerifierFor(provider, secret)
	if err != nil {
		if opts.Logger != nil {
			opts.Logger.Error("captcha: build verifier failed",
				log.String("provider", provider),
				log.String("error", err.Error()))
		}
		return CaptchaResult{
			Allowed: false, RejectCode: "captcha_misconfigured",
			RejectStatus: http.StatusServiceUnavailable, Reason: "verifier-build-failed",
		}
	}

	remoteIP := captchaClientIP(r)
	if _, verifyErr := verifier.Verify(ctx, token, remoteIP, action); verifyErr != nil {
		switch {
		case errors.Is(verifyErr, captcha.ErrMissingToken):
			return CaptchaResult{
				Allowed: false, RejectCode: "captcha_required",
				RejectStatus: http.StatusForbidden, Reason: "verifier-rejected-empty",
			}
		case errors.Is(verifyErr, captcha.ErrInvalidToken):
			return CaptchaResult{
				Allowed: false, RejectCode: "captcha_invalid",
				RejectStatus: http.StatusForbidden, Reason: "verifier-rejected-invalid",
			}
		case errors.Is(verifyErr, captcha.ErrDuplicateToken):
			return CaptchaResult{
				Allowed: false, RejectCode: "captcha_invalid",
				RejectStatus: http.StatusForbidden, Reason: "verifier-rejected-duplicate",
			}
		case errors.Is(verifyErr, captcha.ErrTransientFailure):
			if opts.Logger != nil {
				opts.Logger.Warn("captcha: transient verifier failure",
					log.String("error", verifyErr.Error()))
			}
			return CaptchaResult{
				Allowed: false, RejectCode: "captcha_unavailable",
				RejectStatus: http.StatusServiceUnavailable, Reason: "transient-failure",
			}
		default:
			return CaptchaResult{
				Allowed: false, RejectCode: "captcha_invalid",
				RejectStatus: http.StatusForbidden, Reason: "verifier-rejected-other",
			}
		}
	}

	return CaptchaResult{Allowed: true}
}

// CaptchaMiddleware gates a request on a successful captcha verification.
//
// The token is read from header X-Captcha-Token first, falling back to the
// form/query field "captcha_token" so the same middleware works for JSON
// APIs and HTML form posts.
//
// Behaviour:
//   - If auth.captcha_required is false for the resolved app: pass through.
//   - Token missing: 403 with type="captcha_required".
//   - Token rejected by verifier: 403 with type="captcha_invalid".
//   - Verifier transient failure: 503 with type="captcha_unavailable".
//
// The verifier is constructed lazily and cached per (appID, provider, secret)
// so secret rotation in the dashboard takes effect on the next request
// without a process restart.
//
// Order: this middleware must run BEFORE any CPU-bound work (password
// hashing, etc.) so a captcha-failed probe doesn't pay argon2 cost
// server-side.
func CaptchaMiddleware(opts CaptchaOptions) forge.Middleware {
	if opts.VerifierFor == nil {
		opts.VerifierFor = defaultCaptchaVerifierFor
	}
	if opts.ResolveAppID == nil {
		opts.ResolveAppID = func(ctx forge.Context) (id.AppID, bool) {
			scoped := forge.AppIDFrom(ctx.Context())
			if scoped == "" {
				return id.AppID{}, false
			}
			parsed, err := id.ParseAppID(scoped)
			if err != nil {
				return id.AppID{}, false
			}
			return parsed, true
		}
	}

	var cache sync.Map // key: appID|provider|secret → captcha.Verifier

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			appID, hasApp := opts.ResolveAppID(ctx)
			resolveOpts := settings.ResolveOpts{}
			if hasApp {
				resolveOpts.AppID = appID.String()
			}

			required, err := captchaResolveBool(ctx.Context(), opts.Settings, captchaSettingRequired, resolveOpts)
			if err != nil || !required {
				return next(ctx)
			}

			token := captchaExtractToken(ctx.Request())
			if token == "" {
				return captchaReject(ctx, opts, "captcha_required",
					"captcha challenge required for this endpoint", http.StatusForbidden,
					"missing-token")
			}

			provider, _ := captchaResolveString(ctx.Context(), opts.Settings, captchaSettingProvider, resolveOpts) //nolint:errcheck // best-effort
			if provider == "" {
				provider = "turnstile"
			}
			secret, _ := captchaResolveString(ctx.Context(), opts.Settings, captchaSettingSecretKey, resolveOpts) //nolint:errcheck // best-effort

			cacheKey := appID.String() + "|" + provider + "|" + secret
			verifier, err := captchaVerifierFromCache(&cache, cacheKey, provider, secret, opts.VerifierFor)
			if err != nil {
				if opts.Logger != nil {
					opts.Logger.Error("captcha: build verifier failed",
						log.String("provider", provider),
						log.String("error", err.Error()))
				}
				return captchaReject(ctx, opts, "captcha_misconfigured",
					"captcha provider is not configured correctly", http.StatusServiceUnavailable,
					"verifier-build-failed")
			}

			remoteIP := captchaClientIP(ctx.Request())
			if _, verifyErr := verifier.Verify(ctx.Context(), token, remoteIP, opts.Action); verifyErr != nil {
				return captchaHandleVerifyError(ctx, opts, verifyErr)
			}

			captchaRecordAudit(opts, ctx, "success", "")
			return next(ctx)
		}
	}
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func captchaResolveBool(ctx context.Context, mgr *settings.Manager, key string, opts settings.ResolveOpts) (bool, error) {
	raw, err := mgr.Resolve(ctx, key, opts)
	if err != nil {
		return false, err
	}
	var val bool
	if uErr := json.Unmarshal(raw, &val); uErr != nil {
		return false, uErr
	}
	return val, nil
}

func captchaResolveString(ctx context.Context, mgr *settings.Manager, key string, opts settings.ResolveOpts) (string, error) {
	raw, err := mgr.Resolve(ctx, key, opts)
	if err != nil {
		return "", err
	}
	var val string
	if uErr := json.Unmarshal(raw, &val); uErr != nil {
		return "", uErr
	}
	return val, nil
}

func captchaExtractToken(r *http.Request) string {
	if v := r.Header.Get(captchaTokenHeader); v != "" {
		return v
	}
	return r.FormValue(captchaTokenFormField)
}

func captchaClientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		// First entry is the original client.
		if idx := strings.Index(v, ","); idx >= 0 {
			return strings.TrimSpace(v[:idx])
		}
		return strings.TrimSpace(v)
	}
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return strings.TrimSpace(v)
	}
	if i := strings.LastIndex(r.RemoteAddr, ":"); i >= 0 {
		return r.RemoteAddr[:i]
	}
	return r.RemoteAddr
}

func captchaVerifierFromCache(cache *sync.Map, key, provider, secret string, factory func(string, string) (captcha.Verifier, error)) (captcha.Verifier, error) {
	if v, ok := cache.Load(key); ok {
		return v.(captcha.Verifier), nil //nolint:errcheck // type assertion: stored values are always Verifier
	}
	v, err := factory(provider, secret)
	if err != nil {
		return nil, err
	}
	actual, _ := cache.LoadOrStore(key, v)
	return actual.(captcha.Verifier), nil //nolint:errcheck // type assertion: stored values are always Verifier
}

func captchaHandleVerifyError(ctx forge.Context, opts CaptchaOptions, err error) error {
	switch {
	case errors.Is(err, captcha.ErrMissingToken):
		return captchaReject(ctx, opts, "captcha_required",
			"captcha challenge required", http.StatusForbidden,
			"verifier-rejected-empty")
	case errors.Is(err, captcha.ErrInvalidToken):
		return captchaReject(ctx, opts, "captcha_invalid",
			"captcha challenge failed", http.StatusForbidden,
			"verifier-rejected-invalid")
	case errors.Is(err, captcha.ErrDuplicateToken):
		return captchaReject(ctx, opts, "captcha_invalid",
			"captcha challenge already used", http.StatusForbidden,
			"verifier-rejected-duplicate")
	case errors.Is(err, captcha.ErrTransientFailure):
		if opts.Logger != nil {
			opts.Logger.Warn("captcha: transient verifier failure",
				log.String("error", err.Error()))
		}
		return captchaReject(ctx, opts, "captcha_unavailable",
			"captcha verification temporarily unavailable", http.StatusServiceUnavailable,
			"transient-failure")
	default:
		// Unknown error class — also a *VerifyError (action-mismatch, etc.) or
		// something unexpected. Treat as invalid.
		return captchaReject(ctx, opts, "captcha_invalid",
			"captcha challenge failed", http.StatusForbidden,
			"verifier-rejected-other")
	}
}

func defaultCaptchaVerifierFor(provider, secret string) (captcha.Verifier, error) {
	switch provider {
	case "", "turnstile":
		if secret == "" {
			return nil, errors.New("captcha: empty secret for turnstile")
		}
		return captcha.NewTurnstileVerifier(secret, nil), nil
	default:
		return nil, errors.New("captcha: unknown provider: " + provider)
	}
}

func captchaReject(ctx forge.Context, opts CaptchaOptions, code, message string, status int, auditReason string) error {
	captchaRecordAudit(opts, ctx, "failure", auditReason)
	return ctx.JSON(status, map[string]any{
		"error": message,
		"code":  status,
		"type":  code,
	})
}

func captchaRecordAudit(opts CaptchaOptions, ctx forge.Context, outcome, reason string) {
	if opts.Chronicle == nil {
		return
	}
	meta := map[string]string{}
	if opts.Action != "" {
		meta["action"] = opts.Action
	}
	if reason != "" {
		meta["reason"] = reason
	}
	severity := bridge.SeverityInfo
	if outcome == "failure" {
		severity = bridge.SeverityWarning
	}
	_ = opts.Chronicle.Record(ctx.Context(), &bridge.AuditEvent{ //nolint:errcheck // audit best-effort
		Action:   "captcha.verify",
		Severity: severity,
		Outcome:  outcome,
		Metadata: meta,
	})
}
