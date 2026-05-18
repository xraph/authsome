package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// rateLimitOpt returns a forge.WithMiddleware option for rate limiting the given endpoint,
// or nil if rate limiting is not enabled.
func (a *API) rateLimitOpt(limit int) []forge.RouteOption {
	rl := a.engine.RateLimiter()
	cfg := a.engine.Config().RateLimit
	if rl == nil || !cfg.Enabled {
		return nil
	}
	return []forge.RouteOption{
		forge.WithMiddleware(middleware.RateLimit(rl, middleware.RateLimitConfig{
			Limit:  limit,
			Window: cfg.Window(),
		})),
	}
}

// captchaOpt returns a forge.WithMiddleware option that gates the route on
// captcha verification when auth.captcha_required is true for the resolved
// app. The action label is recorded on audit events and (when supported by
// the provider) bound at widget render time.
//
// IMPORTANT ORDERING: this option must be appended AFTER rateLimitOpt so a
// single IP can't burn captcha quota by spamming, and AFTER any cheap
// validation but BEFORE the route's handler — which for /v1/signup includes
// the dummy-hash budget consumer. If captcha runs after the handler, every
// captcha-failed probe still pays argon2 cost server-side, turning the
// timing-oracle defense into a CPU-DoS amplifier.
func (a *API) captchaOpt(action string) []forge.RouteOption {
	mgr := a.engine.Settings()
	if mgr == nil {
		return nil
	}
	return []forge.RouteOption{
		forge.WithMiddleware(middleware.CaptchaMiddleware(middleware.CaptchaOptions{
			Settings:  mgr,
			Action:    action,
			Chronicle: a.engine.Chronicle(),
			Logger:    a.engine.Logger(),
		})),
	}
}

// ──────────────────────────────────────────────────
// Auth route registration
// ──────────────────────────────────────────────────

