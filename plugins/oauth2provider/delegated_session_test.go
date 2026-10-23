package oauth2provider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/session"
)

// An agent session carries its delegating user's UserID, so every OAuth2
// route that reads the signed-in user would treat it as that user. Approving
// a client, in any of the three ways this plugin offers, hands the client a
// token for the user with no pk claim and no grant behind it, which the
// middleware later reads as the user's own login. Each of them refuses an
// agent session before anything is issued.

// agentCtx puts an agent session for userID in appID on a fresh context, the
// way the auth middleware does for a resolved agent token: the session, its
// app and the delegating user's ID, but no user record.
func agentCtx(userID id.UserID, appID id.AppID) context.Context {
	sess := &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: userID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	}
	ctx := middleware.WithSession(context.Background(), sess)
	ctx = middleware.WithSessionID(ctx, sess.ID)
	ctx = middleware.WithAppID(ctx, appID)
	return middleware.WithUserID(ctx, userID)
}

func TestAuthorize_RefusesAgentSession(t *testing.T) {
	_, st, mux := newFixture(t)
	client, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)

	ctx := agentCtx(id.NewUserID(), client.AppID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/oauth/authorize?"+baseAuthorizeQuery(confidentialID).Encode(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	assert.NotContains(t, rec.Header().Get("Location"), "code=", "no authorization code may be issued")
}

func TestAuthorize_HumanSessionStillIssuesCode(t *testing.T) {
	_, st, mux := newFixture(t)
	client, err := st.GetClient(context.Background(), confidentialID)
	require.NoError(t, err)

	userID := id.NewUserID()
	ctx := middleware.WithSession(context.Background(), &session.Session{ID: id.NewSessionID(), AppID: client.AppID, UserID: userID})
	ctx = middleware.WithAppID(ctx, client.AppID)
	ctx = middleware.WithUserID(ctx, userID)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/oauth/authorize?"+baseAuthorizeQuery(confidentialID).Encode(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.NotEmpty(t, codeFrom(t, rec))
}

// The user starts a third-party authorization and the agent tries to approve
// it on the consent page with a CSRF token it read off that page.
func TestConsentDecision_RefusesAgentSession(t *testing.T) {
	st, mux, appID := newConsentFixture(t)
	user := id.NewUserID()

	cid := consentIDFrom(t, authorizeAs(t, mux, user, consentQuery("openid profile")))
	page := getConsent(t, mux, user, cid, "")
	require.Equal(t, http.StatusOK, page.Code, page.Body.String())
	csrf := csrfFromPage(t, page.Body.String())

	form := url.Values{"cid": {cid}, "csrf": {csrf}, "decision": {"approve"}}
	req := httptest.NewRequestWithContext(agentCtx(user, appID), http.MethodPost, "/v1/oauth/consent", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	assert.NotContains(t, rec.Header().Get("Location"), "code=", "no authorization code may be issued")
	_, err := st.GetGrant(context.Background(), appID, user, thirdPartyID)
	assert.ErrorIs(t, err, oauth2provider.ErrGrantNotFound, "the agent's approval records nothing")

	// The user can still approve it.
	approved := decide(t, mux, user, form)
	require.Equal(t, http.StatusFound, approved.Code, approved.Body.String())
}

func TestConsentPage_RefusesAgentSession(t *testing.T) {
	_, mux, appID := newConsentFixture(t)
	user := id.NewUserID()

	cid := consentIDFrom(t, authorizeAs(t, mux, user, consentQuery("openid profile")))
	req := httptest.NewRequestWithContext(agentCtx(user, appID), http.MethodGet, "/v1/oauth/consent?cid="+cid, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	assert.NotContains(t, rec.Body.String(), `name="csrf"`, "the agent never sees the consent token")
}

func TestDeviceComplete_RefusesAgentSession(t *testing.T) {
	_, st, mux := newFixture(t)
	registerDeviceClients(t, st)
	client, err := st.GetClient(context.Background(), devicePublicID)
	require.NoError(t, err)

	auth := deviceAuthorize(t, mux, devicePublicID)
	body, err := json.Marshal(map[string]string{"user_code": auth.UserCode, "action": "approve"})
	require.NoError(t, err)
	req := httptest.NewRequestWithContext(agentCtx(id.NewUserID(), client.AppID), http.MethodPost, "/v1/oauth/device/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	dc, err := st.GetDeviceCodeByDeviceCode(context.Background(), auth.DeviceCode)
	require.NoError(t, err)
	assert.Equal(t, oauth2provider.DeviceCodeStatusPending, dc.Status, "the device code stays pending")
	assert.True(t, dc.UserID.IsNil(), "no user is bound to the device code")

	// A person signed in as themselves still approves it.
	human := deviceComplete(t, mux, auth.UserCode, "approve")
	require.Equal(t, http.StatusOK, human.Code, "body=%s", human.Body.String())
}

// Creating a client or rotating its secret hands out a client credential that
// no grant bounds, so an agent acting for an admin cannot do either.
func TestAdminClients_RefuseAgentSession(t *testing.T) {
	_, st, mux := newFixture(t)
	appID := id.NewAppID()
	own := seedClient(t, st, appID, "agent-target")

	post := func(path string, body any) *httptest.ResponseRecorder {
		var b []byte
		if body != nil {
			var err error
			b, err = json.Marshal(body)
			require.NoError(t, err)
		}
		req := httptest.NewRequestWithContext(agentCtx(id.NewUserID(), appID), http.MethodPost, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		return serve(mux, req)
	}

	created := post("/v1/admin/oauth/clients", map[string]any{
		"app_id": appID.String(), "name": "Agent's client", "redirect_uris": []string{registeredURI},
	})
	assert.Equal(t, http.StatusForbidden, created.Code, "body=%s", created.Body.String())
	clients, err := st.ListClients(context.Background(), appID)
	require.NoError(t, err)
	assert.Len(t, clients, 1, "no client is created")

	rotated := post("/v1/admin/oauth/clients/"+own.ID.String()+"/secret", nil)
	assert.Equal(t, http.StatusForbidden, rotated.Code, "body=%s", rotated.Body.String())
	after, err := st.GetClient(context.Background(), own.ClientID)
	require.NoError(t, err)
	assert.Equal(t, own.ClientSecret, after.ClientSecret, "the secret is not rotated")

	// The admin's own session still creates one.
	human := serve(mux, adminReq(t, http.MethodPost, "/v1/admin/oauth/clients", &appID, map[string]any{
		"app_id": appID.String(), "name": "Admin's client", "redirect_uris": []string{registeredURI},
	}))
	require.Less(t, human.Code, 300, "body=%s", human.Body.String())
}
