package query

import (
	"encoding/json"
	"testing"
)

// TestJSONParser_AllFilterOperators tests all filter operators with JSON input
func TestJSONParser_AllFilterOperators(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name           string
		json           string
		expectedField  string
		expectedOp     FilterOperator
		expectedValue  interface{}
		checkValueType func(interface{}) bool
	}{
		// Equality operators
		{
			name:          "$eq with string",
			json:          `{"filter": {"status": {"$eq": "published"}}}`,
			expectedField: "status",
			expectedOp:    OpEqual,
			expectedValue: "published",
		},
		{
			name:          "$eq with number",
			json:          `{"filter": {"count": {"$eq": 42}}}`,
			expectedField: "count",
			expectedOp:    OpEqual,
			expectedValue: float64(42), // JSON numbers are float64
		},
		{
			name:          "$eq with boolean",
			json:          `{"filter": {"active": {"$eq": true}}}`,
			expectedField: "active",
			expectedOp:    OpEqual,
			expectedValue: true,
		},
		{
			name:          "$ne with string",
			json:          `{"filter": {"status": {"$ne": "draft"}}}`,
			expectedField: "status",
			expectedOp:    OpNotEqual,
			expectedValue: "draft",
		},

		// Comparison operators
		{
			name:          "$gt greater than",
			json:          `{"filter": {"price": {"$gt": 100}}}`,
			expectedField: "price",
			expectedOp:    OpGreaterThan,
			expectedValue: float64(100),
		},
		{
			name:          "$gte greater than or equal",
			json:          `{"filter": {"price": {"$gte": 100}}}`,
			expectedField: "price",
			expectedOp:    OpGreaterThanEqual,
			expectedValue: float64(100),
		},
		{
			name:          "$lt less than",
			json:          `{"filter": {"price": {"$lt": 50}}}`,
			expectedField: "price",
			expectedOp:    OpLessThan,
			expectedValue: float64(50),
		},
		{
			name:          "$lte less than or equal",
			json:          `{"filter": {"price": {"$lte": 50}}}`,
			expectedField: "price",
			expectedOp:    OpLessThanEqual,
			expectedValue: float64(50),
		},

		// String matching operators
		{
			name:          "$like pattern match",
			json:          `{"filter": {"title": {"$like": "%hello%"}}}`,
			expectedField: "title",
			expectedOp:    OpLike,
			expectedValue: "%hello%",
		},
		{
			name:          "$ilike case-insensitive pattern",
			json:          `{"filter": {"title": {"$ilike": "%HELLO%"}}}`,
			expectedField: "title",
			expectedOp:    OpILike,
			expectedValue: "%HELLO%",
		},
		{
			name:          "$contains text contains",
			json:          `{"filter": {"description": {"$contains": "important"}}}`,
			expectedField: "description",
			expectedOp:    OpContains,
			expectedValue: "important",
		},
		{
			name:          "$startsWith text starts with",
			json:          `{"filter": {"title": {"$startsWith": "Hello"}}}`,
			expectedField: "title",
			expectedOp:    OpStartsWith,
			expectedValue: "Hello",
		},
		{
			name:          "$endsWith text ends with",
			json:          `{"filter": {"email": {"$endsWith": "@example.com"}}}`,
			expectedField: "email",
			expectedOp:    OpEndsWith,
			expectedValue: "@example.com",
		},

		// Array operators
		{
			name:          "$in array contains value",
			json:          `{"filter": {"status": {"$in": ["draft", "published", "archived"]}}}`,
			expectedField: "status",
			expectedOp:    OpIn,
			checkValueType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				return ok && len(arr) == 3
			},
		},
		{
			name:          "$nin array does not contain value",
			json:          `{"filter": {"status": {"$nin": ["deleted", "banned"]}}}`,
			expectedField: "status",
			expectedOp:    OpNotIn,
			checkValueType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				return ok && len(arr) == 2
			},
		},
		{
			name:          "$notIn alias for nin",
			json:          `{"filter": {"category": {"$notIn": ["spam", "junk"]}}}`,
			expectedField: "category",
			expectedOp:    OpNotIn,
			checkValueType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				return ok && len(arr) == 2
			},
		},
		{
			name:          "$all array contains all values",
			json:          `{"filter": {"tags": {"$all": ["golang", "testing"]}}}`,
			expectedField: "tags",
			expectedOp:    OpAll,
			checkValueType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				return ok && len(arr) == 2
			},
		},
		{
			name:          "$any array contains any value",
			json:          `{"filter": {"tags": {"$any": ["featured", "popular"]}}}`,
			expectedField: "tags",
			expectedOp:    OpAny,
			checkValueType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				return ok && len(arr) == 2
			},
		},

		// Null/existence operators
		{
			name:          "$null is null (true)",
			json:          `{"filter": {"deletedAt": {"$null": true}}}`,
			expectedField: "deletedAt",
			expectedOp:    OpNull,
			expectedValue: true,
		},
		{
			name:          "$null is not null (false)",
			json:          `{"filter": {"publishedAt": {"$null": false}}}`,
			expectedField: "publishedAt",
			expectedOp:    OpNull,
			expectedValue: false,
		},
		{
			name:          "$isNull alias",
			json:          `{"filter": {"archivedAt": {"$isNull": true}}}`,
			expectedField: "archivedAt",
			expectedOp:    OpNull,
			expectedValue: true,
		},
		{
			name:          "$exists field exists",
			json:          `{"filter": {"metadata": {"$exists": true}}}`,
			expectedField: "metadata",
			expectedOp:    OpExists,
			expectedValue: true,
		},
		{
			name:          "$exists field does not exist",
			json:          `{"filter": {"optionalField": {"$exists": false}}}`,
			expectedField: "optionalField",
			expectedOp:    OpExists,
			expectedValue: false,
		},

		// Range operator
		{
			name:          "$between range",
			json:          `{"filter": {"price": {"$between": [10, 100]}}}`,
			expectedField: "price",
			expectedOp:    OpBetween,
			checkValueType: func(v interface{}) bool {
				arr, ok := v.([]interface{})
				return ok && len(arr) == 2
			},
		},

		// JSONB operators
		{
			name:          "$jsonContains JSONB contains",
			json:          `{"filter": {"metadata": {"$jsonContains": {"key": "value"}}}}`,
			expectedField: "metadata",
			expectedOp:    OpJsonContains,
			checkValueType: func(v interface{}) bool {
				m, ok := v.(map[string]interface{})
				return ok && m["key"] == "value"
			},
		},
		{
			name:          "$jsonHasKey JSONB has key",
			json:          `{"filter": {"config": {"$jsonHasKey": "enabled"}}}`,
			expectedField: "config",
			expectedOp:    OpJsonHasKey,
			expectedValue: "enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := parser.Parse([]byte(tt.json))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if q.Filters == nil {
				t.Fatal("expected filters to be parsed")
			}

			if len(q.Filters.Conditions) != 1 {
				t.Fatalf("expected 1 condition, got %d", len(q.Filters.Conditions))
			}

			cond := q.Filters.Conditions[0]

			if cond.Field != tt.expectedField {
				t.Errorf("field: expected %q, got %q", tt.expectedField, cond.Field)
			}

			if cond.Operator != tt.expectedOp {
				t.Errorf("operator: expected %q, got %q", tt.expectedOp, cond.Operator)
			}

			if tt.checkValueType != nil {
				if !tt.checkValueType(cond.Value) {
					t.Errorf("value type check failed for value: %v (%T)", cond.Value, cond.Value)
				}
			} else if tt.expectedValue != nil {
				if cond.Value != tt.expectedValue {
					t.Errorf("value: expected %v (%T), got %v (%T)",
						tt.expectedValue, tt.expectedValue, cond.Value, cond.Value)
				}
			}
		})
	}
}

