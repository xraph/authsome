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
	"github.com/xraph/authsome/core/audit/exporters"
)

// =============================================================================
// INTEGRATION TESTS
// =============================================================================
// Note: These tests require actual database connections and are skipped in
// short mode. Run with: go test -tags=integration

// TestEventCreationAndRetrieval tests basic event operations
func TestEventCreationAndRetrieval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	service, _ := setupTestService(t)
	if service == nil {
		t.Skip("Service not available for integration test")
	}

	// Create test event
	req := &audit.CreateEventRequest{
		AppID:     xid.New(),
		Action:    "user.login",
		Resource:  "/api/v1/auth",
		IPAddress: "192.168.1.1",
	}

	event, err := service.Create(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, event)
	assert.Equal(t, req.Action, event.Action)
}

// TestWebSocketStreaming tests real-time event streaming
func TestWebSocketStreaming_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	service, _ := setupTestService(t)
	if service == nil {
		t.Skip("Service not available for integration test")
	}

	// Create streaming service (would use actual PostgreSQL listener in production)
	streamService := setupTestStreamService(t)
	if streamService == nil {
		t.Skip("Stream service not available for integration test")
	}
	defer streamService.Shutdown()

	// Subscribe to stream
	appID := xid.New()
	filter := &audit.StreamFilter{
		AppID:      &appID,
		BufferSize: 10,
	}

	events, clientID, err := streamService.Subscribe(ctx, filter)
	require.NoError(t, err)
	defer streamService.Unsubscribe(clientID)

	// Create events and verify they're streamed
	done := make(chan struct{})
	receivedEvents := make([]*audit.Event, 0)

	go func() {
		timeout := time.After(5 * time.Second)
		for {
			select {
			case event := <-events:
				receivedEvents = append(receivedEvents, event)
				if len(receivedEvents) >= 3 {
					close(done)
					return
				}
			case <-timeout:
				close(done)
				return
			}
		}
	}()

	// Create test events
	for i := 0; i < 3; i++ {
		_, err := service.Create(ctx, &audit.CreateEventRequest{
			AppID:    appID,
			Action:   "user.login",
			Resource: "/api/v1/auth",
		})
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for events to be received
	<-done

	assert.GreaterOrEqual(t, len(receivedEvents), 3, "Should receive at least 3 events")
}

// TestAnomalyDetection tests anomaly detection with baseline
func TestAnomalyDetection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	service, repo := setupTestService(t)
	if service == nil || repo == nil {
		t.Skip("Service not available for integration test")
	}

	userID := xid.New()
	appID := xid.New()

	// Create baseline behavior (normal working hours: 9 AM - 5 PM, weekdays)
	baselineEvents := []struct {
		action string
		hour   int
	}{
		{"user.login", 9},
		{"data.read", 10},
		{"data.update", 11},
		{"data.read", 14},
		{"user.logout", 17},
	}

	// Create baseline events over past 7 days
	for day := 0; day < 7; day++ {
		for _, be := range baselineEvents {
			timestamp := time.Now().AddDate(0, 0, -day).Truncate(24 * time.Hour).Add(time.Duration(be.hour) * time.Hour)
			_, err := service.Create(ctx, &audit.CreateEventRequest{
				UserID:    &userID,
				AppID:     appID,
				Action:    be.action,
				Resource:  "/api/v1/data",
				CreatedAt: &timestamp,
			})
			require.NoError(t, err)
		}
	}

	// Calculate baseline
	baselineCalc := audit.NewBaselineCalculator(repo)
	baseline, err := baselineCalc.Calculate(ctx, userID, 7*24*time.Hour)
	require.NoError(t, err)
	assert.NotNil(t, baseline)

	// Test anomaly detection
	anomalyDetector := audit.NewAnomalyDetector()
	anomalyDetector.SetBaselineCalculator(baselineCalc)

	// Test 1: Unusual action (user.delete - never seen before)
	unusualEvent := &audit.Event{
		UserID:    &userID,
		AppID:     appID,
		Action:    "user.delete",
		Resource:  "/api/v1/users",
		CreatedAt: time.Now(),
	}

	anomalies, err := anomalyDetector.DetectAnomalies(ctx, unusualEvent, baseline)
	require.NoError(t, err)
	assert.NotEmpty(t, anomalies, "Should detect unusual action")
	assert.Equal(t, "unusual_action", anomalies[0].Type)

	// Test 2: Temporal anomaly (access at 3 AM)
	nightEvent := &audit.Event{
		UserID:    &userID,
		AppID:     appID,
		Action:    "data.read",
		Resource:  "/api/v1/data",
		CreatedAt: time.Now().Truncate(24 * time.Hour).Add(3 * time.Hour),
	}

	anomalies, err = anomalyDetector.DetectAnomalies(ctx, nightEvent, baseline)
	require.NoError(t, err)
	assert.NotEmpty(t, anomalies, "Should detect temporal anomaly")
	assert.Equal(t, "temporal_anomaly", anomalies[0].Type)
}

