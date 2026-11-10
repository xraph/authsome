package mfa_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/mfa"
)

// GET /v1/mfa/enrollment answers "is two-step on for me". The account summary
// card reads it, and before it existed that row had to guess.

func newEnrollmentHandler(t *testing.T) (*mfa.MemoryStore, http.Handler) {
	t.Helper()
	p, s := newTestPlugin(t)
	router := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(router))
	return s, router.Handler()
}

func enroll(t *testing.T, s *mfa.MemoryStore, userID id.UserID, method string, verified bool) {
	t.Helper()
	require.NoError(t, s.CreateEnrollment(context.Background(), &mfa.Enrollment{
		ID:        id.NewMFAID(),
		UserID:    userID,
		Method:    method,
		Secret:    "JBSWY3DPEHPK3PXP",
		Verified:  verified,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}))
}

func readEnrollment(t *testing.T, h http.Handler, userID id.UserID) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, authedRequest(t, "GET", "/v1/mfa/enrollment", nil, userID))
	return rec
}

func decodeEnrollment(t *testing.T, rec *httptest.ResponseRecorder) mfa.EnrollmentResponse {
	t.Helper()
	var got mfa.EnrollmentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got
}

func TestGetEnrollment_NothingEnrolled(t *testing.T) {
	_, h := newEnrollmentHandler(t)

	rec := readEnrollment(t, h, id.NewUserID())
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	got := decodeEnrollment(t, rec)
	assert.False(t, got.Enabled)
	assert.Empty(t, got.Factors)
	assert.Zero(t, got.RecoveryCodesRemaining)
}

func TestGetEnrollment_VerifiedTOTPTurnsItOn(t *testing.T) {
	s, h := newEnrollmentHandler(t)
	userID := id.NewUserID()
	enroll(t, s, userID, "totp", true)

	got := decodeEnrollment(t, readEnrollment(t, h, userID))
	assert.True(t, got.Enabled)
	require.Len(t, got.Factors, 1)
	assert.Equal(t, "totp", got.Factors[0].Method)
	assert.True(t, got.Factors[0].Verified)
}

// An enrollment that was started and never verified is reported, because the
// settings page has to show it, but it does not make two-step on.
func TestGetEnrollment_UnverifiedDoesNotCount(t *testing.T) {
	s, h := newEnrollmentHandler(t)
	userID := id.NewUserID()
	enroll(t, s, userID, "totp", false)

	got := decodeEnrollment(t, readEnrollment(t, h, userID))
	assert.False(t, got.Enabled)
	require.Len(t, got.Factors, 1)
	assert.False(t, got.Factors[0].Verified)
}

func TestGetEnrollment_ReportsEveryFactor(t *testing.T) {
	s, h := newEnrollmentHandler(t)
	userID := id.NewUserID()
	enroll(t, s, userID, "totp", true)
	enroll(t, s, userID, "sms", true)

	got := decodeEnrollment(t, readEnrollment(t, h, userID))
	assert.True(t, got.Enabled)
	methods := []string{got.Factors[0].Method, got.Factors[1].Method}
	assert.ElementsMatch(t, []string{"totp", "sms"}, methods)
}

// The count is what an account page needs to warn someone who is running low.
// The codes themselves are never in the response.
func TestGetEnrollment_CountsUnusedRecoveryCodes(t *testing.T) {
	s, h := newEnrollmentHandler(t)
	userID := id.NewUserID()
	enroll(t, s, userID, "totp", true)

	now := time.Now()
	require.NoError(t, s.CreateRecoveryCodes(context.Background(), []*mfa.RecoveryCode{
		{ID: id.NewRecoveryCodeID(), UserID: userID, CodeHash: "h1", CreatedAt: now},
		{ID: id.NewRecoveryCodeID(), UserID: userID, CodeHash: "h2", CreatedAt: now},
		{ID: id.NewRecoveryCodeID(), UserID: userID, CodeHash: "h3", Used: true, UsedAt: &now, CreatedAt: now},
	}))

	rec := readEnrollment(t, h, userID)
	got := decodeEnrollment(t, rec)
	assert.Equal(t, 2, got.RecoveryCodesRemaining, "a used code is not one you have left")
	assert.NotContains(t, rec.Body.String(), "h1", "a code hash has no business in this response")
}

// One user's enrollment says nothing about anyone else's.
func TestGetEnrollment_IsScopedToTheCaller(t *testing.T) {
	s, h := newEnrollmentHandler(t)
	mine, theirs := id.NewUserID(), id.NewUserID()
	enroll(t, s, theirs, "totp", true)

	got := decodeEnrollment(t, readEnrollment(t, h, mine))
	assert.False(t, got.Enabled)
	assert.Empty(t, got.Factors)
}

func TestGetEnrollment_NeverReturnsTheSecret(t *testing.T) {
	s, h := newEnrollmentHandler(t)
	userID := id.NewUserID()
	enroll(t, s, userID, "totp", true)

	rec := readEnrollment(t, h, userID)
	assert.NotContains(t, rec.Body.String(), "JBSWY3DPEHPK3PXP")
}
