package authsome_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	orgplugin "github.com/xraph/authsome/plugins/organization"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/webhook"

	"github.com/xraph/warden"
	wardenmem "github.com/xraph/warden/store/memory"
)

// ──────────────────────────────────────────────────
// E2E test helpers
// ──────────────────────────────────────────────────

func e2eAppID(t *testing.T) id.AppID {
	t.Helper()
	appID, err := id.ParseAppID("aapp_01jf0000000000000000000000")
	require.NoError(t, err)
	return appID
}

func e2eEngine(t *testing.T, opts ...authsome.Option) (*authsome.Engine, *memory.Store) {
	t.Helper()
	s := memory.New()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	baseOpts := []authsome.Option{
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID("aapp_01jf0000000000000000000000"),
	}
	eng, err := authsome.NewEngine(append(baseOpts, opts...)...)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, eng.Start(ctx))
	t.Cleanup(func() { _ = eng.Stop(ctx) })

	return eng, s
}

// ──────────────────────────────────────────────────
// E2E: Sign Up → Sign In → Sign Out
// ──────────────────────────────────────────────────

func TestE2E_SignUpSignInSignOut(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Sign up
	u, sess, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "alice@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Alice",
		Username:  "alice",
	})
	require.NoError(t, err)
	assert.NotNil(t, u)
	assert.NotNil(t, sess)
	assert.Equal(t, "alice@example.com", u.Email)
	assert.Equal(t, "Alice", u.FirstName)
	assert.Equal(t, "alice", u.Username)
	assert.NotEmpty(t, sess.Token)
	assert.NotEmpty(t, sess.RefreshToken)

	// Step 2: Verify session is valid
	resolved, err := eng.ResolveSessionByToken(sess.Token)
	require.NoError(t, err)
	assert.Equal(t, sess.ID, resolved.ID)
	assert.Equal(t, u.ID, resolved.UserID)

	// Step 3: Sign in with same credentials
	u2, sess2, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "alice@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)
	assert.Equal(t, u.ID, u2.ID)
	assert.NotEqual(t, sess.Token, sess2.Token) // New session

	// Step 4: Sign out first session
	err = eng.SignOut(ctx, sess.ID)
	require.NoError(t, err)

	// Step 5: First session should no longer resolve
	_, err = eng.ResolveSessionByToken(sess.Token)
	assert.Error(t, err)

	// Step 6: Second session should still be valid
	resolved2, err := eng.ResolveSessionByToken(sess2.Token)
	require.NoError(t, err)
	assert.Equal(t, sess2.ID, resolved2.ID)
}

// ──────────────────────────────────────────────────
// E2E: Organization Invitation Flow
// ──────────────────────────────────────────────────

func e2eEngineWithOrg(t *testing.T) (*authsome.Engine, *memory.Store, *orgplugin.Plugin) {
	t.Helper()
	s := memory.New()
	w, err := warden.NewEngine(warden.WithStore(wardenmem.New()))
	require.NoError(t, err)
	op := orgplugin.New()
	eng, err := authsome.NewEngine(
		authsome.WithStore(s),
		authsome.WithWarden(w),
		authsome.WithDisableMigrate(),
		authsome.WithAppID("aapp_01jf0000000000000000000000"),
		authsome.WithPlugin(op),
	)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, eng.Start(ctx))
	t.Cleanup(func() { _ = eng.Stop(ctx) })

	return eng, s, op
}

