package apikey_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// An agent session carries its delegating user's UserID, so the create
// handler would read it as that user. A key minted that way belongs to the
// user, is bound to no grant and outlives it, so the route refuses any session
// acting for a user it is not.
func TestPlugin_CreateKey_RefusesDelegatedSessions(t *testing.T) {
	cases := map[string]func(userID id.UserID, appID id.AppID) *session.Session{
		"agent": func(userID id.UserID, appID id.AppID) *session.Session {
			return &session.Session{
				ID: id.NewSessionID(), AppID: appID, UserID: userID,
				PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
			}
		},
		"token exchange": func(userID id.UserID, appID id.AppID) *session.Session {
			return &session.Session{
				ID: id.NewSessionID(), AppID: appID, UserID: userID,
				Actors:     principal.Chain{{Kind: principal.KindWorkload, ID: "svc_actor"}},
				ActorGrant: principal.GrantDelegation,
			}
		},
	}

	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			p, keyStore := newTestPlugin()
			mux := forge.NewRouter()
			require.NoError(t, p.RegisterRoutes(mux))

			userID, appID := id.NewUserID(), id.NewAppID()
			body, err := json.Marshal(map[string]any{"app_id": appID.String(), "name": "laundered"})
			require.NoError(t, err)
			ctx := middleware.WithUserID(context.Background(), userID)
			ctx = middleware.WithSession(ctx, build(userID, appID))
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/keys", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code, "body=%s", w.Body.String())
			keys, err := keyStore.ListAPIKeysByUser(context.Background(), appID, userID)
			require.NoError(t, err)
			assert.Empty(t, keys, "nothing may be minted for the user")
		})
	}
}

// The user's own session still mints a key.
func TestPlugin_CreateKey_HumanSessionStillWorks(t *testing.T) {
	p, keyStore := newTestPlugin()
	mux := forge.NewRouter()
	require.NoError(t, p.RegisterRoutes(mux))

	userID, appID := id.NewUserID(), id.NewAppID()
	body, err := json.Marshal(map[string]any{"app_id": appID.String(), "name": "mine"})
	require.NoError(t, err)
	ctx := middleware.WithUserID(context.Background(), userID)
	ctx = middleware.WithSession(ctx, &session.Session{ID: id.NewSessionID(), AppID: appID, UserID: userID})
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/v1/keys", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "body=%s", w.Body.String())
	keys, err := keyStore.ListAPIKeysByUser(context.Background(), appID, userID)
	require.NoError(t, err)
	assert.Len(t, keys, 1)
}
