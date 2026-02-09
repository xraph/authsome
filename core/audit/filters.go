package audit

import (
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// ListEventsFilter defines filters for listing audit events with pagination.
type ListEventsFilter struct {
	pagination.PaginationParams

	// ========== Full-Text Search ==========
	// Full-text search query across action, resource, metadata
	SearchQuery *string `json:"searchQuery,omitempty" query:"q"`
	// Fields to search (empty = all fields)
	SearchFields []string `json:"searchFields,omitempty" query:"search_fields"`

	// ========== Exact Match Filters ==========
	// Filter by app
	AppID *xid.ID `json:"appId,omitempty" query:"app_id"`

	// Filter by organization (user-created org)
	OrganizationID *xid.ID `json:"organizationId,omitempty" query:"organization_id"`

	// Filter by environment
	EnvironmentID *xid.ID `json:"environmentId,omitempty" query:"environment_id"`

	// Filter by user (single)
	UserID *xid.ID `json:"userId,omitempty" query:"user_id"`

	// Filter by action (single, exact match)
	Action *string `json:"action,omitempty" query:"action"`

	// Filter by resource (single, exact match)
	Resource *string `json:"resource,omitempty" query:"resource"`

	// Filter by IP address (single, exact match)
	IPAddress *string `json:"ipAddress,omitempty" query:"ip_address"`

	// Filter by source (single, exact match)
	Source *AuditSource `json:"source,omitempty" query:"source"`

	// ========== Multiple Value Filters (IN clauses) ==========
	// Filter by multiple apps
	AppIDs []xid.ID `json:"appIds,omitempty" query:"app_ids"`

	// Filter by multiple organizations
	OrganizationIDs []xid.ID `json:"organizationIds,omitempty" query:"organization_ids"`

	// Filter by multiple users
	UserIDs []xid.ID `json:"userIds,omitempty" query:"user_ids"`

	// Filter by multiple actions
	Actions []string `json:"actions,omitempty" query:"actions"`

	// Filter by multiple resources
	Resources []string `json:"resources,omitempty" query:"resources"`

	// Filter by multiple IP addresses
	IPAddresses []string `json:"ipAddresses,omitempty" query:"ip_addresses"`

	// Filter by multiple sources
	Sources []AuditSource `json:"sources,omitempty" query:"sources"`

	// ========== Pattern Matching Filters (ILIKE) ==========
	// Action pattern match (use % for wildcards)
	ActionPattern *string `json:"actionPattern,omitempty" query:"action_pattern"`

	// Resource pattern match (use % for wildcards)
	ResourcePattern *string `json:"resourcePattern,omitempty" query:"resource_pattern"`

	// ========== IP Range Filtering ==========
	// IP range in CIDR notation (e.g., "192.168.1.0/24")
	IPRange *string `json:"ipRange,omitempty" query:"ip_range"`

	// ========== Metadata Filtering ==========
	// Metadata key-value filters (for structured metadata)
	MetadataFilters []MetadataFilter `json:"metadataFilters,omitempty" query:"metadata_filters"`

	// ========== Time Range Filters ==========
	Since *time.Time `json:"since,omitempty" query:"since"`
	Until *time.Time `json:"until,omitempty" query:"until"`

	// ========== Exclusion Filters ==========
	// Exclude specific source
	ExcludeSource *AuditSource `json:"excludeSource,omitempty" query:"exclude_source"`

	// Exclude multiple sources
	ExcludeSources []AuditSource `json:"excludeSources,omitempty" query:"exclude_sources"`

	// Exclude specific action
	ExcludeAction *string `json:"excludeAction,omitempty" query:"exclude_action"`

	// Exclude multiple actions
	ExcludeActions []string `json:"excludeActions,omitempty" query:"exclude_actions"`

	// Exclude specific resource
	ExcludeResource *string `json:"excludeResource,omitempty" query:"exclude_resource"`

	// Exclude multiple resources
	ExcludeResources []string `json:"excludeResources,omitempty" query:"exclude_resources"`

	// Exclude specific user
	ExcludeUserID *xid.ID `json:"excludeUserId,omitempty" query:"exclude_user_id"`

	// Exclude multiple users
	ExcludeUserIDs []xid.ID `json:"excludeUserIds,omitempty" query:"exclude_user_ids"`

	// Exclude specific IP
	ExcludeIPAddress *string `json:"excludeIpAddress,omitempty" query:"exclude_ip_address"`

	// Exclude multiple IPs
	ExcludeIPAddresses []string `json:"excludeIpAddresses,omitempty" query:"exclude_ip_addresses"`

	// Exclude specific app
	ExcludeAppID *xid.ID `json:"excludeAppId,omitempty" query:"exclude_app_id"`

	// Exclude multiple apps
	ExcludeAppIDs []xid.ID `json:"excludeAppIds,omitempty" query:"exclude_app_ids"`

	// Exclude specific organization
	ExcludeOrganizationID *xid.ID `json:"excludeOrganizationId,omitempty" query:"exclude_organization_id"`

	// Exclude multiple organizations
	ExcludeOrganizationIDs []xid.ID `json:"excludeOrganizationIds,omitempty" query:"exclude_organization_ids"`

	// Exclude specific environment
	ExcludeEnvironmentID *xid.ID `json:"excludeEnvironmentId,omitempty" query:"exclude_environment_id"`

	// ========== Sort Order ==========
	SortBy    *string `json:"sortBy,omitempty"    query:"sort_by"`    // created_at, action, resource, rank (for search)
	SortOrder *string `json:"sortOrder,omitempty" query:"sort_order"` // asc, desc
}

// MetadataFilter defines a filter for metadata field.
type MetadataFilter struct {
	Key      string `json:"key"`      // Metadata key to filter on
	Value    any    `json:"value"`    // Value to match
	Operator string `json:"operator"` // equals, contains, exists, not_exists
}

// =============================================================================
// STATISTICS FILTERS AND TYPES
// =============================================================================

// StatisticsFilter defines filters for aggregation statistics queries.
type StatisticsFilter struct {
	// Filter by app
	AppID *xid.ID `json:"appId,omitempty"`

	// Filter by organization (user-created org)
	OrganizationID *xid.ID `json:"organizationId,omitempty"`

	// Filter by environment
	EnvironmentID *xid.ID `json:"environmentId,omitempty"`

	// Filter by user
	UserID *xid.ID `json:"userId,omitempty"`

	// Filter by action (for resource/user statistics)
	Action *string `json:"action,omitempty"`

	// Filter by resource (for action/user statistics)
	Resource *string `json:"resource,omitempty"`

	// Filter by source (single, exact match)
	Source *AuditSource `json:"source,omitempty"`

	// Filter by multiple sources
	Sources []AuditSource `json:"sources,omitempty"`

	// Time range filters
	Since *time.Time `json:"since,omitempty"`
	Until *time.Time `json:"until,omitempty"`

	// Metadata filters
	MetadataFilters []MetadataFilter `json:"metadataFilters,omitempty"`

	// ========== Exclusion Filters ==========
	// Exclude specific source
	ExcludeSource *AuditSource `json:"excludeSource,omitempty"`

	// Exclude multiple sources
	ExcludeSources []AuditSource `json:"excludeSources,omitempty"`

	// Exclude specific action
	ExcludeAction *string `json:"excludeAction,omitempty"`

	// Exclude multiple actions
	ExcludeActions []string `json:"excludeActions,omitempty"`

	// Exclude specific resource
	ExcludeResource *string `json:"excludeResource,omitempty"`

	// Exclude multiple resources
	ExcludeResources []string `json:"excludeResources,omitempty"`

	// Exclude specific user
	ExcludeUserID *xid.ID `json:"excludeUserId,omitempty"`

	// Exclude multiple users
	ExcludeUserIDs []xid.ID `json:"excludeUserIds,omitempty"`

	// Limit for top N results (default: 100)
	Limit int `json:"limit,omitempty"`
}

// ActionStatistic represents aggregated statistics for an action.
type ActionStatistic struct {
	Action        string    `json:"action"`
	Count         int64     `json:"count"`
	FirstOccurred time.Time `json:"firstOccurred"`
	LastOccurred  time.Time `json:"lastOccurred"`
}

// ResourceStatistic represents aggregated statistics for a resource.
type ResourceStatistic struct {
	Resource      string    `json:"resource"`
	Count         int64     `json:"count"`
	FirstOccurred time.Time `json:"firstOccurred"`
	LastOccurred  time.Time `json:"lastOccurred"`
}

// UserStatistic represents aggregated statistics for a user.
type UserStatistic struct {
	UserID        *xid.ID   `json:"userId"`
	Count         int64     `json:"count"`
	FirstOccurred time.Time `json:"firstOccurred"`
	LastOccurred  time.Time `json:"lastOccurred"`
}

// DeleteFilter defines filters for delete operations (subset of ListEventsFilter).
type DeleteFilter struct {
	// Filter by app
	AppID *xid.ID `json:"appId,omitempty"`

	// Filter by organization
	OrganizationID *xid.ID `json:"organizationId,omitempty"`

	// Filter by environment
	EnvironmentID *xid.ID `json:"environmentId,omitempty"`

	// Filter by user
	UserID *xid.ID `json:"userId,omitempty"`

	// Filter by action
	Action *string `json:"action,omitempty"`

	// Filter by resource
	Resource *string `json:"resource,omitempty"`

	// Filter by source
	Source *AuditSource `json:"source,omitempty"`

	// Metadata filters
	MetadataFilters []MetadataFilter `json:"metadataFilters,omitempty"`

	// ========== Exclusion Filters ==========
	// Exclude specific action
	ExcludeAction *string `json:"excludeAction,omitempty"`

	// Exclude multiple actions
	ExcludeActions []string `json:"excludeActions,omitempty"`

	// Exclude specific resource
	ExcludeResource *string `json:"excludeResource,omitempty"`

	// Exclude multiple resources
	ExcludeResources []string `json:"excludeResources,omitempty"`
}

// =============================================================================
// TIME-BASED AGGREGATION TYPES
// =============================================================================

// TimeSeriesInterval defines the grouping interval for time series data.
type TimeSeriesInterval string

const (
	// IntervalHourly groups data by hour.
	IntervalHourly TimeSeriesInterval = "hourly"
	// IntervalDaily groups data by day.
	IntervalDaily TimeSeriesInterval = "daily"
	// IntervalWeekly groups data by week.
	IntervalWeekly TimeSeriesInterval = "weekly"
	// IntervalMonthly groups data by month.
	IntervalMonthly TimeSeriesInterval = "monthly"
)

// TimeSeriesFilter extends StatisticsFilter with interval configuration.
type TimeSeriesFilter struct {
	StatisticsFilter

	// Interval for time series grouping
	Interval TimeSeriesInterval `json:"interval"`
}

// TimeSeriesPoint represents a single data point in a time series.
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

// HourStatistic represents event count for a specific hour of day.
type HourStatistic struct {
	Hour  int   `json:"hour"` // 0-23
	Count int64 `json:"count"`
}

// DayStatistic represents event count for a specific day of week.
type DayStatistic struct {
	Day      string `json:"day"`      // Monday, Tuesday, etc.
	DayIndex int    `json:"dayIndex"` // 0=Sunday, 1=Monday, etc.
	Count    int64  `json:"count"`
}

// DateStatistic represents event count for a specific date.
type DateStatistic struct {
	Date  string `json:"date"` // YYYY-MM-DD format
	Count int64  `json:"count"`
}

// =============================================================================
// IP/NETWORK AGGREGATION TYPES
// =============================================================================

// IPStatistic represents aggregated statistics for an IP address.
type IPStatistic struct {
	IPAddress     string    `json:"ipAddress"`
	Count         int64     `json:"count"`
	FirstOccurred time.Time `json:"firstOccurred"`
	LastOccurred  time.Time `json:"lastOccurred"`
}

// =============================================================================
// MULTI-DIMENSIONAL AGGREGATION TYPES
// =============================================================================

// ActionUserStatistic represents statistics for action-user combinations.
type ActionUserStatistic struct {
	Action string  `json:"action"`
	UserID *xid.ID `json:"userId"`
	Count  int64   `json:"count"`
}

// ResourceActionStatistic represents statistics for resource-action combinations.
type ResourceActionStatistic struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Count    int64  `json:"count"`
}

