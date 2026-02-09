package oidcprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/oidcprovider/consent"
	"github.com/xraph/forge"
	"maragu.dev/gomponents"
)

// Handler handles OIDC provider HTTP endpoints.
type Handler struct {
	svc              *Service
	revocationSvc    *RevokeTokenService
	introspectionSvc *IntrospectionService
	consentSvc       *ConsentService
	discoverySvc     *DiscoveryService
	clientAuth       *ClientAuthenticator
	basePath         string // Base path for building URLs (e.g., "/oauth2")
	loginURL         string // Custom login URL (defaults to "/auth/signin" if not set)
	apiMode          bool   // If true, return JSON instead of HTML redirects
}

// NewHandler creates a new OIDC handler.
func NewHandler(svc *Service, basePath, loginURL string, apiMode bool) *Handler {
	// Default to /auth/signin if not provided
	if loginURL == "" {
		loginURL = "/auth/signin"
	}

	return &Handler{
		svc:              svc,
		revocationSvc:    svc.revocation,
		introspectionSvc: svc.introspection,
		consentSvc:       svc.consent,
		discoverySvc:     svc.discovery,
		clientAuth:       svc.clientAuth,
		basePath:         basePath,
		loginURL:         loginURL,
		apiMode:          apiMode,
	}
}

// =============================================================================
// DISCOVERY ENDPOINT
// =============================================================================

// Discovery handles the OIDC discovery endpoint (.well-known/openid-configuration).
func (h *Handler) Discovery(c forge.Context) error {
	ctx := c.Request().Context()

	// Build base URL from request
	scheme := "https"
	if c.Request().TLS == nil {
		scheme = "http"
	}

	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request().Host)

	doc := h.discoverySvc.GetDiscoveryDocument(ctx, baseURL, h.basePath)

	return c.JSON(http.StatusOK, doc)
}

// JWKS returns the JSON Web Key Set.
func (h *Handler) JWKS(c forge.Context) error {
	jwks, err := h.svc.GetJWKS()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, jwks)
}

// =============================================================================
// AUTHORIZATION FLOW
// =============================================================================

