// Package vaultadapter adapts the vault extension's services to the authsome
// bridge.Vault interface.
package vaultadapter

import (
	"context"

	"github.com/xraph/authsome/bridge"

	vaultconfig "github.com/xraph/vault/config"
	vaultflag "github.com/xraph/vault/flag"
	vaultsecret "github.com/xraph/vault/secret"
)

// Adapter implements bridge.Vault by delegating to the vault extension's
// secret, flag, and config services.
type Adapter struct {
	secrets *vaultsecret.Service
	flags   *vaultflag.Service
	config  *vaultconfig.Service
	appID   string
}

// Compile-time check.
var _ bridge.Vault = (*Adapter)(nil)

// Option configures the adapter.
type Option func(*Adapter)

// WithAppID sets the app ID scope for vault operations.
func WithAppID(appID string) Option {
	return func(a *Adapter) { a.appID = appID }
}

// New creates a new vault adapter. All service parameters are optional;
// nil services return bridge.ErrVaultNotAvailable for their methods.
func New(secrets *vaultsecret.Service, flags *vaultflag.Service, config *vaultconfig.Service, opts ...Option) *Adapter {
	a := &Adapter{
		secrets: secrets,
		flags:   flags,
		config:  config,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// GetSecret returns a decrypted secret by key.
func (a *Adapter) GetSecret(ctx context.Context, key string) ([]byte, error) {
	if a.secrets == nil {
		return nil, bridge.ErrVaultNotAvailable
	}
	s, err := a.secrets.Get(ctx, key, a.appID)
	if err != nil {
		return nil, err
	}
	return s.Value, nil
}

// SetSecret creates or updates an encrypted secret.
func (a *Adapter) SetSecret(ctx context.Context, key string, value []byte) error {
	if a.secrets == nil {
		return bridge.ErrVaultNotAvailable
	}
	_, err := a.secrets.Set(ctx, key, value, a.appID)
	if err != nil {
		return err
	}
	return nil
}

// IsFeatureEnabled evaluates a boolean feature flag.
func (a *Adapter) IsFeatureEnabled(ctx context.Context, flag string) bool {
	if a.flags == nil {
		return false
	}
	return a.flags.Bool(ctx, flag, false)
}

// GetConfig returns a configuration value by key.
func (a *Adapter) GetConfig(ctx context.Context, key string) (string, error) {
	if a.config == nil {
		return "", bridge.ErrVaultNotAvailable
	}
	return a.config.String(ctx, key, ""), nil
}
