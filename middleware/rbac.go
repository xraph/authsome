package middleware

import (
	"context"
	"net/http"
	"strconv"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
)

// PermissionChecker can verify if a user has a specific permission.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID id.UserID, action, resource string) (bool, error)
}

// ChainPermissionChecker is the chain-aware counterpart to PermissionChecker:
// it evaluates a permission for a subject given the actors acting on that
// subject's behalf.
//
// It sits beside PermissionChecker rather than replacing it, following the
// same convention as PermissionChecker and UserResolver: this package must not
// import the engine, so the capability arrives as a narrow interface the
// engine happens to satisfy. *authsome.Engine implements it via Can, and
// plugin.Engine carries Can on its core interface, so every real engine
// reaching RequirePermission is chain-aware without any call site changing.
//
// Why this matters: with an actor chain present, every party must allow the
// action. A session minted by a token exchange carries the human as its
// subject and the agent in its chain, and checking the human alone would hand
// the agent the human's entire authority. Delegation may only narrow, and this
// interface is how the request path gets to say so.
type ChainPermissionChecker interface {
	Can(ctx context.Context, subject principal.Ref, actors principal.Chain, action, resource string) (bool, error)
}

// CallerSubjectFrom returns the principal a permission check must evaluate.
//
// The subject is who the request is FOR, never who is doing the acting: on a
// delegated session that is the human, and the agent is in the actor chain.
// Resolution order matches PrincipalRefFrom (a resolved principal, then a
// user, then a non-human session subject) with one addition at the end. A
// human session whose user row failed to resolve still has its user ID on the
// context, and falling back to it keeps this exactly as permissive as the
// UserIDFrom gate RequirePermission used before chains existed: no more, no
// less.
func CallerSubjectFrom(ctx context.Context) (principal.Ref, bool) {
	if ref, ok := PrincipalRefFrom(ctx); ok && !ref.IsZero() {
		return ref, true
	}
	if uid, ok := UserIDFrom(ctx); ok {
		return principal.UserRef(uid), true
	}
	return principal.Ref{}, false
}

// AuthzActorsFrom returns the actor chain Warden must independently authorize
// for this request.
//
// The session is asked first, because Session.AuthzActors is where the
// impersonation exception lives: an admin acting as a user must be evaluated
// as that user alone, so the chain it returns is deliberately empty. Reading
// sess.Actors here instead would intersect the admin's own permissions into
// every check and invert impersonation, letting the admin do LESS than the
// person they are standing in for.
//
// The context chain is the fallback, for callers that arrive without a session
// row behind them.
func AuthzActorsFrom(ctx context.Context) principal.Chain {
	if s, ok := SessionFrom(ctx); ok && s != nil {
		return s.AuthzActors()
	}
	if c, ok := ActorsFrom(ctx); ok {
		return c
	}
	return nil
}

// loggerProvider is an optional interface for checkers that expose a logger.
type loggerProvider interface {
	Logger() log.Logger
}

// RequirePermission returns a forge.Middleware that checks the authenticated
// caller has the given permission before continuing.
//
// The route is chosen per request, on whether the caller actually carries an
// actor chain, and not once at construction on whether the checker happens to
// have a Can method.
//
//	no actors                       -> requireUserPermission, unchanged
//	actors + chain-aware checker    -> RequireChainPermission
//	actors + checker without Can    -> refuse
//
// Reading the actors first looks like a weakening of the chain fix and is not.
// It narrows WHEN the new path runs without narrowing WHAT it enforces. With an
// empty chain the two paths are semantically identical: HasPermission is
// literally Can with a nil chain (see Engine.HasPermission), so a chainless
// request gets the same answer either way. Every request that carries a chain
// still goes through Can, which is the whole point of the fix.
//
// What it buys is that upgrading does not silently depend on every existing
// PermissionChecker's Can being correct. Any implementation that satisfies
// ChainPermissionChecker with a weak or unimplemented Can, in this repository
// or out of tree, would otherwise start denying requests it used to allow, with
// no compile error and no obvious symptom. That happened here: a plugin test
// double answered HasPermission with real logic and Can with a bare
// `return false, nil`, and every mutating route it guarded turned 403.
//
// The last case does not fall back to evaluating the subject alone. A caller
// with a chain and no way to check it is exactly the Critical this fix exists
// to close, so it is refused and logged rather than quietly granted the human's
// full authority.
func RequirePermission(checker PermissionChecker, action, resource string) forge.Middleware {
	userPath := requireUserPermission(checker, action, resource)
	var chainPath forge.Middleware
	if chainAware, ok := checker.(ChainPermissionChecker); ok {
		chainPath = RequireChainPermission(chainAware, action, resource)
	}

	return func(next forge.Handler) forge.Handler {
		// Both paths are built once per route rather than per request, so
		// this costs one branch on the hot path and no allocation.
		userNext := userPath(next)
		var chainNext forge.Handler
		if chainPath != nil {
			chainNext = chainPath(next)
		}

		return func(ctx forge.Context) error {
			if len(AuthzActorsFrom(ctx.Context())) == 0 {
				return userNext(ctx)
			}
			if chainNext == nil {
				if lp, ok := checker.(loggerProvider); ok {
					lp.Logger().Warn("rbac: refusing a delegated request, permission checker cannot evaluate an actor chain",
						log.String("action", action),
						log.String("resource", resource),
						log.String("path", ctx.Request().URL.Path),
					)
				}
				return forge.Forbidden("insufficient permissions")
			}
			return chainNext(ctx)
		}
	}
}

