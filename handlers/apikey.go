package handlers

import (
	"encoding/json"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/internal/interfaces"
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
		return c.JSON(500, errs.InternalServerError(err.Error(), err))
	}

	return c.JSON(201, key)
}

type ListAPIKeysRequest = apikey.ListAPIKeysRequest

// ListAPIKeys handles GET /api/keys
func (h *APIKeyHandler) ListAPIKeys(c forge.Context, req ListAPIKeysRequest) error {
	response, err := h.service.ListAPIKeys(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(500, errs.InternalServerError("Failed to list API keys", err))
	}

	return c.JSON(200, response)
}

type GetAPIKeyRequest struct {
	AppID  xid.ID  `json:"appId" validate:"required" query:"appId"`
	UserID xid.ID  `json:"userId" validate:"required" query:"userId"`
	OrgID  *xid.ID `json:"orgId" query:"orgId"`
}

// GetAPIKey handles GET /api/keys/{id}
func (h *APIKeyHandler) GetAPIKey(c forge.Context, req GetAPIKeyRequest) error {
	keyID := utils.GetXIDParams(c, "id")
	if keyID.IsNil() {
		return c.JSON(400, errs.BadRequest("Key ID is required"))
	}

	key, err := h.service.GetAPIKey(c.Request().Context(), keyID, req.UserID, req.AppID, req.OrgID)
	if err != nil {
		return c.JSON(404, errs.NotFound("API key not found"))
	}

	return c.JSON(200, key)
}

type UpdateAPIKeyRequest struct {
	ID    xid.ID                     `json:"id" validate:"required" path:"id"`
	AppID xid.ID                     `json:"appId" validate:"required" query:"appId"`
	OrgID *xid.ID                    `json:"orgId" query:"orgId"`
	Body  apikey.UpdateAPIKeyRequest `json:"body" validate:"required"`
}

// UpdateAPIKey handles PUT /api/keys/{id}
func (h *APIKeyHandler) UpdateAPIKey(c forge.Context, req UpdateAPIKeyRequest) error {
	userID := interfaces.GetUserID(c.Request().Context())
	if userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	key, err := h.service.UpdateAPIKey(c.Request().Context(), req.ID, userID, req.AppID, req.OrgID, &req.Body)
	if err != nil {
		return c.JSON(500, errs.InternalServerError("Failed to update API key", err))
	}

	return c.JSON(200, key)
}

type DeleteAPIKeyRequest struct {
	ID    xid.ID  `json:"id" validate:"required" path:"id"`
	AppID xid.ID  `json:"appId" validate:"required" query:"appId"`
	OrgID *xid.ID `json:"orgId" query:"orgId"`
}

// DeleteAPIKey handles DELETE /api/keys/{id}
func (h *APIKeyHandler) DeleteAPIKey(c forge.Context, req DeleteAPIKeyRequest) error {
	userID := interfaces.GetUserID(c.Request().Context())
	if userID.IsNil() {
		return c.JSON(401, errs.Unauthorized())
	}

	err := h.service.DeleteAPIKey(c.Request().Context(), req.ID, userID, req.AppID, req.OrgID)
	if err != nil {
		return c.JSON(500, errs.InternalServerError("Failed to delete API key", err))
	}

	return c.JSON(200, map[string]string{
		"message": "API key deleted successfully",
	})
}

// VerifyAPIKey handles POST /api/keys/verify
func (h *APIKeyHandler) VerifyAPIKey(c forge.Context) error {
	var req apikey.VerifyAPIKeyRequest

	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(400, errs.BadRequest("Invalid request body"))
	}

	if req.Key == "" {
		return c.JSON(400, errs.BadRequest("API key is required"))
	}

	// Set IP and User Agent from request
	req.IP = c.Request().RemoteAddr
	req.UserAgent = c.Request().Header.Get("User-Agent")

	response, err := h.service.VerifyAPIKey(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(500, errs.InternalServerError("Failed to verify API key", err))
	}

	if !response.Valid {
		return c.JSON(401, errs.UnauthorizedWithMessage(response.Error))
	}

	return c.JSON(200, response)
}
