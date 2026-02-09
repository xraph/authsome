package repository

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// AuditRepository implements core audit repository using Bun.
type AuditRepository struct {
	db *bun.DB
}

// NewAuditRepository creates a new audit repository.
func NewAuditRepository(db *bun.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create creates a new audit event.
func (r *AuditRepository) Create(ctx context.Context, e *schema.AuditEvent) error {
	_, err := r.db.NewInsert().Model(e).Exec(ctx)

	return err
}

// Get retrieves an audit event by ID.
func (r *AuditRepository) Get(ctx context.Context, id xid.ID) (*schema.AuditEvent, error) {
	var event schema.AuditEvent

	err := r.db.NewSelect().
		Model(&event).
		Where("id = ?", id.String()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

// List returns paginated audit events with optional filters.
func (r *AuditRepository) List(ctx context.Context, filter *audit.ListEventsFilter) (*pagination.PageResponse[*schema.AuditEvent], error) {
	// Build base query
	baseQuery := r.db.NewSelect().Model((*schema.AuditEvent)(nil))

	// Apply filters
	baseQuery = r.applyFilters(baseQuery, filter)

	// Count total matching records
	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Apply sorting
	sortBy := "created_at"
	sortOrder := "DESC"

	if filter.SortBy != nil {
		sortBy = *filter.SortBy
	}

	if filter.SortOrder != nil {
		sortOrder = *filter.SortOrder
	}

	baseQuery = baseQuery.OrderExpr("? ?", bun.Ident(sortBy), bun.Safe(sortOrder))

	// Apply pagination
	baseQuery = baseQuery.Limit(filter.Limit).Offset(filter.Offset)

	// Execute query
	var events []*schema.AuditEvent
	if err := baseQuery.Scan(ctx, &events); err != nil {
		return nil, err
	}

	// Create pagination params for NewPageResponse
	params := &pagination.PaginationParams{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}

	// Return paginated response
	return pagination.NewPageResponse(events, int64(total), params), nil
}

// applyFilters applies filter conditions to the query.
func (r *AuditRepository) applyFilters(q *bun.SelectQuery, filter *audit.ListEventsFilter) *bun.SelectQuery {
	// ========== Full-Text Search ==========
	if filter.SearchQuery != nil && *filter.SearchQuery != "" {
		q = r.applyFullTextSearch(q, *filter.SearchQuery, filter.SearchFields)
	}

	// ========== App Filtering ==========
	if filter.AppID != nil {
		q = q.Where("app_id = ?", filter.AppID.String())
	}

	// ========== Organization Filtering ==========
	if filter.OrganizationID != nil {
		q = q.Where("organization_id = ?", filter.OrganizationID.String())
	}

	// ========== Environment Filtering ==========
	if filter.EnvironmentID != nil {
		q = q.Where("environment_id = ?", filter.EnvironmentID.String())
	}

	// ========== Exact Match Filters ==========
	if filter.UserID != nil {
		q = q.Where("user_id = ?", filter.UserID.String())
	}

	if filter.Action != nil {
		q = q.Where("action = ?", *filter.Action)
	}

	if filter.Resource != nil {
		q = q.Where("resource = ?", *filter.Resource)
	}

	if filter.IPAddress != nil {
		q = q.Where("ip_address = ?", *filter.IPAddress)
	}

	if filter.Source != nil {
		q = q.Where("source = ?", string(*filter.Source))
	}

	// ========== Multiple Value Filters (IN clauses) ==========
	if len(filter.AppIDs) > 0 {
		appIDStrs := make([]string, len(filter.AppIDs))
		for i, id := range filter.AppIDs {
			appIDStrs[i] = id.String()
		}

		q = q.Where("app_id IN (?)", bun.In(appIDStrs))
	}

	if len(filter.OrganizationIDs) > 0 {
		orgIDStrs := make([]string, len(filter.OrganizationIDs))
		for i, id := range filter.OrganizationIDs {
			orgIDStrs[i] = id.String()
		}

		q = q.Where("organization_id IN (?)", bun.In(orgIDStrs))
	}

	if len(filter.UserIDs) > 0 {
		userIDStrs := make([]string, len(filter.UserIDs))
		for i, id := range filter.UserIDs {
			userIDStrs[i] = id.String()
		}

		q = q.Where("user_id IN (?)", bun.In(userIDStrs))
	}

	if len(filter.Actions) > 0 {
		q = q.Where("action IN (?)", bun.In(filter.Actions))
	}

	if len(filter.Resources) > 0 {
		q = q.Where("resource IN (?)", bun.In(filter.Resources))
	}

	if len(filter.IPAddresses) > 0 {
		q = q.Where("ip_address IN (?)", bun.In(filter.IPAddresses))
	}

	if len(filter.Sources) > 0 {
		sourceStrs := make([]string, len(filter.Sources))
		for i, source := range filter.Sources {
			sourceStrs[i] = string(source)
		}

		q = q.Where("source IN (?)", bun.In(sourceStrs))
	}

	// ========== Pattern Matching (ILIKE) ==========
	if filter.ActionPattern != nil && *filter.ActionPattern != "" {
		q = q.Where("action ILIKE ?", *filter.ActionPattern)
	}

	if filter.ResourcePattern != nil && *filter.ResourcePattern != "" {
		q = q.Where("resource ILIKE ?", *filter.ResourcePattern)
	}

	// ========== IP Range Filtering (CIDR) ==========
	if filter.IPRange != nil && *filter.IPRange != "" {
		// PostgreSQL inet operator for CIDR matching
		q = q.Where("ip_address::inet <<= ?::inet", *filter.IPRange)
	}

	// ========== Metadata Filtering ==========
	if len(filter.MetadataFilters) > 0 {
		q = r.applyMetadataFilters(q, filter.MetadataFilters)
	}

	// ========== Time Range Filters ==========
	if filter.Since != nil {
		q = q.Where("created_at >= ?", *filter.Since)
	}

	if filter.Until != nil {
		q = q.Where("created_at <= ?", *filter.Until)
	}

	// ========== Exclusion Filters ==========
	// Exclude source (single)
	if filter.ExcludeSource != nil {
		q = q.Where("source != ?", string(*filter.ExcludeSource))
	}

	// Exclude sources (multiple)
	if len(filter.ExcludeSources) > 0 {
		sourceStrs := make([]string, len(filter.ExcludeSources))
		for i, source := range filter.ExcludeSources {
			sourceStrs[i] = string(source)
		}

		q = q.Where("source NOT IN (?)", bun.In(sourceStrs))
	}

	// Exclude action (single)
	if filter.ExcludeAction != nil {
		q = q.Where("action != ?", *filter.ExcludeAction)
	}

	// Exclude actions (multiple)
	if len(filter.ExcludeActions) > 0 {
		q = q.Where("action NOT IN (?)", bun.In(filter.ExcludeActions))
	}

	// Exclude resource (single)
	if filter.ExcludeResource != nil {
		q = q.Where("resource != ?", *filter.ExcludeResource)
	}

	// Exclude resources (multiple)
	if len(filter.ExcludeResources) > 0 {
		q = q.Where("resource NOT IN (?)", bun.In(filter.ExcludeResources))
	}

	// Exclude user (single)
	if filter.ExcludeUserID != nil {
		q = q.Where("user_id != ?", filter.ExcludeUserID.String())
	}

	// Exclude users (multiple)
	if len(filter.ExcludeUserIDs) > 0 {
		userIDStrs := make([]string, len(filter.ExcludeUserIDs))
		for i, id := range filter.ExcludeUserIDs {
			userIDStrs[i] = id.String()
		}

		q = q.Where("user_id NOT IN (?)", bun.In(userIDStrs))
	}

	// Exclude IP address (single)
	if filter.ExcludeIPAddress != nil {
		q = q.Where("ip_address != ?", *filter.ExcludeIPAddress)
	}

	// Exclude IP addresses (multiple)
	if len(filter.ExcludeIPAddresses) > 0 {
		q = q.Where("ip_address NOT IN (?)", bun.In(filter.ExcludeIPAddresses))
	}

	// Exclude app (single)
	if filter.ExcludeAppID != nil {
		q = q.Where("app_id != ?", filter.ExcludeAppID.String())
	}

	// Exclude apps (multiple)
	if len(filter.ExcludeAppIDs) > 0 {
		appIDStrs := make([]string, len(filter.ExcludeAppIDs))
		for i, id := range filter.ExcludeAppIDs {
			appIDStrs[i] = id.String()
		}

		q = q.Where("app_id NOT IN (?)", bun.In(appIDStrs))
	}

	// Exclude organization (single)
	if filter.ExcludeOrganizationID != nil {
		q = q.Where("organization_id != ?", filter.ExcludeOrganizationID.String())
	}

	// Exclude organizations (multiple)
	if len(filter.ExcludeOrganizationIDs) > 0 {
		orgIDStrs := make([]string, len(filter.ExcludeOrganizationIDs))
		for i, id := range filter.ExcludeOrganizationIDs {
			orgIDStrs[i] = id.String()
		}

		q = q.Where("organization_id NOT IN (?)", bun.In(orgIDStrs))
	}

	// Exclude environment (single)
	if filter.ExcludeEnvironmentID != nil {
		q = q.Where("environment_id != ?", filter.ExcludeEnvironmentID.String())
	}

	return q
}

// applyFullTextSearch applies PostgreSQL full-text search.
func (r *AuditRepository) applyFullTextSearch(q *bun.SelectQuery, searchQuery string, fields []string) *bun.SelectQuery {
	// Build search vector based on fields
	var searchVector string
	if len(fields) == 0 || contains(fields, "all") {
		// Search all fields
		searchVector = "to_tsvector('english', action || ' ' || resource || ' ' || COALESCE(metadata, '') || ' ' || COALESCE(user_agent, ''))"
	} else {
		// Search specific fields
		vectors := make([]string, 0, len(fields))
		for _, field := range fields {
			switch field {
			case "action":
				vectors = append(vectors, "to_tsvector('english', action)")
			case "resource":
				vectors = append(vectors, "to_tsvector('english', resource)")
			case "metadata":
				vectors = append(vectors, "to_tsvector('english', COALESCE(metadata, ''))")
			case "user_agent":
				vectors = append(vectors, "to_tsvector('english', COALESCE(user_agent, ''))")
			}
		}

		if len(vectors) > 0 {
			searchVector = "(" + vectors[0]
			var searchVectorSb325 strings.Builder
			for i := 1; i < len(vectors); i++ {
				searchVectorSb325.WriteString(" || " + vectors[i])
			}
			searchVector += searchVectorSb325.String()

			searchVector += ")"
		}
	}

	if searchVector != "" {
		// Use websearch_to_tsquery for natural language query parsing
		q = q.Where(searchVector+" @@ websearch_to_tsquery('english', ?)", searchQuery)
	}

	return q
}

// applyMetadataFilters applies metadata JSON filters.
func (r *AuditRepository) applyMetadataFilters(q *bun.SelectQuery, filters []audit.MetadataFilter) *bun.SelectQuery {
	for _, filter := range filters {
		switch filter.Operator {
		case "exists":
			// Check if key exists (note: metadata is currently string, not jsonb)
			// For now, use simple string contains until we migrate to jsonb
			q = q.Where("metadata LIKE ?", "%\""+filter.Key+"\":%")
		case "not_exists":
			q = q.Where("(metadata IS NULL OR metadata NOT LIKE ?)", "%\""+filter.Key+"\":%")
		case "contains":
			// String contains in metadata
			if strVal, ok := filter.Value.(string); ok {
				q = q.Where("metadata LIKE ?", "%"+strVal+"%")
			}
		case "equals":
			// Exact value match (works for string metadata currently)
			if strVal, ok := filter.Value.(string); ok {
				q = q.Where("metadata LIKE ?", "%\""+filter.Key+"\":\""+strVal+"\"%")
			}
		default:
			// Default to contains
			if strVal, ok := filter.Value.(string); ok {
				q = q.Where("metadata LIKE ?", "%"+strVal+"%")
			}
		}
	}

	return q
}

// contains checks if a string slice contains a value.
func contains(slice []string, item string) bool {

	return slices.Contains(slice, item)
}

// =============================================================================
// FULL-TEXT SEARCH IMPLEMENTATION
// =============================================================================

// Search performs full-text search on audit events (implements audit.SearchRepository).
func (r *AuditRepository) Search(ctx context.Context, query *audit.SearchQuery) (*audit.SearchResponse, error) {
	// Detect database type and route to appropriate implementation
	// For now, default to PostgreSQL implementation
	return r.SearchPostgreSQL(ctx, query)
}

// SearchPostgreSQL performs PostgreSQL tsvector full-text search.
func (r *AuditRepository) SearchPostgreSQL(ctx context.Context, query *audit.SearchQuery) (*audit.SearchResponse, error) {
	// Build base query with relevance ranking
	baseQuery := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("ae.*").
		ColumnExpr("ts_rank(to_tsvector('english', action || ' ' || resource || ' ' || COALESCE(metadata, '')), websearch_to_tsquery('english', ?)) AS rank", query.Query)

	// Apply full-text search condition
	var searchVector string
	if len(query.Fields) == 0 {
		// Search all fields
		searchVector = "to_tsvector('english', action || ' ' || resource || ' ' || COALESCE(metadata, '') || ' ' || COALESCE(user_agent, ''))"
	} else {
		// Build vector from specified fields
		vectors := make([]string, 0, len(query.Fields))
		for _, field := range query.Fields {
			switch field {
			case "action":
				vectors = append(vectors, "to_tsvector('english', action)")
			case "resource":
				vectors = append(vectors, "to_tsvector('english', resource)")
			case "metadata":
				vectors = append(vectors, "to_tsvector('english', COALESCE(metadata, ''))")
			case "user_agent":
				vectors = append(vectors, "to_tsvector('english', COALESCE(user_agent, ''))")
			}
		}

		if len(vectors) > 0 {
			searchVector = "(" + vectors[0]
			var searchVectorSb421 strings.Builder
			for i := 1; i < len(vectors); i++ {
				searchVectorSb421.WriteString(" || " + vectors[i])
			}
			searchVector += searchVectorSb421.String()

			searchVector += ")"
		}
	}

	if searchVector != "" {
		if query.FuzzyMatch {
			// Use plainto_tsquery for fuzzy matching (handles stemming)
			baseQuery = baseQuery.Where(searchVector+" @@ plainto_tsquery('english', ?)", query.Query)
		} else {
			// Use websearch_to_tsquery for exact phrase matching
			baseQuery = baseQuery.Where(searchVector+" @@ websearch_to_tsquery('english', ?)", query.Query)
		}
	}

	// Apply standard filters
	if query.AppID != nil {
		baseQuery = baseQuery.Where("app_id = ?", query.AppID.String())
	}

	if query.OrganizationID != nil {
		baseQuery = baseQuery.Where("organization_id = ?", query.OrganizationID.String())
	}

	if query.EnvironmentID != nil {
		baseQuery = baseQuery.Where("environment_id = ?", query.EnvironmentID.String())
	}

	if query.UserID != nil {
		baseQuery = baseQuery.Where("user_id = ?", query.UserID.String())
	}

	if query.Action != "" {
		baseQuery = baseQuery.Where("action = ?", query.Action)
	}

	if query.Since != nil {
		baseQuery = baseQuery.Where("created_at >= ?", *query.Since)
	}

	if query.Until != nil {
		baseQuery = baseQuery.Where("created_at <= ?", *query.Until)
	}

	// Count total results
	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Order by relevance (rank DESC), then by created_at DESC
	baseQuery = baseQuery.OrderExpr("rank DESC, created_at DESC")

	// Apply pagination
	baseQuery = baseQuery.Limit(query.Limit).Offset(query.Offset)

	// Execute query - need custom struct to capture rank
	type ResultRow struct {
		schema.AuditEvent

		Rank float64 `bun:"rank"`
	}

	var rows []ResultRow
	if err := baseQuery.Scan(ctx, &rows); err != nil {
		return nil, err
	}

	// Convert to search results
	results := make([]*audit.SearchResult, len(rows))
	for i, row := range rows {
		event := audit.FromSchemaEvent(&row.AuditEvent)
		results[i] = &audit.SearchResult{
			Event: event,
			Rank:  row.Rank,
		}
	}

	// Create pagination metadata
	pageSize := query.Limit
	if pageSize == 0 {
		pageSize = 50
	}

	currentPage := (query.Offset / pageSize) + 1
	totalPages := (int(total) + pageSize - 1) / pageSize

	paginationMeta := &pagination.PageMeta{
		Total:       int64(total),
		Limit:       pageSize,
		Offset:      query.Offset,
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		HasNext:     currentPage < totalPages,
		HasPrev:     currentPage > 1,
	}

	return &audit.SearchResponse{
		Results:    results,
		Pagination: paginationMeta,
		Query:      query.Query,
		TookMs:     0, // Will be set by service layer
	}, nil
}

// SearchSQLite performs SQLite FTS5 full-text search (placeholder for SQLite support).
func (r *AuditRepository) SearchSQLite(ctx context.Context, query *audit.SearchQuery) (*audit.SearchResponse, error) {
	// TODO: Implement SQLite FTS5 search
	// For now, return error indicating not implemented
	return nil, ErrSearchNotSupported
}

var ErrSearchNotSupported = audit.InvalidFilter("search", "full-text search not supported for this database")

// =============================================================================
// COUNT OPERATIONS
// =============================================================================

// Count returns the count of audit events matching the filter.
func (r *AuditRepository) Count(ctx context.Context, filter *audit.ListEventsFilter) (int64, error) {
	// Build base query
	q := r.db.NewSelect().Model((*schema.AuditEvent)(nil))

	// Apply filters
	q = r.applyFilters(q, filter)

	// Execute count
	count, err := q.Count(ctx)
	if err != nil {
		return 0, err
	}

	return int64(count), nil
}

// =============================================================================
// RETENTION/CLEANUP OPERATIONS
// =============================================================================

// DeleteOlderThan deletes audit events older than the specified time.
func (r *AuditRepository) DeleteOlderThan(ctx context.Context, filter *audit.DeleteFilter, before time.Time) (int64, error) {
	// Build delete query
	q := r.db.NewDelete().Model((*schema.AuditEvent)(nil))

	// Apply delete filters
	q = r.applyDeleteFilters(q, filter)

	// Apply time constraint
	q = q.Where("created_at < ?", before)

	// Execute deletion
	res, err := q.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

// applyDeleteFilters applies filter conditions to delete queries.
func (r *AuditRepository) applyDeleteFilters(q *bun.DeleteQuery, filter *audit.DeleteFilter) *bun.DeleteQuery {
	if filter == nil {
		return q
	}

	// ========== App Filtering ==========
	if filter.AppID != nil {
		q = q.Where("app_id = ?", filter.AppID.String())
	}

	// ========== Organization Filtering ==========
	if filter.OrganizationID != nil {
		q = q.Where("organization_id = ?", filter.OrganizationID.String())
	}

	// ========== Environment Filtering ==========
	if filter.EnvironmentID != nil {
		q = q.Where("environment_id = ?", filter.EnvironmentID.String())
	}

	// ========== User Filtering ==========
	if filter.UserID != nil {
		q = q.Where("user_id = ?", filter.UserID.String())
	}

	// ========== Action Filtering ==========
	if filter.Action != nil {
		q = q.Where("action = ?", *filter.Action)
	}

	// ========== Resource Filtering ==========
	if filter.Resource != nil {
		q = q.Where("resource = ?", *filter.Resource)
	}

	// ========== Source Filtering ==========
	if filter.Source != nil {
		q = q.Where("source = ?", string(*filter.Source))
	}

	// ========== Metadata Filtering ==========
	if len(filter.MetadataFilters) > 0 {
		for _, mf := range filter.MetadataFilters {
			switch mf.Operator {
			case "exists":
				q = q.Where("metadata LIKE ?", "%\""+mf.Key+"\":%")
			case "not_exists":
				q = q.Where("(metadata IS NULL OR metadata NOT LIKE ?)", "%\""+mf.Key+"\":%")
			case "contains":
				if strVal, ok := mf.Value.(string); ok {
					q = q.Where("metadata LIKE ?", "%"+strVal+"%")
				}
			case "equals":
				if strVal, ok := mf.Value.(string); ok {
					q = q.Where("metadata LIKE ?", "%\""+mf.Key+"\":\""+strVal+"\"%")
				}
			default:
				if strVal, ok := mf.Value.(string); ok {
					q = q.Where("metadata LIKE ?", "%"+strVal+"%")
				}
			}
		}
	}

	// ========== Exclusion Filters ==========
	// Exclude action (single)
	if filter.ExcludeAction != nil {
		q = q.Where("action != ?", *filter.ExcludeAction)
	}

	// Exclude actions (multiple)
	if len(filter.ExcludeActions) > 0 {
		q = q.Where("action NOT IN (?)", bun.In(filter.ExcludeActions))
	}

	// Exclude resource (single)
	if filter.ExcludeResource != nil {
		q = q.Where("resource != ?", *filter.ExcludeResource)
	}

	// Exclude resources (multiple)
	if len(filter.ExcludeResources) > 0 {
		q = q.Where("resource NOT IN (?)", bun.In(filter.ExcludeResources))
	}

	return q
}

// =============================================================================
// STATISTICS/AGGREGATION OPERATIONS
// =============================================================================

// GetStatisticsByAction returns aggregated statistics grouped by action.
func (r *AuditRepository) GetStatisticsByAction(ctx context.Context, filter *audit.StatisticsFilter) ([]*audit.ActionStatistic, error) {
	// Build aggregation query
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("action").
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("MIN(created_at) AS first_occurred").
		ColumnExpr("MAX(created_at) AS last_occurred").
		Group("action").
		Order("count DESC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Apply limit
	if filter != nil && filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	} else {
		q = q.Limit(100) // Default limit
	}

	// Execute query
	var results []struct {
		Action        string    `bun:"action"`
		Count         int64     `bun:"count"`
		FirstOccurred time.Time `bun:"first_occurred"`
		LastOccurred  time.Time `bun:"last_occurred"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DTOs
	stats := make([]*audit.ActionStatistic, len(results))
	for i, r := range results {
		stats[i] = &audit.ActionStatistic{
			Action:        r.Action,
			Count:         r.Count,
			FirstOccurred: r.FirstOccurred,
			LastOccurred:  r.LastOccurred,
		}
	}

	return stats, nil
}

// GetStatisticsByResource returns aggregated statistics grouped by resource.
func (r *AuditRepository) GetStatisticsByResource(ctx context.Context, filter *audit.StatisticsFilter) ([]*audit.ResourceStatistic, error) {
	// Build aggregation query
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("resource").
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("MIN(created_at) AS first_occurred").
		ColumnExpr("MAX(created_at) AS last_occurred").
		Group("resource").
		Order("count DESC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Apply limit
	if filter != nil && filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	} else {
		q = q.Limit(100) // Default limit
	}

	// Execute query
	var results []struct {
		Resource      string    `bun:"resource"`
		Count         int64     `bun:"count"`
		FirstOccurred time.Time `bun:"first_occurred"`
		LastOccurred  time.Time `bun:"last_occurred"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DTOs
	stats := make([]*audit.ResourceStatistic, len(results))
	for i, r := range results {
		stats[i] = &audit.ResourceStatistic{
			Resource:      r.Resource,
			Count:         r.Count,
			FirstOccurred: r.FirstOccurred,
			LastOccurred:  r.LastOccurred,
		}
	}

	return stats, nil
}

// GetStatisticsByUser returns aggregated statistics grouped by user.
func (r *AuditRepository) GetStatisticsByUser(ctx context.Context, filter *audit.StatisticsFilter) ([]*audit.UserStatistic, error) {
	// Build aggregation query
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("user_id").
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("MIN(created_at) AS first_occurred").
		ColumnExpr("MAX(created_at) AS last_occurred").
		Group("user_id").
		Order("count DESC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Apply limit
	if filter != nil && filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	} else {
		q = q.Limit(100) // Default limit
	}

	// Execute query
	var results []struct {
		UserID        *string   `bun:"user_id"`
		Count         int64     `bun:"count"`
		FirstOccurred time.Time `bun:"first_occurred"`
		LastOccurred  time.Time `bun:"last_occurred"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DTOs
	stats := make([]*audit.UserStatistic, len(results))
	for i, r := range results {
		stat := &audit.UserStatistic{
			Count:         r.Count,
			FirstOccurred: r.FirstOccurred,
			LastOccurred:  r.LastOccurred,
		}
		// Parse user ID if present
		if r.UserID != nil && *r.UserID != "" {
			if userID, err := xid.FromString(*r.UserID); err == nil {
				stat.UserID = &userID
			}
		}

		stats[i] = stat
	}

	return stats, nil
}

// applyStatisticsFilters applies filter conditions to statistics queries.
func (r *AuditRepository) applyStatisticsFilters(q *bun.SelectQuery, filter *audit.StatisticsFilter) *bun.SelectQuery {
	if filter == nil {
		return q
	}

	// ========== App Filtering ==========
	if filter.AppID != nil {
		q = q.Where("app_id = ?", filter.AppID.String())
	}

	// ========== Organization Filtering ==========
	if filter.OrganizationID != nil {
		q = q.Where("organization_id = ?", filter.OrganizationID.String())
	}

	// ========== Environment Filtering ==========
	if filter.EnvironmentID != nil {
		q = q.Where("environment_id = ?", filter.EnvironmentID.String())
	}

	// ========== User Filtering ==========
	if filter.UserID != nil {
		q = q.Where("user_id = ?", filter.UserID.String())
	}

	// ========== Action Filtering ==========
	if filter.Action != nil {
		q = q.Where("action = ?", *filter.Action)
	}

	// ========== Resource Filtering ==========
	if filter.Resource != nil {
		q = q.Where("resource = ?", *filter.Resource)
	}

	// ========== Source Filtering ==========
	if filter.Source != nil {
		q = q.Where("source = ?", string(*filter.Source))
	}

	if len(filter.Sources) > 0 {
		sourceStrs := make([]string, len(filter.Sources))
		for i, source := range filter.Sources {
			sourceStrs[i] = string(source)
		}

		q = q.Where("source IN (?)", bun.In(sourceStrs))
	}

	// ========== Time Range Filters ==========
	if filter.Since != nil {
		q = q.Where("created_at >= ?", *filter.Since)
	}

	if filter.Until != nil {
		q = q.Where("created_at <= ?", *filter.Until)
	}

	// ========== Metadata Filtering ==========
	if len(filter.MetadataFilters) > 0 {
		q = r.applyMetadataFilters(q, filter.MetadataFilters)
	}

	// ========== Exclusion Filters ==========
	// Exclude source (single)
	if filter.ExcludeSource != nil {
		q = q.Where("source != ?", string(*filter.ExcludeSource))
	}

	// Exclude sources (multiple)
	if len(filter.ExcludeSources) > 0 {
		sourceStrs := make([]string, len(filter.ExcludeSources))
		for i, source := range filter.ExcludeSources {
			sourceStrs[i] = string(source)
		}

		q = q.Where("source NOT IN (?)", bun.In(sourceStrs))
	}

	// Exclude action (single)
	if filter.ExcludeAction != nil {
		q = q.Where("action != ?", *filter.ExcludeAction)
	}

	// Exclude actions (multiple)
	if len(filter.ExcludeActions) > 0 {
		q = q.Where("action NOT IN (?)", bun.In(filter.ExcludeActions))
	}

	// Exclude resource (single)
	if filter.ExcludeResource != nil {
		q = q.Where("resource != ?", *filter.ExcludeResource)
	}

	// Exclude resources (multiple)
	if len(filter.ExcludeResources) > 0 {
		q = q.Where("resource NOT IN (?)", bun.In(filter.ExcludeResources))
	}

	// Exclude user (single)
	if filter.ExcludeUserID != nil {
		q = q.Where("user_id != ?", filter.ExcludeUserID.String())
	}

	// Exclude users (multiple)
	if len(filter.ExcludeUserIDs) > 0 {
		userIDStrs := make([]string, len(filter.ExcludeUserIDs))
		for i, id := range filter.ExcludeUserIDs {
			userIDStrs[i] = id.String()
		}

		q = q.Where("user_id NOT IN (?)", bun.In(userIDStrs))
	}

	return q
}

// =============================================================================
// UTILITY OPERATIONS
// =============================================================================

// GetOldestEvent retrieves the oldest audit event matching the filter.
func (r *AuditRepository) GetOldestEvent(ctx context.Context, filter *audit.ListEventsFilter) (*schema.AuditEvent, error) {
	// Build base query
	q := r.db.NewSelect().Model((*schema.AuditEvent)(nil))

	// Apply filters
	q = r.applyFilters(q, filter)

	// Order by created_at ASC to get oldest first
	q = q.Order("created_at ASC")

	// Limit to 1
	q = q.Limit(1)

	// Execute query
	var event schema.AuditEvent

	err := q.Scan(ctx, &event)
	if err != nil {
		// Check if it's a no rows error
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}

		return nil, err
	}

	// Check if we got a result (ID will be nil/empty if no rows)
	if event.ID.IsNil() {
		return nil, nil
	}

	return &event, nil
}

// =============================================================================
// TIME-BASED AGGREGATION OPERATIONS
// =============================================================================

// GetTimeSeries returns event counts over time with configurable intervals.
func (r *AuditRepository) GetTimeSeries(ctx context.Context, filter *audit.TimeSeriesFilter) ([]*audit.TimeSeriesPoint, error) {
	// Determine the date_trunc interval
	var truncInterval string

	switch filter.Interval {
	case audit.IntervalHourly:
		truncInterval = "hour"
	case audit.IntervalDaily:
		truncInterval = "day"
	case audit.IntervalWeekly:
		truncInterval = "week"
	case audit.IntervalMonthly:
		truncInterval = "month"
	default:
		truncInterval = "day"
	}

	// Build aggregation query
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("date_trunc(?, created_at) AS timestamp", truncInterval).
		ColumnExpr("COUNT(*) AS count").
		Group("timestamp").
		Order("timestamp ASC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, &filter.StatisticsFilter)

	// Execute query
	var results []struct {
		Timestamp time.Time `bun:"timestamp"`
		Count     int64     `bun:"count"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DTOs
	points := make([]*audit.TimeSeriesPoint, len(results))
	for i, r := range results {
		points[i] = &audit.TimeSeriesPoint{
			Timestamp: r.Timestamp,
			Count:     r.Count,
		}
	}

	return points, nil
}

// GetStatisticsByHour returns event distribution by hour of day (0-23).
func (r *AuditRepository) GetStatisticsByHour(ctx context.Context, filter *audit.StatisticsFilter) ([]*audit.HourStatistic, error) {
	// Build aggregation query - EXTRACT(HOUR FROM created_at)
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("EXTRACT(HOUR FROM created_at)::int AS hour").
		ColumnExpr("COUNT(*) AS count").
		Group("hour").
		Order("hour ASC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Execute query
	var results []struct {
		Hour  int   `bun:"hour"`
		Count int64 `bun:"count"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DTOs
	stats := make([]*audit.HourStatistic, len(results))
	for i, r := range results {
		stats[i] = &audit.HourStatistic{
			Hour:  r.Hour,
			Count: r.Count,
		}
	}

	return stats, nil
}

// GetStatisticsByDay returns event distribution by day of week.
func (r *AuditRepository) GetStatisticsByDay(ctx context.Context, filter *audit.StatisticsFilter) ([]*audit.DayStatistic, error) {
	// Build aggregation query - EXTRACT(DOW FROM created_at) returns 0=Sunday, 1=Monday, etc.
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("EXTRACT(DOW FROM created_at)::int AS day_index").
		ColumnExpr("COUNT(*) AS count").
		Group("day_index").
		Order("day_index ASC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Execute query
	var results []struct {
		DayIndex int   `bun:"day_index"`
		Count    int64 `bun:"count"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Day names mapping
	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

	// Convert to DTOs
	stats := make([]*audit.DayStatistic, len(results))
	for i, r := range results {
		dayName := "Unknown"
		if r.DayIndex >= 0 && r.DayIndex < 7 {
			dayName = dayNames[r.DayIndex]
		}

		stats[i] = &audit.DayStatistic{
			Day:      dayName,
			DayIndex: r.DayIndex,
			Count:    r.Count,
		}
	}

	return stats, nil
}

// GetStatisticsByDate returns daily event counts for a date range.
func (r *AuditRepository) GetStatisticsByDate(ctx context.Context, filter *audit.StatisticsFilter) ([]*audit.DateStatistic, error) {
	// Build aggregation query
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("DATE(created_at) AS date").
		ColumnExpr("COUNT(*) AS count").
		Group("date").
		Order("date ASC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Execute query
	var results []struct {
		Date  time.Time `bun:"date"`
		Count int64     `bun:"count"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DTOs
	stats := make([]*audit.DateStatistic, len(results))
	for i, r := range results {
		stats[i] = &audit.DateStatistic{
			Date:  r.Date.Format("2006-01-02"),
			Count: r.Count,
		}
	}

	return stats, nil
}

// =============================================================================
// IP/NETWORK AGGREGATION OPERATIONS
// =============================================================================

// GetStatisticsByIPAddress returns event counts grouped by IP address.
func (r *AuditRepository) GetStatisticsByIPAddress(ctx context.Context, filter *audit.StatisticsFilter) ([]*audit.IPStatistic, error) {
	// Build aggregation query
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("ip_address").
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("MIN(created_at) AS first_occurred").
		ColumnExpr("MAX(created_at) AS last_occurred").
		Where("ip_address IS NOT NULL AND ip_address != ''").
		Group("ip_address").
		Order("count DESC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Apply limit
	if filter != nil && filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	} else {
		q = q.Limit(100) // Default limit
	}

	// Execute query
	var results []struct {
		IPAddress     string    `bun:"ip_address"`
		Count         int64     `bun:"count"`
		FirstOccurred time.Time `bun:"first_occurred"`
		LastOccurred  time.Time `bun:"last_occurred"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DTOs
	stats := make([]*audit.IPStatistic, len(results))
	for i, r := range results {
		stats[i] = &audit.IPStatistic{
			IPAddress:     r.IPAddress,
			Count:         r.Count,
			FirstOccurred: r.FirstOccurred,
			LastOccurred:  r.LastOccurred,
		}
	}

	return stats, nil
}

// GetUniqueIPCount returns the count of unique IP addresses.
func (r *AuditRepository) GetUniqueIPCount(ctx context.Context, filter *audit.StatisticsFilter) (int64, error) {
	// Build count query
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("COUNT(DISTINCT ip_address) AS count").
		Where("ip_address IS NOT NULL AND ip_address != ''")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Execute query
	var result struct {
		Count int64 `bun:"count"`
	}

	if err := q.Scan(ctx, &result); err != nil {
		return 0, err
	}

	return result.Count, nil
}

// =============================================================================
// MULTI-DIMENSIONAL AGGREGATION OPERATIONS
// =============================================================================

// GetStatisticsByActionAndUser returns event counts grouped by action and user.
func (r *AuditRepository) GetStatisticsByActionAndUser(ctx context.Context, filter *audit.StatisticsFilter) ([]*audit.ActionUserStatistic, error) {
	// Build aggregation query
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("action").
		ColumnExpr("user_id").
		ColumnExpr("COUNT(*) AS count").
		Group("action", "user_id").
		Order("count DESC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Apply limit
	if filter != nil && filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	} else {
		q = q.Limit(100) // Default limit
	}

	// Execute query
	var results []struct {
		Action string  `bun:"action"`
		UserID *string `bun:"user_id"`
		Count  int64   `bun:"count"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DTOs
	stats := make([]*audit.ActionUserStatistic, len(results))
	for i, r := range results {
		stat := &audit.ActionUserStatistic{
			Action: r.Action,
			Count:  r.Count,
		}
		// Parse user ID if present
		if r.UserID != nil && *r.UserID != "" {
			if userID, err := xid.FromString(*r.UserID); err == nil {
				stat.UserID = &userID
			}
		}

		stats[i] = stat
	}

	return stats, nil
}

// GetStatisticsByResourceAndAction returns event counts grouped by resource and action.
func (r *AuditRepository) GetStatisticsByResourceAndAction(ctx context.Context, filter *audit.StatisticsFilter) ([]*audit.ResourceActionStatistic, error) {
	// Build aggregation query
	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		ColumnExpr("resource").
		ColumnExpr("action").
		ColumnExpr("COUNT(*) AS count").
		Group("resource", "action").
		Order("count DESC")

	// Apply statistics filters
	q = r.applyStatisticsFilters(q, filter)

	// Apply limit
	if filter != nil && filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	} else {
		q = q.Limit(100) // Default limit
	}

	// Execute query
	var results []struct {
		Resource string `bun:"resource"`
		Action   string `bun:"action"`
		Count    int64  `bun:"count"`
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DTOs
	stats := make([]*audit.ResourceActionStatistic, len(results))
	for i, r := range results {
		stats[i] = &audit.ResourceActionStatistic{
			Resource: r.Resource,
			Action:   r.Action,
			Count:    r.Count,
		}
	}

	return stats, nil
}

// =============================================================================
// AGGREGATION OPERATIONS
// =============================================================================

// GetDistinctActions returns distinct action values with counts.
func (r *AuditRepository) GetDistinctActions(ctx context.Context, filter *audit.AggregationFilter) ([]audit.DistinctValue, error) {
	type result struct {
		Value string `bun:"action"`
		Count int64  `bun:"count"`
	}

	var results []result

	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		Column("action").
		ColumnExpr("COUNT(*) as count").
		Group("action").
		Order("count DESC")

	q = r.applyAggregationFilters(q, filter)

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	// Convert to DistinctValue
	values := make([]audit.DistinctValue, len(results))
	for i, res := range results {
		values[i] = audit.DistinctValue{
			Value: res.Value,
			Count: res.Count,
		}
	}

	return values, nil
}

// GetDistinctSources returns distinct source values with counts.
func (r *AuditRepository) GetDistinctSources(ctx context.Context, filter *audit.AggregationFilter) ([]audit.DistinctValue, error) {
	type result struct {
		Value string `bun:"source"`
		Count int64  `bun:"count"`
	}

	var results []result

	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		Column("source").
		ColumnExpr("COUNT(*) as count").
		Group("source").
		Order("count DESC")

	q = r.applyAggregationFilters(q, filter)

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	values := make([]audit.DistinctValue, len(results))
	for i, res := range results {
		values[i] = audit.DistinctValue{
			Value: res.Value,
			Count: res.Count,
		}
	}

	return values, nil
}

// GetDistinctResources returns distinct resource values with counts.
func (r *AuditRepository) GetDistinctResources(ctx context.Context, filter *audit.AggregationFilter) ([]audit.DistinctValue, error) {
	type result struct {
		Value string `bun:"resource"`
		Count int64  `bun:"count"`
	}

	var results []result

	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		Column("resource").
		ColumnExpr("COUNT(*) as count").
		Where("resource IS NOT NULL AND resource != ''").
		Group("resource").
		Order("count DESC")

	q = r.applyAggregationFilters(q, filter)

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	values := make([]audit.DistinctValue, len(results))
	for i, res := range results {
		values[i] = audit.DistinctValue{
			Value: res.Value,
			Count: res.Count,
		}
	}

	return values, nil
}

// GetDistinctUsers returns distinct user values with counts.
func (r *AuditRepository) GetDistinctUsers(ctx context.Context, filter *audit.AggregationFilter) ([]audit.DistinctValue, error) {
	type result struct {
		Value string `bun:"user_id"`
		Count int64  `bun:"count"`
	}

	var results []result

	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		Column("user_id").
		ColumnExpr("COUNT(*) as count").
		Where("user_id IS NOT NULL").
		Group("user_id").
		Order("count DESC")

	q = r.applyAggregationFilters(q, filter)

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	values := make([]audit.DistinctValue, len(results))
	for i, res := range results {
		values[i] = audit.DistinctValue{
			Value: res.Value,
			Count: res.Count,
		}
	}

	return values, nil
}

// GetDistinctIPs returns distinct IP address values with counts.
func (r *AuditRepository) GetDistinctIPs(ctx context.Context, filter *audit.AggregationFilter) ([]audit.DistinctValue, error) {
	type result struct {
		Value string `bun:"ip_address"`
		Count int64  `bun:"count"`
	}

	var results []result

	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		Column("ip_address").
		ColumnExpr("COUNT(*) as count").
		Where("ip_address IS NOT NULL AND ip_address != ''").
		Group("ip_address").
		Order("count DESC")

	q = r.applyAggregationFilters(q, filter)

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	values := make([]audit.DistinctValue, len(results))
	for i, res := range results {
		values[i] = audit.DistinctValue{
			Value: res.Value,
			Count: res.Count,
		}
	}

	return values, nil
}

// GetDistinctApps returns distinct app values with counts.
func (r *AuditRepository) GetDistinctApps(ctx context.Context, filter *audit.AggregationFilter) ([]audit.DistinctValue, error) {
	type result struct {
		Value string `bun:"app_id"`
		Count int64  `bun:"count"`
	}

	var results []result

	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		Column("app_id").
		ColumnExpr("COUNT(*) as count").
		Group("app_id").
		Order("count DESC")

	q = r.applyAggregationFilters(q, filter)

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	values := make([]audit.DistinctValue, len(results))
	for i, res := range results {
		values[i] = audit.DistinctValue{
			Value: res.Value,
			Count: res.Count,
		}
	}

	return values, nil
}

// GetDistinctOrganizations returns distinct organization values with counts.
func (r *AuditRepository) GetDistinctOrganizations(ctx context.Context, filter *audit.AggregationFilter) ([]audit.DistinctValue, error) {
	type result struct {
		Value string `bun:"organization_id"`
		Count int64  `bun:"count"`
	}

	var results []result

	q := r.db.NewSelect().
		Model((*schema.AuditEvent)(nil)).
		Column("organization_id").
		ColumnExpr("COUNT(*) as count").
		Where("organization_id IS NOT NULL").
		Group("organization_id").
		Order("count DESC")

	q = r.applyAggregationFilters(q, filter)

	if filter.Limit > 0 {
		q = q.Limit(filter.Limit)
	}

	if err := q.Scan(ctx, &results); err != nil {
		return nil, err
	}

	values := make([]audit.DistinctValue, len(results))
	for i, res := range results {
		values[i] = audit.DistinctValue{
			Value: res.Value,
			Count: res.Count,
		}
	}

	return values, nil
}

// applyAggregationFilters applies common filters to aggregation queries.
func (r *AuditRepository) applyAggregationFilters(q *bun.SelectQuery, filter *audit.AggregationFilter) *bun.SelectQuery {
	if filter == nil {
		return q
	}

	if filter.AppID != nil {
		q = q.Where("app_id = ?", filter.AppID.String())
	}

	if filter.OrganizationID != nil {
		q = q.Where("organization_id = ?", filter.OrganizationID.String())
	}

	if filter.EnvironmentID != nil {
		q = q.Where("environment_id = ?", filter.EnvironmentID.String())
	}

	if filter.Since != nil {
		q = q.Where("created_at >= ?", *filter.Since)
	}

	if filter.Until != nil {
		q = q.Where("created_at <= ?", *filter.Until)
	}

	return q
}
