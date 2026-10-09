package authsome_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
)

// A ban must hold on every path that turns a credential into a principal,
// not only at password sign-in.

func bannedFixture(t *testing.T, expires *time.Time) (*authsome.Engine, id.UserID, string) {
	t.Helper()
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	u, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: appID, Email: "banned@example.com", Password: "SecureP@ss123", FirstName: "B",
	})
	require.NoError(t, err)

	u.Banned = true
	u.BanExpires = expires
	require.NoError(t, eng.Store().UpdateUser(context.Background(), u))
	return eng, u.ID, sess.RefreshToken
}

func TestBanned_ResolveUserRefuses(t *testing.T) {
	eng, uid, _ := bannedFixture(t, nil)
	_, err := eng.ResolveUser(context.Background(), uid.String())
	assert.ErrorIs(t, err, account.ErrUserBanned)
}

func TestBanned_RefreshRefuses(t *testing.T) {
	eng, _, refresh := bannedFixture(t, nil)
	_, err := eng.Refresh(context.Background(), refresh)
	assert.ErrorIs(t, err, account.ErrUserBanned)
}

func TestBanned_IssueSessionRefuses(t *testing.T) {
	eng, uid, _ := bannedFixture(t, nil)
	u, err := eng.Store().GetUser(context.Background(), uid)
	require.NoError(t, err)
	_, err = eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{User: u, AuthMethod: "test"})
	assert.ErrorIs(t, err, account.ErrUserBanned)
}

func TestBanned_ExpiredBanNoLongerApplies(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	eng, uid, refresh := bannedFixture(t, &past)
	_, err := eng.ResolveUser(context.Background(), uid.String())
	assert.NoError(t, err, "an expired ban must not block")
	_, err = eng.Refresh(context.Background(), refresh)
	assert.NoError(t, err)
}
