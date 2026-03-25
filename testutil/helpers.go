package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	authclient "github.com/xraph/authsome/sdk/go"
)

// CreateUser signs up a user via the engine (server-side) and returns the
// user data plus the session token. Does NOT set the token on ts.Client —
// call CreateUserClient for that.
func (ts *TestServer) CreateUser(t *testing.T, email, pass string) (*authclient.AuthResponse, string) {
	t.Helper()

	ctx := context.Background()
	appID, err := id.ParseAppID(ts.AppID)
	require.NoError(t, err)

	u, sess, err := ts.Engine.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     email,
		Password:  pass,
		FirstName: "Test",
		LastName:  "User",
	})
	require.NoError(t, err)

	resp := &authclient.AuthResponse{
		SessionToken: sess.Token,
		RefreshToken: sess.RefreshToken,
		User: &authclient.User{
			ID:        u.ID.String(),
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
		},
	}

	return resp, sess.Token
}

// CreateUserClient creates a user and returns a new authclient.Client with
// the user's session token pre-set. Each call creates an independent client.
func (ts *TestServer) CreateUserClient(t *testing.T, email, pass string) *authclient.Client {
	t.Helper()
	_, token := ts.CreateUser(t, email, pass)
	return authclient.NewClient(ts.Server.URL,
		authclient.WithToken(token),
		authclient.WithSessionCookies(),
	)
}

// CreateOrg creates an Authsome organization owned by the given user.
// It uses the org plugin directly (server-side) and returns the created org.
func (ts *TestServer) CreateOrg(t *testing.T, ownerUserID, name, slug string) *organization.Organization {
	t.Helper()
	ctx := context.Background()

	appID, err := id.ParseAppID(ts.AppID)
	require.NoError(t, err)

	creatorID, err := id.ParseUserID(ownerUserID)
	require.NoError(t, err)

	org := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		Name:      name,
		Slug:      slug,
		CreatedBy: creatorID,
	}

	require.NoError(t, ts.OrgPlugin.CreateOrganization(ctx, org))
	return org
}

// AddMember adds a user as a member of an organization (server-side).
func (ts *TestServer) AddMember(t *testing.T, orgID, userID string, role organization.MemberRole) *organization.Member {
	t.Helper()
	ctx := context.Background()

	parsedOrgID, err := id.ParseOrgID(orgID)
	require.NoError(t, err)
	parsedUserID, err := id.ParseUserID(userID)
	require.NoError(t, err)

	m := &organization.Member{
		ID:     id.NewMemberID(),
		OrgID:  parsedOrgID,
		UserID: parsedUserID,
		Role:   role,
	}
	require.NoError(t, ts.OrgPlugin.AddMember(ctx, m))
	return m
}

// TestUserFactory generates test users with sequential emails.
type TestUserFactory struct {
	counter int
	ts      *TestServer
}

// NewTestUserFactory creates a factory bound to the given test server.
func (ts *TestServer) NewTestUserFactory() *TestUserFactory {
	return &TestUserFactory{ts: ts}
}

// Next creates the next test user and returns the auth response and token.
func (f *TestUserFactory) Next(t *testing.T) (*authclient.AuthResponse, string) {
	t.Helper()
	f.counter++
	email := fmt.Sprintf("testuser-%d@example.com", f.counter)
	return f.ts.CreateUser(t, email, "SecureP@ss1")
}

// NextClient creates the next test user and returns a pre-authed client.
func (f *TestUserFactory) NextClient(t *testing.T) *authclient.Client {
	t.Helper()
	f.counter++
	email := fmt.Sprintf("testuser-%d@example.com", f.counter)
	return f.ts.CreateUserClient(t, email, "SecureP@ss1")
}
