package social

import (
	"net/url"
	"strings"
)

// sanitizeRedirectURL validates a post-auth redirect URL.
//
// Policy:
//   - Empty input → empty (caller falls back to a safe default).
//   - Relative paths (no host) → returned as-is (always safe; resolved
//     against the page that loads them).
//   - Absolute URLs → require trustedOrigin to match host (case-insensitive).
//     If trustedOrigin is empty, absolute URLs are REJECTED. The previous
//     behaviour (allow when origin unknown) was an open redirect.
//   - Scheme must be http or https; embedded credentials reject.
//
// trustedOrigin should be the per-app frontend allowlist match (preferred),
// or the request's Origin/Referer header as a last resort, NOT raw caller
// input. Phase 1.2 (Task 3) wires the allowlist source.
func sanitizeRedirectURL(rawURL, trustedOrigin string) string {
	if rawURL == "" {
		return ""
	}
	// Reject backslashes outright. Browsers normalise '\' to '/' in
	// Location headers, so "\\evil.example" becomes "//evil.example" — a
	// protocol-relative redirect to attacker territory.
	if strings.ContainsRune(rawURL, '\\') {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "" && scheme != "http" && scheme != "https" {
		return ""
	}
	if parsed.User != nil {
		return ""
	}
	if parsed.Host == "" {
		// No host: require a clean rooted path. Reject scheme-only
		// (e.g. "http:") and non-rooted relatives (e.g. "foo/bar").
		if scheme != "" || !strings.HasPrefix(rawURL, "/") {
			return ""
		}
		return rawURL
	}
	if trustedOrigin == "" {
		return ""
	}
	originParsed, oErr := url.Parse(trustedOrigin)
	if oErr != nil || !strings.EqualFold(parsed.Host, originParsed.Host) {
		return ""
	}
	return rawURL
}

// sanitizeFrontendURL requires an absolute http(s) URL with no embedded
// credentials. Used both to anchor redirect_url validation and as a fallback
// redirect target on auth failure.
func sanitizeFrontendURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	if strings.ContainsRune(rawURL, '\\') {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	if parsed.Host == "" || parsed.User != nil {
		return ""
	}
	return rawURL
}
