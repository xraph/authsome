package authsome

import (
	"encoding/json"
	"fmt"

	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/settings"
)

// ──────────────────────────────────────────────────
// Core settings
// ──────────────────────────────────────────────────

// Category: Authentication

var (
	// SettingRequireEmailVerification controls whether unverified email addresses
	// block sign-in. When enabled, users must verify their email before they can log in.
	SettingRequireEmailVerification = settings.Define("auth.require_email_verification", false,
		settings.WithDisplayName("Require Email Verification"),
		settings.WithDescription("Block sign-in for accounts with unverified email addresses"),
		settings.WithCategory("Authentication"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, users must verify their email before signing in."),
		settings.WithOrder(5),
	)
)

// Category: Token Lifetimes

var (
	// SettingTokenTTLSeconds controls the access token lifetime in seconds.
	SettingTokenTTLSeconds = settings.Define("session.token_ttl_seconds", 3600,
		settings.WithDisplayName("Access Token TTL (seconds)"),
		settings.WithDescription("Lifetime of access tokens in seconds"),
		settings.WithCategory("Token Lifetimes"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(60), Max: intPtr(86400)}),
		settings.WithHelpText("How long access tokens remain valid. Default: 3600 (1 hour)"),
		settings.WithOrder(10),
		settings.WithValidation(validateTokenTTL),
	)

	// SettingRefreshTokenTTLSeconds controls the refresh token lifetime in seconds.
	SettingRefreshTokenTTLSeconds = settings.Define("session.refresh_token_ttl_seconds", 2592000,
		settings.WithDisplayName("Refresh Token TTL (seconds)"),
		settings.WithDescription("Lifetime of refresh tokens in seconds"),
		settings.WithCategory("Token Lifetimes"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(3600), Max: intPtr(7776000)}),
		settings.WithHelpText("How long refresh tokens remain valid. Default: 2592000 (30 days)"),
		settings.WithOrder(20),
		settings.WithValidation(validateRefreshTokenTTL),
	)
)

// Category: Session Behavior

