package sharedsignals

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

// PostgresStore implements Store on the Grove ORM with PostgreSQL.
type PostgresStore struct {
	db *grove.DB
	pg *pgdriver.PgDB
}

// NewPostgresStore builds a PostgreSQL-backed store.
func NewPostgresStore(db *grove.DB) *PostgresStore {
	return &PostgresStore{db: db, pg: pgdriver.Unwrap(db)}
}

var _ Store = (*PostgresStore)(nil)

// isDuplicateKey reports whether a driver error is a unique-constraint
// violation. The dedupe path depends on telling that apart from a real
// failure, so both SQL stores route through this.
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique violation") ||
		strings.Contains(msg, "23505")
}

func sqlErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *PostgresStore) CreateInboundStream(ctx context.Context, in *InboundStream) error {
	now := time.Now()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = now
	}
	in.UpdatedAt = now
	_, err := s.pg.NewInsert(fromInboundStream(in)).Exec(ctx)
	return sqlErr(err)
}

func (s *PostgresStore) GetInboundStream(ctx context.Context, streamID id.SSFStreamID) (*InboundStream, error) {
	m := new(inboundStreamModel)
	if err := s.pg.NewSelect(m).Where("id = ?", streamID.String()).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toInboundStream(m)
}

func (s *PostgresStore) GetInboundStreamByPushPathHash(ctx context.Context, hash string) (*InboundStream, error) {
	m := new(inboundStreamModel)
	if err := s.pg.NewSelect(m).Where("push_path_hash = ?", hash).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toInboundStream(m)
}