func TestE2E_OrgInvitationFlow(t *testing.T) {
	eng, _, op := e2eEngineWithOrg(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create owner user
	owner, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "owner@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Owner",
	})
	require.NoError(t, err)

	// Step 2: Create invitee user
	_, _, err = eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "invitee@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Invitee",
	})
	require.NoError(t, err)

	// Step 3: Create organization
	org := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		Name:      "Acme Corp",
		Slug:      "acme-corp",
		CreatedBy: owner.ID,
	}
	err = op.CreateOrganization(ctx, org)
	require.NoError(t, err)

	// Step 4: Verify owner is a member
	members, err := op.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, owner.ID, members[0].UserID)
	assert.Equal(t, organization.RoleOwner, members[0].Role)

	// Step 5: Create invitation for invitee
	inv := &organization.Invitation{
		ID:     id.NewInvitationID(),
		OrgID:  org.ID,
		Email:  "invitee@example.com",
		Role:   organization.RoleMember,
		Token:  "invite-token-123",
		Status: organization.InvitationPending,
	}
	err = op.CreateInvitation(ctx, inv)
	require.NoError(t, err)

	// Step 6: List invitations
	invitations, err := op.ListInvitations(ctx, org.ID)
	require.NoError(t, err)
	require.Len(t, invitations, 1)
	assert.Equal(t, "invitee@example.com", invitations[0].Email)

	// Step 7: Accept invitation
	member, err := op.AcceptInvitation(ctx, "invite-token-123")
	require.NoError(t, err)
	assert.NotNil(t, member)
	assert.Equal(t, org.ID, member.OrgID)
	assert.Equal(t, organization.RoleMember, member.Role)

	// Step 8: Verify member is now in the org
	members, err = op.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 2)

	// Step 9: Update member role to admin
	updatedMember, err := op.UpdateMemberRole(ctx, member.ID, organization.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, organization.RoleAdmin, updatedMember.Role)
}

// ──────────────────────────────────────────────────
// E2E: Device Tracking
// ──────────────────────────────────────────────────

func TestE2E_DeviceTracking(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "device-user@example.com",
		Password:  "SecureP@ss1",
		FirstName: "DeviceUser",
	})
	require.NoError(t, err)

	// Step 2: Register a device
	d := &device.Device{
		AppID:       appID,
		UserID:      u.ID,
		Fingerprint: "fp-abc123",
		Browser:     "Mozilla/5.0 TestBrowser",
		IPAddress:   "192.168.1.1",
	}
	registered, err := eng.RegisterDevice(ctx, d)
	require.NoError(t, err)
	assert.NotEmpty(t, registered.ID.String())
	assert.Equal(t, "fp-abc123", registered.Fingerprint)
	assert.False(t, registered.Trusted)

	// Step 3: List devices
	devices, err := eng.ListUserDevices(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, devices, 1)
	assert.Equal(t, registered.ID, devices[0].ID)

	// Step 4: Trust the device
	trusted, err := eng.TrustDevice(ctx, registered.ID)
	require.NoError(t, err)
	assert.True(t, trusted.Trusted)

	// Step 5: Re-register same fingerprint should update, not create
	d2 := &device.Device{
		AppID:       appID,
		UserID:      u.ID,
		Fingerprint: "fp-abc123",
		Browser:     "Mozilla/5.0 UpdatedBrowser",
		IPAddress:   "192.168.1.2",
	}
	updated, err := eng.RegisterDevice(ctx, d2)
	require.NoError(t, err)
	assert.Equal(t, registered.ID, updated.ID) // same device
	assert.Equal(t, "192.168.1.2", updated.IPAddress)

	// Step 6: List should still show 1 device
	devices, err = eng.ListUserDevices(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, devices, 1)

	// Step 7: Delete device
	err = eng.DeleteDevice(ctx, registered.ID)
	require.NoError(t, err)

	devices, err = eng.ListUserDevices(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, devices, 0)
}

// ──────────────────────────────────────────────────
// E2E: Webhook Management
// ──────────────────────────────────────────────────

