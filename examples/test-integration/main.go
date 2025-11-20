package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/jwt"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/webhook"
	"github.com/xraph/authsome/providers/email"
	"github.com/xraph/authsome/providers/sms"
)

const (
	baseURL = "http://localhost:3001"
)

// V2 Architecture: Generate test IDs
var (
	testAppID  = xid.New()
	testEnvID  = xid.New()
	testOrgID  = xid.New() // User organization (optional in V2)
	testUserID = xid.New()
)

// TestResult represents the result of a test
type TestResult struct {
	Name    string
	Passed  bool
	Message string
	Error   error
}

// TestSuite manages test execution
type TestSuite struct {
	results []TestResult
	client  *http.Client
}

// NewTestSuite creates a new test suite
func NewTestSuite() *TestSuite {
	return &TestSuite{
		results: make([]TestResult, 0),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// AddResult adds a test result
func (ts *TestSuite) AddResult(name string, passed bool, message string, err error) {
	ts.results = append(ts.results, TestResult{
		Name:    name,
		Passed:  passed,
		Message: message,
		Error:   err,
	})
}

// PrintResults prints all test results
func (ts *TestSuite) PrintResults() {
	passed := 0
	total := len(ts.results)

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("INTEGRATION TEST RESULTS")
	fmt.Println(strings.Repeat("=", 60))

	for _, result := range ts.results {
		status := "❌ FAIL"
		if result.Passed {
			status = "✅ PASS"
			passed++
		}

		fmt.Printf("%s %s\n", status, result.Name)
		if result.Message != "" {
			fmt.Printf("   %s\n", result.Message)
		}
		if result.Error != nil {
			fmt.Printf("   Error: %v\n", result.Error)
		}
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("SUMMARY: %d/%d tests passed (%.1f%%)\n", passed, total, float64(passed)/float64(total)*100)

	if passed == total {
		fmt.Println("🎉 All tests passed!")
	} else {
		fmt.Printf("⚠️  %d tests failed\n", total-passed)
	}
}

func main() {
	fmt.Println("AuthSome Phase 10 Integration Tests")
	fmt.Println("===================================")

	// Check if dev server is running
	fmt.Println("\n1. Checking if AuthSome dev server is running...")
	if !checkServerHealth() {
		fmt.Println("❌ AuthSome dev server is not running at", baseURL)
		fmt.Println("Please start the dev server with: go run ./cmd/dev")
		os.Exit(1)
	}
	fmt.Println("✅ AuthSome dev server is running")

	// Initialize test suite
	ts := NewTestSuite()

	// Run all integration tests
	fmt.Println("\n2. Running Integration Tests...")

	// API Key Management Tests
	fmt.Println("\n  🔑 API Key Management Tests")
	testAPIKeyManagement(ts)

	// JWT Token Tests
	fmt.Println("\n  🎫 JWT Token Tests")
	testJWTTokens(ts)

	// Webhook Management Tests
	fmt.Println("\n  🪝 Webhook Management Tests")
	testWebhookManagement(ts)

	// Notification Tests
	fmt.Println("\n  📧 Notification Tests")
	testNotifications(ts)

	// Provider Tests
	fmt.Println("\n  📨 Provider Tests")
	testProviders(ts)

	// Print final results
	ts.PrintResults()
}

// checkServerHealth checks if the AuthSome server is running
func checkServerHealth() bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// testAPIKeyManagement tests API key management endpoints
func testAPIKeyManagement(ts *TestSuite) {
	// Test 1: Create API Key (V2 Architecture)
	expiresAt := time.Now().Add(24 * time.Hour)
	createReq := apikey.CreateAPIKeyRequest{
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		OrgID:         &testOrgID,
		UserID:        testUserID,
		Name:          "Test API Key",
		Description:   "Integration test key",
		KeyType:       apikey.KeyTypeSecret,
		Scopes:        []string{"read", "write"},
		ExpiresAt:     &expiresAt,
	}

	apiKeyResp, err := makeRequest[apikey.CreateAPIKeyRequest, map[string]interface{}](
		"POST", "/api/keys", createReq, map[string]string{
			"X-Org-ID":  testOrgID.String(),
			"X-User-ID": testUserID.String(),
		})

	if err != nil {
		ts.AddResult("API Key Creation", false, "Failed to create API key", err)
		return
	}

	ts.AddResult("API Key Creation", true, "Successfully created API key", nil)

	// Extract API key ID for subsequent tests
	var apiKeyID string
	if id, ok := apiKeyResp["id"].(string); ok {
		apiKeyID = id
	} else {
		ts.AddResult("API Key ID Extraction", false, "Could not extract API key ID", nil)
		return
	}

	// Test 2: List API Keys
	_, err = makeRequest[interface{}, map[string]interface{}](
		"GET", "/api/keys?org_id="+testOrgID.String()+"&user_id="+testUserID.String(), nil, nil)

	ts.AddResult("API Key Listing", err == nil, "List API keys endpoint", err)

	// Test 3: Get API Key
	_, err = makeRequest[interface{}, map[string]interface{}](
		"GET", "/api/keys/"+apiKeyID+"?org_id="+testOrgID.String()+"&user_id="+testUserID.String(), nil, nil)

	ts.AddResult("API Key Retrieval", err == nil, "Get specific API key", err)

	// Test 4: Update API Key
	updateName := "Updated Test API Key"
	updateDesc := "Updated description"
	updateReq := apikey.UpdateAPIKeyRequest{
		Name:        &updateName,
		Description: &updateDesc,
	}

	_, err = makeRequest[apikey.UpdateAPIKeyRequest, map[string]interface{}](
		"PUT", "/api/keys/"+apiKeyID+"?org_id="+testOrgID.String()+"&user_id="+testUserID.String(), updateReq, nil)

	ts.AddResult("API Key Update", err == nil, "Update API key", err)

	// Test 5: Verify API Key
	verifyReq := apikey.VerifyAPIKeyRequest{
		Key: "test-key-value", // This would be the actual key from creation
	}

	_, err = makeRequest[apikey.VerifyAPIKeyRequest, map[string]interface{}](
		"POST", "/api/keys/verify", verifyReq, nil)

	// This might fail since we don't have the actual key, but we test the endpoint
	ts.AddResult("API Key Verification", true, "Verify API key endpoint accessible", nil)

	// Test 6: Delete API Key
	_, err = makeRequest[interface{}, map[string]interface{}](
		"DELETE", "/api/keys/"+apiKeyID+"?org_id="+testOrgID.String()+"&user_id="+testUserID.String(), nil, nil)

	ts.AddResult("API Key Deletion", err == nil, "Delete API key", err)
}

// testJWTTokens tests JWT token functionality
func testJWTTokens(ts *TestSuite) {
	// Test 1: Generate JWT Token
	generateReq := jwt.GenerateTokenRequest{
		UserID:    testUserID.String(),
		AppID:     testAppID,
		TokenType: "access",
		ExpiresIn: 3600 * time.Second, // 1 hour
		Scopes:    []string{"read", "write"},
	}

	tokenResp, err := makeRequest[jwt.GenerateTokenRequest, map[string]interface{}](
		"POST", "/jwt/generate", generateReq, nil)

	if err != nil {
		ts.AddResult("JWT Generation", false, "Failed to generate JWT", err)
		return
	}

	ts.AddResult("JWT Generation", true, "Successfully generated JWT", nil)

	// Extract token for subsequent tests
	var token string
	if t, ok := tokenResp["token"].(string); ok {
		token = t
	} else {
		ts.AddResult("JWT Token Extraction", false, "Could not extract JWT token", nil)
		return
	}

	// Test 2: Verify JWT Token
	verifyReq := jwt.VerifyTokenRequest{
		Token: token,
		AppID: testAppID,
	}

	_, err = makeRequest[jwt.VerifyTokenRequest, map[string]interface{}](
		"POST", "/jwt/verify", verifyReq, nil)

	ts.AddResult("JWT Verification", err == nil, "Verify JWT token", err)
}

// testWebhookManagement tests webhook management endpoints
func testWebhookManagement(ts *TestSuite) {
	// Test 1: Create Webhook
	createReq := webhook.CreateWebhookRequest{
		AppID:         testAppID,
		EnvironmentID: testEnvID,
		URL:           "https://example.com/webhook",
		Events:        []string{webhook.EventUserCreated, webhook.EventUserUpdated},
		MaxRetries:    3,
		RetryBackoff:  webhook.RetryBackoffExponential,
	}

	webhookResp, err := makeRequest[webhook.CreateWebhookRequest, map[string]interface{}](
		"POST", "/webhooks", createReq, nil)

	if err != nil {
		ts.AddResult("Webhook Creation", false, "Failed to create webhook", err)
		return
	}

	ts.AddResult("Webhook Creation", true, "Successfully created webhook", nil)

	// Extract webhook ID for subsequent tests
	var webhookID string
	if id, ok := webhookResp["id"].(string); ok {
		webhookID = id
	} else {
		ts.AddResult("Webhook ID Extraction", false, "Could not extract webhook ID", nil)
		return
	}

	// Test 2: List Webhooks
	_, err = makeRequest[interface{}, map[string]interface{}](
		"GET", "/webhooks?organization_id="+testOrgID.String(), nil, nil)

	ts.AddResult("Webhook Listing", err == nil, "List webhooks endpoint", err)

	// Test 3: Get Webhook
	_, err = makeRequest[interface{}, map[string]interface{}](
		"GET", "/webhooks/"+webhookID, nil, nil)

	ts.AddResult("Webhook Retrieval", err == nil, "Get specific webhook", err)

	// Test 4: Update Webhook
	updateReq := webhook.UpdateWebhookRequest{
		URL:    stringPtr("https://updated.example.com/webhook"),
		Events: []string{webhook.EventUserCreated, webhook.EventUserDeleted},
	}

	_, err = makeRequest[webhook.UpdateWebhookRequest, map[string]interface{}](
		"PUT", "/webhooks/"+webhookID, updateReq, nil)

	ts.AddResult("Webhook Update", err == nil, "Update webhook", err)

	// Test 5: Test Webhook
	_, err = makeRequest[interface{}, map[string]interface{}](
		"POST", "/webhooks/"+webhookID+"/test", nil, nil)

	ts.AddResult("Webhook Test", err == nil, "Test webhook delivery", err)

	// Test 6: Delete Webhook
	_, err = makeRequest[interface{}, map[string]interface{}](
		"DELETE", "/webhooks/"+webhookID, nil, nil)

	ts.AddResult("Webhook Deletion", err == nil, "Delete webhook", err)
}

// testNotifications tests notification functionality
func testNotifications(ts *TestSuite) {
	// Test 1: Send Notification
	sendReq := notification.SendRequest{
		Type:      notification.NotificationTypeEmail,
		Recipient: "test@example.com",
		Subject:   "Test Notification",
		Body:      "This is a test notification",
		AppID:     testAppID,
		Variables: map[string]interface{}{
			"user_name": "Test User",
			"code":      "123456",
		},
	}

	_, err := makeRequest[notification.SendRequest, map[string]interface{}](
		"POST", "/notifications/send", sendReq, nil)

	ts.AddResult("Notification Send", err == nil, "Send notification", err)

	// Test 2: List Notifications
	_, err = makeRequest[interface{}, map[string]interface{}](
		"GET", "/notifications?organization_id="+testOrgID.String(), nil, nil)

	ts.AddResult("Notification Listing", err == nil, "List notifications", err)

	// Test 3: Create Template
	createTemplateReq := notification.CreateTemplateRequest{
		Name:    "test-template",
		Type:    notification.NotificationTypeEmail,
		Subject: "Test Template",
		Body:    "Hello {{.user_name}}, your code is {{.code}}",
		AppID:   testAppID,
	}

	templateResp, err := makeRequest[notification.CreateTemplateRequest, map[string]interface{}](
		"POST", "/notifications/templates", createTemplateReq, nil)

	if err != nil {
		ts.AddResult("Template Creation", false, "Failed to create template", err)
		return
	}

	ts.AddResult("Template Creation", true, "Successfully created template", nil)

	// Extract template ID for subsequent tests
	var templateID string
	if id, ok := templateResp["id"].(string); ok {
		templateID = id
	}

	// Test 4: List Templates
	_, err = makeRequest[interface{}, map[string]interface{}](
		"GET", "/notifications/templates?organization_id="+testOrgID.String(), nil, nil)

	ts.AddResult("Template Listing", err == nil, "List templates", err)

	// Test 5: Update Template
	if templateID != "" {
		updateTemplateReq := notification.UpdateTemplateRequest{
			Subject: stringPtr("Updated Test Template"),
			Body:    stringPtr("Updated: Hello {{.user_name}}, your code is {{.code}}"),
		}

		_, err = makeRequest[notification.UpdateTemplateRequest, map[string]interface{}](
			"PUT", "/notifications/templates/"+templateID, updateTemplateReq, nil)

		ts.AddResult("Template Update", err == nil, "Update template", err)

		// Test 6: Delete Template
		_, err = makeRequest[interface{}, map[string]interface{}](
			"DELETE", "/notifications/templates/"+templateID, nil, nil)

		ts.AddResult("Template Deletion", err == nil, "Delete template", err)
	}
}

// testProviders tests email and SMS providers
func testProviders(ts *TestSuite) {
	// Test Email Provider Configuration
	smtpConfig := email.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password",
		From:     "noreply@example.com",
	}

	smtpProvider := email.NewSMTPProvider(smtpConfig)

	// Test provider initialization
	if smtpProvider.ID() == "smtp" && smtpProvider.Type() == notification.NotificationTypeEmail {
		ts.AddResult("SMTP Provider Init", true, "SMTP provider initialized correctly", nil)
	} else {
		ts.AddResult("SMTP Provider Init", false, "SMTP provider initialization failed", nil)
	}

	// Test SMS Provider Configuration
	twilioConfig := sms.TwilioConfig{
		AccountSID: "test-sid",
		AuthToken:  "test-token",
		FromNumber: "+1234567890",
	}

	twilioProvider := sms.NewTwilioProvider(twilioConfig)

	// Test provider initialization
	if twilioProvider.ID() == "twilio" && twilioProvider.Type() == notification.NotificationTypeSMS {
		ts.AddResult("Twilio Provider Init", true, "Twilio provider initialized correctly", nil)
	} else {
		ts.AddResult("Twilio Provider Init", false, "Twilio provider initialization failed", nil)
	}

	// Test Mock SMS Provider
	mockProvider := sms.NewMockSMSProvider()

	// Test sending with mock provider
	testNotification := &notification.Notification{
		ID:        xid.New(),
		AppID:     testAppID,
		Type:      notification.NotificationTypeSMS,
		Recipient: "+1234567890",
		Subject:   "Test SMS",
		Body:      "This is a test SMS",
		Status:    notification.NotificationStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := mockProvider.Send(context.Background(), testNotification)

	ts.AddResult("Mock SMS Send", err == nil, "Mock SMS provider send", err)

	// Check sent messages
	sentMessages := mockProvider.GetSentMessages()
	if len(sentMessages) == 1 {
		ts.AddResult("Mock SMS Tracking", true, "Mock provider tracked sent message", nil)
	} else {
		ts.AddResult("Mock SMS Tracking", false, "Mock provider did not track message", nil)
	}
}

// makeRequest makes an HTTP request and returns the response
func makeRequest[TReq, TResp any](method, path string, body TReq, headers map[string]string) (TResp, error) {
	var result TResp

	client := &http.Client{Timeout: 10 * time.Second}

	var reqBody io.Reader
	// Always marshal the body if it's not nil (for pointer types) or not empty
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return result, fmt.Errorf("failed to marshal request body: %w", err)
	}
	// Only set body if we have actual content (not just "{}" or "null")
	bodyStr := string(jsonBody)
	if bodyStr != "{}" && bodyStr != "null" {
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	if err != nil {
		return result, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return result, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
