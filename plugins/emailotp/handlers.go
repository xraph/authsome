package emailotp

import (
	"encoding/json"
	"net"
	"net/http"

	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct {
	svc *Service
	rl  *rl.Service
}

func NewHandler(s *Service, rls *rl.Service) *Handler { return &Handler{svc: s, rl: rls} }

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// Send handles sending of OTP to email
func (h *Handler) Send(c forge.Context) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	// Basic rate limit: per email for the send path
	if h.rl != nil {
		key := "emailotp:send:" + body.Email
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
	otp, err := h.svc.SendOTP(c.Request().Context(), body.Email, ip, ua)
	if err != nil {
		return handleError(c, err, "SEND_OTP_FAILED", "Failed to send OTP", http.StatusBadRequest)
	}
	// In dev mode we may expose otp
	res := map[string]interface{}{"status": "sent"}
	if otp != "" {
		res["dev_otp"] = otp
	}
	return c.JSON(http.StatusOK, res)
}

// Verify checks the OTP and creates a session on success
func (h *Handler) Verify(c forge.Context) error {
	var body struct {
		Email    string `json:"email"`
		OTP      string `json:"otp"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ua := c.Request().UserAgent()
	res, err := h.svc.VerifyOTP(c.Request().Context(), body.Email, body.OTP, body.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "VERIFY_OTP_FAILED", "Failed to verify OTP", http.StatusBadRequest)
	}
	if res == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("INVALID_OTP", "Invalid or expired OTP code", http.StatusUnauthorized))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"user": res.User, "session": res.Session, "token": res.Token})
}
