package retention

import (
	"context"
	"errors"
	"sort"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/pgdriver"

	"github.com/xraph/authsome/id"
)

// PostgresStore implements Store on the Grove ORM with PostgreSQL.
//
// It shares jobModel, contactRefModel and every mapper with SqliteStore;
// only ClaimDue diverges, because it needs a single atomic statement that
// SQLite's conditional-update-then-check idiom cannot express.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB

	// logger reports rows the mappers cannot read. Never nil: the
	// constructor installs a no-op one, and SetLogger replaces it.
	logger log.Logger
}

// NewPostgresStore builds a PostgreSQL-backed store.
func NewPostgresStore(db *grove.DB) *PostgresStore {
	return &PostgresStore{db: db, pg: pgdriver.Unwrap(db), logger: log.NewNoopLogger()}
}

// SetLogger installs the plugin's logger. Optional: the store works
// without it, it just cannot tell anyone about a row it had to skip.
func (s *PostgresStore) SetLogger(l log.Logger) {
	if l != nil {
		s.logger = l
	}
}

var _ Store = (*PostgresStore)(nil)

// Enqueue inserts a pending job, treating a duplicate idempotency key as
// success. The unique index on idempotency_key is what makes this safe under
// concurrency; a read-then-write check would race two hooks firing at once.
func (s *PostgresStore) Enqueue(ctx context.Context, j *Job) error {
	if j.State == "" {
		j.State = StatePending
	}
	if _, err := s.pg.NewInsert(fromJob(j)).Exec(ctx); err != nil {
		if isUniqueViolation(err) {
			return nil
		}
		return sqlErr(err)
	}
	return nil
}

// claimSQL claims a batch in one statement. FOR UPDATE SKIP LOCKED is doing
// real work here: without it two instances claiming at once block each other
// and take turns where they should be taking disjoint batches, so throughput
// collapses back to single-instance under exactly the load that made you run
// two.
//
// The second OR clause reclaims rows whose lease expired, which is how work
// from a process that died mid-delivery comes back at all: without it those
// rows are invisible to every later claim and the user behind them silently
// stops syncing.
const claimSQL = `
UPDATE authsome_retention_outbox
   SET state = 'in_flight', in_flight_until = $1
 WHERE id IN (
       SELECT id FROM authsome_retention_outbox
        WHERE (state = 'pending'   AND next_attempt_at <= $2)
           OR (state = 'in_flight' AND in_flight_until < $2)
        ORDER BY next_attempt_at ASC, created_at ASC
        LIMIT $3
        FOR UPDATE SKIP LOCKED
       )
RETURNING id, app_id, env_id, user_id, provider, kind, payload,
          idempotency_key, state, attempts, next_attempt_at,
          in_flight_until, last_error, created_at`

// claimSQLNoLimit is the same statement with the LIMIT clause dropped. limit
// <= 0 means "no limit" per the Store contract, and a literal "LIMIT $3" with
// a zero or negative bind would claim nothing instead, so the clause is
// omitted entirely rather than bound to a sentinel value.
const claimSQLNoLimit = `
UPDATE authsome_retention_outbox
   SET state = 'in_flight', in_flight_until = $1
 WHERE id IN (
       SELECT id FROM authsome_retention_outbox
        WHERE (state = 'pending'   AND next_attempt_at <= $2)
           OR (state = 'in_flight' AND in_flight_until < $2)
        ORDER BY next_attempt_at ASC, created_at ASC
        FOR UPDATE SKIP LOCKED
       )
RETURNING id, app_id, env_id, user_id, provider, kind, payload,
          idempotency_key, state, attempts, next_attempt_at,
          in_flight_until, last_error, created_at`

