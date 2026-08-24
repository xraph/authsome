package agentauth

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	// ErrGrantInactive means the grant is missing, revoked or expired.
	ErrGrantInactive = errors.New("agentauth: grant is not active")
	// ErrNoPermissionChecker means the engine exposes no RBAC, so the user
	// gate cannot run and the request is denied.
	ErrNoPermissionChecker = errors.New("agentauth: no permission checker available")
)

// SetPermissionChecker injects the RBAC checker. OnInit does this from the
// engine; tests use it directly.
func (p *Plugin) SetPermissionChecker(pc plugin.PermissionChecker) { p.permChecker = pc }

// Authorize enforces the intersection: an agent may do something only if its
// grant confers it AND the delegating user can do it. A non-agent session
// passes straight through, since this plugin has no opinion on human traffic.
//
// The scope gate runs first. It is cheaper, a map lookup ahead of a Warden
// call, and it is the safe order: reporting a user-gate failure first would
// let an agent enumerate its owner's permissions one probe at a time.
func (p *Plugin) Authorize(ctx context.Context, sess *session.Session, action, resource string) error {
	if sess == nil || sess.PrincipalKind != session.PrincipalKindAgent {
		return nil
	}

	grant, err := p.store.GetAgentGrant(ctx, sess.GrantID)
	if errors.Is(err, ErrNotFound) {
		return ErrGrantInactive
	}
	if err != nil {
		return fmt.Errorf("agentauth: load grant: %w", err)
	}
	if !grant.IsActive(time.Now()) {
		return ErrGrantInactive
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
		return fmt.Errorf("agentauth: delegating user lacks %s on %s", action, resource)
	}
	return nil
}
