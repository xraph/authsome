package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// AppClientConfig store
// ──────────────────────────────────────────────────

// GetAppClientConfig returns the client config override for the given app.
func (s *Store) GetAppClientConfig(ctx context.Context, appID id.AppID) (*appclientconfig.Config, error) {
	var m appClientConfigModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"app_id": appID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, appclientconfig.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get app client config: %w", err)
	}

	return fromAppClientConfigModel(&m)
}

// SetAppClientConfig creates or updates the client config for an app.
func (s *Store) SetAppClientConfig(ctx context.Context, cfg *appclientconfig.Config) error {
	if cfg.ID.IsNil() {
		cfg.ID = id.NewAppClientConfigID()
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now()
	}
	cfg.UpdatedAt = now()

	m := toAppClientConfigModel(cfg)

	// Try update first; if no match, insert.
	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"app_id": m.AppID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: set app client config (update): %w", err)
	}

	if res.MatchedCount() > 0 {
		return nil
	}

	_, err = s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: set app client config (insert): %w", err)
	}

	return nil
}

// DeleteAppClientConfig removes the per-app client config override.
func (s *Store) DeleteAppClientConfig(ctx context.Context, appID id.AppID) error {
	res, err := s.mdb.NewDelete((*appClientConfigModel)(nil)).
		Filter(bson.M{"app_id": appID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete app client config: %w", err)
	}

	if res.DeletedCount() == 0 {
		return appclientconfig.ErrNotFound
	}

	return nil
}
