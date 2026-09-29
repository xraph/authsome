package authsome_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/settings"
)

// The JWT session cross-check is on by default and is resolved under the
// token's own app, so one app opting out does not switch it off for another.

func TestJWTSessionChecker_OnByDefault(t *testing.T) {
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: appID, Email: "jwt-check@example.com", Password: "SecureP@ss123", FirstName: "J",
	})
	require.NoError(t, err)

	got, err := eng.JWTSessionChecker(appID.String(), sess.ID.String())
	require.NoError(t, err)
	require.NotNil(t, got, "the check is active without any configuration")
	assert.Equal(t, sess.ID, got.ID)

	_, err = eng.JWTSessionChecker(appID.String(), id.NewSessionID().String())
	assert.Error(t, err, "a session that is not in the store is refused")
}

func TestJWTSessionChecker_HonoursTheTokensApp(t *testing.T) {
	eng, _ := newTestEngine(t)
	secutil.RelaxAuthDefaults(t, eng)
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	other := id.NewAppID()

	// Only the other app opts out.
	require.NoError(t, eng.Settings().Set(context.Background(), "session.jwt_require_active_session",
		json.RawMessage(`false`), settings.ScopeApp, other.String(), "", "", "test"))

	got, err := eng.JWTSessionChecker(other.String(), id.NewSessionID().String())
	assert.NoError(t, err)
	assert.Nil(t, got, "the opted-out app skips the store lookup")

	_, err = eng.JWTSessionChecker(appID.String(), id.NewSessionID().String())
	assert.Error(t, err, "the platform app still checks")
}

var _ = authsome.SettingJWTRequireActiveSession
