package authsome

import (
	"context"
	"net/http"
	"strings"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/settings"
)

// TrustedDeviceHeader carries a trusted-device token on a sign-in request
// from a client that cannot hold the cookie (a native app, a server-side
// proxy). It is read before the cookie.
const TrustedDeviceHeader = "X-Authsome-Trusted-Device"

// TrustedDeviceCookieName is the httpOnly cookie the mfa plugin sets when a
// user asks to trust the browser they just passed a second factor on.
const TrustedDeviceCookieName = "authsome_trusted_device"

// TrustedDeviceToken returns the trusted-device token a request carries: the
// X-Authsome-Trusted-Device header if set, otherwise the
// authsome_trusted_device cookie. Empty when it carries neither.
//
// Every sign-in path hands this to IssueSessionRequest.TrustedDeviceToken.
// The token is only a claim; the mfa plugin decides whether it is live and
// belongs to the user signing in.
func TrustedDeviceToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := strings.TrimSpace(r.Header.Get(TrustedDeviceHeader)); v != "" {
		return v
	}
	if c, err := r.Cookie(TrustedDeviceCookieName); err == nil {
		return strings.TrimSpace(c.Value)
	}
	return ""
}

// SetTrustedDeviceCookie writes the trusted-device cookie: httpOnly, Path /,
// SameSite Lax, host-only, and Secure exactly when the session cookie would
// be, so a dev server on plain HTTP keeps working and production never sends
// it in the clear. A token that has already expired writes nothing.
func SetTrustedDeviceCookie(w http.ResponseWriter, r *http.Request, mgr *settings.Manager, appID, token string, expiresAt time.Time) {
	if w == nil || r == nil || token == "" {
		return
	}
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge <= 0 {
		return
	}
	secure := middleware.IsHTTPS(r)
	if mgr != nil {
		secure = SessionCookieTemplate(r.Context(), mgr, appID, middleware.IsHTTPS(r)).Secure
	}
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- HttpOnly and SameSite fixed; Secure follows the session cookie so plain-HTTP dev works
		Name:     TrustedDeviceCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearTrustedDeviceCookie expires the trusted-device cookie, for when the
// device the request came from is no longer trusted.
func ClearTrustedDeviceCookie(w http.ResponseWriter, r *http.Request) {
	if w == nil || r == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- an expiring cookie carries no value
		Name:     TrustedDeviceCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   middleware.IsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// trustedDeviceVerifier is the part of the mfa plugin the gate asks whether a
// trusted-device token is live for the user signing in. verifiedAt is when the
// device passed the second factor that earned the trust.
type trustedDeviceVerifier interface {
	VerifyTrustedDevice(ctx context.Context, appID id.AppID, userID id.UserID, token string) (verifiedAt time.Time, ok bool, err error)
}

// trustedDeviceVerifiedAt reports when the device presenting token passed a
// second factor for userID, or the zero time when it is not a trusted device
// for them. A lookup failure counts as untrusted: the gate then asks for the
// second factor, which is the safe way to be wrong.
func (e *Engine) trustedDeviceVerifiedAt(ctx context.Context, appID id.AppID, userID id.UserID, token string) time.Time {
	if token == "" {
		return time.Time{}
	}
	for _, p := range e.plugins.Plugins() {
		v, ok := p.(trustedDeviceVerifier)
		if !ok {
			continue
		}
		at, trusted, err := v.VerifyTrustedDevice(ctx, appID, userID, token)
		if err != nil {
			e.logger.Warn("authsome: check trusted device",
				log.String("user_id", userID.String()),
				log.String("error", err.Error()),
			)
			return time.Time{}
		}
		if !trusted {
			return time.Time{}
		}
		if at.IsZero() {
			at = time.Now()
		}
		return at.UTC()
	}
	return time.Time{}
}