// Authorize handles OAuth2/OIDC authorization requests.
func (h *Handler) Authorize(c forge.Context) error {
	ctx := c.Request().Context()
	q := c.Request().URL.Query()

	// Parse authorization request
	req := &AuthorizeRequest{
		ClientID:            q.Get("client_id"),
		RedirectURI:         q.Get("redirect_uri"),
		ResponseType:        q.Get("response_type"),
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		Nonce:               q.Get("nonce"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
	}

	// Validate the authorization request
	if err := h.svc.ValidateAuthorizeRequest(ctx, req); err != nil {
		return h.redirectWithError(c, req.RedirectURI, "invalid_request", err.Error(), req.State)
	}

	// Check if user is authenticated
	// Priority: Auth context session > Manual session token extraction
	var sess *session.Session

	// Try to get session from auth context first (set by middleware)
	if authCtx, ok := contexts.GetAuthContext(ctx); ok && authCtx.Session != nil {
		sess = authCtx.Session
	}

	// If no session in auth context, try to load from cookie/header
	// This handles cases where API key is present but session is not in context
	if sess == nil {
		sessionToken := h.getSessionToken(c)
		if sessionToken == "" {
			return h.handleAuthRequired(c, c.Request().URL.String())
		}

		// Validate user session
		var err error

		sess, err = h.svc.sessionSvc.FindByToken(ctx, sessionToken)
		if err != nil || sess == nil {
			return h.handleAuthRequired(c, c.Request().URL.String())
		}
	}

	// Check if consent is required
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	hasConsent, err := h.consentSvc.CheckConsent(ctx, sess.UserID, req.ClientID, h.consentSvc.ParseScopes(req.Scope), appID, envID, orgIDPtr)
	if err != nil {
		return h.redirectWithError(c, req.RedirectURI, "server_error", "Failed to check consent", req.State)
	}

	if !hasConsent {
		return h.showConsentScreen(c, req, sess)
	}

	// Generate and store authorization code
	authCode, err := h.svc.CreateAuthorizationCode(ctx, req, sess.UserID, sess.ID)
	if err != nil {
		return h.redirectWithError(c, req.RedirectURI, "server_error", "Failed to generate code", req.State)
	}

	// Redirect back with authorization code
	redirectURL := fmt.Sprintf("%s?code=%s", req.RedirectURI, authCode.Code)
	if req.State != "" {
		redirectURL += "&state=" + url.QueryEscape(req.State)
	}

	c.SetHeader("Location", redirectURL)

	return c.JSON(http.StatusFound, nil)
}

// HandleConsent processes the consent form submission.
func (h *Handler) HandleConsent(c forge.Context) error {
	ctx := c.Request().Context()

	// Parse form data
	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("failed to parse form data"))
	}

	// Parse consent request
	action := c.Request().FormValue("action")
	req := &ConsentRequest{
		Action:              action,
		ClientID:            c.Request().FormValue("client_id"),
		RedirectURI:         c.Request().FormValue("redirect_uri"),
		ResponseType:        c.Request().FormValue("response_type"),
		Scope:               c.Request().FormValue("scope"),
		State:               c.Request().FormValue("state"),
		CodeChallenge:       c.Request().FormValue("code_challenge"),
		CodeChallengeMethod: c.Request().FormValue("code_challenge_method"),
	}

	// Validate required parameters
	if req.ClientID == "" || req.RedirectURI == "" || req.ResponseType == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("missing required parameters"))
	}

	// Get current session
	sessionToken := h.getSessionToken(c)
	if sessionToken == "" {
		return h.redirectWithError(c, req.RedirectURI, "access_denied", "No active session", req.State)
	}

	sess, err := h.svc.sessionSvc.FindByToken(ctx, sessionToken)
	if err != nil || sess == nil {
		return h.redirectWithError(c, req.RedirectURI, "access_denied", "Invalid session", req.State)
	}

	// Handle user decision
	if action == "deny" {
		return h.redirectWithError(c, req.RedirectURI, "access_denied", "User denied the request", req.State)
	}

	if action != "allow" {
		return h.redirectWithError(c, req.RedirectURI, "invalid_request", "Invalid action", req.State)
	}

	// User allowed consent - store it
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)

	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	scopes := h.consentSvc.ParseScopes(req.Scope)
	if err := h.consentSvc.GrantConsent(ctx, sess.UserID, req.ClientID, scopes, appID, envID, orgIDPtr, nil); err != nil {
		// Log error but don't fail
	}

	// Create authorization request and proceed
	authReq := &AuthorizeRequest{
		ClientID:            req.ClientID,
		RedirectURI:         req.RedirectURI,
		ResponseType:        req.ResponseType,
		Scope:               req.Scope,
		State:               req.State,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
	}

	// Validate the authorization request
	if err := h.svc.ValidateAuthorizeRequest(ctx, authReq); err != nil {
		return h.redirectWithError(c, authReq.RedirectURI, "invalid_request", err.Error(), authReq.State)
	}

	// Create authorization code
	authCode, err := h.svc.CreateAuthorizationCode(ctx, authReq, sess.UserID, sess.ID)
	if err != nil {
		return h.redirectWithError(c, authReq.RedirectURI, "server_error", "Failed to create authorization code", authReq.State)
	}

	// Build redirect URL with authorization code
	redirectURL, err := url.Parse(authReq.RedirectURI)
	if err != nil {
		return h.redirectWithError(c, authReq.RedirectURI, "invalid_request", "Invalid redirect URI", authReq.State)
	}

	query := redirectURL.Query()
	query.Set("code", authCode.Code)

	if authReq.State != "" {
		query.Set("state", authReq.State)
	}

	redirectURL.RawQuery = query.Encode()

	c.SetHeader("Location", redirectURL.String())

	return c.JSON(http.StatusFound, nil)
}

// =============================================================================
// TOKEN ENDPOINT
// =============================================================================

// Token handles the token endpoint.
func (h *Handler) Token(c forge.Context) error {
	ctx := c.Request().Context()

	var req TokenRequest

	// Parse form data or JSON
	contentType := c.Request().Header.Get("Content-Type")
	if contentType == "application/x-www-form-urlencoded" || contentType == "" {
		if err := c.Request().ParseForm(); err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("failed to parse form data"))
		}

		req.GrantType = c.Request().FormValue("grant_type")
		req.Code = c.Request().FormValue("code")
		req.RedirectURI = c.Request().FormValue("redirect_uri")
		req.ClientID = c.Request().FormValue("client_id")
		req.ClientSecret = c.Request().FormValue("client_secret")
		req.CodeVerifier = c.Request().FormValue("code_verifier")
		req.RefreshToken = c.Request().FormValue("refresh_token")
		req.Scope = c.Request().FormValue("scope")
		req.DeviceCode = c.Request().FormValue("device_code")
	} else {
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid JSON"))
		}
	}

	// Route to appropriate grant type handler
	switch req.GrantType {
	case "authorization_code":
		return h.handleAuthorizationCodeGrant(ctx, c, &req)
	case "refresh_token":
		return h.handleRefreshTokenGrant(ctx, c, &req)
	case "client_credentials":
		return h.handleClientCredentialsGrant(ctx, c, &req)
	case "urn:ietf:params:oauth:grant-type:device_code":
		return h.handleDeviceCodeGrant(ctx, c, &req)
	default:
		return c.JSON(http.StatusBadRequest, errs.BadRequest("unsupported grant type: "+req.GrantType))
	}
}

