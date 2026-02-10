package multisession

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

type Handler struct{ svc *Service }

// ErrorResponse types - use shared responses from core.
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse

type SessionsResponse = session.ListSessionsResponse

type SessionTokenResponse struct {
	Session *session.Session `json:"session"`
	Token   string           `json:"token"`
}

// RevokeResponse represents the response from revoking sessions.
type RevokeResponse struct {
	RevokedCount int    `json:"revokedCount"`
	Status       string `json:"status"`
}

// SessionStatsResponse represents aggregated session statistics.
type SessionStatsResponse struct {
	TotalSessions  int     `json:"totalSessions"`
	ActiveSessions int     `json:"activeSessions"`
	DeviceCount    int     `json:"deviceCount"`
	LocationCount  int     `json:"locationCount"`
	OldestSession  *string `json:"oldestSession,omitempty"` // ISO8601 timestamp
	NewestSession  *string `json:"newestSession,omitempty"` // ISO8601 timestamp
}

// SetActiveRequest represents the request to set an active session.
type SetActiveRequest struct {
	ID string `json:"id"`
}

// RevokeAllRequest represents the request to revoke all sessions.
type RevokeAllRequest struct {
	IncludeCurrentSession bool `json:"includeCurrentSession"`
}

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

// handleError returns the error in a structured format.
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	authErr := &errs.AuthsomeError{}
	if errors.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// List returns sessions for the current user with optional filtering.
func (h *Handler) List(c forge.Context) error {
	// Get auth context (works for both API key and session auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if !ok || !authCtx.IsAuthenticated {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid := getUserIDFromAuthContext(authCtx)
	if uid.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User authentication required", http.StatusUnauthorized))
	}

	// req filter parameters from query string
	var req ListSessionsRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid query parameters", http.StatusBadRequest).WithError(err))
	}

	// Set defaults if not provided
	if req.Limit <= 0 {
		req.Limit = 50
	}

	if req.Limit > 100 {
		req.Limit = 100 // Cap at 100
	}

	if req.SortBy == nil || *req.SortBy == "" {
		defaultSort := "created_at"
		req.SortBy = &defaultSort
	}

	if req.SortOrder == nil || *req.SortOrder == "" {
		defaultOrder := "desc"
		req.SortOrder = &defaultOrder
	}

	out, err := h.svc.List(c.Request().Context(), uid, &req)
	if err != nil {
		return handleError(c, err, "LIST_SESSIONS_FAILED", "Failed to list sessions", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, out)
}

// SetActive switches the current session cookie to the provided session id.
func (h *Handler) SetActive(c forge.Context) error {
	// Get auth context (works for both API key and session auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if !ok || !authCtx.IsAuthenticated {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid := getUserIDFromAuthContext(authCtx)
	if uid.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User authentication required", http.StatusUnauthorized))
	}

	token := getSessionTokenFromAuthContext(c, authCtx)
	if token == "" {
		return c.JSON(http.StatusUnauthorized, errs.New("NO_SESSION", "Session required for this operation", http.StatusUnauthorized))
	}

	var body SetActiveRequest
	if err := c.BindRequest(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest).WithError(err))
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

