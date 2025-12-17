package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/providers/email"
	"github.com/xraph/authsome/providers/sms"
	repo "github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// Plugin implements the notification template management plugin
type Plugin struct {
	service            *notification.Service
	templateSvc        *TemplateService
	db                 *bun.DB
	config             Config
	defaultConfig      Config
	forgeConfig        forge.ConfigManager
	defaultsAdded      bool
	dashboardExtension *DashboardExtension
	authInst           core.Authsome
	notifAdapter       *Adapter
}

// PluginOption is a functional option for configuring the notification plugin
type PluginOption func(*Plugin)

// WithDefaultConfig sets the default configuration for the plugin
func WithDefaultConfig(cfg Config) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig = cfg
	}
}

// WithAddDefaultTemplates sets whether to add default templates
func WithAddDefaultTemplates(add bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AddDefaultTemplates = add
	}
}

// WithDefaultLanguage sets the default language
func WithDefaultLanguage(lang string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.DefaultLanguage = lang
	}
}

// WithAllowOrgOverrides sets whether to allow organization overrides
func WithAllowAppOverrides(allow bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AllowAppOverrides = allow
	}
}

// WithAutoSendWelcome sets whether to auto-send welcome emails
func WithAutoSendWelcome(auto bool) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.AutoSendWelcome = auto
	}
}

// WithRetryConfig sets the retry configuration
func WithRetryConfig(attempts int, delay time.Duration) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.RetryAttempts = attempts
		p.defaultConfig.RetryDelay = delay
	}
}

// WithEmailProvider sets the email provider configuration
func WithEmailProvider(provider, from, fromName string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Providers.Email.Provider = provider
		p.defaultConfig.Providers.Email.From = from
		p.defaultConfig.Providers.Email.FromName = fromName
	}
}

// WithSMSProvider sets the SMS provider configuration
func WithSMSProvider(provider, from string) PluginOption {
	return func(p *Plugin) {
		p.defaultConfig.Providers.SMS.Provider = provider
		p.defaultConfig.Providers.SMS.From = from
	}
}

// NewPlugin creates a new notification plugin instance with optional configuration
func NewPlugin(opts ...PluginOption) *Plugin {
	p := &Plugin{
		// Set built-in defaults
		defaultConfig: DefaultConfig(),
	}

	// Apply functional options
	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "notification"
}

