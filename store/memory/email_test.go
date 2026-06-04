package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	memory "github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

func emailRow(u *user.User, email string, verified, primary bool) *user.UserEmail {
	return &user.UserEmail{
		ID:        id.NewUserEmailID(),
		UserID:    u.ID,
		AppID:     u.AppID,
		EnvID:     u.EnvID,
		Email:     email,
		Verified:  verified,
		IsPrimary: primary,
		Source:    "test",
	}
}

func seedUser(t *testing.T, s *memory.Store, appID id.AppID, envID id.EnvironmentID, email string, verified bool) *user.User {
	t.Helper()
	u := &user.User{
		ID:            id.NewUserID(),
		AppID:         appID,
		EnvID:         envID,
		Email:         email,
		EmailVerified: verified,
	}
	require.NoError(t, s.CreateUserWithPrimaryEmail(context.Background(), u, emailRow(u, email, verified, true)))
	return u
}

func TestAddUserEmail_FoundByAnyEmail(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "primary@x.com", true)

	require.NoError(t, s.AddUserEmail(ctx, emailRow(u, "secondary@x.com", true, false)))

	got, err := s.GetUserByAnyEmail(ctx, appID, envID, "secondary@x.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), got.ID.String())

	_, err = s.GetUserByAnyEmail(ctx, appID, envID, "nobody@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestGetUserByAnyEmail_IsCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "user@x.com", true)

	got, err := s.GetUserByAnyEmail(ctx, appID, envID, "USER@X.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), got.ID.String())
}

func TestAddUserEmail_DuplicateReturnsEmailTaken(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	seedUser(t, s, appID, envID, "owner@x.com", true)
	other := seedUser(t, s, appID, envID, "other@x.com", true)

	err := s.AddUserEmail(ctx, emailRow(other, "owner@x.com", true, false))
	assert.ErrorIs(t, err, account.ErrEmailTaken)
}

func TestGetUserEmailRecord_ReturnsVerifiedFlag(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "p@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, emailRow(u, "unverified@x.com", false, false)))

	rec, err := s.GetUserEmailRecord(ctx, appID, envID, "unverified@x.com")
	require.NoError(t, err)
	assert.False(t, rec.Verified)
	assert.Equal(t, u.ID.String(), rec.UserID.String())
}

func TestGetUserEmails_ReturnsAllPrimaryFirst(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "primary@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, emailRow(u, "second@x.com", true, false)))

	emails, err := s.GetUserEmails(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, emails, 2)
	assert.True(t, emails[0].IsPrimary, "primary email must be first")
	assert.Equal(t, "primary@x.com", emails[0].Email)
}

func TestMarkUserEmailVerified_UpdatesRecordAndPrimaryMirror(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "primary@x.com", false)

	require.NoError(t, s.MarkUserEmailVerified(ctx, u.ID, "primary@x.com"))

	rec, err := s.GetUserEmailRecord(ctx, appID, envID, "primary@x.com")
	require.NoError(t, err)
	assert.True(t, rec.Verified)

	// Verifying the primary email must mirror onto the user record.
	gotUser, err := s.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.True(t, gotUser.EmailVerified)
}

func TestSetPrimaryEmail_MovesPrimaryAndMirrorsUser(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "old@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, emailRow(u, "new@x.com", true, false)))

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
	assert.Equal(t, 1, primaries, "exactly one primary email")

	gotUser, err := s.GetUser(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "new@x.com", gotUser.Email)
}

func TestSetPrimaryEmail_RejectsUnverified(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "old@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, emailRow(u, "unverified@x.com", false, false)))

	err := s.SetPrimaryEmail(ctx, u.ID, "unverified@x.com")
	assert.ErrorIs(t, err, account.ErrEmailNotVerified)
}

func TestDeleteUserEmail_SoftDeletesAndFreesAddress(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "primary@x.com", true)
	require.NoError(t, s.AddUserEmail(ctx, emailRow(u, "second@x.com", true, false)))

	require.NoError(t, s.DeleteUserEmail(ctx, u.ID, "second@x.com"))

	_, err := s.GetUserByAnyEmail(ctx, appID, envID, "second@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)

	// The freed address can be claimed by another user.
	other := seedUser(t, s, appID, envID, "other@x.com", true)
	assert.NoError(t, s.AddUserEmail(ctx, emailRow(other, "second@x.com", true, false)))
}

func TestDeleteUserEmail_RefusesPrimary(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "primary@x.com", true)

	err := s.DeleteUserEmail(ctx, u.ID, "primary@x.com")
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrConflict), "deleting the primary email must be refused")
}

func TestGetUserByAnyEmail_IsEnvScoped(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID := id.NewAppID()
	envA, envB := id.NewEnvironmentID(), id.NewEnvironmentID()
	seedUser(t, s, appID, envA, "shared@x.com", true)

	// Same address in a different environment must not resolve.
	_, err := s.GetUserByAnyEmail(ctx, appID, envB, "shared@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestGetUserByAnyEmail_NilEnvIsAppScoped(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID := id.NewAppID()
	envA := id.NewEnvironmentID()
	u := seedUser(t, s, appID, envA, "scoped@x.com", true)

	// A nil env matches within the app regardless of the row's env (used by
	// lookup flows that have no environment in scope).
	got, err := s.GetUserByAnyEmail(ctx, appID, id.EnvironmentID{}, "scoped@x.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), got.ID.String())

	// A different app still does not match.
	_, err = s.GetUserByAnyEmail(ctx, id.NewAppID(), id.EnvironmentID{}, "scoped@x.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestGetUserEmailRecord_NilEnvIsAppScoped(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID := id.NewAppID()
	envA := id.NewEnvironmentID()
	u := seedUser(t, s, appID, envA, "rec@x.com", true)

	rec, err := s.GetUserEmailRecord(ctx, appID, id.EnvironmentID{}, "rec@x.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), rec.UserID.String())
}

func TestCreateUserWithPrimaryEmail_CreatesBoth(t *testing.T) {
	ctx := context.Background()
	s := memory.New()
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	u := seedUser(t, s, appID, envID, "fresh@x.com", true)

	got, err := s.GetUserByAnyEmail(ctx, appID, envID, "fresh@x.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), got.ID.String())

	emails, err := s.GetUserEmails(ctx, u.ID)
	require.NoError(t, err)
	require.Len(t, emails, 1)
	assert.True(t, emails[0].IsPrimary)
}
