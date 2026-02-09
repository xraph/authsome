package emailotp

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/contexts"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/helpers"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct {
	svc      *Service
	rl       *rl.Service
	authInst core.Authsome
}

func NewHandler(s *Service, rls *rl.Service, authInst core.Authsome) *Handler {
	return &Handler{svc: s, rl: rls, authInst: authInst}
}

// Request types.
type SendRequest struct {
	Email string `example:"user@example.com" json:"email" validate:"required,email"`
}

type VerifyRequest struct {
	Email    string `example:"user@example.com" json:"email"    validate:"required,email"`
	OTP      string `example:"123456"           json:"otp"      validate:"required"`
	Remember bool   `example:"false"            json:"remember"`
}

// Response types - use shared responses from core.
type ErrorResponse = responses.ErrorResponse
type VerifyResponse = responses.VerifyResponse

// Plugin-specific response.
type SendResponse struct {
	Status string `example:"sent"   json:"status"`
	DevOTP string `example:"123456" json:"dev_otp,omitempty"`
}

// handleError returns the error in a structured format.
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	authErr := &errs.AuthsomeError{}
	if errors.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// Send handles sending of OTP to email.
func (h *Handler) Send(c forge.Context) error {
	var req SendRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get app context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok || appID.IsNil() {
		return handleError(c, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest), "APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest)
	}

	// Basic rate limit: per email for the send path
	if h.rl != nil {
		key := "emailotp:send:" + req.Email

		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, "/api/auth/email-otp/send")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, errs.New("RATE_LIMIT_ERROR", "Rate limit check failed", http.StatusInternalServerError).WithError(err))
		}

		if !ok {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Too many requests, please try again later", http.StatusTooManyRequests))
		}
	}

	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	ua := c.Request().UserAgent()

	otp, err := h.svc.SendOTP(c.Request().Context(), appID, req.Email, ip, ua)
	if err != nil {
		return handleError(c, err, "SEND_OTP_FAILED", "Failed to send OTP", http.StatusBadRequest)
	}

	// Return structured response
	response := SendResponse{
		Status: "sent",
	}
	if otp != "" {
		response.DevOTP = otp
	}

	return c.JSON(http.StatusOK, response)
}

// Verify checks the OTP and creates a session on success.
func (h *Handler) Verify(c forge.Context) error {
	var req VerifyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get app and environment context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok || appID.IsNil() {
		return handleError(c, errs.New("APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest), "APP_CONTEXT_REQUIRED", "App context required", http.StatusBadRequest)
	}

	envID, ok := contexts.GetEnvironmentID(c.Request().Context())
	if !ok || envID.IsNil() {
		return handleError(c, errs.New("ENVIRONMENT_CONTEXT_REQUIRED", "Environment context required", http.StatusBadRequest), "ENVIRONMENT_CONTEXT_REQUIRED", "Environment context required", http.StatusBadRequest)
	}

	// Get optional organization context
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	ua := c.Request().UserAgent()

	res, err := h.svc.VerifyOTP(c.Request().Context(), appID, envID, orgIDPtr, req.Email, req.OTP, req.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "VERIFY_OTP_FAILED", "Failed to verify OTP", http.StatusBadRequest)
	}

	// Set session cookie if enabled
	if h.authInst != nil && res.Session != nil {
		_ = helpers.SetSessionCookieFromAuth(c, h.authInst, res.Token, res.Session.ExpiresAt)
	}

	return c.JSON(http.StatusOK, VerifyResponse{
		User:    res.User,
		Session: res.Session,
		Token:   res.Token,
	})
}
