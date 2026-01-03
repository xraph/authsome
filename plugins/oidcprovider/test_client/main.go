package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL = "http://localhost:3001"
	// For testing, we'll use a pre-registered client or register one dynamically
)

// OIDC Discovery response
type DiscoveryDocument struct {
	Issuer                string   `json:"issuer"`
	AuthorizationEndpoint string   `json:"authorization_endpoint"`
	TokenEndpoint         string   `json:"token_endpoint"`
	UserInfoEndpoint      string   `json:"userinfo_endpoint"`
	JwksURI               string   `json:"jwks_uri"`
	ScopesSupported       []string `json:"scopes_supported"`
}

// Client registration request
type ClientRegistrationRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	ApplicationType         string   `json:"application_type"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	RequirePKCE             bool     `json:"require_pkce"`
}

// Client registration response
type ClientRegistrationResponse struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// Token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope"`
}

// UserInfo response
type UserInfoResponse struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

// Introspection response
type IntrospectionResponse struct {
	Active    bool     `json:"active"`
	Scope     string   `json:"scope"`
	ClientID  string   `json:"client_id"`
	Username  string   `json:"username"`
	TokenType string   `json:"token_type"`
	Exp       int64    `json:"exp"`
	Iat       int64    `json:"iat"`
	Sub       string   `json:"sub"`
	Aud       []string `json:"aud"`
}

// TestClient represents an OIDC test client
type TestClient struct {
	baseURL      string
	clientID     string
	clientSecret string
	redirectURI  string
	discovery    *DiscoveryDocument
}

// NewTestClient creates a new test client
func NewTestClient(baseURL, redirectURI string) *TestClient {
	return &TestClient{
		baseURL:     baseURL,
		redirectURI: redirectURI,
	}
}

// DiscoverEndpoints fetches the OIDC discovery document
func (c *TestClient) DiscoverEndpoints() error {
	log.Println("📡 Discovering OIDC endpoints...")
	
	resp, err := http.Get(c.baseURL + "/.well-known/openid-configuration")
	if err != nil {
		return fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discovery endpoint returned %d: %s", resp.StatusCode, string(body))
	}
	
	var discovery DiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return fmt.Errorf("failed to decode discovery document: %w", err)
	}
	
	c.discovery = &discovery
	
	log.Printf("✅ Discovery successful!")
	log.Printf("   Issuer: %s", discovery.Issuer)
	log.Printf("   Authorization: %s", discovery.AuthorizationEndpoint)
	log.Printf("   Token: %s", discovery.TokenEndpoint)
	log.Printf("   UserInfo: %s", discovery.UserInfoEndpoint)
	log.Printf("   JWKS: %s", discovery.JwksURI)
	log.Printf("   Scopes: %v", discovery.ScopesSupported)
	
	return nil
}

// RegisterClient registers a new OAuth client
func (c *TestClient) RegisterClient(adminToken string) error {
	log.Println("📝 Registering OAuth client...")
	
	reqBody := ClientRegistrationRequest{
		ClientName:              "Test OIDC Client",
		RedirectURIs:            []string{c.redirectURI},
		ApplicationType:         "spa",
		TokenEndpointAuthMethod: "none",
		RequirePKCE:             true,
	}
	
	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", c.baseURL+"/oauth2/register", bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/json")
	if adminToken != "" {
		req.Header.Set("Authorization", "Bearer "+adminToken)
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register client: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("client registration failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var result ClientRegistrationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to decode registration response: %w", err)
	}
	
	c.clientID = result.ClientID
	c.clientSecret = result.ClientSecret
	
	log.Printf("✅ Client registered successfully!")
	log.Printf("   Client ID: %s", c.clientID)
	if c.clientSecret != "" {
		log.Printf("   Client Secret: %s", c.clientSecret)
	}
	
	return nil
}

// UseExistingClient configures the client to use existing credentials
func (c *TestClient) UseExistingClient(clientID, clientSecret string) {
	c.clientID = clientID
	c.clientSecret = clientSecret
	log.Printf("📌 Using existing client: %s", clientID)
}

