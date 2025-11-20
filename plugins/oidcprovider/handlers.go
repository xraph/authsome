package oidcprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/oidcprovider/components"
	"github.com/xraph/forge"
)

// Handler handles OIDC provider HTTP endpoints
type Handler struct {
	svc              *Service
	revocationSvc    *RevokeTokenService
	introspectionSvc *IntrospectionService
	consentSvc       *ConsentService
	discoverySvc     *DiscoveryService
	clientAuth       *ClientAuthenticator
}

// NewHandler creates a new OIDC handler
func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc:              svc,
		revocationSvc:    svc.revocation,
		introspectionSvc: svc.introspection,
		consentSvc:       svc.consent,
		discoverySvc:     svc.discovery,
		clientAuth:       svc.clientAuth,
	}
}

// =============================================================================
// DISCOVERY ENDPOINT
// =============================================================================

// Discovery handles the OIDC discovery endpoint (.well-known/openid-configuration)
func (h *Handler) Discovery(c forge.Context) error {
	ctx := c.Request().Context()

	// Build base URL from request
	scheme := "https"
	if c.Request().TLS == nil {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, c.Request().Host)

	doc := h.discoverySvc.GetDiscoveryDocument(ctx, baseURL)
	return c.JSON(http.StatusOK, doc)
}

// JWKS returns the JSON Web Key Set
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

// Authorize handles OAuth2/OIDC authorization requests
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
	sessionToken := h.getSessionToken(c)
	if sessionToken == "" {
		loginURL := fmt.Sprintf("/auth/signin?return_to=%s", url.QueryEscape(c.Request().URL.String()))
		c.SetHeader("Location", loginURL)
		return c.JSON(http.StatusFound, nil)
	}

	// Validate user session
	sess, err := h.svc.sessionSvc.FindByToken(ctx, sessionToken)
	if err != nil || sess == nil {
		loginURL := fmt.Sprintf("/auth/signin?return_to=%s", url.QueryEscape(c.Request().URL.String()))
		c.SetHeader("Location", loginURL)
		return c.JSON(http.StatusFound, nil)
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

// HandleConsent processes the consent form submission
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

// Token handles the token endpoint
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
	default:
		return c.JSON(http.StatusBadRequest, errs.BadRequest("unsupported grant type: "+req.GrantType))
	}
}

// handleAuthorizationCodeGrant handles the authorization_code grant type
func (h *Handler) handleAuthorizationCodeGrant(ctx context.Context, c forge.Context, req *TokenRequest) error {
	// Validate authorization code
	authCode, err := h.svc.ValidateAuthorizationCode(ctx, req.Code, req.ClientID, req.RedirectURI, req.CodeVerifier)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
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

	userInfo := map[string]interface{}{
		"sub":   user.ID.String(),
		"email": user.Email,
		"name":  user.Name,
	}

	// Exchange code for tokens
	tokenResponse, err := h.svc.ExchangeCodeForTokens(ctx, authCode, userInfo)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, tokenResponse)
}

// handleRefreshTokenGrant handles the refresh_token grant type
func (h *Handler) handleRefreshTokenGrant(ctx context.Context, c forge.Context, req *TokenRequest) error {
	// Extract refresh token
	refreshToken := req.RefreshToken
	if refreshToken == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("refresh_token"))
	}

	// Authenticate client
	clientAuth, client, err := h.clientAuth.AuthenticateClient(ctx, c.Request())
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	_ = clientAuth // Authenticated

	// Refresh the token (with optional rotation)
	tokenResponse, err := h.svc.RefreshAccessToken(ctx, refreshToken, client.ClientID, req.Scope)
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusUnauthorized, errs.UnauthorizedWithMessage("invalid or expired refresh token"))
	}

	return c.JSON(http.StatusOK, tokenResponse)
}

// handleClientCredentialsGrant handles the client_credentials grant type (M2M)
func (h *Handler) handleClientCredentialsGrant(ctx context.Context, c forge.Context, req *TokenRequest) error {
	// Authenticate client (confidential clients only)
	clientAuth, client, err := h.clientAuth.AuthenticateClient(ctx, c.Request())
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
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
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, tokenResponse)
}

// =============================================================================
// USERINFO ENDPOINT
// =============================================================================

// UserInfo returns user information based on the access token
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
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusUnauthorized, errs.UnauthorizedWithMessage("invalid or expired access token"))
	}

	return c.JSON(http.StatusOK, userInfo)
}

// =============================================================================
// TOKEN INTROSPECTION (RFC 7662)
// =============================================================================

// IntrospectToken handles token introspection requests
func (h *Handler) IntrospectToken(c forge.Context) error {
	ctx := c.Request().Context()

	// Authenticate client
	authResult, client, err := h.clientAuth.AuthenticateClient(ctx, c.Request())
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusUnauthorized, errs.Unauthorized())
	}

	// Validate client can introspect (confidential clients only)
	if err := h.clientAuth.ValidateClientForEndpoint(client, "introspect"); err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
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
		if authErr, ok := err.(*errs.AuthsomeError); ok {
			return c.JSON(authErr.HTTPStatus, authErr)
		}
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, response)
}

// =============================================================================
// TOKEN REVOCATION (RFC 7009)
// =============================================================================

// RevokeToken handles token revocation requests
func (h *Handler) RevokeToken(c forge.Context) error {
	ctx := c.Request().Context()

	// Authenticate client
	_, _, err := h.clientAuth.AuthenticateClient(ctx, c.Request())
	if err != nil {
		if authErr, ok := err.(*errs.AuthsomeError); ok {
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
// HELPER METHODS
// =============================================================================

// getSessionToken extracts session token from cookie or Authorization header
func (h *Handler) getSessionToken(c forge.Context) string {
	// Try cookie first
	if cookie, err := c.Request().Cookie("session_token"); err == nil && cookie != nil {
		return cookie.Value
	}

	// Try Authorization header
	auth := c.Request().Header.Get("Authorization")
	if auth != "" && len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}

	return ""
}

// redirectWithError redirects to the client with an OAuth error
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

// showConsentScreen displays the consent screen to the user
func (h *Handler) showConsentScreen(c forge.Context, req *AuthorizeRequest, sess *session.Session) error {
	// Get client information for display
	client, err := h.svc.clientRepo.FindByClientID(c.Request().Context(), req.ClientID)
	if err != nil {
		return h.redirectWithError(c, req.RedirectURI, "server_error", "Failed to load client information", req.State)
	}

	// Parse scopes for display
	scopes := h.consentSvc.GetScopeDescriptions(h.consentSvc.ParseScopes(req.Scope))

	// Convert to component scopes
	componentScopes := make([]components.ScopeInfo, len(scopes))
	for i, s := range scopes {
		componentScopes[i] = components.ScopeInfo{
			Scope:       s.Name,
			Description: s.Description,
		}
	}

	// Build page data
	data := components.ConsentPageData{
		ClientName:          client.Name,
		ClientID:            req.ClientID,
		LogoURI:             client.LogoURI,
		Scopes:              componentScopes,
		RedirectURI:         req.RedirectURI,
		ResponseType:        req.ResponseType,
		State:               req.State,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
	}

	// Render component to HTML
	var buf bytes.Buffer
	err = components.ConsentPage(data).Render(&buf)
	if err != nil {
		return h.redirectWithError(c, req.RedirectURI, "server_error", "Failed to render consent page", req.State)
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	return c.String(http.StatusOK, buf.String())
}
