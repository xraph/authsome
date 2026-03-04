package middleware

import (
	"context"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
)

// PermissionChecker can verify if a user has a specific permission.
type PermissionChecker interface {
	HasPermission(ctx context.Context, userID id.UserID, action, resource string) (bool, error)
}

// RequirePermission returns a forge.Middleware that checks the authenticated
// user has the given permission before continuing.
func RequirePermission(checker PermissionChecker, action, resource string) forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			userID, ok := UserIDFrom(ctx.Context())
			if !ok {
				return ctx.JSON(http.StatusUnauthorized, map[string]any{
					"error": "authentication required",
					"code":  http.StatusUnauthorized,
				})
			}

			allowed, err := checker.HasPermission(ctx.Context(), userID, action, resource)
			if err != nil {
				return ctx.JSON(http.StatusInternalServerError, map[string]any{
					"error": "permission check failed",
					"code":  http.StatusInternalServerError,
				})
			}
			if !allowed {
				return ctx.JSON(http.StatusForbidden, map[string]any{
					"error": "insufficient permissions",
					"code":  http.StatusForbidden,
				})
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
func RequireAnyRole(checker RoleChecker, roles ...string) forge.Middleware {
	roleSet := make(map[string]bool, len(roles))
	for _, r := range roles {
		roleSet[r] = true
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			userID, ok := UserIDFrom(ctx.Context())
			if !ok {
				return ctx.JSON(http.StatusUnauthorized, map[string]any{
					"error": "authentication required",
					"code":  http.StatusUnauthorized,
				})
			}

			userRoles, err := checker.ListUserRoleSlugs(ctx.Context(), userID)
			if err != nil {
				return ctx.JSON(http.StatusInternalServerError, map[string]any{
					"error": "role check failed",
					"code":  http.StatusInternalServerError,
				})
			}

			for _, ur := range userRoles {
				if roleSet[ur] {
					return next(ctx)
				}
			}

			return ctx.JSON(http.StatusForbidden, map[string]any{
				"error": "insufficient role",
				"code":  http.StatusForbidden,
			})
		}
	}
}
