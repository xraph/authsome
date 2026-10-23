package mfa_test

import (
	"bytes"
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

// An agent session is minted fresh and carries its delegating user's UserID,
// so it passes the recent-sign-in step-up and the handlers read it as the
// user. Enrolling a factor, regenerating recovery codes or turning MFA off
// would hand the agent (or strip) a credential on the user's account that no
// grant bounds, so those routes refuse it.

func doAsAgent(t *testing.T, p *mfa.Plugin, method, path string, body *bytes.Buffer, userID id.UserID) *httptest.ResponseRecorder {
	t.Helper()
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))
	if body == nil {
		body = &bytes.Buffer{}
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, body)
	req.Header.Set("Content-Type", "application/json")
	sess := &session.Session{
		ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: userID, CreatedAt: time.Now(),
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	}
	ctx := middleware.WithSession(req.Context(), sess)
	ctx = middleware.WithUserID(ctx, userID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req.WithContext(ctx))
	return rec
}

func seedVerifiedTOTP(t *testing.T, store *mfa.MemoryStore, userID id.UserID) string {
	t.Helper()
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "TestApp", AccountName: "u@example.com"})
	require.NoError(t, err)
	require.NoError(t, store.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID: id.NewMFAID(), UserID: userID, Method: "totp", Secret: key.Secret(), Verified: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	return key.Secret()
}

func TestHandleEnroll_RefusesAgentSession(t *testing.T) {
	p, store := newTestPlugin(t)
	userID := id.NewUserID()

	rec := doAsAgent(t, p, http.MethodPost, "/v1/mfa/enroll", jsonBody(t, map[string]string{"method": "totp"}), userID)

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "secret")
	enrollments, err := store.ListEnrollments(context.Background(), userID)
	require.NoError(t, err)
	assert.Empty(t, enrollments, "no factor is enrolled")

	// The user's own fresh session still enrolls.
	own := enrollWithSession(t, p, userID, map[string]string{"method": "totp"}, time.Now(), true)
	assert.Less(t, own.Code, 300, "body=%s", own.Body.String())
}

func TestHandleRecoveryRegenerate_RefusesAgentSession(t *testing.T) {
	p, store := newTestPlugin(t)
	userID := id.NewUserID()
	seedVerifiedTOTP(t, store, userID)

	rec := doAsAgent(t, p, http.MethodPost, "/v1/mfa/recovery/regenerate", nil, userID)

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	codes, err := store.GetRecoveryCodes(context.Background(), userID)
	require.NoError(t, err)
	assert.Empty(t, codes, "no recovery codes are minted")
}

func TestHandleDisable_RefusesAgentSession(t *testing.T) {
	p, store := newTestPlugin(t)
	userID := id.NewUserID()
	secret := seedVerifiedTOTP(t, store, userID)
	code, err := mfa.GenerateTOTPCode(secret)
	require.NoError(t, err)

	rec := doAsAgent(t, p, http.MethodDelete, "/v1/mfa/enrollment", jsonBody(t, map[string]string{"code": code}), userID)

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	_, err = store.GetEnrollment(context.Background(), userID, "totp")
	assert.NoError(t, err, "the factor stays enrolled")
}
