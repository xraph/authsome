package authsome_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
)

// Sign-in must not say which identifiers exist or which accounts are
// banned: an unknown identifier pays the same hash as a known one, and a
// banned account answers the generic error to a wrong password.

func TestSignIn_UnknownIdentifierPaysTheHash(t *testing.T) {
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	var hashed int
	eng.SetDummyHashObserver(func(p account.PasswordPolicy) {
		hashed++
		assert.Equal(t, eng.Config().Password.BcryptCost, p.BcryptCost, "the dummy pays the configured cost")
	})

	_, _, err = eng.SignIn(context.Background(), &account.SignInRequest{AppID: appID, Email: "nobody@example.com", Password: "whatever"})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials)
	assert.Equal(t, 1, hashed, "the unknown-identifier path hashes once")
}

func TestSignIn_BannedIsOnlyDisclosedToTheRightPassword(t *testing.T) {
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	ctx := context.Background()
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "banned-probe@example.com", Password: "SecureP@ss123", FirstName: "B"})
	require.NoError(t, err)
	u.Banned = true
	require.NoError(t, eng.Store().UpdateUser(ctx, u))

	_, _, err = eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "banned-probe@example.com", Password: "wrong-password"})
	assert.ErrorIs(t, err, account.ErrInvalidCredentials, "a wrong password learns nothing about the ban")

	_, _, err = eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "banned-probe@example.com", Password: "SecureP@ss123"})
	assert.ErrorIs(t, err, account.ErrUserBanned, "the right password is refused as banned")
}
