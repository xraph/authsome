package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/notification"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/store/memory"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

func newStore() *memory.Store { return memory.New() }
func ctx() context.Context    { return context.Background() }

// ──────────────────────────────────────────────────
// Lifecycle
// ──────────────────────────────────────────────────

func TestStore_Lifecycle(t *testing.T) {
	s := newStore()
	require.NoError(t, s.Migrate(ctx()))
	require.NoError(t, s.Ping(ctx()))
	require.NoError(t, s.Close())
}

// ──────────────────────────────────────────────────
// User Store
// ──────────────────────────────────────────────────

func testAppID() id.AppID { return id.NewAppID() }

func testUser(appID id.AppID) *user.User {
	return &user.User{
		ID:           id.NewUserID(),
		AppID:        appID,
		Email:        "alice@example.com",
		FirstName:    "Alice",
		Username:     "alice",
		PasswordHash: "$2a$10$fakehash",
	}
}

func TestUser_CreateAndGet(t *testing.T) {
	s := newStore()
	appID := testAppID()
	u := testUser(appID)

	require.NoError(t, s.CreateUser(ctx(), u))
	assert.False(t, u.CreatedAt.IsZero(), "CreatedAt should be set")

	got, err := s.GetUser(ctx(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, u.Email, got.Email)
}

func TestUser_GetByEmail(t *testing.T) {
	s := newStore()
	appID := testAppID()
	u := testUser(appID)
	require.NoError(t, s.CreateUser(ctx(), u))

	got, err := s.GetUserByEmail(ctx(), appID, "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), got.ID.String())
}

func TestUser_GetByEmail_NotFound(t *testing.T) {
	s := newStore()
	appID := testAppID()
	_, err := s.GetUserByEmail(ctx(), appID, "none@example.com")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestUser_GetByUsername(t *testing.T) {
	s := newStore()
	appID := testAppID()
	u := testUser(appID)
	require.NoError(t, s.CreateUser(ctx(), u))

	got, err := s.GetUserByUsername(ctx(), appID, "alice")
	require.NoError(t, err)
	assert.Equal(t, u.ID.String(), got.ID.String())
}

func TestUser_Update(t *testing.T) {
	s := newStore()
	appID := testAppID()
	u := testUser(appID)
	require.NoError(t, s.CreateUser(ctx(), u))

	u.FirstName = "Alice Updated"
	require.NoError(t, s.UpdateUser(ctx(), u))

	got, err := s.GetUser(ctx(), u.ID)
	require.NoError(t, err)
	assert.Equal(t, "Alice Updated", got.FirstName)
	assert.True(t, got.UpdatedAt.After(got.CreatedAt) || got.UpdatedAt.Equal(got.CreatedAt))
}

func TestUser_Update_NotFound(t *testing.T) {
	s := newStore()
	u := &user.User{ID: id.NewUserID()}
	assert.ErrorIs(t, s.UpdateUser(ctx(), u), store.ErrNotFound)
}

func TestUser_Delete(t *testing.T) {
	s := newStore()
	appID := testAppID()
	u := testUser(appID)
	require.NoError(t, s.CreateUser(ctx(), u))
	require.NoError(t, s.DeleteUser(ctx(), u.ID))

	_, err := s.GetUser(ctx(), u.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestUser_List(t *testing.T) {
	s := newStore()
	appID := testAppID()
	u1 := testUser(appID)
	u2 := testUser(appID)
	u2.Email = "bob@example.com"
	u2.Username = "bob"
	u2.ID = id.NewUserID()
	require.NoError(t, s.CreateUser(ctx(), u1))
	require.NoError(t, s.CreateUser(ctx(), u2))

	list, err := s.ListUsers(ctx(), &user.UserQuery{AppID: appID})
	require.NoError(t, err)
	assert.Equal(t, 2, list.Total)
	assert.Len(t, list.Users, 2)
}

// ──────────────────────────────────────────────────
// Session Store
// ──────────────────────────────────────────────────

func testSession(appID id.AppID, userID id.UserID) *session.Session {
	return &session.Session{
		ID:                    id.NewSessionID(),
		AppID:                 appID,
		UserID:                userID,
		Token:                 "tok_" + time.Now().Format(time.RFC3339Nano),
		RefreshToken:          "rtok_" + time.Now().Format(time.RFC3339Nano),
		ExpiresAt:             time.Now().Add(time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
	}
}

func TestSession_CreateAndGet(t *testing.T) {
	s := newStore()
	appID := testAppID()
	userID := id.NewUserID()
	sess := testSession(appID, userID)

	require.NoError(t, s.CreateSession(ctx(), sess))
	assert.False(t, sess.CreatedAt.IsZero())

	got, err := s.GetSession(ctx(), sess.ID)
	require.NoError(t, err)
	assert.Equal(t, sess.Token, got.Token)
}

func TestSession_GetByToken(t *testing.T) {
	s := newStore()
	appID := testAppID()
	userID := id.NewUserID()
	sess := testSession(appID, userID)
	require.NoError(t, s.CreateSession(ctx(), sess))

	got, err := s.GetSessionByToken(ctx(), sess.Token)
	require.NoError(t, err)
	assert.Equal(t, sess.ID.String(), got.ID.String())
}

func TestSession_GetByRefreshToken(t *testing.T) {
	s := newStore()
	appID := testAppID()
	userID := id.NewUserID()
	sess := testSession(appID, userID)
	require.NoError(t, s.CreateSession(ctx(), sess))

	got, err := s.GetSessionByRefreshToken(ctx(), sess.RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, sess.ID.String(), got.ID.String())
}

func TestSession_Delete(t *testing.T) {
	s := newStore()
	appID := testAppID()
	userID := id.NewUserID()
	sess := testSession(appID, userID)
	require.NoError(t, s.CreateSession(ctx(), sess))
	require.NoError(t, s.DeleteSession(ctx(), sess.ID))

	_, err := s.GetSession(ctx(), sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestSession_DeleteUserSessions(t *testing.T) {
	s := newStore()
	appID := testAppID()
	userID := id.NewUserID()
	s1 := testSession(appID, userID)
	s2 := testSession(appID, userID)
	require.NoError(t, s.CreateSession(ctx(), s1))
	require.NoError(t, s.CreateSession(ctx(), s2))

	require.NoError(t, s.DeleteUserSessions(ctx(), userID))

	sessions, err := s.ListUserSessions(ctx(), userID)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSession_ListUserSessions(t *testing.T) {
	s := newStore()
	appID := testAppID()
	userID := id.NewUserID()
	s1 := testSession(appID, userID)
	s2 := testSession(appID, userID)
	require.NoError(t, s.CreateSession(ctx(), s1))
	require.NoError(t, s.CreateSession(ctx(), s2))

	sessions, err := s.ListUserSessions(ctx(), userID)
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

// ──────────────────────────────────────────────────
// Account Store (Verification + PasswordReset)
// ──────────────────────────────────────────────────

func TestVerification_CreateAndGet(t *testing.T) {
	s := newStore()
	v := &account.Verification{
		ID:        id.NewVerificationID(),
		AppID:     testAppID(),
		UserID:    id.NewUserID(),
		Token:     "verify_token_123",
		Type:      account.VerificationEmail,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}

	require.NoError(t, s.CreateVerification(ctx(), v))

	got, err := s.GetVerification(ctx(), v.Token)
	require.NoError(t, err)
	assert.Equal(t, account.VerificationEmail, got.Type)
	assert.False(t, got.Consumed)
}

func TestVerification_Consume(t *testing.T) {
	s := newStore()
	v := &account.Verification{
		ID:    id.NewVerificationID(),
		Token: "consume_me",
	}
	require.NoError(t, s.CreateVerification(ctx(), v))
	require.NoError(t, s.ConsumeVerification(ctx(), "consume_me"))

	got, err := s.GetVerification(ctx(), "consume_me")
	require.NoError(t, err)
	assert.True(t, got.Consumed)
}

func TestVerification_Consume_NotFound(t *testing.T) {
	s := newStore()
	assert.ErrorIs(t, s.ConsumeVerification(ctx(), "nope"), store.ErrNotFound)
}

func TestPasswordReset_CreateGetConsume(t *testing.T) {
	s := newStore()
	pr := &account.PasswordReset{
		ID:    id.NewPasswordResetID(),
		Token: "reset_token_123",
	}

	require.NoError(t, s.CreatePasswordReset(ctx(), pr))

	got, err := s.GetPasswordReset(ctx(), "reset_token_123")
	require.NoError(t, err)
	assert.False(t, got.Consumed)

	require.NoError(t, s.ConsumePasswordReset(ctx(), "reset_token_123"))

	got2, err := s.GetPasswordReset(ctx(), "reset_token_123")
	require.NoError(t, err)
	assert.True(t, got2.Consumed)
}

func TestPasswordReset_NotFound(t *testing.T) {
	s := newStore()
	_, err := s.GetPasswordReset(ctx(), "nope")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

// ──────────────────────────────────────────────────
// App Store
// ──────────────────────────────────────────────────

func testApp() *app.App {
	return &app.App{
		ID:   id.NewAppID(),
		Name: "Test App",
		Slug: "test-app",
	}
}

func TestApp_CreateAndGet(t *testing.T) {
	s := newStore()
	a := testApp()
	require.NoError(t, s.CreateApp(ctx(), a))

	got, err := s.GetApp(ctx(), a.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test App", got.Name)
}

func TestApp_GetBySlug(t *testing.T) {
	s := newStore()
	a := testApp()
	require.NoError(t, s.CreateApp(ctx(), a))

	got, err := s.GetAppBySlug(ctx(), "test-app")
	require.NoError(t, err)
	assert.Equal(t, a.ID.String(), got.ID.String())
}

func TestApp_UpdateAndDelete(t *testing.T) {
	s := newStore()
	a := testApp()
	require.NoError(t, s.CreateApp(ctx(), a))

	a.Name = "Updated"
	require.NoError(t, s.UpdateApp(ctx(), a))

	got, err := s.GetApp(ctx(), a.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Name)

	require.NoError(t, s.DeleteApp(ctx(), a.ID))
	_, err = s.GetApp(ctx(), a.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestApp_List(t *testing.T) {
	s := newStore()
	a1 := testApp()
	a2 := testApp()
	a2.Slug = "second"
	require.NoError(t, s.CreateApp(ctx(), a1))
	require.NoError(t, s.CreateApp(ctx(), a2))

	apps, err := s.ListApps(ctx())
	require.NoError(t, err)
	assert.Len(t, apps, 2)
}

// ──────────────────────────────────────────────────
// Organization Store
// ──────────────────────────────────────────────────

func testOrg(appID id.AppID) *organization.Organization {
	return &organization.Organization{
		ID:    id.NewOrgID(),
		AppID: appID,
		Name:  "Test Org",
		Slug:  "test-org",
	}
}

func TestOrg_CRUD(t *testing.T) {
	s := newStore()
	appID := testAppID()
	o := testOrg(appID)

	require.NoError(t, s.CreateOrganization(ctx(), o))

	got, err := s.GetOrganization(ctx(), o.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Org", got.Name)

	got2, err := s.GetOrganizationBySlug(ctx(), appID, "test-org")
	require.NoError(t, err)
	assert.Equal(t, o.ID.String(), got2.ID.String())

	o.Name = "Updated Org"
	require.NoError(t, s.UpdateOrganization(ctx(), o))

	require.NoError(t, s.DeleteOrganization(ctx(), o.ID))
	_, err = s.GetOrganization(ctx(), o.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestOrg_List(t *testing.T) {
	s := newStore()
	appID := testAppID()
	o1 := testOrg(appID)
	o2 := testOrg(appID)
	o2.Slug = "second-org"
	require.NoError(t, s.CreateOrganization(ctx(), o1))
	require.NoError(t, s.CreateOrganization(ctx(), o2))

	orgs, err := s.ListOrganizations(ctx(), appID)
	require.NoError(t, err)
	assert.Len(t, orgs, 2)
}

// ──────────────────────────────────────────────────
// Member Store
// ──────────────────────────────────────────────────

func TestMember_CRUD(t *testing.T) {
	s := newStore()
	orgID := id.NewOrgID()
	userID := id.NewUserID()
	m := &organization.Member{
		ID:     id.NewMemberID(),
		OrgID:  orgID,
		UserID: userID,
		Role:   "admin",
	}

	require.NoError(t, s.CreateMember(ctx(), m))

	got, err := s.GetMember(ctx(), m.ID)
	require.NoError(t, err)
	assert.Equal(t, organization.MemberRole("admin"), got.Role)

	got2, err := s.GetMemberByUserAndOrg(ctx(), userID, orgID)
	require.NoError(t, err)
	assert.Equal(t, m.ID.String(), got2.ID.String())

	m.Role = "member"
	require.NoError(t, s.UpdateMember(ctx(), m))

	require.NoError(t, s.DeleteMember(ctx(), m.ID))
	_, err = s.GetMember(ctx(), m.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestMember_List(t *testing.T) {
	s := newStore()
	orgID := id.NewOrgID()
	m1 := &organization.Member{ID: id.NewMemberID(), OrgID: orgID, UserID: id.NewUserID(), Role: "admin"}
	m2 := &organization.Member{ID: id.NewMemberID(), OrgID: orgID, UserID: id.NewUserID(), Role: "member"}
	require.NoError(t, s.CreateMember(ctx(), m1))
	require.NoError(t, s.CreateMember(ctx(), m2))

	members, err := s.ListMembers(ctx(), orgID)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

// ──────────────────────────────────────────────────
// Invitation Store
// ──────────────────────────────────────────────────

func TestInvitation_CRUD(t *testing.T) {
	s := newStore()
	orgID := id.NewOrgID()
	inv := &organization.Invitation{
		ID:    id.NewInvitationID(),
		OrgID: orgID,
		Email: "invite@example.com",
		Token: "inv_token_123",
		Role:  "member",
	}

	require.NoError(t, s.CreateInvitation(ctx(), inv))

	got, err := s.GetInvitation(ctx(), inv.ID)
	require.NoError(t, err)
	assert.Equal(t, "invite@example.com", got.Email)

	got2, err := s.GetInvitationByToken(ctx(), "inv_token_123")
	require.NoError(t, err)
	assert.Equal(t, inv.ID.String(), got2.ID.String())

	inv.Role = "admin"
	require.NoError(t, s.UpdateInvitation(ctx(), inv))

	invitations, err := s.ListInvitations(ctx(), orgID)
	require.NoError(t, err)
	assert.Len(t, invitations, 1)
	assert.Equal(t, organization.MemberRole("admin"), invitations[0].Role)
}

// ──────────────────────────────────────────────────
// Team Store
// ──────────────────────────────────────────────────

func TestTeam_CRUD(t *testing.T) {
	s := newStore()
	orgID := id.NewOrgID()
	team := &organization.Team{
		ID:    id.NewTeamID(),
		OrgID: orgID,
		Name:  "Engineering",
		Slug:  "engineering",
	}

	require.NoError(t, s.CreateTeam(ctx(), team))

	got, err := s.GetTeam(ctx(), team.ID)
	require.NoError(t, err)
	assert.Equal(t, "Engineering", got.Name)

	team.Name = "Platform"
	require.NoError(t, s.UpdateTeam(ctx(), team))

	require.NoError(t, s.DeleteTeam(ctx(), team.ID))
	_, err = s.GetTeam(ctx(), team.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestTeam_List(t *testing.T) {
	s := newStore()
	orgID := id.NewOrgID()
	t1 := &organization.Team{ID: id.NewTeamID(), OrgID: orgID, Name: "A", Slug: "a"}
	t2 := &organization.Team{ID: id.NewTeamID(), OrgID: orgID, Name: "B", Slug: "b"}
	require.NoError(t, s.CreateTeam(ctx(), t1))
	require.NoError(t, s.CreateTeam(ctx(), t2))

	teams, err := s.ListTeams(ctx(), orgID)
	require.NoError(t, err)
	assert.Len(t, teams, 2)
}

// ──────────────────────────────────────────────────
// Device Store
// ──────────────────────────────────────────────────

func TestDevice_CRUD(t *testing.T) {
	s := newStore()
	userID := id.NewUserID()
	d := &device.Device{
		ID:          id.NewDeviceID(),
		UserID:      userID,
		Name:        "Chrome on macOS",
		Fingerprint: "fp_abc123",
	}

	require.NoError(t, s.CreateDevice(ctx(), d))

	got, err := s.GetDevice(ctx(), d.ID)
	require.NoError(t, err)
	assert.Equal(t, "Chrome on macOS", got.Name)

	got2, err := s.GetDeviceByFingerprint(ctx(), userID, "fp_abc123")
	require.NoError(t, err)
	assert.Equal(t, d.ID.String(), got2.ID.String())

	d.Name = "Firefox on macOS"
	require.NoError(t, s.UpdateDevice(ctx(), d))

	require.NoError(t, s.DeleteDevice(ctx(), d.ID))
	_, err = s.GetDevice(ctx(), d.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestDevice_ListUser(t *testing.T) {
	s := newStore()
	userID := id.NewUserID()
	d1 := &device.Device{ID: id.NewDeviceID(), UserID: userID, Name: "A", Fingerprint: "fp1"}
	d2 := &device.Device{ID: id.NewDeviceID(), UserID: userID, Name: "B", Fingerprint: "fp2"}
	require.NoError(t, s.CreateDevice(ctx(), d1))
	require.NoError(t, s.CreateDevice(ctx(), d2))

	devices, err := s.ListUserDevices(ctx(), userID)
	require.NoError(t, err)
	assert.Len(t, devices, 2)
}

// ──────────────────────────────────────────────────
// Webhook Store
// ──────────────────────────────────────────────────

func TestWebhook_CRUD(t *testing.T) {
	s := newStore()
	appID := testAppID()
	w := &webhook.Webhook{
		ID:    id.NewWebhookID(),
		AppID: appID,
		URL:   "https://example.com/hook",
	}

	require.NoError(t, s.CreateWebhook(ctx(), w))

	got, err := s.GetWebhook(ctx(), w.ID)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/hook", got.URL)

	w.URL = "https://example.com/v2/hook"
	require.NoError(t, s.UpdateWebhook(ctx(), w))

	require.NoError(t, s.DeleteWebhook(ctx(), w.ID))
	_, err = s.GetWebhook(ctx(), w.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestWebhook_List(t *testing.T) {
	s := newStore()
	appID := testAppID()
	w1 := &webhook.Webhook{ID: id.NewWebhookID(), AppID: appID, URL: "https://a.com"}
	w2 := &webhook.Webhook{ID: id.NewWebhookID(), AppID: appID, URL: "https://b.com"}
	require.NoError(t, s.CreateWebhook(ctx(), w1))
	require.NoError(t, s.CreateWebhook(ctx(), w2))

	hooks, err := s.ListWebhooks(ctx(), appID)
	require.NoError(t, err)
	assert.Len(t, hooks, 2)
}

// ──────────────────────────────────────────────────
// Notification Store
// ──────────────────────────────────────────────────

func TestNotification_CreateAndGet(t *testing.T) {
	s := newStore()
	n := &notification.Notification{
		ID:      id.NewNotificationID(),
		UserID:  id.NewUserID(),
		Type:    "email_verification",
		Channel: "email",
	}

	require.NoError(t, s.CreateNotification(ctx(), n))

	got, err := s.GetNotification(ctx(), n.ID)
	require.NoError(t, err)
	assert.Equal(t, "email_verification", got.Type)
	assert.False(t, got.Sent)
}

func TestNotification_MarkSent(t *testing.T) {
	s := newStore()
	n := &notification.Notification{
		ID:     id.NewNotificationID(),
		UserID: id.NewUserID(),
		Type:   "welcome",
	}
	require.NoError(t, s.CreateNotification(ctx(), n))
	require.NoError(t, s.MarkSent(ctx(), n.ID))

	got, err := s.GetNotification(ctx(), n.ID)
	require.NoError(t, err)
	assert.True(t, got.Sent)
	assert.NotNil(t, got.SentAt)
}

func TestNotification_MarkSent_NotFound(t *testing.T) {
	s := newStore()
	assert.ErrorIs(t, s.MarkSent(ctx(), id.NewNotificationID()), store.ErrNotFound)
}

func TestNotification_ListUser(t *testing.T) {
	s := newStore()
	userID := id.NewUserID()
	n1 := &notification.Notification{ID: id.NewNotificationID(), UserID: userID, Type: "a"}
	n2 := &notification.Notification{ID: id.NewNotificationID(), UserID: userID, Type: "b"}
	require.NoError(t, s.CreateNotification(ctx(), n1))
	require.NoError(t, s.CreateNotification(ctx(), n2))

	notifs, err := s.ListUserNotifications(ctx(), userID)
	require.NoError(t, err)
	assert.Len(t, notifs, 2)
}

// ──────────────────────────────────────────────────
// ListUserOrganizations
// ──────────────────────────────────────────────────

func TestListUserOrganizations(t *testing.T) {
	s := newStore()
	appID := testAppID()
	userID := id.NewUserID()

	o1 := testOrg(appID)
	o2 := testOrg(appID)
	o2.Slug = "other-org"
	require.NoError(t, s.CreateOrganization(ctx(), o1))
	require.NoError(t, s.CreateOrganization(ctx(), o2))

	// User is a member of o1 only
	m := &organization.Member{ID: id.NewMemberID(), OrgID: o1.ID, UserID: userID, Role: "member"}
	require.NoError(t, s.CreateMember(ctx(), m))

	orgs, err := s.ListUserOrganizations(ctx(), userID)
	require.NoError(t, err)
	assert.Len(t, orgs, 1)
	assert.Equal(t, o1.ID.String(), orgs[0].ID.String())
}

// ──────────────────────────────────────────────────
// Environment Store
// ──────────────────────────────────────────────────

func testEnvironment(appID id.AppID, envType environment.Type, isDefault bool) *environment.Environment {
	return &environment.Environment{
		ID:        id.NewEnvironmentID(),
		AppID:     appID,
		Name:      envType.DefaultName(),
		Slug:      envType.String(),
		Type:      envType,
		IsDefault: isDefault,
		Color:     envType.DefaultColor(),
	}
}

func TestEnvironment_CreateAndGet(t *testing.T) {
	s := newStore()
	appID := testAppID()
	env := testEnvironment(appID, environment.TypeProduction, true)

	require.NoError(t, s.CreateEnvironment(ctx(), env))
	assert.False(t, env.CreatedAt.IsZero(), "CreatedAt should be set")

	got, err := s.GetEnvironment(ctx(), env.ID)
	require.NoError(t, err)
	assert.Equal(t, env.Name, got.Name)
	assert.Equal(t, env.Type, got.Type)
}

func TestEnvironment_GetBySlug(t *testing.T) {
	s := newStore()
	appID := testAppID()
	env := testEnvironment(appID, environment.TypeProduction, true)
	require.NoError(t, s.CreateEnvironment(ctx(), env))

	got, err := s.GetEnvironmentBySlug(ctx(), appID, "production")
	require.NoError(t, err)
	assert.Equal(t, env.ID.String(), got.ID.String())
}

func TestEnvironment_GetBySlug_NotFound(t *testing.T) {
	s := newStore()
	appID := testAppID()
	_, err := s.GetEnvironmentBySlug(ctx(), appID, "nonexistent")
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestEnvironment_GetDefaultEnvironment(t *testing.T) {
	s := newStore()
	appID := testAppID()
	env := testEnvironment(appID, environment.TypeProduction, true)
	require.NoError(t, s.CreateEnvironment(ctx(), env))

	got, err := s.GetDefaultEnvironment(ctx(), appID)
	require.NoError(t, err)
	assert.Equal(t, env.ID.String(), got.ID.String())
	assert.True(t, got.IsDefault)
}

func TestEnvironment_GetDefaultEnvironment_NotFound(t *testing.T) {
	s := newStore()
	appID := testAppID()

	// Create non-default environment.
	env := testEnvironment(appID, environment.TypeDevelopment, false)
	require.NoError(t, s.CreateEnvironment(ctx(), env))

	_, err := s.GetDefaultEnvironment(ctx(), appID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestEnvironment_ListEnvironments(t *testing.T) {
	s := newStore()
	appID := testAppID()
	otherAppID := testAppID()

	env1 := testEnvironment(appID, environment.TypeProduction, true)
	env2 := testEnvironment(appID, environment.TypeDevelopment, false)
	env3 := testEnvironment(otherAppID, environment.TypeProduction, true)
	require.NoError(t, s.CreateEnvironment(ctx(), env1))
	require.NoError(t, s.CreateEnvironment(ctx(), env2))
	require.NoError(t, s.CreateEnvironment(ctx(), env3))

	list, err := s.ListEnvironments(ctx(), appID)
	require.NoError(t, err)
	assert.Len(t, list, 2, "should only return environments for the given AppID")
}

func TestEnvironment_UpdateEnvironment(t *testing.T) {
	s := newStore()
	appID := testAppID()
	env := testEnvironment(appID, environment.TypeProduction, true)
	require.NoError(t, s.CreateEnvironment(ctx(), env))

	env.Name = "Updated Production"
	require.NoError(t, s.UpdateEnvironment(ctx(), env))

	got, err := s.GetEnvironment(ctx(), env.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Production", got.Name)
	assert.True(t, got.UpdatedAt.After(got.CreatedAt) || got.UpdatedAt.Equal(got.CreatedAt))
}

func TestEnvironment_UpdateEnvironment_NotFound(t *testing.T) {
	s := newStore()
	env := &environment.Environment{ID: id.NewEnvironmentID()}
	assert.ErrorIs(t, s.UpdateEnvironment(ctx(), env), store.ErrNotFound)
}

func TestEnvironment_DeleteEnvironment(t *testing.T) {
	s := newStore()
	appID := testAppID()
	env := testEnvironment(appID, environment.TypeDevelopment, false)
	require.NoError(t, s.CreateEnvironment(ctx(), env))

	require.NoError(t, s.DeleteEnvironment(ctx(), env.ID))

	_, err := s.GetEnvironment(ctx(), env.ID)
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestEnvironment_DeleteEnvironment_Default(t *testing.T) {
	s := newStore()
	appID := testAppID()
	env := testEnvironment(appID, environment.TypeProduction, true)
	require.NoError(t, s.CreateEnvironment(ctx(), env))

	err := s.DeleteEnvironment(ctx(), env.ID)
	assert.Error(t, err, "should not allow deleting the default environment")
	assert.Contains(t, err.Error(), "default")
}

func TestEnvironment_DeleteEnvironment_NotFound(t *testing.T) {
	s := newStore()
	err := s.DeleteEnvironment(ctx(), id.NewEnvironmentID())
	assert.ErrorIs(t, err, store.ErrNotFound)
}

func TestEnvironment_SetDefaultEnvironment(t *testing.T) {
	s := newStore()
	appID := testAppID()

	prod := testEnvironment(appID, environment.TypeProduction, true)
	dev := testEnvironment(appID, environment.TypeDevelopment, false)
	require.NoError(t, s.CreateEnvironment(ctx(), prod))
	require.NoError(t, s.CreateEnvironment(ctx(), dev))

	// Switch default from prod to dev.
	require.NoError(t, s.SetDefaultEnvironment(ctx(), appID, dev.ID))

	gotProd, err := s.GetEnvironment(ctx(), prod.ID)
	require.NoError(t, err)
	assert.False(t, gotProd.IsDefault, "old default should be cleared")

	gotDev, err := s.GetEnvironment(ctx(), dev.ID)
	require.NoError(t, err)
	assert.True(t, gotDev.IsDefault, "new default should be set")
}

func TestEnvironment_SetDefaultEnvironment_NotFound(t *testing.T) {
	s := newStore()
	appID := testAppID()
	err := s.SetDefaultEnvironment(ctx(), appID, id.NewEnvironmentID())
	assert.ErrorIs(t, err, store.ErrNotFound)
}
