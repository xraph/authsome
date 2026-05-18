package authsome

import (
	"context"
	"net/http"
	"strings"

	"github.com/xraph/authsome/settings"
)

// hostPrefix is the RFC 6265bis prefix that locks a cookie to the
// originating host: a browser will only accept it when Secure=true,
// Domain is unset, and Path=/. Used by SettingCookieUseHostPrefix.
const hostPrefix = "__Host-"

// SessionCookieTemplate resolves the dynamic cookie configuration for a
// given app and returns a *http.Cookie pre-populated with Name, Path,
// Domain, Secure, HttpOnly, and SameSite. The caller fills in Value and
// MaxAge.
//
// Behaviour:
//
//   - All five base attributes (name, domain, path, secure, http_only,
//     same_site) come from the dynamic settings cascade — same as before.
//   - When SettingCookieUseHostPrefix resolves to true, the cookie name
//     is prefixed with "__Host-" (idempotent if already present), Domain
//     is forced to empty, Path is forced to "/", and Secure is forced
//     to true regardless of isHTTPS. These are the browser's
//     prerequisites for accepting a __Host-prefixed cookie; relaxing
//     any of them silently breaks session persistence.
//   - When the prefix is disabled, isHTTPS gates the Secure attribute
//     (so dev HTTP doesn't break) and Domain is whatever the setting
//     resolves to.
//
// Pass appID="" to resolve global-only.
func SessionCookieTemplate(ctx context.Context, mgr *settings.Manager, appID string, isHTTPS bool) *http.Cookie {
	resolveOpts := settings.ResolveOpts{}
	if appID != "" {
		resolveOpts.AppID = appID
	}

	name, _ := settings.Get(ctx, mgr, SettingCookieName, resolveOpts) //nolint:errcheck // best-effort
	if strings.TrimSpace(name) == "" {
		name = "authsome_session_token"
	}
	domain, _ := settings.Get(ctx, mgr, SettingCookieDomain, resolveOpts)   //nolint:errcheck // best-effort
	cookiePath, _ := settings.Get(ctx, mgr, SettingCookiePath, resolveOpts) //nolint:errcheck // best-effort
	if cookiePath == "" {
		cookiePath = "/"
	}
	secureSetting, _ := settings.Get(ctx, mgr, SettingCookieSecure, resolveOpts)        //nolint:errcheck // best-effort
	httpOnly, _ := settings.Get(ctx, mgr, SettingCookieHTTPOnly, resolveOpts)           //nolint:errcheck // best-effort
	sameSiteStr, _ := settings.Get(ctx, mgr, SettingCookieSameSite, resolveOpts)        //nolint:errcheck // best-effort
	useHostPrefix, _ := settings.Get(ctx, mgr, SettingCookieUseHostPrefix, resolveOpts) //nolint:errcheck // best-effort

	secure := secureSetting && isHTTPS

	if useHostPrefix {
		// Browser requirements for __Host- cookies. Force them rather than
		// silently fail at runtime — a misconfigured __Host- cookie is
		// silently dropped by every modern browser.
		if !strings.HasPrefix(name, hostPrefix) {
			name = hostPrefix + name
		}
		domain = ""
		cookiePath = "/"
		secure = true
	}

	sameSite := http.SameSiteLaxMode
	switch strings.ToLower(strings.TrimSpace(sameSiteStr)) {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}

	cookie := &http.Cookie{ // #nosec G124 -- secure/httpOnly/sameSite resolved dynamically from settings
		Name:     name,
		Path:     cookiePath,
		Domain:   domain,
		HttpOnly: httpOnly,
		Secure:   secure,
		SameSite: sameSite,
	}
	return cookie
}
