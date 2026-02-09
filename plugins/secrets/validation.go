package secrets

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/secrets/core"
)

// SchemaValidator validates secret values against JSON Schema and handles
// serialization/deserialization of different value types.
type SchemaValidator struct {
	// schemas map[string]*jsonschema.Schema // cached compiled schemas
}

// NewSchemaValidator creates a new schema validator.
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{}
}

// ValidateValue validates a value against an optional JSON schema.
// If schemaJSON is empty, only basic type validation is performed.
func (v *SchemaValidator) ValidateValue(value any, valueType core.SecretValueType, schemaJSON string) error {
	// Basic type validation
	if err := v.validateType(value, valueType); err != nil {
		return err
	}

	// Schema validation if provided
	if schemaJSON != "" {
		if err := v.validateAgainstSchema(value, valueType, schemaJSON); err != nil {
			return err
		}
	}

	return nil
}

// validateType performs basic type validation based on value type.
func (v *SchemaValidator) validateType(value any, valueType core.SecretValueType) error {
	switch valueType {
	case core.SecretValueTypePlain:
		// Plain values must be strings
		if _, ok := value.(string); !ok {
			return core.ErrValidationFailed("plain value must be a string", nil)
		}

	case core.SecretValueTypeJSON:
		// JSON values can be any valid JSON type
		// Validate by attempting to marshal/unmarshal
		if _, err := json.Marshal(value); err != nil {
			return core.ErrValidationFailed("invalid JSON value", err)
		}

	case core.SecretValueTypeYAML:
		// YAML values can be any valid YAML type
		// If it's a string, try to parse it as YAML
		if str, ok := value.(string); ok {
			var parsed any
			if err := yaml.Unmarshal([]byte(str), &parsed); err != nil {
				return core.ErrValidationFailed("invalid YAML value", err)
			}
		}

	case core.SecretValueTypeBinary:
		// Binary values must be valid base64 strings
		if str, ok := value.(string); ok {
			if _, err := base64.StdEncoding.DecodeString(str); err != nil {
				return core.ErrValidationFailed("binary value must be valid base64", err)
			}
		} else {
			return core.ErrValidationFailed("binary value must be a base64-encoded string", nil)
		}

	default:
		return core.ErrInvalidValueType(string(valueType))
	}

	return nil
}

// validateAgainstSchema validates a value against a JSON Schema.
func (v *SchemaValidator) validateAgainstSchema(value any, valueType core.SecretValueType, schemaJSON string) error {
	// Parse the schema to validate it's valid JSON
	var schema map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return core.ErrSchemaInvalid("schema is not valid JSON", err)
	}

	// For YAML values, convert to JSON-compatible format first
	var jsonValue any

	if valueType == core.SecretValueTypeYAML {
		if str, ok := value.(string); ok {
			var parsed any
			if err := yaml.Unmarshal([]byte(str), &parsed); err != nil {
				return core.ErrValidationFailed("failed to parse YAML for schema validation", err)
			}

			jsonValue = convertYAMLToJSON(parsed)
		} else {
			jsonValue = value
		}
	} else {
		jsonValue = value
	}

	// Basic schema validation (type checking)
	// Note: For full JSON Schema validation, consider using github.com/santhosh-tekuri/jsonschema/v5
	if schemaType, ok := schema["type"].(string); ok {
		if err := v.validateSchemaType(jsonValue, schemaType); err != nil {
			return err
		}
	}

	// Validate required fields for objects
	if required, ok := schema["required"].([]any); ok {
		if objMap, ok := jsonValue.(map[string]any); ok {
			for _, req := range required {
				if reqStr, ok := req.(string); ok {
					if _, exists := objMap[reqStr]; !exists {
						return core.ErrValidationFailed(fmt.Sprintf("required field '%s' is missing", reqStr), nil)
					}
				}
			}
		}
	}

	// Validate string constraints
	if valueType == core.SecretValueTypePlain {
		if str, ok := jsonValue.(string); ok {
			if minLen, ok := schema["minLength"].(float64); ok {
				if len(str) < int(minLen) {
					return core.ErrValidationFailed(fmt.Sprintf("string length must be at least %d", int(minLen)), nil)
				}
			}

			if maxLen, ok := schema["maxLength"].(float64); ok {
				if len(str) > int(maxLen) {
					return core.ErrValidationFailed(fmt.Sprintf("string length must be at most %d", int(maxLen)), nil)
				}
			}
		}
	}

	return nil
}

// validateSchemaType validates a value against a JSON Schema type.
func (v *SchemaValidator) validateSchemaType(value any, schemaType string) error {
	switch schemaType {
	case "string":
		if _, ok := value.(string); !ok {
			return core.ErrValidationFailed("value must be a string", nil)
		}
	case "number", "integer":
		switch value.(type) {
		case float64, float32, int, int64, int32:
			// Valid number types
		default:
			return core.ErrValidationFailed("value must be a number", nil)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return core.ErrValidationFailed("value must be a boolean", nil)
		}
	case "object":
		if _, ok := value.(map[string]any); !ok {
			return core.ErrValidationFailed("value must be an object", nil)
		}
	case "array":
		if _, ok := value.([]any); !ok {
			return core.ErrValidationFailed("value must be an array", nil)
		}
	case "null":
		if value != nil {
			return core.ErrValidationFailed("value must be null", nil)
		}
	}

	return nil
}

