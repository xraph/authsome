package consent

import (
	"context"
	"database/sql"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
)

// SqliteStore implements consent.Store using the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore creates a new SQLite-backed consent store.
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

func (s *SqliteStore) GrantConsent(ctx context.Context, c *Consent) error {
	now := time.Now()
	if c.ID.IsNil() {
		c.ID = id.NewConsentID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	// Try to update existing record first.
	m := fromConsent(c)
	res, err := s.sdb.NewUpdate(m).
		Set("granted = ?", m.Granted).
		Set("version = ?", m.Version).
		Set("ip_address = ?", m.IPAddress).
		Set("granted_at = ?", m.GrantedAt).
		Set("revoked_at = ?", m.RevokedAt).
		Set("updated_at = ?", m.UpdatedAt).
		Where("user_id = ?", m.UserID).
		Where("app_id = ?", m.AppID).
		Where("purpose = ?", m.Purpose).
		Exec(ctx)
	if err != nil {
		return consentSqliteError(err)
	}

	rows, _ := res.RowsAffected()
	if rows > 0 {
		return nil
	}

	// No existing record — insert new one.
	_, err = s.sdb.NewInsert(m).Exec(ctx)
	return consentSqliteError(err)
}

func (s *SqliteStore) RevokeConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) error {
	now := time.Now()
	res, err := s.sdb.NewUpdate((*consentModel)(nil)).
		Set("granted = ?", false).
		Set("revoked_at = ?", sql.NullTime{Time: now, Valid: true}).
		Set("updated_at = ?", now).
		Where("user_id = ?", userID.String()).
		Where("app_id = ?", appID.String()).
		Where("purpose = ?", purpose).
		Exec(ctx)
	if err != nil {
		return consentSqliteError(err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) GetConsent(ctx context.Context, userID id.UserID, appID id.AppID, purpose string) (*Consent, error) {
	m := new(consentModel)
	err := s.sdb.NewSelect(m).
		Where("user_id = ?", userID.String()).
		Where("app_id = ?", appID.String()).
		Where("purpose = ?", purpose).
		Scan(ctx)
	if err != nil {
		return nil, consentSqliteError(err)
	}
	return toConsent(m)
}

func (s *SqliteStore) ListConsents(ctx context.Context, q *Query) ([]*Consent, string, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	query := s.sdb.NewSelect((*consentModel)(nil))

	if q.UserID.Prefix() != "" {
		query = query.Where("user_id = ?", q.UserID.String())
	}
	if q.AppID.Prefix() != "" {
		query = query.Where("app_id = ?", q.AppID.String())
	}
	if q.Purpose != "" {
		query = query.Where("purpose = ?", q.Purpose)
	}
	if q.Cursor != "" {
		query = query.Where("id > ?", q.Cursor)
	}

	var models []consentModel
	err := query.
		OrderExpr("id ASC").
		Limit(limit + 1).
		Scan(ctx, &models)
	if err != nil {
		return nil, "", consentSqliteError(err)
	}

	var cursor string
	if len(models) > limit {
		cursor = models[limit-1].ID
		models = models[:limit]
	}

	result := make([]*Consent, 0, len(models))
	for i := range models {
		c, err := toConsent(&models[i])
		if err != nil {
			return nil, "", err
		}
		result = append(result, c)
	}

	return result, cursor, nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func consentSqliteError(err error) error {
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	return err
}
