//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/apikey"
	"github.com/xraph/authsome/appclientconfig"
	"github.com/xraph/authsome/device"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/notification"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
	"github.com/xraph/authsome/webhook"
)

// createDefaultEnv creates and returns a default Production environment for an app.
func createDefaultEnv(t *testing.T, s interface {
	CreateEnvironment(context.Context, *environment.Environment) error
}, appID id.AppID,
) *environment.Environment {
	t.Helper()
	env := &environment.Environment{
		ID: id.NewEnvironmentID(), AppID: appID, Name: "Production",
		Slug: "production", Type: environment.TypeProduction, IsDefault: true,
	}
	require.NoError(t, s.CreateEnvironment(context.Background(), env))
	return env
}

// TestApp_DeleteCascade verifies that, with the app_id foreign keys recreated
// ON DELETE CASCADE, deleting an app that has children succeeds and removes
// every child row, while another app's data is left untouched.
func TestApp_DeleteCascade(t *testing.T) {
	s := setupTestStore(t)
	c := context.Background()

	a := createTestApp(t, s, "cascade-doomed")
	env := createDefaultEnv(t, s, a.ID)

	u := &user.User{
		ID: id.NewUserID(), AppID: a.ID, EnvID: env.ID,
		Email: "u@doomed.test", Username: "doomed-u", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateUser(c, u))

	sess := &session.Session{
		ID: id.NewSessionID(), AppID: a.ID, EnvID: env.ID, UserID: u.ID,
		Token: "tok_" + id.NewSessionID().String(), RefreshToken: "rtok_" + id.NewSessionID().String(),
		ExpiresAt: time.Now().Add(time.Hour), RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateSession(c, sess))

	wh := &webhook.Webhook{ID: id.NewWebhookID(), AppID: a.ID, EnvID: env.ID, URL: "https://x.test", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, s.CreateWebhook(c, wh))

	dev := &device.Device{ID: id.NewDeviceID(), AppID: a.ID, EnvID: env.ID, UserID: u.ID, Name: "phone", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, s.CreateDevice(c, dev))

	ak := &apikey.APIKey{
		ID: id.NewAPIKeyID(), AppID: a.ID, EnvID: env.ID, UserID: u.ID, Name: "key",
		KeyHash: "hash_" + id.NewAPIKeyID().String(), KeyPrefix: "pfx_" + id.NewAPIKeyID().String(),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateAPIKey(c, ak))

	notif := &notification.Notification{
		ID: id.NewNotificationID(), AppID: a.ID, EnvID: env.ID, UserID: u.ID,
		Type: "welcome", Channel: notification.ChannelEmail, CreatedAt: time.Now(),
	}
	require.NoError(t, s.CreateNotification(c, notif))

	v := &account.Verification{
		ID: id.NewVerificationID(), AppID: a.ID, EnvID: env.ID, UserID: u.ID,
		Token: "vtok_" + id.NewVerificationID().String(), Type: account.VerificationEmail,
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}
	require.NoError(t, s.CreateVerification(c, v))

	org := &organization.Organization{
		ID: id.NewOrgID(), AppID: a.ID, EnvID: env.ID, Name: "Org", Slug: "org",
		CreatedBy: u.ID, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateOrganization(c, org))

	mem := &organization.Member{ID: id.NewMemberID(), OrgID: org.ID, UserID: u.ID, Role: organization.RoleOwner, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, s.CreateMember(c, mem))

	email := &user.UserEmail{
		ID: id.NewUserEmailID(), UserID: u.ID, AppID: a.ID, EnvID: env.ID,
		Email: "u@doomed.test", IsPrimary: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.AddUserEmail(c, email))

	// App-scoped config table that has an app_id column but no foreign key.
	require.NoError(t, s.SetAppClientConfig(c, &appclientconfig.Config{AppID: a.ID}))

	// Survivor app whose data must remain after the cascade.
	other := createTestApp(t, s, "cascade-keeper")
	otherEnv := createDefaultEnv(t, s, other.ID)
	otherUser := &user.User{
		ID: id.NewUserID(), AppID: other.ID, EnvID: otherEnv.ID,
		Email: "u@keeper.test", Username: "keeper-u", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateUser(c, otherUser))

	// Act: this previously failed with a 23503 foreign-key violation.
	require.NoError(t, s.DeleteApp(c, a.ID))

	// The app and all its children are gone.
	_, err := s.GetApp(c, a.ID)
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

	envs, err := s.ListEnvironments(c, a.ID)
	require.NoError(t, err)
	assert.Empty(t, envs, "environments should be cascaded")
	whs, err := s.ListWebhooks(c, a.ID)
	require.NoError(t, err)
	assert.Empty(t, whs, "webhooks should be cascaded")
	aks, err := s.ListAPIKeysByApp(c, a.ID)
	require.NoError(t, err)
	assert.Empty(t, aks, "api keys should be cascaded")
	members, err := s.ListMembers(c, org.ID)
	require.NoError(t, err)
	assert.Empty(t, members, "org members should be cascaded")
	emails, err := s.GetUserEmails(c, u.ID)
	require.NoError(t, err)
	assert.Empty(t, emails, "user emails should be cascaded")
	_, err = s.GetAppClientConfig(c, a.ID)
	assert.ErrorIs(t, err, appclientconfig.ErrNotFound, "app client config should be removed")

	// The survivor app keeps its data.
	_, err = s.GetApp(c, other.ID)
	require.NoError(t, err, "other app must survive")
	_, err = s.GetUser(c, otherUser.ID)
	require.NoError(t, err, "other app's user must survive")
	otherEnvs, err := s.ListEnvironments(c, other.ID)
	require.NoError(t, err)
	assert.Len(t, otherEnvs, 1, "other app's environment must survive")
}

