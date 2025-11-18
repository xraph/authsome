package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/xraph/authsome/core/notification"
)

// MailerSendConfig holds MailerSend API configuration
type MailerSendConfig struct {
	APIKey   string `json:"api_key"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	ReplyTo  string `json:"reply_to,omitempty"`
}

// MailerSendProvider implements notification.Provider for MailerSend email service
type MailerSendProvider struct {
	config     MailerSendConfig
	httpClient *http.Client
}

// NewMailerSendProvider creates a new MailerSend email provider
func NewMailerSendProvider(config MailerSendConfig) *MailerSendProvider {
	return &MailerSendProvider{
		config:     config,
		httpClient: &http.Client{},
	}
}

// ID returns the provider ID
func (p *MailerSendProvider) ID() string {
	return "mailersend"
}

// Type returns the notification type this provider handles
func (p *MailerSendProvider) Type() notification.NotificationType {
	return notification.NotificationTypeEmail
}

// Send sends an email notification via MailerSend API
func (p *MailerSendProvider) Send(ctx context.Context, notif *notification.Notification) error {
	// Build request payload according to MailerSend API v1
	from := map[string]string{
		"email": p.config.From,
	}
	if p.config.FromName != "" {
		from["name"] = p.config.FromName
	}

	to := []map[string]string{
		{
			"email": notif.Recipient,
		},
	}

	payload := map[string]interface{}{
		"from":    from,
		"to":      to,
		"subject": notif.Subject,
		"html":    notif.Body,
	}

	// Add reply_to if configured
	if p.config.ReplyTo != "" {
		payload["reply_to"] = map[string]string{
			"email": p.config.ReplyTo,
		}
	}

	// Add tags from metadata if available
	if notif.Metadata != nil {
		if tags, ok := notif.Metadata["tags"].([]string); ok && len(tags) > 0 {
			payload["tags"] = tags
		}
		
		// Add template variables if available
		if variables, ok := notif.Metadata["variables"].([]map[string]interface{}); ok && len(variables) > 0 {
			payload["variables"] = variables
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mailersend.com/v1/email", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("mailersend API error (status %d): %s", resp.StatusCode, string(body))
	}

	// MailerSend returns 202 Accepted with X-Message-Id header
	if messageID := resp.Header.Get("X-Message-Id"); messageID != "" && notif.Metadata != nil {
		notif.Metadata["mailersend_message_id"] = messageID
	}

	return nil
}

// GetStatus gets the delivery status of a notification
func (p *MailerSendProvider) GetStatus(ctx context.Context, providerID string) (notification.NotificationStatus, error) {
	// MailerSend doesn't provide a direct status endpoint
	// Status updates typically come via webhooks
	// For now, return pending status
	return notification.NotificationStatusPending, nil
}

// ValidateConfig validates the provider configuration
func (p *MailerSendProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return fmt.Errorf("mailersend API key is required")
	}
	if p.config.From == "" {
		return fmt.Errorf("from email address is required")
	}
	return nil
}

// MailerSendWebhookEvent represents a MailerSend webhook event
type MailerSendWebhookEvent struct {
	Type      string                 `json:"type"`
	Email     string                 `json:"email"`
	MessageID string                 `json:"message_id"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// ParseWebhookEvent parses a MailerSend webhook event
func (p *MailerSendProvider) ParseWebhookEvent(body []byte) (*MailerSendWebhookEvent, error) {
	var event MailerSendWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event: %w", err)
	}
	return &event, nil
}

// MapWebhookEventToStatus maps a MailerSend webhook event type to notification status
func (p *MailerSendProvider) MapWebhookEventToStatus(eventType string) notification.NotificationStatus {
	switch eventType {
	case "activity.sent":
		return notification.NotificationStatusSent
	case "activity.delivered":
		return notification.NotificationStatusDelivered
	case "activity.soft_bounced", "activity.hard_bounced":
		return notification.NotificationStatusBounced
	case "activity.opened":
		return notification.NotificationStatusDelivered // Email was opened, so it was delivered
	case "activity.clicked":
		return notification.NotificationStatusDelivered
	case "activity.unsubscribed", "activity.spam_complaint":
		return notification.NotificationStatusFailed
	default:
		return notification.NotificationStatusPending
	}
}

