package middleware

import (
	"context"
	"net/http"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
)

// PermissionChecker can verify if a user has a specific permission.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID id.UserID, action, resource string) (bool, error)
}

// loggerProvider is an optional interface for checkers that expose a logger.
type loggerProvider interface {
	Logger() log.Logger
}

// RequirePermission returns a forge.Middleware that checks the authenticated
// user has the given permission before continuing.
func RequirePermission(checker PermissionChecker, action, resource string) forge.Middleware {
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
