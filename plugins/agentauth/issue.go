package agentauth

import (
	"context"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
)

// agentSessionTTL bounds an agent access token, and is a hard ceiling: nothing
// in the engine extends an agent session past it. Three separate guards
// enforce that, each closing a different route to the same failure (a
// session outliving the grant that authorized it):
//   - Engine.Refresh (service.go) refuses to rotate an agent-principal
//     session at all, so a refresh cannot renew one.
//   - roleStampingStore's shouldStamp/shouldRestamp (engine_session_roles.go)
//     never re-stamp an agent session, so even if rotation were ever allowed
//     it could not restore the delegating human's roles onto it.
//   - middleware.SessionActivityMiddleware (middleware/activity.go) skips
//     agent sessions entirely, so the sliding session window cannot reset
//     ExpiresAt to now + InactivityTimeout on ordinary request traffic.
//
// Short by design: with no renewal path of any kind, this TTL is genuinely
// the only thing standing between a revoked or expired grant and continued
// access, until agentauth grows its own grant-aware refresh.
const agentSessionTTL = 15 * time.Minute

// IssueMeta carries request-scoped metadata onto the minted session. Without
// it the risk plugins this task exists to inform have nothing to score:
// impossibletravel returns immediately on an empty IPAddress and geoip
// returns on a nil lookup, so the hooks would fire but teach those plugins
// nothing. There are no callers yet, so this shape is free to change.
type IssueMeta struct {
	IPAddress string
	UserAgent string
	DeviceID  id.DeviceID
}

