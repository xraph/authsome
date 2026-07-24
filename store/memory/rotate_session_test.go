package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	memory "github.com/xraph/authsome/store/memory"
)

// TestRotateSession_CompareAndSwap pins the atomic compare-and-swap that fixes
// the refresh-token rotation TOCTOU: an update is applied only when the stored
// access token still equals the expected (pre-rotation) value. A second,
// stale-based rotation must not overwrite the winner.
func TestRotateSession_CompareAndSwap(t *testing.T) {
	s := memory.New()
	ctx := context.Background()

	sess := &session.Session{
		ID:           id.NewSessionID(),
		AppID:        id.NewAppID(),
		UserID:       id.NewUserID(),
		Token:        "T0",
		RefreshToken: "R0",
	}
	require.NoError(t, s.CreateSession(ctx, sess))

	// Winner rotates from T0 -> T1.
	win := *sess
	win.Token = "T1"
	win.RefreshToken = "R1"
	ok, err := s.RotateSession(ctx, &win, "T0")
	require.NoError(t, err)
	require.True(t, ok, "CAS with the current token must succeed")

	// Loser presents the now-stale T0 and must lose.
	lose := *sess
	lose.Token = "T2"
	lose.RefreshToken = "R2"
	ok, err = s.RotateSession(ctx, &lose, "T0")
	require.NoError(t, err)
	require.False(t, ok, "CAS with a stale token must not overwrite the winner")

	// The stored session must reflect the winner's tokens.
	got, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	require.Equal(t, "T1", got.Token, "winner's access token must be persisted")
	require.Equal(t, "R1", got.RefreshToken, "winner's refresh token must be persisted")
}

// TestGetSessionByRefreshToken_ReturnsCopy pins that the memory store hands out
// a copy, not the live map pointer. The refresh flow mutates the returned
// session in place before persisting; if it shared the stored pointer, the CAS
// could never detect the pre-rotation value.
func TestGetSessionByRefreshToken_ReturnsCopy(t *testing.T) {
	s := memory.New()
	ctx := context.Background()

	sess := &session.Session{
		ID:           id.NewSessionID(),
		AppID:        id.NewAppID(),
		UserID:       id.NewUserID(),
		Token:        "T0",
		RefreshToken: "R0",
	}
	require.NoError(t, s.CreateSession(ctx, sess))

	fetched, err := s.GetSessionByRefreshToken(ctx, "R0")
	require.NoError(t, err)

	// Mutating the fetched copy must not affect stored state.
	fetched.Token = "MUTATED"
	fetched.RefreshToken = "MUTATED"

	stored, err := s.GetSession(ctx, sess.ID)
	require.NoError(t, err)
	require.Equal(t, "T0", stored.Token, "mutating a fetched session must not leak into the store")
}
