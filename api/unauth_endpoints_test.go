package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// asUser returns a copy of req whose context carries the given user ID, mimicking
// what the auth middleware sets after resolving a session in production. The bare
// API test handler does not run the identity-resolving AuthMiddleware, so tests
// inject identity directly (matching the existing WithUserID convention).
func asUser(req *http.Request, userID id.UserID) *http.Request {
	return req.WithContext(middleware.WithUserID(req.Context(), userID))
}

// userIDFor resolves the user ID behind a session token.
func userIDFor(t *testing.T, eng *authsome.Engine, token string) id.UserID {
	t.Helper()
	sess, err := eng.ResolveSessionByToken(token)
	require.NoError(t, err)
	return sess.UserID
}

// registerDeviceFor creates a tracked device owned by the given user and returns it.
func registerDeviceFor(t *testing.T, eng *authsome.Engine, userID id.UserID) *device.Device {
	t.Helper()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	d, err := eng.RegisterDevice(context.Background(), &device.Device{
		UserID: userID,
		AppID:  appID,
		Name:   "Test Device",
	})
	require.NoError(t, err)
	return d
}

// ──────────────────────────────────────────────────
// Sessions — DELETE /v1/sessions/:id
// ──────────────────────────────────────────────────

func TestRevokeSession_RequiresAuth(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, token, _ := signUp(t, eng, "sess-noauth@test.com", "SecureP@ss1")
	sess, err := eng.ResolveSessionByToken(token)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/sessions/"+sess.ID.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	_, err = eng.ResolveSessionByToken(token)
	assert.NoError(t, err, "session must not be revoked by an unauthenticated caller")
}

func TestRevokeSession_RejectsNonOwner(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, tokenA, _ := signUp(t, eng, "sess-owner-a@test.com", "SecureP@ss1")
	_, tokenB, _ := signUp(t, eng, "sess-owner-b@test.com", "SecureP@ss1")

	sessB, err := eng.ResolveSessionByToken(tokenB)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/sessions/"+sessB.ID.String(), nil)
	req = asUser(req, userIDFor(t, eng, tokenA))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	_, err = eng.ResolveSessionByToken(tokenB)
	assert.NoError(t, err, "user B's session must survive user A's revoke attempt")
}

func TestRevokeSession_OwnerSucceeds(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, token, _ := signUp(t, eng, "sess-owner@test.com", "SecureP@ss1")
	sess, err := eng.ResolveSessionByToken(token)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/sessions/"+sess.ID.String(), nil)
	req = asUser(req, sess.UserID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "owner must be able to revoke their own session; body=%s", rec.Body.String())
}

// ──────────────────────────────────────────────────
// Devices — GET/DELETE /v1/devices/:id, PATCH /v1/devices/:id/trust
// ──────────────────────────────────────────────────

func TestGetDevice_RequiresAuth(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, token, _ := signUp(t, eng, "dev-noauth@test.com", "SecureP@ss1")
	d := registerDeviceFor(t, eng, userIDFor(t, eng, token))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/devices/"+d.ID.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetDevice_RejectsNonOwner(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, tokenA, _ := signUp(t, eng, "dev-a@test.com", "SecureP@ss1")
	_, tokenB, _ := signUp(t, eng, "dev-b@test.com", "SecureP@ss1")
	dB := registerDeviceFor(t, eng, userIDFor(t, eng, tokenB))

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/devices/"+dB.ID.String(), nil)
	req = asUser(req, userIDFor(t, eng, tokenA))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code, "user A must not read user B's device")
}

func TestGetDevice_OwnerSucceeds(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, token, _ := signUp(t, eng, "dev-owner@test.com", "SecureP@ss1")
	uid := userIDFor(t, eng, token)
	d := registerDeviceFor(t, eng, uid)

	req := httptest.NewRequestWithContext(context.Background(), "GET", "/v1/devices/"+d.ID.String(), nil)
	req = asUser(req, uid)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

func TestDeleteDevice_RequiresAuth(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, token, _ := signUp(t, eng, "dev-del-noauth@test.com", "SecureP@ss1")
	d := registerDeviceFor(t, eng, userIDFor(t, eng, token))

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/devices/"+d.ID.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	_, err := eng.GetDevice(context.Background(), d.ID)
	assert.NoError(t, err, "device must not be deleted by an unauthenticated caller")
}

func TestDeleteDevice_RejectsNonOwner(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, tokenA, _ := signUp(t, eng, "dev-del-a@test.com", "SecureP@ss1")
	_, tokenB, _ := signUp(t, eng, "dev-del-b@test.com", "SecureP@ss1")
	dB := registerDeviceFor(t, eng, userIDFor(t, eng, tokenB))

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/devices/"+dB.ID.String(), nil)
	req = asUser(req, userIDFor(t, eng, tokenA))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	_, err := eng.GetDevice(context.Background(), dB.ID)
	assert.NoError(t, err, "user B's device must survive user A's delete attempt")
}

func TestDeleteDevice_OwnerSucceeds(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, token, _ := signUp(t, eng, "dev-del-owner@test.com", "SecureP@ss1")
	uid := userIDFor(t, eng, token)
	d := registerDeviceFor(t, eng, uid)

	req := httptest.NewRequestWithContext(context.Background(), "DELETE", "/v1/devices/"+d.ID.String(), nil)
	req = asUser(req, uid)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

func TestTrustDevice_RequiresAuth(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, token, _ := signUp(t, eng, "dev-trust-noauth@test.com", "SecureP@ss1")
	d := registerDeviceFor(t, eng, userIDFor(t, eng, token))

	req := httptest.NewRequestWithContext(context.Background(), "PATCH", "/v1/devices/"+d.ID.String()+"/trust", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestTrustDevice_RejectsNonOwner(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, tokenA, _ := signUp(t, eng, "dev-trust-a@test.com", "SecureP@ss1")
	_, tokenB, _ := signUp(t, eng, "dev-trust-b@test.com", "SecureP@ss1")
	dB := registerDeviceFor(t, eng, userIDFor(t, eng, tokenB))

	req := httptest.NewRequestWithContext(context.Background(), "PATCH", "/v1/devices/"+dB.ID.String()+"/trust", nil)
	req = asUser(req, userIDFor(t, eng, tokenA))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	got, err := eng.GetDevice(context.Background(), dB.ID)
	require.NoError(t, err)
	assert.False(t, got.Trusted, "user B's device must not be trusted by user A")
}
