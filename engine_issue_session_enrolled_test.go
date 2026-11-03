package authsome_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/user"
)

// enrollTOTP gives the user a TOTP enrolment in a fresh store on the
// engine's mfa plugin, verified or not.
func enrollTOTP(t *testing.T, eng *authsome.Engine, u *user.User, verified bool) {
	t.Helper()
	plg, ok := eng.Plugins().Plugin("mfa").(*mfa.Plugin)
	require.True(t, ok, "mfa plugin must be registered")
	store := mfa.NewMemoryStore()
	plg.SetStore(store)
	require.NoError(t, store.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID:       id.NewMFAID(),
		UserID:   u.ID,
		Method:   "totp",
		Secret:   "JBSWY3DPEHPK3PXP",
		Verified: verified,
	}))
}

// A user who turned a second factor on is asked for it at sign-in even
// when the app doesn't require MFA of everyone. Without this, turning
// two-step on protects nothing: the password alone still signs in.
func TestIssueSession_EnrolledUser_GateFiresWithoutAppRequirement(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"password", "social:google", "magiclink", "sso:saml-okta", "phone"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			eng, u, appID := issueSessionFixture(t)
			enrollTOTP(t, eng, u, true)

			res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
				User:       u,
				AppID:      appID,
				AuthMethod: method,
			})
			require.Nil(t, res)
			var mfaErr *authsome.MFARequiredError
			require.True(t, errors.As(err, &mfaErr), "AuthMethod=%q: error must be *MFARequiredError, got %v", method, err)
			assert.NotEmpty(t, mfaErr.Ticket)
			assert.Contains(t, mfaErr.AvailableMethods, "totp")
		})
	}
}

// The ticket the per-user gate hands out completes like any other: the
// challenge mints the session with MFAJustVerified, which passes.
func TestIssueSession_EnrolledUser_MFAJustVerifiedPasses(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	enrollTOTP(t, eng, u, true)

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:            u,
		AppID:           appID,
		AuthMethod:      "password+mfa",
		MFAJustVerified: true,
	})
	require.NoError(t, err)
	require.NotNil(t, res.Session)
}

// An enrolment that was started but never confirmed with a code isn't
// a factor the user can produce, so it doesn't gate sign-in.
func TestIssueSession_UnverifiedEnrollment_IssuesSession(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	enrollTOTP(t, eng, u, false)

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "password",
	})
	require.NoError(t, err)
	require.NotNil(t, res.Session)
}

// A passkey is possession plus a local biometric or PIN, already two
// factors, so it isn't followed by a code. An app that requires MFA of
// everyone still asks (TestIssueSession_GateFiresForEveryAuthMethod
// covers the app rule; this covers its passkey case).
func TestIssueSession_EnrolledUser_PasskeySkipsPerUserGate(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	enrollTOTP(t, eng, u, true)

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "passkey",
	})
	require.NoError(t, err)
	require.NotNil(t, res.Session)

	requireMFAOnApp(t, eng, appID)
	_, err = eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "passkey",
	})
	var mfaErr *authsome.MFARequiredError
	require.True(t, errors.As(err, &mfaErr), "the app-wide rule still applies to passkeys, got %v", err)
}

// failingStore fails every enrolment lookup, as a database outage would.
type failingStore struct{ *mfa.MemoryStore }

var errStoreDown = errors.New("store down")

func (failingStore) GetEnrollment(context.Context, id.UserID, string) (*mfa.Enrollment, error) {
	return nil, errStoreDown
}

// When the enrolment can't be read, sign-in is refused rather than let
// through without the second factor.
func TestIssueSession_EnrollmentLookupFails_Refuses(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	plg, ok := eng.Plugins().Plugin("mfa").(*mfa.Plugin)
	require.True(t, ok)
	plg.SetStore(failingStore{mfa.NewMemoryStore()})

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User:       u,
		AppID:      appID,
		AuthMethod: "password",
	})
	require.Nil(t, res)
	require.ErrorIs(t, err, errStoreDown)
}
