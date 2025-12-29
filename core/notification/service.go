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
	DefaultProvider map[NotificationType]string `json:"defaultProvider"`
	RetryAttempts   int                         `json:"retryAttempts"`
	RetryDelay      time.Duration               `json:"retryDelay"`
	CleanupAfter    time.Duration               `json:"cleanupAfter"`

	// Async processing configuration
	AsyncEnabled    bool          `json:"asyncEnabled"`    // Enable async processing for non-critical notifications
	WorkerPoolSize  int           `json:"workerPoolSize"`  // Number of workers per priority level
	QueueSize       int           `json:"queueSize"`       // Buffer size for async queues

	// Retry configuration
	RetryEnabled     bool     `json:"retryEnabled"`     // Enable retry for failed notifications
	MaxRetries       int      `json:"maxRetries"`       // Maximum retry attempts (default: 3)
	RetryBackoff     []string `json:"retryBackoff"`     // Backoff durations (default: ["1m", "5m", "15m"])
	PersistFailures  bool     `json:"persistFailures"`  // Persist permanently failed notifications to DB
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
	// Set defaults for legacy config
	if cfg.RetryAttempts == 0 {
		cfg.RetryAttempts = 3
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 5 * time.Minute
	}
	if cfg.CleanupAfter == 0 {
		cfg.CleanupAfter = 30 * 24 * time.Hour // 30 days
	}

	// Set defaults for async processing
	if cfg.WorkerPoolSize == 0 {
		cfg.WorkerPoolSize = 5
	}
	if cfg.QueueSize == 0 {
		cfg.QueueSize = 1000
	}

	// Set defaults for retry configuration
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if len(cfg.RetryBackoff) == 0 {
		cfg.RetryBackoff = []string{"1m", "5m", "15m"}
	}

	return &Service{
		repo:      repo,
		engine:    engine,
		providers: make(map[string]Provider),
		auditSvc:  auditSvc,
		config:    cfg,
	}
}

// GetDispatcherConfig returns the dispatcher configuration from service config
func (s *Service) GetDispatcherConfig() DispatcherConfig {
	return DispatcherConfig{
		AsyncEnabled:   s.config.AsyncEnabled,
		WorkerPoolSize: s.config.WorkerPoolSize,
		QueueSize:      s.config.QueueSize,
	}
}

// GetRetryConfig returns the retry configuration from service config
func (s *Service) GetRetryConfig() RetryConfig {
	backoffDurations := make([]time.Duration, 0, len(s.config.RetryBackoff))
	for _, d := range s.config.RetryBackoff {
		if duration, err := time.ParseDuration(d); err == nil {
			backoffDurations = append(backoffDurations, duration)
		}
	}
	if len(backoffDurations) == 0 {
		backoffDurations = DefaultRetryConfig().BackoffDurations
	}

	return RetryConfig{
		Enabled:          s.config.RetryEnabled,
		MaxRetries:       s.config.MaxRetries,
		BackoffDurations: backoffDurations,
		PersistFailures:  s.config.PersistFailures,
	}
}

// RegisterProvider registers a notification provider
func (s *Service) RegisterProvider(provider Provider) error {
	if err := provider.ValidateConfig(); err != nil {
		return ProviderValidationFailed(err)
	}

	s.providers[provider.ID()] = provider
	return nil
}

// =============================================================================
// TEMPLATE OPERATIONS
// =============================================================================

