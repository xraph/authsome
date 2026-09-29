package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/session"
)

// /v1/me reports the roles the user holds now, not whatever was stamped on
// the session when it was issued.
func TestGetMe_ReadsLiveRoles(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "me-live@test.com", "SecureP@ss123")
	owner := userIDFor(t, eng, ownerToken)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	stale := &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: owner, Roles: []string{"stale-role"},
		CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/me", nil)
	req = asAdmin(t, req, eng, owner)
	req = req.WithContext(middleware.WithSession(req.Context(), stale))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())

	var resp struct {
		Roles []string `json:"roles"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotContains(t, resp.Roles, "stale-role")
	assert.Contains(t, resp.Roles, rbac.PlatformOwnerSlug)
}
