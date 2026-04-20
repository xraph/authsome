package authsome_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/lockout"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
)

func testAppID(t *testing.T) id.AppID {
	t.Helper()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	return appID
}

func signUpTestUser(t *testing.T, eng *authsome.Engine, email, password string) { //nolint:unparam // test helper
	t.Helper()
	ctx := context.Background()
	appID := testAppID(t)

	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		Password:  password,
		FirstName: "Test User",
	})
	require.NoError(t, err)
}

// ──────────────────────────────────────────────────
// SignUp tests
// ──────────────────────────────────────────────────

func TestSignUp_Success(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "alice@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Alice",
		Username:  "alice",
	})

	require.NoError(t, err)
	assert.NotNil(t, u)
	assert.NotNil(t, sess)

	// User assertions
	assert.Equal(t, "alice@example.com", u.Email)
	assert.Equal(t, "Alice", u.FirstName)
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, appID, u.AppID)
	assert.NotEmpty(t, u.ID.String())
	assert.NotEmpty(t, u.PasswordHash)
	assert.False(t, u.Banned)

	// Session assertions
	assert.NotEmpty(t, sess.Token)
	assert.NotEmpty(t, sess.RefreshToken)
	assert.Equal(t, u.ID, sess.UserID)
	assert.Equal(t, appID, sess.AppID)
	assert.True(t, sess.ExpiresAt.After(time.Now()))
}

func TestSignUp_DuplicateEmail(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	// First signup succeeds
	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "dupe@example.com",
		Password:  "SecureP@ss1",
		FirstName: "First",
	})
	require.NoError(t, err)

	// Second signup with same email fails
	_, _, err = eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "dupe@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Second",
	})
	assert.ErrorIs(t, err, account.ErrEmailTaken)
}

func TestSignUp_DuplicateUsername(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "user1@example.com",
		Password:  "SecureP@ss1",
		FirstName: "User One",
		Username:  "samename",
	})
	require.NoError(t, err)

	_, _, err = eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "user2@example.com",
		Password:  "SecureP@ss1",
		FirstName: "User Two",
		Username:  "samename",
	})
	assert.ErrorIs(t, err, account.ErrUsernameTaken)
}

func TestSignUp_WeakPassword(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "weak@example.com",
		Password:  "short",
		FirstName: "Weak",
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, account.ErrWeakPassword)
}

func TestSignUp_EmailNormalization(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "  ALICE@EXAMPLE.COM  ",
		Password:  "SecureP@ss1",
		FirstName: "Alice",
	})
	require.NoError(t, err)
	assert.Equal(t, "alice@example.com", u.Email)
}

// ──────────────────────────────────────────────────
// SignIn tests
// ──────────────────────────────────────────────────

func TestSignIn_Success_ByEmail(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	signUpTestUser(t, eng, "signin@example.com", "SecureP@ss1")

	u, sess, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "signin@example.com",
		Password: "SecureP@ss1",
	})

	require.NoError(t, err)
	assert.NotNil(t, u)
	assert.NotNil(t, sess)
	assert.Equal(t, "signin@example.com", u.Email)
	assert.NotEmpty(t, sess.Token)
}

func TestSignIn_Success_ByUsername(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "user@example.com",
		Password:  "SecureP@ss1",
		FirstName: "User",
		Username:  "myuser",
	})
	require.NoError(t, err)

	u, sess, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Username: "myuser",
		Password: "SecureP@ss1",
	})

	require.NoError(t, err)
	assert.NotNil(t, u)
	assert.NotNil(t, sess)
	assert.Equal(t, "user@example.com", u.Email)
}

func TestSignIn_WrongPassword(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	signUpTestUser(t, eng, "wrong@example.com", "SecureP@ss1")

	_, _, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "wrong@example.com",
		Password: "WrongPassword1",
	})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials)
}

func TestSignIn_NonexistentUser(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, _, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "nobody@example.com",
		Password: "SecureP@ss1",
	})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials)
}

func TestSignIn_NoEmailOrUsername(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, _, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Password: "SecureP@ss1",
	})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials)
}

func TestSignIn_BannedUser(t *testing.T) {
	eng, s := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	// Sign up and then ban the user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "banned@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Banned User",
	})
	require.NoError(t, err)

	// Ban the user via store
	u.Banned = true
	future := time.Now().Add(24 * time.Hour)
	u.BanExpires = &future
	err = s.UpdateUser(ctx, u)
	require.NoError(t, err)

	_, _, err = eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "banned@example.com",
		Password: "SecureP@ss1",
	})
	assert.ErrorIs(t, err, account.ErrUserBanned)
}

// ──────────────────────────────────────────────────
// Email verification (dynamic setting) tests
// ──────────────────────────────────────────────────