var (
	// SettingRotateRefreshToken controls whether refresh operations issue a new refresh token.
	SettingRotateRefreshToken = settings.Define("session.rotate_refresh_token", true,
		settings.WithDisplayName("Rotate Refresh Token"),
		settings.WithDescription("Issue a new refresh token on each refresh, invalidating the old one"),
		settings.WithCategory("Session Behavior"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, each token refresh issues a new refresh token. Prevents replay attacks."),
		settings.WithOrder(30),
	)

	// SettingBindToIP controls whether sessions are locked to the originating IP.
	SettingBindToIP = settings.Define("session.bind_to_ip", false,
		settings.WithDisplayName("Bind Session to IP"),
		settings.WithDescription("Reject requests from a different IP than the one that created the session"),
		settings.WithCategory("Session Behavior"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, sessions are locked to the original client IP address"),
		settings.WithOrder(40),
	)

	// SettingBindToDevice controls whether sessions are locked to the originating device.
	SettingBindToDevice = settings.Define("session.bind_to_device", false,
		settings.WithDisplayName("Bind Session to Device"),
		settings.WithDescription("Reject requests from a different device/user-agent than the one that created the session"),
		settings.WithCategory("Session Behavior"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, sessions are locked to the original device fingerprint or user-agent"),
		settings.WithOrder(50),
	)

	// SettingCleanupIntervalSeconds controls how often expired sessions are cleaned up.
	SettingCleanupIntervalSeconds = settings.Define("session.cleanup_interval_seconds", 3600,
		settings.WithDisplayName("Cleanup Interval (seconds)"),
		settings.WithDescription("Interval between expired session cleanup runs"),
		settings.WithCategory("Session Behavior"),
		settings.WithScopes(settings.ScopeGlobal),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(60), Max: intPtr(86400)}),
		settings.WithHelpText("How often the engine removes expired sessions. Default: 3600 (1 hour)"),
		settings.WithOrder(60),
	)
)

// Category: Multi-Session

var (
	// SettingMultiSessionEnabled controls whether users can have multiple concurrent sessions.
	SettingMultiSessionEnabled = settings.Define("session.multi_session_enabled", true,
		settings.WithDisplayName("Allow Multiple Sessions"),
		settings.WithDescription("Whether users can have multiple concurrent sessions"),
		settings.WithCategory("Multi-Session"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When disabled, signing in from a new device revokes all existing sessions first"),
		settings.WithOrder(70),
	)

	// SettingMaxActiveSessions controls the maximum number of concurrent sessions per user.
	SettingMaxActiveSessions = settings.Define("session.max_active_sessions", 0,
		settings.WithDisplayName("Maximum Active Sessions"),
		settings.WithDescription("Maximum concurrent sessions per user. 0 means unlimited."),
		settings.WithCategory("Multi-Session"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(0), Max: intPtr(100)}),
		settings.WithHelpText("When set, oldest sessions are evicted when the limit is reached. 0 = unlimited."),
		settings.WithOrder(80),
		settings.WithVisibleWhen("session.multi_session_enabled", true),
	)
)

// Category: Auto-Refresh

var (
	// SettingAutoRefreshEnabled controls whether near-expiry tokens are auto-refreshed in middleware.
	SettingAutoRefreshEnabled = settings.Define("session.auto_refresh_enabled", true,
		settings.WithDisplayName("Auto-Refresh Tokens"),
		settings.WithDescription("Automatically refresh near-expiry access tokens in middleware"),
		settings.WithCategory("Auto-Refresh"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, the middleware transparently refreshes access tokens approaching expiry and returns new tokens in response headers"),
		settings.WithOrder(90),
	)

	// SettingAutoRefreshThresholdSeconds controls the time window before expiry to trigger auto-refresh.
	SettingAutoRefreshThresholdSeconds = settings.Define("session.auto_refresh_threshold_seconds", 300,
		settings.WithDisplayName("Auto-Refresh Threshold (seconds)"),
		settings.WithDescription("Time before expiry to trigger auto-refresh"),
		settings.WithCategory("Auto-Refresh"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(30), Max: intPtr(3600)}),
		settings.WithHelpText("Access tokens within this many seconds of expiry will be auto-refreshed. Default: 300 (5 minutes)"),
		settings.WithOrder(100),
		settings.WithVisibleWhen("session.auto_refresh_enabled", true),
	)

	// SettingAutoRefreshExposeRefreshToken controls whether the refresh token is
	// included in the X-Auth-Refresh-Token header during auto-refresh.
	SettingAutoRefreshExposeRefreshToken = settings.Define("session.auto_refresh_expose_refresh_token", false,
		settings.WithDisplayName("Expose Refresh Token in Auto-Refresh"),
		settings.WithDescription("Include refresh token in X-Auth-Refresh-Token header during auto-refresh"),
		settings.WithCategory("Auto-Refresh"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("When disabled (recommended), only the access token is returned in auto-refresh headers. The refresh token should only be obtained via the /v1/refresh endpoint."),
		settings.WithOrder(101),
		settings.WithVisibleWhen("session.auto_refresh_enabled", true),
	)
)

// Category: JWT Security

var (
	// SettingJWTRequireActiveSession controls whether JWT tokens are cross-checked
	// against the session store to enable immediate revocation.
	SettingJWTRequireActiveSession = settings.Define("session.jwt_require_active_session", false,
		settings.WithDisplayName("Require Active Session for JWT"),
		settings.WithDescription("Cross-check JWT tokens against the session store to enable revocation"),
		settings.WithCategory("JWT Security"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, JWT tokens are validated against the session store on each request. This adds a DB lookup but enables instant revocation and IP/device binding for JWT tokens."),
		settings.WithOrder(55),
	)
)

// Category: Session Extension

var (
	// SettingExtendOnActivity controls whether sessions are extended on each authenticated request.
	SettingExtendOnActivity = settings.Define("session.extend_on_activity", true,
		settings.WithDisplayName("Extend Session on Activity"),
		settings.WithDescription("Automatically extend session expiry on each authenticated request"),
		settings.WithCategory("Session Extension"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, each authenticated request resets the session expiry timer, implementing a sliding session window"),
		settings.WithOrder(105),
	)

	// SettingInactivityTimeoutSeconds controls how long a session lives without activity.
	SettingInactivityTimeoutSeconds = settings.Define("session.inactivity_timeout_seconds", 604800,
		settings.WithDisplayName("Inactivity Timeout (seconds)"),
		settings.WithDescription("Session expires after this many seconds of inactivity"),
		settings.WithCategory("Session Extension"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Required: true, Min: intPtr(60), Max: intPtr(2592000)}),
		settings.WithHelpText("The session expiry is reset to now + this value on each request. Default: 604800 (7 days)"),
		settings.WithOrder(106),
		settings.WithVisibleWhen("session.extend_on_activity", true),
	)
)

// Category: Cookie Configuration

var (
	// SettingCookieName controls the name of the httpOnly session cookie.
	SettingCookieName = settings.Define("session.cookie_name", "authsome_session_token",
		settings.WithDisplayName("Session Cookie Name"),
		settings.WithDescription("Name of the httpOnly session cookie set on sign-in"),
		settings.WithCategory("Cookie Configuration"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldText),
		settings.WithHelpText("The cookie name used to store the session token. Default: authsome_session_token"),
		settings.WithOrder(110),
	)

	// SettingCookieDomain controls the Domain attribute of the session cookie.
	SettingCookieDomain = settings.Define("session.cookie_domain", "",
		settings.WithDisplayName("Cookie Domain"),
		settings.WithDescription("Domain attribute for the session cookie. Empty means exact host only."),
		settings.WithCategory("Cookie Configuration"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldText),
		settings.WithHelpText("Leave empty to scope cookie to the exact host. Set to '.example.com' for cross-subdomain sharing."),
		settings.WithOrder(120),
	)

	// SettingCookiePath controls the Path attribute of the session cookie.
	SettingCookiePath = settings.Define("session.cookie_path", "/",
		settings.WithDisplayName("Cookie Path"),
		settings.WithDescription("Path attribute for the session cookie"),
		settings.WithCategory("Cookie Configuration"),
		settings.WithScopes(settings.ScopeGlobal),
		settings.WithInputType(formconfig.FieldText),
		settings.WithHelpText("URL path scope for the cookie. Default: / (all paths)"),
		settings.WithOrder(130),
	)

	// SettingCookieSecure controls whether the Secure flag is set on the session cookie.
	SettingCookieSecure = settings.Define("session.cookie_secure", true,
		settings.WithDisplayName("Secure Cookie"),
		settings.WithDescription("Only send cookie over HTTPS connections"),
		settings.WithCategory("Cookie Configuration"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, cookies are only sent over HTTPS. Automatically disabled for HTTP in development."),
		settings.WithOrder(140),
	)

	// SettingCookieHTTPOnly controls whether the HttpOnly flag is set on the session cookie.
	SettingCookieHTTPOnly = settings.Define("session.cookie_http_only", true,
		settings.WithDisplayName("HttpOnly Cookie"),
		settings.WithDescription("Prevent JavaScript from accessing the session cookie"),
		settings.WithCategory("Cookie Configuration"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithHelpText("When enabled, cookies are inaccessible to document.cookie (XSS protection). Strongly recommended."),
		settings.WithOrder(150),
	)

	// SettingCookieSameSite controls the SameSite attribute of the session cookie.
	SettingCookieSameSite = settings.Define("session.cookie_same_site", "lax",
		settings.WithDisplayName("SameSite Policy"),
		settings.WithDescription("SameSite attribute for cross-site request control"),
		settings.WithCategory("Cookie Configuration"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldSelect),
		settings.WithOptions(
			formconfig.SelectOption{Value: "strict", Label: "Strict"},
			formconfig.SelectOption{Value: "lax", Label: "Lax (Recommended)"},
			formconfig.SelectOption{Value: "none", Label: "None"},
		),
		settings.WithHelpText("Lax allows top-level navigations (OAuth redirects). Strict blocks all cross-site. None requires Secure=true."),
		settings.WithOrder(160),
	)
)

// registerCoreSessionSettings registers all core session settings under the "session" namespace.
func registerCoreSessionSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "auth", SettingRequireEmailVerification); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingTokenTTLSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingRefreshTokenTTLSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingRotateRefreshToken); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingBindToIP); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingBindToDevice); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingCleanupIntervalSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingMultiSessionEnabled); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingMaxActiveSessions); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingAutoRefreshEnabled); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingAutoRefreshThresholdSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingAutoRefreshExposeRefreshToken); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingJWTRequireActiveSession); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingExtendOnActivity); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingInactivityTimeoutSeconds); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingCookieName); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingCookieDomain); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingCookiePath); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingCookieSecure); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "session", SettingCookieHTTPOnly); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "session", SettingCookieSameSite)
}

// intPtr returns a pointer to the given int.
func intPtr(v int) *int { return &v }

// ──────────────────────────────────────────────────
// Validators
// ──────────────────────────────────────────────────

func validateTokenTTL(val json.RawMessage) error {
	var v int
	if err := json.Unmarshal(val, &v); err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	if v < 60 || v > 86400 {
		return fmt.Errorf("token TTL must be between 60 and 86400 seconds")
	}
	return nil
}

func validateRefreshTokenTTL(val json.RawMessage) error {
	var v int
	if err := json.Unmarshal(val, &v); err != nil {
		return fmt.Errorf("invalid value: %w", err)
	}
	if v < 3600 || v > 7776000 {
		return fmt.Errorf("refresh token TTL must be between 3600 and 7776000 seconds")
	}
	return nil
}
