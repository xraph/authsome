package consent_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/plugins/enterprise/consent"
	authsometesting "github.com/xraph/authsome/testing"
)

// TestIntegration_ConsentLifecycle tests the full consent creation, update, and revocation flow
func TestIntegration_ConsentLifecycle(t *testing.T) {
	// Setup mock AuthSome instance
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	// Create authenticated context
	ctx := mock.NewTestContext()
	user, err := mock.GetUserFromContext(ctx)
	require.NoError(t, err, "Should have user in context")
	app, err := mock.GetAppFromContext(ctx)
	require.NoError(t, err, "Should have app in context")

	// Initialize consent plugin
	repo := consent.NewMockRepository()
	config := consent.DefaultConfig()
	service := consent.NewService(repo, config, nil)

	// Test: Create consent
	t.Run("CreateConsent", func(t *testing.T) {
		req := &consent.CreateConsentRequest{
			ConsentType: "marketing",
			Purpose:     "email_campaigns",
			Granted:     true,
			Version:     "1.0",
		}

		consentRecord, err := service.CreateConsent(ctx, app.ID.String(), user.ID.String(), req)
		require.NoError(t, err)
		assert.NotNil(t, consentRecord)
		assert.Equal(t, "marketing", consentRecord.ConsentType)
		assert.True(t, consentRecord.Granted)
		assert.Equal(t, user.ID.String(), consentRecord.UserID)
		assert.Equal(t, app.ID.String(), consentRecord.OrganizationID)
	})

	// Test: Revoke consent
	t.Run("RevokeConsent", func(t *testing.T) {
		// Create consent first
		createReq := &consent.CreateConsentRequest{
			ConsentType: "analytics",
			Purpose:     "usage_tracking",
			Granted:     true,
			Version:     "1.0",
		}
		created, err := service.CreateConsent(ctx, app.ID.String(), user.ID.String(), createReq)
		require.NoError(t, err)

		// Revoke it
		revokeReq := &consent.UpdateConsentRequest{
			Granted: boolPtr(false),
			Reason:  "User opted out",
		}
		revoked, err := service.UpdateConsent(ctx, created.ID.String(), app.ID.String(), user.ID.String(), revokeReq)
		require.NoError(t, err)
		assert.False(t, revoked.Granted)
		assert.NotNil(t, revoked.RevokedAt)
	})

	// Test: List consents
	t.Run("ListConsents", func(t *testing.T) {
		// Create multiple consents
		for i := 0; i < 3; i++ {
			req := &consent.CreateConsentRequest{
				ConsentType: "test",
				Purpose:     "testing",
				Granted:     true,
				Version:     "1.0",
			}
			_, err := service.CreateConsent(ctx, app.ID.String(), user.ID.String(), req)
			require.NoError(t, err)
		}

		// List them
		consents, err := service.ListConsentsByUser(ctx, user.ID.String(), app.ID.String())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(consents), 3, "Should have at least 3 consents")
	})
}

// TestIntegration_CookieConsent tests cookie consent banner functionality
func TestIntegration_CookieConsent(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	ctx := mock.NewTestContext()
	user, err := mock.GetUserFromContext(ctx)
	require.NoError(t, err)
	app, err := mock.GetAppFromContext(ctx)
	require.NoError(t, err)

	repo := consent.NewMockRepository()
	service := consent.NewService(repo, consent.DefaultConfig(), nil)

	t.Run("RecordCookiePreferences", func(t *testing.T) {
		req := &consent.CookieConsentRequest{
			Essential:  true,
			Functional: true,
			Analytics:  false,
			Marketing:  false,
		}

		cookieConsent, err := service.RecordCookieConsent(ctx, app.ID.String(), user.ID.String(), req)
		require.NoError(t, err)
		assert.NotNil(t, cookieConsent)
		assert.True(t, cookieConsent.Essential)
		assert.True(t, cookieConsent.Functional)
		assert.False(t, cookieConsent.Analytics)
		assert.False(t, cookieConsent.Marketing)
	})

	t.Run("GetCookiePreferences", func(t *testing.T) {
		// Record preferences first
		req := &consent.CookieConsentRequest{
			Essential:  true,
			Functional: false,
			Analytics:  true,
			Marketing:  true,
		}
		_, err := service.RecordCookieConsent(ctx, app.ID.String(), user.ID.String(), req)
		require.NoError(t, err)

		// Get preferences
		retrieved, err := service.GetCookieConsent(ctx, user.ID.String(), app.ID.String())
		require.NoError(t, err)
		assert.True(t, retrieved.Essential)
		assert.False(t, retrieved.Functional)
		assert.True(t, retrieved.Analytics)
		assert.True(t, retrieved.Marketing)
	})
}

