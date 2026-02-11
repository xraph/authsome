package sso

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/responses"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

type Handler struct {
	svc    *Service
	logger forge.Logger
}

func NewHandlerWithLogger(svc *Service, logger forge.Logger) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
	}
}

// ErrorResponse types - use shared responses from core.
//
//nolint:errname // HTTP response DTO, not a Go error type
type ErrorResponse = responses.ErrorResponse
type MessageResponse = responses.MessageResponse
type StatusResponse = responses.StatusResponse
type SuccessResponse = responses.SuccessResponse

func NewHandler(svc *Service) *Handler {
	return &Handler{
		svc:    svc,
		logger: nil, // Will be set if using NewHandlerWithLogger
	}
}

// =============================================================================
// PROVIDER MANAGEMENT
// =============================================================================

// RegisterProvider registers a new SSO provider (SAML or OIDC).
func (h *Handler) RegisterProvider(c forge.Context) error {
	ctx := c.Request().Context()

	// Extract tenant context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("app_id"))
	}

	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("environment_id"))
	}

	orgID, _ := contexts.GetOrganizationID(ctx) // Optional

	// req request
	var req RegisterProviderRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("invalid request body"))
	}

	// Validate type
	if req.Type != "saml" && req.Type != "oidc" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("type must be 'saml' or 'oidc'"))
	}

	// orgIDPtr provider with tenant context
	var orgIDPtr *xid.ID
	if !orgID.IsNil() {
		orgIDPtr = &orgID
	}

	prov := &schema.SSOProvider{
		ID:               xid.New(),
		AppID:            appID,
		EnvironmentID:    envID,
		OrganizationID:   orgIDPtr,
		ProviderID:       req.ProviderID,
		Type:             req.Type,
		Domain:           req.Domain,
		AttributeMapping: req.AttributeMapping,
		SAMLEntryPoint:   req.SAMLEntryPoint,
		SAMLIssuer:       req.SAMLIssuer,
		SAMLCert:         req.SAMLCert,
		OIDCClientID:     req.OIDCClientID,
		OIDCClientSecret: req.OIDCClientSecret,
		OIDCIssuer:       req.OIDCIssuer,
		OIDCRedirectURI:  req.OIDCRedirectURI,
	}

	// Register via service
	if err := h.svc.RegisterProvider(ctx, prov); err != nil {
		if h.logger != nil {
			h.logger.Error("failed to register SSO provider",
				forge.F("provider_id", req.ProviderID),
				forge.F("type", req.Type),
				forge.F("error", err.Error()))
		}

		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	// Audit log: provider registration
	if h.logger != nil {
		h.logger.Debug("SSO provider registered",
			forge.F("provider_id", prov.ProviderID),
			forge.F("type", prov.Type),
			forge.F("app_id", appID.String()),
			forge.F("environment_id", envID.String()),
			forge.F("organization_id", orgID.String()))
	}

	// Return structured response
	return c.JSON(http.StatusOK, ProviderRegisteredResponse{
		ProviderID: prov.ProviderID,
		Type:       prov.Type,
		Status:     "registered",
	})
}

// =============================================================================
// SAML ENDPOINTS
// =============================================================================

// SAMLSPMetadata returns Service Provider metadata.
func (h *Handler) SAMLSPMetadata(c forge.Context) error {
	md := h.svc.SPMetadata()

	return c.JSON(http.StatusOK, MetadataResponse{Metadata: md})
}

// SAMLLogin initiates SAML authentication by generating AuthnRequest.
func (h *Handler) SAMLLogin(c forge.Context) error {
	ctx := c.Request().Context()
	pid := c.Param("providerId")

	if pid == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("providerId"))
	}

	// Get provider with tenant filtering
	provider, err := h.svc.GetProvider(ctx, pid)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("SSO provider not found"))
	}

	if provider.Type != "saml" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("provider is not configured for SAML"))
	}

	if provider.SAMLEntryPoint == "" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("SAML entry point not configured"))
	}

	// req optional request body
	var req SAMLLoginRequest

	_ = c.BindJSON(&req) // Optional, ignore error

	// Generate RelayState for CSRF protection
	relayState := req.RelayState
	if relayState == "" {
		relayState = generateRandomString(16)
	}

	// Generate AuthnRequest and redirect URL
	redirectURL, requestID, err := h.svc.InitiateSAMLLogin(provider.SAMLEntryPoint, relayState)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.SAMLError(err.Error()))
	}

	// Return structured response
	return c.JSON(http.StatusOK, SAMLLoginResponse{
		RedirectURL: redirectURL,
		RequestID:   requestID,
		ProviderID:  pid,
	})
}

