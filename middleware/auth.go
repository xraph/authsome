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
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/hook"
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

	// DPoPValidator enforces RFC 9449 proof-of-possession. When nil, DPoP is
	// not enforced at all, which is only correct in tests: the engine always
	// supplies one.
	DPoPValidator *dpop.Validator

	// DPoPNonceSigner mints the nonces served with a use_dpop_nonce challenge.
	// Nil disables nonce challenges.
	DPoPNonceSigner *dpop.NonceSigner

	// DPoPNonceRequired reports whether the app owning this session demands a
	// nonce. Nil means never.
	DPoPNonceRequired func(ctx context.Context, appID string) bool

	// DPoPAudit records a DPoP failure. Nil disables recording.
	DPoPAudit func(ctx context.Context, action string, md map[string]string)
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
			installDPoPRequestScope(ctx)
			cookieName := resolveCookieName(bindCfg.CookieNameResolver, ctx.Context())
			scheme, token := extractCredentialCtx(ctx.Context(), ctx.Request(), cookieName)
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

			if dpopErr := enforceDPoP(ctx, bindCfg, sess.DPoPJKT, scheme, token, sess.AppID.String(), sess.ID.String(), logger); dpopErr != nil {
				return dpopErr
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
			installDPoPRequestScope(ctx)
			cookieName := resolveCookieName(bindCfg.CookieNameResolver, ctx.Context())
			scheme, token := extractCredentialCtx(ctx.Context(), ctx.Request(), cookieName)

			// Try bearer session resolution first (skip if token looks like an API key).
			if token != "" && !isAPIKeyToken(token) {
				resolved, err := trySessionAuth(ctx, scheme, token, resolveSession, resolveUser, logger, bindCfg)
				if err != nil {
					return err
				}
				if resolved {
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
			installDPoPRequestScope(ctx)
			cookieName := resolveCookieName(bindCfg.CookieNameResolver, ctx.Context())
			scheme, token := extractCredentialCtx(ctx.Context(), ctx.Request(), cookieName)

			if token != "" {
				// JWT detection: tokens with two dots are JWTs.
				if tokenformat.IsJWT(token) && jwtValidator != nil {
					// A DPoP enforcement failure comes back as an error and
					// not as a result, because it has to surface as an actual
					// 401 carrying the RFC 9449 challenge rather than fall
					// through to any other authentication path.
					result, dpopErr := tryJWTAuth(ctx, scheme, token, jwtValidator, resolveUser, logger, bindCfg)
					if dpopErr != nil {
						return dpopErr
					}
					switch result {
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
					resolved, err := trySessionAuth(ctx, scheme, token, resolveSession, resolveUser, logger, bindCfg)
					if err != nil {
						return err
					}
					if resolved {
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
//
// The error return is reserved for one case: a DPoP-bound token that failed
// proof-of-possession enforcement. That failure has to reach the caller as an
// actual 401 carrying the RFC 9449 challenge, never as a silent "try
// something else" signal, or a bound token could be laundered through a
// fallback path that skips enforcement entirely.
func tryJWTAuth(
	ctx forge.Context,
	scheme, token string,
	validator JWTValidator,
	resolveUser UserResolver,
	logger log.Logger,
	bindCfg SessionBindingConfig,
) (jwtAuthResult, error) {
	claims, err := validator.ValidateJWT(token)
	if err != nil {
		logger.Debug("auth middleware: JWT validation failed",
			log.String("error", err.Error()),
		)
		return jwtAuthNotHandled, nil
	}

	// Cross-tenant guard: a JWT minted under one app must not be honored when
	// the caller presents a different app's publishable key.
	if requestAppIDMismatch(ctx.Context(), claims.AppID) {
		logger.Warn("auth middleware: JWT app mismatch (publishable key switched)",
			log.String("session_id", claims.SessionID),
		)
		return jwtAuthRefused, nil
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
			return jwtAuthRefused, nil
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
			return jwtAuthRefused, nil
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
					return jwtAuthRefused, nil
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
					return jwtAuthRefused, nil
				}
			}
		}
	}

	if dpopErr := enforceDPoP(ctx, bindCfg, claims.DPoPJKT, scheme, token, claims.AppID, claims.SessionID, logger); dpopErr != nil {
		return jwtAuthNotHandled, dpopErr
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
		return jwtAuthRefused, nil
	}
	subjectID, ok := parse("sub", claims.UserID)
	if !ok {
		return jwtAuthRefused, nil
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
			return jwtAuthRefused, nil
		}
		goCtx = WithSessionID(goCtx, sessionID)
	}

	if claims.EnvID != "" {
		envID, envOK := parse("env_id", claims.EnvID)
		if !envOK {
			return jwtAuthRefused, nil
		}
		goCtx = WithEnvID(goCtx, envID)
	}

	if claims.OrgID != "" {
		orgID, orgOK := parse("org_id", claims.OrgID)
		if !orgOK {
			return jwtAuthRefused, nil
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
		return jwtAuthAuthenticated, nil
	}

	// Resolve user from claims.
	u, err := resolveUser(claims.UserID)
	if err != nil {
		logger.Debug("auth middleware: JWT user resolution failed",
			log.String("user_id", claims.UserID),
			log.String("error", err.Error()),
		)
		ctx.WithContext(goCtx)
		return jwtAuthAuthenticated, nil // Authenticated via JWT even if user lookup fails
	}
	goCtx = WithUser(goCtx, u)

	ctx.WithContext(goCtx)
	return jwtAuthAuthenticated, nil
}

// trySessionAuth attempts to authenticate via session token resolution.
// Returns (true, nil) if authentication succeeded and context was updated,
// (false, nil) when the token simply did not resolve to a session (so the
// caller can fall back to strategy auth), and (false, err) when a
// DPoP-bound session failed proof-of-possession enforcement. That last case
// must surface as an actual 401 rather than "not resolved": otherwise a
// bound token with a missing or invalid proof would just fall through to
// the strategy chain and, finding nothing there either, be treated as an
// ordinary unauthenticated request instead of a rejected one.
func trySessionAuth(
	ctx forge.Context,
	scheme, token string,
	resolveSession SessionResolver,
	resolveUser UserResolver,
	logger log.Logger,
	bindCfg SessionBindingConfig,
) (bool, error) {
	sess, err := resolveSession(token)
	if err != nil {
		logger.Debug("auth middleware: invalid session token",
			log.String("error", err.Error()),
		)
		return false, nil
	}

	// Cross-tenant guard: a session minted under one app must not be honored
	// when the caller presents a different app's publishable key.
	if requestAppIDMismatch(ctx.Context(), sess.AppID.String()) {
		logger.Warn("auth middleware: session app mismatch (publishable key switched)",
			log.String("session_id", sess.ID.String()),
		)
		return false, nil
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
			return false, nil
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
			return false, nil
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
			return false, nil
		}
	}

	if dpopErr := enforceDPoP(ctx, bindCfg, sess.DPoPJKT, scheme, token, sess.AppID.String(), sess.ID.String(), logger); dpopErr != nil {
		return false, dpopErr
	}

	setSessionContext(ctx, sess, resolveUser, bindCfg.PrincipalResolver, logger)
	return true, nil
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

// Credential schemes returned by extractCredential.
const (
	schemeBearer = "bearer"
	schemeDPoP   = "dpop"
	schemeCookie = "cookie"
)

// cookieBridgeKey carries the token that a cookie-to-header bridge wrote into
// the Authorization header for this request.
type cookieBridgeKey struct{}

// WithCookieBridgedToken records that the Authorization header on this request
// was synthesized from a session cookie rather than sent by the client.
//
// The engine bridges the cookie into "Authorization: Bearer <token>" so every
// downstream reader sees one credential shape. That is convenient and it also
// erased the one fact RFC 9449 section 7.1 turns on, namely whether the
// credential arrived in an Authorization header at all. Recording the bridged
// value here puts the fact back without changing what downstream readers see.
func WithCookieBridgedToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, cookieBridgeKey{}, token)
}

// cookieBridgedTokenFrom returns the token the bridge wrote, or "".
func cookieBridgedTokenFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(cookieBridgeKey{}).(string) //nolint:errcheck // type-safe via key
	return v
}

// ExtractCredentialFromContext pulls the token out of a request and reports
// which scheme carried it, accounting for a cookie the engine bridged into the
// Authorization header earlier in this request. Auth paths outside this
// package that hold a request context should prefer it over ExtractCredential,
// which cannot tell a bridged cookie from a real Bearer header.
func ExtractCredentialFromContext(ctx context.Context, r *http.Request, cookieName string) (scheme, token string) {
	return extractCredentialCtx(ctx, r, cookieName)
}

// extractCredentialCtx is extractCredential plus the knowledge of whether the
// Authorization header holds a value the cookie bridge put there.
//
// The comparison is against the exact token the bridge recorded, so a request
// that also carries a real Authorization header keeps the bearer scheme and
// the strict rule that goes with it.
func extractCredentialCtx(ctx context.Context, r *http.Request, cookieName string) (scheme, token string) {
	scheme, token = extractCredential(r, cookieName)
	if scheme == schemeBearer && token != "" && token == cookieBridgedTokenFrom(ctx) {
		return schemeCookie, token
	}
	return scheme, token
}

// extractCredential pulls the token out of the request and reports which
// scheme carried it.
//
// The scheme matters because a DPoP-bound token must be presented under the
// DPoP scheme (RFC 9449 section 7.1). Returning it here rather than discarding
// it lets the enforcement path refuse a bound token sent as Bearer.
func extractCredential(r *http.Request, cookieName string) (scheme, token string) {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) == 2 {
			switch {
			case strings.EqualFold(parts[0], "bearer"):
				return schemeBearer, parts[1]
			case strings.EqualFold(parts[0], "dpop"):
				return schemeDPoP, parts[1]
			}
		}
	}
	if cookieName == "" {
		cookieName = "authsome_session_token"
	}
	if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
		return schemeCookie, cookie.Value
	}
	return "", ""
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

