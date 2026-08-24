package authsome

import (
	"context"
	"fmt"

	log "github.com/xraph/go-utils/log"
	"github.com/xraph/warden"

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
		// Debug, not Warn: a subject-only denial is exactly the case
		// HasPermission's own wrapper already logs at Warn with tenant and
		// scope diagnostics attached, and permission checks are a
		// high-volume hot path. Doubling every denial into two Warn lines
		// would double the operational log cost for no new information at
		// the subject-only call site. The actor-hop case below stays at
		// Warn: that is the case a plain subject-only caller like
		// HasPermission can never produce, and its decision/reason are not
		// available anywhere else.
		e.logger.Debug("authsome: Can denied by subject",
			log.String("subject", subject.String()),
			log.String("action", action),
			log.String("resource", resource),
			log.String("decision", string(result.Decision)),
			log.String("reason", result.Reason),
		)
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
