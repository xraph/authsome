package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Provider defines the interface for OAuth providers
type Provider interface {
	// ID returns the provider identifier (e.g., "google", "github")
	ID() string

	// Name returns the human-readable provider name
	Name() string

	// GetOAuth2Config returns the OAuth2 configuration
	GetOAuth2Config() *oauth2.Config

	// GetUserInfo fetches user information from the provider
	GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error)

	// GetScopes returns the default scopes for this provider
	GetScopes() []string
}

// UserInfo represents standardized user information from OAuth providers
type UserInfo struct {
	ID            string                 // Provider's user ID
	Email         string                 // User email
	EmailVerified bool                   // Whether email is verified
	Name          string                 // Full name
	FirstName     string                 // First name
	LastName      string                 // Last name
	Avatar        string                 // Profile picture URL
	Username      string                 // Username (if available)
	Raw           map[string]interface{} // Raw provider response
}

// BaseProvider provides common OAuth2 functionality
type BaseProvider struct {
	id           string
	name         string
	authURL      string
	tokenURL     string
	userInfoURL  string
	clientID     string
	clientSecret string
	redirectURL  string
	scopes       []string
	oauth2Config *oauth2.Config
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(id, name, authURL, tokenURL, userInfoURL, clientID, clientSecret, redirectURL string, scopes []string) *BaseProvider {
	bp := &BaseProvider{
		id:           id,
		name:         name,
		authURL:      authURL,
		tokenURL:     tokenURL,
		userInfoURL:  userInfoURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		scopes:       scopes,
	}

	bp.oauth2Config = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	return bp
}

func (bp *BaseProvider) ID() string {
	return bp.id
}

func (bp *BaseProvider) Name() string {
	return bp.name
}

func (bp *BaseProvider) GetOAuth2Config() *oauth2.Config {
	return bp.oauth2Config
}

func (bp *BaseProvider) GetScopes() []string {
	return bp.scopes
}

// FetchJSON is a helper to fetch and decode JSON from an API endpoint
func FetchJSON(ctx context.Context, client *http.Client, url string, v interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// PostForm is a helper for POST requests with form data
func PostForm(ctx context.Context, client *http.Client, url string, data url.Values, v interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to post data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// ProviderConfig holds configuration for a social provider
type ProviderConfig struct {
	ClientID     string   `json:"clientId" yaml:"clientId"`
	ClientSecret string   `json:"clientSecret" yaml:"clientSecret"`
	RedirectURL  string   `json:"redirectUrl" yaml:"redirectUrl"`
	CallbackURL  string   `json:"callbackUrl" yaml:"callbackUrl"`
	Scopes       []string `json:"scopes" yaml:"scopes"`
	Enabled      bool     `json:"enabled" yaml:"enabled"`

	// Advanced options (provider-specific)
	AccessType string `json:"accessType" yaml:"accessType"` // For Google: "offline" for refresh tokens
	Prompt     string `json:"prompt" yaml:"prompt"`         // For Google: "select_account consent"
}

// TokenResponse represents a standardized OAuth token response
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	IDToken      string    `json:"id_token,omitempty"` // For OIDC providers
	ExpiresAt    time.Time `json:"-"`                  // Calculated expiration time
}

// CalculateExpiration calculates the token expiration time
func (tr *TokenResponse) CalculateExpiration() {
	if tr.ExpiresIn > 0 {
		tr.ExpiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	}
}
