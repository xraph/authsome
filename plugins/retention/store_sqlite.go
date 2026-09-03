package retention

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

// SqliteStore implements Store on the Grove ORM with SQLite.
type SqliteStore struct {
	db  *grove.DB
	sdb *sqlitedriver.SqliteDB
}

// NewSqliteStore builds a SQLite-backed store.
func NewSqliteStore(db *grove.DB) *SqliteStore {
	return &SqliteStore{db: db, sdb: sqlitedriver.Unwrap(db)}
}

var _ Store = (*SqliteStore)(nil)

// sqlErr maps a driver miss onto ErrNotFound so callers never see sql.ErrNoRows.
func sqlErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// isUniqueViolation reports whether err is a unique-constraint failure, which
// Enqueue treats as a duplicate rather than a fault.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")
}

// Enqueue inserts a pending job, treating a duplicate idempotency key as
// success. The unique index is what makes this safe under concurrency; a
// read-then-write check would race two hooks firing at once.
func (s *SqliteStore) Enqueue(ctx context.Context, j *Job) error {
	if j.State == "" {
		j.State = StatePending
	}
	if _, err := s.sdb.NewInsert(fromJob(j)).Exec(ctx); err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return sqlErr(err)
	}
	return nil
}

// ClaimDue claims due rows with a conditional update per row. The SELECT is
// only a candidate list: the UPDATE re-checks the due predicate and the row
// is kept only when it actually changed something, so two callers racing on
// the same candidate cannot both claim it.
//
// A plain unconditional "UPDATE ... WHERE id = ?" would let both win. SQLite
// being single-writer does not help, because nothing stops two readers
// seeing the same row before either writes. This is the same conditional
// update + RowsAffected idiom used in store/sqlite/refresh_replay.go.
func (s *SqliteStore) ClaimDue(ctx context.Context, limit int, lease time.Duration,
	now time.Time) ([]*Job, error) {
	var models []*jobModel
	q := s.sdb.NewSelect(&models).
		Where("(state = ? AND next_attempt_at <= ?) OR (state = ? AND in_flight_until < ?)",
			StatePending, now, StateInFlight, now).
		OrderExpr("next_attempt_at ASC, created_at ASC")
	// limit <= 0 means "no limit", matching the memory store; a LIMIT 0
	// clause would claim nothing.
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}

	until := now.Add(lease)
	out := make([]*Job, 0, len(models))
	for _, m := range models {
		res, err := s.sdb.NewUpdate((*jobModel)(nil)).
			Set("state = ?", StateInFlight).
			Set("in_flight_until = ?", until).
			Where("id = ?", m.ID).
			// Re-checks the same due predicate as the SELECT above, so a row
			// claimed by another caller between our select and our update
			// affects zero rows here instead of double-claiming it.
			Where("(state = ? AND next_attempt_at <= ?) OR (state = ? AND in_flight_until < ?)",
				StatePending, now, StateInFlight, now).
			Exec(ctx)
		if err != nil {
			return nil, sqlErr(err)
		}
		n, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
		if n == 0 {
			// Somebody else claimed it between our select and our update.
			continue
		}
		j, err := toJob(m)
		if err != nil {
			return nil, err
		}
		j.State = StateInFlight
		j.InFlightUntil = until
		out = append(out, j)
	}
	return out, nil
}

