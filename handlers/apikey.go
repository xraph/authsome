package handlers

import (
	"encoding/json"

	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/internal/utils"
	"github.com/xraph/forge"
)

// APIKeyHandler handles API key related HTTP requests
type APIKeyHandler struct {
	service *apikey.Service
}

// NewAPIKeyHandler creates a new API key handler
func NewAPIKeyHandler(service *apikey.Service) *APIKeyHandler {
	return &APIKeyHandler{
		service: service,
	}
}

// CreateAPIKey handles POST /api/keys
func (h *APIKeyHandler) CreateAPIKey(c forge.Context) error {
	var req apikey.CreateAPIKeyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.BadRequest("Invalid request body"))
	}

	// Validate required fields
	if req.Name == "" {
		return c.JSON(400, errs.BadRequest("Name is required"))
	}

	if req.AppID.IsNil() {
		return c.JSON(400, errs.BadRequest("App ID is required"))
	}

	key, err := h.service.CreateAPIKey(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(500, errs.InternalServerError(err.Error()))
	}

	return c.JSON(201, key)
}

type ListAPIKeysRequest = apikey.ListAPIKeysRequest

// ListAPIKeys handles GET /api/keys
func (h *APIKeyHandler) ListAPIKeys(c forge.Context, req ListAPIKeysRequest) error {
	response, err := h.service.ListAPIKeys(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(500, errs.InternalServerError(err.Error()))
	}

	return c.JSON(200, response)
}

// GetAPIKey handles GET /api/keys/{id}
func (h *APIKeyHandler) GetAPIKey(c forge.Context) error {
	keyID := utils.GetXIDParams(c, "id")
	if keyID.IsNil() {
		return c.JSON(400, errs.BadRequest("Key ID is required"))
	}

	userID := c.Request().URL.Query().Get("user_id")
	orgID := c.Request().URL.Query().Get("org_id")

	if userID == "" || orgID == "" {
		return c.JSON(400, errs.BadRequest("User ID and Organization ID are required"))
	}

	key, err := h.service.GetAPIKey(c.Request().Context(), keyID, userID, orgID)
	if err != nil {
		return c.JSON(404, errs.NotFound("API key not found"))
	}

	return c.JSON(200, key)
}

// UpdateAPIKey handles PUT /api/keys/{id}
func (h *APIKeyHandler) UpdateAPIKey(c forge.Context) error {
	keyID := c.Param("id")
	if keyID == "" {
		return c.JSON(400, errs.BadRequest("Key ID is required"))
	}

	userID := c.Request().URL.Query().Get("user_id")
	orgID := c.Request().URL.Query().Get("org_id")

	if userID == "" || orgID == "" {
		return c.JSON(400, errs.BadRequest("User ID and Organization ID are required"))
	}

	var req apikey.UpdateAPIKeyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.BadRequest("Invalid request body"))
	}

	key, err := h.service.UpdateAPIKey(c.Request().Context(), keyID, userID, orgID, &req)
	if err != nil {
		return c.JSON(500, errs.InternalServerError(err.Error()))
	}

	return c.JSON(200, key)
}

type UpdateAPIKeyRequest = apikey.UpdateAPIKeyRequest

// DeleteAPIKey handles DELETE /api/keys/{id}
func (h *APIKeyHandler) DeleteAPIKey(c forge.Context) error {
	keyID := c.Param("id")
	if keyID == "" {
		return c.JSON(400, errs.BadRequest("Key ID is required"))
	}

	userID := c.Request().URL.Query().Get("user_id")
	orgID := c.Request().URL.Query().Get("org_id")

	if userID == "" || orgID == "" {
		return c.JSON(400, errs.BadRequest("User ID and Organization ID are required"))
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

// VerifyAPIKey handles POST /api/keys/verify
func (h *APIKeyHandler) VerifyAPIKey(c forge.Context) error {
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
