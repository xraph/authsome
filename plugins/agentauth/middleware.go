package agentauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
)

// Sentinel errors so callers can map to the right HTTP response without
// string matching.
var (
	// ErrInsufficientScope means the grant does not confer the permission the
	// route requires. Safe to report to the agent: it names only what the
	// agent was granted, which the agent already knows.
	ErrInsufficientScope = errors.New("agentauth: insufficient scope")
	// ErrGrantInactive means the grant is missing, revoked, expired, does not
	// belong to this session's principal, or the session itself is malformed
	// in a way that makes "is this an active agent grant" unanswerable.
	ErrGrantInactive = errors.New("agentauth: grant is not active")
	// ErrNoPermissionChecker means the engine exposes no RBAC, so the user
	// gate cannot run and the request is denied.
	ErrNoPermissionChecker = errors.New("agentauth: no permission checker available")
	// ErrUserNotPermitted means the grant and scope were fine but the
	// delegating human cannot do this. Distinct from ErrInsufficientScope so
	// callers can map both without string matching.
	ErrUserNotPermitted = errors.New("agentauth: delegating user lacks permission")
)

// SetPermissionChecker injects the RBAC checker. OnInit does this from the
// engine; tests use it directly.
func (p *Plugin) SetPermissionChecker(pc plugin.PermissionChecker) { p.permChecker = pc }

// SetLogger injects the logger AuthorizeHTTP's default branch uses to record
// an authorization check failure that isn't one of Authorize's named
// sentinels (a store or RBAC transport error). OnInit does this from the
// engine; tests use it directly, mirroring SetPermissionChecker above.
func (p *Plugin) SetLogger(l log.Logger) { p.logger = l }

// Authorize enforces the intersection: an agent may do something only if its
// grant confers it AND the delegating user can do it. A non-agent session
// passes straight through, since this plugin has no opinion on human traffic.
//
// The scope gate runs first. It is cheaper, a map lookup ahead of a Warden
// call, and it is the safe order: reporting a user-gate failure first would
// let an agent enumerate its owner's permissions one probe at a time.
func (p *Plugin) Authorize(ctx context.Context, sess *session.Session, action, resource string) error {
	// A missing session carries no permission, symmetrically with a missing
	// permission checker further down: both are "the thing needed to decide
	// is absent", and both must deny rather than default to allow.
	if sess == nil {
		return ErrGrantInactive
	}

	if sess.PrincipalKind != session.PrincipalKindAgent {
		// A session that carries agent markers (AgentID or GrantID) but
		// claims a different, or absent, PrincipalKind is internally
		// inconsistent. Recognizing "is this an agent" by PrincipalKind
		// alone means any failure to stamp it correctly is a silent full
		// bypass of both gates below, not a degradation — so a mismatch
		// here must deny, not pass through.
		if !sess.AgentID.IsNil() || !sess.GrantID.IsNil() {
			return ErrGrantInactive
		}
		return nil
	}

	grant, ok := p.cache.get(sess.GrantID)
	if !ok {
		// Captured before the store read, not after: this is what lets put
		// tell whether a RevokeGrant landed while the read was in flight, so
		// that revocation is never resurrected by this request's own write.
		gen := p.cache.generation()
		loaded, err := p.store.GetAgentGrant(ctx, sess.GrantID)
		if errors.Is(err, ErrNotFound) {
			return ErrGrantInactive
		}
		if err != nil {
			return fmt.Errorf("agentauth: load grant: %w", err)
		}
		p.cache.put(loaded, gen)
		grant = loaded
	}
	if !grant.IsActive(time.Now()) {
		return ErrGrantInactive
	}

	// The grant named by sess.GrantID must actually belong to this
	// session's principal. Without this check, Authorize decides against
	// whatever grant sess.GrantID happens to name — scoped and permission
	// checked against that grant's user — regardless of which user, agent
	// or app the rest of the session claims to be. That is a confused
	// deputy: the authorization decision and the execution identity would
	// disagree. Compared with .String(), not ==, because id.UserID and
	// friends are type aliases over a typeid and the codebase compares ids
	// by string throughout (see CreateGrant's own IsNil+prefix guard for the
	// same reasoning). grant.UserID.IsNil() is also rejected here, mirroring
	// the invariant CreateGrant already enforces: a grant with no delegating
	// user must never be treated as authorizing anyone.
	if grant.UserID.IsNil() ||
		grant.UserID.String() != sess.UserID.String() ||
		grant.AgentID.String() != sess.AgentID.String() ||
		grant.AppID.String() != sess.AppID.String() {
		return ErrGrantInactive
	}

	// An empty action or resource cannot be a real authorization question.
	// Without this, a scope misregistered as Grants("", "") would match any
	// route call that (by bug, not by intent) supplies neither.
	if action == "" || resource == "" {
		return ErrInsufficientScope
	}

	if !p.scopes.Covers(grant.Scopes, action, resource) {
		return ErrInsufficientScope
	}

	// Fail closed. plugin.PermissionGuard returns nil here on purpose so an
	// RBAC-less engine still enforces sessions. For an agent that fallback is
	// a hole, because the user gate is the security model.
	if p.permChecker == nil {
		return ErrNoPermissionChecker
	}

	allowed, err := p.permChecker.HasPermission(ctx, grant.UserID, action, resource)
	if err != nil {
		return fmt.Errorf("agentauth: permission check: %w", err)
	}
	if !allowed {
		return fmt.Errorf("%w: %s on %s", ErrUserNotPermitted, action, resource)
	}
	return nil
}

