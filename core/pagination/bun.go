package pagination

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

// QueryBuilder provides methods to apply pagination parameters to Bun queries.
type QueryBuilder struct {
	params any
}

// NewQueryBuilder creates a new QueryBuilder from pagination parameters.
func NewQueryBuilder(params any) *QueryBuilder {
	return &QueryBuilder{params: params}
}

// ApplyToQuery applies all pagination parameters to a Bun query
// This is a convenience method that calls ApplyLimit, ApplyOffset, ApplyOrder, and ApplyFields.
func (qb *QueryBuilder) ApplyToQuery(query *bun.SelectQuery) *bun.SelectQuery {
	query = qb.ApplyFields(query)
	query = qb.ApplyLimit(query)
	query = qb.ApplyOffset(query)
	query = qb.ApplyOrder(query)

	return query
}

// ApplyLimit applies the limit parameter to a Bun query
// Note: BaseRequestParams does not have a Limit field (not all requests need pagination).
func (qb *QueryBuilder) ApplyLimit(query *bun.SelectQuery) *bun.SelectQuery {
	switch p := qb.params.(type) {
	case *PaginationParams:
		return query.Limit(p.GetLimit())
	case *CursorParams:
		return query.Limit(p.GetLimit())
	case *BaseRequestParams:
		// BaseRequestParams doesn't have limit - skip
		return query
	default:
		return query
	}
}

// ApplyOffset applies the offset parameter to a Bun query
// Only works with PaginationParams (not cursor-based).
func (qb *QueryBuilder) ApplyOffset(query *bun.SelectQuery) *bun.SelectQuery {
	if p, ok := qb.params.(*PaginationParams); ok {
		return query.Offset(p.GetOffset())
	}

	return query
}

// ApplyFields applies field selection to a Bun query
// Only selects specified fields if Fields parameter is set.
func (qb *QueryBuilder) ApplyFields(query *bun.SelectQuery) *bun.SelectQuery {
	var fields []string

	switch p := qb.params.(type) {
	case *PaginationParams:
		fields = p.GetFields()
	case *CursorParams:
		fields = p.GetFields()
	case *BaseRequestParams:
		fields = p.GetFields()
	default:
		return query
	}

	if len(fields) == 0 {
		return query
	}

	// Apply column selection
	return query.Column(fields...)
}

// ApplyOrder applies the order parameter to a Bun query.
func (qb *QueryBuilder) ApplyOrder(query *bun.SelectQuery) *bun.SelectQuery {
	var (
		sortBy string
		order  SortOrder
	)

	switch p := qb.params.(type) {
	case *PaginationParams:
		sortBy = p.GetSortBy()
		order = p.GetOrder()
	case *CursorParams:
		sortBy = p.GetSortBy()
		order = p.GetOrder()
	case *BaseRequestParams:
		sortBy = p.SortBy
		if sortBy == "" {
			sortBy = "created_at"
		}

		order = p.Order
		if order == "" {
			order = SortOrderDesc
		}
	default:
		return query
	}

	orderClause := fmt.Sprintf("%s %s", sortBy, strings.ToUpper(string(order)))

	return query.Order(orderClause)
}

// ApplySearch applies a search filter to a Bun query
// searchFields should be the column names to search in
// Example: ApplySearch(query, "name", "email", "username")
// Note: Uses LOWER() for case-insensitive search (works with all databases).
func (qb *QueryBuilder) ApplySearch(query *bun.SelectQuery, searchFields ...string) *bun.SelectQuery {
	if len(searchFields) == 0 {
		return query
	}

	var search string

	switch p := qb.params.(type) {
	case *PaginationParams:
		search = p.Search
	case *CursorParams:
		search = p.Search
	case *BaseRequestParams:
		search = p.Search
	default:
		return query
	}

	if search == "" {
		return query
	}

	// Build OR conditions for each search field using LOWER() for case-insensitive search
	searchPattern := "%" + strings.ToLower(search) + "%"
	conditions := make([]string, len(searchFields))
	args := make([]any, len(searchFields))

	for i, field := range searchFields {
		conditions[i] = fmt.Sprintf("LOWER(%s) LIKE ?", field)
		args[i] = searchPattern
	}

	whereClause := strings.Join(conditions, " OR ")

	return query.Where("("+whereClause+")", args...)
}

