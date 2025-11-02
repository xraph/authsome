package emailotp

import (
	"encoding/json"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/forge"
	"net"
)

type Handler struct {
	svc *Service
	rl  *rl.Service
}

func NewHandler(s *Service, rls *rl.Service) *Handler { return &Handler{svc: s, rl: rls} }

// Send handles sending of OTP to email
func (h *Handler) Send(c forge.Context) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	// Basic rate limit: per email for the send path
	if h.rl != nil {
		key := "emailotp:send:" + body.Email
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, "/api/auth/email-otp/send")
		if err != nil {
			return c.JSON(500, map[string]string{"error": "rate limit error"})
		}
		if !ok {
			return c.JSON(429, map[string]string{"error": "too many requests"})
		}
	}
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ua := c.Request().UserAgent()
	otp, err := h.svc.SendOTP(c.Request().Context(), body.Email, ip, ua)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	// In dev mode we may expose otp
	res := map[string]interface{}{"status": "sent"}
	if otp != "" {
		res["dev_otp"] = otp
	}
	return c.JSON(200, res)
}

// Verify checks the OTP and creates a session on success
func (h *Handler) Verify(c forge.Context) error {
	var body struct {
		Email    string `json:"email"`
		OTP      string `json:"otp"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ua := c.Request().UserAgent()
	res, err := h.svc.VerifyOTP(c.Request().Context(), body.Email, body.OTP, body.Remember, ip, ua)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	if res == nil {
		return c.JSON(401, map[string]string{"error": "invalid otp"})
	}
	return c.JSON(200, map[string]interface{}{"user": res.User, "session": res.Session, "token": res.Token})
}
