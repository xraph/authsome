//go:build integration

package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/notification"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

// TestApp_DeleteCascade verifies that DeleteApp removes every child record
// owned by the app (the sqlite backend has no app_id foreign keys, so the
// cascade must be performed explicitly), while leaving other apps untouched.
func TestApp_DeleteCascade(t *testing.T) {
	s := setupTestStore(t)
	c := context.Background()

	// ── App under test, with a full set of children ──────────────────
	appID := id.NewAppID()
	require.NoError(t, s.CreateApp(c, &app.App{ID: appID, Name: "Doomed", Slug: "doomed"}))

	env := &environment.Environment{
		ID: id.NewEnvironmentID(), AppID: appID, Name: "Production",
		Slug: "production", Type: environment.TypeProduction, IsDefault: true,
	}
	require.NoError(t, s.CreateEnvironment(c, env))

	u := &user.User{ID: id.NewUserID(), AppID: appID, EnvID: env.ID, Email: "u@doomed.test", Username: "doomed-u"}
	require.NoError(t, s.CreateUser(c, u))

	sess := &session.Session{
		ID: id.NewSessionID(), AppID: appID, EnvID: env.ID, UserID: u.ID,
		Token: "tok", RefreshToken: "rtok",
		ExpiresAt: time.Now().Add(time.Hour), RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
	}
	require.NoError(t, s.CreateSession(c, sess))

	wh := &webhook.Webhook{ID: id.NewWebhookID(), AppID: appID, EnvID: env.ID, URL: "https://x.test"}
	require.NoError(t, s.CreateWebhook(c, wh))

	dev := &device.Device{ID: id.NewDeviceID(), AppID: appID, EnvID: env.ID, UserID: u.ID, Name: "phone"}
	require.NoError(t, s.CreateDevice(c, dev))

	ak := &apikey.APIKey{ID: id.NewAPIKeyID(), AppID: appID, EnvID: env.ID, UserID: u.ID, Name: "key", KeyHash: "h", KeyPrefix: "p"}
	require.NoError(t, s.CreateAPIKey(c, ak))

	notif := &notification.Notification{
		ID: id.NewNotificationID(), AppID: appID, EnvID: env.ID, UserID: u.ID,
		Type: "welcome", Channel: notification.ChannelEmail,
	}
	require.NoError(t, s.CreateNotification(c, notif))

	v := &account.Verification{
		ID: id.NewVerificationID(), AppID: appID, EnvID: env.ID, UserID: u.ID,
		Token: "vtok", Type: account.VerificationEmail, ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, s.CreateVerification(c, v))

	org := &organization.Organization{
		ID: id.NewOrgID(), AppID: appID, EnvID: env.ID, Name: "Org", Slug: "org", CreatedBy: u.ID,
	}
	require.NoError(t, s.CreateOrganization(c, org))

	mem := &organization.Member{ID: id.NewMemberID(), OrgID: org.ID, UserID: u.ID, Role: organization.RoleOwner}
	require.NoError(t, s.CreateMember(c, mem))

	email := &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: appID, EnvID: env.ID,
		Email: "u@doomed.test", IsPrimary: true,
	}
	require.NoError(t, s.AddUserEmail(c, email))

	// App-scoped config table without an app_id foreign key.
	require.NoError(t, s.SetAppClientConfig(c, &appclientconfig.Config{AppID: appID}))

	// ── Survivor app: its data must be untouched ─────────────────────
	otherID := id.NewAppID()
	require.NoError(t, s.CreateApp(c, &app.App{ID: otherID, Name: "Keeper", Slug: "keeper"}))
	otherEnv := &environment.Environment{
		ID: id.NewEnvironmentID(), AppID: otherID, Name: "Production",
		Slug: "production", Type: environment.TypeProduction, IsDefault: true,
	}
	require.NoError(t, s.CreateEnvironment(c, otherEnv))
	otherUser := &user.User{ID: id.NewUserID(), AppID: otherID, EnvID: otherEnv.ID, Email: "u@keeper.test", Username: "keeper-u"}
	require.NoError(t, s.CreateUser(c, otherUser))

	// ── Act ──────────────────────────────────────────────────────────
	require.NoError(t, s.DeleteApp(c, appID))

	// ── Assert: the app and all its children are gone ────────────────
	_, err := s.GetApp(c, appID)
	assert.ErrorIs(t, err, store.ErrNotFound, "app row should be deleted")

	_, err = s.GetUser(c, u.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "user should be cascaded")

	_, err = s.GetSession(c, sess.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "session should be cascaded")

	_, err = s.GetDevice(c, dev.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "device should be cascaded")

	_, err = s.GetNotification(c, notif.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "notification should be cascaded")

	_, err = s.GetVerification(c, v.Token)
	assert.ErrorIs(t, err, store.ErrNotFound, "verification should be cascaded")

	_, err = s.GetOrganization(c, org.ID)
	assert.ErrorIs(t, err, store.ErrNotFound, "organization should be cascaded")

	envs, err := s.ListEnvironments(c, appID)
	require.NoError(t, err)
	assert.Empty(t, envs, "environments should be cascaded")

	whs, err := s.ListWebhooks(c, appID)
	require.NoError(t, err)
	assert.Empty(t, whs, "webhooks should be cascaded")

	aks, err := s.ListAPIKeysByApp(c, appID)
	require.NoError(t, err)
	assert.Empty(t, aks, "api keys should be cascaded")

	members, err := s.ListMembers(c, org.ID)
	require.NoError(t, err)
	assert.Empty(t, members, "org members should be cascaded")

	emails, err := s.GetUserEmails(c, u.ID)
	require.NoError(t, err)
	assert.Empty(t, emails, "user emails should be cascaded")

	_, err = s.GetAppClientConfig(c, appID)
	assert.ErrorIs(t, err, appclientconfig.ErrNotFound, "app client config should be cascaded")

	// ── Assert: the survivor app keeps its data ──────────────────────
	_, err = s.GetApp(c, otherID)
	require.NoError(t, err, "other app must survive")
	_, err = s.GetUser(c, otherUser.ID)
	require.NoError(t, err, "other app's user must survive")
	otherEnvs, err := s.ListEnvironments(c, otherID)
	require.NoError(t, err)
	assert.Len(t, otherEnvs, 1, "other app's environment must survive")
}
