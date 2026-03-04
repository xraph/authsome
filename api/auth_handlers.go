package api

import (
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
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

	signUpOpts := []forge.RouteOption{
		forge.WithSummary("Sign up"),
		forge.WithDescription("Creates a new user account and returns authentication tokens."),
		forge.WithOperationID("signUp"),
		forge.WithRequestSchema(SignUpRequest{}),
		forge.WithCreatedResponse(AuthResponse{}),
		forge.WithErrorResponses(),
	}
	signUpOpts = append(signUpOpts, a.rateLimitOpt(rlCfg.SignUpLimit)...)
	if err := g.POST("/signup", a.handleSignUp, signUpOpts...); err != nil {
		return err
	}

	signInOpts := []forge.RouteOption{
		forge.WithSummary("Sign in"),
		forge.WithDescription("Authenticates a user with email/username and password."),
		forge.WithOperationID("signIn"),
		forge.WithRequestSchema(SignInRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Authenticated", AuthResponse{}),
		forge.WithErrorResponses(),
	}
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

	u, sess, err := a.engine.SignUp(ctx.Context(), &account.SignUpRequest{
		AppID:    appID,
		Email:    req.Email,
		Password: req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username: req.Username,
	})
	if err != nil {
		return nil, mapError(err)
	}

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

	u, sess, err := a.engine.SignIn(ctx.Context(), &account.SignInRequest{
		AppID:    appID,
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		return nil, mapError(err)
	}

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

	resp := &TokenResponse{
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
		ExpiresAt:    sess.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	return nil, ctx.JSON(http.StatusOK, resp)
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
