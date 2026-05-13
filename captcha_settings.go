package authsome

import (
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/settings"
)

// ──────────────────────────────────────────────────
// Captcha settings (Phase 2B.2)
// ──────────────────────────────────────────────────
//
// These settings configure the per-app captcha middleware. The middleware
// reads them at request time so secrets can be rotated through the dashboard
// without a process restart.
//
// auth.captcha_secret_key holds the *secret* used to call the provider's
// siteverify endpoint. It is marked Sensitive so the API redacts it and the
// dashboard renders it as a password input. NEVER log it, NEVER expose it
// via the manifest, NEVER echo it in API responses.

// Category: Captcha

var (
	// SettingCaptchaRequired toggles the captcha middleware. When false
	// (default) the middleware passes every request through, preserving
	// back-compat for existing deployments.
	SettingCaptchaRequired = settings.Define("auth.captcha_required", false,
		settings.WithDisplayName("Require Captcha"),
		settings.WithDescription("Gate sensitive endpoints (signup, signin) on a verified captcha challenge"),
		settings.WithCategory("Captcha"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, requests to gated endpoints must include a valid captcha token (X-Captcha-Token header or captcha_token form field)."),
		settings.WithOrder(200),
	)

	// SettingCaptchaProvider selects the captcha backend. Phase 2B only
	// ships "turnstile"; future providers may extend this list.
	SettingCaptchaProvider = settings.Define("auth.captcha_provider", "turnstile",
		settings.WithDisplayName("Captcha Provider"),
		settings.WithDescription("Captcha provider used to verify tokens"),
		settings.WithCategory("Captcha"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldSelect),
		settings.WithOptions(
			formconfig.SelectOption{Value: "turnstile", Label: "Cloudflare Turnstile"},
		),
		settings.WithHelpText("Phase 2B ships Turnstile only. hCaptcha and reCAPTCHA may follow."),
		settings.WithOrder(210),
		settings.WithVisibleWhen("auth.captcha_required", true),
	)

	// SettingCaptchaSiteKey is the public site key rendered in the
	// frontend captcha widget. Not used by the verifier — surfaced for
	// clients via the manifest.
	SettingCaptchaSiteKey = settings.Define("auth.captcha_site_key", "",
		settings.WithDisplayName("Captcha Site Key"),
		settings.WithDescription("Public site key for the captcha widget"),
		settings.WithCategory("Captcha"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldText),
		settings.WithHelpText("The public site key as shown by the provider's dashboard. Safe to expose to browsers."),
		settings.WithOrder(220),
		settings.WithVisibleWhen("auth.captcha_required", true),
	)

	// SettingCaptchaSecretKey is the secret used to call the provider's
	// siteverify endpoint. Marked Sensitive: the API redacts it, the
	// dashboard renders a password input, and the manifest never includes
	// it. NEVER log this value.
	SettingCaptchaSecretKey = settings.Define("auth.captcha_secret_key", "",
		settings.WithDisplayName("Captcha Secret Key"),
		settings.WithDescription("Secret key used to call the captcha provider's siteverify endpoint"),
		settings.WithCategory("Captcha"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithSensitive(),
		settings.WithInputType(formconfig.FieldText),
		settings.WithHelpText("Secret — never log, never expose via the manifest, never echo in API responses. Rotating this value invalidates the cached verifier on the next request."),
		settings.WithOrder(230),
		settings.WithVisibleWhen("auth.captcha_required", true),
	)
)

// registerCaptchaSettings registers the auth.captcha_* settings. Called from
// the engine alongside other core registrations.
func registerCaptchaSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "auth", SettingCaptchaRequired); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "auth", SettingCaptchaProvider); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "auth", SettingCaptchaSiteKey); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "auth", SettingCaptchaSecretKey)
}