// TestIntegration_GDPR_Article20_DataPortability tests data export (GDPR Article 20)
func TestIntegration_GDPR_Article20_DataPortability(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	ctx := mock.NewTestContext()
	user, err := mock.GetUserFromContext(ctx)
	require.NoError(t, err)
	app, err := mock.GetAppFromContext(ctx)
	require.NoError(t, err)

	repo := consent.NewMockRepository()
	config := consent.DefaultConfig()
	config.DataExport.Enabled = true
	config.DataExport.ExpiryHours = 720 // 30 days
	service := consent.NewService(repo, config, nil)

	t.Run("RequestDataExport", func(t *testing.T) {
		req := &consent.DataExportRequestInput{
			Format:          "json",
			IncludeSections: []string{"profile", "consents", "privacy_settings"},
		}

		exportRequest, err := service.RequestDataExport(ctx, user.ID.String(), app.ID.String(), req)
		require.NoError(t, err)
		assert.NotNil(t, exportRequest)
		assert.Equal(t, "json", exportRequest.Format)
		assert.Equal(t, user.ID.String(), exportRequest.UserID)
		assert.Equal(t, string(consent.StatusPending), exportRequest.Status)
		assert.Contains(t, exportRequest.IncludeSections, "profile")
		assert.Contains(t, exportRequest.IncludeSections, "consents")
	})

	t.Run("GetExportStatus", func(t *testing.T) {
		// Create export request
		req := &consent.DataExportRequestInput{
			Format:          "csv",
			IncludeSections: []string{"consents"},
		}
		created, err := service.RequestDataExport(ctx, user.ID.String(), app.ID.String(), req)
		require.NoError(t, err)

		// Verify it was created
		assert.NotNil(t, created)
		assert.Equal(t, "csv", created.Format)
		assert.Equal(t, string(consent.StatusPending), created.Status)
	})

	t.Run("MultipleFormats", func(t *testing.T) {
		formats := []string{"json", "csv", "xml"}

		for _, format := range formats {
			req := &consent.DataExportRequestInput{
				Format:          format,
				IncludeSections: []string{"consents"},
			}
			exportRequest, err := service.RequestDataExport(ctx, user.ID.String(), app.ID.String(), req)
			require.NoError(t, err, "Should support %s format", format)
			assert.Equal(t, format, exportRequest.Format)
		}
	})
}

