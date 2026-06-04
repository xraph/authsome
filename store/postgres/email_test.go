//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Register the pg migrate executor for s.Migrate in setupTestStore.
	_ "github.com/xraph/grove/drivers/pgdriver/pgmigrate"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	pgstore "github.com/xraph/authsome/store/postgres"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// seedAppAndEnv creates an app + default environment so the FK constraints on
// authsome_user_emails (app_id, env_id) are satisfied.
func seedAppAndEnv(t *testing.T, s *pgstore.Store) (id.AppID, id.EnvironmentID) {
	t.Helper()
	ctx := context.Background()
	a := &app.App{
		ID:        id.NewAppID(),
		Name:      "Email Test App",
		Slug:      "email-test-" + id.NewAppID().String()[len(id.NewAppID().String())-8:],
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateApp(ctx, a))
	e := &environment.Environment{
		ID:        id.NewEnvironmentID(),
		AppID:     a.ID,
		Name:      "Production",
		Slug:      "production",
		Type:      environment.TypeProduction,
		IsDefault: true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateEnvironment(ctx, e))
	return a.ID, e.ID
}

func seedUserWithEmail(t *testing.T, s *pgstore.Store, appID id.AppID, envID id.EnvironmentID, email string, verified bool) *user.User {
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
		ID:       id.NewUserEmailID(),
		UserID:   u.ID,
		AppID:    u.AppID,
		EnvID:    u.EnvID,
		Email:    email,
		Verified: verified,
		Source:   "test",
	}
}

// pgEmailStore builds a store with the app+env rows the FK constraints on
// authsome_user_emails require, returning the store plus a fresh app/env.
func pgEmailStore(t *testing.T) (*pgstore.Store, id.AppID, id.EnvironmentID) {
	t.Helper()
	s := setupTestStore(t)
	appID, envID := seedAppAndEnv(t, s)
	return s, appID, envID
}

func TestPg_AddUserEmail_FoundByAnyEmail(t *testing.T) {
	ctx := context.Background()
	s, appID, envID := pgEmailStore(t)
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", true)

	require.NoError(t, s.AddUserEmail(ctx, secondaryEmail(u, "second@x.com", true)))

	got, err := s.GetUserByAnyEmail(ctx, appID, envID, "second@x.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), got.ID.String())

	_, err = s.GetUserByAnyEmail(ctx, appID, envID, "nobody@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestPg_AddUserEmail_DuplicateReturnsEmailTaken(t *testing.T) {
	ctx := context.Background()
	s, appID, envID := pgEmailStore(t)
	seedUserWithEmail(t, s, appID, envID, "owner@x.com", true)
	other := seedUserWithEmail(t, s, appID, envID, "other@x.com", true)

	err := s.AddUserEmail(ctx, secondaryEmail(other, "owner@x.com", true))
	assert.ErrorIs(t, err, account.ErrEmailTaken)
}

func TestPg_SetPrimaryEmail_MovesPrimaryAndMirrorsUser(t *testing.T) {
	ctx := context.Background()
	s, appID, envID := pgEmailStore(t)
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
	assert.Equal(t, 1, primaries)

	gotUser, err := s.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "new@x.com", gotUser.Email)
}

func TestPg_SetPrimaryEmail_RejectsUnverified(t *testing.T) {
	ctx := context.Background()
	s, appID, envID := pgEmailStore(t)
	u := seedUserWithEmail(t, s, appID, envID, "old@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, secondaryEmail(u, "unverified@x.com", false)))

	err := s.SetPrimaryEmail(ctx, u.ID, "unverified@x.com")
	assert.ErrorIs(t, err, account.ErrEmailNotVerified)
}

func TestPg_DeleteUserEmail_SoftDeletesAndFrees(t *testing.T) {
	ctx := context.Background()
	s, appID, envID := pgEmailStore(t)
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, secondaryEmail(u, "second@x.com", true)))

	require.NoError(t, s.DeleteUserEmail(ctx, u.ID, "second@x.com"))

	_, err := s.GetUserByAnyEmail(ctx, appID, envID, "second@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)

	other := seedUserWithEmail(t, s, appID, envID, "other@x.com", true)
	assert.NoError(t, s.AddUserEmail(ctx, secondaryEmail(other, "second@x.com", true)))
}

func TestPg_DeleteUserEmail_RefusesPrimary(t *testing.T) {
	ctx := context.Background()
	s, appID, envID := pgEmailStore(t)
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", true)

	err := s.DeleteUserEmail(ctx, u.ID, "primary@x.com")
	assert.ErrorIs(t, err, store.ErrConflict)
}

func TestPg_MarkUserEmailVerified_MirrorsPrimary(t *testing.T) {
	ctx := context.Background()
	s, appID, envID := pgEmailStore(t)
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", false)

	require.NoError(t, s.MarkUserEmailVerified(ctx, u.ID, "primary@x.com"))

	gotUser, err := s.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, gotUser.EmailVerified)
}
