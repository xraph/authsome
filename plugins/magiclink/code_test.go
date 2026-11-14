package magiclink_test

import (
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

	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/store/memory"
)

// codeMailer is a CodeMailer that records every message.
type codeMailer struct {
	mu    sync.Mutex
	msgs  []magiclink.MagicLinkMessage
	plain int
}

func (m *codeMailer) SendMagicLink(_ context.Context, _, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plain++
	return nil
}

func (m *codeMailer) SendMagicLinkWithCode(_ context.Context, msg magiclink.MagicLinkMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return nil
}

func (m *codeMailer) last(t *testing.T) magiclink.MagicLinkMessage {
	t.Helper()
	m.mu.Lock()
	defer m.mu.Unlock()
	require.NotEmpty(t, m.msgs, "no magic link email was sent")
	return m.msgs[len(m.msgs)-1]
}

func (m *codeMailer) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs)
}

type codeFixture struct {
	p      *magiclink.Plugin
	s      *memory.Store
	mailer *codeMailer
	h      http.Handler
}

func newCodeFixture(t *testing.T, ttl time.Duration) codeFixture {
	t.Helper()
	mailer := &codeMailer{}
	p := magiclink.New(magiclink.Config{
		Mailer:    mailer,
		EmailCode: true,
		TokenTTL:  ttl,
	})
	s := memory.New()
	p.SetStore(s)
	p.SetAppID(testAppIDStr)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	return codeFixture{p: p, s: s, mailer: mailer, h: mux.Handler()}
}

func post(t *testing.T, h http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out), "body=%s", rec.Body.String())
	return out
}

func wrongCodeFor(code string) string {
	if code == "000000" {
		return "111111"
	}
	return "000000"
}

func TestSendCode_MailCarriesLinkAndCode(t *testing.T) {
	f := newCodeFixture(t, 5*time.Minute)
	u := createTestUser(t, f.s)

	rec := post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	resp := decode(t, rec)
	assert.Equal(t, "magic link sent", resp["status"])
	assert.InDelta(t, 300, resp["expires_in"], 0)
	assert.InDelta(t, 6, resp["code_length"], 0)

	msg := f.mailer.last(t)
	assert.Equal(t, u.Email, msg.Email)
	assert.Len(t, msg.Token, 64)
	assert.Regexp(t, `^[0-9]{6}$`, msg.Code)
	assert.Equal(t, magiclink.PurposeSignIn, msg.Purpose)
	assert.Equal(t, "Test User", msg.Name)
	assert.Equal(t, testAppIDStr, msg.AppID)
	assert.WithinDuration(t, time.Now().Add(5*time.Minute), msg.ExpiresAt, 5*time.Second)
	assert.Zero(t, f.mailer.plain, "a CodeMailer is sent the full message, not the plain link")
}

// The response is decided before the address is looked up, so an unknown
// address gets exactly the body a registered one does.
func TestSendCode_UnknownEmailGetsSameResponse(t *testing.T) {
	f := newCodeFixture(t, 5*time.Minute)
	u := createTestUser(t, f.s)

	known := post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email})
	unknown := post(t, f.h, "/v1/magic-link/send", map[string]string{"email": "nobody@example.com"})
	require.Equal(t, http.StatusOK, known.Code)
	require.Equal(t, http.StatusOK, unknown.Code)
	assert.JSONEq(t, known.Body.String(), unknown.Body.String())
	assert.Equal(t, 1, f.mailer.count(), "nothing is sent to an unknown address")
}

// A mailer that can't carry a code gets the plain link, and the response
// doesn't promise a code that will never arrive.
func TestSendCode_PlainMailerGetsNoCode(t *testing.T) {
	mailer := &mockMailer{}
	p := magiclink.New(magiclink.Config{Mailer: mailer, EmailCode: true, TokenTTL: 5 * time.Minute})
	s := memory.New()
	p.SetStore(s)
	p.SetAppID(testAppIDStr)
	u := createTestUser(t, s)
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	rec := post(t, mux.Handler(), "/v1/magic-link/send", map[string]string{"email": u.Email})
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, decode(t, rec), "code_length")
	require.Len(t, mailer.sent, 1)
}

func TestVerifyCode_SignsIn(t *testing.T) {
	f := newCodeFixture(t, 5*time.Minute)
	u := createTestUser(t, f.s)

	require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
	msg := f.mailer.last(t)

	// The address is matched however it is cased.
	rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": "  TEST@example.com ", "code": msg.Code})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	resp := decode(t, rec)
	assert.NotEmpty(t, resp["session_token"])
	assert.NotEmpty(t, resp["refresh_token"])
	assert.NotNil(t, resp["user"])

	stored, err := f.s.GetUser(context.Background(), u.ID)
	require.NoError(t, err)
	assert.True(t, stored.EmailVerified, "the code proves the inbox")
}

func TestVerifyCode_MissingFields(t *testing.T) {
	f := newCodeFixture(t, 5*time.Minute)
	assert.Equal(t, http.StatusBadRequest, post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": "a@example.com"}).Code)
	assert.Equal(t, http.StatusBadRequest, post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"code": "123456"}).Code)
}

