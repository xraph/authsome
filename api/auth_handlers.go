package api

import (
	"net/http"
	"strings"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/rbac"
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

// ──────────────────────────────────────────────────
// Auth route registration
// ──────────────────────────────────────────────────

func (a *API) registerAuthRoutes(router forge.Router) error {
	base := a.engine.Config().BasePath
	rlCfg := a.engine.Config().RateLimit
	g := router.Group(base, forge.WithGroupTags("authentication"))

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

	return g.POST("/refresh", a.handleRefresh,
		forge.WithSummary("Refresh tokens"),
		forge.WithDescription("Exchanges a refresh token for new session and refresh tokens."),
		forge.WithOperationID("refreshTokens"),
		forge.WithRequestSchema(RefreshRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Refreshed tokens", TokenResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Auth handlers
// ──────────────────────────────────────────────────

func (a *API) handleSignUp(ctx forge.Context, req *SignUpRequest) (*AuthResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, forge.BadRequest("email and password are required")
	}

	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	httpReq := ctx.Request()
	u, sess, err := a.engine.SignUp(ctx.Context(), &account.SignUpRequest{
		AppID:     appID,
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		IPAddress: clientIPFromRequest(httpReq),
		UserAgent: httpReq.UserAgent(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	// If this is the first user for the platform app, assign platform_owner role.
	platformID := a.engine.PlatformAppID()
	if appID == platformID && !platformID.IsNil() {
		list, _ := a.engine.Store().ListUsers(ctx.Context(), &user.Query{AppID: appID, Limit: 2})
		if list != nil && list.Total == 1 {
			ownerRole, roleErr := a.engine.GetRoleBySlug(ctx.Context(), appID, rbac.PlatformOwnerSlug)
			if roleErr == nil && ownerRole != nil {
				_ = a.engine.AssignUserRole(ctx.Context(), &rbac.UserRole{
					UserID: u.ID.String(),
					RoleID: ownerRole.ID,
				})
			}
		}
	}

	a.setSessionCookie(ctx, sess.Token, a.sessionTokenMaxAge())
	return nil, ctx.JSON(http.StatusCreated, authResponse(u, sess))
}

func (a *API) handleSignIn(ctx forge.Context, req *SignInRequest) (*AuthResponse, error) {
	if req.Password == "" {
		return nil, forge.BadRequest("password is required")
	}
	if req.Email == "" && req.Username == "" {
		return nil, forge.BadRequest("email or username is required")
	}

	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
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

	sess, err := a.engine.Refresh(ctx.Context(), req.RefreshToken)
	if err != nil {
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

	name, _ := settings.Get(goCtx, mgr, authsome.SettingCookieName, opts)
	if name == "" {
		name = "authsome_session_token"
	}
	domain, _ := settings.Get(goCtx, mgr, authsome.SettingCookieDomain, opts)
	path, _ := settings.Get(goCtx, mgr, authsome.SettingCookiePath, opts)
	if path == "" {
		path = "/"
	}
	secureSetting, _ := settings.Get(goCtx, mgr, authsome.SettingCookieSecure, opts)
	httpOnly, _ := settings.Get(goCtx, mgr, authsome.SettingCookieHTTPOnly, opts)
	sameSiteStr, _ := settings.Get(goCtx, mgr, authsome.SettingCookieSameSite, opts)

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
	http.SetCookie(ctx.Response(), &http.Cookie{
		Name:     cc.Name,
		Value:    token,
		Path:     cc.Path,
		Domain:   cc.Domain,
		MaxAge:   maxAge,
		HttpOnly: cc.HTTPOnly,
		Secure:   cc.Secure,
		SameSite: cc.SameSite,
	})
}

// deleteSessionCookie clears the session cookie.
func (a *API) deleteSessionCookie(ctx forge.Context) {
	cc := a.resolveCookieConfig(ctx)
	http.SetCookie(ctx.Response(), &http.Cookie{
		Name:     cc.Name,
		Value:    "",
		Path:     cc.Path,
		Domain:   cc.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cc.Secure,
		SameSite: http.SameSiteLaxMode,
	})
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
