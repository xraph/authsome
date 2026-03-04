package bridge

import (
	"context"
	"errors"
)

// ErrVaultNotAvailable is returned when vault bridge is not configured.
var ErrVaultNotAvailable = errors.New("bridge: vault not available (standalone mode)")

// Vault is a local secrets / feature-flag / config interface. Implementations
// retrieve secrets, check feature flags, and read configuration from a vault
// backend (e.g., the vault extension).
type Vault interface {
	// GetSecret returns a decrypted secret value by key.
	GetSecret(ctx context.Context, key string) ([]byte, error)

	// SetSecret creates or updates an encrypted secret.
	SetSecret(ctx context.Context, key string, value []byte) error

	// IsFeatureEnabled evaluates a boolean feature flag.
	IsFeatureEnabled(ctx context.Context, flag string) bool

	// GetConfig returns a configuration value by key.
	GetConfig(ctx context.Context, key string) (string, error)
}

// VaultFunc is an adapter to use a plain function as a GetSecret call.
// For full Vault implementations, use the vaultadapter package.
type VaultFunc func(ctx context.Context, key string) ([]byte, error)
