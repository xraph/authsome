package phone

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

// SendCode handles sending of verification code via SMS
func (h *Handler) SendCode(c *forge.Context) error {
	var body struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	if h.rl != nil {
		key := "phone:send:" + body.Phone
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, "/api/auth/phone/send-code")
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
	code, err := h.svc.SendCode(c.Request().Context(), body.Phone, ip, ua)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	res := map[string]any{"status": "sent"}
	if code != "" {
		res["dev_code"] = code
	}
	return c.JSON(200, res)
}

// Verify checks the code and creates a session on success
func (h *Handler) Verify(c *forge.Context) error {
	var body struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		Email    string `json:"email"`
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
	res, err := h.svc.Verify(c.Request().Context(), body.Phone, body.Code, body.Email, body.Remember, ip, ua)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	if res == nil {
		return c.JSON(401, map[string]string{"error": "invalid code"})
	}
	return c.JSON(200, map[string]any{"user": res.User, "session": res.Session, "token": res.Token})
}

// SignIn aliases to Verify for convenience
func (h *Handler) SignIn(c *forge.Context) error {
	var body struct {
		Phone    string `json:"phone"`
		Code     string `json:"code"`
		Email    string `json:"email"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()
	res, err := h.svc.Verify(c.Request().Context(), body.Phone, body.Code, body.Email, body.Remember, ip, ua)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	if res == nil {
		return c.JSON(401, map[string]string{"error": "invalid code"})
	}
	return c.JSON(200, map[string]any{"user": res.User, "session": res.Session, "token": res.Token})
}
