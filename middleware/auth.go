package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/tokenformat"
	"github.com/xraph/authsome/user"
)

// authDebugKey carries the most recent strategy/auth rejection
// reason through the request context so RequireAuth can echo it
// back on a 401 when AUTHSOME_DEBUG_AUTH=1. The mechanism is
// debug-only: production deployments leave the env var unset and
// the reason never leaves the server.
type authDebugKey struct{}

// withAuthDebug attaches a human-readable rejection reason to ctx.
func withAuthDebug(ctx context.Context, reason string) context.Context {
	return context.WithValue(ctx, authDebugKey{}, reason)
}

// authDebugFrom reads the most recent rejection reason from ctx.
func authDebugFrom(ctx context.Context) string {
	v, _ := ctx.Value(authDebugKey{}).(string) //nolint:errcheck // type-safe via key

	return v
}

// authDebugEnabled reports whether the operator has opted into
// surfacing strategy rejection reasons on 401 responses. Off by
// default — the reason carries credential-shape details (which
// header was missing, which app the key was minted under) that we
// don't want to expose to unauthenticated callers in production.
func authDebugEnabled() bool {
	return os.Getenv("AUTHSOME_DEBUG_AUTH") == "1"
}

// summarizeAuthHeaders renders a one-line description of the
// auth-relevant inbound headers — never the secret values, only
// presence + length so operators can tell "client never sent it"
// apart from "client sent it but it's malformed". Used by the
// debug_reason surface on 401 responses.
func summarizeAuthHeaders(r *http.Request) string {
	parts := []string{}
	if v := r.Header.Get("Authorization"); v != "" {
		parts = append(parts, "Authorization=present(len="+itoaInt(len(v))+",scheme="+authScheme(v)+")")
	} else {
		parts = append(parts, "Authorization=absent")
	}
	if v := r.Header.Get("X-API-Key"); v != "" {
		parts = append(parts, "X-API-Key=present(len="+itoaInt(len(v))+")")
	} else {
		parts = append(parts, "X-API-Key=absent")
	}
	if v := r.Header.Get("X-App-ID"); v != "" {
		parts = append(parts, "X-App-ID=present(value="+v+")")
	} else {
		parts = append(parts, "X-App-ID=absent")
	}
	if v := r.Header.Get("Cookie"); v != "" {
		parts = append(parts, "Cookie=present(len="+itoaInt(len(v))+")")
	} else {
		parts = append(parts, "Cookie=absent")
	}
	return strings.Join(parts, ", ")
}

func authScheme(authHeader string) string {
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) >= 1 {
		return parts[0]
	}
	return "unknown"
}

