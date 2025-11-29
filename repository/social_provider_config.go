package repository

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/schema"
)

// SocialProviderConfigRepository handles social provider config persistence
type SocialProviderConfigRepository interface {
	// Create creates a new social provider config
	Create(ctx context.Context, config *schema.SocialProviderConfig) error

	// FindByID finds a config by ID
	FindByID(ctx context.Context, id xid.ID) (*schema.SocialProviderConfig, error)

	// FindByProvider finds a config by app, environment, and provider name
	FindByProvider(ctx context.Context, appID, envID xid.ID, providerName string) (*schema.SocialProviderConfig, error)

	// ListByEnvironment lists all configs for an environment
	ListByEnvironment(ctx context.Context, appID, envID xid.ID) ([]*schema.SocialProviderConfig, error)

	// ListEnabledByEnvironment lists only enabled configs for an environment
	ListEnabledByEnvironment(ctx context.Context, appID, envID xid.ID) ([]*schema.SocialProviderConfig, error)

	// Update updates an existing config
	Update(ctx context.Context, config *schema.SocialProviderConfig) error

	// Delete soft-deletes a config by ID
	Delete(ctx context.Context, id xid.ID) error

	// HardDelete permanently deletes a config
	HardDelete(ctx context.Context, id xid.ID) error

	// SetEnabled enables or disables a provider
	SetEnabled(ctx context.Context, id xid.ID, enabled bool) error

	// CountByEnvironment counts providers for an environment
	CountByEnvironment(ctx context.Context, appID, envID xid.ID) (int, error)

	// ExistsByProvider checks if a provider config exists for the environment
	ExistsByProvider(ctx context.Context, appID, envID xid.ID, providerName string) (bool, error)
}

type socialProviderConfigRepository struct {
	db *bun.DB
}

// NewSocialProviderConfigRepository creates a new social provider config repository
func NewSocialProviderConfigRepository(db *bun.DB) SocialProviderConfigRepository {
	return &socialProviderConfigRepository{db: db}
}

func (r *socialProviderConfigRepository) Create(ctx context.Context, config *schema.SocialProviderConfig) error {
	if config.ID.IsNil() {
		config.ID = xid.New()
	}

	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	_, err := r.db.NewInsert().Model(config).Exec(ctx)
	if err != nil {
		return errs.Wrap(err, errs.CodeInternalError, "failed to create social provider config", http.StatusInternalServerError)
	}
	return nil
}

func (r *socialProviderConfigRepository) FindByID(ctx context.Context, id xid.ID) (*schema.SocialProviderConfig, error) {
	config := &schema.SocialProviderConfig{}
	err := r.db.NewSelect().
		Model(config).
		Where("spc.id = ?", id).
		Where("spc.deleted_at IS NULL").
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, errs.New(errs.CodeNotFound, "social provider config not found", http.StatusNotFound)
	}
	if err != nil {
		return nil, errs.Wrap(err, errs.CodeInternalError, "failed to find social provider config", http.StatusInternalServerError)
	}
	return config, nil
}

func (r *socialProviderConfigRepository) FindByProvider(ctx context.Context, appID, envID xid.ID, providerName string) (*schema.SocialProviderConfig, error) {
	config := &schema.SocialProviderConfig{}
	err := r.db.NewSelect().
		Model(config).
		Where("spc.app_id = ?", appID).
		Where("spc.environment_id = ?", envID).
		Where("spc.provider_name = ?", providerName).
		Where("spc.deleted_at IS NULL").
		Scan(ctx)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error for this method
	}
	if err != nil {
		return nil, errs.Wrap(err, errs.CodeInternalError, "failed to find social provider config", http.StatusInternalServerError)
	}
	return config, nil
}

func (r *socialProviderConfigRepository) ListByEnvironment(ctx context.Context, appID, envID xid.ID) ([]*schema.SocialProviderConfig, error) {
	var configs []*schema.SocialProviderConfig
	err := r.db.NewSelect().
		Model(&configs).
		Where("spc.app_id = ?", appID).
		Where("spc.environment_id = ?", envID).
		Where("spc.deleted_at IS NULL").
		Order("spc.provider_name ASC").
		Scan(ctx)

	if err != nil {
		return nil, errs.Wrap(err, errs.CodeInternalError, "failed to list social provider configs", http.StatusInternalServerError)
	}
	return configs, nil
}

