package magiclink

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

// Response types
type VerifyResponse struct {
	User    interface{} `json:"user"`
	Session interface{} `json:"session"`
	Token   string      `json:"token"`
}

func NewHandler(s *Service, rls *rl.Service) *Handler { return &Handler{svc: s, rl: rls} }

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

func (h *Handler) Send(c forge.Context) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if h.rl != nil {
		key := "magiclink:send:" + body.Email
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, "/api/auth/magic-link/send")
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
	url, err := h.svc.Send(c.Request().Context(), body.Email, ip, ua)
	if err != nil {
		return handleError(c, err, "SEND_MAGIC_LINK_FAILED", "Failed to send magic link", http.StatusBadRequest)
	}
	res := map[string]any{"status": "sent"}
	if url != "" {
		res["dev_url"] = url
	}
	return c.JSON(http.StatusOK, res)
}

func (h *Handler) Verify(c forge.Context) error {
	q := c.Request().URL.Query()
	token := q.Get("token")
	if token == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_TOKEN", "Token parameter is required", http.StatusBadRequest))
	}
	remember := q.Get("remember") == "true"
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ua := c.Request().UserAgent()
	res, err := h.svc.Verify(c.Request().Context(), token, remember, ip, ua)
	if err != nil {
		return handleError(c, err, "VERIFY_MAGIC_LINK_FAILED", "Failed to verify magic link", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, &VerifyResponse{User: res.User, Session: res.Session, Token: res.Token})
}