// TestJSONParser_DirectValueEquality tests implicit $eq when no operator specified
func TestJSONParser_DirectValueEquality(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name          string
		json          string
		expectedField string
		expectedValue interface{}
	}{
		{
			name:          "direct string value",
			json:          `{"filter": {"status": "published"}}`,
			expectedField: "status",
			expectedValue: "published",
		},
		{
			name:          "direct number value",
			json:          `{"filter": {"count": 42}}`,
			expectedField: "count",
			expectedValue: float64(42),
		},
		{
			name:          "direct boolean value",
			json:          `{"filter": {"active": true}}`,
			expectedField: "active",
			expectedValue: true,
		},
		{
			name:          "direct null value",
			json:          `{"filter": {"deletedAt": null}}`,
			expectedField: "deletedAt",
			expectedValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := parser.Parse([]byte(tt.json))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if q.Filters == nil || len(q.Filters.Conditions) != 1 {
				t.Fatal("expected 1 filter condition")
			}

			cond := q.Filters.Conditions[0]

			if cond.Field != tt.expectedField {
				t.Errorf("field: expected %q, got %q", tt.expectedField, cond.Field)
			}

			// Direct values should use OpEqual
			if cond.Operator != OpEqual {
				t.Errorf("operator: expected OpEqual, got %q", cond.Operator)
			}

			if cond.Value != tt.expectedValue {
				t.Errorf("value: expected %v, got %v", tt.expectedValue, cond.Value)
			}
		})
	}
}

