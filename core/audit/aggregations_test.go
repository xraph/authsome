//go:build integration

package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// Helper to setup test environment
func setupTestDB(t *testing.T) *bun.DB {
	// TODO: Implement actual test DB setup
	// For now, skip tests that require database
	t.Skip("Skipping aggregation test - requires test database setup")
	return nil
}

// TestGetDistinctActions tests retrieving distinct actions with counts
func TestGetDistinctActions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	// Create test app and user
	appID := xid.New()
	userID := xid.New()

	// Create test events with different actions
	actions := []audit.AuditAction{
		audit.ActionAuthSignin,
		audit.ActionAuthSignin,
		audit.ActionAuthSignin,
		audit.ActionAuthSignup,
		audit.ActionAuthSignup,
		audit.ActionUserUpdated,
	}

	for _, action := range actions {
		event := &schema.AuditEvent{
			ID:       xid.New(),
			AppID:    appID,
			UserID:   &userID,
			Action:   string(action),
			Resource: "test:resource",
			Source:   schema.AuditSource(audit.SourceSystem),
		}
		err := repo.Create(ctx, event)
		require.NoError(t, err)
	}

	// Test: Get distinct actions
	filter := &audit.AggregationFilter{
		Limit: 100,
	}

	result, err := svc.GetDistinctActions(ctx, filter)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, result.Total) // 3 distinct actions

	// Verify counts (sorted by count DESC)
	assert.Equal(t, string(audit.ActionAuthSignin), result.Actions[0].Value)
	assert.Equal(t, int64(3), result.Actions[0].Count)
	assert.Equal(t, string(audit.ActionAuthSignup), result.Actions[1].Value)
	assert.Equal(t, int64(2), result.Actions[1].Count)
	assert.Equal(t, string(audit.ActionUserUpdated), result.Actions[2].Value)
	assert.Equal(t, int64(1), result.Actions[2].Count)
}

// TestGetDistinctActionsWithFilters tests filtering by org, env, time
func TestGetDistinctActionsWithFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	appID := xid.New()
	org1 := xid.New()
	org2 := xid.New()
	userID := xid.New()

	// Create events for org1
	for i := 0; i < 3; i++ {
		event := &schema.AuditEvent{
			ID:             xid.New(),
			AppID:          appID,
			OrganizationID: &org1,
			UserID:         &userID,
			Action:         string(audit.ActionAuthSignin),
			Resource:       "test:resource",
			Source:         schema.AuditSource(audit.SourceSystem),
		}
		err := repo.Create(ctx, event)
		require.NoError(t, err)
	}

	// Create events for org2
	for i := 0; i < 2; i++ {
		event := &schema.AuditEvent{
			ID:             xid.New(),
			AppID:          appID,
			OrganizationID: &org2,
			UserID:         &userID,
			Action:         string(audit.ActionAuthSignup),
			Resource:       "test:resource",
			Source:         schema.AuditSource(audit.SourceSystem),
		}
		err := repo.Create(ctx, event)
		require.NoError(t, err)
	}

	// Test: Filter by organization
	filter := &audit.AggregationFilter{
		OrganizationID: &org1,
		Limit:          100,
	}

	result, err := svc.GetDistinctActions(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, string(audit.ActionAuthSignin), result.Actions[0].Value)
	assert.Equal(t, int64(3), result.Actions[0].Count)
}

// TestGetDistinctSources tests retrieving distinct sources
func TestGetDistinctSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	appID := xid.New()
	userID := xid.New()

	// Create events with different sources
	sources := []audit.AuditSource{
		audit.SourceSystem,
		audit.SourceSystem,
		audit.SourceApplication,
		audit.SourcePlugin,
	}

	for _, source := range sources {
		event := &schema.AuditEvent{
			ID:       xid.New(),
			AppID:    appID,
			UserID:   &userID,
			Action:   string(audit.ActionAuthSignin),
			Resource: "test:resource",
			Source:   schema.AuditSource(source),
		}
		err := repo.Create(ctx, event)
		require.NoError(t, err)
	}

	// Test: Get distinct sources
	filter := &audit.AggregationFilter{
		Limit: 100,
	}

	result, err := svc.GetDistinctSources(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)

	// Verify system source has highest count
	assert.Equal(t, string(audit.SourceSystem), result.Sources[0].Value)
	assert.Equal(t, int64(2), result.Sources[0].Count)
}

