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

// NotificationType represents the type of notification.
type NotificationType string

const (
	NotificationTypeEmail NotificationType = "email"
	NotificationTypeSMS   NotificationType = "sms"
	NotificationTypePush  NotificationType = "push"
)

// NotificationStatus represents the status of a notification.
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusDelivered NotificationStatus = "delivered"
	NotificationStatusBounced   NotificationStatus = "bounced"
)

// NotificationPriority represents the priority/criticality of a notification.
type NotificationPriority string

const (
	// PriorityCritical - MFA codes, password reset - must succeed, blocks auth if fails.
	PriorityCritical NotificationPriority = "critical"
	// PriorityHigh - Email verification - important but can retry async.
	PriorityHigh NotificationPriority = "high"
	// PriorityNormal - Welcome emails - best effort with limited retries.
	PriorityNormal NotificationPriority = "normal"
	// PriorityLow - New device alerts - fire and forget, no retries.
	PriorityLow NotificationPriority = "low"
)

// =============================================================================
// TEMPLATE DTO (Data Transfer Object)
// =============================================================================

// Template represents a notification template DTO
// This is separate from schema.NotificationTemplate to maintain proper separation of concerns.
type Template struct {
	ID          xid.ID           `json:"id"`
	AppID       xid.ID           `json:"appId"`
	TemplateKey string           `json:"templateKey"` // e.g., "auth.welcome", "auth.mfa_code"
	Name        string           `json:"name"`
	Type        NotificationType `json:"type"`
	Language    string           `json:"language"` // e.g., "en", "es", "fr"
	Subject     string           `json:"subject,omitempty"`
	Body        string           `json:"body"`
	Variables   []string         `json:"variables"` // Available template variables
	Metadata    map[string]any   `json:"metadata,omitempty"`
	Active      bool             `json:"active"`
	IsDefault   bool             `json:"isDefault"`   // Is this a default template
	IsModified  bool             `json:"isModified"`  // Has it been modified from default
	DefaultHash string           `json:"defaultHash"` // Hash of default content for comparison
	// Audit fields
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

// ToSchema converts the Template DTO to a schema.NotificationTemplate model.
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

// FromSchemaTemplate converts a schema.NotificationTemplate model to Template DTO.
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

// FromSchemaTemplates converts a slice of schema.NotificationTemplate to Template DTOs.
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
// This is separate from schema.Notification to maintain proper separation of concerns.
type Notification struct {
	ID          xid.ID             `json:"id"`
	AppID       xid.ID             `json:"appId"`
	TemplateID  *xid.ID            `json:"templateId,omitempty"`
	Type        NotificationType   `json:"type"`
	Recipient   string             `json:"recipient"` // Email address or phone number
	Subject     string             `json:"subject,omitempty"`
	Body        string             `json:"body"`
	Status      NotificationStatus `json:"status"`
	Error       string             `json:"error,omitempty"`
	ProviderID  string             `json:"providerId,omitempty"` // External provider message ID
	Metadata    map[string]any     `json:"metadata,omitempty"`
	SentAt      *time.Time         `json:"sentAt,omitempty"`
	DeliveredAt *time.Time         `json:"deliveredAt,omitempty"`
	// Audit fields
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ToSchema converts the Notification DTO to a schema.Notification model.
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

// FromSchemaNotification converts a schema.Notification model to Notification DTO.
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

// FromSchemaNotifications converts a slice of schema.Notification to Notification DTOs.
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

// CreateTemplateRequest represents a request to create a template.
type CreateTemplateRequest struct {
	AppID       xid.ID           `json:"appId"               validate:"required"`
	TemplateKey string           `json:"templateKey"         validate:"required"`
	Name        string           `json:"name"                validate:"required"`
	Type        NotificationType `json:"type"                validate:"required"`
	Language    string           `json:"language,omitempty"`
	Subject     string           `json:"subject,omitempty"`
	Body        string           `json:"body"                validate:"required"`
	Variables   []string         `json:"variables,omitempty"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
}

// UpdateTemplateRequest represents a request to update a template.
type UpdateTemplateRequest struct {
	Name      *string        `json:"name,omitempty"`
	Subject   *string        `json:"subject,omitempty"`
	Body      *string        `json:"body,omitempty"`
	Variables []string       `json:"variables,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Active    *bool          `json:"active,omitempty"`
}

// SendRequest represents a request to send a notification.
type SendRequest struct {
	AppID        xid.ID           `json:"appId"                  validate:"required"`
	TemplateName string           `json:"templateName,omitempty"` // Use template
	Type         NotificationType `json:"type"                   validate:"required"`
	Recipient    string           `json:"recipient"              validate:"required"`
	Subject      string           `json:"subject,omitempty"`   // For email (overrides template)
	Body         string           `json:"body,omitempty"`      // Direct body (overrides template)
	Variables    map[string]any   `json:"variables,omitempty"` // Template variables
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

// ListTemplatesResponse represents a paginated response for templates.
type ListTemplatesResponse = pagination.PageResponse[*Template]

// ListNotificationsResponse represents a paginated response for notifications.
type ListNotificationsResponse = pagination.PageResponse[*Notification]

// =============================================================================
// ANALYTICS REPORT TYPES
// =============================================================================

// TemplateAnalyticsReport represents analytics data for a specific template.
type TemplateAnalyticsReport struct {
	TemplateID      xid.ID    `json:"templateId"`
	TemplateName    string    `json:"templateName"`
	TotalSent       int64     `json:"totalSent"`
	TotalDelivered  int64     `json:"totalDelivered"`
	TotalOpened     int64     `json:"totalOpened"`
	TotalClicked    int64     `json:"totalClicked"`
	TotalConverted  int64     `json:"totalConverted"`
	TotalBounced    int64     `json:"totalBounced"`
	TotalComplained int64     `json:"totalComplained"`
	TotalFailed     int64     `json:"totalFailed"`
	DeliveryRate    float64   `json:"deliveryRate"`   // Percentage of sent that were delivered
	OpenRate        float64   `json:"openRate"`       // Percentage of delivered that were opened
	ClickRate       float64   `json:"clickRate"`      // Percentage of opened that were clicked
	ConversionRate  float64   `json:"conversionRate"` // Percentage of clicked that converted
	BounceRate      float64   `json:"bounceRate"`     // Percentage of sent that bounced
	ComplaintRate   float64   `json:"complaintRate"`  // Percentage of delivered that complained
	StartDate       time.Time `json:"startDate"`
	EndDate         time.Time `json:"endDate"`
}

// AppAnalyticsReport represents aggregate analytics data for an app.
type AppAnalyticsReport struct {
	AppID           xid.ID    `json:"appId"`
	TotalSent       int64     `json:"totalSent"`
	TotalDelivered  int64     `json:"totalDelivered"`
	TotalOpened     int64     `json:"totalOpened"`
	TotalClicked    int64     `json:"totalClicked"`
	TotalConverted  int64     `json:"totalConverted"`
	TotalBounced    int64     `json:"totalBounced"`
	TotalComplained int64     `json:"totalComplained"`
	TotalFailed     int64     `json:"totalFailed"`
	DeliveryRate    float64   `json:"deliveryRate"`
	OpenRate        float64   `json:"openRate"`
	ClickRate       float64   `json:"clickRate"`
	ConversionRate  float64   `json:"conversionRate"`
	BounceRate      float64   `json:"bounceRate"`
	ComplaintRate   float64   `json:"complaintRate"`
	StartDate       time.Time `json:"startDate"`
	EndDate         time.Time `json:"endDate"`
}

// OrgAnalyticsReport represents aggregate analytics data for an organization.
type OrgAnalyticsReport struct {
	OrganizationID  xid.ID    `json:"organizationId"`
	TotalSent       int64     `json:"totalSent"`
	TotalDelivered  int64     `json:"totalDelivered"`
	TotalOpened     int64     `json:"totalOpened"`
	TotalClicked    int64     `json:"totalClicked"`
	TotalConverted  int64     `json:"totalConverted"`
	TotalBounced    int64     `json:"totalBounced"`
	TotalComplained int64     `json:"totalComplained"`
	TotalFailed     int64     `json:"totalFailed"`
	DeliveryRate    float64   `json:"deliveryRate"`
	OpenRate        float64   `json:"openRate"`
	ClickRate       float64   `json:"clickRate"`
	ConversionRate  float64   `json:"conversionRate"`
	BounceRate      float64   `json:"bounceRate"`
	ComplaintRate   float64   `json:"complaintRate"`
	StartDate       time.Time `json:"startDate"`
	EndDate         time.Time `json:"endDate"`
}

// =============================================================================
// PROVIDER INTERFACE
// =============================================================================

// Provider represents a notification provider interface.
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

// Repository defines the notification repository interface.
type Repository interface {
	// Template operations
	CreateTemplate(ctx context.Context, template *schema.NotificationTemplate) error
	FindTemplateByID(ctx context.Context, id xid.ID) (*schema.NotificationTemplate, error)
	FindTemplateByName(ctx context.Context, appID xid.ID, name string) (*schema.NotificationTemplate, error)
	FindTemplateByKey(ctx context.Context, appID xid.ID, templateKey, notifType, language string) (*schema.NotificationTemplate, error)
	FindTemplateByKeyOrgScoped(ctx context.Context, appID xid.ID, orgID *xid.ID, templateKey, notifType, language string) (*schema.NotificationTemplate, error)
	ListTemplates(ctx context.Context, filter *ListTemplatesFilter) (*pagination.PageResponse[*schema.NotificationTemplate], error)
	UpdateTemplate(ctx context.Context, id xid.ID, req *UpdateTemplateRequest) error
	UpdateTemplateMetadata(ctx context.Context, id xid.ID, isDefault, isModified bool, defaultHash string) error
	UpdateTemplateAnalytics(ctx context.Context, id xid.ID, sendCount, openCount, clickCount, conversionCount int64) error
	DeleteTemplate(ctx context.Context, id xid.ID) error

	// Template versioning operations
	CreateTemplateVersion(ctx context.Context, version *schema.NotificationTemplateVersion) error
	FindTemplateVersionByID(ctx context.Context, id xid.ID) (*schema.NotificationTemplateVersion, error)
	ListTemplateVersions(ctx context.Context, templateID xid.ID) ([]*schema.NotificationTemplateVersion, error)
	GetLatestTemplateVersion(ctx context.Context, templateID xid.ID) (*schema.NotificationTemplateVersion, error)

	// Notification operations
	CreateNotification(ctx context.Context, notification *schema.Notification) error
	FindNotificationByID(ctx context.Context, id xid.ID) (*schema.Notification, error)
	ListNotifications(ctx context.Context, filter *ListNotificationsFilter) (*pagination.PageResponse[*schema.Notification], error)
	UpdateNotificationStatus(ctx context.Context, id xid.ID, status NotificationStatus, error string, providerID string) error
	UpdateNotificationDelivery(ctx context.Context, id xid.ID, deliveredAt time.Time) error

	// Provider operations
	CreateProvider(ctx context.Context, provider *schema.NotificationProvider) error
	FindProviderByID(ctx context.Context, id xid.ID) (*schema.NotificationProvider, error)
	FindProviderByTypeOrgScoped(ctx context.Context, appID xid.ID, orgID *xid.ID, providerType string) (*schema.NotificationProvider, error)
	ListProviders(ctx context.Context, appID xid.ID, orgID *xid.ID) ([]*schema.NotificationProvider, error)
	UpdateProvider(ctx context.Context, id xid.ID, config map[string]any, isActive, isDefault bool) error
	DeleteProvider(ctx context.Context, id xid.ID) error

	// Analytics operations
	CreateAnalyticsEvent(ctx context.Context, event *schema.NotificationAnalytics) error
	FindAnalyticsByNotificationID(ctx context.Context, notificationID xid.ID) ([]*schema.NotificationAnalytics, error)
	GetTemplateAnalytics(ctx context.Context, templateID xid.ID, startDate, endDate time.Time) (*TemplateAnalyticsReport, error)
	GetAppAnalytics(ctx context.Context, appID xid.ID, startDate, endDate time.Time) (*AppAnalyticsReport, error)
	GetOrgAnalytics(ctx context.Context, orgID xid.ID, startDate, endDate time.Time) (*OrgAnalyticsReport, error)

	// Test operations
	CreateTest(ctx context.Context, test *schema.NotificationTest) error
	FindTestByID(ctx context.Context, id xid.ID) (*schema.NotificationTest, error)
	ListTests(ctx context.Context, templateID xid.ID) ([]*schema.NotificationTest, error)
	UpdateTestStatus(ctx context.Context, id xid.ID, status string, results map[string]any, successCount, failureCount int) error

	// Cleanup operations
	CleanupOldNotifications(ctx context.Context, olderThan time.Time) error
	CleanupOldAnalytics(ctx context.Context, olderThan time.Time) error
	CleanupOldTests(ctx context.Context, olderThan time.Time) error
}

// =============================================================================
// TEMPLATE ENGINE INTERFACE
// =============================================================================

// TemplateEngine defines the template rendering interface.
type TemplateEngine interface {
	// Render renders a template with variables
	Render(template string, variables map[string]any) (string, error)

	// ValidateTemplate validates template syntax
	ValidateTemplate(template string) error

	// ExtractVariables extracts variable names from template
	ExtractVariables(template string) ([]string, error)
}
