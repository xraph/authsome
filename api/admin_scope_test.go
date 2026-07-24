package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"
)

// seedForeignUser creates a user that belongs to a different app than the test
// platform app, for cross-tenant admin tests.
func seedForeignUser(t *testing.T, eng *authsome.Engine, email string) *user.User {
	t.Helper()
	now := time.Now()
	u := &user.User{
		ID:        id.NewUserID(),
		AppID:     otherAppID(t),
		Email:     email,
		FirstName: "Foreign",
		CreatedAt: now,
		UpdatedAt: now,
	}
	require.NoError(t, eng.Store().CreateUser(context.Background(), u))
	return u
}

// The admin user-management endpoints are gated by manage:user, evaluated
// against the caller's own app. Without a per-target tenant check, a tenant
// admin could view, ban, delete, edit, or impersonate a user in another app.
// Each handler must confirm the target user's app matches the caller's.

func TestAdminGetUser_SameAppSucceeds(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "admin-get-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	// A user in the caller's own (platform) app.
	_, victimToken, _ := signUp(t, eng, "admin-get-victim@test.com", "SecureP@ss1")
	victimID := userIDFor(t, eng, victimToken)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/admin/users/"+victimID.String(), nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "owner must read a user in their own app; body=%s", rec.Body.String())
}

func TestAdminGetUser_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "admin-x-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreign := seedForeignUser(t, eng, "foreign-user@test.com")

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/admin/users/"+foreign.ID.String(), nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "must not read a user from another app; body=%s", rec.Body.String())
}

func TestAdminBanUser_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "admin-ban-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreign := seedForeignUser(t, eng, "foreign-ban@test.com")

	body := []byte(`{"reason":"pwned"}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/admin/users/"+foreign.ID.String()+"/ban", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	got, err := eng.AdminGetUser(context.Background(), foreign.ID)
	require.NoError(t, err)
	assert.False(t, got.Banned, "a user in another app must not be banned")
}

func TestAdminImpersonate_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "admin-imp-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreign := seedForeignUser(t, eng, "foreign-imp@test.com")

	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/admin/impersonate/"+foreign.ID.String(), nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "must not impersonate a user from another app; body=%s", rec.Body.String())
}

func TestAdminDeleteUser_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "admin-del-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreign := seedForeignUser(t, eng, "foreign-del@test.com")

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/admin/users/"+foreign.ID.String(), nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	_, err := eng.AdminGetUser(context.Background(), foreign.ID)
	assert.NoError(t, err, "a user in another app must survive the delete attempt")
}
