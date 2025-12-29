package audit

import (
	"context"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/core/pagination"
)

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// =============================================================================
// FULL-TEXT SEARCH - Database-native search capabilities
// =============================================================================

// SearchQuery represents a full-text search query with filters
type SearchQuery struct {
	// Search query string (supports natural language and operators)
	Query string `json:"query"`

	// Fields to search in (empty = search all fields)
	Fields []string `json:"fields,omitempty"`

	// Enable fuzzy matching (stemming, similar words)
	FuzzyMatch bool `json:"fuzzyMatch"`

	// Pagination
	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	// Standard filters (AND combined with search query)
	AppID  *xid.ID    `json:"appId,omitempty"`
	UserID *xid.ID    `json:"userId,omitempty"`
	Action string     `json:"action,omitempty"`
	Since  *time.Time `json:"since,omitempty"`
	Until  *time.Time `json:"until,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Event *Event  `json:"event"`
	Rank  float64 `json:"rank"` // Relevance score (0-1)
}

// SearchResponse represents paginated search results
type SearchResponse struct {
	Results    []*SearchResult      `json:"results"`
	Pagination *pagination.PageMeta `json:"pagination"`
	Query      string               `json:"query"`  // The processed query
	TookMs     int64                `json:"tookMs"` // Query execution time in milliseconds
}

// SearchRepository defines database-specific search implementation
type SearchRepository interface {
	// Search performs full-text search on audit events
	Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error)

	// SearchPostgreSQL performs PostgreSQL tsvector search
	SearchPostgreSQL(ctx context.Context, query *SearchQuery) (*SearchResponse, error)

	// SearchSQLite performs SQLite FTS5 search
	SearchSQLite(ctx context.Context, query *SearchQuery) (*SearchResponse, error)
}

// Search performs full-text search on audit events
func (s *Service) Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error) {
	// Validate query
	if query == nil {
		return nil, InvalidFilter("query", "query cannot be nil")
	}

	if query.Query == "" {
		return nil, InvalidFilter("query", "search query cannot be empty")
	}

	// Validate pagination
	if query.Limit < 0 {
		return nil, InvalidPagination("limit cannot be negative")
	}
	if query.Offset < 0 {
		return nil, InvalidPagination("offset cannot be negative")
	}

	// Set defaults
	if query.Limit == 0 {
		query.Limit = 50
	}

	// Validate time range
	if query.Since != nil && query.Until != nil && query.Since.After(*query.Until) {
		return nil, InvalidTimeRange("since must be before until")
	}

	// Check if repository supports search
	searchRepo, ok := s.repo.(SearchRepository)
	if !ok {
		// Fall back to filtering on List (less efficient)
		return s.searchFallback(ctx, query)
	}

	// Execute search
	startTime := time.Now()
	results, err := searchRepo.Search(ctx, query)
	if err != nil {
		return nil, QueryFailed("search", err)
	}

	results.TookMs = time.Since(startTime).Milliseconds()

	return results, nil
}

// searchFallback performs search using basic filtering (when FTS not available)
// This is a fallback for repositories that don't implement SearchRepository
func (s *Service) searchFallback(ctx context.Context, query *SearchQuery) (*SearchResponse, error) {
	// Convert search to basic filters
	action := query.Action
	filter := &ListEventsFilter{
		UserID: query.UserID,
		Action: &action,
		Since:  query.Since,
		Until:  query.Until,
		PaginationParams: pagination.PaginationParams{
			Limit:  query.Limit,
			Offset: query.Offset,
			Page:   (query.Offset / max(query.Limit, 1)) + 1,
		},
	}

	// Get events using standard list
	pageResp, err := s.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert to search results (no ranking in fallback)
	results := make([]*SearchResult, len(pageResp.Data))
	for i, event := range pageResp.Data {
		results[i] = &SearchResult{
			Event: event,
			Rank:  1.0, // No ranking in fallback
		}
	}

	return &SearchResponse{
		Results:    results,
		Pagination: pageResp.Pagination,
		Query:      query.Query,
	}, nil
}

// =============================================================================
// SEARCH QUERY BUILDER - Helper for constructing search queries
// =============================================================================

// SearchQueryBuilder provides fluent API for building search queries
type SearchQueryBuilder struct {
	query *SearchQuery
}

// NewSearchQuery creates a new search query builder
func NewSearchQuery(searchText string) *SearchQueryBuilder {
	return &SearchQueryBuilder{
		query: &SearchQuery{
			Query:  searchText,
			Limit:  50,
			Offset: 0,
		},
	}
}

// InFields restricts search to specific fields
func (b *SearchQueryBuilder) InFields(fields ...string) *SearchQueryBuilder {
	b.query.Fields = fields
	return b
}

// Fuzzy enables fuzzy matching
func (b *SearchQueryBuilder) Fuzzy() *SearchQueryBuilder {
	b.query.FuzzyMatch = true
	return b
}

// ForApp filters by app ID
func (b *SearchQueryBuilder) ForApp(appID xid.ID) *SearchQueryBuilder {
	b.query.AppID = &appID
	return b
}

// ForUser filters by user ID
func (b *SearchQueryBuilder) ForUser(userID xid.ID) *SearchQueryBuilder {
	b.query.UserID = &userID
	return b
}

// WithAction filters by action
func (b *SearchQueryBuilder) WithAction(action string) *SearchQueryBuilder {
	b.query.Action = action
	return b
}

// Since filters events after timestamp
func (b *SearchQueryBuilder) Since(t time.Time) *SearchQueryBuilder {
	b.query.Since = &t
	return b
}

// Until filters events before timestamp
func (b *SearchQueryBuilder) Until(t time.Time) *SearchQueryBuilder {
	b.query.Until = &t
	return b
}

// Limit sets result limit
func (b *SearchQueryBuilder) Limit(limit int) *SearchQueryBuilder {
	b.query.Limit = limit
	return b
}

// Offset sets result offset
func (b *SearchQueryBuilder) Offset(offset int) *SearchQueryBuilder {
	b.query.Offset = offset
	return b
}

// Build returns the constructed query
func (b *SearchQueryBuilder) Build() *SearchQuery {
	return b.query
}

// =============================================================================
// SEARCH EXAMPLES
// =============================================================================

/*
Example Usage:

1. Basic search:
   results, err := auditSvc.Search(ctx, &SearchQuery{
       Query: "login failed",
       Limit: 100,
   })

2. Search with filters:
   results, err := auditSvc.Search(ctx, &SearchQuery{
       Query:  "password change",
       AppID:  &appID,
       UserID: &userID,
       Since:  time.Now().AddDate(0, 0, -7),
   })

3. Fluent API:
   query := NewSearchQuery("login failed").
       InFields("action", "metadata").
       ForApp(appID).
       Since(time.Now().AddDate(0, 0, -30)).
       Fuzzy().
       Limit(100).
       Build()

   results, err := auditSvc.Search(ctx, query)

4. Field-specific search:
   results, err := auditSvc.Search(ctx, &SearchQuery{
       Query:  "192.168.1.*",
       Fields: []string{"ip_address"},
   })
*/
