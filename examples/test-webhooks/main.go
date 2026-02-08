package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/webhook"
)

// MockWebhookEndpoint simulates a webhook endpoint for testing
type MockWebhookEndpoint struct {
	URL           string
	ResponseCode  int
	ResponseDelay time.Duration
	CallCount     int
	ReceivedData  []webhook.Event
	mutex         sync.Mutex
}

// NewMockWebhookEndpoint creates a new mock webhook endpoint
func NewMockWebhookEndpoint(url string, responseCode int, delay time.Duration) *MockWebhookEndpoint {
	return &MockWebhookEndpoint{
		URL:           url,
		ResponseCode:  responseCode,
		ResponseDelay: delay,
		ReceivedData:  make([]webhook.Event, 0),
	}
}

// ServeHTTP implements http.Handler for the mock endpoint
func (m *MockWebhookEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.CallCount++

	// Simulate processing delay
	if m.ResponseDelay > 0 {
		time.Sleep(m.ResponseDelay)
	}

	// Parse webhook payload
	var event webhook.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err == nil {
		m.ReceivedData = append(m.ReceivedData, event)
	}

	// Log the request

	// Return configured response
	w.WriteHeader(m.ResponseCode)
	if m.ResponseCode >= 200 && m.ResponseCode < 300 {
		w.Write([]byte(`{"status": "success"}`))
	} else {
		w.Write([]byte(`{"error": "simulated failure"}`))
	}
}

// GetStats returns statistics about the mock endpoint
func (m *MockWebhookEndpoint) GetStats() (int, []webhook.Event) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.CallCount, append([]webhook.Event{}, m.ReceivedData...)
}

// Reset resets the mock endpoint statistics
func (m *MockWebhookEndpoint) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.CallCount = 0
	m.ReceivedData = m.ReceivedData[:0]
}

// IntermittentEndpoint handles intermittent failures
type IntermittentEndpoint struct {
	*MockWebhookEndpoint
}

