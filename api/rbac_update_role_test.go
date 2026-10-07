package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
)

func sendJSON(t *testing.T, handler http.Handler, eng *authsome.Engine, as id.UserID, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req = asAdmin(t, req, eng, as)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

type roleBody struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parent_id"`
}

func decodeRole(t *testing.T, rec *httptest.ResponseRecorder) roleBody {
	t.Helper()
	var r roleBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &r), "body=%s", rec.Body.String())
	// The handlers take the authsome form of a role id, which shares the
	// suffix with the warden form the store hands back.
	if i := strings.IndexByte(r.ID, '_'); i >= 0 {
		r.ID = "arol_" + r.ID[i+1:]
	}
	return r
}

// PATCH /roles/:id does not re-parent a role. It used to answer 200 and
// leave the parent alone, so a parent_id that differs from the stored one is
// now refused out loud instead.
func TestUpdateRole_RefusesParentChange(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "reparent-owner@test.com", "SecureP@ss123")
	owner := userIDFor(t, eng, ownerToken)

	rec := sendJSON(t, handler, eng, owner, http.MethodPost, "/v1/roles", `{"name":"Parent","slug":"reparent-parent"}`)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	parent := decodeRole(t, rec)

	rec = sendJSON(t, handler, eng, owner, http.MethodPost, "/v1/roles", `{"name":"Child","slug":"reparent-child"}`)
	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())
	child := decodeRole(t, rec)

	rec = sendJSON(t, handler, eng, owner, http.MethodPatch, "/v1/roles/"+child.ID,
		`{"name":"Renamed","parent_id":"`+parent.ID+`"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, "body=%s", rec.Body.String())
	assert.Contains(t, rec.Body.String(), "parent_id")

	// Nothing from the refused request landed, the name included.
	rec = sendJSON(t, handler, eng, owner, http.MethodGet, "/v1/roles/"+child.ID, "")
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	got := decodeRole(t, rec)
	assert.Equal(t, "Child", got.Name)
	assert.Empty(t, got.ParentID)

	// Sending the parent the role already has is not a change, so a client
	// that echoes back what it read still gets its rename through.
	rec = sendJSON(t, handler, eng, owner, http.MethodPatch, "/v1/roles/"+child.ID,
		`{"name":"Renamed","parent_id":""}`)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	assert.Equal(t, "Renamed", decodeRole(t, rec).Name)
}