// Init initializes the plugin with dependencies
func (p *Plugin) Init(authInst core.Authsome) error {
	if authInst == nil {
		return fmt.Errorf("notification plugin requires auth instance")
	}

	p.authInst = authInst

	db := authInst.GetDB()
	if db == nil {
		return fmt.Errorf("database not available for notification plugin")
	}

	p.db = db

	// Get Forge app and config manager
	forgeApp := authInst.GetForgeApp()
	if forgeApp != nil {
		configManager := forgeApp.Config()

		// Bind configuration using Forge ConfigManager with provided defaults
		if err := configManager.BindWithDefault("auth.notification", &p.config, p.defaultConfig); err != nil {
			// Log but don't fail - use defaults
			fmt.Printf("[Notification] Warning: failed to bind config: %v\n", err)
			p.config = p.defaultConfig
		}
	} else {
		// Fallback to default config if no Forge app
		p.config = p.defaultConfig
	}

	fmt.Println("[Notification] Initialized", p.config.Providers.Email)

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

	// Initialize dashboard extension
	p.dashboardExtension = NewDashboardExtension(p)

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
		templates.POST("/:id/reset", handler.ResetTemplate,
			forge.WithName("notification.templates.reset"),
			forge.WithSummary("Reset template to default"),
			forge.WithDescription("Resets a notification template to its default values"),
			forge.WithResponseSchema(200, "Template reset", NotificationStatusResponse{}),
			forge.WithResponseSchema(404, "Template not found", NotificationErrorResponse{}),
			forge.WithTags("Notification", "Templates"),
		)
		templates.POST("/reset-all", handler.ResetAllTemplates,
			forge.WithName("notification.templates.reset_all"),
			forge.WithSummary("Reset all templates to defaults"),
			forge.WithDescription("Resets all notification templates for the app to their default values"),
			forge.WithResponseSchema(200, "All templates reset", NotificationStatusResponse{}),
			forge.WithTags("Notification", "Templates"),
		)
		templates.GET("/defaults", handler.GetTemplateDefaults,
			forge.WithName("notification.templates.defaults"),
			forge.WithSummary("Get default template metadata"),
			forge.WithDescription("Returns metadata for all default notification templates including variables and default content"),
			forge.WithResponseSchema(200, "Default templates retrieved", NotificationTemplateListResponse{}),
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

	// Register app creation hook to auto-populate default templates
	if p.config.AutoPopulateTemplates {
		hookRegistry.RegisterAfterAppCreate(func(ctx context.Context, app interface{}) error {
			// Type assert to get app details
			if appData, ok := app.(*schema.App); ok && !appData.IsPlatform {
				// Initialize default templates for new app
				if err := p.service.InitializeDefaultTemplates(ctx, appData.ID); err != nil {
					// Log error but don't fail app creation
					fmt.Printf("Failed to initialize default templates for app %s: %v\n", appData.ID, err)
				}
			}
			return nil
		})
	}

	// Register after user create hook to send welcome email
	// Support both legacy AutoSendWelcome and new AutoSend.Auth.Welcome config
	sendWelcome := p.config.AutoSendWelcome || p.config.AutoSend.Auth.Welcome
	if sendWelcome {
		hookRegistry.RegisterAfterUserCreate(func(ctx context.Context, createdUser *user.User) error {
			// Send welcome email to new user
			if p.service != nil && p.templateSvc != nil && createdUser != nil && createdUser.Email != "" {
				// Get platform app ID
				var platformApp schema.App
				err := p.db.NewSelect().
					Model(&platformApp).
					Where("is_platform = ?", true).
					Limit(1).
					Scan(ctx)
				if err != nil {
					// Log error but don't fail user creation
					fmt.Printf("Failed to find platform app for welcome email: %v\n", err)
					return nil
				}

				// Create adapter for sending
				adapter := NewAdapter(p.templateSvc)

				// Send welcome email
				userName := createdUser.Name
				if userName == "" {
					userName = createdUser.Email
				}

				err = adapter.SendWelcomeEmail(ctx, platformApp.ID, createdUser.Email, userName, "")
				if err != nil {
					// Log error but don't fail user creation
					fmt.Printf("Failed to send welcome email: %v\n", err)
				}
			}
			return nil
		})
	}

	// Register device/session security hooks
	if p.config.AutoSend.Session.NewDevice {
		hookRegistry.RegisterOnNewDeviceDetected(func(ctx context.Context, userID xid.ID, deviceName, location, ipAddress string) error {
			// Get app context
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				fmt.Println("[Notification] App context not available in new device detected hook")
				return nil
			}

			// Get user details
			userSvc := p.authInst.GetServiceRegistry().UserService()
			if userSvc == nil {
				fmt.Println("[Notification] User service not available")
				return nil
			}
			user, err := userSvc.FindByID(ctx, userID)
			if err != nil || user == nil {
				fmt.Printf("[Notification] Failed to get user details in new device hook: %v\n", err)
				return nil
			}

			// Send new device notification
			userName := user.Name
			if userName == "" {
				userName = user.Email
			}

			timestamp := time.Now().Format(time.RFC3339)
			adapter := NewAdapter(p.templateSvc)
			err = adapter.SendNewDeviceLogin(ctx, appID, user.Email, userName, deviceName, location, timestamp, ipAddress)
			if err != nil {
				fmt.Printf("[Notification] Failed to send new device notification: %v\n", err)
			}

			return nil
		})
	}

	if p.config.AutoSend.Session.DeviceRemoved {
		hookRegistry.RegisterOnDeviceRemoved(func(ctx context.Context, userID xid.ID, deviceName string) error {
			// Get app context
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				fmt.Println("[Notification] App context not available in device removed hook")
				return nil
			}

			// Get user details
			userSvc := p.authInst.GetServiceRegistry().UserService()
			if userSvc == nil {
				fmt.Println("[Notification] User service not available")
				return nil
			}
			user, err := userSvc.FindByID(ctx, userID)
			if err != nil || user == nil {
				fmt.Printf("[Notification] Failed to get user details in device removed hook: %v\n", err)
				return nil
			}

			// Send device removed notification
			userName := user.Name
			if userName == "" {
				userName = user.Email
			}

			timestamp := time.Now().Format(time.RFC3339)
			adapter := NewAdapter(p.templateSvc)
			err = adapter.SendDeviceRemoved(ctx, appID, user.Email, userName, deviceName, timestamp)
			if err != nil {
				fmt.Printf("[Notification] Failed to send device removed notification: %v\n", err)
			}

			return nil
		})
	}

	fmt.Println("[Notification] Registered device/session security notification hooks")

	// Register account lifecycle hooks
	if p.config.AutoSend.Account.EmailChangeRequest {
		hookRegistry.RegisterOnEmailChangeRequest(func(ctx context.Context, userID xid.ID, oldEmail, newEmail, confirmationUrl string) error {
			// Get app context
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				fmt.Println("[Notification] App context not available in email change request hook")
				return nil
			}

			// Get user details
			userSvc := p.authInst.GetServiceRegistry().UserService()
			if userSvc == nil {
				fmt.Println("[Notification] User service not available")
				return nil
			}
			user, err := userSvc.FindByID(ctx, userID)
			if err != nil || user == nil {
				fmt.Printf("[Notification] Failed to get user details in email change request hook: %v\n", err)
				return nil
			}

			// Send email change request notification to OLD email
			userName := user.Name
			if userName == "" {
				userName = user.Email
			}

			timestamp := time.Now().Format(time.RFC3339)
			adapter := NewAdapter(p.templateSvc)
			err = adapter.SendEmailChangeRequest(ctx, appID, oldEmail, userName, newEmail, confirmationUrl, timestamp)
			if err != nil {
				fmt.Printf("[Notification] Failed to send email change request notification: %v\n", err)
			}

			return nil
		})
	}

	if p.config.AutoSend.Account.EmailChanged {
		hookRegistry.RegisterOnEmailChanged(func(ctx context.Context, userID xid.ID, oldEmail, newEmail string) error {
			// Get app context
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				fmt.Println("[Notification] App context not available in email changed hook")
				return nil
			}

			// Get user details
			userSvc := p.authInst.GetServiceRegistry().UserService()
			if userSvc == nil {
				fmt.Println("[Notification] User service not available")
				return nil
			}
			user, err := userSvc.FindByID(ctx, userID)
			if err != nil || user == nil {
				fmt.Printf("[Notification] Failed to get user details in email changed hook: %v\n", err)
				return nil
			}

			// Send email changed notification to NEW email
			userName := user.Name
			if userName == "" {
				userName = user.Email
			}

			timestamp := time.Now().Format(time.RFC3339)
			adapter := NewAdapter(p.templateSvc)
			err = adapter.SendEmailChanged(ctx, appID, user.Email, userName, oldEmail, timestamp)
			if err != nil {
				fmt.Printf("[Notification] Failed to send email changed notification: %v\n", err)
			}

			return nil
		})
	}

	if p.config.AutoSend.Account.UsernameChanged {
		hookRegistry.RegisterOnUsernameChanged(func(ctx context.Context, userID xid.ID, oldUsername, newUsername string) error {
			// Get app context
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				fmt.Println("[Notification] App context not available in username changed hook")
				return nil
			}

			// Get user details
			userSvc := p.authInst.GetServiceRegistry().UserService()
			if userSvc == nil {
				fmt.Println("[Notification] User service not available")
				return nil
			}
			user, err := userSvc.FindByID(ctx, userID)
			if err != nil || user == nil {
				fmt.Printf("[Notification] Failed to get user details in username changed hook: %v\n", err)
				return nil
			}

			// Send username changed notification
			userName := user.Name
			if userName == "" {
				userName = user.Email
			}

			timestamp := time.Now().Format(time.RFC3339)
			adapter := NewAdapter(p.templateSvc)
			err = adapter.SendUsernameChanged(ctx, appID, user.Email, userName, oldUsername, newUsername, timestamp)
			if err != nil {
				fmt.Printf("[Notification] Failed to send username changed notification: %v\n", err)
			}

			return nil
		})
	}

	if p.config.AutoSend.Account.Deleted {
		hookRegistry.RegisterOnAccountDeleted(func(ctx context.Context, userID xid.ID) error {
			// Get app context
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				fmt.Println("[Notification] App context not available in account deleted hook")
				return nil
			}

			// Get user details before deletion (may already be deleted, so this might not work)
			userSvc := p.authInst.GetServiceRegistry().UserService()
			if userSvc == nil {
				fmt.Println("[Notification] User service not available")
				return nil
			}
			user, err := userSvc.FindByID(ctx, userID)
			if err != nil || user == nil {
				// User already deleted, can't send notification
				return nil
			}

			// Send account deleted notification
			userName := user.Name
			if userName == "" {
				userName = user.Email
			}

			timestamp := time.Now().Format(time.RFC3339)
			adapter := NewAdapter(p.templateSvc)
			err = adapter.SendAccountDeleted(ctx, appID, user.Email, userName, timestamp)
			if err != nil {
				fmt.Printf("[Notification] Failed to send account deleted notification: %v\n", err)
			}

			return nil
		})
	}

	if p.config.AutoSend.Account.Suspended {
		hookRegistry.RegisterOnAccountSuspended(func(ctx context.Context, userID xid.ID, reason string) error {
			// Get app context
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				fmt.Println("[Notification] App context not available in account suspended hook")
				return nil
			}

			// Get user details
			userSvc := p.authInst.GetServiceRegistry().UserService()
			if userSvc == nil {
				fmt.Println("[Notification] User service not available")
				return nil
			}
			user, err := userSvc.FindByID(ctx, userID)
			if err != nil || user == nil {
				fmt.Printf("[Notification] Failed to get user details in account suspended hook: %v\n", err)
				return nil
			}

			// Send account suspended notification
			userName := user.Name
			if userName == "" {
				userName = user.Email
			}

			timestamp := time.Now().Format(time.RFC3339)
			adapter := NewAdapter(p.templateSvc)
			err = adapter.SendAccountSuspended(ctx, appID, user.Email, userName, reason, timestamp)
			if err != nil {
				fmt.Printf("[Notification] Failed to send account suspended notification: %v\n", err)
			}

			return nil
		})
	}

	if p.config.AutoSend.Account.Reactivated {
		hookRegistry.RegisterOnAccountReactivated(func(ctx context.Context, userID xid.ID) error {
			// Get app context
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				fmt.Println("[Notification] App context not available in account reactivated hook")
				return nil
			}

			// Get user details
			userSvc := p.authInst.GetServiceRegistry().UserService()
			if userSvc == nil {
				fmt.Println("[Notification] User service not available")
				return nil
			}
			user, err := userSvc.FindByID(ctx, userID)
			if err != nil || user == nil {
				fmt.Printf("[Notification] Failed to get user details in account reactivated hook: %v\n", err)
				return nil
			}

			// Send account reactivated notification
			userName := user.Name
			if userName == "" {
				userName = user.Email
			}

			timestamp := time.Now().Format(time.RFC3339)
			adapter := NewAdapter(p.templateSvc)
			err = adapter.SendAccountReactivated(ctx, appID, user.Email, userName, timestamp)
			if err != nil {
				fmt.Printf("[Notification] Failed to send account reactivated notification: %v\n", err)
			}

			return nil
		})
	}

	if p.config.AutoSend.Account.PasswordChanged {
		hookRegistry.RegisterOnPasswordChanged(func(ctx context.Context, userID xid.ID) error {
			// Get app context
			appID, ok := contexts.GetAppID(ctx)
			if !ok || appID.IsNil() {
				fmt.Println("[Notification] App context not available in password changed hook")
				return nil
			}

			// Get user details
			userSvc := p.authInst.GetServiceRegistry().UserService()
			if userSvc == nil {
				fmt.Println("[Notification] User service not available")
				return nil
			}
			user, err := userSvc.FindByID(ctx, userID)
			if err != nil || user == nil {
				fmt.Printf("[Notification] Failed to get user details in password changed hook: %v\n", err)
				return nil
			}

			// Send password changed notification
			userName := user.Name
			if userName == "" {
				userName = user.Email
			}

			timestamp := time.Now().Format(time.RFC3339)
			adapter := NewAdapter(p.templateSvc)
			err = adapter.SendPasswordChanged(ctx, appID, user.Email, userName, timestamp)
			if err != nil {
				fmt.Printf("[Notification] Failed to send password changed notification: %v\n", err)
			}

			return nil
		})
	}

	fmt.Println("[Notification] Registered account lifecycle notification hooks")

	return nil
}

