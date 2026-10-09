package authsome_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/middleware"
)

// Removing access has to reach every credential the person holds, not only
// the sessions the sweep happened to remember.

type accessFixture struct {
	eng   *authsome.Engine
	appID id.AppID
	uid   id.UserID
	first id.SessionID
	other id.SessionID
	key   id.APIKeyID
}

func newAccessFixture(t *testing.T) *accessFixture {
	t.Helper()
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	ctx := context.Background()

	u, first, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "access@example.com", Password: "SecureP@ss123", FirstName: "A"})
	require.NoError(t, err)
	_, other, err := eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "access@example.com", Password: "SecureP@ss123"})
	require.NoError(t, err)

	_, hash, prefix, err := apikey.GenerateKey()
	require.NoError(t, err)
	now := time.Now()
	keyID := id.NewAPIKeyID()
	require.NoError(t, eng.APIKeyStore().CreateAPIKey(ctx, &apikey.APIKey{
		ID: keyID, AppID: appID, UserID: u.ID, Name: "k", KeyHash: hash, KeyPrefix: prefix,
		CreatedAt: now, UpdatedAt: now,
	}))
	return &accessFixture{eng: eng, appID: appID, uid: u.ID, first: first.ID, other: other.ID, key: keyID}
}

func (f *accessFixture) sessionIDs(t *testing.T) []id.SessionID {
	t.Helper()
	list, err := f.eng.Store().ListUserSessions(context.Background(), f.uid)
	require.NoError(t, err)
	out := make([]id.SessionID, 0, len(list))
	for _, s := range list {
		out = append(out, s.ID)
	}
	return out
}

func (f *accessFixture) keyRevoked(t *testing.T) bool {
	t.Helper()
	k, err := f.eng.APIKeyStore().GetAPIKey(context.Background(), f.key)
	require.NoError(t, err)
	return k.Revoked
}

func TestAdminBanUser_RevokesSessionsAndKeys(t *testing.T) {
	f := newAccessFixture(t)
	require.Len(t, f.sessionIDs(t), 2)

	require.NoError(t, f.eng.AdminBanUser(context.Background(), id.NewUserID(), f.uid, "policy", nil))

	assert.Empty(t, f.sessionIDs(t), "every session ends with the ban")
	assert.True(t, f.keyRevoked(t), "every key ends with the ban")
}

func TestRevokeOtherUserSessions_KeepsTheCurrentOne(t *testing.T) {
	f := newAccessFixture(t)
	require.NoError(t, f.eng.RevokeOtherUserSessions(context.Background(), f.uid, f.first))
	assert.Equal(t, []id.SessionID{f.first}, f.sessionIDs(t))
	assert.False(t, f.keyRevoked(t), "a credential change does not touch API keys")
}

func TestChangePassword_SignsOutOtherSessions(t *testing.T) {
	f := newAccessFixture(t)
	ctx := middleware.WithSessionID(context.Background(), f.first)
	require.NoError(t, f.eng.ChangePassword(ctx, f.uid, "SecureP@ss123", "An0therStr0ng!Pass"))
	assert.Equal(t, []id.SessionID{f.first}, f.sessionIDs(t), "only the session that changed the password survives")
}
