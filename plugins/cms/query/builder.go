package query

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/xid"
	"github.com/uptrace/bun"

	"github.com/xraph/authsome/plugins/cms/schema"
)

// QueryBuilder builds Bun queries from parsed Query objects.
type QueryBuilder struct {
	db            *bun.DB
	contentTypeID xid.ID
	fields        map[string]*schema.ContentField
}

// NewQueryBuilder creates a new query builder.
func NewQueryBuilder(db *bun.DB, contentTypeID xid.ID, fields []*schema.ContentField) *QueryBuilder {
	fieldMap := make(map[string]*schema.ContentField)
	for _, f := range fields {
		fieldMap[f.Name] = f
	}

	return &QueryBuilder{
		db:            db,
		contentTypeID: contentTypeID,
		fields:        fieldMap,
	}
}

// Build builds a Bun select query from a parsed Query.
func (b *QueryBuilder) Build(q *Query) *bun.SelectQuery {
	selectQuery := b.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("ce.content_type_id = ?", b.contentTypeID).
		Where("ce.deleted_at IS NULL")

	// Apply status filter (shortcut)
	if q.Status != "" {
		selectQuery = selectQuery.Where("ce.status = ?", q.Status)
	}

	// Apply filters
	if q.Filters != nil {
		selectQuery = b.applyFilterGroup(selectQuery, q.Filters)
	}

	// Apply search
	if q.Search != "" {
		selectQuery = b.applySearch(selectQuery, q.Search)
	}

	// Apply sorting
	if len(q.Sort) > 0 {
		selectQuery = b.applySort(selectQuery, q.Sort)
	} else {
		// Default sort by created_at desc
		selectQuery = selectQuery.Order("ce.created_at DESC")
	}

	// Apply field selection
	if len(q.Select) > 0 {
		selectQuery = b.applySelect(selectQuery, q.Select)
	}

	// Apply pagination
	selectQuery = b.applyPagination(selectQuery, q)

	return selectQuery
}

// BuildCount builds a count query from a parsed Query.
func (b *QueryBuilder) BuildCount(q *Query) *bun.SelectQuery {
	selectQuery := b.db.NewSelect().
		Model((*schema.ContentEntry)(nil)).
		Where("ce.content_type_id = ?", b.contentTypeID).
		Where("ce.deleted_at IS NULL")

	// Apply status filter
	if q.Status != "" {
		selectQuery = selectQuery.Where("ce.status = ?", q.Status)
	}

	// Apply filters
	if q.Filters != nil {
		selectQuery = b.applyFilterGroup(selectQuery, q.Filters)
	}

	// Apply search
	if q.Search != "" {
		selectQuery = b.applySearch(selectQuery, q.Search)
	}

	return selectQuery
}

// applyFilterGroup applies a filter group to the query.
func (b *QueryBuilder) applyFilterGroup(q *bun.SelectQuery, group *FilterGroup) *bun.SelectQuery {
	if group == nil || (len(group.Conditions) == 0 && len(group.Groups) == 0) {
		return q
	}

	switch group.Operator {
	case LogicalOr:
		return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return b.applyOrGroup(sq, group)
		})
	case LogicalNot:
		return q.WhereGroup(" AND NOT ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return b.applyAndGroup(sq, group)
		})
	default: // LogicalAnd
		return b.applyAndGroup(q, group)
	}
}

// applyAndGroup applies conditions with AND logic.
func (b *QueryBuilder) applyAndGroup(q *bun.SelectQuery, group *FilterGroup) *bun.SelectQuery {
	for _, cond := range group.Conditions {
		q = b.applyCondition(q, cond, " AND ")
	}

	for _, subGroup := range group.Groups {
		q = b.applyFilterGroup(q, subGroup)
	}

	return q
}

// applyOrGroup applies conditions with OR logic.
func (b *QueryBuilder) applyOrGroup(q *bun.SelectQuery, group *FilterGroup) *bun.SelectQuery {
	first := true
	for _, cond := range group.Conditions {
		if first {
			q = b.applyCondition(q, cond, " AND ")
			first = false
		} else {
			q = b.applyConditionOr(q, cond)
		}
	}

	for _, subGroup := range group.Groups {
		if first {
			q = b.applyFilterGroup(q, subGroup)
			first = false
		} else {
			q = q.WhereGroup(" OR ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return b.applyFilterGroup(sq, subGroup)
			})
		}
	}

	return q
}

