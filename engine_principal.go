package authsome

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"
	"github.com/xraph/warden"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/serviceaccount"
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
// ordinary delegation that means grantedBy is the subject or an admin over it.
// There is no HTTP route for this yet, so today "the caller" means whatever
// engine-embedding code invokes this method directly; the entitlement check
// belongs there until a route exists to enforce it.
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
//
// appID scopes the lookup to the caller's own tenant: a delegation id from
// one app must not be revocable by a caller authenticated into another, so a
// mismatch reads as not-found rather than forbidden, the same way
// api/tenant_scope.go's assertAppScope treats a cross-tenant resource id.
//
// revokedBy must be the grant's own subject or actor. There is no support
// yet for an admin revoking a grant on someone else's behalf: that would
// need a Warden permission decision, and this codebase has no established
// action/resource convention for delegation management to check against
// yet. Inventing one here would be a guess independent routes could later
// disagree with. Add that check at this call site, once one exists, rather
// than routing an admin path around this method.
func (e *Engine) RevokeDelegation(
	ctx context.Context, appID id.AppID, revokedBy principal.Ref, delID id.DelegationID,
) error {
	if err := e.requireStarted(); err != nil {
		return err
	}

	d, err := e.store.GetDelegation(ctx, delID)
	if err != nil {
		return fmt.Errorf("authsome: revoke delegation: %w", err)
	}
	if d.AppID.String() != appID.String() {
		return principal.ErrNotFound
	}
	if revokedBy != d.Subject && revokedBy != d.Actor {
		return fmt.Errorf("authsome: revoke delegation: %s is not party to this grant", revokedBy)
	}

	if err := e.store.RevokeDelegation(ctx, delID, time.Now()); err != nil {
		return fmt.Errorf("authsome: revoke delegation: %w", err)
	}

	e.hooks.Emit(ctx, &hook.Event{
		Action:     hook.ActionDelegationRevoke,
		Resource:   hook.ResourceSession,
		ResourceID: delID.String(),
		ActorID:    revokedBy.ID,
		Tenant:     d.AppID.String(),
		Metadata: map[string]string{
			"actor":   d.Actor.String(),
			"subject": d.Subject.String(),
		},
	})
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

// ErrChildScopeExceedsParent is returned when a requested child scope falls
// outside the parent's own scopes. This is something the caller can fix by
// asking for less, so the API layer maps it to 400 rather than 403.
var ErrChildScopeExceedsParent = errors.New("authsome: mint child: requested scope exceeds the parent's own scopes")

// ErrChildMintNotPermitted is returned when the parent itself is not
// eligible to mint a child at all: it is inactive, or it is itself an
// ephemeral child (one level only). No amount of narrowing the request
// fixes either case, so the API layer maps it to 403 rather than 400.
var ErrChildMintNotPermitted = errors.New("authsome: mint child: parent is not permitted to mint children")

// MintChildPrincipal creates a short-lived principal under a registered
// parent, with its own API key.
//
// This is what makes per-task agents workable: one durable registration, N
// ephemeral instances, each with its own identity for attribution and its own
// credential to revoke. The two caps are what keep it from being an
// escalation: a child's scopes are a subset of its parent's, and a child's
// expiry never passes its parent's.
//
// ttl is not required to be positive. A caller normally passes a positive
// duration, but a zero or negative one is accepted rather than rejected: it
// mints a child that is already expired, which is exactly what
// ReapExpiredPrincipals needs to be provable against without waiting out a
// real clock. The HTTP handler is where a positive ttl is actually enforced
// for real callers.
//
// The secret is returned once and is not stored.
func (e *Engine) MintChildPrincipal(
	ctx context.Context, parentID id.ServiceAccountID, name string, scopes []string, ttl time.Duration,
) (*serviceaccount.ServiceAccount, *apikey.APIKey, string, error) {
	if err := e.requireStarted(); err != nil {
		return nil, nil, "", err
	}

	parent, err := e.store.GetServiceAccount(ctx, parentID)
	if err != nil {
		return nil, nil, "", fmt.Errorf("authsome: mint child: get parent: %w", err)
	}
	if !parent.Active {
		return nil, nil, "", fmt.Errorf("authsome: mint child: parent %s is inactive: %w", parentID, ErrChildMintNotPermitted)
	}
	if !parent.ParentID.IsNil() {
		// One level only. A tree of ephemeral principals is a revocation
		// problem nobody can reason about, and nothing needs it.
		return nil, nil, "", fmt.Errorf("authsome: mint child: %s is itself an ephemeral child and cannot mint children: %w",
			parentID, ErrChildMintNotPermitted)
	}

	// An empty child scope list inherits the parent's, it does not mean
	// "unrestricted". Empty scopes read as "no restriction" everywhere they
	// are consumed, so minting a child with no scopes under a parent capped
	// to repo:read would produce a child BROADER than the parent it hangs
	// off, which is the opposite of the subset cap this method exists to
	// enforce.
	if len(scopes) == 0 {
		scopes = append([]string(nil), parent.Scopes...)
	}

	if err := requireScopeSubset(scopes, parent.Scopes); err != nil {
		return nil, nil, "", fmt.Errorf("authsome: mint child: %w: %w", ErrChildScopeExceedsParent, err)
	}

	now := time.Now()
	expires := now.Add(ttl)
	if parent.ExpiresAt != nil && expires.After(*parent.ExpiresAt) {
		expires = *parent.ExpiresAt
	}

	kind := parent.Kind
	if kind == "" {
		kind = principal.KindService
	}
	child := &serviceaccount.ServiceAccount{
		ID:          id.NewServiceAccountID(),
		AppID:       parent.AppID,
		EnvID:       parent.EnvID,
		OrgID:       parent.OrgID,
		Kind:        kind,
		Name:        name,
		ParentID:    parent.ID,
		OwnerUserID: parent.OwnerUserID,
		Scopes:      scopes,
		ExpiresAt:   &expires,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := e.store.CreateServiceAccount(ctx, child); err != nil {
		return nil, nil, "", fmt.Errorf("authsome: mint child: %w", err)
	}

	key, secret, err := e.CreateServiceAccountAPIKey(ctx, child.ID, name, scopes, &expires)
	if err != nil {
		return nil, nil, "", fmt.Errorf("authsome: mint child: create key: %w", err)
	}
	return child, key, secret, nil
}

// requireScopeSubset refuses any scope the parent does not itself hold. A
// parent with no scopes at all places no restriction, matching how an empty
// scope list is read everywhere else in this package.
//
// The len(child) == 0 short-circuit only fires when the parent is itself
// unscoped: MintChildPrincipal fills an empty child list from the parent
// before calling here, so an unscoped request against a scoped parent arrives
// already carrying the parent's own list rather than nothing.
func requireScopeSubset(child, parent []string) error {
	if len(parent) == 0 || len(child) == 0 {
		return nil
	}
	has := make(map[string]bool, len(parent))
	for _, s := range parent {
		has[s] = true
	}
	for _, s := range child {
		if !has[s] {
			return fmt.Errorf("scope %q is outside the parent's scopes", s)
		}
	}
	return nil
}

// ReapExpiredPrincipals deletes lapsed ephemeral children and returns how
// many went.
//
// Only children are reaped. A durable principal with an expiry set is one an
// operator chose to time-limit, and deleting it out from under them would
// destroy a registration they can see in the dashboard.
func (e *Engine) ReapExpiredPrincipals(ctx context.Context, appID id.AppID) (int, error) {
	if err := e.requireStarted(); err != nil {
		return 0, err
	}
	all, err := e.store.ListPrincipals(ctx, &principal.Query{AppID: appID})
	if err != nil {
		return 0, fmt.Errorf("authsome: reap: list principals: %w", err)
	}
	now := time.Now()
	reaped := 0
	for _, p := range all {
		if p.Parent == nil || !p.IsExpired(now) {
			continue
		}
		svcID, parseErr := id.ParseServiceAccountID(p.ID)
		if parseErr != nil {
			continue
		}
		if delErr := e.store.DeleteServiceAccount(ctx, svcID); delErr != nil {
			e.logger.Warn("authsome: reap: delete failed",
				log.String("principal", p.Ref.String()),
				log.String("error", delErr.Error()),
			)
			continue
		}
		reaped++
	}
	return reaped, nil
}
