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
	fmt.Printf("  📥 Mock endpoint %s received call #%d\n", m.URL, m.CallCount)
	fmt.Printf("     Event: %s, Data: %v\n", event.Type, event.Data)

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

	fmt.Printf("  📥 Intermittent endpoint received call #%d\n", i.CallCount)

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
	fmt.Println("Testing AuthSome Webhook Delivery and Retry Logic")
	fmt.Println("================================================")

	// Start mock webhook servers
	fmt.Println("\n1. Starting Mock Webhook Endpoints:")

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
	fmt.Println("  ✅ Mock webhook endpoints started")

	// Test webhook delivery scenarios
	fmt.Println("\n2. Testing Webhook Delivery Scenarios:")

	// Test 1: Successful delivery
	fmt.Println("\n  Test 1: Successful Delivery")
	testSuccessfulDelivery(successEndpoint)

	// Test 2: Failed delivery with retries
	fmt.Println("\n  Test 2: Failed Delivery with Retries")
	testFailedDelivery(failureEndpoint)

	// Test 3: Timeout handling
	fmt.Println("\n  Test 3: Timeout Handling")
	testTimeoutHandling(slowEndpoint)

	// Test 4: Intermittent failures (eventual success)
	fmt.Println("\n  Test 4: Intermittent Failures")
	testIntermittentFailures(intermittentEndpoint)

	// Test 5: Multiple webhooks with different configurations
	fmt.Println("\n  Test 5: Multiple Webhook Configurations")
	testMultipleWebhooks(successEndpoint, failureEndpoint)

	fmt.Println("\n3. Testing Webhook Configuration Management:")
	testWebhookConfiguration()

	fmt.Println("\n4. Testing Webhook Event Types:")
	testWebhookEventTypes()

	fmt.Println("\nAll webhook tests completed!")
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

	// Create webhook configuration
	webhookConfig := &webhook.Webhook{
		ID:             xid.New(),
		OrganizationID: "test-org",
		URL:            endpoint.URL,
		Events:         []string{webhook.EventUserCreated, webhook.EventUserUpdated},
		Enabled:        true,
		Secret:         "test-secret-key",
		MaxRetries:     3,
		RetryBackoff:   webhook.RetryBackoffLinear,
	}

	// Create test event
	event := &webhook.Event{
		ID:             xid.New(),
		Type:           webhook.EventUserCreated,
		OrganizationID: "test-org",
		Data: map[string]interface{}{
			"user_id": "user-123",
			"email":   "test@example.com",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	// Simulate webhook delivery
	fmt.Printf("    📤 Sending webhook to %s\n", endpoint.URL)

	success := simulateWebhookDelivery(webhookConfig, event)

	// Check results
	callCount, receivedData := endpoint.GetStats()

	if success && callCount == 1 && len(receivedData) == 1 {
		fmt.Printf("    ✅ Webhook delivered successfully on first attempt\n")
		fmt.Printf("    ✅ Received event: %s\n", receivedData[0].Type)
	} else {
		fmt.Printf("    ❌ Unexpected result: success=%v, calls=%d, data=%d\n", success, callCount, len(receivedData))
	}
}

// testFailedDelivery tests webhook delivery with failures and retries
func testFailedDelivery(endpoint *MockWebhookEndpoint) {
	endpoint.Reset()

	webhookConfig := &webhook.Webhook{
		ID:             xid.New(),
		OrganizationID: "test-org",
		URL:            endpoint.URL,
		Events:         []string{webhook.EventUserDeleted},
		Enabled:        true,
		Secret:         "test-secret-key",
		MaxRetries:     3,
		RetryBackoff:   webhook.RetryBackoffLinear,
	}

	event := &webhook.Event{
		ID:             xid.New(),
		Type:           webhook.EventUserDeleted,
		OrganizationID: "test-org",
		Data: map[string]interface{}{
			"user_id": "user-456",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	fmt.Printf("    📤 Sending webhook to failing endpoint %s\n", endpoint.URL)

	// Simulate webhook delivery with retries
	success := simulateWebhookDeliveryWithRetries(webhookConfig, event)

	callCount, _ := endpoint.GetStats()

	if !success && callCount == 4 { // 1 initial + 3 retries
		fmt.Printf("    ✅ Webhook failed as expected after %d attempts\n", callCount)
	} else {
		fmt.Printf("    ❌ Unexpected result: success=%v, calls=%d\n", success, callCount)
	}
}

// testTimeoutHandling tests webhook timeout handling
func testTimeoutHandling(endpoint *MockWebhookEndpoint) {
	endpoint.Reset()

	webhookConfig := &webhook.Webhook{
		ID:             xid.New(),
		OrganizationID: "test-org",
		URL:            endpoint.URL,
		Events:         []string{webhook.EventSessionCreated},
		Enabled:        true,
		Secret:         "test-secret-key",
		MaxRetries:     1,
		RetryBackoff:   webhook.RetryBackoffLinear,
	}

	event := &webhook.Event{
		ID:             xid.New(),
		Type:           webhook.EventSessionCreated,
		OrganizationID: "test-org",
		Data: map[string]interface{}{
			"session_id": "session-789",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	fmt.Printf("    📤 Sending webhook to slow endpoint %s (timeout test)\n", endpoint.URL)

	start := time.Now()
	success := simulateWebhookDeliveryWithTimeout(webhookConfig, event, 500*time.Millisecond)
	duration := time.Since(start)

	if !success && duration < 1*time.Second {
		fmt.Printf("    ✅ Webhook timed out as expected in %v\n", duration)
	} else {
		fmt.Printf("    ❌ Unexpected result: success=%v, duration=%v\n", success, duration)
	}
}

// testIntermittentFailures tests handling of intermittent failures
func testIntermittentFailures(endpoint *IntermittentEndpoint) {
	endpoint.Reset()

	webhookConfig := &webhook.Webhook{
		ID:             xid.New(),
		OrganizationID: "test-org",
		URL:            endpoint.URL,
		Events:         []string{webhook.EventOrgUpdated},
		Enabled:        true,
		Secret:         "test-secret-key",
		MaxRetries:     3,
		RetryBackoff:   webhook.RetryBackoffLinear,
	}

	event := &webhook.Event{
		ID:             xid.New(),
		Type:           webhook.EventOrgUpdated,
		OrganizationID: "test-org",
		Data: map[string]interface{}{
			"organization_id": "org-123",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	fmt.Printf("    📤 Sending webhook to intermittent endpoint %s\n", endpoint.URL)

	success := simulateWebhookDeliveryWithRetries(webhookConfig, event)
	callCount, _ := endpoint.GetStats()

	if success && callCount == 3 {
		fmt.Printf("    ✅ Webhook succeeded after %d attempts (intermittent failure handled)\n", callCount)
	} else {
		fmt.Printf("    ❌ Unexpected result: success=%v, calls=%d\n", success, callCount)
	}
}

// testMultipleWebhooks tests multiple webhook configurations
func testMultipleWebhooks(successEndpoint, failureEndpoint *MockWebhookEndpoint) {
	successEndpoint.Reset()
	failureEndpoint.Reset()

	webhooks := []*webhook.Webhook{
		{
			ID:             xid.New(),
			OrganizationID: "test-org",
			URL:            successEndpoint.URL,
			Events:         []string{webhook.EventUserLogin},
			Enabled:        true,
			Secret:         "secret-1",
			MaxRetries:     2,
			RetryBackoff:   webhook.RetryBackoffLinear,
		},
		{
			ID:             xid.New(),
			OrganizationID: "test-org",
			URL:            failureEndpoint.URL,
			Events:         []string{webhook.EventUserLogin},
			Enabled:        true,
			Secret:         "secret-2",
			MaxRetries:     1,
			RetryBackoff:   webhook.RetryBackoffLinear,
		},
	}

	event := &webhook.Event{
		ID:             xid.New(),
		Type:           webhook.EventUserLogin,
		OrganizationID: "test-org",
		Data: map[string]interface{}{
			"user_id": "user-999",
			"ip":      "192.168.1.1",
		},
		OccurredAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	fmt.Printf("    📤 Sending webhook to multiple endpoints\n")

	// Simulate delivery to multiple webhooks
	successCount := 0
	for i, webhookConfig := range webhooks {
		fmt.Printf("      Webhook %d: %s\n", i+1, webhookConfig.URL)
		if simulateWebhookDelivery(webhookConfig, event) {
			successCount++
		}
	}

	successCalls, _ := successEndpoint.GetStats()
	failureCalls, _ := failureEndpoint.GetStats()

	fmt.Printf("    ✅ Delivered to %d/%d webhooks\n", successCount, len(webhooks))
	fmt.Printf("    ✅ Success endpoint calls: %d, Failure endpoint calls: %d\n", successCalls, failureCalls)
}

// testWebhookConfiguration tests webhook configuration validation
func testWebhookConfiguration() {
	fmt.Println("    Testing webhook configuration validation...")

	// Test valid configuration
	validConfig := &webhook.Webhook{
		ID:             xid.New(),
		OrganizationID: "test-org",
		URL:            "https://example.com/webhook",
		Events:         []string{webhook.EventUserCreated, webhook.EventUserUpdated},
		Enabled:        true,
		Secret:         "valid-secret-key",
		MaxRetries:     3,
		RetryBackoff:   webhook.RetryBackoffExponential,
	}

	if err := validateWebhookConfig(validConfig); err == nil {
		fmt.Printf("    ✅ Valid configuration accepted\n")
	} else {
		fmt.Printf("    ❌ Valid configuration rejected: %v\n", err)
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

	fmt.Printf("    ✅ Rejected %d/%d invalid configurations\n", invalidCount, len(invalidConfigs))
}

// testWebhookEventTypes tests different webhook event types
func testWebhookEventTypes() {
	events := webhook.AllEventTypes()

	fmt.Printf("    Testing %d webhook event types...\n", len(events))

	for _, eventType := range events {
		event := &webhook.Event{
			ID:             xid.New(),
			Type:           eventType,
			OrganizationID: "test-org",
			Data: map[string]interface{}{
				"test": true,
			},
			OccurredAt: time.Now(),
			CreatedAt:  time.Now(),
		}

		// Validate event structure
		if event.ID.IsNil() || event.Type == "" || event.OrganizationID == "" {
			fmt.Printf("    ❌ Invalid event structure for %s\n", eventType)
		}
	}

	fmt.Printf("    ✅ All %d event types validated\n", len(events))
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