func itoaInt(n int) string {
	// Tiny inline helper to avoid importing strconv just for this
	// debug path. n is always small (header lengths).
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// StrategyAuthenticator authenticates requests using registered strategies.
// The strategy.Registry implements this interface.
type StrategyAuthenticator interface {
	Authenticate(ctx context.Context, r *http.Request) (*strategy.Result, error)
}

// SessionResolver loads a session from a token.
type SessionResolver func(token string) (*session.Session, error)

// UserResolver loads a user by ID string.
type UserResolver func(userID string) (*user.User, error)

// PrincipalResolver resolves a caller by ref. Middleware takes this as a
// function rather than an engine so it does not import the engine package.
type PrincipalResolver func(principal.Ref) (*principal.Principal, error)

// JWTValidator validates JWT access tokens and returns claims.
// The engine implements this via its TokenFormatForApp method.
type JWTValidator interface {
	ValidateJWT(token string) (*tokenformat.TokenClaims, error)
}

// SessionExistsChecker checks whether a session ID still exists in the store.
// Used for JWT revocation support — when enabled, JWT tokens are cross-checked
// against the session store so revoked sessions are rejected immediately.
type SessionExistsChecker func(sessionID string) (*session.Session, error)

// CookieNameResolver returns the session cookie name for the current context.
// When nil, the default "authsome_session_token" is used.
type CookieNameResolver func(ctx context.Context) string

// ExpectedAudienceResolver returns the resource identifiers this deployment
// answers to for the app the presented token was minted under. Nil, or an
// empty result, disables the audience check.
//
// The app id handed to the resolver is the TOKEN's, never the request's. A
// resource identifier is configured per app, and the request context only
// carries an app id when the caller sent a publishable key, which is optional.
// Resolving from the request would let a caller switch the check off by
// omitting a header, and would leave a resource server that never installs
// PublishableKeyMiddleware with no app-scoped enforcement at all.
type ExpectedAudienceResolver func(ctx context.Context, appID string) []string

// SessionBindingConfig controls session binding validation.
type SessionBindingConfig struct {
	// BindToIP rejects requests when the client IP differs from the
	// IP recorded at session creation.
	BindToIP bool

	// BindToDevice rejects requests when the User-Agent differs from
	// the one recorded at session creation.
	BindToDevice bool

	// CookieNameResolver returns the session cookie name for the current
	// request context. Falls back to "authsome_session_token" if nil or empty.
	CookieNameResolver CookieNameResolver

	// JWTSessionChecker, when set, cross-checks JWT tokens against the session
	// store. This enables JWT revocation — revoked sessions are rejected even
	// if the JWT signature is valid. Also enables IP/device binding for JWTs.
	JWTSessionChecker SessionExistsChecker

	// ExpectedAudienceResolver returns the resource identifiers this
	// deployment answers to for the app that minted the presented token.
	// Nil, or an empty result, disables the check.
	//
	// Per app rather than per process: two apps in one deployment are two
	// different resources, and a token minted for one must not authenticate at
	// the other. This runs per request on the same path as
	// CookieNameResolver, so it should read a cached setting rather than hit
	// the database.
	ExpectedAudienceResolver ExpectedAudienceResolver
	// PrincipalResolver, when set, resolves the session's subject onto the
	// context as a *principal.Principal after every auth path that produces a
	// session. Threaded through this config (rather than as a standalone
	// parameter on each middleware constructor) so it defaults to nil and
	// every existing caller keeps working unchanged.
	PrincipalResolver PrincipalResolver
}

// AuthMiddleware extracts the session token from the Authorization header,
// resolves the session and user, and stores them in context.
// This middleware is the forge.Scope producer — it resolves the authenticated
// identity and sets AppID/OrgID on context for all downstream extensions.
func AuthMiddleware(resolveSession SessionResolver, resolveUser UserResolver, logger log.Logger, binding ...SessionBindingConfig) forge.Middleware {
	var bindCfg SessionBindingConfig
	if len(binding) > 0 {
		bindCfg = binding[0]
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			cookieName := resolveCookieName(bindCfg.CookieNameResolver, ctx.Context())
			token := extractBearerToken(ctx.Request(), cookieName)
			if token == "" {
				return next(ctx)
			}

			sess, err := resolveSession(token)
			if err != nil {
				logger.Debug("auth middleware: invalid session token",
					log.String("error", err.Error()),
				)
				return next(ctx)
			}

			// Cross-tenant guard: a session minted under one app must not be
			// honored when the caller presents a different app's publishable key.
			if requestAppIDMismatch(ctx.Context(), sess.AppID.String()) {
				logger.Warn("auth middleware: session app mismatch (publishable key switched)",
					log.String("session_id", sess.ID.String()),
				)
				return next(ctx)
			}

			// Resource indicator guard (RFC 8707): a token audienced at a resource
			// this deployment does not answer to must not authenticate here. Values
			// are resource URIs supplied by the caller, so they are never logged.
			if bindCfg.ExpectedAudienceResolver != nil {
				expected := bindCfg.ExpectedAudienceResolver(ctx.Context(), sess.AppID.String())
				if !audienceAllowed(sess.Audience, expected) {
					logger.Warn("auth middleware: session audience mismatch",
						log.String("session_id", sess.ID.String()),
					)
					return next(ctx)
				}
			}

			// Session binding: validate IP and/or device match
			if bindCfg.BindToIP && sess.IPAddress != "" {
				clientIP := clientIPFromRequest(ctx.Request())
				if clientIP != sess.IPAddress {
					logger.Warn("auth middleware: session IP mismatch",
						log.String("session_ip", sess.IPAddress),
						log.String("client_ip", clientIP),
						log.String("session_id", sess.ID.String()),
					)
					return forge.Unauthorized("session bound to different IP address")
				}
			}
			if bindCfg.BindToDevice && sess.UserAgent != "" {
				ua := ctx.Request().UserAgent()
				if ua != sess.UserAgent {
					logger.Warn("auth middleware: session device mismatch",
						log.String("session_ua", sess.UserAgent),
						log.String("client_ua", ua),
						log.String("session_id", sess.ID.String()),
					)
					return forge.Unauthorized("session bound to different device")
				}
			}

			goCtx := ctx.Context()
			goCtx = WithSession(goCtx, sess)
			goCtx = WithSessionID(goCtx, sess.ID)
			goCtx = WithAppID(goCtx, sess.AppID)
			goCtx = WithUserID(goCtx, sess.UserID)

			// Set environment ID from session (if present).
			if sess.EnvID.Prefix() != "" {
				goCtx = WithEnvID(goCtx, sess.EnvID)
			}

			// Detect impersonation
			if imp := sess.ImpersonatedBy(); imp.Prefix() != "" {
				goCtx = WithImpersonator(goCtx, imp)
			}
			goCtx = setPrincipalContext(goCtx, sess, bindCfg.PrincipalResolver, logger)

			if sess.OrgID.Prefix() != "" {
				goCtx = WithOrgID(goCtx, sess.OrgID)
				goCtx = forge.WithScope(goCtx, forge.NewOrgScope(sess.AppID.String(), sess.OrgID.String()))
			} else {
				goCtx = forge.WithScope(goCtx, forge.NewAppScope(sess.AppID.String()))
			}

			u, err := resolveUser(sess.UserID.String())
			if err != nil {
				logger.Warn("auth middleware: failed to resolve user",
					log.String("user_id", sess.UserID.String()),
					log.String("error", err.Error()),
				)
				ctx.WithContext(goCtx)
				return next(ctx)
			}
			goCtx = WithUser(goCtx, u)

			ctx.WithContext(goCtx)
			return next(ctx)
		}
	}
}

