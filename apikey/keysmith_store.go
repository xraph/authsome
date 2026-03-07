package apikey

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xraph/authsome/id"

	"github.com/xraph/keysmith"
	ksid "github.com/xraph/keysmith/id"
	"github.com/xraph/keysmith/key"
)

// KeysmithStore implements apikey.Store by delegating to a Keysmith engine.
// When Keysmith is available as a first-class citizen, API key CRUD operations
// gain rate limiting, policy enforcement, key rotation with grace periods,
// scope management, usage tracking, and multi-tenant support.
type KeysmithStore struct {
	engine *keysmith.Engine
}

// NewKeymithStore creates a new KeysmithStore backed by the given Keysmith engine.
func NewKeymithStore(eng *keysmith.Engine) *KeysmithStore {
	return &KeysmithStore{engine: eng}
}

// Compile-time interface check.
var _ Store = (*KeysmithStore)(nil)

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *KeysmithStore) CreateAPIKey(ctx context.Context, ak *APIKey) error {
	k := toKeysmithKey(ak)
	if err := s.engine.Store().Keys().Create(ctx, k); err != nil {
		return keysmithError(err)
	}
	return nil
}

func (s *KeysmithStore) GetAPIKey(ctx context.Context, keyID id.APIKeyID) (*APIKey, error) {
	kid, err := ksid.ParseKeyID(keyID.String())
	if err != nil {
		return nil, fmt.Errorf("apikey: invalid key id %q: %w", keyID.String(), err)
	}
	k, err := s.engine.Store().Keys().Get(ctx, kid)
	if err != nil {
		return nil, keysmithError(err)
	}
	return fromKeysmithKey(k)
}

func (s *KeysmithStore) GetAPIKeyByPrefix(ctx context.Context, _ id.AppID, prefix string) (*APIKey, error) {
	// Keysmith's GetByPrefix requires both prefix marker and hint.
	// AuthSome stores the full key prefix (e.g. "ask_abcd1234") as Keysmith's Hint,
	// and uses "ask" as the Keysmith Prefix marker.
	k, err := s.engine.Store().Keys().GetByPrefix(ctx, "ask", prefix)
	if err != nil {
		return nil, keysmithError(err)
	}
	return fromKeysmithKey(k)
}

func (s *KeysmithStore) GetAPIKeyByPublicKey(ctx context.Context, appID id.AppID, publicKey string) (*APIKey, error) {
	// Public key is stored in keysmith metadata. List keys for the tenant
	// and find the matching one by comparing the public key field.
	keys, err := s.engine.Store().Keys().List(ctx, &key.ListFilter{
		TenantID: appID.String(),
	})
	if err != nil {
		return nil, keysmithError(err)
	}
	for _, k := range keys {
		ak, err := fromKeysmithKey(k)
		if err != nil {
			continue
		}
		if ak.PublicKey == publicKey || ak.PublicKeyPrefix == publicKey {
			return ak, nil
		}
	}
	return nil, ErrNotFound
}

func (s *KeysmithStore) UpdateAPIKey(ctx context.Context, ak *APIKey) error {
	k := toKeysmithKey(ak)
	if err := s.engine.Store().Keys().Update(ctx, k); err != nil {
		return keysmithError(err)
	}
	return nil
}

func (s *KeysmithStore) DeleteAPIKey(ctx context.Context, keyID id.APIKeyID) error {
	kid, err := ksid.ParseKeyID(keyID.String())
	if err != nil {
		return fmt.Errorf("apikey: invalid key id %q: %w", keyID.String(), err)
	}
	if err := s.engine.Store().Keys().Delete(ctx, kid); err != nil {
		return keysmithError(err)
	}
	return nil
}

func (s *KeysmithStore) ListAPIKeysByApp(ctx context.Context, appID id.AppID) ([]*APIKey, error) {
	keys, err := s.engine.Store().Keys().List(ctx, &key.ListFilter{
		TenantID: appID.String(),
	})
	if err != nil {
		return nil, keysmithError(err)
	}
	return convertKeyList(keys)
}

