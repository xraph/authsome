package sso

import (
    "encoding/json"
    "net/http"
    "github.com/xraph/authsome/schema"
    "github.com/xraph/forge"
)

type Handler struct {
    svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterProvider registers an SSO provider (SAML or OIDC); org scoping TBD
func (h *Handler) RegisterProvider(c *forge.Context) error {
    var req struct {
        ProviderID       string `json:"providerId"`
        Type             string `json:"type"`
        Domain           string `json:"domain"`
        SAMLEntryPoint   string `json:"SAMLEntryPoint"`
        SAMLIssuer       string `json:"SAMLIssuer"`
        SAMLCert         string `json:"SAMLCert"`
        OIDCClientID     string `json:"OIDCClientID"`
        OIDCClientSecret string `json:"OIDCClientSecret"`
        OIDCIssuer       string `json:"OIDCIssuer"`
        OIDCRedirectURI  string `json:"OIDCRedirectURI"`
    }
    if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
    }
    if req.ProviderID == "" || req.Type == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "providerId and type required"})
    }
    prov := &schema.SSOProvider{
        ProviderID:       req.ProviderID,
        Type:             req.Type,
        Domain:           req.Domain,
        SAMLEntryPoint:   req.SAMLEntryPoint,
        SAMLIssuer:       req.SAMLIssuer,
        SAMLCert:         req.SAMLCert,
        OIDCClientID:     req.OIDCClientID,
        OIDCClientSecret: req.OIDCClientSecret,
        OIDCIssuer:       req.OIDCIssuer,
        OIDCRedirectURI:  req.OIDCRedirectURI,
    }
    _ = h.svc.RegisterProvider(c.Request().Context(), prov)
    return c.JSON(http.StatusOK, map[string]string{"status": "registered", "providerId": prov.ProviderID})
}

// SAMLSPMetadata returns Service Provider metadata (placeholder)
func (h *Handler) SAMLSPMetadata(c *forge.Context) error {
    md := h.svc.SPMetadata()
    return c.JSON(http.StatusOK, map[string]string{"metadata": md})
}

// SAMLCallback handles SAML response callback for given provider
func (h *Handler) SAMLCallback(c *forge.Context) error {
    pid := c.Param("providerId")
    if pid == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing providerId"})
    }
    p, ok := h.svc.GetProvider(pid)
    if !ok {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "provider not found"})
    }
    if p.Type != "saml" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "provider type mismatch"})
    }
    samlResponse := c.Request().FormValue("SAMLResponse")
    if samlResponse == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing SAMLResponse"})
    }
    relayState := c.Request().FormValue("RelayState")
    
    // Use enhanced validation with full security checks
    assertion, err := h.svc.ValidateSAMLResponse(samlResponse, p.SAMLIssuer, relayState)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid SAML response", "details": err.Error()})
    }
    
    return c.JSON(http.StatusOK, map[string]any{
        "status": "saml_callback_ok", 
        "subject": assertion.Subject, 
        "issuer": assertion.Issuer,
        "attributes": assertion.Attributes,
        "providerId": pid,
    })
}

// SAMLLogin initiates SAML authentication by redirecting to IdP
func (h *Handler) SAMLLogin(c *forge.Context) error {
    pid := c.Param("providerId")
    if pid == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing providerId"})
    }
    p, ok := h.svc.GetProvider(pid)
    if !ok {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "provider not found"})
    }
    if p.Type != "saml" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "provider type mismatch"})
    }
    if p.SAMLEntryPoint == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "SAML entry point not configured"})
    }
    
    // Generate RelayState for CSRF protection
    relayState := c.Request().URL.Query().Get("RelayState")
    if relayState == "" {
        relayState = "default"
    }
    
    // Generate AuthnRequest and redirect URL
    redirectURL, requestID, err := h.svc.InitiateSAMLLogin(p.SAMLEntryPoint, relayState)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to initiate SAML login"})
    }
    
    // Return redirect URL for client to follow
    return c.JSON(http.StatusOK, map[string]any{
        "redirect_url": redirectURL,
        "request_id": requestID,
        "provider_id": pid,
    })
}

// OIDCCallback handles OIDC response callback for given provider
func (h *Handler) OIDCCallback(c *forge.Context) error {
    pid := c.Param("providerId")
    if pid == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing providerId"})
    }
    
    provider, ok := h.svc.GetProvider(pid)
    if !ok {
        return c.JSON(http.StatusNotFound, map[string]string{"error": "provider not found"})
    }
    if provider.Type != "oidc" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "provider type mismatch"})
    }
    
    q := c.Request().URL.Query()
    code := q.Get("code")
    if code == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing authorization code"})
    }
    
    state := q.Get("state")
    if state == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing state parameter"})
    }
    
    // Get PKCE code verifier from session/state (in real implementation)
    // For now, we'll use a placeholder
    codeVerifier := "placeholder_code_verifier" // TODO: Retrieve from session
    redirectURI := provider.OIDCRedirectURI
    if redirectURI == "" {
        redirectURI = "http://localhost:8080/api/auth/sso/oidc/callback/" + pid
    }
    
    // Exchange authorization code for tokens
	tokenResponse, err := h.svc.ExchangeOIDCCode(c.Request().Context(), provider, code, redirectURI, codeVerifier)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "token_exchange_failed",
            "message": err.Error(),
        })
    }
    
    var userInfo map[string]interface{}
    
    // If we have an ID token, validate it and extract user info
    if tokenResponse.IDToken != "" {
        nonce := "placeholder_nonce" // TODO: Retrieve from session
        oidcUserInfo, err := h.svc.ValidateOIDCIDToken(c.Request().Context(), provider, tokenResponse.IDToken, nonce)
        if err != nil {
            return c.JSON(http.StatusBadRequest, map[string]string{
                "error": "id_token_validation_failed",
                "message": err.Error(),
            })
        }
        
        userInfo = map[string]interface{}{
            "sub":                oidcUserInfo.Sub,
            "name":               oidcUserInfo.Name,
            "email":              oidcUserInfo.Email,
            "email_verified":     oidcUserInfo.EmailVerified,
            "given_name":         oidcUserInfo.GivenName,
            "family_name":        oidcUserInfo.FamilyName,
            "picture":            oidcUserInfo.Picture,
            "preferred_username": oidcUserInfo.PreferredUsername,
        }
    } else if tokenResponse.AccessToken != "" {
        // Fallback: fetch user info from userinfo endpoint
        oidcUserInfo, err := h.svc.GetOIDCUserInfo(c.Request().Context(), provider, tokenResponse.AccessToken)
        if err != nil {
            return c.JSON(http.StatusBadRequest, map[string]string{
                "error": "userinfo_fetch_failed",
                "message": err.Error(),
            })
        }
        
        userInfo = map[string]interface{}{
            "sub":                oidcUserInfo.Sub,
            "name":               oidcUserInfo.Name,
            "email":              oidcUserInfo.Email,
            "email_verified":     oidcUserInfo.EmailVerified,
            "given_name":         oidcUserInfo.GivenName,
            "family_name":        oidcUserInfo.FamilyName,
            "picture":            oidcUserInfo.Picture,
            "preferred_username": oidcUserInfo.PreferredUsername,
        }
    }
    
    return c.JSON(http.StatusOK, map[string]interface{}{
        "status":       "oidc_callback_success",
        "provider_id":  pid,
        "user_info":    userInfo,
        "access_token": tokenResponse.AccessToken,
        "token_type":   tokenResponse.TokenType,
        "expires_in":   tokenResponse.ExpiresIn,
        "scope":        tokenResponse.Scope,
    })
}