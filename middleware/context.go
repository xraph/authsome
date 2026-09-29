// Package middleware provides HTTP middleware for authentication and context resolution.
package middleware

import (
	"context"

	"github.com/xraph/forge"
	forgemw "github.com/xraph/forge/middleware"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// RequestInfo re-exports hook.RequestInfo so handlers and plugins can read
// the request correlation data without importing the hook package.
type RequestInfo = hook.RequestInfo

// WithRequestInfo stores the request's correlation data on the context.
func WithRequestInfo(ctx context.Context, info RequestInfo) context.Context {
	return hook.WithRequestInfo(ctx, info)
}

// RequestInfoFrom returns the request's correlation data.
func RequestInfoFrom(ctx context.Context) (RequestInfo, bool) {
	return hook.RequestInfoFrom(ctx)
}

// requestInfoFromRequest builds the correlation data for an inbound request.
// The IP comes from the trusted-proxy resolver, never a raw header.
func requestInfoFromRequest(ctx forge.Context) RequestInfo {
	r := ctx.Request()
	return RequestInfo{
		IP:        ClientIP(r),
		UserAgent: r.UserAgent(),
		RequestID: forgemw.GetRequestID(ctx.Context()),
	}
}

// installRequestInfo sets request info on the forge context once per request.
// It runs before credentials are read, so anonymous requests are correlated
// too; WithSessionID adds the session once one is resolved.
func installRequestInfo(ctx forge.Context) {
	if _, ok := hook.RequestInfoFrom(ctx.Context()); ok {
		return
	}
	ctx.WithContext(WithRequestInfo(ctx.Context(), requestInfoFromRequest(ctx)))
}

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
	ctxKeyApp
	ctxKeyPendingOAuth
	ctxKeyCredentialScheme
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

// PrincipalRefFrom returns the authenticated caller as a principal ref,
// whatever kind of caller it is.
//
// Three sources, in order, which is what lets this work now and keep working
// once the principal package is wired all the way through the middleware.
// First a principal already resolved onto the context. Then a user, which is
// every human request today. Then a session whose subject is not a person,
// which is how service accounts, agents and workloads arrive before that
// wiring exists.
//
// A session with no resolved user behind it is deliberately not a caller. An
// empty PrincipalKind means user, so such a session is a human one whose user
// could not be loaded, and treating it as authenticated would turn a failed
// lookup into an authorization bypass.
func PrincipalRefFrom(ctx context.Context) (principal.Ref, bool) {
	if p, ok := principal.FromContext(ctx); ok && p != nil {
		return p.Ref, true
	}

	if u, ok := UserFrom(ctx); ok && u != nil {
		return principal.UserRef(u.ID), true
	}

	if s, ok := SessionFrom(ctx); ok && s != nil && !s.IsHumanPrincipal() {
		return s.Subject(), true
	}

	return principal.Ref{}, false
}

// WithPrincipal returns ctx carrying the resolved caller.
//
// This delegates to the principal package's own context functions rather
// than defining a separate key, so a plugin reading through
// principal.FromContext sees the exact value this middleware wrote, and
// PrincipalRefFrom's first branch (above) resolves against it.
func WithPrincipal(ctx context.Context, p *principal.Principal) context.Context {
	return principal.NewContext(ctx, p)
}

// PrincipalFrom returns the resolved caller previously stored with
// WithPrincipal.
func PrincipalFrom(ctx context.Context) (*principal.Principal, bool) {
	return principal.FromContext(ctx)
}

// WithActors returns ctx carrying the actor chain acting on the subject's
// behalf. Delegates to the principal package's own context functions for the
// same shared-key reason as WithPrincipal.
func WithActors(ctx context.Context, c principal.Chain) context.Context {
	return principal.NewActorsContext(ctx, c)
}

// ActorsFrom returns the actor chain previously stored with WithActors.
func ActorsFrom(ctx context.Context) (principal.Chain, bool) {
	return principal.ActorsFromContext(ctx)
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
	// Every path that resolves a session passes through here, so this is the
	// one place the request correlation learns which session acted.
	if info, ok := hook.RequestInfoFrom(ctx); ok && info.SessionID == "" {
		info.SessionID = sessionID.String()
		ctx = hook.WithRequestInfo(ctx, info)
	}
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

// WithApp stores the resolved app entity in the context. Set by the
// publishable-key middleware so handlers can validate against the full
// app without re-querying the store.
func WithApp(ctx context.Context, a *app.App) context.Context {
	return context.WithValue(ctx, ctxKeyApp, a)
}

// AppFrom retrieves the resolved app from the context.
func AppFrom(ctx context.Context) (*app.App, bool) {
	v, ok := ctx.Value(ctxKeyApp).(*app.App)
	return v, ok && v != nil
}

// WithAuthMethod stores the authentication method used (e.g. "session", "strategy").
// WithCredentialScheme records how the request presented its credential
// (bearer, dpop or cookie), for the checks that only apply to one of them.
func WithCredentialScheme(ctx context.Context, scheme string) context.Context {
	return context.WithValue(ctx, ctxKeyCredentialScheme, scheme)
}

// CredentialSchemeFrom returns how the request presented its credential.
func CredentialSchemeFrom(ctx context.Context) (string, bool) {
	s, ok := ctx.Value(ctxKeyCredentialScheme).(string)
	return s, ok && s != ""
}

func WithAuthMethod(ctx context.Context, method string) context.Context {
	return context.WithValue(ctx, ctxKeyAuthMethod, method)
}

// AuthMethodFrom retrieves the authentication method from the context.
func AuthMethodFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxKeyAuthMethod).(string)
	return v, ok
}
