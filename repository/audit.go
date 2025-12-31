package repository

import (
	"context"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/xraph/authsome/core/audit"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/schema"
)

// AuditRepository implements core audit repository using Bun
type AuditRepository struct {
	db *bun.DB
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(db *bun.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create creates a new audit event
func (r *AuditRepository) Create(ctx context.Context, e *schema.AuditEvent) error {
	_, err := r.db.NewInsert().Model(e).Exec(ctx)
	return err
}

// Get retrieves an audit event by ID
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

// List returns paginated audit events with optional filters
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

// applyFilters applies filter conditions to the query
func (r *AuditRepository) applyFilters(q *bun.SelectQuery, filter *audit.ListEventsFilter) *bun.SelectQuery {
	// ========== Full-Text Search ==========
	if filter.SearchQuery != nil && *filter.SearchQuery != "" {
		q = r.applyFullTextSearch(q, *filter.SearchQuery, filter.SearchFields)
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

	// ========== Multiple Value Filters (IN clauses) ==========
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

	return q
}

// applyFullTextSearch applies PostgreSQL full-text search
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
			for i := 1; i < len(vectors); i++ {
				searchVector += " || " + vectors[i]
			}
			searchVector += ")"
		}
	}

	if searchVector != "" {
		// Use websearch_to_tsquery for natural language query parsing
		q = q.Where(searchVector+" @@ websearch_to_tsquery('english', ?)", searchQuery)
	}

	return q
}

// applyMetadataFilters applies metadata JSON filters
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

// contains checks if a string slice contains a value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// =============================================================================
// FULL-TEXT SEARCH IMPLEMENTATION
// =============================================================================

// Search performs full-text search on audit events (implements audit.SearchRepository)
func (r *AuditRepository) Search(ctx context.Context, query *audit.SearchQuery) (*audit.SearchResponse, error) {
	// Detect database type and route to appropriate implementation
	// For now, default to PostgreSQL implementation
	return r.SearchPostgreSQL(ctx, query)
}

// SearchPostgreSQL performs PostgreSQL tsvector full-text search
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
			for i := 1; i < len(vectors); i++ {
				searchVector += " || " + vectors[i]
			}
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

// SearchSQLite performs SQLite FTS5 full-text search (placeholder for SQLite support)
func (r *AuditRepository) SearchSQLite(ctx context.Context, query *audit.SearchQuery) (*audit.SearchResponse, error) {
	// TODO: Implement SQLite FTS5 search
	// For now, return error indicating not implemented
	return nil, ErrSearchNotSupported
}

var ErrSearchNotSupported = audit.InvalidFilter("search", "full-text search not supported for this database")
