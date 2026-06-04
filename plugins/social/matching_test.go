package social

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"

	"golang.org/x/oauth2"
)

// newResolveTestPlugin wires a plugin with the in-memory core + oauth stores
// and returns both so tests can seed and assert directly. resolveUserForCallback
// is exercised in isolation (no HTTP/state ceremony).
func newResolveTestPlugin(t *testing.T) (*Plugin, *memory.Store) {
	t.Helper()
	p := New()
	ms := memory.New()
	p.SetStore(ms)
	p.SetOAuthStore(NewMemoryStore())
	return p, ms
}

func tok() *oauth2.Token {
	return &oauth2.Token{AccessToken: "access-tok", RefreshToken: "refresh-tok"}
}

func seedPrimaryUser(t *testing.T, ms *memory.Store, appID id.AppID, envID id.EnvironmentID, email string) *user.User {
	t.Helper()
	u := &user.User{ID: id.NewUserID(), AppID: appID, EnvID: envID, Email: email, EmailVerified: true}
	require.NoError(t, ms.CreateUserWithPrimaryEmail(context.Background(), u, &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: appID, EnvID: envID,
		Email: email, Verified: true, IsPrimary: true, Source: "test",
	}))
	return u
}

// The reported bug: a GitHub re-login after the account's email changed used
// to spawn a new user. With the connection (provider, provider_user_id) as the
// authoritative key, it must resolve to the same user.
func TestResolve_ProviderIDMatch_SurvivesEmailChange(t *testing.T) {
	ctx := context.Background()
	p, ms := newResolveTestPlugin(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	u := seedPrimaryUser(t, ms, appID, envID, "a@x.com")
	require.NoError(t, p.oauthStore.CreateOAuthConnection(ctx, &OAuthConnection{
		ID: id.NewOAuthConnectionID(), AppID: appID, UserID: u.ID,
		Provider: "github", ProviderUserID: "gh-1", Email: "a@x.com",
	}))

	// Re-login: same GitHub id, but the primary email changed to b@x.
	pu := &ProviderUser{
		ProviderUserID: "gh-1", Email: "b@x.com", EmailVerified: true,
		Emails: []ProviderEmail{{Email: "b@x.com", Verified: true, Primary: true}},
	}
	got, err := p.resolveUserForCallback(ctx, appID, envID, "github", pu, tok())
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), got.ID.String(), "must resolve to the same user, not a new one")

	// The changed email is reconciled onto the same account.
	owner, err := ms.GetUserByAnyEmail(ctx, appID, envID, "b@x.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), owner.ID.String())
}

// A GitHub account with multiple verified emails must link to an existing
// account that owns ANY of them, rather than creating a duplicate.
func TestResolve_VerifiedEmailMatch_LinksExistingAccount(t *testing.T) {
	ctx := context.Background()
	p, ms := newResolveTestPlugin(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	existing := seedPrimaryUser(t, ms, appID, envID, "a@x.com")

	// First GitHub login (new provider id): primary c@x, but also owns a@x.
	pu := &ProviderUser{
		ProviderUserID: "gh-2", Email: "c@x.com", EmailVerified: true,
		Emails: []ProviderEmail{
			{Email: "c@x.com", Verified: true, Primary: true},
			{Email: "a@x.com", Verified: true},
		},
	}
	got, err := p.resolveUserForCallback(ctx, appID, envID, "github", pu, tok())
	require.NoError(t, err)
	assert.Equal(t, existing.ID.String(), got.ID.String(), "must link to the existing account, not duplicate")

	conn, err := p.oauthStore.GetOAuthConnection(ctx, "github", "gh-2")
	require.NoError(t, err)
	assert.Equal(t, existing.ID.String(), conn.UserID.String())

	owner, err := ms.GetUserByAnyEmail(ctx, appID, envID, "c@x.com")
	require.NoError(t, err)
	assert.Equal(t, existing.ID.String(), owner.ID.String(), "c@x must be attached to the existing account")
}

// An UNVERIFIED provider email matching an existing verified account must not
// link (no takeover) and must not steal the address.
func TestResolve_UnverifiedEmail_NoTakeover(t *testing.T) {
	ctx := context.Background()
	p, ms := newResolveTestPlugin(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	existing := seedPrimaryUser(t, ms, appID, envID, "a@x.com")

	pu := &ProviderUser{
		ProviderUserID: "gh-3", Email: "a@x.com", EmailVerified: false,
		Emails: []ProviderEmail{{Email: "a@x.com", Verified: false, Primary: true}},
	}
	got, err := p.resolveUserForCallback(ctx, appID, envID, "github", pu, tok())
	require.NoError(t, err)
	assert.NotEqual(t, existing.ID.String(), got.ID.String(), "must NOT link to the existing account")

	// The existing account still owns a@x.
	owner, err := ms.GetUserByAnyEmail(ctx, appID, envID, "a@x.com")
	require.NoError(t, err)
	assert.Equal(t, existing.ID.String(), owner.ID.String(), "existing account must keep ownership of a@x")
}

// When verified provider emails belong to two different accounts, refuse
// rather than silently merge.
func TestResolve_AmbiguousMatch_Refuses(t *testing.T) {
	ctx := context.Background()
	p, ms := newResolveTestPlugin(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	seedPrimaryUser(t, ms, appID, envID, "d@x.com")
	seedPrimaryUser(t, ms, appID, envID, "e@x.com")

	pu := &ProviderUser{
		ProviderUserID: "gh-4", Email: "d@x.com", EmailVerified: true,
		Emails: []ProviderEmail{
			{Email: "d@x.com", Verified: true, Primary: true},
			{Email: "e@x.com", Verified: true},
		},
	}
	_, err := p.resolveUserForCallback(ctx, appID, envID, "github", pu, tok())
	require.Error(t, err, "ambiguous match must be refused")

	// No connection was created for the refused identity.
	_, connErr := p.oauthStore.GetOAuthConnection(ctx, "github", "gh-4")
	assert.Error(t, connErr)
}

// A brand-new identity creates a user seeded with all verified provider emails.
func TestResolve_NewUser_SeedsAllVerifiedEmails(t *testing.T) {
	ctx := context.Background()
	p, ms := newResolveTestPlugin(t)
	appID, envID := id.NewAppID(), id.NewEnvironmentID()

	pu := &ProviderUser{
		ProviderUserID: "gh-5", Email: "f@x.com", EmailVerified: true,
		Emails: []ProviderEmail{
			{Email: "f@x.com", Verified: true, Primary: true},
			{Email: "g@x.com", Verified: true},
		},
	}
	got, err := p.resolveUserForCallback(ctx, appID, envID, "github", pu, tok())
	require.NoError(t, err)

	for _, email := range []string{"f@x.com", "g@x.com"} {
		owner, oerr := ms.GetUserByAnyEmail(ctx, appID, envID, email)
		require.NoError(t, oerr, "email %s should be owned by the new user", email)
		assert.Equal(t, got.ID.String(), owner.ID.String())
	}

	conn, err := p.oauthStore.GetOAuthConnection(ctx, "github", "gh-5")
	require.NoError(t, err)
	assert.Equal(t, got.ID.String(), conn.UserID.String())
}
