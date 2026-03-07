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

// DeviceCode represents an OAuth2 device authorization code (RFC 8628).
type DeviceCode struct {
	ID              id.DeviceCodeID `json:"id"`
	DeviceCode      string          `json:"-"`                  // opaque polling token (256-bit hex)
	UserCode        string          `json:"user_code"`          // short human-readable code, e.g. "BCDF-GHJK"
	ClientID        string          `json:"client_id"`          // OAuth2 client_id
	AppID           id.AppID        `json:"app_id"`
	Scopes          []string        `json:"scopes"`
	VerificationURI string          `json:"verification_uri"`   // where the user goes
	ExpiresAt       time.Time       `json:"expires_at"`
	Interval        int             `json:"interval"`           // polling interval in seconds
	Status          string          `json:"status"`             // "pending", "authorized", "denied", "consumed"
	UserID          id.UserID       `json:"user_id,omitempty"`  // set when user authorizes
	LastPolledAt    time.Time       `json:"last_polled_at,omitempty"` // last time the CLI polled for this code
	CreatedAt       time.Time       `json:"created_at"`
}

// Device code status constants.
const (
	DeviceCodeStatusPending    = "pending"
	DeviceCodeStatusAuthorized = "authorized"
	DeviceCodeStatusDenied     = "denied"
	DeviceCodeStatusConsumed   = "consumed"
)

// UserInfo is the OIDC userinfo response.
type UserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Name          string `json:"name,omitempty"`
	Phone         string `json:"phone_number,omitempty"`
}
