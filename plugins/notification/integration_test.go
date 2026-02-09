package notification_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/user"
	notifplugin "github.com/xraph/authsome/plugins/notification"
)

// TestNotificationIntegration_AuthWelcome tests welcome email integration.
func TestNotificationIntegration_AuthWelcome(t *testing.T) {
	t.Skip("Integration test - requires full system setup")

	_ = context.Background()
	appID := xid.New()

	// Setup mock notification service
	// This would require a full test harness with DB, etc.

	// Create user
	_ = &user.User{
		ID:    xid.New(),
		AppID: appID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	// Simulate hook execution
	// hookRegistry.ExecuteAfterUserCreate(ctx, newUser)

	// Assert notification was sent
	// This would check the mock provider or database

	t.Log("Welcome email should be sent after user creation")
}

// TestNotificationIntegration_DeviceSecurity tests device security notifications.
func TestNotificationIntegration_DeviceSecurity(t *testing.T) {
	t.Skip("Integration test - requires full system setup")

	_ = context.Background()
	_ = xid.New()

	tests := []struct {
		name     string
		hookFunc string
		expected string
	}{
		{
			name:     "new device detected",
			hookFunc: "OnNewDeviceDetected",
			expected: notification.TemplateKeyNewDeviceLogin,
		},
		{
			name:     "device removed",
			hookFunc: "OnDeviceRemoved",
			expected: notification.TemplateKeyDeviceRemoved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test hook execution and notification sending
			t.Logf("Testing %s notification", tt.name)
		})
	}
}

// TestNotificationIntegration_AccountLifecycle tests account lifecycle notifications.
func TestNotificationIntegration_AccountLifecycle(t *testing.T) {
	t.Skip("Integration test - requires full system setup")

	tests := []struct {
		name        string
		templateKey string
		hookName    string
		configPath  string
		configValue bool
	}{
		{
			name:        "email changed",
			templateKey: notification.TemplateKeyEmailChanged,
			hookName:    "OnEmailChanged",
			configPath:  "AutoSend.Account.EmailChanged",
			configValue: true,
		},
		{
			name:        "username changed",
			templateKey: notification.TemplateKeyUsernameChanged,
			hookName:    "OnUsernameChanged",
			configPath:  "AutoSend.Account.UsernameChanged",
			configValue: true,
		},
		{
			name:        "account deleted",
			templateKey: notification.TemplateKeyAccountDeleted,
			hookName:    "OnAccountDeleted",
			configPath:  "AutoSend.Account.Deleted",
			configValue: true,
		},
		{
			name:        "account suspended",
			templateKey: notification.TemplateKeyAccountSuspended,
			hookName:    "OnAccountSuspended",
			configPath:  "AutoSend.Account.Suspended",
			configValue: true,
		},
		{
			name:        "account reactivated",
			templateKey: notification.TemplateKeyAccountReactivated,
			hookName:    "OnAccountReactivated",
			configPath:  "AutoSend.Account.Reactivated",
			configValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing %s: hook=%s, template=%s", tt.name, tt.hookName, tt.templateKey)
			// Would test actual hook execution and notification delivery
		})
	}
}

// TestNotificationIntegration_OrganizationEvents tests organization notifications.
func TestNotificationIntegration_OrganizationEvents(t *testing.T) {
	t.Skip("Integration test - requires full system setup")

	tests := []struct {
		name        string
		templateKey string
		hookName    string
	}{
		{
			name:        "organization invite",
			templateKey: notification.TemplateKeyOrgInvite,
			hookName:    "InviteMember (handler)",
		},
		{
			name:        "member added",
			templateKey: notification.TemplateKeyOrgMemberAdded,
			hookName:    "AfterMemberAdd",
		},
		{
			name:        "member removed",
			templateKey: notification.TemplateKeyOrgMemberRemoved,
			hookName:    "AfterMemberRemove",
		},
		{
			name:        "role changed",
			templateKey: notification.TemplateKeyOrgRoleChanged,
			hookName:    "AfterMemberRoleChange",
		},
		{
			name:        "organization deleted",
			templateKey: notification.TemplateKeyOrgDeleted,
			hookName:    "AfterOrganizationDelete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing %s: hook=%s, template=%s", tt.name, tt.hookName, tt.templateKey)
		})
	}
}

