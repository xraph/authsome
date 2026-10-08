package extension

import (
	"context"
	"net/http"
	"strings"

	log "github.com/xraph/go-utils/log"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/middleware"
	authclient "github.com/xraph/authsome/sdk/go"
	"github.com/xraph/authsome/user"
)

// dashboardCookieName is the cookie the dashboard's auth.login intent writes
// (see extension/contract). The checkers below read the same name.
const dashboardCookieName = "auth_token"

// authChecker implements dashauth.AuthChecker for the authsome extension.
type authChecker struct {
	engine *authsome.Engine
}

// Ensure authChecker implements AuthChecker at compile time.
var _ dashauth.AuthChecker = (*authChecker)(nil)

// CheckAuth inspects the request and returns a UserInfo if authenticated.
//
// The DPoP binding is enforced here rather than assumed to have been enforced
// upstream. Ordering in a middleware chain is not an invariant, and this
// checker gates the dashboard, which is the highest-privilege surface in the
// product. It has no way to write an RFC 9449 challenge, so a binding it
// cannot satisfy resolves to no identity at all, the same answer
// authprovider.SessionProvider gives for the same reason.
func (c *authChecker) CheckAuth(ctx context.Context, r *http.Request) (*dashauth.UserInfo, error) {
	scheme, token := middleware.ExtractCredentialFromContext(ctx, r, dashboardCookieName)
	if token == "" {
		return nil, nil
	}

	sess, err := c.engine.ResolveSessionByToken(ctx, token)
	if err != nil {
		return nil, nil
	}

	if dpopErr := middleware.EnforceDPoPForRequest(
		ctx, c.engine.DPoPBindingConfig(), r,
		sess.DPoPJKT, scheme, token,
		sess.AppID.String(), sess.ID.String(),
		c.engine.Logger(),
	); dpopErr != nil {
		return nil, nil
	}

	u, err := c.engine.ResolveUser(ctx, sess.UserID.String())
	if err != nil {
		return nil, nil
	}

	return userToUserInfo(u), nil
}

func userToUserInfo(u *user.User) *dashauth.UserInfo {
	return &dashauth.UserInfo{
		Subject:     u.ID.String(),
		DisplayName: u.Name(),
		Email:       u.Email,
		AvatarURL:   u.Image,
	}
}

// clientAuthChecker validates dashboard sessions by introspecting the auth
// token against the remote authsome service.
type clientAuthChecker struct {
	client *authclient.Client

	// binding carries the validator this service checks RFC 9449 bindings
	// with, shared with ClientAuthMiddleware so both consult one replay cache.
	binding middleware.SessionBindingConfig
	logger  log.Logger
}

var _ dashauth.AuthChecker = (*clientAuthChecker)(nil)

// CheckAuth resolves the dashboard identity by introspecting the token against
// the remote service.
//
// RFC 9449 section 7.3 reports the binding in the introspection response's cnf
// claim, and acting on it is this service's job: the identity server never
// sees this request and cannot check the proof on our behalf. Reading cnf and
// then discarding it would leave the dashboard, the highest-privilege surface
// here, admitting a bound token that arrived with no proof.
func (c *clientAuthChecker) CheckAuth(ctx context.Context, r *http.Request) (*dashauth.UserInfo, error) {
	scheme, token := middleware.ExtractCredentialFromContext(ctx, r, dashboardCookieName)
	if token == "" {
		return nil, nil
	}

	resp, err := c.client.Introspect(ctx, token)
	if err != nil || resp == nil || !resp.Active || resp.User == nil {
		return nil, nil
	}

	jkt := ""
	if resp.Cnf != nil {
		jkt = resp.Cnf.Jkt
	}
	if dpopErr := middleware.EnforceDPoPForRequest(
		ctx, c.binding, r,
		jkt, scheme, token,
		resp.AppID, resp.SessionID,
		c.logger,
	); dpopErr != nil {
		return nil, nil
	}

	display := strings.TrimSpace(resp.User.FirstName + " " + resp.User.LastName)
	if display == "" {
		display = resp.User.Email
	}

	return &dashauth.UserInfo{
		Subject:     resp.User.ID,
		DisplayName: display,
		Email:       resp.User.Email,
	}, nil
}

// extractToken reads a bearer token from the Authorization header, falling
// back to the dashboard's auth_token cookie.
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
	}

	if cookie, err := r.Cookie(dashboardCookieName); err == nil {
		return cookie.Value
	}

	return ""
}
