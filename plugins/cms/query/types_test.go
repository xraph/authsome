package query

import (
	"testing"

	"github.com/xraph/authsome/plugins/cms/core"
)

func TestIsSystemField(t *testing.T) {
	tests := []struct {
		field    string
		expected bool
	}{
		{"id", true},
		{"status", true},
		{"version", true},
		{"createdAt", true},
		{"updatedAt", true},
		{"publishedAt", true},
		{"scheduledAt", true},
		{"createdBy", true},
		{"updatedBy", true},
		{"title", false},
		{"content", false},
		{"customField", false},
		{"data", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := isSystemField(tt.field)
			if result != tt.expected {
				t.Errorf("isSystemField(%q) = %v, expected %v", tt.field, result, tt.expected)
			}
		})
	}
}

func TestIsValidSortField(t *testing.T) {
	fields := map[string]*core.ContentFieldDTO{
		"title":    {Slug: "title"},
		"content":  {Slug: "content"},
		"category": {Slug: "category"},
	}

	tests := []struct {
		field    string
		expected bool
	}{
		// System fields
		{"id", true},
		{"status", true},
		{"createdAt", true},
		{"updatedAt", true},
		// User-defined fields
		{"title", true},
		{"content", true},
		{"category", true},
		// Unknown fields
		{"unknown", false},
		{"nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			result := isValidSortField(tt.field, fields)
			if result != tt.expected {
				t.Errorf("isValidSortField(%q) = %v, expected %v", tt.field, result, tt.expected)
			}
		})
	}
}

func TestNewQuery(t *testing.T) {
	q := NewQuery()

	if q == nil {
		t.Fatal("NewQuery returned nil")
	}

	// NewQuery initializes with default pagination values
	if q.Page != 1 {
		t.Errorf("expected Page 1 (default), got %d", q.Page)
	}

	if q.PageSize != 20 {
		t.Errorf("expected PageSize 20 (default), got %d", q.PageSize)
	}

	if q.Filters != nil {
		t.Error("expected Filters to be nil initially")
	}
}