// AuthMiddlewareWithStrategies creates middleware that tries bearer-session
// resolution first, then falls back to the strategy registry for alternative
// auth methods (API keys, etc.). Bearer tokens with the "ask_" prefix skip
// session resolution and go directly to the strategy chain.
//
// When a JWTValidator is provided, JWT tokens (detected by containing two
// dots) are validated stateless — no DB lookup needed for the access token.
//
// Flow:
//  1. Extract bearer token from Authorization header
//  2. If token is JWT and jwtValidator is set → validate stateless
//  3. If token exists and is NOT an API key prefix → resolve as session
//  4. If no session resolved → try strategies.Authenticate(ctx, r)
//  5. If strategy succeeds → set context with User, Session, AppID, Scope
//  6. If all fail → continue unauthenticated (RequireAuth enforces)
func AuthMiddlewareWithStrategies(
	resolveSession SessionResolver,
	resolveUser UserResolver,
	strategies StrategyAuthenticator,
	logger log.Logger,
	binding ...SessionBindingConfig,
) forge.Middleware {
	var bindCfg SessionBindingConfig
	if len(binding) > 0 {
		bindCfg = binding[0]
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			cookieName := resolveCookieName(bindCfg.CookieNameResolver, ctx.Context())
			token := extractBearerToken(ctx.Request(), cookieName)

			// Try bearer session resolution first (skip if token looks like an API key).
			if token != "" && !isAPIKeyToken(token) {
				if resolved := trySessionAuth(ctx, token, resolveSession, resolveUser, logger, bindCfg); resolved {
					return next(ctx)
				}
			}

			// Fall back to strategy registry.
			if strategies != nil {
				if resolved := tryStrategyAuth(ctx, strategies, resolveUser, bindCfg.PrincipalResolver, logger); resolved {
					return next(ctx)
				}
			}

			// No auth resolved — continue unauthenticated.
			return next(ctx)
		}
	}
}

