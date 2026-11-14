package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/api"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/secutil"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/store/memory"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

const trustedPassword = "SecureP@ss123"

// trustFixture is a real engine with the mfa plugin mounted behind the auth
// middleware, so the trusted-device routes see the signed-in user.
type trustFixture struct {
	eng      *authsome.Engine
	handler  http.Handler
	mfaStore *mfa.MemoryStore
}

// trustedUser is an enrolled user plus the recovery codes used to pass the
// challenge. Recovery codes rather than TOTP so several challenges in one
// test don't trip the TOTP replay guard.
type trustedUser struct {
	email    string
	id       id.UserID
	recovery []string
}

func newTrustFixture(t *testing.T, appWide bool) *trustFixture {
	t.Helper()
	mfaStore := mfa.NewMemoryStore()
	mfaPlugin := mfa.New()
	mfaPlugin.SetStore(mfaStore)

	s := memory.New()
	seedTestPlatformApp(t, s)
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	eng, err := authsome.NewEngine(authsome.WithConfig(testConfig()), authsome.WithChronicle(bridge.NewMemoryChronicle()),
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID(testAppIDStr),
		authsome.WithPlugin(mfaPlugin),
	)
	require.NoError(t, err)
	require.NoError(t, eng.Start(context.Background()))
	secutil.RelaxAuthDefaults(t, eng)

	if appWide {
		appID, err := id.ParseAppID(testAppIDStr)
		require.NoError(t, err)
		tru := true
		require.NoError(t, eng.Store().SetAppClientConfig(context.Background(), &appclientconfig.Config{
			ID: id.NewAppClientConfigID(), AppID: appID, MFARequired: &tru,
		}))
	}

	root := forge.NewRouter()
	root.UseGlobal(eng.AuthMiddleware())
	a := api.New(eng, root)
	require.NoError(t, a.RegisterRoutes(root))
	require.NoError(t, mfaPlugin.RegisterRoutes(root))
	return &trustFixture{eng: eng, handler: withTestKey(root.Handler()), mfaStore: mfaStore}
}

func (f *trustFixture) enrolledUser(t *testing.T, email string) *trustedUser {
	t.Helper()
	ctx := context.Background()
	_, tok, _ := signUp(t, f.eng, email, trustedPassword)
	require.NotEmpty(t, tok)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	u, err := f.eng.Store().GetUserByEmail(ctx, appID, id.Nil, email)
	require.NoError(t, err)

	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "TestApp", AccountName: email})
	require.NoError(t, err)
	require.NoError(t, f.mfaStore.CreateEnrollment(ctx, &mfa.Enrollment{
		ID: id.NewMFAID(), UserID: u.ID, Method: "totp", Secret: key.Secret(), Verified: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	codes, plain, err := mfa.GenerateRecoveryCodes(u.ID, 8)
	require.NoError(t, err)
	require.NoError(t, f.mfaStore.CreateRecoveryCodes(ctx, codes))
	return &trustedUser{email: email, id: u.ID, recovery: plain}
}

func (u *trustedUser) nextCode() string {
	c := u.recovery[0]
	u.recovery = u.recovery[1:]
	return c
}

func (f *trustFixture) do(t *testing.T, method, path string, body any, prep func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
	if prep != nil {
		prep(req)
	}
	rec := httptest.NewRecorder()
	f.handler.ServeHTTP(rec, req)
	return rec
}

func (f *trustFixture) signIn(t *testing.T, email string, prep func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	return f.do(t, http.MethodPost, "/v1/signin", map[string]string{"email": email, "password": trustedPassword}, prep)
}

func withTrustHeader(token string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set(authsome.TrustedDeviceHeader, token) }
}

func withBearer(token string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), "body=%s", rec.Body.String())
	return out
}

// challengeAndTrust drives sign-in to the gate, passes it with trust_device,
// and returns the challenge response.
func (f *trustFixture) challengeAndTrust(t *testing.T, u *trustedUser, trust bool) *httptest.ResponseRecorder {
	t.Helper()
	rec := f.signIn(t, u.email, nil)
	require.Equal(t, http.StatusForbidden, rec.Code, "sign-in must hit the gate; body=%s", rec.Body.String())
	ticket, _ := decodeMap(t, rec)["mfa_ticket"].(string)
	require.NotEmpty(t, ticket)

	rec = f.do(t, http.MethodPost, "/v1/mfa/challenge",
		map[string]any{"mfa_ticket": ticket, "code": u.nextCode(), "trust_device": trust}, nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	return rec
}

func trustedCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == authsome.TrustedDeviceCookieName {
			return c
		}
	}
	return nil
}

func trustToken(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	tok, _ := decodeMap(t, rec)["trusted_device_token"].(string)
	require.NotEmpty(t, tok, "the challenge must return a trusted device token; body=%s", rec.Body.String())
	return tok
}

