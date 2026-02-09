package interfaces

import (
	"context"

	"github.com/rs/xid"
)

// ContextKey type for context keys.
type ContextKey string

const (
	// AppIDKey is the context key for app ID.
	AppIDKey ContextKey = "app_id"

	// EnvironmentIDKey is the context key for environment ID.
	EnvironmentIDKey ContextKey = "environment_id"

	// OrganizationIDKey is the context key for organization ID.
	OrganizationIDKey ContextKey = "organization_id"

	// UserIDKey is the context key for user ID.
	UserIDKey ContextKey = "user_id"
)

// App Context Helpers

// GetAppID retrieves app ID from context.
func GetAppID(ctx context.Context) xid.ID {
	if id := ctx.Value(AppIDKey); id != nil {
		if appID, ok := id.(xid.ID); ok {
			return appID
		}
	}

	return xid.NilID()
}

// SetAppID sets app ID in context.
func SetAppID(ctx context.Context, appID xid.ID) context.Context {
	return context.WithValue(ctx, AppIDKey, appID)
}

// Environment Context Helpers

// GetEnvironmentID retrieves environment ID from context.
func GetEnvironmentID(ctx context.Context) xid.ID {
	if id := ctx.Value(EnvironmentIDKey); id != nil {
		if envID, ok := id.(xid.ID); ok {
			return envID
		}
	}

	return xid.NilID()
}

// SetEnvironmentID sets environment ID in context.
func SetEnvironmentID(ctx context.Context, envID xid.ID) context.Context {
	return context.WithValue(ctx, EnvironmentIDKey, envID)
}

// Organization Context Helpers

// GetOrganizationID retrieves organization ID from context.
func GetOrganizationID(ctx context.Context) xid.ID {
	if id := ctx.Value(OrganizationIDKey); id != nil {
		if orgID, ok := id.(xid.ID); ok {
			return orgID
		}
	}

	return xid.NilID()
}

// SetOrganizationID sets organization ID in context.
func SetOrganizationID(ctx context.Context, orgID xid.ID) context.Context {
	return context.WithValue(ctx, OrganizationIDKey, orgID)
}

// User Context Helpers

// GetUserID retrieves user ID from context.
func GetUserID(ctx context.Context) xid.ID {
	if id := ctx.Value(UserIDKey); id != nil {
		if userID, ok := id.(xid.ID); ok {
			return userID
		}
	}

	return xid.NilID()
}

// SetUserID sets user ID in context.
func SetUserID(ctx context.Context, userID xid.ID) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// HasAppID checks if app ID exists in context.
func HasAppID(ctx context.Context) bool {
	return !GetAppID(ctx).IsNil()
}

// HasEnvironmentID checks if environment ID exists in context.
func HasEnvironmentID(ctx context.Context) bool {
	return !GetEnvironmentID(ctx).IsNil()
}

// HasOrganizationID checks if organization ID exists in context.
func HasOrganizationID(ctx context.Context) bool {
	return !GetOrganizationID(ctx).IsNil()
}

// HasUserID checks if user ID exists in context.
func HasUserID(ctx context.Context) bool {
	return !GetUserID(ctx).IsNil()
}
