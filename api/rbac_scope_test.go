package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/rbac"
)

// seedRole creates an RBAC role in the given app/tenant and returns the
// authsome-format ("arol_") role id, which the HTTP handlers accept and warden
// resolves by suffix. (CreateRole rewrites r.ID to warden format, so we keep
// the original id we generated.)
func seedRole(t *testing.T, eng *authsome.Engine, appIDStr string) string {
	t.Helper()
	rid := id.NewRoleID()
	suffix := rid.String()[len(rid.String())-8:]
	r := &rbac.Role{
		ID:    rid.String(),
		AppID: appIDStr,
		Name:  "Role " + suffix,
		Slug:  "role-" + suffix,
	}
	require.NoError(t, eng.CreateRole(context.Background(), r))
	return rid.String()
}

// The RBAC endpoints require manage/read/assign permissions on "role", but that
// is evaluated against the caller's own app. Without a tenant check on the
// target role, an admin of app A could read, modify, delete, add permissions
// to, or assign users to a role that belongs to app B — including escalating a
// user into another tenant's owner role. Each handler must verify the target
// role's AppID matches the caller's app.

func TestGetRole_SameAppSucceeds(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "role-get-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	roleID := seedRole(t, eng, testAppIDStr)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/roles/"+roleID, nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "owner must read a role in their own app; body=%s", rec.Body.String())
}

func TestGetRole_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "role-x-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreignRole := seedRole(t, eng, otherAppID(t).String())

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/roles/"+foreignRole, nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "a role from another app must not be readable; body=%s", rec.Body.String())
}

func TestDeleteRole_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "role-del-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreignRole := seedRole(t, eng, otherAppID(t).String())

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/roles/"+foreignRole, nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	rid, err := id.ParseRoleID(foreignRole)
	require.NoError(t, err)
	_, err = eng.GetRole(context.Background(), rid)
	assert.NoError(t, err, "the foreign role must survive the delete attempt")
}

func TestAddPermission_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "perm-x-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreignRole := seedRole(t, eng, otherAppID(t).String())

	body := []byte(`{"action":"manage","resource":"app"}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/roles/"+foreignRole+"/permissions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "must not add permissions to another app's role; body=%s", rec.Body.String())
}

func TestAssignRole_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "assign-x-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	foreignRole := seedRole(t, eng, otherAppID(t).String())
	victim := id.NewUserID()

	body := []byte(`{"user_id":"` + victim.String() + `"}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/v1/roles/"+foreignRole+"/assign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "must not assign users into another app's role; body=%s", rec.Body.String())
}
