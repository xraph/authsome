package account_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
)

// ──────────────────────────────────────────────────
// PasswordPolicy
// ──────────────────────────────────────────────────

func TestPasswordPolicy_MinLength(t *testing.T) {
	p := account.PasswordPolicy{MinLength: 8}
	assert.ErrorIs(t, p.ValidatePassword("short"), account.ErrWeakPassword)
	assert.NoError(t, p.ValidatePassword("longEnough1!"))
}

func TestPasswordPolicy_RequireUppercase(t *testing.T) {
	p := account.PasswordPolicy{MinLength: 1, RequireUppercase: true}
	assert.ErrorIs(t, p.ValidatePassword("alllower"), account.ErrWeakPassword)
	assert.NoError(t, p.ValidatePassword("HasUpper"))
}

func TestPasswordPolicy_RequireLowercase(t *testing.T) {
	p := account.PasswordPolicy{MinLength: 1, RequireLowercase: true}
	assert.ErrorIs(t, p.ValidatePassword("ALLUPPER"), account.ErrWeakPassword)
	assert.NoError(t, p.ValidatePassword("HASLower"))
}

func TestPasswordPolicy_RequireDigit(t *testing.T) {
	p := account.PasswordPolicy{MinLength: 1, RequireDigit: true}
	assert.ErrorIs(t, p.ValidatePassword("nodigits"), account.ErrWeakPassword)
	assert.NoError(t, p.ValidatePassword("has1digit"))
}

func TestPasswordPolicy_RequireSpecial(t *testing.T) {
	p := account.PasswordPolicy{MinLength: 1, RequireSpecial: true}
	assert.ErrorIs(t, p.ValidatePassword("nospecial123"), account.ErrWeakPassword)
	assert.NoError(t, p.ValidatePassword("has!special"))
}

func TestPasswordPolicy_AllRules(t *testing.T) {
	p := account.PasswordPolicy{
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireDigit:     true,
		RequireSpecial:   true,
	}
	assert.ErrorIs(t, p.ValidatePassword("abc"), account.ErrWeakPassword)
	assert.NoError(t, p.ValidatePassword("MyPass1!xtra"))
}

// ──────────────────────────────────────────────────
// HashPassword / CheckPassword
// ──────────────────────────────────────────────────

func TestHashPassword(t *testing.T) {
	hash, err := account.HashPassword("secret123", 4) // low cost for speed
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "secret123", hash)
}

func TestCheckPassword(t *testing.T) {
	hash, err := account.HashPassword("secret123", 4)
	require.NoError(t, err)

	assert.NoError(t, account.CheckPassword(hash, "secret123"))
	assert.Error(t, account.CheckPassword(hash, "wrongpassword"))
}

func TestHashPassword_DefaultCost(t *testing.T) {
	hash, err := account.HashPassword("pass", 0) // 0 → default cost
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
}

// ──────────────────────────────────────────────────
// NewSession
// ──────────────────────────────────────────────────

func TestNewSession(t *testing.T) {
	appID := id.NewAppID()
	userID := id.NewUserID()
	cfg := account.SessionConfig{
		TokenTTL:        time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	}

	sess, err := account.NewSession(appID, userID, cfg)
	require.NoError(t, err)
	require.NotNil(t, sess)

	assert.NotEmpty(t, sess.Token)
	assert.NotEmpty(t, sess.RefreshToken)
	assert.NotEqual(t, sess.Token, sess.RefreshToken)
	assert.Equal(t, appID.String(), sess.AppID.String())
	assert.Equal(t, userID.String(), sess.UserID.String())
	assert.True(t, sess.ExpiresAt.After(time.Now()))
	assert.True(t, sess.RefreshTokenExpiresAt.After(sess.ExpiresAt))
}

// ──────────────────────────────────────────────────
// NewUser
// ──────────────────────────────────────────────────

func TestNewUser(t *testing.T) {
	appID := id.NewAppID()
	req := &account.SignUpRequest{
		AppID:     appID,
		Email:     "  Alice@Example.COM  ",
		Password:  "pass",
		FirstName: "Alice",
		Username:  "alice",
	}

	u := account.NewUser(req, "$2a$10$hash")
	assert.Equal(t, "alice@example.com", u.Email, "email should be trimmed and lowercased")
	assert.Equal(t, "Alice", u.FirstName)
	assert.Equal(t, "alice", u.Username)
	assert.Equal(t, "$2a$10$hash", u.PasswordHash)
	assert.False(t, u.CreatedAt.IsZero())
}

// ──────────────────────────────────────────────────
// RefreshSession
// ──────────────────────────────────────────────────

