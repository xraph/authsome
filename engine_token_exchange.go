package authsome

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/tokenformat"
)

// ErrExchangeRefused wraps every reason ExchangeToken declines to mint a
// session: no live grant, a wrong actor, a disabled actor, a scope the grant
// or the actor's own scopes do not carry, or a risk-plugin denial. All of it
// collapses to the same refusal from the caller's point of view, and to the
// same HTTP 403 at the API layer: this endpoint never creates authority, so
// every path that is not "the grant said yes" ends here rather than at a
// distinct error for each reason.
var ErrExchangeRefused = errors.New("authsome: exchange refused")

// ExchangeRequest asks for a session in which Actor acts for RequestedSubject.
type ExchangeRequest struct {
	AppID            id.AppID
	Actor            principal.Ref
	RequestedSubject principal.Ref
	Scopes           []string
	IPAddress        string
	UserAgent        string
	// CredentialID identifies the credential the actor authenticated with, so
	// the risk verdict caches against it.
	CredentialID string

	// CallerActors is the actor chain already stamped on the session or
	// credential presenting this request, if any. Non-empty means Actor is
	// not presenting its own root credential: it already holds a session
	// minted by a prior exchange or by impersonation. See the refusal this
	// guards in ExchangeToken.
	CallerActors principal.Chain
	// CallerDelegationID is the grant the caller's own session was minted
	// against, if any. Checked alongside CallerActors so a session that
	// somehow carries a delegation id without a populated chain is still
	// caught.
	CallerDelegationID id.DelegationID
}

// ExchangeToken mints a session in which the actor acts on the subject's
// behalf, against a grant that already exists.
//
// It never creates authority. Without a live grant matching (app, actor,
// subject) this fails, which is what makes revocation meaningful.
//
// Unlike per-request API-key auth, this does not run through the cached
// principal-auth gate (plugin_principalauth.go). Exchange is rare and mints a
// durable session rather than being called on every request, so the cache
// buys nothing here and its staleness would matter more than the extra
// lookup this costs instead.
func (e *Engine) ExchangeToken(ctx context.Context, req *ExchangeRequest) (*session.Session, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}
	if req.AppID.IsNil() {
		return nil, fmt.Errorf("authsome: exchange: app_id is required")
	}
	if req.Actor.IsZero() || req.RequestedSubject.IsZero() {
		return nil, fmt.Errorf("authsome: exchange: actor and subject are required")
	}

	// Refuse a chained exchange outright. Left unchecked: agent A holds a
	// repo:read grant from Alice, exchanges it for an Alice session, then
	// presents that Alice session back to this same endpoint. A second
	// exchange would land a session acting for whoever Alice herself holds
	// a grant over (Bob, say) even though Bob never named the agent, with
	// Alice's own scope filter gone entirely, and (because Actors below is
	// assigned rather than appended) the agent erased from the resulting
	// session's chain: the audit trail would read as a user acting for a
	// user, no agent anywhere in it. That is exactly the escalation
	// "delegation can only narrow" exists to prevent, so it is refused
	// rather than fixed by appending-and-re-verifying every hop: multi-hop
	// is served instead by task 15's ephemeral children, which mint under a
	// registered parent with scopes capped to a subset and expiry capped by
	// the parent's.
	if len(req.CallerActors) > 0 || !req.CallerDelegationID.IsNil() {
		return nil, fmt.Errorf(
			"authsome: exchange: caller already holds a delegated or impersonated session; chained exchange refused: %w",
			ErrExchangeRefused)
	}

	grant, err := e.store.FindActiveDelegation(
		ctx, req.AppID, req.Actor, req.RequestedSubject, principal.GrantDelegation)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: no active delegation for %s acting as %s: %w: %w",
			req.Actor, req.RequestedSubject, ErrExchangeRefused, err)
	}

	actorPrincipal, err := e.store.GetPrincipal(ctx, req.Actor)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: resolve actor: %w: %w", ErrExchangeRefused, err)
	}
	now := time.Now()
	if !actorPrincipal.IsActive(now) {
		return nil, fmt.Errorf("authsome: exchange: actor %s is disabled or expired: %w", req.Actor, ErrExchangeRefused)
	}

	// Scope narrowing. Refuse rather than silently drop: an agent that asked
	// for repo:write and got a session without it fails later, somewhere far
	// from the cause, and reads as a bug in the agent.
	scopes, err := intersectScopes(req.Scopes, grant, actorPrincipal.Scopes)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: %w: %w", ErrExchangeRefused, err)
	}

	att := &principal.AuthAttempt{
		Subject:        req.RequestedSubject,
		Actors:         principal.Chain{req.Actor},
		AppID:          req.AppID,
		CredentialKind: "token_exchange",
		CredentialID:   req.CredentialID,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		Ephemeral:      actorPrincipal.IsEphemeral(),
		At:             now,
	}
	if err := e.plugins.EmitBeforePrincipalAuth(ctx, att); err != nil {
		return nil, fmt.Errorf("authsome: exchange: %w: %w", ErrExchangeRefused, err)
	}

	userID, err := id.ParseUserID(req.RequestedSubject.ID)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: subject must be a user: %w", err)
	}

	cfg := e.sessionConfigForApp(ctx, req.AppID)
	// An exchanged token is meant to be re-minted, not held: use the
	// dedicated exchange TTL (account.SessionConfig.TokenExchangeTTL, default
	// 5m, per-app overridable) rather than the ordinary session TTL.
	if cfg.TokenExchangeTTL > 0 {
		cfg.TokenTTL = cfg.TokenExchangeTTL
		cfg.RefreshTokenTTL = cfg.TokenExchangeTTL
	}

	sess, err := e.newSession(req.AppID, userID, cfg)
	if err != nil {
		return nil, fmt.Errorf("authsome: exchange: create session: %w", err)
	}

	// The session must never outlive the grant that justified it. Clamped
	// against the exact time.Time the grant carries, rather than by
	// pre-shrinking cfg.TokenTTL to a "remaining" duration computed before
	// newSession's own time.Now(): two independent now() calls leave a few
	// microseconds of drift between them, and any positive drift here would
	// let the session's expiry land after the grant's, precisely the
	// outcome this exists to rule out.
	clamped := false
	if grant.ExpiresAt != nil {
		if sess.ExpiresAt.After(*grant.ExpiresAt) {
			sess.ExpiresAt = *grant.ExpiresAt
			clamped = true
		}
		if sess.RefreshTokenExpiresAt.After(*grant.ExpiresAt) {
			sess.RefreshTokenExpiresAt = *grant.ExpiresAt
		}
	}

	// newSession already minted a JWT (if the app is configured for one)
	// against the pre-clamp expiry. A clamp after the fact must be reflected
	// in the token's own exp claim too, or the bearer credential itself would
	// keep asserting a lifetime the grant no longer backs, regardless of what
	// the session row now says.
	if clamped {
		tokFmt := e.TokenFormatForApp(req.AppID.String())
		if tokFmt.Name() == "jwt" {
			jwtToken, genErr := tokFmt.GenerateAccessToken(tokenformat.TokenClaims{
				UserID:    userID.String(),
				AppID:     req.AppID.String(),
				SessionID: sess.ID.String(),
				IssuedAt:  sess.CreatedAt,
				ExpiresAt: sess.ExpiresAt,
			})
			if genErr != nil {
				return nil, fmt.Errorf("authsome: exchange: regenerate jwt: %w", genErr)
			}
			sess.Token = jwtToken
		}
	}

	sess.PrincipalKind = principal.KindUser
	sess.Actors = principal.Chain{req.Actor}
	sess.ActorGrant = principal.GrantDelegation
	sess.DelegationID = grant.ID
	sess.Scopes = scopes
	sess.IPAddress = req.IPAddress
	sess.UserAgent = req.UserAgent

	if err := e.store.CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("authsome: exchange: store session: %w", err)
	}

	e.plugins.EmitAfterPrincipalAuth(ctx, att, sess)
	// Mirrors plugin_principalauth.go's gate.Observe: EmitAfterPrincipalAuth
	// plus the global hook bus event, so Chronicle and any other bus
	// subscriber sees a token exchange exactly as it would see any other
	// machine authentication, even though this path bypasses the gate.
	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionPrincipalAuth,
		Resource:   hook.ResourceSession,
		ResourceID: sess.ID.String(),
		ActorID:    req.Actor.ID,
		Tenant:     req.AppID.String(),
		Metadata: map[string]string{
			"principal_kind":  string(req.Actor.Kind),
			"credential_kind": att.CredentialKind,
			"credential_id":   att.CredentialID,
			"ip":              req.IPAddress,
			"subject":         req.RequestedSubject.String(),
			"delegation_id":   grant.ID.String(),
		},
	})

	return sess, nil
}