// TestGetAllAggregations tests combined aggregations endpoint
func TestGetAllAggregations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	appID := xid.New()
	userID := xid.New()

	// Create diverse test data
	event := &schema.AuditEvent{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    &userID,
		Action:    string(audit.ActionAuthSignin),
		Resource:  "user:123",
		IPAddress: "192.168.1.1",
		Source:    schema.AuditSource(audit.SourceSystem),
	}
	err := repo.Create(ctx, event)
	require.NoError(t, err)

	// Test: Get all aggregations
	filter := &audit.AggregationFilter{
		Limit: 10,
	}

	result, err := svc.GetAllAggregations(ctx, filter)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify all fields are populated
	assert.NotEmpty(t, result.Actions)
	assert.NotEmpty(t, result.Sources)
	assert.NotEmpty(t, result.Resources)
	assert.NotEmpty(t, result.Users)
	assert.NotEmpty(t, result.IPAddresses)
	assert.NotEmpty(t, result.Apps)
}

// TestAggregationLimit tests limit parameter
func TestAggregationLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	appID := xid.New()
	userID := xid.New()

	// Create more events than the limit
	for i := 0; i < 15; i++ {
		event := &schema.AuditEvent{
			ID:       xid.New(),
			AppID:    appID,
			UserID:   &userID,
			Action:   string(audit.ActionAuthSignin) + "_" + string(rune(i)),
			Resource: "test:resource",
			Source:   schema.AuditSource(audit.SourceSystem),
		}
		err := repo.Create(ctx, event)
		require.NoError(t, err)
	}

	// Test: Limit to 10
	filter := &audit.AggregationFilter{
		Limit: 10,
	}

	result, err := svc.GetDistinctActions(ctx, filter)
	require.NoError(t, err)
	assert.LessOrEqual(t, result.Total, 10)
	assert.LessOrEqual(t, len(result.Actions), 10)
}

// TestAggregationTimeRange tests time-based filtering
func TestAggregationTimeRange(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	appID := xid.New()
	userID := xid.New()

	// Create old event
	oldEvent := &schema.AuditEvent{
		ID:       xid.New(),
		AppID:    appID,
		UserID:   &userID,
		Action:   string(audit.ActionAuthSignin),
		Resource: "test:resource",
		Source:   schema.AuditSource(audit.SourceSystem),
	}
	oldEvent.CreatedAt = time.Now().AddDate(0, 0, -30)
	err := repo.Create(ctx, oldEvent)
	require.NoError(t, err)

	// Create recent event
	recentEvent := &schema.AuditEvent{
		ID:       xid.New(),
		AppID:    appID,
		UserID:   &userID,
		Action:   string(audit.ActionAuthSignup),
		Resource: "test:resource",
		Source:   schema.AuditSource(audit.SourceSystem),
	}
	err = repo.Create(ctx, recentEvent)
	require.NoError(t, err)

	// Test: Filter last 7 days
	since := time.Now().AddDate(0, 0, -7)
	filter := &audit.AggregationFilter{
		Since: &since,
		Limit: 100,
	}

	result, err := svc.GetDistinctActions(ctx, filter)
	require.NoError(t, err)

	// Should only include recent event
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, string(audit.ActionAuthSignup), result.Actions[0].Value)
}

