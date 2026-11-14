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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
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
	return sendAs(t, h, http.MethodPost, path, body, uid)
}

func sendAs(t *testing.T, h http.Handler, method, path string, body any, uid id.UserID) (*httptest.ResponseRecorder, id.SessionID) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewReader(raw))
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

	rec, _ := postAs(t, h, "/v1/mfa/recovery/verify", map[string]string{"code": "wrong"}, uid)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Zero(t, r.calls, "a wrong code stamps nothing")

	rec, sid := postAs(t, h, "/v1/mfa/recovery/verify", map[string]string{"code": plain[0]}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, 1, r.calls)
	assert.Equal(t, sid, r.sid)
}

func TestSMSVerify_StampsTheSession(t *testing.T) {
	p, s, r, h := stampingPlugin(t)
	uid := id.NewUserID()
	smsEnrollment(t, s, uid, true)
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

type recordingSMS struct{ to []string }

func (r *recordingSMS) SendSMS(_ context.Context, msg *bridge.SMSMessage) error {
	r.to = append(r.to, msg.To)
	return nil
}

// smsPhone is the number every SMS enrollment in these tests is made with.
const smsPhone = "+15550001"

func smsEnrollment(t *testing.T, s *MemoryStore, uid id.UserID, verified bool) {
	t.Helper()
	require.NoError(t, s.CreateEnrollment(context.Background(), &Enrollment{
		ID: id.NewMFAID(), UserID: uid, Method: "sms", Secret: smsPhone, Verified: verified,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
}

func pendingSMSCode(t *testing.T, p *Plugin, uid id.UserID) string {
	t.Helper()
	raw, err := p.ceremonies.Get(context.Background(), "mfa:sms:"+uid.String())
	require.NoError(t, err)
	var c SMSChallenge
	require.NoError(t, json.Unmarshal(raw, &c))
	return c.Code
}

// Once the SMS factor is verified, a code always goes to the enrolled
// number. A phone override would let a session that has only the password
// send itself a code and pass the second factor.
func TestSMSSend_VerifiedEnrollmentIgnoresThePhoneOverride(t *testing.T) {
	p, s, r, h := stampingPlugin(t)
	sms := &recordingSMS{}
	p.SetSMSSender(sms)
	uid := id.NewUserID()
	smsEnrollment(t, s, uid, true)

	rec, _ := postAs(t, h, "/v1/mfa/sms/send", map[string]string{"phone": "+15559999"}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []string{"+15550001"}, sms.to, "the code went to the enrolled number only")

	// And the code that went there is what stamps the session.
	rec, sid := postAs(t, h, "/v1/mfa/sms/verify", map[string]string{"code": pendingSMSCode(t, p, uid)}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, 1, r.calls)
	assert.Equal(t, sid, r.sid)
}

// Before the SMS factor is verified, the override is the enrollment flow:
// it moves the pending enrollment to the new number, under the same step-up
// as enrolling. Confirming it verifies the enrollment but does not stamp
// the session: the session has not passed a factor it already held.
func TestSMSSend_UnverifiedEnrollmentOverrideConfirmsWithoutStamping(t *testing.T) {
	p, s, r, h := stampingPlugin(t)
	sms := &recordingSMS{}
	p.SetSMSSender(sms)
	uid := id.NewUserID()
	smsEnrollment(t, s, uid, false)

	rec, _ := postAs(t, h, "/v1/mfa/sms/send", map[string]string{"phone": "+15550002"}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []string{"+15550002"}, sms.to)

	rec, _ = postAs(t, h, "/v1/mfa/sms/verify", map[string]string{"code": pendingSMSCode(t, p, uid)}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Zero(t, r.calls, "confirming an enrollment does not stamp the session")
	e, err := s.GetEnrollment(context.Background(), uid, "sms")
	require.NoError(t, err)
	assert.True(t, e.Verified)
	assert.Equal(t, "+15550002", e.Secret, "the number confirmed is the one the code went to")
}

// A user who already holds a verified TOTP factor must prove it before a
// pending SMS enrollment is pointed at a new number, as enrolling does.
func TestSMSSend_UnverifiedOverrideNeedsTheExistingFactor(t *testing.T) {
	p, s, _, h := stampingPlugin(t)
	sms := &recordingSMS{}
	p.SetSMSSender(sms)
	uid := id.NewUserID()
	require.NoError(t, s.CreateEnrollment(context.Background(), &Enrollment{
		ID: id.NewMFAID(), UserID: uid, Method: "totp", Secret: "JBSWY3DPEHPK3PXP", Verified: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	smsEnrollment(t, s, uid, false)

	rec, _ := postAs(t, h, "/v1/mfa/sms/send", map[string]string{"phone": "+15559999"}, uid)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	assert.Empty(t, sms.to, "nothing was sent to the new number")
	e, err := s.GetEnrollment(context.Background(), uid, "sms")
	require.NoError(t, err)
	assert.Equal(t, "+15550001", e.Secret)

	// Without an override it goes to the pending number, as before.
	rec, _ = postAs(t, h, "/v1/mfa/sms/send", map[string]string{}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, []string{"+15550001"}, sms.to)
}

// Removing the factor removes what the session's stamp stood for, so the
// current session is unstamped too (the others are already revoked).
func TestDisable_ClearsTheSessionStamp(t *testing.T) {
	_, s, r, h := stampingPlugin(t)
	uid := id.NewUserID()
	const secret = "JBSWY3DPEHPK3PXP"
	require.NoError(t, s.CreateEnrollment(context.Background(), &Enrollment{
		ID: id.NewMFAID(), UserID: uid, Method: "totp", Secret: secret, Verified: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	rec, _ := sendAs(t, h, http.MethodDelete, "/v1/mfa/enrollment", map[string]string{"code": "000000"}, uid)
	require.Equal(t, http.StatusUnauthorized, rec.Code, rec.Body.String())
	assert.Zero(t, r.calls, "a refused disable leaves the stamp alone")

	code, err := totp.GenerateCode(secret, time.Now())
	require.NoError(t, err)
	r.at = time.Now()
	rec, sid := sendAs(t, h, http.MethodDelete, "/v1/mfa/enrollment", map[string]string{"code": code}, uid)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, 1, r.calls)
	assert.Equal(t, sid, r.sid)
	assert.True(t, r.at.IsZero(), "the stamp is cleared, not renewed")
}
