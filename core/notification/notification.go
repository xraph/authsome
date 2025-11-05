package notification

import (
	"context"
	"time"

	"github.com/rs/xid"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypePush  NotificationType = "push"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusBounced   NotificationStatus = "bounced"
)

// Template represents a notification template
type Template struct {
	ID             xid.ID                 `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	TemplateKey    string                 `json:"template_key"` // e.g., "auth.welcome", "auth.mfa_code"
	Name           string                 `json:"name"`
	Type           NotificationType       `json:"type"`
	Language       string                 `json:"language"`          // e.g., "en", "es", "fr"
	Subject        string                 `json:"subject,omitempty"` // For email
	Body           string                 `json:"body"`
	Variables      []string               `json:"variables"` // Available template variables
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Active         bool                   `json:"active"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// Notification represents a notification instance
type Notification struct {
	ID             xid.ID                 `json:"id"`
	OrganizationID string                 `json:"organization_id"`
	TemplateID     *xid.ID                `json:"template_id,omitempty"`
	Type           NotificationType       `json:"type"`
	Recipient      string                 `json:"recipient"`         // Email address or phone number
	Subject        string                 `json:"subject,omitempty"` // For email
	Body           string                 `json:"body"`
	Status         NotificationStatus     `json:"status"`
	Error          string                 `json:"error,omitempty"`
	ProviderID     string                 `json:"provider_id,omitempty"` // External provider message ID
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	SentAt         *time.Time             `json:"sent_at,omitempty"`
	DeliveredAt    *time.Time             `json:"delivered_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// SendRequest represents a request to send a notification
type SendRequest struct {
	OrganizationID string                 `json:"organization_id"`
	TemplateName   string                 `json:"template_name,omitempty"` // Use template
	Type           NotificationType       `json:"type"`
	Recipient      string                 `json:"recipient"`
	Subject        string                 `json:"subject,omitempty"`   // For email (overrides template)
	Body           string                 `json:"body,omitempty"`      // Direct body (overrides template)
	Variables      map[string]interface{} `json:"variables,omitempty"` // Template variables
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// CreateTemplateRequest represents a request to create a template
type CreateTemplateRequest struct {
	OrganizationID string                 `json:"organization_id"`
	TemplateKey    string                 `json:"template_key"`
	Name           string                 `json:"name"`
	Type           NotificationType       `json:"type"`
	Language       string                 `json:"language,omitempty"`
	Subject        string                 `json:"subject,omitempty"`
	Body           string                 `json:"body"`
	Variables      []string               `json:"variables,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateTemplateRequest represents a request to update a template
type UpdateTemplateRequest struct {
	Subject   *string                `json:"subject,omitempty"`
	Body      *string                `json:"body,omitempty"`
	Variables []string               `json:"variables,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Active    *bool                  `json:"active,omitempty"`
}

// ListTemplatesRequest represents a request to list templates
type ListTemplatesRequest struct {
	OrganizationID string           `json:"organization_id"`
	Type           NotificationType `json:"type,omitempty"`
	Language       string           `json:"language,omitempty"`
	Active         *bool            `json:"active,omitempty"`
	Offset         int              `json:"offset"`
	Limit          int              `json:"limit"`
}

// ListNotificationsRequest represents a request to list notifications
type ListNotificationsRequest struct {
	OrganizationID string             `json:"organization_id"`
	Type           NotificationType   `json:"type,omitempty"`
	Status         NotificationStatus `json:"status,omitempty"`
	Recipient      string             `json:"recipient,omitempty"`
	Offset         int                `json:"offset"`
	Limit          int                `json:"limit"`
}

// Provider represents a notification provider interface
type Provider interface {
	// ID returns the provider identifier
	ID() string

	// Type returns the notification type this provider handles
	Type() NotificationType

	// Send sends a notification
	Send(ctx context.Context, notification *Notification) error

	// GetStatus gets the delivery status of a notification
	GetStatus(ctx context.Context, providerID string) (NotificationStatus, error)

	// ValidateConfig validates the provider configuration
	ValidateConfig() error
}

// Repository defines the notification repository interface
type Repository interface {
	// Template operations
	CreateTemplate(ctx context.Context, template *Template) error
	FindTemplateByID(ctx context.Context, id xid.ID) (*Template, error)
	FindTemplateByName(ctx context.Context, orgID, name string) (*Template, error)
	FindTemplateByKey(ctx context.Context, orgID, templateKey, notifType, language string) (*Template, error)
	ListTemplates(ctx context.Context, req *ListTemplatesRequest) ([]*Template, int64, error)
	UpdateTemplate(ctx context.Context, id xid.ID, req *UpdateTemplateRequest) error
	DeleteTemplate(ctx context.Context, id xid.ID) error

	// Notification operations
	CreateNotification(ctx context.Context, notification *Notification) error
	FindNotificationByID(ctx context.Context, id xid.ID) (*Notification, error)
	ListNotifications(ctx context.Context, req *ListNotificationsRequest) ([]*Notification, int64, error)
	UpdateNotificationStatus(ctx context.Context, id xid.ID, status NotificationStatus, error string, providerID string) error
	UpdateNotificationDelivery(ctx context.Context, id xid.ID, deliveredAt time.Time) error

	// Cleanup operations
	CleanupOldNotifications(ctx context.Context, olderThan time.Time) error
}

// TemplateEngine defines the template rendering interface
type TemplateEngine interface {
	// Render renders a template with variables
	Render(template string, variables map[string]interface{}) (string, error)

	// ValidateTemplate validates template syntax
	ValidateTemplate(template string) error

	// ExtractVariables extracts variable names from template
	ExtractVariables(template string) ([]string, error)
}
