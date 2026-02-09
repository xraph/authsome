package notification

// =============================================================================
// Bridge Function Input/Output Types
// =============================================================================

// GetSettingsInput is the input for bridgeGetSettings.
type GetSettingsInput struct {
	// No input needed - appId extracted from context
}

// GetSettingsResult is the output for bridgeGetSettings.
type GetSettingsResult struct {
	Settings NotificationSettingsDTO `json:"settings"`
}

// UpdateSettingsInput is the input for bridgeUpdateSettings.
type UpdateSettingsInput struct {
	AppName      string                   `json:"appName,omitempty"`
	Auth         *AuthAutoSendDTO         `json:"auth,omitempty"`
	Organization *OrganizationAutoSendDTO `json:"organization,omitempty"`
	Session      *SessionAutoSendDTO      `json:"session,omitempty"`
	Account      *AccountAutoSendDTO      `json:"account,omitempty"`
}

// UpdateSettingsResult is the output for bridgeUpdateSettings.
type UpdateSettingsResult struct {
	Success  bool                    `json:"success"`
	Settings NotificationSettingsDTO `json:"settings"`
	Message  string                  `json:"message,omitempty"`
}

// =============================================================================
// DTO Types
// =============================================================================

// NotificationSettingsDTO represents notification plugin settings.
type NotificationSettingsDTO struct {
	AppName      string                  `json:"appName"`
	Auth         AuthAutoSendDTO         `json:"auth"`
	Organization OrganizationAutoSendDTO `json:"organization"`
	Session      SessionAutoSendDTO      `json:"session"`
	Account      AccountAutoSendDTO      `json:"account"`
}

// AuthAutoSendDTO represents authentication notification settings.
type AuthAutoSendDTO struct {
	Welcome           bool `json:"welcome"`
	VerificationEmail bool `json:"verificationEmail"`
	MagicLink         bool `json:"magicLink"`
	EmailOTP          bool `json:"emailOtp"`
	MFACode           bool `json:"mfaCode"`
	PasswordReset     bool `json:"passwordReset"`
}

// OrganizationAutoSendDTO represents organization notification settings.
type OrganizationAutoSendDTO struct {
	Invite        bool `json:"invite"`
	MemberAdded   bool `json:"memberAdded"`
	MemberRemoved bool `json:"memberRemoved"`
	RoleChanged   bool `json:"roleChanged"`
	Transfer      bool `json:"transfer"`
	Deleted       bool `json:"deleted"`
	MemberLeft    bool `json:"memberLeft"`
}

// SessionAutoSendDTO represents session/security notification settings.
type SessionAutoSendDTO struct {
	NewDevice       bool `json:"newDevice"`
	NewLocation     bool `json:"newLocation"`
	SuspiciousLogin bool `json:"suspiciousLogin"`
	DeviceRemoved   bool `json:"deviceRemoved"`
	AllRevoked      bool `json:"allRevoked"`
}

// AccountAutoSendDTO represents account lifecycle notification settings.
type AccountAutoSendDTO struct {
	EmailChangeRequest bool `json:"emailChangeRequest"`
	EmailChanged       bool `json:"emailChanged"`
	PasswordChanged    bool `json:"passwordChanged"`
	UsernameChanged    bool `json:"usernameChanged"`
	Deleted            bool `json:"deleted"`
	Suspended          bool `json:"suspended"`
	Reactivated        bool `json:"reactivated"`
}

// =============================================================================
// Templates Bridge Types
// =============================================================================

// ListTemplatesInput is the input for listing templates.
type ListTemplatesInput struct {
	Page     int     `json:"page,omitempty"`
	Limit    int     `json:"limit,omitempty"`
	Type     *string `json:"type,omitempty"`
	Language *string `json:"language,omitempty"`
	Active   *bool   `json:"active,omitempty"`
}