// handleAuthorizationCodeGrant handles the authorization_code grant type.
func (h *Handler) handleAuthorizationCodeGrant(ctx context.Context, c forge.Context, req *TokenRequest) error {
	// Validate authorization code
	authCode, err := h.svc.ValidateAuthorizationCode(ctx, req.Code, req.ClientID, req.RedirectURI, req.CodeVerifier)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusBadRequest, errs.BadRequest(err.Error()))
	}

	// Mark code as used
	if err := h.svc.MarkCodeAsUsed(ctx, req.Code); err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	// Get user info for ID token
	user, err := h.svc.userSvc.FindByID(ctx, authCode.UserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.DatabaseError("find user", err))
	}

	userInfo := map[string]any{
		"sub":   user.ID.String(),
		"email": user.Email,
		"name":  user.Name,
	}

	// Exchange code for tokens
	tokenResponse, err := h.svc.ExchangeCodeForTokens(ctx, authCode, userInfo)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, tokenResponse)
}

// handleRefreshTokenGrant handles the refresh_token grant type.
func (h *Handler) handleRefreshTokenGrant(ctx context.Context, c forge.Context, req *TokenRequest) error {
	// Extract refresh token
	refreshToken := req.RefreshToken
	if refreshToken == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("refresh_token"))
	}

	// Authenticate client
	clientAuth, client, err := h.clientAuth.AuthenticateClient(ctx, c.Request())
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	_ = clientAuth // Authenticated

	// Refresh the token (with optional rotation)
	tokenResponse, err := h.svc.RefreshAccessToken(ctx, refreshToken, client.ClientID, req.Scope)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusUnauthorized, errs.UnauthorizedWithMessage("invalid or expired refresh token"))
	}

	return c.JSON(http.StatusOK, tokenResponse)
}

// handleClientCredentialsGrant handles the client_credentials grant type (M2M).
func (h *Handler) handleClientCredentialsGrant(ctx context.Context, c forge.Context, req *TokenRequest) error {
	// Authenticate client (confidential clients only)
	clientAuth, client, err := h.clientAuth.AuthenticateClient(ctx, c.Request())
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	// Client credentials grant is only for confidential clients
	if client.TokenEndpointAuthMethod == "none" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("public clients cannot use client_credentials grant"))
	}

	_ = clientAuth // Authenticated

	// Determine scope (use requested scope or client's default scope)
	scope := req.Scope
	if scope == "" {
		// Use client's allowed scope or default to basic M2M scope
		scope = "api:read api:write"
	}

	// Generate tokens for client (no user context)
	tokenResponse, err := h.svc.GenerateClientCredentialsToken(ctx, client, scope)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, tokenResponse)
}

// handleDeviceCodeGrant handles the device_code grant type (RFC 8628).
func (h *Handler) handleDeviceCodeGrant(ctx context.Context, c forge.Context, req *TokenRequest) error {
	// Check if device flow is enabled
	if h.svc.deviceFlowService == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "unsupported_grant_type",
			"error_description": "device flow is not enabled",
		})
	}

	// Validate required parameters
	if req.DeviceCode == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "invalid_request",
			"error_description": "device_code is required",
		})
	}

	if req.ClientID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "invalid_request",
			"error_description": "client_id is required",
		})
	}

	// Poll for device code status
	deviceCode, shouldSlowDown, err := h.svc.deviceFlowService.PollDeviceCode(ctx, req.DeviceCode)
	if err != nil {
		// Return appropriate OAuth error based on the error type
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			errorCode := "invalid_grant"
			switch authErr.HTTPStatus {
			case http.StatusForbidden:
				errorCode = "access_denied"
			case http.StatusBadRequest:
				if strings.Contains(authErr.Message, "expired") {
					errorCode = "expired_token"
				}
			}

			return c.JSON(http.StatusBadRequest, map[string]string{
				"error":             errorCode,
				"error_description": authErr.Message,
			})
		}

		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "invalid_grant",
			"error_description": "invalid device code",
		})
	}

	// Check if device is polling too frequently
	if shouldSlowDown {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "slow_down",
			"error_description": "polling too frequently, slow down by 5 seconds",
		})
	}

	// Check if authorization is still pending
	if deviceCode.Status == "pending" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "authorization_pending",
			"error_description": "user has not yet authorized the device",
		})
	}

	// Check if authorization was denied
	if deviceCode.Status == "denied" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "access_denied",
			"error_description": "user denied the authorization request",
		})
	}

	// Check if device code was already consumed
	if deviceCode.Status == "consumed" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "invalid_grant",
			"error_description": "device code has already been used",
		})
	}

	// Verify client_id matches
	if deviceCode.ClientID != req.ClientID {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "invalid_grant",
			"error_description": "client_id does not match",
		})
	}

	// Device code must be authorized
	if deviceCode.Status != "authorized" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "authorization_pending",
			"error_description": "device authorization not yet complete",
		})
	}

	// Validate client exists
	client, err := h.svc.clientRepo.FindByClientID(ctx, req.ClientID)
	if err != nil || client == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "invalid_client",
			"error_description": "client not found",
		})
	}

	// Get user and session
	if deviceCode.UserID == nil || deviceCode.SessionID == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error":             "server_error",
			"error_description": "device code missing user or session information",
		})
	}

	// Generate tokens
	tokenResponse, err := h.svc.GenerateTokensForDeviceCode(ctx, deviceCode, client)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, map[string]string{
				"error":             "server_error",
				"error_description": authErr.Message,
			})
		}

		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":             "server_error",
			"error_description": "failed to generate tokens",
		})
	}

	// Mark device code as consumed
	if err := h.svc.deviceFlowService.ConsumeDeviceCode(ctx, req.DeviceCode); err != nil {
		// Log error but don't fail the request since tokens were already generated
	}

	return c.JSON(http.StatusOK, tokenResponse)
}

