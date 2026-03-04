// Package keysmithadapter bridges AuthSome key management to the Keysmith extension.
package keysmithadapter

import (
	"context"

	"github.com/xraph/keysmith"
	ksid "github.com/xraph/keysmith/id"
	"github.com/xraph/keysmith/key"

	"github.com/xraph/authsome/bridge"
)

// Adapter translates AuthSome key management to Keysmith operations.
type Adapter struct {
	engine *keysmith.Engine
}

// New creates a Keysmith bridge adapter.
func New(engine *keysmith.Engine) *Adapter {
	return &Adapter{engine: engine}
}

// CreateKey implements bridge.KeyManager.
func (a *Adapter) CreateKey(ctx context.Context, input *bridge.CreateKeyInput) (*bridge.KeyResult, error) {
	md := make(map[string]any, len(input.Metadata))
	for k, v := range input.Metadata {
		md[k] = v
	}

	result, err := a.engine.CreateKey(ctx, &keysmith.CreateKeyInput{
		Name:        input.Name,
		Prefix:      "authsome",
		Environment: key.EnvLive,
		Scopes:      input.Scopes,
		Metadata:    md,
		CreatedBy:   input.Owner,
	})
	if err != nil {
		return nil, err
	}

	return &bridge.KeyResult{
		ID:     result.Key.ID.String(),
		RawKey: result.RawKey,
	}, nil
}

// ValidateKey implements bridge.KeyManager.
func (a *Adapter) ValidateKey(ctx context.Context, rawKey string) (*bridge.ValidatedKey, error) {
	result, err := a.engine.ValidateKey(ctx, rawKey)
	if err != nil {
		return nil, err
	}

	md := make(map[string]string)
	for k, v := range result.Key.Metadata {
		if s, ok := v.(string); ok {
			md[k] = s
		}
	}

	return &bridge.ValidatedKey{
		ID:       result.Key.ID.String(),
		Name:     result.Key.Name,
		Owner:    result.Key.CreatedBy,
		Scopes:   result.Scopes,
		Metadata: md,
	}, nil
}

// RevokeKey implements bridge.KeyManager.
func (a *Adapter) RevokeKey(ctx context.Context, keyID string) error {
	kid, err := ksid.ParseKeyID(keyID)
	if err != nil {
		return err
	}
	return a.engine.RevokeKey(ctx, kid, "revoked by authsome")
}

// Compile-time check.
var _ bridge.KeyManager = (*Adapter)(nil)
