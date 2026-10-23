package extension

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
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