func TestSignIn_EmailVerificationRequired_DynamicSetting(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	// Enable email verification via the dynamic setting.
	mgr := eng.Settings()
	require.NotNil(t, mgr)
	err := mgr.Set(ctx, "auth.require_email_verification", json.RawMessage(`true`),
		settings.ScopeGlobal, "", "", "", "test-admin")
	require.NoError(t, err)

	// Sign up a user (EmailVerified defaults to false).
	signUpTestUser(t, eng, "unverified@example.com", "SecureP@ss1")

	// Attempt sign-in — should fail with ErrEmailNotVerified.
	u, sess, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "unverified@example.com",
		Password: "SecureP@ss1",
	})
	assert.Nil(t, sess)
	assert.NotNil(t, u)
	assert.ErrorIs(t, err, account.ErrEmailNotVerified)
}

func TestSignIn_EmailVerificationDisabled_DynamicSetting(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	// Leave default (false) — sign-in should succeed for unverified users.
	signUpTestUser(t, eng, "unverified2@example.com", "SecureP@ss1")

	u, sess, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "unverified2@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	assert.NotNil(t, u)
	assert.NotNil(t, sess)
}

// ──────────────────────────────────────────────────
// SignOut tests
// ──────────────────────────────────────────────────

func TestSignOut_Success(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "signout@example.com",
		Password:  "SecureP@ss1",
		FirstName: "SignOut User",
	})
	require.NoError(t, err)

	err = eng.SignOut(ctx, sess.ID)
	require.NoError(t, err)

	// Session should be deleted
	_, err = eng.Store().GetSession(ctx, sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestSignOut_NonexistentSession(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()

	err := eng.SignOut(ctx, id.NewSessionID())
	assert.Error(t, err)
}

// ──────────────────────────────────────────────────
// Refresh tests
// ──────────────────────────────────────────────────

func TestRefresh_Success(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "refresh@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Refresh User",
	})
	require.NoError(t, err)

	// Save original values before refresh (memory store returns same pointer,
	// so after Refresh mutates the session, the original pointers are stale)
	originalToken := sess.Token
	originalRefreshToken := sess.RefreshToken

	newSess, err := eng.Refresh(ctx, originalRefreshToken)
	require.NoError(t, err)
	assert.NotNil(t, newSess)
	assert.NotEqual(t, originalToken, newSess.Token, "new session should have different token")
	assert.NotEqual(t, originalRefreshToken, newSess.RefreshToken, "new session should have different refresh token")
	assert.True(t, newSess.ExpiresAt.After(time.Now()))
}

func TestRefresh_InvalidToken(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()

	_, err := eng.Refresh(ctx, "nonexistent-refresh-token")
	assert.ErrorIs(t, err, account.ErrInvalidCredentials)
}

// ──────────────────────────────────────────────────
// GetMe / UpdateMe tests
// ──────────────────────────────────────────────────

func TestGetMe_Success(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "me@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Me User",
	})
	require.NoError(t, err)

	got, err := eng.GetMe(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.ID, got.ID)
	assert.Equal(t, u.Email, got.Email)
}

func TestGetMe_NonexistentUser(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()

	_, err := eng.GetMe(ctx, id.NewUserID())
	assert.Error(t, err)
}

func TestUpdateMe_Success(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "update@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Original",
		LastName:  "Name",
	})
	require.NoError(t, err)

	u.FirstName = "Updated"
	u.LastName = "Name"
	u.Image = "https://example.com/photo.jpg"

	err = eng.UpdateMe(ctx, u)
	require.NoError(t, err)

	got, err := eng.GetMe(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.FirstName)
	assert.Equal(t, "https://example.com/photo.jpg", got.Image)
}

// ──────────────────────────────────────────────────
// Session management tests
// ──────────────────────────────────────────────────

func TestListSessions(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	// Sign up creates one session
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "sessions@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Sessions User",
	})
	require.NoError(t, err)

	// Sign in creates another session
	_, _, err = eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "sessions@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)

	sessions, err := eng.ListSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

func TestRevokeSession_Success(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "revoke@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Revoke User",
	})
	require.NoError(t, err)

	err = eng.RevokeSession(ctx, sess.ID)
	require.NoError(t, err)

	sessions, err := eng.ListSessions(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestRevokeSession_NonexistentSession(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()

	err := eng.RevokeSession(ctx, id.NewSessionID())
	assert.Error(t, err)
}

// ──────────────────────────────────────────────────
// ResolveSessionByToken / ResolveUser tests
// ──────────────────────────────────────────────────

func TestResolveSessionByToken_Success(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	_, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "resolve@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Resolve User",
	})
	require.NoError(t, err)

	resolved, err := eng.ResolveSessionByToken(sess.Token)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, resolved.ID)
	assert.Equal(t, sess.UserID, resolved.UserID)
}

