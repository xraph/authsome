package apikey

import (
	"errors"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles API key related HTTP requests
// Updated for V2 architecture: App → Environment → Organization.
type Handler struct {
	service *apikey.Service
	config  Config
}

// Request types.
type CreateAPIKeyRequest struct {
	Name        string            `json:"name"        validate:"required"`
	Description string            `json:"description"`
	Scopes      []string          `json:"scopes"      validate:"required,min=1"`
	Permissions map[string]string `json:"permissions"`
	RateLimit   int               `json:"rate_limit"`
	AllowedIPs  []string          `json:"allowed_ips"`
	Metadata    map[string]string `json:"metadata"`
}

type ListAPIKeysRequest struct {
	Page   int   `query:"page"`
	Limit  int   `query:"limit"`
	Active *bool `query:"active"`
}

type GetAPIKeyRequest struct {
	ID string `path:"id" validate:"required"`
}

type UpdateAPIKeyRequest struct {
	ID          string            `path:"id"          validate:"required"`
	Name        *string           `json:"name"`
	Description *string           `json:"description"`
	Scopes      []string          `json:"scopes"`
	Permissions map[string]string `json:"permissions"`
	RateLimit   *int              `json:"rate_limit"`
	AllowedIPs  []string          `json:"allowed_ips"`
	Metadata    map[string]string `json:"metadata"`
}

type DeleteAPIKeyRequest struct {
	ID string `path:"id" validate:"required"`
}

type RotateAPIKeyRequest struct {
	ID string `path:"id" validate:"required"`
}

type VerifyAPIKeyRequest struct {
	Key string `json:"key" validate:"required"`
}

type AssignRoleRequest struct {
	ID     string `path:"id"     validate:"required"`
	RoleID string `json:"roleID" validate:"required"`
}

type UnassignRoleRequest struct {
	ID     string `path:"id"     validate:"required"`
	RoleID string `path:"roleId" validate:"required"`
}

type GetRolesRequest struct {
	ID string `path:"id" validate:"required"`
}

type GetEffectivePermissionsRequest struct {
	ID string `path:"id" validate:"required"`
}

// Response types.
type CreateAPIKeyResponse struct {
	APIKey  *apikey.APIKey `json:"api_key"`
	Message string         `json:"message"`
}

// Use shared response type.
type MessageResponse = responses.MessageResponse

type RotateAPIKeyResponse struct {
	APIKey  *apikey.APIKey `json:"api_key"`
	Message string         `json:"message"`
}

type RolesResponse struct {
	Roles []*apikey.Role `json:"roles"`
}

// NewHandler creates a new API key handler.
func NewHandler(service *apikey.Service, config Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
	}
}

// CreateAPIKey handles POST /api-keys.
func (h *Handler) CreateAPIKey(c forge.Context) error {
	// Extract context (set by auth middleware)
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		err := errs.New("MISSING_APP_ID", "App context required", 400)

		return c.JSON(err.HTTPStatus, err)
	}

	if envID.IsNil() {
		err := errs.New("MISSING_ENV_ID", "Environment context required", 400)

		return c.JSON(err.HTTPStatus, err)
	}

	if userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)

		return c.JSON(err.HTTPStatus, err)
	}

	var req CreateAPIKeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	// Build request with context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	serviceReq := &apikey.CreateAPIKeyRequest{
		AppID:         appID,
		EnvironmentID: envID,
		OrgID:         orgIDPtr,
		UserID:        userID,
		Name:          req.Name,
		Description:   req.Description,
		Scopes:        req.Scopes,
		Permissions:   req.Permissions,
		RateLimit:     req.RateLimit,
		AllowedIPs:    req.AllowedIPs,
		Metadata:      req.Metadata,
	}

	key, err := h.service.CreateAPIKey(c.Request().Context(), serviceReq)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(201, &CreateAPIKeyResponse{
		APIKey:  key,
		Message: "API key created successfully. Store the key securely - it won't be shown again.",
	})
}

// ListAPIKeys handles GET /api-keys.
func (h *Handler) ListAPIKeys(c forge.Context) error {
	// Extract context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() {
		err := errs.New("MISSING_APP_ID", "App context required", 400)

		return c.JSON(err.HTTPStatus, err)
	}

	var req ListAPIKeysRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	// Build filter with optional parameters
	filter := &apikey.ListAPIKeysFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  req.Page,
			Limit: req.Limit,
		},
		AppID:  appID,
		Active: req.Active,
	}

	// Optional filters
	if !envID.IsNil() {
		filter.EnvironmentID = &envID
	}

	if !orgID.IsNil() {
		filter.OrganizationID = &orgID
	}

	if !userID.IsNil() {
		filter.UserID = &userID
	}

	response, err := h.service.ListAPIKeys(c.Request().Context(), filter)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, response)
}

// GetAPIKey handles GET /api-keys/:id.
func (h *Handler) GetAPIKey(c forge.Context) error {
	// Extract context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)

		return c.JSON(err.HTTPStatus, err)
	}

	var req GetAPIKeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	keyID, err := xid.FromString(req.ID)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	key, err := h.service.GetAPIKey(c.Request().Context(), appID, keyID, userID, orgIDPtr)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, key)
}

