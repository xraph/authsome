package query

// OperatorInfo provides metadata about an operator
type OperatorInfo struct {
	// Operator is the operator code
	Operator FilterOperator `json:"operator"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Description describes what the operator does
	Description string `json:"description"`

	// Example shows usage example
	Example string `json:"example"`

	// ApplicableTo lists field types this operator can be used with
	ApplicableTo []string `json:"applicableTo"`

	// ValueType describes the expected value type
	ValueType string `json:"valueType"`
}

// GetAllOperators returns information about all available operators
func GetAllOperators() []OperatorInfo {
	return []OperatorInfo{
		// Comparison operators
		{
			Operator:     OpEqual,
			Name:         "Equals",
			Description:  "Field value equals the specified value",
			Example:      "filter[status]=eq.published",
			ApplicableTo: []string{"all"},
			ValueType:    "any",
		},
		{
			Operator:     OpNotEqual,
			Name:         "Not Equals",
			Description:  "Field value does not equal the specified value",
			Example:      "filter[status]=ne.draft",
			ApplicableTo: []string{"all"},
			ValueType:    "any",
		},
		{
			Operator:     OpGreaterThan,
			Name:         "Greater Than",
			Description:  "Field value is greater than the specified value",
			Example:      "filter[price]=gt.100",
			ApplicableTo: []string{"number", "integer", "float", "decimal", "date", "datetime"},
			ValueType:    "number or date",
		},
		{
			Operator:     OpGreaterThanEqual,
			Name:         "Greater Than or Equal",
			Description:  "Field value is greater than or equal to the specified value",
			Example:      "filter[price]=gte.100",
			ApplicableTo: []string{"number", "integer", "float", "decimal", "date", "datetime"},
			ValueType:    "number or date",
		},
		{
			Operator:     OpLessThan,
			Name:         "Less Than",
			Description:  "Field value is less than the specified value",
			Example:      "filter[price]=lt.100",
			ApplicableTo: []string{"number", "integer", "float", "decimal", "date", "datetime"},
			ValueType:    "number or date",
		},
		{
			Operator:     OpLessThanEqual,
			Name:         "Less Than or Equal",
			Description:  "Field value is less than or equal to the specified value",
			Example:      "filter[price]=lte.100",
			ApplicableTo: []string{"number", "integer", "float", "decimal", "date", "datetime"},
			ValueType:    "number or date",
		},

		// String operators
		{
			Operator:     OpLike,
			Name:         "Like (Case Sensitive)",
			Description:  "Field value matches pattern (use % as wildcard)",
			Example:      "filter[title]=like.%hello%",
			ApplicableTo: []string{"text", "textarea", "richText", "markdown", "email", "url"},
			ValueType:    "string with wildcards",
		},
		{
			Operator:     OpILike,
			Name:         "Like (Case Insensitive)",
			Description:  "Field value matches pattern case-insensitively",
			Example:      "filter[title]=ilike.%hello%",
			ApplicableTo: []string{"text", "textarea", "richText", "markdown", "email", "url"},
			ValueType:    "string with wildcards",
		},
		{
			Operator:     OpContains,
			Name:         "Contains",
			Description:  "Field value contains the specified substring",
			Example:      "filter[description]=contains.important",
			ApplicableTo: []string{"text", "textarea", "richText", "markdown"},
			ValueType:    "string",
		},
		{
			Operator:     OpStartsWith,
			Name:         "Starts With",
			Description:  "Field value starts with the specified string",
			Example:      "filter[title]=startsWith.How",
			ApplicableTo: []string{"text", "textarea", "email", "url", "slug"},
			ValueType:    "string",
		},
		{
			Operator:     OpEndsWith,
			Name:         "Ends With",
			Description:  "Field value ends with the specified string",
			Example:      "filter[email]=endsWith.@example.com",
			ApplicableTo: []string{"text", "textarea", "email", "url"},
			ValueType:    "string",
		},

		// Array operators
		{
			Operator:     OpIn,
			Name:         "In",
			Description:  "Field value is in the specified array",
			Example:      "filter[status]=in.(draft,review,published)",
			ApplicableTo: []string{"all"},
			ValueType:    "array",
		},
		{
			Operator:     OpNotIn,
			Name:         "Not In",
			Description:  "Field value is not in the specified array",
			Example:      "filter[status]=nin.(archived,deleted)",
			ApplicableTo: []string{"all"},
			ValueType:    "array",
		},
		{
			Operator:     OpAll,
			Name:         "All",
			Description:  "Array field contains all specified values",
			Example:      "filter[tags]=all.(featured,news)",
			ApplicableTo: []string{"multiSelect", "relation"},
			ValueType:    "array",
		},
		{
			Operator:     OpAny,
			Name:         "Any",
			Description:  "Array field contains any of the specified values",
			Example:      "filter[tags]=any.(featured,popular)",
			ApplicableTo: []string{"multiSelect", "relation"},
			ValueType:    "array",
		},

		// Null operators
		{
			Operator:     OpNull,
			Name:         "Is Null",
			Description:  "Field value is null (true) or not null (false)",
			Example:      "filter[deletedAt]=null.true",
			ApplicableTo: []string{"all"},
			ValueType:    "boolean",
		},
		{
			Operator:     OpExists,
			Name:         "Exists",
			Description:  "Field exists in the entry data",
			Example:      "filter[metadata]=exists.true",
			ApplicableTo: []string{"json"},
			ValueType:    "boolean",
		},

		// JSON operators
		{
			Operator:     OpJsonContains,
			Name:         "JSON Contains",
			Description:  "JSON field contains the specified object",
			Example:      `filter[metadata]=jsonContains.{"active":true}`,
			ApplicableTo: []string{"json"},
			ValueType:    "json",
		},
		{
			Operator:     OpJsonHasKey,
			Name:         "JSON Has Key",
			Description:  "JSON field has the specified key",
			Example:      "filter[metadata]=jsonHasKey.version",
			ApplicableTo: []string{"json"},
			ValueType:    "string",
		},

		// Range operators
		{
			Operator:     OpBetween,
			Name:         "Between",
			Description:  "Field value is between two values (inclusive)",
			Example:      "filter[price]=between.(10,100)",
			ApplicableTo: []string{"number", "integer", "float", "decimal", "date", "datetime"},
			ValueType:    "array of two values",
		},
	}
}

// GetOperatorInfo returns information about a specific operator
func GetOperatorInfo(op FilterOperator) *OperatorInfo {
	for _, info := range GetAllOperators() {
		if info.Operator == op {
			return &info
		}
	}
	return nil
}

// GetOperatorsForFieldType returns operators applicable to a field type
func GetOperatorsForFieldType(fieldType string) []OperatorInfo {
	var applicable []OperatorInfo
	for _, info := range GetAllOperators() {
		for _, ft := range info.ApplicableTo {
			if ft == "all" || ft == fieldType {
				applicable = append(applicable, info)
				break
			}
		}
	}
	return applicable
}

// ParseOperator parses an operator string
func ParseOperator(s string) (FilterOperator, bool) {
	op := FilterOperator(s)
	return op, op.IsValid()
}

// OperatorAliases maps common aliases to operators
var OperatorAliases = map[string]FilterOperator{
	"=":           OpEqual,
	"==":          OpEqual,
	"eq":          OpEqual,
	"equals":      OpEqual,
	"!=":          OpNotEqual,
	"<>":          OpNotEqual,
	"ne":          OpNotEqual,
	"neq":         OpNotEqual,
	"notEquals":   OpNotEqual,
	">":           OpGreaterThan,
	"gt":          OpGreaterThan,
	">=":          OpGreaterThanEqual,
	"gte":         OpGreaterThanEqual,
	"<":           OpLessThan,
	"lt":          OpLessThan,
	"<=":          OpLessThanEqual,
	"lte":         OpLessThanEqual,
	"like":        OpLike,
	"ilike":       OpILike,
	"contains":    OpContains,
	"startsWith":  OpStartsWith,
	"endsWith":    OpEndsWith,
	"in":          OpIn,
	"notIn":       OpNotIn,
	"nin":         OpNotIn,
	"all":         OpAll,
	"any":         OpAny,
	"null":        OpNull,
	"isNull":      OpNull,
	"exists":      OpExists,
	"between":     OpBetween,
	"jsonContains": OpJsonContains,
	"jsonHasKey":  OpJsonHasKey,
}

// ResolveOperator resolves an operator string to a FilterOperator
func ResolveOperator(s string) (FilterOperator, bool) {
	// First try direct parse
	op := FilterOperator(s)
	if op.IsValid() {
		return op, true
	}

	// Then try aliases
	if alias, ok := OperatorAliases[s]; ok {
		return alias, true
	}

	return "", false
}