func TestResolveSessionByToken_Invalid(t *testing.T) {
	eng, _ := newTestEngine(t)

	_, err := eng.ResolveSessionByToken("nonexistent-token")
	assert.Error(t, err)
}

func TestResolveUser_Success(t *testing.T) {
	eng, _ := newTestEngine(t)
	ctx := context.Background()
	appID := testAppID(t)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "resolveuser@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Resolve User",
	})
	require.NoError(t, err)

	resolved, err := eng.ResolveUser(u.ID.String())
	require.NoError(t, err)
	assert.Equal(t, u.ID, resolved.ID)
	assert.Equal(t, u.Email, resolved.Email)
}

func TestResolveUser_InvalidID(t *testing.T) {
	eng, _ := newTestEngine(t)

	_, err := eng.ResolveUser("not-a-valid-id")
	assert.Error(t, err)
}

func TestResolveUser_NonexistentUser(t *testing.T) {
	eng, _ := newTestEngine(t)

	userID := id.NewUserID()
	_, err := eng.ResolveUser(userID.String())
	assert.Error(t, err)
}

// ──────────────────────────────────────────────────
// Account Lockout tests
// ──────────────────────────────────────────────────

func TestSignIn_AccountLockout_LocksAfterMaxAttempts(t *testing.T) {
	tracker := lockout.NewMemoryTracker(
		lockout.WithMaxAttempts(3),
		lockout.WithLockoutDuration(10*time.Minute),
	)

	eng, _ := newTestEngine(t,
		authsome.WithLockoutTracker(tracker),
		authsome.WithLockoutConfig(authsome.LockoutConfig{
			MaxAttempts: 3,
			Enabled:     true,
		}),
	)
	ctx := context.Background()
	appID := testAppID(t)

	signUpTestUser(t, eng, "lockout@example.com", "SecureP@ss1")

	// 3 failed attempts with wrong password
	for i := 0; i < 3; i++ {
		_, _, err := eng.SignIn(ctx, &account.SignInRequest{
			AppID:    appID,
			Email:    "lockout@example.com",
			Password: "WrongPassword1!",
		})
		assert.ErrorIs(t, err, account.ErrInvalidCredentials, "attempt %d", i+1)
	}

	// 4th attempt should be locked out (even with correct password)
	_, _, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "lockout@example.com",
		Password: "SecureP@ss1",
	})
	assert.ErrorIs(t, err, account.ErrAccountLocked)
}

func TestSignIn_AccountLockout_ResetOnSuccess(t *testing.T) {
	tracker := lockout.NewMemoryTracker(
		lockout.WithMaxAttempts(3),
		lockout.WithLockoutDuration(10*time.Minute),
	)

	eng, _ := newTestEngine(t,
		authsome.WithLockoutTracker(tracker),
		authsome.WithLockoutConfig(authsome.LockoutConfig{
			MaxAttempts: 3,
			Enabled:     true,
		}),
	)
	ctx := context.Background()
	appID := testAppID(t)

	signUpTestUser(t, eng, "reset@example.com", "SecureP@ss1")

	// 2 failed attempts (below threshold)
	for i := 0; i < 2; i++ {
		_, _, err := eng.SignIn(ctx, &account.SignInRequest{
			AppID:    appID,
			Email:    "reset@example.com",
			Password: "WrongPassword1!",
		})
		assert.ErrorIs(t, err, account.ErrInvalidCredentials)
	}

	// Successful login should reset the counter
	_, _, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "reset@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)

	// Another 2 failed attempts should NOT lock (counter was reset)
	for i := 0; i < 2; i++ {
		_, _, signInErr := eng.SignIn(ctx, &account.SignInRequest{
			AppID:    appID,
			Email:    "reset@example.com",
			Password: "WrongPassword1!",
		})
		assert.ErrorIs(t, signInErr, account.ErrInvalidCredentials)
	}

	// Should still be able to sign in (only 2 failures since reset, not 3)
	_, _, err = eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "reset@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
}

func TestSignIn_NoLockout_WhenTrackerNil(t *testing.T) {
	eng, _ := newTestEngine(t) // no lockout tracker
	ctx := context.Background()
	appID := testAppID(t)

	signUpTestUser(t, eng, "nolockout@example.com", "SecureP@ss1")

	// Many failed attempts without lockout
	for i := 0; i < 10; i++ {
		_, _, err := eng.SignIn(ctx, &account.SignInRequest{
			AppID:    appID,
			Email:    "nolockout@example.com",
			Password: "WrongPassword1!",
		})
		assert.ErrorIs(t, err, account.ErrInvalidCredentials)
	}

	// Should still be able to sign in
	_, _, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "nolockout@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
}
