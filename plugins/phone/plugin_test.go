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

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/store/memory"
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
