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

// PostmarkConfig holds Postmark API configuration
type PostmarkConfig struct {
	ServerToken string `json:"server_token"`
	From        string `json:"from"`
	FromName    string `json:"from_name"`
	ReplyTo     string `json:"reply_to,omitempty"`
	TrackOpens  bool   `json:"track_opens"`
	TrackLinks  string `json:"track_links"` // None, HtmlAndText, HtmlOnly, TextOnly
}

// PostmarkProvider implements notification.Provider for Postmark email service
type PostmarkProvider struct {
	config     PostmarkConfig
	httpClient *http.Client
}

// NewPostmarkProvider creates a new Postmark email provider
func NewPostmarkProvider(config PostmarkConfig) *PostmarkProvider {
	return &PostmarkProvider{
		config:     config,
		httpClient: &http.Client{},
	}
}

// ID returns the provider ID
func (p *PostmarkProvider) ID() string {
	return "postmark"
}

// Type returns the notification type this provider handles
func (p *PostmarkProvider) Type() notification.NotificationType {
	return notification.NotificationTypeEmail
}

// Send sends an email notification via Postmark API
func (p *PostmarkProvider) Send(ctx context.Context, notif *notification.Notification) error {
	// Build request payload according to Postmark API
	from := p.config.From
	if p.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", p.config.FromName, p.config.From)
	}

	payload := map[string]interface{}{
		"From":       from,
		"To":         notif.Recipient,
		"Subject":    notif.Subject,
		"HtmlBody":   notif.Body,
		"TrackOpens": p.config.TrackOpens,
	}

	// Add reply_to if configured
	if p.config.ReplyTo != "" {
		payload["ReplyTo"] = p.config.ReplyTo
	}

	// Add track links if configured
	if p.config.TrackLinks != "" {
		payload["TrackLinks"] = p.config.TrackLinks
	}

	// Add message stream if available in metadata
	if notif.Metadata != nil {
		if messageStream, ok := notif.Metadata["message_stream"].(string); ok && messageStream != "" {
			payload["MessageStream"] = messageStream
		}
		
		// Add tags if available
		if tag, ok := notif.Metadata["tag"].(string); ok && tag != "" {
			payload["Tag"] = tag
		}
		
		// Add metadata
		if metadata, ok := notif.Metadata["postmark_metadata"].(map[string]string); ok && len(metadata) > 0 {
			payload["Metadata"] = metadata
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.postmarkapp.com/email", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("X-Postmark-Server-Token", p.config.ServerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("postmark API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response to get message ID
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Store message ID in metadata for tracking
	if messageID, ok := result["MessageID"].(string); ok && notif.Metadata != nil {
		notif.Metadata["postmark_message_id"] = messageID
	}

	return nil
}

// GetStatus gets the delivery status of a notification
func (p *PostmarkProvider) GetStatus(ctx context.Context, providerID string) (notification.NotificationStatus, error) {
	// Create HTTP request to get message details
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("https://api.postmarkapp.com/messages/outbound/%s/details", providerID), nil)
	if err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("X-Postmark-Server-Token", p.config.ServerToken)
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return notification.NotificationStatusFailed, fmt.Errorf("postmark API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to parse response: %w", err)
	}

	// Map Postmark status to notification status
	status, _ := result["Status"].(string)
	switch status {
	case "Sent":
		return notification.NotificationStatusSent, nil
	case "Delivered":
		return notification.NotificationStatusDelivered, nil
	case "Bounced":
		return notification.NotificationStatusBounced, nil
	default:
		return notification.NotificationStatusPending, nil
	}
}

// ValidateConfig validates the provider configuration
func (p *PostmarkProvider) ValidateConfig() error {
	if p.config.ServerToken == "" {
		return fmt.Errorf("postmark server token is required")
	}
	if p.config.From == "" {
		return fmt.Errorf("from email address is required")
	}
	return nil
}

// PostmarkWebhookEvent represents a Postmark webhook event
type PostmarkWebhookEvent struct {
	RecordType  string                 `json:"RecordType"`
	MessageID   string                 `json:"MessageID"`
	Recipient   string                 `json:"Recipient"`
	Tag         string                 `json:"Tag"`
	DeliveredAt string                 `json:"DeliveredAt"`
	Details     map[string]interface{} `json:"Details"`
}

// ParseWebhookEvent parses a Postmark webhook event
func (p *PostmarkProvider) ParseWebhookEvent(body []byte) (*PostmarkWebhookEvent, error) {
	var event PostmarkWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook event: %w", err)
	}
	return &event, nil
}

// MapWebhookEventToStatus maps a Postmark webhook event type to notification status
func (p *PostmarkProvider) MapWebhookEventToStatus(recordType string) notification.NotificationStatus {
	switch recordType {
	case "Delivery":
		return notification.NotificationStatusDelivered
	case "Bounce":
		return notification.NotificationStatusBounced
	case "SpamComplaint":
		return notification.NotificationStatusFailed
	case "Open":
		return notification.NotificationStatusDelivered // Email was opened, so it was delivered
	case "Click":
		return notification.NotificationStatusDelivered
	default:
		return notification.NotificationStatusPending
	}
}

