package api_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// Engine.Impersonate and Engine.ChangePassword refuse an agent session on the
// context too, for a caller that reaches them without the HTTP routes.
func TestEngine_RefusesAgentSessionOnContext(t *testing.T) {
	_, eng := newBootstrappedAPI(t)
	_, ownerToken, _ := signUp(t, eng, "agent-eng-owner@test.com", "SecureP@ss123")
	ownerID := userIDFor(t, eng, ownerToken)
	_, targetToken, _ := signUp(t, eng, "agent-eng-target@test.com", "SecureP@ss123")
	targetID := userIDFor(t, eng, targetToken)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	agent := &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: ownerID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	}
	ctx := middleware.WithSession(context.Background(), agent)

	_, _, err = eng.Impersonate(ctx, ownerID, targetID)
	require.ErrorIs(t, err, authsome.ErrDelegatedSession)

	err = eng.ChangePassword(ctx, ownerID, "SecureP@ss123", "AgentKn0ws!Pass")
	require.ErrorIs(t, err, authsome.ErrDelegatedSession)

	// Without a delegated session on the context both still work.
	_, _, err = eng.Impersonate(context.Background(), ownerID, targetID)
	require.NoError(t, err)
	require.NoError(t, eng.ChangePassword(context.Background(), ownerID, "SecureP@ss123", "Own3rChanged!Pass"))
}

// AdminCreateUser writes the account itself, so it refuses a delegated
// session on the context too.
func TestEngine_AdminCreateUserRefusesAgentSession(t *testing.T) {
	_, eng := newBootstrappedAPI(t)
	_, ownerToken, _ := signUp(t, eng, "agent-eng-create-owner@test.com", "SecureP@ss123")
	ownerID := userIDFor(t, eng, ownerToken)
	appID, err := id.ParseAppID(testAppIDStr)
	require.NoError(t, err)

	ctx := middleware.WithSession(context.Background(), &session.Session{
		ID: id.NewSessionID(), AppID: appID, UserID: ownerID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: id.NewAgentID(), GrantID: id.NewAgentGrantID(),
	})
	_, err = eng.AdminCreateUser(ctx, ownerID, appID, id.EnvironmentID{}, "eng-agent-made@test.com", "AgentKn0ws!Pass", "Made", "", "")
	require.ErrorIs(t, err, authsome.ErrDelegatedSession)

	_, err = eng.AdminCreateUser(context.Background(), ownerID, appID, id.EnvironmentID{}, "eng-owner-made@test.com", "Own3rMade!Pass", "Made", "", "")
	require.NoError(t, err)
}
