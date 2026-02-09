package phone

import (
	"errors"
	"net"
	"net/http"

	"github.com/xraph/authsome/core"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/helpers"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct {
	svc      *Service
	rl       *rl.Service
	authInst core.Authsome
}

// Request types.
type SendCodeRequest struct {
	Phone string `example:"+1234567890" json:"phone" validate:"required"`
}

type VerifyRequest struct {
	Phone    string `example:"+1234567890"      json:"phone"    validate:"required"`
	Code     string `example:"123456"           json:"code"     validate:"required"`
	Email    string `example:"user@example.com" json:"email"    validate:"required,email"`
	Remember bool   `example:"false"            json:"remember"`
}

// Response types.
type SendCodeResponse struct {
	Status  string `example:"sent"   json:"status"`
	DevCode string `example:"123456" json:"dev_code,omitempty"`
}

type PhoneVerifyResponse struct {
	User    *user.User       `json:"user"`
	Session *session.Session `json:"session"`
	Token   string           `example:"session_token_abc123" json:"token"`
}

func NewHandler(s *Service, rls *rl.Service, authInst core.Authsome) *Handler {
	return &Handler{svc: s, rl: rls, authInst: authInst}
}

// handleError returns the error in a structured format.
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	authErr := &errs.AuthsomeError{}
	if errors.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// SendCode handles sending of verification code via SMS.
func (h *Handler) SendCode(c forge.Context) error {
	var req SendCodeRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	if h.rl != nil {
		key := "phone:send:" + req.Phone

		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, "/api/auth/phone/send-code")
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

	code, err := h.svc.SendCode(c.Request().Context(), req.Phone, ip, ua)
	if err != nil {
		return handleError(c, err, "SEND_CODE_FAILED", "Failed to send verification code", http.StatusBadRequest)
	}

	res := &SendCodeResponse{Status: "sent"}
	if code != "" {
		res.DevCode = code
	}

	return c.JSON(http.StatusOK, res)
}

// Verify checks the code and creates a session on success.
func (h *Handler) Verify(c forge.Context) error {
	var req VerifyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	ua := c.Request().UserAgent()

	authRes, err := h.svc.Verify(c.Request().Context(), req.Phone, req.Code, req.Email, req.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "VERIFY_CODE_FAILED", "Failed to verify code", http.StatusBadRequest)
	}

	if authRes == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("INVALID_CODE", "Invalid or expired verification code", http.StatusUnauthorized))
	}

	// Set session cookie if enabled
	if h.authInst != nil && authRes.Session != nil {
		_ = helpers.SetSessionCookieFromAuth(c, h.authInst, authRes.Token, authRes.Session.ExpiresAt)
	}

	return c.JSON(http.StatusOK, &PhoneVerifyResponse{
		User:    authRes.User,
		Session: authRes.Session,
		Token:   authRes.Token,
	})
}

// SignIn aliases to Verify for convenience.
func (h *Handler) SignIn(c forge.Context) error {
	var req VerifyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	ua := c.Request().UserAgent()

	authRes, err := h.svc.Verify(c.Request().Context(), req.Phone, req.Code, req.Email, req.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "SIGNIN_FAILED", "Failed to sign in", http.StatusBadRequest)
	}

	if authRes == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("INVALID_CODE", "Invalid or expired verification code", http.StatusUnauthorized))
	}

	// Set session cookie if enabled
	if h.authInst != nil && authRes.Session != nil {
		_ = helpers.SetSessionCookieFromAuth(c, h.authInst, authRes.Token, authRes.Session.ExpiresAt)
	}

	return c.JSON(http.StatusOK, &PhoneVerifyResponse{
		User:    authRes.User,
		Session: authRes.Session,
		Token:   authRes.Token,
	})
}