// applyCondition applies a single filter condition.
func (b *QueryBuilder) applyCondition(q *bun.SelectQuery, cond FilterCondition, separator string) *bun.SelectQuery {
	// Check if it's a system field
	if isSystemField(cond.Field) {
		return b.applySystemFieldCondition(q, cond)
	}

	// Get field info for type casting
	field := b.fields[cond.Field]

	// Build JSONB query based on operator
	return b.applyJSONBCondition(q, cond, field)
}

// applyConditionOr applies a single filter condition with OR.
func (b *QueryBuilder) applyConditionOr(q *bun.SelectQuery, cond FilterCondition) *bun.SelectQuery {
	// Check if it's a system field
	if isSystemField(cond.Field) {
		return b.applySystemFieldConditionOr(q, cond)
	}

	// Get field info for type casting
	field := b.fields[cond.Field]

	// Build JSONB query based on operator
	return b.applyJSONBConditionOr(q, cond, field)
}

// applySystemFieldCondition applies a condition on a system field.
func (b *QueryBuilder) applySystemFieldCondition(q *bun.SelectQuery, cond FilterCondition) *bun.SelectQuery {
	// Extract the actual field name (strips _meta. prefix if present)
	fieldName := getSystemFieldName(cond.Field)
	column := "ce." + toSnakeCase(fieldName)

	switch cond.Operator {
	case OpEqual:
		return q.Where(column+" = ?", cond.Value)
	case OpNotEqual:
		return q.Where(column+" != ?", cond.Value)
	case OpGreaterThan:
		return q.Where(column+" > ?", cond.Value)
	case OpGreaterThanEqual:
		return q.Where(column+" >= ?", cond.Value)
	case OpLessThan:
		return q.Where(column+" < ?", cond.Value)
	case OpLessThanEqual:
		return q.Where(column+" <= ?", cond.Value)
	case OpIn:
		return q.Where(column+" IN (?)", bun.In(cond.Value))
	case OpNotIn:
		return q.Where(column+" NOT IN (?)", bun.In(cond.Value))
	case OpNull:
		if toBool(cond.Value) {
			return q.Where(column + " IS NULL")
		}

		return q.Where(column + " IS NOT NULL")
	default:
		return q.Where(column+" = ?", cond.Value)
	}
}

// applySystemFieldConditionOr applies a condition on a system field with OR.
func (b *QueryBuilder) applySystemFieldConditionOr(q *bun.SelectQuery, cond FilterCondition) *bun.SelectQuery {
	// Extract the actual field name (strips _meta. prefix if present)
	fieldName := getSystemFieldName(cond.Field)
	column := "ce." + toSnakeCase(fieldName)

	switch cond.Operator {
	case OpEqual:
		return q.WhereOr(column+" = ?", cond.Value)
	case OpNotEqual:
		return q.WhereOr(column+" != ?", cond.Value)
	case OpIn:
		return q.WhereOr(column+" IN (?)", bun.In(cond.Value))
	case OpNull:
		if toBool(cond.Value) {
			return q.WhereOr(column + " IS NULL")
		}

		return q.WhereOr(column + " IS NOT NULL")
	default:
		return q.WhereOr(column+" = ?", cond.Value)
	}
}