// ActivitySummary provides a comprehensive dashboard summary.
type ActivitySummary struct {
	// Total counts
	TotalEvents int64 `json:"totalEvents"`
	UniqueUsers int64 `json:"uniqueUsers"`
	UniqueIPs   int64 `json:"uniqueIps"`

	// Top N breakdowns
	TopActions   []*ActionStatistic   `json:"topActions"`
	TopResources []*ResourceStatistic `json:"topResources"`
	TopUsers     []*UserStatistic     `json:"topUsers"`
	TopIPs       []*IPStatistic       `json:"topIps"`

	// Temporal breakdown
	HourlyBreakdown []*HourStatistic `json:"hourlyBreakdown"`
	DailyBreakdown  []*DayStatistic  `json:"dailyBreakdown"`

	// Filter used for this summary
	Filter *StatisticsFilter `json:"filter,omitempty"`
}

// =============================================================================
// TREND ANALYSIS TYPES
// =============================================================================

// TrendData represents comparison between current and previous periods.
type TrendData struct {
	CurrentPeriod   int64   `json:"currentPeriod"`
	PreviousPeriod  int64   `json:"previousPeriod"`
	ChangeAbsolute  int64   `json:"changeAbsolute"`
	ChangePercent   float64 `json:"changePercent"`
	ChangeDirection string  `json:"changeDirection"` // "up", "down", "stable"
}

