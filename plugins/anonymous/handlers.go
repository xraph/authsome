package anonymous

import (
	"encoding/json"
	"fmt"
	"github.com/xraph/forge"
)

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// SignIn creates a guest user and session
func (h *Handler) SignIn(c forge.Context) error {
	// Optional body to allow passing remember later; currently unused
	var body struct{}
	_ = json.NewDecoder(c.Request().Body).Decode(&body)
	sess, err := h.svc.SignInGuest(c.Request().Context(), c.Request().RemoteAddr, c.Request().UserAgent())
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{"token": sess.Token, "session": sess})
}

// Link upgrades an anonymous session to a real account
func (h *Handler) Link(c forge.Context) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}
	// Read session token from cookie for current guest
	ck, err := c.Request().Cookie("session_token")
	if err != nil || ck == nil || ck.Value == "" {
		return c.JSON(401, map[string]string{"error": "missing session"})
	}
	u, err := h.svc.LinkGuest(c.Request().Context(), ck.Value, body.Email, body.Password, body.Name)
	if err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, map[string]interface{}{"user": u, "message": fmt.Sprintf("linked %s", u.Email)})
}
