package passkey

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/user"
)

func seedPasskeyUser(t *testing.T, eng *authsome.Engine) *user.User {
	t.Helper()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	u := &user.User{
		ID:        id.NewUserID(),
		AppID:     appID,
		Email:     "passwordless@test.com",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, eng.Store().CreateUser(context.Background(), u))
	return u
}

// TestResolveDiscoverableUser is the core of passwordless login: the WebAuthn
// user handle (the authsome user id) must map back to a real user, WITHOUT any
// pre-existing authenticated session. A bad handle or unknown user errors.
func TestResolveDiscoverableUser(t *testing.T) {
	p := New()
	eng := secutil.NewTestEngine(t, authsome.WithPlugin(p))
	u := seedPasskeyUser(t, eng)
	ctx := context.Background()

	gotUser, wau, err := p.resolveDiscoverableUser(ctx, []byte(u.ID.String()))
	require.NoError(t, err)
	require.NotNil(t, wau)
	assert.Equal(t, u.ID.String(), gotUser.ID.String())
	assert.Equal(t, []byte(u.ID.String()), wau.WebAuthnID(), "the webauthn user handle must round-trip")

	_, _, err = p.resolveDiscoverableUser(ctx, []byte("not-a-valid-user-id"))
	assert.Error(t, err, "an unparseable user handle must error")

	_, _, err = p.resolveDiscoverableUser(ctx, []byte(id.NewUserID().String()))
	assert.Error(t, err, "an unknown user must error")
}

// TestCeremonyCookieRoundTrip pins that the discoverable ceremony is correlated
// via a cookie set at begin and read at finish, and that each ceremony gets a
// unique id (so concurrent passwordless logins don't clobber a shared key).
func TestCeremonyCookieRoundTrip(t *testing.T) {
	id1, err := newCeremonyID()
	require.NoError(t, err)
	id2, err := newCeremonyID()
	require.NoError(t, err)
	assert.NotEmpty(t, id1)
	assert.NotEqual(t, id1, id2, "each ceremony must get a distinct id")

	rec := httptest.NewRecorder()
	setCeremonyCookie(rec, id1, time.Minute)

	// The cookie the server set becomes the request cookie on the follow-up call.
	req := httptest.NewRequest(http.MethodPost, "/v1/passkey/login/finish", nil)
	for _, c := range rec.Result().Cookies() {
		req.AddCookie(c)
	}
	assert.Equal(t, id1, readCeremonyCookie(req), "the ceremony id must round-trip via cookie")

	// A request without the cookie yields an empty id (identified flow).
	bare := httptest.NewRequest(http.MethodPost, "/v1/passkey/login/finish", nil)
	assert.Empty(t, readCeremonyCookie(bare))
}

// TestDiscoverableKeyIsPerCeremony confirms the ceremony store key is derived
// from the ceremony id rather than a single global constant.
func TestDiscoverableKeyIsPerCeremony(t *testing.T) {
	assert.NotEqual(t, discoverableKey("a"), discoverableKey("b"))
	assert.Contains(t, discoverableKey("abc"), "abc")
}
