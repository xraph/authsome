package middleware

import (
	"github.com/rs/xid"
)

// ContextConfig configures how app and environment context is populated
type ContextConfig struct {
	// DefaultAppID is used when no app ID is found in headers or API key
	// Should be a valid xid string (e.g., "c7ndh411g9k8pdunveeg")
	DefaultAppID string

	// DefaultEnvironmentID is used when no environment ID is found
	// Should be a valid xid string (e.g., "c7ndh411g9k8pdunveeg")
	DefaultEnvironmentID string

	// AppIDHeader is the header name to check for app ID (default: X-App-ID)
	AppIDHeader string

	// EnvironmentIDHeader is the header name to check for environment ID (default: X-Environment-ID)
	EnvironmentIDHeader string

	// AutoDetectFromConfig enables auto-detection of app/environment from AuthSome config
	// When enabled in standalone mode, uses the default app automatically
	AutoDetectFromConfig bool

	// AutoDetectFromAPIKey enables inferring app/environment from verified API key
	// This is the most common pattern - API key contains app and environment context
	AutoDetectFromAPIKey bool
}

// DefaultContextConfig returns a ContextConfig with sensible defaults
func DefaultContextConfig() ContextConfig {
	return ContextConfig{
		AppIDHeader:          "X-App-ID",
		EnvironmentIDHeader:  "X-Environment-ID",
		AutoDetectFromAPIKey: true,  // Most common pattern
		AutoDetectFromConfig: false, // Explicit opt-in for standalone mode
	}
}

// ContextSource indicates where the context value came from
type ContextSource string

const (
	ContextSourceNone       ContextSource = "none"
	ContextSourceExisting   ContextSource = "existing"    // Already in request context
	ContextSourceHeader     ContextSource = "header"      // From HTTP header
	ContextSourceAPIKey     ContextSource = "api_key"     // From verified API key
	ContextSourceDefault    ContextSource = "default"     // From default config
	ContextSourceAutoDetect ContextSource = "auto_detect" // From AuthSome config
)

// ContextResolution tracks how context values were resolved
type ContextResolution struct {
	AppID               xid.ID
	AppIDSource         ContextSource
	EnvironmentID       xid.ID
	EnvironmentIDSource ContextSource
}
