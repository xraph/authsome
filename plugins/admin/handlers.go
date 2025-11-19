package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/types"
	"github.com/xraph/forge"
)

// Handler handles admin HTTP requests
// Updated for V2 architecture: App → Environment → Organization
type Handler struct {
	service *Service
}

// Response types
type ErrorResponse struct {
	Error string `json:"error"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type StatsResponse struct {
	ActiveUsers    int    `json:"active_users"`
	ActiveSessions int    `json:"active_sessions"`
	TotalUsers     int    `json:"total_users"`
	TotalSessions  int    `json:"total_sessions"`
	BannedUsers    int    `json:"banned_users"`
	Timestamp      string `json:"timestamp"`
}

// NewHandler creates a new admin handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateUser handles POST /admin/users
func (h *Handler) CreateUser(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "App context required",
		})
	}

	var reqBody struct {
		Email         string            `json:"email"`
		Password      string            `json:"password,omitempty"`
		Name          string            `json:"name,omitempty"`
		Username      string            `json:"username,omitempty"`
		Role          string            `json:"role,omitempty"`
		EmailVerified bool              `json:"email_verified"`
		Metadata      map[string]string `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid request body",
		})
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &CreateUserRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		Email:              reqBody.Email,
		Password:           reqBody.Password,
		Name:               reqBody.Name,
		Username:           reqBody.Username,
		Role:               reqBody.Role,
		EmailVerified:      reqBody.EmailVerified,
		Metadata:           reqBody.Metadata,
		AdminID:            userID,
	}

	user, err := h.service.CreateUser(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, user)
}

// ListUsers handles GET /admin/users
func (h *Handler) ListUsers(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "App context required",
		})
	}

	// Parse query parameters
	q := c.Request().URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	status := q.Get("status")
	search := q.Get("search")
	role := q.Get("role")

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &ListUsersRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		Page:               page,
		Limit:              limit,
		Search:             search,
		Role:               role,
		Status:             status,
		AdminID:            userID,
	}

	result, err := h.service.ListUsers(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// DeleteUser handles DELETE /admin/users/:id
func (h *Handler) DeleteUser(c forge.Context) error {
	// Extract V2 context
	userID, _ := contexts.GetUserID(c.Request().Context())

	if userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, &ErrorResponse{Error: "Unauthorized",
		})
	}

	// Parse target user ID from URL
	targetUserIDStr := c.Param("id")
	if targetUserIDStr == "" {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "User ID is required",
		})
	}

	targetUserID, err := xid.FromString(targetUserIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid user ID",
		})
	}

	err = h.service.DeleteUser(c.Request().Context(), targetUserID, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "User deleted successfully",
	})
}

// BanUser handles POST /admin/users/:id/ban
func (h *Handler) BanUser(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "App context and user required",
		})
	}

	// Parse target user ID from URL
	targetUserIDStr := c.Param("id")
	if targetUserIDStr == "" {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "User ID is required",
		})
	}

	targetUserID, err := xid.FromString(targetUserIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid user ID",
		})
	}

	var reqBody struct {
		Reason    string     `json:"reason"`
		ExpiresAt *time.Time `json:"expires_at,omitempty"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid request body",
		})
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &BanUserRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		UserID:             targetUserID,
		Reason:             reqBody.Reason,
		ExpiresAt:          reqBody.ExpiresAt,
		AdminID:            adminID,
	}

	err = h.service.BanUser(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "User banned successfully",
	})
}

// UnbanUser handles POST /admin/users/:id/unban
func (h *Handler) UnbanUser(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "App context and user required",
		})
	}

	// Parse target user ID from URL
	targetUserIDStr := c.Param("id")
	if targetUserIDStr == "" {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "User ID is required",
		})
	}

	targetUserID, err := xid.FromString(targetUserIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid user ID",
		})
	}

	var reqBody struct {
		Reason string `json:"reason,omitempty"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid request body",
		})
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &UnbanUserRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		UserID:             targetUserID,
		Reason:             reqBody.Reason,
		AdminID:            adminID,
	}

	err = h.service.UnbanUser(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "User unbanned successfully",
	})
}

