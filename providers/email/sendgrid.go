package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/xraph/authsome/core/notification"
)

// SendGridConfig holds SendGrid configuration
type SendGridConfig struct {
	APIKey   string `json:"api_key"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	BaseURL  string `json:"base_url"`
}

// SendGridProvider implements notification.Provider for SendGrid
type SendGridProvider struct {
	config     SendGridConfig
	httpClient *http.Client
}

// NewSendGridProvider creates a new SendGrid email provider
func NewSendGridProvider(config SendGridConfig) *SendGridProvider {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.sendgrid.com"
	}

	return &SendGridProvider{
		config:     config,
		httpClient: &http.Client{},
	}
}

// ID returns the provider ID
func (p *SendGridProvider) ID() string {
	return "sendgrid"
}

// Type returns the notification type this provider handles
func (p *SendGridProvider) Type() notification.NotificationType {
	return notification.NotificationTypeEmail
}

// Send sends an email notification via SendGrid
func (p *SendGridProvider) Send(ctx context.Context, notif *notification.Notification) error {
	// Build SendGrid API request
	payload := SendGridRequest{
		Personalizations: []SendGridPersonalization{
			{
				To: []SendGridEmail{
					{Email: notif.Recipient},
				},
				Subject: notif.Subject,
			},
		},
		From: SendGridEmail{
			Email: p.config.From,
			Name:  p.config.FromName,
		},
		Content: []SendGridContent{
			{
				Type:  "text/html",
				Value: notif.Body,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	apiURL := fmt.Sprintf("%s/v3/mail/send", p.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode >= 400 {
		var errorResp SendGridErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("email failed with status %d", resp.StatusCode)
		}
		return fmt.Errorf("email failed: %v", errorResp.Errors)
	}

	// Extract message ID from response headers
	if messageID := resp.Header.Get("X-Message-Id"); messageID != "" {
		if notif.Metadata == nil {
			notif.Metadata = make(map[string]interface{})
		}
		notif.Metadata["sendgrid_message_id"] = messageID
		notif.ProviderID = messageID
	}

	return nil
}

// GetStatus gets the delivery status from SendGrid
func (p *SendGridProvider) GetStatus(ctx context.Context, providerID string) (notification.NotificationStatus, error) {
	// SendGrid Activity Feed API
	apiURL := fmt.Sprintf("%s/v3/messages/%s", p.config.BaseURL, providerID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to query status: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var activity SendGridActivity
	if err := json.NewDecoder(resp.Body).Decode(&activity); err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to parse response: %w", err)
	}

	// Map SendGrid status to notification status
	switch activity.Status {
	case "processed", "delivered":
		return notification.NotificationStatusDelivered, nil
	case "dropped", "bounce", "blocked":
		return notification.NotificationStatusFailed, nil
	case "deferred":
		return notification.NotificationStatusSent, nil
	default:
		return notification.NotificationStatusSent, nil
	}
}

// ValidateConfig validates the provider configuration
func (p *SendGridProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("SendGrid API key is required")
	}
	if p.config.From == "" {
		return fmt.Errorf("from email address is required")
	}
	return nil
}

// SendGrid API types

type SendGridRequest struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmail             `json:"from"`
	Content          []SendGridContent         `json:"content"`
}

type SendGridPersonalization struct {
	To      []SendGridEmail `json:"to"`
	Subject string          `json:"subject"`
}

type SendGridEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type SendGridErrorResponse struct {
	Errors []struct {
		Message string `json:"message"`
		Field   string `json:"field"`
	} `json:"errors"`
}

type SendGridActivity struct {
	Status string `json:"status"`
	Events []struct {
		EventType string `json:"event"`
		Timestamp int64  `json:"timestamp"`
	} `json:"events"`
}
