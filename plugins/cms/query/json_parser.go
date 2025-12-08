package query

import (
	"encoding/json"

	"github.com/xraph/authsome/plugins/cms/core"
)

// JSONParser parses JSON query bodies into Query objects
type JSONParser struct{}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// JSONQuery represents a JSON query body
type JSONQuery struct {
	// Filters can be a simple object or complex nested structure
	// Simple: {"status": "published", "title": {"$contains": "hello"}}
	// Complex: {"$and": [{"status": "published"}, {"$or": [...]}]}
	Filters interface{} `json:"filters,omitempty"`

	// Filter is an alias for filters (singular form)
	Filter interface{} `json:"filter,omitempty"`

	// Alternative filter syntax
	Where interface{} `json:"where,omitempty"`

	// Sort can be a string or array
	// String: "-createdAt" or "createdAt:desc"
	// Array: ["-createdAt", "title"]
	Sort interface{} `json:"sort,omitempty"`

	// Select fields to return
	Select []string `json:"select,omitempty"`

	// Fields is an alias for select
	Fields []string `json:"fields,omitempty"`

	// Populate relations
	Populate interface{} `json:"populate,omitempty"`

	// Include is an alias for populate
	Include interface{} `json:"include,omitempty"`

	// Pagination
	Page     int `json:"page,omitempty"`
	PageSize int `json:"pageSize,omitempty"`
	PerPage  int `json:"perPage,omitempty"`
	Offset   int `json:"offset,omitempty"`
	Limit    int `json:"limit,omitempty"`

	// Search for full-text search
	Search string `json:"search,omitempty"`
	Q      string `json:"q,omitempty"`

	// Status shorthand
	Status string `json:"status,omitempty"`
}

// Parse parses JSON data into a Query object
func (p *JSONParser) Parse(data []byte) (*Query, error) {
	var jq JSONQuery
	if err := json.Unmarshal(data, &jq); err != nil {
		return nil, core.ErrInvalidQuery("invalid JSON: " + err.Error())
	}

	return p.ParseJSONQuery(&jq)
}

// ParseJSONQuery parses a JSONQuery struct into a Query object
func (p *JSONParser) ParseJSONQuery(jq *JSONQuery) (*Query, error) {
	q := NewQuery()

	// Parse filters (use filters, filter, or where - in order of precedence)
	filters := jq.Filters
	if filters == nil {
		filters = jq.Filter
	}
	if filters == nil {
		filters = jq.Where
	}
	if filters != nil {
		filterGroup, err := p.parseFilters(filters)
		if err != nil {
			return nil, err
		}
		q.Filters = filterGroup
	}

	// Parse sort
	if jq.Sort != nil {
		q.Sort = p.parseSort(jq.Sort)
	}

	// Parse select (use either select or fields)
	if len(jq.Select) > 0 {
		q.Select = jq.Select
	} else if len(jq.Fields) > 0 {
		q.Select = jq.Fields
	}

	// Parse populate (use either populate or include)
	populate := jq.Populate
	if populate == nil {
		populate = jq.Include
	}
	if populate != nil {
		q.Populate = p.parsePopulate(populate)
	}

	// Parse pagination
	if jq.Page > 0 {
		q.Page = jq.Page
	}
	if jq.PageSize > 0 {
		q.PageSize = jq.PageSize
	} else if jq.PerPage > 0 {
		q.PageSize = jq.PerPage
	} else if jq.Limit > 0 && jq.Offset >= 0 {
		// Convert offset/limit to page-based
		q.Limit = jq.Limit
		q.Offset = jq.Offset
	}

	// Parse search
	if jq.Search != "" {
		q.Search = jq.Search
	} else if jq.Q != "" {
		q.Search = jq.Q
	}

	// Status shorthand
	if jq.Status != "" {
		q.Status = jq.Status
	}

	return q, nil
}

// parseFilters parses the filters structure
func (p *JSONParser) parseFilters(filters interface{}) (*FilterGroup, error) {
	switch f := filters.(type) {
	case map[string]interface{}:
		return p.parseFilterObject(f)
	case []interface{}:
		// Array at top level is treated as $and
		return p.parseFilterArray(f, LogicalAnd)
	default:
		return nil, core.ErrInvalidFilter("filters", "must be an object or array")
	}
}

