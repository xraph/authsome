package phone

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

// SendCode handles sending of verification code via SMS
func (h *Handler) SendCode(c forge.Context) error {
	var body struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if h.rl != nil {
		key := "phone:send:" + body.Phone
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
	code, err := h.svc.SendCode(c.Request().Context(), body.Phone, ip, ua)
	if err != nil {
		return handleError(c, err, "SEND_CODE_FAILED", "Failed to send verification code", http.StatusBadRequest)
	}
	res := map[string]any{"status": "sent"}
	if code != "" {
		res["dev_code"] = code
	}
	return c.JSON(http.StatusOK, res)
}

// Verify checks the code and creates a session on success
func (h *Handler) Verify(c forge.Context) error {
	var body struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		Email    string `json:"email"`
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
	res, err := h.svc.Verify(c.Request().Context(), body.Phone, body.Code, body.Email, body.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "VERIFY_CODE_FAILED", "Failed to verify code", http.StatusBadRequest)
	}
	if res == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("INVALID_CODE", "Invalid or expired verification code", http.StatusUnauthorized))
	}
	return c.JSON(http.StatusOK, map[string]any{"user": res.User, "session": res.Session, "token": res.Token})
}

// SignIn aliases to Verify for convenience
func (h *Handler) SignIn(c forge.Context) error {
	var body struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		Email    string `json:"email"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()
	res, err := h.svc.Verify(c.Request().Context(), body.Phone, body.Code, body.Email, body.Remember, ip, ua)
	if err != nil {
		return handleError(c, err, "SIGNIN_FAILED", "Failed to sign in", http.StatusBadRequest)
	}
	if res == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("INVALID_CODE", "Invalid or expired verification code", http.StatusUnauthorized))
	}
	return c.JSON(http.StatusOK, map[string]any{"user": res.User, "session": res.Session, "token": res.Token})
}
