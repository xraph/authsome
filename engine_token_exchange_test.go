package authsome_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/serviceaccount"
	"github.com/xraph/authsome/tokenformat"
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
	require.NoError(t, e.RevokeDelegation(ctx, appID, userRef, d.ID))

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

// A chained exchange must not be able to launder authority from one grant
// into another. The agent holds a repo:read grant from Alice; Alice
// separately holds a grant over Bob. Exchanging the agent's credential for
// an Alice session, then presenting THAT session back to this same
// endpoint, must not land a session acting for Bob: Bob never named the
// agent, Alice's own scope filter would be gone, and because ExchangeToken
// assigns rather than appends to sess.Actors, the agent would vanish from
// the resulting session's chain entirely, leaving an audit trail that reads
// as a user acting for a user with no agent in it anywhere.
func TestExchangeRefusesChainedExchange(t *testing.T) {
	e, appID, agent, alice := setupExchangeFixture(t)
	ctx := context.Background()
	bob := newExchangeUserRef(t, e, appID)

	_, err := e.GrantDelegation(ctx, appID, agent, alice, []string{"repo:read"}, alice, nil)
	require.NoError(t, err)
	_, err = e.GrantDelegation(ctx, appID, alice, bob, nil, bob, nil)
	require.NoError(t, err)

	aliceSess, err := e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: alice,
		Scopes: []string{"repo:read"},
	})
	require.NoError(t, err)
	require.Equal(t, principal.Chain{agent}, aliceSess.Actors,
		"sanity: the exchanged session must actually carry the agent, or this test proves nothing")

	_, err = e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: alice, RequestedSubject: bob,
		CallerActors:       aliceSess.Actors,
		CallerDelegationID: aliceSess.DelegationID,
	})
	assert.Error(t, err, "a session minted by a prior exchange must not itself be exchangeable")
}

// An impersonation session carries a populated actor chain the same way an
// exchanged one does (Session.Actors, see SetImpersonatedBy), so it must be
// refused here too: an admin impersonating a user must not be able to spend
// that session exchanging for some third party the impersonated user holds
// a grant over.
func TestExchangeRefusesImpersonationSession(t *testing.T) {
	e, appID, agent, alice := setupExchangeFixture(t)
	ctx := context.Background()

	_, err := e.GrantDelegation(ctx, appID, agent, alice, nil, alice, nil)
	require.NoError(t, err)

	admin := newExchangeUserRef(t, e, appID)
	_, err = e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: alice,
		// Shape SetImpersonatedBy leaves on a caller's own session: a
		// single-user actor chain, no delegation id.
		CallerActors: principal.Chain{admin},
	})
	assert.Error(t, err, "a session that is itself the result of impersonation must not be exchangeable")
}

// An exchanged credential must be opaque even when the app is configured to
// issue JWTs, and the session row must still be clamped to the grant.
//
// tokenformat.TokenClaims carries no RFC 8693 act claim, and
// AuthMiddlewareWithJWT tries JWT validation first and returns on success, so
// a JWT-format exchanged token would never cause the session row to be loaded
// and the actor chain would not exist on the request at all. Opaque is what
// keeps the chain reachable. See newOpaqueSession.
func TestExchangeIssuesAnOpaqueTokenOnAJWTApp(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)

	e, appID, agent, userRef := setupExchangeFixture(t,
		authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt))
	ctx := context.Background()

	soon := time.Now().Add(45 * time.Second)
	_, err = e.GrantDelegation(ctx, appID, agent, userRef, nil, userRef, &soon)
	require.NoError(t, err)

	sess, err := e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
	})
	require.NoError(t, err)

	assert.False(t, tokenformat.IsJWT(sess.Token),
		"an exchanged token must be opaque even on a JWT app, or the actor chain never reaches the request")
	assert.False(t, sess.ExpiresAt.After(soon),
		"the session must still not outlive the grant")
}