// applyJSONBCondition applies a condition on a JSONB field.
func (b *QueryBuilder) applyJSONBCondition(q *bun.SelectQuery, cond FilterCondition, field *schema.ContentField) *bun.SelectQuery {
	// Determine if we need type casting
	castType := b.getTypeCast(field)

	switch cond.Operator {
	case OpEqual:
		if castType != "" {
			return q.Where(fmt.Sprintf("(ce.data->>?)%s = ?", castType), cond.Field, cond.Value)
		}

		return q.Where("ce.data->>? = ?", cond.Field, toString(cond.Value))

	case OpNotEqual:
		if castType != "" {
			return q.Where(fmt.Sprintf("(ce.data->>?)%s != ?", castType), cond.Field, cond.Value)
		}

		return q.Where("ce.data->>? != ?", cond.Field, toString(cond.Value))

	case OpGreaterThan:
		return q.Where(fmt.Sprintf("(ce.data->>?)%s > ?", castType), cond.Field, cond.Value)

	case OpGreaterThanEqual:
		return q.Where(fmt.Sprintf("(ce.data->>?)%s >= ?", castType), cond.Field, cond.Value)

	case OpLessThan:
		return q.Where(fmt.Sprintf("(ce.data->>?)%s < ?", castType), cond.Field, cond.Value)

	case OpLessThanEqual:
		return q.Where(fmt.Sprintf("(ce.data->>?)%s <= ?", castType), cond.Field, cond.Value)

	case OpLike:
		return q.Where("ce.data->>? LIKE ?", cond.Field, cond.Value)

	case OpILike:
		return q.Where("ce.data->>? ILIKE ?", cond.Field, cond.Value)

	case OpContains:
		return q.Where("ce.data->>? ILIKE ?", cond.Field, "%"+toString(cond.Value)+"%")

	case OpStartsWith:
		return q.Where("ce.data->>? LIKE ?", cond.Field, toString(cond.Value)+"%")

	case OpEndsWith:
		return q.Where("ce.data->>? LIKE ?", cond.Field, "%"+toString(cond.Value))

	case OpIn:
		values := toStringSlice(cond.Value)

		return q.Where("ce.data->>? IN (?)", cond.Field, bun.In(values))

	case OpNotIn:
		values := toStringSlice(cond.Value)

		return q.Where("ce.data->>? NOT IN (?)", cond.Field, bun.In(values))

	case OpNull:
		if toBool(cond.Value) {
			return q.Where("(ce.data->>? IS NULL OR ce.data->>? = 'null' OR ce.data->>? = '')",
				cond.Field, cond.Field, cond.Field)
		}

		return q.Where("ce.data->>? IS NOT NULL AND ce.data->>? != 'null' AND ce.data->>? != ''",
			cond.Field, cond.Field, cond.Field)

	case OpExists:
		if toBool(cond.Value) {
			return q.Where("ce.data ? ?", cond.Field)
		}

		return q.Where("NOT (ce.data ? ?)", cond.Field)

	case OpJsonContains:
		return q.Where("ce.data->? @> ?", cond.Field, cond.Value)

	case OpJsonHasKey:
		return q.Where("ce.data->? ? ?", cond.Field, cond.Value)

	case OpBetween:
		values := toSlice(cond.Value)
		if len(values) >= 2 {
			return q.Where(fmt.Sprintf("(ce.data->>?)%s BETWEEN ? AND ?", castType),
				cond.Field, values[0], values[1])
		}

		return q

	case OpAll:
		// Array field contains all values
		values := toStringSlice(cond.Value)
		for _, v := range values {
			q = q.Where("ce.data->? @> ?", cond.Field, fmt.Sprintf(`["%s"]`, v))
		}

		return q

	case OpAny:
		// Array field contains any value
		values := toStringSlice(cond.Value)

		return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			for i, v := range values {
				if i == 0 {
					sq = sq.Where("ce.data->? @> ?", cond.Field, fmt.Sprintf(`["%s"]`, v))
				} else {
					sq = sq.WhereOr("ce.data->? @> ?", cond.Field, fmt.Sprintf(`["%s"]`, v))
				}
			}

			return sq
		})

	default:
		return q.Where("ce.data->>? = ?", cond.Field, toString(cond.Value))
	}
}

// applyJSONBConditionOr applies a JSONB condition with OR.
func (b *QueryBuilder) applyJSONBConditionOr(q *bun.SelectQuery, cond FilterCondition, field *schema.ContentField) *bun.SelectQuery {
	switch cond.Operator {
	case OpEqual:
		return q.WhereOr("ce.data->>? = ?", cond.Field, toString(cond.Value))
	case OpContains:
		return q.WhereOr("ce.data->>? ILIKE ?", cond.Field, "%"+toString(cond.Value)+"%")
	default:
		return q.WhereOr("ce.data->>? = ?", cond.Field, toString(cond.Value))
	}
}

