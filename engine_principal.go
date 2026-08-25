package authsome

import (
	"context"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"
	"github.com/xraph/warden"

	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
)

// wardenSubjectKind maps a principal kind onto a Warden subject kind.
//
// All three non-human kinds collapse onto SubjectServiceAcct so role
// assignments live in one namespace and an operator grants a role once rather
// than once per kind. The finer kind rides along in the subject attributes, so
// an ABAC policy can still tell an agent from a CI workload.
func wardenSubjectKind(k principal.Kind) warden.SubjectKind {
	if k == principal.KindUser || k == "" {
		return warden.SubjectUser
	}
	return warden.SubjectServiceAcct
}

// wardenSubject builds the Warden subject for ref.
//
// onBehalfOf is set when ref is an actor rather than the subject, so a policy
// can deny an agent an action it would allow the same agent calling for
// itself.
//
// WARNING: warden's cache key (cache/memory.go) is built from
// tenant:kind:id:action:resource:resID:namespace and does not include
// Subject.Attributes. That means on_behalf_of and actor_kind, set below, do
// not participate in cache keying. If a warden cache is ever configured, two
// checks for the same agent that differ only in on_behalf_of hash to the SAME
// cache entry: agent X checked "on behalf of user A" and agent X checked
// "on behalf of user B" collide, and whichever runs second is served the
// first's verdict rather than being evaluated against its own ABAC policy.
// Concretely, an ALLOW computed for one user gets silently reused for a
// different user the same agent is acting for. This is inert today because
// authsome configures no warden cache and ships no ABAC policy keyed on
// on_behalf_of, but it is armed the moment either changes. Fixing it means
// changing warden's cache key, which is out of this package's reach.
func wardenSubject(ref principal.Ref, onBehalfOf *principal.Ref) warden.Subject {
	attrs := map[string]any{"principal_kind": string(ref.Kind)}
	if onBehalfOf != nil {
		attrs["on_behalf_of"] = onBehalfOf.String()
		attrs["actor_kind"] = string(ref.Kind)
	}
	return warden.Subject{
		Kind:       wardenSubjectKind(ref.Kind),
		ID:         ref.ID,
		Attributes: attrs,
	}
}

// Can reports whether subject may perform action on resource, given that
// actors are acting on the subject's behalf.
//
// With no actors this is a single Warden check and behaves exactly as
// HasPermission always has. With actors, every party must allow: the subject,
// and each hop of the chain. Delegation can only narrow. The first denial
// short-circuits, so a denied agent costs one check rather than the whole
// chain.
//
// Impersonation does not reach here with a populated chain. Session.AuthzActors
// returns nil for it, because impersonating somebody is the request to
// evaluate as them.
func (e *Engine) Can(
	ctx context.Context, subject principal.Ref, actors principal.Chain, action, resource string,
) (bool, error) {
	ctx = e.ensureWardenScope(ctx)

	result, err := e.checkOne(ctx, wardenSubject(subject, nil), action, resource)
	if err != nil {
		return false, err
	}
	if !result.Allowed {
		if len(actors) > 0 {
			// Warn, not Debug: with a non-empty chain this is no longer
			// HasPermission's own nil-chain call site, the only place whose
			// wrapper already logs a subject-only denial at Warn. Nothing
			// else logs this one, so a delegated permission denial would be
			// invisible at default log level if it stayed at Debug. This is
			// exactly the traffic an operator most wants to see.
			e.logger.Warn("authsome: Can denied by subject",
				log.String("subject", subject.String()),
				log.String("action", action),
				log.String("resource", resource),
				log.String("decision", string(result.Decision)),
				log.String("reason", result.Reason),
			)
		} else {
			// Debug, not Warn: a subject-only denial with no actor chain is
			// exactly the case HasPermission's own wrapper already logs at
			// Warn with tenant and scope diagnostics attached, and
			// permission checks are a high-volume hot path. Doubling every
			// denial into two Warn lines would double the operational log
			// cost for no new information at the subject-only call site.
			e.logger.Debug("authsome: Can denied by subject",
				log.String("subject", subject.String()),
				log.String("action", action),
				log.String("resource", resource),
				log.String("decision", string(result.Decision)),
				log.String("reason", result.Reason),
			)
		}
		return false, nil
	}

	for _, actor := range actors {
		actorResult, actorErr := e.checkOne(ctx, wardenSubject(actor, &subject), action, resource)
		if actorErr != nil {
			return false, actorErr
		}
		if !actorResult.Allowed {
			e.logger.Warn("authsome: Can denied by actor",
				log.String("subject", subject.String()),
				log.String("actor", actor.String()),
				log.String("action", action),
				log.String("resource", resource),
				log.String("decision", string(actorResult.Decision)),
				log.String("reason", actorResult.Reason),
			)
			return false, nil
		}
	}

	return true, nil
}

