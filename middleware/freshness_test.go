package middleware_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

func TestSessionIsFresh(t *testing.T) {
	now := time.Now()
	fresh := &session.Session{ID: id.NewSessionID(), CreatedAt: now.Add(-time.Minute)}
	stale := &session.Session{ID: id.NewSessionID(), CreatedAt: now.Add(-time.Hour)}

	assert.True(t, middleware.SessionIsFresh(middleware.WithSession(context.Background(), fresh), now, 5*time.Minute, nil))
	assert.False(t, middleware.SessionIsFresh(middleware.WithSession(context.Background(), stale), now, 5*time.Minute, nil))
	assert.False(t, middleware.SessionIsFresh(context.Background(), now, 5*time.Minute, nil), "no session is never fresh")

	lookup := func(_ context.Context, sid id.SessionID) (*session.Session, error) {
		if sid == fresh.ID {
			return fresh, nil
		}
		return nil, errors.New("not found")
	}
	byID := middleware.WithSessionID(context.Background(), fresh.ID)
	assert.True(t, middleware.SessionIsFresh(byID, now, 5*time.Minute, lookup), "a JWT caller is checked through the row")
	assert.False(t, middleware.SessionIsFresh(middleware.WithSessionID(context.Background(), id.NewSessionID()), now, 5*time.Minute, lookup))

	assert.True(t, middleware.SessionIsFresh(middleware.WithSession(context.Background(), fresh), now, 0, nil), "zero window means the default")
}
