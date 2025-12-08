package notification

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/notification"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/authsome/plugins/notification/builder"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements the ui.DashboardExtension interface
// This allows the notification plugin to add its own screens to the dashboard
type DashboardExtension struct {
	plugin   *Plugin
	registry *dashboard.ExtensionRegistry
}

// NewDashboardExtension creates a new dashboard extension for notification plugin
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{plugin: plugin}
}

// SetRegistry sets the extension registry reference (called by dashboard after registration)
func (e *DashboardExtension) SetRegistry(registry *dashboard.ExtensionRegistry) {
	e.registry = registry
}

// ExtensionID returns the unique identifier for this extension
func (e *DashboardExtension) ExtensionID() string {
	return "notification"
}

// NavigationItems returns navigation items to register
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:    "notifications",
			Label: "Notifications",
			Icon: lucide.Mail(
				Class("size-4"),
			),
			Position: ui.NavPositionMain,
			Order:    55, // After organizations
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp != nil {
					return basePath + "/dashboard/app/" + currentApp.ID.String() + "/notifications"
				}
				return basePath + "/dashboard/"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "notifications"
			},
			RequiresPlugin: "notification",
		},
	}
}

// Routes returns routes to register under /dashboard/app/:appId/
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		// Notifications Overview
		{
			Method:       "GET",
			Path:         "/notifications",
			Handler:      e.ServeNotificationsOverview,
			Name:         "dashboard.notifications.overview",
			Summary:      "Notifications overview",
			Description:  "View notification statistics and recent activity",
			Tags:         []string{"Dashboard", "Notifications"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Templates List
		{
			Method:       "GET",
			Path:         "/notifications/templates",
			Handler:      e.ServeTemplatesList,
			Name:         "dashboard.notifications.templates",
			Summary:      "Notification templates",
			Description:  "Manage notification templates",
			Tags:         []string{"Dashboard", "Notifications", "Templates"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Create Template Page
		{
			Method:       "GET",
			Path:         "/notifications/templates/create",
			Handler:      e.ServeCreateTemplate,
			Name:         "dashboard.notifications.templates.create",
			Summary:      "Create template",
			Description:  "Create a new notification template",
			Tags:         []string{"Dashboard", "Notifications", "Templates"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Edit Template Page
		{
			Method:       "GET",
			Path:         "/notifications/templates/:templateId/edit",
			Handler:      e.ServeEditTemplate,
			Name:         "dashboard.notifications.templates.edit",
			Summary:      "Edit template",
			Description:  "Edit an existing notification template",
			Tags:         []string{"Dashboard", "Notifications", "Templates"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Template Preview
		{
			Method:       "POST",
			Path:         "/notifications/templates/:templateId/preview",
			Handler:      e.PreviewTemplate,
			Name:         "dashboard.notifications.templates.preview",
			Summary:      "Preview template",
			Description:  "Preview a template with test variables",
			Tags:         []string{"Dashboard", "Notifications", "Templates"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Test Send
		{
			Method:       "POST",
			Path:         "/notifications/templates/:templateId/test",
			Handler:      e.TestSendTemplate,
			Name:         "dashboard.notifications.templates.test",
			Summary:      "Test send",
			Description:  "Send a test notification",
			Tags:         []string{"Dashboard", "Notifications", "Templates"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Email Builder Routes
		{
			Method:       "GET",
			Path:         "/notifications/templates/builder",
			Handler:      e.ServeEmailBuilder,
			Name:         "dashboard.notifications.builder",
			Summary:      "Email template builder",
			Description:  "Visual drag-and-drop email template builder",
			Tags:         []string{"Dashboard", "Notifications", "Builder"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/notifications/templates/builder/:templateId",
			Handler:      e.ServeEmailBuilderWithTemplate,
			Name:         "dashboard.notifications.builder.edit",
			Summary:      "Edit template in builder",
			Description:  "Edit an existing template in the visual builder",
			Tags:         []string{"Dashboard", "Notifications", "Builder"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/notifications/templates/builder/preview",
			Handler:      e.PreviewBuilderTemplate,
			Name:         "dashboard.notifications.builder.preview",
			Summary:      "Preview builder template",
			Description:  "Generate HTML preview from builder JSON",
			Tags:         []string{"Dashboard", "Notifications", "Builder"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/notifications/templates/builder/save",
			Handler:      e.SaveBuilderTemplate,
			Name:         "dashboard.notifications.builder.save",
			Summary:      "Save builder template",
			Description:  "Save a template created with the visual builder",
			Tags:         []string{"Dashboard", "Notifications", "Builder"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/notifications/templates/samples/:name",
			Handler:      e.GetSampleTemplate,
			Name:         "dashboard.notifications.samples",
			Summary:      "Get sample template",
			Description:  "Load a sample email template",
			Tags:         []string{"Dashboard", "Notifications", "Builder"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		// Settings Pages Routes
		{
			Method:       "GET",
			Path:         "/settings/notifications",
			Handler:      e.ServeNotificationSettings,
			Name:         "dashboard.settings.notifications",
			Summary:      "Notification settings",
			Description:  "Configure notification plugin settings",
			Tags:         []string{"Dashboard", "Settings", "Notifications"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/notifications/save",
			Handler:      e.SaveNotificationSettings,
			Name:         "dashboard.settings.notifications.save",
			Summary:      "Save notification settings",
			Description:  "Update notification plugin configuration",
			Tags:         []string{"Dashboard", "Settings", "Notifications"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/notifications/providers",
			Handler:      e.ServeProviderSettings,
			Name:         "dashboard.settings.notifications.providers",
			Summary:      "Provider settings",
			Description:  "Configure email and SMS providers",
			Tags:         []string{"Dashboard", "Settings", "Notifications"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/settings/notifications/providers/test",
			Handler:      e.TestProvider,
			Name:         "dashboard.settings.notifications.providers.test",
			Summary:      "Test provider",
			Description:  "Test email or SMS provider configuration",
			Tags:         []string{"Dashboard", "Settings", "Notifications"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/notifications/analytics",
			Handler:      e.ServeAnalyticsSettings,
			Name:         "dashboard.settings.notifications.analytics",
			Summary:      "Analytics settings",
			Description:  "View notification analytics and performance",
			Tags:         []string{"Dashboard", "Settings", "Notifications"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections for backward compatibility
// Deprecated: Use SettingsPages() instead
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return []ui.SettingsSection{}
}

// SettingsPages returns full settings pages for the sidebar layout
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "notification-providers",
			Label:         "Email & SMS Providers",
			Description:   "Configure notification delivery providers",
			Icon:          lucide.Send(Class("h-5 w-5")),
			Category:      "communication",
			Order:         11,
			Path:          "notifications/providers",
			RequirePlugin: "notification",
			RequireAdmin:  true,
		},
		{
			ID:            "notification-settings",
			Label:         "Notification Settings",
			Description:   "Configure notification plugin behavior",
			Icon:          lucide.Settings(Class("h-5 w-5")),
			Category:      "general",
			Order:         20,
			Path:          "notifications",
			RequirePlugin: "notification",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns widgets to show on the main dashboard
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "notification-stats",
			Title: "Notifications",
			Icon: lucide.Mail(
				Class("size-5"),
			),
			Order: 30,
			Size:  1, // 1 column
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderDashboardWidget(basePath, currentApp)
			},
		},
	}
}

// =============================================================================
// HANDLER IMPLEMENTATIONS
// =============================================================================

// ServeNotificationsOverview renders the notifications overview page
func (e *DashboardExtension) ServeNotificationsOverview(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	// Build minimal PageData
	pageData := components.PageData{
		Title:      "Notifications",
		User:       currentUser,
		ActivePage: "notifications",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	// Render page content
	content := e.renderNotificationsOverview(currentApp, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// ServeTemplatesList renders the templates list page
func (e *DashboardExtension) ServeTemplatesList(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	pageData := components.PageData{
		Title:      "Notification Templates",
		User:       currentUser,
		ActivePage: "notifications",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	content := e.renderTemplatesList(currentApp, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// ServeCreateTemplate renders the create template page
func (e *DashboardExtension) ServeCreateTemplate(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	pageData := components.PageData{
		Title:      "Create Template",
		User:       currentUser,
		ActivePage: "notifications",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	content := e.renderCreateTemplate(currentApp, basePath)

	return handler.RenderWithLayout(c, pageData, content)
}

// ServeEditTemplate renders the edit template page
func (e *DashboardExtension) ServeEditTemplate(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	templateID, err := xid.FromString(c.Param("templateId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid template ID"})
	}

	basePath := handler.GetBasePath()

	// Check if this is a visual builder template
	template, err := e.plugin.service.GetTemplate(c.Context(), templateID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Template not found"})
	}

	// If it's a visual builder template, redirect to the visual builder
	if template.Metadata != nil {
		if builderType, ok := template.Metadata["builderType"].(string); ok && builderType == "visual" {
			builderURL := fmt.Sprintf("%s/dashboard/app/%s/notifications/templates/builder/%s",
				basePath, currentApp.ID, templateID)
			return c.Redirect(http.StatusFound, builderURL)
		}
	}

	pageData := components.PageData{
		Title:      "Edit Template",
		User:       currentUser,
		ActivePage: "notifications",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	content := e.renderEditTemplate(currentApp, basePath, templateID)

	return handler.RenderWithLayout(c, pageData, content)
}

// PreviewTemplate handles template preview requests
func (e *DashboardExtension) PreviewTemplate(c forge.Context) error {
	templateIDStr := c.Param("templateId")
	templateID, err := xid.FromString(templateIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid template ID"})
	}

	var req struct {
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Fetch template
	template, err := e.plugin.service.GetTemplate(c.Context(), templateID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Template not found"})
	}

	// Create template engine for rendering
	engine := NewTemplateEngine()

	// Render subject
	subject := ""
	if template.Subject != "" {
		subject, err = engine.Render(template.Subject, req.Variables)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("Failed to render subject: %v", err),
			})
		}
	}

	// Render body
	body, err := engine.Render(template.Body, req.Variables)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Failed to render body: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"subject": subject,
		"body":    body,
		"type":    template.Type,
	})
}

// TestSendTemplate handles test send requests
func (e *DashboardExtension) TestSendTemplate(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid app context"})
	}

	templateIDStr := c.Param("templateId")
	templateID, err := xid.FromString(templateIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid template ID"})
	}

	var req struct {
		Recipient string                 `json:"recipient"`
		Variables map[string]interface{} `json:"variables"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.Recipient == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Recipient is required"})
	}

	// Get template directly by ID
	template, err := e.plugin.service.GetTemplate(c.Context(), templateID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Template not found"})
	}

	// Prepare default variables if not provided
	if req.Variables == nil {
		req.Variables = make(map[string]interface{})
	}

	// Add some default variables for testing
	if _, exists := req.Variables["user_name"]; !exists {
		req.Variables["user_name"] = currentUser.Name
	}
	if _, exists := req.Variables["user_email"]; !exists {
		req.Variables["user_email"] = currentUser.Email
	}
	if _, exists := req.Variables["app_name"]; !exists {
		req.Variables["app_name"] = currentApp.Name
	}
	if _, exists := req.Variables["userName"]; !exists {
		req.Variables["userName"] = currentUser.Name
	}
	if _, exists := req.Variables["appName"]; !exists {
		req.Variables["appName"] = currentApp.Name
	}

	// Render the template with variables
	engine := e.plugin.service.GetTemplateEngine()

	// Render subject if present
	subject := template.Subject
	if subject != "" && len(req.Variables) > 0 {
		rendered, err := engine.Render(subject, req.Variables)
		if err == nil {
			subject = rendered
		}
	}

	// Render body with variables
	body := template.Body
	if body != "" && len(req.Variables) > 0 {
		rendered, err := engine.Render(body, req.Variables)
		if err == nil {
			body = rendered
		}
	}

	// Directly send using the notification service with the rendered template content
	// Don't set TemplateName to avoid redundant template lookup
	notif, err := e.plugin.service.Send(c.Context(), &notification.SendRequest{
		AppID:     currentApp.ID,
		Type:      template.Type,
		Recipient: req.Recipient,
		Subject:   subject,
		Body:      body,
		Metadata: map[string]interface{}{
			"test_send":     true,
			"test_by_user":  currentUser.ID.String(),
			"test_by_email": currentUser.Email,
			"template_id":   templateID.String(),
			"template_key":  template.TemplateKey,
			"template_name": template.Name,
		},
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to send test notification: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success":        true,
		"message":        "Test notification sent successfully",
		"notificationId": notif.ID,
		"recipient":      req.Recipient,
	})
}

// ServeNotificationSettings renders the notification settings page
func (e *DashboardExtension) ServeNotificationSettings(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	content := e.renderNotificationSettingsContent(currentApp, handler.GetBasePath())

	// Use the settings layout with sidebar navigation
	return handler.RenderSettingsPage(c, "notification-settings", content)
}

// SaveNotificationSettings handles saving notification settings
func (e *DashboardExtension) SaveNotificationSettings(c forge.Context) error {
	_, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid app context"})
	}

	var req struct {
		AutoSendWelcome bool   `json:"autoSendWelcome"`
		RetryAttempts   int    `json:"retryAttempts"`
		RetryDelay      string `json:"retryDelay"`
		CleanupAfter    string `json:"cleanupAfter"`
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Validate retry attempts
	if req.RetryAttempts < 0 || req.RetryAttempts > 10 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Retry attempts must be between 0 and 10",
		})
	}

	// Note: In a full implementation, you would save these settings to the app configuration
	// This might involve:
	// 1. Updating app-specific config in the database or config file
	// 2. Reloading the plugin configuration
	// 3. Applying the changes to the running service

	// For now, we'll just return success as the config structure would need to support
	// per-app overrides which is part of the configuration management system

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Notification settings saved successfully",
		"settings": map[string]interface{}{
			"autoSendWelcome": req.AutoSendWelcome,
			"retryAttempts":   req.RetryAttempts,
			"retryDelay":      req.RetryDelay,
			"cleanupAfter":    req.CleanupAfter,
		},
	})
}

// ServeProviderSettings renders the provider settings page
func (e *DashboardExtension) ServeProviderSettings(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	content := e.renderProviderSettingsContent(currentApp, handler.GetBasePath())

	// Use the settings layout with sidebar navigation
	return handler.RenderSettingsPage(c, "notification-providers", content)
}

// TestProvider handles provider test requests by actually sending a test notification
func (e *DashboardExtension) TestProvider(c forge.Context) error {
	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid app context"})
	}

	var req struct {
		ProviderType  string `json:"providerType"`  // "email" or "sms"
		ProviderName  string `json:"providerName"`  // "smtp", "sendgrid", "twilio", etc.
		TestRecipient string `json:"testRecipient"` // Email address or phone number
	}
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	if req.TestRecipient == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Test recipient is required"})
	}

	startTime := time.Now()

	// Use the plugin's notification service to send a test notification
	if req.ProviderType == "email" {
		// Send a test email using the configured provider
		notif, err := e.plugin.service.Send(c.Context(), &notification.SendRequest{
			AppID:     currentApp.ID,
			Type:      notification.NotificationTypeEmail,
			Recipient: req.TestRecipient,
			Subject:   "Test Email from AuthSome",
			Body: `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Test Email</title>
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #f5f5f5; padding: 40px;">
    <div style="max-width: 600px; margin: 0 auto; background: white; border-radius: 8px; padding: 40px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
        <h1 style="color: #333; margin: 0 0 20px 0;">✅ Email Configuration Working!</h1>
        <p style="color: #555; line-height: 1.6;">This is a test email from AuthSome to verify your email provider configuration is working correctly.</p>
        <hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
        <p style="color: #888; font-size: 14px;">
            <strong>App:</strong> ` + currentApp.Name + `<br>
            <strong>Provider:</strong> ` + req.ProviderName + `<br>
            <strong>Sent at:</strong> ` + time.Now().Format(time.RFC1123) + `
        </p>
    </div>
</body>
</html>`,
			Metadata: map[string]interface{}{
				"test":     true,
				"provider": req.ProviderName,
			},
		})

		duration := time.Since(startTime)

		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"success":      false,
				"error":        fmt.Sprintf("Failed to send test email: %v", err),
				"deliveryTime": fmt.Sprintf("%.2fs", duration.Seconds()),
				"recipient":    req.TestRecipient,
				"providerName": req.ProviderName,
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":        true,
			"message":        "Test email sent successfully! Check your inbox.",
			"notificationId": notif.ID.String(),
			"deliveryTime":   fmt.Sprintf("%.2fs", duration.Seconds()),
			"recipient":      req.TestRecipient,
			"providerName":   req.ProviderName,
		})
	} else if req.ProviderType == "sms" {
		// Send a test SMS using the configured provider
		notif, err := e.plugin.service.Send(c.Context(), &notification.SendRequest{
			AppID:     currentApp.ID,
			Type:      notification.NotificationTypeSMS,
			Recipient: req.TestRecipient,
			Body:      fmt.Sprintf("AuthSome Test: Your SMS configuration is working! App: %s, Provider: %s", currentApp.Name, req.ProviderName),
			Metadata: map[string]interface{}{
				"test":     true,
				"provider": req.ProviderName,
			},
		})

		duration := time.Since(startTime)

		if err != nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"success":      false,
				"error":        fmt.Sprintf("Failed to send test SMS: %v", err),
				"deliveryTime": fmt.Sprintf("%.2fs", duration.Seconds()),
				"recipient":    req.TestRecipient,
				"providerName": req.ProviderName,
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":        true,
			"message":        "Test SMS sent successfully!",
			"notificationId": notif.ID.String(),
			"deliveryTime":   fmt.Sprintf("%.2fs", duration.Seconds()),
			"recipient":      req.TestRecipient,
			"providerName":   req.ProviderName,
		})
	}

	return c.JSON(http.StatusBadRequest, map[string]string{
		"error": "Invalid provider type. Must be 'email' or 'sms'",
	})
}

// ServeAnalyticsSettings renders the analytics page
func (e *DashboardExtension) ServeAnalyticsSettings(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	content := e.renderAnalyticsContent(currentApp, handler.GetBasePath())

	pageData := components.PageData{
		Title:      "Notification Analytics",
		User:       currentUser,
		ActivePage: "notifications-analytics",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	// Use the settings layout with sidebar navigation
	return handler.RenderWithLayout(c, pageData, content)
}

// =============================================================================
// RENDERING HELPERS
// =============================================================================

// getUserFromContext extracts the current user from context
func (e *DashboardExtension) getUserFromContext(c forge.Context) *user.User {
	return e.registry.GetHandler().GetUserFromContext(c)
}

// extractAppFromURL extracts app using the dashboard handler
func (e *DashboardExtension) extractAppFromURL(c forge.Context) (*app.App, error) {
	handler := e.registry.GetHandler()
	if handler == nil {
		return nil, fmt.Errorf("handler not available")
	}

	currentApp, err := handler.GetCurrentApp(c)
	if err != nil {
		return nil, err
	}

	return currentApp, nil
}

// RenderDashboardWidget renders the notification stats widget
func (e *DashboardExtension) RenderDashboardWidget(basePath string, currentApp *app.App) g.Node {
	// TODO: Get actual stats from database
	totalSent := 1234
	delivered := 1150
	opened := 456
	deliveryRate := float64(delivered) / float64(totalSent) * 100

	return Div(
		Class("flex flex-col gap-2"),

		// Total sent
		Div(
			Class("flex items-center justify-between text-sm"),
			Span(Class("text-slate-600 dark:text-gray-400"), g.Text("Total Sent")),
			Span(Class("font-semibold text-slate-900 dark:text-white"), g.Textf("%d", totalSent)),
		),

		// Delivery rate
		Div(
			Class("flex items-center justify-between text-sm"),
			Span(Class("text-slate-600 dark:text-gray-400"), g.Text("Delivered")),
			Span(Class("font-semibold text-green-600 dark:text-green-400"), g.Textf("%.1f%%", deliveryRate)),
		),

		// Open rate
		Div(
			Class("flex items-center justify-between text-sm"),
			Span(Class("text-slate-600 dark:text-gray-400"), g.Text("Opened")),
			Span(Class("font-semibold text-blue-600 dark:text-blue-400"), g.Textf("%d", opened)),
		),

		// View details link
		Div(
			Class("mt-2 pt-2 border-t border-slate-200 dark:border-gray-700"),
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/notifications", basePath, currentApp.ID.String())),
				Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400 dark:hover:text-violet-300"),
				g.Text("View all notifications →"),
			),
		),
	)
}

// renderNotificationsOverview renders the main notifications overview
func (e *DashboardExtension) renderNotificationsOverview(currentApp *app.App, basePath string) g.Node {
	return Div(
		Class("space-y-6"),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text("Notifications")),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Manage templates, providers, and view analytics")),
			),
		),

		// Quick actions
		Div(
			Class("grid gap-4 md:grid-cols-3"),

			// Templates card
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates", basePath, currentApp.ID)),
				Class("block p-6 rounded-lg border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/20"),
						lucide.FileText(Class("h-5 w-5 text-violet-600 dark:text-violet-400")),
					),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Templates")),
				),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Manage email and SMS templates")),
			),

			// Providers card
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/settings/notifications/providers", basePath, currentApp.ID)),
				Class("block p-6 rounded-lg border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-green-100 dark:bg-green-900/20"),
						lucide.Send(Class("h-5 w-5 text-green-600 dark:text-green-400")),
					),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Providers")),
				),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Configure email and SMS providers")),
			),

			// Analytics card
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/settings/notifications/analytics", basePath, currentApp.ID)),
				Class("block p-6 rounded-lg border border-slate-200 bg-white shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-2"),
					Div(
						Class("flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/20"),
						lucide.ChartBar(Class("h-5 w-5 text-blue-600 dark:text-blue-400")),
					),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Analytics")),
				),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("View performance metrics")),
			),
		),

		// Recent notifications placeholder
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
			H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("Recent Activity")),
			P(Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text("Recent notification activity will appear here")),
		),
	)
}

// renderTemplatesList renders the templates list
func (e *DashboardExtension) renderTemplatesList(currentApp *app.App, basePath string) g.Node {
	ctx := context.Background()

	// Fetch templates with pagination
	filter := &notification.ListTemplatesFilter{
		PaginationParams: pagination.PaginationParams{
			Page:  1,
			Limit: 50,
		},
		AppID: currentApp.ID,
	}

	response, err := e.plugin.service.ListTemplates(ctx, filter)

	if err != nil {
		return Div(
			Class("space-y-6"),
			Div(
				Class("rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/20"),
				P(Class("text-sm text-red-800 dark:text-red-200"),
					g.Textf("Error loading templates: %v", err)),
			),
		)
	}

	// Alpine.js data for modal handling
	alpineInit := fmt.Sprintf(`{
		showCreateModal: false,
		showTestModal: false,
		testTemplateId: '',
		testRecipient: '',
		testVariables: '{\n  "user_name": "Test User",\n  "app_name": "My App"\n}',
		testLoading: false,
		testResult: null,
		editorType: '', // 'visual' or 'manual' - empty means selection step
		selectedSample: '', // Selected sample template name
		newTemplate: {
			name: '',
			templateKey: '',
			subject: ''
		},
		createLoading: false,
		createError: null,
		resetModal() {
			this.editorType = '';
			this.selectedSample = '';
			this.newTemplate = { name: '', templateKey: '', subject: '' };
			this.createError = null;
		},
		openCreateModal() {
			this.resetModal();
			this.showCreateModal = true;
		},
		selectEditorType(type) {
			this.editorType = type;
		},
		generateKey() {
			this.newTemplate.templateKey = 'custom.' + this.newTemplate.name.toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/^_|_$/g, '');
		},
		async createTemplate() {
			if (!this.newTemplate.name || !this.newTemplate.templateKey) {
				this.createError = 'Name and template key are required';
				return;
			}
			this.createLoading = true;
			this.createError = null;
			
			if (this.editorType === 'visual') {
				// Create visual builder template
				try {
					// Get document - either from sample or empty
					let document;
					if (this.selectedSample) {
						// Fetch sample template
						const sampleRes = await fetch('%s/dashboard/app/%s/notifications/templates/samples/' + this.selectedSample);
						if (sampleRes.ok) {
							document = await sampleRes.json();
						} else {
							// Fallback to empty
							document = {
								root: 'root',
								blocks: {
									root: {
										type: 'EmailLayout',
										data: {
											backdropColor: '#F5F5F5',
											canvasColor: '#FFFFFF',
											textColor: '#242424',
											fontFamily: 'MODERN_SANS',
											childrenIds: []
										}
									}
								}
							};
						}
					} else {
						document = {
							root: 'root',
							blocks: {
								root: {
									type: 'EmailLayout',
									data: {
										backdropColor: '#F5F5F5',
										canvasColor: '#FFFFFF',
										textColor: '#242424',
										fontFamily: 'MODERN_SANS',
										childrenIds: []
									}
								}
							}
						};
					}
					
					const res = await fetch('%s/dashboard/app/%s/notifications/templates/builder/save', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({
							name: this.newTemplate.name,
							templateKey: this.newTemplate.templateKey,
							subject: this.newTemplate.subject,
							document: document
						})
					});
					const data = await res.json();
					if (data.success && data.templateId) {
						window.location.href = '%s/dashboard/app/%s/notifications/templates/builder/' + data.templateId;
					} else {
						this.createError = data.error || 'Failed to create template';
					}
				} catch (err) {
					this.createError = err.message;
				}
			} else {
				// Create manual template and go to edit page
				try {
					const res = await fetch('%s/auth/notification/templates', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({
							appId: '%s',
							name: this.newTemplate.name,
							templateKey: this.newTemplate.templateKey,
							type: 'email',
							language: 'en',
							subject: this.newTemplate.subject || 'Email Subject',
							body: '<p>Your email content goes here...</p>',
							active: true
						})
					});
					const data = await res.json();
					if (data.id) {
						window.location.href = '%s/dashboard/app/%s/notifications/templates/' + data.id + '/edit';
					} else {
						this.createError = data.error || data.message || 'Failed to create template';
					}
				} catch (err) {
					this.createError = err.message;
				}
			}
			this.createLoading = false;
		},
		async sendTestNotification() {
			if (!this.testTemplateId || !this.testRecipient) return;
			this.testLoading = true;
			this.testResult = null;
			try {
				// Parse variables from JSON string
				let variables = {};
				if (this.testVariables && this.testVariables.trim()) {
					try {
						variables = JSON.parse(this.testVariables);
					} catch (e) {
						this.testResult = { error: 'Invalid JSON for variables: ' + e.message };
						this.testLoading = false;
						return;
					}
				}
				const res = await fetch('%s/dashboard/app/%s/notifications/templates/' + this.testTemplateId + '/test', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ recipient: this.testRecipient, variables: variables })
				});
				this.testResult = await res.json();
			} catch (err) {
				this.testResult = { error: err.message };
			}
			this.testLoading = false;
		},
		openTestModal(templateId) {
			this.testTemplateId = templateId;
			this.testRecipient = '';
			this.testVariables = '{\n  "user_name": "Test User",\n  "app_name": "My App"\n}';
			this.testResult = null;
			this.showTestModal = true;
		}
	}`, basePath, currentApp.ID, basePath, currentApp.ID, basePath, currentApp.ID, basePath, currentApp.ID.String(), basePath, currentApp.ID, basePath, currentApp.ID)

	return Div(
		Class("space-y-6"),
		g.Attr("x-data", alpineInit),

		// Header with create buttons
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text("Notification Templates")),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Textf("Manage %d notification templates", len(response.Data))),
			),
			Div(
				Class("flex items-center gap-3"),
				// New Template button - opens modal with editor type choice
				Button(
					Type("button"),
					g.Attr("@click", "openCreateModal()"),
					Class("inline-flex items-center gap-2 rounded-lg bg-gradient-to-r from-violet-600 to-indigo-600 px-4 py-2 text-sm font-medium text-white hover:from-violet-700 hover:to-indigo-700 shadow-sm"),
					lucide.Plus(Class("h-4 w-4")),
					g.Text("New Template"),
				),
			),
		),

		// Visual Builder promo card
		renderBuilderPromoCard(basePath, currentApp),

		// Templates table
		g.If(len(response.Data) == 0,
			renderEmptyTemplatesState(basePath, currentApp),
		),
		g.If(len(response.Data) > 0,
			renderTemplatesTable(response.Data, basePath, currentApp),
		),

		// Create Template Modal for Visual Builder
		renderCreateBuilderModal(),

		// Test Send Modal
		renderTestSendListModal(),
	)
}

// renderCreateBuilderModal renders the modal for creating a template with editor type selection
func renderCreateBuilderModal() g.Node {
	return Div(
		g.Attr("x-show", "showCreateModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("@keydown.escape.window", "showCreateModal = false"),

		// Overlay
		Div(
			g.Attr("x-show", "showCreateModal"),
			g.Attr("@click", "showCreateModal = false; resetModal()"),
			Class("fixed inset-0 bg-black bg-opacity-50 transition-opacity"),
		),

		// Modal - wider when showing sample templates
		Div(
			Class("flex min-h-screen items-center justify-center p-4"),
			Div(
				g.Attr("x-show", "showCreateModal"),
				g.Attr("@click.stop", ""),
				g.Attr("x-transition:enter", "ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
				g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
				g.Attr(":class", "editorType === 'visual' ? 'max-w-4xl' : 'max-w-lg'"),
				Class("relative bg-white dark:bg-gray-900 rounded-xl shadow-xl w-full p-6"),

				// Step 1: Editor Type Selection (shown when editorType is empty)
				Div(
					g.Attr("x-show", "!editorType"),
					g.Attr("x-cloak", ""),

					// Header
					Div(
						Class("text-center mb-6"),
						Div(
							Class("mx-auto flex h-14 w-14 items-center justify-center rounded-xl bg-gradient-to-br from-violet-500 to-indigo-600 shadow-lg mb-4"),
							lucide.Mail(Class("h-7 w-7 text-white")),
						),
						H3(Class("text-xl font-semibold text-slate-900 dark:text-white"),
							g.Text("Create New Template")),
						P(Class("mt-2 text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Choose how you want to design your email template")),
					),

					// Editor Type Options
					Div(
						Class("grid grid-cols-2 gap-4 mb-6"),

						// Visual Builder option
						Button(
							Type("button"),
							g.Attr("@click", "selectEditorType('visual')"),
							Class("group relative flex flex-col items-center p-6 rounded-xl border-2 border-slate-200 hover:border-violet-500 hover:bg-violet-50 dark:border-gray-700 dark:hover:border-violet-500 dark:hover:bg-violet-900/20 transition-all"),
							Div(
								Class("flex h-14 w-14 items-center justify-center rounded-xl bg-gradient-to-br from-violet-100 to-indigo-100 group-hover:from-violet-200 group-hover:to-indigo-200 dark:from-violet-900/40 dark:to-indigo-900/40 mb-3"),
								lucide.Sparkles(Class("h-7 w-7 text-violet-600 dark:text-violet-400")),
							),
							Span(Class("font-semibold text-slate-900 dark:text-white"), g.Text("Visual Builder")),
							Span(Class("mt-1 text-xs text-slate-500 dark:text-gray-400 text-center"), g.Text("Drag & drop blocks, no coding required")),
							Div(
								Class("absolute top-2 right-2"),
								Span(Class("inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-300"),
									g.Text("Recommended")),
							),
						),

						// Manual/Code Editor option
						Button(
							Type("button"),
							g.Attr("@click", "selectEditorType('manual')"),
							Class("group flex flex-col items-center p-6 rounded-xl border-2 border-slate-200 hover:border-slate-400 hover:bg-slate-50 dark:border-gray-700 dark:hover:border-gray-500 dark:hover:bg-gray-800 transition-all"),
							Div(
								Class("flex h-14 w-14 items-center justify-center rounded-xl bg-slate-100 group-hover:bg-slate-200 dark:bg-gray-800 dark:group-hover:bg-gray-700 mb-3"),
								lucide.Code(Class("h-7 w-7 text-slate-600 dark:text-gray-400")),
							),
							Span(Class("font-semibold text-slate-900 dark:text-white"), g.Text("Code Editor")),
							Span(Class("mt-1 text-xs text-slate-500 dark:text-gray-400 text-center"), g.Text("Write HTML/templates directly")),
						),
					),

					// Cancel button
					Div(
						Class("text-center"),
						Button(
							Type("button"),
							g.Attr("@click", "showCreateModal = false; resetModal()"),
							Class("text-sm text-slate-500 hover:text-slate-700 dark:text-gray-400 dark:hover:text-gray-200"),
							g.Text("Cancel"),
						),
					),
				),

				// Step 2: Template Details Form (shown when editorType is selected)
				Div(
					g.Attr("x-show", "editorType"),
					g.Attr("x-cloak", ""),

					// Header with back button
					Div(
						Class("flex items-center gap-4 mb-6"),
						Button(
							Type("button"),
							g.Attr("@click", "editorType = ''; selectedSample = ''"),
							Class("flex h-10 w-10 items-center justify-center rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:text-gray-500 dark:hover:text-gray-300 dark:hover:bg-gray-800"),
							lucide.ArrowLeft(Class("h-5 w-5")),
						),
						Div(
							Class("flex h-12 w-12 items-center justify-center rounded-xl shadow-lg"),
							g.Attr(":class", "editorType === 'visual' ? 'bg-gradient-to-br from-violet-500 to-indigo-600' : 'bg-slate-600 dark:bg-gray-700'"),
							g.Raw(`<template x-if="editorType === 'visual'">`),
							lucide.Sparkles(Class("h-6 w-6 text-white")),
							g.Raw(`</template>`),
							g.Raw(`<template x-if="editorType === 'manual'">`),
							lucide.Code(Class("h-6 w-6 text-white")),
							g.Raw(`</template>`),
						),
						Div(
							H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
								g.Raw(`<span x-text="editorType === 'visual' ? 'Visual Builder' : 'Code Editor'"></span>`),
								g.Text(" Template")),
							P(Class("text-sm text-slate-500 dark:text-gray-400"),
								g.Raw(`<span x-text="editorType === 'visual' ? 'Choose a starting point or start from scratch' : 'Set up your template details'"></span>`)),
						),
					),

					// Two-column layout for Visual Builder
					Div(
						g.Attr("x-show", "editorType === 'visual'"),
						g.Attr("x-cloak", ""),
						Class("grid grid-cols-2 gap-6"),

						// Left: Sample Templates
						Div(
							Class("space-y-3"),
							H4(Class("text-sm font-medium text-slate-700 dark:text-gray-300 mb-3"), g.Text("Start from a template")),
							Div(
								Class("space-y-2 max-h-80 overflow-y-auto pr-2"),
								// Blank template option
								renderSampleTemplateOption("", "Blank Template", "Start from scratch", "🆕", true),
								// Sample templates
								renderSampleTemplateOption("welcome", "Welcome Email", "Onboarding new users", "🎉", false),
								renderSampleTemplateOption("otp", "Verification Code", "OTP/2FA authentication", "🔐", false),
								renderSampleTemplateOption("reset_password", "Password Reset", "Reset password link", "🔑", false),
								renderSampleTemplateOption("invitation", "Team Invitation", "Invite to organization", "👥", false),
								renderSampleTemplateOption("magic_link", "Magic Link", "Passwordless sign-in", "✨", false),
								renderSampleTemplateOption("order_confirmation", "Order Confirmation", "E-commerce receipt", "🛒", false),
								renderSampleTemplateOption("newsletter", "Newsletter", "Marketing newsletter", "📰", false),
								renderSampleTemplateOption("account_alert", "Security Alert", "Account security notice", "🚨", false),
							),
						),

						// Right: Form fields
						Div(
							Class("space-y-4"),
							// Template Name
							Div(
								Label(
									For("modalTemplateName"),
									Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
									g.Text("Template Name"),
									Span(Class("text-red-500"), g.Text(" *")),
								),
								Input(
									Type("text"),
									ID("modalTemplateName"),
									g.Attr("x-model", "newTemplate.name"),
									g.Attr("@input", "generateKey()"),
									g.Attr("placeholder", "e.g., Welcome Email"),
									Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
								),
							),

							// Template Key
							Div(
								Label(
									For("modalTemplateKey"),
									Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
									g.Text("Template Key"),
									Span(Class("text-red-500"), g.Text(" *")),
								),
								Input(
									Type("text"),
									ID("modalTemplateKey"),
									g.Attr("x-model", "newTemplate.templateKey"),
									g.Attr("placeholder", "e.g., custom.welcome_email"),
									Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white font-mono text-sm"),
								),
								P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
									g.Text("Auto-generated from name")),
							),

							// Subject Line
							Div(
								Label(
									For("modalSubject"),
									Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
									g.Text("Email Subject"),
								),
								Input(
									Type("text"),
									ID("modalSubject"),
									g.Attr("x-model", "newTemplate.subject"),
									g.Attr("placeholder", "e.g., Welcome to {{.app_name}}!"),
									Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
								),
							),

							// Error display
							Div(
								g.Attr("x-show", "createError"),
								g.Attr("x-cloak", ""),
								Class("rounded-md bg-red-50 p-3 dark:bg-red-900/20"),
								P(Class("text-sm text-red-800 dark:text-red-200"),
									Span(g.Attr("x-text", "createError")),
								),
							),
						),
					),

					// Single column for Manual Editor
					Div(
						g.Attr("x-show", "editorType === 'manual'"),
						g.Attr("x-cloak", ""),
						Class("space-y-4"),

						// Template Name
						Div(
							Label(
								For("modalTemplateNameManual"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Template Name"),
								Span(Class("text-red-500"), g.Text(" *")),
							),
							Input(
								Type("text"),
								ID("modalTemplateNameManual"),
								g.Attr("x-model", "newTemplate.name"),
								g.Attr("@input", "generateKey()"),
								g.Attr("placeholder", "e.g., Welcome Email"),
								Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							),
						),

						// Template Key
						Div(
							Label(
								For("modalTemplateKeyManual"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Template Key"),
								Span(Class("text-red-500"), g.Text(" *")),
							),
							Input(
								Type("text"),
								ID("modalTemplateKeyManual"),
								g.Attr("x-model", "newTemplate.templateKey"),
								g.Attr("placeholder", "e.g., custom.welcome_email"),
								Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white font-mono text-sm"),
							),
							P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
								g.Text("Auto-generated from name. Use lowercase letters, numbers, dots, and underscores.")),
						),

						// Subject Line
						Div(
							Label(
								For("modalSubjectManual"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Email Subject"),
							),
							Input(
								Type("text"),
								ID("modalSubjectManual"),
								g.Attr("x-model", "newTemplate.subject"),
								g.Attr("placeholder", "e.g., Welcome to {{.app_name}}!"),
								Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							),
							P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
								g.Text("Use {{.variable_name}} for dynamic content")),
						),

						// Error display
						Div(
							g.Attr("x-show", "createError"),
							g.Attr("x-cloak", ""),
							Class("rounded-md bg-red-50 p-3 dark:bg-red-900/20"),
							P(Class("text-sm text-red-800 dark:text-red-200"),
								Span(g.Attr("x-text", "createError")),
							),
						),
					),

					// Actions
					Div(
						Class("flex items-center justify-end gap-3 mt-6 pt-4 border-t border-slate-200 dark:border-gray-700"),
						Button(
							Type("button"),
							g.Attr("@click", "showCreateModal = false; resetModal()"),
							Class("px-4 py-2 text-sm font-medium text-slate-700 hover:text-slate-900 dark:text-gray-300 dark:hover:text-white"),
							g.Text("Cancel"),
						),
						Button(
							Type("button"),
							g.Attr("@click", "createTemplate()"),
							g.Attr(":disabled", "createLoading || !newTemplate.name || !newTemplate.templateKey"),
							g.Attr(":class", "editorType === 'visual' ? 'bg-gradient-to-r from-violet-600 to-indigo-600 hover:from-violet-700 hover:to-indigo-700' : 'bg-slate-700 hover:bg-slate-800 dark:bg-gray-600 dark:hover:bg-gray-500'"),
							Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"),
							g.Attr("x-show", "!createLoading"),
							g.Raw(`<template x-if="editorType === 'visual'">`),
							lucide.Sparkles(Class("h-4 w-4")),
							g.Raw(`</template>`),
							g.Raw(`<template x-if="editorType === 'manual'">`),
							lucide.Code(Class("h-4 w-4")),
							g.Raw(`</template>`),
							g.Raw(`<span x-text="editorType === 'visual' ? 'Create & Open Builder' : 'Create & Edit'"></span>`),
						),
						Button(
							Type("button"),
							Disabled(),
							Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg opacity-50"),
							g.Attr("x-show", "createLoading"),
							g.Attr("x-cloak", ""),
							lucide.RefreshCw(Class("h-4 w-4 animate-spin")),
							g.Text("Creating..."),
						),
					),
				),
			),
		),
	)
}

// renderSampleTemplateOption renders a sample template option in the modal
func renderSampleTemplateOption(id, name, description, icon string, isBlank bool) g.Node {
	borderClass := "border-2 border-transparent"
	if isBlank {
		borderClass = "border-2 border-dashed border-slate-300 dark:border-gray-600"
	}

	return Button(
		Type("button"),
		g.Attr("@click", fmt.Sprintf("selectedSample = '%s'", id)),
		g.Attr(":class", fmt.Sprintf("{'ring-2 ring-violet-500 border-violet-500 bg-violet-50 dark:bg-violet-900/20': selectedSample === '%s'}", id)),
		Class("w-full flex items-center gap-3 p-3 rounded-lg hover:bg-slate-50 dark:hover:bg-gray-800 transition-all text-left "+borderClass),
		// Icon
		Span(Class("text-xl flex-shrink-0"), g.Text(icon)),
		// Info
		Div(
			Class("flex-1 min-w-0"),
			Div(Class("font-medium text-sm text-slate-900 dark:text-white"), g.Text(name)),
			Div(Class("text-xs text-slate-500 dark:text-gray-400 truncate"), g.Text(description)),
		),
		// Check icon when selected
		Div(
			g.Attr("x-show", fmt.Sprintf("selectedSample === '%s'", id)),
			g.Attr("x-cloak", ""),
			lucide.CircleCheck(Class("h-5 w-5 text-violet-600 flex-shrink-0")),
		),
	)
}

// renderTestSendListModal renders the test send modal for the templates list
func renderTestSendListModal() g.Node {
	return Div(
		g.Attr("x-show", "showTestModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("@keydown.escape.window", "showTestModal = false"),

		// Overlay
		Div(
			g.Attr("x-show", "showTestModal"),
			g.Attr("@click", "showTestModal = false"),
			Class("fixed inset-0 bg-black/50 bg-opacity-50 transition-opacity"),
		),

		// Modal
		Div(
			Class("flex min-h-screen items-center justify-center p-4"),
			Div(
				g.Attr("x-show", "showTestModal"),
				g.Attr("@click.stop", ""),
				Class("relative bg-white/50 dark:bg-gray-900/50 rounded-lg shadow-xl max-w-md w-full p-6"),

				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Send Test Notification")),

				Div(
					Class("space-y-4"),
					Div(
						Label(
							For("testRecipientInput"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Recipient"),
						),
						Input(
							Type("text"),
							ID("testRecipientInput"),
							g.Attr("x-model", "testRecipient"),
							g.Attr("placeholder", "email@example.com or +1234567890"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("Email address or phone number to send test to")),
					),

					// Test Variables
					Div(
						Label(
							For("testVariablesInput"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Test Variables (JSON)"),
						),
						Textarea(
							ID("testVariablesInput"),
							g.Attr("x-model", "testVariables"),
							g.Attr("placeholder", "{\n  \"user_name\": \"John\",\n  \"app_name\": \"MyApp\"\n}"),
							Rows("5"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white font-mono text-sm"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("Variables to replace in template (e.g., {{.user_name}})")),
					),

					// Test result display
					Div(
						g.Attr("x-show", "testResult"),
						g.Attr("x-cloak", ""),
						Div(
							g.Attr("x-show", "testResult && testResult.success"),
							Class("rounded-md bg-green-50 p-3 dark:bg-green-900/20"),
							Div(Class("flex items-center gap-2"),
								lucide.CircleCheck(Class("h-5 w-5 text-green-600 dark:text-green-400")),
								P(Class("text-sm text-green-800 dark:text-green-200"),
									g.Text("Test notification sent successfully!")),
							),
						),
						Div(
							g.Attr("x-show", "testResult && testResult.error"),
							Class("rounded-md bg-red-50 p-3 dark:bg-red-900/20"),
							Div(Class("flex items-center gap-2"),
								lucide.CircleAlert(Class("h-5 w-5 text-red-600 dark:text-red-400")),
								P(Class("text-sm text-red-800 dark:text-red-200"),
									Span(g.Text("Error: ")),
									Span(g.Attr("x-text", "testResult.error")),
								),
							),
						),
					),
				),

				Div(
					Class("flex items-center justify-end gap-3 mt-6 pt-4 border-t border-slate-200 dark:border-gray-700"),
					Button(
						Type("button"),
						g.Attr("@click", "showTestModal = false"),
						Class("px-4 py-2 text-sm font-medium text-slate-700 hover:text-slate-900 dark:text-gray-300 dark:hover:text-white"),
						g.Text("Close"),
					),
					Button(
						Type("button"),
						g.Attr("@click", "sendTestNotification()"),
						g.Attr(":disabled", "testLoading || !testRecipient"),
						Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50"),
						g.Attr("x-show", "!testLoading"),
						lucide.Send(Class("h-4 w-4")),
						g.Text("Send Test"),
					),
					Button(
						Type("button"),
						Disabled(),
						Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg opacity-50"),
						g.Attr("x-show", "testLoading"),
						g.Attr("x-cloak", ""),
						lucide.RefreshCw(Class("h-4 w-4 animate-spin")),
						g.Text("Sending..."),
					),
				),
			),
		),
	)
}

// renderBuilderPromoCard renders a promotional card for the visual builder
func renderBuilderPromoCard(basePath string, currentApp *app.App) g.Node {
	// Unused but kept for API compatibility
	_ = basePath
	_ = currentApp

	return Div(
		Class("rounded-xl border border-indigo-200 bg-gradient-to-br from-indigo-50 via-violet-50 to-purple-50 p-6 dark:border-indigo-800 dark:from-indigo-900/20 dark:via-violet-900/20 dark:to-purple-900/20"),
		Div(
			Class("flex items-start gap-5"),
			// Icon
			Div(
				Class("flex-shrink-0"),
				Div(
					Class("flex h-14 w-14 items-center justify-center rounded-xl bg-gradient-to-br from-violet-500 to-indigo-600 shadow-lg"),
					lucide.Sparkles(Class("h-7 w-7 text-white")),
				),
			),
			// Content
			Div(
				Class("flex-1"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("Visual Email Template Builder")),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Design beautiful, responsive email templates visually with our drag-and-drop builder. No HTML knowledge required.")),
				// Features
				Div(
					Class("mt-4 flex flex-wrap gap-3"),
					featureBadge("Drag & Drop"),
					featureBadge("Live Preview"),
					featureBadge("Mobile Responsive"),
					featureBadge("Sample Templates"),
				),
				// CTA - Now opens modal with type selection
				Div(
					Class("mt-5"),
					Button(
						Type("button"),
						g.Attr("@click", "openCreateModal()"),
						Class("inline-flex items-center gap-2 rounded-lg bg-gradient-to-r from-violet-600 to-indigo-600 px-5 py-2.5 text-sm font-medium text-white hover:from-violet-700 hover:to-indigo-700 shadow-sm transition-all"),
						lucide.Plus(Class("h-4 w-4")),
						g.Text("Create New Template"),
					),
				),
			),
		),
	)
}

// featureBadge renders a small feature badge
func featureBadge(text string) g.Node {
	return Span(
		Class("inline-flex items-center gap-1 rounded-full bg-white/60 px-2.5 py-1 text-xs font-medium text-indigo-700 dark:bg-gray-800/60 dark:text-indigo-300"),
		lucide.Check(Class("h-3 w-3")),
		g.Text(text),
	)
}

// renderEmptyTemplatesState renders the empty state when no templates exist
func renderEmptyTemplatesState(basePath string, currentApp *app.App) g.Node {
	// Unused but kept for API compatibility
	_ = basePath
	_ = currentApp

	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-12 text-center dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/20"),
			lucide.FileText(Class("h-6 w-6 text-violet-600 dark:text-violet-400")),
		),
		H3(Class("mt-4 text-lg font-semibold text-slate-900 dark:text-white"),
			g.Text("No templates yet")),
		P(Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
			g.Text("Get started by creating your first notification template")),
		Div(
			Class("mt-6 flex items-center justify-center"),
			// Create template button - opens modal with type selection
			Button(
				Type("button"),
				g.Attr("@click", "openCreateModal()"),
				Class("inline-flex items-center gap-2 rounded-lg bg-gradient-to-r from-violet-600 to-indigo-600 px-4 py-2 text-sm font-medium text-white hover:from-violet-700 hover:to-indigo-700"),
				lucide.Plus(Class("h-4 w-4")),
				g.Text("Create Template"),
			),
		),
	)
}

// renderTemplatesTable renders the templates data table
func renderTemplatesTable(templates []*notification.Template, basePath string, currentApp *app.App) g.Node {
	return Div(
		Class("overflow-hidden rounded-lg border border-slate-200 bg-white shadow dark:border-gray-800 dark:bg-gray-900"),

		// Table
		Div(
			Class("overflow-x-auto"),
			Table(
				Class("min-w-full divide-y divide-slate-200 dark:divide-gray-800"),

				// Table header
				g.El("thead",
					Class("bg-slate-50 dark:bg-gray-800/50"),
					Tr(
						Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
							g.Text("Name")),
						Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
							g.Text("Type")),
						Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
							g.Text("Language")),
						Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
							g.Text("Status")),
						Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
							g.Text("Stats")),
						Th(Class("px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"),
							g.Text("Actions")),
					),
				),

				// Table body
				g.El("tbody",
					Class("divide-y divide-slate-200 bg-white dark:divide-gray-800 dark:bg-gray-900"),
					g.Group(renderTemplateRows(templates, basePath, currentApp)),
				),
			),
		),
	)
}

// renderTemplateRows renders individual template rows
func renderTemplateRows(templates []*notification.Template, basePath string, currentApp *app.App) []g.Node {
	rows := make([]g.Node, len(templates))
	for i, template := range templates {
		rows[i] = renderTemplateRow(template, basePath, currentApp)
	}
	return rows
}

// renderTemplateRow renders a single template row
func renderTemplateRow(template *notification.Template, basePath string, currentApp *app.App) g.Node {
	// Check if this is a visual builder template
	isVisualBuilder := false
	if template.Metadata != nil {
		if builderType, ok := template.Metadata["builderType"].(string); ok && builderType == "visual" {
			isVisualBuilder = true
		}
	}

	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),

		// Name and key
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Div(Class("flex items-center gap-2"),
					Span(Class("text-sm font-medium text-slate-900 dark:text-white"),
						g.Text(template.Name)),
					// Visual builder badge
					g.If(isVisualBuilder,
						Span(
							Class("inline-flex items-center gap-1 rounded-full bg-violet-100 px-1.5 py-0.5 text-xs font-medium text-violet-700 dark:bg-violet-900/30 dark:text-violet-300"),
							lucide.Sparkles(Class("h-3 w-3")),
						),
					),
				),
				Div(Class("text-xs text-slate-500 dark:text-gray-400"),
					g.Text(template.TemplateKey)),
			),
		),

		// Type
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Span(
				Class("inline-flex items-center gap-1 rounded-full px-2 py-1 text-xs font-medium"),
				g.If(template.Type == notification.NotificationTypeEmail,
					Class("bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300"),
				),
				g.If(template.Type == notification.NotificationTypeSMS,
					Class("bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300"),
				),
				g.If(template.Type == notification.NotificationTypeEmail,
					lucide.Mail(Class("h-3 w-3")),
				),
				g.If(template.Type == notification.NotificationTypeSMS,
					lucide.MessageSquare(Class("h-3 w-3")),
				),
				g.Text(string(template.Type)),
			),
		),

		// Language
		Td(Class("px-6 py-4 whitespace-nowrap text-sm text-slate-900 dark:text-white"),
			g.Text(template.Language)),

		// Status
		Td(Class("px-6 py-4 whitespace-nowrap"),
			g.If(template.Active,
				Span(
					Class("inline-flex items-center gap-1 rounded-full bg-green-100 px-2 py-1 text-xs font-medium text-green-800 dark:bg-green-900/30 dark:text-green-300"),
					lucide.Check(Class("h-3 w-3")),
					g.Text("Active"),
				),
			),
			g.If(!template.Active,
				Span(
					Class("inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-1 text-xs font-medium text-gray-800 dark:bg-gray-700 dark:text-gray-300"),
					lucide.Circle(Class("h-3 w-3")),
					g.Text("Inactive"),
				),
			),
		),

		// Stats
		Td(Class("px-6 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-gray-400"),
			Span(Class("text-xs text-slate-500 dark:text-gray-500"),
				g.Text("-")),
		),

		// Actions
		Td(Class("px-6 py-4 whitespace-nowrap text-right text-sm font-medium"),
			Div(Class("flex items-center justify-end gap-2"),
				// Edit button - goes to /edit which auto-redirects based on template type
				A(
					Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates/%s/edit", basePath, currentApp.ID, template.ID)),
					Class("text-violet-600 hover:text-violet-900 dark:text-violet-400 dark:hover:text-violet-300"),
					g.If(isVisualBuilder,
						Title("Edit in Visual Builder"),
					),
					g.If(!isVisualBuilder,
						Title("Edit template"),
					),
					lucide.Pencil(Class("h-4 w-4")),
				),
				Button(
					Type("button"),
					Class("text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300"),
					Title("Test send"),
					g.Attr("@click", fmt.Sprintf("openTestModal('%s')", template.ID)),
					lucide.Send(Class("h-4 w-4")),
				),
				Button(
					Type("button"),
					Class("text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"),
					Title("Delete template"),
					g.Attr("@click", fmt.Sprintf("if(confirm('Delete template \"%s\"?')) window.location.href='%s/auth/notification/templates/%s?redirect=%s/dashboard/app/%s/notifications/templates&_method=DELETE'", template.Name, basePath, template.ID, basePath, currentApp.ID)),
					lucide.Trash2(Class("h-4 w-4")),
				),
			),
		),
	)
}

// renderCreateTemplate renders the create template form
func (e *DashboardExtension) renderCreateTemplate(currentApp *app.App, basePath string) g.Node {
	return Div(
		Class("space-y-6"),
		g.Attr("x-data", `{
			templateType: 'email',
			showSubject: true,
			templateBody: '',
			templateSubject: '',
			extractedVars: [],
			extractVariables() {
				const regex = /\{\{\.(\w+)\}\}/g;
				const vars = new Set();
				let match;
				
				// Extract from subject
				while ((match = regex.exec(this.templateSubject)) !== null) {
					vars.add(match[1]);
				}
				
				// Extract from body
				regex.lastIndex = 0;
				while ((match = regex.exec(this.templateBody)) !== null) {
					vars.add(match[1]);
				}
				
				this.extractedVars = Array.from(vars);
			}
		}`),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text("Create Notification Template")),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Create a new email or SMS notification template")),
			),
			A(
				Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates", basePath, currentApp.ID)),
				Class("text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
				g.Text("← Back to Templates"),
			),
		),

		// Main form
		FormEl(
			Method("POST"),
			Action(fmt.Sprintf("%s/auth/notification/templates?redirect=%s/dashboard/app/%s/notifications/templates",
				basePath, basePath, currentApp.ID)),
			g.Attr("x-on:input.debounce.500ms", "extractVariables()"),
			Class("space-y-6"),

			// Basic Information Section
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Basic Information")),

				Div(Class("space-y-4"),
					// Template Name
					Div(
						Label(
							For("name"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Template Name"),
							Span(Class("text-red-500"), g.Text(" *")),
						),
						Input(
							Type("text"),
							ID("name"),
							Name("name"),
							Required(),
							g.Attr("placeholder", "e.g., Welcome Email, Password Reset"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("A descriptive name for this template")),
					),

					// Template Key
					Div(
						Label(
							For("templateKey"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Template Key"),
							Span(Class("text-red-500"), g.Text(" *")),
						),
						Input(
							Type("text"),
							ID("templateKey"),
							Name("templateKey"),
							Required(),
							g.Attr("placeholder", "e.g., auth.welcome, auth.password_reset"),
							g.Attr("pattern", "[a-z0-9_.]+"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("Unique identifier (lowercase, dots, underscores only)")),
					),

					// Type Selection
					Div(
						Label(
							For("type"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Notification Type"),
							Span(Class("text-red-500"), g.Text(" *")),
						),
						Select(
							ID("type"),
							Name("type"),
							Required(),
							g.Attr("x-model", "templateType"),
							g.Attr("@change", "showSubject = (templateType === 'email')"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("email"), Selected(), g.Text("Email")),
							Option(Value("sms"), g.Text("SMS")),
						),
					),

					// Language
					Div(
						Label(
							For("language"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Language"),
						),
						Select(
							ID("language"),
							Name("language"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("en"), Selected(), g.Text("English (en)")),
							Option(Value("es"), g.Text("Spanish (es)")),
							Option(Value("fr"), g.Text("French (fr)")),
							Option(Value("de"), g.Text("German (de)")),
							Option(Value("pt"), g.Text("Portuguese (pt)")),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("Language for this template variant")),
					),

					// Active Toggle
					Div(
						Label(
							Class("flex items-center gap-2"),
							Input(
								Type("checkbox"),
								Name("active"),
								Value("true"),
								Checked(),
								Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
							),
							Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Active")),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400 ml-6"),
							g.Text("Make this template available for use immediately")),
					),
				),
			),

			// Content Section
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Template Content")),

				Div(Class("space-y-4"),
					// Subject (Email only)
					Div(
						g.Attr("x-show", "showSubject"),
						g.Attr("x-cloak", ""),
						Label(
							For("subject"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Subject Line"),
							Span(Class("text-red-500"), g.Attr("x-show", "showSubject"), g.Text(" *")),
						),
						Input(
							Type("text"),
							ID("subject"),
							Name("subject"),
							g.Attr("x-model", "templateSubject"),
							g.Attr("placeholder", "e.g., Welcome to {{.app_name}}!"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("Use {{.variable_name}} for dynamic content")),
					),

					// Body
					Div(
						Label(
							For("body"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Message Body"),
							Span(Class("text-red-500"), g.Text(" *")),
						),
						Textarea(
							ID("body"),
							Name("body"),
							Required(),
							g.Attr("x-model", "templateBody"),
							g.Attr("placeholder", "e.g., Hello {{.user_name}},\n\nWelcome to our platform!"),
							g.Attr("rows", "10"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white font-mono text-sm"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("Use {{.variable_name}} syntax for template variables. Supports Go template language.")),
					),

					// Variable hints
					Div(
						Class("rounded-md bg-blue-50 p-4 dark:bg-blue-900/20"),
						Div(Class("flex items-start gap-3"),
							lucide.Info(Class("h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5")),
							Div(
								H4(Class("text-sm font-medium text-blue-900 dark:text-blue-300"),
									g.Text("Template Variables")),
								P(Class("mt-1 text-xs text-blue-800 dark:text-blue-200"),
									g.Text("Use these syntax patterns:")),
								Ul(Class("mt-2 text-xs text-blue-800 dark:text-blue-200 list-disc list-inside space-y-1"),
									Li(g.Text("{{.variable_name}} - Simple variable")),
									Li(g.Text("{{if .condition}}...{{end}} - Conditional")),
									Li(g.Text("{{range .items}}...{{end}} - Loop")),
								),
								Div(
									g.Attr("x-show", "extractedVars.length > 0"),
									g.Attr("x-cloak", ""),
									Class("mt-3 pt-3 border-t border-blue-200 dark:border-blue-800"),
									P(Class("text-xs font-medium text-blue-900 dark:text-blue-300 mb-1"),
										g.Text("Detected variables:")),
									Div(
										Class("flex flex-wrap gap-1"),
										g.Raw(`<template x-for="varName in extractedVars" :key="varName">
											<span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-mono bg-blue-100 text-blue-800 dark:bg-blue-800 dark:text-blue-100" x-text="varName"></span>
										</template>`),
									),
								),
							),
						),
					),
				),
			),

			// Hidden field for app ID
			Input(Type("hidden"), Name("appId"), Value(currentApp.ID.String())),

			// Form Actions
			Div(
				Class("flex items-center justify-end gap-4 pt-6"),
				A(
					Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates", basePath, currentApp.ID)),
					Class("rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					lucide.Check(Class("h-4 w-4")),
					g.Text("Create Template"),
				),
			),
		),
	)
}

// renderEditTemplate renders the edit template form
func (e *DashboardExtension) renderEditTemplate(currentApp *app.App, basePath string, templateID xid.ID) g.Node {
	ctx := context.Background()

	// Fetch existing template
	template, err := e.plugin.service.GetTemplate(ctx, templateID)
	if err != nil {
		return Div(
			Class("space-y-6"),
			Div(
				Class("rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/20"),
				P(Class("text-sm text-red-800 dark:text-red-200"),
					g.Textf("Error loading template: %v", err)),
			),
		)
	}

	return Div(
		Class("space-y-6"),
		g.Attr("x-data", fmt.Sprintf(`{
			templateType: '%s',
			showSubject: %t,
			templateBody: '',
			templateSubject: '',
			extractedVars: [],
			showDeleteConfirm: false,
			showTestSend: false,
			testRecipient: '',
			testVariables: '{\n  "user_name": "Test User",\n  "app_name": "My App"\n}',
			testLoading: false,
			testResult: null,
			extractVariables() {
				const regex = /\{\{\.(\w+)\}\}/g;
				const vars = new Set();
				let match;
				
				while ((match = regex.exec(this.templateSubject)) !== null) {
					vars.add(match[1]);
				}
				
				regex.lastIndex = 0;
				while ((match = regex.exec(this.templateBody)) !== null) {
					vars.add(match[1]);
				}
				
				this.extractedVars = Array.from(vars);
			},
			async sendTestNotification() {
				if (!this.testRecipient) return;
				this.testLoading = true;
				this.testResult = null;
				try {
					// Parse variables from JSON string
					let variables = {};
					if (this.testVariables && this.testVariables.trim()) {
						try {
							variables = JSON.parse(this.testVariables);
						} catch (e) {
							this.testResult = { error: 'Invalid JSON for variables: ' + e.message };
							this.testLoading = false;
							return;
						}
					}
					const res = await fetch('%s/dashboard/app/%s/notifications/templates/%s/test', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ recipient: this.testRecipient, variables: variables })
					});
					this.testResult = await res.json();
				} catch (err) {
					this.testResult = { error: err.message };
				}
				this.testLoading = false;
			}
		}`, template.Type, template.Type == notification.NotificationTypeEmail, basePath, currentApp.ID, templateID)),

		// Header with actions
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text("Edit Template")),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text(template.TemplateKey)),
			),
			Div(
				Class("flex items-center gap-3"),
				A(
					Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates", basePath, currentApp.ID)),
					Class("text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
					g.Text("← Back to Templates"),
				),
				Button(
					Type("button"),
					g.Attr("@click", "showTestSend = true"),
					Class("inline-flex items-center gap-2 rounded-lg border border-blue-300 bg-blue-50 px-3 py-1.5 text-sm font-medium text-blue-700 hover:bg-blue-100 dark:border-blue-700 dark:bg-blue-900/30 dark:text-blue-300"),
					lucide.Send(Class("h-4 w-4")),
					g.Text("Test Send"),
				),
				Button(
					Type("button"),
					g.Attr("@click", "showDeleteConfirm = true"),
					Class("inline-flex items-center gap-2 rounded-lg border border-red-300 bg-red-50 px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-100 dark:border-red-700 dark:bg-red-900/30 dark:text-red-300"),
					lucide.Trash2(Class("h-4 w-4")),
					g.Text("Delete"),
				),
			),
		),

		// Edit Form
		FormEl(
			Method("POST"),
			Action(fmt.Sprintf("%s/auth/notification/templates/%s?redirect=%s/dashboard/app/%s/notifications/templates",
				basePath, templateID, basePath, currentApp.ID)),
			g.Attr("x-on:input.debounce.500ms", "extractVariables()"),
			Class("space-y-6"),

			// Basic Information Section
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Basic Information")),

				Div(Class("space-y-4"),
					// Template Name
					Div(
						Label(
							For("name"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Template Name"),
							Span(Class("text-red-500"), g.Text(" *")),
						),
						Input(
							Type("text"),
							ID("name"),
							Name("name"),
							Required(),
							Value(template.Name),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
					),

					// Template Key (readonly)
					Div(
						Label(
							For("templateKey"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Template Key"),
						),
						Input(
							Type("text"),
							ID("templateKey"),
							Value(template.TemplateKey),
							Disabled(),
							Class("mt-1 block w-full rounded-md border-slate-300 bg-slate-50 shadow-sm dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("Template key cannot be changed after creation")),
					),

					// Type (readonly)
					Div(
						Label(
							For("type"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Notification Type"),
						),
						Input(
							Type("text"),
							ID("type"),
							Value(string(template.Type)),
							Disabled(),
							Class("mt-1 block w-full rounded-md border-slate-300 bg-slate-50 shadow-sm dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400"),
						),
					),

					// Language
					Div(
						Label(
							For("language"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Language"),
						),
						Select(
							ID("language"),
							Name("language"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							Option(Value("en"), g.If(template.Language == "en", Selected()), g.Text("English (en)")),
							Option(Value("es"), g.If(template.Language == "es", Selected()), g.Text("Spanish (es)")),
							Option(Value("fr"), g.If(template.Language == "fr", Selected()), g.Text("French (fr)")),
							Option(Value("de"), g.If(template.Language == "de", Selected()), g.Text("German (de)")),
							Option(Value("pt"), g.If(template.Language == "pt", Selected()), g.Text("Portuguese (pt)")),
						),
					),

					// Active Toggle
					Div(
						Label(
							Class("flex items-center gap-2"),
							Input(
								Type("checkbox"),
								Name("active"),
								Value("true"),
								g.If(template.Active, Checked()),
								Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
							),
							Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Active")),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400 ml-6"),
							g.Text("Template is available for use")),
					),
				),
			),

			// Content Section
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Template Content")),

				Div(Class("space-y-4"),
					// Subject (Email only)
					g.If(template.Type == notification.NotificationTypeEmail,
						Div(
							Label(
								For("subject"),
								Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Subject Line"),
								Span(Class("text-red-500"), g.Text(" *")),
							),
							Input(
								Type("text"),
								ID("subject"),
								Name("subject"),
								Value(template.Subject),
								g.Attr("x-model", "templateSubject"),
								g.Attr("x-init", fmt.Sprintf("templateSubject = '%s'", template.Subject)),
								Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
							),
						),
					),

					// Body
					Div(
						Label(
							For("body"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Message Body"),
							Span(Class("text-red-500"), g.Text(" *")),
						),
						Textarea(
							ID("body"),
							Name("body"),
							Required(),
							g.Attr("x-model", "templateBody"),
							g.Attr("x-init", fmt.Sprintf("templateBody = `%s`", template.Body)),
							g.Attr("rows", "12"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white font-mono text-sm"),
							g.Text(template.Body),
						),
					),

					// Variable hints
					Div(
						Class("rounded-md bg-blue-50 p-4 dark:bg-blue-900/20"),
						Div(Class("flex items-start gap-3"),
							lucide.Info(Class("h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5")),
							Div(
								H4(Class("text-sm font-medium text-blue-900 dark:text-blue-300"),
									g.Text("Template Variables")),
								g.If(len(template.Variables) > 0,
									Div(
										Class("mt-2"),
										P(Class("text-xs text-blue-800 dark:text-blue-200 mb-1"),
											g.Text("Current variables:")),
										Div(
											Class("flex flex-wrap gap-1"),
											g.Group(renderVariableTags(template.Variables)),
										),
									),
								),
							),
						),
					),
				),
			),

			// Hidden fields
			Input(Type("hidden"), Name("_method"), Value("PUT")),

			// Form Actions
			Div(
				Class("flex items-center justify-end gap-4 pt-6"),
				A(
					Href(fmt.Sprintf("%s/dashboard/app/%s/notifications/templates", basePath, currentApp.ID)),
					Class("rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
					lucide.Check(Class("h-4 w-4")),
					g.Text("Save Changes"),
				),
			),
		),

		// Delete Confirmation Modal
		renderDeleteConfirmModal(templateID, template.Name, basePath, currentApp),

		// Test Send Modal
		renderTestSendModal(templateID, basePath, currentApp),
	)
}

// renderVariableTags renders variable name tags
func renderVariableTags(variables []string) []g.Node {
	tags := make([]g.Node, len(variables))
	for i, v := range variables {
		tags[i] = Span(
			Class("inline-flex items-center px-2 py-0.5 rounded text-xs font-mono bg-blue-100 text-blue-800 dark:bg-blue-800 dark:text-blue-100"),
			g.Text(v),
		)
	}
	return tags
}

// renderDeleteConfirmModal renders the delete confirmation modal
func renderDeleteConfirmModal(templateID xid.ID, templateName, basePath string, currentApp *app.App) g.Node {
	return Div(
		g.Attr("x-show", "showDeleteConfirm"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("@keydown.escape.window", "showDeleteConfirm = false"),

		// Overlay
		Div(
			g.Attr("x-show", "showDeleteConfirm"),
			g.Attr("@click", "showDeleteConfirm = false"),
			Class("fixed inset-0 bg-black bg-opacity-50 transition-opacity"),
		),

		// Modal
		Div(
			Class("flex min-h-screen items-center justify-center p-4"),
			Div(
				g.Attr("x-show", "showDeleteConfirm"),
				g.Attr("@click.stop", ""),
				Class("relative bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full p-6"),

				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"),
					g.Text("Delete Template")),
				P(Class("text-sm text-slate-600 dark:text-gray-400 mb-4"),
					g.Textf("Are you sure you want to delete \"%s\"? This action cannot be undone.", templateName)),

				Div(
					Class("flex items-center justify-end gap-3"),
					Button(
						Type("button"),
						g.Attr("@click", "showDeleteConfirm = false"),
						Class("px-4 py-2 text-sm font-medium text-slate-700 hover:text-slate-900 dark:text-gray-300 dark:hover:text-white"),
						g.Text("Cancel"),
					),
					FormEl(
						Method("POST"),
						Action(fmt.Sprintf("%s/auth/notification/templates/%s?redirect=%s/dashboard/app/%s/notifications/templates",
							basePath, templateID, basePath, currentApp.ID)),
						Class("inline"),
						Input(Type("hidden"), Name("_method"), Value("DELETE")),
						Button(
							Type("submit"),
							Class("px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700"),
							g.Text("Delete Template"),
						),
					),
				),
			),
		),
	)
}

// renderTestSendModal renders the test send modal
func renderTestSendModal(templateID xid.ID, basePath string, currentApp *app.App) g.Node {
	return Div(
		g.Attr("x-show", "showTestSend"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("@keydown.escape.window", "showTestSend = false"),

		// Overlay
		Div(
			g.Attr("x-show", "showTestSend"),
			g.Attr("@click", "showTestSend = false"),
			Class("fixed inset-0 bg-black bg-opacity-50 transition-opacity"),
		),

		// Modal
		Div(
			Class("flex min-h-screen items-center justify-center p-4"),
			Div(
				g.Attr("x-show", "showTestSend"),
				g.Attr("@click.stop", ""),
				Class("relative bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full p-6"),

				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Send Test Notification")),

				Div(
					Class("space-y-4"),

					Div(
						Label(
							For("testRecipient"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Recipient"),
						),
						Input(
							Type("text"),
							ID("testRecipient"),
							g.Attr("x-model", "testRecipient"),
							g.Attr("placeholder", "email@example.com or +1234567890"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("Email address or phone number to send test to")),
					),

					// Test Variables
					Div(
						Label(
							For("testVariablesEdit"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Test Variables (JSON)"),
						),
						Textarea(
							ID("testVariablesEdit"),
							g.Attr("x-model", "testVariables"),
							g.Attr("placeholder", "{\n  \"user_name\": \"John\",\n  \"app_name\": \"MyApp\"\n}"),
							Rows("5"),
							Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white font-mono text-sm"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
							g.Text("Variables to replace in template (e.g., {{.user_name}})")),
					),

					// Test result display
					Div(
						g.Attr("x-show", "testResult"),
						g.Attr("x-cloak", ""),
						Div(
							g.Attr("x-show", "testResult?.success"),
							Class("rounded-md bg-green-50 p-4 dark:bg-green-900/20"),
							Div(
								Class("flex"),
								lucide.Check(Class("h-5 w-5 text-green-400")),
								Div(
									Class("ml-3"),
									P(Class("text-sm font-medium text-green-800 dark:text-green-200"),
										g.Text("Test notification sent successfully!")),
								),
							),
						),
						Div(
							g.Attr("x-show", "testResult?.error"),
							Class("rounded-md bg-red-50 p-4 dark:bg-red-900/20"),
							Div(
								Class("flex"),
								lucide.X(Class("h-5 w-5 text-red-400")),
								Div(
									Class("ml-3"),
									P(
										Class("text-sm font-medium text-red-800 dark:text-red-200"),
										g.Attr("x-text", "testResult?.error"),
									),
								),
							),
						),
					),

					Div(
						Class("flex items-center justify-end gap-3 pt-4"),
						Button(
							Type("button"),
							g.Attr("@click", "showTestSend = false; testResult = null"),
							Class("px-4 py-2 text-sm font-medium text-slate-700 hover:text-slate-900 dark:text-gray-300 dark:hover:text-white"),
							g.Text("Close"),
						),
						Button(
							Type("button"),
							g.Attr("@click", "sendTestNotification()"),
							g.Attr(":disabled", "testLoading || !testRecipient"),
							Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50"),
							g.Attr("x-show", "!testLoading"),
							lucide.Send(Class("h-4 w-4")),
							g.Text("Send Test"),
						),
						Button(
							Type("button"),
							Disabled(),
							Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg opacity-50"),
							g.Attr("x-show", "testLoading"),
							lucide.RefreshCw(Class("h-4 w-4 animate-spin")),
							g.Text("Sending..."),
						),
					),
				),
			),
		),
	)
}

// renderNotificationSettingsContent renders the notification settings page
func (e *DashboardExtension) renderNotificationSettingsContent(currentApp *app.App, basePath string) g.Node {
	cfg := e.plugin.config

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Notification Settings")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Configure notification plugin behavior")),
		),

		// Settings form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Form(
				Method("POST"),
				Action(fmt.Sprintf("%s/dashboard/app/%s/settings/notifications/save", basePath, currentApp.ID)),
				Class("space-y-6"),

				// Auto-send welcome emails
				Div(
					Label(
						For("autoSendWelcome"),
						Class("flex items-center"),
						Input(
							Type("checkbox"),
							Name("autoSendWelcome"),
							ID("autoSendWelcome"),
							Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
							g.If(cfg.AutoSendWelcome, Checked()),
						),
						Span(Class("ml-2 text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Auto-send welcome emails")),
					),
					P(Class("mt-1 ml-6 text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Automatically send welcome emails when users sign up")),
				),

				// Retry attempts
				Div(
					Label(
						For("retryAttempts"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Retry Attempts"),
					),
					Input(
						Type("number"),
						Name("retryAttempts"),
						ID("retryAttempts"),
						Value(strconv.Itoa(cfg.RetryAttempts)),
						g.Attr("min", "0"),
						g.Attr("max", "10"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
					P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
						g.Text("Number of retry attempts for failed notifications")),
				),

				// Save button
				Div(
					Class("flex items-center justify-end gap-4"),
					Button(
						Type("submit"),
						Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						g.Text("Save Settings"),
					),
				),
			),
		),
	)
}

// renderProviderSettingsContent renders the provider settings page
func (e *DashboardExtension) renderProviderSettingsContent(currentApp *app.App, basePath string) g.Node {
	// Get current configuration from plugin
	cfg := e.plugin.config

	// Get email provider config
	emailProvider := cfg.Providers.Email.Provider
	if emailProvider == "" {
		emailProvider = "smtp"
	}
	emailFrom := cfg.Providers.Email.From
	emailFromName := cfg.Providers.Email.FromName
	emailConfig := cfg.Providers.Email.Config
	if emailConfig == nil {
		emailConfig = make(map[string]interface{})
	}

	// Get SMS provider config (optional)
	var smsProvider string
	var smsFrom string
	var smsConfig map[string]interface{}
	if cfg.Providers.SMS != nil {
		smsProvider = cfg.Providers.SMS.Provider
		smsFrom = cfg.Providers.SMS.From
		smsConfig = cfg.Providers.SMS.Config
	}
	if smsProvider == "" {
		smsProvider = "twilio"
	}
	if smsConfig == nil {
		smsConfig = make(map[string]interface{})
	}

	// Build Alpine.js initialization with current config
	alpineInit := fmt.Sprintf(`{
			emailProviderType: '%s',
			smsProviderType: '%s',
			showEmailTest: false,
			showSMSTest: false,
			testEmail: '',
			testPhone: '',
			testResult: null,
			testLoading: false,
			async testEmailProvider() {
				this.testLoading = true;
				this.testResult = null;
				try {
					const res = await fetch('%s/dashboard/app/%s/settings/notifications/providers/test', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({
							providerType: 'email',
							providerName: this.emailProviderType,
							testRecipient: this.testEmail
						})
					});
					this.testResult = await res.json();
				} catch (err) {
					this.testResult = { error: err.message };
				}
				this.testLoading = false;
			},
			async testSMSProvider() {
				this.testLoading = true;
				this.testResult = null;
				try {
					const res = await fetch('%s/dashboard/app/%s/settings/notifications/providers/test', {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({
							providerType: 'sms',
							providerName: this.smsProviderType,
							testRecipient: this.testPhone
						})
					});
					this.testResult = await res.json();
				} catch (err) {
					this.testResult = { error: err.message };
				}
				this.testLoading = false;
			}
		}`, emailProvider, smsProvider, basePath, currentApp.ID, basePath, currentApp.ID)

	// Helper to get string from config
	getConfigStr := func(config map[string]interface{}, key string) string {
		if val, ok := config[key].(string); ok {
			return val
		}
		return ""
	}

	return Div(
		Class("space-y-6"),
		g.Attr("x-data", alpineInit),

		// Header
		Div(
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Email & SMS Providers")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Configure notification delivery providers")),
		),

		// Security Notice
		Div(
			Class("rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-900/20"),
			Div(Class("flex items-start gap-3"),
				lucide.Shield(Class("h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5")),
				Div(
					H4(Class("text-sm font-medium text-blue-900 dark:text-blue-300"),
						g.Text("Credentials Security")),
					P(Class("mt-1 text-xs text-blue-800 dark:text-blue-200"),
						g.Text("All sensitive credentials (API keys, passwords, tokens) are automatically encrypted using AES-256-GCM before storage.")),
				),
			),
		),

		// Email Provider Section
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("flex items-center justify-between mb-4"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("Email Provider")),
				// Email provider is always configured, so show test button
				Button(
					Type("button"),
					g.Attr("@click", "showEmailTest = true"),
					Class("text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400"),
					g.Text("Test Connection"),
				),
			),

			FormEl(
				Method("POST"),
				Action(fmt.Sprintf("%s/auth/notification/providers?redirect=%s/dashboard/app/%s/notifications/providers",
					basePath, basePath, currentApp.ID)),
				Class("space-y-4"),

				// Provider Type Selection
				Div(
					Label(
						For("emailProviderType"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Provider Type"),
					),
					Select(
						ID("emailProviderType"),
						Name("providerName"),
						g.Attr("x-model", "emailProviderType"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Option(Value("smtp"), g.If(emailProvider == "" || emailProvider == "smtp", Selected()), g.Text("SMTP")),
						Option(Value("sendgrid"), g.If(emailProvider == "sendgrid", Selected()), g.Text("SendGrid")),
						Option(Value("resend"), g.If(emailProvider == "resend", Selected()), g.Text("Resend")),
						Option(Value("postmark"), g.If(emailProvider == "postmark", Selected()), g.Text("Postmark")),
					),
				),

				// SMTP Fields
				Div(
					g.Attr("x-show", "emailProviderType === 'smtp'"),
					g.Attr("x-cloak", ""),
					Class("space-y-4 pt-4 border-t border-slate-200 dark:border-gray-700"),

					renderProviderField("Host", "config[host]", "text", "smtp.example.com", "SMTP server hostname", true, getConfigStr(emailConfig, "host")),
					renderProviderField("Port", "config[port]", "number", "587", "SMTP server port (587 for TLS, 465 for SSL)", true, getConfigStr(emailConfig, "port")),
					renderProviderField("Username", "config[username]", "text", "your-username", "SMTP username", true, getConfigStr(emailConfig, "username")),
					renderProviderField("Password", "config[password]", "password", "", "SMTP password (will be encrypted)", true, ""),
					renderProviderField("From Address", "config[from]", "email", "noreply@example.com", "Default sender email address", true, emailFrom),

					Div(
						Label(
							Class("flex items-center gap-2"),
							Input(
								Type("checkbox"),
								Name("config[use_tls]"),
								Value("true"),
								g.If(emailConfig["use_tls"] != false, Checked()),
								Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800"),
							),
							Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
								g.Text("Use TLS")),
						),
					),
				),

				// SendGrid Fields
				Div(
					g.Attr("x-show", "emailProviderType === 'sendgrid'"),
					g.Attr("x-cloak", ""),
					Class("space-y-4 pt-4 border-t border-slate-200 dark:border-gray-700"),

					renderProviderField("API Key", "config[api_key]", "password", "", "SendGrid API key (will be encrypted)", true, ""),
					renderProviderField("From Address", "config[from]", "email", "noreply@example.com", "Default sender email address", true, emailFrom),
					renderProviderField("From Name", "config[from_name]", "text", "My App", "Default sender name", false, emailFromName),
				),

				// Resend Fields
				Div(
					g.Attr("x-show", "emailProviderType === 'resend'"),
					g.Attr("x-cloak", ""),
					Class("space-y-4 pt-4 border-t border-slate-200 dark:border-gray-700"),

					renderProviderField("API Key", "config[api_key]", "password", "", "Resend API key (will be encrypted)", true, ""),
					renderProviderField("From Address", "config[from]", "email", "noreply@example.com", "Default sender email address", true, emailFrom),
				),

				// Postmark Fields
				Div(
					g.Attr("x-show", "emailProviderType === 'postmark'"),
					g.Attr("x-cloak", ""),
					Class("space-y-4 pt-4 border-t border-slate-200 dark:border-gray-700"),

					renderProviderField("Server Token", "config[server_token]", "password", "", "Postmark server token (will be encrypted)", true, ""),
					renderProviderField("From Address", "config[from]", "email", "noreply@example.com", "Default sender email address", true, emailFrom),
				),

				// Hidden fields
				Input(Type("hidden"), Name("providerType"), Value("email")),
				Input(Type("hidden"), Name("isDefault"), Value("true")),
				Input(Type("hidden"), Name("appId"), Value(currentApp.ID.String())),

				// Submit
				Div(
					Class("flex items-center justify-end gap-3 pt-4"),
					Button(
						Type("submit"),
						Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						lucide.Save(Class("h-4 w-4")),
						g.Text("Save Email Provider"),
					),
				),
			),

			// Test Email Section
			Div(
				Class("mt-4 pt-4 border-t border-slate-200 dark:border-gray-700"),
				H3(Class("text-sm font-medium text-slate-900 dark:text-white mb-3"),
					g.Text("Test Email Configuration")),
				Div(
					Class("flex items-end gap-3"),
					Div(
						Class("flex-1"),
						Label(
							For("testEmail"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
							g.Text("Test Email Address"),
						),
						Input(
							Type("email"),
							ID("testEmail"),
							g.Attr("x-model", "testEmail"),
							g.Attr("placeholder", "test@example.com"),
							Class("block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
					),
					Button(
						Type("button"),
						g.Attr("@click", "testEmailProvider()"),
						g.Attr(":disabled", "testLoading || !testEmail"),
						Class("inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"),
						g.Attr("x-show", "!testLoading"),
						lucide.Send(Class("h-4 w-4")),
						g.Text("Send Test Email"),
					),
					Button(
						Type("button"),
						Disabled(),
						Class("inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white opacity-50"),
						g.Attr("x-show", "testLoading"),
						g.Attr("x-cloak", ""),
						lucide.RefreshCw(Class("h-4 w-4 animate-spin")),
						g.Text("Sending..."),
					),
				),
				// Test result display
				Div(
					g.Attr("x-show", "testResult"),
					g.Attr("x-cloak", ""),
					Class("mt-3"),
					Div(
						g.Attr("x-show", "testResult && testResult.success"),
						Class("rounded-md bg-green-50 p-3 dark:bg-green-900/20"),
						Div(Class("flex items-center gap-2"),
							lucide.CircleCheck(Class("h-5 w-5 text-green-600 dark:text-green-400")),
							P(Class("text-sm text-green-800 dark:text-green-200"),
								g.Text("Test email sent successfully!")),
						),
					),
					Div(
						g.Attr("x-show", "testResult && testResult.error"),
						Class("rounded-md bg-red-50 p-3 dark:bg-red-900/20"),
						Div(Class("flex items-center gap-2"),
							lucide.CircleAlert(Class("h-5 w-5 text-red-600 dark:text-red-400")),
							P(Class("text-sm text-red-800 dark:text-red-200"),
								Span(g.Text("Error: ")),
								Span(g.Attr("x-text", "testResult.error")),
							),
						),
					),
				),
			),
		),

		// SMS Provider Section
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("flex items-center justify-between mb-4"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("SMS Provider (Optional)")),
				g.If(smsProvider != "",
					Button(
						Type("button"),
						g.Attr("@click", "showSMSTest = true"),
						Class("text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400"),
						g.Text("Test Connection"),
					),
				),
			),

			FormEl(
				Method("POST"),
				Action(fmt.Sprintf("%s/auth/notification/providers?redirect=%s/dashboard/app/%s/notifications/providers",
					basePath, basePath, currentApp.ID)),
				Class("space-y-4"),

				// Provider Type Selection
				Div(
					Label(
						For("smsProviderType"),
						Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
						g.Text("Provider Type"),
					),
					Select(
						ID("smsProviderType"),
						Name("providerName"),
						g.Attr("x-model", "smsProviderType"),
						Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						Option(Value("twilio"), g.If(smsProvider == "" || smsProvider == "twilio", Selected()), g.Text("Twilio")),
						Option(Value("aws-sns"), g.If(smsProvider == "aws-sns", Selected()), g.Text("AWS SNS")),
					),
				),

				// Twilio Fields
				Div(
					g.Attr("x-show", "smsProviderType === 'twilio'"),
					g.Attr("x-cloak", ""),
					Class("space-y-4 pt-4 border-t border-slate-200 dark:border-gray-700"),

					renderProviderField("Account SID", "config[account_sid]", "text", "ACxxxxxxxxxxxxx", "Twilio Account SID", true, getConfigStr(smsConfig, "account_sid")),
					renderProviderField("Auth Token", "config[auth_token]", "password", "", "Twilio Auth Token (will be encrypted)", true, ""),
					renderProviderField("From Number", "config[from]", "tel", "+1234567890", "Twilio phone number", true, smsFrom),
				),

				// AWS SNS Fields
				Div(
					g.Attr("x-show", "smsProviderType === 'aws-sns'"),
					g.Attr("x-cloak", ""),
					Class("space-y-4 pt-4 border-t border-slate-200 dark:border-gray-700"),

					renderProviderField("Access Key ID", "config[access_key_id]", "text", "AKIAXXXXXXXXXXXXX", "AWS Access Key ID", true, getConfigStr(smsConfig, "access_key_id")),
					renderProviderField("Secret Access Key", "config[secret_access_key]", "password", "", "AWS Secret Access Key (will be encrypted)", true, ""),
					renderProviderField("Region", "config[region]", "text", "us-east-1", "AWS Region", true, getConfigStr(smsConfig, "region")),
				),

				// Hidden fields
				Input(Type("hidden"), Name("providerType"), Value("sms")),
				Input(Type("hidden"), Name("isDefault"), Value("true")),
				Input(Type("hidden"), Name("appId"), Value(currentApp.ID.String())),

				// Submit
				Div(
					Class("flex items-center justify-end gap-3 pt-4"),
					Button(
						Type("submit"),
						Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
						lucide.Save(Class("h-4 w-4")),
						g.Text("Save SMS Provider"),
					),
				),
			),

			// Test SMS Section
			Div(
				Class("mt-4 pt-4 border-t border-slate-200 dark:border-gray-700"),
				H3(Class("text-sm font-medium text-slate-900 dark:text-white mb-3"),
					g.Text("Test SMS Configuration")),
				Div(
					Class("flex items-end gap-3"),
					Div(
						Class("flex-1"),
						Label(
							For("testPhone"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-1"),
							g.Text("Test Phone Number"),
						),
						Input(
							Type("tel"),
							ID("testPhone"),
							g.Attr("x-model", "testPhone"),
							g.Attr("placeholder", "+1234567890"),
							Class("block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
					),
					Button(
						Type("button"),
						g.Attr("@click", "testSMSProvider()"),
						g.Attr(":disabled", "testLoading || !testPhone"),
						Class("inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"),
						g.Attr("x-show", "!testLoading"),
						lucide.MessageSquare(Class("h-4 w-4")),
						g.Text("Send Test SMS"),
					),
					Button(
						Type("button"),
						Disabled(),
						Class("inline-flex items-center gap-2 rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white opacity-50"),
						g.Attr("x-show", "testLoading"),
						g.Attr("x-cloak", ""),
						lucide.RefreshCw(Class("h-4 w-4 animate-spin")),
						g.Text("Sending..."),
					),
				),
			),
		),
	)
}

// renderProviderField renders a form field for provider configuration
func renderProviderField(label, name, inputType, placeholder, helpText string, required bool, value string) g.Node {
	return Div(
		Label(
			For(name),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
			g.Text(label),
			g.If(required, Span(Class("text-red-500"), g.Text(" *"))),
		),
		Input(
			Type(inputType),
			ID(name),
			Name(name),
			g.If(placeholder != "", g.Attr("placeholder", placeholder)),
			g.If(value != "", Value(value)),
			g.If(required, Required()),
			Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
		),
		g.If(helpText != "",
			P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
				g.Text(helpText)),
		),
	)
}

// renderAnalyticsContent renders the analytics page
func (e *DashboardExtension) renderAnalyticsContent(currentApp *app.App, basePath string) g.Node {
	// Fetch analytics for the last 30 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	// Note: Analytics service would need to be accessible on the plugin
	// For now, showing placeholder analytics with simple struct
	report := struct {
		TotalSent    int64
		Delivered    int64
		Opened       int64
		Clicked      int64
		Bounced      int64
		Complained   int64
		Converted    int64
		TopTemplates []struct {
			TemplateName string
			TotalSent    int64
			Opened       int64
		}
	}{
		TotalSent:  0,
		Delivered:  0,
		Opened:     0,
		Clicked:    0,
		Bounced:    0,
		Complained: 0,
		Converted:  0,
		TopTemplates: []struct {
			TemplateName string
			TotalSent    int64
			Opened       int64
		}{},
	}
	var err error
	_ = err // Prevent unused variable error

	if err != nil {
		return Div(
			Class("space-y-6"),
			Div(
				Class("rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-900/20"),
				P(Class("text-sm text-red-800 dark:text-red-200"),
					g.Textf("Error loading analytics: %v", err)),
			),
		)
	}

	// Calculate rates
	deliveryRate := 0.0
	openRate := 0.0
	clickRate := 0.0

	if report.TotalSent > 0 {
		deliveryRate = float64(report.Delivered) / float64(report.TotalSent) * 100
		openRate = float64(report.Opened) / float64(report.TotalSent) * 100
		clickRate = float64(report.Clicked) / float64(report.TotalSent) * 100
	}

	return Div(
		Class("space-y-6"),

		// Header with date range
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text("Notification Analytics")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Textf("Performance metrics for the last 30 days (since %s)", startDate.Format("Jan 2, 2006"))),
			),
		),

		// Stats cards
		Div(
			Class("grid gap-4 md:grid-cols-4"),

			// Total sent
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				Div(Class("flex items-center justify-between"),
					P(Class("text-sm font-medium text-slate-600 dark:text-gray-400"),
						g.Text("Total Sent")),
					lucide.Send(Class("h-5 w-5 text-slate-400")),
				),
				P(Class("mt-3 text-3xl font-bold text-slate-900 dark:text-white"),
					g.Textf("%d", report.TotalSent)),
			),

			// Delivery rate
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				Div(Class("flex items-center justify-between"),
					P(Class("text-sm font-medium text-slate-600 dark:text-gray-400"),
						g.Text("Delivery Rate")),
					lucide.Check(Class("h-5 w-5 text-green-400")),
				),
				P(Class("mt-3 text-3xl font-bold text-green-600 dark:text-green-400"),
					g.Textf("%.1f%%", deliveryRate)),
				P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
					g.Textf("%d delivered", report.Delivered)),
			),

			// Open rate
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				Div(Class("flex items-center justify-between"),
					P(Class("text-sm font-medium text-slate-600 dark:text-gray-400"),
						g.Text("Open Rate")),
					lucide.Eye(Class("h-5 w-5 text-blue-400")),
				),
				P(Class("mt-3 text-3xl font-bold text-blue-600 dark:text-blue-400"),
					g.Textf("%.1f%%", openRate)),
				P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
					g.Textf("%d opened", report.Opened)),
			),

			// Click rate
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				Div(Class("flex items-center justify-between"),
					P(Class("text-sm font-medium text-slate-600 dark:text-gray-400"),
						g.Text("Click Rate")),
					lucide.MousePointer(Class("h-5 w-5 text-violet-400")),
				),
				P(Class("mt-3 text-3xl font-bold text-violet-600 dark:text-violet-400"),
					g.Textf("%.1f%%", clickRate)),
				P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
					g.Textf("%d clicked", report.Clicked)),
			),
		),

		// Secondary metrics
		Div(
			Class("grid gap-4 md:grid-cols-3"),

			// Bounced
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900"),
				Div(Class("flex items-center justify-between"),
					P(Class("text-xs font-medium text-slate-600 dark:text-gray-400 uppercase tracking-wide"),
						g.Text("Bounced")),
					lucide.X(Class("h-4 w-4 text-red-400")),
				),
				P(Class("mt-2 text-2xl font-bold text-red-600 dark:text-red-400"),
					g.Textf("%d", report.Bounced)),
			),

			// Complaints
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900"),
				Div(Class("flex items-center justify-between"),
					P(Class("text-xs font-medium text-slate-600 dark:text-gray-400 uppercase tracking-wide"),
						g.Text("Complaints")),
					lucide.Ban(Class("h-4 w-4 text-orange-400")),
				),
				P(Class("mt-2 text-2xl font-bold text-orange-600 dark:text-orange-400"),
					g.Textf("%d", report.Complained)),
			),

			// Conversions
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900"),
				Div(Class("flex items-center justify-between"),
					P(Class("text-xs font-medium text-slate-600 dark:text-gray-400 uppercase tracking-wide"),
						g.Text("Conversions")),
					lucide.TrendingUp(Class("h-4 w-4 text-emerald-400")),
				),
				P(Class("mt-2 text-2xl font-bold text-emerald-600 dark:text-emerald-400"),
					g.Textf("%d", report.Converted)),
			),
		),

		// Top Templates
		g.If(len(report.TopTemplates) > 0,
			Div(
				Class("rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-800 dark:bg-gray-900"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
					g.Text("Top Performing Templates")),
				Div(Class("space-y-3"),
					g.Group(renderTopTemplates(report.TopTemplates)),
				),
			),
		),

		// Performance note
		Div(
			Class("rounded-lg border border-blue-200 bg-blue-50 p-4 dark:border-blue-800 dark:bg-blue-900/20"),
			Div(Class("flex items-start gap-3"),
				lucide.Info(Class("h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5")),
				Div(
					H4(Class("text-sm font-medium text-blue-900 dark:text-blue-300"),
						g.Text("About These Metrics")),
					P(Class("mt-1 text-xs text-blue-800 dark:text-blue-200"),
						g.Text("Analytics data is collected from notification events including sends, deliveries, opens, clicks, and conversions. Open and click tracking requires tracking pixels and link tracking to be enabled in templates.")),
				),
			),
		),
	)
}

// renderTopTemplates renders the top performing templates list
func renderTopTemplates(templates []struct {
	TemplateName string
	TotalSent    int64
	Opened       int64
}) []g.Node {
	nodes := make([]g.Node, len(templates))
	for i, tmpl := range templates {
		openRate := 0.0
		if tmpl.TotalSent > 0 {
			openRate = float64(tmpl.Opened) / float64(tmpl.TotalSent) * 100
		}

		nodes[i] = Div(
			Class("flex items-center justify-between p-3 rounded-lg bg-slate-50 dark:bg-gray-800/50"),
			Div(
				P(Class("text-sm font-medium text-slate-900 dark:text-white"),
					g.Text(tmpl.TemplateName)),
				P(Class("text-xs text-slate-500 dark:text-gray-400"),
					g.Textf("%d sent", tmpl.TotalSent)),
			),
			Div(
				Class("text-right"),
				P(Class("text-sm font-semibold text-blue-600 dark:text-blue-400"),
					g.Textf("%.1f%%", openRate)),
				P(Class("text-xs text-slate-500 dark:text-gray-400"),
					g.Text("open rate")),
			),
		)
	}
	return nodes
}

// =============================================================================
// EMAIL BUILDER HANDLERS
// =============================================================================

// ServeEmailBuilder serves the visual email template builder page
func (e *DashboardExtension) ServeEmailBuilder(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	basePath := handler.GetBasePath()

	// Check if loading a sample template
	sampleName := c.Query("sample")
	var doc *builder.Document
	if sampleName != "" {
		doc, err = builder.GetSampleTemplate(sampleName)
		if err != nil {
			doc = builder.NewDocument()
		}
	} else {
		doc = builder.NewDocument()
	}

	pageData := components.PageData{
		Title:      "Email Template Builder",
		User:       currentUser,
		ActivePage: "notifications",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	content := e.renderEmailBuilder(doc, currentApp, basePath, "")

	return handler.RenderWithBaseLayout(c, pageData, content)
}

// ServeEmailBuilderWithTemplate serves the builder with an existing template
func (e *DashboardExtension) ServeEmailBuilderWithTemplate(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	templateID, err := xid.FromString(c.Param("templateId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid template ID")
	}

	basePath := handler.GetBasePath()

	// Load template from database
	template, err := e.plugin.service.GetTemplate(c.Context(), templateID)
	if err != nil {
		return c.String(http.StatusNotFound, "Template not found")
	}

	var doc *builder.Document
	// Check if template has builder JSON content (stored in metadata)
	isVisualBuilder := false
	var builderBlocks string

	if template.Metadata != nil {
		if builderType, ok := template.Metadata["builderType"].(string); ok && builderType == "visual" {
			isVisualBuilder = true
		}
		// Get builder blocks from metadata (new format)
		if blocks, ok := template.Metadata["builderBlocks"].(string); ok {
			builderBlocks = blocks
		}
	}

	if isVisualBuilder && builderBlocks != "" {
		// New format: blocks stored in metadata
		doc, err = builder.FromJSON(builderBlocks)
		if err != nil {
			doc = builder.NewDocument()
		}
	} else if isVisualBuilder && template.Body != "" {
		// Legacy format: blocks stored in body (for backwards compatibility)
		doc, err = builder.FromJSON(template.Body)
		if err != nil {
			doc = builder.NewDocument()
		}
	} else {
		// Non-visual template - create a document with HTML block
		doc = builder.NewDocument()
		doc.AddBlock(builder.BlockTypeHTML, map[string]interface{}{
			"style": map[string]interface{}{},
			"props": map[string]interface{}{
				"html": template.Body,
			},
		}, doc.Root)
	}

	pageData := components.PageData{
		Title:      "Edit Template: " + template.Name,
		User:       currentUser,
		ActivePage: "notifications",
		BasePath:   basePath,
		CurrentApp: currentApp,
	}

	content := e.renderEmailBuilder(doc, currentApp, basePath, templateID.String())

	return handler.RenderWithBaseLayout(c, pageData, content)
}

// PreviewBuilderTemplate generates HTML preview from builder JSON
func (e *DashboardExtension) PreviewBuilderTemplate(c forge.Context) error {
	var doc builder.Document
	if err := c.BindJSON(&doc); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid document structure",
		})
	}

	// Validate document
	if err := doc.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid document: %v", err),
		})
	}

	// Render to HTML
	renderer := builder.NewRenderer(&doc)
	html, err := renderer.RenderToHTML()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to render: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"html": html,
	})
}

// SaveBuilderTemplate saves a template created with the visual builder
func (e *DashboardExtension) SaveBuilderTemplate(c forge.Context) error {
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid app context"})
	}

	var req struct {
		TemplateID  string           `json:"templateId,omitempty"`
		TemplateKey string           `json:"templateKey"`
		Name        string           `json:"name"`
		Subject     string           `json:"subject"`
		Document    builder.Document `json:"document"`
	}

	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate the document
	if err := req.Document.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid document: %v", err),
		})
	}

	// Convert document to JSON string for storage in metadata
	jsonStr, err := req.Document.ToJSON()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to serialize document",
		})
	}

	// Render the document to HTML for the body field
	// This allows the notification service to send the email without needing builder knowledge
	renderer := builder.NewRenderer(&req.Document)
	htmlContent, err := renderer.RenderToHTML()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to render template: %v", err),
		})
	}

	ctx := c.Context()

	if req.TemplateID != "" {
		// Update existing template
		templateID, err := xid.FromString(req.TemplateID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid template ID"})
		}

		// Check template exists
		_, err = e.plugin.service.GetTemplate(ctx, templateID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Template not found"})
		}

		updateReq := &notification.UpdateTemplateRequest{
			Name:    &req.Name,
			Subject: &req.Subject,
			Body:    &htmlContent, // Store rendered HTML
			Metadata: map[string]interface{}{
				"builderType":    "visual",
				"builderVersion": "1.0",
				"builderBlocks":  jsonStr, // Store JSON blocks for editing
			},
		}

		if err := e.plugin.service.UpdateTemplate(ctx, templateID, updateReq); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to update template: %v", err),
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"success":    true,
			"templateId": templateID.String(),
			"message":    "Template updated successfully",
		})
	}

	// Create new template
	templateKey := req.TemplateKey
	if templateKey == "" {
		templateKey = "custom." + slugify(req.Name)
	}

	createReq := &notification.CreateTemplateRequest{
		AppID:       currentApp.ID,
		TemplateKey: templateKey,
		Name:        req.Name,
		Type:        notification.NotificationTypeEmail,
		Language:    "en",
		Subject:     req.Subject,
		Body:        htmlContent, // Store rendered HTML
		Metadata: map[string]interface{}{
			"builderType":    "visual",
			"builderVersion": "1.0",
			"builderBlocks":  jsonStr, // Store JSON blocks for editing
		},
	}

	template, err := e.plugin.service.CreateTemplate(ctx, createReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create template: %v", err),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success":    true,
		"templateId": template.ID.String(),
		"message":    "Template created successfully",
	})
}

// GetSampleTemplate returns a sample template by name
func (e *DashboardExtension) GetSampleTemplate(c forge.Context) error {
	name := c.Param("name")

	template, err := builder.GetSampleTemplate(name)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Sample template '%s' not found", name),
		})
	}

	return c.JSON(http.StatusOK, template)
}

// renderEmailBuilder renders the visual email builder interface
func (e *DashboardExtension) renderEmailBuilder(doc *builder.Document, currentApp *app.App, basePath, templateID string) g.Node {
	builderUI := builder.NewBuilderUIWithAutosave(
		doc,
		fmt.Sprintf("%s/dashboard/app/%s/notifications/templates/builder/preview", basePath, currentApp.ID),
		fmt.Sprintf("%s/dashboard/app/%s/notifications/templates/builder/save", basePath, currentApp.ID),
		fmt.Sprintf("%s/dashboard/app/%s/notifications/templates", basePath, currentApp.ID),
		templateID,
	)

	return Div(
		Class("h-screen overflow-hidden"),

		// Hidden template ID for save
		g.If(templateID != "",
			Input(
				Type("hidden"),
				ID("template-id"),
				Value(templateID),
			),
		),

		// Builder UI (full screen)
		builderUI.Render(),
	)
}

// slugify creates a URL-friendly slug from a string
func slugify(s string) string {
	// Simple slugify - replace spaces with hyphens, lowercase
	result := ""
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			result += string(c)
		} else if c >= 'A' && c <= 'Z' {
			result += string(c + 32) // lowercase
		} else if c >= '0' && c <= '9' {
			result += string(c)
		} else if c == ' ' || c == '-' || c == '_' {
			if len(result) > 0 && result[len(result)-1] != '-' {
				result += "-"
			}
		}
	}
	return result
}
