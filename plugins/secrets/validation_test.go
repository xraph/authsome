package secrets

import (
	"testing"

	"github.com/xraph/authsome/plugins/secrets/core"
)

func TestSchemaValidator_ValidateType(t *testing.T) {
	v := NewSchemaValidator()

	tests := []struct {
		name      string
		value     interface{}
		valueType core.SecretValueType
		wantErr   bool
	}{
		// Plain text tests
		{"plain valid", "hello world", core.SecretValueTypePlain, false},
		{"plain invalid", 123, core.SecretValueTypePlain, true},

		// JSON tests
		{"json string", `{"key": "value"}`, core.SecretValueTypeJSON, false},
		{"json object", map[string]interface{}{"key": "value"}, core.SecretValueTypeJSON, false},
		{"json array", []interface{}{1, 2, 3}, core.SecretValueTypeJSON, false},

		// YAML tests
		{"yaml string", "key: value\nother: data", core.SecretValueTypeYAML, false},
		{"yaml complex", "---\nkey: value", core.SecretValueTypeYAML, false},

		// Binary tests
		{"binary valid", "aGVsbG8gd29ybGQ=", core.SecretValueTypeBinary, false},
		{"binary invalid", "not-valid-base64!@#$", core.SecretValueTypeBinary, true},
		{"binary non-string", 123, core.SecretValueTypeBinary, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateValue(tt.value, tt.valueType, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_ValidateWithSchema(t *testing.T) {
	v := NewSchemaValidator()

	tests := []struct {
		name      string
		value     interface{}
		valueType core.SecretValueType
		schema    string
		wantErr   bool
	}{
		{
			name:      "object with required fields - valid",
			value:     map[string]interface{}{"host": "localhost", "port": 5432},
			valueType: core.SecretValueTypeJSON,
			schema:    `{"type": "object", "required": ["host", "port"]}`,
			wantErr:   false,
		},
		{
			name:      "object with required fields - missing",
			value:     map[string]interface{}{"host": "localhost"},
			valueType: core.SecretValueTypeJSON,
			schema:    `{"type": "object", "required": ["host", "port"]}`,
			wantErr:   true,
		},
		{
			name:      "string with minLength - valid",
			value:     "hello",
			valueType: core.SecretValueTypePlain,
			schema:    `{"type": "string", "minLength": 3}`,
			wantErr:   false,
		},
		{
			name:      "string with minLength - too short",
			value:     "hi",
			valueType: core.SecretValueTypePlain,
			schema:    `{"type": "string", "minLength": 3}`,
			wantErr:   true,
		},
		{
			name:      "string with maxLength - valid",
			value:     "hi",
			valueType: core.SecretValueTypePlain,
			schema:    `{"type": "string", "maxLength": 10}`,
			wantErr:   false,
		},
		{
			name:      "string with maxLength - too long",
			value:     "this is a very long string",
			valueType: core.SecretValueTypePlain,
			schema:    `{"type": "string", "maxLength": 10}`,
			wantErr:   true,
		},
		{
			name:      "invalid schema JSON",
			value:     "test",
			valueType: core.SecretValueTypePlain,
			schema:    `{not valid json`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateValue(tt.value, tt.valueType, tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_ParseValue(t *testing.T) {
	v := NewSchemaValidator()

	tests := []struct {
		name      string
		raw       string
		valueType core.SecretValueType
		wantErr   bool
	}{
		{"plain text", "hello world", core.SecretValueTypePlain, false},
		{"json object", `{"key": "value"}`, core.SecretValueTypeJSON, false},
		{"json array", `[1, 2, 3]`, core.SecretValueTypeJSON, false},
		{"json invalid", `{invalid}`, core.SecretValueTypeJSON, true},
		{"yaml simple", "key: value", core.SecretValueTypeYAML, false},
		{"yaml complex", "---\nkey: value\narray:\n  - item1\n  - item2", core.SecretValueTypeYAML, false},
		{"binary valid", "aGVsbG8gd29ybGQ=", core.SecretValueTypeBinary, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.ParseValue(tt.raw, tt.valueType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_SerializeValue(t *testing.T) {
	v := NewSchemaValidator()

	tests := []struct {
		name      string
		value     interface{}
		valueType core.SecretValueType
		wantErr   bool
	}{
		{"plain string", "hello world", core.SecretValueTypePlain, false},
		{"plain non-string", 123, core.SecretValueTypePlain, true},
		{"json object", map[string]interface{}{"key": "value"}, core.SecretValueTypeJSON, false},
		{"json array", []interface{}{1, 2, 3}, core.SecretValueTypeJSON, false},
		{"yaml string", "key: value", core.SecretValueTypeYAML, false},
		{"yaml object", map[string]interface{}{"key": "value"}, core.SecretValueTypeYAML, false},
		{"binary string", "aGVsbG8gd29ybGQ=", core.SecretValueTypeBinary, false},
		{"binary non-string", 123, core.SecretValueTypeBinary, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := v.SerializeValue(tt.value, tt.valueType)
			if (err != nil) != tt.wantErr {
				t.Errorf("SerializeValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_DetectValueType(t *testing.T) {
	v := NewSchemaValidator()

	tests := []struct {
		name  string
		value interface{}
		want  core.SecretValueType
	}{
		{"plain string", "hello world", core.SecretValueTypePlain},
		{"json object string", `{"key": "value"}`, core.SecretValueTypeJSON},
		{"json array string", `[1, 2, 3]`, core.SecretValueTypeJSON},
		{"yaml string", "key: value\nother: data", core.SecretValueTypeYAML},
		{"map value", map[string]interface{}{"key": "value"}, core.SecretValueTypeJSON},
		{"slice value", []interface{}{1, 2, 3}, core.SecretValueTypeJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := v.DetectValueType(tt.value); got != tt.want {
				t.Errorf("DetectValueType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSchemaValidator_ValidateSchema(t *testing.T) {
	v := NewSchemaValidator()

	tests := []struct {
		name    string
		schema  string
		wantErr bool
	}{
		{"empty schema", "", false},
		{"valid object schema", `{"type": "object"}`, false},
		{"valid string schema", `{"type": "string", "minLength": 1}`, false},
		{"valid array schema", `{"type": "array"}`, false},
		{"invalid JSON", `{not valid}`, true},
		{"invalid type", `{"type": "invalid_type"}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateSchema(tt.schema)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaValidator_RoundTrip(t *testing.T) {
	v := NewSchemaValidator()

	tests := []struct {
		name      string
		value     interface{}
		valueType core.SecretValueType
	}{
		{"plain text", "hello world", core.SecretValueTypePlain},
		{"json object", map[string]interface{}{"key": "value", "nested": map[string]interface{}{"a": 1}}, core.SecretValueTypeJSON},
		{"json array", []interface{}{1, "two", 3.0}, core.SecretValueTypeJSON},
		{"yaml", "database:\n  host: localhost\n  port: 5432", core.SecretValueTypeYAML},
		{"binary", "aGVsbG8gd29ybGQ=", core.SecretValueTypeBinary},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			serialized, err := v.SerializeValue(tt.value, tt.valueType)
			if err != nil {
				t.Fatalf("SerializeValue() error = %v", err)
			}

			// Deserialize
			deserialized, err := v.DeserializeValue(serialized, tt.valueType)
			if err != nil {
				t.Fatalf("DeserializeValue() error = %v", err)
			}

			// For plain and binary, compare strings directly
			if tt.valueType == core.SecretValueTypePlain || tt.valueType == core.SecretValueTypeBinary {
				if deserialized != tt.value {
					t.Errorf("Round-trip mismatch: got %v, want %v", deserialized, tt.value)
				}
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		valueType core.SecretValueType
		wantLen   int // Check that output is reasonable length
	}{
		{"plain", "hello", core.SecretValueTypePlain, 5},
		{"json", map[string]interface{}{"key": "value"}, core.SecretValueTypeJSON, 10},
		{"yaml", map[string]interface{}{"key": "value"}, core.SecretValueTypeYAML, 10},
		{"binary", "YmluYXJ5", core.SecretValueTypeBinary, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatValue(tt.value, tt.valueType)
			if len(got) < tt.wantLen {
				t.Errorf("FormatValue() = %v, want length >= %d", got, tt.wantLen)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateValue(b *testing.B) {
	v := NewSchemaValidator()
	value := map[string]interface{}{"host": "localhost", "port": 5432}
	schema := `{"type": "object", "required": ["host", "port"]}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidateValue(value, core.SecretValueTypeJSON, schema)
	}
}

func BenchmarkParseJSON(b *testing.B) {
	v := NewSchemaValidator()
	raw := `{"key": "value", "nested": {"a": 1, "b": 2}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ParseValue(raw, core.SecretValueTypeJSON)
	}
}

func BenchmarkParseYAML(b *testing.B) {
	v := NewSchemaValidator()
	raw := "key: value\nnested:\n  a: 1\n  b: 2"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ParseValue(raw, core.SecretValueTypeYAML)
	}
}

