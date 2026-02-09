package audit

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// Repository defines persistence for audit events.
type Repository interface {
	// Core CRUD operations
	Create(ctx context.Context, e *schema.AuditEvent) error
	Get(ctx context.Context, id xid.ID) (*schema.AuditEvent, error)
	List(ctx context.Context, filter *ListEventsFilter) (*pagination.PageResponse[*schema.AuditEvent], error)

	// Count operations
	Count(ctx context.Context, filter *ListEventsFilter) (int64, error)

	// Retention/cleanup operations
	DeleteOlderThan(ctx context.Context, filter *DeleteFilter, before time.Time) (int64, error)

	// Statistics/aggregation operations
	GetStatisticsByAction(ctx context.Context, filter *StatisticsFilter) ([]*ActionStatistic, error)
	GetStatisticsByResource(ctx context.Context, filter *StatisticsFilter) ([]*ResourceStatistic, error)
	GetStatisticsByUser(ctx context.Context, filter *StatisticsFilter) ([]*UserStatistic, error)

	// Time-based aggregation operations
	GetTimeSeries(ctx context.Context, filter *TimeSeriesFilter) ([]*TimeSeriesPoint, error)
	GetStatisticsByHour(ctx context.Context, filter *StatisticsFilter) ([]*HourStatistic, error)
	GetStatisticsByDay(ctx context.Context, filter *StatisticsFilter) ([]*DayStatistic, error)
	GetStatisticsByDate(ctx context.Context, filter *StatisticsFilter) ([]*DateStatistic, error)

	// IP/Network aggregation operations
	GetStatisticsByIPAddress(ctx context.Context, filter *StatisticsFilter) ([]*IPStatistic, error)
	GetUniqueIPCount(ctx context.Context, filter *StatisticsFilter) (int64, error)

	// Multi-dimensional aggregation operations
	GetStatisticsByActionAndUser(ctx context.Context, filter *StatisticsFilter) ([]*ActionUserStatistic, error)
	GetStatisticsByResourceAndAction(ctx context.Context, filter *StatisticsFilter) ([]*ResourceActionStatistic, error)

	// Utility operations
	GetOldestEvent(ctx context.Context, filter *ListEventsFilter) (*schema.AuditEvent, error)

	// Aggregation operations for distinct values
	GetDistinctActions(ctx context.Context, filter *AggregationFilter) ([]DistinctValue, error)
	GetDistinctSources(ctx context.Context, filter *AggregationFilter) ([]DistinctValue, error)
	GetDistinctResources(ctx context.Context, filter *AggregationFilter) ([]DistinctValue, error)
	GetDistinctUsers(ctx context.Context, filter *AggregationFilter) ([]DistinctValue, error)
	GetDistinctIPs(ctx context.Context, filter *AggregationFilter) ([]DistinctValue, error)
	GetDistinctApps(ctx context.Context, filter *AggregationFilter) ([]DistinctValue, error)
	GetDistinctOrganizations(ctx context.Context, filter *AggregationFilter) ([]DistinctValue, error)
}

// Service handles audit logging.
type Service struct {
	repo      Repository
	providers *ProviderRegistry
}

// NewService creates a new audit service with optional providers.
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

// GetProviders returns the provider registry (for external use).
func (s *Service) GetProviders() *ProviderRegistry {
	return s.providers
}