// Five tries per code, wrong or not. The fifth may still be the right one;
// after five wrong ones the code is gone, right or not.
func TestVerifyCode_AttemptsAreBounded(t *testing.T) {
	t.Run("right on the last try", func(t *testing.T) {
		f := newCodeFixture(t, 5*time.Minute)
		u := createTestUser(t, f.s)
		require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
		code := f.mailer.last(t).Code

		for i := 0; i < magiclink.MaxCodeAttempts-1; i++ {
			rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": wrongCodeFor(code)})
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		}
		rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": code})
		assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	})

	t.Run("dead after five wrong", func(t *testing.T) {
		f := newCodeFixture(t, 5*time.Minute)
		u := createTestUser(t, f.s)
		require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
		msg := f.mailer.last(t)

		for i := 0; i < magiclink.MaxCodeAttempts; i++ {
			rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": wrongCodeFor(msg.Code)})
			require.Equal(t, http.StatusUnauthorized, rec.Code)
		}
		rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": msg.Code})
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "the right code no longer works")
		assert.NotContains(t, rec.Body.String(), "session_token")

		// The link is a 256-bit secret and isn't charged for code guesses.
		link := post(t, f.h, "/v1/magic-link/verify", map[string]string{"token": msg.Token})
		assert.Equal(t, http.StatusOK, link.Code, "body=%s", link.Body.String())
	})
}

func TestVerifyCode_Expired(t *testing.T) {
	f := newCodeFixture(t, 150*time.Millisecond)
	u := createTestUser(t, f.s)
	require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
	msg := f.mailer.last(t)

	time.Sleep(300 * time.Millisecond)

	rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": msg.Code})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "invalid or expired code", decode(t, rec)["error"])
}

func TestVerifyCode_SingleUseAcrossLinkAndCode(t *testing.T) {
	t.Run("code first", func(t *testing.T) {
		f := newCodeFixture(t, 5*time.Minute)
		u := createTestUser(t, f.s)
		require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
		msg := f.mailer.last(t)

		require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": msg.Code}).Code)

		again := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": msg.Code})
		assert.Equal(t, http.StatusUnauthorized, again.Code, "a code works once")

		link := post(t, f.h, "/v1/magic-link/verify", map[string]string{"token": msg.Token})
		assert.Equal(t, http.StatusUnauthorized, link.Code, "the link went with the code; body=%s", link.Body.String())

		v, err := f.s.GetVerification(context.Background(), msg.Token)
		require.NoError(t, err)
		assert.True(t, v.Consumed, "the link's row is marked used as well")
	})

	t.Run("link first", func(t *testing.T) {
		f := newCodeFixture(t, 5*time.Minute)
		u := createTestUser(t, f.s)
		require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
		msg := f.mailer.last(t)

		require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/verify", map[string]string{"token": msg.Token}).Code)

		code := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": msg.Code})
		assert.Equal(t, http.StatusUnauthorized, code.Code, "the code went with the link")
	})

	t.Run("concurrent", func(t *testing.T) {
		f := newCodeFixture(t, 5*time.Minute)
		u := createTestUser(t, f.s)
		require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
		msg := f.mailer.last(t)

		reqs := []*http.Request{
			httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/magic-link/verify",
				jsonBody(t, map[string]string{"token": msg.Token})),
			httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/magic-link/verify-code",
				jsonBody(t, map[string]string{"email": u.Email, "code": msg.Code})),
		}
		var wg sync.WaitGroup
		codes := make([]int, len(reqs))
		for i, req := range reqs {
			req.Header.Set("Content-Type", "application/json")
			wg.Add(1)
			go func() {
				defer wg.Done()
				rec := httptest.NewRecorder()
				f.h.ServeHTTP(rec, req)
				codes[i] = rec.Code
			}()
		}
		wg.Wait()

		ok := 0
		for _, c := range codes {
			if c == http.StatusOK {
				ok++
			}
		}
		assert.Equal(t, 1, ok, "exactly one of link and code signs in; got %v", codes)
	})
}

// Only the newest code works: a resend retires the one before it.
func TestVerifyCode_ResendReplacesCode(t *testing.T) {
	f := newCodeFixture(t, 5*time.Minute)
	u := createTestUser(t, f.s)
	require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
	first := f.mailer.last(t)
	require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
	second := f.mailer.last(t)

	if first.Code != second.Code {
		rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": first.Code})
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	}
	rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": second.Code})
	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

// An unknown address, a wrong code and a never-sent code all get the same
// answer, so /verify-code can't be used to find accounts.
func TestVerifyCode_AnswersDoNotRevealAccounts(t *testing.T) {
	f := newCodeFixture(t, 5*time.Minute)
	u := createTestUser(t, f.s)

	noCodeYet := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": "123456"})
	unknown := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": "nobody@example.com", "code": "123456"})

	require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": u.Email}).Code)
	wrong := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": u.Email, "code": wrongCodeFor(f.mailer.last(t).Code)})

	for name, rec := range map[string]*httptest.ResponseRecorder{"no code yet": noCodeYet, "unknown": unknown, "wrong": wrong} {
		assert.Equal(t, http.StatusUnauthorized, rec.Code, name)
		assert.JSONEq(t, unknown.Body.String(), rec.Body.String(), name)
	}
}

// The code is bound to the account it was sent for.
func TestVerifyCode_OtherAccountsCodeRefused(t *testing.T) {
	f := newCodeFixture(t, 5*time.Minute)
	victim := createTestUser(t, f.s)
	require.Equal(t, http.StatusOK, post(t, f.h, "/v1/magic-link/send", map[string]string{"email": victim.Email}).Code)
	code := f.mailer.last(t).Code

	rec := post(t, f.h, "/v1/magic-link/verify-code", map[string]string{"email": "someone-else@example.com", "code": code})
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
