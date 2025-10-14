package repository

import (
    "context"

    "github.com/uptrace/bun"
    "github.com/rs/xid"

    core "github.com/xraph/authsome/core/user"
    "github.com/xraph/authsome/schema"
)

// UserRepository is a Bun-backed implementation of core user repository
type UserRepository struct {
    db *bun.DB
}

func NewUserRepository(db *bun.DB) *UserRepository {
    return &UserRepository{db: db}
}

func (r *UserRepository) toSchema(u *core.User) *schema.User {
    return &schema.User{
        ID:             u.ID,
        Email:          u.Email,
        EmailVerified:  u.EmailVerified,
        EmailVerifiedAt: u.EmailVerifiedAt,
        Name:           u.Name,
        Image:          u.Image,
        PasswordHash:   u.PasswordHash,
        Username:       u.Username,
        DisplayUsername: u.DisplayUsername,
        // Auditable fields: default to self for dev/standalone creates
        AuditableModel: schema.AuditableModel{
            CreatedAt: u.CreatedAt,
            UpdatedAt: bun.NullTime{Time: u.UpdatedAt},
            CreatedBy: u.ID,
            UpdatedBy: u.ID,
        },
    }
}

func (r *UserRepository) fromSchema(su *schema.User) *core.User {
    if su == nil {
        return nil
    }
    return &core.User{
        ID:             su.ID,
        Email:          su.Email,
        EmailVerified:  su.EmailVerified,
        EmailVerifiedAt: su.EmailVerifiedAt,
        Name:           su.Name,
        Image:          su.Image,
        PasswordHash:   su.PasswordHash,
        Username:       su.Username,
        DisplayUsername: su.DisplayUsername,
        CreatedAt:      su.CreatedAt,
        UpdatedAt:      su.UpdatedAt.Time,
    }
}

// Create inserts a new user
func (r *UserRepository) Create(ctx context.Context, u *core.User) error {
    su := r.toSchema(u)
    _, err := r.db.NewInsert().Model(su).Exec(ctx)
    return err
}

// FindByID finds a user by id
func (r *UserRepository) FindByID(ctx context.Context, id xid.ID) (*core.User, error) {
    su := new(schema.User)
    err := r.db.NewSelect().Model(su).Where("id = ?", id).Scan(ctx)
    if err != nil {
        return nil, err
    }
    return r.fromSchema(su), nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*core.User, error) {
    su := new(schema.User)
    err := r.db.NewSelect().Model(su).Where("email = ?", email).Scan(ctx)
    if err != nil {
        return nil, err
    }
    return r.fromSchema(su), nil
}

// FindByUsername finds a user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*core.User, error) {
    su := new(schema.User)
    err := r.db.NewSelect().Model(su).Where("username = ?", username).Scan(ctx)
    if err != nil {
        return nil, err
    }
    return r.fromSchema(su), nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, u *core.User) error {
    su := r.toSchema(u)
    _, err := r.db.NewUpdate().Model(su).WherePK().Exec(ctx)
    return err
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id xid.ID) error {
    _, err := r.db.NewDelete().Model((*schema.User)(nil)).Where("id = ?", id).Exec(ctx)
    return err
}

// List returns a list of users
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*core.User, error) {
    var sus []schema.User
    err := r.db.NewSelect().Model(&sus).OrderExpr("created_at DESC").Limit(limit).Offset(offset).Scan(ctx)
    if err != nil {
        return nil, err
    }
    res := make([]*core.User, 0, len(sus))
    for i := range sus {
        res = append(res, r.fromSchema(&sus[i]))
    }
    return res, nil
}

// Count returns total users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
    count, err := r.db.NewSelect().Model((*schema.User)(nil)).Count(ctx)
    return count, err
}