func TestE2E_WebhookManagement(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create webhook
	w := &webhook.Webhook{
		AppID:  appID,
		URL:    "https://example.com/webhook",
		Events: []string{"user.created", "auth.signin"},
	}
	err := eng.CreateWebhook(ctx, w)
	require.NoError(t, err)
	assert.NotEmpty(t, w.ID.String())
	assert.NotEmpty(t, w.Secret) // auto-generated
	assert.True(t, w.Active)

	// Step 2: List webhooks
	webhooks, err := eng.ListWebhooks(ctx, appID)
	require.NoError(t, err)
	assert.Len(t, webhooks, 1)
	assert.Equal(t, w.ID, webhooks[0].ID)

	// Step 3: Get webhook
	got, err := eng.GetWebhook(ctx, w.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/webhook", got.URL)
	assert.Equal(t, []string{"user.created", "auth.signin"}, got.Events)

	// Step 4: Update webhook
	w.URL = "https://example.com/webhook-v2"
	w.Events = []string{"user.created"}
	err = eng.UpdateWebhook(ctx, w)
	require.NoError(t, err)

	got, err = eng.GetWebhook(ctx, w.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/webhook-v2", got.URL)
	assert.Equal(t, []string{"user.created"}, got.Events)

	// Step 5: Delete webhook
	err = eng.DeleteWebhook(ctx, w.ID)
	require.NoError(t, err)

	_, err = eng.GetWebhook(ctx, w.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// ──────────────────────────────────────────────────
// E2E: Password Reset Flow
// ──────────────────────────────────────────────────

func TestE2E_PasswordResetFlow(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create user
	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "reset-user@example.com",
		Password:  "OldP@ssword1",
		FirstName: "ResetUser",
	})
	require.NoError(t, err)

	// Step 2: Request password reset
	pr, err := eng.ForgotPassword(ctx, appID, "reset-user@example.com")
	require.NoError(t, err)
	require.NotNil(t, pr)
	assert.NotEmpty(t, pr.Token)
	assert.False(t, pr.Consumed)
	assert.True(t, pr.ExpiresAt.After(time.Now()))

	// Step 3: Reset password with token
	err = eng.ResetPassword(ctx, pr.Token, "NewP@ssword1")
	require.NoError(t, err)

	// Step 4: Old password should fail
	_, _, err = eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "reset-user@example.com",
		Password: "OldP@ssword1",
	})
	assert.Error(t, err)

	// Step 5: New password should succeed
	u, sess, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "reset-user@example.com",
		Password: "NewP@ssword1",
	})
	require.NoError(t, err)
	assert.NotNil(t, u)
	assert.NotNil(t, sess)
	assert.Equal(t, "reset-user@example.com", u.Email)

	// Step 6: Token should not be reusable
	err = eng.ResetPassword(ctx, pr.Token, "AnotherP@ss1")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────────
// E2E: Session Management
// ──────────────────────────────────────────────────

func TestE2E_SessionManagement(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create user
	_, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "session-user@example.com",
		Password:  "SecureP@ss1",
		FirstName: "SessionUser",
	})
	require.NoError(t, err)

	// Step 2: Create multiple sessions by signing in multiple times
	_, sess1, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "session-user@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)

	_, sess2, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "session-user@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)

	_, sess3, err := eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "session-user@example.com",
		Password: "SecureP@ss1",
	})
	require.NoError(t, err)

	// Step 3: List sessions (should have 4: 1 from signup + 3 from signin)
	sessions, err := eng.ListSessions(ctx, sess1.UserID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(sessions), 3)

	// Step 4: Revoke the second session
	err = eng.RevokeSession(ctx, sess2.ID)
	require.NoError(t, err)

	// Step 5: Second session should be gone
	_, err = eng.ResolveSessionByToken(sess2.Token)
	assert.Error(t, err)

	// Step 6: First and third sessions should still work
	_, err = eng.ResolveSessionByToken(sess1.Token)
	require.NoError(t, err)

	_, err = eng.ResolveSessionByToken(sess3.Token)
	require.NoError(t, err)

	// Step 7: Refresh the first session
	oldToken := sess1.Token
	oldRefresh := sess1.RefreshToken
	refreshed, err := eng.Refresh(ctx, oldRefresh)
	require.NoError(t, err)
	assert.NotEmpty(t, refreshed.Token)
	assert.NotEqual(t, oldToken, refreshed.Token) // New token
}

// ──────────────────────────────────────────────────
// E2E: Change Password
// ──────────────────────────────────────────────────

