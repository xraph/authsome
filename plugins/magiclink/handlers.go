package magiclink

import (
	"encoding/json"
	rl "github.com/xraph/authsome/core/ratelimit"
	"github.com/xraph/forge"
	"net"
	"net/http"
)

type Handler struct {
	svc *Service
	rl  *rl.Service
}

func NewHandler(s *Service, rls *rl.Service) *Handler { return &Handler{svc: s, rl: rls} }

func (h *Handler) Send(c forge.Context) error {
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}
	if h.rl != nil {
		key := "magiclink:send:" + body.Email
		ok, err := h.rl.CheckLimitForPath(c.Request().Context(), key, "/api/auth/magic-link/send")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "rate limit error"})
		}
		if !ok {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many requests"})
		}
	}
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ua := c.Request().UserAgent()
	url, err := h.svc.Send(c.Request().Context(), body.Email, ip, ua)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
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
	remember := q.Get("remember") == "true"
	ip := c.Request().RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	ua := c.Request().UserAgent()
	res, err := h.svc.Verify(c.Request().Context(), token, remember, ip, ua)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"user": res.User, "session": res.Session, "token": res.Token})
}
