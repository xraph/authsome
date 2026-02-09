package twofa

import (
	"net/http"

	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler exposes HTTP endpoints for 2FA operations.
type Handler struct{ svc *Service }

// Request types.
type EnableRequest2FA struct {
	UserID string `json:"user_id" validate:"required"`
	Method string `json:"method"`
}

type VerifyRequest2FA struct {
	UserID         string `json:"user_id"         validate:"required"`
	Code           string `json:"code"            validate:"required"`
	RememberDevice bool   `json:"remember_device"`
	DeviceID       string `json:"device_id"`
}

type DisableRequest struct {
	UserID string `json:"user_id" validate:"required"`
}

type RegenerateCodesRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Count  int    `json:"count"`
}

type SendOTPRequest struct {
	UserID string `json:"user_id" validate:"required"`
}

type GetStatusRequest struct {
	UserID   string `json:"user_id"   validate:"required"`
	DeviceID string `json:"device_id"`
}

// Response types - use shared responses from core.
type StatusResponse = responses.StatusResponse

// Plugin-specific responses.
type CodesResponse struct {
	Codes []string `json:"codes"`
}

type OTPSentResponse struct {
	Status string `json:"status"`
	Code   string `json:"code"`
}

type TwoFAStatusResponse struct {
	Enabled bool   `json:"enabled"`
	Method  string `json:"method"`
	Trusted bool   `json:"trusted"`
}

type EnableResponse struct {
	Status  string `json:"status"`
	TOTPURI string `json:"totp_uri,omitempty"`
}

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// handleError returns the error in a structured format.
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	authErr := &errs.AuthsomeError{}
	if errs.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

func (h *Handler) Enable(c forge.Context) error {
	var req EnableRequest2FA
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	bundle, err := h.svc.Enable(c.Request().Context(), req.UserID, &EnableRequest{Method: req.Method})
	if err != nil {
		return handleError(c, err, "ENABLE_2FA_FAILED", "Failed to enable 2FA", http.StatusBadRequest)
	}

	resp := EnableResponse{Status: "2fa_enabled"}
	if bundle != nil {
		resp.TOTPURI = bundle.URI
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) Verify(c forge.Context) error {
	var req VerifyRequest2FA
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	ok, err := h.svc.Verify(c.Request().Context(), req.UserID, &VerifyRequest{Code: req.Code})
	if err != nil || !ok {
		return c.JSON(http.StatusUnauthorized, errs.New("INVALID_CODE", "Invalid or expired 2FA code", http.StatusUnauthorized))
	}
	// Optionally mark device as trusted
	if req.RememberDevice && req.DeviceID != "" {
		_ = h.svc.MarkTrusted(c.Request().Context(), req.UserID, req.DeviceID, 30)
	}

	return c.JSON(http.StatusOK, &StatusResponse{Status: "verified"})
}

func (h *Handler) Disable(c forge.Context) error {
	var req DisableRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	if err := h.svc.Disable(c.Request().Context(), req.UserID); err != nil {
		return handleError(c, err, "DISABLE_2FA_FAILED", "Failed to disable 2FA", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &StatusResponse{Status: "2fa_disabled"})
}

func (h *Handler) GenerateBackupCodes(c forge.Context) error {
	var req RegenerateCodesRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	count := req.Count
	if count == 0 {
		count = 10
	}

	codes, err := h.svc.GenerateBackupCodes(c.Request().Context(), req.UserID, count)
	if err != nil {
		return handleError(c, err, "GENERATE_CODES_FAILED", "Failed to generate backup codes", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &CodesResponse{Codes: codes})
}

// SendOTP triggers generation of an OTP code for a user (returns code in response for dev/testing).
func (h *Handler) SendOTP(c forge.Context) error {
	var req SendOTPRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	code, err := h.svc.SendOTP(c.Request().Context(), req.UserID)
	if err != nil {
		return handleError(c, err, "SEND_OTP_FAILED", "Failed to send OTP", http.StatusBadRequest)
	}
	// In production, deliver via email/SMS; here we return for testing
	return c.JSON(http.StatusOK, &OTPSentResponse{Status: "otp_sent", Code: code})
}

// Status returns whether 2FA is enabled and whether the device is trusted.
func (h *Handler) Status(c forge.Context) error {
	var req GetStatusRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	st, err := h.svc.GetStatus(c.Request().Context(), req.UserID, req.DeviceID)
	if err != nil {
		// Provide a friendlier message when the user_id is not a valid xid
		if err.Error() == "xid: invalid ID" {
			return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest))
		}

		return handleError(c, err, "GET_STATUS_FAILED", "Failed to get 2FA status", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &TwoFAStatusResponse{Enabled: st.Enabled, Method: st.Method, Trusted: st.Trusted})
}
