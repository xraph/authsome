package repository

import (
	"context"
	"database/sql"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/schema"
)

// SSOProviderRepository provides persistence for SSO provider configurations with multi-tenant scoping
type SSOProviderRepository struct{ db *bun.DB }

func NewSSOProviderRepository(db *bun.DB) *SSOProviderRepository {
	return &SSOProviderRepository{db: db}
}

// Create inserts a new SSOProvider record
func (r *SSOProviderRepository) Create(ctx context.Context, p *schema.SSOProvider) error {
	_, err := r.db.NewInsert().Model(p).Exec(ctx)
	return err
}

// Upsert creates or updates an SSOProvider by ProviderID within the tenant scope
func (r *SSOProviderRepository) Upsert(ctx context.Context, p *schema.SSOProvider) error {
	// Extract tenant context
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)

	// Try find existing within tenant scope
	existing := new(schema.SSOProvider)
	query := r.db.NewSelect().
		Model(existing).
		Where("provider_id = ?", p.ProviderID).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	if !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	err := query.Scan(ctx)
	if err == sql.ErrNoRows {
		// Create new provider
		_, err2 := r.db.NewInsert().Model(p).Exec(ctx)
		return err2
	}
	if err != nil {
		return err
	}

	// Update existing provider
	p.ID = existing.ID
	_, err = r.db.NewUpdate().Model(p).WherePK().Exec(ctx)
	return err
}

// FindByProviderID returns an SSOProvider by ProviderID within the tenant scope
func (r *SSOProviderRepository) FindByProviderID(ctx context.Context, providerID string) (*schema.SSOProvider, error) {
	// Extract tenant context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, nil // No app context, can't query
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, nil // No env context, can't query
	}
	orgID, _ := contexts.GetOrganizationID(ctx)

	p := new(schema.SSOProvider)
	query := r.db.NewSelect().
		Model(p).
		Where("provider_id = ?", providerID).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	// Organization filter (optional)
	if !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}

	err := query.Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// FindByDomain returns SSO providers matching a domain within the tenant scope
func (r *SSOProviderRepository) FindByDomain(ctx context.Context, domain string) ([]*schema.SSOProvider, error) {
	// Extract tenant context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return nil, nil
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return nil, nil
	}
	orgID, _ := contexts.GetOrganizationID(ctx)

	var providers []*schema.SSOProvider
	query := r.db.NewSelect().
		Model(&providers).
		Where("domain = ?", domain).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	if !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// List returns all SSO providers within the tenant scope
func (r *SSOProviderRepository) List(ctx context.Context) ([]*schema.SSOProvider, error) {
	// Extract tenant context
	appID, ok := contexts.GetAppID(ctx)
	if !ok {
		return []*schema.SSOProvider{}, nil
	}
	envID, ok := contexts.GetEnvironmentID(ctx)
	if !ok {
		return []*schema.SSOProvider{}, nil
	}
	orgID, _ := contexts.GetOrganizationID(ctx)

	var providers []*schema.SSOProvider
	query := r.db.NewSelect().
		Model(&providers).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	if !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	}

	err := query.Scan(ctx)
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// Delete removes an SSO provider by ID within the tenant scope
func (r *SSOProviderRepository) Delete(ctx context.Context, id xid.ID) error {
	// Extract tenant context for safety
	appID, _ := contexts.GetAppID(ctx)
	envID, _ := contexts.GetEnvironmentID(ctx)
	orgID, _ := contexts.GetOrganizationID(ctx)

	query := r.db.NewDelete().
		Model((*schema.SSOProvider)(nil)).
		Where("id = ?", id).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)

	if !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	}

	_, err := query.Exec(ctx)
	return err
}