// IssueAgentSession mints a session for an agent acting under grant.
//
// Issuance deliberately goes through the engine's normal store and hook path
// so BeforeSessionCreate and AfterSignIn fire. That is what puts agent
// traffic in front of riskengine, impossibletravel, ipreputation and the
// rest. The API key plugin builds a synthetic session by hand and fires
// neither, which is why its traffic is invisible to all of them. Do not
// copy that shape here.
//
// riskengine's own OnBeforeSessionCreate is a documented no-op today, so
// nothing actually vetoes an agent session through this hook yet. What this
// function guarantees is the mechanism: an emit that returns an error and
// aborts the mint before anything is persisted, so whichever plugin later
// implements the veto can rely on it working. See
// TestIssueAgentSession_VetoBlocksPersistence.
func (p *Plugin) IssueAgentSession(ctx context.Context, grant *AgentGrant, meta IssueMeta) (*session.Session, error) {
	if err := validateGrant(grant); err != nil {
		return nil, err
	}

	now := time.Now()
	if !grant.IsActive(now) {
		return nil, ErrGrantInactive
	}

	// Resolved before anything is written. A GetUser failure that happened
	// after the store write would leave a persisted session behind that the
	// caller believes never got created (it saw an error) and that
	// AfterSignIn never fires for — invisible to exactly the plugins this
	// task exists to inform.
	u, err := p.engine.GetUser(ctx, grant.UserID)
	if err != nil {
		return nil, fmt.Errorf("agentauth: resolve delegating user: %w", err)
	}

	ttl := agentSessionTTL
	if until := grant.ExpiresAt.Sub(now); until < ttl {
		// A session must never outlive the grant that authorized it. Without
		// this clamp, a short-lived grant close to expiry would still mint a
		// full-length session, and revoking or letting the grant lapse would
		// not actually end the agent's access until the session separately
		// expired.
		ttl = until
	}

	// Token and RefreshToken are generated exactly the way every other mint
	// path generates them: account.NewSession, the same helper
	// engine_issue_session.go and the oauth2provider plugin call. Both
	// columns carry a non-partial unique index on postgres and sqlite,
	// so a session hand-built with empty strings for both inserts fine
	// once and then collides with "authsome: conflict" on the second.
	sess, err := account.NewSession(grant.AppID, grant.UserID, account.SessionConfig{
		TokenTTL:        ttl,
		RefreshTokenTTL: ttl,
	})
	if err != nil {
		return nil, fmt.Errorf("agentauth: generate session tokens: %w", err)
	}
	// account.NewSession takes its own time.Now() snapshot to add the ttl
	// to, a few instructions after ours. Re-clamping against grant.ExpiresAt
	// directly (rather than trusting the duration alone) keeps "never
	// outlives the grant" true regardless of that skew.
	if sess.ExpiresAt.After(grant.ExpiresAt) {
		sess.ExpiresAt = grant.ExpiresAt
	}
	if sess.RefreshTokenExpiresAt.After(grant.ExpiresAt) {
		sess.RefreshTokenExpiresAt = grant.ExpiresAt
	}

	sess.OrgID = grant.OrgID
	sess.PrincipalKind = session.PrincipalKindAgent
	sess.AgentID = grant.AgentID
	// GrantID must always be set: the postgres CHECK constraint
	// authsome_sessions_principal_check requires grant_id <> '' for an
	// agent-principal row, and the authorization core in middleware.go
	// refuses to honor a session whose grant does not match the session's
	// UserID, AgentID and AppID. Leaving it unset would mint a session that
	// passes on sqlite/mongo/memory (no such constraint) but fails outright
	// on postgres, and would be unusable everywhere regardless.
	sess.GrantID = grant.ID
	sess.IPAddress = meta.IPAddress
	sess.UserAgent = meta.UserAgent
	sess.DeviceID = meta.DeviceID

	// authsome_sessions.env_id is NOT NULL with a foreign key to
	// authsome_environments, and a zero id.EnvironmentID stringifies to "",
	// which fails that FK on the very first insert on postgres (sqlite and
	// mongo have no such constraint). Best-effort, matching the pattern in
	// engine_issue_session.go and plugins/oauth2provider/plugin.go: an app
	// with no default environment configured should not block issuance.
	if env, envErr := p.engine.Store().GetDefaultEnvironment(ctx, grant.AppID); envErr == nil && env != nil {
		sess.EnvID = env.ID
	}

	// The Emit* methods live on *plugin.Registry, reached through
	// engine.Plugins(). engine.Hooks() returns a *hook.Bus, which is a
	// different type and has no Emit methods on it.
	reg := p.engine.Plugins()

	// BeforeSessionCreate returns an error and can veto: an implementer can
	// abort the mint before anything is persisted. This must stay ahead of
	// the store write below — moving it after would let a vetoed session
	// persist anyway, which TestIssueAgentSession_VetoBlocksPersistence
	// guards against.
	if err := reg.EmitBeforeSessionCreate(ctx, sess); err != nil {
		return nil, fmt.Errorf("agentauth: before session create: %w", err)
	}
	if err := p.engine.Store().CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("agentauth: create session: %w", err)
	}

	reg.EmitAfterSessionCreate(ctx, sess)
	// EmitAfterSignIn returns nothing. Notification hooks are fire-and-forget
	// by design, so there is no error to handle here.
	reg.EmitAfterSignIn(ctx, u, sess)

	// A delegated-agent credential minted with no audit trail is the wrong
	// default for this feature specifically: it is the record that lets an
	// operator answer "what did this agent do, and who let it" after the
	// fact. Mirrors the shape Engine.IssueSession writes in
	// engine_issue_session.go. Best-effort: an audit-log outage must not
	// block issuance, the same trade-off engine.audit already makes.
	if p.chronicle != nil {
		if aerr := p.chronicle.Record(ctx, &bridge.AuditEvent{
			Action:     "issue_agent_session",
			Resource:   "session",
			ResourceID: sess.ID.String(),
			ActorID:    grant.UserID.String(),
			Tenant:     grant.AppID.String(),
			Outcome:    bridge.OutcomeSuccess,
			Severity:   bridge.SeverityInfo,
			Category:   "auth",
			Metadata: map[string]string{
				"agent_id": grant.AgentID.String(),
				"grant_id": grant.ID.String(),
			},
		}); aerr != nil {
			p.logger.Warn("agentauth: audit record failed", log.Error(aerr))
		}
	}

	// StampLastUsed, not UpdateAgentGrant: UpdateAgentGrant replaces the
	// whole row from this function's local grant, which was read before the
	// session write above and can be stale by the time this runs. A revoke
	// landing in that window would have its RevokedAt undone the moment this
	// stale copy was written back — see the Store interface doc comment on
	// StampLastUsed. A targeted update that only ever mentions
	// last_used_at/updated_at cannot do that, because it never reads or
	// writes RevokedAt at all. The caller's grant is only mutated once the
	// write actually succeeds, so it never claims a stamp that wasn't
	// persisted.
	if err := p.store.StampLastUsed(ctx, grant.ID, now); err != nil {
		p.logger.Warn("agentauth: could not stamp grant last-used", log.Error(err))
	} else {
		stamp := now
		grant.LastUsedAt = &stamp
		grant.UpdatedAt = now
	}

	return sess, nil
}

// validateGrant rejects a caller-supplied grant this package cannot safely
// mint a session from, before anything about it is dereferenced or trusted.
// Task 8's ruling was that Authorize must not trust the issuer; the
// symmetric obligation is that the issuer validates what it is handed.
// IssueAgentSession is exported, so a nil grant, or a grant missing one of
// the ids Authorize binds sessions to (UserID, AgentID, AppID), is exactly
// the input a careless or malicious caller can supply — and grant.IsActive
// would panic on a nil grant before any of this ran.
func validateGrant(grant *AgentGrant) error {
	if grant == nil {
		return ErrGrantInactive
	}
	if grant.UserID.IsNil() || grant.AgentID.IsNil() || grant.AppID.IsNil() {
		return ErrGrantInactive
	}
	return nil
}