// AuthMiddlewareWithJWT creates middleware that supports JWT token validation
// in addition to opaque session tokens and strategy auth. JWT tokens are
// validated stateless (no DB lookup for the access token itself).
func AuthMiddlewareWithJWT(
	resolveSession SessionResolver,
	resolveUser UserResolver,
	strategies StrategyAuthenticator,
	jwtValidator JWTValidator,
	logger log.Logger,
	binding ...SessionBindingConfig,
) forge.Middleware {
	var bindCfg SessionBindingConfig
	if len(binding) > 0 {
		bindCfg = binding[0]
	}

	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			cookieName := resolveCookieName(bindCfg.CookieNameResolver, ctx.Context())
			token := extractBearerToken(ctx.Request(), cookieName)

			if token != "" {
				// JWT detection: tokens with two dots are JWTs.
				if tokenformat.IsJWT(token) && jwtValidator != nil {
					switch tryJWTAuth(ctx, token, jwtValidator, resolveUser, logger, bindCfg) {
					case jwtAuthAuthenticated:
						return next(ctx)
					case jwtAuthRefused:
						// The signature verified, so this token is one of ours,
						// and something below the signature refused it. Stop
						// here. Access tokens are stored as sess.Token too, so
						// falling through would look the same string up in the
						// session store and re-run the checks against the
						// session row instead of the claims, handing back the
						// authentication that was just denied.
						return next(ctx)
					case jwtAuthNotHandled:
						// Not a token this deployment can validate. Something
						// else may still claim it, so carry on.
					}
				}

				// Try opaque session resolution (skip API key prefixed tokens).
				if !isAPIKeyToken(token) {
					if resolved := trySessionAuth(ctx, token, resolveSession, resolveUser, logger, bindCfg); resolved {
						return next(ctx)
					}
				}
			}

			// Fall back to strategy registry.
			if strategies != nil {
				if resolved := tryStrategyAuth(ctx, strategies, resolveUser, bindCfg.PrincipalResolver, logger); resolved {
					return next(ctx)
				}
			}

			return next(ctx)
		}
	}
}

// jwtAuthResult says what tryJWTAuth decided about a bearer token.
//
// A plain bool cannot carry this. "I could not validate this" and "I validated
// this and it is refused" both used to read as false, and the caller answered
// both by trying the opaque session store with the same string. Access tokens
// are stored as sess.Token, so that second attempt would find the row behind
// the very token the first attempt rejected, and check the row's audience
// rather than the aud claim.
type jwtAuthResult int

const (
	// jwtAuthNotHandled means the string did not validate as a JWT this
	// deployment issued. Other authentication methods may still claim it.
	jwtAuthNotHandled jwtAuthResult = iota

	// jwtAuthAuthenticated means the claims verified and the request context
	// now carries the identity.
	jwtAuthAuthenticated

	// jwtAuthRefused means the signature verified, so the token is one of
	// ours, and a check below the signature rejected it. Nothing else may
	// re-authenticate it.
	jwtAuthRefused
)

