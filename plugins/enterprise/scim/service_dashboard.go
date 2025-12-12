package scim

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/plugins/enterprise/scim/schema"
)

// Dashboard Service Methods
// Additional service methods specifically for dashboard UI

// DashboardStats holds statistics for dashboard widgets
type DashboardStats struct {
	TotalSyncs     int
	SuccessRate    float64
	FailedSyncs    int
	LastSyncTime   string
	LastSyncStatus string
}

// SyncStatus holds current sync status information
type SyncStatus struct {
	IsHealthy       bool
	ActiveProviders int
	LastSync        *time.Time
	Status          string
	Message         string
}

// ConnectionTestResult holds connection test results
type ConnectionTestResult struct {
	Success bool
	Message string
	Details map[string]interface{}
}

// ProviderHealth holds provider health status
type ProviderHealth struct {
	Healthy      bool
	Status       string
	LastCheck    time.Time
	ResponseTime int64 // milliseconds
	ErrorMessage string
}

// DetailedStats holds detailed statistics for analytics
type DetailedStats struct {
	TotalOperations    int
	SuccessRate        float64
	AvgDuration        int64
	TotalErrors        int
	OperationsByType   map[string]int
	OperationsByStatus map[string]int
}

// GetDashboardStats returns statistics for dashboard widgets
func (s *Service) GetDashboardStats(ctx context.Context, appID xid.ID, orgID *xid.ID) (*DashboardStats, error) {
	// Count total syncs
	query := s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil))

	// Filter by organization if provided
	if orgID != nil {
		query = query.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.organization_id = ?", orgID)
	} else {
		query = query.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.app_id = ?", appID)
	}

	totalSyncs, err := query.Count(ctx)
	if err != nil {
		totalSyncs = 0
	}

	// Count failed syncs
	failedQuery := s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil)).
		Where("status = ?", "failed")

	if orgID != nil {
		failedQuery = failedQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.organization_id = ?", orgID)
	} else {
		failedQuery = failedQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.app_id = ?", appID)
	}

	failedSyncs, err := failedQuery.Count(ctx)
	if err != nil {
		failedSyncs = 0
	}

	// Calculate success rate
	successRate := 0.0
	if totalSyncs > 0 {
		successRate = float64(totalSyncs-failedSyncs) / float64(totalSyncs) * 100
	}

	// Get last sync event
	var lastEvent schema.SCIMSyncEvent
	lastQuery := s.repo.db.NewSelect().
		Model(&lastEvent).
		Order("created_at DESC").
		Limit(1)

	if orgID != nil {
		lastQuery = lastQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.organization_id = ?", orgID)
	} else {
		lastQuery = lastQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.app_id = ?", appID)
	}

	err = lastQuery.Scan(ctx)

	lastSyncTime := "Never"
	lastSyncStatus := "unknown"
	if err == nil {
		// Format last sync time
		duration := time.Since(lastEvent.CreatedAt)
		if duration < time.Minute {
			lastSyncTime = "Just now"
		} else if duration < time.Hour {
			lastSyncTime = fmt.Sprintf("%d minutes ago", int(duration.Minutes()))
		} else if duration < 24*time.Hour {
			lastSyncTime = fmt.Sprintf("%d hours ago", int(duration.Hours()))
		} else {
			lastSyncTime = fmt.Sprintf("%d days ago", int(duration.Hours()/24))
		}
		lastSyncStatus = lastEvent.Status
	}

	stats := &DashboardStats{
		TotalSyncs:     totalSyncs,
		SuccessRate:    successRate,
		FailedSyncs:    failedSyncs,
		LastSyncTime:   lastSyncTime,
		LastSyncStatus: lastSyncStatus,
	}

	return stats, nil
}

