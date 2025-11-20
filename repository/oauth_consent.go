package repository

import (
	"context"
	"database/sql"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// OAuthConsentRepository handles OAuth consent persistence
type OAuthConsentRepository struct {
	db *bun.DB
}

// NewOAuthConsentRepository creates a new OAuth consent repository
func NewOAuthConsentRepository(db *bun.DB) *OAuthConsentRepository {
	return &OAuthConsentRepository{db: db}
}

// Create stores a new consent decision
func (r *OAuthConsentRepository) Create(ctx context.Context, consent *schema.OAuthConsent) error {
	_, err := r.db.NewInsert().Model(consent).Exec(ctx)
	return err
}

// FindByUserAndClient retrieves consent for a user and client
func (r *OAuthConsentRepository) FindByUserAndClient(ctx context.Context, userID xid.ID, clientID string, appID, envID xid.ID, orgID *xid.ID) (*schema.OAuthConsent, error) {
	consent := &schema.OAuthConsent{}
	query := r.db.NewSelect().Model(consent).
		Where("user_id = ?", userID).
		Where("client_id = ?", clientID).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID)
	
	if orgID != nil && !orgID.IsNil() {
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
	return consent, nil
}

// ListByUser retrieves all consents for a user
func (r *OAuthConsentRepository) ListByUser(ctx context.Context, userID xid.ID, appID, envID xid.ID, orgID *xid.ID) ([]*schema.OAuthConsent, error) {
	var consents []*schema.OAuthConsent
	query := r.db.NewSelect().Model(&consents).
		Where("user_id = ?", userID).
		Where("app_id = ?", appID).
		Where("environment_id = ?", envID).
		Order("created_at DESC")
	
	if orgID != nil && !orgID.IsNil() {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("organization_id IS NULL")
	}
	
	err := query.Scan(ctx)
	return consents, err
}

// Update updates an existing consent
func (r *OAuthConsentRepository) Update(ctx context.Context, consent *schema.OAuthConsent) error {
	_, err := r.db.NewUpdate().Model(consent).WherePK().Exec(ctx)
	return err
}

// Delete removes a consent
func (r *OAuthConsentRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().Model((*schema.OAuthConsent)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}

// DeleteByUserAndClient removes consent for a specific user and client
func (r *OAuthConsentRepository) DeleteByUserAndClient(ctx context.Context, userID xid.ID, clientID string) error {
	_, err := r.db.NewDelete().Model((*schema.OAuthConsent)(nil)).
		Where("user_id = ? AND client_id = ?", userID, clientID).
		Exec(ctx)
	return err
}

// DeleteExpired removes expired consents
func (r *OAuthConsentRepository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.NewDelete().
		Model((*schema.OAuthConsent)(nil)).
		Where("expires_at IS NOT NULL").
		Where("expires_at < NOW()").
		Exec(ctx)
	return err
}

// HasValidConsent checks if user has valid consent for client with required scopes
func (r *OAuthConsentRepository) HasValidConsent(ctx context.Context, userID xid.ID, clientID string, requiredScopes []string, appID, envID xid.ID, orgID *xid.ID) (bool, error) {
	consent, err := r.FindByUserAndClient(ctx, userID, clientID, appID, envID, orgID)
	if err != nil || consent == nil {
		return false, err
	}
	
	// Check if consent is expired
	if !consent.IsValid() {
		return false, nil
	}
	
	// Check if all required scopes are granted
	for _, required := range requiredScopes {
		if !consent.HasScope(required) {
			return false, nil
		}
	}
	
	return true, nil
}