// Passing the challenge with trust_device lets the next password sign-in
// from that device skip the second factor, whether the app requires MFA of
// everyone or only this user turned it on, and the session it gets counts
// as verified.
func TestTrustedDevice_HeaderSkipsGate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		appWide bool
	}{{"per-user", false}, {"app-wide", true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			f := newTrustFixture(t, tc.appWide)
			u := f.enrolledUser(t, "trust-header-"+tc.name+"@example.com")

			rec := f.challengeAndTrust(t, u, true)
			body := decodeMap(t, rec)
			token := trustToken(t, rec)
			expStr, _ := body["trusted_device_expires_at"].(string)
			exp, err := time.Parse(time.RFC3339, expStr)
			require.NoError(t, err)
			assert.WithinDuration(t, time.Now().Add(mfa.DefaultTrustedDeviceTTL), exp, time.Minute)

			c := trustedCookie(rec)
			require.NotNil(t, c, "a browser gets the token as a cookie")
			assert.Equal(t, token, c.Value)
			assert.True(t, c.HttpOnly)
			assert.Equal(t, "/", c.Path)
			assert.Equal(t, http.SameSiteLaxMode, c.SameSite)
			assert.False(t, c.Secure, "plain-HTTP test request: Secure follows the session cookie")

			rec = f.signIn(t, u.email, withTrustHeader(token))
			require.Equal(t, http.StatusOK, rec.Code, "a trusted device skips the gate; body=%s", rec.Body.String())
			sessTok, _ := decodeMap(t, rec)["session_token"].(string)
			require.NotEmpty(t, sessTok)

			sess, err := f.eng.ResolveSessionByToken(context.Background(), sessTok)
			require.NoError(t, err)
			assert.False(t, sess.MFAVerifiedAt.IsZero(), "the session passed a second factor through the device")
			assert.False(t, sess.MFAVerifiedAt.After(time.Now()))

			devices, err := f.mfaStore.ListTrustedDevices(context.Background(), u.id)
			require.NoError(t, err)
			require.Len(t, devices, 1)
			assert.False(t, devices[0].LastUsedAt.Before(devices[0].CreatedAt), "using the device records it")
			assert.Equal(t, "Chrome on macOS", devices[0].Label)
		})
	}
}

// A browser that only holds the cookie is recognised the same way.
func TestTrustedDevice_CookieSkipsGate(t *testing.T) {
	t.Parallel()
	f := newTrustFixture(t, false)
	u := f.enrolledUser(t, "trust-cookie@example.com")

	c := trustedCookie(f.challengeAndTrust(t, u, true))
	require.NotNil(t, c)

	rec := f.signIn(t, u.email, func(r *http.Request) { r.AddCookie(&http.Cookie{Name: c.Name, Value: c.Value}) })
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.NotEmpty(t, decodeMap(t, rec)["session_token"])
}

// Without trust_device nothing is trusted: no token, no cookie, and the next
// sign-in still asks.
func TestTrustedDevice_OnlyWhenAsked(t *testing.T) {
	t.Parallel()
	f := newTrustFixture(t, false)
	u := f.enrolledUser(t, "trust-not-asked@example.com")

	rec := f.challengeAndTrust(t, u, false)
	assert.NotContains(t, decodeMap(t, rec), "trusted_device_token")
	assert.Nil(t, trustedCookie(rec))
	devices, err := f.mfaStore.ListTrustedDevices(context.Background(), u.id)
	require.NoError(t, err)
	assert.Empty(t, devices)
}

// Tokens that are not a live device of the user signing in change nothing.
func TestTrustedDevice_RejectedTokens(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := newTrustFixture(t, true)
	alice := f.enrolledUser(t, "trust-alice@example.com")
	bob := f.enrolledUser(t, "trust-bob@example.com")
	aliceToken := trustToken(t, f.challengeAndTrust(t, alice, true))

	t.Run("unknown token", func(t *testing.T) {
		rec := f.signIn(t, alice.email, withTrustHeader("not-a-real-token"))
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	})

	t.Run("another user's token", func(t *testing.T) {
		rec := f.signIn(t, bob.email, withTrustHeader(aliceToken))
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	})

	t.Run("expired", func(t *testing.T) {
		token := trustToken(t, f.challengeAndTrust(t, bob, true))
		devices, err := f.mfaStore.ListTrustedDevices(ctx, bob.id)
		require.NoError(t, err)
		require.Len(t, devices, 1)
		d := devices[0]
		require.NoError(t, f.mfaStore.DeleteTrustedDevice(ctx, bob.id, d.ID))
		d.ExpiresAt = time.Now().Add(-time.Minute)
		require.NoError(t, f.mfaStore.CreateTrustedDevice(ctx, d))

		rec := f.signIn(t, bob.email, withTrustHeader(token))
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	})

	t.Run("removed", func(t *testing.T) {
		rec := f.signIn(t, alice.email, withTrustHeader(aliceToken))
		require.Equal(t, http.StatusOK, rec.Code, "still trusted before removal; body=%s", rec.Body.String())
		sessTok, _ := decodeMap(t, rec)["session_token"].(string)

		rec = f.do(t, http.MethodDelete, "/v1/mfa/trusted-devices", nil, withBearer(sessTok))
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

		rec = f.signIn(t, alice.email, withTrustHeader(aliceToken))
		assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	})
}

