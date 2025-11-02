package language

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParser(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)
	assert.NotNil(t, parser)
	assert.NotNil(t, parser.env)
}

func TestParser_Parse_ValidExpressions(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)
	
	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "simple equality",
			expression: `principal.id == "user_123"`,
			wantErr:    false,
		},
		{
			name:       "resource owner check",
			expression: `resource.owner == principal.id`,
			wantErr:    false,
		},
		{
			name:       "boolean OR",
			expression: `has_role("admin") || resource.owner == principal.id`,
			wantErr:    false,
		},
		{
			name:       "boolean AND",
			expression: `principal.department == "engineering" && resource.visibility == "internal"`,
			wantErr:    false,
		},
		{
			name:       "nested property access",
			expression: `resource.metadata.confidentiality == "public"`,
			wantErr:    false,
		},
		{
			name:       "collection membership",
			expression: `principal.department in ["engineering", "operations"]`,
			wantErr:    false,
		},
		{
			name:       "collection exists",
			expression: `principal.roles.exists(r, r == "admin")`,
			wantErr:    false,
		},
		{
			name:       "has_role function",
			expression: `has_role("admin")`,
			wantErr:    false,
		},
		{
			name:       "in_time_range function",
			expression: `in_time_range("09:00", "17:00")`,
			wantErr:    false,
		},
		{
			name:       "is_weekday function",
			expression: `is_weekday()`,
			wantErr:    false,
		},
		{
			name:       "ip_in_range function",
			expression: `ip_in_range(["10.0.0.0/8"])`,
			wantErr:    false,
		},
		{
			name:       "complex expression",
			expression: `(resource.owner == principal.id || has_role("admin")) && resource.enabled == true`,
			wantErr:    false,
		},
		{
			name:       "ternary operator",
			expression: `has_role("admin") ? true : resource.owner == principal.id`,
			wantErr:    false,
		},
		{
			name:       "string contains",
			expression: `principal.email.contains("@company.com")`,
			wantErr:    false,
		},
		{
			name:       "string starts with",
			expression: `resource.id.startsWith("project:")`,
			wantErr:    false,
		},
		{
			name:       "comparison operators",
			expression: `resource.size < 1000000`,
			wantErr:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parser.Parse(tt.expression)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, ast)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ast)
			}
		})
	}
}

func TestParser_Parse_InvalidExpressions(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)
	
	tests := []struct {
		name       string
		expression string
		errorMsg   string
	}{
		{
			name:       "empty expression",
			expression: "",
			errorMsg:   "expression cannot be empty",
		},
		{
			name:       "syntax error - unclosed string",
			expression: `principal.id == "admin`,
			errorMsg:   "parse error",
		},
		{
			name:       "syntax error - undefined variable",
			expression: `unknown_var == "value"`,
			errorMsg:   "parse error",
		},
		{
			name:       "non-boolean return type",
			expression: `principal.id`,
			errorMsg:   "must return boolean",
		},
		{
			name:       "non-boolean return type - int",
			expression: `1 + 2`,
			errorMsg:   "must return boolean",
		},
		{
			name:       "invalid function call",
			expression: `has_role()`,
			errorMsg:   "parse error",
		},
		{
			name:       "type mismatch",
			expression: `principal.id + 123`,
			errorMsg:   "expression must return boolean",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parser.Parse(tt.expression)
			assert.Error(t, err)
			assert.Nil(t, ast)
			assert.Contains(t, err.Error(), tt.errorMsg)
		})
	}
}

func TestParser_Program(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)
	
	expression := `resource.owner == principal.id`
	ast, err := parser.Parse(expression)
	require.NoError(t, err)
	
	prg, err := parser.Program(ast)
	require.NoError(t, err)
	assert.NotNil(t, prg)
}

func TestParser_ValidateExpression(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)
	
	tests := []struct {
		name       string
		expression string
		wantErr    bool
	}{
		{
			name:       "valid expression",
			expression: `resource.owner == principal.id`,
			wantErr:    false,
		},
		{
			name:       "invalid expression",
			expression: `resource.owner ==`,
			wantErr:    true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.ValidateExpression(tt.expression)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParser_ExampleExpressions(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)
	
	// Verify all example expressions are valid
	for name, expression := range ExampleExpressions {
		t.Run(name, func(t *testing.T) {
			ast, err := parser.Parse(expression)
			assert.NoError(t, err, "Example expression '%s' should be valid", name)
			assert.NotNil(t, ast)
		})
	}
}

func TestParser_ExpressionComplexity(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)
	
	tests := []struct {
		name           string
		expression     string
		minComplexity  int
		maxComplexity  int
	}{
		{
			name:          "simple equality",
			expression:    `principal.id == "user_123"`,
			minComplexity: 1,
			maxComplexity: 10,
		},
		{
			name:          "complex expression",
			expression:    `(resource.owner == principal.id || has_role("admin")) && resource.enabled == true`,
			minComplexity: 5,
			maxComplexity: 20,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parser.Parse(tt.expression)
			require.NoError(t, err)
			
			complexity := parser.ExpressionComplexity(ast)
			assert.GreaterOrEqual(t, complexity, tt.minComplexity)
			assert.LessOrEqual(t, complexity, tt.maxComplexity)
		})
	}
}

func TestParser_GetFunctionHelp(t *testing.T) {
	parser, err := NewParser()
	require.NoError(t, err)
	
	help := parser.GetFunctionHelp()
	assert.NotEmpty(t, help)
	assert.Contains(t, help, "has_role(role)")
	assert.Contains(t, help, "in_time_range(start, end)")
	assert.Contains(t, help, "is_weekday()")
}

func BenchmarkParser_Parse(b *testing.B) {
	parser, err := NewParser()
	require.NoError(b, err)
	
	expression := `resource.owner == principal.id || has_role("admin")`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(expression)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParser_Program(b *testing.B) {
	parser, err := NewParser()
	require.NoError(b, err)
	
	expression := `resource.owner == principal.id || has_role("admin")`
	ast, err := parser.Parse(expression)
	require.NoError(b, err)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Program(ast)
		if err != nil {
			b.Fatal(err)
		}
	}
}

