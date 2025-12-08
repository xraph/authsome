// Package query provides a query language parser and builder for the CMS plugin.
// It supports URL-based and JSON-based query syntax for filtering, sorting, and pagination.
package query

import (
	"strings"

	"github.com/xraph/authsome/plugins/cms/core"
)

// Query represents a parsed content query
type Query struct {
	// Filters contains all filter conditions
	Filters *FilterGroup `json:"filters,omitempty"`

	// Sort specifies the sort order
	Sort []SortField `json:"sort,omitempty"`

	// Select specifies which fields to return (projection)
	Select []string `json:"select,omitempty"`

	// Populate specifies which relations to populate
	Populate []PopulateOption `json:"populate,omitempty"`

	// Pagination options
	Page     int `json:"page,omitempty"`
	PageSize int `json:"pageSize,omitempty"`
	Offset   int `json:"offset,omitempty"`
	Limit    int `json:"limit,omitempty"`

	// Search for full-text search
	Search string `json:"search,omitempty"`

	// Status filter (shortcut)
	Status string `json:"status,omitempty"`
}

// FilterGroup represents a group of filters with a logical operator
type FilterGroup struct {
	// Operator is AND, OR, or NOT
	Operator LogicalOperator `json:"operator,omitempty"`

	// Conditions are the filter conditions in this group
	Conditions []FilterCondition `json:"conditions,omitempty"`

	// Groups are nested filter groups (for complex queries)
	Groups []*FilterGroup `json:"groups,omitempty"`
}

// FilterCondition represents a single filter condition
type FilterCondition struct {
	// Field is the field name to filter on
	Field string `json:"field"`

	// Operator is the comparison operator
	Operator FilterOperator `json:"operator"`

	// Value is the value to compare against
	Value interface{} `json:"value"`

	// Type hint for value parsing (optional)
	Type string `json:"type,omitempty"`
}

// SortField represents a field to sort by
type SortField struct {
	// Field is the field name to sort by
	Field string `json:"field"`

	// Descending is true for DESC, false for ASC
	Descending bool `json:"descending"`
}

// PopulateOption specifies how to populate a relation
type PopulateOption struct {
	// Path is the field path to populate (e.g., "author" or "author.avatar")
	Path string `json:"path"`

	// Select specifies which fields to include from the related entity
	Select []string `json:"select,omitempty"`

	// Populate nested relations
	Populate []PopulateOption `json:"populate,omitempty"`
}

// AggregateQuery represents an aggregation query
type AggregateQuery struct {
	// GroupBy specifies fields to group by
	GroupBy []string `json:"groupBy,omitempty"`

	// Aggregations specifies aggregation operations
	Aggregations []Aggregation `json:"aggregations"`

	// Filters to apply before aggregation
	Filters *FilterGroup `json:"filters,omitempty"`

	// Having conditions (filters on aggregated values)
	Having *FilterGroup `json:"having,omitempty"`

	// Sort the aggregation results
	Sort []SortField `json:"sort,omitempty"`

	// Limit the number of results
	Limit int `json:"limit,omitempty"`
}

// Aggregation represents a single aggregation operation
type Aggregation struct {
	// Operator is the aggregation function (count, sum, avg, min, max)
	Operator AggregateOperator `json:"operator"`

	// Field is the field to aggregate (not needed for count)
	Field string `json:"field,omitempty"`

	// Alias is the name for the aggregated value in the result
	Alias string `json:"alias"`
}

// =============================================================================
// Operators
// =============================================================================

// LogicalOperator represents a logical operator for combining filters
type LogicalOperator string

const (
	LogicalAnd LogicalOperator = "and"
	LogicalOr  LogicalOperator = "or"
	LogicalNot LogicalOperator = "not"
)

// IsValid returns true if the operator is valid
func (op LogicalOperator) IsValid() bool {
	switch op {
	case LogicalAnd, LogicalOr, LogicalNot:
		return true
	default:
		return false
	}
}

// FilterOperator represents a comparison operator
type FilterOperator string

const (
	// Comparison operators
	OpEqual            FilterOperator = "eq"
	OpNotEqual         FilterOperator = "ne"
	OpGreaterThan      FilterOperator = "gt"
	OpGreaterThanEqual FilterOperator = "gte"
	OpLessThan         FilterOperator = "lt"
	OpLessThanEqual    FilterOperator = "lte"

	// String operators
	OpLike       FilterOperator = "like"     // Case-sensitive pattern match
	OpILike      FilterOperator = "ilike"    // Case-insensitive pattern match
	OpContains   FilterOperator = "contains" // String contains
	OpStartsWith FilterOperator = "startsWith"
	OpEndsWith   FilterOperator = "endsWith"

	// Array operators
	OpIn    FilterOperator = "in"  // Value is in array
	OpNotIn FilterOperator = "nin" // Value is not in array
	OpAll   FilterOperator = "all" // Array contains all values
	OpAny   FilterOperator = "any" // Array contains any value

	// Null operators
	OpNull   FilterOperator = "null"   // Field is null (value: true) or not null (value: false)
	OpExists FilterOperator = "exists" // Field exists (for JSON fields)

	// JSON operators
	OpJsonContains FilterOperator = "jsonContains" // JSON contains
	OpJsonHasKey   FilterOperator = "jsonHasKey"   // JSON has key

	// Date operators
	OpBetween FilterOperator = "between" // Value is between two values
)

// IsValid returns true if the operator is valid
func (op FilterOperator) IsValid() bool {
	switch op {
	case OpEqual, OpNotEqual, OpGreaterThan, OpGreaterThanEqual, OpLessThan, OpLessThanEqual,
		OpLike, OpILike, OpContains, OpStartsWith, OpEndsWith,
		OpIn, OpNotIn, OpAll, OpAny,
		OpNull, OpExists,
		OpJsonContains, OpJsonHasKey,
		OpBetween:
		return true
	default:
		return false
	}
}

