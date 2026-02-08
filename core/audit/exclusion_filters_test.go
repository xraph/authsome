// +build integration

package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xraph/authsome/core/audit"
)

// setupAuditService sets up a test audit service
func setupAuditService(t *testing.T) *audit.Service {
	t.Skip("Skipping test - requires test infrastructure setup")
	return nil
}

// createTestApp creates a test app
func createTestApp(t *testing.T) xid.ID {
	return xid.New()
}

// createTestUser creates a test user
func createTestUser(t *testing.T, appID xid.ID) xid.ID {
	return xid.New()
}

// TestExcludeSource tests excluding specific audit sources
func TestExcludeSource(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)
	userID := createTestUser(t, appID)

	// Create events from different sources
	events := []struct {
		action audit.AuditAction
		source *audit.AuditSource
	}{
		{audit.ActionAuthSignin, ptr(audit.SourceSystem)},
		{audit.ActionAuthSignup, ptr(audit.SourceSystem)},
		{audit.ActionUserCreated, ptr(audit.SourceApplication)},
		{audit.ActionUserUpdated, ptr(audit.SourceApplication)},
		{audit.ActionAPIKeyCreated, ptr(audit.SourcePlugin)},
	}

	for _, e := range events {
		_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
			AppID:    appID,
			UserID:   &userID,
			Action:   e.action,
			Resource: "test:resource",
			Source:   e.source,
		})
		require.NoError(t, err)
	}

	// Test: Exclude system events
	excludeSource := audit.SourceSystem
	filter := &audit.ListEventsFilter{
		ExcludeSource: &excludeSource,
		Limit:         100,
	}

	response, err := auditSvc.List(ctx, filter)
	require.NoError(t, err)
	assert.Greater(t, response.Total, int64(0))

	// Verify no system events in results
	for _, event := range response.Events {
		assert.NotEqual(t, audit.SourceSystem, event.Source)
	}
}

// TestExcludeSources tests excluding multiple sources
func TestExcludeSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)
	userID := createTestUser(t, appID)

	// Create events from different sources
	events := []struct {
		action audit.AuditAction
		source *audit.AuditSource
	}{
		{audit.ActionAuthSignin, ptr(audit.SourceSystem)},
		{audit.ActionUserCreated, ptr(audit.SourceApplication)},
		{audit.ActionAPIKeyCreated, ptr(audit.SourcePlugin)},
	}

	for _, e := range events {
		_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
			AppID:    appID,
			UserID:   &userID,
			Action:   e.action,
			Resource: "test:resource",
			Source:   e.source,
		})
		require.NoError(t, err)
	}

	// Test: Exclude system and application events (show only plugin)
	filter := &audit.ListEventsFilter{
		ExcludeSources: []audit.AuditSource{audit.SourceSystem, audit.SourceApplication},
		Limit:          100,
	}

	response, err := auditSvc.List(ctx, filter)
	require.NoError(t, err)

	// Verify only plugin events in results
	for _, event := range response.Events {
		assert.Equal(t, audit.SourcePlugin, event.Source)
	}
}

// TestExcludeActions tests excluding specific actions
func TestExcludeActions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)
	userID := createTestUser(t, appID)

	// Create various events
	actions := []audit.AuditAction{
		audit.ActionAuthSignin,
		audit.ActionAuthSigninFailed,
		audit.ActionSessionChecked,
		audit.ActionSessionRefreshed,
		audit.ActionUserUpdated,
	}

	for _, action := range actions {
		_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
			AppID:    appID,
			UserID:   &userID,
			Action:   action,
			Resource: "test:resource",
		})
		require.NoError(t, err)
	}

	// Test: Exclude noisy actions (session checks and refreshes)
	filter := &audit.ListEventsFilter{
		ExcludeActions: []string{
			string(audit.ActionSessionChecked),
			string(audit.ActionSessionRefreshed),
		},
		Limit: 100,
	}

	response, err := auditSvc.List(ctx, filter)
	require.NoError(t, err)

	// Verify excluded actions not in results
	for _, event := range response.Events {
		assert.NotEqual(t, audit.ActionSessionChecked, event.Action)
		assert.NotEqual(t, audit.ActionSessionRefreshed, event.Action)
	}
}

// TestExcludeUsers tests excluding specific users
func TestExcludeUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)

	// Create test users and a regular user
	testUser1 := createTestUser(t, appID)
	testUser2 := createTestUser(t, appID)
	regularUser := createTestUser(t, appID)

	// Create events for all users
	users := []xid.ID{testUser1, testUser2, regularUser}
	for _, uid := range users {
		_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
			AppID:    appID,
			UserID:   &uid,
			Action:   audit.ActionAuthSignin,
			Resource: "test:resource",
		})
		require.NoError(t, err)
	}

	// Test: Exclude test users
	filter := &audit.ListEventsFilter{
		ExcludeUserIDs: []xid.ID{testUser1, testUser2},
		Limit:          100,
	}

	response, err := auditSvc.List(ctx, filter)
	require.NoError(t, err)

	// Verify only regular user in results
	for _, event := range response.Events {
		if event.UserID != nil {
			assert.NotEqual(t, testUser1.String(), event.UserID.String())
			assert.NotEqual(t, testUser2.String(), event.UserID.String())
		}
	}
}

