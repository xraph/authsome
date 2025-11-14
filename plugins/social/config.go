package social

import (
	"github.com/xraph/authsome/plugins/social/providers"
)

// Config holds the configuration for social auth providers
type Config struct {
	// Base URL for OAuth callbacks (e.g., "https://example.com")
	BaseURL string `json:"baseUrl" yaml:"baseUrl"`

	// Providers configuration
	Providers ProvidersConfig `json:"providers" yaml:"providers"`

	// Advanced options
	AllowAccountLinking  bool `json:"allowAccountLinking" yaml:"allowAccountLinking"`   // Allow linking multiple providers to one user
	AutoCreateUser       bool `json:"autoCreateUser" yaml:"autoCreateUser"`             // Auto-create user on OAuth sign-in
	RequireEmailVerified bool `json:"requireEmailVerified" yaml:"requireEmailVerified"` // Require email verification from provider
	TrustEmailVerified   bool `json:"trustEmailVerified" yaml:"trustEmailVerified"`     // Trust email verification from provider
}

// ProvidersConfig holds configuration for each provider
type ProvidersConfig struct {
	Google    *providers.ProviderConfig `json:"google,omitempty" yaml:"google,omitempty"`
	GitHub    *providers.ProviderConfig `json:"github,omitempty" yaml:"github,omitempty"`
	Microsoft *providers.ProviderConfig `json:"microsoft,omitempty" yaml:"microsoft,omitempty"`
	Apple     *providers.ProviderConfig `json:"apple,omitempty" yaml:"apple,omitempty"`
	Facebook  *providers.ProviderConfig `json:"facebook,omitempty" yaml:"facebook,omitempty"`
	Discord   *providers.ProviderConfig `json:"discord,omitempty" yaml:"discord,omitempty"`
	Twitter   *providers.ProviderConfig `json:"twitter,omitempty" yaml:"twitter,omitempty"`
	LinkedIn  *providers.ProviderConfig `json:"linkedin,omitempty" yaml:"linkedin,omitempty"`
	Spotify   *providers.ProviderConfig `json:"spotify,omitempty" yaml:"spotify,omitempty"`
	Twitch    *providers.ProviderConfig `json:"twitch,omitempty" yaml:"twitch,omitempty"`
	Dropbox   *providers.ProviderConfig `json:"dropbox,omitempty" yaml:"dropbox,omitempty"`
	GitLab    *providers.ProviderConfig `json:"gitlab,omitempty" yaml:"gitlab,omitempty"`
	LINE      *providers.ProviderConfig `json:"line,omitempty" yaml:"line,omitempty"`
	Reddit    *providers.ProviderConfig `json:"reddit,omitempty" yaml:"reddit,omitempty"`
	Slack     *providers.ProviderConfig `json:"slack,omitempty" yaml:"slack,omitempty"`
	Bitbucket *providers.ProviderConfig `json:"bitbucket,omitempty" yaml:"bitbucket,omitempty"`
	Notion    *providers.ProviderConfig `json:"notion,omitempty" yaml:"notion,omitempty"`
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		BaseURL:              "http://localhost:3000",
		AllowAccountLinking:  true,
		AutoCreateUser:       true,
		RequireEmailVerified: false,
		Providers:            ProvidersConfig{},
	}
}
