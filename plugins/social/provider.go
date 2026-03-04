// Package social provides OAuth2 social authentication for AuthSome.
//
// The social plugin handles OAuth2 sign-in flows for external providers
// like Google and GitHub. It manages OAuth connections that link external
// provider accounts to AuthSome users.
//
// Usage:
//
//	p := social.New(social.Config{
//	    Providers: []social.Provider{
//	        social.NewGoogleProvider(social.ProviderConfig{
//	            ClientID:     "your-client-id",
//	            ClientSecret: "your-secret",
//	            RedirectURL:  "https://app.com/api/auth/social/google/callback",
//	        }),
//	        social.NewGitHubProvider(social.ProviderConfig{
//	            ClientID:     "your-client-id",
//	            ClientSecret: "your-secret",
//	            RedirectURL:  "https://app.com/api/auth/social/github/callback",
//	        }),
//	    },
//	})
package social

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"

	"golang.org/x/oauth2"
)

// ──────────────────────────────────────────────────
// Provider interface
// ──────────────────────────────────────────────────

// Provider represents an OAuth2 social authentication provider.
type Provider interface {
	// Name returns the provider's unique identifier (e.g., "google", "github").
	Name() string

	// OAuth2Config returns the OAuth2 configuration for this provider.
	OAuth2Config() *oauth2.Config

	// FetchUser uses an OAuth2 token to retrieve the user's profile
	// from the provider's API.
	FetchUser(ctx context.Context, token *oauth2.Token) (*ProviderUser, error)
}

// ProviderUser is the normalized user profile returned by a social provider.
type ProviderUser struct {
	ProviderUserID string `json:"provider_user_id"`
	Email          string `json:"email"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	AvatarURL      string `json:"avatar_url,omitempty"`
}

// ProviderConfig holds common OAuth2 configuration for a provider.
type ProviderConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// ──────────────────────────────────────────────────
// OAuth Connection (data model)
// ──────────────────────────────────────────────────

// OAuthConnection links an external provider account to an AuthSome user.
type OAuthConnection struct {
	ID             id.OAuthConnectionID `json:"id"`
	AppID          id.AppID             `json:"app_id"`
	UserID         id.UserID            `json:"user_id"`
	Provider       string               `json:"provider"`
	ProviderUserID string               `json:"provider_user_id"`
	Email          string               `json:"email"`
	AccessToken    string               `json:"-"`
	RefreshToken   string               `json:"-"`
	ExpiresAt      time.Time            `json:"expires_at,omitempty"`
	Metadata       map[string]string    `json:"metadata,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// ──────────────────────────────────────────────────
// Store interface for OAuth connections
// ──────────────────────────────────────────────────

// Store persists OAuth connections.
type Store interface {
	CreateOAuthConnection(ctx context.Context, c *OAuthConnection) error
	GetOAuthConnection(ctx context.Context, provider, providerUserID string) (*OAuthConnection, error)
	GetOAuthConnectionsByUserID(ctx context.Context, userID id.UserID) ([]*OAuthConnection, error)
	DeleteOAuthConnection(ctx context.Context, connID id.OAuthConnectionID) error
}