// Delete revokes a session by id for the current user.
func (h *Handler) Delete(c forge.Context) error {
	// Get auth context (works for both API key and session auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if !ok || !authCtx.IsAuthenticated {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid := getUserIDFromAuthContext(authCtx)
	if uid.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User authentication required", http.StatusUnauthorized))
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

// GetCurrent returns details about the currently active session.
func (h *Handler) GetCurrent(c forge.Context) error {
	// Get auth context (works for both API key and session auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if !ok || !authCtx.IsAuthenticated {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid := getUserIDFromAuthContext(authCtx)
	if uid.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User authentication required", http.StatusUnauthorized))
	}

	token := getSessionTokenFromAuthContext(c, authCtx)
	if token == "" {
		return c.JSON(http.StatusUnauthorized, errs.New("NO_SESSION", "Session required for this operation", http.StatusUnauthorized))
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

// GetByID returns details about a specific session by ID.
func (h *Handler) GetByID(c forge.Context) error {
	// Get auth context (works for both API key and session auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if !ok || !authCtx.IsAuthenticated {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid := getUserIDFromAuthContext(authCtx)
	if uid.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User authentication required", http.StatusUnauthorized))
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

// RevokeAll revokes all sessions for the current user.
func (h *Handler) RevokeAll(c forge.Context) error {
	// Get auth context (works for both API key and session auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if !ok || !authCtx.IsAuthenticated {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid := getUserIDFromAuthContext(authCtx)
	if uid.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User authentication required", http.StatusUnauthorized))
	}

	token := getSessionTokenFromAuthContext(c, authCtx)
	if token == "" {
		return c.JSON(http.StatusUnauthorized, errs.New("NO_SESSION", "Session required for this operation", http.StatusUnauthorized))
	}

	// Get current session ID
	currentSID, err := h.svc.GetCurrentSessionID(c.Request().Context(), token)
	if err != nil {
		return handleError(c, err, "INVALID_TOKEN", "Failed to extract session ID", http.StatusUnauthorized)
	}

	// body request body for optional includeCurrentSession flag
	var body RevokeAllRequest
	// _ decode errors - default to false
	_ = c.BindRequest(&body)

	count, err := h.svc.RevokeAll(c.Request().Context(), uid, body.IncludeCurrentSession, currentSID)
	if err != nil {
		return handleError(c, err, "REVOKE_ALL_FAILED", "Failed to revoke sessions", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, &RevokeResponse{
		RevokedCount: count,
		Status:       "revoked",
	})
}

// RevokeOthers revokes all sessions except the current one.
func (h *Handler) RevokeOthers(c forge.Context) error {
	// Get auth context (works for both API key and session auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if !ok || !authCtx.IsAuthenticated {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid := getUserIDFromAuthContext(authCtx)
	if uid.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User authentication required", http.StatusUnauthorized))
	}

	token := getSessionTokenFromAuthContext(c, authCtx)
	if token == "" {
		return c.JSON(http.StatusUnauthorized, errs.New("NO_SESSION", "Session required for this operation", http.StatusUnauthorized))
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

	return c.JSON(http.StatusOK, &RevokeResponse{
		RevokedCount: count,
		Status:       "revoked",
	})
}

// Refresh extends the current session's expiry time.
func (h *Handler) Refresh(c forge.Context) error {
	// Get auth context (works for both API key and session auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if !ok || !authCtx.IsAuthenticated {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid := getUserIDFromAuthContext(authCtx)
	if uid.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User authentication required", http.StatusUnauthorized))
	}

	token := getSessionTokenFromAuthContext(c, authCtx)
	if token == "" {
		return c.JSON(http.StatusUnauthorized, errs.New("NO_SESSION", "Session required for this operation", http.StatusUnauthorized))
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

// GetStats returns aggregated session statistics for the current user.
func (h *Handler) GetStats(c forge.Context) error {
	// Get auth context (works for both API key and session auth)
	authCtx, ok := contexts.GetAuthContext(c.Request().Context())
	if !ok || !authCtx.IsAuthenticated {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "Authentication required", http.StatusUnauthorized))
	}

	uid := getUserIDFromAuthContext(authCtx)
	if uid.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User authentication required", http.StatusUnauthorized))
	}

	stats, err := h.svc.GetStats(c.Request().Context(), uid)
	if err != nil {
		return handleError(c, err, "STATS_FAILED", "Failed to retrieve session statistics", http.StatusInternalServerError)
	}

	// Convert to response format
	response := &SessionStatsResponse{
		TotalSessions:  stats.TotalSessions,
		ActiveSessions: stats.ActiveSessions,
		DeviceCount:    stats.DeviceCount,
		LocationCount:  stats.LocationCount,
	}

	if stats.OldestSession != nil {
		oldest := stats.OldestSession.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		response.OldestSession = &oldest
	}

	if stats.NewestSession != nil {
		newest := stats.NewestSession.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		response.NewestSession = &newest
	}

	return c.JSON(http.StatusOK, response)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// getUserIDFromAuthContext extracts user ID from auth context
// getUserIDFromAuthContext with both session and API key authentication.
func getUserIDFromAuthContext(authCtx *contexts.AuthContext) xid.ID {
	if authCtx.User != nil {
		return authCtx.User.ID
	}

	if authCtx.Session != nil {
		return authCtx.Session.UserID
	}

	return xid.NilID()
}

// getSessionTokenFromAuthContext extracts session token from auth context or cookie fallback.
func getSessionTokenFromAuthContext(c forge.Context, authCtx *contexts.AuthContext) string {
	// Priority 1: Session from auth context
	if authCtx.Session != nil && authCtx.Session.Token != "" {
		return authCtx.Session.Token
	}

	// Priority 2: Fallback to cookie (for backward compatibility)
	if cookie, err := c.Request().Cookie("session_token"); err == nil && cookie != nil {
		return cookie.Value
	}

	return ""
}