// ImpersonateUser handles POST /admin/users/:id/impersonate
func (h *Handler) ImpersonateUser(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "App context and user required",
		})
	}

	// Parse target user ID from URL
	targetUserIDStr := c.Param("id")
	if targetUserIDStr == "" {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "User ID is required",
		})
	}

	targetUserID, err := xid.FromString(targetUserIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid user ID",
		})
	}

	var reqBody struct {
		Duration time.Duration `json:"duration,omitempty"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid request body",
		})
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &ImpersonateUserRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		UserID:             targetUserID,
		Duration:           reqBody.Duration,
		IPAddress:          c.Request().RemoteAddr,
		UserAgent:          c.Request().UserAgent(),
		AdminID:            adminID,
	}

	session, err := h.service.ImpersonateUser(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, session)
}

// SetUserRole handles POST /admin/users/:id/role
func (h *Handler) SetUserRole(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "App context and user required",
		})
	}

	// Parse target user ID from URL
	targetUserIDStr := c.Param("id")
	if targetUserIDStr == "" {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "User ID is required",
		})
	}

	targetUserID, err := xid.FromString(targetUserIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid user ID",
		})
	}

	var reqBody struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid request body",
		})
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &SetUserRoleRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		UserID:             targetUserID,
		Role:               reqBody.Role,
		AdminID:            adminID,
	}

	err = h.service.SetUserRole(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "User role updated successfully",
	})
}

// ListSessions handles GET /admin/sessions
func (h *Handler) ListSessions(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "App context and user required",
		})
	}

	// Parse query parameters
	q := c.Request().URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Optional user ID filter
	var userIDPtr *xid.ID
	if userIDStr := q.Get("user_id"); userIDStr != "" {
		if uid, err := xid.FromString(userIDStr); err == nil {
			userIDPtr = &uid
		}
	}

	// Build service request with V2 context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &ListSessionsRequest{
		AppID:              appID,
		UserOrganizationID: orgIDPtr,
		UserID:             userIDPtr,
		Page:               page,
		Limit:              limit,
		AdminID:            adminID,
	}

	result, err := h.service.ListSessions(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// RevokeSession handles DELETE /admin/sessions/:id
func (h *Handler) RevokeSession(c forge.Context) error {
	// Extract V2 context
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if adminID.IsNil() {
		return c.JSON(http.StatusUnauthorized, &ErrorResponse{Error: "Unauthorized",
		})
	}

	// Parse session ID from URL
	sessionIDStr := c.Param("id")
	if sessionIDStr == "" {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Session ID is required",
		})
	}

	sessionID, err := xid.FromString(sessionIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: "Invalid session ID",
		})
	}

	err = h.service.RevokeSession(c.Request().Context(), sessionID, adminID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &ErrorResponse{Error: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "Session revoked successfully",
	})
}

// GetStats handles GET /admin/stats
func (h *Handler) GetStats(c forge.Context) error {
	// Get admin user from context
	adminID, _ := contexts.GetUserID(c.Request().Context())
	if adminID.IsNil() {
		return c.JSON(http.StatusUnauthorized, &ErrorResponse{Error: "Unauthorized",
		})
	}

	// For now, return basic stats
	// In a real implementation, you would gather actual statistics
	stats := &StatsResponse{
		TotalUsers:     0,
		ActiveUsers:    0,
		BannedUsers:    0,
		TotalSessions:  0,
		ActiveSessions: 0,
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	return c.JSON(http.StatusOK, stats)
}

// GetAuditLogs handles GET /admin/audit
func (h *Handler) GetAuditLogs(c forge.Context) error {
	// Get admin user from context
	adminID, _ := contexts.GetUserID(c.Request().Context())
	if adminID.IsNil() {
		return c.JSON(http.StatusUnauthorized, &ErrorResponse{Error: "Unauthorized",
		})
	}

	// Parse query parameters
	q := c.Request().URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// For now, return empty audit logs
	// In a real implementation, you would query the audit service
	result := types.PaginatedResult{
		Data:       []interface{}{},
		Total:      0,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 0,
	}

	return c.JSON(http.StatusOK, result)
}
