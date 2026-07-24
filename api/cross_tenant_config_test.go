package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
)

// The per-app client/session config admin endpoints are gated by
// manage:app, but that permission is evaluated against the caller's OWN app.
// Without a tenant-match check, an admin of app A could read or rewrite app B's
// security-relevant config by putting B's id in the path. scopedAppID closes
// that: a request whose path app differs from the caller's bound app is 403.

func TestGetAppClientConfig_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())

	_, ownerToken, _ := signUp(t, eng, "cfg-get-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/admin/apps/"+otherAppID(t).String()+"/client-config", nil)
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "reading another app's client config must be forbidden; body=%s", rec.Body.String())
}

func TestSetAppClientConfig_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())

	_, ownerToken, _ := signUp(t, eng, "cfg-set-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	body := []byte(`{"signup_enabled":false}`)
	req := httptest.NewRequestWithContext(context.Background(), "PUT", "/v1/admin/apps/"+otherAppID(t).String()+"/client-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "rewriting another app's client config must be forbidden; body=%s", rec.Body.String())

	// The foreign app must have no config written.
	_, err := eng.Store().GetAppClientConfig(context.Background(), otherAppID(t))
	assert.Error(t, err, "no config should have been created for the foreign app")
}

func TestSetAppClientConfig_SameAppSucceeds(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())

	_, ownerToken, _ := signUp(t, eng, "cfg-same-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	appID, err := id.ParseAppID(testAppIDStr)
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"signup_enabled":false}`)
	req := httptest.NewRequestWithContext(context.Background(), "PUT", "/v1/admin/apps/"+appID.String()+"/client-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "owner must configure their own app; body=%s", rec.Body.String())
}

func TestSetAppSessionConfig_RejectsCrossTenant(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())

	_, ownerToken, _ := signUp(t, eng, "sess-cfg-owner@test.com", "SecureP@ss1")
	ownerID := userIDFor(t, eng, ownerToken)

	body := []byte(`{}`)
	req := httptest.NewRequestWithContext(context.Background(), "PUT", "/v1/admin/apps/"+otherAppID(t).String()+"/session-config", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, ownerID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "rewriting another app's session config must be forbidden; body=%s", rec.Body.String())
}
