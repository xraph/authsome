package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/grove/drivers/mongodriver"

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

// GetAppByPublishableKey returns an app by its publishable key.
func (s *Store) GetAppByPublishableKey(ctx context.Context, key string) (*app.App, error) {
	var m appModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"publishable_key": key}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get app by publishable key: %w", err)
	}

	return fromAppModel(&m)
}

// GetPlatformApp returns the single platform app (is_platform=true).
func (s *Store) GetPlatformApp(ctx context.Context) (*app.App, error) {
	var m appModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"is_platform": true}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get platform app: %w", err)
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
// DeleteApp removes an app and cascades to every app-scoped collection, so a
// deleted app leaves no orphaned users, sessions, org memberships, API keys,
// etc. behind (which the SQL backends achieve via ON DELETE CASCADE / explicit
// deletes). Runs inside a MongoTx for atomicity — this requires the deployment
// to run against a replica set or sharded cluster.
func (s *Store) DeleteApp(ctx context.Context, appID id.AppID) error {
	mtx, err := s.mdb.GroveTx(ctx, 0, false)
	if err != nil {
		return fmt.Errorf("authsome/mongo: begin tx for app cascade: %w", err)
	}
	tx, ok := mtx.(*mongodriver.MongoTx)
	if !ok {
		return fmt.Errorf("authsome/mongo: unexpected tx type %T", mtx)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback() //nolint:errcheck // best-effort rollback
		}
	}()

	aid := appID.String()
	appFilter := bson.M{"app_id": aid}

	// Organization-scoped children (members, teams, invitations) are keyed by
	// org_id. Mongo has no subqueries, so resolve this app's org ids first,
	// then delete their children by org_id.
	var orgs []organizationModel
	if err := tx.NewFind(&orgs).Filter(appFilter).Scan(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: list app orgs for cascade: %w", err)
	}
	if len(orgs) > 0 {
		orgIDs := make([]string, 0, len(orgs))
		for i := range orgs {
			orgIDs = append(orgIDs, orgs[i].ID)
		}
		orgFilter := bson.M{"org_id": bson.M{"$in": orgIDs}}
		for _, m := range []any{(*memberModel)(nil), (*teamModel)(nil), (*invitationModel)(nil)} {
			if _, err := tx.NewDelete(m).Filter(orgFilter).Exec(ctx); err != nil {
				return fmt.Errorf("authsome/mongo: delete org children: %w", err)
			}
		}
	}

	// App-scoped rows (every model carrying an app_id).
	for _, m := range []any{
		(*organizationModel)(nil),
		(*userEmailModel)(nil),
		(*sessionModel)(nil),
		(*verificationModel)(nil),
		(*passwordResetModel)(nil),
		(*deviceModel)(nil),
		(*webhookModel)(nil),
		(*notificationModel)(nil),
		(*apiKeyModel)(nil),
		(*userModel)(nil),
		(*environmentModel)(nil),
		(*formConfigModel)(nil),
		(*brandingConfigModel)(nil),
		(*appSessionConfigModel)(nil),
		(*appClientConfigModel)(nil),
		(*settingModel)(nil),
	} {
		if _, err := tx.NewDelete(m).Filter(appFilter).Exec(ctx); err != nil {
			return fmt.Errorf("authsome/mongo: delete app children: %w", err)
		}
	}

	// Delete is idempotent (matching the SQL backends and every other Delete*
	// method): a missing app is a no-op, not an error.
	if _, err := tx.NewDelete((*appModel)(nil)).Filter(bson.M{"_id": aid}).Exec(ctx); err != nil {
		return fmt.Errorf("authsome/mongo: delete app row: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("authsome/mongo: commit app cascade: %w", err)
	}
	committed = true
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
