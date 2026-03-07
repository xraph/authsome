// Package apikey defines the API key domain entity and its store interface.
// API keys enable machine-to-machine authentication, providing an alternative
// to session-based auth for programmatic access.
package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when an API key cannot be found.
var ErrNotFound = errors.New("apikey: not found")

// APIKey represents an API key for programmatic access.
type APIKey struct {
	ID         id.APIKeyID      `json:"id"`
	AppID      id.AppID         `json:"app_id"`
	EnvID      id.EnvironmentID `json:"env_id"`
	UserID     id.UserID        `json:"user_id"`
	Name       string           `json:"name"`
	KeyHash         string           `json:"-"`
	KeyPrefix       string           `json:"key_prefix"`
	PublicKey       string           `json:"public_key,omitempty"`
	PublicKeyPrefix string           `json:"public_key_prefix,omitempty"`
	Scopes          []string         `json:"scopes,omitempty"`
	ExpiresAt  *time.Time       `json:"expires_at,omitempty"`
	LastUsedAt *time.Time       `json:"last_used_at,omitempty"`
	Revoked    bool             `json:"revoked"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// IsExpired returns true if the API key has expired.
func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}

// IsValid returns true if the API key is neither revoked nor expired.
func (k *APIKey) IsValid() bool {
	return !k.Revoked && !k.IsExpired()
}

// Store is the persistence interface for API keys.
type Store interface {
	// CreateAPIKey stores a new API key.
	CreateAPIKey(ctx context.Context, key *APIKey) error

	// GetAPIKey returns an API key by ID.
	GetAPIKey(ctx context.Context, keyID id.APIKeyID) (*APIKey, error)

	// GetAPIKeyByPrefix returns an API key by its prefix.
	// Used during authentication to look up the key.
	GetAPIKeyByPrefix(ctx context.Context, appID id.AppID, prefix string) (*APIKey, error)

	// GetAPIKeyByPublicKey returns an API key by its public key.
	GetAPIKeyByPublicKey(ctx context.Context, appID id.AppID, publicKey string) (*APIKey, error)

	// UpdateAPIKey updates an existing API key.
	UpdateAPIKey(ctx context.Context, key *APIKey) error

	// DeleteAPIKey permanently deletes an API key.
	DeleteAPIKey(ctx context.Context, keyID id.APIKeyID) error

	// ListAPIKeysByApp lists all API keys for a given app.
	ListAPIKeysByApp(ctx context.Context, appID id.AppID) ([]*APIKey, error)

	// ListAPIKeysByUser lists all API keys for a given user within an app.
	ListAPIKeysByUser(ctx context.Context, appID id.AppID, userID id.UserID) ([]*APIKey, error)
}
