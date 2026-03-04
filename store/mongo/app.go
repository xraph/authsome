package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// CreateApp persists a new app.
func (s *Store) CreateApp(ctx context.Context, a *app.App) error {
	m := toAppModel(a)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create app: %w", err)
	}

	return nil
}

// GetApp returns an app by ID.
func (s *Store) GetApp(ctx context.Context, appID id.AppID) (*app.App, error) {
	var m appModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": appID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get app: %w", err)
	}

	return fromAppModel(&m)
}

// GetAppBySlug returns an app by its slug.
func (s *Store) GetAppBySlug(ctx context.Context, slug string) (*app.App, error) {
	var m appModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"slug": slug}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get app by slug: %w", err)
	}

	return fromAppModel(&m)
}

// UpdateApp modifies an existing app.
func (s *Store) UpdateApp(ctx context.Context, a *app.App) error {
	m := toAppModel(a)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update app: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteApp removes an app.
func (s *Store) DeleteApp(ctx context.Context, appID id.AppID) error {
	res, err := s.mdb.NewDelete((*appModel)(nil)).
		Filter(bson.M{"_id": appID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete app: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// ListApps returns all apps, ordered by creation date descending.
func (s *Store) ListApps(ctx context.Context) ([]*app.App, error) {
	var models []appModel

	err := s.mdb.NewFind(&models).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list apps: %w", err)
	}

	result := make([]*app.App, 0, len(models))

	for i := range models {
		a, err := fromAppModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, a)
	}

	return result, nil
}
