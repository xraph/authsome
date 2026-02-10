package notification

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetDefaultTemplateMetadata verifies all 39 templates are available.
func TestGetDefaultTemplateMetadata(t *testing.T) {
	templates := GetDefaultTemplateMetadata()

	// Should have 39 total templates (8 original + 31 new)
	assert.Len(t, templates, 39, "Should have 39 total templates")

	// Verify templates have required fields
	for _, template := range templates {
		assert.NotEmpty(t, template.Key, "Template key should not be empty")
		assert.NotEmpty(t, template.Name, "Template name should not be empty")
		assert.NotEmpty(t, template.Description, "Template description should not be empty")
		assert.NotEmpty(t, template.DefaultBody, "Template body should not be empty")
		assert.NotEmpty(t, template.Variables, "Template should have variables defined")

		// Email templates should have subject and HTML
		if template.Type == NotificationTypeEmail {
			assert.NotEmpty(t, template.DefaultSubject, "Email template %s should have subject", template.Key)
			assert.NotEmpty(t, template.DefaultBodyHTML, "Email template %s should have HTML body", template.Key)
			assert.Greater(t, len(template.DefaultBodyHTML), 500, "Email template %s HTML should be substantial", template.Key)
		}

		// SMS templates may not have subject or HTML
		if template.Type == NotificationTypeSMS {
			assert.Greater(t, len(template.DefaultBody), 10, "SMS template %s should have body text", template.Key)
		}
	}
}

// TestTemplateKeyConstants verifies all constants are unique and properly formatted.
func TestTemplateKeyConstants(t *testing.T) {
	keys := []string{
		// Auth templates
		TemplateKeyWelcome, TemplateKeyVerifyEmail, TemplateKeyPasswordReset,
		TemplateKeyMFACode, TemplateKeyMagicLink, TemplateKeyEmailOTP,
		TemplateKeyPhoneOTP, TemplateKeySecurityAlert,
		// Org templates
		TemplateKeyOrgInvite, TemplateKeyOrgMemberAdded, TemplateKeyOrgMemberRemoved,
		TemplateKeyOrgRoleChanged, TemplateKeyOrgTransfer, TemplateKeyOrgDeleted,
		TemplateKeyOrgMemberLeft,
		// Account templates
		TemplateKeyEmailChangeRequest, TemplateKeyEmailChanged, TemplateKeyPasswordChanged,
		TemplateKeyUsernameChanged, TemplateKeyAccountDeleted, TemplateKeyAccountSuspended,
		TemplateKeyAccountReactivated, TemplateKeyDataExportReady,
		// Session templates
		TemplateKeyNewDeviceLogin, TemplateKeyNewLocationLogin, TemplateKeySuspiciousLogin,
		TemplateKeyDeviceRemoved, TemplateKeyAllSessionsRevoked,
		// Reminder templates
		TemplateKeyVerificationReminder, TemplateKeyInactiveAccount, TemplateKeyTrialExpiring,
		TemplateKeySubscriptionExpiring, TemplateKeyPasswordExpiring,
		// Admin templates
		TemplateKeyAccountLocked, TemplateKeyAccountUnlocked, TemplateKeyTermsUpdate,
		TemplateKeyPrivacyUpdate, TemplateKeyMaintenanceScheduled, TemplateKeySecurityBreach,
	}

	assert.Len(t, keys, 39, "Should have 39 template keys")

	// Verify all keys are unique
	keyMap := make(map[string]bool)
	for _, key := range keys {
		assert.False(t, keyMap[key], "Template key %s should be unique", key)
		keyMap[key] = true
	}

	// Verify all keys follow naming convention (category.name)
	for _, key := range keys {
		assert.Contains(t, key, ".", "Template key %s should contain category separator", key)
	}
}

// TestGetDefaultTemplate verifies individual template retrieval.
func TestGetDefaultTemplate(t *testing.T) {
	testCases := []struct {
		key         string
		shouldExist bool
	}{
		{TemplateKeyOrgInvite, true},
		{TemplateKeyEmailChanged, true},
		{TemplateKeyNewDeviceLogin, true},
		{TemplateKeyTrialExpiring, true},
		{TemplateKeyAccountLocked, true},
		{"nonexistent.template", false},
	}

	for _, tc := range testCases {
		t.Run(tc.key, func(t *testing.T) {
			template, err := GetDefaultTemplate(tc.key)
			if tc.shouldExist {
				require.NoError(t, err, "Template %s should exist", tc.key)
				assert.NotNil(t, template, "Template %s should not be nil", tc.key)
				assert.Equal(t, tc.key, template.Key, "Template key should match")
			} else {
				require.Error(t, err, "Non-existent template should return error")
				assert.Nil(t, template, "Non-existent template should be nil")
			}
		})
	}
}

// TestValidateTemplateKey verifies template key validation.
func TestValidateTemplateKey(t *testing.T) {
	validKeys := []string{
		TemplateKeyOrgInvite,
		TemplateKeyEmailChanged,
		TemplateKeyNewDeviceLogin,
	}

	invalidKeys := []string{
		"invalid.key",
		"",
		"random",
	}

	for _, key := range validKeys {
		assert.True(t, ValidateTemplateKey(key), "Key %s should be valid", key)
	}

	for _, key := range invalidKeys {
		assert.False(t, ValidateTemplateKey(key), "Key %s should be invalid", key)
	}
}

// TestGetTemplateKeysByType verifies filtering by notification type.
func TestGetTemplateKeysByType(t *testing.T) {
	emailKeys := GetTemplateKeysByType(NotificationTypeEmail)
	smsKeys := GetTemplateKeysByType(NotificationTypeSMS)

	// Most templates should be email
	assert.Greater(t, len(emailKeys), 35, "Should have many email templates")

	// Only phone OTP is SMS
	assert.Len(t, smsKeys, 1, "Should have 1 SMS template")
	assert.Contains(t, smsKeys, TemplateKeyPhoneOTP, "SMS templates should include phone OTP")
}
