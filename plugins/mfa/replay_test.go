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
	mfa "github.com/xraph/authsome/plugins/mfa"
)

// TestHandleVerify_RejectsReplay pins TOTP replay protection: a code accepted
// once cannot be accepted again within its validity window. Without this an
// intercepted code can be replayed for the ~90s it stays valid.
func TestHandleVerify_RejectsReplay(t *testing.T) {
	p, s := newTestPlugin(t)

	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	userID := id.NewUserID()
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "TestApp", AccountName: "replay@test.com"})
	require.NoError(t, err)
	require.NoError(t, s.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    "totp",
		Secret:    key.Secret(),
		Verified:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}))

	code, err := mfa.GenerateTOTPCode(key.Secret())
	require.NoError(t, err)

	// First use succeeds.
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, authedRequest(t, "POST", "/v1/mfa/verify", jsonBody(t, map[string]string{"code": code}), userID))
	require.Equal(t, http.StatusOK, rec1.Code, "first use of a valid code should succeed; body=%s", rec1.Body.String())

	// Replay of the same code is rejected.
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, authedRequest(t, "POST", "/v1/mfa/verify", jsonBody(t, map[string]string{"code": code}), userID))
	assert.Equal(t, http.StatusUnauthorized, rec2.Code, "a replayed TOTP code must be rejected; body=%s", rec2.Body.String())
}

// TestValidateTOTPStep_ReturnsStep confirms the step accessor returns a valid,
// nonzero step for a good code and reports failure for a bad one.
func TestValidateTOTPStep_ReturnsStep(t *testing.T) {
	key, err := mfa.GenerateTOTPKey(mfa.TOTPConfig{Issuer: "TestApp", AccountName: "step@test.com"})
	require.NoError(t, err)
	code, err := mfa.GenerateTOTPCode(key.Secret())
	require.NoError(t, err)

	ok, step := mfa.ValidateTOTPStep(code, key.Secret())
	require.True(t, ok, "a fresh code must validate")
	assert.Positive(t, step, "a valid code must map to a positive time-step")

	bad, _ := mfa.ValidateTOTPStep("000000", key.Secret())
	assert.False(t, bad || code == "000000", "an incorrect code must not validate")
}
