package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/lockout"
)

// POST /v1/admin/users/:id/unlock lifts the lockout on every network.
func TestAdminUnlockUser_LiftsTheLock(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	eng.SetLockoutTracker(lockout.NewMemoryTracker(lockout.WithMaxAttempts(2), lockout.WithLockoutDuration(10*time.Minute)))
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "unlock-owner@test.com", "SecureP@ss1")
	owner := userIDFor(t, eng, ownerToken)
	_, victimToken, _ := signUp(t, eng, "unlock-victim@test.com", "SecureP@ss1")
	victim := userIDFor(t, eng, victimToken)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	attempt := func(password string) error {
		_, _, err := eng.SignIn(context.Background(), &account.SignInRequest{
			AppID: appID, Email: "unlock-victim@test.com", Password: password, IPAddress: "203.0.113.5",
		})
		return err
	}
	for i := 0; i < 2; i++ {
		require.ErrorIs(t, attempt("wrong-password"), account.ErrInvalidCredentials)
	}
	require.ErrorIs(t, attempt("SecureP@ss1"), account.ErrAccountLocked)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/admin/users/"+victim.String()+"/unlock", nil)
	req = asAdmin(t, req, eng, owner)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	assert.NoError(t, attempt("SecureP@ss1"), "the victim signs in again from the locked network")
}

func TestAdminUnlockUser_OtherAppIs404(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "unlock-owner2@test.com", "SecureP@ss1")
	owner := userIDFor(t, eng, ownerToken)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/admin/users/"+id.NewUserID().String()+"/unlock", nil)
	req = asAdmin(t, req, eng, owner)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code, "an unknown or foreign user reads as absent")
}