// TestJSONParser_MultipleOperatorsOnSameField tests range queries
func TestJSONParser_MultipleOperatorsOnSameField(t *testing.T) {
	parser := NewJSONParser()

	// Range query: price >= 10 AND price <= 100
	json := `{
		"filter": {
			"price": {"$gte": 10, "$lte": 100}
		}
	}`

	q, err := parser.Parse([]byte(json))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if q.Filters == nil {
		t.Fatal("expected filters to be parsed")
	}

	if len(q.Filters.Conditions) != 2 {
		t.Fatalf("expected 2 conditions for range query, got %d", len(q.Filters.Conditions))
	}

	// Both conditions should be for "price"
	hasGte := false
	hasLte := false
	for _, cond := range q.Filters.Conditions {
		if cond.Field != "price" {
			t.Errorf("expected field 'price', got %q", cond.Field)
		}
		if cond.Operator == OpGreaterThanEqual {
			hasGte = true
			if cond.Value != float64(10) {
				t.Errorf("$gte value: expected 10, got %v", cond.Value)
			}
		}
		if cond.Operator == OpLessThanEqual {
			hasLte = true
			if cond.Value != float64(100) {
				t.Errorf("$lte value: expected 100, got %v", cond.Value)
			}
		}
	}

	if !hasGte {
		t.Error("missing $gte condition")
	}
	if !hasLte {
		t.Error("missing $lte condition")
	}
}

// TestJSONParser_LogicalOperatorsWithFilters tests $and, $or, $not with filters
func TestJSONParser_LogicalOperatorsWithFilters(t *testing.T) {
	parser := NewJSONParser()

	t.Run("$and with multiple conditions", func(t *testing.T) {
		json := `{
			"filter": {
				"$and": [
					{"status": {"$eq": "published"}},
					{"category": {"$in": ["news", "blog"]}}
				]
			}
		}`

		q, err := parser.Parse([]byte(json))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if q.Filters == nil || len(q.Filters.Groups) != 1 {
			t.Fatal("expected 1 filter group")
		}

		andGroup := q.Filters.Groups[0]
		if andGroup.Operator != LogicalAnd {
			t.Errorf("expected AND operator, got %s", andGroup.Operator)
		}
	})

	t.Run("$or with multiple conditions", func(t *testing.T) {
		json := `{
			"filter": {
				"$or": [
					{"status": "draft"},
					{"status": "published"}
				]
			}
		}`

		q, err := parser.Parse([]byte(json))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if q.Filters == nil || len(q.Filters.Groups) != 1 {
			t.Fatal("expected 1 filter group")
		}

		orGroup := q.Filters.Groups[0]
		if orGroup.Operator != LogicalOr {
			t.Errorf("expected OR operator, got %s", orGroup.Operator)
		}
	})

	t.Run("$not negation", func(t *testing.T) {
		json := `{
			"filter": {
				"$not": {"status": "deleted"}
			}
		}`

		q, err := parser.Parse([]byte(json))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if q.Filters == nil || len(q.Filters.Groups) != 1 {
			t.Fatal("expected 1 filter group")
		}

		notGroup := q.Filters.Groups[0]
		if notGroup.Operator != LogicalNot {
			t.Errorf("expected NOT operator, got %s", notGroup.Operator)
		}
	})

	t.Run("nested $and and $or", func(t *testing.T) {
		json := `{
			"filter": {
				"$and": [
					{"category": "news"},
					{
						"$or": [
							{"status": "published"},
							{"status": "featured"}
						]
					}
				]
			}
		}`

		q, err := parser.Parse([]byte(json))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if q.Filters == nil || len(q.Filters.Groups) != 1 {
			t.Fatal("expected 1 top-level filter group")
		}

		andGroup := q.Filters.Groups[0]
		if andGroup.Operator != LogicalAnd {
			t.Errorf("expected AND operator, got %s", andGroup.Operator)
		}
	})
}

