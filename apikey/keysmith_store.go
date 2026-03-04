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

	// Parse UserID from CreatedBy (graceful fallback for empty).
	if kk.CreatedBy != "" {
		userID, err := id.ParseUserID(kk.CreatedBy)
		if err != nil {
			return nil, fmt.Errorf("apikey: parse user id from created_by: %w", err)
		}
		ak.UserID = userID
	}

	// Copy LastUsedAt.
	ak.LastUsedAt = kk.LastUsedAt

	return ak, nil
}

// convertKeyList converts a slice of Keysmith keys to AuthSome API keys.
func convertKeyList(keys []*key.Key) ([]*APIKey, error) {
	result := make([]*APIKey, 0, len(keys))
	for _, kk := range keys {
		ak, err := fromKeysmithKey(kk)
		if err != nil {
			return nil, err
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
