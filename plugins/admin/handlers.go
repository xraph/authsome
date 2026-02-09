package admin

import (
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/types"
	"github.com/xraph/forge"
)

// Handler handles admin HTTP requests
// Updated for V2 architecture: App → Environment → Organization.
type Handler struct {
	service *Service
}

// Request types.
type CreateUserRequestDTO struct {
	Email         string            `json:"email"              validate:"required,email"`
	Password      string            `json:"password,omitempty"`
	Name          string            `json:"name,omitempty"`
	Username      string            `json:"username,omitempty"`
	Role          string            `json:"role,omitempty"`
	EmailVerified bool              `json:"email_verified"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type ListUsersRequestDTO struct {
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	Search string `query:"search"`
	Role   string `query:"role"`
	Status string `query:"status"`
}

type DeleteUserRequestDTO struct {
	ID string `path:"id" validate:"required"`
}

type BanUserRequestDTO struct {
	ID        string     `path:"id"                   validate:"required"`
	Reason    string     `json:"reason,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type UnbanUserRequestDTO struct {
	ID     string `path:"id"               validate:"required"`
	Reason string `json:"reason,omitempty"`
}

type SetUserRoleRequestDTO struct {
	ID   string `path:"id"   validate:"required"`
	Role string `json:"role" validate:"required"`
}

type RevokeSessionRequestDTO struct {
	ID string `path:"id" validate:"required"`
}

type ImpersonateUserRequestDTO struct {
	ID       string        `path:"id"                 validate:"required"`
	Duration time.Duration `json:"duration,omitempty"`
}

type GetStatsRequestDTO struct {
	Period string `query:"period"`
}

type ListSessionsRequestDTO struct {
	Page   int    `query:"page"`
	Limit  int    `query:"limit"`
	UserID string `query:"user_id"`
}

type GetAuditLogsRequestDTO struct {
	Page     int `query:"page"`
	PageSize int `query:"page_size"`
}

// Response types - use shared responses from core.
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse

type StatsResponse struct {
	ActiveUsers    int    `json:"active_users"`
	ActiveSessions int    `json:"active_sessions"`
	TotalUsers     int    `json:"total_users"`
	TotalSessions  int    `json:"total_sessions"`
	BannedUsers    int    `json:"banned_users"`
	Timestamp      string `json:"timestamp"`
}

// NewHandler creates a new admin handler.
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateUser handles POST /admin/users.
func (h *Handler) CreateUser(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("app_id"))
	}

	var reqBody CreateUserRequestDTO
	if err := c.BindRequest(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
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
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusCreated, user)
}

// ListUsers handles GET /admin/users.
func (h *Handler) ListUsers(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("App context required"))
	}

	var reqDTO ListUsersRequestDTO
	if err := c.BindRequest(&reqDTO); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Apply defaults
	page := max(reqDTO.Page, 1)

	limit := reqDTO.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

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
		Search:             reqDTO.Search,
		Role:               reqDTO.Role,
		Status:             reqDTO.Status,
		AdminID:            userID,
	}

	result, err := h.service.ListUsers(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, result)
}

// DeleteUser handles DELETE /admin/users/:id.
func (h *Handler) DeleteUser(c forge.Context) error {
	// Extract V2 context
	userID, _ := contexts.GetUserID(c.Request().Context())

	if userID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.BadRequest("Unauthorized"))
	}

	var req DeleteUserRequestDTO
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	targetUserID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
	}

	err = h.service.DeleteUser(c.Request().Context(), targetUserID, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "User deleted successfully"})
}

// BanUser handles POST /admin/users/:id/ban.
func (h *Handler) BanUser(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("App context and user required"))
	}

	var reqDTO BanUserRequestDTO
	if err := c.BindRequest(&reqDTO); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	targetUserID, err := xid.FromString(reqDTO.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
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
		Reason:             reqDTO.Reason,
		ExpiresAt:          reqDTO.ExpiresAt,
		AdminID:            adminID,
	}

	err = h.service.BanUser(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "User banned successfully"})
}

// UnbanUser handles POST /admin/users/:id/unban.
func (h *Handler) UnbanUser(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("App context and user required"))
	}

	var reqDTO UnbanUserRequestDTO
	if err := c.BindRequest(&reqDTO); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	targetUserID, err := xid.FromString(reqDTO.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
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
		Reason:             reqDTO.Reason,
		AdminID:            adminID,
	}

	err = h.service.UnbanUser(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "User unbanned successfully"})
}

// ImpersonateUser handles POST /admin/users/:id/impersonate.
func (h *Handler) ImpersonateUser(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("App context and user required"))
	}

	var reqDTO ImpersonateUserRequestDTO
	if err := c.BindRequest(&reqDTO); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	targetUserID, err := xid.FromString(reqDTO.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
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
		Duration:           reqDTO.Duration,
		IPAddress:          c.Request().RemoteAddr,
		UserAgent:          c.Request().UserAgent(),
		AdminID:            adminID,
	}

	session, err := h.service.ImpersonateUser(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, session)
}

// SetUserRole handles POST /admin/users/:id/role.
func (h *Handler) SetUserRole(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("App context and user required"))
	}

	var reqDTO SetUserRoleRequestDTO
	if err := c.BindRequest(&reqDTO); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	targetUserID, err := xid.FromString(reqDTO.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid user ID"))
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
		Role:               reqDTO.Role,
		AdminID:            adminID,
	}

	err = h.service.SetUserRole(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "User role updated successfully"})
}

// ListSessions handles GET /admin/sessions.
func (h *Handler) ListSessions(c forge.Context) error {
	// Extract V2 context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || adminID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("App context and user required"))
	}

	var reqDTO ListSessionsRequestDTO
	if err := c.BindRequest(&reqDTO); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	page := max(reqDTO.Page, 1)

	limit := reqDTO.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Optional user ID filter
	var userIDPtr *xid.ID

	if reqDTO.UserID != "" {
		if uid, err := xid.FromString(reqDTO.UserID); err == nil {
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
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, result)
}

// RevokeSession handles DELETE /admin/sessions/:id.
func (h *Handler) RevokeSession(c forge.Context) error {
	// Extract V2 context
	adminID, _ := contexts.GetUserID(c.Request().Context())

	if adminID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.BadRequest("Unauthorized"))
	}

	var req RevokeSessionRequestDTO
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	sessionID, err := xid.FromString(req.ID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid session ID"))
	}

	err = h.service.RevokeSession(c.Request().Context(), sessionID, adminID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "Session revoked successfully"})
}

// GetStats handles GET /admin/stats.
func (h *Handler) GetStats(c forge.Context) error {
	// Get admin user from context
	adminID, _ := contexts.GetUserID(c.Request().Context())
	if adminID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.BadRequest("Unauthorized"))
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

// GetAuditLogs handles GET /admin/audit.
func (h *Handler) GetAuditLogs(c forge.Context) error {
	// Get admin user from context
	adminID, _ := contexts.GetUserID(c.Request().Context())
	if adminID.IsNil() {
		return c.JSON(http.StatusUnauthorized, errs.BadRequest("Unauthorized"))
	}

	var reqDTO GetAuditLogsRequestDTO
	if err := c.BindRequest(&reqDTO); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	page := max(reqDTO.Page, 1)

	pageSize := reqDTO.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// For now, return empty audit logs
	// In a real implementation, you would query the audit service
	result := types.PaginatedResult{
		Data:       []any{},
		Total:      0,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: 0,
	}

	return c.JSON(http.StatusOK, result)
}
