package bridge

import (
	"context"
	"errors"
)

// ErrKeyManagerNotAvailable is returned when key management is not configured.
var ErrKeyManagerNotAvailable = errors.New("bridge: key manager not available (standalone mode)")

// KeyManager is a local key management interface. Implementations create,
// validate, and revoke keys (e.g., via keysmith).
type KeyManager interface {
	CreateKey(ctx context.Context, input *CreateKeyInput) (*KeyResult, error)
	ValidateKey(ctx context.Context, rawKey string) (*ValidatedKey, error)
	RevokeKey(ctx context.Context, keyID string) error
}

// CreateKeyInput represents the input for key creation.
type CreateKeyInput struct {
	Name        string            `json:"name"`
	Owner       string            `json:"owner"`
	Environment string            `json:"environment,omitempty"`
	Scopes      []string          `json:"scopes,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// KeyResult represents a created key.
type KeyResult struct {
	ID     string `json:"id"`
	RawKey string `json:"raw_key"` // Only returned on creation
}

// ValidatedKey represents a validated key with resolved metadata.
type ValidatedKey struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Owner       string            `json:"owner"`
	Environment string            `json:"environment,omitempty"`
	Scopes      []string          `json:"scopes,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
