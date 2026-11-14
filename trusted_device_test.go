package authsome_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/mfa"
)

func TestTrustedDeviceToken_HeaderBeforeCookie(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/signin", nil)
	assert.Empty(t, authsome.TrustedDeviceToken(r))
	assert.Empty(t, authsome.TrustedDeviceToken(nil))

	r.AddCookie(&http.Cookie{Name: authsome.TrustedDeviceCookieName, Value: "from-cookie"})
	assert.Equal(t, "from-cookie", authsome.TrustedDeviceToken(r))

	r.Header.Set(authsome.TrustedDeviceHeader, "from-header")
	assert.Equal(t, "from-header", authsome.TrustedDeviceToken(r))
}

// trustDevice stores a trusted device for the user directly, the way the
// challenge would, and returns the token it answers to. The plugin stores the
// token's SHA-256, hex encoded.
func trustDevice(t *testing.T, store mfa.Store, userID id.UserID, created, expires time.Time) string {
	t.Helper()
	token := id.NewTrustedDeviceID().String() // any unguessable string will do
	sum := sha256.Sum256([]byte(token))
	require.NoError(t, store.CreateTrustedDevice(context.Background(), &mfa.TrustedDevice{
		ID: id.NewTrustedDeviceID(), UserID: userID, TokenHash: hex.EncodeToString(sum[:]),
		CreatedAt: created, LastUsedAt: created, ExpiresAt: expires,
	}))
	return token
}

func mfaStoreOf(t *testing.T, eng *authsome.Engine) *mfa.MemoryStore {
	t.Helper()
	plg, ok := eng.Plugins().Plugin("mfa").(*mfa.Plugin)
	require.True(t, ok)
	s := mfa.NewMemoryStore()
	plg.SetStore(s)
	return s
}

// The session a trusted device gets is verified as of when the device passed
// its second factor, not as of now, so a host asking for a recent second
// factor is not fooled by a month-old trust.
func TestIssueSession_TrustedDeviceStampsWhenTrusted(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	store := mfaStoreOf(t, eng)
	require.NoError(t, store.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID: id.NewMFAID(), UserID: u.ID, Method: "totp", Secret: "JBSWY3DPEHPK3PXP", Verified: true,
	}))
	trustedAt := time.Now().Add(-10 * 24 * time.Hour).UTC().Truncate(time.Second)
	token := trustDevice(t, store, u.ID, trustedAt, trustedAt.Add(30*24*time.Hour))

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User: u, AppID: appID, AuthMethod: "password", TrustedDeviceToken: token,
	})
	require.NoError(t, err)
	assert.True(t, trustedAt.Equal(res.Session.MFAVerifiedAt), "want %v, got %v", trustedAt, res.Session.MFAVerifiedAt)
}

// A trusted-device token never earns a stamp when the gate would not have
// fired: a user without two-step has no second factor to vouch for.
func TestIssueSession_TrustedDeviceIgnoredWithoutGate(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	store := mfaStoreOf(t, eng)
	token := trustDevice(t, store, u.ID, time.Now(), time.Now().Add(time.Hour))

	res, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User: u, AppID: appID, AuthMethod: "password", TrustedDeviceToken: token,
	})
	require.NoError(t, err)
	assert.True(t, res.Session.MFAVerifiedAt.IsZero())
}

// failingTrustStore fails every trusted-device lookup.
type failingTrustStore struct{ *mfa.MemoryStore }

func (failingTrustStore) GetTrustedDeviceByTokenHash(context.Context, string) (*mfa.TrustedDevice, error) {
	return nil, errors.New("database unavailable")
}

// A lookup that fails does not skip the second factor.
func TestIssueSession_TrustedDeviceFailsClosed(t *testing.T) {
	t.Parallel()
	eng, u, appID := issueSessionFixture(t)
	plg, ok := eng.Plugins().Plugin("mfa").(*mfa.Plugin)
	require.True(t, ok)
	inner := mfa.NewMemoryStore()
	plg.SetStore(failingTrustStore{inner})
	require.NoError(t, inner.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID: id.NewMFAID(), UserID: u.ID, Method: "totp", Secret: "JBSWY3DPEHPK3PXP", Verified: true,
	}))
	token := trustDevice(t, inner, u.ID, time.Now(), time.Now().Add(time.Hour))

	_, err := eng.IssueSession(context.Background(), &authsome.IssueSessionRequest{
		User: u, AppID: appID, AuthMethod: "password", TrustedDeviceToken: token,
	})
	var mfaErr *authsome.MFARequiredError
	assert.True(t, errors.As(err, &mfaErr), "the gate fires when the device cannot be checked, got %v", err)
}