// =============================================================================
// USERINFO ENDPOINT
// =============================================================================

// UserInfo returns user information based on the access token.
func (h *Handler) UserInfo(c forge.Context) error {
	ctx := c.Request().Context()

	// Extract access token from Authorization header
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return c.JSON(http.StatusUnauthorized, errs.UnauthorizedWithMessage("invalid authorization header"))
	}

	accessToken := authHeader[len(bearerPrefix):]
	if accessToken == "" {
		return c.JSON(http.StatusUnauthorized, errs.UnauthorizedWithMessage("access token required"))
	}

	// Get user information from the service
	userInfo, err := h.svc.GetUserInfoFromToken(ctx, accessToken)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusUnauthorized, errs.UnauthorizedWithMessage("invalid or expired access token"))
	}

	return c.JSON(http.StatusOK, userInfo)
}

// =============================================================================
// TOKEN INTROSPECTION (RFC 7662)
// =============================================================================

// IntrospectToken handles token introspection requests.
func (h *Handler) IntrospectToken(c forge.Context) error {
	ctx := c.Request().Context()

	// Authenticate client
	authResult, client, err := h.clientAuth.AuthenticateClient(ctx, c.Request())
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	// Validate client can introspect (confidential clients only)
	if err := h.clientAuth.ValidateClientForEndpoint(client, "introspect"); err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusForbidden, errs.PermissionDenied("introspect", "endpoint"))
	}

	// Parse introspection request
	var req TokenIntrospectionRequest

	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("failed to parse form data"))
	}

	req.Token = c.Request().FormValue("token")
	req.TokenTypeHint = c.Request().FormValue("token_type_hint")

	// Introspect token
	response, err := h.introspectionSvc.IntrospectToken(ctx, &req, authResult.ClientID)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, response)
}

// =============================================================================
// TOKEN REVOCATION (RFC 7009)
// =============================================================================

// RevokeToken handles token revocation requests.
func (h *Handler) RevokeToken(c forge.Context) error {
	ctx := c.Request().Context()

	// Authenticate client
	_, _, err := h.clientAuth.AuthenticateClient(ctx, c.Request())
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	// Parse revocation request
	var req TokenRevocationRequest

	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("failed to parse form data"))
	}

	req.Token = c.Request().FormValue("token")
	req.TokenTypeHint = c.Request().FormValue("token_type_hint")

	// Revoke token (returns success even if token doesn't exist per RFC 7009)
	if err := h.revocationSvc.RevokeToken(ctx, &req); err != nil {
		// Log error but return success per RFC 7009
	}

	return c.JSON(http.StatusOK, responses.StatusResponse{Status: "revoked"})
}

// =============================================================================
// DEVICE FLOW ENDPOINTS (RFC 8628)
// =============================================================================

