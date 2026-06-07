package api

import (
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/middleware"
)

// ──────────────────────────────────────────────────
// Password route registration
// ──────────────────────────────────────────────────

func (a *API) registerPasswordRoutes(router forge.Router) error {
	rlCfg := a.engine.Config().RateLimit
	g := router.Group("/v1", forge.WithGroupTags("password"))

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

	verifyOpts := []forge.RouteOption{
		forge.WithSummary("Verify email"),
		forge.WithDescription("Verifies a user's email address using a 6-digit OTP code (per-user, attempt-limited) or a verification token."),
		forge.WithOperationID("verifyEmail"),
		forge.WithRequestSchema(VerifyEmailRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Email verified", StatusResponse{}),
		forge.WithErrorResponses(),
	}
	verifyOpts = append(verifyOpts, a.rateLimitOpt(rlCfg.VerifyEmailLimit)...)
	if err := g.POST("/verify-email", a.handleVerifyEmail, verifyOpts...); err != nil {
		return err
	}

	resendOpts := []forge.RouteOption{
		forge.WithSummary("Resend email verification"),
		forge.WithDescription("Issues a fresh email verification token and emits the auth.email_verification_requested hook so a delivery handler (notification plugin or custom mailer) can send the link. Always returns 200 to avoid email-existence enumeration; callers cannot tell whether the email was registered or already verified."),
		forge.WithOperationID("resendEmailVerification"),
		forge.WithRequestSchema(ResendVerificationRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Verification queued", StatusResponse{}),
		forge.WithErrorResponses(),
	}
	resendOpts = append(resendOpts, a.rateLimitOpt(rlCfg.ResendVerificationLimit)...)
	return g.POST("/verify-email/resend", a.handleResendVerification, resendOpts...)
}

// ──────────────────────────────────────────────────
// Password handlers
// ──────────────────────────────────────────────────

func (a *API) handleForgotPassword(ctx forge.Context, req *ForgotPasswordRequest) (*ForgotPasswordResponse, error) {
	if req.Email == "" {
		return nil, forge.BadRequest("email is required")
	}

	appID, err := a.resolvePublicAppID(ctx, req.AppID)
	if err != nil {
		return nil, err
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
	// The OTP form submits {token: <6-digit code>} (the @authsome/ui-components
	// EmailVerificationForm). req.Code is also accepted for forward-compat.
	candidate := req.Code
	if candidate == "" {
		candidate = req.Token
	}
	if candidate == "" {
		return nil, forge.BadRequest("code or token is required")
	}

	// OTP path: a 6-digit numeric value from an authenticated session user (the
	// account created at signup) is verified via the secure per-user code path
	// (per-user lookup + attempt limiting + constant-time compare). We do NOT
	// fall through to the global token lookup on failure — that could match a
	// different user's identical code.
	if isOTPCode(candidate) {
		// Resolve the user the code belongs to. Prefer the authenticated session
		// user; when the request is unauthenticated (e.g. cross-origin signup
		// where the session cookie isn't carried), fall back to the email in the
		// body. Either way verification goes through the secure per-user code
		// path (per-user lookup + attempt limiting + constant-time compare), so
		// scoping by email is not a brute-force oracle.
		userID, ok := middleware.UserIDFrom(ctx.Context())
		if !ok && req.Email != "" {
			if appID, appErr := a.resolvePublicAppID(ctx, req.AppID); appErr == nil {
				if u, userErr := a.engine.GetUserByEmail(ctx.Context(), appID, req.Email); userErr == nil {
					userID = u.ID
					ok = true
				}
			}
		}
		if ok {
			if err := a.engine.VerifyEmailCode(ctx.Context(), userID, candidate); err != nil {
				return nil, mapError(err)
			}
			// Auto-login: signup withholds a client session until the email is
			// verified, so on success we mint a fresh session and return the
			// standard auth response. The SDK persists it (authsome:session),
			// signing the user in without a second login round-trip. If session
			// issuance trips the MFA gate (or otherwise fails), verification has
			// still succeeded — fall back to a plain status so the client can
			// route to its own login/MFA surface.
			if resp, sessErr := a.issueSessionForUser(ctx, userID, "email_verification"); sessErr == nil {
				return nil, ctx.JSON(http.StatusOK, resp)
			}
			return nil, ctx.JSON(http.StatusOK, &StatusResponse{Status: "email verified"})
		}

		// A 6-digit OTP that can't be tied to a specific user (no session and
		// no/unknown email) MUST NOT fall through to the global token lookup
		// below: GetVerification matches by token value alone, so a short code
		// there is brute-forceable across the entire user pool with no per-user
		// attempt limiting. Return the same generic error a wrong code yields so
		// this isn't an email-existence oracle either.
		return nil, mapError(account.ErrInvalidCredentials)
	}

	// Token path: high-entropy verification tokens only (link flows, e.g. magic
	// link). 6-digit OTP codes never reach here (handled + returned above).
	if err := a.engine.VerifyEmail(ctx.Context(), candidate); err != nil {
		return nil, mapError(err)
	}
	return nil, ctx.JSON(http.StatusOK, &StatusResponse{Status: "email verified"})
}

// isOTPCode reports whether s is a 6-digit numeric OTP code.
func isOTPCode(s string) bool {
	if len(s) != 6 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (a *API) handleResendVerification(ctx forge.Context, req *ResendVerificationRequest) (*StatusResponse, error) {
	// Anti-enumeration: never surface "no such user" or "already
	// verified" — both leak the same registration signal /v1/signup
	// closes off. We always return 200 with the same body. A missing
	// app context (no pk header, no app_id body) is also folded into
	// the same 200 — emitting a 400 here would let an attacker
	// distinguish "endpoint shape is wrong" from "nothing to do",
	// which still narrows the search space.
	if req.Email == "" {
		return nil, ctx.JSON(http.StatusOK, &StatusResponse{Status: "ok"})
	}
	appID, err := a.resolvePublicAppID(ctx, req.AppID)
	if err != nil {
		return nil, ctx.JSON(http.StatusOK, &StatusResponse{Status: "ok"})
	}
	_ = a.engine.ResendEmailVerification(ctx.Context(), appID, req.Email) //nolint:errcheck // best-effort, error-suppressed by design
	return nil, ctx.JSON(http.StatusOK, &StatusResponse{Status: "ok"})
}
