package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/serviceaccount"
)

// seedMintChildParent creates a service account under the test platform app.
// The returned ServiceAccount has default scopes ["repo:read"]; opts may
// override any field (Active, ParentID, ExpiresAt, ...) before it is stored.
func seedMintChildParent(
	t *testing.T, eng *authsome.Engine, opts func(*serviceaccount.ServiceAccount),
) *serviceaccount.ServiceAccount {
	t.Helper()
	ctx := context.Background()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	svcID := id.NewServiceAccountID()
	now := time.Now()
	svc := &serviceaccount.ServiceAccount{
		ID:        svcID,
		AppID:     appID,
		Name:      "mint-child-parent-" + svcID.String(),
		Kind:      principal.KindAgent,
		Scopes:    []string{"repo:read"},
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if opts != nil {
		opts(svc)
	}
	require.NoError(t, eng.Store().CreateServiceAccount(ctx, svc))
	return svc
}

// asServiceAccount injects the resolved principal for svcID onto the request
// context, mirroring what the auth middleware's strategy resolution sets in
// production (see setPrincipalContext in middleware/auth.go). The bare
// forge.Router these tests build via a.RegisterRoutes never runs that
// middleware: it is only wired in by the Forge Extension system in a real
// deployment, so every authenticated test in this package injects context
// directly rather than presenting real credentials over HTTP; asAdmin
// (unauth_admin_endpoints_test.go) does the same for human callers.
func asServiceAccount(t *testing.T, req *http.Request, eng *authsome.Engine, svcID id.ServiceAccountID) *http.Request {
	t.Helper()
	ref := principal.Ref{Kind: principal.KindService, ID: svcID.String()}
	p, err := eng.ResolvePrincipal(context.Background(), ref)
	require.NoError(t, err)
	return req.WithContext(middleware.WithPrincipal(req.Context(), p))
}

// mintChildRequest builds a POST /v1/principals/:id/children request body.
// Callers attach the caller identity separately via asServiceAccount.
func mintChildRequest(pathID string, body map[string]any) *http.Request {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost,
		"/v1/principals/"+pathID+"/children", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// A scope the parent does not hold is something the caller can fix by
// asking for less: it must come back as 400, not a 500 carrying the wrapped
// internal error text. Before FINDING 1 was fixed, ErrChildScopeExceedsParent
// had no case in the handler and fell through to the generic mapError ->
// forge.InternalError, so this assertion is what actually pins the fix: a
// bare "err != nil" check would have passed against that 500 too.
func TestMintChildAPI_ScopeExceedsParent_Returns400(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())
	parent := seedMintChildParent(t, eng, nil) // Scopes: ["repo:read"]

	req := mintChildRequest(parent.ID.String(), map[string]any{
		"name":        "child",
		"scopes":      []string{"repo:write"},
		"ttl_seconds": 3600,
	})
	req = asServiceAccount(t, req, eng, parent.ID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code,
		"a child scope outside the parent's must be a 400, not a 500; body=%s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	msg, _ := resp["error"].(string)
	assert.NotContains(t, msg, "authsome:", "the response body must carry a stable message, not the wrapped internal error text")
}

// An inactive parent is not permitted to mint at all, no matter what the
// caller asks for: that is a 403, distinct from the 400 a fixable request
// gets.
func TestMintChildAPI_InactiveParent_Returns403(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())
	parent := seedMintChildParent(t, eng, func(svc *serviceaccount.ServiceAccount) {
		svc.Active = false
	})

	req := mintChildRequest(parent.ID.String(), map[string]any{
		"name":        "child",
		"ttl_seconds": 3600,
	})
	req = asServiceAccount(t, req, eng, parent.ID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code,
		"an inactive parent must be a 403, not a 500; body=%s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	msg, _ := resp["error"].(string)
	assert.NotContains(t, msg, "authsome:", "the response body must carry a stable message, not the wrapped internal error text")
}

// One level only: an ephemeral child must not be able to mint a grandchild.
// Same 403 as the inactive-parent case, both routed through
// ErrChildMintNotPermitted.
func TestMintChildAPI_EphemeralParent_Returns403(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())
	grandparent := seedMintChildParent(t, eng, nil)

	child, _, _, err := eng.MintChildPrincipal(context.Background(), grandparent.ID, "child", nil, time.Hour)
	require.NoError(t, err)

	req := mintChildRequest(child.ID.String(), map[string]any{
		"name":        "grandchild",
		"ttl_seconds": 3600,
	})
	req = asServiceAccount(t, req, eng, child.ID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code,
		"an ephemeral child minting a child of its own must be a 403, not a 500; body=%s", rec.Body.String())
}

// Sanity: the new error-mapping in front of mapError must not disturb the
// success path. A permitted request still returns 201 with the one-time
// secret.
func TestMintChildAPI_Success_Returns201(t *testing.T) {
	a, eng := newTestAPI(t)
	handler := withTestKey(a.Handler())
	parent := seedMintChildParent(t, eng, nil) // Scopes: ["repo:read"]

	req := mintChildRequest(parent.ID.String(), map[string]any{
		"name":        "child",
		"scopes":      []string{"repo:read"},
		"ttl_seconds": 3600,
	})
	req = asServiceAccount(t, req, eng, parent.ID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code, "body=%s", rec.Body.String())

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	key, _ := resp["key"].(string)
	assert.NotEmpty(t, key, "the response must carry the one-time secret")
	assert.Equal(t, parent.ID.String(), resp["parent_id"])
}
