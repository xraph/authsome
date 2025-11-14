package notification

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// =============================================================================
// NOTIFICATION TYPE ENUMS
// =============================================================================

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

// =============================================================================
// TEMPLATE DTO (Data Transfer Object)
// =============================================================================

// Template represents a notification template DTO
// This is separate from schema.NotificationTemplate to maintain proper separation of concerns
type Template struct {
	ID          xid.ID                 `json:"id"`
	AppID       xid.ID                 `json:"appId"`
	TemplateKey string                 `json:"templateKey"` // e.g., "auth.welcome", "auth.mfa_code"
	Name        string                 `json:"name"`
	Type        NotificationType       `json:"type"`
	Language    string                 `json:"language"` // e.g., "en", "es", "fr"
	Subject     string                 `json:"subject,omitempty"`
	Body        string                 `json:"body"`
	Variables   []string               `json:"variables"` // Available template variables
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Active      bool                   `json:"active"`
	IsDefault   bool                   `json:"isDefault"`   // Is this a default template
	IsModified  bool                   `json:"isModified"`  // Has it been modified from default
	DefaultHash string                 `json:"defaultHash"` // Hash of default content for comparison
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Template DTO to a schema.NotificationTemplate model
func (t *Template) ToSchema() *schema.NotificationTemplate {
	return &schema.NotificationTemplate{
		ID:          t.ID,
		AppID:       t.AppID,
		TemplateKey: t.TemplateKey,
		Name:        t.Name,
		Type:        string(t.Type),
		Language:    t.Language,
		Subject:     t.Subject,
		Body:        t.Body,
		Variables:   t.Variables,
		Metadata:    t.Metadata,
		Active:      t.Active,
		IsDefault:   t.IsDefault,
		IsModified:  t.IsModified,
		DefaultHash: t.DefaultHash,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		DeletedAt:   t.DeletedAt,
	}
}

// FromSchemaTemplate converts a schema.NotificationTemplate model to Template DTO
func FromSchemaTemplate(st *schema.NotificationTemplate) *Template {
	if st == nil {
		return nil
	}
	return &Template{
		ID:          st.ID,
		AppID:       st.AppID,
		TemplateKey: st.TemplateKey,
		Name:        st.Name,
		Type:        NotificationType(st.Type),
		Language:    st.Language,
		Subject:     st.Subject,
		Body:        st.Body,
		Variables:   st.Variables,
		Metadata:    st.Metadata,
		Active:      st.Active,
		IsDefault:   st.IsDefault,
		IsModified:  st.IsModified,
		DefaultHash: st.DefaultHash,
		CreatedAt:   st.CreatedAt,
		UpdatedAt:   st.UpdatedAt,
		DeletedAt:   st.DeletedAt,
	}
}

// FromSchemaTemplates converts a slice of schema.NotificationTemplate to Template DTOs
func FromSchemaTemplates(templates []*schema.NotificationTemplate) []*Template {
	result := make([]*Template, len(templates))
	for i, t := range templates {
		result[i] = FromSchemaTemplate(t)
	}
	return result
}

// =============================================================================
// NOTIFICATION DTO (Data Transfer Object)
// =============================================================================

// Notification represents a notification instance DTO
// This is separate from schema.Notification to maintain proper separation of concerns
type Notification struct {
	ID         xid.ID                 `json:"id"`
	AppID      xid.ID                 `json:"appId"`
	TemplateID *xid.ID                `json:"templateId,omitempty"`
	Type       NotificationType       `json:"type"`
	Recipient  string                 `json:"recipient"` // Email address or phone number
	Subject    string                 `json:"subject,omitempty"`
	Body       string                 `json:"body"`
	Status     NotificationStatus     `json:"status"`
	Error      string                 `json:"error,omitempty"`
	ProviderID string                 `json:"providerId,omitempty"` // External provider message ID
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	SentAt     *time.Time             `json:"sentAt,omitempty"`
	DeliveredAt *time.Time            `json:"deliveredAt,omitempty"`
	// Audit fields
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ToSchema converts the Notification DTO to a schema.Notification model
func (n *Notification) ToSchema() *schema.Notification {
	return &schema.Notification{
		ID:          n.ID,
		AppID:       n.AppID,
		TemplateID:  n.TemplateID,
		Type:        string(n.Type),
		Recipient:   n.Recipient,
		Subject:     n.Subject,
		Body:        n.Body,
		Status:      string(n.Status),
		Error:       n.Error,
		ProviderID:  n.ProviderID,
		Metadata:    n.Metadata,
		SentAt:      n.SentAt,
		DeliveredAt: n.DeliveredAt,
		CreatedAt:   n.CreatedAt,
		UpdatedAt:   n.UpdatedAt,
	}
}

// FromSchemaNotification converts a schema.Notification model to Notification DTO
func FromSchemaNotification(sn *schema.Notification) *Notification {
	if sn == nil {
		return nil
	}
	return &Notification{
		ID:          sn.ID,
		AppID:       sn.AppID,
		TemplateID:  sn.TemplateID,
		Type:        NotificationType(sn.Type),
		Recipient:   sn.Recipient,
		Subject:     sn.Subject,
		Body:        sn.Body,
		Status:      NotificationStatus(sn.Status),
		Error:       sn.Error,
		ProviderID:  sn.ProviderID,
		Metadata:    sn.Metadata,
		SentAt:      sn.SentAt,
		DeliveredAt: sn.DeliveredAt,
		CreatedAt:   sn.CreatedAt,
		UpdatedAt:   sn.UpdatedAt,
	}
}

// FromSchemaNotifications converts a slice of schema.Notification to Notification DTOs
func FromSchemaNotifications(notifications []*schema.Notification) []*Notification {
	result := make([]*Notification, len(notifications))
	for i, n := range notifications {
		result[i] = FromSchemaNotification(n)
	}
	return result
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// CreateTemplateRequest represents a request to create a template
type CreateTemplateRequest struct {
	AppID       xid.ID                 `json:"appId" validate:"required"`
	TemplateKey string                 `json:"templateKey" validate:"required"`
	Name        string                 `json:"name" validate:"required"`
	Type        NotificationType       `json:"type" validate:"required"`
	Language    string                 `json:"language,omitempty"`
	Subject     string                 `json:"subject,omitempty"`
	Body        string                 `json:"body" validate:"required"`
	Variables   []string               `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateTemplateRequest represents a request to update a template
type UpdateTemplateRequest struct {
	Name      *string                `json:"name,omitempty"`
	Subject   *string                `json:"subject,omitempty"`
	Body      *string                `json:"body,omitempty"`
	Variables []string               `json:"variables,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Active    *bool                  `json:"active,omitempty"`
}

// SendRequest represents a request to send a notification
type SendRequest struct {
	AppID        xid.ID                 `json:"appId" validate:"required"`
	TemplateName string                 `json:"templateName,omitempty"` // Use template
	Type         NotificationType       `json:"type" validate:"required"`
	Recipient    string                 `json:"recipient" validate:"required"`
	Subject      string                 `json:"subject,omitempty"`   // For email (overrides template)
	Body         string                 `json:"body,omitempty"`      // Direct body (overrides template)
	Variables    map[string]interface{} `json:"variables,omitempty"` // Template variables
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ListTemplatesResponse represents a paginated response for templates
type ListTemplatesResponse = pagination.PageResponse[*Template]

// ListNotificationsResponse represents a paginated response for notifications
type ListNotificationsResponse = pagination.PageResponse[*Notification]

// =============================================================================
// PROVIDER INTERFACE
// =============================================================================

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

// =============================================================================
// REPOSITORY INTERFACE
// =============================================================================

// Repository defines the notification repository interface
type Repository interface {
	// Template operations
	CreateTemplate(ctx context.Context, template *schema.NotificationTemplate) error
	FindTemplateByID(ctx context.Context, id xid.ID) (*schema.NotificationTemplate, error)
	FindTemplateByName(ctx context.Context, appID xid.ID, name string) (*schema.NotificationTemplate, error)
	FindTemplateByKey(ctx context.Context, appID xid.ID, templateKey, notifType, language string) (*schema.NotificationTemplate, error)
	ListTemplates(ctx context.Context, filter *ListTemplatesFilter) (*pagination.PageResponse[*schema.NotificationTemplate], error)
	UpdateTemplate(ctx context.Context, id xid.ID, req *UpdateTemplateRequest) error
	UpdateTemplateMetadata(ctx context.Context, id xid.ID, isDefault, isModified bool, defaultHash string) error
	DeleteTemplate(ctx context.Context, id xid.ID) error

	// Notification operations
	CreateNotification(ctx context.Context, notification *schema.Notification) error
	FindNotificationByID(ctx context.Context, id xid.ID) (*schema.Notification, error)
	ListNotifications(ctx context.Context, filter *ListNotificationsFilter) (*pagination.PageResponse[*schema.Notification], error)
	UpdateNotificationStatus(ctx context.Context, id xid.ID, status NotificationStatus, error string, providerID string) error
	UpdateNotificationDelivery(ctx context.Context, id xid.ID, deliveredAt time.Time) error

	// Cleanup operations
	CleanupOldNotifications(ctx context.Context, olderThan time.Time) error
}

// =============================================================================
// TEMPLATE ENGINE INTERFACE
// =============================================================================

// TemplateEngine defines the template rendering interface
type TemplateEngine interface {
	// Render renders a template with variables
	Render(template string, variables map[string]interface{}) (string, error)

	// ValidateTemplate validates template syntax
	ValidateTemplate(template string) error

	// ExtractVariables extracts variable names from template
	ExtractVariables(template string) ([]string, error)
}