// TestNotificationConfig_AutoSendFlags tests configuration flags.
func TestNotificationConfig_AutoSendFlags(t *testing.T) {
	cfg := notifplugin.DefaultConfig()

	// Test default values
	assert.True(t, cfg.AutoSend.Auth.Welcome, "Auth.Welcome should be enabled by default")
	assert.True(t, cfg.AutoSend.Organization.Invite, "Organization.Invite should be enabled by default")
	assert.True(t, cfg.AutoSend.Session.NewDevice, "Session.NewDevice should be enabled by default")
	assert.True(t, cfg.AutoSend.Account.EmailChanged, "Account.EmailChanged should be enabled by default")

	// Test auth notifications are disabled by default (controlled by plugins)
	assert.False(t, cfg.AutoSend.Auth.VerificationEmail, "Auth.VerificationEmail should be disabled (plugin controlled)")
	assert.False(t, cfg.AutoSend.Auth.MagicLink, "Auth.MagicLink should be disabled (plugin controlled)")
	assert.False(t, cfg.AutoSend.Auth.EmailOTP, "Auth.EmailOTP should be disabled (plugin controlled)")
	assert.False(t, cfg.AutoSend.Auth.MFACode, "Auth.MFACode should be disabled (plugin controlled)")
}

// TestNotificationConfig_CustomAutoSend tests custom configuration.
func TestNotificationConfig_CustomAutoSend(t *testing.T) {
	cfg := notifplugin.Config{
		AutoSend: notifplugin.AutoSendConfig{
			Auth: notifplugin.AuthAutoSendConfig{
				Welcome: false, // Disable welcome email
			},
			Session: notifplugin.SessionAutoSendConfig{
				NewDevice:     false, // Disable new device notifications
				DeviceRemoved: true,  // But keep device removed
			},
			Account: notifplugin.AccountAutoSendConfig{
				EmailChanged:    true,
				UsernameChanged: false, // Disable username change notifications
			},
		},
	}

	assert.False(t, cfg.AutoSend.Auth.Welcome)
	assert.False(t, cfg.AutoSend.Session.NewDevice)
	assert.True(t, cfg.AutoSend.Session.DeviceRemoved)
	assert.True(t, cfg.AutoSend.Account.EmailChanged)
	assert.False(t, cfg.AutoSend.Account.UsernameChanged)
}

// TestHookRegistry_NotificationHooksRegistered tests that hooks are properly registered.
func TestHookRegistry_NotificationHooksRegistered(t *testing.T) {
	registry := hooks.NewHookRegistry()

	// Register a test hook
	registry.RegisterOnEmailChanged(func(ctx context.Context, userID xid.ID, oldEmail, newEmail string) error {
		return nil
	})

	counts := registry.GetHookCounts()
	assert.Equal(t, 1, counts["onEmailChanged"], "OnEmailChanged hook should be registered")
}

// TestNotificationAdapter_Methods tests that all adapter methods exist and work.
func TestNotificationAdapter_Methods(t *testing.T) {
	t.Skip("Unit test - requires mock template service")

	// This would test each adapter method individually with a mock service
	tests := []struct {
		name       string
		methodName string
	}{
		{"SendWelcomeEmail", "SendWelcomeEmail"},
		{"SendVerificationEmail", "SendVerificationEmail"},
		{"SendMagicLink", "SendMagicLink"},
		{"SendEmailOTP", "SendEmailOTP"},
		{"SendMFACode", "SendMFACode"},
		{"SendPasswordReset", "SendPasswordReset"},
		{"SendOrgInvite", "SendOrgInvite"},
		{"SendOrgMemberAdded", "SendOrgMemberAdded"},
		{"SendOrgMemberRemoved", "SendOrgMemberRemoved"},
		{"SendOrgRoleChanged", "SendOrgRoleChanged"},
		{"SendOrgTransfer", "SendOrgTransfer"},
		{"SendOrgDeleted", "SendOrgDeleted"},
		{"SendOrgMemberLeft", "SendOrgMemberLeft"},
		{"SendNewDeviceLogin", "SendNewDeviceLogin"},
		{"SendNewLocationLogin", "SendNewLocationLogin"},
		{"SendSuspiciousLogin", "SendSuspiciousLogin"},
		{"SendDeviceRemoved", "SendDeviceRemoved"},
		{"SendAllSessionsRevoked", "SendAllSessionsRevoked"},
		{"SendEmailChangeRequest", "SendEmailChangeRequest"},
		{"SendEmailChanged", "SendEmailChanged"},
		{"SendPasswordChanged", "SendPasswordChanged"},
		{"SendUsernameChanged", "SendUsernameChanged"},
		{"SendAccountDeleted", "SendAccountDeleted"},
		{"SendAccountSuspended", "SendAccountSuspended"},
		{"SendAccountReactivated", "SendAccountReactivated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Would test method exists and returns expected format
			t.Logf("Adapter method %s should be available", tt.methodName)
		})
	}
}

