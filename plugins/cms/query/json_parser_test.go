package query

import (
	"testing"
)

func TestJSONParser_Parse_FilterSingular(t *testing.T) {
	parser := NewJSONParser()

	// Test filter (singular) is accepted
	input := []byte(`{
		"filter": {
			"status": {"$eq": "draft"}
		},
		"page": 1,
		"pageSize": 10
	}`)

	q, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Filters == nil {
		t.Fatal("expected filters to be parsed")
	}

	if len(q.Filters.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(q.Filters.Conditions))
	}

	cond := q.Filters.Conditions[0]
	if cond.Field != "status" {
		t.Errorf("expected field 'status', got '%s'", cond.Field)
	}
	if cond.Operator != OpEqual {
		t.Errorf("expected operator OpEqual, got '%s'", cond.Operator)
	}
	if cond.Value != "draft" {
		t.Errorf("expected value 'draft', got '%v'", cond.Value)
	}
}

func TestJSONParser_Parse_FiltersPlural(t *testing.T) {
	parser := NewJSONParser()

	// Test filters (plural) is accepted
	input := []byte(`{
		"filters": {
			"status": {"$eq": "published"}
		},
		"page": 1,
		"pageSize": 10
	}`)

	q, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Filters == nil {
		t.Fatal("expected filters to be parsed")
	}

	if len(q.Filters.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(q.Filters.Conditions))
	}

	cond := q.Filters.Conditions[0]
	if cond.Field != "status" {
		t.Errorf("expected field 'status', got '%s'", cond.Field)
	}
	if cond.Value != "published" {
		t.Errorf("expected value 'published', got '%v'", cond.Value)
	}
}

func TestJSONParser_Parse_WhereAlternative(t *testing.T) {
	parser := NewJSONParser()

	// Test where is accepted as alternative to filters
	input := []byte(`{
		"where": {
			"title": {"$contains": "hello"}
		}
	}`)

	q, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Filters == nil {
		t.Fatal("expected filters to be parsed from 'where'")
	}

	if len(q.Filters.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(q.Filters.Conditions))
	}

	cond := q.Filters.Conditions[0]
	if cond.Field != "title" {
		t.Errorf("expected field 'title', got '%s'", cond.Field)
	}
	if cond.Operator != OpContains {
		t.Errorf("expected operator OpContains, got '%s'", cond.Operator)
	}
}

func TestJSONParser_Parse_FilterPrecedence(t *testing.T) {
	parser := NewJSONParser()

	// Test that filters takes precedence over filter takes precedence over where
	input := []byte(`{
		"filters": {"status": "published"},
		"filter": {"status": "draft"},
		"where": {"status": "archived"}
	}`)

	q, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Filters == nil {
		t.Fatal("expected filters to be parsed")
	}

	// filters should take precedence
	cond := q.Filters.Conditions[0]
	if cond.Value != "published" {
		t.Errorf("expected value 'published' (from filters), got '%v'", cond.Value)
	}
}

