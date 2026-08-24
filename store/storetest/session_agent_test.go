package storetest_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

// An agent session keeps the delegating human in UserID. That is what lets
// DeleteUserSessions, audit records and org resolution keep working with no
// new branches, and it is the whole reason agentauth does not copy the
// service-account shape.
func TestSession_AgentPrincipal_RoundTrips(t *testing.T) {
	st := memory.New()
	ctx := context.Background()
	userID := id.NewUserID()
	agentID := id.NewAgentID()
	grantID := id.NewAgentGrantID()

	s := &session.Session{
		ID:            id.NewSessionID(),
		AppID:         id.NewAppID(),
		UserID:        userID,
		PrincipalKind: session.PrincipalKindAgent,
		AgentID:       agentID,
		GrantID:       grantID,
		Token:         "tok_agent_roundtrip",
		ExpiresAt:     time.Now().Add(time.Hour),
		CreatedAt:     time.Now(),
	}
	require.NoError(t, st.CreateSession(ctx, s))

	got, err := st.GetSessionByToken(ctx, "tok_agent_roundtrip")
	require.NoError(t, err)
	assert.Equal(t, session.PrincipalKindAgent, got.PrincipalKind)
	assert.Equal(t, userID.String(), got.UserID.String(), "the delegating human must survive the round trip")
	assert.Equal(t, agentID.String(), got.AgentID.String())
	assert.Equal(t, grantID.String(), got.GrantID.String())
}

// Offboarding leans entirely on this. DeleteUserSessions already exists and is
// already called on user deletion, so an agent session carrying the delegating
// user's id is swept up with no change to that code path.
func TestSession_DeleteUserSessions_SweepsAgentSessions(t *testing.T) {
	st := memory.New()
	ctx := context.Background()
	userID := id.NewUserID()

	human := &session.Session{
		ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: userID,
		Token: "tok_human", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}
	agent := &session.Session{
		ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: userID,
		PrincipalKind: session.PrincipalKindAgent,
		AgentID:       id.NewAgentID(), GrantID: id.NewAgentGrantID(),
		Token: "tok_agent", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}
	require.NoError(t, st.CreateSession(ctx, human))
	require.NoError(t, st.CreateSession(ctx, agent))

	require.NoError(t, st.DeleteUserSessions(ctx, userID))

	_, err := st.GetSessionByToken(ctx, "tok_agent")
	require.Error(t, err, "an agent session must die with its delegating user")
}
