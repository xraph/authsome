package oauth2provider

import (
	"time"

	"github.com/xraph/authsome/id"
)

// OAuth2Client represents a registered OAuth2 client application.
type OAuth2Client struct {
	ID           id.OAuth2ClientID `json:"id"`
	AppID        id.AppID          `json:"app_id"`
	Name         string            `json:"name"`
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"-"` // Hashed; never exposed
	RedirectURIs []string          `json:"redirect_uris"`
	Scopes       []string          `json:"scopes"`
	GrantTypes   []string          `json:"grant_types"` // "authorization_code", "client_credentials"
	Public       bool              `json:"public"`       // Public clients (SPAs, mobile) don't have a secret
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// AuthorizationCode represents a short-lived authorization code.
type AuthorizationCode struct {
	ID                  id.AuthCodeID `json:"id"`
	Code                string        `json:"-"`
	ClientID            string        `json:"client_id"`
	UserID              id.UserID     `json:"user_id"`
	AppID               id.AppID      `json:"app_id"`
	RedirectURI         string        `json:"redirect_uri"`
	Scopes              []string      `json:"scopes"`
	CodeChallenge       string        `json:"-"` // PKCE
	CodeChallengeMethod string        `json:"-"` // "S256" or "plain"
	ExpiresAt           time.Time     `json:"expires_at"`
	Consumed            bool          `json:"consumed"`
	CreatedAt           time.Time     `json:"created_at"`
}

// TokenResponse is the OAuth2 token endpoint response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// UserInfo is the OIDC userinfo response.
type UserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	Phone         string `json:"phone_number,omitempty"`
}
