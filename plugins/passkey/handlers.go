package passkey

import (
	"encoding/json"
	"github.com/xraph/forge"
)

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

func (h *Handler) BeginRegister(c forge.Context) error {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	res, err := h.svc.BeginRegistration(c.Request().Context(), body.UserID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, res)
}

func (h *Handler) FinishRegister(c forge.Context) error {
	var body struct {
		UserID       string `json:"user_id"`
		CredentialID string `json:"credential_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()
	if err := h.svc.FinishRegistration(c.Request().Context(), body.UserID, body.CredentialID, ip, ua); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "registered"})
}

func (h *Handler) BeginLogin(c forge.Context) error {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	res, err := h.svc.BeginLogin(c.Request().Context(), body.UserID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, res)
}

func (h *Handler) FinishLogin(c forge.Context) error {
	var body struct {
		UserID   string `json:"user_id"`
		Remember bool   `json:"remember"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()
	res, err := h.svc.FinishLogin(c.Request().Context(), body.UserID, body.Remember, ip, ua)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]any{"user": res.User, "session": res.Session, "token": res.Token})
}

func (h *Handler) List(c forge.Context) error {
	userID := c.Request().URL.Query().Get("user_id")
	out, err := h.svc.List(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, out)
}

func (h *Handler) Delete(c forge.Context) error {
	id := c.Param("id")
	ip := c.Request().RemoteAddr
	ua := c.Request().UserAgent()
	if err := h.svc.Delete(c.Request().Context(), id, ip, ua); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]string{"status": "deleted"})
}