// tryJWTAuth validates a JWT token stateless and sets context from claims.
// When a session checker is provided, the session is cross-checked against
// the store to support JWT revocation and IP/device binding.
//
// Every rejection past the signature check returns jwtAuthRefused, not
// jwtAuthNotHandled. Once the signature verifies, this deployment minted the
// token and this function is the authority on it.
func tryJWTAuth(
	ctx forge.Context,
	token string,
	validator JWTValidator,
	resolveUser UserResolver,
	logger log.Logger,
	bindCfg SessionBindingConfig,
) jwtAuthResult {
	claims, err := validator.ValidateJWT(token)
	if err != nil {
		logger.Debug("auth middleware: JWT validation failed",
			log.String("error", err.Error()),
		)
		return jwtAuthNotHandled
	}

	// Cross-tenant guard: a JWT minted under one app must not be honored when
	// the caller presents a different app's publishable key.
	if requestAppIDMismatch(ctx.Context(), claims.AppID) {
		logger.Warn("auth middleware: JWT app mismatch (publishable key switched)",
			log.String("session_id", claims.SessionID),
		)
		return jwtAuthRefused
	}

	// Resource indicator guard (RFC 8707): a token audienced at a resource
	// this deployment does not answer to must not authenticate here. Values
	// are resource URIs supplied by the caller, so they are never logged.
	if bindCfg.ExpectedAudienceResolver != nil {
		expected := bindCfg.ExpectedAudienceResolver(ctx.Context(), claims.AppID)
		if !audienceAllowed(claims.Audience, expected) {
			logger.Warn("auth middleware: JWT audience mismatch",
				log.String("session_id", claims.SessionID),
			)
			return jwtAuthRefused
		}
	}

	// When a session checker is configured, cross-check the JWT's session ID
	// against the store. This enables revocation and IP/device binding for JWTs.
	// The checker returns (nil, nil) when the feature is disabled via settings.
	if bindCfg.JWTSessionChecker != nil && claims.SessionID != "" {
		sess, sessErr := bindCfg.JWTSessionChecker(claims.SessionID)
		if sessErr != nil {
			logger.Debug("auth middleware: JWT session not found in store (revoked?)",
				log.String("session_id", claims.SessionID),
			)
			return jwtAuthRefused
		}

		// sess == nil && sessErr == nil means "feature disabled, skip checks".
		if sess != nil {
			// Validate IP binding for JWT.
			if bindCfg.BindToIP && sess.IPAddress != "" {
				clientIP := clientIPFromRequest(ctx.Request())
				if clientIP != sess.IPAddress {
					logger.Warn("auth middleware: JWT session IP mismatch",
						log.String("session_ip", sess.IPAddress),
						log.String("client_ip", clientIP),
						log.String("session_id", sess.ID.String()),
					)
					return jwtAuthRefused
				}
			}

			// Validate device binding for JWT.
			if bindCfg.BindToDevice && sess.UserAgent != "" {
				ua := ctx.Request().UserAgent()
				if ua != sess.UserAgent {
					logger.Warn("auth middleware: JWT session device mismatch",
						log.String("session_ua", sess.UserAgent),
						log.String("client_ua", ua),
						log.String("session_id", sess.ID.String()),
					)
					return jwtAuthRefused
				}
			}
		}
	}

	goCtx := ctx.Context()

	// Build a virtual session from JWT claims (no DB record needed).
	//
	// Every id below arrives inside the token, so it is attacker-influenced
	// even though a valid signature is required to reach this code. These
	// parses used to be id.MustParse, which panics on anything malformed and
	// panicked outright on a machine token, whose sub is a service account id
	// and was empty before the principal claims existed. Refuse the token
	// instead: a panic in auth middleware is a bad failure mode to leave in
	// place for a case the signature check is the only thing preventing.
	parse := func(field, raw string) (id.ID, bool) {
		parsed, parseErr := id.Parse(raw)
		if parseErr != nil {
			logger.Warn("auth middleware: JWT carries a malformed id claim",
				log.String("claim", field),
				log.String("error", parseErr.Error()),
			)
			return id.Nil, false
		}
		return parsed, true
	}

	appID, ok := parse("app_id", claims.AppID)
	if !ok {
		return jwtAuthRefused
	}
	subjectID, ok := parse("sub", claims.UserID)
	if !ok {
		return jwtAuthRefused
	}

	goCtx = WithAppID(goCtx, appID)
	goCtx = WithAuthMethod(goCtx, "jwt")

	// An empty PrincipalKind means user, which is what every token minted
	// before that claim existed carries.
	kind := principal.Kind(claims.PrincipalKind)
	isUser := kind == "" || kind == principal.KindUser
	if isUser {
		goCtx = WithUserID(goCtx, subjectID)
	}

	if claims.SessionID != "" {
		sessionID, sidOK := parse("sid", claims.SessionID)
		if !sidOK {
			return jwtAuthRefused
		}
		goCtx = WithSessionID(goCtx, sessionID)
	}

	if claims.EnvID != "" {
		envID, envOK := parse("env_id", claims.EnvID)
		if !envOK {
			return jwtAuthRefused
		}
		goCtx = WithEnvID(goCtx, envID)
	}

	if claims.OrgID != "" {
		orgID, orgOK := parse("org_id", claims.OrgID)
		if !orgOK {
			return jwtAuthRefused
		}
		goCtx = WithOrgID(goCtx, orgID)
		goCtx = forge.WithScope(goCtx, forge.NewOrgScope(claims.AppID, claims.OrgID))
	} else {
		goCtx = forge.WithScope(goCtx, forge.NewAppScope(claims.AppID))
	}

	// A machine caller has no user row to resolve. Put the principal on the
	// context so PrincipalRefFrom finds it, and stop.
	if !isUser {
		goCtx = principal.NewContext(goCtx, &principal.Principal{
			Ref:    principal.Ref{Kind: kind, ID: subjectID.String()},
			AppID:  appID,
			Scopes: claims.Scopes,
		})
		ctx.WithContext(goCtx)
		return jwtAuthAuthenticated
	}

	// Resolve user from claims.
	u, err := resolveUser(claims.UserID)
	if err != nil {
		logger.Debug("auth middleware: JWT user resolution failed",
			log.String("user_id", claims.UserID),
			log.String("error", err.Error()),
		)
		ctx.WithContext(goCtx)
		return jwtAuthAuthenticated // Authenticated via JWT even if user lookup fails
	}
	goCtx = WithUser(goCtx, u)

	ctx.WithContext(goCtx)
	return jwtAuthAuthenticated
}