func (s *KeysmithStore) ListAPIKeysByUser(ctx context.Context, appID id.AppID, userID id.UserID) ([]*APIKey, error) {
	keys, err := s.engine.Store().Keys().List(ctx, &key.ListFilter{
		TenantID:  appID.String(),
		CreatedBy: userID.String(),
	})
	if err != nil {
		return nil, keysmithError(err)
	}
	return convertKeyList(keys)
}

// ──────────────────────────────────────────────────
// Converters
// ──────────────────────────────────────────────────

// toKeysmithKey converts an AuthSome APIKey to a Keysmith Key.
func toKeysmithKey(ak *APIKey) *key.Key {
	kid, _ := ksid.ParseKeyID(ak.ID.String())

	state := key.StateActive
	if ak.Revoked {
		state = key.StateRevoked
	}

	k := &key.Key{
		ID:          kid,
		TenantID:    ak.AppID.String(),
		Name:        ak.Name,
		Prefix:      "ask",
		Hint:        ak.KeyPrefix,
		KeyHash:     ak.KeyHash,
		Environment: key.EnvLive,
		State:       state,
		Scopes:      ak.Scopes,
		CreatedBy:   ak.UserID.String(),
		ExpiresAt:   ak.ExpiresAt,
		LastUsedAt:  ak.LastUsedAt,
		CreatedAt:   ak.CreatedAt,
		UpdatedAt:   ak.UpdatedAt,
	}

	// Store public key fields in metadata.
	if ak.PublicKey != "" || ak.PublicKeyPrefix != "" {
		k.Metadata = map[string]any{
			"public_key":        ak.PublicKey,
			"public_key_prefix": ak.PublicKeyPrefix,
		}
	}

	return k
}

// fromKeysmithKey converts a Keysmith Key to an AuthSome APIKey.
func fromKeysmithKey(kk *key.Key) (*APIKey, error) {
	akID, err := id.ParseAPIKeyID(kk.ID.String())
	if err != nil {
		return nil, fmt.Errorf("apikey: parse key id: %w", err)
	}

	appID, err := id.ParseAppID(kk.TenantID)
	if err != nil {
		return nil, fmt.Errorf("apikey: parse app id from tenant: %w", err)
	}

	ak := &APIKey{
		ID:        akID,
		AppID:     appID,
		Name:      kk.Name,
		KeyHash:   kk.KeyHash,
		KeyPrefix: kk.Hint,
		Scopes:    kk.Scopes,
		ExpiresAt: kk.ExpiresAt,
		Revoked:   kk.State == key.StateRevoked,
		CreatedAt: kk.CreatedAt,
		UpdatedAt: kk.UpdatedAt,
	}

	// Parse UserID from CreatedBy (graceful fallback for empty or invalid).
	if kk.CreatedBy != "" {
		userID, err := id.ParseUserID(kk.CreatedBy)
		if err == nil {
			ak.UserID = userID
		}
		// Invalid CreatedBy is non-fatal; leave UserID as zero value.
		// This can happen with keys created from the dashboard without user context.
	}

	// Copy LastUsedAt.
	ak.LastUsedAt = kk.LastUsedAt

	// Read public key fields from metadata.
	if kk.Metadata != nil {
		if v, ok := kk.Metadata["public_key"].(string); ok {
			ak.PublicKey = v
		}
		if v, ok := kk.Metadata["public_key_prefix"].(string); ok {
			ak.PublicKeyPrefix = v
		}
	}

	return ak, nil
}

// convertKeyList converts a slice of Keysmith keys to AuthSome API keys.
// Keys that fail conversion are skipped rather than failing the entire list.
func convertKeyList(keys []*key.Key) ([]*APIKey, error) {
	result := make([]*APIKey, 0, len(keys))
	for _, kk := range keys {
		ak, err := fromKeysmithKey(kk)
		if err != nil {
			continue // skip unconvertible keys
		}
		result = append(result, ak)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Error mapping
// ──────────────────────────────────────────────────

// keysmithError translates keysmith-level errors to apikey errors.
func keysmithError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keysmith.ErrKeyNotFound) {
		return ErrNotFound
	}
	// Keysmith's memory store returns a generic "not found" error that does
	// not wrap keysmith.ErrKeyNotFound. Handle it by checking the message.
	if strings.Contains(err.Error(), "not found") {
		return ErrNotFound
	}
	return err
}
