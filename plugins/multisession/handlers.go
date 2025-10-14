package multisession

import (
    "encoding/json"
    "fmt"
    "github.com/rs/xid"
    "github.com/xraph/forge"
)

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// List returns sessions for the current user based on cookie
func (h *Handler) List(c *forge.Context) error {
    token, err := c.Cookie("session_token")
    if err != nil || token == "" {
        return c.JSON(401, map[string]string{"error": "not authenticated"})
    }
    uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
    if err != nil {
        return c.JSON(401, map[string]string{"error": err.Error()})
    }
    out, err := h.svc.List(c.Request().Context(), uid)
    if err != nil { return c.JSON(400, map[string]string{"error": err.Error()}) }
    return c.JSON(200, map[string]any{"sessions": out})
}

// SetActive switches the current session cookie to the provided session id
func (h *Handler) SetActive(c *forge.Context) error {
    token, err := c.Cookie("session_token")
    if err != nil || token == "" { return c.JSON(401, map[string]string{"error": "not authenticated"}) }
    uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
    if err != nil { return c.JSON(401, map[string]string{"error": err.Error()}) }
    var body struct{ ID string `json:"id"` }
    if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
        return c.JSON(400, map[string]string{"error": "invalid request"})
    }
    if body.ID == "" { return c.JSON(400, map[string]string{"error": "id required"}) }
    sid, err := xid.FromString(body.ID)
    if err != nil { return c.JSON(400, map[string]string{"error": "invalid id"}) }
    sess, err := h.svc.Find(c.Request().Context(), uid, sid)
    if err != nil { return c.JSON(400, map[string]string{"error": err.Error()}) }
    // Set cookie header manually (no helper on Context)
    cookie := fmt.Sprintf("session_token=%s; Path=/; HttpOnly; SameSite=Lax", sess.Token)
    c.Header().Add("Set-Cookie", cookie)
    return c.JSON(200, map[string]any{"session": sess, "token": sess.Token})
}

// Delete revokes a session by id for the current user
func (h *Handler) Delete(c *forge.Context) error {
    token, err := c.Cookie("session_token")
    if err != nil || token == "" { return c.JSON(401, map[string]string{"error": "not authenticated"}) }
    uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
    if err != nil { return c.JSON(401, map[string]string{"error": err.Error()}) }
    idStr := c.Param("id")
    if idStr == "" { return c.JSON(400, map[string]string{"error": "id required"}) }
    sid, err := xid.FromString(idStr)
    if err != nil { return c.JSON(400, map[string]string{"error": "invalid id"}) }
    if err := h.svc.Delete(c.Request().Context(), uid, sid); err != nil {
        return c.JSON(400, map[string]string{"error": err.Error()})
    }
    return c.JSON(200, map[string]string{"status": "deleted"})
}