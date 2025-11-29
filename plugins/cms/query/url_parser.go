package query

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/xraph/authsome/plugins/cms/core"
)

// URLParser parses query parameters from URLs into Query objects
type URLParser struct{}

// NewURLParser creates a new URL parser
func NewURLParser() *URLParser {
	return &URLParser{}
}

// Parse parses URL query parameters into a Query object
// Supported formats:
//   - filter[field]=op.value (e.g., filter[status]=eq.published)
//   - filter[field]=value (shorthand for eq)
//   - sort=-field (descending) or sort=field (ascending)
//   - page=1&pageSize=20
//   - offset=0&limit=20
//   - select=field1,field2
//   - populate=relation1,relation2
//   - search=text
//   - status=published (shorthand for filter[status]=eq.published)
func (p *URLParser) Parse(values url.Values) (*Query, error) {
	q := NewQuery()

	// Parse filters
	filters, err := p.parseFilters(values)
	if err != nil {
		return nil, err
	}
	if len(filters) > 0 {
		q.Filters = &FilterGroup{
			Operator:   LogicalAnd,
			Conditions: filters,
		}
	}

	// Parse sort
	if sortStr := values.Get("sort"); sortStr != "" {
		q.Sort = p.parseSort(sortStr)
	}

	// Parse pagination
	if pageStr := values.Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err == nil && page > 0 {
			q.Page = page
		}
	}
	if pageSizeStr := values.Get("pageSize"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err == nil && pageSize > 0 {
			q.PageSize = pageSize
		}
	}

	// Also support per_page and limit
	if perPage := values.Get("per_page"); perPage != "" {
		pageSize, err := strconv.Atoi(perPage)
		if err == nil && pageSize > 0 {
			q.PageSize = pageSize
		}
	}

	// Offset/limit pagination
	if offsetStr := values.Get("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err == nil && offset >= 0 {
			q.Offset = offset
		}
	}
	if limitStr := values.Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err == nil && limit > 0 {
			q.Limit = limit
		}
	}

	// Parse select
	if selectStr := values.Get("select"); selectStr != "" {
		q.Select = strings.Split(selectStr, ",")
		for i := range q.Select {
			q.Select[i] = strings.TrimSpace(q.Select[i])
		}
	}

	// Also support fields parameter
	if fieldsStr := values.Get("fields"); fieldsStr != "" {
		q.Select = strings.Split(fieldsStr, ",")
		for i := range q.Select {
			q.Select[i] = strings.TrimSpace(q.Select[i])
		}
	}

	// Parse populate
	if populateStr := values.Get("populate"); populateStr != "" {
		q.Populate = p.parsePopulate(populateStr)
	}

	// Also support include parameter
	if includeStr := values.Get("include"); includeStr != "" {
		q.Populate = p.parsePopulate(includeStr)
	}

	// Parse search
	if search := values.Get("search"); search != "" {
		q.Search = search
	}
	if search := values.Get("q"); search != "" {
		q.Search = search
	}

	// Parse status shorthand
	if status := values.Get("status"); status != "" {
		q.Status = status
	}

	return q, nil
}

// filterPattern matches filter[field]=op.value format
var filterPattern = regexp.MustCompile(`^filter\[([^\]]+)\]$`)

// parseFilters parses filter parameters from URL values
func (p *URLParser) parseFilters(values url.Values) ([]FilterCondition, error) {
	var conditions []FilterCondition

	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}

		// Check if this is a filter parameter
		matches := filterPattern.FindStringSubmatch(key)
		if matches == nil {
			continue
		}

		field := matches[1]
		value := vals[0]

		// Parse operator and value
		cond, err := p.parseFilterValue(field, value)
		if err != nil {
			return nil, err
		}

		conditions = append(conditions, cond)
	}

	return conditions, nil
}

// parseFilterValue parses a filter value in the format "op.value" or just "value"
func (p *URLParser) parseFilterValue(field, value string) (FilterCondition, error) {
	cond := FilterCondition{Field: field}

	// Check for operator prefix (e.g., "eq.value", "gt.100")
	dotIdx := strings.Index(value, ".")
	if dotIdx > 0 {
		opStr := value[:dotIdx]
		op, valid := ResolveOperator(opStr)
		if valid {
			cond.Operator = op
			value = value[dotIdx+1:]
		} else {
			// No valid operator, treat as equality
			cond.Operator = OpEqual
		}
	} else {
		// No operator prefix, default to equality
		cond.Operator = OpEqual
	}

	// Parse the value based on operator
	parsedValue, err := p.parseValue(value, cond.Operator)
	if err != nil {
		return cond, core.ErrInvalidFilter(field, err.Error())
	}
	cond.Value = parsedValue

	return cond, nil
}