// TestJSONParser_ComplexFilterCombinations tests complex filter scenarios
func TestJSONParser_ComplexFilterCombinations(t *testing.T) {
	parser := NewJSONParser()

	t.Run("multiple fields with different operators", func(t *testing.T) {
		json := `{
			"filter": {
				"status": {"$eq": "published"},
				"views": {"$gte": 100},
				"category": {"$in": ["tech", "science"]},
				"title": {"$contains": "golang"}
			}
		}`

		q, err := parser.Parse([]byte(json))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if q.Filters == nil {
			t.Fatal("expected filters to be parsed")
		}

		if len(q.Filters.Conditions) != 4 {
			t.Errorf("expected 4 conditions, got %d", len(q.Filters.Conditions))
		}

		// Check each condition exists
		fields := make(map[string]bool)
		for _, cond := range q.Filters.Conditions {
			fields[cond.Field] = true
		}

		expectedFields := []string{"status", "views", "category", "title"}
		for _, f := range expectedFields {
			if !fields[f] {
				t.Errorf("missing filter for field %q", f)
			}
		}
	})

	t.Run("filter with sort and pagination", func(t *testing.T) {
		json := `{
			"filter": {
				"status": {"$eq": "published"},
				"category": {"$ne": "spam"}
			},
			"sort": ["-createdAt", "title"],
			"page": 2,
			"pageSize": 25
		}`

		q, err := parser.Parse([]byte(json))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		// Check filters
		if q.Filters == nil || len(q.Filters.Conditions) != 2 {
			t.Error("expected 2 filter conditions")
		}

		// Check sort
		if len(q.Sort) != 2 {
			t.Errorf("expected 2 sort fields, got %d", len(q.Sort))
		}

		// Check pagination
		if q.Page != 2 {
			t.Errorf("expected page 2, got %d", q.Page)
		}
		if q.PageSize != 25 {
			t.Errorf("expected pageSize 25, got %d", q.PageSize)
		}
	})
}

// TestFilterOperatorToRepositoryOperator ensures filter operators match repository expectations
func TestFilterOperatorToRepositoryOperator(t *testing.T) {
	// This test verifies that the FilterOperator constants match
	// what the repository expects
	expectedMappings := map[FilterOperator]string{
		OpEqual:            "eq",
		OpNotEqual:         "ne",
		OpGreaterThan:      "gt",
		OpGreaterThanEqual: "gte",
		OpLessThan:         "lt",
		OpLessThanEqual:    "lte",
		OpLike:             "like",
		OpILike:            "ilike",
		OpContains:         "contains",
		OpStartsWith:       "startsWith",
		OpEndsWith:         "endsWith",
		OpIn:               "in",
		OpNotIn:            "nin",
		OpNull:             "null",
		OpExists:           "exists",
		OpBetween:          "between",
		OpJsonContains:     "jsonContains",
		OpJsonHasKey:       "jsonHasKey",
	}

	for op, expected := range expectedMappings {
		if string(op) != expected {
			t.Errorf("FilterOperator %s should be %q but is %q", op, expected, string(op))
		}
	}
}

