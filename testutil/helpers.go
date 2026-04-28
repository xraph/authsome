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
	"github.com/xraph/authsome/session"
)

// CreateUser signs up a user via the engine (server-side) and returns the
// user data plus the session token. Does NOT set the token on ts.Client —
// call CreateUserClient for that.
func (ts *TestServer) CreateUser(t *testing.T, email, pass string) (resp *authclient.AuthResponse, token string) {
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

	resp = &authclient.AuthResponse{
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
func (f *TestUserFactory) Next(t *testing.T) (resp *authclient.AuthResponse, token string) {
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

// SwitchOrg flips the active organization on a session by calling the
// engine directly (server-side). Returns the updated session for
// assertion. Use this when a test needs to verify
// /me/switch-org-equivalent flows without driving the SDK.
func (ts *TestServer) SwitchOrg(t *testing.T, sessionID, orgID string) *session.Session {
	t.Helper()
	ctx := context.Background()

	parsedSession, err := id.ParseSessionID(sessionID)
	require.NoError(t, err, "parse session id %q", sessionID)

	var parsedOrg id.OrgID
	if orgID != "" {
		parsedOrg, err = id.ParseOrgID(orgID)
		require.NoError(t, err, "parse org id %q", orgID)
	}

	updated, err := ts.Engine.SwitchActiveOrg(ctx, parsedSession, parsedOrg)
	require.NoError(t, err, "SwitchActiveOrg")
	return updated
}

// SetMemberRole mutates the role of an existing org member. Works
// off the member ID returned by AddMember. Panics via t.Fatal when
// the member doesn't exist or the org plugin rejects the change.
//
// Use this to set up role-gate tests (owner vs admin vs member) on
// the same user without recreating the org.
func (ts *TestServer) SetMemberRole(t *testing.T, memberID string, role organization.MemberRole) *organization.Member {
	t.Helper()
	parsed, err := id.ParseMemberID(memberID)
	require.NoError(t, err, "parse member id %q", memberID)

	updated, err := ts.OrgPlugin.UpdateMemberRole(context.Background(), parsed, role)
	require.NoError(t, err, "UpdateMemberRole")
	return updated
}

// SessionByToken fetches a session by its bearer token. Useful for
// asserting session state (e.g. OrgID after a switch-org call) in
// tests that drive the SDK and only have the token in hand.
func (ts *TestServer) SessionByToken(t *testing.T, token string) *session.Session {
	t.Helper()
	sess, err := ts.Store.GetSessionByToken(context.Background(), token)
	require.NoError(t, err, "GetSessionByToken")
	return sess
}
