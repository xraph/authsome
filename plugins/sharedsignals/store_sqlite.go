package sharedsignals

import (
	"context"
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

func (s *SqliteStore) CreateInboundStream(ctx context.Context, in *InboundStream) error {
	now := time.Now()
	if in.CreatedAt.IsZero() {
		in.CreatedAt = now
	}
	in.UpdatedAt = now
	_, err := s.sdb.NewInsert(fromInboundStream(in)).Exec(ctx)
	return sqlErr(err)
}

func (s *SqliteStore) GetInboundStream(ctx context.Context, streamID id.SSFStreamID) (*InboundStream, error) {
	m := new(inboundStreamModel)
	if err := s.sdb.NewSelect(m).Where("id = ?", streamID.String()).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toInboundStream(m)
}

func (s *SqliteStore) GetInboundStreamByPushPathHash(ctx context.Context, hash string) (*InboundStream, error) {
	m := new(inboundStreamModel)
	if err := s.sdb.NewSelect(m).Where("push_path_hash = ?", hash).Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toInboundStream(m)
}

func (s *SqliteStore) ListInboundStreams(ctx context.Context, appID id.AppID) ([]*InboundStream, error) {
	var models []*inboundStreamModel
	if err := s.sdb.NewSelect(&models).
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

func (s *SqliteStore) UpdateInboundStream(ctx context.Context, in *InboundStream) error {
	in.UpdatedAt = time.Now()
	res, err := s.sdb.NewUpdate(fromInboundStream(in)).
		Where("id = ?", in.ID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) DeleteInboundStream(ctx context.Context, streamID id.SSFStreamID) error {
	res, err := s.sdb.NewDelete((*inboundStreamModel)(nil)).
		Where("id = ?", streamID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) UpsertSubjectLink(ctx context.Context, l *SubjectLink) error {
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	l.LastSeenAt = now

	// See PostgresStore.UpsertSubjectLink: a single INSERT ... ON CONFLICT
	// DO UPDATE makes this atomic, so it stays idempotent under concurrent
	// upserts of the same tuple instead of racing a read-then-write.
	_, err := s.sdb.NewInsert(fromSubjectLink(l)).
		OnConflict(subjectLinkOnConflict).
		Exec(ctx)
	return sqlErr(err)
}

func (s *SqliteStore) GetSubjectLink(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, issuer, subject string) (*SubjectLink, error) {
	m := new(subjectLinkModel)
	if err := s.sdb.NewSelect(m).
		Where("app_id = ?", appID.String()).
		Where("env_id = ?", envID.String()).
		Where("issuer = ?", issuer).
		Where("subject = ?", subject).
		Scan(ctx); err != nil {
		return nil, sqlErr(err)
	}
	return toSubjectLink(m)
}

func (s *SqliteStore) InsertReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now()
	}
	_, err := s.sdb.NewInsert(fromReceivedEvent(e)).Exec(ctx)
	if isDuplicateKey(err) {
		return ErrDuplicateJTI
	}
	return sqlErr(err)
}

func (s *SqliteStore) UpdateReceivedEvent(ctx context.Context, e *ReceivedEvent) error {
	res, err := s.sdb.NewUpdate(fromReceivedEvent(e)).
		Where("id = ?", e.ID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) DeleteReceivedEvent(ctx context.Context, eventID id.SSFEventID) error {
	res, err := s.sdb.NewDelete((*receivedEventModel)(nil)).
		Where("id = ?", eventID.String()).Exec(ctx)
	if err != nil {
		return sqlErr(err)
	}
	if n, rerr := res.RowsAffected(); rerr == nil && n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SqliteStore) CountActionsSince(ctx context.Context,
	streamID id.SSFStreamID, since time.Time) (int, error) {
	count, err := s.sdb.NewSelect((*receivedEventModel)(nil)).
		Where("stream_id = ?", streamID.String()).
		Where("action_taken <> ?", "").
		Where("received_at > ?", since).
		Count(ctx)
	if err != nil {
		return 0, sqlErr(err)
	}
	return int(count), nil
}

func (s *SqliteStore) CreateSignal(ctx context.Context, sig *Signal) error {
	if sig.CreatedAt.IsZero() {
		sig.CreatedAt = time.Now()
	}
	_, err := s.sdb.NewInsert(fromSignal(sig)).Exec(ctx)
	return sqlErr(err)
}

func (s *SqliteStore) ListActiveSignals(ctx context.Context, appID id.AppID,
	envID id.EnvironmentID, userID id.UserID, now time.Time) ([]*Signal, error) {
	var models []*signalModel
	if err := s.sdb.NewSelect(&models).
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
