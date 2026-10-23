package extension

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	authclient "github.com/xraph/authsome/sdk/go"
	"github.com/xraph/authsome/session"
)

// The engine-mode checker resolved a token to its session and the session's
// UserID to a user. An agent session carries its delegating user's UserID, so
// an agent token opened the dashboard as that user, and the dashboard can mint
// API keys for anyone. A session acting for a user it is not gets no identity.
func TestAuthChecker_AgentSessionGetsNoIdentity(t *testing.T) {
	checker, humanToken := newDashEngineChecker(t, "", "dash-agent@example.com")
	ctx := context.Background()

	human, err := checker.engine.ResolveSessionByToken(ctx, humanToken)
	require.NoError(t, err)

	agent, err := account.NewSession(human.AppID, human.UserID, account.SessionConfig{
		TokenTTL: 15 * time.Minute, RefreshTokenTTL: 15 * time.Minute,
	})
	require.NoError(t, err)
	agent.EnvID = human.EnvID
	agent.PrincipalKind = session.PrincipalKindAgent
	agent.AgentID = id.NewAgentID()
	agent.GrantID = id.NewAgentGrantID()
	require.NoError(t, checker.engine.Store().CreateSession(ctx, agent))

	info, err := checker.CheckAuth(ctx, dashCheckerRequest(t, agent.Token, ""))
	require.NoError(t, err)
	assert.Nil(t, info, "an agent token must not open the dashboard as its user")

	info, err = checker.CheckAuth(ctx, dashCheckerRequest(t, humanToken, ""))
	require.NoError(t, err)
	require.NotNil(t, info, "the user's own token still does")
	assert.Equal(t, human.UserID.String(), info.Subject)
}

// Client mode checks the token by introspection. The response names the
// delegating user, so the checker reads principal_kind and delegated and
// gives an agent or delegated token no identity. A server that predates the
// fields sends neither, and its tokens resolve as before.
func TestClientAuthChecker_DelegatedTokenGetsNoIdentity(t *testing.T) {
	cases := map[string]struct {
		kind      string
		delegated bool
		want      bool
	}{
		"user's own":           {"user", false, true},
		"older server":         {"", false, true},
		"agent":                {"agent", true, false},
		"token exchange":       {"user", true, false},
		"agent, no delegation": {"agent", false, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			userID := id.NewUserID().String()
			identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				resp := authclient.IntrospectResponse{
					Active: true, UserID: userID, AppID: id.NewAppID().String(), SessionID: id.NewSessionID().String(),
					User:          &authclient.IntrospectUser{ID: userID, Email: "dash@test.com"},
					PrincipalKind: tc.kind, Delegated: tc.delegated,
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			}))
			t.Cleanup(identity.Close)
			c := &clientAuthChecker{
				client:  authclient.NewClient(identity.URL),
				binding: middleware.SessionBindingConfig{DPoPValidator: dpop.NewValidator(dpop.Config{})},
				logger:  log.NewNoopLogger(),
			}

			info, err := c.CheckAuth(context.Background(), dashCheckerRequest(t, "dash-token", ""))
			require.NoError(t, err)
			assert.Equal(t, tc.want, info != nil)
		})
	}
}