// SAMLCallback handles SAML response callback and provisions user.
func (h *Handler) SAMLCallback(c forge.Context) error {
	ctx := c.Request().Context()
	pid := c.Param("providerId")

	if pid == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("providerId"))
	}

	// Get provider with tenant filtering
	provider, err := h.svc.GetProvider(ctx, pid)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("SSO provider not found"))
	}

	if provider.Type != "saml" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("provider is not configured for SAML"))
	}

	// Parse SAML response from form
	samlResponse := c.Request().FormValue("SAMLResponse")
	if samlResponse == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("SAMLResponse"))
	}

	relayState := c.Request().FormValue("RelayState")

	// Validate SAML response with full security checks
	assertion, err := h.svc.ValidateSAMLResponse(samlResponse, provider.SAMLIssuer, relayState)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.SAMLError(err.Error()))
	}

	// Extract email from assertion (subject or attributes)
	email := assertion.Subject
	if email == "" {
		// Try to get email from attributes
		if emailAttrs, ok := assertion.Attributes["email"]; ok && len(emailAttrs) > 0 {
			email = emailAttrs[0]
		}
	}

	if email == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("email in SAML assertion"))
	}

	// Provision user (find or create with JIT)
	usr, err := h.svc.ProvisionUser(ctx, email, assertion.Attributes, provider)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	// Create SSO session
	sess, token, err := h.svc.CreateSSOSession(ctx, usr.ID, provider)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to create SSO session after SAML authentication",
				forge.F("provider_id", pid),
				forge.F("user_id", usr.ID.String()),
				forge.F("email", email),
				forge.F("error", err.Error()))
		}

		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	// Audit log: successful SAML authentication
	if h.logger != nil {
		h.logger.Info("SAML authentication successful",
			forge.F("provider_id", pid),
			forge.F("user_id", usr.ID.String()),
			forge.F("email", email),
			forge.F("session_id", sess.ID.String()),
			forge.F("issuer", assertion.Issuer))
	}

	// Return auth response
	return c.JSON(http.StatusOK, SSOAuthResponse{
		User:    usr,
		Session: sess,
		Token:   token,
	})
}

// =============================================================================
// OIDC ENDPOINTS
// =============================================================================

// OIDCLogin initiates OIDC authentication flow with PKCE.
func (h *Handler) OIDCLogin(c forge.Context) error {
	ctx := c.Request().Context()
	pid := c.Param("providerId")

	if pid == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("providerId"))
	}

	// Get provider with tenant filtering
	provider, err := h.svc.GetProvider(ctx, pid)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("SSO provider not found"))
	}

	if provider.Type != "oidc" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("provider is not configured for OIDC"))
	}

	// req optional request body
	var req OIDCLoginRequest

	_ = c.BindJSON(&req) // Optional, ignore error

	// Generate state and nonce if not provided
	state := req.State
	if state == "" {
		state = generateRandomString(32)
	}

	nonce := req.Nonce
	if nonce == "" {
		nonce = generateRandomString(32)
	}

	// Build redirect URI if not provided
	redirectURI := req.RedirectURI
	if redirectURI == "" {
		redirectURI = provider.OIDCRedirectURI
		if redirectURI == "" {
			// Build from request
			scheme := "https"
			if c.Request().TLS == nil {
				scheme = "http"
			}

			redirectURI = fmt.Sprintf("%s://%s/api/auth/sso/oidc/callback/%s", scheme, c.Request().Host, pid)
		}
	}

	// Initiate OIDC flow with PKCE
	authURL, pkce, err := h.svc.InitiateOIDCLogin(ctx, provider, redirectURI, state, nonce)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.OIDCError(err.Error()))
	}

	// Store PKCE code_verifier and nonce in state store for callback verification
	oidcState := &OIDCState{
		State:        state,
		Nonce:        nonce,
		CodeVerifier: pkce.CodeVerifier,
		ProviderID:   pid,
		RedirectURI:  redirectURI,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}

	if err := h.svc.stateStore.Store(ctx, oidcState); err != nil {
		if h.logger != nil {
			h.logger.Error("failed to store OIDC state",
				forge.F("state", state),
				forge.F("error", err.Error()))
		}

		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	return c.JSON(http.StatusOK, OIDCLoginResponse{
		AuthURL:    authURL,
		State:      state,
		Nonce:      nonce,
		ProviderID: pid,
	})
}

