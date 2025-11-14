package repository

import (
	"context"
	"database/sql"

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

// FindByClientID returns an OAuthClient by client_id
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