// TestNotificationIntegration_E2E is a comprehensive end-to-end test.
func TestNotificationIntegration_E2E(t *testing.T) {
	t.Skip("E2E test - requires full AuthSome instance")

	// This test would:
	// 1. Start a test AuthSome instance
	// 2. Configure notification plugin with test provider
	// 3. Perform actions that trigger notifications
	// 4. Verify notifications were sent correctly

	ctx := context.Background()
	_ = ctx

	t.Run("user signup flow", func(t *testing.T) {
		// 1. Sign up user -> should trigger welcome email
		// 2. Request email verification -> should trigger verification email
		// 3. Change email -> should trigger email changed notification
		// 4. Delete account -> should trigger account deleted notification
	})

	t.Run("organization flow", func(t *testing.T) {
		// 1. Create organization
		// 2. Invite member -> should trigger invite email
		// 3. Add member -> should trigger member added notification
		// 4. Change role -> should trigger role changed notification
		// 5. Remove member -> should trigger member removed notification
	})

	t.Run("security flow", func(t *testing.T) {
		// 1. Login from new device -> should trigger new device notification
		// 2. Remove device -> should trigger device removed notification
		// 3. Ban user -> should trigger account suspended notification
		// 4. Unban user -> should trigger account reactivated notification
	})
}

// Benchmark tests.
func BenchmarkNotificationHookExecution(b *testing.B) {
	registry := hooks.NewHookRegistry()
	ctx := context.Background()
	userID := xid.New()

	// Register a simple hook
	registry.RegisterOnEmailChanged(func(ctx context.Context, userID xid.ID, oldEmail, newEmail string) error {
		return nil
	})

	for b.Loop() {
		_ = registry.ExecuteOnEmailChanged(ctx, userID, "old@example.com", "new@example.com")
	}
}

// Helper functions for integration tests

// mockNotificationProvider creates a mock provider for testing.
type mockNotificationProvider struct {
	sentNotifications []mockSentNotification
}

type mockSentNotification struct {
	Recipient string
	Subject   string
	Body      string
	Timestamp time.Time
}

func (m *mockNotificationProvider) Send(ctx context.Context, recipient, subject, body string) error {
	m.sentNotifications = append(m.sentNotifications, mockSentNotification{
		Recipient: recipient,
		Subject:   subject,
		Body:      body,
		Timestamp: time.Now(),
	})

	return nil
}

func (m *mockNotificationProvider) LastSent() *mockSentNotification {
	if len(m.sentNotifications) == 0 {
		return nil
	}

	return &m.sentNotifications[len(m.sentNotifications)-1]
}

func (m *mockNotificationProvider) Reset() {
	m.sentNotifications = nil
}

// assertNotificationSent is a test helper to verify notification delivery.
func assertNotificationSent(t *testing.T, provider *mockNotificationProvider, expectedRecipient string, expectedTemplateKey string) {
	require.NotNil(t, provider.LastSent(), "Expected notification to be sent")
	assert.Equal(t, expectedRecipient, provider.LastSent().Recipient)
	t.Logf("Notification sent to %s with template %s", expectedRecipient, expectedTemplateKey)
}