// installDPoPRequestScope marks the request context so that every enforcement
// point on this request shares one record of which proofs already cleared the
// replay cache.
//
// Without it the second point to look at a proof sees the jti the first one
// recorded and refuses the request as a replay of itself. That is not
// hypothetical: the global auth middleware and a route's own session auth
// provider both enforce, and so do the refresh and token endpoints when a
// bound token rides along in the Authorization header.
//
// Requests with no DPoP header carry no proof for anything to re-check, so
// they skip the allocation and take the path they took before this existed.
func installDPoPRequestScope(ctx forge.Context) {
	if ctx.Request().Header.Get("DPoP") == "" {
		return
	}
	ctx.WithContext(dpop.WithRequestScope(ctx.Context()))
}

// dpopCheck is the outcome of applying RFC 9449 to a presented credential. It
// makes no HTTP response decisions, so the middleware (which can answer with a
// challenge) and the forge auth provider (which only gets to say yes or no)
// run one rule instead of two that drift apart.
type dpopCheck struct {
	ok    bool
	proof *dpop.Proof

	// code is the RFC 9449 error code for the challenge. Empty means refuse
	// without a challenge: the server could not check the binding at all, so
	// there is no proof the client could retry with.
	code string

	// action and reason describe the audit record. action is empty when
	// there is nothing worth auditing.
	action string
	reason string

	// nonceRequired carries what the app's nonce policy said, so a caller
	// that rotates the nonce does not resolve the setting a second time.
	nonceRequired bool
}

