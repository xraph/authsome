package authsome_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/serviceaccount"
	"github.com/xraph/authsome/user"
)

// Exchange without a grant must fail. The endpoint exercises authority
// somebody already gave; it has no path that creates any.
func TestExchangeWithoutGrantIsRefused(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)

	_, err := e.ExchangeToken(context.Background(), &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
	})
	assert.Error(t, err, "no grant means no exchange")
}

func TestExchangeMintsADelegatedSession(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	_, err := e.GrantDelegation(ctx, appID, agent, userRef,
		[]string{"repo:read"}, userRef, nil)
	require.NoError(t, err)

	sess, err := e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
		Scopes: []string{"repo:read"},
	})
	require.NoError(t, err)

	// The subject is the human. This is what keeps session.UserID meaning
	// "the person this request is for" for every existing consumer.
	assert.Equal(t, userRef.ID, sess.UserID.String())
	assert.Equal(t, principal.KindUser, sess.PrincipalKind)
	assert.Equal(t, principal.Chain{agent}, sess.Actors)
	assert.Equal(t, principal.GrantDelegation, sess.ActorGrant)
	assert.False(t, sess.DelegationID.IsNil())
	assert.True(t, sess.ImpersonatedBy().IsNil(), "a delegation is not impersonation")
	assert.Equal(t, []string{"repo:read"}, sess.Scopes)
}

// A scope the grant does not carry must not survive the exchange. This is
// where the spec's scope filter is enforced.
func TestExchangeIntersectsScopes(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	_, err := e.GrantDelegation(ctx, appID, agent, userRef, []string{"repo:read"}, userRef, nil)
	require.NoError(t, err)

	_, err = e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
		Scopes: []string{"repo:write"},
	})
	assert.Error(t, err, "a scope outside the grant must be refused, not silently dropped")
}

// A revoked grant stops working immediately, which is the entire point of
// storing grants rather than asserting the chain per request.
func TestExchangeRefusesRevokedGrant(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	d, err := e.GrantDelegation(ctx, appID, agent, userRef, nil, userRef, nil)
	require.NoError(t, err)
	require.NoError(t, e.RevokeDelegation(ctx, d.ID))

	_, err = e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
	})
	assert.Error(t, err)
}

// The session's lifetime cannot outlive the grant's. A one-hour grant that
// mints a thirty-day session is a grant nobody actually revoked.
func TestExchangeBoundsSessionTTLByGrantExpiry(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	soon := time.Now().Add(5 * time.Minute)
	_, err := e.GrantDelegation(ctx, appID, agent, userRef, nil, userRef, &soon)
	require.NoError(t, err)

	sess, err := e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
	})
	require.NoError(t, err)
	assert.False(t, sess.ExpiresAt.After(soon), "the session must not outlive the grant")
}

// A grant naming a different actor than the caller must not be found: the
// (app, actor, subject) triple is the whole key, not just (app, subject).
func TestExchangeRefusesForWrongActor(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	otherAgent := newExchangeAgentRef(t, e, appID)

	_, err := e.GrantDelegation(ctx, appID, otherAgent, userRef, nil, userRef, nil)
	require.NoError(t, err)

	_, err = e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
	})
	assert.Error(t, err, "a grant naming a different actor must not authorize this one")
}

// ──────────────────────────────────────────────────
// fixtures
// ──────────────────────────────────────────────────

// setupExchangeFixture builds an engine with a real user (the delegation
// subject) and a real agent-kind service account (the delegation actor), both
// seeded under the app ID Can/ExchangeToken's checks actually resolve to.
//
// newTestEngine wires WithAppID but never a bootstrap config, so
// testTenantID (engine_principal_test.go) reads the tenant back off
// eng.Config().AppID rather than trusting Engine.PlatformAppID, which stays
// the zero value here.
func setupExchangeFixture(t *testing.T) (eng *authsome.Engine, appID id.AppID, agentRef, userRef principal.Ref) {
	t.Helper()
	eng, s := newTestEngine(t)
	appID = testTenantID(t, eng)
	ctx := context.Background()

	userID := id.NewUserID()
	require.NoError(t, s.CreateUser(ctx, &user.User{
		ID:           userID,
		AppID:        appID,
		Email:        "exchange-subject@example.com",
		FirstName:    "Exchange",
		Username:     "exchange-subject",
		PasswordHash: "$2a$10$fakehash",
	}))
	userRef = principal.UserRef(userID)

	agentRef = newExchangeAgentRef(t, eng, appID)

	return eng, appID, agentRef, userRef
}

// newExchangeAgentRef seeds a fresh agent-kind service account under appID
// and returns its ref. CreateServiceAccount (the exported engine method)
// has no way to set Kind, so this writes the row through the store directly,
// the same way engine_principal_test.go reaches into Warden's store for its
// own fixtures.
func newExchangeAgentRef(t *testing.T, eng *authsome.Engine, appID id.AppID) principal.Ref {
	t.Helper()
	store, ok := eng.PrincipalStore().(interface {
		CreateServiceAccount(ctx context.Context, svc *serviceaccount.ServiceAccount) error
	})
	require.True(t, ok, "engine store must support CreateServiceAccount")

	svcID := id.NewServiceAccountID()
	svc := &serviceaccount.ServiceAccount{
		ID: svcID,
		// The store enforces a unique (app_id, name); the ID is already
		// unique, so folding it into the name lets this fixture be called
		// more than once per app without a name collision.
		Name:      "exchange-agent-" + svcID.String(),
		AppID:     appID,
		Kind:      principal.KindAgent,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateServiceAccount(context.Background(), svc))
	return principal.Ref{Kind: principal.KindAgent, ID: svc.ID.String()}
}