// CreateTemplate creates a new notification template
func (s *Service) CreateTemplate(ctx context.Context, req *CreateTemplateRequest) (*Template, error) {
	// Validate template syntax
	if err := s.engine.ValidateTemplate(req.Body); err != nil {
		return nil, TemplateRenderFailed(err)
	}

	if req.Subject != "" {
		if err := s.engine.ValidateTemplate(req.Subject); err != nil {
			return nil, TemplateRenderFailed(err)
		}
	}

	// Extract variables if not provided
	if len(req.Variables) == 0 {
		vars, err := s.engine.ExtractVariables(req.Body)
		if err != nil {
			return nil, TemplateRenderFailed(err)
		}
		req.Variables = vars

		if req.Subject != "" {
			subjectVars, err := s.engine.ExtractVariables(req.Subject)
			if err != nil {
				return nil, TemplateRenderFailed(err)
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

	now := time.Now().UTC()
	template := &Template{
		ID:          xid.New(),
		AppID:       req.AppID,
		TemplateKey: req.TemplateKey,
		Name:        req.Name,
		Type:        req.Type,
		Language:    language,
		Subject:     req.Subject,
		Body:        req.Body,
		Variables:   req.Variables,
		Metadata:    req.Metadata,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateTemplate(ctx, template.ToSchema()); err != nil {
		return nil, err
	}

	// Audit log
	if s.auditSvc != nil {
		userID := xid.NilID()
		s.auditSvc.Log(ctx, &userID, "notification_template.create", "template:"+template.ID.String(), "", "", fmt.Sprintf(`{"template_id":"%s","name":"%s"}`, template.ID, template.Name))
	}

	return template, nil
}

// GetTemplate gets a template by ID
func (s *Service) GetTemplate(ctx context.Context, id xid.ID) (*Template, error) {
	schemaTemplate, err := s.repo.FindTemplateByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if schemaTemplate == nil {
		return nil, TemplateNotFound()
	}
	return FromSchemaTemplate(schemaTemplate), nil
}

// UpdateTemplate updates a template
func (s *Service) UpdateTemplate(ctx context.Context, id xid.ID, req *UpdateTemplateRequest) error {
	// Validate template syntax if body is being updated
	if req.Body != nil {
		if err := s.engine.ValidateTemplate(*req.Body); err != nil {
			return TemplateRenderFailed(err)
		}
	}

	if req.Subject != nil && *req.Subject != "" {
		if err := s.engine.ValidateTemplate(*req.Subject); err != nil {
			return TemplateRenderFailed(err)
		}
	}

	if err := s.repo.UpdateTemplate(ctx, id, req); err != nil {
		return err
	}

	// Audit log
	if s.auditSvc != nil {
		userID := xid.NilID()
		s.auditSvc.Log(ctx, &userID, "notification_template.update", "template:"+id.String(), "", "", fmt.Sprintf(`{"template_id":"%s"}`, id))
	}

	return nil
}

// DeleteTemplate deletes a template
func (s *Service) DeleteTemplate(ctx context.Context, id xid.ID) error {
	if err := s.repo.DeleteTemplate(ctx, id); err != nil {
		return err
	}

	// Audit log
	if s.auditSvc != nil {
		userID := xid.NilID()
		s.auditSvc.Log(ctx, &userID, "notification_template.delete", "template:"+id.String(), "", "", fmt.Sprintf(`{"template_id":"%s"}`, id))
	}

	return nil
}

// ListTemplates lists templates with pagination
func (s *Service) ListTemplates(ctx context.Context, filter *ListTemplatesFilter) (*ListTemplatesResponse, error) {
	// Get paginated results from repository
	pageResp, err := s.repo.ListTemplates(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema templates to DTOs
	dtoTemplates := FromSchemaTemplates(pageResp.Data)

	// Return paginated response with DTOs
	return &ListTemplatesResponse{
		Data:       dtoTemplates,
		Pagination: pageResp.Pagination,
		Cursor:     pageResp.Cursor,
	}, nil
}

// =============================================================================
// NOTIFICATION OPERATIONS
// =============================================================================

// Send sends a notification
func (s *Service) Send(ctx context.Context, req *SendRequest) (*Notification, error) {
	now := time.Now().UTC()
	notification := &Notification{
		ID:        xid.New(),
		AppID:     req.AppID,
		Type:      req.Type,
		Recipient: req.Recipient,
		Status:    NotificationStatusPending,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Use template if specified
	if req.TemplateName != "" {
		schemaTemplate, err := s.repo.FindTemplateByName(ctx, req.AppID, req.TemplateName)
		if err != nil {
			return nil, err
		}
		if schemaTemplate == nil {
			return nil, TemplateNotFound()
		}
		if !schemaTemplate.Active {
			return nil, TemplateInactive(req.TemplateName)
		}

		notification.TemplateID = &schemaTemplate.ID

		// Render template
		body, err := s.engine.Render(schemaTemplate.Body, req.Variables)
		if err != nil {
			return nil, TemplateRenderFailed(err)
		}
		notification.Body = body

		if schemaTemplate.Subject != "" {
			subject, err := s.engine.Render(schemaTemplate.Subject, req.Variables)
			if err != nil {
				return nil, TemplateRenderFailed(err)
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
	if err := s.repo.CreateNotification(ctx, notification.ToSchema()); err != nil {
		return nil, err
	}

	// Send notification
	if err := s.sendNotification(ctx, notification); err != nil {
		// Update status to failed
		s.repo.UpdateNotificationStatus(ctx, notification.ID, NotificationStatusFailed, err.Error(), "")
		return notification, NotificationSendFailed(err)
	}

	return notification, nil
}

// GetNotification gets a notification by ID
func (s *Service) GetNotification(ctx context.Context, id xid.ID) (*Notification, error) {
	schemaNotification, err := s.repo.FindNotificationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if schemaNotification == nil {
		return nil, NotificationNotFound()
	}
	return FromSchemaNotification(schemaNotification), nil
}

// ListNotifications lists notifications with pagination
func (s *Service) ListNotifications(ctx context.Context, filter *ListNotificationsFilter) (*ListNotificationsResponse, error) {
	// Get paginated results from repository
	pageResp, err := s.repo.ListNotifications(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema notifications to DTOs
	dtoNotifications := FromSchemaNotifications(pageResp.Data)

	// Return paginated response with DTOs
	return &ListNotificationsResponse{
		Data:       dtoNotifications,
		Pagination: pageResp.Pagination,
		Cursor:     pageResp.Cursor,
	}, nil
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
		return ProviderNotConfigured(notification.Type)
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
	schemaNotification, err := s.repo.FindNotificationByID(ctx, id)
	if err != nil {
		return err
	}
	if schemaNotification == nil {
		return NotificationNotFound()
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

// GetRepository returns the repository for use by sub-services
func (s *Service) GetRepository() Repository {
	return s.repo
}

// GetTemplateEngine returns the template engine for external rendering
func (s *Service) GetTemplateEngine() TemplateEngine {
	return s.engine
}
