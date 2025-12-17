package builder

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAllTemplatesRender verifies all 39 templates can be rendered
func TestAllTemplatesRender(t *testing.T) {
	expectedTemplates := []string{
		// Existing 8 templates
		"welcome", "otp", "reset_password", "invitation", "magic_link",
		"order_confirmation", "newsletter", "account_alert",
		// Organization templates (7)
		"org_invite", "org_member_added", "org_member_removed", "org_role_changed",
		"org_transfer", "org_deleted", "org_member_left",
		// Account templates (8)
		"email_change_request", "email_changed", "password_changed", "username_changed",
		"account_deleted", "account_suspended", "account_reactivated", "data_export_ready",
		// Session templates (5)
		"new_device_login", "new_location_login", "suspicious_login",
		"device_removed", "all_sessions_revoked",
		// Reminder templates (5)
		"verification_reminder", "inactive_account", "trial_expiring",
		"subscription_expiring", "password_expiring",
		// Admin templates (6)
		"account_locked", "account_unlocked", "terms_update",
		"privacy_update", "maintenance_scheduled", "security_breach",
	}

	assert.Equal(t, 39, len(expectedTemplates), "Should have 39 total templates")
	assert.Equal(t, 39, len(SampleTemplates), "SampleTemplates should contain all 39 templates")

	for _, templateName := range expectedTemplates {
		t.Run(templateName, func(t *testing.T) {
			doc, err := GetSampleTemplate(templateName)
			assert.NoError(t, err, "Template %s should exist", templateName)
			assert.NotNil(t, doc, "Template %s should not be nil", templateName)

			// Verify template can be rendered to HTML
			renderer := NewRenderer(doc)
			html, err := renderer.RenderToHTML()
			assert.NoError(t, err, "Template %s should render to HTML", templateName)
			assert.NotEmpty(t, html, "Template %s should produce non-empty HTML", templateName)

			// Verify HTML contains basic structure (case-insensitive for doctype)
			assert.True(t, len(html) > 100, "Template %s should produce substantial HTML", templateName)
			assert.Contains(t, html, "<body", "Template %s should have body tag", templateName)
			assert.Contains(t, html, "</html>", "Template %s should close html tag", templateName)

			fmt.Printf("✓ Template %s renders successfully (%d bytes)\n", templateName, len(html))
		})
	}
}

// TestTemplateVariableSubstitution verifies variables can be substituted
func TestTemplateVariableSubstitution(t *testing.T) {
	testCases := []struct {
		templateName  string
		variables     map[string]interface{}
		shouldContain []string
	}{
		{
			templateName: "org_invite",
			variables: map[string]interface{}{
				"userName":    "John Doe",
				"inviterName": "Jane Smith",
				"orgName":     "Acme Corp",
				"role":        "Developer",
				"inviteURL":   "https://example.com/invite/123",
				"expiresIn":   "7 days",
			},
			shouldContain: []string{"Jane Smith", "Acme Corp", "Developer"},
		},
		{
			templateName: "email_change_request",
			variables: map[string]interface{}{
				"userName":   "John Doe",
				"oldEmail":   "old@example.com",
				"newEmail":   "new@example.com",
				"confirmURL": "https://example.com/confirm",
			},
			shouldContain: []string{"John Doe", "old@example.com", "new@example.com"},
		},
		{
			templateName: "new_device_login",
			variables: map[string]interface{}{
				"userName":    "John Doe",
				"deviceName":  "iPhone 15",
				"location":    "San Francisco, CA",
				"browserName": "Safari",
			},
			shouldContain: []string{"John Doe", "iPhone 15", "San Francisco"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.templateName, func(t *testing.T) {
			html, err := RenderTemplate(SampleTemplates[tc.templateName], tc.variables)
			assert.NoError(t, err)
			assert.NotEmpty(t, html)

			for _, expected := range tc.shouldContain {
				assert.Contains(t, html, expected, "Rendered template should contain '%s'", expected)
			}

			fmt.Printf("✓ Template %s variable substitution works\n", tc.templateName)
		})
	}
}

// TestTemplateCategories verifies templates are organized correctly
func TestTemplateCategories(t *testing.T) {
	orgTemplates := []string{"org_invite", "org_member_added", "org_member_removed", "org_role_changed", "org_transfer", "org_deleted", "org_member_left"}
	accountTemplates := []string{"email_change_request", "email_changed", "password_changed", "username_changed", "account_deleted", "account_suspended", "account_reactivated", "data_export_ready"}
	sessionTemplates := []string{"new_device_login", "new_location_login", "suspicious_login", "device_removed", "all_sessions_revoked"}
	reminderTemplates := []string{"verification_reminder", "inactive_account", "trial_expiring", "subscription_expiring", "password_expiring"}
	adminTemplates := []string{"account_locked", "account_unlocked", "terms_update", "privacy_update", "maintenance_scheduled", "security_breach"}

	assert.Equal(t, 7, len(orgTemplates), "Should have 7 org templates")
	assert.Equal(t, 8, len(accountTemplates), "Should have 8 account templates")
	assert.Equal(t, 5, len(sessionTemplates), "Should have 5 session templates")
	assert.Equal(t, 5, len(reminderTemplates), "Should have 5 reminder templates")
	assert.Equal(t, 6, len(adminTemplates), "Should have 6 admin templates")

	// Verify all exist in SampleTemplates
	allNewTemplates := append(append(append(append(orgTemplates, accountTemplates...), sessionTemplates...), reminderTemplates...), adminTemplates...)
	for _, template := range allNewTemplates {
		_, exists := SampleTemplates[template]
		assert.True(t, exists, "Template %s should exist in SampleTemplates", template)
	}
}
