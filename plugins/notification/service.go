package notification

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/notification"
)

// TemplateService provides template-aware notification functionality
type TemplateService struct {
	notificationSvc *notification.Service
	repo            notification.Repository
	config          Config
}

// NewTemplateService creates a new template service
func NewTemplateService(
	notificationSvc *notification.Service,
	repo notification.Repository,
	config Config,
) *TemplateService {
	return &TemplateService{
		notificationSvc: notificationSvc,
		repo:            repo,
		config:          config,
	}
}

// SendWithTemplateRequest represents a request to send a notification using a template
type SendWithTemplateRequest struct {
	AppID       xid.ID                        `json:"appId"`
	TemplateKey string                        `json:"templateKey"`
	Type        notification.NotificationType `json:"type"`
	Recipient   string                        `json:"recipient"`
	Variables   map[string]interface{}        `json:"variables"`
	Language    string                        `json:"language,omitempty"`
	Metadata    map[string]interface{}        `json:"metadata,omitempty"`
}

// SendWithTemplate sends a notification using a template
func (s *TemplateService) SendWithTemplate(ctx context.Context, req *SendWithTemplateRequest) (*notification.Notification, error) {
	// Determine language
	language := req.Language
	if language == "" {
		language = s.config.DefaultLanguage
	}

	// Find template
	schemaTemplate, err := s.repo.FindTemplateByKey(ctx, req.AppID, req.TemplateKey, string(req.Type), language)
	if err != nil {
		return nil, fmt.Errorf("failed to find template: %w", err)
	}
	if schemaTemplate == nil {
		// Template not found - try to initialize default templates for this app
		if err := s.notificationSvc.InitializeDefaultTemplates(ctx, req.AppID); err != nil {
			return nil, notification.TemplateNotFound()
		}

		// Retry finding the template after initialization
		schemaTemplate, err = s.repo.FindTemplateByKey(ctx, req.AppID, req.TemplateKey, string(req.Type), language)
		if err != nil {
			return nil, fmt.Errorf("failed to find template after initialization: %w", err)
		}
		if schemaTemplate == nil {
			return nil, notification.TemplateNotFound()
		}
	}

	// Send notification using the template
	// Use schemaTemplate.Name (not the key) since core Send() looks up by name
	return s.notificationSvc.Send(ctx, &notification.SendRequest{
		AppID:        req.AppID,
		Type:         req.Type,
		Recipient:    req.Recipient,
		TemplateName: schemaTemplate.Name,
		Variables:    req.Variables,
		Metadata:     req.Metadata,
	})
}

// SendEmail sends an email notification
func (s *TemplateService) SendEmail(ctx context.Context, appID xid.ID, templateKey, to string, variables map[string]interface{}) error {
	_, err := s.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: templateKey,
		Type:        notification.NotificationTypeEmail,
		Recipient:   to,
		Variables:   variables,
	})
	return err
}

// SendSMS sends an SMS notification
func (s *TemplateService) SendSMS(ctx context.Context, appID xid.ID, templateKey, to string, variables map[string]interface{}) error {
	_, err := s.SendWithTemplate(ctx, &SendWithTemplateRequest{
		AppID:       appID,
		TemplateKey: templateKey,
		Type:        notification.NotificationTypeSMS,
		Recipient:   to,
		Variables:   variables,
	})
	return err
}

// SendDirect sends a notification without using a template
func (s *TemplateService) SendDirect(ctx context.Context, appID xid.ID, notifType notification.NotificationType, recipient, subject, body string, metadata map[string]interface{}) (*notification.Notification, error) {
	return s.notificationSvc.Send(ctx, &notification.SendRequest{
		AppID:     appID,
		Type:      notifType,
		Recipient: recipient,
		Subject:   subject,
		Body:      body,
		Metadata:  metadata,
	})
}

// RenderTemplate renders a template with variables without sending
func (s *TemplateService) RenderTemplate(ctx context.Context, templateID xid.ID, variables map[string]interface{}) (subject, body string, err error) {
	template, err := s.notificationSvc.GetTemplate(ctx, templateID)
	if err != nil {
		return "", "", err
	}

	// Create a temporary template engine
	engine := NewTemplateEngine()

	// Render subject
	if template.Subject != "" {
		subject, err = engine.Render(template.Subject, variables)
		if err != nil {
			return "", "", fmt.Errorf("failed to render subject: %w", err)
		}
	}

	// Render body
	body, err = engine.Render(template.Body, variables)
	if err != nil {
		return "", "", fmt.Errorf("failed to render body: %w", err)
	}

	return subject, body, nil
}
