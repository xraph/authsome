package audit

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// Repository defines persistence for audit events
type Repository interface {
	Create(ctx context.Context, e *schema.AuditEvent) error
	Get(ctx context.Context, id xid.ID) (*schema.AuditEvent, error)
	List(ctx context.Context, filter *ListEventsFilter) (*pagination.PageResponse[*schema.AuditEvent], error)
}

// Service handles audit logging
type Service struct {
	repo      Repository
	providers *ProviderRegistry
}

// NewService creates a new audit service with optional providers
func NewService(repo Repository, opts ...ServiceOption) *Service {
	cfg := &ServiceConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	svc := &Service{
		repo:      repo,
		providers: cfg.Providers,
	}

	// Initialize with null providers if not provided (backward compatibility)
	if svc.providers == nil {
		svc.providers = NewProviderRegistry()
	}

	return svc
}

// GetProviders returns the provider registry (for external use)
func (s *Service) GetProviders() *ProviderRegistry {
	return s.providers
}

// Log creates an audit event with timestamps
func (s *Service) Log(ctx context.Context, userID *xid.ID, action, resource, ip, ua, metadata string) error {
	// Extract AppID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		// Skip audit logging if AppID is not in context
		return nil
	}

	e := &Event{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		IPAddress: ip,
		UserAgent: ua,
		Metadata:  metadata,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Convert to schema and create
	if err := s.repo.Create(ctx, e.ToSchema()); err != nil {
		return AuditEventCreateFailed(err)
	}

	// Notify audit providers (non-blocking)
	if s.providers != nil {
		s.providers.NotifyAuditEvent(ctx, e)
	}

	return nil
}

// Create creates a new audit event from a request
func (s *Service) Create(ctx context.Context, req *CreateEventRequest) (*Event, error) {
	// Extract AppID from context or use from request
	appID := req.AppID
	if appID.IsNil() {
		// Try to get from context
		ctxAppID, ok := contexts.GetAppID(ctx)
		if !ok || ctxAppID.IsNil() {
			return nil, InvalidFilter("appId", "appId is required in request or context")
		}
		appID = ctxAppID
	}

	// Validate required fields
	if req.Action == "" {
		return nil, InvalidFilter("action", "action is required")
	}
	if req.Resource == "" {
		return nil, InvalidFilter("resource", "resource is required")
	}

	now := time.Now().UTC()
	event := &Event{
		ID:        xid.New(),
		AppID:     appID,
		UserID:    req.UserID,
		Action:    req.Action,
		Resource:  req.Resource,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Metadata:  req.Metadata,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Convert to schema and create
	if err := s.repo.Create(ctx, event.ToSchema()); err != nil {
		return nil, AuditEventCreateFailed(err)
	}

	// Notify audit providers (non-blocking)
	if s.providers != nil {
		s.providers.NotifyAuditEvent(ctx, event)
	}

	return event, nil
}

// Get retrieves an audit event by ID
func (s *Service) Get(ctx context.Context, req *GetEventRequest) (*Event, error) {
	schemaEvent, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return nil, QueryFailed("get", err)
	}

	if schemaEvent == nil {
		return nil, AuditEventNotFound(req.ID.String())
	}

	return FromSchemaEvent(schemaEvent), nil
}

// List returns paginated audit events with optional filters
func (s *Service) List(ctx context.Context, filter *ListEventsFilter) (*ListEventsResponse, error) {
	// Validate pagination
	if filter.Limit < 0 {
		return nil, InvalidPagination("limit cannot be negative")
	}
	if filter.Offset < 0 {
		return nil, InvalidPagination("offset cannot be negative")
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Set defaults
	if filter.Limit == 0 {
		filter.Limit = 50
	}

	// Query repository
	pageResp, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, QueryFailed("list", err)
	}

	// Convert schema events to DTOs
	events := FromSchemaEvents(pageResp.Data)

	// Return paginated response with DTOs
	return &pagination.PageResponse[*Event]{
		Data:       events,
		Pagination: pageResp.Pagination,
		Cursor:     pageResp.Cursor,
	}, nil
}
