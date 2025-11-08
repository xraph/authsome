package notification

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/providers/email"
	"github.com/xraph/authsome/providers/sms"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the notification template management plugin
type Plugin struct {
	service       *notification.Service
	templateSvc   *TemplateService
	db            *bun.DB
	config        Config
	forgeConfig   forge.ConfigManager
	defaultsAdded bool
}

// NewPlugin creates a new notification plugin instance
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "notification"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(dep interface{}) error {
	type authInstance interface {
		GetDB() *bun.DB
	}

	authInst, ok := dep.(authInstance)
	if !ok {
		return fmt.Errorf("notification plugin requires auth instance with GetDB method")
	}

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for notification plugin")
	}

	p.db = db

	// Use default config
	p.config = DefaultConfig()

	// Initialize repositories
	notificationRepo := repo.NewNotificationRepository(db)
	auditSvc := audit.NewService(repo.NewAuditRepository(db))

	// Create template engine
	templateEngine := NewTemplateEngine()

	// Initialize core notification service
	notificationConfig := notification.Config{
		DefaultProvider: make(map[notification.NotificationType]string),
		RetryAttempts:   p.config.RetryAttempts,
		RetryDelay:      p.config.RetryDelay,
		CleanupAfter:    p.config.CleanupAfter,
	}

	p.service = notification.NewService(
		notificationRepo,
		templateEngine,
		auditSvc,
		notificationConfig,
	)

	// Register providers based on configuration
	if err := p.registerProviders(); err != nil {
		return fmt.Errorf("failed to register providers: %w", err)
	}

	// Initialize template service
	p.templateSvc = NewTemplateService(p.service, notificationRepo, p.config)

	return nil
}