// RequiresValue returns true if the operator requires a value
func (op FilterOperator) RequiresValue() bool {
	switch op {
	case OpNull, OpExists:
		return false // These can have optional boolean values
	default:
		return true
	}
}

// AggregateOperator represents an aggregation function
type AggregateOperator string

const (
	AggCount AggregateOperator = "count"
	AggSum   AggregateOperator = "sum"
	AggAvg   AggregateOperator = "avg"
	AggMin   AggregateOperator = "min"
	AggMax   AggregateOperator = "max"
)

// IsValid returns true if the operator is valid
func (op AggregateOperator) IsValid() bool {
	switch op {
	case AggCount, AggSum, AggAvg, AggMin, AggMax:
		return true
	default:
		return false
	}
}

// =============================================================================
// Helper Methods
// =============================================================================

// NewQuery creates a new empty query
func NewQuery() *Query {
	return &Query{
		Page:     1,
		PageSize: 20,
	}
}

// AddFilter adds a filter condition to the query
func (q *Query) AddFilter(field string, operator FilterOperator, value interface{}) *Query {
	if q.Filters == nil {
		q.Filters = &FilterGroup{Operator: LogicalAnd}
	}
	q.Filters.Conditions = append(q.Filters.Conditions, FilterCondition{
		Field:    field,
		Operator: operator,
		Value:    value,
	})
	return q
}

// AddSort adds a sort field to the query
func (q *Query) AddSort(field string, descending bool) *Query {
	q.Sort = append(q.Sort, SortField{
		Field:      field,
		Descending: descending,
	})
	return q
}

// AddSelect adds fields to the select list
func (q *Query) AddSelect(fields ...string) *Query {
	q.Select = append(q.Select, fields...)
	return q
}

// AddPopulate adds a relation to populate
func (q *Query) AddPopulate(path string, selectFields ...string) *Query {
	q.Populate = append(q.Populate, PopulateOption{
		Path:   path,
		Select: selectFields,
	})
	return q
}

// SetPagination sets pagination options
func (q *Query) SetPagination(page, pageSize int) *Query {
	q.Page = page
	q.PageSize = pageSize
	return q
}

// SetOffsetLimit sets offset-based pagination
func (q *Query) SetOffsetLimit(offset, limit int) *Query {
	q.Offset = offset
	q.Limit = limit
	return q
}

// Validate validates the query
func (q *Query) Validate(fields map[string]*core.ContentFieldDTO) error {
	// Validate filters
	if q.Filters != nil {
		if err := q.validateFilterGroup(q.Filters, fields); err != nil {
			return err
		}
	}

	// Validate sort fields
	for _, sort := range q.Sort {
		if !isValidSortField(sort.Field, fields) {
			return core.ErrInvalidSort(sort.Field, "unknown field")
		}
	}

	// Validate select fields
	for _, field := range q.Select {
		if _, exists := fields[field]; !exists && !isSystemField(field) {
			return core.ErrInvalidQuery("unknown select field: " + field)
		}
	}

	return nil
}

// validateFilterGroup validates a filter group recursively
func (q *Query) validateFilterGroup(group *FilterGroup, fields map[string]*core.ContentFieldDTO) error {
	for _, cond := range group.Conditions {
		if !cond.Operator.IsValid() {
			return core.ErrInvalidOperator(string(cond.Operator))
		}
		if _, exists := fields[cond.Field]; !exists && !isSystemField(cond.Field) {
			return core.ErrInvalidFilter(cond.Field, "unknown field")
		}
	}

	for _, nested := range group.Groups {
		if err := q.validateFilterGroup(nested, fields); err != nil {
			return err
		}
	}

	return nil
}

// MetaPrefix is the prefix used for system/internal fields to avoid conflicts with user-defined fields
const MetaPrefix = "_meta."

// SystemFields lists all valid system field names (without the _meta prefix)
var SystemFields = map[string]bool{
	"id":          true,
	"status":      true,
	"version":     true,
	"createdAt":   true,
	"updatedAt":   true,
	"publishedAt": true,
	"scheduledAt": true,
	"createdBy":   true,
	"updatedBy":   true,
}

// IsSystemField returns true if the field is a system field (not user-defined)
// Supports both legacy format (e.g., "status") and new meta format (e.g., "_meta.status")
func IsSystemField(field string) bool {
	// Check for _meta prefixed fields
	if strings.HasPrefix(field, MetaPrefix) {
		actualField := strings.TrimPrefix(field, MetaPrefix)
		return SystemFields[actualField]
	}
	// Legacy support: direct field names
	return SystemFields[field]
}

// isSystemField is an internal alias for backward compatibility within the package
func isSystemField(field string) bool {
	return IsSystemField(field)
}

// IsMetaField returns true if the field uses the _meta prefix format
func IsMetaField(field string) bool {
	return strings.HasPrefix(field, MetaPrefix)
}

// GetSystemFieldName extracts the actual field name from a potentially _meta prefixed field
// e.g., "_meta.status" -> "status", "status" -> "status"
func GetSystemFieldName(field string) string {
	if strings.HasPrefix(field, MetaPrefix) {
		return strings.TrimPrefix(field, MetaPrefix)
	}
	return field
}

// getSystemFieldName is an internal alias for backward compatibility within the package
func getSystemFieldName(field string) string {
	return GetSystemFieldName(field)
}

// isValidSortField returns true if the field can be sorted
func isValidSortField(field string, fields map[string]*core.ContentFieldDTO) bool {
	if isSystemField(field) {
		return true
	}
	_, exists := fields[field]
	return exists
}
