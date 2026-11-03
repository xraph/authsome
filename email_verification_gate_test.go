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
	"github.com/xraph/authsome/plugins/password"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

func requireVerification(t *testing.T, eng *authsome.Engine, on bool) {
	t.Helper()
	raw := json.RawMessage(`false`)
	if on {
		raw = json.RawMessage(`true`)
	}
	require.NoError(t, eng.Settings().Set(context.Background(), "auth.require_email_verification", raw,
		settings.ScopeGlobal, "", "", "", "verification-gate-test"))
}

// A session from sign-up worked in any tab, so "verify your email first"
// was only ever a screen. With verification required, sign-up creates the
// account and sends the code, and hands out no session.
func TestSignUp_NoSessionUntilEmailVerified(t *testing.T) {
	eng, _ := newTestEngine(t)
	requireVerification(t, eng, true)

	u, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: testAppID(t), Email: "pending@example.com", Password: "SecureP@ss123",
	})
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.False(t, u.EmailVerified)
	assert.Nil(t, sess, "an unverified sign-up must not get a session")
}

func TestSignUp_SessionWhenVerificationIsOff(t *testing.T) {
	eng, _ := newTestEngine(t)
	requireVerification(t, eng, false)

	_, sess, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID: testAppID(t), Email: "open@example.com", Password: "SecureP@ss123",
	})
	require.NoError(t, err)
	assert.NotNil(t, sess)
}

// IssueSession mints for every sign-in method, so the rule holds for
// passkeys, magic links and social sign-in too, not just passwords.
func TestIssueSession_RefusesAnUnverifiedEmail(t *testing.T) {
	eng, st := newTestEngine(t)
	ctx := context.Background()
	requireVerification(t, eng, true)

	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID: testAppID(t), Email: "gate@example.com", Password: "SecureP@ss123",
	})
	require.NoError(t, err)

	_, err = eng.IssueSession(ctx, &authsome.IssueSessionRequest{User: u, AppID: testAppID(t), AuthMethod: "passkey"})
	require.ErrorIs(t, err, account.ErrEmailNotVerified)

	u.EmailVerified = true
	require.NoError(t, st.UpdateUser(ctx, u))
	res, err := eng.IssueSession(ctx, &authsome.IssueSessionRequest{User: u, AppID: testAppID(t), AuthMethod: "passkey"})
	require.NoError(t, err)
	assert.NotNil(t, res.Session)
}

// Phone and anonymous accounts have no email to verify.
func TestIssueSession_AllowsAnAccountWithNoEmail(t *testing.T) {
	eng, st := newTestEngine(t)
	ctx := context.Background()
	requireVerification(t, eng, true)

	now := time.Now()
	u := &user.User{ID: id.NewUserID(), AppID: testAppID(t), Phone: "+15555550100", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, st.CreateUser(ctx, u))
	_, err := eng.IssueSession(ctx, &authsome.IssueSessionRequest{User: u, AppID: testAppID(t), AuthMethod: "phone"})
	require.NoError(t, err)
}

// The client config tells the UI whether to wait for verification. It
// used to read only the environment, so it could say "verify first"
// while sign-in let the account straight in, or the other way round.
func TestClientConfig_VerificationMatchesEnforcement(t *testing.T) {
	eng, _ := newTestEngine(t, authsome.WithPlugin(password.New()))
	ctx := context.Background()
	for _, on := range []bool{false, true} {
		requireVerification(t, eng, on)
		assert.Equal(t, on, eng.EmailVerificationRequired(ctx, testAppID(t)))
		cfg := eng.ClientConfig(ctx, testAppID(t))
		require.NotNil(t, cfg.EmailVerification)
		assert.Equal(t, on, cfg.EmailVerification.Required, "client config with the setting %v", on)
	}
}
