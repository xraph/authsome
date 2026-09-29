package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
)

const scopedSettingKey = "session.jwt_require_active_session"

func putSetting(t *testing.T, handler http.Handler, eng *authsome.Engine, as id.UserID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/v1/admin/settings/values/"+scopedSettingKey, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, as)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// A platform owner may write settings, but only inside their own app unless
// the scope is global; a tenant admin cannot touch settings at all.

func TestSetSetting_OwnerCannotWriteAnotherAppsScope(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "settings-owner@test.com", "SecureP@ss1")
	owner := userIDFor(t, eng, ownerToken)

	rec := putSetting(t, handler, eng, owner,
		`{"value":true,"scope":"app","scope_id":"`+otherAppID(t).String()+`"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "app scope must be the caller's own app; body=%s", rec.Body.String())
}

func TestSetSetting_OwnerWritesOwnAppScope(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "settings-owner2@test.com", "SecureP@ss1")
	owner := userIDFor(t, eng, ownerToken)

	rec := putSetting(t, handler, eng, owner,
		`{"value":true,"scope":"app","scope_id":"`+testAppIDStr+`"}`)
	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

func TestSetSetting_TenantAdminIsRefused(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	admin := appAdmin(t, eng, "settings-admin@test.com")

	rec := putSetting(t, handler, eng, admin, `{"value":true,"scope":"global"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "settings are owner-only; body=%s", rec.Body.String())
}