// DeviceAuthorize initiates the device authorization flow.
func (h *Handler) DeviceAuthorize(c forge.Context) error {
	ctx := c.Request().Context()

	// Parse request
	var req DeviceAuthorizationRequest

	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("failed to parse form data"))
	}

	req.ClientID = c.Request().FormValue("client_id")
	req.Scope = c.Request().FormValue("scope")

	// Validate client_id
	if req.ClientID == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("client_id is required"))
	}

	// Validate client exists
	client, err := h.svc.clientRepo.FindByClientID(ctx, req.ClientID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid client"))
	}

	if client == nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("client not found"))
	}

	// Check if device flow is enabled
	if h.svc.deviceFlowService == nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("device flow not enabled"))
	}

	// Use client's context IDs (appID, envID, orgID) instead of request context
	// OAuth clients are scoped to an app and environment, so we use those values
	appID := client.AppID
	envID := client.EnvironmentID
	orgIDPtr := client.OrganizationID

	// Initiate device authorization
	deviceCode, err := h.svc.deviceFlowService.InitiateDeviceAuthorization(ctx, req.ClientID, req.Scope, appID, envID, orgIDPtr)
	if err != nil {
		authErr := &errs.AuthsomeError{}
		if errors.As(err, &authErr) {
			return c.JSON(authErr.HTTPStatus, authErr)
		}

		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	// Build verification URI
	scheme := "https"
	if c.Request().TLS == nil {
		scheme = "http"
	}

	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request().Host)
	// Prepend base path to the verification URI path
	verificationURI := baseURL + h.basePath + deviceCode.VerificationURI

	// Calculate expires_in
	expiresIn := int(time.Until(deviceCode.ExpiresAt).Seconds())

	// Build response with formatted user code for display
	formattedUserCode := deviceCode.FormattedUserCode()
	verificationURIComplete := fmt.Sprintf("%s?user_code=%s", verificationURI, formattedUserCode)

	resp := DeviceAuthorizationResponse{
		DeviceCode:              deviceCode.DeviceCode,
		UserCode:                formattedUserCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
		ExpiresIn:               expiresIn,
		Interval:                deviceCode.Interval,
	}

	return c.JSON(http.StatusOK, resp)
}

// DeviceCodeEntry shows the device code entry form.
func (h *Handler) DeviceCodeEntry(c forge.Context) error {
	// Check if device flow is enabled
	if h.svc.deviceFlowService == nil {
		return returnHTML(c, http.StatusNotFound, "<h1>404 Not Found</h1><p>Device flow is not enabled.</p>")
	}

	// Get optional pre-filled user code and redirect URL from query params
	userCode := c.Request().URL.Query().Get("user_code")
	redirectURL := c.Request().URL.Query().Get("redirect")

	// Use default branding (no client info at this point)
	branding := consent.DefaultBranding()

	page := consent.DeviceCodeEntryPage(consent.CodeEntryPageData{
		UserCode:    userCode,
		Branding:    branding,
		BasePath:    h.basePath,
		RedirectURL: redirectURL,
	})

	return renderNode(c, http.StatusOK, page)
}