// GeneratePKCE generates PKCE challenge and verifier
func GeneratePKCE() (verifier, challenge string, err error) {
	// Generate random verifier
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	verifier = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)
	
	// Generate S256 challenge
	hash := sha256.Sum256([]byte(verifier))
	challenge = base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
	
	return verifier, challenge, nil
}

// BuildAuthorizationURL creates the authorization URL with PKCE
func (c *TestClient) BuildAuthorizationURL() (authURL, state, codeVerifier string, err error) {
	if c.discovery == nil {
		return "", "", "", fmt.Errorf("discovery not completed")
	}
	
	// Generate PKCE
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		return "", "", "", err
	}
	
	// Generate state
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state = base64.URLEncoding.EncodeToString(stateBytes)
	
	// Build authorization URL
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("redirect_uri", c.redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", "openid profile email")
	params.Set("state", state)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	
	authURL = c.discovery.AuthorizationEndpoint + "?" + params.Encode()
	
	return authURL, state, verifier, nil
}

// ExchangeCodeForTokens exchanges authorization code for tokens
func (c *TestClient) ExchangeCodeForTokens(code, codeVerifier string) (*TokenResponse, error) {
	log.Println("🔄 Exchanging authorization code for tokens...")
	
	if c.discovery == nil {
		return nil, fmt.Errorf("discovery not completed")
	}
	
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", c.redirectURI)
	data.Set("client_id", c.clientID)
	data.Set("code_verifier", codeVerifier)
	
	if c.clientSecret != "" {
		data.Set("client_secret", c.clientSecret)
	}
	
	resp, err := http.Post(
		c.discovery.TokenEndpoint,
		"application/x-www-form-urlencoded",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var tokens TokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	
	log.Printf("✅ Tokens received successfully!")
	log.Printf("   Access Token: %s...", tokens.AccessToken[:50])
	log.Printf("   Token Type: %s", tokens.TokenType)
	log.Printf("   Expires In: %d seconds", tokens.ExpiresIn)
	if tokens.RefreshToken != "" {
		log.Printf("   Refresh Token: %s...", tokens.RefreshToken[:50])
	}
	if tokens.IDToken != "" {
		log.Printf("   ID Token: %s...", tokens.IDToken[:50])
	}
	log.Printf("   Scope: %s", tokens.Scope)
	
	return &tokens, nil
}

// GetUserInfo retrieves user information using access token
func (c *TestClient) GetUserInfo(accessToken string) (*UserInfoResponse, error) {
	log.Println("👤 Fetching user info...")
	
	if c.discovery == nil {
		return nil, fmt.Errorf("discovery not completed")
	}
	
	req, err := http.NewRequest("GET", c.discovery.UserInfoEndpoint, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Authorization", "Bearer "+accessToken)
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned %d: %s", resp.StatusCode, string(body))
	}
	
	var userInfo UserInfoResponse
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode userinfo response: %w", err)
	}
	
	log.Printf("✅ User info retrieved!")
	log.Printf("   Subject: %s", userInfo.Sub)
	log.Printf("   Email: %s", userInfo.Email)
	log.Printf("   Email Verified: %v", userInfo.EmailVerified)
	log.Printf("   Name: %s", userInfo.Name)
	
	return &userInfo, nil
}

// IntrospectToken introspects an access token (requires confidential client)
func (c *TestClient) IntrospectToken(token string) (*IntrospectionResponse, error) {
	log.Println("🔍 Introspecting token...")
	
	data := url.Values{}
	data.Set("token", token)
	data.Set("token_type_hint", "access_token")
	
	req, err := http.NewRequest(
		"POST",
		c.baseURL+"/oauth2/introspect",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Use HTTP Basic Auth if we have a client secret
	if c.clientSecret != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
		req.Header.Set("Authorization", "Basic "+auth)
	} else {
		// For public clients, send client_id in body
		data.Set("client_id", c.clientID)
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect token: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("introspection failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var result IntrospectionResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode introspection response: %w", err)
	}
	
	log.Printf("✅ Token introspection complete!")
	log.Printf("   Active: %v", result.Active)
	if result.Active {
		log.Printf("   Client ID: %s", result.ClientID)
		log.Printf("   Scope: %s", result.Scope)
		log.Printf("   Subject: %s", result.Sub)
		log.Printf("   Expires: %s", time.Unix(result.Exp, 0))
	}
	
	return &result, nil
}

