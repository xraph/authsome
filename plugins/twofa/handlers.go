package twofa

import (
	"encoding/json"
	"net/http"

	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler exposes HTTP endpoints for 2FA operations
type Handler struct{ svc *Service }

// Response types
type StatusResponse struct {
	Status string `json:"status"`
}

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

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

func (h *Handler) Enable(c forge.Context) error {
	var body struct {
		UserID string `json:"user_id"`
		Method string `json:"method"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if body.UserID == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_USER_ID", "User ID is required", http.StatusBadRequest))
	}
	bundle, err := h.svc.Enable(c.Request().Context(), body.UserID, &EnableRequest{Method: body.Method})
	if err != nil {
		return handleError(c, err, "ENABLE_2FA_FAILED", "Failed to enable 2FA", http.StatusBadRequest)
	}
	resp := map[string]interface{}{"status": "2fa_enabled"}
	if bundle != nil {
		resp["totp_uri"] = bundle.URI
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) Verify(c forge.Context) error {
	var body struct {
		UserID         string `json:"user_id"`
		Code           string `json:"code"`
		RememberDevice bool   `json:"remember_device"`
		DeviceID       string `json:"device_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if body.UserID == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_USER_ID", "User ID is required", http.StatusBadRequest))
	}
	ok, err := h.svc.Verify(c.Request().Context(), body.UserID, &VerifyRequest{Code: body.Code})
	if err != nil || !ok {
		return c.JSON(http.StatusUnauthorized, errs.New("INVALID_CODE", "Invalid or expired 2FA code", http.StatusUnauthorized))
	}
	// Optionally mark device as trusted
	if body.RememberDevice && body.DeviceID != "" {
		_ = h.svc.MarkTrusted(c.Request().Context(), body.UserID, body.DeviceID, 30)
	}
	return c.JSON(http.StatusOK, &StatusResponse{Status: "verified"})
}

func (h *Handler) Disable(c forge.Context) error {
	var body struct {
		UserID string `json:"user_id"`
	}
	_ = json.NewDecoder(c.Request().Body).Decode(&body)
	if body.UserID == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_USER_ID", "User ID is required", http.StatusBadRequest))
	}
	if err := h.svc.Disable(c.Request().Context(), body.UserID); err != nil {
		return handleError(c, err, "DISABLE_2FA_FAILED", "Failed to disable 2FA", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, &StatusResponse{Status: "2fa_disabled"})
}

func (h *Handler) GenerateBackupCodes(c forge.Context) error {
	var body struct {
		UserID string `json:"user_id"`
		Count  int    `json:"count"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		body.Count = 10
	}
	if body.UserID == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_USER_ID", "User ID is required", http.StatusBadRequest))
	}
	codes, err := h.svc.GenerateBackupCodes(c.Request().Context(), body.UserID, body.Count)
	if err != nil {
		return handleError(c, err, "GENERATE_CODES_FAILED", "Failed to generate backup codes", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, &CodesResponse{Codes: codes})
}

// SendOTP triggers generation of an OTP code for a user (returns code in response for dev/testing)
func (h *Handler) SendOTP(c forge.Context) error {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if body.UserID == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_USER_ID", "User ID is required", http.StatusBadRequest))
	}
	code, err := h.svc.SendOTP(c.Request().Context(), body.UserID)
	if err != nil {
		return handleError(c, err, "SEND_OTP_FAILED", "Failed to send OTP", http.StatusBadRequest)
	}
	// In production, deliver via email/SMS; here we return for testing
	return c.JSON(http.StatusOK, &OTPSentResponse{Status: "otp_sent", Code: code})
}

// Status returns whether 2FA is enabled and whether the device is trusted
func (h *Handler) Status(c forge.Context) error {
	var body struct {
		UserID   string `json:"user_id"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if body.UserID == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_USER_ID", "User ID is required", http.StatusBadRequest))
	}
	st, err := h.svc.GetStatus(c.Request().Context(), body.UserID, body.DeviceID)
	if err != nil {
		// Provide a friendlier message when the user_id is not a valid xid
		if err.Error() == "xid: invalid ID" {
			return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest))
		}
		return handleError(c, err, "GET_STATUS_FAILED", "Failed to get 2FA status", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, &TwoFAStatusResponse{Enabled: st.Enabled, Method: st.Method, Trusted: st.Trusted})
}
