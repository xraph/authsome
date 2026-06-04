//go:build integration

package mongo_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store"
	mongostore "github.com/xraph/authsome/store/mongo"
	"github.com/xraph/authsome/user"
)

func seedUserWithEmail(t *testing.T, s *mongostore.Store, appID id.AppID, envID id.EnvironmentID, email string, verified bool) *user.User {
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

func TestMongo_AddUserEmail_FoundByAnyEmail(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", true)

	require.NoError(t, s.AddUserEmail(ctx, secondaryEmail(u, "second@x.com", true)))

	got, err := s.GetUserByAnyEmail(ctx, appID, envID, "second@x.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), got.ID.String())

	_, err = s.GetUserByAnyEmail(ctx, appID, envID, "nobody@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestMongo_AddUserEmail_DuplicateReturnsEmailTaken(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	seedUserWithEmail(t, s, appID, envID, "owner@x.com", true)
	other := seedUserWithEmail(t, s, appID, envID, "other@x.com", true)

	err := s.AddUserEmail(ctx, secondaryEmail(other, "owner@x.com", true))
	assert.ErrorIs(t, err, account.ErrEmailTaken)
}

func TestMongo_GetUserByAnyEmail_EnvScoped(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID := id.NewAppID()
	envA, envB := id.NewEnvironmentID(), id.NewEnvironmentID()
	seedUserWithEmail(t, s, appID, envA, "shared@x.com", true)

	_, err := s.GetUserByAnyEmail(ctx, appID, envB, "shared@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestMongo_SetPrimaryEmail_MovesPrimaryAndMirrors(t *testing.T) {
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
	assert.Equal(t, 1, primaries)

	gotUser, err := s.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "new@x.com", gotUser.Email)
}

func TestMongo_SetPrimaryEmail_RejectsUnverified(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "old@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, secondaryEmail(u, "unverified@x.com", false)))

	err := s.SetPrimaryEmail(ctx, u.ID, "unverified@x.com")
	assert.ErrorIs(t, err, account.ErrEmailNotVerified)
}

func TestMongo_DeleteUserEmail_SoftDeletesAndFrees(t *testing.T) {
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

func TestMongo_DeleteUserEmail_RefusesPrimary(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", true)

	err := s.DeleteUserEmail(ctx, u.ID, "primary@x.com")
	assert.ErrorIs(t, err, store.ErrConflict)
}

func TestMongo_MarkUserEmailVerified_MirrorsPrimary(t *testing.T) {
	ctx := context.Background()
	s := setupTestStore(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUserWithEmail(t, s, appID, envID, "primary@x.com", false)

	require.NoError(t, s.MarkUserEmailVerified(ctx, u.ID, "primary@x.com"))

	gotUser, err := s.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, gotUser.EmailVerified)
}