func (s *PostgresStore) ListInboundStreams(ctx context.Context, appID id.AppID) ([]*InboundStream, error) {
	var models []*inboundStreamModel
	if err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		OrderExpr("created_at DESC").
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	out := make([]*InboundStream, 0, len(models))
	for _, m := range models {
		converted, err := toInboundStream(m)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}

func (s *PostgresStore) UpdateInboundStream(ctx context.Context, in *InboundStream) error {
	in.UpdatedAt = time.Now()
	res, err := s.pg.NewUpdate(fromInboundStream(in)).
		Where("id = ?", in.ID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteInboundStream(ctx context.Context, streamID id.SSFStreamID) error {
	res, err := s.pg.NewDelete((*inboundStreamModel)(nil)).
		Where("id = ?", streamID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

// subjectLinkOnConflict is the upsert clause shared by both SQL backends.
// It leaves id and created_at untouched on a conflicting row, so a losing
// concurrent writer never clobbers the winner's identity or creation time,
// only the fields that are meant to move: user_id, source, last_seen_at.
const subjectLinkOnConflict = "(app_id, env_id, issuer, subject) DO UPDATE SET " +
	"user_id = EXCLUDED.user_id, source = EXCLUDED.source, last_seen_at = EXCLUDED.last_seen_at"

func (s *PostgresStore) UpsertSubjectLink(ctx context.Context, l *SubjectLink) error {
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	l.LastSeenAt = now

	// A single INSERT ... ON CONFLICT DO UPDATE is atomic, so concurrent
	// upserts of the same (app_id, env_id, issuer, subject) tuple never
	// race a read-then-write: the database serializes them and the loser
	// updates instead of colliding on the unique index.
	_, err := s.pg.NewInsert(fromSubjectLink(l)).
		OnConflict(subjectLinkOnConflict).
		Exec(ctx)
	return sqlErr(err)
}

func (s *PostgresStore) GetSubjectLink(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, issuer, subject string) (*SubjectLink, error) {
	m := new(subjectLinkModel)
	if err := s.pg.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("env_id = ?", envID.String()).
		Where("issuer = ?", issuer).
		Where("subject = ?", subject).
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toSubjectLink(m)
}

func (s *PostgresStore) InsertReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now()
	}
	_, err := s.pg.NewInsert(fromReceivedEvent(e)).Exec(ctx)
	if isDuplicateKey(err) {
		return ErrDuplicateJTI
	}
	return sqlErr(err)
}

func (s *PostgresStore) UpdateReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	res, err := s.pg.NewUpdate(fromReceivedEvent(e)).
		Where("id = ?", e.ID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresStore) DeleteReceivedEvent(ctx context.Context, eventID id.SSFEventID) error {
	res, err := s.pg.NewDelete((*receivedEventModel)(nil)).
		Where("id = ?", eventID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetReceivedEvent loads one audit row by ID and confirms it belongs to
// appID. The row itself carries no app_id, so the scope comes from the
// stream it arrived on.
func (s *PostgresStore) GetReceivedEvent(ctx context.Context, appID id.AppID,
	eventID id.SSFEventID) (*ReceivedEvent, error) {
	m := new(receivedEventModel)
	if err := s.pg.NewSelect(m).Where("id = ?", eventID.String()).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	e, err := toReceivedEvent(m)
	if err != nil {
		return nil, err
	}
	if oerr := streamOwnedBy(ctx, s, appID, e.StreamID); oerr != nil {
		return nil, oerr
	}
	return e, nil
}

// ListReceivedEvents returns one stream's audit rows newest first. The
// (stream_id, received_at DESC) index from the create migration is exactly
// this query's access path.
func (s *PostgresStore) ListReceivedEvents(ctx context.Context, appID id.AppID,
	f ReceivedEventFilter) ([]*ReceivedEvent, error) {
	// Ownership before rows: a stream belonging to another tenant answers
	// ErrNotFound, so a probe cannot tell "not yours" from "yours but quiet".
	if err := streamOwnedBy(ctx, s, appID, f.StreamID); err != nil {
		return nil, err
	}
	f = f.normalized()

	var models []*receivedEventModel
	q := s.pg.NewSelect(&models).Where("stream_id = ?", f.StreamID.String())
	if !f.Since.IsZero() {
		q = q.Where("received_at >= ?", f.Since)
	}
	if !f.Until.IsZero() {
		q = q.Where("received_at < ?", f.Until)
	}
	// The id tie-break makes the page deterministic when several rows share
	// a received_at, which one multi-event SET produces by construction.
	if err := q.OrderExpr("received_at DESC").OrderExpr("id DESC").
		Limit(f.Limit).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}

	out := make([]*ReceivedEvent, 0, len(models))
	for _, m := range models {
		converted, cerr := toReceivedEvent(m)
		if cerr != nil {
			return nil, cerr
		}
		out = append(out, converted)
	}
	return out, nil
}

func (s *PostgresStore) CountEventsSince(ctx context.Context,
	streamID id.SSFStreamID, since time.Time) (int, error) {
	// Every recorded event counts, not just the ones that acted -- see
	// Store.CountEventsSince.
	count, err := s.pg.NewSelect((*receivedEventModel)(nil)).
		Where("stream_id = ?", streamID.String()).
		Where("received_at > ?", since).
		Count(ctx)
	if err != nil {
		return 0, sqlErr(err)
	}
	return int(count), nil
}

func (s *PostgresStore) CreateSignal(ctx context.Context, sig *Signal) error {
	if sig.CreatedAt.IsZero() {
		sig.CreatedAt = time.Now()
	}
	_, err := s.pg.NewInsert(fromSignal(sig)).Exec(ctx)
	return sqlErr(err)
}

func (s *PostgresStore) ListActiveSignals(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, userID id.UserID, now time.Time) ([]*Signal, error) {
	var models []*signalModel
	if err := s.pg.NewSelect(&models).
		Where("app_id = ?", appID.String()).
		Where("env_id = ?", envID.String()).
		Where("user_id = ?", userID.String()).
		Where("expires_at > ?", now).
		OrderExpr("severity DESC").
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	out := make([]*Signal, 0, len(models))
	for _, m := range models {
		converted, err := toSignal(m)
		if err != nil {
			return nil, err
		}
		out = append(out, converted)
	}
	return out, nil
}
