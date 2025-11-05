package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
)

// Config holds the notification service configuration
type Config struct {
	DefaultProvider map[NotificationType]string `json:"default_provider"`
	RetryAttempts   int                         `json:"retry_attempts"`
	RetryDelay      time.Duration               `json:"retry_delay"`
	CleanupAfter    time.Duration               `json:"cleanup_after"`
}

// Service provides notification functionality
type Service struct {
	repo      Repository
	engine    TemplateEngine
	providers map[string]Provider
	auditSvc  *audit.Service
	config    Config
}

// NewService creates a new notification service
func NewService(
	repo Repository,
	engine TemplateEngine,
	auditSvc *audit.Service,
	cfg Config,
) *Service {
	// Set defaults
	if cfg.RetryAttempts == 0 {
		cfg.RetryAttempts = 3
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 5 * time.Minute
	}
	if cfg.CleanupAfter == 0 {
		cfg.CleanupAfter = 30 * 24 * time.Hour // 30 days
	}

	return &Service{
		repo:      repo,
		engine:    engine,
		providers: make(map[string]Provider),
		auditSvc:  auditSvc,
		config:    cfg,
	}
}

// RegisterProvider registers a notification provider
func (s *Service) RegisterProvider(provider Provider) error {
	if err := provider.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid provider config: %w", err)
	}

	s.providers[provider.ID()] = provider
	return nil
}

// CreateTemplate creates a new notification template
func (s *Service) CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*Template, error) {
	// Validate template syntax
	if err := s.engine.ValidateTemplate(req.Body); err != nil {
		return nil, fmt.Errorf("invalid template body: %w", err)
	}

	if req.Subject != "" {
		if err := s.engine.ValidateTemplate(req.Subject); err != nil {
			return nil, fmt.Errorf("invalid template subject: %w", err)
		}
	}

	// Extract variables if not provided
	if len(req.Variables) == 0 {
		vars, err := s.engine.ExtractVariables(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to extract variables: %w", err)
		}
		req.Variables = vars

		if req.Subject != "" {
			subjectVars, err := s.engine.ExtractVariables(req.Subject)
			if err != nil {
				return nil, fmt.Errorf("failed to extract subject variables: %w", err)
			}
			// Merge variables
			varMap := make(map[string]bool)
			for _, v := range req.Variables {
				varMap[v] = true
			}
			for _, v := range subjectVars {
				if !varMap[v] {
					req.Variables = append(req.Variables, v)
				}
			}
		}
	}

	// Set default language
	language := req.Language
	if language == "" {
		language = "en"
	}

	template := &Template{
		ID:             xid.New(),
		OrganizationID: req.OrganizationID,
		TemplateKey:    req.TemplateKey,
		Name:           req.Name,
		Type:           req.Type,
		Language:       language,
		Subject:        req.Subject,
		Body:           req.Body,
		Variables:      req.Variables,
		Metadata:       req.Metadata,
		Active:         true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.CreateTemplate(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Audit log
	if err := s.auditSvc.Log(ctx, nil, "template.create", "template", "", "", fmt.Sprintf("template_id=%s,name=%s", template.ID, template.Name)); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit: %v\n", err)
	}

	return template, nil
}

// GetTemplate gets a template by ID
func (s *Service) GetTemplate(ctx context.Context, id xid.ID) (*Template, error) {
	return s.repo.FindTemplateByID(ctx, id)
}

// UpdateTemplate updates a template
func (s *Service) UpdateTemplate(ctx context.Context, id xid.ID, req *UpdateTemplateRequest) error {
	// Validate template syntax if body is being updated
	if req.Body != nil {
		if err := s.engine.ValidateTemplate(*req.Body); err != nil {
			return fmt.Errorf("invalid template body: %w", err)
		}
	}

	if req.Subject != nil && *req.Subject != "" {
		if err := s.engine.ValidateTemplate(*req.Subject); err != nil {
			return fmt.Errorf("invalid template subject: %w", err)
		}
	}

	if err := s.repo.UpdateTemplate(ctx, id, req); err != nil {
		return fmt.Errorf("failed to update template: %w", err)
	}

	// Audit log
	if err := s.auditSvc.Log(ctx, nil, "template.update", "template", "", "", fmt.Sprintf("template_id=%s", id)); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit: %v\n", err)
	}

	return nil
}

// DeleteTemplate deletes a template
func (s *Service) DeleteTemplate(ctx context.Context, id xid.ID) error {
	if err := s.repo.DeleteTemplate(ctx, id); err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	// Audit log
	if err := s.auditSvc.Log(ctx, nil, "template.delete", "template", "", "", fmt.Sprintf("template_id=%s", id)); err != nil {
		// Log error but don't fail the operation
		fmt.Printf("Failed to log audit: %v\n", err)
	}

	return nil
}