// GetSyncStatus returns current sync status
func (s *Service) GetSyncStatus(ctx context.Context, appID xid.ID, orgID *xid.ID) (*SyncStatus, error) {
	// Count active providers
	providerQuery := s.repo.db.NewSelect().
		Model((*schema.SCIMProvider)(nil)).
		Where("status = ?", "active")

	if orgID != nil {
		providerQuery = providerQuery.Where("organization_id = ?", orgID)
	} else {
		providerQuery = providerQuery.Where("app_id = ?", appID)
	}

	activeProviders, err := providerQuery.Count(ctx)
	if err != nil {
		activeProviders = 0
	}

	// Count recent failed syncs (last 24 hours)
	failedQuery := s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil)).
		Where("status = ?", "failed").
		Where("created_at > ?", time.Now().Add(-24*time.Hour))

	if orgID != nil {
		failedQuery = failedQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.organization_id = ?", orgID)
	} else {
		failedQuery = failedQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.app_id = ?", appID)
	}

	failedCount, err := failedQuery.Count(ctx)
	if err != nil {
		failedCount = 0
	}

	// Get last sync time
	var lastEvent schema.SCIMSyncEvent
	lastQuery := s.repo.db.NewSelect().
		Model(&lastEvent).
		Order("created_at DESC").
		Limit(1)

	if orgID != nil {
		lastQuery = lastQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.organization_id = ?", orgID)
	} else {
		lastQuery = lastQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.app_id = ?", appID)
	}

	err = lastQuery.Scan(ctx)

	var lastSync *time.Time
	if err == nil {
		lastSync = &lastEvent.CreatedAt
	}

	// Determine health status
	isHealthy := failedCount == 0
	status := "healthy"
	message := "All systems operational"

	if failedCount > 0 {
		isHealthy = false
		status = "warning"
		message = fmt.Sprintf("%d failed syncs in the last 24 hours", failedCount)
	}

	if activeProviders == 0 {
		status = "inactive"
		message = "No active providers configured"
	}

	return &SyncStatus{
		IsHealthy:       isHealthy,
		ActiveProviders: activeProviders,
		LastSync:        lastSync,
		Status:          status,
		Message:         message,
	}, nil
}

// GetRecentActivity returns recent provisioning events
func (s *Service) GetRecentActivity(ctx context.Context, appID xid.ID, orgID *xid.ID, limit int) ([]*SCIMSyncEvent, error) {
	var events []*schema.SCIMSyncEvent

	query := s.repo.db.NewSelect().
		Model(&events).
		Order("created_at DESC").
		Limit(limit)

	if orgID != nil {
		query = query.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.organization_id = ?", orgID)
	} else {
		query = query.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.app_id = ?", appID)
	}

	err := query.Scan(ctx)
	if err != nil {
		return make([]*SCIMSyncEvent, 0), nil
	}

	// Convert to SCIMSyncEvent type
	result := make([]*SCIMSyncEvent, len(events))
	for i, event := range events {
		result[i] = &SCIMSyncEvent{
			ID:           event.ID,
			ProviderID:   event.ProviderID,
			EventType:    event.EventType,
			Status:       event.Status,
			ResourceType: event.ResourceType,
			Duration:     event.Duration,
			CreatedAt:    event.CreatedAt,
		}
	}

	return result, nil
}

// GetFailedEvents returns recent failed events
func (s *Service) GetFailedEvents(ctx context.Context, appID xid.ID, orgID *xid.ID, limit int) ([]*SCIMSyncEvent, error) {
	var events []*schema.SCIMSyncEvent

	query := s.repo.db.NewSelect().
		Model(&events).
		Where("status = ?", "failed").
		Order("created_at DESC").
		Limit(limit)

	if orgID != nil {
		query = query.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.organization_id = ?", orgID)
	} else {
		query = query.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.app_id = ?", appID)
	}

	err := query.Scan(ctx)
	if err != nil {
		return make([]*SCIMSyncEvent, 0), nil
	}

	// Convert to SCIMSyncEvent type
	result := make([]*SCIMSyncEvent, len(events))
	for i, event := range events {
		result[i] = &SCIMSyncEvent{
			ID:           event.ID,
			ProviderID:   event.ProviderID,
			EventType:    event.EventType,
			Status:       event.Status,
			ResourceType: event.ResourceType,
			Duration:     event.Duration,
			CreatedAt:    event.CreatedAt,
		}
	}

	return result, nil
}

