package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// Cross-tenant guards on the admin surfaces that used to take an app id from
// the request: service accounts, key minting, user listing, bulk session
// revocation and app deletion.

const adminServiceAccountPath = "/v1/admin/service-accounts"

func TestAdminServiceAccount_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "sa-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreign, err := eng.CreateServiceAccount(context.Background(), otherAppID(t), "foreign-svc", "", []string{"read"})
	require.NoError(t, err)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, adminServiceAccountPath + "/" + foreign.ID.String()},
		{http.MethodDelete, adminServiceAccountPath + "/" + foreign.ID.String()},
	} {
		req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.path, nil)
		req = asAdmin(t, req, eng, ownerID)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code, "%s %s must not reach another app's service account; body=%s", tc.method, tc.path, rec.Body.String())
	}
}

func TestAdminServiceAccountKey_RejectsScopeEscalation(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "sa-scope-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	platformApp, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	svc, err := eng.CreateServiceAccount(context.Background(), platformApp, "narrow-svc", "", []string{"read"})
	require.NoError(t, err)

	body := []byte(`{"name":"k","scopes":["read","admin"]}`)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, adminServiceAccountPath+"/"+svc.ID.String()+"/api-keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "a key must not carry scopes its account lacks; body=%s", rec.Body.String())
}

func TestAdminListUsers_RejectsForeignAppID(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "list-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/admin/users?app_id="+otherAppID(t).String(), nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code, "listing another app's users must be refused; body=%s", rec.Body.String())
}

func TestAdminBulkRevokeSessions_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "revoke-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)
	foreign := seedForeignUser(t, eng, "revoke-foreign@test.com")

	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/admin/bulk/sessions?user_id="+foreign.ID.String(), nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code, "revoking another app's sessions must be refused; body=%s", rec.Body.String())
}

func TestAdminDeleteApp_RefusesThePlatformApp(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "delapp-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/v1/admin/apps/"+testAppIDStr, nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code, "the platform app must never be deletable; body=%s", rec.Body.String())
}
