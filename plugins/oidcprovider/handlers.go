package oidcprovider

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"

    "github.com/rs/xid"
    "github.com/xraph/forge"
    "github.com/xraph/authsome/core/session"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Authorize handles OAuth2/OIDC authorization requests
func (h *Handler) Authorize(c *forge.Context) error {
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
    if err := h.svc.ValidateAuthorizeRequest(c.Request().Context(), req); err != nil {
        return h.redirectWithError(c, req.RedirectURI, "invalid_request", err.Error(), req.State)
    }
    
    // Check if user is authenticated (session check)
    sessionToken := h.getSessionToken(c)
    if sessionToken == "" {
        // Redirect to login with return URL
        loginURL := fmt.Sprintf("/auth/signin?return_to=%s", 
            url.QueryEscape(c.Request().URL.String()))
        c.Header().Set("Location", loginURL)
        return c.JSON(302, nil)
    }
    
    // Validate user session
    sess, err := h.svc.CheckUserSession(c.Request().Context(), sessionToken)
    if err != nil {
        // Invalid session, redirect to login
        loginURL := fmt.Sprintf("/auth/signin?return_to=%s", 
            url.QueryEscape(c.Request().URL.String()))
        c.Header().Set("Location", loginURL)
        return c.JSON(302, nil)
    }
    
    // Check if consent is required
    if h.requiresConsent(req.Scope) {
        // Check if user has already consented
        hasConsent, err := h.checkExistingConsent(c.Request().Context(), sess.UserID, req.ClientID, req.Scope)
        if err != nil {
            return h.redirectWithError(c, req.RedirectURI, "server_error", "Failed to check consent", req.State)
        }
        
        if !hasConsent {
            // Show consent screen
            return h.showConsentScreen(c, req, sess)
        }
    }
    
    // Generate and store authorization code
    authCode, err := h.svc.CreateAuthorizationCode(c.Request().Context(), req, sess.UserID)
    if err != nil {
        return h.redirectWithError(c, req.RedirectURI, "server_error", "Failed to generate code", req.State)
    }
    
    // Redirect back with authorization code
    redirectURL := fmt.Sprintf("%s?code=%s", req.RedirectURI, authCode.Code)
    if req.State != "" {
        redirectURL += "&state=" + url.QueryEscape(req.State)
    }
    
    c.Header().Set("Location", redirectURL)
    return c.JSON(302, nil)
}

// TokenRequest represents the token endpoint request
type TokenRequest struct {
    GrantType    string `json:"grant_type" form:"grant_type"`
    Code         string `json:"code" form:"code"`
    RedirectURI  string `json:"redirect_uri" form:"redirect_uri"`
    ClientID     string `json:"client_id" form:"client_id"`
    ClientSecret string `json:"client_secret" form:"client_secret"`
    CodeVerifier string `json:"code_verifier" form:"code_verifier"`
}

// Token handles the token endpoint
func (h *Handler) Token(c *forge.Context) error {
    var req TokenRequest
    
    // Parse form data or JSON
    if c.Request().Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
        if err := c.Request().ParseForm(); err != nil {
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "Failed to parse form data"})
        }
        req.GrantType = c.Request().FormValue("grant_type")
        req.Code = c.Request().FormValue("code")
        req.RedirectURI = c.Request().FormValue("redirect_uri")
        req.ClientID = c.Request().FormValue("client_id")
        req.ClientSecret = c.Request().FormValue("client_secret")
        req.CodeVerifier = c.Request().FormValue("code_verifier")
    } else {
        if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
            return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request", "error_description": "Invalid JSON"})
        }
    }

    // Validate grant type
    if req.GrantType != "authorization_code" {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
    }

    // Validate authorization code
    authCode, err := h.svc.ValidateAuthorizationCode(c.Request().Context(), req.Code, req.ClientID, req.RedirectURI, req.CodeVerifier)
    if err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_grant", "error_description": err.Error()})
    }

    // Mark code as used
    if err := h.svc.MarkCodeAsUsed(c.Request().Context(), req.Code); err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error", "error_description": "Failed to mark code as used"})
    }

    // Get user info for ID token (placeholder - should get from user service)
    userInfo := map[string]interface{}{
        "sub":   authCode.UserID.String(),
        "email": "demo@example.com", // TODO: Get from user service
        "name":  "Demo User",        // TODO: Get from user service
    }

    // Exchange code for tokens
    tokenResponse, err := h.svc.ExchangeCodeForTokens(c.Request().Context(), authCode, userInfo)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error", "error_description": err.Error()})
    }

    return c.JSON(http.StatusOK, tokenResponse)
}