// parseValue parses a filter value string into the appropriate type
func (p *URLParser) parseValue(value string, op FilterOperator) (interface{}, error) {
	// Handle array values for in/nin/all/any operators
	if op == OpIn || op == OpNotIn || op == OpAll || op == OpAny || op == OpBetween {
		return p.parseArrayValue(value)
	}

	// Handle boolean values
	if value == "true" {
		return true, nil
	}
	if value == "false" {
		return false, nil
	}

	// Handle null
	if value == "null" || value == "" {
		return nil, nil
	}

	// Try to parse as number
	if num, err := strconv.ParseFloat(value, 64); err == nil {
		// Check if it's an integer
		if num == float64(int64(num)) {
			return int64(num), nil
		}
		return num, nil
	}

	// Return as string
	return value, nil
}

// parseArrayValue parses an array value in format "(val1,val2,val3)"
func (p *URLParser) parseArrayValue(value string) ([]interface{}, error) {
	// Remove parentheses if present
	value = strings.TrimPrefix(value, "(")
	value = strings.TrimSuffix(value, ")")

	// Also handle square brackets
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")

	if value == "" {
		return []interface{}{}, nil
	}

	// Split by comma
	parts := strings.Split(value, ",")
	result := make([]interface{}, len(parts))

	for i, part := range parts {
		part = strings.TrimSpace(part)
		// Parse each value
		parsed, _ := p.parseValue(part, OpEqual)
		result[i] = parsed
	}

	return result, nil
}

// parseSort parses sort parameters (e.g., "-createdAt,title")
func (p *URLParser) parseSort(sortStr string) []SortField {
	var sorts []SortField

	fields := strings.Split(sortStr, ",")
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		sort := SortField{}
		if strings.HasPrefix(field, "-") {
			sort.Field = strings.TrimPrefix(field, "-")
			sort.Descending = true
		} else if strings.HasPrefix(field, "+") {
			sort.Field = strings.TrimPrefix(field, "+")
			sort.Descending = false
		} else {
			sort.Field = field
			sort.Descending = false
		}

		sorts = append(sorts, sort)
	}

	return sorts
}

// parsePopulate parses populate parameters (e.g., "author,category.parent")
func (p *URLParser) parsePopulate(populateStr string) []PopulateOption {
	var populates []PopulateOption

	paths := strings.Split(populateStr, ",")
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		populates = append(populates, PopulateOption{Path: path})
	}

	return populates
}

// ToURLValues converts a Query back to URL query parameters
func (p *URLParser) ToURLValues(q *Query) url.Values {
	values := url.Values{}

	// Add filters
	if q.Filters != nil {
		for _, cond := range q.Filters.Conditions {
			key := "filter[" + cond.Field + "]"
			value := string(cond.Operator) + "." + formatValue(cond.Value)
			values.Add(key, value)
		}
	}

	// Add sort
	if len(q.Sort) > 0 {
		var sortParts []string
		for _, sort := range q.Sort {
			if sort.Descending {
				sortParts = append(sortParts, "-"+sort.Field)
			} else {
				sortParts = append(sortParts, sort.Field)
			}
		}
		values.Set("sort", strings.Join(sortParts, ","))
	}

	// Add pagination
	if q.Page > 0 {
		values.Set("page", strconv.Itoa(q.Page))
	}
	if q.PageSize > 0 {
		values.Set("pageSize", strconv.Itoa(q.PageSize))
	}
	if q.Offset > 0 {
		values.Set("offset", strconv.Itoa(q.Offset))
	}
	if q.Limit > 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}

	// Add select
	if len(q.Select) > 0 {
		values.Set("select", strings.Join(q.Select, ","))
	}

	// Add populate
	if len(q.Populate) > 0 {
		var paths []string
		for _, pop := range q.Populate {
			paths = append(paths, pop.Path)
		}
		values.Set("populate", strings.Join(paths, ","))
	}

	// Add search
	if q.Search != "" {
		values.Set("search", q.Search)
	}

	// Add status
	if q.Status != "" {
		values.Set("status", q.Status)
	}

	return values
}

// formatValue formats a value for URL encoding
func formatValue(value interface{}) string {
	switch v := value.(type) {
	case []interface{}:
		var parts []string
		for _, item := range v {
			parts = append(parts, formatValue(item))
		}
		return "(" + strings.Join(parts, ",") + ")"
	case []string:
		return "(" + strings.Join(v, ",") + ")"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		// For any other type, try to convert to string
		return ""
	}
}