// TrendFilter extends StatisticsFilter with period configuration.
type TrendFilter struct {
	StatisticsFilter

	// Period duration for comparison (e.g., 24h, 7d, 30d)
	// Current period: Since to Until (or now if Until is nil)
	// Previous period: calculated automatically as same duration before Since
	PeriodDuration time.Duration `json:"periodDuration,omitempty"`
}

// GrowthMetrics provides growth rate analysis over multiple time windows.
type GrowthMetrics struct {
	// Growth rates as percentage change
	DailyGrowth   float64 `json:"dailyGrowth"`   // vs yesterday
	WeeklyGrowth  float64 `json:"weeklyGrowth"`  // vs last week
	MonthlyGrowth float64 `json:"monthlyGrowth"` // vs last month

	// Absolute counts for context
	TodayCount     int64 `json:"todayCount"`
	YesterdayCount int64 `json:"yesterdayCount"`
	ThisWeekCount  int64 `json:"thisWeekCount"`
	LastWeekCount  int64 `json:"lastWeekCount"`
	ThisMonthCount int64 `json:"thisMonthCount"`
	LastMonthCount int64 `json:"lastMonthCount"`
}

// =============================================================================
// RESPONSE TYPES FOR AGGREGATION METHODS
// =============================================================================

// GetTimeSeriesResponse wraps time series results.
type GetTimeSeriesResponse struct {
	Points   []*TimeSeriesPoint `json:"points"`
	Interval TimeSeriesInterval `json:"interval"`
	Total    int64              `json:"total"`
}

