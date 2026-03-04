package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// CreateAPIKey persists a new API key.
func (s *Store) CreateAPIKey(ctx context.Context, k *apikey.APIKey) error {
	m := toAPIKeyModel(k)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create api key: %w", err)
	}

	return nil
}

// GetAPIKey returns an API key by ID.
func (s *Store) GetAPIKey(ctx context.Context, keyID id.APIKeyID) (*apikey.APIKey, error) {
	var m apiKeyModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": keyID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get api key: %w", err)
	}

	return fromAPIKeyModel(&m)
}

// GetAPIKeyByPrefix returns an API key by app ID and key prefix.
func (s *Store) GetAPIKeyByPrefix(ctx context.Context, appID id.AppID, prefix string) (*apikey.APIKey, error) {
	var m apiKeyModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"app_id":     appID.String(),
			"key_prefix": prefix,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get api key by prefix: %w", err)
	}

	return fromAPIKeyModel(&m)
}

// UpdateAPIKey modifies an existing API key.
func (s *Store) UpdateAPIKey(ctx context.Context, k *apikey.APIKey) error {
	m := toAPIKeyModel(k)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update api key: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteAPIKey removes an API key.
func (s *Store) DeleteAPIKey(ctx context.Context, keyID id.APIKeyID) error {
	res, err := s.mdb.NewDelete((*apiKeyModel)(nil)).
		Filter(bson.M{"_id": keyID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete api key: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListAPIKeysByApp returns all API keys for an app, ordered by creation date descending.
func (s *Store) ListAPIKeysByApp(ctx context.Context, appID id.AppID) ([]*apikey.APIKey, error) {
	var models []apiKeyModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"app_id": appID.String()}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list api keys by app: %w", err)
	}

	result := make([]*apikey.APIKey, 0, len(models))

	for i := range models {
		k, err := fromAPIKeyModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, k)
	}

	return result, nil
}

// ListAPIKeysByUser returns all API keys for a user within an app, ordered by creation date descending.
func (s *Store) ListAPIKeysByUser(ctx context.Context, appID id.AppID, userID id.UserID) ([]*apikey.APIKey, error) {
	var models []apiKeyModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{
			"app_id":  appID.String(),
			"user_id": userID.String(),
		}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list api keys by user: %w", err)
	}

	result := make([]*apikey.APIKey, 0, len(models))

	for i := range models {
		k, err := fromAPIKeyModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, k)
	}

	return result, nil
}