func TestE2E_ChangePassword(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "change-pw@example.com",
		Password:  "OldP@ssword1",
		FirstName: "ChangePW",
	})
	require.NoError(t, err)

	// Step 2: Change password
	err = eng.ChangePassword(ctx, u.ID, "OldP@ssword1", "NewP@ssword1")
	require.NoError(t, err)

	// Step 3: Old password should fail
	_, _, err = eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "change-pw@example.com",
		Password: "OldP@ssword1",
	})
	assert.Error(t, err)

	// Step 4: New password should succeed
	_, _, err = eng.SignIn(ctx, &account.SignInRequest{
		AppID:    appID,
		Email:    "change-pw@example.com",
		Password: "NewP@ssword1",
	})
	require.NoError(t, err)

	// Step 5: Wrong current password should fail
	err = eng.ChangePassword(ctx, u.ID, "WrongCurrent1", "AnotherP@ss1")
	assert.Error(t, err)
}

// ──────────────────────────────────────────────────
// E2E: Organization + Team Management
// ──────────────────────────────────────────────────

func TestE2E_OrgTeamManagement(t *testing.T) {
	eng, _, op := e2eEngineWithOrg(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "team-owner@example.com",
		Password:  "SecureP@ss1",
		FirstName: "TeamOwner",
	})
	require.NoError(t, err)

	// Step 2: Create organization
	org := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		Name:      "Team Org",
		Slug:      "team-org",
		CreatedBy: u.ID,
	}
	err = op.CreateOrganization(ctx, org)
	require.NoError(t, err)

	// Step 3: Check slug availability
	available, err := op.IsOrgSlugAvailable(ctx, appID, "team-org")
	require.NoError(t, err)
	assert.False(t, available) // already taken

	available, err = op.IsOrgSlugAvailable(ctx, appID, "unique-slug")
	require.NoError(t, err)
	assert.True(t, available)

	// Step 4: Create teams
	team1 := &organization.Team{
		ID:    id.NewTeamID(),
		OrgID: org.ID,
		Name:  "Engineering",
		Slug:  "engineering",
	}
	err = op.CreateTeam(ctx, team1)
	require.NoError(t, err)

	team2 := &organization.Team{
		ID:    id.NewTeamID(),
		OrgID: org.ID,
		Name:  "Design",
		Slug:  "design",
	}
	err = op.CreateTeam(ctx, team2)
	require.NoError(t, err)

	// Step 5: List teams
	teams, err := op.ListTeams(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, teams, 2)

	// Step 6: Update team
	team1.Name = "Platform Engineering"
	err = op.UpdateTeam(ctx, team1)
	require.NoError(t, err)

	got, err := op.GetTeam(ctx, team1.ID)
	require.NoError(t, err)
	assert.Equal(t, "Platform Engineering", got.Name)

	// Step 7: Delete team
	err = op.DeleteTeam(ctx, team2.ID)
	require.NoError(t, err)

	teams, err = op.ListTeams(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, teams, 1)

	// Step 8: Delete organization
	err = op.DeleteOrganization(ctx, org.ID)
	require.NoError(t, err)

	_, err = op.GetOrganization(ctx, org.ID)
	assert.Error(t, err)
}

// ──────────────────────────────────────────────────
// E2E: RBAC Permission Flow
// ──────────────────────────────────────────────────

