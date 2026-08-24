package agentauth

import (
	"context"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
)

// agentSessionTTL bounds an agent access token. Short by design: the refresh
// path re-checks the grant, so a short session is what makes revocation and
// expiry take effect quickly.
const agentSessionTTL = 15 * time.Minute

// IssueAgentSession mints a session for an agent acting under grant.
//
// Issuance deliberately goes through the engine's normal store and hook path
// so BeforeSessionCreate and AfterSignIn fire. That is what puts agent
// traffic in front of riskengine, impossibletravel, ipreputation and the
// rest. The API key plugin builds a synthetic session by hand and fires
// neither, which is why its traffic is invisible to all of them. Do not
// copy that shape here.
func (p *Plugin) IssueAgentSession(ctx context.Context, grant *AgentGrant) (*session.Session, error) {
	now := time.Now()
	if !grant.IsActive(now) {
		return nil, ErrGrantInactive
	}

	expires := now.Add(agentSessionTTL)
	if expires.After(grant.ExpiresAt) {
		// A session must never outlive the grant that authorized it. Without
		// this clamp, a short-lived grant close to expiry would still mint a
		// full-length session, and revoking or letting the grant lapse would
		// not actually end the agent's access until the session separately
		// expired.
		expires = grant.ExpiresAt
	}

	sess := &session.Session{
		ID:            id.NewSessionID(),
		AppID:         grant.AppID,
		UserID:        grant.UserID,
		OrgID:         grant.OrgID,
		PrincipalKind: session.PrincipalKindAgent,
		AgentID:       grant.AgentID,
		// GrantID must always be set: the postgres CHECK constraint
		// authsome_sessions_principal_check requires grant_id <> '' for an
		// agent-principal row, and the authorization core in middleware.go
		// refuses to honor a session whose grant does not match the
		// session's UserID, AgentID and AppID. Leaving it unset would mint a
		// session that passes on sqlite/mongo/memory (no such constraint)
		// but fails outright on postgres, and would be unusable everywhere
		// regardless.
		GrantID:   grant.ID,
		ExpiresAt: expires,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// The Emit* methods live on *plugin.Registry, reached through
	// engine.Plugins(). engine.Hooks() returns a *hook.Bus, which is a
	// different type and has no Emit methods on it.
	reg := p.engine.Plugins()

	// BeforeSessionCreate returns an error and can veto. riskengine uses it
	// to block a session outright when the score is high enough.
	if err := reg.EmitBeforeSessionCreate(ctx, sess); err != nil {
		return nil, fmt.Errorf("agentauth: before session create: %w", err)
	}
	if err := p.engine.Store().CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("agentauth: create session: %w", err)
	}

	u, err := p.engine.GetUser(ctx, grant.UserID)
	if err != nil {
		return nil, fmt.Errorf("agentauth: resolve delegating user: %w", err)
	}
	// EmitAfterSignIn returns nothing. Notification hooks are fire-and-forget
	// by design, so there is no error to handle here.
	reg.EmitAfterSignIn(ctx, u, sess)

	stamp := now
	grant.LastUsedAt = &stamp
	if err := p.store.UpdateAgentGrant(ctx, grant); err != nil {
		p.logger.Warn("agentauth: could not stamp grant last-used", log.Error(err))
	}

	return sess, nil
}