// ClaimDue atomically moves up to limit due jobs to in_flight in one
// statement and returns them. See claimSQL for why the statement is shaped
// the way it is.
func (s *PostgresStore) ClaimDue(ctx context.Context, limit int, lease time.Duration,
	now time.Time) ([]*Job, error) {
	until := now.Add(lease)

	query, args := claimSQL, []any{until, now, limit}
	if limit <= 0 {
		query, args = claimSQLNoLimit, []any{until, now}
	}

	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil {
		return nil, sqlErr(err)
	}
	defer func() { _ = rows.Close() }()

	var out []*Job
	for rows.Next() {
		m := new(jobModel)
		if serr := rows.Scan(&m.ID, &m.AppID, &m.EnvID, &m.UserID, &m.Provider, &m.Kind,
			&m.Payload, &m.IdempotencyKey, &m.State, &m.Attempts, &m.NextAttemptAt,
			&m.InFlightUntil, &m.LastError, &m.CreatedAt); serr != nil {
			return nil, serr
		}
		j, jerr := toJob(m)
		if jerr != nil {
			// One unreadable row must not discard the batch. Every row in
			// it is already in_flight, so returning here strands all of
			// them until their leases expire, and the same poison row is
			// then re-claimed and stalls the queue again, forever. Skipping
			// it means it comes back on every claim until somebody fixes or
			// removes the row: noisy, but it blocks nothing behind it.
			s.logger.Warn("retention: skipping unreadable outbox row",
				log.String("job_id", m.ID),
				log.String("error", jerr.Error()))
			continue
		}
		out = append(out, j)
	}
	if rerr := rows.Err(); rerr != nil {
		return nil, sqlErr(rerr)
	}
	// UPDATE ... RETURNING does not promise to preserve the subquery's
	// ORDER BY: Postgres is free to return the updated rows in whatever
	// order it happened to process them. The claim contract (oldest due
	// first) is about which rows get claimed, not just their order in the
	// result, so sort explicitly rather than relying on statement order.
	sort.Slice(out, func(a, b int) bool {
		if out[a].NextAttemptAt.Equal(out[b].NextAttemptAt) {
			return out[a].CreatedAt.Before(out[b].CreatedAt)
		}
		return out[a].NextAttemptAt.Before(out[b].NextAttemptAt)
	})
	return out, nil
}

