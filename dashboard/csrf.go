// Package dashboard CSRF helper for unauthenticated HTML forms.
//
// Signup, login, password-reset and setup all submit BEFORE a session
// exists, so we cannot bind a token to a session ID (as the in-session
// scoped-nonce helper does). Instead this implements the double-submit
// cookie pattern with a __Host- prefixed cookie:
//
//   - The browser refuses to set a __Host- cookie unless it has Secure,
//     no Domain attribute, and Path=/. Per RFC 6265bis §4.1.3.2 that
//     prevents a sibling subdomain from planting a value the verifier
//     would accept.
//   - SameSite=Strict means an attacker page on another origin cannot
//     cause the cookie to ride along on a top-level navigation POST.
//   - HttpOnly means JavaScript on the same origin cannot read or
//     overwrite it via document.cookie.
//   - Comparison is constant-time.
package dashboard

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

const (
	// FormCSRFCookieName is the __Host- prefixed cookie that carries the
	// pre-session CSRF token for unauthenticated HTML forms.
	FormCSRFCookieName = "__Host-authsome-csrf"

	// FormCSRFFormField is the hidden form field that mirrors the cookie
	// value back in the POST body.
	FormCSRFFormField = "csrf_token"

	formCSRFTokenBytes = 32
)

// GenerateFormCSRFToken creates a fresh random token, sets it as a __Host-
// cookie on w, and returns the value so the caller can embed it in a
// hidden form field named FormCSRFFormField.
func GenerateFormCSRFToken(w http.ResponseWriter) string {
	buf := make([]byte, formCSRFTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand.Read does not fail on supported platforms; if it
		// does we panic rather than emit a guessable token.
		panic("authsome: crypto/rand.Read failed: " + err.Error())
	}
	tok := base64.RawURLEncoding.EncodeToString(buf)

	http.SetCookie(w, &http.Cookie{
		Name:     FormCSRFCookieName,
		Value:    tok,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		// Domain intentionally left empty: __Host- requires no Domain attr.
	})
	return tok
}

// VerifyFormCSRFToken returns true iff r carries a FormCSRFCookieName
// cookie whose value is byte-equal to formValue. An empty formValue or a
// missing cookie is rejected. Comparison is constant-time.
func VerifyFormCSRFToken(r *http.Request, formValue string) bool {
	if formValue == "" {
		return false
	}
	c, err := r.Cookie(FormCSRFCookieName)
	if err != nil || c == nil || c.Value == "" {
		return false
	}
	a := []byte(c.Value)
	b := []byte(formValue)
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}
