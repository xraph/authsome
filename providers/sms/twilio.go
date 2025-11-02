package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/xraph/authsome/core/notification"
)

// TwilioConfig holds Twilio configuration
type TwilioConfig struct {
	AccountSID string `json:"account_sid"`
	AuthToken  string `json:"auth_token"`
	FromNumber string `json:"from_number"`
	BaseURL    string `json:"base_url"`
}

// TwilioProvider implements notification.Provider for Twilio SMS
type TwilioProvider struct {
	config     TwilioConfig
	httpClient *http.Client
}

// NewTwilioProvider creates a new Twilio SMS provider
func NewTwilioProvider(config TwilioConfig) *TwilioProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.twilio.com"
	}
	
	return &TwilioProvider{
		config:     config,
		httpClient: &http.Client{},
	}
}

// ID returns the provider ID
func (p *TwilioProvider) ID() string {
	return "twilio"
}

// Type returns the notification type this provider handles
func (p *TwilioProvider) Type() notification.NotificationType {
	return notification.NotificationTypeSMS
}

// Send sends an SMS notification
func (p *TwilioProvider) Send(ctx context.Context, notif *notification.Notification) error {
	// Validate phone number format
	if !p.isValidPhoneNumber(notif.Recipient) {
		return fmt.Errorf("invalid phone number format: %s", notif.Recipient)
	}

	// Prepare request data
	data := url.Values{}
	data.Set("From", p.config.FromNumber)
	data.Set("To", notif.Recipient)
	data.Set("Body", notif.Body)

	// Create HTTP request
	apiURL := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Messages.json", 
		p.config.BaseURL, p.config.AccountSID)
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, 
		strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.SetBasicAuth(p.config.AccountSID, p.config.AuthToken)

	// Send request
	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}
	defer resp.Body.Close()

	// Parse response to get message SID
	var twilioResp TwilioMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&twilioResp); err != nil {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("SMS failed with status %d", resp.StatusCode)
		}
		// Success but couldn't parse response
		return nil
	}

	// Check response status
	if resp.StatusCode >= 400 {
		return fmt.Errorf("SMS failed: %s (code: %d)", twilioResp.ErrorMessage, twilioResp.ErrorCode)
	}

	// Store the Twilio message SID in notification metadata
	if notif.Metadata == nil {
		notif.Metadata = make(map[string]interface{})
	}
	notif.Metadata["twilio_sid"] = twilioResp.Sid
	notif.ProviderID = twilioResp.Sid

	return nil
}

// GetStatus gets the delivery status of a notification from Twilio
func (p *TwilioProvider) GetStatus(ctx context.Context, providerID string) (notification.NotificationStatus, error) {
	// Query Twilio API for message status
	apiURL := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Messages/%s.json",
		p.config.BaseURL, p.config.AccountSID, providerID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.SetBasicAuth(p.config.AccountSID, p.config.AuthToken)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to query status: %w", err)
	}
	defer resp.Body.Close()

	var twilioResp TwilioMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&twilioResp); err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to parse response: %w", err)
	}

	// Map Twilio status to notification status
	switch twilioResp.Status {
	case "queued", "sending":
		return notification.NotificationStatusSent, nil
	case "sent", "delivered":
		return notification.NotificationStatusDelivered, nil
	case "failed", "undelivered":
		return notification.NotificationStatusFailed, nil
	default:
		return notification.NotificationStatusSent, nil
	}
}

// ValidateConfig validates the provider configuration
func (p *TwilioProvider) ValidateConfig() error {
	return p.Validate()
}

// TwilioMessageResponse represents a Twilio message response
type TwilioMessageResponse struct {
	Sid          string `json:"sid"`
	Status       string `json:"status"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// TwilioErrorResponse represents a Twilio error response
type TwilioErrorResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message"`
	MoreInfo string `json:"more_info"`
	Status   int    `json:"status"`
}

// isValidPhoneNumber performs basic phone number validation
func (p *TwilioProvider) isValidPhoneNumber(phone string) bool {
	// Remove common formatting characters
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")

	// Must start with + and contain only digits after that
	if !strings.HasPrefix(cleaned, "+") {
		return false
	}

	digits := cleaned[1:]
	if len(digits) < 10 || len(digits) > 15 {
		return false
	}

	for _, char := range digits {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// Validate validates the provider configuration
func (p *TwilioProvider) Validate() error {
	if p.config.AccountSID == "" {
		return fmt.Errorf("Twilio Account SID is required")
	}
	if p.config.AuthToken == "" {
		return fmt.Errorf("Twilio Auth Token is required")
	}
	if p.config.FromNumber == "" {
		return fmt.Errorf("Twilio from number is required")
	}
	if !p.isValidPhoneNumber(p.config.FromNumber) {
		return fmt.Errorf("invalid Twilio from number format")
	}
	return nil
}

// MockSMSProvider is a mock SMS provider for testing
type MockSMSProvider struct {
	SentMessages []MockSMSMessage
}

// MockSMSMessage represents a sent SMS message for testing
type MockSMSMessage struct {
	Recipient string
	Subject   string
	Body      string
}

// NewMockSMSProvider creates a new mock SMS provider
func NewMockSMSProvider() *MockSMSProvider {
	return &MockSMSProvider{
		SentMessages: make([]MockSMSMessage, 0),
	}
}

// ID returns the provider ID
func (p *MockSMSProvider) ID() string {
	return "mock-sms"
}

// Type returns the notification type this provider handles
func (p *MockSMSProvider) Type() notification.NotificationType {
	return notification.NotificationTypeSMS
}

// Send sends a mock SMS notification
func (p *MockSMSProvider) Send(ctx context.Context, notif *notification.Notification) error {
	p.SentMessages = append(p.SentMessages, MockSMSMessage{
		Recipient: notif.Recipient,
		Subject:   notif.Subject,
		Body:      notif.Body,
	})
	return nil
}

// GetStatus returns the status (always delivered for mock)
func (p *MockSMSProvider) GetStatus(ctx context.Context, providerID string) (notification.NotificationStatus, error) {
	return notification.NotificationStatusDelivered, nil
}

// ValidateConfig validates the mock provider (always valid)
func (p *MockSMSProvider) ValidateConfig() error {
	return nil
}

// Validate validates the mock provider (always valid)
func (p *MockSMSProvider) Validate() error {
	return nil
}

// GetSentMessages returns all sent messages
func (p *MockSMSProvider) GetSentMessages() []MockSMSMessage {
	return p.SentMessages
}

// ClearSentMessages clears all sent messages
func (p *MockSMSProvider) ClearSentMessages() {
	p.SentMessages = make([]MockSMSMessage, 0)
}