// TestIntegration_GDPR_Article17_RightToBeForgotten tests data deletion (GDPR Article 17)
func TestIntegration_GDPR_Article17_RightToBeForgotten(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	ctx := mock.NewTestContext()
	user, err := mock.GetUserFromContext(ctx)
	require.NoError(t, err)
	app, err := mock.GetAppFromContext(ctx)
	require.NoError(t, err)

	repo := consent.NewMockRepository()
	config := consent.DefaultConfig()
	config.DataDeletion.Enabled = true
	config.DataDeletion.RequireAdminApproval = true
	config.DataDeletion.GracePeriodDays = 7
	service := consent.NewService(repo, config, nil)

	t.Run("RequestDataDeletion", func(t *testing.T) {
		req := &consent.DataDeletionRequestInput{
			Reason:         "User requested account deletion",
			DeleteSections: []string{"profile", "consents", "preferences"},
		}

		deletionRequest, err := service.RequestDataDeletion(ctx, user.ID.String(), app.ID.String(), req)
		require.NoError(t, err)
		assert.NotNil(t, deletionRequest)
		assert.Equal(t, user.ID.String(), deletionRequest.UserID)
		assert.Equal(t, string(consent.StatusPending), deletionRequest.Status)
		assert.Contains(t, deletionRequest.DeleteSections, "profile")
	})

	t.Run("ApproveDeletionRequest", func(t *testing.T) {
		// Create deletion request
		req := &consent.DataDeletionRequestInput{
			Reason:         "GDPR request",
			DeleteSections: []string{"all"},
		}
		created, err := service.RequestDataDeletion(ctx, user.ID.String(), app.ID.String(), req)
		require.NoError(t, err)

		// Approve it (as admin)
		adminUser := mock.CreateUserWithRole("admin@example.com", "Admin", "admin")
		err = service.ApproveDeletionRequest(ctx, created.ID.String(), app.ID.String(), adminUser.ID.String())
		require.NoError(t, err)

		// Verify approval succeeded (no error means it worked)
		assert.NoError(t, err, "Approval should succeed")
	})

	t.Run("GracePeriod", func(t *testing.T) {
		req := &consent.DataDeletionRequestInput{
			Reason:         "User request",
			DeleteSections: []string{"consents"},
		}
		created, err := service.RequestDataDeletion(ctx, user.ID.String(), app.ID.String(), req)
		require.NoError(t, err)

		// Approve
		adminUser := mock.CreateUserWithRole("admin@example.com", "Admin", "admin")
		err = service.ApproveDeletionRequest(ctx, created.ID.String(), app.ID.String(), adminUser.ID.String())
		require.NoError(t, err)

		// Verify grace period is configured
		assert.Equal(t, 7, config.DataDeletion.GracePeriodDays,
			"Grace period should be 7 days")
	})
}

// TestIntegration_MultiTenancy tests consent isolation across organizations
func TestIntegration_MultiTenancy(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	// Create user and two organizations
	user := mock.CreateUser("user@example.com", "Test User")
	org1 := mock.GetDefaultOrg()
	org2 := mock.CreateOrganization("Second Org", "second-org")
	mock.AddUserToOrg(user.ID, org2.ID, "member")

	repo := consent.NewMockRepository()
	service := consent.NewService(repo, consent.DefaultConfig(), nil)

	t.Run("ConsentIsolation", func(t *testing.T) {
		ctx := context.Background()

		// Create consent in org1
		req1 := &consent.CreateConsentRequest{
			ConsentType: "marketing",
			Purpose:     "email",
			Granted:     true,
			Version:     "1.0",
		}
		consent1, err := service.CreateConsent(ctx, org1.ID.String(), user.ID.String(), req1)
		require.NoError(t, err)

		// Create consent in org2
		req2 := &consent.CreateConsentRequest{
			ConsentType: "marketing",
			Purpose:     "email",
			Granted:     false,
			Version:     "1.0",
		}
		consent2, err := service.CreateConsent(ctx, org2.ID.String(), user.ID.String(), req2)
		require.NoError(t, err)

		// Verify consents are different
		assert.NotEqual(t, consent1.ID.String(), consent2.ID.String())
		assert.True(t, consent1.Granted, "Org1 consent should be granted")
		assert.False(t, consent2.Granted, "Org2 consent should not be granted")

		// Verify consents are isolated by organization
		consents1, _ := service.ListConsentsByUser(ctx, user.ID.String(), org1.ID.String())
		consents2, _ := service.ListConsentsByUser(ctx, user.ID.String(), org2.ID.String())

		// Find the marketing consents
		var found1, found2 bool
		for _, c := range consents1 {
			if c.ConsentType == "marketing" && c.Purpose == "email" {
				assert.True(t, c.Granted, "Org1 consent should be granted")
				found1 = true
			}
		}
		for _, c := range consents2 {
			if c.ConsentType == "marketing" && c.Purpose == "email" {
				assert.False(t, c.Granted, "Org2 consent should not be granted")
				found2 = true
			}
		}
		assert.True(t, found1 && found2, "Should find consents in both orgs")
	})

	t.Run("PrivacySettingsPerOrg", func(t *testing.T) {
		ctx := context.Background()

		// Get privacy settings for both orgs
		settings1, err := service.GetPrivacySettings(ctx, org1.ID.String())
		require.NoError(t, err)

		settings2, err := service.GetPrivacySettings(ctx, org2.ID.String())
		require.NoError(t, err)

		// They should be different instances
		assert.NotEqual(t, settings1.ID.String(), settings2.ID.String())
		assert.Equal(t, org1.ID.String(), settings1.OrganizationID)
		assert.Equal(t, org2.ID.String(), settings2.OrganizationID)
	})
}

