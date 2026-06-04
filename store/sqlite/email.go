package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/xraph/grove"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// ──────────────────────────────────────────────────
// User email model (multiple emails per account)
// ──────────────────────────────────────────────────

type UserEmailModel struct {
	grove.BaseModel `grove:"table:authsome_user_emails,alias:ue"`

	ID        string       `grove:"id,pk"`
	UserID    string       `grove:"user_id,notnull"`
	AppID     string       `grove:"app_id,notnull"`
	EnvID     string       `grove:"env_id,notnull"`
	Email     string       `grove:"email,notnull"`
	Verified  bool         `grove:"verified"`
	IsPrimary bool         `grove:"is_primary"`
	Source    string       `grove:"source"`
	CreatedAt time.Time    `grove:"created_at,notnull,default:now()"`
	UpdatedAt time.Time    `grove:"updated_at,notnull,default:now()"`
	DeletedAt sql.NullTime `grove:"deleted_at"`
}

func toUserEmail(m *UserEmailModel) (*user.UserEmail, error) {
	ueID, err := id.ParseUserEmailID(m.ID)
	if err != nil {
		return nil, err
	}
	uid, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	appID, err := id.ParseAppID(m.AppID)
	if err != nil {
		return nil, err
	}
	envID, err := id.ParseEnvironmentID(m.EnvID)
	if err != nil {
		return nil, err
	}
	e := &user.UserEmail{
		ID:        ueID,
		UserID:    uid,
		AppID:     appID,
		EnvID:     envID,
		Email:     m.Email,
		Verified:  m.Verified,
		IsPrimary: m.IsPrimary,
		Source:    m.Source,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.DeletedAt.Valid {
		e.DeletedAt = &m.DeletedAt.Time
	}
	return e, nil
}

func fromUserEmail(e *user.UserEmail) *UserEmailModel {
	now := time.Now()
	m := &UserEmailModel{
		ID:        e.ID.String(),
		UserID:    e.UserID.String(),
		AppID:     e.AppID.String(),
		EnvID:     e.EnvID.String(),
		Email:     user.NormalizeEmail(e.Email),
		Verified:  e.Verified,
		IsPrimary: e.IsPrimary,
		Source:    e.Source,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
	if m.ID == "" {
		m.ID = id.NewUserEmailID().String()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = now
	}
	if e.DeletedAt != nil {
		m.DeletedAt = sql.NullTime{Time: *e.DeletedAt, Valid: true}
	}
	return m
}

// ──────────────────────────────────────────────────
// User email store methods
// ──────────────────────────────────────────────────

func (s *Store) CreateUserWithPrimaryEmail(ctx context.Context, u *user.User, primary *user.UserEmail) error {
	stx, err := s.sdb.BeginTxQuery(ctx, nil)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: begin tx for user+email: %w", err)
	}
	defer func() { _ = stx.Rollback() }() //nolint:errcheck // best-effort rollback after commit

	if _, err := stx.NewInsert(fromUser(u)).Exec(ctx); err != nil {
		return sqliteError(err)
	}
	pm := fromUserEmail(primary)
	pm.IsPrimary = true
	if _, err := stx.NewInsert(pm).Exec(ctx); err != nil {
		return sqliteError(err)
	}
	if err := stx.Commit(); err != nil {
		return fmt.Errorf("authsome/sqlite: commit user+email: %w", err)
	}
	return nil
}

func (s *Store) AddUserEmail(ctx context.Context, e *user.UserEmail) error {
	_, err := s.sdb.NewInsert(fromUserEmail(e)).Exec(ctx)
	return sqliteError(err)
}

func (s *Store) GetUserByAnyEmail(ctx context.Context, appID id.AppID, envID id.EnvironmentID, email string) (*user.User, error) {
	m := new(UserEmailModel)
	q := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("email = ?", user.NormalizeEmail(email)).
		Where("deleted_at IS NULL")
	if !envID.IsNil() {
		q = q.Where("env_id = ?", envID.String())
	}
	if err := q.Scan(ctx); err != nil {
		return nil, sqliteError(err)
	}
	uid, err := id.ParseUserID(m.UserID)
	if err != nil {
		return nil, err
	}
	return s.GetUser(ctx, uid)
}

func (s *Store) GetUserEmailRecord(ctx context.Context, appID id.AppID, envID id.EnvironmentID, email string) (*user.UserEmail, error) {
	m := new(UserEmailModel)
	q := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("email = ?", user.NormalizeEmail(email)).
		Where("deleted_at IS NULL")
	if !envID.IsNil() {
		q = q.Where("env_id = ?", envID.String())
	}
	if err := q.Scan(ctx); err != nil {
		return nil, sqliteError(err)
	}
	return toUserEmail(m)
}

func (s *Store) GetUserEmails(ctx context.Context, userID id.UserID) ([]*user.UserEmail, error) {
	var models []UserEmailModel
	err := s.sdb.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		Where("deleted_at IS NULL").
		OrderExpr("is_primary DESC, created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, sqliteError(err)
	}
	out := make([]*user.UserEmail, 0, len(models))
	for i := range models {
		e, err := toUserEmail(&models[i])
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

func (s *Store) MarkUserEmailVerified(ctx context.Context, userID id.UserID, email string) error {
	m := new(UserEmailModel)
	err := s.sdb.NewSelect(m).
		Where("user_id = ?", userID.String()).
		Where("email = ?", user.NormalizeEmail(email)).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return sqliteError(err)
	}

	stx, err := s.sdb.BeginTxQuery(ctx, nil)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: begin tx mark verified: %w", err)
	}
	defer func() { _ = stx.Rollback() }() //nolint:errcheck // best-effort rollback after commit

	now := time.Now()
	if _, err := stx.NewUpdate((*UserEmailModel)(nil)).
		Set("verified = ?", true).
		Set("updated_at = ?", now).
		Where("id = ?", m.ID).Exec(ctx); err != nil {
		return sqliteError(err)
	}
	if m.IsPrimary {
		if _, err := stx.NewUpdate((*UserModel)(nil)).
			Set("email_verified = ?", true).
			Set("updated_at = ?", now).
			Where("id = ?", userID.String()).Exec(ctx); err != nil {
			return sqliteError(err)
		}
	}
	if err := stx.Commit(); err != nil {
		return fmt.Errorf("authsome/sqlite: commit mark verified: %w", err)
	}
	return nil
}

func (s *Store) SetPrimaryEmail(ctx context.Context, userID id.UserID, email string) error {
	target := new(UserEmailModel)
	err := s.sdb.NewSelect(target).
		Where("user_id = ?", userID.String()).
		Where("email = ?", user.NormalizeEmail(email)).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return sqliteError(err)
	}
	if !target.Verified {
		return account.ErrEmailNotVerified
	}

	stx, err := s.sdb.BeginTxQuery(ctx, nil)
	if err != nil {
		return fmt.Errorf("authsome/sqlite: begin tx set primary: %w", err)
	}
	defer func() { _ = stx.Rollback() }() //nolint:errcheck // best-effort rollback after commit

	now := time.Now()
	// Clear the existing primary BEFORE setting the new one so the partial
	// unique index on (user_id) WHERE is_primary never sees two primaries.
	if _, err := stx.NewUpdate((*UserEmailModel)(nil)).
		Set("is_primary = ?", false).
		Set("updated_at = ?", now).
		Where("user_id = ?", userID.String()).
		Where("is_primary = ?", true).Exec(ctx); err != nil {
		return sqliteError(err)
	}
	if _, err := stx.NewUpdate((*UserEmailModel)(nil)).
		Set("is_primary = ?", true).
		Set("updated_at = ?", now).
		Where("id = ?", target.ID).Exec(ctx); err != nil {
		return sqliteError(err)
	}
	if _, err := stx.NewUpdate((*UserModel)(nil)).
		Set("email = ?", target.Email).
		Set("email_verified = ?", target.Verified).
		Set("updated_at = ?", now).
		Where("id = ?", userID.String()).Exec(ctx); err != nil {
		return sqliteError(err)
	}
	if err := stx.Commit(); err != nil {
		return fmt.Errorf("authsome/sqlite: commit set primary: %w", err)
	}
	return nil
}

func (s *Store) DeleteUserEmail(ctx context.Context, userID id.UserID, email string) error {
	m := new(UserEmailModel)
	err := s.sdb.NewSelect(m).
		Where("user_id = ?", userID.String()).
		Where("email = ?", user.NormalizeEmail(email)).
		Where("deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return sqliteError(err)
	}
	if m.IsPrimary {
		return store.ErrConflict
	}
	now := time.Now()
	_, err = s.sdb.NewUpdate((*UserEmailModel)(nil)).
		Set("deleted_at = ?", now).
		Set("updated_at = ?", now).
		Where("id = ?", m.ID).Exec(ctx)
	return sqliteError(err)
}
