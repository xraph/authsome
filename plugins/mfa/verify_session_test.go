package mfa

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/xraph/authsome/session"
	"github.com/xraph/forge"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

type recordingStamper struct {
	calls int
	sid   id.SessionID
	at    time.Time
}

func (r *recordingStamper) MarkSessionMFAVerified(_ context.Context, sid id.SessionID, at time.Time) error {
	r.calls++
	r.sid = sid
	r.at = at
	return nil
}

func TestMarkSessionVerified_StampsTheRequestSession(t *testing.T) {
	r := &recordingStamper{}
	p := &Plugin{stamper: r}
	sid := id.NewSessionID()

	require.NoError(t, p.markSessionVerified(middleware.WithSessionID(context.Background(), sid)))

	assert.Equal(t, 1, r.calls)
	assert.Equal(t, sid, r.sid)
	assert.WithinDuration(t, time.Now(), r.at, time.Minute)
}

func TestMarkSessionVerified_NoSessionOrNoEngineIsNoop(t *testing.T) {
	r := &recordingStamper{}
	require.NoError(t, (&Plugin{stamper: r}).markSessionVerified(context.Background()))
	assert.Zero(t, r.calls)
	require.NoError(t, (&Plugin{}).markSessionVerified(middleware.WithSessionID(context.Background(), id.NewSessionID())))
}

// Every proof of a second factor from a signed-in session stamps that session.

func stampingPlugin(t *testing.T) (*Plugin, *MemoryStore, *recordingStamper, http.Handler) {
	t.Helper()
	p := New(Config{Issuer: "T"})
	s := NewMemoryStore()
	p.SetStore(s)
	r := &recordingStamper{}
	p.stamper = r
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	return p, s, r, router.Handler()
}

func postAs(t *testing.T, h http.Handler, path string, body any, uid id.UserID) (*httptest.ResponseRecorder, id.SessionID) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	sess := &session.Session{ID: id.NewSessionID(), UserID: uid, CreatedAt: time.Now()}
	ctx := middleware.WithUserID(req.Context(), uid)
	ctx = middleware.WithSession(ctx, sess)
	ctx = middleware.WithSessionID(ctx, sess.ID)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req.WithContext(ctx))
	return rec, sess.ID
}

func TestRecoveryVerify_StampsTheSession(t *testing.T) {
	_, s, r, h := stampingPlugin(t)
	uid := id.NewUserID()
	require.NoError(t, s.CreateEnrollment(context.Background(), &Enrollment{
		ID: id.NewMFAID(), UserID: uid, Method: "totp", Secret: "JBSWY3DPEHPK3PXP", Verified: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	codes, plain, err := GenerateRecoveryCodes(uid, 2)
	require.NoError(t, err)
	require.NoError(t, s.CreateRecoveryCodes(context.Background(), codes))

	rec, sid := postAs(t, h, "/v1/mfa/recovery/verify", map[string]string{"code": "wrong"}, uid)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Zero(t, r.calls, "a wrong code stamps nothing")

	rec, sid = postAs(t, h, "/v1/mfa/recovery/verify", map[string]string{"code": plain[0]}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, 1, r.calls)
	assert.Equal(t, sid, r.sid)
}

func TestSMSVerify_StampsTheSession(t *testing.T) {
	p, _, r, h := stampingPlugin(t)
	uid := id.NewUserID()
	raw, err := json.Marshal(SMSChallenge{Code: "123456", ExpiresAt: time.Now().Add(time.Minute)})
	require.NoError(t, err)
	require.NoError(t, p.ceremonies.Set(context.Background(), "mfa:sms:"+uid.String(), raw, time.Minute))

	rec, _ := postAs(t, h, "/v1/mfa/sms/verify", map[string]string{"code": "000000"}, uid)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Zero(t, r.calls)

	rec, sid := postAs(t, h, "/v1/mfa/sms/verify", map[string]string{"code": "123456"}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, 1, r.calls)
	assert.Equal(t, sid, r.sid)
}

func TestVerify_StampsTheSession(t *testing.T) {
	_, s, r, h := stampingPlugin(t)
	uid := id.NewUserID()
	const secret = "JBSWY3DPEHPK3PXP"
	require.NoError(t, s.CreateEnrollment(context.Background(), &Enrollment{
		ID: id.NewMFAID(), UserID: uid, Method: "totp", Secret: secret, Verified: false,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)
	rec, sid := postAs(t, h, "/v1/mfa/verify", map[string]string{"code": code}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, 1, r.calls)
	assert.Equal(t, sid, r.sid)
}