func TestE2E_RBACPermissionFlow(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "rbac-user@example.com",
		Password:  "SecureP@ss1",
		FirstName: "RBACUser",
	})
	require.NoError(t, err)

	// Step 2: Create admin role
	adminRoleID := id.NewRoleID()
	adminRole := &rbac.Role{
		ID:          adminRoleID.String(),
		AppID:       appID.String(),
		Name:        "Admin",
		Slug:        "admin",
		Description: "Full access administrator",
	}
	err = eng.CreateRole(ctx, adminRole)
	require.NoError(t, err)
	assert.NotEmpty(t, adminRole.CreatedAt)

	// Step 3: Create viewer role
	viewerRoleID := id.NewRoleID()
	viewerRole := &rbac.Role{
		ID:          viewerRoleID.String(),
		AppID:       appID.String(),
		Name:        "Viewer",
		Slug:        "viewer",
		Description: "Read-only access",
	}
	err = eng.CreateRole(ctx, viewerRole)
	require.NoError(t, err)

	// Step 4: Add permissions to admin role
	err = eng.AddPermission(ctx, &rbac.Permission{
		ID:       id.NewPermissionID().String(),
		RoleID:   adminRole.ID,
		Action:   "*",
		Resource: "*",
	})
	require.NoError(t, err)

	// Step 5: Add permissions to viewer role
	err = eng.AddPermission(ctx, &rbac.Permission{
		ID:       id.NewPermissionID().String(),
		RoleID:   viewerRole.ID,
		Action:   "read",
		Resource: "user",
	})
	require.NoError(t, err)

	err = eng.AddPermission(ctx, &rbac.Permission{
		ID:       id.NewPermissionID().String(),
		RoleID:   viewerRole.ID,
		Action:   "read",
		Resource: "org",
	})
	require.NoError(t, err)

	// Step 6: List roles
	roles, err := eng.ListRoles(ctx, appID)
	require.NoError(t, err)
	assert.Len(t, roles, 2)

	// Step 7: List role permissions
	adminPerms, err := eng.ListRolePermissions(ctx, adminRoleID)
	require.NoError(t, err)
	assert.Len(t, adminPerms, 1)
	assert.Equal(t, "*", adminPerms[0].Action)

	viewerPerms, err := eng.ListRolePermissions(ctx, viewerRoleID)
	require.NoError(t, err)
	assert.Len(t, viewerPerms, 2)

	// Step 8: Assign viewer role to user
	err = eng.AssignUserRole(ctx, &rbac.UserRole{
		UserID: u.ID.String(),
		RoleID: viewerRole.ID,
	})
	require.NoError(t, err)

	// Step 9: Check permissions
	ok, err := eng.HasPermission(ctx, u.ID, "read", "user")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = eng.HasPermission(ctx, u.ID, "write", "user")
	require.NoError(t, err)
	assert.False(t, ok) // viewer can't write

	ok, err = eng.HasPermission(ctx, u.ID, "read", "org")
	require.NoError(t, err)
	assert.True(t, ok)

	// Step 10: Upgrade to admin
	err = eng.AssignUserRole(ctx, &rbac.UserRole{
		UserID: u.ID.String(),
		RoleID: adminRole.ID,
	})
	require.NoError(t, err)

	// Step 11: Now write should be allowed (admin has wildcard)
	ok, err = eng.HasPermission(ctx, u.ID, "write", "user")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = eng.HasPermission(ctx, u.ID, "delete", "anything")
	require.NoError(t, err)
	assert.True(t, ok) // wildcard

	// Step 12: List user roles
	userRoles, err := eng.ListUserRoles(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 2) // viewer + admin

	// Step 13: Unassign viewer role
	err = eng.UnassignUserRole(ctx, u.ID, viewerRoleID)
	require.NoError(t, err)

	userRoles, err = eng.ListUserRoles(ctx, u.ID)
	require.NoError(t, err)
	assert.Len(t, userRoles, 1) // admin only

	// Step 14: Delete viewer role
	err = eng.DeleteRole(ctx, viewerRoleID)
	require.NoError(t, err)

	roles, err = eng.ListRoles(ctx, appID)
	require.NoError(t, err)
	assert.Len(t, roles, 1) // admin only
}

// ──────────────────────────────────────────────────
// E2E: User Update Flow
// ──────────────────────────────────────────────────

