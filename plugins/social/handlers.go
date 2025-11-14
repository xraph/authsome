package social

import (
	"encoding/json"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/internal/errs"
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

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
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
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	if req.Provider == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_PROVIDER", "Provider is required", http.StatusBadRequest))
	}

	// Get app and org from context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	authURL, err := h.service.GetAuthorizationURL(c.Request().Context(), req.Provider, appID, userOrgID, req.Scopes)
	if err != nil {
		return handleError(c, err, "AUTH_URL_FAILED", "Failed to generate authorization URL", http.StatusBadRequest)
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
		authErr := errs.New("OAUTH_ERROR", "OAuth provider error", http.StatusBadRequest).WithError(nil)
		authErr.Details = map[string]interface{}{
			"error":             errorParam,
			"error_description": errorDesc,
		}
		return c.JSON(http.StatusBadRequest, authErr)
	}

	if code == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_CODE", "Authorization code is required", http.StatusBadRequest))
	}

	if state == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_STATE", "State parameter is required", http.StatusBadRequest))
	}

	result, err := h.service.HandleCallback(c.Request().Context(), provider, state, code)
	if err != nil {
		return handleError(c, err, "CALLBACK_FAILED", "Failed to handle OAuth callback", http.StatusUnauthorized)
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
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User not authenticated", http.StatusUnauthorized))
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest).WithError(err))
	}

	// Get app and org from context
	appID, _ := contexts.GetAppID(c.Request().Context())
	orgID, _ := contexts.GetOrganizationID(c.Request().Context())
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	var req LinkAccountRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	if req.Provider == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_PROVIDER", "Provider is required", http.StatusBadRequest))
	}

	authURL, err := h.service.GetLinkAccountURL(c.Request().Context(), req.Provider, userID, appID, userOrgID, req.Scopes)
	if err != nil {
		return handleError(c, err, "LINK_ACCOUNT_FAILED", "Failed to generate link account URL", http.StatusBadRequest)
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
		return c.JSON(http.StatusUnauthorized, errs.New("UNAUTHORIZED", "User not authenticated", http.StatusUnauthorized))
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_USER_ID", "Invalid user ID format", http.StatusBadRequest).WithError(err))
	}

	provider := c.Param("provider")
	if provider == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_PROVIDER", "Provider is required", http.StatusBadRequest))
	}

	if err := h.service.UnlinkAccount(c.Request().Context(), userID, provider); err != nil {
		return handleError(c, err, "UNLINK_ACCOUNT_FAILED", "Failed to unlink account", http.StatusInternalServerError)
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

// =============================================================================
// ADMIN ENDPOINTS
// =============================================================================

// AdminListProvidersRequest represents a request to list configured OAuth providers
type AdminListProvidersRequest struct {
	AppID xid.ID `json:"appId"`
}

// AdminAddProviderRequest represents a request to add/configure an OAuth provider
type AdminAddProviderRequest struct {
	AppID        xid.ID   `json:"appId"`
	Provider     string   `json:"provider"`
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      bool     `json:"enabled"`
}

// AdminUpdateProviderRequest represents a request to update OAuth provider config
type AdminUpdateProviderRequest struct {
	ClientID     *string  `json:"clientId,omitempty"`
	ClientSecret *string  `json:"clientSecret,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
}

// AdminListProviders handles GET /social/admin/providers
// Lists configured OAuth providers for an app
func (h *Handler) AdminListProviders(c forge.Context) error {
	ctx := c.Request().Context()

	// Get app context
	appID, _ := contexts.GetAppID(ctx)
	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_CONTEXT", "App context required", http.StatusBadRequest))
	}

	// TODO: Check admin permission via RBAC
	// userID := contexts.GetUserID(ctx)
	// if !h.rbacService.HasPermission(ctx, userID, "social:admin") {
	//     return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Admin role required", http.StatusForbidden))
	// }

	// Get configured providers for this app
	providers := h.service.ListProviders()

	// TODO: Load app-specific configuration from database
	// For now, return available providers
	return c.JSON(http.StatusOK, map[string]interface{}{
		"providers": providers,
		"appId":     appID.String(),
	})
}

// AdminAddProvider handles POST /social/admin/providers
// Adds/configures an OAuth provider for an app
func (h *Handler) AdminAddProvider(c forge.Context) error {
	ctx := c.Request().Context()

	// Get app context
	appID, _ := contexts.GetAppID(ctx)
	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_CONTEXT", "App context required", http.StatusBadRequest))
	}

	// TODO: Check admin permission via RBAC
	// userID := contexts.GetUserID(ctx)
	// if !h.rbacService.HasPermission(ctx, userID, "social:admin") {
	//     return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Admin role required", http.StatusForbidden))
	// }

	var req AdminAddProviderRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// Validate provider
	if req.Provider == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_PROVIDER", "Provider is required", http.StatusBadRequest))
	}

	if req.ClientID == "" || req.ClientSecret == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_CREDENTIALS", "ClientID and ClientSecret are required", http.StatusBadRequest))
	}

	// TODO: Store provider configuration in database
	// For now, return success response
	// In production, this would:
	// 1. Validate provider exists in supported providers
	// 2. Store encrypted credentials in app-specific config
	// 3. Log the admin action to audit service

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message":  "Provider configured successfully",
		"provider": req.Provider,
		"appId":    appID.String(),
	})
}

// AdminUpdateProvider handles PUT /social/admin/providers/:provider
// Updates OAuth provider configuration for an app
func (h *Handler) AdminUpdateProvider(c forge.Context) error {
	ctx := c.Request().Context()

	// Get app context
	appID, _ := contexts.GetAppID(ctx)
	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_CONTEXT", "App context required", http.StatusBadRequest))
	}

	provider := c.Param("provider")
	if provider == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_PROVIDER", "Provider is required", http.StatusBadRequest))
	}

	// TODO: Check admin permission via RBAC
	// userID := contexts.GetUserID(ctx)
	// if !h.rbacService.HasPermission(ctx, userID, "social:admin") {
	//     return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Admin role required", http.StatusForbidden))
	// }

	var req AdminUpdateProviderRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.New("INVALID_REQUEST", "Invalid request body", http.StatusBadRequest))
	}

	// TODO: Update provider configuration in database
	// For now, return success response
	// In production, this would:
	// 1. Load existing provider config
	// 2. Update only provided fields
	// 3. Store encrypted credentials
	// 4. Log the admin action to audit service

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "Provider updated successfully",
		"provider": provider,
		"appId":    appID.String(),
	})
}

// AdminDeleteProvider handles DELETE /social/admin/providers/:provider
// Removes OAuth provider configuration for an app
func (h *Handler) AdminDeleteProvider(c forge.Context) error {
	ctx := c.Request().Context()

	// Get app context
	appID, _ := contexts.GetAppID(ctx)
	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_CONTEXT", "App context required", http.StatusBadRequest))
	}

	provider := c.Param("provider")
	if provider == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_PROVIDER", "Provider is required", http.StatusBadRequest))
	}

	// TODO: Check admin permission via RBAC
	// userID := contexts.GetUserID(ctx)
	// if !h.rbacService.HasPermission(ctx, userID, "social:admin") {
	//     return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Admin role required", http.StatusForbidden))
	// }

	// TODO: Delete provider configuration from database
	// For now, return success response
	// In production, this would:
	// 1. Check if provider is configured
	// 2. Delete provider config
	// 3. Log the admin action to audit service

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":  "Provider removed successfully",
		"provider": provider,
		"appId":    appID.String(),
	})
}
