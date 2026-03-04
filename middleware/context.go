// Package middleware provides HTTP middleware for authentication and context resolution.
package middleware

import (
	"context"

	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// Context keys for typed access to auth state.
type contextKey int

const (
	ctxKeyUser contextKey = iota
	ctxKeySession
	ctxKeyAppID
	ctxKeyOrgID
	ctxKeyUserID
	ctxKeySessionID
	ctxKeyImpersonator
	ctxKeyEnvID
	ctxKeyEnvironment
	ctxKeyEnvironmentSettings
	ctxKeyAuthMethod
)

// WithUser stores a user in the context.
func WithUser(ctx context.Context, u *user.User) context.Context {
	return context.WithValue(ctx, ctxKeyUser, u)
}

// UserFrom retrieves the user from the context.
func UserFrom(ctx context.Context) (*user.User, bool) {
	u, ok := ctx.Value(ctxKeyUser).(*user.User)
	return u, ok
}

// WithSession stores a session in the context.
func WithSession(ctx context.Context, s *session.Session) context.Context {
	return context.WithValue(ctx, ctxKeySession, s)
}

// SessionFrom retrieves the session from the context.
func SessionFrom(ctx context.Context) (*session.Session, bool) {
	s, ok := ctx.Value(ctxKeySession).(*session.Session)
	return s, ok
}

// WithAppID stores an app ID in the context.
func WithAppID(ctx context.Context, appID id.AppID) context.Context {
	return context.WithValue(ctx, ctxKeyAppID, appID)
}

// AppIDFrom retrieves the app ID from the context.
func AppIDFrom(ctx context.Context) (id.AppID, bool) {
	v, ok := ctx.Value(ctxKeyAppID).(id.AppID)
	return v, ok
}

// WithOrgID stores an organization ID in the context.
func WithOrgID(ctx context.Context, orgID id.OrgID) context.Context {
	return context.WithValue(ctx, ctxKeyOrgID, orgID)
}

// OrgIDFrom retrieves the organization ID from the context.
func OrgIDFrom(ctx context.Context) (id.OrgID, bool) {
	v, ok := ctx.Value(ctxKeyOrgID).(id.OrgID)
	return v, ok
}

// WithUserID stores a user ID in the context.
func WithUserID(ctx context.Context, userID id.UserID) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

// UserIDFrom retrieves the user ID from the context.
func UserIDFrom(ctx context.Context) (id.UserID, bool) {
	v, ok := ctx.Value(ctxKeyUserID).(id.UserID)
	return v, ok
}

// WithSessionID stores a session ID in the context.
func WithSessionID(ctx context.Context, sessionID id.SessionID) context.Context {
	return context.WithValue(ctx, ctxKeySessionID, sessionID)
}

// SessionIDFrom retrieves the session ID from the context.
func SessionIDFrom(ctx context.Context) (id.SessionID, bool) {
	v, ok := ctx.Value(ctxKeySessionID).(id.SessionID)
	return v, ok
}

// WithImpersonator stores the ID of the admin who initiated the impersonation.
func WithImpersonator(ctx context.Context, adminID id.UserID) context.Context {
	return context.WithValue(ctx, ctxKeyImpersonator, adminID)
}

// ImpersonatorFrom retrieves the impersonator admin user ID from the context.
// Returns the zero value and false if the session is not impersonated.
func ImpersonatorFrom(ctx context.Context) (id.UserID, bool) {
	v, ok := ctx.Value(ctxKeyImpersonator).(id.UserID)
	return v, ok
}

// WithEnvID stores an environment ID in the context.
func WithEnvID(ctx context.Context, envID id.EnvironmentID) context.Context {
	return context.WithValue(ctx, ctxKeyEnvID, envID)
}

// EnvIDFrom retrieves the environment ID from the context.
func EnvIDFrom(ctx context.Context) (id.EnvironmentID, bool) {
	v, ok := ctx.Value(ctxKeyEnvID).(id.EnvironmentID)
	return v, ok
}

// WithEnvironment stores a full environment entity in the context.
func WithEnvironment(ctx context.Context, env *environment.Environment) context.Context {
	return context.WithValue(ctx, ctxKeyEnvironment, env)
}

// EnvironmentFrom retrieves the full environment from the context.
func EnvironmentFrom(ctx context.Context) (*environment.Environment, bool) {
	v, ok := ctx.Value(ctxKeyEnvironment).(*environment.Environment)
	return v, ok
}

// WithEnvironmentSettings stores the resolved environment settings in the context.
func WithEnvironmentSettings(ctx context.Context, s *environment.Settings) context.Context {
	return context.WithValue(ctx, ctxKeyEnvironmentSettings, s)
}

// EnvironmentSettingsFrom retrieves the resolved environment settings from the context.
func EnvironmentSettingsFrom(ctx context.Context) (*environment.Settings, bool) {
	v, ok := ctx.Value(ctxKeyEnvironmentSettings).(*environment.Settings)
	return v, ok
}

// WithAuthMethod stores the authentication method used (e.g. "session", "strategy").
func WithAuthMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, ctxKeyAuthMethod, method)
}

// AuthMethodFrom retrieves the authentication method from the context.
func AuthMethodFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyAuthMethod).(string)
	return v, ok
}
