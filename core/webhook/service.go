package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/internal/crypto"
)

// Config holds the webhook service configuration
type Config struct {
	MaxRetries       int           `json:"max_retries"`
	DefaultTimeout   time.Duration `json:"default_timeout"`
	MaxDeliveryDelay time.Duration `json:"max_delivery_delay"`
	WorkerCount      int           `json:"worker_count"`
	BatchSize        int           `json:"batch_size"`
}

// Service provides webhook functionality
type Service struct {
	repo     Repository
	auditSvc *audit.Service
	config   Config
	client   *http.Client
	workers  chan struct{}
}

// NewService creates a new webhook service
func NewService(config Config, repo Repository, auditSvc *audit.Service) *Service {
	// Set defaults
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = 30 * time.Second
	}
	if config.MaxDeliveryDelay == 0 {
		config.MaxDeliveryDelay = 24 * time.Hour
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = 10
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}

	client := &http.Client{
		Timeout: config.DefaultTimeout,
	}

	service := &Service{
		repo:     repo,
		auditSvc: auditSvc,
		config:   config,
		client:   client,
		workers:  make(chan struct{}, config.WorkerCount),
	}

	// Start background workers
	go service.startDeliveryWorkers()

	return service
}

// CreateWebhook creates a new webhook subscription
func (s *Service) CreateWebhook(ctx context.Context, req *CreateWebhookRequest) (*Webhook, error) {
	// Validate app and environment context
	if req.AppID.IsNil() {
		return nil, MissingAppContext()
	}
	if req.EnvironmentID.IsNil() {
		return nil, MissingEnvironmentContext()
	}

	// Validate event types
	for _, eventType := range req.Events {
		if !IsValidEventType(eventType) {
			return nil, InvalidEventType(eventType)
		}
	}

	// Generate webhook secret
	secret, err := crypto.GenerateToken(32)
	if err != nil {
		return nil, WebhookCreationFailed(err)
	}

	webhook := &Webhook{
		ID:            xid.New(),
		AppID:         req.AppID,
		EnvironmentID: req.EnvironmentID,
		URL:           req.URL,
		Events:        req.Events,
		Secret:        secret,
		Enabled:       true,
		MaxRetries:    req.MaxRetries,
		RetryBackoff:  req.RetryBackoff,
		Headers:       req.Headers,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		FailureCount:  0,
	}

	if webhook.MaxRetries == 0 {
		webhook.MaxRetries = s.config.MaxRetries
	}
	if webhook.RetryBackoff == "" {
		webhook.RetryBackoff = RetryBackoffExponential
	}

	if err := s.repo.CreateWebhook(ctx, webhook.ToSchema()); err != nil {
		return nil, WebhookCreationFailed(err)
	}

	// Audit log
	if s.auditSvc != nil {
		userID := (*xid.ID)(nil)
		s.auditSvc.Log(ctx, userID, "webhook.create", "webhook:"+webhook.ID.String(), "", "",
			fmt.Sprintf(`{"webhook_id":"%s","app_id":"%s","environment_id":"%s","url":"%s","events":%s}`,
				webhook.ID.String(), webhook.AppID.String(), webhook.EnvironmentID.String(), webhook.URL, mustMarshal(webhook.Events)))
	}

	return webhook, nil
}

// UpdateWebhook updates an existing webhook
func (s *Service) UpdateWebhook(ctx context.Context, id xid.ID, req *UpdateWebhookRequest) (*Webhook, error) {
	schemaWebhook, err := s.repo.FindWebhookByID(ctx, id)
	if err != nil {
		return nil, WebhookNotFound()
	}

	webhook := FromSchemaWebhook(schemaWebhook)

	// Update fields
	if req.URL != nil {
		webhook.URL = *req.URL
	}
	if req.Events != nil {
		// Validate event types
		for _, eventType := range req.Events {
			if !IsValidEventType(eventType) {
				return nil, InvalidEventType(eventType)
			}
		}
		webhook.Events = req.Events
	}
	if req.Enabled != nil {
		webhook.Enabled = *req.Enabled
	}
	if req.MaxRetries != nil {
		webhook.MaxRetries = *req.MaxRetries
	}
	if req.RetryBackoff != nil {
		webhook.RetryBackoff = *req.RetryBackoff
	}
	if req.Headers != nil {
		webhook.Headers = req.Headers
	}

	webhook.UpdatedAt = time.Now()

	if err := s.repo.UpdateWebhook(ctx, webhook.ToSchema()); err != nil {
		return nil, WebhookUpdateFailed(err)
	}

	// Audit log
	if s.auditSvc != nil {
		userID := (*xid.ID)(nil)
		s.auditSvc.Log(ctx, userID, "webhook.update", "webhook:"+webhook.ID.String(), "", "",
			fmt.Sprintf(`{"webhook_id":"%s"}`, webhook.ID.String()))
	}

	return webhook, nil
}

