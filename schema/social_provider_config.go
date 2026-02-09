package schema

import (
	"slices"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// SocialProviderConfig stores OAuth provider configuration per app/environment
// This enables dashboard-based configuration of social providers instead of code-only config.
type SocialProviderConfig struct {
	bun.BaseModel `bun:"table:social_provider_configs,alias:spc"`

	ID        xid.ID    `bun:"id,pk,type:varchar(20)"                                json:"id"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	// Multi-tenant scoping: App → Environment
	AppID         xid.ID `bun:"app_id,notnull,type:varchar(20)"         json:"appId"`         // Platform tenant (required)
	EnvironmentID xid.ID `bun:"environment_id,notnull,type:varchar(20)" json:"environmentId"` // Environment within app (required)

	// Provider identification
	ProviderName string `bun:"provider_name,notnull" json:"providerName"` // google, github, microsoft, apple, facebook, discord, twitter, linkedin, spotify, twitch, dropbox, gitlab, line, reddit, slack, bitbucket, notion

	// OAuth credentials
	ClientID     string `bun:"client_id,notnull"     json:"clientId"`              // OAuth client ID
	ClientSecret string `bun:"client_secret,notnull" json:"-"`                     // OAuth client secret (encrypted, never exposed in JSON)
	RedirectURL  string `bun:"redirect_url"          json:"redirectUrl,omitempty"` // Custom redirect URL (optional, defaults to system URL)

	// OAuth scopes
	Scopes []string `bun:"scopes,type:jsonb" json:"scopes"` // OAuth scopes to request

	// Status
	IsEnabled bool `bun:"is_enabled,notnull,default:false" json:"isEnabled"` // Whether provider is active for this environment

	// Advanced provider-specific configuration
	// Examples: accessType for Google ("offline"), prompt settings, custom endpoints
	AdvancedConfig map[string]any `bun:"advanced_config,type:jsonb" json:"advancedConfig,omitempty"`

	// Metadata for UI/tracking
	DisplayName string `bun:"display_name" json:"displayName,omitempty"` // Custom display name (optional)
	Description string `bun:"description"  json:"description,omitempty"` // Admin notes/description

	// Soft delete
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"-"`

	// Relations
	App         *App         `bun:"rel:belongs-to,join:app_id=id"         json:"app,omitempty"`
	Environment *Environment `bun:"rel:belongs-to,join:environment_id=id" json:"environment,omitempty"`
}

// SupportedProviders returns the list of all supported OAuth provider names.
var SupportedProviders = []string{
	"google",
	"github",
	"microsoft",
	"apple",
	"facebook",
	"discord",
	"twitter",
	"linkedin",
	"spotify",
	"twitch",
	"dropbox",
	"gitlab",
	"line",
	"reddit",
	"slack",
	"bitbucket",
	"notion",
}

// ProviderDisplayNames maps provider names to human-readable display names.
var ProviderDisplayNames = map[string]string{
	"google":    "Google",
	"github":    "GitHub",
	"microsoft": "Microsoft",
	"apple":     "Apple",
	"facebook":  "Facebook",
	"discord":   "Discord",
	"twitter":   "Twitter / X",
	"linkedin":  "LinkedIn",
	"spotify":   "Spotify",
	"twitch":    "Twitch",
	"dropbox":   "Dropbox",
	"gitlab":    "GitLab",
	"line":      "LINE",
	"reddit":    "Reddit",
	"slack":     "Slack",
	"bitbucket": "Bitbucket",
	"notion":    "Notion",
}

// ProviderDefaultScopes maps provider names to their default OAuth scopes.
var ProviderDefaultScopes = map[string][]string{
	"google":    {"openid", "email", "profile"},
	"github":    {"user:email", "read:user"},
	"microsoft": {"openid", "email", "profile", "User.Read"},
	"apple":     {"name", "email"},
	"facebook":  {"email", "public_profile"},
	"discord":   {"identify", "email"},
	"twitter":   {"users.read", "tweet.read"},
	"linkedin":  {"openid", "profile", "email"},
	"spotify":   {"user-read-email", "user-read-private"},
	"twitch":    {"user:read:email"},
	"dropbox":   {"account_info.read"},
	"gitlab":    {"read_user", "openid", "email"},
	"line":      {"profile", "openid", "email"},
	"reddit":    {"identity"},
	"slack":     {"users:read", "users:read.email"},
	"bitbucket": {"account", "email"},
	"notion":    {},
}

// IsValidProvider checks if the given provider name is supported.
func IsValidProvider(name string) bool {

	return slices.Contains(SupportedProviders, name)
}

// GetProviderDisplayName returns the display name for a provider.
func GetProviderDisplayName(name string) string {
	if displayName, ok := ProviderDisplayNames[name]; ok {
		return displayName
	}

	return name
}

// GetProviderDefaultScopes returns the default scopes for a provider.
func GetProviderDefaultScopes(name string) []string {
	if scopes, ok := ProviderDefaultScopes[name]; ok {
		return scopes
	}

	return []string{}
}

// MaskClientSecret returns a masked version of the client secret for display.
func (c *SocialProviderConfig) MaskClientSecret() string {
	if len(c.ClientSecret) <= 8 {
		return "••••••••"
	}

	return "••••••••" + c.ClientSecret[len(c.ClientSecret)-4:]
}

// HasCustomRedirectURL returns true if a custom redirect URL is configured.
func (c *SocialProviderConfig) HasCustomRedirectURL() bool {
	return c.RedirectURL != ""
}

// GetEffectiveScopes returns the configured scopes or defaults if empty.
func (c *SocialProviderConfig) GetEffectiveScopes() []string {
	if len(c.Scopes) > 0 {
		return c.Scopes
	}

	return GetProviderDefaultScopes(c.ProviderName)
}

// GetDisplayName returns the display name or provider name if not set.
func (c *SocialProviderConfig) GetDisplayName() string {
	if c.DisplayName != "" {
		return c.DisplayName
	}

	return GetProviderDisplayName(c.ProviderName)
}
