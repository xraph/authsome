package definition

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xraph/authsome/internal/errs"
)

// Validate validates the schema for correctness.
func (s *Schema) Validate() error {
	if s.Version == "" {
		return errs.RequiredField("version")
	}

	if len(s.Models) == 0 {
		return errs.RequiredField("models")
	}

	// Validate each model
	for name, model := range s.Models {
		if err := s.validateModel(name, model); err != nil {
			return fmt.Errorf("model %s: %w", name, err)
		}
	}

	// Validate references
	if err := s.validateReferences(); err != nil {
		return fmt.Errorf("reference validation: %w", err)
	}

	return nil
}

// validateModel validates a single model.
func (s *Schema) validateModel(name string, model Model) error {
	if model.Name == "" {
		return errs.RequiredField("name")
	}

	if model.Table == "" {
		return errs.RequiredField("table")
	}

	if len(model.Fields) == 0 {
		return errs.RequiredField("fields")
	}

	// Check for primary key
	hasPrimaryKey := false
	fieldNames := make(map[string]bool)

	for _, field := range model.Fields {
		// Check for duplicate field names
		if fieldNames[field.Name] {
			return fmt.Errorf("duplicate field name: %s", field.Name)
		}

		fieldNames[field.Name] = true

		// Validate field
		if err := validateField(field); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}

		if field.Primary {
			if hasPrimaryKey {
				return errs.BadRequest("multiple primary keys defined")
			}

			hasPrimaryKey = true
		}
	}

	if !hasPrimaryKey {
		return errs.BadRequest("model must have a primary key")
	}

	// Validate indexes
	for _, idx := range model.Indexes {
		if err := s.validateIndex(model, idx); err != nil {
			return fmt.Errorf("index %s: %w", idx.Name, err)
		}
	}

	return nil
}

// validateField validates a single field.
func validateField(field Field) error {
	if field.Name == "" {
		return errs.RequiredField("name")
	}

	if field.Column == "" {
		return errs.RequiredField("column")
	}

	if field.Type == "" {
		return errs.RequiredField("type")
	}

	// Validate field type
	validTypes := []FieldType{
		FieldTypeString, FieldTypeText, FieldTypeInteger, FieldTypeBigInt,
		FieldTypeFloat, FieldTypeDecimal, FieldTypeBoolean, FieldTypeTimestamp,
		FieldTypeDate, FieldTypeTime, FieldTypeUUID, FieldTypeJSON,
		FieldTypeJSONB, FieldTypeBinary, FieldTypeEnum,
	}

	isValidType := slices.Contains(validTypes, field.Type)

	if !isValidType {
		return fmt.Errorf("invalid field type: %s", field.Type)
	}

	// Primary keys must be required
	if field.Primary && !field.Required {
		return errs.BadRequest("primary key must be required")
	}

	// Can't be both required and nullable
	if field.Required && field.Nullable {
		return errs.BadRequest("field cannot be both required and nullable")
	}

	return nil
}

// validateIndex validates an index.
func (s *Schema) validateIndex(model Model, idx Index) error {
	if idx.Name == "" {
		return errs.RequiredField("name")
	}

	if len(idx.Columns) == 0 {
		return errs.RequiredField("columns")
	}

	// Verify columns exist in model
	fieldMap := make(map[string]bool)
	for _, field := range model.Fields {
		fieldMap[field.Column] = true
	}

	for _, col := range idx.Columns {
		if !fieldMap[col] {
			return fmt.Errorf("index column %s does not exist in model", col)
		}
	}

	return nil
}

// validateReferences validates all foreign key references.
func (s *Schema) validateReferences() error {
	for modelName, model := range s.Models {
		for _, field := range model.Fields {
			if field.References != nil {
				// Check if referenced model exists
				refModel, exists := s.Models[field.References.Model]
				if !exists {
					return fmt.Errorf("model %s field %s references non-existent model %s",
						modelName, field.Name, field.References.Model)
				}

				// Check if referenced field exists
				fieldExists := false

				for _, refField := range refModel.Fields {
					if refField.Name == field.References.Field {
						fieldExists = true

						break
					}
				}

				if !fieldExists {
					return fmt.Errorf("model %s field %s references non-existent field %s in model %s",
						modelName, field.Name, field.References.Field, field.References.Model)
				}
			}
		}

		// Validate relations
		for _, rel := range model.Relations {
			if _, exists := s.Models[rel.Model]; !exists {
				return fmt.Errorf("model %s relation %s references non-existent model %s",
					modelName, rel.Name, rel.Model)
			}
		}
	}

	return nil
}

// ValidateName checks if a name is valid (alphanumeric + underscore).
func ValidateName(name string) error {
	if name == "" {
		return errs.RequiredField("name")
	}

	for i, c := range name {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			return fmt.Errorf("invalid character '%c' at position %d in name '%s'", c, i, name)
		}
	}

	// Cannot start with a number
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Errorf("name cannot start with a number: %s", name)
	}

	return nil
}

// ToSnakeCase converts a string to snake_case.
func ToSnakeCase(s string) string {
	var result strings.Builder

	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result.WriteRune('_')
			}

			result.WriteRune(c + 32) // Convert to lowercase
		} else {
			result.WriteRune(c)
		}
	}

	return result.String()
}
