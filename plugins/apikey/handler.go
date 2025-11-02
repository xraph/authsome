package apikey

import (
	"encoding/json"
	"strconv"

	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/forge"
)

// Handler handles API key related HTTP requests
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
	var req apikey.CreateAPIKeyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Name == "" {
		return c.JSON(400, map[string]string{
			"error": "Name is required",
		})
	}

	if req.OrgID == "" {
		return c.JSON(400, map[string]string{
			"error": "Organization ID is required",
		})
	}

	if req.UserID == "" {
		return c.JSON(400, map[string]string{
			"error": "User ID is required",
		})
	}

	key, err := h.service.CreateAPIKey(c.Request().Context(), &req)
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
	orgID := c.Request().URL.Query().Get("org_id")
	userID := c.Request().URL.Query().Get("user_id")

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

	req := &apikey.ListAPIKeysRequest{
		OrgID:  orgID,
		UserID: userID,
		Limit:  limit,
		Offset: offset,
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
	keyID := c.Param("id")
	if keyID == "" {
		return c.JSON(400, map[string]string{
			"error": "Key ID is required",
		})
	}

	userID := c.Request().URL.Query().Get("user_id")
	orgID := c.Request().URL.Query().Get("org_id")

	if userID == "" || orgID == "" {
		return c.JSON(400, map[string]string{
			"error": "User ID and Organization ID are required",
		})
	}

	key, err := h.service.GetAPIKey(c.Request().Context(), keyID, userID, orgID)
	if err != nil {
		return c.JSON(404, map[string]string{
			"error": "API key not found",
		})
	}

	return c.JSON(200, key)
}

// UpdateAPIKey handles PUT /api-keys/:id
func (h *Handler) UpdateAPIKey(c forge.Context) error {
	keyID := c.Param("id")
	if keyID == "" {
		return c.JSON(400, map[string]string{
			"error": "Key ID is required",
		})
	}

	userID := c.Request().URL.Query().Get("user_id")
	orgID := c.Request().URL.Query().Get("org_id")

	if userID == "" || orgID == "" {
		return c.JSON(400, map[string]string{
			"error": "User ID and Organization ID are required",
		})
	}

	var req apikey.UpdateAPIKeyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	key, err := h.service.UpdateAPIKey(c.Request().Context(), keyID, userID, orgID, &req)
	if err != nil {
		return c.JSON(500, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(200, key)
}

// DeleteAPIKey handles DELETE /api-keys/:id
func (h *Handler) DeleteAPIKey(c forge.Context) error {
	keyID := c.Param("id")
	if keyID == "" {
		return c.JSON(400, map[string]string{
			"error": "Key ID is required",
		})
	}

	userID := c.Request().URL.Query().Get("user_id")
	orgID := c.Request().URL.Query().Get("org_id")

	if userID == "" || orgID == "" {
		return c.JSON(400, map[string]string{
			"error": "User ID and Organization ID are required",
		})
	}

	err := h.service.DeleteAPIKey(c.Request().Context(), keyID, userID, orgID)
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
	keyID := c.Param("id")
	if keyID == "" {
		return c.JSON(400, map[string]string{
			"error": "Key ID is required",
		})
	}

	var req struct {
		OrgID  string `json:"org_id"`
		UserID string `json:"user_id"`
	}
	
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.UserID == "" || req.OrgID == "" {
		return c.JSON(400, map[string]string{
			"error": "User ID and Organization ID are required",
		})
	}

	rotateReq := &apikey.RotateAPIKeyRequest{
		ID:     keyID,
		OrgID:  req.OrgID,
		UserID: req.UserID,
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

