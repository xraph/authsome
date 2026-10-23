package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// An agent session carries its delegating user's UserID and no user record,
// which is enough for RequireAuth and for a permission check run against the
// user. These routes mint something the grant does not bound (a session as
// another user, a service-account key, a password the agent then knows), so
// they refuse an agent session even when the user behind it could do the same.

// asAgentOf puts an agent session acting for userID on req, the way the auth
// middleware does for a resolved agent token.
func asAgentOf(t *testing.T, req *http.Request, userID id.UserID) *http.Request {
	t.Helper()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	sess := &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: userID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	}
	ctx := middleware.WithSession(req.Context(), sess)
	ctx = middleware.WithSessionID(ctx, sess.ID)
	ctx = middleware.WithAppID(ctx, appID)
	ctx = middleware.WithUserID(ctx, userID)
	return req.WithContext(ctx)
}

// asOwnSession adds the user's own session to a request asAdmin built.
func asOwnSession(t *testing.T, req *http.Request, userID id.UserID) *http.Request {
	t.Helper()
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	sess := &session.Session{ID: id.NewSessionID(), AppID: appID, UserID: userID}
	ctx := middleware.WithSession(req.Context(), sess)
	ctx = middleware.WithSessionID(ctx, sess.ID)
	return req.WithContext(ctx)
}

func TestAdminImpersonate_RefusesAgentSession(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "agent-imp-owner@test.com", "SecureP@ss123")
	ownerID := userIDFor(t, eng, ownerToken)
	_, targetToken, _ := signUp(t, eng, "agent-imp-target@test.com", "SecureP@ss123")
	targetID := userIDFor(t, eng, targetToken)

	before, err := eng.Store().ListUserSessions(context.Background(), targetID)
	require.NoError(t, err)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/admin/impersonate/"+targetID.String(), nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, asAgentOf(t, req, ownerID))

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "session_token")
	after, err := eng.Store().ListUserSessions(context.Background(), targetID)
	require.NoError(t, err)
	assert.Len(t, after, len(before), "no session is minted for the target")

	// The owner's own session still impersonates.
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/admin/impersonate/"+targetID.String(), nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, asOwnSession(t, asAdmin(t, req, eng, ownerID), ownerID))
	assert.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
}

func TestAdminCreateServiceAccountAPIKey_RefusesAgentSession(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "agent-sa-owner@test.com", "SecureP@ss123")
	ownerID := userIDFor(t, eng, ownerToken)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)
	env, err := eng.Store().GetDefaultEnvironment(context.Background(), appID)
	require.NoError(t, err)
	sa, err := eng.CreateServiceAccountInEnvironment(context.Background(), appID, env.ID, "ci", "", nil)
	require.NoError(t, err)

	path := "/v1/admin/service-accounts/" + sa.ID.String() + "/api-keys"
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, jsonBody(t, map[string]any{"name": "agent key"}))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, asAgentOf(t, req, ownerID))

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	keys, err := eng.APIKeyStore().ListAPIKeysByApp(context.Background(), appID)
	require.NoError(t, err)
	assert.Empty(t, keys, "no key is minted")

	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, jsonBody(t, map[string]any{"name": "owner key"}))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, asOwnSession(t, asAdmin(t, req, eng, ownerID), ownerID))
	assert.Less(t, rec.Code, 300, "the owner's own session still mints one; body=%s", rec.Body.String())
}

func TestChangePassword_RefusesAgentSession(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, token, _ := signUp(t, eng, "agent-pw@test.com", "SecureP@ss123")
	userID := userIDFor(t, eng, token)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	body := map[string]string{"current_password": "SecureP@ss123", "new_password": "AgentKn0ws!Pass"}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/change-password", jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, asAgentOf(t, req, userID))

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	_, _, err = eng.SignIn(context.Background(), &account.SignInRequest{AppID: appID, Email: "agent-pw@test.com", Password: "SecureP@ss123"})
	require.NoError(t, err, "the old password still signs in")

	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/change-password", jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, asOwnSession(t, asAdmin(t, req, eng, userID), userID))
	assert.Equal(t, http.StatusOK, rec.Code, "the user's own session still changes it; body=%s", rec.Body.String())
}

// Creating a user with a password the agent picked gives it a login of its
// own that no grant bounds and that survives revoking the agent.
func TestAdminCreateUser_RefusesAgentSession(t *testing.T) {
	a, eng := newBootstrappedAPI(t)
	handler := withTestKey(a.Handler())
	_, ownerToken, _ := signUp(t, eng, "agent-create-owner@test.com", "SecureP@ss123")
	ownerID := userIDFor(t, eng, ownerToken)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	body := map[string]string{"email": "agent-made@test.com", "password": "AgentKn0ws!Pass", "first_name": "Made"}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/admin/users/create", jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, asAgentOf(t, req, ownerID))

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	_, _, err = eng.SignIn(context.Background(), &account.SignInRequest{AppID: appID, Email: "agent-made@test.com", Password: "AgentKn0ws!Pass"})
	require.Error(t, err, "no account exists to sign in to")

	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/admin/users/create", jsonBody(t, body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, asOwnSession(t, asAdmin(t, req, eng, ownerID), ownerID))
	assert.Less(t, rec.Code, 300, "the owner's own session still creates one; body=%s", rec.Body.String())
}
