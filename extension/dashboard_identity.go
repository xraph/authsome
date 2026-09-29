package extension

// The dashboard identity: what authsome puts on dashauth.UserInfo, and so on
// every contract request's Principal, once a session has been resolved.
//
// Forge builds each request's principal with contract.PrincipalFor(user),
// which copies UserInfo.Claims onto Principal.Claims, and its transport
// checks an intent's `requires` predicate against UserInfo.Roles and
// UserInfo.Scopes. Before this file existed authsome filled in none of the
// three, so every contributor that scoped by app fell back to a default and
// every intent with a `requires` predicate was denied to everyone.
//
// Claims
//
//	app_id  Always set. The session's app, or the app the operator picked
//	        with apps.switch when the session may act there (see
//	        activeApp). Chronicle refuses a principal without it, because
//	        an empty app matches every app in its stores.
//	env_id  Set when there is one. The environment picked with
//	        environments.switch when it belongs to the resolved app,
//	        otherwise the session's own environment while the operator is
//	        still in the session's app.
//	org_id  Set only when the session carries an organization and the
//	        operator is in the session's app. An org belongs to one app, so
//	        it is dropped on a switch rather than carried somewhere it
//	        means nothing. Chronicle reads it as its tenant dimension.
//
// "tenant_id" is deliberately NOT set. The two consumers that read it
// disagree about what it is: Chronicle treats it as the tenant inside an app
// (authsome records audit events with TenantID set to the org), while Warden
// treats it as its own tenant, which in authsome is the app. Any single value
// is wrong for one of them. org_id says what it means.
//
// Roles
//
// The role slugs stamped onto the session at issuance (sess.Roles), the same
// strings authprovider.SessionProvider hands forge for route-level role
// checks. Stamped roles are restamped when an assignment changes, so they
// follow grants and revocations. A predicate declares them as "role:<slug>",
// for example "role:platform-admin".
//
// Scopes
//
// Extensions gate commands with "scope:<contributor>.admin", where
// <contributor> is the name in their contract manifest (Chronicle declares
// "scope:chronicle.admin"). An operator gets that scope for every contributor
// registered with the dashboard when they hold an admin role: platform-owner,
// platform-admin, owner or admin. Scopes are relative to the resolved app,
// because the app_id claim is what bounds every query the scope unlocks.
// Dashboard.RoleScopes in the extension config adds explicit grants on top,
// for a role that is not an admin but should reach one extension's gated
// intents.