// checkDPoP runs the RFC 9449 checks for a credential already known to be
// bound. Callers deal with the unbound case before getting here.
func checkDPoP(
	ctx context.Context,
	cfg SessionBindingConfig,
	r *http.Request,
	boundJKT, scheme, token, appID string,
	logger log.Logger,
) dpopCheck {
	if cfg.DPoPValidator == nil {
		// A bound token with no validator configured cannot be checked. Refuse
		// rather than admit it: failing closed here turns a misconfiguration
		// into an outage, and failing open turns it into a silent bypass.
		logger.Error("auth middleware: DPoP-bound token but no validator configured")
		return dpopCheck{}
	}

	// RFC 9449 section 7.1 governs the Authorization header, so strict scheme
	// matching applies to credentials that actually arrived in one. A cookie is
	// not an Authorization header and has no scheme to match, so a bound
	// session presented by cookie is honoured when a valid proof accompanies it
	// and refused when one does not. The binding still holds either way: what
	// changes is only which of the two ways of failing applies.
	//
	// Bearer stays strict. A bound token sent as "Authorization: Bearer" with a
	// proof alongside is exactly the case section 7.1 speaks to, and it is
	// still refused.
	if scheme != schemeDPoP && scheme != schemeCookie {
		logger.Warn("auth middleware: DPoP-bound token presented under the wrong scheme",
			log.String("scheme", scheme),
		)
		return dpopCheck{code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "wrong_scheme"}
	}

	raw := r.Header.Get("DPoP")
	if raw == "" {
		return dpopCheck{code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "missing_proof"}
	}

	proof, err := dpop.Parse(raw)
	if err != nil {
		logger.Warn("auth middleware: DPoP proof rejected",
			log.String("error", err.Error()),
		)
		return dpopCheck{code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "parse_failed"}
	}

	nonceRequired := cfg.DPoPNonceRequired != nil && cfg.DPoPNonceRequired(ctx, appID)

	err = cfg.DPoPValidator.Validate(ctx, proof, dpop.Expectation{
		Method:        r.Method,
		URL:           RequestURL(r),
		AccessToken:   token,
		ExpectedJKT:   boundJKT,
		NonceRequired: nonceRequired,
	})
	switch {
	case err == nil:
		return dpopCheck{ok: true, proof: proof, nonceRequired: nonceRequired}

	case errors.Is(err, dpop.ErrNonceRequired), errors.Is(err, dpop.ErrNonceMismatch):
		return dpopCheck{proof: proof, code: dpopErrUseNonce}

	case errors.Is(err, dpop.ErrKeyMismatch):
		// A structurally valid proof for the wrong key has no innocent
		// explanation the way a changed IP does, so it is logged apart from
		// ordinary client breakage.
		logger.Warn("auth middleware: DPoP key mismatch",
			log.String("bound_jkt", boundJKT),
			log.String("proof_jkt", proof.JKT),
		)
		return dpopCheck{proof: proof, code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "key_mismatch"}

	case errors.Is(err, dpop.ErrReplayed):
		// Rare, and interesting when it happens: it means somebody captured a
		// proof and the token it was minted for.
		logger.Warn("auth middleware: DPoP proof replayed",
			log.String("proof_jkt", proof.JKT),
		)
		return dpopCheck{proof: proof, code: dpopErrInvalidToken, action: hook.ActionDPoPProofReplayed}

	default:
		logger.Warn("auth middleware: DPoP validation failed",
			log.String("error", err.Error()),
		)
		return dpopCheck{proof: proof, code: dpopErrInvalidToken, action: hook.ActionDPoPProofInvalid, reason: "validation_failed"}
	}
}