// trySessionAuth attempts to authenticate via session token resolution.
// Returns true if authentication succeeded and context was updated.
func trySessionAuth(
	ctx forge.Context,
	token string,
	resolveSession SessionResolver,
	resolveUser UserResolver,
	logger log.Logger,
	bindCfg SessionBindingConfig,
) bool {
	sess, err := resolveSession(token)
	if err != nil {
		logger.Debug("auth middleware: invalid session token",
			log.String("error", err.Error()),
		)
		return false
	}

	// Cross-tenant guard: a session minted under one app must not be honored
	// when the caller presents a different app's publishable key.
	if requestAppIDMismatch(ctx.Context(), sess.AppID.String()) {
		logger.Warn("auth middleware: session app mismatch (publishable key switched)",
			log.String("session_id", sess.ID.String()),
		)
		return false
	}

	// Resource indicator guard (RFC 8707): a token audienced at a resource
	// this deployment does not answer to must not authenticate here. Values
	// are resource URIs supplied by the caller, so they are never logged.
	if bindCfg.ExpectedAudienceResolver != nil {
		expected := bindCfg.ExpectedAudienceResolver(ctx.Context(), sess.AppID.String())
		if !audienceAllowed(sess.Audience, expected) {
			logger.Warn("auth middleware: session audience mismatch",
				log.String("session_id", sess.ID.String()),
			)
			return false
		}
	}

	// Session binding: validate IP and/or device match.
	if bindCfg.BindToIP && sess.IPAddress != "" {
		clientIP := clientIPFromRequest(ctx.Request())
		if clientIP != sess.IPAddress {
			logger.Warn("auth middleware: session IP mismatch",
				log.String("session_ip", sess.IPAddress),
				log.String("client_ip", clientIP),
				log.String("session_id", sess.ID.String()),
			)
			return false
		}
	}
	if bindCfg.BindToDevice && sess.UserAgent != "" {
		ua := ctx.Request().UserAgent()
		if ua != sess.UserAgent {
			logger.Warn("auth middleware: session device mismatch",
				log.String("session_ua", sess.UserAgent),
				log.String("client_ua", ua),
				log.String("session_id", sess.ID.String()),
			)
			return false
		}
	}

	setSessionContext(ctx, sess, resolveUser, bindCfg.PrincipalResolver, logger)
	return true
}

