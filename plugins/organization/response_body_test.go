package organization_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/organization"
)

// The handlers that answer with a single record wrote it with ctx.JSON and
// then also returned it, so forge serialized it a second time. The status was
// right but the body held two JSON documents back to back, and no client
// could decode it. Every one of those routes must answer exactly one document.

// appReq stamps the test app and an environment on r, the way the auth and
// environment middleware do for a signed-in caller, so routes that scope by
// app (create org) accept it.
func appReq(t *testing.T, r *http.Request) *http.Request {
	t.Helper()
	appID, err := id.ParseAppID(orgTestAppID)
	require.NoError(t, err)
	ctx := middleware.WithAppID(r.Context(), appID)
	ctx = middleware.WithEnvID(ctx, id.NewEnvironmentID())
	return r.WithContext(ctx)
}

// decodeOne fails unless body is exactly one JSON value.
func decodeOne(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	dec := json.NewDecoder(rec.Body)
	var v map[string]any
	require.NoError(t, dec.Decode(&v), "body must be JSON")
	assert.False(t, dec.More(), "body must hold one JSON document, got a second one after it")
	return v
}

func TestOrgHandlers_WriteOneJSONBody(t *testing.T) {
	p, h := newOrgHTTP(t)
	owner := id.NewUserID()
	o := seedOrg(t, p, owner)
	orgPath := "/v1/orgs/" + o.ID.String()

	serve := func(method, target string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var raw []byte
		if body != nil {
			raw = postJSON(t, body)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, appReq(t, orgReq(method, target, raw, owner)))
		return rec
	}

	t.Run("create org", func(t *testing.T) {
		rec := serve("POST", "/v1/orgs", map[string]string{"name": "Beta", "slug": "beta"})
		require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
		assert.Equal(t, "beta", decodeOne(t, rec)["slug"])
	})

	t.Run("get org", func(t *testing.T) {
		rec := serve("GET", orgPath, nil)
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		decodeOne(t, rec)
	})

	t.Run("update org", func(t *testing.T) {
		rec := serve("PATCH", orgPath, map[string]string{"name": "Acme Two"})
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		assert.Equal(t, "Acme Two", decodeOne(t, rec)["name"])
	})

	var memberID string
	t.Run("add member", func(t *testing.T) {
		rec := serve("POST", orgPath+"/members", map[string]string{"user_id": id.NewUserID().String(), "role": "member"})
		require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
		v := decodeOne(t, rec)
		memberID, _ = v["id"].(string)
		require.NotEmpty(t, memberID)
	})

	t.Run("update member", func(t *testing.T) {
		rec := serve("PATCH", orgPath+"/members/"+memberID, map[string]string{"role": string(organization.RoleAdmin)})
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		decodeOne(t, rec)
	})

	t.Run("create invitation", func(t *testing.T) {
		rec := serve("POST", orgPath+"/invitations", map[string]string{"email": "guest@example.com"})
		require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
		decodeOne(t, rec)
	})

	var teamID string
	t.Run("create team", func(t *testing.T) {
		rec := serve("POST", orgPath+"/teams", map[string]string{"name": "Platform", "slug": "platform"})
		require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
		teamID, _ = decodeOne(t, rec)["id"].(string)
		require.NotEmpty(t, teamID)
	})

	t.Run("get team", func(t *testing.T) {
		rec := serve("GET", orgPath+"/teams/"+teamID, nil)
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		decodeOne(t, rec)
	})

	t.Run("update team", func(t *testing.T) {
		rec := serve("PATCH", orgPath+"/teams/"+teamID, map[string]string{"name": "Platform Two"})
		require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
		decodeOne(t, rec)
	})
}