// TestApp_DeleteCascade_OAuth2 verifies the oauth2provider plugin's app_id
// foreign keys (clients, auth codes, device codes) also cascade on app delete.
func TestApp_DeleteCascade_OAuth2(t *testing.T) {
	s := setupTestStore(t)
	c := context.Background()

	// Apply the oauth2 plugin migrations alongside the core schema.
	require.NoError(t, s.Migrate(c, oauth2provider.PostgresMigrations), "migrate oauth2 plugin")

	a := createTestApp(t, s, "cascade-oauth2")
	env := createDefaultEnv(t, s, a.ID)
	u := &user.User{
		ID: id.NewUserID(), AppID: a.ID, EnvID: env.ID,
		Email: "o@oauth2.test", Username: "oauth2-u", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateUser(c, u))

	oa := oauth2provider.NewPostgresStore(s.DB())

	client := &oauth2provider.OAuth2Client{
		ID: id.NewOAuth2ClientID(), AppID: a.ID, Name: "CLI",
		ClientID:     "client_" + id.NewOAuth2ClientID().String(),
		RedirectURIs: []string{"https://app.test/cb"}, Scopes: []string{"openid"},
		GrantTypes: []string{"authorization_code"},
	}
	require.NoError(t, oa.CreateClient(c, client))

	authCode := &oauth2provider.AuthorizationCode{
		ID: id.NewAuthCodeID(), Code: "ac_" + id.NewAuthCodeID().String(),
		ClientID: client.ClientID, UserID: u.ID, AppID: a.ID,
		RedirectURI: "https://app.test/cb", Scopes: []string{"openid"},
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	require.NoError(t, oa.CreateAuthCode(c, authCode))

	deviceCode := &oauth2provider.DeviceCode{
		ID: id.NewDeviceCodeID(), DeviceCode: "dc_" + id.NewDeviceCodeID().String(),
		UserCode: "ABCD-EFGH", ClientID: client.ClientID, AppID: a.ID,
		Scopes: []string{"openid"}, ExpiresAt: time.Now().Add(10 * time.Minute),
		Interval: 5, Status: "pending",
	}
	require.NoError(t, oa.CreateDeviceCode(c, deviceCode))

	// Act: deleting the app must cascade into the plugin tables.
	require.NoError(t, s.DeleteApp(c, a.ID))

	clients, err := oa.ListClients(c, a.ID)
	require.NoError(t, err)
	assert.Empty(t, clients, "oauth2 clients should be cascaded")

	_, err = oa.GetAuthCode(c, authCode.Code)
	assert.Error(t, err, "oauth2 auth code should be cascaded")

	_, err = oa.GetDeviceCodeByDeviceCode(c, deviceCode.DeviceCode)
	assert.Error(t, err, "oauth2 device code should be cascaded")
}
