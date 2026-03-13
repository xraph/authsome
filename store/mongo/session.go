package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
)

// CreateSession persists a new session.
func (s *Store) CreateSession(ctx context.Context, sess *session.Session) error {
	m := toSessionModel(sess)

	_, err := s.mdb.NewInsert(m).Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: create session: %w", err)
	}

	return nil
}

// GetSession returns a session by ID.
func (s *Store) GetSession(ctx context.Context, sessionID id.SessionID) (*session.Session, error) {
	var m sessionModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"_id": sessionID.String()}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get session: %w", err)
	}

	return fromSessionModel(&m)
}

// GetSessionByToken returns a session by its access token.
func (s *Store) GetSessionByToken(ctx context.Context, token string) (*session.Session, error) {
	var m sessionModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"token": token}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get session by token: %w", err)
	}

	return fromSessionModel(&m)
}

// GetSessionByRefreshToken returns a session by its refresh token.
func (s *Store) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*session.Session, error) {
	var m sessionModel

	err := s.mdb.NewFind(&m).
		Filter(bson.M{"refresh_token": refreshToken}).
		Scan(ctx)
	if err != nil {
		if isNoDocuments(err) {
			return nil, store.ErrNotFound
		}

		return nil, fmt.Errorf("authsome/mongo: get session by refresh token: %w", err)
	}

	return fromSessionModel(&m)
}

// UpdateSession modifies an existing session.
func (s *Store) UpdateSession(ctx context.Context, sess *session.Session) error {
	m := toSessionModel(sess)
	m.UpdatedAt = now()

	res, err := s.mdb.NewUpdate(m).
		Filter(bson.M{"_id": m.ID}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: update session: %w", err)
	}

	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// TouchSession performs a lightweight update of last_activity_at, expires_at, and updated_at.
func (s *Store) TouchSession(ctx context.Context, sessionID id.SessionID, lastActivityAt, expiresAt time.Time) error {
	res, err := s.mdb.NewUpdate((*sessionModel)(nil)).
		Filter(bson.M{"_id": sessionID.String()}).
		Set("last_activity_at", lastActivityAt).
		Set("expires_at", expiresAt).
		Set("updated_at", now()).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: touch session: %w", err)
	}
	if res.MatchedCount() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// DeleteSession removes a session by ID.
func (s *Store) DeleteSession(ctx context.Context, sessionID id.SessionID) error {
	res, err := s.mdb.NewDelete((*sessionModel)(nil)).
		Filter(bson.M{"_id": sessionID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete session: %w", err)
	}

	if res.DeletedCount() == 0 {
		return store.ErrNotFound
	}

	return nil
}

// DeleteUserSessions removes all sessions for a user.
func (s *Store) DeleteUserSessions(ctx context.Context, userID id.UserID) error {
	_, err := s.mdb.NewDelete((*sessionModel)(nil)).
		Many().
		Filter(bson.M{"user_id": userID.String()}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("authsome/mongo: delete user sessions: %w", err)
	}

	return nil
}

// ListUserSessions returns all sessions for a user, ordered by creation date descending.
func (s *Store) ListUserSessions(ctx context.Context, userID id.UserID) ([]*session.Session, error) {
	var models []sessionModel

	err := s.mdb.NewFind(&models).
		Filter(bson.M{"user_id": userID.String()}).
		Sort(bson.D{{Key: "created_at", Value: -1}}).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("authsome/mongo: list user sessions: %w", err)
	}

	result := make([]*session.Session, 0, len(models))

	for i := range models {
		sess, err := fromSessionModel(&models[i])
		if err != nil {
			return nil, err
		}

		result = append(result, sess)
	}

	return result, nil
}

func (s *Store) ListSessions(ctx context.Context, limit int) ([]*session.Session, error) {
	var models []sessionModel

	q := s.mdb.NewFind(&models).
		Sort(bson.D{{Key: "created_at", Value: -1}})
	if limit > 0 {
		q = q.Limit(int64(limit))
	}
	if err := q.Scan(ctx); err != nil {
		return nil, fmt.Errorf("authsome/mongo: list sessions: %w", err)
	}

	result := make([]*session.Session, 0, len(models))
	for i := range models {
		sess, err := fromSessionModel(&models[i])
		if err != nil {
			return nil, err
		}
		result = append(result, sess)
	}
	return result, nil
}
