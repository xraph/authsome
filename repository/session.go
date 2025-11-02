package repository

import (
    "context"

    "github.com/rs/xid"
    "github.com/uptrace/bun"

    core "github.com/xraph/authsome/core/session"
    "github.com/xraph/authsome/schema"
)

// SessionRepository is a Bun-backed implementation of core session repository
type SessionRepository struct {
	db *bun.DB
}

func NewSessionRepository(db *bun.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) toSchema(s *core.Session) *schema.Session {
    return &schema.Session{
        ID:        s.ID,
        Token:     s.Token,
        UserID:    s.UserID,
        ExpiresAt: s.ExpiresAt,
        IPAddress: s.IPAddress,
        UserAgent: s.UserAgent,
        AuditableModel: schema.AuditableModel{
            CreatedAt: s.CreatedAt,
            UpdatedAt: bun.NullTime{Time: s.UpdatedAt},
            CreatedBy: s.UserID,
            UpdatedBy: s.UserID,
        },
    }
}

func (r *SessionRepository) fromSchema(ss *schema.Session) *core.Session {
    if ss == nil {
        return nil
    }
    return &core.Session{
        ID:        ss.ID,
        Token:     ss.Token,
        UserID:    ss.UserID,
        ExpiresAt: ss.ExpiresAt,
        IPAddress: ss.IPAddress,
        UserAgent: ss.UserAgent,
        CreatedAt: ss.CreatedAt,
        UpdatedAt: ss.UpdatedAt.Time,
    }
}

// Create inserts a new session
func (r *SessionRepository) Create(ctx context.Context, s *core.Session) error {
	ss := r.toSchema(s)
	_, err := r.db.NewInsert().Model(ss).Exec(ctx)
	return err
}

// FindByToken retrieves a session by token
func (r *SessionRepository) FindByToken(ctx context.Context, token string) (*core.Session, error) {
	ss := new(schema.Session)
	err := r.db.NewSelect().Model(ss).Where("token = ?", token).Scan(ctx)
	if err != nil {
		return nil, err
	}
	return r.fromSchema(ss), nil
}

// Revoke deletes a session by token
func (r *SessionRepository) Revoke(ctx context.Context, token string) error {
    _, err := r.db.NewDelete().Model((*schema.Session)(nil)).Where("token = ?", token).Exec(ctx)
    return err
}

// FindByID retrieves a session by id
func (r *SessionRepository) FindByID(ctx context.Context, id xid.ID) (*core.Session, error) {
    ss := new(schema.Session)
    err := r.db.NewSelect().Model(ss).Where("id = ?", id).Scan(ctx)
    if err != nil {
        return nil, err
    }
    return r.fromSchema(ss), nil
}

// ListByUser lists sessions for a user, ordered by recency
func (r *SessionRepository) ListByUser(ctx context.Context, userID xid.ID, limit, offset int) ([]*core.Session, error) {
    var srows []schema.Session
    q := r.db.NewSelect().Model(&srows).Where("user_id = ?", userID).OrderExpr("created_at DESC")
    if limit > 0 { q = q.Limit(limit) }
    if offset > 0 { q = q.Offset(offset) }
    if err := q.Scan(ctx); err != nil {
        return nil, err
    }
    res := make([]*core.Session, 0, len(srows))
    for i := range srows {
        res = append(res, r.fromSchema(&srows[i]))
    }
    return res, nil
}

// RevokeByID deletes a session by id
func (r *SessionRepository) RevokeByID(ctx context.Context, id xid.ID) error {
    _, err := r.db.NewDelete().Model((*schema.Session)(nil)).Where("id = ?", id).Exec(ctx)
    return err
}

// ListAll lists all sessions in the system, ordered by recency
func (r *SessionRepository) ListAll(ctx context.Context, limit, offset int) ([]*core.Session, error) {
    var srows []schema.Session
    q := r.db.NewSelect().Model(&srows).OrderExpr("created_at DESC")
    if limit > 0 { q = q.Limit(limit) }
    if offset > 0 { q = q.Offset(offset) }
    if err := q.Scan(ctx); err != nil {
        return nil, err
    }
    res := make([]*core.Session, 0, len(srows))
    for i := range srows {
        res = append(res, r.fromSchema(&srows[i]))
    }
    return res, nil
}
