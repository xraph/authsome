// Package browserbind ties a browser-driven OAuth or SSO ceremony to the
// browser that started it.
//
// The state parameter alone proves that a callback belongs to some login the
// server minted, not that it belongs to the browser presenting it. A cookie
// set when the login starts, carrying a digest of that state, closes the gap:
// the callback must arrive with the cookie that matches its state, which an
// attacker who only holds a login URL cannot supply.
package browserbind

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

// CookieName is the state-binding cookie's name on https requests. The
// __Host- prefix makes browsers refuse it unless it is Secure, has no Domain
// and has Path=/ , so a subdomain cannot plant one.
const CookieName = "__Host-authsome_oauth_state"

// InsecureCookieName is the name used when the request is plain http, where
// browsers reject a __Host- cookie outright. Development only.
const InsecureCookieName = "authsome_oauth_state"

// TTL matches the ceremony state's own lifetime.
const TTL = 10 * time.Minute

// IsBrowser reports whether the request came from a web browser, which
// always attaches fetch metadata or an Origin to a script-initiated request.
// A native client sends neither and is bound by PKCE instead.
func IsBrowser(r *http.Request) bool {
	if r == nil {
		return false
	}
	return r.Header.Get("Sec-Fetch-Site") != "" || r.Header.Get("Origin") != "" || r.Header.Get("Sec-Fetch-Mode") != ""
}

// IsHTTPS reports whether the request reached the server over TLS, directly
// or through a proxy that says so.
func IsHTTPS(r *http.Request) bool {
	return r != nil && (r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"))
}

// Digest is the cookie value for a state: its hex SHA-256. The state itself
// travels in URLs, so the cookie holds something derived from it rather than
// the state, and the callback compares digests in constant time.
func Digest(state string) string {
	sum := sha256.Sum256([]byte(state))
	return hex.EncodeToString(sum[:])
}

// SetStateCookie writes the binding cookie for state on w. Over https it is
// the __Host- cookie with SameSite=None, which a cross-site POST such as a
// SAML assertion still carries; over plain http it is the unprefixed, Lax
// variant, since browsers refuse __Host- without Secure.
func SetStateCookie(w http.ResponseWriter, r *http.Request, state string) {
	c := &http.Cookie{
		Name:     InsecureCookieName,
		Value:    Digest(state),
		Path:     "/",
		MaxAge:   int(TTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if IsHTTPS(r) {
		c.Name = CookieName
		c.Secure = true
		c.SameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, c)
}

// ClearStateCookie expires the binding cookie once the ceremony completed.
//
// It is built the way SetStateCookie builds the cookie it clears: the
// plain-http variant first, then the __Host- name and Secure over https.
func ClearStateCookie(w http.ResponseWriter, r *http.Request) {
	c := &http.Cookie{
		Name:     InsecureCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if IsHTTPS(r) {
		c.Name = CookieName
		c.Secure = true
	}
	http.SetCookie(w, c)
}

// Matches reports whether r carries the binding cookie for state.
func Matches(r *http.Request, state string) bool {
	if r == nil || state == "" {
		return false
	}
	want := Digest(state)
	for _, name := range []string{CookieName, InsecureCookieName} {
		if c, err := r.Cookie(name); err == nil && c != nil {
			if subtle.ConstantTimeCompare([]byte(c.Value), []byte(want)) == 1 {
				return true
			}
		}
	}
	return false
}
