package phone_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
)

const (
	testAppIDStr = "aapp_01jf0000000000000000000000"
	testPhone    = "+14155551234"
)

// mockSMS captures outbound messages so tests can read the OTP the user would
// have received.
type mockSMS struct {
	mu   sync.Mutex
	sent []string
}

func (m *mockSMS) SendSMS(_ context.Context, msg *bridge.SMSMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg.Body)
	return nil
}

// ttlSpy wraps a ceremony store and records the TTL of every Set, so a test
// can assert that a failed attempt does not extend a challenge's deadline.
type ttlSpy struct {
	ceremony.Store
	mu   sync.Mutex
	ttls []time.Duration
}

func (s *ttlSpy) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	s.mu.Lock()
	s.ttls = append(s.ttls, ttl)
	s.mu.Unlock()
	return s.Store.Set(ctx, key, data, ttl)
}

func (s *ttlSpy) recorded() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]time.Duration(nil), s.ttls...)
}

func newTestPlugin(t *testing.T) (*phone.Plugin, *mockSMS, *ttlSpy) {
	t.Helper()
	sms := &mockSMS{}
	spy := &ttlSpy{Store: ceremony.NewMemory()}
	p := phone.New(phone.Config{
		SMSSender: sms,
		CodeTTL:   5 * time.Minute,
	})
	p.SetStore(memory.New())
	p.SetAppID(testAppIDStr)
	p.SetCeremonyStore(spy)
	return p, sms, spy
}

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// startChallenge runs /start and returns the OTP the user was sent.
func startChallenge(t *testing.T, mux forge.Router, sms *mockSMS) string {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/phone/start", jsonBody(t, map[string]string{"phone": testPhone}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "start: %s", rec.Body.String())

	sms.mu.Lock()
	defer sms.mu.Unlock()
	require.Len(t, sms.sent, 1)
	// Body is "Your verification code is: NNNNNN. It expires in N minutes."
	var code string
	for _, field := range bytes.Fields([]byte(sms.sent[0])) {
		trimmed := bytes.TrimRight(field, ".")
		if len(trimmed) == 6 && bytes.IndexFunc(trimmed, func(r rune) bool {
			return r < '0' || r > '9'
		}) == -1 {
			code = string(trimmed)
		}
	}
	require.NotEmpty(t, code, "could not extract OTP from %q", sms.sent[0])
	return code
}

func submitCode(t *testing.T, mux forge.Router, code string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), "POST",
		"/v1/phone/verify", jsonBody(t, map[string]string{"phone": testPhone, "code": code}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// wrongCode returns a 6-digit code guaranteed to differ from the real one.
func wrongCode(actual string) string {
	if actual == "000000" {
		return "111111"
	}
	return "000000"
}

// A 6-digit code over a 5-minute window is only 10^6 wide. Without a cap on
// attempts an attacker can walk the whole space against a single challenge,
// and route rate limiting can be disabled by config — so the challenge itself
// must stop accepting guesses.
func TestVerify_ExhaustingAttemptsKillsChallenge(t *testing.T) {
	p, sms, _ := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	code := startChallenge(t, mux, sms)

	for i := 0; i < 5; i++ {
		rec := submitCode(t, mux, wrongCode(code))
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "wrong guess %d", i+1)
	}

	// The correct code must now be worthless — the challenge is gone.
	rec := submitCode(t, mux, code)
	assert.Equal(t, http.StatusUnauthorized, rec.Code,
		"challenge must be discarded once the attempt cap is reached")
	assert.NotContains(t, rec.Body.String(), "session_token",
		"no session may be issued after the attempt cap")
}

// The cap must not be so eager that a user who fat-fingers a digit is locked
// out of a code they still hold.
func TestVerify_CorrectCodeStillWorksBelowCap(t *testing.T) {
	p, sms, _ := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	code := startChallenge(t, mux, sms)

	for i := 0; i < 4; i++ {
		require.Equal(t, http.StatusUnauthorized, submitCode(t, mux, wrongCode(code)).Code)
	}

	rec := submitCode(t, mux, code)
	assert.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "session_token")
}