// GetStatisticsByHourResponse wraps hour statistics.
type GetStatisticsByHourResponse struct {
	Statistics []*HourStatistic `json:"statistics"`
	Total      int64            `json:"total"`
}

// GetStatisticsByDayResponse wraps day statistics.
type GetStatisticsByDayResponse struct {
	Statistics []*DayStatistic `json:"statistics"`
	Total      int64           `json:"total"`
}

// GetStatisticsByDateResponse wraps date statistics.
type GetStatisticsByDateResponse struct {
	Statistics []*DateStatistic `json:"statistics"`
	Total      int64            `json:"total"`
}

// GetStatisticsByIPAddressResponse wraps IP statistics.
type GetStatisticsByIPAddressResponse struct {
	Statistics []*IPStatistic `json:"statistics"`
	Total      int64          `json:"total"`
}

// GetStatisticsByActionAndUserResponse wraps action-user statistics.
type GetStatisticsByActionAndUserResponse struct {
	Statistics []*ActionUserStatistic `json:"statistics"`
	Total      int64                  `json:"total"`
}

// GetStatisticsByResourceAndActionResponse wraps resource-action statistics.
type GetStatisticsByResourceAndActionResponse struct {
	Statistics []*ResourceActionStatistic `json:"statistics"`
	Total      int64                      `json:"total"`
}

