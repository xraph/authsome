// Package appclientconfig defines per-app client-facing configuration overrides.
// When set, these override the plugin-level defaults for a specific app,
// controlling which auth methods, social providers, and branding the client SDK sees.
// Fields are pointers so nil means "inherit from the plugin defaults".
package appclientconfig

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when no per-app client config exists for the given app.
var ErrNotFound = errors.New("appclientconfig: not found")

// Config holds per-app client configuration overrides.
type Config struct {
	ID    id.AppClientConfigID `json:"id"`
	AppID id.AppID             `json:"app_id"`

	// Auth method overrides (nil = inherit from plugin defaults).
	PasswordEnabled  *bool `json:"password_enabled,omitempty"`
	PasskeyEnabled   *bool `json:"passkey_enabled,omitempty"`
	MagicLinkEnabled *bool `json:"magic_link_enabled,omitempty"`
	MFAEnabled       *bool `json:"mfa_enabled,omitempty"`
	SSOEnabled       *bool `json:"sso_enabled,omitempty"`
	SocialEnabled    *bool `json:"social_enabled,omitempty"`
	WaitlistEnabled           *bool `json:"waitlist_enabled,omitempty"`
	RequireEmailVerification *bool `json:"require_email_verification,omitempty"`

	// Social provider filter. When set, only these providers (from the global
	// plugin-level list) are exposed to the client SDK for this app.
	SocialProviders []string `json:"social_providers,omitempty"`

	// MFA method filter. When set, only these methods are exposed.
	MFAMethods []string `json:"mfa_methods,omitempty"`

	// Branding overrides.
	AppName *string `json:"app_name,omitempty"`
	LogoURL *string `json:"logo_url,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store is the persistence interface for per-app client configuration.
type Store interface {
	// GetAppClientConfig returns the client config override for an app.
	// Returns ErrNotFound if no override is set for this app.
	GetAppClientConfig(ctx context.Context, appID id.AppID) (*Config, error)

	// SetAppClientConfig creates or updates the client config for an app.
	SetAppClientConfig(ctx context.Context, cfg *Config) error

	// DeleteAppClientConfig removes the per-app client config override.
	DeleteAppClientConfig(ctx context.Context, appID id.AppID) error
}