// Re-storing the challenge on a failed attempt must preserve its original
// deadline. Writing back a full CodeTTL would let an attacker hold a challenge
// open indefinitely by guessing — turning the attempt counter into a way to
// extend the very window it exists to bound.
func TestVerify_FailedAttemptDoesNotExtendDeadline(t *testing.T) {
	p, sms, spy := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	code := startChallenge(t, mux, sms)
	initial := spy.recorded()
	require.Len(t, initial, 1, "start should write the challenge once")

	submitCode(t, mux, wrongCode(code))

	ttls := spy.recorded()
	require.Len(t, ttls, 2, "a failed attempt should rewrite the challenge")
	assert.Less(t, ttls[1], ttls[0],
		"rewritten TTL must be the remaining time, not a fresh full TTL")
	assert.Positive(t, ttls[1])
}

// A challenge for one phone number must not be consumable via another, and a
// verify with no prior start must not authenticate.
func TestVerify_WithoutChallengeIsRejected(t *testing.T) {
	p, _, _ := newTestPlugin(t)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	rec := submitCode(t, mux, "123456")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.NotContains(t, rec.Body.String(), "session_token")
}

// newTestPluginWithStore is newTestPlugin plus a handle on the backing store,
// for tests that need to inspect the rows the flow wrote.
func newTestPluginWithStore(t *testing.T) (*phone.Plugin, *mockSMS, store.Store) {
	t.Helper()
	sms := &mockSMS{}
	st := memory.New()
	p := phone.New(phone.Config{SMSSender: sms, CodeTTL: 5 * time.Minute})
	p.SetStore(st)
	p.SetAppID(testAppIDStr)
	p.SetCeremonyStore(ceremony.NewMemory())
	return p, sms, st
}

// seedAppWithDefaultEnv gives the store the app and default environment the
// phone flow is expected to resolve against.
func seedAppWithDefaultEnv(t *testing.T, st store.Store) *environment.Environment {
	t.Helper()
	ctx := context.Background()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	require.NoError(t, st.CreateApp(ctx, &app.App{
		ID: appID, Name: "Phone App", Slug: "phone-app",
		PublishableKey: "pk_test_phone",
		CreatedAt:      time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}))
	env := &environment.Environment{
		ID: id.NewEnvironmentID(), AppID: appID,
		Name: "Production", Slug: "production",
		Type: environment.TypeProduction, IsDefault: true,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, st.CreateEnvironment(ctx, env))
	return env
}

// A user minted by the phone flow must carry the app's default environment.
// Without it the account lands with a nil EnvID, which both breaks the
// env-scoped phone lookup on the user's next sign-in and leaves a row that no
// environment owns.
func TestVerify_NewUserGetsDefaultEnvironment(t *testing.T) {
	p, sms, st := newTestPluginWithStore(t)
	env := seedAppWithDefaultEnv(t, st)

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	code := startChallenge(t, mux, sms)
	rec := submitCode(t, mux, code)
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	u, err := st.GetUserByPhone(context.Background(), appID, env.ID, testPhone)
	require.NoError(t, err, "phone user must be findable in the default environment")
	assert.Equal(t, env.ID.String(), u.EnvID.String(),
		"a phone signup must be stamped with the app's default environment")
}

// A second sign-in with the same number must re-find the existing account
// rather than minting a duplicate. This is what regresses if the flow writes
// users into one environment but looks them up in another.
func TestVerify_ReturningUserIsNotDuplicated(t *testing.T) {
	p, sms, st := newTestPluginWithStore(t)
	env := seedAppWithDefaultEnv(t, st)

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	require.Equal(t, http.StatusOK, submitCode(t, mux, startChallenge(t, mux, sms)).Code)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	first, err := st.GetUserByPhone(context.Background(), appID, env.ID, testPhone)
	require.NoError(t, err)

	sms.mu.Lock()
	sms.sent = nil
	sms.mu.Unlock()

	require.Equal(t, http.StatusOK, submitCode(t, mux, startChallenge(t, mux, sms)).Code)
	list, err := st.ListUsers(context.Background(), &user.Query{AppID: appID, Limit: 50})
	require.NoError(t, err)
	assert.Len(t, list.Users, 1, "returning phone user must not be duplicated")
	assert.Equal(t, first.ID.String(), list.Users[0].ID.String())
}
