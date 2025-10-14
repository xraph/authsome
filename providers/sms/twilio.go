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
func (p *TwilioProvider) Send(ctx context.Context, req *notification.SendRequest) error {
	// Validate phone number format
	if !p.isValidPhoneNumber(req.Recipient) {
		return fmt.Errorf("invalid phone number format: %s", req.Recipient)
	}

	// Prepare request data
	data := url.Values{}
	data.Set("From", p.config.FromNumber)
	data.Set("To", req.Recipient)
	data.Set("Body", req.Body)

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

	// Check response status
	if resp.StatusCode >= 400 {
		var errorResp TwilioErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("SMS failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("SMS failed: %s (code: %d)", errorResp.Message, errorResp.Code)
	}

	return nil
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
func (p *MockSMSProvider) Send(ctx context.Context, req *notification.SendRequest) error {
	p.SentMessages = append(p.SentMessages, MockSMSMessage{
		Recipient: req.Recipient,
		Subject:   req.Subject,
		Body:      req.Body,
	})
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