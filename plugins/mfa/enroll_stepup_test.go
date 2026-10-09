package mfa_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/mfa"
	"github.com/xraph/authsome/session"
)

// Enrolling a factor changes how the account is protected, so it needs more
// than a session that may have been stolen: a recent sign-in for the first
// factor, and the existing factor for every later one.

func enrollWithSession(t *testing.T, p *mfa.Plugin, userID id.UserID, body map[string]string, signedInAt time.Time, withSession bool) *httptest.ResponseRecorder {
	t.Helper()
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/mfa/enroll", jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	ctx := middleware.WithUserID(req.Context(), userID)
	if withSession {
		ctx = middleware.WithSession(ctx, &session.Session{ID: id.NewSessionID(), UserID: userID, CreatedAt: signedInAt})
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req.WithContext(ctx))
	return rec
}

func TestHandleEnroll_StaleSessionIsRefused(t *testing.T) {
	p, _ := newTestPlugin(t)
	rec := enrollWithSession(t, p, id.NewUserID(), map[string]string{"method": "totp"}, time.Now().Add(-time.Hour), true)
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
}

func TestHandleEnroll_NoSessionIsRefused(t *testing.T) {
	p, _ := newTestPlugin(t)
	rec := enrollWithSession(t, p, id.NewUserID(), map[string]string{"method": "totp"}, time.Time{}, false)
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
}

func TestHandleEnroll_ExistingFactorMustBeProven(t *testing.T) {
	p, store := newTestPlugin(t)
	userID := id.NewUserID()
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "TestApp", AccountName: "u@example.com"})
	require.NoError(t, err)
	require.NoError(t, store.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID: id.NewMFAID(), UserID: userID, Method: "totp", Secret: key.Secret(), Verified: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// A fresh session is not enough once a factor exists.
	rec := enrollWithSession(t, p, userID, map[string]string{"method": "sms", "phone": "+15550000000"}, time.Now(), true)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "body=%s", rec.Body.String())

	rec = enrollWithSession(t, p, userID, map[string]string{"method": "sms", "phone": "+15550000000", "code": "000000"}, time.Now(), true)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "a wrong code is refused; body=%s", rec.Body.String())

	code, err := mfa.GenerateTOTPCode(key.Secret())
	require.NoError(t, err)
	rec = enrollWithSession(t, p, userID, map[string]string{"method": "sms", "phone": "+15550000000", "code": code}, time.Now().Add(-time.Hour), true)
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code, "the current factor opens the gate even on an old session; body=%s", rec.Body.String())
	assert.NotEqual(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
}