// OIDCCallback handles OIDC callback and provisions user.
func (h *Handler) OIDCCallback(c forge.Context) error {
	ctx := c.Request().Context()
	pid := c.Param("providerId")

	if pid == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("providerId"))
	}

	// Get provider with tenant filtering
	provider, err := h.svc.GetProvider(ctx, pid)
	if err != nil {
		return c.JSON(http.StatusNotFound, errs.NotFound("SSO provider not found"))
	}

	if provider.Type != "oidc" {
		return c.JSON(http.StatusBadRequest, errs.BadRequest("provider is not configured for OIDC"))
	}

	// Parse callback parameters
	q := c.Request().URL.Query()
	code := q.Get("code")
	state := q.Get("state")

	if code == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("code"))
	}

	if state == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("state"))
	}

	// Retrieve stored PKCE code_verifier and nonce from state storage
	oidcState, err := h.svc.stateStore.Get(ctx, state)
	if err != nil || oidcState == nil {
		if h.logger != nil {
			h.logger.Error("failed to retrieve OIDC state or state expired/not found",
				forge.F("state", state),
				forge.F("provider_id", pid))
		}

		return c.JSON(http.StatusBadRequest, errs.OIDCError("invalid or expired state parameter"))
	}

	// Verify the state belongs to this provider
	if oidcState.ProviderID != pid {
		if h.logger != nil {
			h.logger.Error("state provider mismatch",
				forge.F("expected", pid),
				forge.F("got", oidcState.ProviderID))
		}

		return c.JSON(http.StatusBadRequest, errs.OIDCError("state provider mismatch"))
	}

	codeVerifier := oidcState.CodeVerifier
	nonce := oidcState.Nonce

	// _ the state after retrieval (one-time use)
	_ = h.svc.stateStore.Delete(ctx, state)

	// Build redirect URI
	redirectURI := provider.OIDCRedirectURI
	if redirectURI == "" {
		scheme := "https"
		if c.Request().TLS == nil {
			scheme = "http"
		}

		redirectURI = fmt.Sprintf("%s://%s/api/auth/sso/oidc/callback/%s", scheme, c.Request().Host, pid)
	}

	// Exchange authorization code for tokens
	tokenResp, err := h.svc.ExchangeOIDCCode(ctx, provider, code, redirectURI, codeVerifier)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errs.OIDCError("token exchange failed: "+err.Error()))
	}

	// Validate and extract user info
	var (
		email      string
		attributes map[string][]string
	)

	if tokenResp.IDToken != "" {
		// Validate ID token
		oidcUserInfo, err := h.svc.ValidateOIDCIDToken(ctx, provider, tokenResp.IDToken, nonce)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.OIDCError("ID token validation failed: "+err.Error()))
		}

		email = oidcUserInfo.Email
		attributes = map[string][]string{
			"email":              {oidcUserInfo.Email},
			"name":               {oidcUserInfo.Name},
			"given_name":         {oidcUserInfo.GivenName},
			"family_name":        {oidcUserInfo.FamilyName},
			"picture":            {oidcUserInfo.Picture},
			"preferred_username": {oidcUserInfo.PreferredUsername},
		}
	} else if tokenResp.AccessToken != "" {
		// Fallback: fetch user info from userinfo endpoint
		oidcUserInfo, err := h.svc.GetOIDCUserInfo(ctx, provider, tokenResp.AccessToken)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errs.OIDCError("userinfo fetch failed: "+err.Error()))
		}

		email = oidcUserInfo.Email
		attributes = map[string][]string{
			"email":              {oidcUserInfo.Email},
			"name":               {oidcUserInfo.Name},
			"given_name":         {oidcUserInfo.GivenName},
			"family_name":        {oidcUserInfo.FamilyName},
			"picture":            {oidcUserInfo.Picture},
			"preferred_username": {oidcUserInfo.PreferredUsername},
		}
	} else {
		return c.JSON(http.StatusBadRequest, errs.OIDCError("no ID token or access token in response"))
	}

	if email == "" {
		return c.JSON(http.StatusBadRequest, errs.RequiredField("email in OIDC user info"))
	}

	// Provision user (find or create with JIT)
	usr, err := h.svc.ProvisionUser(ctx, email, attributes, provider)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	// Create SSO session
	sess, token, err := h.svc.CreateSSOSession(ctx, usr.ID, provider)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to create SSO session after OIDC authentication",
				forge.F("provider_id", pid),
				forge.F("user_id", usr.ID.String()),
				forge.F("email", email),
				forge.F("error", err.Error()))
		}

		return c.JSON(http.StatusInternalServerError, errs.InternalError(err))
	}

	// Audit log: successful OIDC authentication
	if h.logger != nil {
		h.logger.Info("OIDC authentication successful",
			forge.F("provider_id", pid),
			forge.F("user_id", usr.ID.String()),
			forge.F("email", email),
			forge.F("session_id", sess.ID.String()),
			forge.F("issuer", provider.OIDCIssuer))
	}

	return c.JSON(http.StatusOK, SSOAuthResponse{
		User:    usr,
		Session: sess,
		Token:   token,
	})
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// generateRandomString generates a cryptographically secure random string.
func generateRandomString(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to xid if random fails
		return xid.New().String()
	}

	return strings.TrimRight(base64.URLEncoding.EncodeToString(bytes), "=")[:length]
}