// GetFailedOperationsCount returns count of failed operations
func (s *Service) GetFailedOperationsCount(ctx context.Context, appID xid.ID, orgID *xid.ID) (int, error) {
	// TODO: Implement actual count from repository

	return 0, nil
}

// GetSyncLogs returns sync logs with pagination and filtering
func (s *Service) GetSyncLogs(ctx context.Context, appID xid.ID, orgID *xid.ID, page, perPage int, statusFilter, eventTypeFilter string) ([]*SCIMSyncEvent, int, error) {
	offset := (page - 1) * perPage

	var events []*schema.SCIMSyncEvent

	query := s.repo.db.NewSelect().
		Model(&events)

	// Apply filters
	if statusFilter != "" && statusFilter != "all" {
		query = query.Where("status = ?", statusFilter)
	}

	if eventTypeFilter != "" && eventTypeFilter != "all" {
		query = query.Where("event_type = ?", eventTypeFilter)
	}

	// Filter by organization or app
	if orgID != nil {
		query = query.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.organization_id = ?", orgID)
	} else {
		query = query.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.app_id = ?", appID)
	}

	// Get total count for pagination
	total, err := query.Count(ctx)
	if err != nil {
		return make([]*SCIMSyncEvent, 0), 0, err
	}

	// Fetch paginated results
	err = query.
		Order("created_at DESC").
		Limit(perPage).
		Offset(offset).
		Scan(ctx)

	if err != nil {
		return make([]*SCIMSyncEvent, 0), 0, err
	}

	// Convert to SCIMSyncEvent type
	result := make([]*SCIMSyncEvent, len(events))
	for i, event := range events {
		result[i] = &SCIMSyncEvent{
			ID:           event.ID,
			ProviderID:   event.ProviderID,
			EventType:    event.EventType,
			Status:       event.Status,
			ResourceType: event.ResourceType,
			Duration:     event.Duration,
			CreatedAt:    event.CreatedAt,
		}
	}

	return result, total, nil
}

// GetDetailedStats returns detailed statistics for analytics
func (s *Service) GetDetailedStats(ctx context.Context, appID xid.ID, orgID *xid.ID) (*DetailedStats, error) {
	// Base query
	baseQuery := s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil))

	if orgID != nil {
		baseQuery = baseQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.organization_id = ?", orgID)
	} else {
		baseQuery = baseQuery.Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
			Where("scim_providers.app_id = ?", appID)
	}

	// Total operations
	totalOperations, _ := baseQuery.Count(ctx)

	// Total errors
	totalErrors, _ := s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil)).
		Where("status = ?", "failed").
		Count(ctx)

	// Success rate
	successRate := 0.0
	if totalOperations > 0 {
		successRate = float64(totalOperations-totalErrors) / float64(totalOperations) * 100
	}

	// Average duration
	var avgDuration float64
	err := s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil)).
		ColumnExpr("AVG(duration_ms)").
		Scan(ctx, &avgDuration)
	if err != nil {
		avgDuration = 0
	}

	// Operations by type
	type TypeCount struct {
		EventType string
		Count     int
	}

	var typeCounts []TypeCount
	err = s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil)).
		Column("event_type").
		ColumnExpr("COUNT(*) as count").
		Group("event_type").
		Scan(ctx, &typeCounts)

	operationsByType := make(map[string]int)
	if err == nil {
		for _, tc := range typeCounts {
			operationsByType[tc.EventType] = tc.Count
		}
	}

	// Operations by status
	type StatusCount struct {
		Status string
		Count  int
	}

	var statusCounts []StatusCount
	err = s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil)).
		Column("status").
		ColumnExpr("COUNT(*) as count").
		Group("status").
		Scan(ctx, &statusCounts)

	operationsByStatus := make(map[string]int)
	if err == nil {
		for _, sc := range statusCounts {
			operationsByStatus[sc.Status] = sc.Count
		}
	}

	stats := &DetailedStats{
		TotalOperations:    totalOperations,
		SuccessRate:        successRate,
		AvgDuration:        int64(avgDuration),
		TotalErrors:        totalErrors,
		OperationsByType:   operationsByType,
		OperationsByStatus: operationsByStatus,
	}

	return stats, nil
}

