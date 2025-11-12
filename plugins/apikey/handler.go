package apikey

import (
	"encoding/json"
	"strconv"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/internal/interfaces"
	"github.com/xraph/forge"
)

// Handler handles API key related HTTP requests
// Updated for V2 architecture: App → Environment → Organization
type Handler struct {
	service *apikey.Service
	config  Config
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
	appID := interfaces.GetAppID(c.Request().Context())
	envID := interfaces.GetEnvironmentID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	userID := interfaces.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, map[string]string{
			"error": "App context required",
		})
	}
	if userID.IsNil() {
		return c.JSON(401, map[string]string{
			"error": "Authentication required",
		})
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
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if reqBody.Name == "" {
		return c.JSON(400, map[string]string{
			"error": "Name is required",
		})
	}
	if len(reqBody.Scopes) == 0 {
		return c.JSON(400, map[string]string{
			"error": "At least one scope is required",
		})
	}

	// Build request with context
	var envIDPtr *xid.ID
	if !envID.IsNil() {
		envIDPtr = &envID
	}
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	req := &apikey.CreateAPIKeyRequest{
		AppID:         appID,
		EnvironmentID: envIDPtr,
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
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(201, map[string]interface{}{
		"api_key": key,
		"message": "API key created successfully. Store the key securely - it won't be shown again.",
	})
}

// ListAPIKeys handles GET /api-keys
func (h *Handler) ListAPIKeys(c forge.Context) error {
	// Extract context
	appID := interfaces.GetAppID(c.Request().Context())
	envID := interfaces.GetEnvironmentID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	userID := interfaces.GetUserID(c.Request().Context())

	if appID.IsNil() {
		return c.JSON(400, map[string]string{
			"error": "App context required",
		})
	}

	// Parse pagination parameters
	limitStr := c.Request().URL.Query().Get("limit")
	offsetStr := c.Request().URL.Query().Get("offset")

	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build request with context
	var envIDPtr *xid.ID
	if !envID.IsNil() {
		envIDPtr = &envID
	}
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}
	var userIDPtr *xid.ID
	if !userID.IsNil() {
		userIDPtr = &userID
	}

	req := &apikey.ListAPIKeysRequest{
		AppID:          appID,
		EnvironmentID:  envIDPtr,
		OrganizationID: orgIDPtr,
		UserID:         userIDPtr,
		Limit:          limit,
		Offset:         offset,
	}

	response, err := h.service.ListAPIKeys(c.Request().Context(), req)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, response)
}

// GetAPIKey handles GET /api-keys/:id
func (h *Handler) GetAPIKey(c forge.Context) error {
	// Extract context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	userID := interfaces.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		return c.JSON(401, map[string]string{
			"error": "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		return c.JSON(400, map[string]string{
			"error": "Key ID is required",
		})
	}

	keyID, err := xid.FromString(keyIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid key ID format",
		})
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	key, err := h.service.GetAPIKey(c.Request().Context(), appID, keyID, userID, orgIDPtr)
	if err != nil {
		return c.JSON(404, map[string]string{
			"error": "API key not found",
		})
	}

	return c.JSON(200, key)
}

// UpdateAPIKey handles PATCH /api-keys/:id
func (h *Handler) UpdateAPIKey(c forge.Context) error {
	// Extract context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	userID := interfaces.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		return c.JSON(401, map[string]string{
			"error": "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		return c.JSON(400, map[string]string{
			"error": "Key ID is required",
		})
	}

	keyID, err := xid.FromString(keyIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid key ID format",
		})
	}

	var req apikey.UpdateAPIKeyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	key, err := h.service.UpdateAPIKey(c.Request().Context(), appID, keyID, userID, orgIDPtr, &req)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, key)
}

// DeleteAPIKey handles DELETE /api-keys/:id
func (h *Handler) DeleteAPIKey(c forge.Context) error {
	// Extract context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	userID := interfaces.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		return c.JSON(401, map[string]string{
			"error": "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		return c.JSON(400, map[string]string{
			"error": "Key ID is required",
		})
	}

	keyID, err := xid.FromString(keyIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid key ID format",
		})
	}

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	err = h.service.DeleteAPIKey(c.Request().Context(), appID, keyID, userID, orgIDPtr)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, map[string]string{
		"message": "API key deleted successfully",
	})
}

// RotateAPIKey handles POST /api-keys/:id/rotate
func (h *Handler) RotateAPIKey(c forge.Context) error {
	// Extract context
	appID := interfaces.GetAppID(c.Request().Context())
	envID := interfaces.GetEnvironmentID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	userID := interfaces.GetUserID(c.Request().Context())

	if appID.IsNil() || userID.IsNil() {
		return c.JSON(401, map[string]string{
			"error": "Authentication required",
		})
	}

	// Parse key ID
	keyIDStr := c.Param("id")
	if keyIDStr == "" {
		return c.JSON(400, map[string]string{
			"error": "Key ID is required",
		})
	}

	keyID, err := xid.FromString(keyIDStr)
	if err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid key ID format",
		})
	}

	var envIDPtr *xid.ID
	if !envID.IsNil() {
		envIDPtr = &envID
	}
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	rotateReq := &apikey.RotateAPIKeyRequest{
		ID:             keyID,
		AppID:          appID,
		EnvironmentID:  envIDPtr,
		OrganizationID: orgIDPtr,
		UserID:         userID,
	}

	newKey, err := h.service.RotateAPIKey(c.Request().Context(), rotateReq)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, map[string]interface{}{
		"api_key": newKey,
		"message": "API key rotated successfully. Store the new key securely - it won't be shown again.",
	})
}

// VerifyAPIKey handles POST /api-keys/verify
func (h *Handler) VerifyAPIKey(c forge.Context) error {
	var req apikey.VerifyAPIKeyRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Key == "" {
		return c.JSON(400, map[string]string{
			"error": "API key is required",
		})
	}

	// Set IP and User Agent from request
	req.IP = c.Request().RemoteAddr
	req.UserAgent = c.Request().Header.Get("User-Agent")

	response, err := h.service.VerifyAPIKey(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	if !response.Valid {
		return c.JSON(401, map[string]string{
			"error": response.Error,
		})
	}

	return c.JSON(200, response)
}
