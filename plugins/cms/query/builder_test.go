package query

import (
	"testing"
)

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"createdAt", "created_at"},
		{"updatedAt", "updated_at"},
		{"publishedAt", "published_at"},
		{"scheduledAt", "scheduled_at"},
		{"id", "id"},
		{"status", "status"},
		{"firstName", "first_name"},
		{"lastName", "last_name"},
		{"HTMLParser", "h_t_m_l_parser"},
		{"simpleTest", "simple_test"},
		{"", ""},
		{"a", "a"},
		{"A", "a"},
		{"ABC", "a_b_c"},
		{"contentTypeID", "content_type_i_d"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"int64", int64(123), "123"},
		{"float64", 3.14, "3.14"},
		{"float64 whole", 10.0, "10"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "<nil>"},
		{"slice", []int{1, 2}, "[1 2]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			if result != tt.expected {
				t.Errorf("toString(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string true", "true", true},
		{"string 1", "1", true},
		{"string false", "false", false},
		{"string 0", "0", false},
		{"string empty", "", false},
		{"int non-zero", 1, true},
		{"int zero", 0, false},
		{"int negative", -1, true},
		{"float64 non-zero", 1.0, true},
		{"float64 zero", 0.0, false},
		{"nil", nil, false},
		{"string random", "random", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toBool(tt.input)
			if result != tt.expected {
				t.Errorf("toBool(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToSlice(t *testing.T) {
	t.Run("interface slice", func(t *testing.T) {
		input := []interface{}{"a", "b", "c"}
		result := toSlice(input)
		if len(result) != 3 {
			t.Errorf("expected length 3, got %d", len(result))
		}
	})

	t.Run("string slice", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := toSlice(input)
		if len(result) != 3 {
			t.Errorf("expected length 3, got %d", len(result))
		}
	})

	t.Run("single value", func(t *testing.T) {
		input := "single"
		result := toSlice(input)
		if len(result) != 1 {
			t.Errorf("expected length 1, got %d", len(result))
		}
		if result[0] != "single" {
			t.Errorf("expected 'single', got %v", result[0])
		}
	})

	t.Run("nil", func(t *testing.T) {
		result := toSlice(nil)
		if len(result) != 1 {
			t.Errorf("expected length 1 (wrapping nil), got %d", len(result))
		}
	})
}

func TestToStringSlice(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := toStringSlice(input)
		if len(result) != 3 {
			t.Errorf("expected length 3, got %d", len(result))
		}
		if result[0] != "a" || result[1] != "b" || result[2] != "c" {
			t.Errorf("unexpected values: %v", result)
		}
	})

	t.Run("interface slice", func(t *testing.T) {
		input := []interface{}{"a", 1, true}
		result := toStringSlice(input)
		if len(result) != 3 {
			t.Errorf("expected length 3, got %d", len(result))
		}
		if result[0] != "a" {
			t.Errorf("expected 'a', got %s", result[0])
		}
		if result[1] != "1" {
			t.Errorf("expected '1', got %s", result[1])
		}
		if result[2] != "true" {
			t.Errorf("expected 'true', got %s", result[2])
		}
	})

	t.Run("single string", func(t *testing.T) {
		input := "single"
		result := toStringSlice(input)
		if len(result) != 1 {
			t.Errorf("expected length 1, got %d", len(result))
		}
		if result[0] != "single" {
			t.Errorf("expected 'single', got %s", result[0])
		}
	})

	t.Run("single int", func(t *testing.T) {
		input := 42
		result := toStringSlice(input)
		if len(result) != 1 {
			t.Errorf("expected length 1, got %d", len(result))
		}
		if result[0] != "42" {
			t.Errorf("expected '42', got %s", result[0])
		}
	})
}

// Test helper functions from json_parser.go
func TestSplitFirst(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"a:b", ":", []string{"a", "b"}},
		{"field:desc", ":", []string{"field", "desc"}},
		{"noseparator", ":", []string{"noseparator"}},
		{"a:b:c", ":", []string{"a", "b:c"}},
		{"", ":", []string{""}},
		{":value", ":", []string{"", "value"}},
		{"key:", ":", []string{"key", ""}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitFirst(tt.input, tt.sep)
			if len(result) != len(tt.expected) {
				t.Errorf("splitFirst(%q, %q) returned %d parts, expected %d", tt.input, tt.sep, len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitFirst(%q, %q)[%d] = %q, expected %q", tt.input, tt.sep, i, v, tt.expected[i])
				}
			}
		})
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello  ", "hello"},
		{"hello", "hello"},
		{"  hello", "hello"},
		{"hello  ", "hello"},
		{"\t\nhello\t\n", "hello"},
		{"   ", ""},
		{"", ""},
		{" \t\r\n ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := trimSpace(tt.input)
			if result != tt.expected {
				t.Errorf("trimSpace(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"a , b", ",", []string{"a", "b"}}, // Note: splitFirst only splits on first occurrence
		{" hello ", ",", []string{"hello"}},
		{"", ",", []string{}},
		{"   ", ",", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitAndTrim(tt.input, tt.sep)
			if len(result) != len(tt.expected) {
				t.Errorf("splitAndTrim(%q, %q) returned %d parts, expected %d: %v", tt.input, tt.sep, len(result), len(tt.expected), result)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitAndTrim(%q, %q)[%d] = %q, expected %q", tt.input, tt.sep, i, v, tt.expected[i])
				}
			}
		})
	}
}

// Benchmark helper functions
func BenchmarkToSnakeCase(b *testing.B) {
	inputs := []string{"createdAt", "updatedAt", "publishedAt", "contentTypeID"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			toSnakeCase(input)
		}
	}
}

func BenchmarkToString(b *testing.B) {
	inputs := []interface{}{"hello", 42, 3.14, true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			toString(input)
		}
	}
}
