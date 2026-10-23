package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/tokenformat"
)

// An agent token introspected as the user_id of its delegating user, with
// nothing to say an agent held it. A resource server, or a dashboard checking
// tokens remotely, took it for the user's own sign-in. Introspection now says
// what kind of caller holds the token and whether it acts for user_id without
// being that user, and says so to an unprivileged caller too, since that is
// exactly who enforces it.
func TestIntrospect_ReportsPrincipalKindAndDelegation(t *testing.T) {
	eng := newIntrospectAudienceEngine(t, nil)
	router := newAPIWithRouter(t, eng)
	ctx := context.Background()

	human := signUpForAudience(t, eng, "intro-pk@example.com")
	humanSess, err := eng.Store().GetSessionByToken(ctx, human)
	require.NoError(t, err)

	mint := func(mut func(*session.Session)) string {
		t.Helper()
		s, err := account.NewSession(humanSess.AppID, humanSess.UserID, account.SessionConfig{
			TokenTTL: 15 * time.Minute, RefreshTokenTTL: 15 * time.Minute,
		})
		require.NoError(t, err)
		s.EnvID = humanSess.EnvID
		mut(s)
		require.NoError(t, eng.Store().CreateSession(ctx, s))
		return s.Token
	}
	agent := mint(func(s *session.Session) {
		s.PrincipalKind = session.PrincipalKindAgent
		s.AgentID = id.NewAgentID()
		s.GrantID = id.NewAgentGrantID()
	})
	exchanged := mint(func(s *session.Session) {
		s.Actors = principal.Chain{{Kind: principal.KindWorkload, ID: "svc_actor"}}
		s.ActorGrant = principal.GrantDelegation
	})

	got := introspect(t, router, eng, human)
	assert.Equal(t, "user", got["principal_kind"])
	assert.NotContains(t, got, "delegated", "the user's own token is not delegated")

	got = introspect(t, router, eng, agent)
	assert.Equal(t, true, got["active"])
	assert.Equal(t, "agent", got["principal_kind"])
	assert.Equal(t, true, got["delegated"])

	got = introspect(t, router, eng, exchanged)
	assert.Equal(t, "user", got["principal_kind"])
	assert.Equal(t, true, got["delegated"])
}

func TestIntrospect_JWTReportsPrincipalKind(t *testing.T) {
	jwtFmt := newHMACJWTFormat(t)
	eng := newIntrospectAudienceEngine(t, jwtFmt)
	router := newAPIWithRouter(t, eng)

	sign := func(pk string) string {
		t.Helper()
		token, err := jwtFmt.GenerateAccessToken(tokenformat.TokenClaims{
			UserID:        id.NewUserID().String(),
			AppID:         testAppIDStr,
			SessionID:     id.NewSessionID().String(),
			PrincipalKind: pk,
			IssuedAt:      time.Now(),
			ExpiresAt:     time.Now().Add(time.Hour),
		})
		require.NoError(t, err)
		return token
	}

	got := introspect(t, router, eng, sign(""))
	assert.Equal(t, "user", got["principal_kind"])
	assert.NotContains(t, got, "delegated")

	got = introspect(t, router, eng, sign("agent"))
	assert.Equal(t, "agent", got["principal_kind"])
	assert.Equal(t, true, got["delegated"])
}