// MarkDone completes a job. now is intentionally unused: there is no
// completed_at column in the DDL.
func (s *SqliteStore) MarkDone(ctx context.Context, jobID id.RetentionJobID, _ time.Time) error {
	res, err := s.sdb.NewUpdate((*jobModel)(nil)).
		Set("state = ?", StateDone).
		Set("last_error = ?", "").
		Where("id = ?", jobID.String()).
		Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkRetry returns a job to pending with an incremented attempt count.
func (s *SqliteStore) MarkRetry(ctx context.Context, jobID id.RetentionJobID,
	nextAttemptAt time.Time, lastErr string) error {
	res, err := s.sdb.NewUpdate((*jobModel)(nil)).
		Set("state = ?", StatePending).
		Set("attempts = attempts + 1").
		Set("next_attempt_at = ?", nextAttemptAt).
		Set("in_flight_until = ?", nil).
		Set("last_error = ?", lastErr).
		Where("id = ?", jobID.String()).
		Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkDead parks a job permanently after too many attempts.
func (s *SqliteStore) MarkDead(ctx context.Context, jobID id.RetentionJobID, lastErr string) error {
	res, err := s.sdb.NewUpdate((*jobModel)(nil)).
		Set("state = ?", StateDead).
		Set("last_error = ?", lastErr).
		Where("id = ?", jobID.String()).
		Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkSuppressed records that the job was deliberately not delivered.
func (s *SqliteStore) MarkSuppressed(ctx context.Context, jobID id.RetentionJobID, reason string) error {
	res, err := s.sdb.NewUpdate((*jobModel)(nil)).
		Set("state = ?", StateSuppressed).
		Set("last_error = ?", reason).
		Where("id = ?", jobID.String()).
		Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetJob fetches one job. Returns ErrNotFound when absent.
func (s *SqliteStore) GetJob(ctx context.Context, jobID id.RetentionJobID) (*Job, error) {
	m := new(jobModel)
	if err := s.sdb.NewSelect(m).Where("id = ?", jobID.String()).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toJob(m)
}

// ListDead returns dead-lettered jobs for an app, newest first.
func (s *SqliteStore) ListDead(ctx context.Context, appID id.AppID, limit int) ([]*Job, error) {
	var models []*jobModel
	q := s.sdb.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Where("state = ?", StateDead).
		OrderExpr("created_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	out := make([]*Job, 0, len(models))
	for _, m := range models {
		j, err := toJob(m)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, nil
}

// GetRef returns the contact ref. Returns ErrNotFound when absent.
func (s *SqliteStore) GetRef(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) (*ContactRef, error) {
	m := new(contactRefModel)
	if err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("env_id = ?", envID.String()).
		Where("user_id = ?", userID.String()).
		Where("provider = ?", provider).
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toRef(m)
}

// PutRef inserts or updates the contact ref for the unique tuple. It selects
// by the unique (app_id, env_id, user_id, provider) tuple first: an insert
// when absent, otherwise an update of the fields that can change.
func (s *SqliteStore) PutRef(ctx context.Context, r *ContactRef) error {
	existing := new(contactRefModel)
	err := s.sdb.NewSelect(existing).
		Where("app_id = ?", r.AppID.String()).
		Where("env_id = ?", r.EnvID.String()).
		Where("user_id = ?", r.UserID.String()).
		Where("provider = ?", r.Provider).
		Scan(ctx)
	switch {
	case err == nil:
		_, uerr := s.sdb.NewUpdate((*contactRefModel)(nil)).
			Set("remote_object_type = ?", r.RemoteObjectType).
			Set("remote_id = ?", r.RemoteID).
			Set("synced_at = ?", r.SyncedAt).
			Where("id = ?", existing.ID).
			Exec(ctx)
		return sqlErr(uerr)
	case errors.Is(sqlErr(err), ErrNotFound):
		_, ierr := s.sdb.NewInsert(fromRef(r)).Exec(ctx)
		return sqlErr(ierr)
	default:
		return sqlErr(err)
	}
}

// DeleteRef removes the contact ref. Deleting an absent ref is not an error.
func (s *SqliteStore) DeleteRef(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) error {
	_, err := s.sdb.NewDelete((*contactRefModel)(nil)).
		Where("app_id = ?", appID.String()).
		Where("env_id = ?", envID.String()).
		Where("user_id = ?", userID.String()).
		Where("provider = ?", provider).
		Exec(ctx)
	return sqlErr(err)
}