func TestJSONParser_Parse_AllOperators(t *testing.T) {
	parser := NewJSONParser()

	tests := []struct {
		name     string
		input    string
		expected FilterOperator
	}{
		{"$eq", `{"filter": {"field": {"$eq": "value"}}}`, OpEqual},
		{"$ne", `{"filter": {"field": {"$ne": "value"}}}`, OpNotEqual},
		{"$gt", `{"filter": {"field": {"$gt": 10}}}`, OpGreaterThan},
		{"$gte", `{"filter": {"field": {"$gte": 10}}}`, OpGreaterThanEqual},
		{"$lt", `{"filter": {"field": {"$lt": 10}}}`, OpLessThan},
		{"$lte", `{"filter": {"field": {"$lte": 10}}}`, OpLessThanEqual},
		{"$like", `{"filter": {"field": {"$like": "%test%"}}}`, OpLike},
		{"$ilike", `{"filter": {"field": {"$ilike": "%test%"}}}`, OpILike},
		{"$contains", `{"filter": {"field": {"$contains": "test"}}}`, OpContains},
		{"$startsWith", `{"filter": {"field": {"$startsWith": "test"}}}`, OpStartsWith},
		{"$endsWith", `{"filter": {"field": {"$endsWith": "test"}}}`, OpEndsWith},
		{"$in", `{"filter": {"field": {"$in": ["a", "b"]}}}`, OpIn},
		{"$nin", `{"filter": {"field": {"$nin": ["a", "b"]}}}`, OpNotIn},
		{"$notIn", `{"filter": {"field": {"$notIn": ["a", "b"]}}}`, OpNotIn},
		{"$null", `{"filter": {"field": {"$null": true}}}`, OpNull},
		{"$isNull", `{"filter": {"field": {"$isNull": true}}}`, OpNull},
		{"$exists", `{"filter": {"field": {"$exists": true}}}`, OpExists},
		{"$between", `{"filter": {"field": {"$between": [1, 10]}}}`, OpBetween},
		{"$jsonContains", `{"filter": {"field": {"$jsonContains": {"key": "val"}}}}`, OpJsonContains},
		{"$jsonHasKey", `{"filter": {"field": {"$jsonHasKey": "key"}}}`, OpJsonHasKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := parser.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if q.Filters == nil || len(q.Filters.Conditions) != 1 {
				t.Fatal("expected 1 filter condition")
			}

			if q.Filters.Conditions[0].Operator != tt.expected {
				t.Errorf("expected operator %s, got %s", tt.expected, q.Filters.Conditions[0].Operator)
			}
		})
	}
}

func TestJSONParser_Parse_DirectValueFilter(t *testing.T) {
	parser := NewJSONParser()

	// Test direct value (implicit $eq)
	input := []byte(`{
		"filter": {
			"status": "published"
		}
	}`)

	q, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Filters == nil {
		t.Fatal("expected filters to be parsed")
	}

	cond := q.Filters.Conditions[0]
	if cond.Field != "status" {
		t.Errorf("expected field 'status', got '%s'", cond.Field)
	}
	if cond.Operator != OpEqual {
		t.Errorf("expected operator OpEqual (implicit), got '%s'", cond.Operator)
	}
	if cond.Value != "published" {
		t.Errorf("expected value 'published', got '%v'", cond.Value)
	}
}

func TestJSONParser_Parse_LogicalOperators(t *testing.T) {
	parser := NewJSONParser()

	t.Run("$and", func(t *testing.T) {
		input := []byte(`{
			"filter": {
				"$and": [
					{"status": "published"},
					{"category": "news"}
				]
			}
		}`)

		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Filters == nil {
			t.Fatal("expected filters to be parsed")
		}

		if len(q.Filters.Groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(q.Filters.Groups))
		}

		andGroup := q.Filters.Groups[0]
		if andGroup.Operator != LogicalAnd {
			t.Errorf("expected LogicalAnd operator, got %s", andGroup.Operator)
		}
	})

	t.Run("$or", func(t *testing.T) {
		input := []byte(`{
			"filter": {
				"$or": [
					{"status": "draft"},
					{"status": "published"}
				]
			}
		}`)

		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Filters == nil {
			t.Fatal("expected filters to be parsed")
		}

		if len(q.Filters.Groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(q.Filters.Groups))
		}

		orGroup := q.Filters.Groups[0]
		if orGroup.Operator != LogicalOr {
			t.Errorf("expected LogicalOr operator, got %s", orGroup.Operator)
		}
	})

	t.Run("$not", func(t *testing.T) {
		input := []byte(`{
			"filter": {
				"$not": {"status": "archived"}
			}
		}`)

		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Filters == nil {
			t.Fatal("expected filters to be parsed")
		}

		if len(q.Filters.Groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(q.Filters.Groups))
		}

		notGroup := q.Filters.Groups[0]
		if notGroup.Operator != LogicalNot {
			t.Errorf("expected LogicalNot operator, got %s", notGroup.Operator)
		}
	})
}

