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
	OrganizationID string                        `json:"organization_id"`
	TemplateKey    string                        `json:"template_key"`
	Type           notification.NotificationType `json:"type"`
	Recipient      string                        `json:"recipient"`
	Variables      map[string]interface{}        `json:"variables"`
	Language       string                        `json:"language,omitempty"`
	Metadata       map[string]interface{}        `json:"metadata,omitempty"`
}

// SendWithTemplate sends a notification using a template
func (s *TemplateService) SendWithTemplate(ctx context.Context, req *SendWithTemplateRequest) (*notification.Notification, error) {
	// Determine organization ID
	orgID := req.OrganizationID
	if orgID == "" {
		orgID = "default"
	}

	// Determine language
	language := req.Language
	if language == "" {
		language = s.config.DefaultLanguage
	}

	// Find template - try org-specific first, then default
	template, err := s.findTemplate(ctx, orgID, req.TemplateKey, string(req.Type), language)
	if err != nil {
		return nil, fmt.Errorf("failed to find template: %w", err)
	}
	if template == nil {
		return nil, fmt.Errorf("template not found: %s", req.TemplateKey)
	}

	// Send notification using the template
	return s.notificationSvc.Send(ctx, &notification.SendRequest{
		OrganizationID: orgID,
		Type:           req.Type,
		Recipient:      req.Recipient,
		TemplateName:   req.TemplateKey,
		Variables:      req.Variables,
		Metadata:       req.Metadata,
	})
}

// SendEmail sends an email notification
func (s *TemplateService) SendEmail(ctx context.Context, orgID, templateKey, to string, variables map[string]interface{}) error {
	_, err := s.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    templateKey,
		Type:           notification.NotificationTypeEmail,
		Recipient:      to,
		Variables:      variables,
	})
	return err
}

// SendSMS sends an SMS notification
func (s *TemplateService) SendSMS(ctx context.Context, orgID, templateKey, to string, variables map[string]interface{}) error {
	_, err := s.SendWithTemplate(ctx, &SendWithTemplateRequest{
		OrganizationID: orgID,
		TemplateKey:    templateKey,
		Type:           notification.NotificationTypeSMS,
		Recipient:      to,
		Variables:      variables,
	})
	return err
}

// SendDirect sends a notification without using a template
func (s *TemplateService) SendDirect(ctx context.Context, orgID string, notifType notification.NotificationType, recipient, subject, body string, metadata map[string]interface{}) (*notification.Notification, error) {
	return s.notificationSvc.Send(ctx, &notification.SendRequest{
		OrganizationID: orgID,
		Type:           notifType,
		Recipient:      recipient,
		Subject:        subject,
		Body:           body,
		Metadata:       metadata,
	})
}

// findTemplate finds a template by key, checking org-specific first, then default
func (s *TemplateService) findTemplate(ctx context.Context, orgID, templateKey, notifType, language string) (*notification.Template, error) {
	// Try org-specific template first if in SaaS mode
	if s.config.AllowOrgOverrides && orgID != "default" {
		template, err := s.repo.FindTemplateByKey(ctx, orgID, templateKey, notifType, language)
		if err == nil && template != nil && template.Active {
			return template, nil
		}
	}

	// Fall back to default template
	return s.repo.FindTemplateByKey(ctx, "default", templateKey, notifType, language)
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
