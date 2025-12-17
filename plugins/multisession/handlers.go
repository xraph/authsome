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

// GetCurrent returns details about the currently active session
func (h *Handler) GetCurrent(c forge.Context) error {
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}
	token := cookie.Value

	// Get user ID from token
	uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Invalid or expired session token", http.StatusUnauthorized)
	}

	// Get session ID from token
	sid, err := h.svc.GetCurrentSessionID(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Failed to extract session ID", http.StatusUnauthorized)
	}

	// Get current session
	sess, err := h.svc.GetCurrent(c.Request().Context(), uid, sid)
	if err != nil {
		return handleError(c, err, "SESSION_NOT_FOUND", "Current session not found", http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, &SessionTokenResponse{Session: sess, Token: token})
}

// GetByID returns details about a specific session by ID
func (h *Handler) GetByID(c forge.Context) error {
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

	sess, err := h.svc.Find(c.Request().Context(), uid, sid)
	if err != nil {
		return handleError(c, err, "SESSION_NOT_FOUND", "Session not found", http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, &SessionTokenResponse{Session: sess, Token: sess.Token})
}

// RevokeAll revokes all sessions for the current user
func (h *Handler) RevokeAll(c forge.Context) error {
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}
	token := cookie.Value
	uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Invalid or expired session token", http.StatusUnauthorized)
	}

	// Get current session ID
	currentSID, err := h.svc.GetCurrentSessionID(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Failed to extract session ID", http.StatusUnauthorized)
	}

	// Parse request body for optional includeCurrentSession flag
	var body struct {
		IncludeCurrentSession bool `json:"includeCurrentSession"`
	}
	// Ignore decode errors - default to false
	_ = json.NewDecoder(c.Request().Body).Decode(&body)

	count, err := h.svc.RevokeAll(c.Request().Context(), uid, body.IncludeCurrentSession, currentSID)
	if err != nil {
		return handleError(c, err, "REVOKE_ALL_FAILED", "Failed to revoke sessions", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"revokedCount": count,
		"status":       "revoked",
	})
}

// RevokeOthers revokes all sessions except the current one
func (h *Handler) RevokeOthers(c forge.Context) error {
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}
	token := cookie.Value
	uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Invalid or expired session token", http.StatusUnauthorized)
	}

	// Get current session ID
	currentSID, err := h.svc.GetCurrentSessionID(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Failed to extract session ID", http.StatusUnauthorized)
	}

	count, err := h.svc.RevokeAllExceptCurrent(c.Request().Context(), uid, currentSID)
	if err != nil {
		return handleError(c, err, "REVOKE_OTHERS_FAILED", "Failed to revoke other sessions", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"revokedCount": count,
		"status":       "revoked",
	})
}

// Refresh extends the current session's expiry time
func (h *Handler) Refresh(c forge.Context) error {
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}
	token := cookie.Value
	uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Invalid or expired session token", http.StatusUnauthorized)
	}

	// Get current session ID
	currentSID, err := h.svc.GetCurrentSessionID(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Failed to extract session ID", http.StatusUnauthorized)
	}

	sess, err := h.svc.RefreshCurrent(c.Request().Context(), uid, currentSID)
	if err != nil {
		return handleError(c, err, "REFRESH_FAILED", "Failed to refresh session", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, &SessionTokenResponse{Session: sess, Token: sess.Token})
}

// GetStats returns aggregated session statistics for the current user
func (h *Handler) GetStats(c forge.Context) error {
	cookie, err := c.Request().Cookie("session_token")
	if err != nil || cookie == nil {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}
	token := cookie.Value
	uid, err := h.svc.CurrentUserFromToken(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Invalid or expired session token", http.StatusUnauthorized)
	}

	stats, err := h.svc.GetStats(c.Request().Context(), uid)
	if err != nil {
		return handleError(c, err, "STATS_FAILED", "Failed to retrieve session statistics", http.StatusInternalServerError)
	}

	// Convert to response format
	response := map[string]interface{}{
		"totalSessions":  stats.TotalSessions,
		"activeSessions": stats.ActiveSessions,
		"deviceCount":    stats.DeviceCount,
		"locationCount":  stats.LocationCount,
	}

	if stats.OldestSession != nil {
		response["oldestSession"] = stats.OldestSession.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if stats.NewestSession != nil {
		response["newestSession"] = stats.NewestSession.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return c.JSON(http.StatusOK, response)
}