// Log creates an audit event with timestamps.
func (s *Service) Log(ctx context.Context, userID *xid.ID, action, resource, ip, ua, metadata string) error {
	// Extract AppID from context
	appID, ok := contexts.GetAppID(ctx)
	if !ok || appID.IsNil() {
		// Skip audit logging if AppID is not in context
		return nil
	}

	// Extract OrganizationID from context (optional)
	var organizationID *xid.ID
	if orgID, ok := contexts.GetOrganizationID(ctx); ok && !orgID.IsNil() {
		organizationID = &orgID
	}

	// Extract EnvironmentID from context (optional)
	var environmentID *xid.ID
	if envID, ok := contexts.GetEnvironmentID(ctx); ok && !envID.IsNil() {
		environmentID = &envID
	}

	e := &Event{
		ID:             xid.New(),
		AppID:          appID,
		OrganizationID: organizationID,
		EnvironmentID:  environmentID,
		UserID:         userID,
		Action:         action,
		Resource:       resource,
		Source:         SourceSystem, // Internal authsome calls are marked as system
		IPAddress:      ip,
		UserAgent:      ua,
		Metadata:       metadata,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
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

// Create creates a new audit event from a request.
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

	// Use OrganizationID from request (optional)
	organizationID := req.OrganizationID

	// Extract EnvironmentID from context or use from request
	environmentID := req.EnvironmentID
	if environmentID == nil || environmentID.IsNil() {
		// Try to get from context
		if envID, ok := contexts.GetEnvironmentID(ctx); ok && !envID.IsNil() {
			environmentID = &envID
		}
	}

	// Validate required fields
	if req.Action == "" {
		return nil, InvalidFilter("action", "action is required")
	}

	if req.Resource == "" {
		return nil, InvalidFilter("resource", "resource is required")
	}

	// Determine source - default to application if not provided
	source := SourceApplication

	if req.Source != nil {
		// Validate the provided source
		if !req.Source.IsValid() {
			return nil, InvalidFilter("source", "invalid source value")
		}

		source = *req.Source
	}

	now := time.Now().UTC()
	event := &Event{
		ID:             xid.New(),
		AppID:          appID,
		OrganizationID: organizationID,
		EnvironmentID:  environmentID,
		UserID:         req.UserID,
		Action:         req.Action,
		Resource:       req.Resource,
		Source:         source,
		IPAddress:      req.IPAddress,
		UserAgent:      req.UserAgent,
		Metadata:       req.Metadata,
		CreatedAt:      now,
		UpdatedAt:      now,
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

// Get retrieves an audit event by ID.
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

// List returns paginated audit events with optional filters.
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

	// Validate exclusion filters
	if err := filter.ValidateExclusionFilters(); err != nil {
		return nil, err
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

// =============================================================================
// COUNT OPERATIONS
// =============================================================================

// Count returns the count of audit events matching the filter.
func (s *Service) Count(ctx context.Context, filter *ListEventsFilter) (int64, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &ListEventsFilter{}
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return 0, InvalidTimeRange("since must be before until")
	}

	// Query repository
	count, err := s.repo.Count(ctx, filter)
	if err != nil {
		return 0, QueryFailed("count", err)
	}

	return count, nil
}

// CountEvents is an alias for Count for better readability.
func (s *Service) CountEvents(ctx context.Context, filter *ListEventsFilter) (int64, error) {
	return s.Count(ctx, filter)
}

// =============================================================================
// RETENTION/CLEANUP OPERATIONS
// =============================================================================

// DeleteOlderThan deletes audit events older than the specified time
// Returns the number of deleted events.
func (s *Service) DeleteOlderThan(ctx context.Context, filter *DeleteFilter, before time.Time) (int64, error) {
	// Validate before time is not in the future
	if before.After(time.Now()) {
		return 0, InvalidTimeRange("before time cannot be in the future")
	}

	// Validate before time is not zero
	if before.IsZero() {
		return 0, InvalidTimeRange("before time cannot be zero")
	}

	// Ensure filter is not nil
	if filter == nil {
		filter = &DeleteFilter{}
	}

	// Execute deletion
	count, err := s.repo.DeleteOlderThan(ctx, filter, before)
	if err != nil {
		return 0, QueryFailed("delete_older_than", err)
	}

	return count, nil
}

// =============================================================================
// STATISTICS/AGGREGATION OPERATIONS
// =============================================================================

// GetStatisticsByAction returns aggregated statistics grouped by action.
func (s *Service) GetStatisticsByAction(ctx context.Context, filter *StatisticsFilter) (*GetStatisticsByActionResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Set default limit
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	stats, err := s.repo.GetStatisticsByAction(ctx, filter)
	if err != nil {
		return nil, QueryFailed("statistics_by_action", err)
	}

	// Calculate total count
	var total int64
	for _, stat := range stats {
		total += stat.Count
	}

	return &GetStatisticsByActionResponse{
		Statistics: stats,
		Total:      total,
	}, nil
}

// GetStatisticsByResource returns aggregated statistics grouped by resource.
func (s *Service) GetStatisticsByResource(ctx context.Context, filter *StatisticsFilter) (*GetStatisticsByResourceResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Set default limit
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	stats, err := s.repo.GetStatisticsByResource(ctx, filter)
	if err != nil {
		return nil, QueryFailed("statistics_by_resource", err)
	}

	// Calculate total count
	var total int64
	for _, stat := range stats {
		total += stat.Count
	}

	return &GetStatisticsByResourceResponse{
		Statistics: stats,
		Total:      total,
	}, nil
}

// GetStatisticsByUser returns aggregated statistics grouped by user.
func (s *Service) GetStatisticsByUser(ctx context.Context, filter *StatisticsFilter) (*GetStatisticsByUserResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Set default limit
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	stats, err := s.repo.GetStatisticsByUser(ctx, filter)
	if err != nil {
		return nil, QueryFailed("statistics_by_user", err)
	}

	// Calculate total count
	var total int64
	for _, stat := range stats {
		total += stat.Count
	}

	return &GetStatisticsByUserResponse{
		Statistics: stats,
		Total:      total,
	}, nil
}

// =============================================================================
// TIME-BASED AGGREGATION OPERATIONS
// =============================================================================

// GetTimeSeries returns event counts over time with configurable intervals.
func (s *Service) GetTimeSeries(ctx context.Context, filter *TimeSeriesFilter) (*GetTimeSeriesResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &TimeSeriesFilter{
			Interval: IntervalDaily,
		}
	}

	// Set default interval
	if filter.Interval == "" {
		filter.Interval = IntervalDaily
	}

	// Validate interval
	switch filter.Interval {
	case IntervalHourly, IntervalDaily, IntervalWeekly, IntervalMonthly:
		// Valid
	default:
		return nil, InvalidFilter("interval", "invalid interval; must be hourly, daily, weekly, or monthly")
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	points, err := s.repo.GetTimeSeries(ctx, filter)
	if err != nil {
		return nil, QueryFailed("time_series", err)
	}

	// Calculate total count
	var total int64
	for _, point := range points {
		total += point.Count
	}

	return &GetTimeSeriesResponse{
		Points:   points,
		Interval: filter.Interval,
		Total:    total,
	}, nil
}

// GetStatisticsByHour returns event distribution by hour of day (0-23).
func (s *Service) GetStatisticsByHour(ctx context.Context, filter *StatisticsFilter) (*GetStatisticsByHourResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	stats, err := s.repo.GetStatisticsByHour(ctx, filter)
	if err != nil {
		return nil, QueryFailed("statistics_by_hour", err)
	}

	// Calculate total count
	var total int64
	for _, stat := range stats {
		total += stat.Count
	}

	return &GetStatisticsByHourResponse{
		Statistics: stats,
		Total:      total,
	}, nil
}

// GetStatisticsByDay returns event distribution by day of week.
func (s *Service) GetStatisticsByDay(ctx context.Context, filter *StatisticsFilter) (*GetStatisticsByDayResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	stats, err := s.repo.GetStatisticsByDay(ctx, filter)
	if err != nil {
		return nil, QueryFailed("statistics_by_day", err)
	}

	// Calculate total count
	var total int64
	for _, stat := range stats {
		total += stat.Count
	}

	return &GetStatisticsByDayResponse{
		Statistics: stats,
		Total:      total,
	}, nil
}

// GetStatisticsByDate returns daily event counts for a date range.
func (s *Service) GetStatisticsByDate(ctx context.Context, filter *StatisticsFilter) (*GetStatisticsByDateResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	stats, err := s.repo.GetStatisticsByDate(ctx, filter)
	if err != nil {
		return nil, QueryFailed("statistics_by_date", err)
	}

	// Calculate total count
	var total int64
	for _, stat := range stats {
		total += stat.Count
	}

	return &GetStatisticsByDateResponse{
		Statistics: stats,
		Total:      total,
	}, nil
}

// =============================================================================
// IP/NETWORK AGGREGATION OPERATIONS
// =============================================================================

// GetStatisticsByIPAddress returns event counts grouped by IP address.
func (s *Service) GetStatisticsByIPAddress(ctx context.Context, filter *StatisticsFilter) (*GetStatisticsByIPAddressResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Set default limit
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	stats, err := s.repo.GetStatisticsByIPAddress(ctx, filter)
	if err != nil {
		return nil, QueryFailed("statistics_by_ip", err)
	}

	// Calculate total count
	var total int64
	for _, stat := range stats {
		total += stat.Count
	}

	return &GetStatisticsByIPAddressResponse{
		Statistics: stats,
		Total:      total,
	}, nil
}

// GetUniqueIPCount returns the count of unique IP addresses.
func (s *Service) GetUniqueIPCount(ctx context.Context, filter *StatisticsFilter) (int64, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return 0, InvalidTimeRange("since must be before until")
	}

	// Query repository
	count, err := s.repo.GetUniqueIPCount(ctx, filter)
	if err != nil {
		return 0, QueryFailed("unique_ip_count", err)
	}

	return count, nil
}

// =============================================================================
// MULTI-DIMENSIONAL AGGREGATION OPERATIONS
// =============================================================================

// GetStatisticsByActionAndUser returns event counts grouped by action and user.
func (s *Service) GetStatisticsByActionAndUser(ctx context.Context, filter *StatisticsFilter) (*GetStatisticsByActionAndUserResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Set default limit
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	stats, err := s.repo.GetStatisticsByActionAndUser(ctx, filter)
	if err != nil {
		return nil, QueryFailed("statistics_by_action_user", err)
	}

	// Calculate total count
	var total int64
	for _, stat := range stats {
		total += stat.Count
	}

	return &GetStatisticsByActionAndUserResponse{
		Statistics: stats,
		Total:      total,
	}, nil
}

// GetStatisticsByResourceAndAction returns event counts grouped by resource and action.
func (s *Service) GetStatisticsByResourceAndAction(ctx context.Context, filter *StatisticsFilter) (*GetStatisticsByResourceAndActionResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Set default limit
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	stats, err := s.repo.GetStatisticsByResourceAndAction(ctx, filter)
	if err != nil {
		return nil, QueryFailed("statistics_by_resource_action", err)
	}

	// Calculate total count
	var total int64
	for _, stat := range stats {
		total += stat.Count
	}

	return &GetStatisticsByResourceAndActionResponse{
		Statistics: stats,
		Total:      total,
	}, nil
}

// GetActivitySummary returns a comprehensive activity summary dashboard.
func (s *Service) GetActivitySummary(ctx context.Context, filter *StatisticsFilter) (*ActivitySummary, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	// Set default limit for top N results
	if filter.Limit <= 0 {
		filter.Limit = 10
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Build summary by calling multiple aggregation methods
	summary := &ActivitySummary{
		Filter: filter,
	}

	// Get total event count
	listFilter := &ListEventsFilter{
		AppID:          filter.AppID,
		OrganizationID: filter.OrganizationID,
		EnvironmentID:  filter.EnvironmentID,
		UserID:         filter.UserID,
		Since:          filter.Since,
		Until:          filter.Until,
	}

	totalCount, err := s.Count(ctx, listFilter)
	if err != nil {
		return nil, err
	}

	summary.TotalEvents = totalCount

	// Get unique users count (from user statistics)
	userStats, err := s.repo.GetStatisticsByUser(ctx, filter)
	if err == nil {
		summary.UniqueUsers = int64(len(userStats))
		// Limit for display
		if len(userStats) > filter.Limit {
			userStats = userStats[:filter.Limit]
		}

		summary.TopUsers = userStats
	}

	// Get unique IPs count
	uniqueIPs, err := s.GetUniqueIPCount(ctx, filter)
	if err == nil {
		summary.UniqueIPs = uniqueIPs
	}

	// Get top actions
	actionStats, err := s.repo.GetStatisticsByAction(ctx, filter)
	if err == nil {
		if len(actionStats) > filter.Limit {
			actionStats = actionStats[:filter.Limit]
		}

		summary.TopActions = actionStats
	}

	// Get top resources
	resourceStats, err := s.repo.GetStatisticsByResource(ctx, filter)
	if err == nil {
		if len(resourceStats) > filter.Limit {
			resourceStats = resourceStats[:filter.Limit]
		}

		summary.TopResources = resourceStats
	}

	// Get top IPs
	ipStats, err := s.repo.GetStatisticsByIPAddress(ctx, filter)
	if err == nil {
		if len(ipStats) > filter.Limit {
			ipStats = ipStats[:filter.Limit]
		}

		summary.TopIPs = ipStats
	}

	// Get hourly breakdown
	hourStats, err := s.repo.GetStatisticsByHour(ctx, filter)
	if err == nil {
		summary.HourlyBreakdown = hourStats
	}

	// Get daily breakdown
	dayStats, err := s.repo.GetStatisticsByDay(ctx, filter)
	if err == nil {
		summary.DailyBreakdown = dayStats
	}

	return summary, nil
}

// =============================================================================
// TREND ANALYSIS OPERATIONS
// =============================================================================

// GetTrends compares event counts between current and previous periods.
func (s *Service) GetTrends(ctx context.Context, filter *TrendFilter) (*GetTrendsResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &TrendFilter{}
	}

	// Calculate periods
	now := time.Now().UTC()

	var currentEnd, currentStart, previousEnd, previousStart time.Time

	if filter.Until != nil {
		currentEnd = *filter.Until
	} else {
		currentEnd = now
	}

	if filter.Since != nil {
		currentStart = *filter.Since
	} else {
		// Default to last 24 hours
		currentStart = currentEnd.Add(-24 * time.Hour)
	}

	// Calculate period duration
	periodDuration := filter.PeriodDuration
	if periodDuration == 0 {
		periodDuration = currentEnd.Sub(currentStart)
	}

	previousEnd = currentStart
	previousStart = previousEnd.Add(-periodDuration)

	// Get current period count
	currentFilter := &ListEventsFilter{
		AppID:          filter.AppID,
		OrganizationID: filter.OrganizationID,
		EnvironmentID:  filter.EnvironmentID,
		UserID:         filter.UserID,
		Since:          &currentStart,
		Until:          &currentEnd,
	}

	currentCount, err := s.Count(ctx, currentFilter)
	if err != nil {
		return nil, err
	}

	// Get previous period count
	previousFilter := &ListEventsFilter{
		AppID:          filter.AppID,
		OrganizationID: filter.OrganizationID,
		EnvironmentID:  filter.EnvironmentID,
		UserID:         filter.UserID,
		Since:          &previousStart,
		Until:          &previousEnd,
	}

	previousCount, err := s.Count(ctx, previousFilter)
	if err != nil {
		return nil, err
	}

	// Calculate trend
	eventTrend := s.calculateTrend(currentCount, previousCount)

	return &GetTrendsResponse{
		Events: eventTrend,
	}, nil
}

// GetGrowthRate returns growth metrics across different time windows.
func (s *Service) GetGrowthRate(ctx context.Context, filter *StatisticsFilter) (*GetGrowthRateResponse, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &StatisticsFilter{}
	}

	now := time.Now().UTC()
	metrics := &GrowthMetrics{}

	// Calculate time boundaries
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	thisWeekStart := todayStart.AddDate(0, 0, -int(now.Weekday()))
	lastWeekStart := thisWeekStart.AddDate(0, 0, -7)
	lastWeekEnd := thisWeekStart

	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonthStart := thisMonthStart.AddDate(0, -1, 0)
	lastMonthEnd := thisMonthStart

	// Helper to create filter with time range
	createFilter := func(since, until time.Time) *ListEventsFilter {
		return &ListEventsFilter{
			AppID:          filter.AppID,
			OrganizationID: filter.OrganizationID,
			EnvironmentID:  filter.EnvironmentID,
			UserID:         filter.UserID,
			Since:          &since,
			Until:          &until,
		}
	}

	// Get counts for each period
	todayCount, _ := s.Count(ctx, createFilter(todayStart, now))
	yesterdayCount, _ := s.Count(ctx, createFilter(yesterdayStart, todayStart))
	thisWeekCount, _ := s.Count(ctx, createFilter(thisWeekStart, now))
	lastWeekCount, _ := s.Count(ctx, createFilter(lastWeekStart, lastWeekEnd))
	thisMonthCount, _ := s.Count(ctx, createFilter(thisMonthStart, now))
	lastMonthCount, _ := s.Count(ctx, createFilter(lastMonthStart, lastMonthEnd))

	// Store absolute counts
	metrics.TodayCount = todayCount
	metrics.YesterdayCount = yesterdayCount
	metrics.ThisWeekCount = thisWeekCount
	metrics.LastWeekCount = lastWeekCount
	metrics.ThisMonthCount = thisMonthCount
	metrics.LastMonthCount = lastMonthCount

	// Calculate growth rates
	metrics.DailyGrowth = s.calculateGrowthPercent(todayCount, yesterdayCount)
	metrics.WeeklyGrowth = s.calculateGrowthPercent(thisWeekCount, lastWeekCount)
	metrics.MonthlyGrowth = s.calculateGrowthPercent(thisMonthCount, lastMonthCount)

	return &GetGrowthRateResponse{
		Metrics: metrics,
		Filter:  filter,
	}, nil
}

// calculateTrend calculates trend data between two counts.
func (s *Service) calculateTrend(current, previous int64) *TrendData {
	trend := &TrendData{
		CurrentPeriod:  current,
		PreviousPeriod: previous,
		ChangeAbsolute: current - previous,
	}

	if previous == 0 {
		if current > 0 {
			trend.ChangePercent = 100.0
			trend.ChangeDirection = "up"
		} else {
			trend.ChangePercent = 0
			trend.ChangeDirection = "stable"
		}
	} else {
		trend.ChangePercent = float64(current-previous) / float64(previous) * 100
		if trend.ChangePercent > 1 {
			trend.ChangeDirection = "up"
		} else if trend.ChangePercent < -1 {
			trend.ChangeDirection = "down"
		} else {
			trend.ChangeDirection = "stable"
		}
	}

	return trend
}

// calculateGrowthPercent calculates percentage growth.
func (s *Service) calculateGrowthPercent(current, previous int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}

		return 0
	}

	return float64(current-previous) / float64(previous) * 100
}

