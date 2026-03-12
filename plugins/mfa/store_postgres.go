package mfa

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/id"
)

// PostgresStore implements mfa.Store using the Grove ORM with PostgreSQL.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// NewPostgresStore creates a new PostgreSQL-backed MFA store.
func NewPostgresStore(db *grove.DB) *PostgresStore {
	return &PostgresStore{
		db: db,
		pg: pgdriver.Unwrap(db),
	}
}

// Compile-time interface check.
var _ Store = (*PostgresStore)(nil)

// ──────────────────────────────────────────────────
// Store methods
// ──────────────────────────────────────────────────

func (s *PostgresStore) CreateEnrollment(ctx context.Context, e *Enrollment) error {
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	if e.UpdatedAt.IsZero() {
		e.UpdatedAt = now
	}
	m := fromEnrollment(e)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return mfaPgError(err)
}

func (s *PostgresStore) GetEnrollment(ctx context.Context, userID id.UserID, method string) (*Enrollment, error) {
	m := new(enrollmentModel)
	err := s.pg.NewSelect(m).
		Where("user_id = ?", userID.String()).
		Where("method = ?", method).
		Scan(ctx)
	if err != nil {
		return nil, mfaPgError(err)
	}
	return toEnrollment(m)
}

func (s *PostgresStore) GetEnrollmentByID(ctx context.Context, mfaID id.MFAID) (*Enrollment, error) {
	m := new(enrollmentModel)
	err := s.pg.NewSelect(m).
		Where("id = ?", mfaID.String()).
		Scan(ctx)
	if err != nil {
		return nil, mfaPgError(err)
	}
	return toEnrollment(m)
}

func (s *PostgresStore) UpdateEnrollment(ctx context.Context, e *Enrollment) error {
	e.UpdatedAt = time.Now()
	m := fromEnrollment(e)
	_, err := s.pg.NewUpdate(m).WherePK().Exec(ctx)
	return mfaPgError(err)
}

func (s *PostgresStore) DeleteEnrollment(ctx context.Context, mfaID id.MFAID) error {
	_, err := s.pg.NewDelete((*enrollmentModel)(nil)).
		Where("id = ?", mfaID.String()).
		Exec(ctx)
	return mfaPgError(err)
}

func (s *PostgresStore) ListEnrollments(ctx context.Context, userID id.UserID) ([]*Enrollment, error) {
	var models []enrollmentModel
	err := s.pg.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, mfaPgError(err)
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

func (s *PostgresStore) CreateRecoveryCodes(ctx context.Context, codes []*RecoveryCode) error {
	for _, c := range codes {
		m := &recoveryCodeModel{
			ID:        c.ID.String(),
			UserID:    c.UserID.String(),
			CodeHash:  c.CodeHash,
			Used:      false,
			CreatedAt: c.CreatedAt,
		}
		if _, err := s.pg.NewInsert(m).Exec(ctx); err != nil {
			return mfaPgError(err)
		}
	}
	return nil
}

func (s *PostgresStore) GetRecoveryCodes(ctx context.Context, userID id.UserID) ([]*RecoveryCode, error) {
	var models []recoveryCodeModel
	err := s.pg.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		OrderExpr("created_at ASC").
		Scan(ctx)
	if err != nil {
		return nil, mfaPgError(err)
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

func (s *PostgresStore) ConsumeRecoveryCode(ctx context.Context, codeID id.RecoveryCodeID) error {
	now := time.Now()
	_, err := s.pg.NewUpdate((*recoveryCodeModel)(nil)).
		Set("used = ?", true).
		Set("used_at = ?", now).
		Where("id = ?", codeID.String()).
		Where("used = ?", false).
		Exec(ctx)
	return mfaPgError(err)
}

func (s *PostgresStore) DeleteRecoveryCodes(ctx context.Context, userID id.UserID) error {
	_, err := s.pg.NewDelete((*recoveryCodeModel)(nil)).
		Where("user_id = ?", userID.String()).
		Exec(ctx)
	return mfaPgError(err)
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func mfaPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrEnrollmentNotFound
	}
	return err
}