// tryStrategyAuth attempts to authenticate via the strategy registry.
// Returns true if a strategy successfully authenticated the request.
func tryStrategyAuth(
	ctx forge.Context,
	strategies StrategyAuthenticator,
	_ UserResolver,
	resolvePrincipal PrincipalResolver,
	logger log.Logger,
) bool {
	result, err := strategies.Authenticate(ctx.Context(), ctx.Request())
	if err != nil {
		// NotApplicableError means "no credentials of my flavor in
		// this request" — that's silent (a JWT request hitting an
		// API-key strategy is normal). Anything else is the
		// strategy explicitly rejecting credentials it found
		// (wrong key, expired, missing X-App-ID header, etc.) and
		// the operator needs to see the reason in their logs;
		// otherwise the 401 chain looks like an unexplained
		// "authentication required" with no diagnostic.
		var notApplicable strategy.NotApplicableError
		if errors.As(err, &notApplicable) {
			logger.Debug("auth middleware: strategy not applicable",
				log.String("reason", err.Error()),
			)
			// Even NotApplicable surfaces something — record it so
			// the operator can see *which* strategy bowed out and
			// which inbound headers were present. Without this, the
			// 401 chain blames "no strategy ran" with zero
			// information about what arrived on the wire.
			r := ctx.Request()
			present := summarizeAuthHeaders(r)
			ctx.WithContext(withAuthDebug(ctx.Context(),
				"strategy NotApplicable: "+err.Error()+"; inbound auth headers: "+present))
		} else {
			logger.Warn("auth middleware: strategy rejected credentials",
				log.String("error", err.Error()),
				log.String("path", ctx.Request().URL.Path),
				log.String("method", ctx.Request().Method),
			)
			// Stash the rejection reason on the context so RequireAuth
			// can echo it on 401 — gives operators a diagnostic
			// without needing access to the server logs.
			ctx.WithContext(withAuthDebug(ctx.Context(), "strategy: "+err.Error()))
		}
		return false
	}
	if result == nil {
		// Strategy returned no error but produced no result at all.
		ctx.WithContext(withAuthDebug(ctx.Context(), "strategy: returned nil result"))
		return false
	}
	isServiceAccount := result.Session != nil && !result.Session.IsHumanPrincipal()
	if result.User == nil && !isServiceAccount {
		// no user and not a service account — treat as unauthenticated
		ctx.WithContext(withAuthDebug(ctx.Context(), "strategy: returned no user (resolveUser nil or Result empty)"))
		return false
	}

	// Cross-tenant guard: credentials that resolve a session bound to one app
	// must not be honored when the caller presents a different app's
	// publishable key (e.g. an API key for app A sent with app B's pk header).
	if result.Session != nil && requestAppIDMismatch(ctx.Context(), result.Session.AppID.String()) {
		logger.Warn("auth middleware: strategy app mismatch (publishable key switched)",
			log.String("session_id", result.Session.ID.String()),
		)
		ctx.WithContext(withAuthDebug(ctx.Context(), "strategy: session app mismatch with request publishable key"))
		return false
	}

	goCtx := ctx.Context()

	// Set session context from strategy result.
	if result.Session != nil {
		goCtx = WithSession(goCtx, result.Session)
		goCtx = WithSessionID(goCtx, result.Session.ID)
		goCtx = WithAppID(goCtx, result.Session.AppID)
		if !result.Session.UserID.IsNil() {
			goCtx = WithUserID(goCtx, result.Session.UserID)
		}

		if result.Session.EnvID.Prefix() != "" {
			goCtx = WithEnvID(goCtx, result.Session.EnvID)
		}
		if result.Session.OrgID.Prefix() != "" {
			goCtx = WithOrgID(goCtx, result.Session.OrgID)
			goCtx = forge.WithScope(goCtx, forge.NewOrgScope(result.Session.AppID.String(), result.Session.OrgID.String()))
		} else {
			goCtx = forge.WithScope(goCtx, forge.NewAppScope(result.Session.AppID.String()))
		}
		goCtx = setPrincipalContext(goCtx, result.Session, resolvePrincipal, logger)
	}

	if result.User != nil {
		goCtx = WithUser(goCtx, result.User)
	}
	goCtx = WithAuthMethod(goCtx, "strategy")

	ctx.WithContext(goCtx)
	return true
}

// setPrincipalContext resolves the session's subject and puts it, and the
// actor chain, on the context.
//
// A resolution failure is logged and passed over rather than failing the
// request. The session already authenticated the caller; this is enrichment,
// and refusing the request over it would turn a principal-store blip into an
// outage on traffic that is otherwise fine.
func setPrincipalContext(
	goCtx context.Context, sess *session.Session, resolve PrincipalResolver, logger log.Logger,
) context.Context {
	if len(sess.Actors) > 0 {
		goCtx = WithActors(goCtx, sess.Actors)
	}
	if resolve == nil {
		return goCtx
	}
	ref := sess.Subject()
	if ref.IsZero() {
		return goCtx
	}
	p, err := resolve(ref)
	if err != nil {
		logger.Warn("auth middleware: failed to resolve principal",
			log.String("principal", ref.String()),
			log.String("error", err.Error()),
		)
		return goCtx
	}
	return WithPrincipal(goCtx, p)
}

// setSessionContext populates the forge context with session and user data.
func setSessionContext(
	ctx forge.Context, sess *session.Session, resolveUser UserResolver, resolvePrincipal PrincipalResolver, logger log.Logger,
) {
	goCtx := ctx.Context()
	goCtx = WithSession(goCtx, sess)
	goCtx = WithSessionID(goCtx, sess.ID)
	goCtx = WithAppID(goCtx, sess.AppID)
	goCtx = WithUserID(goCtx, sess.UserID)

	if sess.EnvID.Prefix() != "" {
		goCtx = WithEnvID(goCtx, sess.EnvID)
	}
	if imp := sess.ImpersonatedBy(); imp.Prefix() != "" {
		goCtx = WithImpersonator(goCtx, imp)
	}
	goCtx = setPrincipalContext(goCtx, sess, resolvePrincipal, logger)

	if sess.OrgID.Prefix() != "" {
		goCtx = WithOrgID(goCtx, sess.OrgID)
		goCtx = forge.WithScope(goCtx, forge.NewOrgScope(sess.AppID.String(), sess.OrgID.String()))
	} else {
		goCtx = forge.WithScope(goCtx, forge.NewAppScope(sess.AppID.String()))
	}

	u, err := resolveUser(sess.UserID.String())
	if err != nil {
		logger.Warn("auth middleware: failed to resolve user",
			log.String("user_id", sess.UserID.String()),
			log.String("error", err.Error()),
		)
		goCtx = WithAuthMethod(goCtx, "session")
		ctx.WithContext(goCtx)
		return
	}
	goCtx = WithUser(goCtx, u)
	goCtx = WithAuthMethod(goCtx, "session")

	ctx.WithContext(goCtx)
}