func TestJSONParser_Parse_Sort(t *testing.T) {
	parser := NewJSONParser()

	t.Run("sort string with minus prefix", func(t *testing.T) {
		input := []byte(`{"sort": "-createdAt"}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Sort) != 1 {
			t.Fatalf("expected 1 sort field, got %d", len(q.Sort))
		}

		if q.Sort[0].Field != "createdAt" {
			t.Errorf("expected field 'createdAt', got '%s'", q.Sort[0].Field)
		}
		if !q.Sort[0].Descending {
			t.Error("expected descending to be true")
		}
	})

	t.Run("sort string with plus prefix", func(t *testing.T) {
		input := []byte(`{"sort": "+updatedAt"}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Sort) != 1 {
			t.Fatalf("expected 1 sort field, got %d", len(q.Sort))
		}

		if q.Sort[0].Field != "updatedAt" {
			t.Errorf("expected field 'updatedAt', got '%s'", q.Sort[0].Field)
		}
		if q.Sort[0].Descending {
			t.Error("expected descending to be false")
		}
	})

	t.Run("sort string with colon suffix", func(t *testing.T) {
		input := []byte(`{"sort": "title:desc"}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Sort) != 1 {
			t.Fatalf("expected 1 sort field, got %d", len(q.Sort))
		}

		if q.Sort[0].Field != "title" {
			t.Errorf("expected field 'title', got '%s'", q.Sort[0].Field)
		}
		if !q.Sort[0].Descending {
			t.Error("expected descending to be true")
		}
	})

	t.Run("sort array of strings", func(t *testing.T) {
		input := []byte(`{"sort": ["-updatedAt", "title"]}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Sort) != 2 {
			t.Fatalf("expected 2 sort fields, got %d", len(q.Sort))
		}

		if q.Sort[0].Field != "updatedAt" || !q.Sort[0].Descending {
			t.Errorf("expected updatedAt desc, got %s %v", q.Sort[0].Field, q.Sort[0].Descending)
		}
		if q.Sort[1].Field != "title" || q.Sort[1].Descending {
			t.Errorf("expected title asc, got %s %v", q.Sort[1].Field, q.Sort[1].Descending)
		}
	})

	t.Run("sort array of objects", func(t *testing.T) {
		input := []byte(`{"sort": [{"field": "createdAt", "order": "desc"}]}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Sort) != 1 {
			t.Fatalf("expected 1 sort field, got %d", len(q.Sort))
		}

		if q.Sort[0].Field != "createdAt" {
			t.Errorf("expected field 'createdAt', got '%s'", q.Sort[0].Field)
		}
		if !q.Sort[0].Descending {
			t.Error("expected descending to be true")
		}
	})

	t.Run("sort object map", func(t *testing.T) {
		input := []byte(`{"sort": {"createdAt": "desc", "title": "asc"}}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Sort) != 2 {
			t.Fatalf("expected 2 sort fields, got %d", len(q.Sort))
		}
	})
}

