package contexts

import (
	"context"

	"github.com/rs/xid"
)

// Context keys for multi-level tenancy.
type contextKey string

const (
	// AppContextKey is the context key for the current app ID (platform tenant).
	AppContextKey contextKey = "app_id"

	// EnvironmentContextKey is the context key for the current environment ID.
	EnvironmentContextKey contextKey = "environment_id"

	// OrganizationContextKey is the context key for the current organization ID (end-user workspace).
	OrganizationContextKey contextKey = "organization_id"

	// UserContextKey is the context key for the current authenticated user ID.
	UserContextKey contextKey = "user_id"
)

// =============================================================================
// APP CONTEXT (Platform Tenant)
// =============================================================================

// GetAppID retrieves the app ID from context
// Returns the app ID and true if found, or xid.NilID() and false if not found.
func GetAppID(ctx context.Context) (xid.ID, bool) {
	if val := ctx.Value(AppContextKey); val != nil {
		if appID, ok := val.(xid.ID); ok {
			return appID, true
		}
		// Try string conversion
		if appIDStr, ok := val.(string); ok {
			if id, err := xid.FromString(appIDStr); err == nil {
				return id, true
			}
		}
	}

	return xid.NilID(), false
}

// SetAppID sets the app ID in context.
func SetAppID(ctx context.Context, appID xid.ID) context.Context {
	return context.WithValue(ctx, AppContextKey, appID)
}

// RequireAppID retrieves the app ID from context or returns an error.
func RequireAppID(ctx context.Context) (xid.ID, error) {
	appID, ok := GetAppID(ctx)
	if !ok || appID.IsNil() {
		return xid.NilID(), ErrAppContextRequired
	}

	return appID, nil
}

// =============================================================================
// ENVIRONMENT CONTEXT
// =============================================================================

// GetEnvironmentID retrieves the environment ID from context.
func GetEnvironmentID(ctx context.Context) (xid.ID, bool) {
	if val := ctx.Value(EnvironmentContextKey); val != nil {
		if envID, ok := val.(xid.ID); ok {
			return envID, true
		}

		if envIDStr, ok := val.(string); ok {
			if id, err := xid.FromString(envIDStr); err == nil {
				return id, true
			}
		}
	}

	return xid.NilID(), false
}

// SetEnvironmentID sets the environment ID in context.
func SetEnvironmentID(ctx context.Context, envID xid.ID) context.Context {
	return context.WithValue(ctx, EnvironmentContextKey, envID)
}

// RequireEnvironmentID retrieves the environment ID from context or returns an error.
func RequireEnvironmentID(ctx context.Context) (xid.ID, error) {
	envID, ok := GetEnvironmentID(ctx)
	if !ok || envID.IsNil() {
		return xid.NilID(), ErrEnvironmentContextRequired
	}

	return envID, nil
}

// =============================================================================
// ORGANIZATION CONTEXT (End-user Workspace)
// =============================================================================

// GetOrganizationID retrieves the organization ID from context.
func GetOrganizationID(ctx context.Context) (xid.ID, bool) {
	if val := ctx.Value(OrganizationContextKey); val != nil {
		if orgID, ok := val.(xid.ID); ok {
			return orgID, true
		}

		if orgIDStr, ok := val.(string); ok {
			if id, err := xid.FromString(orgIDStr); err == nil {
				return id, true
			}
		}
	}

	return xid.NilID(), false
}

// SetOrganizationID sets the organization ID in context.
func SetOrganizationID(ctx context.Context, orgID xid.ID) context.Context {
	return context.WithValue(ctx, OrganizationContextKey, orgID)
}

// RequireOrganizationID retrieves the organization ID from context or returns an error.
func RequireOrganizationID(ctx context.Context) (xid.ID, error) {
	orgID, ok := GetOrganizationID(ctx)
	if !ok || orgID.IsNil() {
		return xid.NilID(), ErrOrganizationContextRequired
	}

	return orgID, nil
}

// =============================================================================
// USER CONTEXT
// =============================================================================

// GetUserID retrieves the user ID from context.
func GetUserID(ctx context.Context) (xid.ID, bool) {
	if val := ctx.Value(UserContextKey); val != nil {
		if userID, ok := val.(xid.ID); ok {
			return userID, true
		}

		if userIDStr, ok := val.(string); ok {
			if id, err := xid.FromString(userIDStr); err == nil {
				return id, true
			}
		}
	}

	return xid.NilID(), false
}

// SetUserID sets the user ID in context.
func SetUserID(ctx context.Context, userID xid.ID) context.Context {
	return context.WithValue(ctx, UserContextKey, userID)
}

// RequireUserID retrieves the user ID from context or returns an error.
func RequireUserID(ctx context.Context) (xid.ID, error) {
	userID, ok := GetUserID(ctx)
	if !ok || userID.IsNil() {
		return xid.NilID(), ErrUserContextRequired
	}

	return userID, nil
}

// =============================================================================
// COMPOSITE CONTEXT HELPERS
// =============================================================================

// WithAppAndOrganization sets both app and organization context.
func WithAppAndOrganization(ctx context.Context, appID, orgID xid.ID) context.Context {
	ctx = SetAppID(ctx, appID)

	return SetOrganizationID(ctx, orgID)
}

// WithAppAndUser sets both app and user context.
func WithAppAndUser(ctx context.Context, appID, userID xid.ID) context.Context {
	ctx = SetAppID(ctx, appID)

	return SetUserID(ctx, userID)
}

// WithAppEnvironmentAndOrganization sets app, environment, and organization context.
func WithAppEnvironmentAndOrganization(ctx context.Context, appID, envID, orgID xid.ID) context.Context {
	ctx = SetAppID(ctx, appID)
	ctx = SetEnvironmentID(ctx, envID)

	return SetOrganizationID(ctx, orgID)
}

// WithAll sets all context values.
func WithAll(ctx context.Context, appID, envID, orgID, userID xid.ID) context.Context {
	ctx = SetAppID(ctx, appID)
	ctx = SetEnvironmentID(ctx, envID)
	ctx = SetOrganizationID(ctx, orgID)

	return SetUserID(ctx, userID)
}