// RFC 9449 error codes carried in a WWW-Authenticate challenge.
const (
	dpopErrInvalidToken = "invalid_token"
	dpopErrUseNonce     = "use_dpop_nonce"
)

// auditDPoP records a DPoP failure. session_id and app_id identify who tripped
// the rule; the raw proof and the token itself never go in, since metadata
// routinely ends up in longer-lived audit storage than the request that
// produced it.
func auditDPoP(ctx context.Context, cfg SessionBindingConfig, result dpopCheck, appID, sessionID string) {
	if cfg.DPoPAudit == nil || result.action == "" {
		return
	}
	md := map[string]string{"session_id": sessionID, "app_id": appID}
	if result.reason != "" {
		md["reason"] = result.reason
	}
	cfg.DPoPAudit(ctx, result.action, md)
}

// enforceDPoP applies RFC 9449 to a resolved credential.
//
// Enforcement follows the token, not the route and not the configuration. A
// token with no binding takes one string comparison and the identical path it
// took before this function existed, which is what lets DPoP roll out without
// a flag day. A token that is bound cannot be used without a matching proof,
// whatever the app or client mode says at the time of the request.
func enforceDPoP(
	ctx forge.Context,
	cfg SessionBindingConfig,
	boundJKT, scheme, token, appID, sessionID string,
	logger log.Logger,
) error {
	if boundJKT == "" {
		return nil // Unbound. Nothing to enforce.
	}

	result := checkDPoP(ctx.Context(), cfg, ctx.Request(), boundJKT, scheme, token, appID, logger)
	auditDPoP(ctx.Context(), cfg, result, appID, sessionID)

	switch {
	case result.ok:
		// Rotate the nonce before the client's current one expires, so a
		// long-lived client never has to eat a challenge round trip.
		if result.nonceRequired && cfg.DPoPNonceSigner != nil && cfg.DPoPNonceSigner.NeedsRefresh(result.proof.Nonce) {
			ctx.Response().Header().Set("DPoP-Nonce", cfg.DPoPNonceSigner.Issue(result.proof.JKT))
		}
		return nil

	case result.code == "":
		// The binding could not be checked at all, so there is no retry to
		// point the client at and no challenge to write.
		return forge.Unauthorized("token requires proof of possession")

	case result.code == dpopErrUseNonce:
		return dpopChallenge(ctx, cfg, result.proof.JKT, result.code)

	default:
		return dpopChallenge(ctx, cfg, "", result.code)
	}
}