// getTypeCast returns the PostgreSQL type cast for a field.
func (b *QueryBuilder) getTypeCast(field *schema.ContentField) string {
	if field == nil {
		return ""
	}

	switch field.Type {
	case "number", "float", "decimal":
		return "::numeric"
	case "integer", "bigInteger":
		return "::bigint"
	case "boolean":
		return "::boolean"
	case "date":
		return "::date"
	case "datetime":
		return "::timestamp"
	default:
		return ""
	}
}

// applySearch applies full-text search.
func (b *QueryBuilder) applySearch(q *bun.SelectQuery, search string) *bun.SelectQuery {
	// Get searchable fields
	var searchableFields []string

	for slug, field := range b.fields {
		if field.IsSearchable() {
			searchableFields = append(searchableFields, slug)
		}
	}

	if len(searchableFields) == 0 {
		return q
	}

	searchPattern := "%" + strings.ToLower(search) + "%"

	return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		for i, field := range searchableFields {
			if i == 0 {
				sq = sq.Where("LOWER(ce.data->>?) LIKE ?", field, searchPattern)
			} else {
				sq = sq.WhereOr("LOWER(ce.data->>?) LIKE ?", field, searchPattern)
			}
		}

		return sq
	})
}

// applySort applies sorting to the query.
func (b *QueryBuilder) applySort(q *bun.SelectQuery, sorts []SortField) *bun.SelectQuery {
	for _, sort := range sorts {
		direction := "ASC"
		if sort.Descending {
			direction = "DESC"
		}

		if isSystemField(sort.Field) {
			// Extract actual field name (strips _meta. prefix if present)
			fieldName := getSystemFieldName(sort.Field)
			column := "ce." + toSnakeCase(fieldName)
			q = q.OrderExpr(fmt.Sprintf("%s %s NULLS LAST", column, direction))
		} else {
			// Sort by JSONB field
			field := b.fields[sort.Field]

			cast := b.getTypeCast(field)
			if cast != "" {
				q = q.OrderExpr(fmt.Sprintf("(ce.data->>?)%s %s NULLS LAST", cast, direction), sort.Field)
			} else {
				q = q.OrderExpr(fmt.Sprintf("ce.data->>? %s NULLS LAST", direction), sort.Field)
			}
		}
	}

	return q
}

// applySelect applies field selection (projection).
func (b *QueryBuilder) applySelect(q *bun.SelectQuery, fields []string) *bun.SelectQuery {
	// Always include system fields
	q = q.Column("ce.id", "ce.content_type_id", "ce.app_id", "ce.environment_id",
		"ce.status", "ce.version", "ce.published_at", "ce.scheduled_at",
		"ce.created_by", "ce.updated_by", "ce.created_at", "ce.updated_at")

	// For JSONB, we need to construct the data field with only selected fields
	// This is complex in PostgreSQL, so for now we return all data
	// A more sophisticated implementation would use jsonb_build_object
	q = q.Column("ce.data")

	return q
}

// applyPagination applies pagination to the query.
func (b *QueryBuilder) applyPagination(q *bun.SelectQuery, query *Query) *bun.SelectQuery {
	// Prefer offset/limit if both are set
	if query.Offset > 0 || query.Limit > 0 {
		if query.Offset > 0 {
			q = q.Offset(query.Offset)
		}

		if query.Limit > 0 {
			q = q.Limit(query.Limit)
		}

		return q
	}

	// Fall back to page-based pagination
	page := query.Page
	if page <= 0 {
		page = 1
	}

	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	return q.Offset(offset).Limit(pageSize)
}

// =============================================================================
// Helper functions
// =============================================================================

// toSnakeCase converts camelCase to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}

		if r >= 'A' && r <= 'Z' {
			result.WriteByte(byte(r + 32))
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// toString converts a value to string.
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	case float64:
		return fmt.Sprintf("%v", val)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// toBool converts a value to boolean.
func toBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1"
	case int:
		return val != 0
	case float64:
		return val != 0
	default:
		return false
	}
}

// toSlice converts a value to a slice.
func toSlice(v any) []any {
	switch val := v.(type) {
	case []any:
		return val
	case []string:
		result := make([]any, len(val))
		for i, s := range val {
			result[i] = s
		}

		return result
	default:
		return []any{v}
	}
}

// toStringSlice converts a value to a string slice.
func toStringSlice(v any) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = toString(item)
		}

		return result
	case string:
		return []string{val}
	default:
		return []string{toString(v)}
	}
}