func (r *socialProviderConfigRepository) ListEnabledByEnvironment(ctx context.Context, appID, envID xid.ID) ([]*schema.SocialProviderConfig, error) {
	var configs []*schema.SocialProviderConfig
	err := r.db.NewSelect().
		Model(&configs).
		Where("spc.app_id = ?", appID).
		Where("spc.environment_id = ?", envID).
		Where("spc.is_enabled = true").
		Where("spc.deleted_at IS NULL").
		Order("spc.provider_name ASC").
		Scan(ctx)

	if err != nil {
		return nil, errs.Wrap(err, errs.CodeInternalError, "failed to list enabled social provider configs", http.StatusInternalServerError)
	}
	return configs, nil
}

func (r *socialProviderConfigRepository) Update(ctx context.Context, config *schema.SocialProviderConfig) error {
	config.UpdatedAt = time.Now()

	result, err := r.db.NewUpdate().
		Model(config).
		WherePK().
		Where("deleted_at IS NULL").
		Exec(ctx)

	if err != nil {
		return errs.Wrap(err, errs.CodeInternalError, "failed to update social provider config", http.StatusInternalServerError)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errs.New(errs.CodeNotFound, "social provider config not found", http.StatusNotFound)
	}

	return nil
}

func (r *socialProviderConfigRepository) Delete(ctx context.Context, id xid.ID) error {
	now := time.Now()
	result, err := r.db.NewUpdate().
		Model((*schema.SocialProviderConfig)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)

	if err != nil {
		return errs.Wrap(err, errs.CodeInternalError, "failed to delete social provider config", http.StatusInternalServerError)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errs.New(errs.CodeNotFound, "social provider config not found", http.StatusNotFound)
	}

	return nil
}

func (r *socialProviderConfigRepository) HardDelete(ctx context.Context, id xid.ID) error {
	result, err := r.db.NewDelete().
		Model((*schema.SocialProviderConfig)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	if err != nil {
		return errs.Wrap(err, errs.CodeInternalError, "failed to permanently delete social provider config", http.StatusInternalServerError)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errs.New(errs.CodeNotFound, "social provider config not found", http.StatusNotFound)
	}

	return nil
}

func (r *socialProviderConfigRepository) SetEnabled(ctx context.Context, id xid.ID, enabled bool) error {
	result, err := r.db.NewUpdate().
		Model((*schema.SocialProviderConfig)(nil)).
		Set("is_enabled = ?", enabled).
		Set("updated_at = ?", time.Now()).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Exec(ctx)

	if err != nil {
		return errs.Wrap(err, errs.CodeInternalError, "failed to toggle social provider config", http.StatusInternalServerError)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errs.New(errs.CodeNotFound, "social provider config not found", http.StatusNotFound)
	}

	return nil
}

func (r *socialProviderConfigRepository) CountByEnvironment(ctx context.Context, appID, envID xid.ID) (int, error) {
	count, err := r.db.NewSelect().
		Model((*schema.SocialProviderConfig)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("deleted_at IS NULL").
		Count(ctx)

	if err != nil {
		return 0, errs.Wrap(err, errs.CodeInternalError, "failed to count social provider configs", http.StatusInternalServerError)
	}
	return count, nil
}

func (r *socialProviderConfigRepository) ExistsByProvider(ctx context.Context, appID, envID xid.ID, providerName string) (bool, error) {
	exists, err := r.db.NewSelect().
		Model((*schema.SocialProviderConfig)(nil)).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("provider_name = ?", providerName).
		Where("deleted_at IS NULL").
		Exists(ctx)

	if err != nil {
		return false, errs.Wrap(err, errs.CodeInternalError, "failed to check if social provider config exists", http.StatusInternalServerError)
	}
	return exists, nil
}
