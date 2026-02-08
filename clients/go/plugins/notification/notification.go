package notification

import (
	"context"
	"net/url"

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

// CreateTemplate CreateTemplate creates a new notification template
func (p *Plugin) CreateTemplate(ctx context.Context) (*authsome.CreateTemplateResponse, error) {
	path := "/templates"
	var result authsome.CreateTemplateResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTemplate GetTemplate retrieves a template by ID
func (p *Plugin) GetTemplate(ctx context.Context, id xid.ID) (*authsome.GetTemplateResponse, error) {
	path := "/templates/:id"
	var result authsome.GetTemplateResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListTemplates ListTemplates lists all templates with pagination
func (p *Plugin) ListTemplates(ctx context.Context) (*authsome.ListTemplatesResponse, error) {
	path := "/templates"
	var result authsome.ListTemplatesResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateTemplate UpdateTemplate updates a template
func (p *Plugin) UpdateTemplate(ctx context.Context, id xid.ID) (*authsome.UpdateTemplateResponse, error) {
	path := "/templates/:id"
	var result authsome.UpdateTemplateResponse
	err := p.client.Request(ctx, "PUT", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteTemplate DeleteTemplate deletes a template
func (p *Plugin) DeleteTemplate(ctx context.Context, id xid.ID) (*authsome.DeleteTemplateResponse, error) {
	path := "/templates/:id"
	var result authsome.DeleteTemplateResponse
	err := p.client.Request(ctx, "DELETE", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ResetTemplate ResetTemplate resets a template to default values
func (p *Plugin) ResetTemplate(ctx context.Context, id xid.ID) (*authsome.ResetTemplateResponse, error) {
	path := "/templates/:id/reset"
	var result authsome.ResetTemplateResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ResetAllTemplates ResetAllTemplates resets all templates for an app to defaults
func (p *Plugin) ResetAllTemplates(ctx context.Context) (*authsome.ResetAllTemplatesResponse, error) {
	path := "/templates/reset-all"
	var result authsome.ResetAllTemplatesResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTemplateDefaults GetTemplateDefaults returns default template metadata
func (p *Plugin) GetTemplateDefaults(ctx context.Context) (*authsome.GetTemplateDefaultsResponse, error) {
	path := "/templates/defaults"
	var result authsome.GetTemplateDefaultsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// PreviewTemplate PreviewTemplate renders a template with provided variables
func (p *Plugin) PreviewTemplate(ctx context.Context, req *authsome.PreviewTemplateRequest, id xid.ID) (*authsome.PreviewTemplateResponse, error) {
	path := "/templates/:id/preview"
	var result authsome.PreviewTemplateResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RenderTemplate RenderTemplate renders a template string with variables (no template ID required)
func (p *Plugin) RenderTemplate(ctx context.Context, req *authsome.RenderTemplateRequest) (*authsome.RenderTemplateResponse, error) {
	path := "/templates/render"
	var result authsome.RenderTemplateResponse
	err := p.client.Request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SendNotification SendNotification sends a notification
func (p *Plugin) SendNotification(ctx context.Context) (*authsome.SendNotificationResponse, error) {
	path := "/notifications/send"
	var result authsome.SendNotificationResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetNotification GetNotification retrieves a notification by ID
func (p *Plugin) GetNotification(ctx context.Context, id xid.ID) (*authsome.GetNotificationResponse, error) {
	path := "/notifications/:id"
	var result authsome.GetNotificationResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListNotifications ListNotifications lists all notifications with pagination
func (p *Plugin) ListNotifications(ctx context.Context) (*authsome.ListNotificationsResponse, error) {
	path := "/notifications"
	var result authsome.ListNotificationsResponse
	err := p.client.Request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ResendNotification ResendNotification resends a notification
func (p *Plugin) ResendNotification(ctx context.Context, id xid.ID) (*authsome.ResendNotificationResponse, error) {
	path := "/notifications/:id/resend"
	var result authsome.ResendNotificationResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// HandleWebhook HandleWebhook handles provider webhook callbacks
func (p *Plugin) HandleWebhook(ctx context.Context, provider string) (*authsome.HandleWebhookResponse, error) {
	path := "/notifications/webhook/:provider"
	var result authsome.HandleWebhookResponse
	err := p.client.Request(ctx, "POST", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