func TestQuery_Validate(t *testing.T) {
	fields := map[string]*core.ContentFieldDTO{
		"title":    {Slug: "title"},
		"content":  {Slug: "content"},
		"category": {Slug: "category"},
	}

	t.Run("valid query with system field filter", func(t *testing.T) {
		q := NewQuery()
		q.Filters = &FilterGroup{
			Conditions: []FilterCondition{
				{Field: "status", Operator: OpEqual, Value: "published"},
			},
		}

		err := q.Validate(fields)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("valid query with user field filter", func(t *testing.T) {
		q := NewQuery()
		q.Filters = &FilterGroup{
			Conditions: []FilterCondition{
				{Field: "title", Operator: OpContains, Value: "hello"},
			},
		}

		err := q.Validate(fields)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("valid query with select", func(t *testing.T) {
		q := NewQuery()
		q.Select = []string{"title", "status", "createdAt"}

		err := q.Validate(fields)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("invalid filter field", func(t *testing.T) {
		q := NewQuery()
		q.Filters = &FilterGroup{
			Conditions: []FilterCondition{
				{Field: "unknownField", Operator: OpEqual, Value: "value"},
			},
		}

		err := q.Validate(fields)
		if err == nil {
			t.Error("expected error for unknown filter field")
		}
	})

	t.Run("invalid select field", func(t *testing.T) {
		q := NewQuery()
		q.Select = []string{"title", "unknownField"}

		err := q.Validate(fields)
		if err == nil {
			t.Error("expected error for unknown select field")
		}
	})

	t.Run("invalid operator", func(t *testing.T) {
		q := NewQuery()
		q.Filters = &FilterGroup{
			Conditions: []FilterCondition{
				{Field: "title", Operator: FilterOperator("invalid"), Value: "value"},
			},
		}

		err := q.Validate(fields)
		if err == nil {
			t.Error("expected error for invalid operator")
		}
	})

	t.Run("nested filter groups", func(t *testing.T) {
		q := NewQuery()
		q.Filters = &FilterGroup{
			Groups: []*FilterGroup{
				{
					Operator: LogicalOr,
					Conditions: []FilterCondition{
						{Field: "status", Operator: OpEqual, Value: "draft"},
						{Field: "status", Operator: OpEqual, Value: "published"},
					},
				},
			},
		}

		err := q.Validate(fields)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("invalid nested filter field", func(t *testing.T) {
		q := NewQuery()
		q.Filters = &FilterGroup{
			Groups: []*FilterGroup{
				{
					Conditions: []FilterCondition{
						{Field: "unknownField", Operator: OpEqual, Value: "value"},
					},
				},
			},
		}

		err := q.Validate(fields)
		if err == nil {
			t.Error("expected error for unknown nested filter field")
		}
	})

	t.Run("nil filters", func(t *testing.T) {
		q := NewQuery()
		q.Filters = nil

		err := q.Validate(fields)
		if err != nil {
			t.Errorf("expected no error for nil filters, got %v", err)
		}
	})
}

func TestFilterOperator_IsValid(t *testing.T) {
	validOperators := []FilterOperator{
		OpEqual, OpNotEqual, OpGreaterThan, OpGreaterThanEqual,
		OpLessThan, OpLessThanEqual, OpLike, OpILike,
		OpContains, OpStartsWith, OpEndsWith, OpIn, OpNotIn,
		OpAll, OpAny, OpNull, OpExists, OpBetween,
		OpJsonContains, OpJsonHasKey,
	}

	for _, op := range validOperators {
		t.Run(string(op), func(t *testing.T) {
			if !op.IsValid() {
				t.Errorf("expected operator %s to be valid", op)
			}
		})
	}

	t.Run("invalid operator", func(t *testing.T) {
		op := FilterOperator("invalid")
		if op.IsValid() {
			t.Error("expected invalid operator to be invalid")
		}
	})
}

func TestResolveOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected FilterOperator
		valid    bool
	}{
		{"eq", OpEqual, true},
		{"ne", OpNotEqual, true},
		{"gt", OpGreaterThan, true},
		{"gte", OpGreaterThanEqual, true},
		{"lt", OpLessThan, true},
		{"lte", OpLessThanEqual, true},
		{"like", OpLike, true},
		{"ilike", OpILike, true},
		{"contains", OpContains, true},
		{"startsWith", OpStartsWith, true},
		{"endsWith", OpEndsWith, true},
		{"in", OpIn, true},
		{"notIn", OpNotIn, true},
		{"all", OpAll, true},
		{"any", OpAny, true},
		{"null", OpNull, true},
		{"isNull", OpNull, true},
		{"exists", OpExists, true},
		{"between", OpBetween, true},
		{"invalid", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, valid := ResolveOperator(tt.input)
			if valid != tt.valid {
				t.Errorf("ResolveOperator(%q) valid = %v, expected %v", tt.input, valid, tt.valid)
			}
			if tt.valid && result != tt.expected {
				t.Errorf("ResolveOperator(%q) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSortField(t *testing.T) {
	t.Run("ascending sort", func(t *testing.T) {
		sf := SortField{Field: "title", Descending: false}
		if sf.Field != "title" {
			t.Errorf("expected field 'title', got '%s'", sf.Field)
		}
		if sf.Descending {
			t.Error("expected Descending to be false")
		}
	})

	t.Run("descending sort", func(t *testing.T) {
		sf := SortField{Field: "createdAt", Descending: true}
		if sf.Field != "createdAt" {
			t.Errorf("expected field 'createdAt', got '%s'", sf.Field)
		}
		if !sf.Descending {
			t.Error("expected Descending to be true")
		}
	})
}

func TestFilterCondition(t *testing.T) {
	cond := FilterCondition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "published",
	}

	if cond.Field != "status" {
		t.Errorf("expected field 'status', got '%s'", cond.Field)
	}
	if cond.Operator != OpEqual {
		t.Errorf("expected operator OpEqual, got '%s'", cond.Operator)
	}
	if cond.Value != "published" {
		t.Errorf("expected value 'published', got '%v'", cond.Value)
	}
}

func TestFilterGroup(t *testing.T) {
	t.Run("AND group", func(t *testing.T) {
		group := &FilterGroup{
			Operator: LogicalAnd,
			Conditions: []FilterCondition{
				{Field: "status", Operator: OpEqual, Value: "published"},
				{Field: "category", Operator: OpEqual, Value: "news"},
			},
		}

		if group.Operator != LogicalAnd {
			t.Errorf("expected LogicalAnd, got %s", group.Operator)
		}
		if len(group.Conditions) != 2 {
			t.Errorf("expected 2 conditions, got %d", len(group.Conditions))
		}
	})

	t.Run("OR group", func(t *testing.T) {
		group := &FilterGroup{
			Operator: LogicalOr,
			Conditions: []FilterCondition{
				{Field: "status", Operator: OpEqual, Value: "draft"},
				{Field: "status", Operator: OpEqual, Value: "published"},
			},
		}

		if group.Operator != LogicalOr {
			t.Errorf("expected LogicalOr, got %s", group.Operator)
		}
	})

	t.Run("nested groups", func(t *testing.T) {
		group := &FilterGroup{
			Operator: LogicalAnd,
			Groups: []*FilterGroup{
				{
					Operator: LogicalOr,
					Conditions: []FilterCondition{
						{Field: "status", Operator: OpEqual, Value: "draft"},
						{Field: "status", Operator: OpEqual, Value: "published"},
					},
				},
			},
		}

		if len(group.Groups) != 1 {
			t.Errorf("expected 1 nested group, got %d", len(group.Groups))
		}
		if group.Groups[0].Operator != LogicalOr {
			t.Errorf("expected nested group to be LogicalOr, got %s", group.Groups[0].Operator)
		}
	})
}

func TestPopulateOption(t *testing.T) {
	t.Run("simple populate", func(t *testing.T) {
		pop := PopulateOption{Path: "author"}
		if pop.Path != "author" {
			t.Errorf("expected path 'author', got '%s'", pop.Path)
		}
	})

	t.Run("populate with select", func(t *testing.T) {
		pop := PopulateOption{
			Path:   "author",
			Select: []string{"name", "email"},
		}

		if pop.Path != "author" {
			t.Errorf("expected path 'author', got '%s'", pop.Path)
		}
		if len(pop.Select) != 2 {
			t.Errorf("expected 2 select fields, got %d", len(pop.Select))
		}
	})

	t.Run("nested populate", func(t *testing.T) {
		pop := PopulateOption{
			Path: "author",
			Populate: []PopulateOption{
				{Path: "company"},
			},
		}

		if len(pop.Populate) != 1 {
			t.Errorf("expected 1 nested populate, got %d", len(pop.Populate))
		}
		if pop.Populate[0].Path != "company" {
			t.Errorf("expected nested path 'company', got '%s'", pop.Populate[0].Path)
		}
	})
}

func TestLogicalOperator(t *testing.T) {
	tests := []struct {
		op       LogicalOperator
		expected string
	}{
		{LogicalAnd, "and"},
		{LogicalOr, "or"},
		{LogicalNot, "not"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.op) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.op))
			}
		})
	}
}