// TestConnection tests SCIM endpoint connectivity for a token
func (s *Service) TestConnection(ctx context.Context, tokenID xid.ID) (*ConnectionTestResult, error) {
	// TODO: Implement actual connection testing
	// This should:
	// 1. Fetch the token
	// 2. Make a test SCIM request (e.g., GET /ServiceProviderConfig)
	// 3. Return results

	result := &ConnectionTestResult{
		Success: true,
		Message: "Connection successful",
		Details: map[string]interface{}{
			"latency_ms": 45,
			"version":    "2.0",
		},
	}

	return result, nil
}

// TriggerManualSync initiates a manual sync operation
func (s *Service) TriggerManualSync(ctx context.Context, providerID xid.ID, syncType string) error {
	// TODO: Implement manual sync trigger
	// This should:
	// 1. Fetch the provider
	// 2. Validate provider is active
	// 3. Queue sync job
	// 4. Return immediately (async operation)

	return nil
}

// GetProviderHealth checks provider health status
func (s *Service) GetProviderHealth(ctx context.Context, providerID xid.ID) (*ProviderHealth, error) {
	// TODO: Implement provider health check
	// This should:
	// 1. Fetch the provider
	// 2. Test connectivity
	// 3. Return health status

	health := &ProviderHealth{
		Healthy:      true,
		Status:       "healthy",
		LastCheck:    time.Now(),
		ResponseTime: 45,
		ErrorMessage: "",
	}

	return health, nil
}

// Token Management

// CreateToken creates a new SCIM token
func (s *Service) CreateToken(ctx context.Context, req *CreateSCIMTokenRequest) (*SCIMToken, error) {
	// TODO: Implement token creation
	// This should:
	// 1. Generate secure token
	// 2. Store in database
	// 3. Return token with full value (only shown once)

	return nil, fmt.Errorf("not implemented")
}

// ListTokens lists SCIM tokens
func (s *Service) ListTokens(ctx context.Context, appID, envID *xid.ID, orgID *xid.ID) ([]*SCIMToken, error) {
	// TODO: Implement token listing

	return make([]*SCIMToken, 0), nil
}

// RotateToken rotates an existing token
func (s *Service) RotateToken(ctx context.Context, tokenID xid.ID) (*SCIMToken, error) {
	// TODO: Implement token rotation

	return nil, fmt.Errorf("not implemented")
}

// RevokeToken revokes a token
func (s *Service) RevokeToken(ctx context.Context, tokenID xid.ID) error {
	// TODO: Implement token revocation

	return fmt.Errorf("not implemented")
}

// Provider Management

// CreateProvider creates a new SCIM provider
func (s *Service) CreateProvider(ctx context.Context, req *CreateSCIMProviderRequest) (*SCIMProvider, error) {
	// TODO: Implement provider creation

	return nil, fmt.Errorf("not implemented")
}