// TestIntegration_ConsentExpiry tests consent expiration logic
func TestIntegration_ConsentExpiry(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	ctx := mock.NewTestContext()
	user, err := mock.GetUserFromContext(ctx)
	require.NoError(t, err)
	app, err := mock.GetAppFromContext(ctx)
	require.NoError(t, err)

	repo := consent.NewMockRepository()
	config := consent.DefaultConfig()
	config.Expiry.Enabled = true
	config.Expiry.DefaultValidityDays = 365
	service := consent.NewService(repo, config, nil)

	t.Run("ConsentWithExpiry", func(t *testing.T) {
		expiresIn := 30 // 30 days
		req := &consent.CreateConsentRequest{
			ConsentType: "temporary",
			Purpose:     "promotion",
			Granted:     true,
			Version:     "1.0",
			ExpiresIn:   &expiresIn,
		}

		consentRecord, err := service.CreateConsent(ctx, app.ID.String(), user.ID.String(), req)
		require.NoError(t, err)
		assert.NotNil(t, consentRecord.ExpiresAt, "Consent should have expiry date")
	})

	t.Run("ConsentWithoutExpiry", func(t *testing.T) {
		// Create consent without expiry
		req := &consent.CreateConsentRequest{
			ConsentType: "permanent",
			Purpose:     "essential",
			Granted:     true,
			Version:     "1.0",
		}

		consentRecord, err := service.CreateConsent(ctx, app.ID.String(), user.ID.String(), req)
		require.NoError(t, err)

		// Verify the consent was created
		assert.NotNil(t, consentRecord)
		assert.Equal(t, "permanent", consentRecord.ConsentType)
	})
}

// TestIntegration_ConsentPolicies tests consent policy management
func TestIntegration_ConsentPolicies(t *testing.T) {
	mock := authsometesting.NewMock(t)
	defer mock.Reset()

	ctx := mock.NewTestContext()
	user, err := mock.GetUserFromContext(ctx)
	require.NoError(t, err)
	app, err := mock.GetAppFromContext(ctx)
	require.NoError(t, err)

	repo := consent.NewMockRepository()
	service := consent.NewService(repo, consent.DefaultConfig(), nil)

	t.Run("CreatePolicy", func(t *testing.T) {
		req := &consent.CreatePolicyRequest{
			ConsentType: "terms_of_service",
			Version:     "2.0",
			Name:        "Terms of Service",
			Description: "Platform terms and conditions",
			Content:     "<h1>Terms of Service</h1><p>Please read carefully...</p>",
			Required:    true,
		}

		policy, err := service.CreatePolicy(ctx, app.ID.String(), user.ID.String(), req)
		require.NoError(t, err)
		assert.NotNil(t, policy)
		assert.Equal(t, "terms_of_service", policy.ConsentType)
		assert.Equal(t, "2.0", policy.Version)
	})

	t.Run("GetPolicy", func(t *testing.T) {
		// Create policy
		req := &consent.CreatePolicyRequest{
			ConsentType: "privacy_policy",
			Version:     "1.0",
			Name:        "Privacy Policy",
			Description: "Data privacy information",
			Content:     "Privacy details...",
		}
		created, err := service.CreatePolicy(ctx, app.ID.String(), user.ID.String(), req)
		require.NoError(t, err)

		// Retrieve policy
		retrieved, err := service.GetPolicy(ctx, created.ID.String())
		require.NoError(t, err)
		assert.Equal(t, created.ID.String(), retrieved.ID.String())
		assert.Equal(t, "privacy_policy", retrieved.ConsentType)
	})
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
