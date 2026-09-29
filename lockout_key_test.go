package authsome_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/lockout"
)

// A lockout is keyed by the account and the client's network, so an
// attacker who fails from one network cannot lock the owner out from
// another, and an operator can lift every network's lock in one call.

func lockoutFixture(t *testing.T) (*authsome.Engine, id.AppID, id.UserID) {
	t.Helper()
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	eng.SetLockoutTracker(lockout.NewMemoryTracker(lockout.WithMaxAttempts(3), lockout.WithLockoutDuration(10*time.Minute)))
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{AppID: appID, Email: "locked@example.com", Password: "SecureP@ss123", FirstName: "L"})
	require.NoError(t, err)
	return eng, appID, u.ID
}

func signInFrom(eng *authsome.Engine, appID id.AppID, ip, password string) error {
	_, _, err := eng.SignIn(context.Background(), &account.SignInRequest{
		AppID: appID, Email: "locked@example.com", Password: password, IPAddress: ip,
	})
	return err
}

func lockFrom(t *testing.T, eng *authsome.Engine, appID id.AppID, ip string) {
	t.Helper()
	for i := 0; i < 3; i++ {
		require.ErrorIs(t, signInFrom(eng, appID, ip, "wrong-password"), account.ErrInvalidCredentials)
	}
}

func TestLockout_IsScopedToTheClientNetwork(t *testing.T) {
	eng, appID, _ := lockoutFixture(t)
	lockFrom(t, eng, appID, "203.0.113.5")

	err := signInFrom(eng, appID, "203.0.113.77", "SecureP@ss123")
	require.ErrorIs(t, err, account.ErrAccountLocked, "the same /24 is locked")
	var locked *account.LockedError
	require.True(t, errors.As(err, &locked), "the error carries when the lock lifts")
	assert.True(t, locked.Until.After(time.Now()))
	assert.GreaterOrEqual(t, locked.RetryAfter(time.Now()), 1)

	assert.NoError(t, signInFrom(eng, appID, "198.51.100.9", "SecureP@ss123"), "the owner on another network still signs in")
}

func TestLockout_IPv6UsesA64(t *testing.T) {
	eng, appID, _ := lockoutFixture(t)
	lockFrom(t, eng, appID, "2001:db8:1:2::10")
	assert.ErrorIs(t, signInFrom(eng, appID, "2001:db8:1:2:ffff::1", "SecureP@ss123"), account.ErrAccountLocked, "same /64")
	assert.NoError(t, signInFrom(eng, appID, "2001:db8:9:9::1", "SecureP@ss123"), "another /64 is untouched")
}

func TestAdminUnlockUser_ClearsEveryNetwork(t *testing.T) {
	eng, appID, uid := lockoutFixture(t)
	lockFrom(t, eng, appID, "203.0.113.5")
	lockFrom(t, eng, appID, "192.0.2.5")
	require.ErrorIs(t, signInFrom(eng, appID, "203.0.113.5", "SecureP@ss123"), account.ErrAccountLocked)

	var seen []*hook.Event
	eng.Hooks().On("unlock-test", func(_ context.Context, ev *hook.Event) error {
		if ev.Action == hook.ActionAdminUnlockUser {
			seen = append(seen, ev)
		}
		return nil
	})
	require.NoError(t, eng.AdminUnlockUser(context.Background(), id.NewUserID(), uid))

	assert.NoError(t, signInFrom(eng, appID, "203.0.113.5", "SecureP@ss123"))
	assert.NoError(t, signInFrom(eng, appID, "192.0.2.5", "SecureP@ss123"))
	require.Len(t, seen, 1, "the unlock is on the trail")
	assert.Equal(t, uid.String(), seen[0].ResourceID)
}
