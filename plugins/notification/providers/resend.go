package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/internal/errs"
)

// ResendConfig holds Resend API configuration.
type ResendConfig struct {
	APIKey   string `json:"api_key"`
	From     string `json:"from"`
	FromName string `json:"from_name"`
	ReplyTo  string `json:"reply_to,omitempty"`
}

// ResendProvider implements notification.Provider for Resend email service.
type ResendProvider struct {
	config     ResendConfig
	httpClient *http.Client
}

// NewResendProvider creates a new Resend email provider.
func NewResendProvider(config ResendConfig) *ResendProvider {
	return &ResendProvider{
		config:     config,
		httpClient: &http.Client{},
	}
}

// ID returns the provider ID.
func (p *ResendProvider) ID() string {
	return "resend"
}

// Type returns the notification type this provider handles.
func (p *ResendProvider) Type() notification.NotificationType {
	return notification.NotificationTypeEmail
}

// Send sends an email notification via Resend API.
func (p *ResendProvider) Send(ctx context.Context, notif *notification.Notification) error {
	// Build request payload
	from := p.config.From
	if p.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", p.config.FromName, p.config.From)
	}

	payload := map[string]any{
		"from":    from,
		"to":      []string{notif.Recipient},
		"subject": notif.Subject,
		"html":    notif.Body,
	}

	// Add reply_to if configured
	if p.config.ReplyTo != "" {
		payload["reply_to"] = p.config.ReplyTo
	}

	// Add tags from metadata if available
	if notif.Metadata != nil {
		if tags, ok := notif.Metadata["tags"].([]string); ok && len(tags) > 0 {
			payload["tags"] = tags
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

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
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response to get email ID
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Store email ID in metadata for tracking
	if emailID, ok := result["id"].(string); ok && notif.Metadata != nil {
		notif.Metadata["resend_email_id"] = emailID
	}

	return nil
}

// GetStatus gets the delivery status of a notification.
func (p *ResendProvider) GetStatus(ctx context.Context, providerID string) (notification.NotificationStatus, error) {
	// Create HTTP request to get email status
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.resend.com/emails/"+providerID, nil)
	if err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)

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
		return notification.NotificationStatusFailed, fmt.Errorf("resend API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return notification.NotificationStatusFailed, fmt.Errorf("failed to parse response: %w", err)
	}

	// Map Resend status to notification status
	status, _ := result["last_event"].(string)
	switch status {
	case "delivered":
		return notification.NotificationStatusDelivered, nil
	case "bounced":
		return notification.NotificationStatusBounced, nil
	case "complained":
		return notification.NotificationStatusFailed, nil
	case "sent":
		return notification.NotificationStatusSent, nil
	default:
		return notification.NotificationStatusPending, nil
	}
}

// ValidateConfig validates the provider configuration.
func (p *ResendProvider) ValidateConfig() error {
	if p.config.APIKey == "" {
		return errs.RequiredField("api_key")
	}

	if p.config.From == "" {
		return errs.RequiredField("from")
	}

	return nil
}
