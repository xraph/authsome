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
		s, mintErr := account.NewSession(humanSess.AppID, humanSess.UserID, account.SessionConfig{
			TokenTTL: 15 * time.Minute, RefreshTokenTTL: 15 * time.Minute,
		})
		require.NoError(t, mintErr)
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

	// A service account registered with the agent kind, acting as itself.
	svc, err := account.NewSession(humanSess.AppID, id.UserID{}, account.SessionConfig{
		TokenTTL: 15 * time.Minute, RefreshTokenTTL: 15 * time.Minute,
	})
	require.NoError(t, err)
	svc.EnvID = humanSess.EnvID
	svc.PrincipalKind = session.PrincipalKindAgent
	svc.ServiceAccountID = id.NewServiceAccountID()
	require.NoError(t, eng.Store().CreateSession(ctx, svc))
	got = introspect(t, router, eng, svc.Token)
	assert.Equal(t, "agent", got["principal_kind"])
	assert.NotContains(t, got, "delegated", "a principal acting as itself is not delegated")
}

func TestIntrospect_JWTReportsPrincipalKind(t *testing.T) {
	jwtFmt := newHMACJWTFormat(t)
	eng := newIntrospectAudienceEngine(t, jwtFmt)
	router := newAPIWithRouter(t, eng)

	signAs := func(pk, sub string, act *tokenformat.ActClaim) string {
		t.Helper()
		pid := ""
		if pk != "" {
			pid = sub
		}
		token, err := jwtFmt.GenerateAccessToken(tokenformat.TokenClaims{
			UserID:        sub,
			AppID:         testAppIDStr,
			SessionID:     id.NewSessionID().String(),
			PrincipalKind: pk,
			PrincipalID:   pid,
			Act:           act,
			IssuedAt:      time.Now(),
			ExpiresAt:     time.Now().Add(time.Hour),
		})
		require.NoError(t, err)
		return token
	}
	sign := func(pk string) string { return signAs(pk, id.NewUserID().String(), nil) }

	got := introspect(t, router, eng, sign(""))
	assert.Equal(t, "user", got["principal_kind"])
	assert.NotContains(t, got, "delegated")

	got = introspect(t, router, eng, sign("agent"))
	assert.Equal(t, "agent", got["principal_kind"])
	assert.Equal(t, true, got["delegated"], "an agent token whose sub is a user acts for that user")

	// A machine token's sub is the caller itself. It acts for nobody.
	for _, pk := range []string{"service_account", "workload", "agent"} {
		got = introspect(t, router, eng, signAs(pk, id.NewServiceAccountID().String(), nil))
		assert.Equal(t, pk, got["principal_kind"])
		assert.NotContains(t, got, "delegated", "%s acting as itself", pk)
	}

	// An act claim is RFC 8693's way of saying someone else is calling.
	got = introspect(t, router, eng, signAs("", id.NewUserID().String(), &tokenformat.ActClaim{Subject: id.NewServiceAccountID().String()}))
	assert.Equal(t, "user", got["principal_kind"])
	assert.Equal(t, true, got["delegated"])
}