// DeviceVerify verifies the user code and shows the consent screen.
func (h *Handler) DeviceVerify(c forge.Context) error {
	ctx := c.Request().Context()

	// Default branding for early errors
	branding := consent.DefaultBranding()

	// Check if device flow is enabled
	if h.svc.deviceFlowService == nil {
		if h.apiMode {
			return c.JSON(http.StatusNotFound, map[string]any{
				"error":             "not_found",
				"error_description": "Device flow is not enabled",
			})
		}

		return returnHTML(c, http.StatusNotFound, "<h1>404 Not Found</h1><p>Device flow is not enabled.</p>")
	}

	// Parse request body (support both JSON and form-encoded)
	var userCode, redirectURL string

	contentType := c.Request().Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		// Parse JSON body
		var body struct {
			UserCode    string `json:"user_code"`
			RedirectURL string `json:"redirect"`
		}
		if err := c.BindJSON(&body); err != nil {
			return h.handleDeviceError(c, http.StatusBadRequest, "invalid_request", "Failed to parse JSON body", "")
		}

		userCode = body.UserCode
		redirectURL = body.RedirectURL
	} else {
		// Parse form data
		if err := c.Request().ParseForm(); err != nil {
			return h.handleDeviceError(c, http.StatusBadRequest, "invalid_request", "Failed to parse form data", "")
		}

		userCode = c.Request().FormValue("user_code")
		redirectURL = c.Request().FormValue("redirect")
	}

	if userCode == "" {
		return h.handleDeviceError(c, http.StatusBadRequest, "invalid_request", "User code is required", redirectURL)
	}

	// Normalize user code (remove spaces, hyphens, uppercase)
	userCode = normalizeUserCode(userCode)

	// Get device code from service
	deviceCode, err := h.svc.deviceFlowService.GetDeviceCodeByUserCode(ctx, userCode)
	if err != nil {
		return h.handleDeviceError(c, http.StatusNotFound, "invalid_code", "Invalid or expired verification code", redirectURL)
	}

	// Check if device code is still pending
	if !deviceCode.IsPending() {
		errMsg := "This code has already been used or has expired"
		errorCode := "code_expired"

		if deviceCode.Status == "denied" {
			errMsg = "This authorization has been denied"
			errorCode = "access_denied"
		}

		return h.handleDeviceError(c, http.StatusBadRequest, errorCode, errMsg, redirectURL)
	}

	// Check if user is authenticated
	// Priority: Auth context session > Manual session token extraction
	var sess *session.Session

	// Try to get session from auth context first (set by middleware)
	if authCtx, ok := contexts.GetAuthContext(ctx); ok && authCtx.Session != nil {
		sess = authCtx.Session
	}

	// If no session in auth context, try to load from cookie/header
	// This handles cases where API key is present but session is not in context
	if sess == nil {
		sessionToken := h.getSessionToken(c)

		formattedCode := deviceCode.FormattedUserCode()
		if sessionToken == "" {
			// Build return URL (preserve redirect parameter if present)
			returnURL := fmt.Sprintf("%s/device/verify?user_code=%s", h.basePath, formattedCode)
			if redirectURL != "" {
				returnURL += "&redirect=" + url.QueryEscape(redirectURL)
			}

			return h.handleAuthRequired(c, returnURL)
		}

		// Validate user session
		var err error

		sess, err = h.svc.sessionSvc.FindByToken(ctx, sessionToken)
		if err != nil || sess == nil {
			returnURL := fmt.Sprintf("%s/device/verify?user_code=%s", h.basePath, formattedCode)
			if redirectURL != "" {
				returnURL += "&redirect=" + url.QueryEscape(redirectURL)
			}

			return h.handleAuthRequired(c, returnURL)
		}
	}

	// Get client information
	client, err := h.svc.clientRepo.FindByClientID(ctx, deviceCode.ClientID)
	if err != nil || client == nil {
		return h.handleDeviceError(c, http.StatusBadRequest, "invalid_client", "Invalid client", redirectURL)
	}

	// Extract branding from client
	branding = consent.ExtractBranding(client, nil)

	// Parse scopes for display
	scopes := h.consentSvc.GetScopeDescriptions(h.consentSvc.ParseScopes(deviceCode.Scope))

	componentScopes := make([]consent.ScopeInfo, len(scopes))
	for i, s := range scopes {
		componentScopes[i] = consent.ScopeInfo{
			Scope:       s.Name,
			Description: s.Description,
		}
	}

	// In API mode, return JSON with consent data
	if h.apiMode {
		return c.JSON(http.StatusOK, map[string]any{
			"requireConsent": true,
			"userCode":       deviceCode.UserCode,
			"clientName":     client.Name,
			"clientLogoUri":  client.LogoURI,
			"scopes":         componentScopes,
			"authorizeUrl":   h.basePath + "/device/authorize",
			"redirectUrl":    redirectURL,
		})
	}

	// Show verification/consent page (HTML mode)
	// Pass normalized code for form submission, formatted for display
	page := consent.DeviceVerificationPage(consent.VerificationPageData{
		UserCode:          deviceCode.UserCode,            // Normalized (for form field)
		UserCodeFormatted: deviceCode.FormattedUserCode(), // Formatted (for display)
		ClientName:        client.Name,
		LogoURI:           client.LogoURI,
		Scopes:            componentScopes,
		Branding:          branding,
		BasePath:          h.basePath,
		RedirectURL:       redirectURL, // Pass through redirect parameter
	})

	return renderNode(c, http.StatusOK, page)
}