// ApplyCursor applies cursor-based pagination to a Bun query
// cursorData should be obtained from DecodeCursor()
// cursorField is the field to use for cursor comparison (default: "id")
// timestampField is the timestamp field (default: "created_at").
func (qb *QueryBuilder) ApplyCursor(query *bun.SelectQuery, cursorData *CursorData, cursorField, timestampField string) *bun.SelectQuery {
	if cursorData == nil {
		return query
	}

	if cursorField == "" {
		cursorField = "id"
	}

	if timestampField == "" {
		timestampField = "created_at"
	}

	var order SortOrder
	if cp, ok := qb.params.(*CursorParams); ok {
		order = cp.GetOrder()
	} else {
		order = SortOrderDesc
	}

	// Apply cursor condition based on sort order
	if order == SortOrderDesc {
		query = query.Where(
			fmt.Sprintf("(%s < ? OR (%s = ? AND %s < ?))", timestampField, timestampField, cursorField),
			cursorData.Timestamp, cursorData.Timestamp, cursorData.ID,
		)
	} else {
		query = query.Where(
			fmt.Sprintf("(%s > ? OR (%s = ? AND %s > ?))", timestampField, timestampField, cursorField),
			cursorData.Timestamp, cursorData.Timestamp, cursorData.ID,
		)
	}

	return query
}

// ParseFilters parses a filter string into a map
// Format: "key1:value1,key2:value2"
// Example: "status:active,role:admin".
func ParseFilters(filterStr string) map[string]string {
	filters := make(map[string]string)
	if filterStr == "" {
		return filters
	}

	pairs := strings.SplitSeq(filterStr, ",")
	for pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			filters[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return filters
}

// ApplyFilters applies parsed filters to a Bun query
// filters should be from ParseFilters()
// It applies exact match filters as WHERE conditions.
func (qb *QueryBuilder) ApplyFilters(query *bun.SelectQuery, filters map[string]string) *bun.SelectQuery {
	for field, value := range filters {
		query = query.Where(field+" = ?", value)
	}

	return query
}

// Helper functions for common patterns

// Apply applies all standard parameters (limit, offset, order) to a query.
func Apply(query *bun.SelectQuery, params *PaginationParams) *bun.SelectQuery {
	return NewQueryBuilder(params).ApplyToQuery(query)
}

// ApplyWithSearch applies standard parameters plus search to a query.
func ApplyWithSearch(query *bun.SelectQuery, params *PaginationParams, searchFields ...string) *bun.SelectQuery {
	qb := NewQueryBuilder(params)
	query = qb.ApplyToQuery(query)
	query = qb.ApplySearch(query, searchFields...)

	return query
}

// ApplyWithFilters applies standard parameters plus filters to a query.
func ApplyWithFilters(query *bun.SelectQuery, params *PaginationParams) *bun.SelectQuery {
	qb := NewQueryBuilder(params)
	query = qb.ApplyToQuery(query)

	if params.Filter != "" {
		filters := ParseFilters(params.Filter)
		query = qb.ApplyFilters(query, filters)
	}

	return query
}

// ApplyAll applies all parameters including search and filters.
func ApplyAll(query *bun.SelectQuery, params *PaginationParams, searchFields ...string) *bun.SelectQuery {
	qb := NewQueryBuilder(params)
	query = qb.ApplyToQuery(query)
	query = qb.ApplySearch(query, searchFields...)

	if params.Filter != "" {
		filters := ParseFilters(params.Filter)
		query = qb.ApplyFilters(query, filters)
	}

	return query
}

// ApplyCursorPagination applies cursor-based pagination to a query.
func ApplyCursorPagination(query *bun.SelectQuery, params *CursorParams, cursorField, timestampField string) (*bun.SelectQuery, error) {
	qb := NewQueryBuilder(params)

	// Apply limit and order
	query = qb.ApplyLimit(query)
	query = qb.ApplyOrder(query)

	// Apply cursor if present
	if params.Cursor != "" {
		cursorData, err := DecodeCursor(params.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}

		query = qb.ApplyCursor(query, cursorData, cursorField, timestampField)
	}

	return query, nil
}

// ScanAndCount is a helper that executes a query and returns results with total count
// This is useful for offset-based pagination.
func ScanAndCount[T any](ctx context.Context, query *bun.SelectQuery, dest *[]T) (int64, error) {
	count, err := query.ScanAndCount(ctx, dest)
	if err != nil {
		return 0, fmt.Errorf("failed to scan and count: %w", err)
	}

	return int64(count), nil
}

// ApplyBase applies sorting, searching, filtering, and field selection from BaseRequestParams
// This is useful for non-paginated requests that still need sorting/filtering.
func ApplyBase(query *bun.SelectQuery, params *BaseRequestParams, searchFields ...string) *bun.SelectQuery {
	qb := NewQueryBuilder(params)
	query = qb.ApplyFields(query)
	query = qb.ApplyOrder(query)
	query = qb.ApplySearch(query, searchFields...)

	if params.Filter != "" {
		filters := ParseFilters(params.Filter)
		query = qb.ApplyFilters(query, filters)
	}

	return query
}
