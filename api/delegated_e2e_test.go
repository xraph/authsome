package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/serviceaccount"
)

// The other tests in this area put a session on the context by hand. This
// one mints real tokens, an agent session through agentauth and a delegated
// session through token exchange, and sends them through the engine's own
// auth middleware, so what the guard sees is what production resolves.
func TestDelegatedTokens_EndToEndThroughAuthMiddleware(t *testing.T) {
	agents := agentauth.New(agentauth.WithStore(agentauth.NewMemoryStore()))
	a, eng := newBootstrappedAPI(t, authsome.WithPlugin(agents))
	handler := realChainHandler(t, a, eng)
	ctx := context.Background()

	_, ownerToken, _ := signUp(t, eng, "e2e-owner@test.com", "SecureP@ss123")
	owner := userIDFor(t, eng, ownerToken)
	_, targetToken, _ := signUp(t, eng, "e2e-target@test.com", "SecureP@ss123")
	target := userIDFor(t, eng, targetToken)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	grant := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: appID, AgentID: id.NewAgentID(), UserID: owner, OrgID: id.NewOrgID(),
		Scopes: []string{"invoices:read"}, ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, agents.Store().CreateAgentGrant(ctx, grant))
	agentSess, err := agents.IssueAgentSession(ctx, grant, agentauth.IssueMeta{})
	require.NoError(t, err)

	svcStore, ok := eng.PrincipalStore().(interface {
		CreateServiceAccount(ctx context.Context, svc *serviceaccount.ServiceAccount) error
	})
	require.True(t, ok)
	svc := &serviceaccount.ServiceAccount{
		ID: id.NewServiceAccountID(), Name: "e2e-actor", AppID: appID, Kind: principal.KindAgent,
		Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, svcStore.CreateServiceAccount(ctx, svc))
	actor := principal.Ref{Kind: principal.KindAgent, ID: svc.ID.String()}
	_, err = eng.GrantDelegation(ctx, appID, actor, principal.UserRef(owner), nil, principal.UserRef(owner), nil)
	require.NoError(t, err)
	exchanged, err := eng.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: actor, RequestedSubject: principal.UserRef(owner),
	})
	require.NoError(t, err)

	call := func(token, method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		var req *http.Request
		if body == "" {
			req = httptest.NewRequestWithContext(ctx, method, path, nil)
		} else {
			req = httptest.NewRequestWithContext(ctx, method, path, jsonBody(t, rawJSON(body)))
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(middleware.PublishableKeyHeader, testPublishableKey)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	for name, token := range map[string]string{"agent": agentSess.Token, "token exchange": exchanged.Token} {
		t.Run(name, func(t *testing.T) {
			me := call(token, http.MethodGet, "/v1/me", "")
			assert.Equal(t, http.StatusOK, me.Code, "a read stays open; body=%s", me.Body.String())

			pw := call(token, http.MethodPost, "/v1/change-password",
				`{"current_password":"SecureP@ss123","new_password":"AgentKn0ws!Pass"}`)
			assert.Equal(t, http.StatusForbidden, pw.Code, "body=%s", pw.Body.String())
			assert.Contains(t, pw.Body.String(), "delegated session")
			_, _, err := eng.SignIn(ctx, &account.SignInRequest{AppID: appID, Email: "e2e-owner@test.com", Password: "SecureP@ss123"})
			require.NoError(t, err, "the password is unchanged")

			ban := call(token, http.MethodPost, "/v1/admin/users/"+target.String()+"/ban", `{"reason":"agent"}`)
			assert.Equal(t, http.StatusForbidden, ban.Code, "body=%s", ban.Body.String())
			u, err := eng.AdminGetUser(ctx, target)
			require.NoError(t, err)
			assert.False(t, u.Banned, "nobody is banned")
		})
	}

	// The owner's own token through the same chain still bans.
	ban := call(ownerToken, http.MethodPost, "/v1/admin/users/"+target.String()+"/ban", `{"reason":"owner"}`)
	assert.Less(t, ban.Code, 300, "body=%s", ban.Body.String())
}

// rawJSON lets call pass a literal body through jsonBody unchanged.
type rawJSON string

func (r rawJSON) MarshalJSON() ([]byte, error) { return []byte(r), nil }