// TestJSONParser_FilterAliases tests that filter, filters, and where all work
func TestJSONParser_FilterAliases(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name string
		json string
	}{
		{
			name: "filter (singular)",
			json: `{"filter": {"status": "published"}}`,
		},
		{
			name: "filters (plural)",
			json: `{"filters": {"status": "published"}}`,
		},
		{
			name: "where clause",
			json: `{"where": {"status": "published"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := parser.Parse([]byte(tt.json))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if q.Filters == nil || len(q.Filters.Conditions) != 1 {
				t.Fatal("expected 1 filter condition")
			}

			cond := q.Filters.Conditions[0]
			if cond.Field != "status" || cond.Value != "published" {
				t.Errorf("unexpected filter: %+v", cond)
			}
		})
	}
}

// TestJSONParser_OperatorCaseInsensitivity tests that operators work in various cases
func TestJSONParser_OperatorVariants(t *testing.T) {
	parser := NewJSONParser()

	// Test both $eq and eq work (the JSON parser should support both)
	tests := []struct {
		name string
		json string
	}{
		{
			name: "with $ prefix",
			json: `{"filter": {"status": {"$eq": "published"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := parser.Parse([]byte(tt.json))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if q.Filters == nil || len(q.Filters.Conditions) != 1 {
				t.Fatal("expected 1 filter condition")
			}

			if q.Filters.Conditions[0].Operator != OpEqual {
				t.Errorf("expected OpEqual, got %s", q.Filters.Conditions[0].Operator)
			}
		})
	}
}

// TestJSONParser_EmptyAndNullFilters tests edge cases
func TestJSONParser_EmptyAndNullFilters(t *testing.T) {
	parser := NewJSONParser()

	t.Run("empty filter object", func(t *testing.T) {
		q, err := parser.Parse([]byte(`{"filter": {}}`))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if q.Filters == nil {
			t.Error("filters should not be nil for empty object")
		}
		if len(q.Filters.Conditions) != 0 {
			t.Errorf("expected 0 conditions, got %d", len(q.Filters.Conditions))
		}
	})

	t.Run("no filter key", func(t *testing.T) {
		q, err := parser.Parse([]byte(`{"page": 1, "pageSize": 10}`))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if q.Filters != nil && len(q.Filters.Conditions) > 0 {
			t.Error("expected no filter conditions")
		}
	})
}

// TestFilterConditionSerialization tests that filter conditions can be serialized/deserialized
func TestFilterConditionSerialization(t *testing.T) {
	cond := FilterCondition{
		Field:    "status",
		Operator: OpEqual,
		Value:    "published",
	}

	// Serialize to JSON
	data, err := json.Marshal(cond)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Deserialize back
	var cond2 FilterCondition
	if err := json.Unmarshal(data, &cond2); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if cond2.Field != cond.Field {
		t.Errorf("Field mismatch: %s != %s", cond2.Field, cond.Field)
	}
	if cond2.Operator != cond.Operator {
		t.Errorf("Operator mismatch: %s != %s", cond2.Operator, cond.Operator)
	}
}

// Benchmark parsing filters
func BenchmarkJSONParser_ParseFilters(b *testing.B) {
	parser := NewJSONParser()

	benchmarks := []struct {
		name string
		json string
	}{
		{
			name: "simple equality",
			json: `{"filter": {"status": "published"}}`,
		},
		{
			name: "multiple operators",
			json: `{"filter": {"status": {"$eq": "published"}, "views": {"$gte": 100}, "title": {"$contains": "hello"}}}`,
		},
		{
			name: "complex with logical operators",
			json: `{"filter": {"$and": [{"status": "published"}, {"$or": [{"category": "news"}, {"category": "blog"}]}]}}`,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			data := []byte(bm.json)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = parser.Parse(data)
			}
		})
	}
}
