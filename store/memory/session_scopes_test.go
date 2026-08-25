package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

func TestSessionScopesRoundTrip(t *testing.T) {
	st := memory.New()
	ctx := context.Background()

	require.NoError(t, st.CreateSession(ctx, &session.Session{
		ID:        id.NewSessionID(),
		AppID:     id.NewAppID(),
		UserID:    id.NewUserID(),
		Token:     "tok-scopes",
		Scopes:    []string{"invoices:read", "invoices:write"},
		ExpiresAt: time.Now().Add(time.Hour),
	}))

	got, err := st.GetSessionByToken(ctx, "tok-scopes")
	require.NoError(t, err)
	assert.Equal(t, []string{"invoices:read", "invoices:write"}, got.Scopes)
}