// =============================================================================
// UTILITY OPERATIONS
// =============================================================================

// GetOldestEvent retrieves the oldest audit event matching the filter.
func (s *Service) GetOldestEvent(ctx context.Context, filter *ListEventsFilter) (*Event, error) {
	// Ensure filter is not nil
	if filter == nil {
		filter = &ListEventsFilter{}
	}

	// Validate time range
	if filter.Since != nil && filter.Until != nil && filter.Since.After(*filter.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Query repository
	schemaEvent, err := s.repo.GetOldestEvent(ctx, filter)
	if err != nil {
		return nil, QueryFailed("get_oldest_event", err)
	}

	if schemaEvent == nil {
		return nil, nil
	}

	return FromSchemaEvent(schemaEvent), nil
}

// =============================================================================
// AGGREGATION OPERATIONS
// =============================================================================

// GetDistinctActions returns distinct action values with counts.
func (s *Service) GetDistinctActions(ctx context.Context, filter *AggregationFilter) (*ActionAggregation, error) {
	// Set default limit
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	values, err := s.repo.GetDistinctActions(ctx, filter)
	if err != nil {
		return nil, QueryFailed("get_distinct_actions", err)
	}

	return &ActionAggregation{
		Actions: values,
		Total:   len(values),
	}, nil
}

// GetDistinctSources returns distinct source values with counts.
func (s *Service) GetDistinctSources(ctx context.Context, filter *AggregationFilter) (*SourceAggregation, error) {
	// Set default limit
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	values, err := s.repo.GetDistinctSources(ctx, filter)
	if err != nil {
		return nil, QueryFailed("get_distinct_sources", err)
	}

	return &SourceAggregation{
		Sources: values,
		Total:   len(values),
	}, nil
}

// GetDistinctResources returns distinct resource values with counts.
func (s *Service) GetDistinctResources(ctx context.Context, filter *AggregationFilter) (*ResourceAggregation, error) {
	// Set default limit
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	values, err := s.repo.GetDistinctResources(ctx, filter)
	if err != nil {
		return nil, QueryFailed("get_distinct_resources", err)
	}

	return &ResourceAggregation{
		Resources: values,
		Total:     len(values),
	}, nil
}

// GetDistinctUsers returns distinct user values with counts.
func (s *Service) GetDistinctUsers(ctx context.Context, filter *AggregationFilter) (*UserAggregation, error) {
	// Set default limit
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	values, err := s.repo.GetDistinctUsers(ctx, filter)
	if err != nil {
		return nil, QueryFailed("get_distinct_users", err)
	}

	return &UserAggregation{
		Users: values,
		Total: len(values),
	}, nil
}

// GetDistinctIPs returns distinct IP address values with counts.
func (s *Service) GetDistinctIPs(ctx context.Context, filter *AggregationFilter) (*IPAggregation, error) {
	// Set default limit
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	values, err := s.repo.GetDistinctIPs(ctx, filter)
	if err != nil {
		return nil, QueryFailed("get_distinct_ips", err)
	}

	return &IPAggregation{
		IPAddresses: values,
		Total:       len(values),
	}, nil
}

// GetDistinctApps returns distinct app values with counts.
func (s *Service) GetDistinctApps(ctx context.Context, filter *AggregationFilter) (*AppAggregation, error) {
	// Set default limit
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	values, err := s.repo.GetDistinctApps(ctx, filter)
	if err != nil {
		return nil, QueryFailed("get_distinct_apps", err)
	}

	return &AppAggregation{
		Apps:  values,
		Total: len(values),
	}, nil
}

// GetDistinctOrganizations returns distinct organization values with counts.
func (s *Service) GetDistinctOrganizations(ctx context.Context, filter *AggregationFilter) (*OrgAggregation, error) {
	// Set default limit
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	values, err := s.repo.GetDistinctOrganizations(ctx, filter)
	if err != nil {
		return nil, QueryFailed("get_distinct_organizations", err)
	}

	return &OrgAggregation{
		Organizations: values,
		Total:         len(values),
	}, nil
}

// GetAllAggregations returns all aggregations in one call.
func (s *Service) GetAllAggregations(ctx context.Context, filter *AggregationFilter) (*AllAggregations, error) {
	// Set default limit for each field
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	// Query all aggregations in parallel using error group for better error handling
	type result struct {
		actions   []DistinctValue
		sources   []DistinctValue
		resources []DistinctValue
		users     []DistinctValue
		ips       []DistinctValue
		apps      []DistinctValue
		orgs      []DistinctValue
	}

	var (
		res      result
		firstErr error
	)

	// Use channels to collect results
	actionsCh := make(chan []DistinctValue, 1)
	sourcesCh := make(chan []DistinctValue, 1)
	resourcesCh := make(chan []DistinctValue, 1)
	usersCh := make(chan []DistinctValue, 1)
	ipsCh := make(chan []DistinctValue, 1)
	appsCh := make(chan []DistinctValue, 1)
	orgsCh := make(chan []DistinctValue, 1)
	errCh := make(chan error, 7)

	// Launch goroutines for parallel queries
	go func() {
		vals, err := s.repo.GetDistinctActions(ctx, filter)
		if err != nil {
			errCh <- err

			return
		}

		actionsCh <- vals
	}()

	go func() {
		vals, err := s.repo.GetDistinctSources(ctx, filter)
		if err != nil {
			errCh <- err

			return
		}

		sourcesCh <- vals
	}()

	go func() {
		vals, err := s.repo.GetDistinctResources(ctx, filter)
		if err != nil {
			errCh <- err

			return
		}

		resourcesCh <- vals
	}()

	go func() {
		vals, err := s.repo.GetDistinctUsers(ctx, filter)
		if err != nil {
			errCh <- err

			return
		}

		usersCh <- vals
	}()

	go func() {
		vals, err := s.repo.GetDistinctIPs(ctx, filter)
		if err != nil {
			errCh <- err

			return
		}

		ipsCh <- vals
	}()

	go func() {
		vals, err := s.repo.GetDistinctApps(ctx, filter)
		if err != nil {
			errCh <- err

			return
		}

		appsCh <- vals
	}()

	go func() {
		vals, err := s.repo.GetDistinctOrganizations(ctx, filter)
		if err != nil {
			errCh <- err

			return
		}

		orgsCh <- vals
	}()

	// Collect results with timeout protection
	completed := 0
	for completed < 7 {
		select {
		case res.actions = <-actionsCh:
			completed++
		case res.sources = <-sourcesCh:
			completed++
		case res.resources = <-resourcesCh:
			completed++
		case res.users = <-usersCh:
			completed++
		case res.ips = <-ipsCh:
			completed++
		case res.apps = <-appsCh:
			completed++
		case res.orgs = <-orgsCh:
			completed++
		case err := <-errCh:
			if firstErr == nil {
				firstErr = err
			}

			completed++
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if firstErr != nil {
		return nil, QueryFailed("get_all_aggregations", firstErr)
	}

	return &AllAggregations{
		Actions:       res.actions,
		Sources:       res.sources,
		Resources:     res.resources,
		Users:         res.users,
		IPAddresses:   res.ips,
		Apps:          res.apps,
		Organizations: res.orgs,
	}, nil
}
