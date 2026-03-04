package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/appsessionconfig"
	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// AppSessionConfig store
// ──────────────────────────────────────────────────

// GetAppSessionConfig returns the session config override for the given app.
func (s *Store) GetAppSessionConfig(ctx context.Context, appID id.AppID) (*appsessionconfig.Config, error) {
	var m appSessionConfigModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"app_id": appID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, appsessionconfig.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get app session config: %w", err)
	}

	return fromAppSessionConfigModel(&m)
}

// SetAppSessionConfig creates or updates the session config for an app.
func (s *Store) SetAppSessionConfig(ctx context.Context, cfg *appsessionconfig.Config) error {
	if cfg.ID.IsNil() {
		cfg.ID = id.NewAppSessionConfigID()
	}
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now()
	}
	cfg.UpdatedAt = now()

	m := toAppSessionConfigModel(cfg)

	// Try update first; if no match, insert.
	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"app_id": m.AppID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: set app session config (update): %w", err)
	}

	if res.MatchedCount() > 0 {
		return nil
	}

	_, err = s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: set app session config (insert): %w", err)
	}

	return nil
}

// DeleteAppSessionConfig removes the per-app session config override.
func (s *Store) DeleteAppSessionConfig(ctx context.Context, appID id.AppID) error {
	res, err := s.mdb.NewDelete((*appSessionConfigModel)(nil)).
		Filter(bson.M{"app_id": appID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete app session config: %w", err)
	}

	if res.DeletedCount() == 0 {
		return appsessionconfig.ErrNotFound
	}

	return nil
}