// parseFilterObject parses a filter object
func (p *JSONParser) parseFilterObject(obj map[string]interface{}) (*FilterGroup, error) {
	group := &FilterGroup{
		Operator: LogicalAnd,
	}

	for key, value := range obj {
		// Check for logical operators
		switch key {
		case "$and", "and", "AND":
			arr, ok := value.([]interface{})
			if !ok {
				return nil, core.ErrInvalidFilter(key, "must be an array")
			}
			subGroup, err := p.parseFilterArray(arr, LogicalAnd)
			if err != nil {
				return nil, err
			}
			group.Groups = append(group.Groups, subGroup)

		case "$or", "or", "OR":
			arr, ok := value.([]interface{})
			if !ok {
				return nil, core.ErrInvalidFilter(key, "must be an array")
			}
			subGroup, err := p.parseFilterArray(arr, LogicalOr)
			if err != nil {
				return nil, err
			}
			group.Groups = append(group.Groups, subGroup)

		case "$not", "not", "NOT":
			subObj, ok := value.(map[string]interface{})
			if !ok {
				return nil, core.ErrInvalidFilter(key, "must be an object")
			}
			subGroup, err := p.parseFilterObject(subObj)
			if err != nil {
				return nil, err
			}
			subGroup.Operator = LogicalNot
			group.Groups = append(group.Groups, subGroup)

		default:
			// Regular field filter
			cond, err := p.parseFieldFilter(key, value)
			if err != nil {
				return nil, err
			}
			group.Conditions = append(group.Conditions, cond...)
		}
	}

	return group, nil
}

// parseFilterArray parses an array of filter conditions
func (p *JSONParser) parseFilterArray(arr []interface{}, operator LogicalOperator) (*FilterGroup, error) {
	group := &FilterGroup{
		Operator: operator,
	}

	for _, item := range arr {
		switch v := item.(type) {
		case map[string]interface{}:
			subGroup, err := p.parseFilterObject(v)
			if err != nil {
				return nil, err
			}
			// Flatten single-condition groups
			if len(subGroup.Conditions) > 0 || len(subGroup.Groups) > 0 {
				group.Groups = append(group.Groups, subGroup)
			}
		default:
			return nil, core.ErrInvalidFilter("filter array", "items must be objects")
		}
	}

	return group, nil
}

// parseFieldFilter parses a single field filter
func (p *JSONParser) parseFieldFilter(field string, value interface{}) ([]FilterCondition, error) {
	var conditions []FilterCondition

	switch v := value.(type) {
	case map[string]interface{}:
		// Object with operators: {"title": {"$contains": "hello", "$ne": null}}
		for opKey, opValue := range v {
			op, valid := p.resolveJSONOperator(opKey)
			if !valid {
				return nil, core.ErrInvalidOperator(opKey)
			}
			conditions = append(conditions, FilterCondition{
				Field:    field,
				Operator: op,
				Value:    opValue,
			})
		}
	default:
		// Direct value: {"status": "published"}
		conditions = append(conditions, FilterCondition{
			Field:    field,
			Operator: OpEqual,
			Value:    v,
		})
	}

	return conditions, nil
}

// resolveJSONOperator resolves JSON operator keys to FilterOperator
func (p *JSONParser) resolveJSONOperator(key string) (FilterOperator, bool) {
	// Map JSON operators to internal operators
	jsonOperators := map[string]FilterOperator{
		"$eq":           OpEqual,
		"$ne":           OpNotEqual,
		"$gt":           OpGreaterThan,
		"$gte":          OpGreaterThanEqual,
		"$lt":           OpLessThan,
		"$lte":          OpLessThanEqual,
		"$like":         OpLike,
		"$ilike":        OpILike,
		"$contains":     OpContains,
		"$startsWith":   OpStartsWith,
		"$endsWith":     OpEndsWith,
		"$in":           OpIn,
		"$nin":          OpNotIn,
		"$notIn":        OpNotIn,
		"$all":          OpAll,
		"$any":          OpAny,
		"$null":         OpNull,
		"$isNull":       OpNull,
		"$exists":       OpExists,
		"$between":      OpBetween,
		"$jsonContains": OpJsonContains,
		"$jsonHasKey":   OpJsonHasKey,
	}

	if op, ok := jsonOperators[key]; ok {
		return op, true
	}

	// Also try without $ prefix
	return ResolveOperator(key)
}