// ListTemplatesResult is the output for listing templates.
type ListTemplatesResult struct {
	Templates  []TemplateDTO `json:"templates"`
	Pagination PaginationDTO `json:"pagination"`
}

// GetTemplateInput is the input for getting a single template.
type GetTemplateInput struct {
	TemplateID string `json:"templateId"`
}

// GetTemplateResult is the output for getting a single template.
type GetTemplateResult struct {
	Template TemplateDTO `json:"template"`
}

// CreateTemplateInput is the input for creating a template.
type CreateTemplateInput struct {
	TemplateKey string         `json:"templateKey"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Language    string         `json:"language,omitempty"`
	Subject     string         `json:"subject,omitempty"`
	Body        string         `json:"body"`
	Variables   []string       `json:"variables,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// CreateTemplateResult is the output for creating a template.
type CreateTemplateResult struct {
	Success  bool        `json:"success"`
	Template TemplateDTO `json:"template"`
	Message  string      `json:"message,omitempty"`
}

// UpdateTemplateInput is the input for updating a template.
type UpdateTemplateInput struct {
	TemplateID string         `json:"templateId"`
	Name       *string        `json:"name,omitempty"`
	Subject    *string        `json:"subject,omitempty"`
	Body       *string        `json:"body,omitempty"`
	Variables  []string       `json:"variables,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Active     *bool          `json:"active,omitempty"`
}

// UpdateTemplateResult is the output for updating a template.
type UpdateTemplateResult struct {
	Success  bool        `json:"success"`
	Template TemplateDTO `json:"template"`
	Message  string      `json:"message,omitempty"`
}

// DeleteTemplateInput is the input for deleting a template.
type DeleteTemplateInput struct {
	TemplateID string `json:"templateId"`
}

// DeleteTemplateResult is the output for deleting a template.
type DeleteTemplateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// PreviewTemplateInput is the input for previewing a template.
type PreviewTemplateInput struct {
	TemplateID string         `json:"templateId"`
	Variables  map[string]any `json:"variables,omitempty"`
}

// PreviewTemplateResult is the output for previewing a template.
type PreviewTemplateResult struct {
	Subject    string `json:"subject"`
	Body       string `json:"body"`
	RenderedAt string `json:"renderedAt"`
}

// TestSendTemplateInput is the input for test sending a template.
type TestSendTemplateInput struct {
	TemplateID string         `json:"templateId"`
	Recipient  string         `json:"recipient"`
	Variables  map[string]any `json:"variables,omitempty"`
}

// TestSendTemplateResult is the output for test sending a template.
type TestSendTemplateResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TemplateDTO represents a notification template.
type TemplateDTO struct {
	ID          string         `json:"id"`
	AppID       string         `json:"appId"`
	TemplateKey string         `json:"templateKey"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Language    string         `json:"language"`
	Subject     string         `json:"subject,omitempty"`
	Body        string         `json:"body"`
	Variables   []string       `json:"variables"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Active      bool           `json:"active"`
	IsDefault   bool           `json:"isDefault"`
	IsModified  bool           `json:"isModified"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
}

// =============================================================================
// Overview/Statistics Bridge Types
// =============================================================================

// GetOverviewStatsInput is the input for getting overview statistics.
type GetOverviewStatsInput struct {
	Days      *int    `json:"days,omitempty"`      // Number of days to fetch stats for
	StartDate *string `json:"startDate,omitempty"` // ISO date
	EndDate   *string `json:"endDate,omitempty"`   // ISO date
}

// GetOverviewStatsResult is the output for getting overview statistics.
type GetOverviewStatsResult struct {
	Stats OverviewStatsDTO `json:"stats"`
}

// OverviewStatsDTO represents overview statistics.
type OverviewStatsDTO struct {
	TotalSent      int64   `json:"totalSent"`
	TotalDelivered int64   `json:"totalDelivered"`
	TotalOpened    int64   `json:"totalOpened"`
	TotalClicked   int64   `json:"totalClicked"`
	TotalBounced   int64   `json:"totalBounced"`
	TotalFailed    int64   `json:"totalFailed"`
	DeliveryRate   float64 `json:"deliveryRate"`
	OpenRate       float64 `json:"openRate"`
	ClickRate      float64 `json:"clickRate"`
	BounceRate     float64 `json:"bounceRate"`
}

// =============================================================================
// Provider Bridge Types
// =============================================================================

// GetProvidersInput is the input for getting providers configuration.
type GetProvidersInput struct {
}

// GetProvidersResult is the output for getting providers configuration.
type GetProvidersResult struct {
	Providers ProvidersConfigDTO `json:"providers"`
}

// UpdateProvidersInput is the input for updating providers configuration.
type UpdateProvidersInput struct {
	EmailProvider *EmailProviderDTO `json:"emailProvider,omitempty"`
	SMSProvider   *SMSProviderDTO   `json:"smsProvider,omitempty"`
}

// UpdateProvidersResult is the output for updating providers configuration.
type UpdateProvidersResult struct {
	Success   bool               `json:"success"`
	Providers ProvidersConfigDTO `json:"providers"`
	Message   string             `json:"message,omitempty"`
}

// TestProviderInput is the input for testing a provider.
type TestProviderInput struct {
	ProviderType string `json:"providerType"` // "email" or "sms"
	Recipient    string `json:"recipient"`
}

// TestProviderResult is the output for testing a provider.
type TestProviderResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ProvidersConfigDTO represents providers configuration.
type ProvidersConfigDTO struct {
	EmailProvider EmailProviderDTO `json:"emailProvider"`
	SMSProvider   SMSProviderDTO   `json:"smsProvider"`
}

// EmailProviderDTO represents email provider configuration.
type EmailProviderDTO struct {
	Type      string         `json:"type"` // "smtp", "sendgrid", "postmark", "mailersend", "resend"
	Enabled   bool           `json:"enabled"`
	Config    map[string]any `json:"config,omitempty"`
	FromName  string         `json:"fromName"`
	FromEmail string         `json:"fromEmail"`
}

// SMSProviderDTO represents SMS provider configuration.
type SMSProviderDTO struct {
	Type    string         `json:"type"` // "twilio", "vonage", "aws-sns"
	Enabled bool           `json:"enabled"`
	Config  map[string]any `json:"config,omitempty"`
}

// =============================================================================
// Analytics Bridge Types
// =============================================================================

// GetAnalyticsInput is the input for getting analytics data.
type GetAnalyticsInput struct {
	Days       *int    `json:"days,omitempty"`
	StartDate  *string `json:"startDate,omitempty"`
	EndDate    *string `json:"endDate,omitempty"`
	TemplateID *string `json:"templateId,omitempty"`
}

// GetAnalyticsResult is the output for getting analytics data.
type GetAnalyticsResult struct {
	Analytics AnalyticsDTO `json:"analytics"`
}

// AnalyticsDTO represents detailed analytics.
type AnalyticsDTO struct {
	Overview     OverviewStatsDTO         `json:"overview"`
	ByTemplate   []TemplateAnalyticsDTO   `json:"byTemplate"`
	ByDay        []DailyAnalyticsDTO      `json:"byDay"`
	TopTemplates []TemplatePerformanceDTO `json:"topTemplates"`
}

// TemplateAnalyticsDTO represents analytics for a specific template.
type TemplateAnalyticsDTO struct {
	TemplateID     string  `json:"templateId"`
	TemplateName   string  `json:"templateName"`
	TotalSent      int64   `json:"totalSent"`
	TotalDelivered int64   `json:"totalDelivered"`
	TotalOpened    int64   `json:"totalOpened"`
	TotalClicked   int64   `json:"totalClicked"`
	DeliveryRate   float64 `json:"deliveryRate"`
	OpenRate       float64 `json:"openRate"`
	ClickRate      float64 `json:"clickRate"`
}

// DailyAnalyticsDTO represents analytics for a specific day.
type DailyAnalyticsDTO struct {
	Date           string  `json:"date"`
	TotalSent      int64   `json:"totalSent"`
	TotalDelivered int64   `json:"totalDelivered"`
	TotalOpened    int64   `json:"totalOpened"`
	TotalClicked   int64   `json:"totalClicked"`
	DeliveryRate   float64 `json:"deliveryRate"`
	OpenRate       float64 `json:"openRate"`
}

// TemplatePerformanceDTO represents template performance ranking.
type TemplatePerformanceDTO struct {
	TemplateID   string  `json:"templateId"`
	TemplateName string  `json:"templateName"`
	TotalSent    int64   `json:"totalSent"`
	OpenRate     float64 `json:"openRate"`
	ClickRate    float64 `json:"clickRate"`
}

// =============================================================================
// Common DTOs
// =============================================================================

// PaginationDTO represents pagination metadata.
type PaginationDTO struct {
	CurrentPage int   `json:"currentPage"`
	TotalPages  int   `json:"totalPages"`
	TotalCount  int64 `json:"totalCount"`
	PageSize    int   `json:"pageSize"`
	HasNext     bool  `json:"hasNext"`
	HasPrev     bool  `json:"hasPrev"`
}

// =============================================================================
// Email Builder Bridge Types
// =============================================================================

// SaveBuilderTemplateInput is the input for saving a template from the visual builder.
type SaveBuilderTemplateInput struct {
	TemplateID  string `json:"templateId,omitempty"` // Empty for new template
	Name        string `json:"name"`
	TemplateKey string `json:"templateKey"`
	Subject     string `json:"subject"`
	BuilderJSON string `json:"builderJson"` // JSON of builder.Document
}

// SaveBuilderTemplateResult is the output for saving a builder template.
type SaveBuilderTemplateResult struct {
	Success    bool   `json:"success"`
	TemplateID string `json:"templateId,omitempty"`
	Message    string `json:"message"`
}

// ListNotificationsHistoryInput is the input for listing notification history.
type ListNotificationsHistoryInput struct {
	Page      int     `json:"page,omitempty"`
	Limit     int     `json:"limit,omitempty"`
	Type      *string `json:"type,omitempty"`      // email, sms, push
	Status    *string `json:"status,omitempty"`    // pending, sent, failed, delivered, bounced
	Recipient *string `json:"recipient,omitempty"` // Filter by recipient
}

// NotificationHistoryDTO represents a notification record in the history.
type NotificationHistoryDTO struct {
	ID          string         `json:"id"`
	AppID       string         `json:"appId"`
	TemplateID  *string        `json:"templateId,omitempty"`
	Type        string         `json:"type"`
	Recipient   string         `json:"recipient"`
	Subject     string         `json:"subject,omitempty"`
	Body        string         `json:"body"`
	Status      string         `json:"status"`
	Error       string         `json:"error,omitempty"`
	ProviderID  string         `json:"providerId,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	SentAt      *string        `json:"sentAt,omitempty"`      // ISO 8601
	DeliveredAt *string        `json:"deliveredAt,omitempty"` // ISO 8601
	CreatedAt   string         `json:"createdAt"`             // ISO 8601
	UpdatedAt   string         `json:"updatedAt"`             // ISO 8601
}

// ListNotificationsHistoryResult is the output for listing notification history.
type ListNotificationsHistoryResult struct {
	Notifications []NotificationHistoryDTO `json:"notifications"`
	Pagination    PaginationDTO            `json:"pagination"`
}

// GetNotificationDetailInput is the input for getting a single notification.
type GetNotificationDetailInput struct {
	NotificationID string `json:"notificationId"`
}

// GetNotificationDetailResult is the output for getting a single notification.
type GetNotificationDetailResult struct {
	Notification NotificationHistoryDTO `json:"notification"`
}