// DeleteWebhook deletes a webhook
func (s *Service) DeleteWebhook(ctx context.Context, id xid.ID) error {
	schemaWebhook, err := s.repo.FindWebhookByID(ctx, id)
	if err != nil {
		return WebhookNotFound()
	}

	if err := s.repo.DeleteWebhook(ctx, id); err != nil {
		return WebhookDeletionFailed(err)
	}

	// Audit log
	if s.auditSvc != nil {
		userID := (*xid.ID)(nil)
		s.auditSvc.Log(ctx, userID, "webhook.delete", "webhook:"+schemaWebhook.ID.String(), "", "",
			fmt.Sprintf(`{"webhook_id":"%s"}`, schemaWebhook.ID.String()))
	}

	return nil
}

// GetWebhook retrieves a webhook by ID
func (s *Service) GetWebhook(ctx context.Context, id xid.ID) (*Webhook, error) {
	schemaWebhook, err := s.repo.FindWebhookByID(ctx, id)
	if err != nil {
		return nil, WebhookNotFound()
	}
	return FromSchemaWebhook(schemaWebhook), nil
}

// ListWebhooks lists webhooks with filtering and pagination
func (s *Service) ListWebhooks(ctx context.Context, filter *ListWebhooksFilter) (*ListWebhooksResponse, error) {
	pageResp, err := s.repo.ListWebhooks(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema webhooks to DTOs
	dtoWebhooks := FromSchemaWebhooks(pageResp.Data)

	return &ListWebhooksResponse{
		Data:       dtoWebhooks,
		Pagination: pageResp.Pagination,
	}, nil
}

// EmitEvent emits an event to all subscribed webhooks
func (s *Service) EmitEvent(ctx context.Context, appID, envID xid.ID, eventType string, data map[string]interface{}) error {
	if appID.IsNil() {
		return MissingAppContext()
	}
	if envID.IsNil() {
		return MissingEnvironmentContext()
	}
	if !IsValidEventType(eventType) {
		return InvalidEventType(eventType)
	}

	event := &Event{
		ID:            xid.New(),
		AppID:         appID,
		EnvironmentID: envID,
		Type:          eventType,
		Data:          data,
		OccurredAt:    time.Now(),
		CreatedAt:     time.Now(),
	}

	// Store the event
	if err := s.repo.CreateEvent(ctx, event.ToSchema()); err != nil {
		return EventCreationFailed(err)
	}

	// Find webhooks subscribed to this event
	schemaWebhooks, err := s.repo.FindWebhooksByAppAndEvent(ctx, appID, envID, eventType)
	if err != nil {
		return err
	}

	// Deliver to each webhook asynchronously
	for _, schemaWebhook := range schemaWebhooks {
		if schemaWebhook.Enabled {
			webhook := FromSchemaWebhook(schemaWebhook)
			go s.deliverToWebhook(ctx, webhook, event)
		}
	}

	return nil
}

// deliverToWebhook delivers an event to a specific webhook
func (s *Service) deliverToWebhook(ctx context.Context, webhook *Webhook, event *Event) {
	// Acquire worker slot
	s.workers <- struct{}{}
	defer func() { <-s.workers }()

	delivery := &Delivery{
		ID:        xid.New(),
		WebhookID: webhook.ID,
		EventID:   event.ID,
		Attempt:   1,
		Status:    DeliveryStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Store delivery record
	if err := s.repo.CreateDelivery(ctx, delivery.ToSchema()); err != nil {
		return
	}

	s.attemptDelivery(ctx, webhook, event, delivery)
}

// attemptDelivery attempts to deliver an event to a webhook
func (s *Service) attemptDelivery(ctx context.Context, webhook *Webhook, event *Event, delivery *Delivery) {
	// Prepare payload
	payload, err := json.Marshal(event)
	if err != nil {
		delivery.Status = DeliveryStatusFailed
		delivery.Error = fmt.Sprintf("failed to marshal event: %v", err)
		delivery.UpdatedAt = time.Now()
		s.repo.UpdateDelivery(ctx, delivery.ToSchema())
		return
	}

	// Generate signature
	signature := s.generateSignature(payload, webhook.Secret)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewBuffer(payload))
	if err != nil {
		delivery.Status = DeliveryStatusFailed
		delivery.Error = fmt.Sprintf("failed to create request: %v", err)
		delivery.UpdatedAt = time.Now()
		s.repo.UpdateDelivery(ctx, delivery.ToSchema())
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", "sha256="+signature)
	req.Header.Set("X-Webhook-Event", event.Type)
	req.Header.Set("X-Webhook-ID", event.ID.String())
	req.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", event.OccurredAt.Unix()))

	// Add custom headers
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		s.handleDeliveryFailure(ctx, webhook, event, delivery, 0, err.Error())
		return
	}
	defer resp.Body.Close()

	// Read response
	var responseBody bytes.Buffer
	responseBody.ReadFrom(resp.Body)

	delivery.StatusCode = resp.StatusCode
	delivery.Response = responseBody.String()
	delivery.UpdatedAt = time.Now()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success
		delivery.Status = DeliveryStatusDelivered
		now := time.Now()
		delivery.DeliveredAt = &now

		// Update webhook last delivery
		s.repo.UpdateLastDelivery(ctx, webhook.ID, now)

		// Reset failure count on success
		if webhook.FailureCount > 0 {
			s.repo.UpdateFailureCount(ctx, webhook.ID, 0)
		}
	} else {
		// Failure
		s.handleDeliveryFailure(ctx, webhook, event, delivery, resp.StatusCode, delivery.Response)
		return
	}

	s.repo.UpdateDelivery(ctx, delivery.ToSchema())
}