// TestCombineIncludeExclude tests combining inclusion and exclusion filters
func TestCombineIncludeExclude(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)
	userID := createTestUser(t, appID)

	// Create various auth events
	actions := []audit.AuditAction{
		audit.ActionAuthSignin,
		audit.ActionAuthSigninFailed,
		audit.ActionAuthSignup,
		audit.ActionAuthSignout,
	}

	for _, action := range actions {
		_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
			AppID:    appID,
			UserID:   &userID,
			Action:   action,
			Resource: "test:resource",
		})
		require.NoError(t, err)
	}

	// Test: Include auth actions but exclude failed attempts
	authActions := []string{
		string(audit.ActionAuthSignin),
		string(audit.ActionAuthSignup),
		string(audit.ActionAuthSigninFailed),
	}
	excludeActions := []string{
		string(audit.ActionAuthSigninFailed),
	}

	filter := &audit.ListEventsFilter{
		Actions:        authActions,
		ExcludeActions: excludeActions,
		Limit:          100,
	}

	response, err := auditSvc.List(ctx, filter)
	require.NoError(t, err)

	// Verify only signin and signup (no failed) in results
	for _, event := range response.Events {
		assert.NotEqual(t, audit.ActionAuthSigninFailed, event.Action)
		// Should only have signin or signup
		isValid := event.Action == audit.ActionAuthSignin || event.Action == audit.ActionAuthSignup
		assert.True(t, isValid)
	}
}

// TestExcludeIPAddresses tests excluding specific IP addresses
func TestExcludeIPAddresses(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)
	userID := createTestUser(t, appID)

	// Create events from different IPs
	ips := []string{"192.168.1.1", "192.168.1.2", "10.0.0.1"}
	for _, ip := range ips {
		_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
			AppID:     appID,
			UserID:    &userID,
			Action:    audit.ActionAuthSignin,
			Resource:  "test:resource",
			IPAddress: ip,
		})
		require.NoError(t, err)
	}

	// Test: Exclude internal IPs
	filter := &audit.ListEventsFilter{
		ExcludeIPAddresses: []string{"192.168.1.1", "192.168.1.2"},
		Limit:              100,
	}

	response, err := auditSvc.List(ctx, filter)
	require.NoError(t, err)

	// Verify excluded IPs not in results
	for _, event := range response.Events {
		assert.NotEqual(t, "192.168.1.1", event.IPAddress)
		assert.NotEqual(t, "192.168.1.2", event.IPAddress)
	}
}

// TestExcludeResources tests excluding specific resources
func TestExcludeResources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)
	userID := createTestUser(t, appID)

	// Create events with different resources
	resources := []string{"user:123", "user:456", "session:789"}
	for _, resource := range resources {
		_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
			AppID:    appID,
			UserID:   &userID,
			Action:   audit.ActionUserUpdated,
			Resource: resource,
		})
		require.NoError(t, err)
	}

	// Test: Exclude specific user resources
	filter := &audit.ListEventsFilter{
		ExcludeResources: []string{"user:123", "user:456"},
		Limit:            100,
	}

	response, err := auditSvc.List(ctx, filter)
	require.NoError(t, err)

	// Verify excluded resources not in results
	for _, event := range response.Events {
		assert.NotEqual(t, "user:123", event.Resource)
		assert.NotEqual(t, "user:456", event.Resource)
	}
}

// TestStatisticsFilterExclusion tests exclusion in statistics queries
func TestStatisticsFilterExclusion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)
	userID := createTestUser(t, appID)

	// Create events
	actions := []audit.AuditAction{
		audit.ActionAuthSignin,
		audit.ActionAuthSigninFailed,
		audit.ActionSessionChecked,
		audit.ActionUserUpdated,
	}

	for _, action := range actions {
		for i := 0; i < 5; i++ {
			_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
				AppID:    appID,
				UserID:   &userID,
				Action:   action,
				Resource: "test:resource",
			})
			require.NoError(t, err)
		}
	}

	// Test: Get action statistics excluding noisy events
	filter := &audit.StatisticsFilter{
		ExcludeActions: []string{
			string(audit.ActionSessionChecked),
		},
		Limit: 100,
	}

	stats, err := auditSvc.GetActionStatistics(ctx, filter)
	require.NoError(t, err)

	// Verify excluded action not in statistics
	for _, stat := range stats.TopActions {
		assert.NotEqual(t, string(audit.ActionSessionChecked), stat.Action)
	}
}

