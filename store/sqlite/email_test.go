//go:build integration

// These tests verify the SQL/migration layer of the new authsome_user_emails
// table (uniqueness indexes, env scoping, soft-delete, the single-primary
// partial index). They deliberately assert through the email-table accessors
// (GetUserEmailRecord/GetUserEmails) rather than GetUserByAnyEmail/GetUser,
// because reading a *user.User back on sqlite hits a pre-existing,
// backend-wide bug: authsome_users declares its timestamp columns as TEXT,
// which grove+modernc never convert back to time.Time (tracked separately).
// The user-mirror read behavior (User.Email/EmailVerified) is proven in the
// memory-store tests, which have no such limitation.
package sqlite_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
	"github.com/xraph/authsome/user"
)

func seedUserWithEmail(t *testing.T, s *sqlitestore.Store, appID id.AppID, envID id.EnvironmentID, email string, verified bool) *user.User {
	t.Helper()
	u := &user.User{
		ID:            id.NewUserID(),
		AppID:         appID,
		EnvID:         envID,
		Email:         email,
		EmailVerified: verified,
	}
	primary := &user.UserEmail{
		ID:        id.NewUserEmailID(),
		UserID:    u.ID,
		AppID:     appID,
		EnvID:     envID,
		Email:     email,
		Verified:  verified,
		IsPrimary: true,
		Source:    "test",
	}
	require.NoError(t, s.CreateUserWithPrimaryEmail(context.Background(), u, primary))
	return u
}

func secondaryEmail(u *user.User, email string, verified bool) *user.UserEmail {
	return &user.UserEmail{
		ID:        id.NewUserEmailID(),
		UserID:    u.ID,
		AppID:     u.AppID,
		EnvID:     u.EnvID,
		Email:     email,
		Verified:  verified,
		IsPrimary: false,
		Source:    "test",
	}
}

func TestSqlite_AddUserEmail_ResolvesToOwner(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", true)

	require.NoError(t, s.AddUserEmail(ctx, secondaryEmail(u, "second@x.com", true)))

	// The secondary address resolves to the owning user (lookup layer of
	// GetUserByAnyEmail, asserted via the email row to avoid the pre-existing
	// GetUser sqlite limitation — see file header).
	rec, err := s.GetUserEmailRecord(ctx, appID, envID, "second@x.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), rec.UserID.String())

	// An unknown address resolves to nothing (this path returns before GetUser).
	_, err = s.GetUserByAnyEmail(ctx, appID, envID, "nobody@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestSqlite_AddUserEmail_DuplicateReturnsEmailTaken(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	seedUserWithEmail(t, s, appID, envID, "owner@x.com", true)
	other := seedUserWithEmail(t, s, appID, envID, "other@x.com", true)

	err := s.AddUserEmail(ctx, secondaryEmail(other, "owner@x.com", true))
	assert.ErrorIs(t, err, account.ErrEmailTaken)
}

func TestSqlite_GetUserByAnyEmail_NilEnvIsAppScoped(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "scoped@x.com", true)

	// nil env matches within the app (exercises the SQL conditional env filter).
	rec, err := s.GetUserEmailRecord(ctx, appID, id.Nil, "scoped@x.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), rec.UserID.String())

	_, err = s.GetUserEmailRecord(ctx, id.NewAppID(), id.Nil, "scoped@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestSqlite_GetUserByAnyEmail_EnvScoped(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID := id.NewAppID()
	envA, envB := id.NewEnvironmentID(), id.NewEnvironmentID()
	seedUserWithEmail(t, s, appID, envA, "shared@x.com", true)

	_, err := s.GetUserByAnyEmail(ctx, appID, envB, "shared@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestSqlite_SetPrimaryEmail_MovesPrimaryAndMirrors(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "old@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, secondaryEmail(u, "new@x.com", true)))

	require.NoError(t, s.SetPrimaryEmail(ctx, u.ID, "new@x.com"))

	emails, err := s.GetUserEmails(ctx, u.ID)
	require.NoError(t, err)
	primaries := 0
	for _, e := range emails {
		if e.IsPrimary {
			primaries++
			assert.Equal(t, "new@x.com", e.Email)
		}
	}
	assert.Equal(t, 1, primaries, "exactly one primary email (partial unique index must hold)")
}

func TestSqlite_SetPrimaryEmail_RejectsUnverified(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "old@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, secondaryEmail(u, "unverified@x.com", false)))

	err := s.SetPrimaryEmail(ctx, u.ID, "unverified@x.com")
	assert.ErrorIs(t, err, account.ErrEmailNotVerified)
}

func TestSqlite_DeleteUserEmail_SoftDeletesAndFrees(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, secondaryEmail(u, "second@x.com", true)))

	require.NoError(t, s.DeleteUserEmail(ctx, u.ID, "second@x.com"))

	_, err := s.GetUserByAnyEmail(ctx, appID, envID, "second@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)

	other := seedUserWithEmail(t, s, appID, envID, "other@x.com", true)
	assert.NoError(t, s.AddUserEmail(ctx, secondaryEmail(other, "second@x.com", true)))
}

func TestSqlite_DeleteUserEmail_RefusesPrimary(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", true)

	err := s.DeleteUserEmail(ctx, u.ID, "primary@x.com")
	assert.ErrorIs(t, err, store.ErrConflict)
}

func TestSqlite_MarkUserEmailVerified_UpdatesRecord(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", false)

	require.NoError(t, s.MarkUserEmailVerified(ctx, u.ID, "primary@x.com"))

	rec, err := s.GetUserEmailRecord(ctx, appID, envID, "primary@x.com")
	require.NoError(t, err)
	assert.True(t, rec.Verified)
}
