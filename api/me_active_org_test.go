package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

// GET /v1/me reports the session's active organization, so a client that
// reloads knows which organization the session is in without having to
// remember it.
func TestGetMe_ActiveOrgID(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())

	_, token, _ := signUp(t, eng, "activeorg@test.com", "SecureP@ss123")
	sess, err := eng.ResolveSessionByToken(context.Background(), token)
	require.NoError(t, err)

	get := func(ctx context.Context) map[string]any {
		t.Helper()
		req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/me", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var resp map[string]any
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
		return resp
	}

	base := middleware.WithUserID(context.Background(), sess.UserID)
	_, present := get(middleware.WithSession(base, sess))["active_org_id"]
	assert.False(t, present, "a session without an organization reports none")

	orgID := id.NewOrgID()
	withOrg := *sess
	withOrg.OrgID = orgID
	assert.Equal(t, orgID.String(), get(middleware.WithSession(base, &withOrg))["active_org_id"])

	// A JWT carries the organization in its claims and there's no session
	// row; the auth middleware puts the org on the context instead.
	jwtOrg := id.NewOrgID()
	assert.Equal(t, jwtOrg.String(), get(middleware.WithOrgID(base, jwtOrg))["active_org_id"])
}
