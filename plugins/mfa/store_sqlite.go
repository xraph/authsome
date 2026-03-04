package mfa

import (
	"context"
	"database/sql"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
)

// SqliteStore implements mfa.Store using the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore creates a new SQLite-backed MFA store.
func NewSqliteStore(db *grove.DB) *SqliteStore {
	return &SqliteStore{
		db:  db,
		sdb: sqlitedriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*SqliteStore)(nil)

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *SqliteStore) CreateEnrollment(ctx context.Context, e *Enrollment) error {
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = now
	}
	m := fromEnrollment(e)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return mfaSqliteError(err)
}

func (s *SqliteStore) GetEnrollment(ctx context.Context, userID id.UserID, method string) (*Enrollment, error) {
	m := new(enrollmentModel)
	err := s.sdb.NewSelect(m).
		Where("user_id = ?", userID.String()).
		Where("method = ?", method).
		Scan(ctx)
	if err != nil {
		return nil, mfaSqliteError(err)
	}
	return toEnrollment(m)
}

func (s *SqliteStore) GetEnrollmentByID(ctx context.Context, mfaID id.MFAID) (*Enrollment, error) {
	m := new(enrollmentModel)
	err := s.sdb.NewSelect(m).
		Where("id = ?", mfaID.String()).
		Scan(ctx)
	if err != nil {
		return nil, mfaSqliteError(err)
	}
	return toEnrollment(m)
}

func (s *SqliteStore) UpdateEnrollment(ctx context.Context, e *Enrollment) error {
	e.UpdatedAt = time.Now()
	m := fromEnrollment(e)
	_, err := s.sdb.NewUpdate(m).WherePK().Exec(ctx)
	return mfaSqliteError(err)
}

func (s *SqliteStore) DeleteEnrollment(ctx context.Context, mfaID id.MFAID) error {
	_, err := s.sdb.NewDelete((*enrollmentModel)(nil)).
		Where("id = ?", mfaID.String()).
		Exec(ctx)
	return mfaSqliteError(err)
}

func (s *SqliteStore) ListEnrollments(ctx context.Context, userID id.UserID) ([]*Enrollment, error) {
	var models []enrollmentModel
	err := s.sdb.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, mfaSqliteError(err)
	}
	result := make([]*Enrollment, 0, len(models))
	for i := range models {
		e, err := toEnrollment(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Recovery code store methods
// ──────────────────────────────────────────────────

func (s *SqliteStore) CreateRecoveryCodes(ctx context.Context, codes []*RecoveryCode) error {
	for _, c := range codes {
		m := &recoveryCodeModel{
			ID:        c.ID.String(),
			UserID:    c.UserID.String(),
			CodeHash:  c.CodeHash,
			Used:      false,
			CreatedAt: c.CreatedAt,
		}
		if _, err := s.sdb.NewInsert(m).Exec(ctx); err != nil {
			return mfaSqliteError(err)
		}
	}
	return nil
}

func (s *SqliteStore) GetRecoveryCodes(ctx context.Context, userID id.UserID) ([]*RecoveryCode, error) {
	var models []recoveryCodeModel
	err := s.sdb.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, mfaSqliteError(err)
	}
	result := make([]*RecoveryCode, 0, len(models))
	for i := range models {
		rc, err := toRecoveryCode(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, rc)
	}
	return result, nil
}

func (s *SqliteStore) ConsumeRecoveryCode(ctx context.Context, codeID id.RecoveryCodeID) error {
	now := time.Now()
	_, err := s.sdb.NewUpdate((*recoveryCodeModel)(nil)).
		Set("used = ?", true).
		Set("used_at = ?", now).
		Where("id = ?", codeID.String()).
		Where("used = ?", false).
		Exec(ctx)
	return mfaSqliteError(err)
}

func (s *SqliteStore) DeleteRecoveryCodes(ctx context.Context, userID id.UserID) error {
	_, err := s.sdb.NewDelete((*recoveryCodeModel)(nil)).
		Where("user_id = ?", userID.String()).
		Exec(ctx)
	return mfaSqliteError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func mfaSqliteError(err error) error {
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return ErrEnrollmentNotFound
	}
	return err
}