// TestGetDistinctResources tests resource aggregation
func TestGetDistinctResources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	appID := xid.New()
	userID := xid.New()

	// Create events with different resources
	resources := []string{"user:123", "user:123", "user:456", "session:789"}

	for _, resource := range resources {
		event := &schema.AuditEvent{
			ID:       xid.New(),
			AppID:    appID,
			UserID:   &userID,
			Action:   string(audit.ActionUserUpdated),
			Resource: resource,
			Source:   schema.AuditSource(audit.SourceSystem),
		}
		err := repo.Create(ctx, event)
		require.NoError(t, err)
	}

	// Test: Get distinct resources
	filter := &audit.AggregationFilter{
		Limit: 100,
	}

	result, err := svc.GetDistinctResources(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Total)

	// Verify user:123 has highest count
	assert.Equal(t, "user:123", result.Resources[0].Value)
	assert.Equal(t, int64(2), result.Resources[0].Count)
}

// TestGetDistinctUsers tests user aggregation
func TestGetDistinctUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	appID := xid.New()
	user1 := xid.New()
	user2 := xid.New()

	// Create events for different users
	for i := 0; i < 3; i++ {
		event := &schema.AuditEvent{
			ID:       xid.New(),
			AppID:    appID,
			UserID:   &user1,
			Action:   string(audit.ActionAuthSignin),
			Resource: "test:resource",
			Source:   schema.AuditSource(audit.SourceSystem),
		}
		err := repo.Create(ctx, event)
		require.NoError(t, err)
	}

	event := &schema.AuditEvent{
		ID:       xid.New(),
		AppID:    appID,
		UserID:   &user2,
		Action:   string(audit.ActionAuthSignin),
		Resource: "test:resource",
		Source:   schema.AuditSource(audit.SourceSystem),
	}
	err := repo.Create(ctx, event)
	require.NoError(t, err)

	// Test: Get distinct users
	filter := &audit.AggregationFilter{
		Limit: 100,
	}

	result, err := svc.GetDistinctUsers(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)

	// Verify user1 has highest count
	assert.Equal(t, user1.String(), result.Users[0].Value)
	assert.Equal(t, int64(3), result.Users[0].Count)
}

// TestGetDistinctIPs tests IP address aggregation
func TestGetDistinctIPs(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	appID := xid.New()
	userID := xid.New()

	// Create events from different IPs
	ips := []string{"192.168.1.1", "192.168.1.1", "10.0.0.1"}

	for _, ip := range ips {
		event := &schema.AuditEvent{
			ID:        xid.New(),
			AppID:     appID,
			UserID:    &userID,
			Action:    string(audit.ActionAuthSignin),
			Resource:  "test:resource",
			IPAddress: ip,
			Source:    schema.AuditSource(audit.SourceSystem),
		}
		err := repo.Create(ctx, event)
		require.NoError(t, err)
	}

	// Test: Get distinct IPs
	filter := &audit.AggregationFilter{
		Limit: 100,
	}

	result, err := svc.GetDistinctIPs(ctx, filter)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Total)

	// Verify 192.168.1.1 has highest count
	assert.Equal(t, "192.168.1.1", result.IPAddresses[0].Value)
	assert.Equal(t, int64(2), result.IPAddresses[0].Count)
}

// TestDefaultLimit tests default limit behavior
func TestDefaultLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	appID := xid.New()
	userID := xid.New()

	// Create one event
	event := &schema.AuditEvent{
		ID:       xid.New(),
		AppID:    appID,
		UserID:   &userID,
		Action:   string(audit.ActionAuthSignin),
		Resource: "test:resource",
		Source:   schema.AuditSource(audit.SourceSystem),
	}
	err := repo.Create(ctx, event)
	require.NoError(t, err)

	// Test: No limit specified (should use default of 100)
	filter := &audit.AggregationFilter{}

	result, err := svc.GetDistinctActions(ctx, filter)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestEmptyResults tests behavior with no data
func TestEmptyResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	if db == nil {
		return
	}
	defer db.Close()

	repo := repository.NewAuditRepository(db)
	svc := audit.NewService(repo)

	// Test: Query with no data
	filter := &audit.AggregationFilter{
		Limit: 100,
	}

	result, err := svc.GetDistinctActions(ctx, filter)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Total)
	assert.Empty(t, result.Actions)
}
