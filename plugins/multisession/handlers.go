package multisession

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct{ svc *Service }

// Response types - use shared responses from core
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse

type SessionsResponse struct {
	Sessions interface{} `json:"sessions"`
}

type SessionTokenResponse struct {
	Session interface{} `json:"session"`
	Token   string      `json:"token"`
}

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// List returns sessions for the current user based on cookie
func (h *Handler) List(c forge.Context) error {
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}
	token := cookie.Value
	uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Invalid or expired session token", http.StatusUnauthorized)
	}
	out, err := h.svc.List(c.Request().Context(), uid)
	if err != nil {
		return handleError(c, err, "LIST_SESSIONS_FAILED", "Failed to list sessions", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, &SessionsResponse{Sessions: out})
}

// SetActive switches the current session cookie to the provided session id
func (h *Handler) SetActive(c forge.Context) error {
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}
	token := cookie.Value
	uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Invalid or expired session token", http.StatusUnauthorized)
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}
	if body.ID == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_SESSION_ID", "Session ID is required", http.StatusBadRequest))
	}
	sid, err := xid.FromString(body.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_SESSION_ID", "Invalid session ID format", http.StatusBadRequest).WithError(err))
	}
	sess, err := h.svc.Find(c.Request().Context(), uid, sid)
	if err != nil {
		return handleError(c, err, "SESSION_NOT_FOUND", "Session not found", http.StatusNotFound)
	}
	// Set cookie header manually (no helper on Context)
	cookieStr := fmt.Sprintf("session_token=%s; Path=/; HttpOnly; SameSite=Lax", sess.Token)
	c.SetHeader("Set-Cookie", cookieStr)
	return c.JSON(http.StatusOK, &SessionTokenResponse{Session: sess, Token: sess.Token})
}

// Delete revokes a session by id for the current user
func (h *Handler) Delete(c forge.Context) error {
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}
	token := cookie.Value
	uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Invalid or expired session token", http.StatusUnauthorized)
	}
	idStr := c.Param("id")
	if idStr == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_SESSION_ID", "Session ID is required", http.StatusBadRequest))
	}
	sid, err := xid.FromString(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_SESSION_ID", "Invalid session ID format", http.StatusBadRequest).WithError(err))
	}
	if err := h.svc.Delete(c.Request().Context(), uid, sid); err != nil {
		return handleError(c, err, "DELETE_SESSION_FAILED", "Failed to delete session", http.StatusBadRequest)
	}
	return c.JSON(http.StatusOK, &StatusResponse{Status: "deleted"})
}
