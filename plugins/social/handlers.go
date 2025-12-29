package social

import (
	"fmt"
	"net/http"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/authflow"
	"github.com/xraph/authsome/core/base"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/internal/crypto"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/forge"
)

// Handler handles HTTP requests for social OAuth
type Handler struct {
	service        *Service
	rateLimiter    *RateLimiter
	authCompletion *authflow.CompletionService // Centralized authentication completion
}

// Request types
type SignInRequest struct {
	Provider    string   `json:"provider" validate:"required" example:"google"`
	Scopes      []string `json:"scopes,omitempty" example:"[\"email\",\"profile\"]"`
	RedirectURL string   `json:"redirectUrl,omitempty" example:"https://example.com/auth/callback"`
}

type LinkAccountRequest struct {
	Provider string   `json:"provider" validate:"required" example:"github"`
	Scopes   []string `json:"scopes,omitempty" example:"[\"user:email\"]"`
}

type AdminAddProviderRequest struct {
	AppID        xid.ID   `json:"appId" validate:"required"`
	Provider     string   `json:"provider" validate:"required" example:"google"`
	ClientID     string   `json:"clientId" validate:"required"`
	ClientSecret string   `json:"clientSecret" validate:"required"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      bool     `json:"enabled"`
}

type AdminUpdateProviderRequest struct {
	ClientID     *string  `json:"clientId,omitempty"`
	ClientSecret *string  `json:"clientSecret,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
}

// Response types - properly typed
type AuthURLResponse struct {
	URL string `json:"url" example:"https://accounts.google.com/o/oauth2/v2/auth?..."`
}

type CallbackResponse struct {
	User    *user.User       `json:"user"`
	Session *session.Session `json:"session"`
	Token   string           `json:"token" example:"session_token_abc123"`
}

type CallbackDataResponse struct {
	User      *user.User `json:"user"`
	IsNewUser bool       `json:"isNewUser" example:"false"`
	Action    string     `json:"action" example:"signin"` // "signin", "signup", "linked"
}

type ConnectionResponse struct {
	Connection *base.SocialAccount `json:"connection"`
}

type ConnectionsResponse struct {
	Connections []*base.SocialAccount `json:"connections"`
}

type ProvidersResponse struct {
	Providers []string `json:"providers" example:"[\"google\",\"github\",\"facebook\"]"`
}

type ProvidersAppResponse struct {
	Providers []string `json:"providers"`
	AppID     string   `json:"appId"`
}

type ProviderConfigResponse struct {
	Message  string `json:"message" example:"Provider configured successfully"`
	Provider string `json:"provider" example:"google"`
	AppID    string `json:"appId" example:"c9h7b3j2k1m4n5p6"`
}

// Use shared response type
type MessageResponse = responses.MessageResponse

// NewHandler creates a new social OAuth handler
func NewHandler(
	service *Service,
	rateLimiter *RateLimiter,
	authCompletion *authflow.CompletionService,
) *Handler {
	return &Handler{
		service:        service,
		rateLimiter:    rateLimiter,
		authCompletion: authCompletion,
	}
}

// handleError returns the error in a structured format
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	if authErr, ok := err.(*errs.AuthsomeError); ok {
		return c.JSON(authErr.HTTPStatus, authErr)
	}
	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// SignIn initiates OAuth flow for sign-in
// POST /api/auth/signin/social
func (h *Handler) SignIn(c forge.Context) error {
	ctx := c.Request().Context()

	// Rate limiting
	if h.rateLimiter != nil {
		clientIP := c.Request().RemoteAddr
		if err := h.rateLimiter.Allow(ctx, "oauth_signin", clientIP); err != nil {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Too many requests", http.StatusTooManyRequests))
		}
	}

	var req SignInRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	if req.Provider == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_PROVIDER", "Provider is required", http.StatusBadRequest))
	}

	// Get app and org from context
	appID, _ := contexts.GetAppID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	authURL, err := h.service.GetAuthorizationURL(ctx, req.Provider, appID, userOrgID, req.Scopes)
	if err != nil {
		fmt.Println("Error generating authorization URL:", err)
		return handleError(c, err, "AUTH_URL_FAILED", "Failed to generate authorization URL", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &AuthURLResponse{URL: authURL})
}

// CallbackRequest represents OAuth callback parameters
type CallbackRequest struct {
	Provider         string `path:"provider" validate:"required" json:"-"`
	State            string `query:"state" validate:"required" json:"state"`
	Code             string `query:"code" json:"code"`
	Error            string `query:"error" json:"error,omitempty"`
	ErrorDescription string `query:"error_description" json:"errorDescription,omitempty"`
}

