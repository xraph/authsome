package jwt

import (
	"encoding/json"
	"net/http"

	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/forge"
)

// Handler handles JWT-related HTTP requests
type Handler struct {
	service *jwt.Service
}

// NewHandler creates a new JWT handler
func NewHandler(service *jwt.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// CreateJWTKey creates a new JWT signing key
func (h *Handler) CreateJWTKey(c forge.Context) error {
	var req jwt.CreateJWTKeyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}

	// Get organization ID from header
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}
	req.OrgID = orgID

	key, err := h.service.CreateJWTKey(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, key)
}

// ListJWTKeys lists JWT signing keys
func (h *Handler) ListJWTKeys(c forge.Context) error {
	var req jwt.ListJWTKeysRequest

	// Get organization ID from header
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}
	req.OrgID = orgID

	// Set default pagination
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	keys, err := h.service.ListJWTKeys(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, keys)
}

// GetJWKS returns the JSON Web Key Set
func (h *Handler) GetJWKS(c forge.Context) error {
	// Get organization ID from header
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}

	jwks, err := h.service.GetJWKS(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, jwks)
}

// GenerateToken generates a new JWT token
func (h *Handler) GenerateToken(c forge.Context) error {
	var req jwt.GenerateTokenRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}

	// Get organization ID from header
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}
	req.OrgID = orgID

	response, err := h.service.GenerateToken(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, response)
}

// VerifyToken verifies a JWT token
func (h *Handler) VerifyToken(c forge.Context) error {
	var req jwt.VerifyTokenRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "Invalid request body",
		})
	}

	// Get organization ID from header
	orgID := c.Request().Header.Get("X-Organization-ID")
	if orgID == "" {
		orgID = "default"
	}
	req.OrgID = orgID

	result, err := h.service.VerifyToken(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}
