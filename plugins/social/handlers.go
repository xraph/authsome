package social

import (
	"encoding/json"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/internal/interfaces"
	"github.com/xraph/forge"
)

// Handler handles HTTP requests for social OAuth
type Handler struct {
	service *Service
}

// NewHandler creates a new social OAuth handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// SignInRequest represents a social sign-in request
type SignInRequest struct {
	Provider    string   `json:"provider"`
	Scopes      []string `json:"scopes,omitempty"`
	RedirectURL string   `json:"redirectUrl,omitempty"`
}

// LinkAccountRequest represents a request to link a social account
type LinkAccountRequest struct {
	Provider string   `json:"provider"`
	Scopes   []string `json:"scopes,omitempty"`
}

// SignIn initiates OAuth flow for sign-in
// POST /api/auth/signin/social
func (h *Handler) SignIn(c forge.Context) error {
	var req SignInRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request",
		})
	}

	if req.Provider == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Provider is required",
		})
	}

	// Get app and org from context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	authURL, err := h.service.GetAuthorizationURL(c.Request().Context(), req.Provider, appID, userOrgID, req.Scopes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"url": authURL,
	})
}

// Callback handles OAuth provider callback
// GET /api/auth/callback/:provider
func (h *Handler) Callback(c forge.Context) error {
	provider := c.Param("provider")
	query := c.Request().URL.Query()
	state := query.Get("state")
	code := query.Get("code")
	errorParam := query.Get("error")

	// Check for OAuth error
	if errorParam != "" {
		errorDesc := query.Get("error_description")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             errorParam,
			"error_description": errorDesc,
		})
	}

	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Authorization code is required",
		})
	}

	if state == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "State parameter is required",
		})
	}

	result, err := h.service.HandleCallback(c.Request().Context(), provider, state, code)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	// In production, create session and redirect to app
	// For now, return user data
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":      result.User,
		"isNewUser": result.IsNewUser,
		"action":    result.Action,
	})
}

// LinkAccount links a social provider to the current user
// POST /api/auth/account/link
func (h *Handler) LinkAccount(c forge.Context) error {
	// Get current user from session - in production, extract from JWT/session
	// For now, require user_id to be passed (or get from session cookie)
	userIDStr := c.Request().Header.Get("X-User-ID")
	if userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	// Get app and org from context
	appID := interfaces.GetAppID(c.Request().Context())
	orgID := interfaces.GetOrganizationID(c.Request().Context())
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	var req LinkAccountRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request",
		})
	}

	if req.Provider == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Provider is required",
		})
	}

	authURL, err := h.service.GetLinkAccountURL(c.Request().Context(), req.Provider, userID, appID, userOrgID, req.Scopes)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"url": authURL,
	})
}

// UnlinkAccount unlinks a social provider from the current user
// DELETE /api/auth/account/unlink/:provider
func (h *Handler) UnlinkAccount(c forge.Context) error {
	userIDStr := c.Request().Header.Get("X-User-ID")
	if userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	provider := c.Param("provider")
	if provider == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Provider is required",
		})
	}

	if err := h.service.UnlinkAccount(c.Request().Context(), userID, provider); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Account unlinked successfully",
	})
}

// ListProviders returns available OAuth providers
// GET /api/auth/providers
func (h *Handler) ListProviders(c forge.Context) error {
	providers := h.service.ListProviders()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"providers": providers,
	})
}