func (a *API) registerAuthRoutes(router forge.Router) error {
	rlCfg := a.engine.Config().RateLimit
	g := router.Group("/v1", forge.WithGroupTags("authentication"))

	signUpOpts := make([]forge.RouteOption, 0, 7) //nolint:mnd // base options + rate limit
	signUpOpts = append(signUpOpts,
		forge.WithSummary("Sign up"),
		forge.WithDescription("Creates a new user account and returns authentication tokens."),
		forge.WithOperationID("signUp"),
		forge.WithRequestSchema(SignUpRequest{}),
		forge.WithCreatedResponse(AuthResponse{}),
		forge.WithErrorResponses(),
	)
	signUpOpts = append(signUpOpts, a.rateLimitOpt(rlCfg.SignUpLimit)...)
	signUpOpts = append(signUpOpts, a.captchaOpt("signup")...)
	if err := g.POST("/signup", a.handleSignUp, signUpOpts...); err != nil {
		return err
	}

	signInOpts := make([]forge.RouteOption, 0, 7) //nolint:mnd // base options + rate limit
	signInOpts = append(signInOpts,
		forge.WithSummary("Sign in"),
		forge.WithDescription("Authenticates a user with email/username and password."),
		forge.WithOperationID("signIn"),
		forge.WithRequestSchema(SignInRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Authenticated", AuthResponse{}),
		forge.WithErrorResponses(),
	)
	signInOpts = append(signInOpts, a.rateLimitOpt(rlCfg.SignInLimit)...)
	signInOpts = append(signInOpts, a.captchaOpt("signin")...)
	if err := g.POST("/signin", a.handleSignIn, signInOpts...); err != nil {
		return err
	}

	if err := g.POST("/signout", a.handleSignOut,
		forge.WithSummary("Sign out"),
		forge.WithDescription("Terminates the current session."),
		forge.WithOperationID("signOut"),
		forge.WithResponseSchema(http.StatusOK, "Signed out", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	refreshOpts := make([]forge.RouteOption, 0, 7) //nolint:mnd // base options + rate limit
	refreshOpts = append(refreshOpts,
		forge.WithSummary("Refresh tokens"),
		forge.WithDescription("Exchanges a refresh token for new session and refresh tokens."),
		forge.WithOperationID("refreshTokens"),
		forge.WithRequestSchema(RefreshRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Refreshed tokens", TokenResponse{}),
		forge.WithErrorResponses(),
	)
	refreshOpts = append(refreshOpts, a.rateLimitOpt(rlCfg.RefreshLimit)...)
	return g.POST("/refresh", a.handleRefresh, refreshOpts...)
}

// ──────────────────────────────────────────────────
// Auth handlers
// ──────────────────────────────────────────────────

func (a *API) handleSignUp(ctx forge.Context, req *SignUpRequest) (*AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, forge.BadRequest("email and password are required")
	}

	appID, err := a.resolvePublicAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	httpReq := ctx.Request()
	u, sess, err := a.engine.SignUp(ctx.Context(), &account.SignUpRequest{
		AppID:     appID,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Metadata:  req.Metadata,
		IPAddress: clientIPFromRequest(httpReq),
		UserAgent: httpReq.UserAgent(),
	})
	if err != nil {
		// Enumeration resistance: a duplicate email must NOT be a probe-able
		// oracle. Return the same 201 response shape as a fresh signup with
		// no real session token instead of bubbling up 409. The existing
		// user is intentionally NOT signed in (that would be account
		// hijack); their session is left untouched and no cookie is set.
		//
		// NOTE: ideally we'd queue a "someone tried to register with your
		// email" notification to the legitimate owner here, but the engine
		// does not yet expose a notifier hook for this. Phase 2A Task 4
		// (default-on email verification) will give every fresh signup a
		// structurally identical "verification pending" path, at which
		// point this synthetic shape becomes indistinguishable from a real
		// pending verification. Until then the lack of a session token
		// is still ambiguous to an attacker — could be a fresh signup
		// awaiting verification — so it is strictly better than 409.
		if errors.Is(err, account.ErrEmailTaken) {
			// Run a dummy password hash on the duplicate path so the
			// HTTP-response timing is indistinguishable from a fresh
			// signup (which spends ~100ms in argon2id/bcrypt). Without
			// this, an attacker can probe /v1/signup with arbitrary
			// emails and use the response time to enumerate registered
			// addresses (duplicate ~1ms vs fresh ~100ms+). Result is
			// discarded.
			a.consumeDummyHashBudget(req.Password)
			return nil, ctx.JSON(http.StatusCreated, a.syntheticSignupResponse(req.Email, appID))
		}
		return nil, mapError(err)
	}

	a.setSessionCookie(ctx, sess.Token, a.sessionTokenMaxAge())
	return nil, ctx.JSON(http.StatusCreated, authResponse(u, sess))
}

// consumeDummyHashBudget runs the engine's configured password hashing
// algorithm against the supplied password and discards the result. Used on
// the duplicate-email path so the response time matches a real signup. Any
// error from the hash function is intentionally ignored — this is purely a
// time-budget consumer.
func (a *API) consumeDummyHashBudget(password string) {
	if password == "" {
		// Match the synthetic case where we still want to spend the
		// time budget. Hash a fixed sentinel.
		password = "x"
	}
	cfg := a.engine.Config().Password
	policy := account.PasswordPolicy{
		BcryptCost: cfg.BcryptCost,
		Algorithm:  cfg.Algorithm,
		Argon2Params: account.Argon2Params{
			Memory:      cfg.Argon2.Memory,
			Iterations:  cfg.Argon2.Iterations,
			Parallelism: cfg.Argon2.Parallelism,
			SaltLength:  cfg.Argon2.SaltLength,
			KeyLength:   cfg.Argon2.KeyLength,
		},
	}
	_, _ = account.HashPasswordWithPolicy(password, policy) //nolint:errcheck // dummy hash for timing budget
}

// syntheticSignupResponse returns a response shaped exactly like a real
// fresh-signup response — populated user, non-empty session_token /
// refresh_token, and a forward-dated expires_at — so an attacker cannot use
// the response shape itself to distinguish duplicate-email signups from
// fresh ones. The synthetic tokens are generated via crypto/rand and will
// not validate against the session store; any attacker probe that tries to
// USE one in a follow-up request gets the same "invalid session" response
// they would get from a fully forged token.
//
// This is a Phase 2A Task 4 stopgap. Once RequireEmailVerification defaults
// to true, fresh signups will also return empty tokens until verification
// completes, at which point this synthesis can be collapsed.
func (a *API) syntheticSignupResponse(email string, appID id.AppID) map[string]any {
	token, err := syntheticOpaqueToken()
	if err != nil {
		token = ""
	}
	refresh, err := syntheticOpaqueToken()
	if err != nil {
		refresh = ""
	}

	ttl := a.engine.Config().Session.TokenTTL
	if ttl <= 0 {
		ttl = time.Hour
	}
	expiresAt := time.Now().Add(ttl)

	syntheticUser := &user.User{
		ID:        id.NewUserID(),
		AppID:     appID,
		Email:     strings.ToLower(strings.TrimSpace(email)),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return map[string]any{
		"user":          syntheticUser,
		"session_token": token,
		"refresh_token": refresh,
		"expires_at":    expiresAt,
	}
}

// syntheticOpaqueToken returns a 64-character hex string (32 random bytes)
// matching the shape of real session tokens produced by account.NewSession.
func syntheticOpaqueToken() (string, error) {
	b := make([]byte, 32) //nolint:mnd // matches account.generateSecureToken default
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (a *API) handleSignIn(ctx forge.Context, req *SignInRequest) (*AuthResponse, error) {
	if req.Password == "" {
		return nil, forge.BadRequest("password is required")
	}
	if req.Email == "" && req.Username == "" {
		return nil, forge.BadRequest("email or username is required")
	}

	appID, err := a.resolvePublicAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
	}

	httpReq := ctx.Request()
	u, sess, err := a.engine.SignIn(ctx.Context(), &account.SignInRequest{
		AppID:     appID,
		Email:     req.Email,
		Username:  req.Username,
		Password:  req.Password,
		IPAddress: clientIPFromRequest(httpReq),
		UserAgent: httpReq.UserAgent(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	a.setSessionCookie(ctx, sess.Token, a.sessionTokenMaxAge())
	return nil, ctx.JSON(http.StatusOK, authResponse(u, sess))
}

func (a *API) handleSignOut(ctx forge.Context, _ *SignOutRequest) (*StatusResponse, error) {
	sessID, ok := middleware.SessionIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if err := a.engine.SignOut(ctx.Context(), sessID); err != nil {
		return nil, mapError(err)
	}

	a.deleteSessionCookie(ctx)
	resp := &StatusResponse{Status: "signed out"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleRefresh(ctx forge.Context, req *RefreshRequest) (*TokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, forge.BadRequest("refresh_token is required")
	}

	httpReq := ctx.Request()
	sess, err := a.engine.Refresh(ctx.Context(), req.RefreshToken, authsome.RefreshOpts{
		IPAddress: clientIPFromRequest(httpReq),
		UserAgent: httpReq.UserAgent(),
	})
	if err != nil {
		a.deleteSessionCookie(ctx)
		return nil, mapError(err)
	}

	a.setSessionCookie(ctx, sess.Token, a.sessionTokenMaxAge())
	resp := &TokenResponse{
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
		ExpiresAt:    sess.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	return nil, ctx.JSON(http.StatusOK, resp)
}

// ──────────────────────────────────────────────────
// Cookie helpers
// ──────────────────────────────────────────────────

// cookieConfig holds resolved cookie configuration from dynamic settings.
type cookieConfig struct {
	Name     string
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

// resolveCookieConfig reads cookie settings from the dynamic settings manager.
func (a *API) resolveCookieConfig(ctx forge.Context) cookieConfig {
	goCtx := ctx.Context()
	mgr := a.engine.Settings()
	opts := settings.ResolveOpts{}

	name, _ := settings.Get(goCtx, mgr, authsome.SettingCookieName, opts) //nolint:errcheck // best-effort settings
	if name == "" {
		name = "authsome_session_token"
	}
	domain, _ := settings.Get(goCtx, mgr, authsome.SettingCookieDomain, opts) //nolint:errcheck // best-effort settings
	path, _ := settings.Get(goCtx, mgr, authsome.SettingCookiePath, opts)     //nolint:errcheck // best-effort settings
	if path == "" {
		path = "/"
	}
	secureSetting, _ := settings.Get(goCtx, mgr, authsome.SettingCookieSecure, opts) //nolint:errcheck // best-effort settings
	httpOnly, _ := settings.Get(goCtx, mgr, authsome.SettingCookieHTTPOnly, opts)    //nolint:errcheck // best-effort settings
	sameSiteStr, _ := settings.Get(goCtx, mgr, authsome.SettingCookieSameSite, opts) //nolint:errcheck // best-effort settings

	// Auto-detect secure: if setting is true but request is plain HTTP, disable for dev.
	r := ctx.Request()
	isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	secure := secureSetting && isHTTPS

	sameSite := http.SameSiteLaxMode
	switch sameSiteStr {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}

	return cookieConfig{
		Name: name, Domain: domain, Path: path,
		Secure: secure, HTTPOnly: httpOnly, SameSite: sameSite,
	}
}

// setSessionCookie sets the httpOnly session token cookie on the response.
func (a *API) setSessionCookie(ctx forge.Context, token string, maxAge int) {
	cc := a.resolveCookieConfig(ctx)
	cookie := &http.Cookie{ // #nosec G124 -- secure/httpOnly/sameSite resolved dynamically via cookieConfig
		Name:     cc.Name,
		Value:    token,
		Path:     cc.Path,
		Domain:   cc.Domain,
		MaxAge:   maxAge,
		HttpOnly: cc.HTTPOnly,
		Secure:   cc.Secure,
		SameSite: cc.SameSite,
	}
	http.SetCookie(ctx.Response(), cookie)
}

// deleteSessionCookie clears the session cookie.
func (a *API) deleteSessionCookie(ctx forge.Context) {
	cc := a.resolveCookieConfig(ctx)
	cookie := &http.Cookie{ // #nosec G124 -- secure/httpOnly/sameSite resolved dynamically via cookieConfig
		Name:     cc.Name,
		Value:    "",
		Path:     cc.Path,
		Domain:   cc.Domain,
		MaxAge:   -1,
		HttpOnly: cc.HTTPOnly,
		Secure:   cc.Secure,
		SameSite: cc.SameSite,
	}
	http.SetCookie(ctx.Response(), cookie)
}

// sessionTokenMaxAge returns the session token TTL in seconds from engine config.
func (a *API) sessionTokenMaxAge() int {
	maxAge := int(a.engine.Config().Session.TokenTTL.Seconds())
	if maxAge <= 0 {
		maxAge = 3600
	}
	return maxAge
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func (a *API) resolveAppID(raw string) (id.AppID, error) {
	if raw != "" {
		return id.ParseAppID(raw)
	}
	return id.ParseAppID(a.engine.Config().AppID)
}

// resolvePublicAppID resolves the app for an unauthenticated public-auth
// request (signup, signin, forgot-password, resend verification).
//
// Resolution order:
//  1. AppID stashed on the context by PublishableKeyMiddleware — when a
//     pk_* is on the request, the publishable key is the source of truth.
//  2. Explicit req.AppID body field — for server-to-server callers that
//     don't ship a publishable key.
//  3. 400 — never silently fall back to the platform app, even on a
//     single-app install. A signup that doesn't say which app it belongs to
//     is a programmer error, not a default; and the silent fallback was the
//     mechanism by which non-platform tenants' users were leaking into the
//     platform tenant.
//
// If both a pk-derived context AppID AND a body app_id are present and they
// disagree, this is a misconfigured client (or tampering): fail closed.
func (a *API) resolvePublicAppID(ctx forge.Context, raw string) (id.AppID, error) {
	fromKey, hasKey := middleware.AppIDFrom(ctx.Context())
	if hasKey && raw != "" {
		parsed, err := id.ParseAppID(raw)
		if err != nil {
			return id.AppID{}, forge.BadRequest("invalid app_id")
		}
		if parsed != fromKey {
			return id.AppID{}, forge.BadRequest("app_id does not match publishable key")
		}
		return fromKey, nil
	}
	if hasKey {
		return fromKey, nil
	}
	if raw != "" {
		return id.ParseAppID(raw)
	}
	return id.AppID{}, forge.BadRequest("app context required: send X-Publishable-Key header or app_id in the body")
}

func authResponse(u *user.User, sess *session.Session) map[string]any {
	return map[string]any{
		"user":          u,
		"session_token": sess.Token,
		"refresh_token": sess.RefreshToken,
		"expires_at":    sess.ExpiresAt,
	}
}

// clientIPFromRequest extracts the client IP from the request, checking
// X-Forwarded-For and X-Real-IP headers before falling back to RemoteAddr.
func clientIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i > 0 {
		return addr[:i]
	}
	return addr
}