// ServeHTTP implements http.Handler for intermittent failures
func (i *IntermittentEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	i.CallCount++

	var event webhook.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err == nil {
		i.ReceivedData = append(i.ReceivedData, event)
	}

	// Fail first 2 attempts, succeed on 3rd
	if i.CallCount <= 2 {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "temporary failure"}`))
	} else {
		w.WriteHeader(200)
		w.Write([]byte(`{"status": "success"}`))
	}
}

func main() {

	// Start mock webhook servers

	// Success endpoint (always returns 200)
	successEndpoint := NewMockWebhookEndpoint("http://localhost:8081/webhook", 200, 100*time.Millisecond)

	// Failure endpoint (always returns 500)
	failureEndpoint := NewMockWebhookEndpoint("http://localhost:8082/webhook", 500, 50*time.Millisecond)

	// Slow endpoint (returns 200 but with delay)
	slowEndpoint := NewMockWebhookEndpoint("http://localhost:8083/webhook", 200, 2*time.Second)

	// Intermittent endpoint (fails first 2 times, then succeeds)
	intermittentEndpoint := &IntermittentEndpoint{
		MockWebhookEndpoint: NewMockWebhookEndpoint("http://localhost:8084/webhook", 500, 0),
	}

	// Start HTTP servers for mock endpoints
	go startMockServer(8081, successEndpoint)
	go startMockServer(8082, failureEndpoint)
	go startMockServer(8083, slowEndpoint)
	go startMockServer(8084, intermittentEndpoint)

	// Wait for servers to start
	time.Sleep(500 * time.Millisecond)

	// Test webhook delivery scenarios

	// Test 1: Successful delivery

	testSuccessfulDelivery(successEndpoint)

	// Test 2: Failed delivery with retries

	testFailedDelivery(failureEndpoint)

	// Test 3: Timeout handling

	testTimeoutHandling(slowEndpoint)

	// Test 4: Intermittent failures (eventual success)

	testIntermittentFailures(intermittentEndpoint)

	// Test 5: Multiple webhooks with different configurations

	testMultipleWebhooks(successEndpoint, failureEndpoint)

	testWebhookConfiguration()

	testWebhookEventTypes()

}

// startMockServer starts a mock HTTP server for webhook testing
func startMockServer(port int, handler http.Handler) {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("Mock server on port %d failed: %v", port, err)
	}
}

// testSuccessfulDelivery tests successful webhook delivery
func testSuccessfulDelivery(endpoint *MockWebhookEndpoint) {
	endpoint.Reset()

	// V2 Architecture: Create test app and environment IDs
	testAppID := xid.New()
	testEnvID := xid.New()

	// Create webhook configuration
	webhookConfig := &webhook.Webhook{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		URL:           endpoint.URL,
		Events:        []string{webhook.EventUserCreated, webhook.EventUserUpdated},
		Enabled:       true,
		Secret:        "test-secret-key",
		MaxRetries:    3,
		RetryBackoff:  webhook.RetryBackoffLinear,
	}

	// Create test event
	event := &webhook.Event{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		Type:          webhook.EventUserCreated,
		Data: map[string]interface{}{
			"user_id": "user-123",
			"email":   "test@example.com",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	// Simulate webhook delivery

	success := simulateWebhookDelivery(webhookConfig, event)

	// Check results
	callCount, receivedData := endpoint.GetStats()

	if success && callCount == 1 && len(receivedData) == 1 {

	} else {
	}
}

// testFailedDelivery tests webhook delivery with failures and retries
func testFailedDelivery(endpoint *MockWebhookEndpoint) {
	endpoint.Reset()

	// V2 Architecture: Create test app and environment IDs
	testAppID := xid.New()
	testEnvID := xid.New()

	webhookConfig := &webhook.Webhook{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		URL:           endpoint.URL,
		Events:        []string{webhook.EventUserDeleted},
		Enabled:       true,
		Secret:        "test-secret-key",
		MaxRetries:    3,
		RetryBackoff:  webhook.RetryBackoffLinear,
	}

	event := &webhook.Event{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		Type:          webhook.EventUserDeleted,
		Data: map[string]interface{}{
			"user_id": "user-456",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	// Simulate webhook delivery with retries
	success := simulateWebhookDeliveryWithRetries(webhookConfig, event)

	callCount, _ := endpoint.GetStats()

	if !success && callCount == 4 { // 1 initial + 3 retries

	} else {

	}
}

// testTimeoutHandling tests webhook timeout handling
func testTimeoutHandling(endpoint *MockWebhookEndpoint) {
	endpoint.Reset()

	// V2 Architecture: Create test app and environment IDs
	testAppID := xid.New()
	testEnvID := xid.New()

	webhookConfig := &webhook.Webhook{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		URL:           endpoint.URL,
		Events:        []string{webhook.EventSessionCreated},
		Enabled:       true,
		Secret:        "test-secret-key",
		MaxRetries:    1,
		RetryBackoff:  webhook.RetryBackoffLinear,
	}

	event := &webhook.Event{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		Type:          webhook.EventSessionCreated,
		Data: map[string]interface{}{
			"session_id": "session-789",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	start := time.Now()
	success := simulateWebhookDeliveryWithTimeout(webhookConfig, event, 500*time.Millisecond)
	duration := time.Since(start)

	if !success && duration < 1*time.Second {

	} else {

	}
}

// testIntermittentFailures tests handling of intermittent failures
func testIntermittentFailures(endpoint *IntermittentEndpoint) {
	endpoint.Reset()

	// V2 Architecture: Create test app and environment IDs
	testAppID := xid.New()
	testEnvID := xid.New()

	webhookConfig := &webhook.Webhook{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		URL:           endpoint.URL,
		Events:        []string{webhook.EventOrgUpdated},
		Enabled:       true,
		Secret:        "test-secret-key",
		MaxRetries:    3,
		RetryBackoff:  webhook.RetryBackoffLinear,
	}

	event := &webhook.Event{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		Type:          webhook.EventOrgUpdated,
		Data: map[string]interface{}{
			"organization_id": "org-123",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	success := simulateWebhookDeliveryWithRetries(webhookConfig, event)
	callCount, _ := endpoint.GetStats()

	if success && callCount == 3 {
	} else {

	}
}

// testMultipleWebhooks tests multiple webhook configurations
func testMultipleWebhooks(successEndpoint, failureEndpoint *MockWebhookEndpoint) {
	successEndpoint.Reset()
	failureEndpoint.Reset()

	// V2 Architecture: Create test app and environment IDs
	testAppID := xid.New()
	testEnvID := xid.New()

	webhooks := []*webhook.Webhook{
		{
			ID:            xid.New(),
			AppID:         testAppID,
			EnvironmentID: testEnvID,
			URL:           successEndpoint.URL,
			Events:        []string{webhook.EventUserLogin},
			Enabled:       true,
			Secret:        "secret-1",
			MaxRetries:    2,
			RetryBackoff:  webhook.RetryBackoffLinear,
		},
		{
			ID:            xid.New(),
			AppID:         testAppID,
			EnvironmentID: testEnvID,
			URL:           failureEndpoint.URL,
			Events:        []string{webhook.EventUserLogin},
			Enabled:       true,
			Secret:        "secret-2",
			MaxRetries:    1,
			RetryBackoff:  webhook.RetryBackoffLinear,
		},
	}

	event := &webhook.Event{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		Type:          webhook.EventUserLogin,
		Data: map[string]interface{}{
			"user_id": "user-999",
			"ip":      "192.168.1.1",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	// Simulate delivery to multiple webhooks
	successCount := 0
	for _, webhookConfig := range webhooks {
		if simulateWebhookDelivery(webhookConfig, event) {
			successCount++
		}
	}

	_, _ = successEndpoint.GetStats()
	_, _ = failureEndpoint.GetStats()

}

// testWebhookConfiguration tests webhook configuration validation
func testWebhookConfiguration() {

	// V2 Architecture: Create test app and environment IDs
	testAppID := xid.New()
	testEnvID := xid.New()

	// Test valid configuration
	validConfig := &webhook.Webhook{
		ID:            xid.New(),
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		URL:           "https://example.com/webhook",
		Events:        []string{webhook.EventUserCreated, webhook.EventUserUpdated},
		Enabled:       true,
		Secret:        "valid-secret-key",
		MaxRetries:    3,
		RetryBackoff:  webhook.RetryBackoffExponential,
	}

	if err := validateWebhookConfig(validConfig); err == nil {

	} else {

	}

	// Test invalid configurations
	invalidConfigs := []*webhook.Webhook{
		{URL: "", Events: []string{webhook.EventUserCreated}},                                            // Empty URL
		{URL: "invalid-url", Events: []string{webhook.EventUserCreated}},                                 // Invalid URL
		{URL: "https://example.com/webhook", Events: []string{}},                                         // No events
		{URL: "https://example.com/webhook", Events: []string{webhook.EventUserCreated}, MaxRetries: -1}, // Invalid retries
	}

	invalidCount := 0
	for _, config := range invalidConfigs {
		if err := validateWebhookConfig(config); err != nil {
			invalidCount++
		}
	}

}

// testWebhookEventTypes tests different webhook event types
func testWebhookEventTypes() {
	// V2 Architecture: Create test app and environment IDs
	testAppID := xid.New()
	testEnvID := xid.New()

	events := webhook.AllEventTypes()

	for _, eventType := range events {
		event := &webhook.Event{
			ID:            xid.New(),
			AppID:         testAppID,
			EnvironmentID: testEnvID,
			Type:          eventType,
			Data: map[string]interface{}{
				"test": true,
			},
			OccurredAt: time.Now(),
			CreatedAt:  time.Now(),
		}

		// Validate event structure (V2 architecture)
		if event.ID.IsNil() || event.Type == "" || event.AppID.IsNil() || event.EnvironmentID.IsNil() {

		}
	}

}

// Helper functions for simulating webhook delivery

func simulateWebhookDelivery(config *webhook.Webhook, event *webhook.Event) bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, _ := http.NewRequest("POST", config.URL, nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func simulateWebhookDeliveryWithRetries(config *webhook.Webhook, event *webhook.Event) bool {
	for attempt := 1; attempt <= config.MaxRetries+1; attempt++ {
		if simulateWebhookDelivery(config, event) {
			return true
		}

		if attempt <= config.MaxRetries {
			time.Sleep(100 * time.Millisecond) // Simple retry delay
		}
	}

	return false
}

func simulateWebhookDeliveryWithTimeout(config *webhook.Webhook, event *webhook.Event, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan bool, 1)

	go func() {
		done <- simulateWebhookDelivery(config, event)
	}()

	select {
	case success := <-done:
		return success
	case <-ctx.Done():
		return false
	}
}

func validateWebhookConfig(config *webhook.Webhook) error {
	if config.URL == "" {
		return fmt.Errorf("URL is required")
	}

	if len(config.Events) == 0 {
		return fmt.Errorf("at least one event is required")
	}

	if config.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	return nil
}