// TestRiskScoring tests risk score calculation
func TestRiskScoring_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	riskEngine := audit.NewRiskEngine()

	// Test 1: Low-risk event (normal read operation)
	lowRiskEvent := &audit.Event{
		Action:    "data.read",
		Resource:  "/api/v1/data",
		IPAddress: "192.168.1.1",
		CreatedAt: time.Now().Truncate(24 * time.Hour).Add(10 * time.Hour), // 10 AM
	}

	score, err := riskEngine.Calculate(ctx, lowRiskEvent, nil, nil)
	require.NoError(t, err)
	assert.Less(t, score.Score, 50.0, "Low-risk event should have score < 50")
	assert.Equal(t, "low", score.Level)

	// Test 2: High-risk event (user deletion with anomalies)
	highRiskEvent := &audit.Event{
		Action:    "user.delete",
		Resource:  "/api/v1/users/123",
		IPAddress: "203.0.113.0", // Different IP
		CreatedAt: time.Now().Truncate(24 * time.Hour).Add(3 * time.Hour), // 3 AM
	}

	anomalies := []*audit.Anomaly{
		{
			Type:     "unusual_action",
			Severity: "high",
			Score:    75.0,
		},
		{
			Type:     "temporal_anomaly",
			Severity: "medium",
			Score:    60.0,
		},
	}

	score, err = riskEngine.Calculate(ctx, highRiskEvent, anomalies, nil)
	require.NoError(t, err)
	assert.Greater(t, score.Score, 50.0, "High-risk event should have score > 50")
	assert.Contains(t, []string{"high", "critical"}, score.Level)
}

// TestSplunkExporter tests Splunk HEC exporter (requires Splunk instance)
func TestSplunkExporter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This would require actual Splunk instance
	t.Skip("Requires Splunk instance")
}

// TestDatadogExporter tests Datadog Logs exporter (requires Datadog API key)
func TestDatadogExporter_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This would require actual Datadog API key
	t.Skip("Requires Datadog API key")
}

// TestExportManager tests export manager with mock exporter
func TestExportManager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	ctx := context.Background()
	manager := exporters.NewExportManager()

	// Register mock exporter
	mockExporter := &MockExporter{name: "mock"}
	config := exporters.DefaultExporterConfig("mock")

	err := manager.RegisterExporter(mockExporter, config)
	require.NoError(t, err)

	// Export events
	testEvents := createTestEvents(10)
	for _, event := range testEvents {
		err := manager.Export(event)
		require.NoError(t, err)
	}

	// Give time for batch processing
	time.Sleep(2 * time.Second)

	// Check stats
	stats := manager.GetStats()
	assert.Contains(t, stats, "mock")
	assert.Greater(t, stats["mock"].EventsExported, int64(0))

	// Shutdown
	err = manager.Shutdown(5 * time.Second)
	assert.NoError(t, err)
}

// =============================================================================
// TEST HELPERS
// =============================================================================

func setupTestService(t *testing.T) (*audit.Service, audit.Repository) {
	// Would setup actual database connection in real integration tests
	// For now, return nils to skip tests
	return nil, nil
}

func setupTestStreamService(t *testing.T) *audit.PollingStreamService {
	// Setup polling-based stream service for testing
	repo := setupTestRepo(t)
	if repo == nil {
		return nil
	}
	return audit.NewPollingStreamService(repo)
}

func setupTestRepo(t *testing.T) audit.Repository {
	// Return mock repository
	return nil
}

func createTestEvents(count int) []*audit.Event {
	events := make([]*audit.Event, count)
	for i := 0; i < count; i++ {
		events[i] = &audit.Event{
			ID:        xid.New(),
			AppID:     xid.New(),
			Action:    "test.action",
			Resource:  "/api/v1/test",
			IPAddress: "192.168.1.1",
			CreatedAt: time.Now(),
		}
	}
	return events
}

// MockExporter for testing
type MockExporter struct {
	name          string
	exportedCount int
}

func (m *MockExporter) Name() string {
	return m.name
}

func (m *MockExporter) Export(ctx context.Context, events []*audit.Event) error {
	m.exportedCount += len(events)
	return nil
}

func (m *MockExporter) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockExporter) Close() error {
	return nil
}