// RevokeToken revokes a token
func (c *TestClient) RevokeToken(token string) error {
	log.Println("🗑️  Revoking token...")
	
	data := url.Values{}
	data.Set("token", token)
	
	req, err := http.NewRequest(
		"POST",
		c.baseURL+"/oauth2/revoke",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return err
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Use HTTP Basic Auth if we have a client secret
	if c.clientSecret != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
		req.Header.Set("Authorization", "Basic "+auth)
	} else {
		data.Set("client_id", c.clientID)
	}
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revocation failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	log.Printf("✅ Token revoked successfully!")
	
	return nil
}

func main() {
	log.Println("🚀 OIDC Provider Test Client")
	log.Println("================================")
	log.Println()
	
	// Create test client
	redirectURI := "http://localhost:8080/callback"
	client := NewTestClient(baseURL, redirectURI)
	
	// Step 1: Discover endpoints
	if err := client.DiscoverEndpoints(); err != nil {
		log.Fatalf("❌ Discovery failed: %v", err)
	}
	log.Println()
	
	// Step 2: Register client (or use existing)
	// For testing, you can either:
	// A. Register a new client (requires admin token)
	// B. Use pre-registered client credentials
	
	// Option A: Register new client
	// adminToken := "your-admin-token-here"
	// if err := client.RegisterClient(adminToken); err != nil {
	// 	log.Printf("⚠️  Client registration failed: %v", err)
	// 	log.Println("   Using manual client configuration...")
	// 	client.UseExistingClient("your-client-id", "")
	// }
	
	// Option B: Use existing client
	client.UseExistingClient("client_test123", "")
	log.Println()
	
	// Step 3: Build authorization URL
	authURL, state, codeVerifier, err := client.BuildAuthorizationURL()
	if err != nil {
		log.Fatalf("❌ Failed to build authorization URL: %v", err)
	}
	
	log.Println("🔗 Authorization URL:")
	log.Println(authURL)
	log.Println()
	log.Println("📋 Manual Steps:")
	log.Println("1. Open the above URL in a browser")
	log.Println("2. Log in if not already authenticated")
	log.Println("3. Grant consent if prompted")
	log.Println("4. Copy the authorization code from the redirect URL")
	log.Println()
	
	// Wait for user input

	var authCode string
	fmt.Scanln(&authCode)
	
	if authCode == "" {
		log.Println("⚠️  No authorization code provided. Testing discovery and URL generation only.")
		return
	}
	
	// Step 4: Exchange code for tokens
	tokens, err := client.ExchangeCodeForTokens(authCode, codeVerifier)
	if err != nil {
		log.Fatalf("❌ Token exchange failed: %v", err)
	}
	log.Println()
	
	// Step 5: Get user info
	userInfo, err := client.GetUserInfo(tokens.AccessToken)
	if err != nil {
		log.Printf("⚠️  Failed to get user info: %v", err)
	} else {
		log.Println()
	}
	
	// Step 6: Introspect token (if using confidential client)
	if client.clientSecret != "" {
		introspection, err := client.IntrospectToken(tokens.AccessToken)
		if err != nil {
			log.Printf("⚠️  Token introspection failed: %v", err)
		} else {
			log.Println()
		}
		_ = introspection
	}
	
	// Step 7: Revoke token
	log.Println("⏳ Waiting 3 seconds before revocation...")
	time.Sleep(3 * time.Second)
	
	if err := client.RevokeToken(tokens.AccessToken); err != nil {
		log.Printf("⚠️  Token revocation failed: %v", err)
	}
	log.Println()
	
	// Step 8: Verify token is revoked by trying to use it
	log.Println("🔍 Verifying token revocation...")
	_, err = client.GetUserInfo(tokens.AccessToken)
	if err != nil {
		log.Println("✅ Token is properly revoked (UserInfo failed as expected)")
	} else {
		log.Println("⚠️  Token still works after revocation!")
	}
	
	log.Println()
	log.Println("================================")
	log.Println("✨ OIDC flow test complete!")
}