// RegisterServiceDecorators registers the notification service and adapter
func (p *Plugin) RegisterServiceDecorators(services *registry.ServiceRegistry) error {
	if services == nil {
		return nil
	}

	// Initialize adapter with app service for dynamic app name lookup
	if p.notifAdapter == nil && p.templateSvc != nil {
		p.notifAdapter = NewAdapter(p.templateSvc).
			WithAppService(p.authInst.GetServiceRegistry().AppService()).
			WithAppName(p.config.AppName)
	}

	// Register notification adapter for other plugins to use
	if p.notifAdapter != nil {
		if err := services.Register("notification.adapter", p.notifAdapter); err != nil {
			fmt.Printf("[Notification] Warning: failed to register notification adapter in service registry: %v\n", err)
			// Don't fail - adapter will still work for this plugin's own use
		} else {
			fmt.Println("[Notification] Registered notification adapter in service registry")
		}
	}

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

	// Create notification_providers table
	_, err = p.db.NewCreateTable().
		Model((*schema.NotificationProvider)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notification_providers table: %w", err)
	}

	// Create notification_template_versions table
	_, err = p.db.NewCreateTable().
		Model((*schema.NotificationTemplateVersion)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notification_template_versions table: %w", err)
	}

	// Create notification_analytics table
	_, err = p.db.NewCreateTable().
		Model((*schema.NotificationAnalytics)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notification_analytics table: %w", err)
	}

	// Create notification_tests table
	_, err = p.db.NewCreateTable().
		Model((*schema.NotificationTest)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notification_tests table: %w", err)
	}

	// Create indexes
	_, err = p.db.NewCreateIndex().
		Model((*schema.NotificationTemplate)(nil)).
		Index("idx_notification_templates_app_org_key").
		Column("app_id", "organization_id", "template_key", "type", "language").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notification_templates index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.NotificationTemplate)(nil)).
		Index("idx_notification_templates_ab_test").
		Column("ab_test_group", "ab_test_enabled").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create ab_test index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.Notification)(nil)).
		Index("idx_notifications_app_status").
		Column("app_id", "status").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create notifications index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.NotificationProvider)(nil)).
		Index("idx_notification_providers_app_org_type").
		Column("app_id", "organization_id", "provider_type", "is_default").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create providers index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.NotificationAnalytics)(nil)).
		Index("idx_notification_analytics_notification").
		Column("notification_id").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create analytics notification index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.NotificationAnalytics)(nil)).
		Index("idx_notification_analytics_template").
		Column("template_id", "event", "created_at").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create analytics template index: %w", err)
	}

	_, err = p.db.NewCreateIndex().
		Model((*schema.NotificationTest)(nil)).
		Index("idx_notification_tests_template").
		Column("template_id", "created_at").
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create tests index: %w", err)
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
	// Get platform app ID
	var platformApp schema.App
	err := p.db.NewSelect().
		Model(&platformApp).
		Where("is_platform = ?", true).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to find platform app: %w", err)
	}

	// Use the new InitializeDefaultTemplates method that uses template key constants
	if err := p.service.InitializeDefaultTemplates(ctx, platformApp.ID); err != nil {
		return fmt.Errorf("failed to initialize default templates: %w", err)
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

// DashboardExtension returns the dashboard extension interface implementation
func (p *Plugin) DashboardExtension() ui.DashboardExtension {
	return p.dashboardExtension
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
		smtp       func() notification.Provider
		sendgrid   func() notification.Provider
		resend     func() notification.Provider
		mailersend func() notification.Provider
		postmark   func() notification.Provider
		mock       func() notification.Provider
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
		resend: func() notification.Provider {
			return NewResendProvider(ResendConfig{
				APIKey:   getStringConfig(p.config.Providers.Email.Config, "api_key", ""),
				From:     p.config.Providers.Email.From,
				FromName: p.config.Providers.Email.FromName,
				ReplyTo:  p.config.Providers.Email.ReplyTo,
			})
		},
		mailersend: func() notification.Provider {
			return NewMailerSendProvider(MailerSendConfig{
				APIKey:   getStringConfig(p.config.Providers.Email.Config, "api_key", ""),
				From:     p.config.Providers.Email.From,
				FromName: p.config.Providers.Email.FromName,
				ReplyTo:  p.config.Providers.Email.ReplyTo,
			})
		},
		postmark: func() notification.Provider {
			return NewPostmarkProvider(PostmarkConfig{
				ServerToken: getStringConfig(p.config.Providers.Email.Config, "server_token", ""),
				From:        p.config.Providers.Email.From,
				FromName:    p.config.Providers.Email.FromName,
				ReplyTo:     p.config.Providers.Email.ReplyTo,
				TrackOpens:  getBoolConfig(p.config.Providers.Email.Config, "track_opens", false),
				TrackLinks:  getStringConfig(p.config.Providers.Email.Config, "track_links", "None"),
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
	case "resend":
		return emailProviders.resend(), nil
	case "mailersend":
		return emailProviders.mailersend(), nil
	case "postmark":
		return emailProviders.postmark(), nil
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
	// SMS provider is optional - return nil if not configured
	if p.config.Providers.SMS == nil {
		return nil, nil
	}

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
		// No provider specified, return nil (SMS is optional)
		return nil, nil
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
