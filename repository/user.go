package repository

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/core/pagination"
	core "github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// USER REPOSITORY IMPLEMENTATION
// =============================================================================

// UserRepository is a Bun-backed implementation of core user repository.
type UserRepository struct {
	db *bun.DB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(db *bun.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, user *schema.User) error {
	_, err := r.db.NewInsert().Model(user).Exec(ctx)

	return err
}

// FindByID finds a user by ID.
func (r *UserRepository) FindByID(ctx context.Context, id xid.ID) (*schema.User, error) {
	user := new(schema.User)

	err := r.db.NewSelect().
		Model(user).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByEmail finds a user by email (global search, not app-scoped).
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*schema.User, error) {
	user := new(schema.User)

	err := r.db.NewSelect().
		Model(user).
		Where("email = ?", email).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByAppAndEmail finds a user by app ID and email (app-scoped search).
func (r *UserRepository) FindByAppAndEmail(ctx context.Context, appID xid.ID, email string) (*schema.User, error) {
	user := new(schema.User)

	err := r.db.NewSelect().
		Model(user).
		Where("app_id = ?", appID).
		Where("email = ?", email).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// FindByUsername finds a user by username.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*schema.User, error) {
	user := new(schema.User)

	err := r.db.NewSelect().
		Model(user).
		Where("username = ?", username).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Update updates a user.
func (r *UserRepository) Update(ctx context.Context, user *schema.User) error {
	_, err := r.db.NewUpdate().
		Model(user).
		WherePK().
		Exec(ctx)

	return err
}

// Delete deletes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, id xid.ID) error {
	_, err := r.db.NewDelete().
		Model((*schema.User)(nil)).
		Where("id = ?", id).
		Exec(ctx)

	return err
}

// ListUsers lists users with pagination and filtering.
func (r *UserRepository) ListUsers(ctx context.Context, filter *core.ListUsersFilter) (*pagination.PageResponse[*schema.User], error) {
	var users []*schema.User

	query := r.db.NewSelect().Model(&users)

	// Filter by app ID (required)
	query = query.Where("app_id = ?", filter.AppID)

	// Filter by email verified status
	if filter.EmailVerified != nil {
		query = query.Where("email_verified = ?", *filter.EmailVerified)
	}

	// Search by email or name
	if filter.Search != nil && *filter.Search != "" {
		searchPattern := "%" + *filter.Search + "%"
		query = query.Where("(LOWER(email) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?))", searchPattern, searchPattern)
	}

	// Apply ordering
	query = query.Order(filter.GetOrderClause())

	// Apply pagination
	query = query.Limit(filter.GetLimit()).Offset(filter.GetOffset())

	// Execute query and get total count
	total, err := query.ScanAndCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Return paginated response
	return pagination.NewPageResponse(users, int64(total), &filter.PaginationParams), nil
}

// CountUsers counts users with filtering.
func (r *UserRepository) CountUsers(ctx context.Context, filter *core.CountUsersFilter) (int, error) {
	query := r.db.NewSelect().Model((*schema.User)(nil))

	// Filter by app ID (required)
	query = query.Where("app_id = ?", filter.AppID)

	// Filter by creation date
	if filter.CreatedSince != nil {
		query = query.Where("created_at >= ?", *filter.CreatedSince)
	}

	count, err := query.Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}