// ListTemplates lists templates
func (s *Service) ListTemplates(ctx context.Context, req *ListTemplatesRequest) ([]*Template, int64, error) {
	return s.repo.ListTemplates(ctx, req)
}

// Send sends a notification
func (s *Service) Send(ctx context.Context, req *SendRequest) (*Notification, error) {
	notification := &Notification{
		ID:             xid.New(),
		OrganizationID: req.OrganizationID,
		Type:           req.Type,
		Recipient:      req.Recipient,
		Status:         NotificationStatusPending,
		Metadata:       req.Metadata,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Use template if specified
	if req.TemplateName != "" {
		template, err := s.repo.FindTemplateByName(ctx, req.OrganizationID, req.TemplateName)
		if err != nil {
			return nil, fmt.Errorf("failed to find template: %w", err)
		}
		if template == nil {
			return nil, fmt.Errorf("template not found: %s", req.TemplateName)
		}
		if !template.Active {
			return nil, fmt.Errorf("template is inactive: %s", req.TemplateName)
		}

		notification.TemplateID = &template.ID

		// Render template
		body, err := s.engine.Render(template.Body, req.Variables)
		if err != nil {
			return nil, fmt.Errorf("failed to render template body: %w", err)
		}
		notification.Body = body

		if template.Subject != "" {
			subject, err := s.engine.Render(template.Subject, req.Variables)
			if err != nil {
				return nil, fmt.Errorf("failed to render template subject: %w", err)
			}
			notification.Subject = subject
		}
	} else {
		// Use direct content
		notification.Body = req.Body
		notification.Subject = req.Subject
	}

	// Override with request values if provided
	if req.Subject != "" {
		notification.Subject = req.Subject
	}
	if req.Body != "" {
		notification.Body = req.Body
	}

	// Save notification
	if err := s.repo.CreateNotification(ctx, notification); err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// Send notification
	if err := s.sendNotification(ctx, notification); err != nil {
		// Update status to failed
		s.repo.UpdateNotificationStatus(ctx, notification.ID, NotificationStatusFailed, err.Error(), "")
		return notification, fmt.Errorf("failed to send notification: %w", err)
	}

	return notification, nil
}

// GetNotification gets a notification by ID
func (s *Service) GetNotification(ctx context.Context, id xid.ID) (*Notification, error) {
	return s.repo.FindNotificationByID(ctx, id)
}

// ListNotifications lists notifications
func (s *Service) ListNotifications(ctx context.Context, req *ListNotificationsRequest) ([]*Notification, int64, error) {
	return s.repo.ListNotifications(ctx, req)
}

// sendNotification sends a notification using the appropriate provider
func (s *Service) sendNotification(ctx context.Context, notification *Notification) error {
	// Find provider
	var provider Provider

	// Use default provider for type
	if defaultProviderID, ok := s.config.DefaultProvider[notification.Type]; ok {
		if p, exists := s.providers[defaultProviderID]; exists {
			provider = p
		}
	}

	// Fallback to first provider of the type
	if provider == nil {
		for _, p := range s.providers {
			if p.Type() == notification.Type {
				provider = p
				break
			}
		}
	}

	if provider == nil {
		return fmt.Errorf("no provider found for notification type: %s", notification.Type)
	}

	// Send notification
	if err := provider.Send(ctx, notification); err != nil {
		return err
	}

	// Update status
	now := time.Now()
	notification.Status = NotificationStatusSent
	notification.SentAt = &now
	notification.UpdatedAt = now

	return s.repo.UpdateNotificationStatus(ctx, notification.ID, NotificationStatusSent, "", "")
}

// UpdateDeliveryStatus updates the delivery status of a notification
func (s *Service) UpdateDeliveryStatus(ctx context.Context, id xid.ID, status NotificationStatus) error {
	notification, err := s.repo.FindNotificationByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find notification: %w", err)
	}
	if notification == nil {
		return fmt.Errorf("notification not found")
	}

	if status == NotificationStatusDelivered {
		return s.repo.UpdateNotificationDelivery(ctx, id, time.Now())
	}

	return s.repo.UpdateNotificationStatus(ctx, id, status, "", "")
}

// CleanupOldNotifications removes old notifications
func (s *Service) CleanupOldNotifications(ctx context.Context) error {
	cutoff := time.Now().Add(-s.config.CleanupAfter)
	return s.repo.CleanupOldNotifications(ctx, cutoff)
}