// UserInfo returns user info based on scopes (placeholder user)
// UserInfo returns user information based on the access token
func (h *Handler) UserInfo(c *forge.Context) error {
    // Extract access token from Authorization header
    authHeader := c.Request().Header.Get("Authorization")
    if authHeader == "" {
        return c.JSON(http.StatusUnauthorized, map[string]string{
            "error": "invalid_request",
            "error_description": "Missing Authorization header",
        })
    }
    
    // Check for Bearer token format
    const bearerPrefix = "Bearer "
    if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
        return c.JSON(http.StatusUnauthorized, map[string]string{
            "error": "invalid_request", 
            "error_description": "Authorization header must use Bearer token",
        })
    }
    
    accessToken := authHeader[len(bearerPrefix):]
    if accessToken == "" {
        return c.JSON(http.StatusUnauthorized, map[string]string{
            "error": "invalid_token",
            "error_description": "Access token is required",
        })
    }
    
    // Get user information from the service
    userInfo, err := h.svc.GetUserInfoFromToken(c.Request().Context(), accessToken)
    if err != nil {
        return c.JSON(http.StatusUnauthorized, map[string]string{
            "error": "invalid_token",
            "error_description": "Invalid or expired access token",
        })
    }
    
    return c.JSON(http.StatusOK, userInfo)
}

// JWKS returns the JSON Web Key Set
func (h *Handler) JWKS(c *forge.Context) error {
    jwks, err := h.svc.GetJWKS()
    if err != nil {
        return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get JWKS"})
    }
    return c.JSON(http.StatusOK, jwks)
}

// RegisterClient registers a new OAuth client
func (h *Handler) RegisterClient(c *forge.Context) error {
    var req struct{ Name string `json:"name"`; RedirectURI string `json:"redirect_uri"` }
    if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid_request"})
    }
    cl, err := h.svc.RegisterClient(c.Request().Context(), req.Name, req.RedirectURI)
    if err != nil { return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()}) }
    return c.JSON(http.StatusOK, map[string]any{"client_id": cl.ClientID, "client_secret": cl.ClientSecret, "redirect_uri": cl.RedirectURI})
}

// Helper methods for authorization flow

// getSessionToken extracts session token from cookie or Authorization header
func (h *Handler) getSessionToken(c *forge.Context) string {
    // Try cookie first
    if token, err := c.Cookie("session_token"); err == nil {
        return token
    }
    
    // Try Authorization header
    auth := c.Request().Header.Get("Authorization")
    if auth != "" && len(auth) > 7 && auth[:7] == "Bearer " {
        return auth[7:]
    }
    
    return ""
}

// redirectWithError redirects to the client with an OAuth error
func (h *Handler) redirectWithError(c *forge.Context, redirectURI, errorCode, errorDescription, state string) error {
    if redirectURI == "" {
        return c.JSON(400, map[string]string{
            "error": errorCode,
            "error_description": errorDescription,
        })
    }
    
    redirectURL := fmt.Sprintf("%s?error=%s&error_description=%s", 
        redirectURI, 
        url.QueryEscape(errorCode), 
        url.QueryEscape(errorDescription))
    
    if state != "" {
        redirectURL += "&state=" + url.QueryEscape(state)
    }
    
    c.Header().Set("Location", redirectURL)
    return c.JSON(302, nil)
}

// requiresConsent checks if the requested scope requires user consent
func (h *Handler) requiresConsent(scope string) bool {
    // For now, require consent for all scopes except basic openid
    return scope != "" && scope != "openid"
}

// checkExistingConsent checks if user has already consented to the scope for this client
func (h *Handler) checkExistingConsent(ctx context.Context, userID xid.ID, clientID, scope string) (bool, error) {
    // TODO: Implement consent storage and checking
    // For now, always require consent
    return false, nil
}

// showConsentScreen displays the consent screen to the user
func (h *Handler) showConsentScreen(c *forge.Context, req *AuthorizeRequest, sess *session.Session) error {
    // Get client information for display
    client, err := h.svc.clientRepo.FindByClientID(c.Request().Context(), req.ClientID)
    if err != nil {
        return h.redirectWithError(c, req.RedirectURI, "server_error", "Failed to load client information", req.State)
    }

    // Parse scopes for display
    scopes := h.parseScopes(req.Scope)
    
    // Render consent screen HTML
    html := h.generateConsentHTML(client.Name, client.ClientID, scopes, req, sess)
    
    return c.HTML(200, html)
}

