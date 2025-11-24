package notification

import (
	"context"

	"github.com/xraph/authsome/clients/go"
)

// Auto-generated notification plugin

// Plugin implements the notification plugin
type Plugin struct {
	client *authsome.Client
}

// NewPlugin creates a new notification plugin
func NewPlugin() *Plugin {
	return &Plugin{}
}

// ID returns the plugin identifier
func (p *Plugin) ID() string {
	return "notification"
}

// Init initializes the plugin
func (p *Plugin) Init(client *authsome.Client) error {
	p.client = client
	return nil
}

// PreviewTemplate PreviewTemplate handles template preview requests
func (p *Plugin) PreviewTemplate(ctx context.Context, req *authsome.PreviewTemplateRequest) error {
	path := "/:id/preview"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// CreateTemplate CreateTemplate creates a new notification template
func (p *Plugin) CreateTemplate(ctx context.Context) error {
	path := "/createtemplate"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetTemplate GetTemplate retrieves a template by ID
func (p *Plugin) GetTemplate(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListTemplates ListTemplates lists all templates with pagination
func (p *Plugin) ListTemplates(ctx context.Context) error {
	path := "/listtemplates"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// UpdateTemplate UpdateTemplate updates a template
func (p *Plugin) UpdateTemplate(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "PUT", path, nil, nil, false)
	return err
}

// DeleteTemplate DeleteTemplate deletes a template
func (p *Plugin) DeleteTemplate(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "DELETE", path, nil, nil, false)
	return err
}

// ResetTemplate ResetTemplate resets a template to default values
func (p *Plugin) ResetTemplate(ctx context.Context) error {
	path := "/:id/reset"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// ResetAllTemplates ResetAllTemplates resets all templates for an app to defaults
func (p *Plugin) ResetAllTemplates(ctx context.Context) error {
	path := "/reset-all"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetTemplateDefaults GetTemplateDefaults returns default template metadata
func (p *Plugin) GetTemplateDefaults(ctx context.Context) error {
	path := "/defaults"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// PreviewTemplate PreviewTemplate renders a template with provided variables
func (p *Plugin) PreviewTemplate(ctx context.Context, req *authsome.PreviewTemplateRequest) error {
	path := "/:id/preview"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// RenderTemplate RenderTemplate renders a template string with variables (no template ID required)
func (p *Plugin) RenderTemplate(ctx context.Context, req *authsome.RenderTemplateRequest) error {
	path := "/render"
	err := p.client.Request(ctx, "POST", path, req, nil, false)
	return err
}

// SendNotification SendNotification sends a notification
func (p *Plugin) SendNotification(ctx context.Context) error {
	path := "/send"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// GetNotification GetNotification retrieves a notification by ID
func (p *Plugin) GetNotification(ctx context.Context) error {
	path := "/:id"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ListNotifications ListNotifications lists all notifications with pagination
func (p *Plugin) ListNotifications(ctx context.Context) error {
	path := "/listnotifications"
	err := p.client.Request(ctx, "GET", path, nil, nil, false)
	return err
}

// ResendNotification ResendNotification resends a notification
func (p *Plugin) ResendNotification(ctx context.Context) error {
	path := "/:id/resend"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

// HandleWebhook HandleWebhook handles provider webhook callbacks
func (p *Plugin) HandleWebhook(ctx context.Context) error {
	path := "/notifications/webhook/:provider"
	err := p.client.Request(ctx, "POST", path, nil, nil, false)
	return err
}