// RequireChainPermission returns a forge.Middleware that checks the caller's
// subject AND every actor standing between that subject and this request.
//
// This is the guard that makes "delegation can only narrow" true on the
// request path rather than only inside the engine. Without it, an agent that
// exchanged a grant for a session whose subject is the human it acts for would
// be evaluated as that human alone, with its own permissions never intersected
// and the grant's narrowing thrown away.
func RequireChainPermission(checker ChainPermissionChecker, action, resource string) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			subject, ok := CallerSubjectFrom(ctx.Context())
			if !ok {
				return forge.Unauthorized("authentication required")
			}
			actors := AuthzActorsFrom(ctx.Context())

			allowed, err := checker.Can(ctx.Context(), subject, actors, action, resource)
			if err != nil {
				if lp, ok := checker.(loggerProvider); ok {
					lp.Logger().Warn("rbac: permission check error",
						log.String("subject", subject.String()),
						log.String("actor_depth", strconv.Itoa(actors.Depth())),
						log.String("action", action),
						log.String("resource", resource),
						log.String("error", err.Error()),
					)
				}
				// Written by hand rather than returned as a typed error:
				// forge replaces the message on a 500 with a generic
				// "Internal Server Error", which would drop the specific
				// reason consumers currently receive.
				return ctx.JSON(http.StatusInternalServerError, map[string]any{
					"error": "permission check failed",
					"code":  http.StatusInternalServerError,
				})
			}
			if !allowed {
				if lp, ok := checker.(loggerProvider); ok {
					lp.Logger().Warn("rbac: permission denied",
						log.String("subject", subject.String()),
						log.String("actor_depth", strconv.Itoa(actors.Depth())),
						log.String("action", action),
						log.String("resource", resource),
						log.String("path", ctx.Request().URL.Path),
					)
				}
				return forge.Forbidden("insufficient permissions")
			}

			return next(ctx)
		}
	}
}

// requireUserPermission is the pre-chain guard, kept verbatim for checkers
// that expose HasPermission and nothing else.
func requireUserPermission(checker PermissionChecker, action, resource string) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			userID, ok := UserIDFrom(ctx.Context())
			if !ok {
				return forge.Unauthorized("authentication required")
			}

			allowed, err := checker.HasPermission(ctx.Context(), userID, action, resource)
			if err != nil {
				if lp, ok := checker.(loggerProvider); ok {
					lp.Logger().Warn("rbac: permission check error",
						log.String("user_id", userID.String()),
						log.String("action", action),
						log.String("resource", resource),
						log.String("error", err.Error()),
					)
				}
				// Written by hand rather than returned as a typed error:
				// forge replaces the message on a 500 with a generic
				// "Internal Server Error", which would drop the specific
				// reason consumers currently receive.
				return ctx.JSON(http.StatusInternalServerError, map[string]any{
					"error": "permission check failed",
					"code":  http.StatusInternalServerError,
				})
			}
			if !allowed {
				if lp, ok := checker.(loggerProvider); ok {
					lp.Logger().Warn("rbac: permission denied",
						log.String("user_id", userID.String()),
						log.String("action", action),
						log.String("resource", resource),
						log.String("path", ctx.Request().URL.Path),
					)
				}
				return forge.Forbidden("insufficient permissions")
			}

			return next(ctx)
		}
	}
}

// RoleChecker can list a user's role slugs.
type RoleChecker interface {
	ListUserRoleSlugs(ctx context.Context, userID id.UserID) ([]string, error)
}

// RequireAnyRole returns middleware that checks the user has at least one of the given roles.
// It first checks for a direct role slug match (fast path), then falls back to
// Warden's full RBAC+ReBAC+ABAC evaluation via PermissionChecker. This allows
// platform roles with wildcard permissions (e.g. platform_owner with *:*) to
// satisfy any role check without explicit slug matching.
func RequireAnyRole(checker RoleChecker, roles ...string) forge.Middleware {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			userID, ok := UserIDFrom(ctx.Context())
			if !ok {
				return forge.Unauthorized("authentication required")
			}

			userRoles, err := checker.ListUserRoleSlugs(ctx.Context(), userID)
			if err != nil {
				// Written by hand rather than returned as a typed error:
				// forge replaces the message on a 500 with a generic
				// "Internal Server Error", which would drop the specific
				// reason consumers currently receive.
				return ctx.JSON(http.StatusInternalServerError, map[string]any{
					"error": "role check failed",
					"code":  http.StatusInternalServerError,
				})
			}

			// Fast path: direct role slug match.
			for _, ur := range userRoles {
				if roleSet[ur] {
					return next(ctx)
				}
			}

			// Warden fallback: if checker also implements PermissionChecker,
			// evaluate through Warden's full RBAC engine. This allows platform
			// roles with wildcard permissions (e.g. platform_owner with *:*)
			// to satisfy any role check without explicit slug matching.
			if pc, ok := checker.(PermissionChecker); ok {
				allowed, permErr := pc.HasPermission(ctx.Context(), userID, "manage", "app")
				if permErr == nil && allowed {
					return next(ctx)
				}
			}

			return forge.Forbidden("insufficient role")
		}
	}
}