// RequireAuth returns a forge middleware that rejects unauthenticated requests.
//
// Authenticated means a resolved principal of any kind, not a user
// specifically. A machine caller carries no *user.User, so gating on one
// turned away every service account, agent and workload credential before it
// reached a handler.
func RequireAuth() forge.Middleware {
	return func(next forge.Handler) forge.Handler {
		return func(ctx forge.Context) error {
			if _, ok := PrincipalRefFrom(ctx.Context()); !ok {
				body := map[string]any{
					"error": "authentication required",
					"code":  http.StatusUnauthorized,
				}
				// Surface the strategy rejection reason only when the
				// operator has opted in via AUTHSOME_DEBUG_AUTH=1. The
				// reason carries credential-shape reconnaissance
				// (which header was missing, scheme used, presence of
				// app-id) that an unauthenticated caller in production
				// must never see.
				if authDebugEnabled() {
					if reason := authDebugFrom(ctx.Context()); reason != "" {
						body["debug_reason"] = reason
					} else {
						body["debug_reason"] = "no strategy ran (no credentials presented or all strategies returned NotApplicable)"
					}
				}

				// Written by hand: this envelope carries a conditional
				// debug_reason field, which the typed constructors cannot
				// express — they take a message only.
				return ctx.JSON(http.StatusUnauthorized, body)
			}
			return next(ctx)
		}
	}
}

// isAPIKeyToken returns true if the token has an API key prefix (ask_, sk_*, pk_*).
func isAPIKeyToken(token string) bool {
	return apikey.IsAPIKey(token)
}

// requestAppIDMismatch reports whether the request carries a resolved
// publishable-key AppID that differs from the credential's bound app. A
// session/JWT minted under one app must not be honored when the caller
// presents a different app's publishable key. Returns false when the request
// has no publishable-key app (nothing to compare) so bearer-only flows are
// unaffected.
func requestAppIDMismatch(ctx context.Context, boundAppID string) bool {
	reqAppID, ok := AppIDFrom(ctx)
	if !ok || reqAppID.IsNil() || boundAppID == "" {
		return false
	}
	return reqAppID.String() != boundAppID
}

// audienceAllowed reports whether a token may be used at this resource.
//
// An empty expected set means the deployment has not declared what it
// answers to, so no check is possible. An empty token audience means an
// unrestricted token, which every token issued before RFC 8707 support
// carries, so it passes. Anything else has to intersect.
func audienceAllowed(tokenAudience, expected []string) bool {
	if len(expected) == 0 || len(tokenAudience) == 0 {
		return true
	}
	want := make(map[string]struct{}, len(expected))
	for _, e := range expected {
		want[e] = struct{}{}
	}
	for _, a := range tokenAudience {
		if _, ok := want[a]; ok {
			return true
		}
	}
	return false
}

func extractBearerToken(r *http.Request, cookieName string) string {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return parts[1]
		}
	}
	// Fallback: read session token from httpOnly cookie set by the backend.
	if cookieName == "" {
		cookieName = "authsome_session_token"
	}
	if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	return ""
}

// resolveCookieName returns the session cookie name using the resolver, or the default.
func resolveCookieName(resolver CookieNameResolver, ctx context.Context) string { //nolint:revive // ctx not first param: resolver is the primary input
	if resolver != nil {
		if name := resolver(ctx); name != "" {
			return name
		}
	}
	return "authsome_session_token"
}

// clientIPFromRequest extracts the client IP from the request. It delegates to
// the trusted-proxy-aware resolver so X-Forwarded-For / X-Real-IP are honored
// only when the immediate peer is a trusted proxy (a direct client cannot spoof
// its IP). Kept as a thin alias so existing call sites read naturally.
func clientIPFromRequest(r *http.Request) string {
	return ClientIP(r)
}
