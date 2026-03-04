// Package appsessionconfig defines per-app session configuration overrides.
// When set, these override the global engine SessionConfig for a specific app.
// Fields are pointers so nil means "inherit from the global/environment config".
package appsessionconfig

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when no per-app session config exists for the given app.
var ErrNotFound = errors.New("appsessionconfig: not found")

// Config holds per-app session configuration overrides.
type Config struct {
	ID    id.AppSessionConfigID `json:"id"`
	AppID id.AppID              `json:"app_id"`

	// Token behavior overrides (nil = inherit from global).
	TokenTTLSeconds        *int  `json:"token_ttl_seconds,omitempty"`
	RefreshTokenTTLSeconds *int  `json:"refresh_token_ttl_seconds,omitempty"`
	MaxActiveSessions      *int  `json:"max_active_sessions,omitempty"`
	RotateRefreshToken     *bool `json:"rotate_refresh_token,omitempty"`

	// Session binding overrides (nil = inherit from global).
	BindToIP     *bool `json:"bind_to_ip,omitempty"`
	BindToDevice *bool `json:"bind_to_device,omitempty"`

	// Token format: "opaque" (default) or "jwt". Empty = inherit.
	TokenFormat string `json:"token_format,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ApplyTo applies this per-app config on top of a base account.SessionConfig.
// Only non-nil fields override the base values.
func (c *Config) ApplyTo(base *account.SessionConfig) {
	if c == nil {
		return
	}
	if c.TokenTTLSeconds != nil {
		base.TokenTTL = time.Duration(*c.TokenTTLSeconds) * time.Second
	}
	if c.RefreshTokenTTLSeconds != nil {
		base.RefreshTokenTTL = time.Duration(*c.RefreshTokenTTLSeconds) * time.Second
	}
	if c.MaxActiveSessions != nil {
		base.MaxActiveSessions = *c.MaxActiveSessions
	}
	if c.RotateRefreshToken != nil {
		base.RotateRefreshToken = *c.RotateRefreshToken
	}
}

// Store is the persistence interface for per-app session configuration.
type Store interface {
	// GetAppSessionConfig returns the session config override for an app.
	// Returns ErrNotFound if no override is set for this app.
	GetAppSessionConfig(ctx context.Context, appID id.AppID) (*Config, error)

	// SetAppSessionConfig creates or updates the session config for an app.
	SetAppSessionConfig(ctx context.Context, cfg *Config) error

	// DeleteAppSessionConfig removes the per-app session config override.
	DeleteAppSessionConfig(ctx context.Context, appID id.AppID) error
}