// ErrDPoPBindingFailed reports a bound token presented without an acceptable
// proof, for callers outside the middleware chain that only get to answer yes
// or no and cannot write an RFC 9449 challenge.
var ErrDPoPBindingFailed = errors.New("authsome: token requires proof of possession")

// EnforceDPoPForRequest applies the rule enforceDPoP applies, for a credential
// resolved outside the HTTP middleware chain. The forge auth providers are the
// case that needs it: they return an auth context rather than a response, so
// they cannot answer with a challenge, but the binding still has to hold. A
// caller that can write a response should go through the middleware instead,
// so the client learns which retry would work.
func EnforceDPoPForRequest(
	ctx context.Context,
	cfg SessionBindingConfig,
	r *http.Request,
	boundJKT, scheme, token, appID, sessionID string,
	logger log.Logger,
) error {
	if boundJKT == "" {
		return nil // Unbound. Nothing to enforce.
	}

	result := checkDPoP(ctx, cfg, r, boundJKT, scheme, token, appID, logger)
	auditDPoP(ctx, cfg, result, appID, sessionID)
	if result.ok {
		return nil
	}
	return ErrDPoPBindingFailed
}

// ExtractCredential pulls the token out of a request and reports which scheme
// carried it, for auth paths outside this package that have to read a
// credential the way the middleware reads it. cookieName may be empty for the
// default session cookie.
func ExtractCredential(r *http.Request, cookieName string) (scheme, token string) {
	return extractCredential(r, cookieName)
}

// dpopChallenge writes an RFC 9449 challenge and returns the 401 to propagate.
func dpopChallenge(ctx forge.Context, cfg SessionBindingConfig, jkt, code string) error {
	header := `DPoP error="` + code + `", algs="` + strings.Join(dpop.SupportedAlgs(), " ") + `"`
	ctx.Response().Header().Set("WWW-Authenticate", header)

	if code == "use_dpop_nonce" && cfg.DPoPNonceSigner != nil && jkt != "" {
		ctx.Response().Header().Set("DPoP-Nonce", cfg.DPoPNonceSigner.Issue(jkt))
	}
	return forge.Unauthorized("invalid proof of possession")
}