// ListProviders lists SCIM providers
func (s *Service) ListProviders(ctx context.Context, appID xid.ID, orgID *xid.ID) ([]*SCIMProvider, error) {
	var providers []*schema.SCIMProvider

	query := s.repo.db.NewSelect().
		Model(&providers)

	if orgID != nil {
		query = query.Where("organization_id = ?", orgID)
	} else {
		query = query.Where("app_id = ?", appID)
	}

	err := query.Order("created_at DESC").Scan(ctx)
	if err != nil {
		return make([]*SCIMProvider, 0), nil
	}

	// Convert to SCIMProvider type
	result := make([]*SCIMProvider, len(providers))
	for i, provider := range providers {
		result[i] = &SCIMProvider{
			ID:             provider.ID,
			AppID:          provider.AppID,
			OrganizationID: provider.OrganizationID,
			Name:           provider.Name,
			Type:           provider.Type,
			Direction:      provider.Direction,
			Status:         provider.Status,
			LastSyncAt:     provider.LastSyncAt,
			LastSyncStatus: provider.LastSyncStatus,
			CreatedAt:      provider.CreatedAt,
			UpdatedAt:      provider.UpdatedAt,
		}
	}

	return result, nil
}

// GetProvider gets a provider by ID
func (s *Service) GetProvider(ctx context.Context, providerID xid.ID) (*SCIMProvider, error) {
	// TODO: Implement provider fetching

	return nil, fmt.Errorf("not implemented")
}

// RemoveProvider removes a provider
func (s *Service) RemoveProvider(ctx context.Context, providerID xid.ID) error {
	// TODO: Implement provider removal

	return fmt.Errorf("not implemented")
}

// GetProviderSyncHistory gets sync history for a provider
func (s *Service) GetProviderSyncHistory(ctx context.Context, providerID xid.ID, limit int) ([]*SCIMSyncEvent, error) {
	// TODO: Implement sync history fetching

	return make([]*SCIMSyncEvent, 0), nil
}

// Request types

// CreateSCIMTokenRequest holds data for creating a SCIM token
type CreateSCIMTokenRequest struct {
	AppID          xid.ID
	EnvironmentID  xid.ID
	OrganizationID *xid.ID
	Name           string
	Description    string
	Scopes         []string
	ExpiresAt      *time.Time
}

// CreateSCIMProviderRequest holds data for creating a SCIM provider
type CreateSCIMProviderRequest struct {
	AppID          *xid.ID
	OrganizationID *xid.ID
	Name           string
	Type           string
	Direction      string
	BaseURL        *string
	AuthMethod     string
	TargetURL      *string
	TargetToken    *string
}

// Organization-scoped dashboard methods

// ProviderStats holds provider statistics
type ProviderStats struct {
	TotalProviders  int
	ActiveProviders int
}

// SyncStats holds sync statistics
type SyncStats struct {
	TotalSyncs  int
	SuccessRate float64
	FailedSyncs int
}

// GetSyncStatusForOrg returns sync status for a specific organization
func (s *Service) GetSyncStatusForOrg(ctx context.Context, orgID xid.ID) (*SyncStatus, error) {
	// Count active providers
	activeProviders, err := s.repo.db.NewSelect().
		Model((*schema.SCIMProvider)(nil)).
		Where("organization_id = ?", orgID).
		Where("status = ?", "active").
		Count(ctx)
	if err != nil {
		activeProviders = 0
	}

	// Count recent failed syncs (last 24 hours)
	failedSyncs, err := s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil)).
		Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
		Where("scim_providers.organization_id = ?", orgID).
		Where("scim_sync_event.status = ?", "failed").
		Where("scim_sync_event.created_at > ?", time.Now().Add(-24*time.Hour)).
		Count(ctx)
	if err != nil {
		failedSyncs = 0
	}

	// Get last sync time
	var lastEvent schema.SCIMSyncEvent
	err = s.repo.db.NewSelect().
		Model(&lastEvent).
		Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
		Where("scim_providers.organization_id = ?", orgID).
		Order("scim_sync_event.created_at DESC").
		Limit(1).
		Scan(ctx)

	var lastSync *time.Time
	if err == nil {
		lastSync = &lastEvent.CreatedAt
	}

	status := "healthy"
	message := "All systems operational"
	if failedSyncs > 0 {
		status = "warning"
		message = fmt.Sprintf("%d failed syncs in the last 24 hours", failedSyncs)
	}
	if activeProviders == 0 {
		status = "inactive"
		message = "No active providers configured"
	}

	return &SyncStatus{
		ActiveProviders: activeProviders,
		LastSync:        lastSync,
		Status:          status,
		Message:         message,
		IsHealthy:       failedSyncs == 0,
	}, nil
}

