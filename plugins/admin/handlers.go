package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/types"
	"github.com/xraph/forge"
)

// Handler handles admin HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new admin handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// CreateUser handles POST /admin/users
func (h *Handler) CreateUser(c forge.Context) error {
	var req CreateUserRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Set admin ID from context
	if adminUser, ok := adminUserValue.(*user.User); ok {
		req.AdminID = adminUser.ID.String()
	}

	user, err := h.service.CreateUser(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, user)
}

// ListUsers handles GET /admin/users
func (h *Handler) ListUsers(c forge.Context) error {
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
	orgID := q.Get("organization_id")

	req := &ListUsersRequest{
		OrganizationID: orgID,
		Page:           page,
		Limit:          limit,
		Search:         search,
		Role:           role,
		Status:         status,
	}

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Set admin ID from context
	if adminUser, ok := adminUserValue.(*user.User); ok {
		req.AdminID = adminUser.ID.String()
	}

	result, err := h.service.ListUsers(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// DeleteUser handles DELETE /admin/users/:id
func (h *Handler) DeleteUser(c forge.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID is required",
		})
	}

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var adminID string
	if adminUser, ok := adminUserValue.(*user.User); ok {
		adminID = adminUser.ID.String()
	}

	err := h.service.DeleteUser(c.Request().Context(), userID, adminID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User deleted successfully",
	})
}

// BanUser handles POST /admin/users/:id/ban
func (h *Handler) BanUser(c forge.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID is required",
		})
	}

	var req BanUserRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	req.UserID = userID

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Set admin ID from context
	if adminUser, ok := adminUserValue.(*user.User); ok {
		req.AdminID = adminUser.ID.String()
	}

	err := h.service.BanUser(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User banned successfully",
	})
}

// UnbanUser handles POST /admin/users/:id/unban
func (h *Handler) UnbanUser(c forge.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID is required",
		})
	}

	var req UnbanUserRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	req.UserID = userID

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Set admin ID from context
	if adminUser, ok := adminUserValue.(*user.User); ok {
		req.AdminID = adminUser.ID.String()
	}

	err := h.service.UnbanUser(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User unbanned successfully",
	})
}

// ImpersonateUser handles POST /admin/users/:id/impersonate
func (h *Handler) ImpersonateUser(c forge.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID is required",
		})
	}

	var req ImpersonateUserRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	req.UserID = userID
	req.IPAddress = c.Request().RemoteAddr
	req.UserAgent = c.Request().UserAgent()

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Set admin ID from context
	if adminUser, ok := adminUserValue.(*user.User); ok {
		req.AdminID = adminUser.ID.String()
	}

	session, err := h.service.ImpersonateUser(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, session)
}

// SetUserRole handles POST /admin/users/:id/role
func (h *Handler) SetUserRole(c forge.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID is required",
		})
	}

	var req SetUserRoleRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	req.UserID = userID

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Set admin ID from context
	if adminUser, ok := adminUserValue.(*user.User); ok {
		req.AdminID = adminUser.ID.String()
	}

	err := h.service.SetUserRole(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User role updated successfully",
	})
}

// ListSessions handles GET /admin/sessions
func (h *Handler) ListSessions(c forge.Context) error {
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

	userID := q.Get("user_id")
	orgID := q.Get("organization_id")

	req := &ListSessionsRequest{
		UserID:         userID,
		OrganizationID: orgID,
		Page:           page,
		Limit:          limit,
	}

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Set admin ID from context
	if adminUser, ok := adminUserValue.(*user.User); ok {
		req.AdminID = adminUser.ID.String()
	}

	result, err := h.service.ListSessions(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// RevokeSession handles DELETE /admin/sessions/:id
func (h *Handler) RevokeSession(c forge.Context) error {
	sessionID := c.Param("id")
	if sessionID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Session ID is required",
		})
	}

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var adminID string
	if adminUser, ok := adminUserValue.(*user.User); ok {
		adminID = adminUser.ID.String()
	}

	err := h.service.RevokeSession(c.Request().Context(), sessionID, adminID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Session revoked successfully",
	})
}

// GetStats handles GET /admin/stats
func (h *Handler) GetStats(c forge.Context) error {
	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// For now, return basic stats
	// In a real implementation, you would gather actual statistics
	stats := map[string]interface{}{
		"total_users":     0,
		"active_users":    0,
		"banned_users":    0,
		"total_sessions":  0,
		"active_sessions": 0,
		"timestamp":       time.Now(),
	}

	return c.JSON(http.StatusOK, stats)
}

// GetAuditLogs handles GET /admin/audit
func (h *Handler) GetAuditLogs(c forge.Context) error {
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

	// Get admin user from context
	adminUserValue := c.Request().Context().Value("user")
	if adminUserValue == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
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
