package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
)

// ──────────────────────────────────────────────────
// FormConfig store
// ──────────────────────────────────────────────────

func (s *Store) CreateFormConfig(ctx context.Context, fc *formconfig.FormConfig) error {
	m := toFormConfigModel(fc)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create form config: %w", err)
	}

	return nil
}

func (s *Store) GetFormConfig(ctx context.Context, appID id.AppID, formType string) (*formconfig.FormConfig, error) {
	var m formConfigModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{
			"app_id":    appID.String(),
			"form_type": formType,
		}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get form config: %w", err)
	}

	return fromFormConfigModel(&m)
}

func (s *Store) UpdateFormConfig(ctx context.Context, fc *formconfig.FormConfig) error {
	m := toFormConfigModel(fc)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update form config: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

func (s *Store) DeleteFormConfig(ctx context.Context, appID id.AppID, formType string) error {
	res, err := s.mdb.NewDelete((*formConfigModel)(nil)).
		Filter(bson.M{
			"app_id":    appID.String(),
			"form_type": formType,
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete form config: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

func (s *Store) ListFormConfigs(ctx context.Context, appID id.AppID) ([]*formconfig.FormConfig, error) {
	var models []formConfigModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"app_id": appID.String()}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list form configs: %w", err)
	}

	result := make([]*formconfig.FormConfig, 0, len(models))

	for i := range models {
		fc, err := fromFormConfigModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, fc)
	}

	return result, nil
}

// ──────────────────────────────────────────────────
// Branding store
// ──────────────────────────────────────────────────

func (s *Store) GetBranding(ctx context.Context, orgID id.OrgID) (*formconfig.BrandingConfig, error) {
	var m brandingConfigModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"org_id": orgID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get branding: %w", err)
	}

	return fromBrandingConfigModel(&m)
}

func (s *Store) SaveBranding(ctx context.Context, b *formconfig.BrandingConfig) error {
	m := toBrandingConfigModel(b)
	m.UpdatedAt = now()

	// Try update first; if no match, insert.
	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: save branding (update): %w", err)
	}

	if res.MatchedCount() > 0 {
		return nil
	}

	_, err = s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: save branding (insert): %w", err)
	}

	return nil
}

func (s *Store) DeleteBranding(ctx context.Context, orgID id.OrgID) error {
	res, err := s.mdb.NewDelete((*brandingConfigModel)(nil)).
		Filter(bson.M{"org_id": orgID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete branding: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}