// TestDeleteFilterExclusion tests exclusion in delete operations
func TestDeleteFilterExclusion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)
	userID := createTestUser(t, appID)

	// Create events
	actions := []audit.AuditAction{
		audit.ActionSessionChecked,
		audit.ActionSessionRefreshed,
		audit.ActionUserUpdated,
	}

	for _, action := range actions {
		_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
			AppID:    appID,
			UserID:   &userID,
			Action:   action,
			Resource: "test:resource",
		})
		require.NoError(t, err)
	}

	// Test: Delete old events but exclude important user updates
	oldDate := time.Now().AddDate(0, 0, -30)
	filter := &audit.DeleteFilter{
		ExcludeActions: []string{
			string(audit.ActionUserUpdated),
		},
	}

	// Note: In real test, you'd set time range for deletion
	// This is a conceptual test showing the filter structure
	_ = oldDate
	_ = filter

	// Verify the filter has exclusion set
	assert.True(t, filter.HasExclusionFilters())
}

// TestValidateExclusionFilters tests filter validation
func TestValidateExclusionFilters(t *testing.T) {
	t.Run("ListEventsFilter validation", func(t *testing.T) {
		filter := &audit.ListEventsFilter{
			ExcludeActions: []string{string(audit.ActionSessionChecked)},
		}
		err := filter.ValidateExclusionFilters()
		assert.NoError(t, err)
	})

	t.Run("StatisticsFilter validation", func(t *testing.T) {
		filter := &audit.StatisticsFilter{
			ExcludeSource: ptr(audit.SourceSystem),
		}
		err := filter.ValidateExclusionFilters()
		assert.NoError(t, err)
	})

	t.Run("DeleteFilter validation", func(t *testing.T) {
		filter := &audit.DeleteFilter{
			ExcludeResources: []string{"test:resource"},
		}
		err := filter.ValidateExclusionFilters()
		assert.NoError(t, err)
	})
}

// TestHasExclusionFilters tests the helper method
func TestHasExclusionFilters(t *testing.T) {
	t.Run("ListEventsFilter with exclusions", func(t *testing.T) {
		filter := &audit.ListEventsFilter{
			ExcludeActions: []string{string(audit.ActionSessionChecked)},
		}
		assert.True(t, filter.HasExclusionFilters())
	})

	t.Run("ListEventsFilter without exclusions", func(t *testing.T) {
		filter := &audit.ListEventsFilter{
			Actions: []string{string(audit.ActionAuthSignin)},
		}
		assert.False(t, filter.HasExclusionFilters())
	})

	t.Run("StatisticsFilter with exclusions", func(t *testing.T) {
		filter := &audit.StatisticsFilter{
			ExcludeUserID: ptr(xid.New()),
		}
		assert.True(t, filter.HasExclusionFilters())
	})

	t.Run("DeleteFilter with exclusions", func(t *testing.T) {
		filter := &audit.DeleteFilter{
			ExcludeAction: ptr(string(audit.ActionUserDeleted)),
		}
		assert.True(t, filter.HasExclusionFilters())
	})
}

// TestComplexExclusionScenario tests a complex real-world scenario
func TestComplexExclusionScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	auditSvc := setupAuditService(t)
	if auditSvc == nil {
		return
	}
	appID := createTestApp(t)
	
	// Create production and test users
	prodUser := createTestUser(t, appID)
	testUser := createTestUser(t, appID)

	// Create mix of events
	scenarios := []struct {
		user   xid.ID
		action audit.AuditAction
		source audit.AuditSource
	}{
		{prodUser, audit.ActionAuthSignin, audit.SourceSystem},
		{prodUser, audit.ActionSessionChecked, audit.SourceSystem},
		{prodUser, audit.ActionUserUpdated, audit.SourceApplication},
		{testUser, audit.ActionAuthSignin, audit.SourceSystem},
		{testUser, audit.ActionSessionChecked, audit.SourceSystem},
	}

	for _, s := range scenarios {
		_, err := auditSvc.Create(ctx, &audit.CreateEventRequest{
			AppID:    appID,
			UserID:   &s.user,
			Action:   s.action,
			Resource: "test:resource",
			Source:   &s.source,
		})
		require.NoError(t, err)
	}

	// Test: Get production user events, excluding noisy actions and system source
	excludeSource := audit.SourceSystem
	filter := &audit.ListEventsFilter{
		ExcludeUserIDs: []xid.ID{testUser},         // Exclude test user
		ExcludeSource:  &excludeSource,             // Exclude system events
		ExcludeActions: []string{                   // Exclude noisy actions
			string(audit.ActionSessionChecked),
		},
		Limit: 100,
	}

	response, err := auditSvc.List(ctx, filter)
	require.NoError(t, err)

	// Should only have production user's application events (excluding session checks)
	for _, event := range response.Events {
		// Verify test user excluded
		if event.UserID != nil {
			assert.NotEqual(t, testUser.String(), event.UserID.String())
		}
		// Verify system source excluded
		assert.NotEqual(t, audit.SourceSystem, event.Source)
		// Verify session checked excluded
		assert.NotEqual(t, audit.ActionSessionChecked, event.Action)
	}
}

// Helper function to create pointer
func ptr[T any](v T) *T {
	return &v
}