// UpdateAPIKey handles PATCH /api-keys/:id.
func (h *Handler) UpdateAPIKey(c forge.Context) error {
	// Extract context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)

		return c.JSON(err.HTTPStatus, err)
	}

	var req UpdateAPIKeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	keyID, err := xid.FromString(req.ID)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	serviceReq := &apikey.UpdateAPIKeyRequest{
		Name:        req.Name,
		Description: req.Description,
		Scopes:      req.Scopes,
		Permissions: req.Permissions,
		RateLimit:   req.RateLimit,
		Metadata:    req.Metadata,
	}

	key, err := h.service.UpdateAPIKey(c.Request().Context(), appID, keyID, userID, orgIDPtr, serviceReq)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, key)
}

// DeleteAPIKey handles DELETE /api-keys/:id.
func (h *Handler) DeleteAPIKey(c forge.Context) error {
	// Extract context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)

		return c.JSON(err.HTTPStatus, err)
	}

	var req DeleteAPIKeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	keyID, err := xid.FromString(req.ID)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	err = h.service.DeleteAPIKey(c.Request().Context(), appID, keyID, userID, orgIDPtr)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, &MessageResponse{
		Message: "API key deleted successfully",
	})
}

// RotateAPIKey handles POST /api-keys/:id/rotate.
func (h *Handler) RotateAPIKey(c forge.Context) error {
	// Extract context
	appID, _ := contexts.GetAppID(c.Request().Context())
	envID, _ := contexts.GetEnvironmentID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || envID.IsNil() || userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication and environment context required", 401)

		return c.JSON(err.HTTPStatus, err)
	}

	var req RotateAPIKeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	keyID, err := xid.FromString(req.ID)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	serviceReq := &apikey.RotateAPIKeyRequest{
		ID:             keyID,
		AppID:          appID,
		EnvironmentID:  envID,
		OrganizationID: orgIDPtr,
		UserID:         userID,
	}

	newKey, err := h.service.RotateAPIKey(c.Request().Context(), serviceReq)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, &RotateAPIKeyResponse{
		APIKey:  newKey,
		Message: "API key rotated successfully. Store the new key securely - it won't be shown again.",
	})
}

// VerifyAPIKey handles POST /api-keys/verify.
func (h *Handler) VerifyAPIKey(c forge.Context) error {
	var req VerifyAPIKeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	// Set IP and User Agent from request
	serviceReq := &apikey.VerifyAPIKeyRequest{
		Key:       req.Key,
		IP:        c.Request().RemoteAddr,
		UserAgent: c.Request().Header.Get("User-Agent"),
	}

	response, err := h.service.VerifyAPIKey(c.Request().Context(), serviceReq)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	if !response.Valid {
		err := errs.New("INVALID_KEY", response.Error, 401)

		return c.JSON(err.HTTPStatus, err)
	}

	return c.JSON(200, response)
}

// =============================================================================
// RBAC ROLE MANAGEMENT HANDLERS
// =============================================================================

// AssignRole handles POST /api-keys/:id/roles.
func (h *Handler) AssignRole(c forge.Context) error {
	// Extract context
	userID, _ := contexts.GetUserID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)

		return c.JSON(err.HTTPStatus, err)
	}

	var req AssignRoleRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	keyID, err := xid.FromString(req.ID)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	roleID, err := xid.FromString(req.RoleID)
	if err != nil {
		authErr := errs.New("INVALID_ROLE_ID", "Invalid role ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	// Assign role
	if err := h.service.AssignRole(c.Request().Context(), keyID, roleID, orgIDPtr, &userID); err != nil {
		internalErr := errs.New("INTERNAL_ERROR", err.Error(), 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, &MessageResponse{
		Message: "Role assigned successfully",
	})
}

// UnassignRole handles DELETE /api-keys/:id/roles/:roleId.
func (h *Handler) UnassignRole(c forge.Context) error {
	// Extract context
	userID, _ := contexts.GetUserID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)

		return c.JSON(err.HTTPStatus, err)
	}

	var req UnassignRoleRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	keyID, err := xid.FromString(req.ID)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	roleID, err := xid.FromString(req.RoleID)
	if err != nil {
		authErr := errs.New("INVALID_ROLE_ID", "Invalid role ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	// Unassign role
	if err := h.service.UnassignRole(c.Request().Context(), keyID, roleID, orgIDPtr, &userID); err != nil {
		internalErr := errs.New("INTERNAL_ERROR", err.Error(), 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, &MessageResponse{
		Message: "Role unassigned successfully",
	})
}

// GetRoles handles GET /api-keys/:id/roles.
func (h *Handler) GetRoles(c forge.Context) error {
	// Extract context
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	var req GetRolesRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	keyID, err := xid.FromString(req.ID)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	// Get roles
	roles, err := h.service.GetRoles(c.Request().Context(), keyID, orgIDPtr)
	if err != nil {
		internalErr := errs.New("INTERNAL_ERROR", err.Error(), 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, &RolesResponse{
		Roles: roles,
	})
}

// GetEffectivePermissions handles GET /api-keys/:id/permissions.
func (h *Handler) GetEffectivePermissions(c forge.Context) error {
	// Extract context
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	var req GetEffectivePermissionsRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(400, errs.BadRequest(err.Error()))
	}

	keyID, err := xid.FromString(req.ID)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)

		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	// Get effective permissions
	effectivePerms, err := h.service.GetEffectivePermissions(c.Request().Context(), keyID, orgIDPtr)
	if err != nil {
		internalErr := errs.New("INTERNAL_ERROR", err.Error(), 500)

		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, effectivePerms)
}
