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
	"strings"
	"time"

	"github.com/xraph/authsome/id"

	"golang.org/x/oauth2"
)

// normalizeProviderEmail lowercases and trims a provider email so matching is
// case- and whitespace-insensitive (mirrors user.NormalizeEmail).
func normalizeProviderEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

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

// ProviderEmail is a single email address reported by a social provider,
// along with whether the provider considers it verified and primary.
type ProviderEmail struct {
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
	Primary  bool   `json:"primary"`
}

// ProviderUser is the normalized user profile returned by a social provider.
type ProviderUser struct {
	ProviderUserID string `json:"provider_user_id"`
	// Email is the provider's primary email (kept for backward compatibility).
	Email string `json:"email"`
	// EmailVerified reports whether the provider considers Email verified.
	// Defaults to false when the provider doesn't expose verification — the
	// matching algorithm treats unknown verification as unverified so an
	// unverified address can never silently link to or hijack an account.
	EmailVerified bool `json:"email_verified"`
	// Emails is the full set of addresses the provider exposes (may be empty,
	// in which case Email/EmailVerified is used as a single entry).
	Emails    []ProviderEmail `json:"emails,omitempty"`
	FirstName string          `json:"first_name"`
	LastName  string          `json:"last_name"`
	AvatarURL string          `json:"avatar_url,omitempty"`
}

// VerifiedEmails returns the provider's verified addresses (lowercased,
// de-duplicated), falling back to the primary Email when the provider didn't
// populate Emails. Only verified addresses are returned, so callers can match
// against existing accounts without risking takeover via an unverified email.
func (p *ProviderUser) VerifiedEmails() []ProviderEmail {
	seen := make(map[string]bool)
	var out []ProviderEmail
	add := func(e ProviderEmail) {
		norm := normalizeProviderEmail(e.Email)
		if norm == "" || !e.Verified || seen[norm] {
			return
		}
		seen[norm] = true
		out = append(out, ProviderEmail{Email: norm, Verified: true, Primary: e.Primary})
	}
	for _, e := range p.Emails {
		add(e)
	}
	if len(p.Emails) == 0 {
		add(ProviderEmail{Email: p.Email, Verified: p.EmailVerified, Primary: true})
	}
	return out
}

// AllEmails returns every address the provider exposes (verified or not),
// lowercased and de-duplicated, falling back to the primary Email. Used when
// seeding a brand-new user so their known addresses are recorded.
func (p *ProviderUser) AllEmails() []ProviderEmail {
	seen := make(map[string]bool)
	var out []ProviderEmail
	add := func(e ProviderEmail) {
		norm := normalizeProviderEmail(e.Email)
		if norm == "" || seen[norm] {
			return
		}
		seen[norm] = true
		out = append(out, ProviderEmail{Email: norm, Verified: e.Verified, Primary: e.Primary})
	}
	for _, e := range p.Emails {
		add(e)
	}
	if len(p.Emails) == 0 {
		add(ProviderEmail{Email: p.Email, Verified: p.EmailVerified, Primary: true})
	}
	return out
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
	// UpdateOAuthConnection persists changes to an existing connection
	// (refreshed access/refresh tokens, expiry, email).
	UpdateOAuthConnection(ctx context.Context, c *OAuthConnection) error
	DeleteOAuthConnection(ctx context.Context, connID id.OAuthConnectionID) error
}
