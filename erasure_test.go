package authsome_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/principal"
)

// An admin deletion erases the way a self-service deletion does: the user
// row stays as an anonymised tombstone, sessions end, devices go, and every
// delegation the user is party to is revoked.
func TestAdminDeleteUser_AnonymisesLikeDeleteAccount(t *testing.T) {
	eng, st := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	ctx := context.Background()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	u, sess, err := eng.SignUp(ctx, &account.SignUpRequest{AppID: appID, Email: "erase-me@example.com", Password: "SecureP@ss123", FirstName: "Era", LastName: "Sure"})
	require.NoError(t, err)

	require.NoError(t, st.CreateDevice(ctx, &device.Device{ID: id.NewDeviceID(), UserID: u.ID, AppID: appID, Fingerprint: "fp-1", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	userRef := principal.Ref{Kind: principal.KindUser, ID: u.ID.String()}
	agentRef := principal.Ref{Kind: principal.KindAgent, ID: id.NewAgentID().String()}
	require.NoError(t, st.CreateDelegation(ctx, &principal.Delegation{
		ID: id.NewDelegationID(), AppID: appID, Actor: agentRef, Subject: userRef, GrantKind: principal.GrantDelegation,
		CreatedAt: time.Now(),
	}))

	require.NoError(t, eng.AdminDeleteUser(ctx, id.NewUserID(), u.ID))

	got, err := st.GetUser(ctx, u.ID)
	require.NoError(t, err, "the row stays as a tombstone")
	require.NotNil(t, got.DeletedAt)
	assert.NotEqual(t, "erase-me@example.com", got.Email)
	assert.Contains(t, got.Email, "@deleted.local")
	assert.Empty(t, got.FirstName)
	assert.Empty(t, got.LastName)
	assert.Empty(t, got.PasswordHash)

	_, err = eng.ResolveSessionByToken(context.Background(), sess.Token)
	assert.Error(t, err, "sessions end with the account")
	devices, err := st.ListUserDevices(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, devices, "devices go with the account")
	active, err := st.ListDelegations(ctx, &principal.DelegationQuery{AppID: appID, Subject: &userRef, ActiveOnly: true, ActiveAsOf: time.Now()})
	require.NoError(t, err)
	assert.Empty(t, active, "delegations naming the user are revoked")
}
