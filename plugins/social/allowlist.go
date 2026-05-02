package social

import (
	"context"
	"net/url"
	"strings"

	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
)

// SettingAllowedFrontendURLs is the per-app CSV of origin hosts the social
// plugin trusts as redirect/frontend authorities. Empty means absolute
// redirects are rejected outright.
//
// Match policy (see isAllowedOrigin):
//   - Comparison is case-insensitive on host.
//   - Port matters: "https://app.example" does NOT match "https://app.example:8080".
//   - Path/query/fragment of the candidate are ignored.
//   - Schemes other than http/https are rejected.
var SettingAllowedFrontendURLs = settings.Define("auth.allowed_frontend_urls", "",
	settings.WithDisplayName("Allowed Frontend URLs"),
	settings.WithDescription("Comma-separated origins (scheme://host[:port]) that may be used as frontend_url or redirect_url targets."),
	settings.WithCategory("Authentication"),
	settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
	settings.WithInputType(formconfig.FieldText),
	settings.WithPlaceholder("https://app.example,https://staging.app.example"),
)

// isAllowedOrigin reports whether candidate's host is on the configured
// allowlist for appID (cascading from global to app scope).
//
// candidate must be an absolute http(s) URL. The check ignores
// path/query/fragment — only scheme://host[:port] is compared. Empty
// allowlist or empty/invalid candidate always rejects.
func isAllowedOrigin(ctx context.Context, mgr *settings.Manager, appID id.AppID, candidate string) bool {
	if mgr == nil || candidate == "" {
		return false
	}
	candParsed, err := url.Parse(candidate)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(candParsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return false
	}
	if candParsed.Host == "" {
		return false
	}

	csv, err := settings.Get(ctx, mgr, SettingAllowedFrontendURLs, settings.ResolveOpts{AppID: appID.String()})
	if err != nil {
		return false
	}
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return false
	}

	candHost := strings.ToLower(candParsed.Host)
	for _, entry := range strings.Split(csv, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		entryParsed, err := url.Parse(entry)
		if err != nil || entryParsed.Host == "" {
			continue
		}
		if strings.EqualFold(entryParsed.Host, candHost) {
			return true
		}
	}
	return false
}

// selectTrustedOrigin picks the trust authority used by handleStart for
// validating redirect_url. The order is:
//
//  1. Caller-supplied frontend_url, IF its host is on the allowlist.
//  2. Origin / Referer header, IF its host is on the allowlist.
//
// Returns "" when neither passes — falling back to relative-redirect-only
// behaviour in sanitizeRedirectURL.
//
// Factored out of handleStart so it can be exercised by unit tests
// without spinning up a forge.Context.
func selectTrustedOrigin(ctx context.Context, mgr *settings.Manager, appID id.AppID, frontendURL, originHeader, refererHeader string) string {
	safeFrontend := sanitizeFrontendURL(frontendURL)
	if safeFrontend != "" && isAllowedOrigin(ctx, mgr, appID, safeFrontend) {
		return safeFrontend
	}

	candidate := originHeader
	if candidate == "" {
		candidate = refererHeader
	}
	if candidate != "" && isAllowedOrigin(ctx, mgr, appID, candidate) {
		return candidate
	}
	return ""
}
