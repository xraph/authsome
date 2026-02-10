package jwt

import (
	"errors"
	"net/http"
	"time"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles JWT-related HTTP requests.
type Handler struct {
	service *jwt.Service
}

// CreateJWTKeyRequest represents request types.
type CreateJWTKeyRequest struct {
	IsPlatformKey bool           `json:"isPlatformKey"`
	Algorithm     string         `json:"algorithm"     validate:"required"`
	KeyType       string         `json:"keyType"       validate:"required"`
	Curve         string         `json:"curve"`
	ExpiresAt     *time.Time     `json:"expiresAt"`
	Metadata      map[string]any `json:"metadata"`
}

type ListJWTKeysRequest struct {
	Page          int   `query:"page"`
	Limit         int   `query:"limit"`
	Active        *bool `query:"active"`
	IsPlatformKey *bool `query:"is_platform_key"`
}

type GenerateTokenRequest struct {
	UserID      string         `json:"userId"      validate:"required"`
	SessionID   string         `json:"sessionId"`
	TokenType   string         `json:"tokenType"   validate:"required"`
	Scopes      []string       `json:"scopes"`
	Permissions []string       `json:"permissions"`
	Audience    []string       `json:"audience"`
	ExpiresIn   time.Duration  `json:"expiresIn"`
	Metadata    map[string]any `json:"metadata"`
}

type VerifyTokenRequest struct {
	Token     string   `json:"token"     validate:"required"`
	Audience  []string `json:"audience"`
	TokenType string   `json:"tokenType"`
}

// NewHandler creates a new JWT handler.
func NewHandler(service *jwt.Service) *Handler {
	return &Handler{
		service: service,
	}
}

// handleError returns the error in a structured format.
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	authErr := &errs.AuthsomeError{}
	if errors.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// CreateJWTKey creates a new JWT signing key.
func (h *Handler) CreateJWTKey(c forge.Context) error {
	var req CreateJWTKeyRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID is required", http.StatusBadRequest))
	}

	serviceReq := &jwt.CreateJWTKeyRequest{
		AppID:         appID,
		IsPlatformKey: req.IsPlatformKey,
		Algorithm:     req.Algorithm,
		KeyType:       req.KeyType,
		Curve:         req.Curve,
		ExpiresAt:     req.ExpiresAt,
		Metadata:      req.Metadata,
	}

	key, err := h.service.CreateJWTKey(c.Request().Context(), serviceReq)
	if err != nil {
		return handleError(c, err, "CREATE_JWT_KEY_FAILED", "Failed to create JWT key", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, key)
}

// ListJWTKeys lists JWT signing keys.
func (h *Handler) ListJWTKeys(c forge.Context) error {
	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID is required", http.StatusBadRequest))
	}

	var req ListJWTKeysRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Set defaults
	if req.Page == 0 {
		req.Page = 1
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	// Create filter
	filter := &jwt.ListJWTKeysFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  req.Page,
			Limit: req.Limit,
		},
		AppID:         appID,
		IsPlatformKey: req.IsPlatformKey,
		Active:        req.Active,
	}

	// Get paginated results
	pageResp, err := h.service.ListJWTKeys(c.Request().Context(), filter)
	if err != nil {
		return handleError(c, err, "LIST_JWT_KEYS_FAILED", "Failed to list JWT keys", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, pageResp)
}

// GetJWKS returns the JSON Web Key Set.
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

// GenerateToken generates a new JWT token.
func (h *Handler) GenerateToken(c forge.Context) error {
	var req GenerateTokenRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID is required", http.StatusBadRequest))
	}

	serviceReq := &jwt.GenerateTokenRequest{
		AppID:       appID,
		UserID:      req.UserID,
		SessionID:   req.SessionID,
		TokenType:   req.TokenType,
		Scopes:      req.Scopes,
		Permissions: req.Permissions,
		Audience:    req.Audience,
		ExpiresIn:   req.ExpiresIn,
		Metadata:    req.Metadata,
	}

	response, err := h.service.GenerateToken(c.Request().Context(), serviceReq)
	if err != nil {
		return handleError(c, err, "GENERATE_TOKEN_FAILED", "Failed to generate token", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, response)
}

// VerifyToken verifies a JWT token.
func (h *Handler) VerifyToken(c forge.Context) error {
	var req VerifyTokenRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Get app ID from context
	appID, ok := contexts.GetAppID(c.Request().Context())
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_ID", "App ID is required", http.StatusBadRequest))
	}

	serviceReq := &jwt.VerifyTokenRequest{
		AppID:     appID,
		Token:     req.Token,
		Audience:  req.Audience,
		TokenType: req.TokenType,
	}

	result, err := h.service.VerifyToken(c.Request().Context(), serviceReq)
	if err != nil {
		return handleError(c, err, "VERIFY_TOKEN_FAILED", "Failed to verify token", http.StatusUnauthorized)
	}

	return c.JSON(http.StatusOK, result)
}