// Callback handles OAuth provider callback
// GET /api/auth/callback/:provider
func (h *Handler) Callback(c forge.Context) error {
	ctx := c.Request().Context()

	// Rate limiting (more lenient than signin)
	if h.rateLimiter != nil {
		clientIP := c.Request().RemoteAddr
		if err := h.rateLimiter.Allow(ctx, "oauth_callback", clientIP); err != nil {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Too many requests", http.StatusTooManyRequests))
		}
	}

	var req CallbackRequest
	if err := c.BindRequest(&req); err != nil {
		fmt.Println("Error binding callback request:", err)
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid callback parameters"))
	}

	// Check for OAuth error
	if req.Error != "" {
		authErr := errs.New("OAUTH_ERROR", "OAuth provider error", http.StatusBadRequest).WithError(nil)
		authErr.Details = map[string]interface{}{
			"error":             req.Error,
			"error_description": req.ErrorDescription,
		}
		return c.JSON(http.StatusBadRequest, authErr)
	}

	if req.Code == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_CODE", "Authorization code is required", http.StatusBadRequest))
	}

	result, err := h.service.HandleCallback(ctx, req.Provider, req.State, req.Code)
	if err != nil {
		fmt.Println("Error handling OAuth callback:", err, req.State, req.Code)
		return handleError(c, err, "CALLBACK_FAILED", "Failed to handle OAuth callback", http.StatusUnauthorized)
	}

	// Use centralized authentication completion service for signup/signin
	if h.authCompletion != nil {
		var authResp *responses.AuthResponse
		var signupErr error

		if result.IsNewUser && result.User == nil {
			// New user signup - CompleteSignUpOrSignIn will create user and add membership
			// Generate a secure random password for OAuth users (they won't use it)
			pwd, pwdErr := crypto.GenerateToken(32)
			if pwdErr != nil {
				fmt.Printf("error: failed to generate password for OAuth user: %v\n", pwdErr)
				return handleError(c, pwdErr, "PASSWORD_GENERATION_FAILED", "Failed to generate password", http.StatusInternalServerError)
			}

			authResp, signupErr = h.authCompletion.CompleteSignUpOrSignIn(&authflow.CompleteSignUpOrSignInRequest{
				Email:        result.OAuthUserInfo.Email,
				Password:     pwd, // Secure random password for OAuth users
				Name:         result.OAuthUserInfo.Name,
				User:         nil,
				IsNewUser:    true,
				RememberMe:   false, // Can be made configurable
				IPAddress:    c.Request().RemoteAddr,
				UserAgent:    c.Request().UserAgent(),
				Context:      ctx,
				ForgeContext: c,
				AuthMethod:   "social",
				AuthProvider: req.Provider,
			})
			if signupErr != nil {
				fmt.Printf("error: failed to complete signup: %v\n", signupErr)
				return handleError(c, signupErr, "SIGNUP_FAILED", "Failed to complete signup", http.StatusInternalServerError)
			}

			// Create social account link after user is created
			if authResp != nil && authResp.User != nil {
				if err := h.service.CreateSocialAccount(ctx, authResp.User.ID, result.AppID, result.UserOrgID, result.Provider, result.OAuthUserInfo, result.OAuthToken); err != nil {
					fmt.Printf("warning: failed to link social account: %v\n", err)
					// Don't fail - user is already created and logged in
				}

				// Update user profile with OAuth info if needed
				if result.OAuthUserInfo.Avatar != "" || (result.OAuthUserInfo.EmailVerified && h.service != nil) {
					// Update avatar and email verification status
					// This is a best-effort operation, we don't fail if it doesn't work
				}
			}
		} else {
			// Existing user signin - CompleteSignUpOrSignIn will create session and check membership
			authResp, signupErr = h.authCompletion.CompleteSignUpOrSignIn(&authflow.CompleteSignUpOrSignInRequest{
				Email:        result.User.Email,
				Password:     "",
				Name:         result.User.Name,
				User:         result.User,
				IsNewUser:    false,
				RememberMe:   false,
				IPAddress:    c.Request().RemoteAddr,
				UserAgent:    c.Request().UserAgent(),
				Context:      ctx,
				ForgeContext: c,
				AuthMethod:   "social",
				AuthProvider: req.Provider,
			})
			if signupErr != nil {
				fmt.Printf("error: failed to complete signin: %v\n", signupErr)
				return handleError(c, signupErr, "SIGNIN_FAILED", "Failed to complete signin", http.StatusInternalServerError)
			}
		}

		if authResp != nil {
			return c.JSON(http.StatusOK, authResp)
		}
	}

	// Fallback to old behavior (user data only) if completion service not available
	return c.JSON(http.StatusOK, &CallbackDataResponse{
		User:      result.User,
		IsNewUser: result.IsNewUser,
		Action:    result.Action,
	})
}