// The chained-exchange refusal must hold on a JWT-configured app, not only on
// an opaque one.
//
// This is the seam the refusal used to fall through. The engine-level check
// reads ExchangeRequest.CallerActors, and the HTTP handler fills that from
// middleware.SessionFrom. On a JWT app the exchanged credential used to be a
// JWT, tryJWTAuth resolved it without ever loading the session row, no session
// reached the context, and the handler passed an empty chain: the engine was
// told the caller had no chain and could not refuse what it could not see.
//
// The proof runs at the layer where the credential's format decides the
// outcome. The exchanged token must be opaque, so the middleware resolves it
// through the session path, so the row's chain is what the second exchange is
// judged on.
func TestExchangedCredentialCannotBeReExchangedOnAJWTApp(t *testing.T) {
	jwtFmt, err := tokenformat.NewJWT(tokenformat.JWTConfig{
		SigningMethod: jwt.SigningMethodHS256,
		SigningKey:    []byte("test-signing-key-0123456789abcdef"),
	})
	require.NoError(t, err)

	e, appID, agent, alice := setupExchangeFixture(t,
		authsome.WithJWTFormat("aapp_01jf0000000000000000000000", jwtFmt))
	ctx := context.Background()
	bob := newExchangeUserRef(t, e, appID)

	_, err = e.GrantDelegation(ctx, appID, agent, alice, []string{"repo:read"}, alice, nil)
	require.NoError(t, err)
	_, err = e.GrantDelegation(ctx, appID, alice, bob, nil, bob, nil)
	require.NoError(t, err)

	aliceSess, err := e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: alice,
		Scopes: []string{"repo:read"},
	})
	require.NoError(t, err)

	// The load-bearing assertion. A JWT here means the credential resolves
	// through tryJWTAuth, which loads no session row, which means the chain
	// below is not something a real request could ever have recovered.
	require.False(t, tokenformat.IsJWT(aliceSess.Token),
		"the exchanged credential must be opaque, or the chain never reaches the second exchange and the refusal below is unreachable in production")

	// What the session lookup hands the handler on the replay attempt.
	// ResolveSessionByToken is exactly what middleware.trySessionAuth calls,
	// so this is the chain a real replay would arrive carrying.
	stored, err := e.ResolveSessionByToken(aliceSess.Token)
	require.NoError(t, err, "an opaque exchanged token must resolve back to its own row")
	require.Equal(t, principal.Chain{agent}, stored.Actors,
		"the row the middleware loads must carry the chain")

	_, err = e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: alice, RequestedSubject: bob,
		CallerActors:       stored.Actors,
		CallerDelegationID: stored.DelegationID,
	})
	assert.Error(t, err,
		"a credential minted by a prior exchange must not itself be exchangeable, JWT-configured app included")
}

// An exchange that names no scopes must land on the intersection of the grant
// and the actor, never on nil.
//
// Empty scopes read as "no restriction" everywhere they are consumed, so
// returning nil for a grant limited to repo:read would hand back a session
// broader than the grant that justified it. Asking for nothing is not asking
// for everything.
func TestExchangeWithNoRequestedScopesInheritsTheGrant(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	_, err := e.GrantDelegation(ctx, appID, agent, userRef, []string{"repo:read"}, userRef, nil)
	require.NoError(t, err)

	sess, err := e.ExchangeToken(ctx, &authsome.ExchangeRequest{
		AppID: appID, Actor: agent, RequestedSubject: userRef,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"repo:read"}, sess.Scopes,
		"an unscoped request must inherit the grant's filter, not discard it")
}

// RevokeDelegation must refuse a delegation id from a different app: without
// this, an id disclosed in one tenant could revoke a grant in another this
// caller was never a party to.
func TestRevokeDelegationRefusesWrongApp(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	d, err := e.GrantDelegation(ctx, appID, agent, userRef, nil, userRef, nil)
	require.NoError(t, err)

	err = e.RevokeDelegation(ctx, id.NewAppID(), userRef, d.ID)
	assert.Error(t, err, "a delegation id from a different app must not be revocable")
}

// RevokeDelegation must refuse a caller that is neither the grant's subject
// nor its actor. Any id revoking any grant, regardless of who asks, is
// exactly the gap this closes.
func TestRevokeDelegationRefusesNonParty(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	d, err := e.GrantDelegation(ctx, appID, agent, userRef, nil, userRef, nil)
	require.NoError(t, err)

	stranger := newExchangeUserRef(t, e, appID)
	err = e.RevokeDelegation(ctx, appID, stranger, d.ID)
	assert.Error(t, err, "a caller that is not the grant's subject or actor must not be able to revoke it")
}

// The actor side, not just the subject, must also be able to revoke: an
// agent giving up authority it holds is as legitimate as the human taking
// it back.
func TestRevokeDelegationAllowsActor(t *testing.T) {
	e, appID, agent, userRef := setupExchangeFixture(t)
	ctx := context.Background()

	d, err := e.GrantDelegation(ctx, appID, agent, userRef, nil, userRef, nil)
	require.NoError(t, err)

	assert.NoError(t, e.RevokeDelegation(ctx, appID, agent, d.ID))
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
func setupExchangeFixture(t *testing.T, opts ...authsome.Option) (eng *authsome.Engine, appID id.AppID, agentRef, userRef principal.Ref) {
	t.Helper()
	eng, s := newTestEngine(t, opts...)
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

// newExchangeUserRef seeds a fresh, unrelated human user under appID and
// returns its ref. Used where a test needs a second or third party beyond
// setupExchangeFixture's own subject (Bob in the chained-exchange proof, the
// admin in the impersonation one, a stranger to a grant neither side names).
func newExchangeUserRef(t *testing.T, eng *authsome.Engine, appID id.AppID) principal.Ref {
	t.Helper()
	store, ok := eng.PrincipalStore().(interface {
		CreateUser(ctx context.Context, u *user.User) error
	})
	require.True(t, ok, "engine store must support CreateUser")

	userID := id.NewUserID()
	require.NoError(t, store.CreateUser(context.Background(), &user.User{
		ID:           userID,
		AppID:        appID,
		Email:        "exchange-user-" + userID.String() + "@example.com",
		PasswordHash: "$2a$10$fakehash",
	}))
	return principal.UserRef(userID)
}