// parseSort parses sort specification
func (p *JSONParser) parseSort(sort interface{}) []SortField {
	var sorts []SortField

	switch s := sort.(type) {
	case string:
		// Single sort field
		sorts = append(sorts, p.parseSortString(s))
	case []interface{}:
		// Array of sort fields
		for _, item := range s {
			switch v := item.(type) {
			case string:
				sorts = append(sorts, p.parseSortString(v))
			case map[string]interface{}:
				// Object format: {"field": "createdAt", "order": "desc"}
				if field, ok := v["field"].(string); ok {
					sort := SortField{Field: field}
					if order, ok := v["order"].(string); ok {
						sort.Descending = order == "desc" || order == "DESC"
					}
					if dir, ok := v["direction"].(string); ok {
						sort.Descending = dir == "desc" || dir == "DESC"
					}
					sorts = append(sorts, sort)
				}
			}
		}
	case map[string]interface{}:
		// Object format: {"createdAt": "desc", "title": "asc"}
		for field, order := range s {
			sort := SortField{Field: field}
			if orderStr, ok := order.(string); ok {
				sort.Descending = orderStr == "desc" || orderStr == "DESC" || orderStr == "-1"
			}
			if orderNum, ok := order.(float64); ok {
				sort.Descending = orderNum < 0
			}
			sorts = append(sorts, sort)
		}
	}

	return sorts
}

// parseSortString parses a sort string like "-createdAt" or "createdAt:desc"
func (p *JSONParser) parseSortString(s string) SortField {
	sort := SortField{}

	// Check for prefix notation
	if len(s) > 0 && s[0] == '-' {
		sort.Field = s[1:]
		sort.Descending = true
	} else if len(s) > 0 && s[0] == '+' {
		sort.Field = s[1:]
		sort.Descending = false
	} else {
		// Check for suffix notation (field:desc)
		parts := splitFirst(s, ":")
		sort.Field = parts[0]
		if len(parts) > 1 {
			sort.Descending = parts[1] == "desc" || parts[1] == "DESC"
		}
	}

	return sort
}

// parsePopulate parses populate specification
func (p *JSONParser) parsePopulate(populate interface{}) []PopulateOption {
	var options []PopulateOption

	switch pop := populate.(type) {
	case string:
		// Single path or comma-separated
		paths := splitAndTrim(pop, ",")
		for _, path := range paths {
			options = append(options, PopulateOption{Path: path})
		}
	case []interface{}:
		// Array of paths or objects
		for _, item := range pop {
			switch v := item.(type) {
			case string:
				options = append(options, PopulateOption{Path: v})
			case map[string]interface{}:
				opt := p.parsePopulateObject(v)
				if opt.Path != "" {
					options = append(options, opt)
				}
			}
		}
	case map[string]interface{}:
		// Single populate object
		opt := p.parsePopulateObject(pop)
		if opt.Path != "" {
			options = append(options, opt)
		}
	}

	return options
}

// parsePopulateObject parses a populate object
func (p *JSONParser) parsePopulateObject(obj map[string]interface{}) PopulateOption {
	opt := PopulateOption{}

	if path, ok := obj["path"].(string); ok {
		opt.Path = path
	}

	if sel, ok := obj["select"].([]interface{}); ok {
		for _, s := range sel {
			if str, ok := s.(string); ok {
				opt.Select = append(opt.Select, str)
			}
		}
	}

	if pop, ok := obj["populate"]; ok {
		opt.Populate = p.parsePopulate(pop)
	}

	return opt
}

// Helper functions

// splitFirst splits a string at the first occurrence of sep
func splitFirst(s, sep string) []string {
	for i := 0; i < len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return []string{s[:i], s[i+len(sep):]}
		}
	}
	return []string{s}
}

// splitAndTrim splits a string and trims whitespace from each part
func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitFirst(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// trimSpace trims whitespace from a string
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
