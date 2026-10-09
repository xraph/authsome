package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Introspection admits nobody anonymous, shows a plain caller only what a
// resource server needs, and shows the user block to a caller with manage
// on app.
func TestIntrospect_GatedAndMinimal(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	router := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "introspect-gate-owner@test.com", "SecureP@ss123")
	ownerID := userIDFor(t, eng, ownerToken)
	_, subjectToken, _ := signUp(t, eng, "introspect-subject@test.com", "SecureP@ss123")

	post := func(mutate func(*http.Request) *http.Request) (*httptest.ResponseRecorder, map[string]any) {
		body, err := json.Marshal(map[string]string{"token": subjectToken})
		require.NoError(t, err)
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/introspect", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if mutate != nil {
			req = mutate(req)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return rec, resp
	}

	rec, _ := post(nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "anonymous introspection is refused")

	rec, resp := post(func(r *http.Request) *http.Request { return asCaller(t, r, eng) })
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, true, resp["active"])
	assert.NotEmpty(t, resp["user_id"], "the subject is named")
	assert.NotEmpty(t, resp["expires_at"])
	for _, hidden := range []string{"user", "app_id", "session_id", "env_id"} {
		assert.NotContains(t, resp, hidden, "a plain caller does not see %s", hidden)
	}

	rec, resp = post(func(r *http.Request) *http.Request { return asAdmin(t, r, eng, ownerID) })
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, resp, "user", "manage on app shows the user block")
	assert.NotEmpty(t, resp["app_id"])
}
