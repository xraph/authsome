package authsome

import (
	"context"
)

// Auto-generated context management utilities

type authContextKey string

const (
	appContextKey    authContextKey = "authsome_app_id"
	envContextKey    authContextKey = "authsome_env_id"
	userIDContextKey authContextKey = "authsome_user_id"
)

// WithAppID adds app ID to context
func WithAppID(ctx context.Context, appID string) context.Context {
	return context.WithValue(ctx, appContextKey, appID)
}

// GetAppID retrieves app ID from context
func GetAppID(ctx context.Context) (string, bool) {
	appID, ok := ctx.Value(appContextKey).(string)
	return appID, ok
}

// WithEnvironmentID adds environment ID to context
func WithEnvironmentID(ctx context.Context, envID string) context.Context {
	return context.WithValue(ctx, envContextKey, envID)
}

// GetEnvironmentID retrieves environment ID from context
func GetEnvironmentID(ctx context.Context) (string, bool) {
	envID, ok := ctx.Value(envContextKey).(string)
	return envID, ok
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDContextKey).(string)
	return userID, ok
}

// SetContextAppAndEnvironment adds both app and environment IDs to context
func SetContextAppAndEnvironment(ctx context.Context, appID, envID string) context.Context {
	ctx = context.WithValue(ctx, appContextKey, appID)
	ctx = context.WithValue(ctx, envContextKey, envID)
	return ctx
}