// MarkDone completes a job. now is intentionally unused: there is no
// completed_at column in the DDL.
func (s *PostgresStore) MarkDone(ctx context.Context, jobID id.RetentionJobID, _ time.Time) error {
	res, err := s.pg.NewUpdate((*jobModel)(nil)).
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
func (s *PostgresStore) MarkRetry(ctx context.Context, jobID id.RetentionJobID,
	nextAttemptAt time.Time, lastErr string) error {
	res, err := s.pg.NewUpdate((*jobModel)(nil)).
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

// MarkDeferred returns a job to pending at nextAttemptAt without touching
// the attempt count. See the Store interface for why that is a separate
// method rather than a flag on MarkRetry.
func (s *PostgresStore) MarkDeferred(ctx context.Context, jobID id.RetentionJobID,
	nextAttemptAt time.Time, reason string) error {
	res, err := s.pg.NewUpdate((*jobModel)(nil)).
		Set("state = ?", StatePending).
		Set("next_attempt_at = ?", nextAttemptAt).
		Set("in_flight_until = ?", nil).
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

// MarkDead parks a job permanently after too many attempts.
func (s *PostgresStore) MarkDead(ctx context.Context, jobID id.RetentionJobID, lastErr string) error {
	res, err := s.pg.NewUpdate((*jobModel)(nil)).
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
func (s *PostgresStore) MarkSuppressed(ctx context.Context, jobID id.RetentionJobID, reason string) error {
	res, err := s.pg.NewUpdate((*jobModel)(nil)).
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

// PurgeTerminal deletes terminal rows older than their class cutoff, one
// statement per class. Same shape as SqliteStore.PurgeTerminal: three cheap
// statements running hourly, rather than one predicate whose AND/OR
// precedence a reader has to check before believing that a pending row is
// out of reach.
//
// Deleting the row releases its idempotency key, because the unique index
// is on the table and not on a side ledger. That is deliberate and it is
// the property the retention window is chosen to make safe.
func (s *PostgresStore) PurgeTerminal(ctx context.Context, doneBefore, auditBefore time.Time) (int, error) {
	total := 0
	for _, c := range purgeClasses(doneBefore, auditBefore) {
		res, err := s.pg.NewDelete((*jobModel)(nil)).
			Where("created_at < ?", c.Before.UTC()).
			Where("state = ?", c.State).
			Exec(ctx)
		if err != nil {
			return total, sqlErr(err)
		}
		n, _ := res.RowsAffected() //nolint:errcheck // driver always supports RowsAffected
		total += int(n)
	}
	return total, nil
}

// GetJob fetches one job. Returns ErrNotFound when absent.
func (s *PostgresStore) GetJob(ctx context.Context, jobID id.RetentionJobID) (*Job, error) {
	m := new(jobModel)
	if err := s.pg.NewSelect(m).Where("id = ?", jobID.String()).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toJob(m)
}

// ListDead returns dead-lettered jobs for an app, newest first.
func (s *PostgresStore) ListDead(ctx context.Context, appID id.AppID, limit int) ([]*Job, error) {
	var models []*jobModel
	q := s.pg.NewSelect(&models).
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
func (s *PostgresStore) GetRef(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) (*ContactRef, error) {
	m := new(contactRefModel)
	if err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("env_id = ?", envID.String()).
		Where("user_id = ?", userID.String()).
		Where("provider = ?", provider).
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toRef(m)
}

// updateRefByTuple updates the mutable columns of the ref addressed by the
// unique (app_id, env_id, user_id, provider) tuple. Addressing it by the
// tuple rather than by a primary key read a moment ago is what lets the
// insert path fall back onto it after losing a race.
func (s *PostgresStore) updateRefByTuple(ctx context.Context, r *ContactRef) error {
	_, err := s.pg.NewUpdate((*contactRefModel)(nil)).
		Set("remote_object_type = ?", r.RemoteObjectType).
		Set("remote_id = ?", r.RemoteID).
		Set("synced_at = ?", r.SyncedAt).
		Where("app_id = ?", r.AppID.String()).
		Where("env_id = ?", r.EnvID.String()).
		Where("user_id = ?", r.UserID.String()).
		Where("provider = ?", r.Provider).
		Exec(ctx)
	return sqlErr(err)
}

// PutRef inserts or updates the contact ref for the unique tuple. It selects
// by the unique (app_id, env_id, user_id, provider) tuple first: an insert
// when absent, otherwise an update of the fields that can change.
//
// Select-then-insert is not atomic, so two workers can both miss and both
// insert. That is likely rather than theoretical: a signup enqueues a
// contact_upsert and an activity_log for the same user in the same breath.
// The loser gets a unique violation on ux_retention_ref, so it falls back
// to the update instead of returning an error that would cost the job its
// retry budget for a row that is already exactly where it should be.
func (s *PostgresStore) PutRef(ctx context.Context, r *ContactRef) error {
	existing := new(contactRefModel)
	err := s.pg.NewSelect(existing).
		Where("app_id = ?", r.AppID.String()).
		Where("env_id = ?", r.EnvID.String()).
		Where("user_id = ?", r.UserID.String()).
		Where("provider = ?", r.Provider).
		Scan(ctx)
	switch {
	case err == nil:
		return s.updateRefByTuple(ctx, r)
	case errors.Is(sqlErr(err), ErrNotFound):
		_, ierr := s.pg.NewInsert(fromRef(r)).Exec(ctx)
		if ierr != nil && isUniqueViolation(ierr) {
			return s.updateRefByTuple(ctx, r)
		}
		return sqlErr(ierr)
	default:
		return sqlErr(err)
	}
}

// DeleteRef removes the contact ref. Deleting an absent ref is not an error.
func (s *PostgresStore) DeleteRef(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) error {
	_, err := s.pg.NewDelete((*contactRefModel)(nil)).
		Where("app_id = ?", appID.String()).
		Where("env_id = ?", envID.String()).
		Where("user_id = ?", userID.String()).
		Where("provider = ?", provider).
		Exec(ctx)
	return sqlErr(err)
}

// ListRefsForUser returns every ref held for the user across all apps and
// providers. Deliberately unscoped by app: a data-subject export covers the
// person, not one app's view of them.
func (s *PostgresStore) ListRefsForUser(ctx context.Context, userID id.UserID) ([]*ContactRef, error) {
	var models []*contactRefModel
	if err := s.pg.NewSelect(&models).
		Where("user_id = ?", userID.String()).
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	out := make([]*ContactRef, 0, len(models))
	for _, m := range models {
		r, err := toRef(m)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}