// handleDeliveryFailure handles a failed delivery attempt
func (s *Service) handleDeliveryFailure(ctx context.Context, webhook *Webhook, event *Event, delivery *Delivery, statusCode int, errorMsg string) {
	delivery.StatusCode = statusCode
	delivery.Error = errorMsg
	delivery.UpdatedAt = time.Now()

	if delivery.Attempt >= webhook.MaxRetries {
		// Max retries reached
		delivery.Status = DeliveryStatusFailed
		s.repo.UpdateDelivery(ctx, delivery.ToSchema())

		// Increment webhook failure count
		s.repo.UpdateFailureCount(ctx, webhook.ID, webhook.FailureCount+1)
		return
	}

	// Schedule retry
	delivery.Status = DeliveryStatusRetrying
	s.repo.UpdateDelivery(ctx, delivery.ToSchema())

	// Calculate delay
	delay := s.calculateRetryDelay(webhook.RetryBackoff, delivery.Attempt)

	// Schedule retry
	go func() {
		time.Sleep(delay)

		// Create new delivery attempt
		newDelivery := &Delivery{
			ID:        xid.New(),
			WebhookID: webhook.ID,
			EventID:   event.ID,
			Attempt:   delivery.Attempt + 1,
			Status:    DeliveryStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		s.repo.CreateDelivery(ctx, newDelivery.ToSchema())
		s.attemptDelivery(ctx, webhook, event, newDelivery)
	}()
}

// calculateRetryDelay calculates the delay for retry attempts
func (s *Service) calculateRetryDelay(backoffType string, attempt int) time.Duration {
	var delay time.Duration

	switch backoffType {
	case RetryBackoffExponential:
		delay = time.Duration(math.Pow(2, float64(attempt))) * time.Second
	case RetryBackoffLinear:
		delay = time.Duration(attempt) * 5 * time.Second
	default:
		delay = time.Duration(attempt) * 5 * time.Second
	}

	// Cap the delay
	if delay > s.config.MaxDeliveryDelay {
		delay = s.config.MaxDeliveryDelay
	}

	return delay
}

// generateSignature generates HMAC signature for webhook payload
func (s *Service) generateSignature(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature verifies webhook signature
func (s *Service) VerifySignature(payload []byte, signature, secret string) bool {
	expectedSignature := s.generateSignature(payload, secret)
	return hmac.Equal([]byte(signature), []byte("sha256="+expectedSignature))
}

// ListDeliveries lists deliveries with filtering and pagination
func (s *Service) ListDeliveries(ctx context.Context, filter *ListDeliveriesFilter) (*ListDeliveriesResponse, error) {
	pageResp, err := s.repo.ListDeliveries(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert schema deliveries to DTOs
	dtoDeliveries := FromSchemaDeliveries(pageResp.Data)

	return &ListDeliveriesResponse{
		Data:       dtoDeliveries,
		Pagination: pageResp.Pagination,
	}, nil
}

// startDeliveryWorkers starts background workers for processing failed deliveries
func (s *Service) startDeliveryWorkers() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.processFailedDeliveries()
	}
}

// processFailedDeliveries processes failed deliveries for retry
func (s *Service) processFailedDeliveries() {
	ctx := context.Background()

	schemaDeliveries, err := s.repo.FindPendingDeliveries(ctx, s.config.BatchSize)
	if err != nil {
		return
	}

	for _, schemaDelivery := range schemaDeliveries {
		schemaWebhook, err := s.repo.FindWebhookByID(ctx, schemaDelivery.WebhookID)
		if err != nil || !schemaWebhook.Enabled {
			continue
		}

		schemaEvent, err := s.repo.FindEventByID(ctx, schemaDelivery.EventID)
		if err != nil {
			continue
		}

		webhook := FromSchemaWebhook(schemaWebhook)
		event := FromSchemaEvent(schemaEvent)
		delivery := FromSchemaDelivery(schemaDelivery)

		go s.attemptDelivery(ctx, webhook, event, delivery)
	}
}

// mustMarshal marshals data to JSON, panicking on error (for internal use)
func mustMarshal(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