// RegisterRoutes registers HTTP routes for the plugin
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	if p.service == nil || p.templateSvc == nil {
		return nil
	}

	// Create handler
	handler := NewHandler(p.service, p.templateSvc, p.config)

	// Template management routes
	templates := router.Group("/templates")
	{
		templates.POST("", handler.CreateTemplate,
			forge.WithName("notification.templates.create"),
			forge.WithSummary("Create notification template"),
			forge.WithDescription("Creates a new notification template for email or SMS with subject, body, and variables"),
			forge.WithResponseSchema(201, "Template created", NotificationTemplateResponse{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithTags("Notification", "Templates"),
			forge.WithValidation(true),
		)
		templates.GET("", handler.ListTemplates,
			forge.WithName("notification.templates.list"),
			forge.WithSummary("List notification templates"),
			forge.WithDescription("Lists all notification templates with optional filtering by organization, type, and language"),
			forge.WithResponseSchema(200, "Templates retrieved", NotificationTemplateListResponse{}),
			forge.WithTags("Notification", "Templates"),
		)
		templates.GET("/:id", handler.GetTemplate,
			forge.WithName("notification.templates.get"),
			forge.WithSummary("Get notification template"),
			forge.WithDescription("Retrieves details of a specific notification template by ID"),
			forge.WithResponseSchema(200, "Template retrieved", NotificationTemplateResponse{}),
			forge.WithResponseSchema(404, "Template not found", NotificationErrorResponse{}),
			forge.WithTags("Notification", "Templates"),
		)
		templates.PUT("/:id", handler.UpdateTemplate,
			forge.WithName("notification.templates.update"),
			forge.WithSummary("Update notification template"),
			forge.WithDescription("Updates an existing notification template with new subject, body, or variables"),
			forge.WithResponseSchema(200, "Template updated", NotificationTemplateResponse{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithResponseSchema(404, "Template not found", NotificationErrorResponse{}),
			forge.WithTags("Notification", "Templates"),
			forge.WithValidation(true),
		)
		templates.DELETE("/:id", handler.DeleteTemplate,
			forge.WithName("notification.templates.delete"),
			forge.WithSummary("Delete notification template"),
			forge.WithDescription("Deletes a notification template by ID"),
			forge.WithResponseSchema(200, "Template deleted", NotificationStatusResponse{}),
			forge.WithResponseSchema(404, "Template not found", NotificationErrorResponse{}),
			forge.WithTags("Notification", "Templates"),
		)
		templates.POST("/:id/preview", handler.PreviewTemplate,
			forge.WithName("notification.templates.preview"),
			forge.WithSummary("Preview notification template"),
			forge.WithDescription("Renders a notification template with provided variables for preview"),
			forge.WithResponseSchema(200, "Template preview", NotificationPreviewResponse{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithResponseSchema(404, "Template not found", NotificationErrorResponse{}),
			forge.WithTags("Notification", "Templates"),
			forge.WithValidation(true),
		)
		templates.POST("/render", handler.RenderTemplate,
			forge.WithName("notification.templates.render"),
			forge.WithSummary("Render notification template"),
			forge.WithDescription("Renders a notification template with provided variables without saving"),
			forge.WithResponseSchema(200, "Template rendered", NotificationPreviewResponse{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithTags("Notification", "Templates"),
			forge.WithValidation(true),
		)
	}

	// Notification sending routes
	notifications := router.Group("/notifications")
	{
		notifications.POST("/send", handler.SendNotification,
			forge.WithName("notification.send"),
			forge.WithSummary("Send notification"),
			forge.WithDescription("Sends a notification (email or SMS) using a template with provided variables"),
			forge.WithResponseSchema(200, "Notification sent", NotificationResponse{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithTags("Notification", "Sending"),
			forge.WithValidation(true),
		)
		notifications.GET("", handler.ListNotifications,
			forge.WithName("notification.list"),
			forge.WithSummary("List notifications"),
			forge.WithDescription("Lists all sent notifications with optional filtering by organization, status, and type"),
			forge.WithResponseSchema(200, "Notifications retrieved", NotificationListResponse{}),
			forge.WithTags("Notification", "History"),
		)
		notifications.GET("/:id", handler.GetNotification,
			forge.WithName("notification.get"),
			forge.WithSummary("Get notification"),
			forge.WithDescription("Retrieves details of a specific sent notification by ID including delivery status"),
			forge.WithResponseSchema(200, "Notification retrieved", NotificationResponse{}),
			forge.WithResponseSchema(404, "Notification not found", NotificationErrorResponse{}),
			forge.WithTags("Notification", "History"),
		)
		notifications.POST("/:id/resend", handler.ResendNotification,
			forge.WithName("notification.resend"),
			forge.WithSummary("Resend notification"),
			forge.WithDescription("Resends a previously sent notification by ID"),
			forge.WithResponseSchema(200, "Notification resent", NotificationResponse{}),
			forge.WithResponseSchema(400, "Invalid request", NotificationErrorResponse{}),
			forge.WithResponseSchema(404, "Notification not found", NotificationErrorResponse{}),
			forge.WithTags("Notification", "Sending"),
		)
	}

	// Webhook for provider callbacks (e.g., delivery status)
	router.POST("/notifications/webhook/:provider", handler.HandleWebhook,
		forge.WithName("notification.webhook"),
		forge.WithSummary("Handle provider webhook"),
		forge.WithDescription("Receives webhook events from notification providers (SendGrid, Twilio, etc.) for delivery status updates"),
		forge.WithResponseSchema(200, "Webhook processed", NotificationWebhookResponse{}),
		forge.WithResponseSchema(400, "Invalid webhook", NotificationErrorResponse{}),
		forge.WithTags("Notification", "Webhooks"),
	)

	return nil
}

// RegisterHooks registers plugin hooks
func (p *Plugin) RegisterHooks(hookRegistry *hooks.HookRegistry) error {
	if hookRegistry == nil {
		return nil
	}

	// Register after user create hook to send welcome email
	if p.config.AutoSendWelcome {
		hookRegistry.RegisterAfterUserCreate(func(ctx context.Context, createdUser *user.User) error {
			// Send welcome email to new user
			if p.service != nil && p.templateSvc != nil && createdUser != nil && createdUser.Email != "" {
				// Use "default" organization
				orgID := "default"

				// Create adapter for sending
				adapter := NewAdapter(p.templateSvc)

				// Send welcome email
				userName := createdUser.Name
				if userName == "" {
					userName = createdUser.Email
				}

				err := adapter.SendWelcomeEmail(ctx, orgID, createdUser.Email, userName, "")
				if err != nil {
					// Log error but don't fail user creation
					fmt.Printf("Failed to send welcome email: %v\n", err)
				}
			}
			return nil
		})
	}

	return nil
}

// RegisterServiceDecorators registers the notification service
// TODO: Implement when service registry is available
func (p *Plugin) RegisterServiceDecorators(svcRegistry interface{}) error {
	// Service registry integration will be implemented when the service registry
	// infrastructure is added to AuthSome core
	return nil
}

// Migrate runs database migrations
func (p *Plugin) Migrate() error {
	if p.db == nil {
		return nil
	}

	ctx := context.Background()

	// Create notification_templates table
	_, err := p.db.NewCreateTable().
		Model((*schema.NotificationTemplate)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notification_templates table: %w", err)
	}

	// Create notifications table
	_, err = p.db.NewCreateTable().
		Model((*schema.Notification)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notifications table: %w", err)
	}

	// Create indexes
	_, err = p.db.NewCreateIndex().
		Model((*schema.NotificationTemplate)(nil)).
		Index("idx_notification_templates_org_key").
		Column("organization_id", "template_key", "type", "language").
		Unique().
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notification_templates index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.Notification)(nil)).
		Index("idx_notifications_org_status").
		Column("organization_id", "status").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notifications index: %w", err)
	}

	// Add default templates if enabled
	if p.config.AddDefaultTemplates && !p.defaultsAdded {
		if err := p.addDefaultTemplates(ctx); err != nil {
			return fmt.Errorf("failed to add default templates: %w", err)
		}
		p.defaultsAdded = true
	}

	return nil
}

// addDefaultTemplates adds default notification templates
func (p *Plugin) addDefaultTemplates(ctx context.Context) error {
	templates := DefaultTemplates()

	for _, tmpl := range templates {
		// Check if template already exists
		existing, err := p.service.CreateTemplate(ctx, &notification.CreateTemplateRequest{
			OrganizationID: "default",
			TemplateKey:    tmpl.TemplateKey,
			Name:           tmpl.TemplateKey,
			Type:           notification.NotificationType(tmpl.Type),
			Language:       "en",
			Subject:        tmpl.Subject,
			Body:           tmpl.BodyText,
			Variables:      tmpl.Variables,
			Metadata: map[string]interface{}{
				"default":     true,
				"description": tmpl.Description,
			},
		})

		if err != nil {
			// Template might already exist, continue
			continue
		}

		// Store HTML version if available
		if tmpl.BodyHTML != "" && existing != nil {
			_ = p.service.UpdateTemplate(ctx, existing.ID, &notification.UpdateTemplateRequest{
				Body: &tmpl.BodyHTML,
			})
		}
	}

	return nil
}

// GetService returns the notification service for use by other plugins
func (p *Plugin) GetService() *notification.Service {
	return p.service
}

// GetTemplateService returns the template service for use by other plugins
func (p *Plugin) GetTemplateService() *TemplateService {
	return p.templateSvc
}

// registerProviders registers email and SMS providers based on configuration
func (p *Plugin) registerProviders() error {
	// Register email provider
	emailProvider, err := p.createEmailProvider()
	if err != nil {
		return fmt.Errorf("failed to create email provider: %w", err)
	}
	if emailProvider != nil {
		if err := p.service.RegisterProvider(emailProvider); err != nil {
			return fmt.Errorf("failed to register email provider: %w", err)
		}
	}

	// Register SMS provider
	smsProvider, err := p.createSMSProvider()
	if err != nil {
		return fmt.Errorf("failed to create SMS provider: %w", err)
	}
	if smsProvider != nil {
		if err := p.service.RegisterProvider(smsProvider); err != nil {
			return fmt.Errorf("failed to register SMS provider: %w", err)
		}
	}

	return nil
}

// createEmailProvider creates an email provider based on configuration
func (p *Plugin) createEmailProvider() (notification.Provider, error) {
	// Import email providers
	emailProviders := struct {
		smtp     func() notification.Provider
		sendgrid func() notification.Provider
		mock     func() notification.Provider
	}{
		smtp: func() notification.Provider {
			return email.NewSMTPProvider(email.SMTPConfig{
				Host:     getStringConfig(p.config.Providers.Email.Config, "host", ""),
				Port:     getIntConfig(p.config.Providers.Email.Config, "port", 587),
				Username: getStringConfig(p.config.Providers.Email.Config, "username", ""),
				Password: getStringConfig(p.config.Providers.Email.Config, "password", ""),
				From:     p.config.Providers.Email.From,
				FromName: p.config.Providers.Email.FromName,
				UseTLS:   getBoolConfig(p.config.Providers.Email.Config, "use_tls", true),
			})
		},
		sendgrid: func() notification.Provider {
			return email.NewSendGridProvider(email.SendGridConfig{
				APIKey:   getStringConfig(p.config.Providers.Email.Config, "api_key", ""),
				From:     p.config.Providers.Email.From,
				FromName: p.config.Providers.Email.FromName,
			})
		},
		mock: func() notification.Provider {
			return email.NewMockEmailProvider()
		},
	}

	switch p.config.Providers.Email.Provider {
	case "smtp":
		return emailProviders.smtp(), nil
	case "sendgrid":
		return emailProviders.sendgrid(), nil
	case "mock":
		return emailProviders.mock(), nil
	case "":
		// No provider configured, use mock for development
		return emailProviders.mock(), nil
	default:
		return nil, fmt.Errorf("unknown email provider: %s", p.config.Providers.Email.Provider)
	}
}

// createSMSProvider creates an SMS provider based on configuration
func (p *Plugin) createSMSProvider() (notification.Provider, error) {
	// Import SMS providers
	smsProviders := struct {
		twilio func() notification.Provider
		mock   func() notification.Provider
	}{
		twilio: func() notification.Provider {
			return sms.NewTwilioProvider(sms.TwilioConfig{
				AccountSID: getStringConfig(p.config.Providers.SMS.Config, "account_sid", ""),
				AuthToken:  getStringConfig(p.config.Providers.SMS.Config, "auth_token", ""),
				FromNumber: p.config.Providers.SMS.From,
			})
		},
		mock: func() notification.Provider {
			return sms.NewMockSMSProvider()
		},
	}

	switch p.config.Providers.SMS.Provider {
	case "twilio":
		return smsProviders.twilio(), nil
	case "mock":
		return smsProviders.mock(), nil
	case "":
		// No provider configured, use mock for development
		return smsProviders.mock(), nil
	default:
		return nil, fmt.Errorf("unknown SMS provider: %s", p.config.Providers.SMS.Provider)
	}
}

// Helper functions to extract config values

func getStringConfig(config map[string]interface{}, key string, defaultValue string) string {
	if val, ok := config[key].(string); ok {
		return val
	}
	return defaultValue
}

func getIntConfig(config map[string]interface{}, key string, defaultValue int) int {
	if val, ok := config[key].(int); ok {
		return val
	}
	if val, ok := config[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

func getBoolConfig(config map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := config[key].(bool); ok {
		return val
	}
	return defaultValue
}

// Response types for notification routes
type NotificationErrorResponse struct {
	Error string `json:"error" example:"Error message"`
}

type NotificationStatusResponse struct {
	Status string `json:"status" example:"success"`
}

type NotificationTemplateResponse struct {
	Template interface{} `json:"template"`
}

type NotificationTemplateListResponse struct {
	Templates []interface{} `json:"templates"`
	Total     int           `json:"total" example:"10"`
}

type NotificationPreviewResponse struct {
	Subject string `json:"subject" example:"Welcome to AuthSome"`
	Body    string `json:"body" example:"Hello {{name}}, welcome to AuthSome!"`
}

type NotificationResponse struct {
	Notification interface{} `json:"notification"`
}

type NotificationListResponse struct {
	Notifications []interface{} `json:"notifications"`
	Total         int           `json:"total" example:"50"`
}

type NotificationWebhookResponse struct {
	Status string `json:"status" example:"processed"`
}