func TestRefreshSession(t *testing.T) {
	appID := id.NewAppID()
	userID := id.NewUserID()
	cfg := account.SessionConfig{
		TokenTTL:           time.Hour,
		RefreshTokenTTL:    24 * time.Hour,
		RotateRefreshToken: true,
	}

	sess, err := account.NewSession(appID, userID, cfg)
	require.NoError(t, err)

	oldToken := sess.Token
	oldRefresh := sess.RefreshToken

	err = account.RefreshSession(sess, cfg)
	require.NoError(t, err)

	assert.NotEqual(t, oldToken, sess.Token, "new access token expected")
	assert.NotEqual(t, oldRefresh, sess.RefreshToken, "new refresh token expected")
}

func TestRefreshSession_NoRotation(t *testing.T) {
	appID := id.NewAppID()
	userID := id.NewUserID()
	cfg := account.SessionConfig{
		TokenTTL:           time.Hour,
		RefreshTokenTTL:    24 * time.Hour,
		RotateRefreshToken: false,
	}

	sess, err := account.NewSession(appID, userID, cfg)
	require.NoError(t, err)

	oldToken := sess.Token
	oldRefresh := sess.RefreshToken
	oldRefreshExpiry := sess.RefreshTokenExpiresAt

	err = account.RefreshSession(sess, cfg)
	require.NoError(t, err)

	assert.NotEqual(t, oldToken, sess.Token, "new access token expected")
	assert.Equal(t, oldRefresh, sess.RefreshToken, "refresh token should be reused")
	assert.Equal(t, oldRefreshExpiry, sess.RefreshTokenExpiresAt, "refresh token expiry should not change")
}

// ──────────────────────────────────────────────────
// Argon2id Hashing
// ──────────────────────────────────────────────────

func TestArgon2_HashAndVerify(t *testing.T) {
	params := account.DefaultArgon2Params()
	hash, err := account.HashPasswordArgon2("testPassword123!", params)
	require.NoError(t, err)
	assert.True(t, account.IsArgon2Hash(hash))

	// Correct password
	err = account.CheckPasswordArgon2(hash, "testPassword123!")
	assert.NoError(t, err)

	// Wrong password
	err = account.CheckPasswordArgon2(hash, "wrongPassword")
	assert.Error(t, err)
}

func TestArgon2_BackwardCompat(t *testing.T) {
	// Bcrypt hash should not be detected as Argon2
	bcryptHash, err := account.HashPassword("test123", 10)
	require.NoError(t, err)
	assert.False(t, account.IsArgon2Hash(bcryptHash))

	// CheckPassword auto-detects algorithm
	err = account.CheckPassword(bcryptHash, "test123")
	assert.NoError(t, err)

	// Argon2 hash detected and verified via CheckPassword
	argon2Hash, err := account.HashPasswordArgon2("test123", account.DefaultArgon2Params())
	require.NoError(t, err)
	err = account.CheckPassword(argon2Hash, "test123")
	assert.NoError(t, err)
}

func TestNeedsRehash(t *testing.T) {
	bcryptHash, _ := account.HashPassword("test", 10)
	argon2Hash, _ := account.HashPasswordArgon2("test", account.DefaultArgon2Params())

	// bcrypt hash with argon2id policy -> needs rehash
	assert.True(t, account.NeedsRehash(bcryptHash, account.PasswordPolicy{Algorithm: "argon2id"}))

	// argon2 hash with bcrypt policy -> needs rehash
	assert.True(t, account.NeedsRehash(argon2Hash, account.PasswordPolicy{Algorithm: "bcrypt"}))

	// matching algorithms -> no rehash
	assert.False(t, account.NeedsRehash(bcryptHash, account.PasswordPolicy{Algorithm: "bcrypt"}))
	assert.False(t, account.NeedsRehash(argon2Hash, account.PasswordPolicy{Algorithm: "argon2id"}))

	// empty algorithm defaults to bcrypt
	assert.False(t, account.NeedsRehash(bcryptHash, account.PasswordPolicy{}))
}

func TestHashPasswordWithPolicy(t *testing.T) {
	// bcrypt
	hash, err := account.HashPasswordWithPolicy("test123", account.PasswordPolicy{BcryptCost: 10})
	require.NoError(t, err)
	assert.False(t, account.IsArgon2Hash(hash))

	// argon2id
	hash, err = account.HashPasswordWithPolicy("test123", account.PasswordPolicy{
		Algorithm:    "argon2id",
		Argon2Params: account.DefaultArgon2Params(),
	})
	require.NoError(t, err)
	assert.True(t, account.IsArgon2Hash(hash))
}

// ──────────────────────────────────────────────────
// NewVerification / NewPasswordReset
// ──────────────────────────────────────────────────

func TestNewVerification(t *testing.T) {
	v, err := account.NewVerification(context.Background(), id.NewAppID(), id.NewUserID(), account.VerificationEmail, time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, v.Token)
	assert.Equal(t, account.VerificationEmail, v.Type)
	assert.True(t, v.ExpiresAt.After(time.Now()))
}

func TestNewPasswordReset(t *testing.T) {
	pr, err := account.NewPasswordReset(context.Background(), id.NewAppID(), id.NewUserID(), time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, pr.Token)
	assert.True(t, pr.ExpiresAt.After(time.Now()))
}
