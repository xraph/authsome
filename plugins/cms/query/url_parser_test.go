package query

import (
	"net/url"
	"testing"
)

func TestURLParser_Parse_Filters(t *testing.T) {
	parser := NewURLParser()

	t.Run("simple equality filter", func(t *testing.T) {
		values := url.Values{}
		values.Set("filter[status]", "published")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Filters == nil {
			t.Fatal("expected filters to be parsed")
		}

		found := false
		for _, cond := range q.Filters.Conditions {
			if cond.Field == "status" && cond.Value == "published" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected status=published filter condition")
		}
	})

	t.Run("filter with operator", func(t *testing.T) {
		values := url.Values{}
		values.Set("filter[price]", "gte.100")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Filters == nil {
			t.Fatal("expected filters to be parsed")
		}

		found := false
		for _, cond := range q.Filters.Conditions {
			if cond.Field == "price" && cond.Operator == OpGreaterThanEqual {
				found = true
				// Value is a string
				break
			}
		}
		if !found {
			t.Error("expected price>=100 filter condition")
		}
	})

	t.Run("multiple filters", func(t *testing.T) {
		values := url.Values{}
		values.Set("filter[status]", "published")
		values.Set("filter[category]", "news")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Filters == nil {
			t.Fatal("expected filters to be parsed")
		}

		if len(q.Filters.Conditions) != 2 {
			t.Errorf("expected 2 filter conditions, got %d", len(q.Filters.Conditions))
		}
	})

	t.Run("filter with in operator", func(t *testing.T) {
		values := url.Values{}
		values.Set("filter[status]", "in.draft,published")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Filters == nil {
			t.Fatal("expected filters to be parsed")
		}

		found := false
		for _, cond := range q.Filters.Conditions {
			if cond.Field == "status" && cond.Operator == OpIn {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected status IN filter condition")
		}
	})
}

func TestURLParser_Parse_Pagination(t *testing.T) {
	parser := NewURLParser()

	t.Run("page and pageSize", func(t *testing.T) {
		values := url.Values{}
		values.Set("page", "2")
		values.Set("pageSize", "25")

		q, err := parser.Parse(values)
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

	t.Run("limit and offset", func(t *testing.T) {
		values := url.Values{}
		values.Set("limit", "10")
		values.Set("offset", "20")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Limit != 10 {
			t.Errorf("expected limit 10, got %d", q.Limit)
		}
		if q.Offset != 20 {
			t.Errorf("expected offset 20, got %d", q.Offset)
		}
	})

	t.Run("per_page alias", func(t *testing.T) {
		values := url.Values{}
		values.Set("page", "1")
		values.Set("per_page", "50") // URL parser uses per_page not perPage

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.PageSize != 50 {
			t.Errorf("expected pageSize 50, got %d", q.PageSize)
		}
	})

	t.Run("invalid page number uses default", func(t *testing.T) {
		values := url.Values{}
		values.Set("page", "invalid")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error (invalid values should be ignored), got %v", err)
		}

		// NewQuery() initializes Page to 1
		if q.Page != 1 {
			t.Errorf("expected page 1 (default) for invalid input, got %d", q.Page)
		}
	})
}

func TestURLParser_Parse_Sort(t *testing.T) {
	parser := NewURLParser()

	t.Run("single sort field descending", func(t *testing.T) {
		values := url.Values{}
		values.Set("sort", "-createdAt")

		q, err := parser.Parse(values)
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

	t.Run("single sort field ascending", func(t *testing.T) {
		values := url.Values{}
		values.Set("sort", "title")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Sort) != 1 {
			t.Fatalf("expected 1 sort field, got %d", len(q.Sort))
		}

		if q.Sort[0].Field != "title" {
			t.Errorf("expected field 'title', got '%s'", q.Sort[0].Field)
		}
		if q.Sort[0].Descending {
			t.Error("expected descending to be false")
		}
	})

	t.Run("multiple sort fields comma separated", func(t *testing.T) {
		values := url.Values{}
		values.Set("sort", "-updatedAt,title")

		q, err := parser.Parse(values)
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

}

func TestURLParser_Parse_Search(t *testing.T) {
	parser := NewURLParser()

	t.Run("search param", func(t *testing.T) {
		values := url.Values{}
		values.Set("search", "hello world")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Search != "hello world" {
			t.Errorf("expected search 'hello world', got '%s'", q.Search)
		}
	})

	t.Run("q alias", func(t *testing.T) {
		values := url.Values{}
		values.Set("q", "search term")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if q.Search != "search term" {
			t.Errorf("expected search 'search term', got '%s'", q.Search)
		}
	})

}

func TestURLParser_Parse_Select(t *testing.T) {
	parser := NewURLParser()

	t.Run("select param", func(t *testing.T) {
		values := url.Values{}
		values.Set("select", "id,title,status")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Select) != 3 {
			t.Fatalf("expected 3 select fields, got %d", len(q.Select))
		}

		expected := []string{"id", "title", "status"}
		for i, v := range expected {
			if q.Select[i] != v {
				t.Errorf("expected select[%d] = '%s', got '%s'", i, v, q.Select[i])
			}
		}
	})

	t.Run("fields alias", func(t *testing.T) {
		values := url.Values{}
		values.Set("fields", "id,title")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Select) != 2 {
			t.Fatalf("expected 2 select fields, got %d", len(q.Select))
		}
	})
}

func TestURLParser_Parse_Status(t *testing.T) {
	parser := NewURLParser()

	values := url.Values{}
	values.Set("status", "published")

	q, err := parser.Parse(values)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Status != "published" {
		t.Errorf("expected status 'published', got '%s'", q.Status)
	}
}

func TestURLParser_Parse_Populate(t *testing.T) {
	parser := NewURLParser()

	t.Run("single populate", func(t *testing.T) {
		values := url.Values{}
		values.Set("populate", "author")

		q, err := parser.Parse(values)
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

	t.Run("multiple populate comma separated", func(t *testing.T) {
		values := url.Values{}
		values.Set("populate", "author,category")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Populate) != 2 {
			t.Fatalf("expected 2 populate options, got %d", len(q.Populate))
		}
	})

	t.Run("include alias", func(t *testing.T) {
		values := url.Values{}
		values.Set("include", "author")

		q, err := parser.Parse(values)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(q.Populate) != 1 {
			t.Fatalf("expected 1 populate option, got %d", len(q.Populate))
		}
	})
}

func TestURLParser_Parse_CompleteQuery(t *testing.T) {
	parser := NewURLParser()

	values := url.Values{}
	values.Set("filter[status]", "eq.published")
	values.Set("filter[category]", "news")
	values.Set("sort", "-updatedAt")
	values.Set("page", "2")
	values.Set("pageSize", "25")
	values.Set("search", "hello")
	values.Set("select", "id,title,status")
	values.Set("populate", "author")

	q, err := parser.Parse(values)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify filters
	if q.Filters == nil || len(q.Filters.Conditions) < 2 {
		t.Error("expected at least 2 filter conditions")
	}

	// Verify sort
	if len(q.Sort) != 1 || q.Sort[0].Field != "updatedAt" {
		t.Error("expected sort by updatedAt")
	}

	// Verify pagination
	if q.Page != 2 || q.PageSize != 25 {
		t.Error("expected page 2 and pageSize 25")
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

func TestURLParser_Parse_EmptyValues(t *testing.T) {
	parser := NewURLParser()

	values := url.Values{}

	q, err := parser.Parse(values)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if q.Filters != nil && len(q.Filters.Conditions) > 0 {
		t.Error("expected no filters for empty values")
	}

	if len(q.Sort) != 0 {
		t.Error("expected no sort for empty values")
	}

	// NewQuery initializes Page to 1
	if q.Page != 1 {
		t.Errorf("expected page 1 (default), got %d", q.Page)
	}
}

func TestURLParser_Parse_AllOperators(t *testing.T) {
	parser := NewURLParser()

	tests := []struct {
		name     string
		value    string
		expected FilterOperator
	}{
		{"eq", "eq.value", OpEqual},
		{"ne", "ne.value", OpNotEqual},
		{"gt", "gt.10", OpGreaterThan},
		{"gte", "gte.10", OpGreaterThanEqual},
		{"lt", "lt.10", OpLessThan},
		{"lte", "lte.10", OpLessThanEqual},
		{"like", "like.%test%", OpLike},
		{"ilike", "ilike.%test%", OpILike},
		{"in", "in.a,b,c", OpIn},
		{"nin", "nin.a,b,c", OpNotIn},
		{"null", "null.true", OpNull},
		{"contains", "contains.test", OpContains},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := url.Values{}
			values.Set("filter[field]", tt.value)

			q, err := parser.Parse(values)
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

// Benchmark tests
func BenchmarkURLParser_Parse_Simple(b *testing.B) {
	parser := NewURLParser()
	values := url.Values{}
	values.Set("filter[status]", "published")
	values.Set("page", "1")
	values.Set("pageSize", "10")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(values)
	}
}

func BenchmarkURLParser_Parse_Complex(b *testing.B) {
	parser := NewURLParser()
	values := url.Values{}
	values.Set("filter[status]", "in.published,draft")
	values.Set("filter[category]", "news")
	values.Set("filter[price]", "gte.100")
	values.Set("sort", "-updatedAt,title")
	values.Set("page", "2")
	values.Set("pageSize", "25")
	values.Set("search", "hello")
	values.Set("populate", "author,category")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(values)
	}
}

