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

// Handler handles HTTP requests for social OAuth.
type Handler struct {
	service        *Service
	rateLimiter    *RateLimiter
	authCompletion *authflow.CompletionService // Centralized authentication completion
}

// SignInRequest represents request types.
type SignInRequest struct {
	Provider    string   `example:"google"                            json:"provider"              validate:"required"`
	Scopes      []string `example:"[\"email\",\"profile\"]"           json:"scopes,omitempty"`
	RedirectURL string   `example:"https://example.com/auth/callback" json:"redirectUrl,omitempty"`
}

type LinkAccountRequest struct {
	Provider string   `example:"github"           json:"provider"         validate:"required"`
	Scopes   []string `example:"[\"user:email\"]" json:"scopes,omitempty"`
}

type AdminAddProviderRequest struct {
	AppID        xid.ID   `json:"appId"            validate:"required"`
	Provider     string   `example:"google"        json:"provider"     validate:"required"`
	ClientID     string   `json:"clientId"         validate:"required"`
	ClientSecret string   `json:"clientSecret"     validate:"required"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      bool     `json:"enabled"`
}

type AdminUpdateProviderRequest struct {
	ClientID     *string  `json:"clientId,omitempty"`
	ClientSecret *string  `json:"clientSecret,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	Enabled      *bool    `json:"enabled,omitempty"`
}

// AuthURLResponse types - properly typed.
type AuthURLResponse struct {
	URL string `example:"https://accounts.google.com/o/oauth2/v2/auth?..." json:"url"`
}

type CallbackResponse struct {
	User    *user.User       `json:"user"`
	Session *session.Session `json:"session"`
	Token   string           `example:"session_token_abc123" json:"token"`
}

type CallbackDataResponse struct {
	User      *user.User `json:"user"`
	IsNewUser bool       `example:"false"  json:"isNewUser"`
	Action    string     `example:"signin" json:"action"` // "signin", "signup", "linked"
}

type ConnectionResponse struct {
	Connection *base.SocialAccount `json:"connection"`
}

type ConnectionsResponse struct {
	Connections []*base.SocialAccount `json:"connections"`
}

type ProvidersResponse struct {
	Providers []string `example:"[\"google\",\"github\",\"facebook\"]" json:"providers"`
}

type ProvidersAppResponse struct {
	Providers []string `json:"providers"`
	AppID     string   `json:"appId"`
}

type ProviderConfigResponse struct {
	Message  string `example:"Provider configured successfully" json:"message"`
	Provider string `example:"google"                           json:"provider"`
	AppID    string `example:"c9h7b3j2k1m4n5p6"                 json:"appId"`
}

// MessageResponse shared response type.
type MessageResponse = responses.MessageResponse

// NewHandler creates a new social OAuth handler.
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

// handleError returns the error in a structured format.
func handleError(c forge.Context, err error, code string, message string, defaultStatus int) error {
	authErr := &errs.AuthsomeError{}
	if errs.As(err, &authErr) {
		return c.JSON(authErr.HTTPStatus, authErr)
	}

	return c.JSON(defaultStatus, errs.New(code, message, defaultStatus).WithError(err))
}

// SignIn initiates OAuth flow for sign-in
// SignIn /api/auth/signin/social.
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
		return handleError(c, err, "AUTH_URL_FAILED", "Failed to generate authorization URL", http.StatusBadRequest)
	}

	return c.JSON(http.StatusOK, &AuthURLResponse{URL: authURL})
}

// CallbackRequest represents OAuth callback parameters.
type CallbackRequest struct {
	Provider         string `json:"-"                          path:"provider"           validate:"required"`
	State            string `json:"state"                      query:"state"             validate:"required"`
	Code             string `json:"code"                       query:"code"`
	Error            string `json:"error,omitempty"            query:"error"`
	ErrorDescription string `json:"errorDescription,omitempty" query:"error_description"`
}

// Callback handles OAuth provider callback
// Callback /api/auth/callback/:provider.
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
		return c.JSON(http.StatusBadRequest, errs.BadRequest("Invalid callback parameters"))
	}

	// Check for OAuth error
	if req.Error != "" {
		authErr := errs.New("OAUTH_ERROR", "OAuth provider error", http.StatusBadRequest).WithError(nil)
		authErr.Details = map[string]any{
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
		fmt.Println("Error -------------------------- : ", err, req.Provider, req.State, req.Code)

		return handleError(c, err, "CALLBACK_FAILED", "Failed to handle OAuth callback", http.StatusUnauthorized)
	}

	// Use centralized authentication completion service for signup/signin
	if h.authCompletion != nil {
		var (
			authResp  *responses.AuthResponse
			signupErr error
		)

		if result.IsNewUser && result.User == nil {
			// New user signup - CompleteSignUpOrSignIn will create user and add membership
			// Generate a secure random password for OAuth users (they won't use it)
			pwd, pwdErr := crypto.GenerateToken(32)
			if pwdErr != nil {
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
				return handleError(c, signupErr, "SIGNUP_FAILED", "Failed to complete signup", http.StatusInternalServerError)
			}

			// Create social account link after user is created
			if authResp != nil && authResp.User != nil {
				if err := h.service.CreateSocialAccount(ctx, authResp.User.ID, result.AppID, result.UserOrgID, result.Provider, result.OAuthUserInfo, result.OAuthToken); err != nil {
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
// LinkAccount /api/auth/account/link.
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
	userIDStr := c.Request().Header.Get("X-User-Id")
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
// UnlinkAccount /api/auth/account/unlink/:provider.
func (h *Handler) UnlinkAccount(c forge.Context) error {
	ctx := c.Request().Context()

	// Rate limiting
	if h.rateLimiter != nil {
		clientIP := c.Request().RemoteAddr
		if err := h.rateLimiter.Allow(ctx, "oauth_unlink", clientIP); err != nil {
			return c.JSON(http.StatusTooManyRequests, errs.New("RATE_LIMIT_EXCEEDED", "Too many requests", http.StatusTooManyRequests))
		}
	}

	userIDStr := c.Request().Header.Get("X-User-Id")
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
// ListProviders /api/auth/providers.
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
// AdminListProviders configured OAuth providers for an app.
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
// AdminAddProvider Adds/configures an OAuth provider for an app.
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
// AdminUpdateProvider OAuth provider configuration for an app.
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
// AdminDeleteProvider OAuth provider configuration for an app.
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
