package apikey

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles API key related HTTP requests
// Updated for V2 architecture: App → Environment → Organization
type Handler struct {
	service *apikey.Service
	config  Config
}

// Response types
type CreateAPIKeyResponse struct {
	APIKey  *apikey.APIKey `json:"api_key"`
	Message string         `json:"message"`
}

// Use shared response type
type MessageResponse = responses.MessageResponse

type RotateAPIKeyResponse struct {
	APIKey  *apikey.APIKey `json:"api_key"`
	Message string         `json:"message"`
}

type RolesResponse struct {
	Roles []*apikey.Role `json:"roles"`
}

// NewHandler creates a new API key handler
func NewHandler(service *apikey.Service, config Config) *Handler {
	return &Handler{
		service: service,
		config:  config,
	}
}

// CreateAPIKey handles POST /api-keys
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

	// Parse request body (only for mutable fields)
	var reqBody struct {
		Name        string            `json:"name"`
		Description string            `json:"description,omitempty"`
		Scopes      []string          `json:"scopes"`
		Permissions map[string]string `json:"permissions,omitempty"`
		RateLimit   int               `json:"rate_limit,omitempty"`
		AllowedIPs  []string          `json:"allowed_ips,omitempty"`
		Metadata    map[string]string `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		authErr := errs.New("INVALID_REQUEST", "Invalid request body", 400)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	// Validate required fields
	if reqBody.Name == "" {
		err := errs.New("NAME_REQUIRED", "Name is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}
	if len(reqBody.Scopes) == 0 {
		err := errs.New("SCOPES_REQUIRED", "At least one scope is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	// Build request with context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &apikey.CreateAPIKeyRequest{
		AppID:         appID,
		EnvironmentID: envID,
		OrgID:         orgIDPtr,
		UserID:        userID,
		Name:          reqBody.Name,
		Description:   reqBody.Description,
		Scopes:        reqBody.Scopes,
		Permissions:   reqBody.Permissions,
		RateLimit:     reqBody.RateLimit,
		AllowedIPs:    reqBody.AllowedIPs,
		Metadata:      reqBody.Metadata,
	}

	key, err := h.service.CreateAPIKey(c.Request().Context(), req)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
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

// ListAPIKeys handles GET /api-keys
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

	// Parse pagination and filter parameters
	query := c.Request().URL.Query()

	page := 1 // default
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	limit := 20 // default
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Build filter with optional parameters
	filter := &apikey.ListAPIKeysFilter{
		AppID: appID,
	}
	filter.Page = page
	filter.Limit = limit

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

	// Parse active filter
	if activeStr := query.Get("active"); activeStr != "" {
		if active, err := strconv.ParseBool(activeStr); err == nil {
			filter.Active = &active
		}
	}

	response, err := h.service.ListAPIKeys(c.Request().Context(), filter)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)
		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, response)
}

// GetAPIKey handles GET /api-keys/:id
func (h *Handler) GetAPIKey(c forge.Context) error {
	// Extract context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)
		return c.JSON(err.HTTPStatus, err)
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		err := errs.New("KEY_ID_REQUIRED", "Key ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	keyID, err := xid.FromString(keyIDStr)
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
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)
		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, key)
}

// UpdateAPIKey handles PATCH /api-keys/:id
func (h *Handler) UpdateAPIKey(c forge.Context) error {
	// Extract context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)
		return c.JSON(err.HTTPStatus, err)
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		err := errs.New("KEY_ID_REQUIRED", "Key ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	keyID, err := xid.FromString(keyIDStr)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var req apikey.UpdateAPIKeyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		authErr := errs.New("INVALID_REQUEST", "Invalid request body", 400)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	key, err := h.service.UpdateAPIKey(c.Request().Context(), appID, keyID, userID, orgIDPtr, &req)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)
		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, key)
}

// DeleteAPIKey handles DELETE /api-keys/:id
func (h *Handler) DeleteAPIKey(c forge.Context) error {
	// Extract context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	userID, _ := contexts.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)
		return c.JSON(err.HTTPStatus, err)
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		err := errs.New("KEY_ID_REQUIRED", "Key ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	keyID, err := xid.FromString(keyIDStr)
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
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		internalErr := errs.New("INTERNAL_ERROR", "Internal server error", 500)
		return c.JSON(internalErr.HTTPStatus, internalErr)
	}

	return c.JSON(200, &MessageResponse{
		Message: "API key deleted successfully",
	})
}

// RotateAPIKey handles POST /api-keys/:id/rotate
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

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		err := errs.New("KEY_ID_REQUIRED", "Key ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	keyID, err := xid.FromString(keyIDStr)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	rotateReq := &apikey.RotateAPIKeyRequest{
		ID:             keyID,
		AppID:          appID,
		EnvironmentID:  envID,
		OrganizationID: orgIDPtr,
		UserID:         userID,
	}

	newKey, err := h.service.RotateAPIKey(c.Request().Context(), rotateReq)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
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

// VerifyAPIKey handles POST /api-keys/verify
func (h *Handler) VerifyAPIKey(c forge.Context) error {
	var req apikey.VerifyAPIKeyRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		authErr := errs.New("INVALID_REQUEST", "Invalid request body", 400)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	if req.Key == "" {
		err := errs.New("KEY_REQUIRED", "API key is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	// Set IP and User Agent from request
	req.IP = c.Request().RemoteAddr
	req.UserAgent = c.Request().Header.Get("User-Agent")

	response, err := h.service.VerifyAPIKey(c.Request().Context(), &req)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
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

// AssignRole handles POST /api-keys/:id/roles
func (h *Handler) AssignRole(c forge.Context) error {
	// Extract context
	userID, _ := contexts.GetUserID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)
		return c.JSON(err.HTTPStatus, err)
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		err := errs.New("KEY_ID_REQUIRED", "Key ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	keyID, err := xid.FromString(keyIDStr)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	// Parse request body
	var reqBody struct {
		RoleID string `json:"roleID"`
	}

	if err := json.NewDecoder(c.Request().Body).Decode(&reqBody); err != nil {
		authErr := errs.New("INVALID_REQUEST", "Invalid request body", 400)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	if reqBody.RoleID == "" {
		err := errs.New("ROLE_ID_REQUIRED", "Role ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	roleID, err := xid.FromString(reqBody.RoleID)
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

// UnassignRole handles DELETE /api-keys/:id/roles/:roleId
func (h *Handler) UnassignRole(c forge.Context) error {
	// Extract context
	userID, _ := contexts.GetUserID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	if userID.IsNil() {
		err := errs.New("AUTHENTICATION_REQUIRED", "Authentication required", 401)
		return c.JSON(err.HTTPStatus, err)
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		err := errs.New("KEY_ID_REQUIRED", "Key ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	keyID, err := xid.FromString(keyIDStr)
	if err != nil {
		authErr := errs.New("INVALID_KEY_ID", "Invalid key ID format", 400)
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	// Parse role ID
	roleIDStr := c.Param("roleId")
	if roleIDStr == "" {
		err := errs.New("ROLE_ID_REQUIRED", "Role ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	roleID, err := xid.FromString(roleIDStr)
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

// GetRoles handles GET /api-keys/:id/roles
func (h *Handler) GetRoles(c forge.Context) error {
	// Extract context
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		err := errs.New("KEY_ID_REQUIRED", "Key ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	keyID, err := xid.FromString(keyIDStr)
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

// GetEffectivePermissions handles GET /api-keys/:id/permissions
func (h *Handler) GetEffectivePermissions(c forge.Context) error {
	// Extract context
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		err := errs.New("KEY_ID_REQUIRED", "Key ID is required", 400)
		return c.JSON(err.HTTPStatus, err)
	}

	keyID, err := xid.FromString(keyIDStr)
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
