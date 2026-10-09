package sso

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	memory "github.com/xraph/authsome/store/memory"
)

// The identity a provider asserts lands on exactly one account: the one
// bound to its subject, else the verified holder of its email, else a new
// user. Nothing is touched when the email is outside the connection's
// domains or the provider says it is unverified.

func identityFixture(t *testing.T) (*Plugin, *memory.Store, *Connection, id.AppID, id.EnvironmentID) {
	t.Helper()
	s := memory.New()
	p := New()
	p.SetStore(s)
	p.SetSSOStore(NewMemoryStore())
	appID, envID := id.NewAppID(), id.NewEnvironmentID()
	conn := &Connection{ID: id.NewSSOConnectionID(), AppID: appID, Provider: "okta", Protocol: "oidc", Domain: "corp.example", AllowedDomains: []string{"subsidiary.example"}, Active: true}
	require.NoError(t, p.ssoStore.CreateConnection(context.Background(), conn))
	return p, s, conn, appID, envID
}

func httpStatus(t *testing.T, err error) int {
	t.Helper()
	var he interface{ StatusCode() int }
	require.ErrorAs(t, err, &he)
	return he.StatusCode()
}

func TestResolveIdentity_RefusesEmailOutsideTheConnectionDomains(t *testing.T) {
	p, _, conn, appID, envID := identityFixture(t)
	_, _, err := p.resolveIdentity(context.Background(), appID, envID, conn, &User{ProviderUserID: "s1", Email: "alice@attacker.example"})
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, httpStatus(t, err))

	u, isNew, err := p.resolveIdentity(context.Background(), appID, envID, conn, &User{ProviderUserID: "s2", Email: "Bob@Subsidiary.Example"})
	require.NoError(t, err, "a listed extra domain is accepted whatever its case")
	assert.True(t, isNew)
	assert.Equal(t, "bob@subsidiary.example", u.Email)
}

func TestResolveIdentity_RefusesAnUnverifiedClaim(t *testing.T) {
	p, _, conn, appID, envID := identityFixture(t)
	no := false
	_, _, err := p.resolveIdentity(context.Background(), appID, envID, conn, &User{ProviderUserID: "s1", Email: "alice@corp.example", EmailVerified: &no})
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, httpStatus(t, err))
	yes := true
	_, _, err = p.resolveIdentity(context.Background(), appID, envID, conn, &User{ProviderUserID: "s1", Email: "alice@corp.example", EmailVerified: &yes})
	require.NoError(t, err)
}

func TestResolveIdentity_SubjectBindingOutranksEmail(t *testing.T) {
	p, s, conn, appID, envID := identityFixture(t)
	ctx := context.Background()
	first, isNew, err := p.resolveIdentity(ctx, appID, envID, conn, &User{ProviderUserID: "subject-7", Email: "alice@corp.example"})
	require.NoError(t, err)
	require.True(t, isNew)
	ident, err := p.ssoStore.GetIdentity(ctx, conn.ID, "subject-7")
	require.NoError(t, err, "the first login records the binding")
	assert.Equal(t, first.ID.String(), ident.UserID.String())

	// A verified account now holds a different email; the IdP asserts that
	// email for the same subject. The subject wins.
	other := seedUserWithEmail(t, s, appID, envID, "carol@corp.example", true, "")
	again, isNew, err := p.resolveIdentity(ctx, appID, envID, conn, &User{ProviderUserID: "subject-7", Email: "carol@corp.example"})
	require.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, first.ID.String(), again.ID.String(), "the bound user, not the email holder")
	assert.NotEqual(t, other.ID.String(), again.ID.String())

	// An unseen subject asserting a verified holder's email links to it and
	// binds, so the next login by that subject no longer needs the email.
	linked, isNew, err := p.resolveIdentity(ctx, appID, envID, conn, &User{ProviderUserID: "subject-8", Email: "carol@corp.example"})
	require.NoError(t, err)
	assert.False(t, isNew)
	assert.Equal(t, other.ID.String(), linked.ID.String())
	viaSubject, err := p.ssoStore.GetIdentity(ctx, conn.ID, "subject-8")
	require.NoError(t, err)
	assert.Equal(t, other.ID.String(), viaSubject.UserID.String())
}

func TestResolveIdentity_AdoptsTheSubjectOnlyForTrustedFederation(t *testing.T) {
	p, _, conn, appID, envID := identityFixture(t)
	ctx := context.Background()
	adoptable := id.NewUserID().String()
	u, _, err := p.resolveIdentity(ctx, appID, envID, conn, &User{ProviderUserID: adoptable, Email: "dan@corp.example"})
	require.NoError(t, err)
	assert.NotEqual(t, adoptable, u.ID.String(), "an untrusted connection never lets the IdP pick local ids")

	conn.TrustedFederation = true
	require.NoError(t, p.ssoStore.UpdateConnection(ctx, conn))
	adoptable2 := id.NewUserID().String()
	u2, _, err := p.resolveIdentity(ctx, appID, envID, conn, &User{ProviderUserID: adoptable2, Email: "erin@corp.example"})
	require.NoError(t, err)
	assert.Equal(t, adoptable2, u2.ID.String(), "a trusted federation adopts the IdP subject")
}

func TestResolveIdentity_BoundUserFromAnotherAppIsIgnored(t *testing.T) {
	p, s, conn, appID, envID := identityFixture(t)
	ctx := context.Background()
	foreign := seedUserWithEmail(t, s, id.NewAppID(), envID, "frank@corp.example", true, "")
	require.NoError(t, p.ssoStore.CreateIdentity(ctx, &Identity{ConnectionID: conn.ID, Subject: "cross-app", UserID: foreign.ID}))
	u, _, err := p.resolveIdentity(ctx, appID, envID, conn, &User{ProviderUserID: "cross-app", Email: "frank@corp.example"})
	require.NoError(t, err)
	assert.NotEqual(t, foreign.ID.String(), u.ID.String(), "a binding that points outside the app is not followed")
}