// GetTrendsResponse wraps trend analysis results.
type GetTrendsResponse struct {
	Events    *TrendData `json:"events"`
	Users     *TrendData `json:"users,omitempty"`
	Actions   *TrendData `json:"actions,omitempty"`
	Resources *TrendData `json:"resources,omitempty"`
}

// GetGrowthRateResponse wraps growth metrics.
type GetGrowthRateResponse struct {
	Metrics *GrowthMetrics    `json:"metrics"`
	Filter  *StatisticsFilter `json:"filter,omitempty"`
}

// =============================================================================
// FILTER HELPER METHODS
// =============================================================================

// ValidateExclusionFilters validates that exclusion filters don't conflict
// For now, we allow both include and exclude for same field.
func (f *ListEventsFilter) ValidateExclusionFilters() error {
	// Optional: Add validation logic if needed
	// Current behavior: Both include and exclude filters can be used together
	// The repository layer will apply: WHERE field IN (...) AND field NOT IN (...)
	return nil
}

// HasExclusionFilters returns true if any exclusion filters are set.
func (f *ListEventsFilter) HasExclusionFilters() bool {
	return f.ExcludeSource != nil ||
		len(f.ExcludeSources) > 0 ||
		f.ExcludeAction != nil ||
		len(f.ExcludeActions) > 0 ||
		f.ExcludeResource != nil ||
		len(f.ExcludeResources) > 0 ||
		f.ExcludeUserID != nil ||
		len(f.ExcludeUserIDs) > 0 ||
		f.ExcludeIPAddress != nil ||
		len(f.ExcludeIPAddresses) > 0 ||
		f.ExcludeAppID != nil ||
		len(f.ExcludeAppIDs) > 0 ||
		f.ExcludeOrganizationID != nil ||
		len(f.ExcludeOrganizationIDs) > 0 ||
		f.ExcludeEnvironmentID != nil
}

// ValidateExclusionFilters validates that exclusion filters don't conflict for StatisticsFilter.
func (f *StatisticsFilter) ValidateExclusionFilters() error {
	return nil
}

// HasExclusionFilters returns true if any exclusion filters are set for StatisticsFilter.
func (f *StatisticsFilter) HasExclusionFilters() bool {
	return f.ExcludeSource != nil ||
		len(f.ExcludeSources) > 0 ||
		f.ExcludeAction != nil ||
		len(f.ExcludeActions) > 0 ||
		f.ExcludeResource != nil ||
		len(f.ExcludeResources) > 0 ||
		f.ExcludeUserID != nil ||
		len(f.ExcludeUserIDs) > 0
}

// ValidateExclusionFilters validates that exclusion filters don't conflict for DeleteFilter.
func (f *DeleteFilter) ValidateExclusionFilters() error {
	return nil
}

// HasExclusionFilters returns true if any exclusion filters are set for DeleteFilter.
func (f *DeleteFilter) HasExclusionFilters() bool {
	return f.ExcludeAction != nil ||
		len(f.ExcludeActions) > 0 ||
		f.ExcludeResource != nil ||
		len(f.ExcludeResources) > 0
}
