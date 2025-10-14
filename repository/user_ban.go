package repository

import (
	"context"
	"time"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

// UserBanRepository implements the user.BanRepository interface using Bun ORM
type UserBanRepository struct {
	db *bun.DB
}

// NewUserBanRepository creates a new user ban repository
func NewUserBanRepository(db *bun.DB) *UserBanRepository {
	return &UserBanRepository{db: db}
}

// CreateBan creates a new user ban record
func (r *UserBanRepository) CreateBan(ctx context.Context, ban *schema.UserBan) error {
	_, err := r.db.NewInsert().Model(ban).Exec(ctx)
	return err
}

// FindActiveBan finds an active ban for a user
func (r *UserBanRepository) FindActiveBan(ctx context.Context, userID string) (*schema.UserBan, error) {
	ban := &schema.UserBan{}
	err := r.db.NewSelect().
		Model(ban).
		Where("user_id = ?", userID).
		Where("is_active = ?", true).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Order("created_at DESC").
		Limit(1).
		Scan(ctx)
	
	if err != nil {
		return nil, err
	}
	
	return ban, nil
}

// FindBansByUser finds all bans for a user (active and inactive)
func (r *UserBanRepository) FindBansByUser(ctx context.Context, userID string) ([]*schema.UserBan, error) {
	var bans []*schema.UserBan
	err := r.db.NewSelect().
		Model(&bans).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Scan(ctx)
	
	if err != nil {
		return nil, err
	}
	
	return bans, nil
}

// UpdateBan updates an existing ban record
func (r *UserBanRepository) UpdateBan(ctx context.Context, ban *schema.UserBan) error {
	_, err := r.db.NewUpdate().
		Model(ban).
		Where("id = ?", ban.ID).
		Exec(ctx)
	return err
}

// FindBanByID finds a ban by its ID
func (r *UserBanRepository) FindBanByID(ctx context.Context, banID string) (*schema.UserBan, error) {
	ban := &schema.UserBan{}
	err := r.db.NewSelect().
		Model(ban).
		Where("id = ?", banID).
		Scan(ctx)
	
	if err != nil {
		return nil, err
	}
	
	return ban, nil
}