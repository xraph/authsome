package repository

import (
    "context"
    "database/sql"
    "github.com/uptrace/bun"
    "github.com/xraph/authsome/schema"
)

// SSOProviderRepository provides persistence for SSO provider configurations
type SSOProviderRepository struct{ db *bun.DB }

func NewSSOProviderRepository(db *bun.DB) *SSOProviderRepository { return &SSOProviderRepository{db: db} }

// Create inserts a new SSOProvider record
func (r *SSOProviderRepository) Create(ctx context.Context, p *schema.SSOProvider) error {
    _, err := r.db.NewInsert().Model(p).Exec(ctx)
    return err
}

// Upsert creates or updates an SSOProvider by ProviderID
func (r *SSOProviderRepository) Upsert(ctx context.Context, p *schema.SSOProvider) error {
    // Try find existing
    existing := new(schema.SSOProvider)
    err := r.db.NewSelect().Model(existing).Where("provider_id = ?", p.ProviderID).Scan(ctx)
    if err == sql.ErrNoRows {
        _, err2 := r.db.NewInsert().Model(p).Exec(ctx)
        return err2
    }
    if err != nil { return err }
    // Update existing with new fields
    p.ID = existing.ID
    _, err = r.db.NewUpdate().Model(p).WherePK().Exec(ctx)
    return err
}

// FindByProviderID returns an SSOProvider by ProviderID
func (r *SSOProviderRepository) FindByProviderID(ctx context.Context, providerID string) (*schema.SSOProvider, error) {
    p := new(schema.SSOProvider)
    err := r.db.NewSelect().Model(p).Where("provider_id = ?", providerID).Scan(ctx)
    if err == sql.ErrNoRows { return nil, nil }
    if err != nil { return nil, err }
    return p, nil
}