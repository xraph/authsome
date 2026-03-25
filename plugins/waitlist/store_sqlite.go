package waitlist

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"

	"github.com/xraph/authsome/id"
)

// SqliteStore implements waitlist.Store using the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore creates a new SQLite-backed waitlist store.
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

func (s *SqliteStore) CreateEntry(ctx context.Context, e *WaitlistEntry) error {
	now := time.Now()
	if e.ID.IsNil() {
		e.ID = id.NewWaitlistID()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now

	m := fromWaitlistEntry(e)
	_, err := s.sdb.NewInsert(m).Exec(ctx)
	return waitlistSqliteError(err)
}

func (s *SqliteStore) GetEntry(ctx context.Context, entryID id.WaitlistID) (*WaitlistEntry, error) {
	m := new(waitlistModel)
	err := s.sdb.NewSelect(m).
		Where("id = ?", entryID.String()).
		Scan(ctx)
	if err != nil {
		return nil, waitlistSqliteError(err)
	}
	return toWaitlistEntry(m)
}

func (s *SqliteStore) GetEntryByEmail(ctx context.Context, appID id.AppID, email string) (*WaitlistEntry, error) {
	m := new(waitlistModel)
	err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("email = ?", strings.ToLower(email)).
		Scan(ctx)
	if err != nil {
		return nil, waitlistSqliteError(err)
	}
	return toWaitlistEntry(m)
}

func (s *SqliteStore) UpdateEntryStatus(ctx context.Context, entryID id.WaitlistID, status WaitlistStatus, note string) error {
	now := time.Now()
	res, err := s.sdb.NewUpdate((*waitlistModel)(nil)).
		Set("status = ?", string(status)).
		Set("note = ?", note).
		Set("updated_at = ?", now).
		Where("id = ?", entryID.String()).
		Exec(ctx)
	if err != nil {
		return waitlistSqliteError(err)
	}

	rows, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) ListEntries(ctx context.Context, q *WaitlistQuery) (*WaitlistList, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	query := s.sdb.NewSelect((*waitlistModel)(nil))

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
		return nil, waitlistSqliteError(err)
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

func (s *SqliteStore) CountByStatus(ctx context.Context, appID id.AppID) (pending int, approved int, rejected int, err error) {
	type countRow struct {
		Status string `grove:"status"`
		Count  int    `grove:"count"`
	}

	var rows []countRow
	err = s.sdb.NewRaw(
		"SELECT status, COUNT(*) AS count FROM authsome_waitlist_entries WHERE app_id = ? GROUP BY status",
		appID.String(),
	).Scan(ctx, &rows)
	if err != nil {
		return 0, 0, 0, waitlistSqliteError(err)
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

func (s *SqliteStore) DeleteEntry(ctx context.Context, entryID id.WaitlistID) error {
	res, err := s.sdb.NewDelete((*waitlistModel)(nil)).
		Where("id = ?", entryID.String()).
		Exec(ctx)
	if err != nil {
		return waitlistSqliteError(err)
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

func waitlistSqliteError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return ErrDuplicateEmail
	}
	return err
}
