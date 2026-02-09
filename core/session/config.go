package session

import (
	"encoding/json"
	"net/http"
)

// CookieConfig represents the configuration for session cookies.
type CookieConfig struct {
	Enabled  bool   `json:"enabled"`            // Enable/disable cookie setting
	Name     string `json:"name"`               // Cookie name (default: "authsome_session")
	Domain   string `json:"domain,omitempty"`   // Cookie domain
	Path     string `json:"path"`               // Cookie path (default: "/")
	Secure   *bool  `json:"secure,omitempty"`   // Secure flag (nil = auto-detect based on TLS)
	HttpOnly bool   `json:"httpOnly"`           // HttpOnly flag (default: true)
	SameSite string `json:"sameSite,omitempty"` // SameSite: "Strict", "Lax", "None" (default: "Lax")
	MaxAge   *int   `json:"maxAge,omitempty"`   // MaxAge in seconds (nil = use session duration)
}

// DefaultCookieConfig returns a cookie configuration with sensible defaults.
func DefaultCookieConfig() CookieConfig {
	return CookieConfig{
		Enabled:  false, // Opt-in by default
		Name:     "authsome_session",
		Path:     "/",
		HttpOnly: true,
		SameSite: "Lax",
	}
}

// Merge applies per-app overrides to the base config and returns a new merged config
// The override config takes precedence over the base config for non-zero values
// Special handling for boolean fields:
// - Enabled: Only override if override.Enabled is true (can't distinguish false from unset)
// - HttpOnly: Only override if override.HttpOnly is false (since default is true).
func (c *CookieConfig) Merge(override *CookieConfig) *CookieConfig {
	if override == nil {
		// Return a copy of the base config
		merged := *c

		return &merged
	}

	// Start with a copy of the base config
	merged := *c

	// Apply overrides for each field if the override has a non-zero value
	// For booleans, we can only detect explicit true (Go zero value is false)

	// Enabled: Only override if explicitly set to true
	// This prevents app metadata with empty/default config from disabling cookies
	// Apps that want to disable must be handled differently (e.g., explicit disable flag)
	if override.Enabled {
		merged.Enabled = true
	}
	// Note: If base.Enabled is true and override.Enabled is false, we KEEP base.Enabled = true
	// This is intentional - app metadata shouldn't accidentally disable globally-enabled cookies

	// Name: Override if not empty
	if override.Name != "" {
		merged.Name = override.Name
	}

	// Domain: Override if not empty
	if override.Domain != "" {
		merged.Domain = override.Domain
	}

	// Path: Override if not empty
	if override.Path != "" {
		merged.Path = override.Path
	}

	// Secure: Override if explicitly set (pointer allows distinguishing nil from false)
	if override.Secure != nil {
		merged.Secure = override.Secure
	}

	// HttpOnly: Only override if explicitly set to false (default is true)
	// This allows apps to explicitly disable HttpOnly if needed
	if !override.HttpOnly && c.HttpOnly {
		// Only if base had HttpOnly=true and override explicitly sets false
		// But since we can't distinguish, keep base value
		// merged.HttpOnly = override.HttpOnly
	}
	// Keep base HttpOnly value - safer default

	// SameSite: Override if not empty
	if override.SameSite != "" {
		merged.SameSite = override.SameSite
	}

	// MaxAge: Override if explicitly set
	if override.MaxAge != nil {
		merged.MaxAge = override.MaxAge
	}

	return &merged
}

// ParseSameSite converts a string to http.SameSite constant
// Returns Lax as default for invalid values.
func ParseSameSite(s string) http.SameSite {
	switch s {
	case "Strict", "strict":
		return http.SameSiteStrictMode
	case "Lax", "lax", "":
		return http.SameSiteLaxMode
	case "None", "none":
		return http.SameSiteNoneMode
	default:
		// Invalid value, default to Lax
		return http.SameSiteLaxMode
	}
}

// UnmarshalCookieConfigFromJSON unmarshals cookie config from JSON bytes
// This is a helper for extracting cookie config from app metadata.
func UnmarshalCookieConfigFromJSON(data []byte) (*CookieConfig, error) {
	var config CookieConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