// DeviceAuthorizeDecision handles the user's authorization decision.
func (h *Handler) DeviceAuthorizeDecision(c forge.Context) error {
	ctx := c.Request().Context()

	// Default branding for early errors
	branding := consent.DefaultBranding()

	// Check if device flow is enabled
	if h.svc.deviceFlowService == nil {
		if h.apiMode {
			return c.JSON(http.StatusNotFound, map[string]any{
				"error":             "not_found",
				"error_description": "Device flow is not enabled",
			})
		}

		return returnHTML(c, http.StatusNotFound, "<h1>404 Not Found</h1><p>Device flow is not enabled.</p>")
	}

	// Parse request body (support both JSON and form-encoded)
	var userCode, action, redirectURL string

	contentType := c.Request().Header.Get("Content-Type")

	if strings.Contains(contentType, "application/json") {
		// Parse JSON body
		var body struct {
			UserCode    string `json:"user_code"`
			Action      string `json:"action"`
			RedirectURL string `json:"redirect"`
		}
		if err := c.BindJSON(&body); err != nil {
			if h.apiMode {
				return c.JSON(http.StatusBadRequest, map[string]any{
					"error":             "invalid_request",
					"error_description": "Failed to parse JSON body",
				})
			}

			return returnHTML(c, http.StatusBadRequest, "<h1>400 Bad Request</h1><p>Failed to parse JSON body</p>")
		}

		userCode = body.UserCode
		action = body.Action
		redirectURL = body.RedirectURL
	} else {
		// Parse form data
		if err := c.Request().ParseForm(); err != nil {
			if h.apiMode {
				return c.JSON(http.StatusBadRequest, map[string]any{
					"error":             "invalid_request",
					"error_description": "Failed to parse form data",
				})
			}

			return returnHTML(c, http.StatusBadRequest, "<h1>400 Bad Request</h1><p>Failed to parse form data</p>")
		}

		userCode = c.Request().FormValue("user_code")
		action = c.Request().FormValue("action")
		redirectURL = c.Request().FormValue("redirect")
	}

	// Normalize user code
	userCode = normalizeUserCode(userCode)

	// Check if user is authenticated
	// Priority: Auth context session > Manual session token extraction
	var sess *session.Session

	// Try to get session from auth context first (set by middleware)
	if authCtx, ok := contexts.GetAuthContext(ctx); ok && authCtx.Session != nil {
		sess = authCtx.Session
	}

	// If no session in auth context, try to load from cookie/header
	// This handles cases where API key is present but session is not in context
	if sess == nil {
		sessionToken := h.getSessionToken(c)
		if sessionToken == "" {
			if h.apiMode {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"error":             "authentication_required",
					"error_description": "No active session",
				})
			}

			return returnHTML(c, http.StatusUnauthorized, "<h1>401 Unauthorized</h1><p>No active session</p>")
		}

		var err error

		sess, err = h.svc.sessionSvc.FindByToken(ctx, sessionToken)
		if err != nil || sess == nil {
			if h.apiMode {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"error":             "authentication_required",
					"error_description": "Invalid session",
				})
			}

			return returnHTML(c, http.StatusUnauthorized, "<h1>401 Unauthorized</h1><p>Invalid session</p>")
		}
	}

	// Get device code to extract client info for branding
	deviceCode, err := h.svc.deviceFlowService.GetDeviceCodeByUserCode(ctx, userCode)
	if err == nil && deviceCode != nil {
		// Get client information for branding
		client, err := h.svc.clientRepo.FindByClientID(ctx, deviceCode.ClientID)
		if err == nil && client != nil {
			branding = consent.ExtractBranding(client, nil)
		}
	}

	// Handle user decision
	switch action {
	case "approve":
		// Authorize device
		if err := h.svc.deviceFlowService.AuthorizeDevice(ctx, userCode, sess.UserID, sess.ID); err != nil {
			if h.apiMode {
				return c.JSON(http.StatusBadRequest, map[string]any{
					"error":             "authorization_failed",
					"error_description": "Failed to authorize device",
				})
			}

			page := consent.DeviceCodeEntryPage(consent.CodeEntryPageData{
				ErrorMsg: "Failed to authorize device",
				Branding: branding,
				BasePath: h.basePath,
			})

			return renderNode(c, http.StatusBadRequest, page)
		}

		// In API mode, return JSON success
		if h.apiMode {
			response := map[string]any{
				"success": true,
				"action":  "approved",
				"message": "Device has been authorized successfully",
			}
			if redirectURL != "" {
				response["redirectUrl"] = redirectURL
			}

			return c.JSON(http.StatusOK, response)
		}

		// HTML mode: If redirect URL is provided, redirect there
		if redirectURL != "" {
			c.SetHeader("Location", redirectURL)

			return c.JSON(http.StatusFound, nil)
		}

		// Otherwise show success page
		page := consent.DeviceSuccessPage(true, branding)

		return renderNode(c, http.StatusOK, page)
	case "deny":
		// Deny device
		if err := h.svc.deviceFlowService.DenyDevice(ctx, userCode); err != nil {
			if h.apiMode {
				return c.JSON(http.StatusBadRequest, map[string]any{
					"error":             "denial_failed",
					"error_description": "Failed to deny device",
				})
			}

			page := consent.DeviceCodeEntryPage(consent.CodeEntryPageData{
				ErrorMsg: "Failed to deny device",
				Branding: branding,
				BasePath: h.basePath,
			})

			return renderNode(c, http.StatusBadRequest, page)
		}

		// In API mode, return JSON success
		if h.apiMode {
			response := map[string]any{
				"success": true,
				"action":  "denied",
				"message": "Device authorization has been denied",
			}
			if redirectURL != "" {
				response["redirectUrl"] = redirectURL
			}

			return c.JSON(http.StatusOK, response)
		}

		// HTML mode: If redirect URL is provided, redirect there
		if redirectURL != "" {
			c.SetHeader("Location", redirectURL)

			return c.JSON(http.StatusFound, nil)
		}

		// Otherwise show denial page
		page := consent.DeviceSuccessPage(false, branding)

		return renderNode(c, http.StatusOK, page)
	}

	if h.apiMode {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":             "invalid_action",
			"error_description": "Invalid action. Must be 'approve' or 'deny'",
		})
	}

	return returnHTML(c, http.StatusBadRequest, "<h1>400 Bad Request</h1><p>Invalid action</p>")
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// getSessionToken extracts session token from cookie or Authorization header.
func (h *Handler) getSessionToken(c forge.Context) string {
	// Try cookie first - check both default name and legacy name
	cookieNames := []string{"authsome_session", "session_token"}
	for _, name := range cookieNames {
		if cookie, err := c.Request().Cookie(name); err == nil && cookie != nil && cookie.Value != "" {
			return cookie.Value
		}
	}

	// Try Authorization header (Bearer token)
	auth := c.Request().Header.Get("Authorization")
	if auth != "" && len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}

	return ""
}

