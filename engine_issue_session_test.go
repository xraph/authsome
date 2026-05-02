package authsome_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/user"
)

// issueSessionFixture spins up a test engine with the MFA plugin
// registered (using the in-memory MFA store) and returns the engine
// plus a freshly-created user. Tests can additionally call
// requireMFAOnApp(t, eng, appID) to flip the per-app MFARequired
// flag.
func issueSessionFixture(t *testing.T) (*authsome.Engine, *user.User, id.AppID) {
	t.Helper()

	mfaPlugin := mfa.New()
	mfaPlugin.SetStore(mfa.NewMemoryStore())

	eng := secutil.NewTestEngine(t, authsome.WithPlugin(mfaPlugin))
	secutil.RelaxAuthDefaults(t, eng)

	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)

	u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:     appID,
		Email:     "issuesession@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Issue",
	})
	require.NoError(t, err)

	return eng, u, appID
}

func requireMFAOnApp(t *testing.T, eng *authsome.Engine, appID id.AppID) {
	t.Helper()
	tru := true
	require.NoError(t, eng.Store().SetAppClientConfig(context.Background(), &appclientconfig.Config{
		ID:          id.NewAppClientConfigID(),
		AppID:       appID,
		MFARequired: &tru,
	}))
}

// TestIssueSession_NoMFARequired_IssuesSession pins the happy path:
// when MFARequired is not set on the app config the gate is dormant
// and IssueSession returns a real session.
func TestIssueSession_NoMFARequired_IssuesSession(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "password",
		IPAddress:  "127.0.0.1",
		UserAgent:  "go-test/1.0",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Session)
	assert.Equal(t, u.ID, res.Session.UserID)
	assert.NotEmpty(t, res.Session.Token, "session token must be issued")
}

// TestIssueSession_MFARequiredButUserVerified_IssuesSession pins that
// a user with verified MFA still gets a session — the gate fires only
// when the user *lacks* a verified factor.
func TestIssueSession_MFARequiredButUserVerified_IssuesSession(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	requireMFAOnApp(t, eng, appID)

	// Plant a verified MFA enrollment via the plugin's store directly.
	plg, ok := eng.Plugins().Plugin("mfa").(*mfa.Plugin)
	require.True(t, ok, "mfa plugin must be registered")
	store := mfa.NewMemoryStore()
	plg.SetStore(store)
	require.NoError(t, store.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID:       id.NewMFAID(),
		UserID:   u.ID,
		Method:   "totp",
		Secret:   "JBSWY3DPEHPK3PXP",
		Verified: true,
	}))

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "password",
	})
	require.NoError(t, err)
	require.NotNil(t, res.Session)
}

// TestIssueSession_MFARequiredAndUserUnenrolled_ReturnsTicket pins
// the gate firing path: the user lacks MFA, MFARequired is true, so
// the function returns *MFARequiredError carrying a non-empty ticket.
func TestIssueSession_MFARequiredAndUserUnenrolled_ReturnsTicket(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	requireMFAOnApp(t, eng, appID)

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "password",
	})
	require.Error(t, err)
	require.Nil(t, res)

	var mfaErr *authsome.MFARequiredError
	require.True(t, errors.As(err, &mfaErr), "error must be *MFARequiredError, got %T", err)
	assert.NotEmpty(t, mfaErr.Ticket, "ticket must be non-empty so client can complete the round-trip")
	assert.True(t, errors.Is(err, account.ErrMFARequired), "must wrap account.ErrMFARequired so existing callers keep working")
}

// TestIssueSession_TicketPersistedToCeremony pins that the ticket
// returned from the gate is loadable via Engine.LoadMFATicket.
func TestIssueSession_TicketPersistedToCeremony(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	requireMFAOnApp(t, eng, appID)

	_, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "password",
		IPAddress:  "10.0.0.5",
		UserAgent:  "go-test/2.0",
	})
	var mfaErr *authsome.MFARequiredError
	require.True(t, errors.As(err, &mfaErr))

	loaded, err := eng.LoadMFATicket(context.Background(), mfaErr.Ticket)
	require.NoError(t, err)
	assert.Equal(t, u.ID, loaded.UserID)
	assert.Equal(t, "password", loaded.AuthMethod)
	assert.Equal(t, "10.0.0.5", loaded.IPAddress)
	assert.Equal(t, "go-test/2.0", loaded.UserAgent)
}

// TestIssueSession_MfaJustVerifiedBypassesGate pins that the
// post-MFA-verify path bypasses the gate; the challenge handler will
// pass MFAJustVerified=true after validating a code against a ticket.
func TestIssueSession_MfaJustVerifiedBypassesGate(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	requireMFAOnApp(t, eng, appID)

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:            u,
		AppID:           appID,
		AuthMethod:      "password+mfa",
		MFAJustVerified: true,
	})
	require.NoError(t, err, "MFAJustVerified must bypass the gate")
	require.NotNil(t, res.Session)
}

// TestIssueSession_AuditMetadataIncludesAuthMethod pins that the
// audit log captures the auth_method dimension so operators can
// distinguish password vs social vs magiclink etc.
func TestIssueSession_AuditMetadataIncludesAuthMethod(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)

	ch := secutil.NewBufferedChronicle()
	secutil.AttachChronicle(t, eng, ch)

	_, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "social:google",
	})
	require.NoError(t, err)

	secutil.AssertAuditEvent(t, ch, "issue_session", func(ev *bridge.AuditEvent) {
		require.NotNil(t, ev)
		assert.Equal(t, "social:google", ev.Metadata["auth_method"])
	})
}
