package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// CreateEnvironment persists a new environment.
func (s *Store) CreateEnvironment(ctx context.Context, e *environment.Environment) error {
	m := toEnvironmentModel(e)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create environment: %w", err)
	}

	return nil
}

// GetEnvironment returns an environment by ID.
func (s *Store) GetEnvironment(ctx context.Context, envID id.EnvironmentID) (*environment.Environment, error) {
	var m environmentModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": envID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get environment: %w", err)
	}

	return fromEnvironmentModel(&m)
}

// GetEnvironmentBySlug returns an environment by app ID and slug.
func (s *Store) GetEnvironmentBySlug(ctx context.Context, appID id.AppID, slug string) (*environment.Environment, error) {
	var m environmentModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"app_id": appID.String(),
			"slug":   slug,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get environment by slug: %w", err)
	}

	return fromEnvironmentModel(&m)
}

// GetDefaultEnvironment returns the default environment for an app.
func (s *Store) GetDefaultEnvironment(ctx context.Context, appID id.AppID) (*environment.Environment, error) {
	var m environmentModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"app_id":     appID.String(),
			"is_default": true,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get default environment: %w", err)
	}

	return fromEnvironmentModel(&m)
}

// UpdateEnvironment modifies an existing environment.
func (s *Store) UpdateEnvironment(ctx context.Context, e *environment.Environment) error {
	m := toEnvironmentModel(e)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update environment: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteEnvironment removes an environment by ID.
// Returns an error if the environment is the default for its app.
func (s *Store) DeleteEnvironment(ctx context.Context, envID id.EnvironmentID) error {
	var m environmentModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": envID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return store.ErrNotFound
		}

		return fmt.Errorf("authsome/mongo: delete environment (lookup): %w", err)
	}

	if m.IsDefault {
		return fmt.Errorf("authsome/mongo: cannot delete the default environment")
	}

	res, err := s.mdb.NewDelete((*environmentModel)(nil)).
		Filter(bson.M{"_id": envID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete environment: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListEnvironments returns all environments for an app, ordered by creation date ascending.
func (s *Store) ListEnvironments(ctx context.Context, appID id.AppID) ([]*environment.Environment, error) {
	var models []environmentModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"app_id": appID.String()}).
		Sort(bson.D{{Key: "created_at", Value: 1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list environments: %w", err)
	}

	result := make([]*environment.Environment, 0, len(models))

	for i := range models {
		e, err := fromEnvironmentModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, e)
	}

	return result, nil
}

// SetDefaultEnvironment sets the given environment as the default for its app,
// clearing the default flag on any previously default environment.
func (s *Store) SetDefaultEnvironment(ctx context.Context, appID id.AppID, envID id.EnvironmentID) error {
	// Clear any existing default for this app.
	_, err := s.mdb.NewUpdate((*environmentModel)(nil)).
		Many().
		Filter(bson.M{
			"app_id":     appID.String(),
			"is_default": true,
		}).
		Set("is_default", false).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: clear default environment: %w", err)
	}

	// Set the new default.
	res, err := s.mdb.NewUpdate((*environmentModel)(nil)).
		Filter(bson.M{"_id": envID.String()}).
		Set("is_default", true).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: set default environment: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}
