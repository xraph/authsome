package permissions

import (
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/permissions/handlers"
	"github.com/xraph/forge"
)

// PermissionMiddleware provides middleware for automatic permission checking on routes
type PermissionMiddleware struct {
	service *Service
	config  PermissionMiddlewareConfig
	logger  forge.Logger
}

// PermissionMiddlewareConfig configures the permission middleware behavior
type PermissionMiddlewareConfig struct {
	// ResourceType is the type of resource being protected (e.g., "document", "project")
	ResourceType string

	// Action is the action being performed (e.g., "read", "write", "delete")
	Action string

	// ResourceIDParam is the URL parameter name containing the resource ID (e.g., "id", "documentId")
	// If empty, the middleware checks access at the resource type level without a specific resource
	ResourceIDParam string

	// DenyHandler is an optional custom handler for denied requests
	// If nil, a default 403 response is returned
	DenyHandler forge.Handler

	// AllowAnonymous allows unauthenticated requests to proceed (useful with other conditions)
	AllowAnonymous bool

	// SkipIfNoUser skips permission check if no user is in context (useful for optional auth routes)
	SkipIfNoUser bool

	// CustomContext provides additional context for the permission check
	CustomContext map[string]interface{}
}

// NewPermissionMiddleware creates a new permission middleware
func NewPermissionMiddleware(service *Service, config PermissionMiddlewareConfig, logger forge.Logger) *PermissionMiddleware {
	return &PermissionMiddleware{
		service: service,
		config:  config,
		logger:  logger,
	}
}

// Middleware returns the forge middleware function
func (m *PermissionMiddleware) Middleware() func(next forge.Handler) forge.Handler {
	return func(next forge.Handler) forge.Handler {
		return func(c forge.Context) error {
			// Extract context values
			ctx := c.Request().Context()

			// Get app and environment (required)
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				return c.JSON(http.StatusBadRequest, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest))
			}

			envID, ok := contexts.GetEnvironmentID(ctx)
			if !ok || envID.IsNil() {
				return c.JSON(http.StatusBadRequest, errs.New("ENV_CONTEXT_REQUIRED", "Environment context required", http.StatusBadRequest))
			}

			// Get organization (optional)
			var orgID *xid.ID
			if orgIDVal, ok := contexts.GetOrganizationID(ctx); ok && !orgIDVal.IsNil() {
				orgID = &orgIDVal
			}

			// Get user (required unless SkipIfNoUser or AllowAnonymous)
			userID, hasUser := contexts.GetUserID(ctx)
			if !hasUser || userID.IsNil() {
				if m.config.SkipIfNoUser {
					return next(c)
				}
				if !m.config.AllowAnonymous {
					return c.JSON(http.StatusUnauthorized, errs.New("USER_REQUIRED", "Authentication required", http.StatusUnauthorized))
				}
			}

			// Get resource ID from URL param if configured
			var resourceID string
			if m.config.ResourceIDParam != "" {
				resourceID = c.Param(m.config.ResourceIDParam)
			}

			// Build evaluation request
			evalReq := &handlers.EvaluateRequest{
				ResourceType: m.config.ResourceType,
				ResourceID:   resourceID,
				Action:       m.config.Action,
				Principal: map[string]interface{}{
					"id": userID.String(),
				},
				Resource: map[string]interface{}{
					"type": m.config.ResourceType,
					"id":   resourceID,
				},
			}

			// Add custom context if provided
			if m.config.CustomContext != nil {
				evalReq.Context = m.config.CustomContext
			}

			// Evaluate permission
			decision, err := m.service.Evaluate(ctx, appID, envID, orgID, userID, evalReq)
			if err != nil {
				m.logger.Error("permission evaluation failed",
					forge.F("error", err.Error()),
					forge.F("resource_type", m.config.ResourceType),
					forge.F("action", m.config.Action),
				)
				return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
			}

			// Check decision
			if !decision.Allowed {
				if m.config.DenyHandler != nil {
					return m.config.DenyHandler(c)
				}
				return c.JSON(http.StatusForbidden, errs.PermissionDenied(m.config.Action, m.config.ResourceType))
			}

			// Permission granted, continue to next handler
			return next(c)
		}
	}
}

// PermissionMiddlewareFunc is the middleware function type
type PermissionMiddlewareFunc func(next forge.Handler) forge.Handler

// RequirePermission creates a permission-checking middleware for specific resource type and action
// This is a convenience function for common use cases
func (p *Plugin) RequirePermission(resourceType, action string) PermissionMiddlewareFunc {
	return NewPermissionMiddleware(p.service, PermissionMiddlewareConfig{
		ResourceType: resourceType,
		Action:       action,
	}, p.logger).Middleware()
}

// RequirePermissionWithID creates a middleware that checks permission for a specific resource instance
func (p *Plugin) RequirePermissionWithID(resourceType, action, resourceIDParam string) PermissionMiddlewareFunc {
	return NewPermissionMiddleware(p.service, PermissionMiddlewareConfig{
		ResourceType:    resourceType,
		Action:          action,
		ResourceIDParam: resourceIDParam,
	}, p.logger).Middleware()
}

// RequirePermissionWithConfig creates a middleware with full configuration
func (p *Plugin) RequirePermissionWithConfig(config PermissionMiddlewareConfig) PermissionMiddlewareFunc {
	return NewPermissionMiddleware(p.service, config, p.logger).Middleware()
}

// RequireOwnership creates a middleware that requires the user to own the resource
// This is a common pattern for user-owned resources
func (p *Plugin) RequireOwnership(resourceType, resourceIDParam string) PermissionMiddlewareFunc {
	return NewPermissionMiddleware(p.service, PermissionMiddlewareConfig{
		ResourceType:    resourceType,
		Action:          "own",
		ResourceIDParam: resourceIDParam,
	}, p.logger).Middleware()
}

// RequireRead creates a middleware that requires read permission
func (p *Plugin) RequireRead(resourceType string) PermissionMiddlewareFunc {
	return p.RequirePermission(resourceType, "read")
}

// RequireWrite creates a middleware that requires write permission
func (p *Plugin) RequireWrite(resourceType string) PermissionMiddlewareFunc {
	return p.RequirePermission(resourceType, "write")
}

// RequireDelete creates a middleware that requires delete permission
func (p *Plugin) RequireDelete(resourceType string) PermissionMiddlewareFunc {
	return p.RequirePermission(resourceType, "delete")
}

// RequireAdmin creates a middleware that requires admin permission on a resource type
func (p *Plugin) RequireAdmin(resourceType string) PermissionMiddlewareFunc {
	return p.RequirePermission(resourceType, "admin")
}

// CheckPermission is a helper that evaluates a permission check inline without middleware
// Useful for conditional permission checks within handlers
func (p *Plugin) CheckPermission(c forge.Context, resourceType, action, resourceID string) (bool, error) {
	ctx := c.Request().Context()

	// Extract context
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	userID, _ := contexts.GetUserID(ctx)

	var orgID *xid.ID
	if orgIDVal, ok := contexts.GetOrganizationID(ctx); ok && !orgIDVal.IsNil() {
		orgID = &orgIDVal
	}

	// Build evaluation request
	evalReq := &handlers.EvaluateRequest{
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Action:       action,
		Principal: map[string]interface{}{
			"id": userID.String(),
		},
		Resource: map[string]interface{}{
			"type": resourceType,
			"id":   resourceID,
		},
	}

	// Evaluate
	decision, err := p.service.Evaluate(ctx, appID, envID, orgID, userID, evalReq)
	if err != nil {
		return false, err
	}

	return decision.Allowed, nil
}