func TestJSONParser_Parse_Pagination(t *testing.T) {
	parser := NewJSONParser()

	t.Run("page and pageSize", func(t *testing.T) {
		input := []byte(`{"page": 2, "pageSize": 25}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Page != 2 {
			t.Errorf("expected page 2, got %d", q.Page)
		}
		if q.PageSize != 25 {
			t.Errorf("expected pageSize 25, got %d", q.PageSize)
		}
	})

	t.Run("perPage alias", func(t *testing.T) {
		input := []byte(`{"page": 1, "perPage": 50}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.PageSize != 50 {
			t.Errorf("expected pageSize 50 (from perPage), got %d", q.PageSize)
		}
	})

	t.Run("offset and limit", func(t *testing.T) {
		input := []byte(`{"offset": 20, "limit": 10}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Offset != 20 {
			t.Errorf("expected offset 20, got %d", q.Offset)
		}
		if q.Limit != 10 {
			t.Errorf("expected limit 10, got %d", q.Limit)
		}
	})
}

func TestJSONParser_Parse_Search(t *testing.T) {
	parser := NewJSONParser()

	t.Run("search field", func(t *testing.T) {
		input := []byte(`{"search": "hello world"}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Search != "hello world" {
			t.Errorf("expected search 'hello world', got '%s'", q.Search)
		}
	})

	t.Run("q alias", func(t *testing.T) {
		input := []byte(`{"q": "search term"}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Search != "search term" {
			t.Errorf("expected search 'search term' (from q), got '%s'", q.Search)
		}
	})
}

func TestJSONParser_Parse_StatusShorthand(t *testing.T) {
	parser := NewJSONParser()

	input := []byte(`{"status": "published"}`)
	q, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Status != "published" {
		t.Errorf("expected status 'published', got '%s'", q.Status)
	}
}

func TestJSONParser_Parse_Select(t *testing.T) {
	parser := NewJSONParser()

	t.Run("select field", func(t *testing.T) {
		input := []byte(`{"select": ["id", "title", "status"]}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Select) != 3 {
			t.Fatalf("expected 3 select fields, got %d", len(q.Select))
		}
	})

	t.Run("fields alias", func(t *testing.T) {
		input := []byte(`{"fields": ["id", "title"]}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Select) != 2 {
			t.Fatalf("expected 2 select fields (from fields alias), got %d", len(q.Select))
		}
	})
}

func TestJSONParser_Parse_Populate(t *testing.T) {
	parser := NewJSONParser()

	t.Run("populate string", func(t *testing.T) {
		input := []byte(`{"populate": "author"}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Populate) != 1 {
			t.Fatalf("expected 1 populate option, got %d", len(q.Populate))
		}
		if q.Populate[0].Path != "author" {
			t.Errorf("expected path 'author', got '%s'", q.Populate[0].Path)
		}
	})

	t.Run("populate array", func(t *testing.T) {
		input := []byte(`{"populate": ["author", "category"]}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Populate) != 2 {
			t.Fatalf("expected 2 populate options, got %d", len(q.Populate))
		}
	})

	t.Run("populate object", func(t *testing.T) {
		input := []byte(`{"populate": {"path": "author", "select": ["name", "email"]}}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Populate) != 1 {
			t.Fatalf("expected 1 populate option, got %d", len(q.Populate))
		}
		if q.Populate[0].Path != "author" {
			t.Errorf("expected path 'author', got '%s'", q.Populate[0].Path)
		}
		if len(q.Populate[0].Select) != 2 {
			t.Errorf("expected 2 select fields, got %d", len(q.Populate[0].Select))
		}
	})

	t.Run("include alias", func(t *testing.T) {
		input := []byte(`{"include": "category"}`)
		q, err := parser.Parse(input)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Populate) != 1 {
			t.Fatalf("expected 1 populate option (from include), got %d", len(q.Populate))
		}
	})
}

func TestJSONParser_Parse_CompleteQuery(t *testing.T) {
	parser := NewJSONParser()

	// Test a complete query with all features
	input := []byte(`{
		"filter": {
			"status": {"$eq": "draft"},
			"category": "news"
		},
		"sort": ["-updatedAt"],
		"page": 1,
		"pageSize": 10,
		"search": "hello",
		"select": ["id", "title", "status"],
		"populate": ["author"]
	}`)

	q, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify filters
	if q.Filters == nil {
		t.Fatal("expected filters")
	}
	if len(q.Filters.Conditions) != 2 {
		t.Errorf("expected 2 filter conditions, got %d", len(q.Filters.Conditions))
	}

	// Verify sort
	if len(q.Sort) != 1 {
		t.Errorf("expected 1 sort field, got %d", len(q.Sort))
	}
	if q.Sort[0].Field != "updatedAt" || !q.Sort[0].Descending {
		t.Errorf("expected updatedAt desc")
	}

	// Verify pagination
	if q.Page != 1 {
		t.Errorf("expected page 1, got %d", q.Page)
	}
	if q.PageSize != 10 {
		t.Errorf("expected pageSize 10, got %d", q.PageSize)
	}

	// Verify search
	if q.Search != "hello" {
		t.Errorf("expected search 'hello', got '%s'", q.Search)
	}

	// Verify select
	if len(q.Select) != 3 {
		t.Errorf("expected 3 select fields, got %d", len(q.Select))
	}

	// Verify populate
	if len(q.Populate) != 1 {
		t.Errorf("expected 1 populate option, got %d", len(q.Populate))
	}
}

func TestJSONParser_Parse_InvalidJSON(t *testing.T) {
	parser := NewJSONParser()

	input := []byte(`{invalid json}`)
	_, err := parser.Parse(input)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestJSONParser_Parse_InvalidFilterType(t *testing.T) {
	parser := NewJSONParser()

	// Filters must be object or array, not string
	input := []byte(`{"filter": "invalid"}`)
	_, err := parser.Parse(input)
	if err == nil {
		t.Error("expected error for invalid filter type")
	}
}

func TestJSONParser_Parse_InvalidOperator(t *testing.T) {
	parser := NewJSONParser()

	input := []byte(`{"filter": {"field": {"$invalidOp": "value"}}}`)
	_, err := parser.Parse(input)
	if err == nil {
		t.Error("expected error for invalid operator")
	}
}

func TestJSONParser_Parse_InvalidLogicalOperatorValue(t *testing.T) {
	parser := NewJSONParser()

	// $and must be an array
	input := []byte(`{"filter": {"$and": "not an array"}}`)
	_, err := parser.Parse(input)
	if err == nil {
		t.Error("expected error for invalid $and value")
	}

	// $or must be an array
	input = []byte(`{"filter": {"$or": "not an array"}}`)
	_, err = parser.Parse(input)
	if err == nil {
		t.Error("expected error for invalid $or value")
	}

	// $not must be an object
	input = []byte(`{"filter": {"$not": "not an object"}}`)
	_, err = parser.Parse(input)
	if err == nil {
		t.Error("expected error for invalid $not value")
	}
}

func TestJSONParser_Parse_EmptyFilter(t *testing.T) {
	parser := NewJSONParser()

	input := []byte(`{"filter": {}}`)
	q, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Filters == nil {
		t.Fatal("expected filters to exist but be empty")
	}

	if len(q.Filters.Conditions) != 0 {
		t.Errorf("expected 0 conditions, got %d", len(q.Filters.Conditions))
	}
}

func TestJSONParser_Parse_MultipleConditionsOnSameField(t *testing.T) {
	parser := NewJSONParser()

	// Test range query: field >= 10 AND field <= 100
	input := []byte(`{
		"filter": {
			"price": {"$gte": 10, "$lte": 100}
		}
	}`)

	q, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Filters == nil {
		t.Fatal("expected filters to be parsed")
	}

	if len(q.Filters.Conditions) != 2 {
		t.Fatalf("expected 2 conditions (for range), got %d", len(q.Filters.Conditions))
	}

	// Verify both conditions are for the 'price' field
	for _, cond := range q.Filters.Conditions {
		if cond.Field != "price" {
			t.Errorf("expected field 'price', got '%s'", cond.Field)
		}
	}
}

// Benchmark tests
func BenchmarkJSONParser_Parse_Simple(b *testing.B) {
	parser := NewJSONParser()
	input := []byte(`{"filter": {"status": "published"}, "page": 1, "pageSize": 10}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(input)
	}
}

func BenchmarkJSONParser_Parse_Complex(b *testing.B) {
	parser := NewJSONParser()
	input := []byte(`{
		"filter": {
			"$and": [
				{"status": {"$in": ["published", "draft"]}},
				{"$or": [
					{"category": "news"},
					{"category": "blog"}
				]}
			]
		},
		"sort": ["-updatedAt", "title"],
		"page": 1,
		"pageSize": 20,
		"search": "hello",
		"populate": ["author", "category"]
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(input)
	}
}
