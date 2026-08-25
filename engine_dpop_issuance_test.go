package authsome_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
)

// setAppDPoPMode writes the dpop.mode setting at app scope, the same way the
// dashboard would, so Engine.DPoPModeForApp resolves it.
func setAppDPoPMode(t *testing.T, eng *authsome.Engine, appID id.AppID, mode string) {
	t.Helper()
	mgr := eng.Settings()
	require.NotNil(t, mgr)
	raw, err := json.Marshal(mode)
	require.NoError(t, err)
	require.NoError(t, mgr.Set(context.Background(), "dpop.mode", raw,
		settings.ScopeApp, appID.String(), appID.String(), "", "test"))
}

// TestIssueSession_RequiredMode_RefusesUnboundSession is the backstop the
// whole issuance-coverage fix rests on.
//
// Enforcement follows the token: middleware.enforceDPoP returns nil the moment
// it sees an empty thumbprint, so a session minted without one is exempt from
// proof-of-possession for its entire life no matter what the app's mode says.
// That makes every mint site a policy decision, and any site that forgets to
// resolve a binding silently opts its users out of the app's own mandate.
//
// Putting the refusal in IssueSession means the default for a forgetful caller
// is closed rather than open.
func TestIssueSession_RequiredMode_RefusesUnboundSession(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	setAppDPoPMode(t, eng, appID, "required")

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "password",
		IPAddress:  "127.0.0.1",
		UserAgent:  "go-test/1.0",
	})

	require.Error(t, err, "mode=required must refuse to mint a session with no thumbprint")
	assert.Nil(t, res)
	assert.ErrorIs(t, err, authsome.ErrDPoPBindingRequired)
}

// TestIssueSession_MFATicketCarriesThumbprint pins the fix for the inversion
// the review found: enabling MFA used to silently disable DPoP for that user.
//
// Sign-in proves a thumbprint, but IssueSession returns MFARequiredError
// before minting anything, and the MFA plugin then issues its own session at
// the end of the ceremony. With nothing carrying the binding across the
// ticket, the second factor turned a bound session into an unbound one — the
// more secure configuration dropping the stronger token binding.
//
// The ticket is the only thing that spans both legs, so it carries the
// thumbprint.
func TestIssueSession_MFATicketCarriesThumbprint(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	requireMFAOnApp(t, eng, appID)

	const jkt = "test-thumbprint-from-signin"
	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "password",
		DPoPJKT:    jkt,
	})
	require.Nil(t, res)

	var mfaErr *authsome.MFARequiredError
	require.ErrorAs(t, err, &mfaErr, "MFA gate must fire and hand back a ticket")
	require.NotEmpty(t, mfaErr.Ticket)

	payload, loadErr := eng.LoadMFATicket(context.Background(), mfaErr.Ticket)
	require.NoError(t, loadErr)
	assert.Equal(t, jkt, payload.DPoPJKT,
		"the thumbprint proved at sign-in must survive to the challenge handler")
}

// TestImpersonate_RequiredMode_RefusesUnboundSession covers the mint site the
// review did not list.
//
// Impersonate builds its session with account.NewSession directly rather than
// through IssueSession, so the central gate never sees it. Under mode=required
// that made admin impersonation a standing way to obtain an unbound session
// for an arbitrary user — the widest exemption of the set, since the whole
// point of the session is that it acts as somebody else.
func TestImpersonate_RequiredMode_RefusesUnboundSession(t *testing.T) {
	t.Parallel()
	eng, target, appID := issueSessionFixture(t)
	setAppDPoPMode(t, eng, appID, "required")

	_, sess, err := eng.Impersonate(context.Background(), id.NewUserID(), target.ID)

	require.Error(t, err, "an unbound impersonation session must be refused like any other")
	assert.Nil(t, sess)
	assert.ErrorIs(t, err, authsome.ErrDPoPBindingRequired)
}

// TestImpersonate_RequiredMode_BindsWhenProofPresented is the other half. The
// admin console requesting the impersonation is an ordinary HTTP client that
// holds a key and can prove possession, so the session it gets back is bound
// to that key rather than refused.
func TestImpersonate_RequiredMode_BindsWhenProofPresented(t *testing.T) {
	t.Parallel()
	eng, target, appID := issueSessionFixture(t)
	setAppDPoPMode(t, eng, appID, "required")

	const jkt = "admin-console-thumbprint"
	_, sess, err := eng.Impersonate(context.Background(), id.NewUserID(), target.ID,
		authsome.WithImpersonationDPoPBinding(jkt))

	require.NoError(t, err)
	require.NotNil(t, sess)
	assert.Equal(t, jkt, sess.DPoPJKT)
}