func TestE2E_UserUpdateFlow(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create user
	u, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "update-user@example.com",
		Password:  "SecureP@ss1",
		FirstName: "Original Name",
	})
	require.NoError(t, err)

	// Step 2: Get user
	got, err := eng.GetMe(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Original Name", got.FirstName)

	// Step 3: Update user
	got.FirstName = "Updated Name"
	got.Image = "https://example.com/avatar.jpg"
	err = eng.UpdateMe(ctx, got)
	require.NoError(t, err)

	// Step 4: Verify update
	got2, err := eng.GetMe(ctx, u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", got2.FirstName)
	assert.Equal(t, "https://example.com/avatar.jpg", got2.Image)
	assert.True(t, got2.UpdatedAt.After(u.CreatedAt))
}

// ──────────────────────────────────────────────────
// E2E: Decline Invitation
// ──────────────────────────────────────────────────

func TestE2E_DeclineInvitation(t *testing.T) {
	eng, _, op := e2eEngineWithOrg(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Create owner
	owner, _, err := eng.SignUp(ctx, &account.SignUpRequest{
		AppID:     appID,
		Email:     "org-owner@example.com",
		Password:  "SecureP@ss1",
		FirstName: "OrgOwner",
	})
	require.NoError(t, err)

	// Create org
	org := &organization.Organization{
		ID:        id.NewOrgID(),
		AppID:     appID,
		Name:      "Decline Org",
		Slug:      "decline-org",
		CreatedBy: owner.ID,
	}
	err = op.CreateOrganization(ctx, org)
	require.NoError(t, err)

	// Create invitation
	inv := &organization.Invitation{
		ID:     id.NewInvitationID(),
		OrgID:  org.ID,
		Email:  "decliner@example.com",
		Role:   organization.RoleMember,
		Token:  "decline-token-456",
		Status: organization.InvitationPending,
	}
	err = op.CreateInvitation(ctx, inv)
	require.NoError(t, err)

	// Decline invitation
	err = op.DeclineInvitation(ctx, "decline-token-456")
	require.NoError(t, err)

	// Should not be able to accept after declining
	_, err = op.AcceptInvitation(ctx, "decline-token-456")
	assert.Error(t, err)

	// Members should still be just the owner
	members, err := op.ListMembers(ctx, org.ID)
	require.NoError(t, err)
	assert.Len(t, members, 1)
}

// ──────────────────────────────────────────────────
// E2E: Environment Lifecycle
// ──────────────────────────────────────────────────

func TestE2E_EnvironmentLifecycle(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create production environment (default)
	prod := &environment.Environment{
		AppID:     appID,
		Name:      "Production",
		Slug:      "production",
		Type:      environment.TypeProduction,
		IsDefault: true,
	}
	err := eng.CreateEnvironment(ctx, prod)
	require.NoError(t, err)
	assert.False(t, prod.ID.IsNil(), "ID should be generated")
	assert.False(t, prod.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.Equal(t, "#ef4444", prod.Color, "default color for production")

	// Step 2: Create development environment
	dev := &environment.Environment{
		AppID: appID,
		Name:  "Development",
		Slug:  "development",
		Type:  environment.TypeDevelopment,
	}
	err = eng.CreateEnvironment(ctx, dev)
	require.NoError(t, err)

	// Step 3: List environments -> 2
	envs, err := eng.ListEnvironments(ctx, appID)
	require.NoError(t, err)
	assert.Len(t, envs, 2)

	// Step 4: Get environment by ID -> correct name
	got, err := eng.GetEnvironment(ctx, dev.ID)
	require.NoError(t, err)
	assert.Equal(t, "Development", got.Name)
	assert.Equal(t, environment.TypeDevelopment, got.Type)

	// Step 5: Update development name
	dev.Name = "Dev Environment"
	err = eng.UpdateEnvironment(ctx, dev)
	require.NoError(t, err)

	got, err = eng.GetEnvironment(ctx, dev.ID)
	require.NoError(t, err)
	assert.Equal(t, "Dev Environment", got.Name)

	// Step 6: Set development as default
	err = eng.SetDefaultEnvironment(ctx, appID, dev.ID)
	require.NoError(t, err)

	// Step 7: Verify production no longer default
	gotProd, err := eng.GetEnvironment(ctx, prod.ID)
	require.NoError(t, err)
	assert.False(t, gotProd.IsDefault, "production should no longer be default")

	gotDev, err := eng.GetEnvironment(ctx, dev.ID)
	require.NoError(t, err)
	assert.True(t, gotDev.IsDefault, "development should now be default")

	// Step 8: Delete production (non-default)
	err = eng.DeleteEnvironment(ctx, prod.ID)
	require.NoError(t, err)

	// Step 9: Verify deletion, list -> 1
	_, err = eng.GetEnvironment(ctx, prod.ID)
	assert.Error(t, err)

	envs, err = eng.ListEnvironments(ctx, appID)
	require.NoError(t, err)
	assert.Len(t, envs, 1)
	assert.Equal(t, dev.ID.String(), envs[0].ID.String())

	// Step 10: Cannot delete default environment
	err = eng.DeleteEnvironment(ctx, dev.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default")
}

// ──────────────────────────────────────────────────
// E2E: Environment Clone
// ──────────────────────────────────────────────────

func TestE2E_EnvironmentClone(t *testing.T) {
	eng, _ := e2eEngine(t)
	ctx := context.Background()
	appID := e2eAppID(t)

	// Step 1: Create production environment (default)
	prod := &environment.Environment{
		AppID:     appID,
		Name:      "Production",
		Slug:      "production",
		Type:      environment.TypeProduction,
		IsDefault: true,
	}
	err := eng.CreateEnvironment(ctx, prod)
	require.NoError(t, err)

	// Step 2: Create 2 roles (admin as parent, viewer as child) with EnvID
	adminRole := &rbac.Role{
		ID:    id.NewRoleID().String(),
		AppID: appID.String(),
		EnvID: prod.ID.String(),
		Name:  "Admin",
		Slug:  "admin",
	}
	err = eng.CreateRole(ctx, adminRole)
	require.NoError(t, err)

	viewerRole := &rbac.Role{
		ID:       id.NewRoleID().String(),
		AppID:    appID.String(),
		EnvID:    prod.ID.String(),
		Name:     "Viewer",
		Slug:     "viewer",
		ParentID: adminRole.ID,
	}
	err = eng.CreateRole(ctx, viewerRole)
	require.NoError(t, err)

	// Step 3: Add permission to admin role
	err = eng.AddPermission(ctx, &rbac.Permission{
		ID:       id.NewPermissionID().String(),
		RoleID:   adminRole.ID,
		Action:   "*",
		Resource: "*",
	})
	require.NoError(t, err)

	// Step 4: Create webhook with EnvID
	w := &webhook.Webhook{
		AppID:  appID,
		EnvID:  prod.ID,
		URL:    "https://prod.example.com/hook",
		Events: []string{"user.created"},
		Active: true,
	}
	err = eng.CreateWebhook(ctx, w)
	require.NoError(t, err)

	// Step 5: Clone production -> staging
	result, err := eng.CloneEnvironment(ctx, environment.CloneRequest{
		SourceEnvID: prod.ID,
		Name:        "Staging",
		Slug:        "staging",
		Type:        environment.TypeStaging,
		Description: "Cloned from production",
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Step 6: Verify cloned env
	assert.Equal(t, "Staging", result.Environment.Name)
	assert.Equal(t, "staging", result.Environment.Slug)
	assert.Equal(t, environment.TypeStaging, result.Environment.Type)
	assert.Equal(t, prod.ID, result.Environment.ClonedFrom)
	assert.False(t, result.Environment.IsDefault)
	assert.Equal(t, "Cloned from production", result.Environment.Description)

	// Step 7: Verify counts
	assert.Equal(t, 2, result.RolesCloned)
	assert.Equal(t, 1, result.PermissionsCloned)
	assert.Equal(t, 1, result.WebhooksCloned)

	// Step 8: Verify all cloned IDs are new
	for oldID, newID := range result.RoleIDMap {
		assert.NotEqual(t, oldID, newID, "cloned role ID should be new")
	}
	for oldID, newID := range result.WebhookIDMap {
		assert.NotEqual(t, oldID, newID, "cloned webhook ID should be new")
	}

	// Step 9: List environments -> 2
	envs, err := eng.ListEnvironments(ctx, appID)
	require.NoError(t, err)
	assert.Len(t, envs, 2)

	// Step 10: Verify cloned env is persisted and retrievable
	staging, err := eng.GetEnvironment(ctx, result.Environment.ID)
	require.NoError(t, err)
	assert.Equal(t, "Staging", staging.Name)
}