// handleAuthRequired handles the case when authentication is required
// In API mode: returns JSON with loginUrl for client-side handling
// In redirect mode: performs HTTP redirect to login page.
func (h *Handler) handleAuthRequired(c forge.Context, returnURL string) error {
	loginRedirectURL := fmt.Sprintf("%s?return_to=%s", h.loginURL, url.QueryEscape(returnURL))

	if h.apiMode {
		// API mode: return JSON response for client-side handling
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error":             "authentication_required",
			"error_description": "User authentication is required to continue",
			"loginUrl":          loginRedirectURL,
			"returnUrl":         returnURL,
		})
	}

	// Traditional mode: HTTP redirect
	c.SetHeader("Location", loginRedirectURL)

	return c.JSON(http.StatusFound, nil)
}

// handleDeviceError handles device flow errors
// In API mode: returns JSON error
// In HTML mode: returns HTML error page.
func (h *Handler) handleDeviceError(c forge.Context, statusCode int, errorCode, errorMsg, redirectURL string) error {
	if h.apiMode {
		return c.JSON(statusCode, map[string]any{
			"error":             errorCode,
			"error_description": errorMsg,
		})
	}

	// HTML mode: render error page
	branding := consent.DefaultBranding()
	page := consent.DeviceCodeEntryPage(consent.CodeEntryPageData{
		ErrorMsg:    errorMsg,
		Branding:    branding,
		BasePath:    h.basePath,
		RedirectURL: redirectURL,
	})

	return renderNode(c, statusCode, page)
}

// redirectWithError redirects to the client with an OAuth error.
func (h *Handler) redirectWithError(c forge.Context, redirectURI, errorCode, errorDescription, state string) error {
	if redirectURI == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest(errorDescription))
	}

	redirectURL := fmt.Sprintf("%s?error=%s&error_description=%s",
		redirectURI,
		url.QueryEscape(errorCode),
		url.QueryEscape(errorDescription))

	if state != "" {
		redirectURL += "&state=" + url.QueryEscape(state)
	}

	c.SetHeader("Location", redirectURL)

	return c.JSON(http.StatusFound, nil)
}

// showConsentScreen displays the consent screen to the user.
func (h *Handler) showConsentScreen(c forge.Context, req *AuthorizeRequest, sess *session.Session) error {
	// Get client information for display
	client, err := h.svc.clientRepo.FindByClientID(c.Request().Context(), req.ClientID)
	if err != nil {
		return h.redirectWithError(c, req.RedirectURI, "server_error", "Failed to load client information", req.State)
	}

	// Extract branding from client
	branding := consent.ExtractBranding(client, nil)

	// Parse scopes for display
	scopes := h.consentSvc.GetScopeDescriptions(h.consentSvc.ParseScopes(req.Scope))

	// Convert to component scopes
	componentScopes := make([]consent.ScopeInfo, len(scopes))
	for i, s := range scopes {
		componentScopes[i] = consent.ScopeInfo{
			Scope:       s.Name,
			Description: s.Description,
		}
	}

	// Build page data
	data := consent.ConsentPageData{
		ClientName:          client.Name,
		ClientID:            req.ClientID,
		LogoURI:             client.LogoURI,
		Scopes:              componentScopes,
		RedirectURI:         req.RedirectURI,
		ResponseType:        req.ResponseType,
		State:               req.State,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		Nonce:               req.Nonce,
		Branding:            branding,
	}

	// Render ForgeUI page
	page := consent.OAuthConsentPage(data)

	return renderNode(c, http.StatusOK, page)
}

// normalizeUserCode normalizes a user code by removing spaces, hyphens, and converting to uppercase.
func normalizeUserCode(code string) string {
	// Remove spaces and hyphens
	code = strings.ReplaceAll(code, " ", "")
	code = strings.ReplaceAll(code, "-", "")
	// Convert to uppercase
	return strings.ToUpper(code)
}

// returnHTML is a helper to return HTML content.
func returnHTML(c forge.Context, statusCode int, html string) error {
	c.SetHeader("Content-Type", "text/html; charset=utf-8")

	return c.String(statusCode, html)
}

// renderNode renders a gomponents node to HTML.
func renderNode(c forge.Context, statusCode int, node gomponents.Node) error {
	var buf bytes.Buffer
	if err := node.Render(&buf); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to render page")
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	_, err := c.Response().Write(buf.Bytes())

	return err
}