import (
	"context"
	"net/http"
	"slices"
	"sync"

	log "github.com/xraph/go-utils/log"

	dashauth "github.com/xraph/forge/extensions/dashboard/auth"
	dashcontract "github.com/xraph/forge/extensions/dashboard/contract"

	authsome "github.com/xraph/authsome"
	authcontract "github.com/xraph/authsome/extension/contract"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// Claim keys authsome writes onto the dashboard identity.
const (
	claimAppID = "app_id"
	claimEnvID = "env_id"
	claimOrgID = "org_id"
)

// adminScopeSuffix is appended to a contributor name to form its admin scope.
const adminScopeSuffix = ".admin"

// dashboardAdminRoles are the role slugs that earn "<contributor>.admin" for
// every contributor. The platform pair are cross-app roles on the platform
// app; owner and admin are the per-app pair every app is bootstrapped with.
var dashboardAdminRoles = []string{
	rbac.PlatformOwnerSlug,
	rbac.PlatformAdminSlug,
	rbac.AppOwnerSlug,
	rbac.AppAdminSlug,
}

// crossAppRoles are the role slugs that let a platform-app session act in an
// app other than its own. platform-user is left out on purpose: it is the
// role every platform sign-up receives by default, so counting it would let
// anyone who can register switch into any tenant.
var crossAppRoles = []string{
	rbac.PlatformOwnerSlug,
	rbac.PlatformAdminSlug,
}

// dashboardScopes derives UserInfo.Scopes from the operator's roles.
type dashboardScopes struct {
	// contributors lists the contract contributor names registered with the
	// dashboard. Read per request so contributors registered after authsome
	// (including remote ones) are covered. Nil means no admin scopes.
	contributors func() []string

	// roleScopes is Dashboard.RoleScopes: explicit grants keyed by role slug.
	roleScopes map[string][]string
}

// forRoles returns the sorted, de-duplicated scopes the given roles confer.
func (s dashboardScopes) forRoles(roles []string) []string {
	set := make(map[string]struct{})

	if s.contributors != nil && hasAnyRole(roles, dashboardAdminRoles) {
		for _, name := range s.contributors() {
			if name != "" {
				set[name+adminScopeSuffix] = struct{}{}
			}
		}
	}

	for _, role := range roles {
		for _, scope := range s.roleScopes[role] {
			if scope != "" {
				set[scope] = struct{}{}
			}
		}
	}

	if len(set) == 0 {
		return nil
	}

	out := make([]string, 0, len(set))
	for scope := range set {
		out = append(out, scope)
	}
	slices.Sort(out)
	return out
}

// contributorSource hands the dashboard's contract registry to the auth
// checker. The dashboard calls RegisterDashboardAuth and
// RegisterContractContributor separately, and not necessarily in that order,
// so the checker holds this and reads it at request time.
type contributorSource struct {
	mu  sync.RWMutex
	reg dashcontract.Registry
}

func (s *contributorSource) set(reg dashcontract.Registry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reg = reg
}

// names returns the name of every registered contract contributor.
func (s *contributorSource) names() []string {
	s.mu.RLock()
	reg := s.reg
	s.mu.RUnlock()
	if reg == nil {
		return nil
	}

	manifests := reg.All()
	out := make([]string, 0, len(manifests))
	for _, m := range manifests {
		if m != nil {
			out = append(out, m.Contributor.Name)
		}
	}
	return out
}

// dashboardIdentity builds the identity for a resolved session and its user.
func (c *authChecker) dashboardIdentity(ctx context.Context, r *http.Request, sess *session.Session, u *user.User) *dashauth.UserInfo {
	info := userToUserInfo(u)

	appID := c.activeApp(ctx, r, sess)
	claims := make(map[string]any, 3)
	if !appID.IsNil() {
		claims[claimAppID] = appID.String()
	}
	if envID := c.activeEnv(ctx, r, sess, appID); !envID.IsNil() {
		claims[claimEnvID] = envID.String()
	}
	// An org only means something inside the app it belongs to, so org_id
	// never goes out without app_id. Chronicle refuses that shape anyway.
	if !appID.IsNil() && appID == sess.AppID && !sess.OrgID.IsNil() {
		claims[claimOrgID] = sess.OrgID.String()
	}

	info.Claims = claims
	info.Roles = slices.Clone(sess.Roles)
	info.Scopes = c.scopes.forRoles(sess.Roles)
	return info
}

// activeApp decides which app the dashboard acts in for this request.
//
// The session's own app unless the authsome_app cookie names a different one
// AND the session may act there. The cookie is written by apps.switch, but
// it is only a cookie: anyone can set it to anything, so it is treated as a
// request the session has to be entitled to. When it is not, the answer is
// the session's app, never the cookie's, so a forged or stale cookie yields
// exactly the scope the operator had without it.
func (c *authChecker) activeApp(ctx context.Context, r *http.Request, sess *session.Session) id.AppID {
	target, ok := cookieID(r, authcontract.AppSwitcherCookie, id.ParseAppID)
	if !ok || target == sess.AppID {
		return sess.AppID
	}

	if !mayActAcrossApps(sess, platformAppID(c.engine)) {
		c.engine.Logger().Debug("authsome: dashboard app switch ignored; session may not act outside its app",
			log.String("session_id", sess.ID.String()),
			log.String("requested_app_id", target.String()),
		)
		return sess.AppID
	}

	// The operator is entitled, but the app has to still exist. A deleted
	// app's ID would scope every query to nothing at best.
	if _, err := c.engine.GetApp(ctx, target); err != nil {
		return sess.AppID
	}
	return target
}

// mayActAcrossApps reports whether a session may act in apps other than its
// own: it must belong to the platform app and hold a cross-app role there.
//
// The roles are the ones stamped on the session, the same ones the
// route-level SessionProvider enforces. A role assignment change restamps
// live sessions, so a revoked platform-admin loses the switch on their next
// request rather than at the next sign-in.
func mayActAcrossApps(sess *session.Session, platform id.AppID) bool {
	if platform.IsNil() || sess.AppID != platform {
		return false
	}
	return hasAnyRole(sess.Roles, crossAppRoles)
}

// activeEnv decides the environment for this request, given the app
// activeApp settled on. The authsome_env cookie wins only when it names an
// environment of that app. Otherwise the session's environment applies, but
// only while the operator is still in the session's app: it means nothing in
// another one.
func (c *authChecker) activeEnv(ctx context.Context, r *http.Request, sess *session.Session, appID id.AppID) id.EnvironmentID {
	if envID, ok := cookieID(r, authcontract.EnvSwitcherCookie, id.ParseEnvironmentID); ok {
		if env, err := c.engine.GetEnvironment(ctx, envID); err == nil && env != nil && env.AppID == appID {
			return envID
		}
	}
	if appID == sess.AppID {
		return sess.EnvID
	}
	return id.EnvironmentID{}
}

// cookieID reads a switcher cookie and parses it with parse. ok is false when
// the cookie is absent, empty, or does not parse.
func cookieID(r *http.Request, name string, parse func(string) (id.ID, error)) (id.ID, bool) {
	if r == nil {
		return id.ID{}, false
	}
	ck, err := r.Cookie(name)
	if err != nil || ck.Value == "" {
		return id.ID{}, false
	}
	parsed, err := parse(ck.Value)
	if err != nil || parsed.IsNil() {
		return id.ID{}, false
	}
	return parsed, true
}

// platformAppID is the engine's platform app, falling back to the configured
// AppID when bootstrap has not run (the same resolution defaultAppID uses).
func platformAppID(eng *authsome.Engine) id.AppID {
	if platformID := eng.PlatformAppID(); !platformID.IsNil() {
		return platformID
	}
	appID, _ := id.ParseAppID(eng.Config().AppID) //nolint:errcheck // best-effort parse
	return appID
}

func hasAnyRole(held, wanted []string) bool {
	for _, role := range held {
		if slices.Contains(wanted, role) {
			return true
		}
	}
	return false
}
