package jwt

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/errs"
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

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// CreateJWTKey creates a new JWT signing key
func (h *Handler) CreateJWTKey(c forge.Context) error {
	var req jwt.CreateJWTKeyRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID is required", http.StatusBadRequest))
	}
	req.AppID = appID

	key, err := h.service.CreateJWTKey(c.Request().Context(), &req)
	if err != nil {
		return handleError(c, err, "CREATE_JWT_KEY_FAILED", "Failed to create JWT key", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, key)
}

// ListJWTKeys lists JWT signing keys
func (h *Handler) ListJWTKeys(c forge.Context) error {
	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID is required", http.StatusBadRequest))
	}

	// Parse query parameters
	query := c.Request().URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	limit, _ := strconv.Atoi(query.Get("limit"))

	// Parse active filter
	var active *bool
	if activeStr := query.Get("active"); activeStr != "" {
		if activeBool, err := strconv.ParseBool(activeStr); err == nil {
			active = &activeBool
		}
	}

	// Parse platform key filter
	var isPlatformKey *bool
	if platformStr := query.Get("is_platform_key"); platformStr != "" {
		if platformBool, err := strconv.ParseBool(platformStr); err == nil {
			isPlatformKey = &platformBool
		}
	}

	// Set defaults
	if page == 0 {
		page = 1
	}
	if limit == 0 {
		limit = 20
	}

	// Create filter
	filter := &jwt.ListJWTKeysFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  page,
			Limit: limit,
		},
		AppID:         appID,
		IsPlatformKey: isPlatformKey,
		Active:        active,
	}

	// Get paginated results
	pageResp, err := h.service.ListJWTKeys(c.Request().Context(), filter)
	if err != nil {
		return handleError(c, err, "LIST_JWT_KEYS_FAILED", "Failed to list JWT keys", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, pageResp)
}

// GetJWKS returns the JSON Web Key Set
func (h *Handler) GetJWKS(c forge.Context) error {
	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID is required", http.StatusBadRequest))
	}

	jwks, err := h.service.GetJWKS(c.Request().Context(), appID)
	if err != nil {
		return handleError(c, err, "GET_JWKS_FAILED", "Failed to get JWKS", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, jwks)
}

// GenerateToken generates a new JWT token
func (h *Handler) GenerateToken(c forge.Context) error {
	var req jwt.GenerateTokenRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID is required", http.StatusBadRequest))
	}
	req.AppID = appID

	response, err := h.service.GenerateToken(c.Request().Context(), &req)
	if err != nil {
		return handleError(c, err, "GENERATE_TOKEN_FAILED", "Failed to generate token", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, response)
}

// VerifyToken verifies a JWT token
func (h *Handler) VerifyToken(c forge.Context) error {
	var req jwt.VerifyTokenRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID is required", http.StatusBadRequest))
	}
	req.AppID = appID

	result, err := h.service.VerifyToken(c.Request().Context(), &req)
	if err != nil {
		return handleError(c, err, "VERIFY_TOKEN_FAILED", "Failed to verify token", http.StatusUnauthorized)
	}

	return c.JSON(http.StatusOK, result)
}
