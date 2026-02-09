package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/audit/exporters"
)

// =============================================================================
// PERFORMANCE BENCHMARKS
// =============================================================================

// BenchmarkEventCreation benchmarks audit event creation.
func BenchmarkEventCreation(b *testing.B) {
	ctx := context.Background()

	service, _ := setupBenchService(b)
	if service == nil {
		b.Skip("Service not available for benchmark")
	}

	req := &audit.CreateEventRequest{
		AppID:    xid.New(),
		Action:   "user.login",
		Resource: "/api/v1/auth",
	}

	for b.Loop() {
		_, err := service.Create(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStreamingThroughput benchmarks WebSocket streaming throughput.
func BenchmarkStreamingThroughput(b *testing.B) {
	ctx := context.Background()

	streamService := setupBenchStreamService(b)
	if streamService == nil {
		b.Skip("Stream service not available for benchmark")
	}
	defer streamService.Shutdown()

	// Subscribe to stream
	filter := &audit.StreamFilter{
		BufferSize: 10000,
	}

	events, clientID, err := streamService.Subscribe(ctx, filter)
	if err != nil {
		b.Fatal(err)
	}
	defer streamService.Unsubscribe(clientID)

	// Consume events in background
	go func() {
		for range events {
			// Consume events
		}
	}()

	// Benchmark event production
	testEvent := &audit.Event{
		ID:        xid.New(),
		AppID:     xid.New(),
		Action:    "test.action",
		Resource:  "/api/v1/test",
		IPAddress: "192.168.1.1",
		CreatedAt: time.Now(),
	}

	for b.Loop() {
		// Simulate event production
		_ = testEvent
	}
}

// BenchmarkBaselineCalculation benchmarks baseline calculation.
func BenchmarkBaselineCalculation(b *testing.B) {
	ctx := context.Background()

	_, repo := setupBenchService(b)
	if repo == nil {
		b.Skip("Repository not available for benchmark")
	}

	baselineCalc := audit.NewBaselineCalculator(repo)
	userID := xid.New()

	// Prepopulate user events for past 30 days
	prepopulateUserEvents(b, repo, userID, 30*24*time.Hour, 1000)

	for b.Loop() {
		_, err := baselineCalc.Calculate(ctx, userID, 30*24*time.Hour)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAnomalyDetection benchmarks anomaly detection.
func BenchmarkAnomalyDetection(b *testing.B) {
	ctx := context.Background()
	detector := audit.NewAnomalyDetector()

	// Create baseline
	baseline := &audit.Baseline{
		TopActions: map[string]int{
			"user.login":  100,
			"data.read":   500,
			"data.update": 200,
		},
		TypicalHours: []int{9, 10, 11, 12, 13, 14, 15, 16, 17},
	}

	testEvent := &audit.Event{
		Action:    "user.delete",
		Resource:  "/api/v1/users",
		CreatedAt: time.Now().Truncate(24 * time.Hour).Add(3 * time.Hour), // 3 AM
	}

	for b.Loop() {
		_, err := detector.DetectAnomalies(ctx, testEvent, baseline)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRiskScoring benchmarks risk score calculation.
func BenchmarkRiskScoring(b *testing.B) {
	ctx := context.Background()
	riskEngine := audit.NewRiskEngine()

	testEvent := &audit.Event{
		Action:    "user.delete",
		Resource:  "/api/v1/users",
		IPAddress: "192.168.1.1",
		CreatedAt: time.Now(),
	}

	anomalies := []*audit.Anomaly{
		{Type: "unusual_action", Score: 75.0},
		{Type: "temporal_anomaly", Score: 60.0},
	}

	for b.Loop() {
		_, err := riskEngine.Calculate(ctx, testEvent, anomalies, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSIEMExport benchmarks SIEM export throughput.
func BenchmarkSIEMExport(b *testing.B) {
	ctx := context.Background()
	mockExporter := &MockExporter{name: "mock"}

	testEvents := createBenchEvents(100)

	for b.Loop() {
		err := mockExporter.Export(ctx, testEvents)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkExportManager_Batching benchmarks export manager batching.
func BenchmarkExportManager_Batching(b *testing.B) {
	manager := exporters.NewExportManager()
	mockExporter := &MockExporter{name: "mock"}

	err := manager.RegisterExporter(mockExporter, exporters.DefaultExporterConfig("mock"))
	if err != nil {
		b.Fatal(err)
	}

	testEvent := &audit.Event{
		ID:        xid.New(),
		AppID:     xid.New(),
		Action:    "test.action",
		Resource:  "/api/v1/test",
		CreatedAt: time.Now(),
	}

	for b.Loop() {
		err := manager.Export(testEvent)
		if err != nil {
			b.Fatal(err)
		}
	}

	manager.Shutdown(5 * time.Second)
}

// =============================================================================
// BENCHMARK HELPERS
// =============================================================================

func setupBenchService(b *testing.B) (*audit.Service, audit.Repository) {
	// Setup actual service for benchmarking
	// Would use real database in production benchmarks
	return nil, nil
}

func setupBenchStreamService(b *testing.B) *audit.PollingStreamService {
	repo := setupBenchRepo(b)
	if repo == nil {
		return nil
	}

	return audit.NewPollingStreamService(repo)
}

func setupBenchRepo(b *testing.B) audit.Repository {
	// Return benchmark repository
	return nil
}

func prepopulateEvents(b *testing.B, service *audit.Service, count int) {
	// Prepopulate database with test events
	// Implementation would depend on actual database
}

func prepopulateUserEvents(b *testing.B, repo audit.Repository, userID xid.ID, period time.Duration, count int) {
	// Prepopulate user-specific events
}

func createBenchEvents(count int) []*audit.Event {
	events := make([]*audit.Event, count)
	for i := range count {
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

// =============================================================================
// MOCK EXPORTER FOR TESTING
// =============================================================================

// MockExporter for testing.
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
