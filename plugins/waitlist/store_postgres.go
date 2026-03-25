package waitlist

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/id"
)

// PostgresStore implements waitlist.Store using the Grove ORM with PostgreSQL.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// NewPostgresStore creates a new PostgreSQL-backed waitlist store.
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

func (s *PostgresStore) CreateEntry(ctx context.Context, e *WaitlistEntry) error {
	now := time.Now()
	if e.ID.IsNil() {
		e.ID = id.NewWaitlistID()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now

	m := fromWaitlistEntry(e)
	_, err := s.pg.NewInsert(m).Exec(ctx)
	return waitlistPgError(err)
}

func (s *PostgresStore) GetEntry(ctx context.Context, entryID id.WaitlistID) (*WaitlistEntry, error) {
	m := new(waitlistModel)
	err := s.pg.NewSelect(m).
		Where("id = ?", entryID.String()).
		Scan(ctx)
	if err != nil {
		return nil, waitlistPgError(err)
	}
	return toWaitlistEntry(m)
}

func (s *PostgresStore) GetEntryByEmail(ctx context.Context, appID id.AppID, email string) (*WaitlistEntry, error) {
	m := new(waitlistModel)
	err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("email = ?", strings.ToLower(email)).
		Scan(ctx)
	if err != nil {
		return nil, waitlistPgError(err)
	}
	return toWaitlistEntry(m)
}

func (s *PostgresStore) UpdateEntryStatus(ctx context.Context, entryID id.WaitlistID, status WaitlistStatus, note string) error {
	now := time.Now()
	res, err := s.pg.NewUpdate((*waitlistModel)(nil)).
		Set("status = ?", string(status)).
		Set("note = ?", note).
		Set("updated_at = ?", now).
		Where("id = ?", entryID.String()).
		Exec(ctx)
	if err != nil {
		return waitlistPgError(err)
	}

	rows, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) ListEntries(ctx context.Context, q *WaitlistQuery) (*WaitlistList, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	query := s.pg.NewSelect((*waitlistModel)(nil))

	if q.AppID.Prefix() != "" {
		query = query.Where("app_id = ?", q.AppID.String())
	}
	if q.Email != "" {
		query = query.Where("email = ?", strings.ToLower(q.Email))
	}
	if q.Status != "" {
		query = query.Where("status = ?", string(q.Status))
	}
	if q.Cursor != "" {
		query = query.Where("id > ?", q.Cursor)
	}

	var models []waitlistModel
	err := query.
		OrderExpr("id ASC").
		Limit(limit + 1).
		Scan(ctx, &models)
	if err != nil {
		return nil, waitlistPgError(err)
	}

	var cursor string
	if len(models) > limit {
		cursor = models[limit-1].ID
		models = models[:limit]
	}

	entries := make([]*WaitlistEntry, 0, len(models))
	for i := range models {
		e, err := toWaitlistEntry(&models[i])
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return &WaitlistList{
		Entries:    entries,
		Total:      len(entries),
		NextCursor: cursor,
	}, nil
}

func (s *PostgresStore) CountByStatus(ctx context.Context, appID id.AppID) (pending int, approved int, rejected int, err error) {
	type countRow struct {
		Status string `grove:"status"`
		Count  int    `grove:"count"`
	}

	var rows []countRow
	err = s.pg.NewRaw(
		"SELECT status, COUNT(*) AS count FROM authsome_waitlist_entries WHERE app_id = ? GROUP BY status",
		appID.String(),
	).Scan(ctx, &rows)
	if err != nil {
		return 0, 0, 0, waitlistPgError(err)
	}

	for _, r := range rows {
		switch WaitlistStatus(r.Status) {
		case StatusPending:
			pending = r.Count
		case StatusApproved:
			approved = r.Count
		case StatusRejected:
			rejected = r.Count
		}
	}
	return pending, approved, rejected, nil
}

func (s *PostgresStore) DeleteEntry(ctx context.Context, entryID id.WaitlistID) error {
	res, err := s.pg.NewDelete((*waitlistModel)(nil)).
		Where("id = ?", entryID.String()).
		Exec(ctx)
	if err != nil {
		return waitlistPgError(err)
	}

	rows, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func waitlistPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	// Check for unique constraint violation (duplicate email).
	if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
		return ErrDuplicateEmail
	}
	return err
}