// checkOne runs a single Warden check and returns the full result so callers
// can log the decision and reason for whichever hop denied, not only the
// subject's.
func (e *Engine) checkOne(
	ctx context.Context, sub warden.Subject, action, resource string,
) (*warden.CheckResult, error) {
	return e.wardenEng.Check(ctx, &warden.CheckRequest{
		Subject:  sub,
		Action:   warden.Action{Name: action},
		Resource: warden.Resource{Type: resource},
	})
}

// ResolvePrincipal resolves any caller by ref.
func (e *Engine) ResolvePrincipal(ctx context.Context, ref principal.Ref) (*principal.Principal, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}
	p, err := e.store.GetPrincipal(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("authsome: resolve principal %s: %w", ref, err)
	}
	return p, nil
}

// PrincipalStore returns the principal and delegation store.
func (e *Engine) PrincipalStore() principal.Store { return e.store }

// GrantDelegation records that actor may act for subject.
//
// grantedBy is who consented and is recorded for audit. The caller is
// responsible for having checked that grantedBy is entitled to consent: for an
// ordinary delegation that means grantedBy is the subject or an admin over it,
// and the API layer enforces it before calling here.
func (e *Engine) GrantDelegation(
	ctx context.Context, appID id.AppID, actor, subject principal.Ref,
	scopes []string, grantedBy principal.Ref, expiresAt *time.Time,
) (*principal.Delegation, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}
	if appID.IsNil() {
		return nil, fmt.Errorf("authsome: app_id is required")
	}
	if actor.IsZero() || subject.IsZero() {
		return nil, fmt.Errorf("authsome: actor and subject are required")
	}
	if actor == subject {
		// Acting for yourself is not a delegation, and storing one would put a
		// chain on a session that has no second principal on it.
		return nil, fmt.Errorf("authsome: a principal cannot be delegated to itself")
	}

	now := time.Now()
	d := &principal.Delegation{
		ID:        id.NewDelegationID(),
		AppID:     appID,
		Actor:     actor,
		Subject:   subject,
		GrantKind: principal.GrantDelegation,
		Scopes:    scopes,
		GrantedBy: grantedBy,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := e.store.CreateDelegation(ctx, d); err != nil {
		return nil, fmt.Errorf("authsome: grant delegation: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionDelegationGrant,
		Resource:   hook.ResourceSession,
		ResourceID: d.ID.String(),
		ActorID:    grantedBy.ID,
		Tenant:     appID.String(),
		Metadata: map[string]string{
			"actor":   actor.String(),
			"subject": subject.String(),
		},
	})
	return d, nil
}

// RevokeDelegation ends a grant.
func (e *Engine) RevokeDelegation(ctx context.Context, delID id.DelegationID) error {
	if err := e.requireStarted(); err != nil {
		return err
	}
	// Read first so the hook event can carry the actor/subject the grant
	// named. RevokeDelegation itself only takes the ID: the caller may be
	// revoking a grant by ID alone (e.g. from a listing) without having
	// either ref in hand. A grant already gone is not fatal here — the
	// revoke below still runs and its own not-found is what the caller sees.
	d, getErr := e.store.GetDelegation(ctx, delID)

	if err := e.store.RevokeDelegation(ctx, delID, time.Now()); err != nil {
		return fmt.Errorf("authsome: revoke delegation: %w", err)
	}

	evt := &hook.Event{
		Action:     hook.ActionDelegationRevoke,
		Resource:   hook.ResourceSession,
		ResourceID: delID.String(),
	}
	if getErr == nil && d != nil {
		evt.Tenant = d.AppID.String()
		evt.Metadata = map[string]string{
			"actor":   d.Actor.String(),
			"subject": d.Subject.String(),
		}
	}
	e.hooks.Emit(ctx, evt)
	return nil
}

// ListDelegationsForSubject returns what may act for subject, so a person can
// see and revoke the agents holding authority over their account.
func (e *Engine) ListDelegationsForSubject(
	ctx context.Context, appID id.AppID, subject principal.Ref,
) ([]*principal.Delegation, error) {
	if err := e.requireStarted(); err != nil {
		return nil, err
	}
	return e.store.ListDelegations(ctx, &principal.DelegationQuery{
		AppID: appID, Subject: &subject, ActiveOnly: true, ActiveAsOf: time.Now(),
	})
}
