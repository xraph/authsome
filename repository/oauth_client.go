package repository

import (
	"context"
	"database/sql"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// OAuthClientRepository provides persistence for OAuth client registrations
type OAuthClientRepository struct{ db *bun.DB }

func NewOAuthClientRepository(db *bun.DB) *OAuthClientRepository {
	return &OAuthClientRepository{db: db}
}

// Create inserts a new OAuthClient record
func (r *OAuthClientRepository) Create(ctx context.Context, c *schema.OAuthClient) error {
	_, err := r.db.NewInsert().Model(c).Exec(ctx)
	return err
}

// FindByClientID returns an OAuthClient by client_id (no context filtering)
func (r *OAuthClientRepository) FindByClientID(ctx context.Context, clientID string) (*schema.OAuthClient, error) {
	c := new(schema.OAuthClient)
	err := r.db.NewSelect().Model(c).Where("client_id = ?", clientID).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// FindByClientIDWithContext returns an OAuthClient with org hierarchy support
// Tries org-specific client first, then falls back to app-level
func (r *OAuthClientRepository) FindByClientIDWithContext(ctx context.Context, appID, envID xid.ID, orgID *xid.ID, clientID string) (*schema.OAuthClient, error) {
	c := new(schema.OAuthClient)

	// If orgID provided, try org-specific client first
	if orgID != nil && !orgID.IsNil() {
		err := r.db.NewSelect().Model(c).
			Where("client_id = ?", clientID).
			Where("app_id = ?", appID).
			Where("environment_id = ?", envID).
			Where("organization_id = ?", orgID).
			Scan(ctx)
		if err == nil {
			return c, nil
		}
		if err != sql.ErrNoRows {
			return nil, err
		}
	}

	// Fall back to app-level client (organization_id IS NULL)
	err := r.db.NewSelect().Model(c).
		Where("client_id = ?", clientID).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id IS NULL").
		Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// FindByID returns an OAuthClient by ID
func (r *OAuthClientRepository) FindByID(ctx context.Context, id xid.ID) (*schema.OAuthClient, error) {
	c := new(schema.OAuthClient)
	err := r.db.NewSelect().Model(c).Where("id = ?", id).Scan(ctx)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ListByApp returns all clients for an app and environment
func (r *OAuthClientRepository) ListByApp(ctx context.Context, appID, envID xid.ID, limit, offset int) ([]*schema.OAuthClient, int, error) {
	var clients []*schema.OAuthClient

	query := r.db.NewSelect().Model(&clients).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	// Get total count
	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

// ListByOrg returns all org-specific clients
func (r *OAuthClientRepository) ListByOrg(ctx context.Context, appID, envID, orgID xid.ID, limit, offset int) ([]*schema.OAuthClient, int, error) {
	var clients []*schema.OAuthClient

	query := r.db.NewSelect().Model(&clients).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Where("organization_id = ?", orgID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

// Update updates an existing OAuth client
func (r *OAuthClientRepository) Update(ctx context.Context, c *schema.OAuthClient) error {
	_, err := r.db.NewUpdate().Model(c).WherePK().Exec(ctx)
	return err
}

// Delete removes an OAuth client
func (r *OAuthClientRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.OAuthClient)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// ExistsByClientID checks if a client with the given client_id exists
func (r *OAuthClientRepository) ExistsByClientID(ctx context.Context, clientID string) (bool, error) {
	exists, err := r.db.NewSelect().Model((*schema.OAuthClient)(nil)).
		Where("client_id = ?", clientID).
		Exists(ctx)
	return exists, err
}