// The settings page lists the caller's devices, marks the one asking, and
// removes one or all of them.
func TestTrustedDevice_ListAndRemove(t *testing.T) {
	t.Parallel()
	f := newTrustFixture(t, false)
	u := f.enrolledUser(t, "trust-manage@example.com")
	other := f.enrolledUser(t, "trust-manage-other@example.com")

	first := trustToken(t, f.challengeAndTrust(t, u, true))
	rec := f.challengeAndTrust(t, u, true)
	second := trustToken(t, rec)
	sessTok, _ := decodeMap(t, rec)["session_token"].(string)
	otherTok := trustToken(t, f.challengeAndTrust(t, other, true))
	require.NotEmpty(t, otherTok)

	asSecond := func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+sessTok)
		r.Header.Set(authsome.TrustedDeviceHeader, second)
	}

	rec = f.do(t, http.MethodGet, "/v1/mfa/trusted-devices", nil, asSecond)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var list mfa.TrustedDevicesResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list.Devices, 2, "only the caller's devices")
	current := 0
	var firstID string
	for _, d := range list.Devices {
		assert.Equal(t, "Chrome on macOS", d.Label)
		assert.NotEmpty(t, d.UserAgent)
		assert.False(t, d.ExpiresAt.IsZero())
		if d.Current {
			current++
		} else {
			firstID = d.ID
		}
	}
	assert.Equal(t, 1, current, "exactly the requesting device is current")
	assert.NotContains(t, rec.Body.String(), first, "tokens are never listed")

	rec = f.do(t, http.MethodGet, "/v1/mfa/enrollment", nil, withBearer(sessTok))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.EqualValues(t, 2, decodeMap(t, rec)["trusted_devices"])

	// Another user's device id answers as not found.
	otherDevices, err := f.mfaStore.ListTrustedDevices(context.Background(), other.id)
	require.NoError(t, err)
	require.Len(t, otherDevices, 1)
	rec = f.do(t, http.MethodDelete, "/v1/mfa/trusted-devices/"+otherDevices[0].ID.String(), nil, withBearer(sessTok))
	assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())
	rec = f.do(t, http.MethodDelete, "/v1/mfa/trusted-devices/not-an-id", nil, withBearer(sessTok))
	assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())

	rec = f.do(t, http.MethodDelete, "/v1/mfa/trusted-devices/"+firstID, nil, withBearer(sessTok))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.EqualValues(t, 1, decodeMap(t, rec)["removed"])
	assert.Equal(t, http.StatusForbidden, f.signIn(t, u.email, withTrustHeader(first)).Code, "the removed device asks again")
	assert.Equal(t, http.StatusOK, f.signIn(t, u.email, withTrustHeader(second)).Code, "the other device is untouched")

	rec = f.do(t, http.MethodDelete, "/v1/mfa/trusted-devices", nil, asSecond)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.EqualValues(t, 1, decodeMap(t, rec)["removed"])
	if c := trustedCookie(rec); assert.NotNil(t, c, "removing the current device clears its cookie") {
		assert.Less(t, c.MaxAge, 0)
	}
	assert.Equal(t, http.StatusOK, f.signIn(t, other.email, withTrustHeader(otherTok)).Code, "another user's devices stay")

	rec = f.do(t, http.MethodGet, "/v1/mfa/trusted-devices", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body=%s", rec.Body.String())
}

// Turning MFA off forgets every trusted device, so turning it back on later
// does not quietly resurrect them.
func TestTrustedDevice_DisableMFAClears(t *testing.T) {
	t.Parallel()
	f := newTrustFixture(t, false)
	u := f.enrolledUser(t, "trust-disable@example.com")
	rec := f.challengeAndTrust(t, u, true)
	sessTok, _ := decodeMap(t, rec)["session_token"].(string)

	rec = f.do(t, http.MethodDelete, "/v1/mfa/enrollment", map[string]string{"code": u.nextCode()}, withBearer(sessTok))
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	devices, err := f.mfaStore.ListTrustedDevices(context.Background(), u.id)
	require.NoError(t, err)
	assert.Empty(t, devices)
}

// A password change or reset forgets them too.
func TestTrustedDevice_PasswordChangeClears(t *testing.T) {
	t.Parallel()
	f := newTrustFixture(t, false)
	u := f.enrolledUser(t, "trust-password@example.com")
	trustToken(t, f.challengeAndTrust(t, u, true))

	require.NoError(t, f.eng.ChangePassword(context.Background(), u.id, trustedPassword, "N3wSecureP@ss456!"))

	devices, err := f.mfaStore.ListTrustedDevices(context.Background(), u.id)
	require.NoError(t, err)
	assert.Empty(t, devices)
}