// intersectScopes returns the scopes an exchanged session may carry.
//
// Every requested scope must be inside both the grant's filter and the
// actor's own scopes. Asking for one that is not is an error rather than a
// quiet removal: an agent that asked for repo:write and got a session
// without it fails later, far from the cause, and reads as a bug in the
// agent rather than as the refusal it actually is.
//
// An actor with no recorded scopes ([]string(nil) or empty) is read the same
// way principal.Delegation.Scopes reads empty: no restriction of its own,
// rather than "may hold nothing." That is a deliberate choice, not an
// oversight, made to match the grant side's existing convention
// (principal/delegation.go's AllowsScope) rather than introduce a second,
// asymmetric default. Most service accounts in this codebase are created
// without ever setting Scopes, and nothing outside plugins/oauth2provider
// enforces session.Scopes today, so treating an unset actor scope list as
// "may hold nothing" would silently refuse exchanges for callers that were
// never restricted in the first place. If a caller wants an actor capped to
// an empty scope set, it should record that explicitly rather than rely on
// the zero value meaning it.
func intersectScopes(requested []string, grant *principal.Delegation, actorScopes []string) ([]string, error) {
	if len(requested) == 0 {
		return nil, nil
	}
	actorHas := make(map[string]bool, len(actorScopes))
	for _, s := range actorScopes {
		actorHas[s] = true
	}
	out := make([]string, 0, len(requested))
	for _, s := range requested {
		if !grant.AllowsScope(s) {
			return nil, fmt.Errorf("scope %q is outside the delegation grant", s)
		}
		if len(actorScopes) > 0 && !actorHas[s] {
			return nil, fmt.Errorf("scope %q is outside the actor's own scopes", s)
		}
		out = append(out, s)
	}
	return out, nil
}