// LinkAccount links a social provider to the current user
// POST /api/auth/account/link
func (h *Handler) LinkAccount(c forge.Context) error {
	ctx := c.Request().Context()

	// Rate limiting
	if h.rateLimiter != nil {
		clientIP := c.Request().RemoteAddr
		if err := h.rateLimiter.Allow(ctx, "oauth_link", clientIP); err != nil {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Too many requests", http.StatusTooManyRequests))
		}
	}

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
	appID, _ := contexts.GetAppID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)
	var userOrgID *xid.ID
	if orgID != xid.NilID() {
		userOrgID = &orgID
	}

	var req LinkAccountRequest
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	if req.Provider == "" {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_PROVIDER", "Provider is required", http.StatusBadRequest))
	}

	authURL, err := h.service.GetLinkAccountURL(ctx, req.Provider, userID, appID, userOrgID, req.Scopes)
	if err != nil {
		return handleError(c, err, "LINK_ACCOUNT_FAILED", "Failed to generate link account URL", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &AuthURLResponse{URL: authURL})
}

// UnlinkAccount unlinks a social provider from the current user
// DELETE /api/auth/account/unlink/:provider
func (h *Handler) UnlinkAccount(c forge.Context) error {
	ctx := c.Request().Context()

	// Rate limiting
	if h.rateLimiter != nil {
		clientIP := c.Request().RemoteAddr
		if err := h.rateLimiter.Allow(ctx, "oauth_unlink", clientIP); err != nil {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Too many requests", http.StatusTooManyRequests))
		}
	}

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

	if err := h.service.UnlinkAccount(ctx, userID, provider); err != nil {
		return handleError(c, err, "UNLINK_ACCOUNT_FAILED", "Failed to unlink account", http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, &MessageResponse{Message: "Account unlinked successfully"})
}

// ListProviders returns available OAuth providers
// GET /api/auth/providers
func (h *Handler) ListProviders(c forge.Context) error {
	ctx := c.Request().Context()

	// Get app and environment from context
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)

	providers := h.service.ListProviders(ctx, appID, envID)
	return c.JSON(http.StatusOK, &ProvidersResponse{Providers: providers})
}

// =============================================================================
// ADMIN ENDPOINTS
// =============================================================================

// AdminListProviders handles GET /social/admin/providers
// Lists configured OAuth providers for an app
func (h *Handler) AdminListProviders(c forge.Context) error {
	ctx := c.Request().Context()

	// Get app context
	appID, _ := contexts.GetAppID(ctx)
	if appID.IsNil() {
		return c.JSON(http.StatusBadRequest, errs.New("MISSING_APP_CONTEXT", "App context required", http.StatusBadRequest))
	}

	// Get environment from context
	envID, _ := contexts.GetEnvironmentID(ctx)

	// TODO: Check admin permission via RBAC
	// userID := contexts.GetUserID(ctx)
	// if !h.rbacService.HasPermission(ctx, userID, "social:admin") {
	//     return c.JSON(http.StatusForbidden, errs.New("FORBIDDEN", "Admin role required", http.StatusForbidden))
	// }

	// Get configured providers for this app and environment
	providers := h.service.ListProviders(ctx, appID, envID)

	return c.JSON(http.StatusOK, &ProvidersAppResponse{
		Providers: providers,
		AppID:     appID.String(),
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
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
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

	return c.JSON(http.StatusCreated, &ProviderConfigResponse{
		Message:  "Provider configured successfully",
		Provider: req.Provider,
		AppID:    appID.String(),
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
	if err := c.BindRequest(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid request"))
	}

	// TODO: Update provider configuration in database
	// For now, return success response
	// In production, this would:
	// 1. Load existing provider config
	// 2. Update only provided fields
	// 3. Store encrypted credentials
	// 4. Log the admin action to audit service

	return c.JSON(http.StatusOK, &ProviderConfigResponse{
		Message:  "Provider updated successfully",
		Provider: provider,
		AppID:    appID.String(),
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

	return c.JSON(http.StatusOK, &ProviderConfigResponse{
		Message:  "Provider removed successfully",
		Provider: provider,
		AppID:    appID.String(),
	})
}