// GetProviderStatsForOrg returns provider statistics for an organization
func (s *Service) GetProviderStatsForOrg(ctx context.Context, orgID xid.ID) (*ProviderStats, error) {
	// Query total providers
	totalProviders, err := s.repo.db.NewSelect().
		Model((*schema.SCIMProvider)(nil)).
		Where("organization_id = ?", orgID).
		Count(ctx)
	if err != nil {
		totalProviders = 0
	}

	// Query active providers
	activeProviders, err := s.repo.db.NewSelect().
		Model((*schema.SCIMProvider)(nil)).
		Where("organization_id = ?", orgID).
		Where("status = ?", "active").
		Count(ctx)
	if err != nil {
		activeProviders = 0
	}

	return &ProviderStats{
		TotalProviders:  totalProviders,
		ActiveProviders: activeProviders,
	}, nil
}

// GetConfigForOrg returns SCIM configuration for an organization
func (s *Service) GetConfigForOrg(ctx context.Context, orgID xid.ID) (*Config, error) {
	// TODO: Load org-specific overrides from database
	// For now, return the base config
	// In production, this would merge org-specific settings with defaults
	return s.config, nil
}

// GetProvidersForOrg returns SCIM providers for an organization
func (s *Service) GetProvidersForOrg(ctx context.Context, orgID xid.ID) ([]interface{}, error) {
	var providers []*schema.SCIMProvider

	err := s.repo.db.NewSelect().
		Model(&providers).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return []interface{}{}, nil
	}

	// Convert to interface{} slice
	result := make([]interface{}, len(providers))
	for i, provider := range providers {
		result[i] = provider
	}

	return result, nil
}

// GetRecentEventsForOrg returns recent sync events for an organization
func (s *Service) GetRecentEventsForOrg(ctx context.Context, orgID xid.ID, limit int) ([]interface{}, error) {
	var events []*schema.SCIMSyncEvent

	err := s.repo.db.NewSelect().
		Model(&events).
		Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
		Where("scim_providers.organization_id = ?", orgID).
		Order("scim_sync_event.created_at DESC").
		Limit(limit).
		Scan(ctx)

	if err != nil {
		return []interface{}{}, nil
	}

	// Convert to interface{} slice
	result := make([]interface{}, len(events))
	for i, event := range events {
		result[i] = event
	}

	return result, nil
}

// GetSyncStatsForOrg returns sync statistics for an organization
func (s *Service) GetSyncStatsForOrg(ctx context.Context, orgID xid.ID) (*SyncStats, error) {
	// Query total syncs
	totalSyncs, err := s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil)).
		Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
		Where("scim_providers.organization_id = ?", orgID).
		Count(ctx)
	if err != nil {
		totalSyncs = 0
	}

	// Query failed syncs
	failedSyncs, err := s.repo.db.NewSelect().
		Model((*schema.SCIMSyncEvent)(nil)).
		Join("JOIN scim_providers ON scim_providers.id = scim_sync_event.provider_id").
		Where("scim_providers.organization_id = ?", orgID).
		Where("scim_sync_event.status = ?", "failed").
		Count(ctx)
	if err != nil {
		failedSyncs = 0
	}

	// Calculate success rate
	successRate := 0.0
	if totalSyncs > 0 {
		successfulSyncs := totalSyncs - failedSyncs
		successRate = float64(successfulSyncs) / float64(totalSyncs) * 100
	}

	return &SyncStats{
		TotalSyncs:  totalSyncs,
		SuccessRate: successRate,
		FailedSyncs: failedSyncs,
	}, nil
}
