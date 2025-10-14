package types

import "context"

// Context keys
type contextKey string

const (
    ContextKeyUser         contextKey = "user"
    ContextKeySession      contextKey = "session"
    ContextKeyOrganization contextKey = "organization"
)

// AuthContext contains authentication context
type AuthContext struct {
    UserID         string
    SessionID      string
    OrganizationID string
    Roles          []string
}

// GetAuthContext retrieves auth context from context
func GetAuthContext(ctx context.Context) (*AuthContext, bool) {
    authCtx, ok := ctx.Value(ContextKeyUser).(*AuthContext)
    return authCtx, ok
}

// SetAuthContext sets auth context in context
func SetAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
    return context.WithValue(ctx, ContextKeyUser, authCtx)
}