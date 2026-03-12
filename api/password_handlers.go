package api

import (
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/middleware"
)

// ──────────────────────────────────────────────────
// Password route registration
// ──────────────────────────────────────────────────

func (a *API) registerPasswordRoutes(router forge.Router) error {
	base := a.engine.Config().BasePath
	rlCfg := a.engine.Config().RateLimit
	g := router.Group(base, forge.WithGroupTags("password"))

	forgotOpts := make([]forge.RouteOption, 0, 7) //nolint:mnd // base options + rate limit
	forgotOpts = append(forgotOpts,
		forge.WithSummary("Forgot password"),
		forge.WithDescription("Initiates a password reset flow. Always returns success to prevent email enumeration."),
		forge.WithOperationID("forgotPassword"),
		forge.WithRequestSchema(ForgotPasswordRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Reset initiated", ForgotPasswordResponse{}),
		forge.WithErrorResponses(),
	)
	forgotOpts = append(forgotOpts, a.rateLimitOpt(rlCfg.ForgotPasswordLimit)...)
	if err := g.POST("/forgot-password", a.handleForgotPassword, forgotOpts...); err != nil {
		return err
	}

	if err := g.POST("/reset-password", a.handleResetPassword,
		forge.WithSummary("Reset password"),
		forge.WithDescription("Resets a user's password using a reset token. Revokes all existing sessions."),
		forge.WithOperationID("resetPassword"),
		forge.WithRequestSchema(ResetPasswordRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Password reset", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.POST("/change-password", a.handleChangePassword,
		forge.WithSummary("Change password"),
		forge.WithDescription("Changes the authenticated user's password. Requires current password."),
		forge.WithOperationID("changePassword"),
		forge.WithRequestSchema(ChangePasswordRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Password changed", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.POST("/verify-email", a.handleVerifyEmail,
		forge.WithSummary("Verify email"),
		forge.WithDescription("Verifies a user's email address using a verification token."),
		forge.WithOperationID("verifyEmail"),
		forge.WithRequestSchema(VerifyEmailRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Email verified", StatusResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Password handlers
// ──────────────────────────────────────────────────

func (a *API) handleForgotPassword(ctx forge.Context, req *ForgotPasswordRequest) (*ForgotPasswordResponse, error) {
	if req.Email == "" {
		return nil, forge.BadRequest("email is required")
	}

	appID, err := a.resolveAppID(req.AppID)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	// ForgotPassword returns nil, nil for unknown emails (avoids email enumeration).
	_, _ = a.engine.ForgotPassword(ctx.Context(), appID, req.Email) //nolint:errcheck // best-effort lookup

	// Always return success regardless of whether the email exists.
	resp := &ForgotPasswordResponse{Status: "ok"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleResetPassword(ctx forge.Context, req *ResetPasswordRequest) (*StatusResponse, error) {
	if req.Token == "" || req.NewPassword == "" {
		return nil, forge.BadRequest("token and new_password are required")
	}

	if err := a.engine.ResetPassword(ctx.Context(), req.Token, req.NewPassword); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "password reset"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleChangePassword(ctx forge.Context, req *ChangePasswordRequest) (*StatusResponse, error) {
	userID, ok := middleware.UserIDFrom(ctx.Context())
	if !ok {
		return nil, forge.Unauthorized("authentication required")
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return nil, forge.BadRequest("current_password and new_password are required")
	}

	if err := a.engine.ChangePassword(ctx.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "password changed"}
	return nil, ctx.JSON(http.StatusOK, resp)
}

func (a *API) handleVerifyEmail(ctx forge.Context, req *VerifyEmailRequest) (*StatusResponse, error) {
	if req.Token == "" {
		return nil, forge.BadRequest("token is required")
	}

	if err := a.engine.VerifyEmail(ctx.Context(), req.Token); err != nil {
		return nil, mapError(err)
	}

	resp := &StatusResponse{Status: "email verified"}
	return nil, ctx.JSON(http.StatusOK, resp)
}