// ParseValue parses a raw string value based on the value type.
func (v *SchemaValidator) ParseValue(raw string, valueType core.SecretValueType) (any, error) {
	switch valueType {
	case core.SecretValueTypePlain:
		return raw, nil

	case core.SecretValueTypeJSON:
		var result any
		if err := json.Unmarshal([]byte(raw), &result); err != nil {
			return nil, core.ErrDeserializationFailed("json", err)
		}

		return result, nil

	case core.SecretValueTypeYAML:
		var result any
		if err := yaml.Unmarshal([]byte(raw), &result); err != nil {
			return nil, core.ErrDeserializationFailed("yaml", err)
		}
		// Convert YAML-specific types to JSON-compatible types
		return convertYAMLToJSON(result), nil

	case core.SecretValueTypeBinary:
		// Return as-is (base64 string)
		// Validation happens separately
		return raw, nil

	default:
		return nil, core.ErrInvalidValueType(string(valueType))
	}
}

// SerializeValue serializes a value for storage based on the value type.
func (v *SchemaValidator) SerializeValue(value any, valueType core.SecretValueType) ([]byte, error) {
	switch valueType {
	case core.SecretValueTypePlain:
		if str, ok := value.(string); ok {
			return []byte(str), nil
		}

		return nil, core.ErrSerializationFailed("plain", errs.BadRequest("value must be a string"))

	case core.SecretValueTypeJSON:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, core.ErrSerializationFailed("json", err)
		}

		return data, nil

	case core.SecretValueTypeYAML:
		// If value is already a string (raw YAML), return as-is
		if str, ok := value.(string); ok {
			return []byte(str), nil
		}
		// Otherwise, marshal the value to YAML
		data, err := yaml.Marshal(value)
		if err != nil {
			return nil, core.ErrSerializationFailed("yaml", err)
		}

		return data, nil

	case core.SecretValueTypeBinary:
		if str, ok := value.(string); ok {
			return []byte(str), nil
		}

		return nil, core.ErrSerializationFailed("binary", errs.BadRequest("value must be a base64-encoded string"))

	default:
		return nil, core.ErrInvalidValueType(string(valueType))
	}
}

// DeserializeValue deserializes stored bytes back to a value based on the value type.
func (v *SchemaValidator) DeserializeValue(data []byte, valueType core.SecretValueType) (any, error) {
	return v.ParseValue(string(data), valueType)
}

// DetectValueType attempts to detect the value type from the value content.
func (v *SchemaValidator) DetectValueType(value any) core.SecretValueType {
	switch val := value.(type) {
	case string:
		// Check if it's valid JSON
		if strings.HasPrefix(strings.TrimSpace(val), "{") || strings.HasPrefix(strings.TrimSpace(val), "[") {
			var js any
			if err := json.Unmarshal([]byte(val), &js); err == nil {
				return core.SecretValueTypeJSON
			}
		}

		// Check if it looks like YAML (has key: value patterns)
		if strings.Contains(val, ":") && (strings.Contains(val, "\n") || strings.HasPrefix(val, "---")) {
			var ym any
			if err := yaml.Unmarshal([]byte(val), &ym); err == nil {
				// Only return YAML if it parsed to something other than a plain string
				if _, isStr := ym.(string); !isStr {
					return core.SecretValueTypeYAML
				}
			}
		}

		// Check if it's base64
		if _, err := base64.StdEncoding.DecodeString(val); err == nil {
			// Only consider it binary if it's relatively long and doesn't look like plain text
			if len(val) > 20 && !isPrintableASCII(val) {
				return core.SecretValueTypeBinary
			}
		}

		return core.SecretValueTypePlain

	case map[string]any, []any:
		return core.SecretValueTypeJSON

	default:
		return core.SecretValueTypePlain
	}
}

// ValidateSchema validates that a JSON schema is valid.
func (v *SchemaValidator) ValidateSchema(schemaJSON string) error {
	if schemaJSON == "" {
		return nil
	}

	var schema map[string]any
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return core.ErrSchemaInvalid("schema is not valid JSON", err)
	}

	// Basic validation: check for common schema properties
	if schemaType, ok := schema["type"]; ok {
		validTypes := map[string]bool{
			"string": true, "number": true, "integer": true,
			"boolean": true, "object": true, "array": true, "null": true,
		}
		if typeStr, ok := schemaType.(string); ok {
			if !validTypes[typeStr] {
				return core.ErrSchemaInvalid(fmt.Sprintf("invalid type '%s'", typeStr), nil)
			}
		}
	}

	return nil
}

// Helper functions

// convertYAMLToJSON converts YAML-parsed values to JSON-compatible types.
func convertYAMLToJSON(value any) any {
	switch v := value.(type) {
	case map[any]any:
		// YAML maps have interface{} keys, convert to string keys
		result := make(map[string]any)
		for key, val := range v {
			result[fmt.Sprintf("%v", key)] = convertYAMLToJSON(val)
		}

		return result

	case map[string]any:
		result := make(map[string]any)
		for key, val := range v {
			result[key] = convertYAMLToJSON(val)
		}

		return result

	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = convertYAMLToJSON(val)
		}

		return result

	default:
		return v
	}
}

// isPrintableASCII checks if a string contains only printable ASCII characters.
func isPrintableASCII(s string) bool {
	for _, r := range s {
		if r < 32 || r > 126 {
			return false
		}
	}

	return true
}

// FormatValue formats a value for display based on its type.
func FormatValue(value any, valueType core.SecretValueType) string {
	switch valueType {
	case core.SecretValueTypePlain:
		if str, ok := value.(string); ok {
			return str
		}

		return fmt.Sprintf("%v", value)

	case core.SecretValueTypeJSON:
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return fmt.Sprintf("%v", value)
		}

		return string(data)

	case core.SecretValueTypeYAML:
		if str, ok := value.(string); ok {
			return str
		}

		data, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}

		return string(data)

	case core.SecretValueTypeBinary:
		if str, ok := value.(string); ok {
			return str
		}

		return fmt.Sprintf("%v", value)

	default:
		return fmt.Sprintf("%v", value)
	}
}