// Guard returns middleware enforcing the agent intersection for a route that
// requires action on resource. It resolves the session with
// middleware.SessionFrom — the whole *session.Session, not just a user id,
// since Authorize needs PrincipalKind, AgentID and GrantID off it, which is
// why this is not middleware/rbac.go's UserIDFrom (rbac.go's two middlewares
// only ever need the id). It then delegates the decision entirely to
// AuthorizeHTTP: a human session (or a request AuthorizeHTTP otherwise has no
// opinion on) passes through untouched, and every denial gets the response
// AuthorizeHTTP built for it, headers included, since forge's own error
// handling would otherwise drop them on the floor.
// denyAgentPrincipal refuses any request whose resolved session is
// agent-principal, before the wrapped handler ever runs.
//
// agentauth's own admin and /me routes (RegisterRoutes) are guarded by
// plugin.SessionGuard and plugin.AdminGuard, both of which resolve to
// middleware.RequirePermission — checking a Warden permission against
// sess.UserID with no awareness of PrincipalKind at all. An agent session
// carries its delegating human's UserID by design (see session.go), so an
// agent belonging to an org admin would otherwise reach these routes with
// that admin's full permission and no granted scope required for any of
// it: PUT /admin/agents/policy could set the agent's own governing org to
// open with unrestricted scopes, PATCH /admin/agents/:id/status could
// unblock a blocked agent, and DELETE /me/agents/:id could revoke a
// competing grant. This plugin ships the intersection guard (Guard, above)
// for every route a host app protects; it must not leave its own policy
// surface outside that same discipline.
//
// A missing session is not this middleware's business — SessionGuard (or
// its absence, in a minimal test wiring) already governs whether an
// unauthenticated request reaches this far.
func denyAgentPrincipal() forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			if sess, ok := middleware.SessionFrom(ctx.Context()); ok && sess != nil &&
				sess.PrincipalKind == session.PrincipalKindAgent {
				return forge.Forbidden("agent sessions may not access this endpoint")
			}
			return next(ctx)
		}
	}
}

func (p *Plugin) Guard(action, resource string) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			sess, ok := middleware.SessionFrom(ctx.Context())
			if !ok {
				// No session in context at all is a different condition
				// from a grant that existed and is now revoked or expired.
				// Authorize's own nil-session branch maps both to
				// ErrGrantInactive — correct for Authorize's other callers,
				// who already know whether they resolved a session — but
				// routing an anonymous request through AuthorizeHTTP here
				// would render error_description="agent grant revoked or
				// expired" for a caller that never named a grant, sending
				// an agent developer hunting a revocation that never
				// happened. middleware/rbac.go:30-32 says "authentication
				// required" for the same missing-identity condition;
				// matching that here instead.
				return forge.Unauthorized("authentication required")
			}

			err := p.AuthorizeHTTP(ctx.Context(), sess, action, resource)
			if err == nil {
				return next(ctx)
			}

			var he *httpError
			if !errors.As(err, &he) {
				return err
			}
			for name, value := range he.headers {
				ctx.Response().Header().Set(name, value)
			}
			return ctx.JSON(he.status, he.ResponseBody())
		}
	}
}
