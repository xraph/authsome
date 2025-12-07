package query

import (
	"context"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// QueryExecutor executes queries and returns results
type QueryExecutor struct {
	db *bun.DB
}

// NewQueryExecutor creates a new query executor
func NewQueryExecutor(db *bun.DB) *QueryExecutor {
	return &QueryExecutor{db: db}
}

// QueryResult holds the result of a query execution
type QueryResult struct {
	Entries    []*schema.ContentEntry `json:"entries"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"pageSize"`
	TotalItems int                    `json:"totalItems"`
	TotalPages int                    `json:"totalPages"`
}

// Execute executes a query and returns the results
func (e *QueryExecutor) Execute(ctx context.Context, contentType *schema.ContentType, q *Query) (*QueryResult, error) {
	// Validate query against content type fields
	fieldMap := make(map[string]*core.ContentFieldDTO)
	for _, f := range contentType.Fields {
		fieldMap[f.Name] = &core.ContentFieldDTO{
			ID:   f.ID.String(),
			Name: f.Name,
			Type: f.Type,
		}
	}

	if err := q.Validate(fieldMap); err != nil {
		return nil, err
	}

	// Build queries
	builder := NewQueryBuilder(e.db, contentType.ID, contentType.Fields)

	// Get total count
	countQuery := builder.BuildCount(q)
	total, err := countQuery.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Get entries
	selectQuery := builder.Build(q)
	var entries []*schema.ContentEntry
	if err := selectQuery.Scan(ctx, &entries); err != nil {
		return nil, err
	}

	// Calculate pagination info
	page := q.Page
	if page <= 0 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if q.Limit > 0 {
		pageSize = q.Limit
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &QueryResult{
		Entries:    entries,
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

// ExecuteByID executes a query to find a single entry by ID
func (e *QueryExecutor) ExecuteByID(ctx context.Context, entryID xid.ID) (*schema.ContentEntry, error) {
	entry := new(schema.ContentEntry)
	err := e.db.NewSelect().
		Model(entry).
		Where("ce.id = ?", entryID).
		Where("ce.deleted_at IS NULL").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// ExecuteCount executes a count query
func (e *QueryExecutor) ExecuteCount(ctx context.Context, contentType *schema.ContentType, q *Query) (int, error) {
	builder := NewQueryBuilder(e.db, contentType.ID, contentType.Fields)
	countQuery := builder.BuildCount(q)
	return countQuery.Count(ctx)
}

// AggregateResult holds the result of an aggregation
type AggregateResult struct {
	// GroupKey holds the group by values (if any)
	GroupKey map[string]interface{} `json:"groupKey,omitempty"`

	// Values holds the aggregated values
	Values map[string]interface{} `json:"values"`
}

// ExecuteAggregate executes an aggregation query
func (e *QueryExecutor) ExecuteAggregate(ctx context.Context, contentType *schema.ContentType, q *AggregateQuery) ([]AggregateResult, error) {
	// Build aggregation query
	selectQuery := e.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("ce.content_type_id = ?", contentType.ID).
		Where("ce.deleted_at IS NULL")

	// Apply filters
	if q.Filters != nil {
		builder := NewQueryBuilder(e.db, contentType.ID, contentType.Fields)
		selectQuery = builder.applyFilterGroup(selectQuery, q.Filters)
	}

	// Build select columns for aggregations
	var selectCols []string

	// Add group by columns
	for _, field := range q.GroupBy {
		if isSystemField(field) {
			column := "ce." + toSnakeCase(field)
			selectCols = append(selectCols, column)
			selectQuery = selectQuery.GroupExpr(column)
		} else {
			selectCols = append(selectCols,
				"ce.data->>? AS "+field)
			selectQuery = selectQuery.GroupExpr("ce.data->>?", field)
		}
	}

	// Add aggregation columns
	for _, agg := range q.Aggregations {
		var aggExpr string
		switch agg.Operator {
		case AggCount:
			if agg.Field == "" || agg.Field == "*" {
				aggExpr = "COUNT(*)"
			} else {
				aggExpr = "COUNT(ce.data->>'" + agg.Field + "')"
			}
		case AggSum:
			aggExpr = "SUM((ce.data->>'" + agg.Field + "')::numeric)"
		case AggAvg:
			aggExpr = "AVG((ce.data->>'" + agg.Field + "')::numeric)"
		case AggMin:
			aggExpr = "MIN((ce.data->>'" + agg.Field + "')::numeric)"
		case AggMax:
			aggExpr = "MAX((ce.data->>'" + agg.Field + "')::numeric)"
		}
		selectCols = append(selectCols, aggExpr+" AS "+agg.Alias)
	}

	selectQuery = selectQuery.ColumnExpr(joinStrings(selectCols, ", "))

	// Apply having (filters on aggregated values)
	// Note: This requires special handling for aggregate conditions

	// Apply sorting
	for _, sort := range q.Sort {
		direction := "ASC"
		if sort.Descending {
			direction = "DESC"
		}
		selectQuery = selectQuery.OrderExpr(sort.Field + " " + direction)
	}

	// Apply limit
	if q.Limit > 0 {
		selectQuery = selectQuery.Limit(q.Limit)
	}

	// Execute and scan results
	var results []map[string]interface{}
	rows, err := selectQuery.Rows(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	// Convert to AggregateResult
	var aggResults []AggregateResult
	for _, row := range results {
		aggResult := AggregateResult{
			Values: make(map[string]interface{}),
		}

		// Separate group keys from values
		if len(q.GroupBy) > 0 {
			aggResult.GroupKey = make(map[string]interface{})
			for _, field := range q.GroupBy {
				if val, ok := row[field]; ok {
					aggResult.GroupKey[field] = val
				}
			}
		}

		// Add aggregated values
		for _, agg := range q.Aggregations {
			if val, ok := row[agg.Alias]; ok {
				aggResult.Values[agg.Alias] = val
			}
		}

		aggResults = append(aggResults, aggResult)
	}

	return aggResults, nil
}

// joinStrings joins strings with a separator
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// ExecuteIDs returns just the IDs matching a query
func (e *QueryExecutor) ExecuteIDs(ctx context.Context, contentType *schema.ContentType, q *Query) ([]xid.ID, error) {
	builder := NewQueryBuilder(e.db, contentType.ID, contentType.Fields)
	selectQuery := builder.Build(q)

	// Only select IDs
	selectQuery = selectQuery.Column("ce.id")

	var ids []xid.ID
	err := selectQuery.Scan(ctx, &ids)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

// ExecuteExists checks if any entries match the query
func (e *QueryExecutor) ExecuteExists(ctx context.Context, contentType *schema.ContentType, q *Query) (bool, error) {
	builder := NewQueryBuilder(e.db, contentType.ID, contentType.Fields)
	countQuery := builder.BuildCount(q)

	exists, err := countQuery.Exists(ctx)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// ExecuteDistinct returns distinct values for a field
func (e *QueryExecutor) ExecuteDistinct(ctx context.Context, contentType *schema.ContentType, field string, q *Query) ([]interface{}, error) {
	builder := NewQueryBuilder(e.db, contentType.ID, contentType.Fields)

	// Start with filtered query
	selectQuery := builder.BuildCount(q)

	// Select distinct values
	if isSystemField(field) {
		column := "ce." + toSnakeCase(field)
		selectQuery = selectQuery.ColumnExpr("DISTINCT " + column)
	} else {
		selectQuery = selectQuery.ColumnExpr("DISTINCT ce.data->>?", field)
	}

	// Execute
	var values []interface{}
	err := selectQuery.Scan(ctx, &values)
	if err != nil {
		return nil, err
	}

	return values, nil
}