// parseScopes converts scope string to user-friendly descriptions
func (h *Handler) parseScopes(scope string) []ScopeInfo {
    scopeDescriptions := map[string]string{
        "openid":  "Verify your identity",
        "profile": "Access your basic profile information (name, username)",
        "email":   "Access your email address",
        "offline_access": "Keep you signed in",
    }
    
    var scopes []ScopeInfo
    if scope == "" {
        return scopes
    }
    
    for _, s := range strings.Split(scope, " ") {
        s = strings.TrimSpace(s)
        if s == "" {
            continue
        }
        
        description, exists := scopeDescriptions[s]
        if !exists {
            description = fmt.Sprintf("Access %s permissions", s)
        }
        
        scopes = append(scopes, ScopeInfo{
            Name:        s,
            Description: description,
        })
    }
    
    return scopes
}

// ScopeInfo represents a scope with its description
type ScopeInfo struct {
    Name        string
    Description string
}

// generateConsentHTML generates the consent screen HTML
func (h *Handler) generateConsentHTML(clientName, clientID string, scopes []ScopeInfo, req *AuthorizeRequest, sess *session.Session) string {
    return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authorization Required</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .consent-container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            max-width: 480px;
            width: 100%%;
            padding: 40px;
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .app-icon {
            width: 64px;
            height: 64px;
            background: #667eea;
            border-radius: 12px;
            margin: 0 auto 16px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-size: 24px;
            font-weight: bold;
        }
        .title {
            font-size: 24px;
            font-weight: 600;
            color: #1a202c;
            margin-bottom: 8px;
        }
        .subtitle {
            color: #718096;
            font-size: 16px;
        }
        .client-info {
            background: #f7fafc;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 24px;
            border-left: 4px solid #667eea;
        }
        .client-name {
            font-weight: 600;
            color: #2d3748;
            margin-bottom: 4px;
        }
        .permissions {
            margin-bottom: 32px;
        }
        .permissions h3 {
            font-size: 18px;
            font-weight: 600;
            color: #2d3748;
            margin-bottom: 16px;
        }
        .permission-item {
            display: flex;
            align-items: center;
            padding: 12px 0;
            border-bottom: 1px solid #e2e8f0;
        }
        .permission-item:last-child {
            border-bottom: none;
        }
        .permission-icon {
            width: 20px;
            height: 20px;
            background: #48bb78;
            border-radius: 50%%;
            margin-right: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-size: 12px;
        }
        .permission-text {
            color: #4a5568;
            font-size: 14px;
        }
        .actions {
            display: flex;
            gap: 12px;
        }
        .btn {
            flex: 1;
            padding: 12px 24px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
        }
        .btn-deny {
            background: #f7fafc;
            color: #4a5568;
            border: 2px solid #e2e8f0;
        }
        .btn-deny:hover {
            background: #edf2f7;
            border-color: #cbd5e0;
        }
        .btn-allow {
            background: #667eea;
            color: white;
        }
        .btn-allow:hover {
            background: #5a67d8;
        }
        .security-note {
            margin-top: 24px;
            padding: 16px;
            background: #fef5e7;
            border-radius: 8px;
            border-left: 4px solid #f6ad55;
        }
        .security-note p {
            color: #744210;
            font-size: 14px;
            line-height: 1.5;
        }
    </style>
</head>
<body>
    <div class="consent-container">
        <div class="header">
            <div class="app-icon">🔐</div>
            <h1 class="title">Authorization Required</h1>
            <p class="subtitle">%s wants to access your account</p>
        </div>
        
        <div class="client-info">
            <div class="client-name">%s</div>
            <div style="color: #718096; font-size: 14px;">Client ID: %s</div>
        </div>
        
        <div class="permissions">
            <h3>This application will be able to:</h3>
            %s
        </div>
        
        <form method="POST" action="/oauth2/consent">
            <input type="hidden" name="client_id" value="%s">
            <input type="hidden" name="redirect_uri" value="%s">
            <input type="hidden" name="response_type" value="%s">
            <input type="hidden" name="scope" value="%s">
            <input type="hidden" name="state" value="%s">
            <input type="hidden" name="code_challenge" value="%s">
            <input type="hidden" name="code_challenge_method" value="%s">
            
            <div class="actions">
                <button type="submit" name="action" value="deny" class="btn btn-deny">
                    Deny
                </button>
                <button type="submit" name="action" value="allow" class="btn btn-allow">
                    Allow Access
                </button>
            </div>
        </form>
        
        <div class="security-note">
            <p><strong>Security Notice:</strong> Only authorize applications you trust. You can revoke access at any time in your account settings.</p>
        </div>
    </div>
</body>
</html>`,
        clientName,
        clientName,
        clientID,
        h.generatePermissionsHTML(scopes),
        req.ClientID,
        req.RedirectURI,
        req.ResponseType,
        req.Scope,
        req.State,
        req.CodeChallenge,
        req.CodeChallengeMethod,
    )
}

// generatePermissionsHTML generates HTML for the permissions list
func (h *Handler) generatePermissionsHTML(scopes []ScopeInfo) string {
    var html strings.Builder
    
    for _, scope := range scopes {
        html.WriteString(fmt.Sprintf(`
            <div class="permission-item">
                <div class="permission-icon">✓</div>
                <div class="permission-text">%s</div>
            </div>`, scope.Description))
    }
    
    return html.String()
}

// HandleConsent processes the consent form submission
func (h *Handler) HandleConsent(c *forge.Context) error {
    // Parse form data
    if err := c.Request().ParseForm(); err != nil {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "invalid_request",
            "error_description": "Failed to parse form data",
        })
    }

    // Get form values
    action := c.Request().FormValue("action")
    clientID := c.Request().FormValue("client_id")
    redirectURI := c.Request().FormValue("redirect_uri")
    responseType := c.Request().FormValue("response_type")
    scope := c.Request().FormValue("scope")
    state := c.Request().FormValue("state")
    codeChallenge := c.Request().FormValue("code_challenge")
    codeChallengeMethod := c.Request().FormValue("code_challenge_method")

    // Validate required parameters
    if clientID == "" || redirectURI == "" || responseType == "" {
        return c.JSON(http.StatusBadRequest, map[string]string{
            "error": "invalid_request",
            "error_description": "Missing required parameters",
        })
    }

    // Get current session
    sessionToken := h.getSessionToken(c)
    if sessionToken == "" {
        return h.redirectWithError(c, redirectURI, "access_denied", "No active session", state)
    }

    sess, err := h.svc.sessionSvc.FindByToken(c.Request().Context(), sessionToken)
    if err != nil || sess == nil {
        return h.redirectWithError(c, redirectURI, "access_denied", "Invalid session", state)
    }

    // Handle user decision
    if action == "deny" {
        // User denied consent - redirect with error
        return h.redirectWithError(c, redirectURI, "access_denied", "User denied the request", state)
    }

    if action != "allow" {
        return h.redirectWithError(c, redirectURI, "invalid_request", "Invalid action", state)
    }

    // User allowed consent - create authorization request and proceed
    req := &AuthorizeRequest{
        ClientID:            clientID,
        RedirectURI:         redirectURI,
        ResponseType:        responseType,
        Scope:               scope,
        State:               state,
        CodeChallenge:       codeChallenge,
        CodeChallengeMethod: codeChallengeMethod,
    }

    // Validate the authorization request
    if err := h.svc.ValidateAuthorizeRequest(c.Request().Context(), req); err != nil {
        return h.redirectWithError(c, req.RedirectURI, "invalid_request", err.Error(), req.State)
    }

    // Store consent decision (for future requests)
    if err := h.storeConsentDecision(c.Request().Context(), sess.UserID, clientID, scope, true); err != nil {
        // Log error but don't fail the request
        // TODO: Add proper logging
    }

    // Create authorization code
    authCode, err := h.svc.CreateAuthorizationCode(c.Request().Context(), req, sess.UserID)
    if err != nil {
        return h.redirectWithError(c, req.RedirectURI, "server_error", "Failed to create authorization code", req.State)
    }

    // Build redirect URL with authorization code
    redirectURL, err := url.Parse(req.RedirectURI)
    if err != nil {
        return h.redirectWithError(c, req.RedirectURI, "invalid_request", "Invalid redirect URI", req.State)
    }

    query := redirectURL.Query()
    query.Set("code", authCode.Code)
    if req.State != "" {
        query.Set("state", req.State)
    }
    redirectURL.RawQuery = query.Encode()

    // Redirect to client with authorization code
    c.Header().Set("Location", redirectURL.String())
    return c.JSON(http.StatusFound, nil)
}

// storeConsentDecision stores the user's consent decision for future reference
func (h *Handler) storeConsentDecision(ctx context.Context, userID xid.ID, clientID, scope string, granted bool) error {
    // TODO: Implement consent storage
    // For now, this is a placeholder - in a real implementation, you would:
    // 1. Store the consent decision in a database table
    // 2. Include timestamp, scope details, etc.
    // 3. Allow users to revoke consent later
    return nil
}