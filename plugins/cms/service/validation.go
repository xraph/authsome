package service

import (
	"fmt"

	"github.com/rs/xid"

	"github.com/xraph/authsome/plugins/cms/core"
	"github.com/xraph/authsome/plugins/cms/schema"
)

// EntryValidator validates content entry data against content type schema
type EntryValidator struct {
	contentType *schema.ContentType
	fields      map[string]*schema.ContentField
}

// NewEntryValidator creates a new entry validator for a content type
func NewEntryValidator(contentType *schema.ContentType) *EntryValidator {
	fields := make(map[string]*schema.ContentField)
	for _, f := range contentType.Fields {
		fields[f.Name] = f
	}
	return &EntryValidator{
		contentType: contentType,
		fields:      fields,
	}
}

// ValidateCreate validates entry data for creation
func (v *EntryValidator) ValidateCreate(data map[string]interface{}) *core.ValidationResult {
	return v.validateData(data, nil)
}

// ValidateUpdate validates entry data for update
func (v *EntryValidator) ValidateUpdate(data map[string]interface{}, existingEntry *schema.ContentEntry) *core.ValidationResult {
	// Merge existing data with new data
	mergedData := make(map[string]interface{})
	if existingEntry != nil {
		for k, v := range existingEntry.Data {
			mergedData[k] = v
		}
	}
	for k, v := range data {
		mergedData[k] = v
	}
	
	return v.validateData(mergedData, existingEntry)
}

// validateData validates data against all fields
func (v *EntryValidator) validateData(data map[string]interface{}, existingEntry *schema.ContentEntry) *core.ValidationResult {
	result := &core.ValidationResult{Valid: true}

	// Validate each field defined in the content type
	for slug, field := range v.fields {
		// Skip hidden and read-only fields for validation
		if field.Hidden || field.ReadOnly {
			continue
		}

		value := data[slug]

		// Build field options DTO for validator
		options := v.buildOptionsDTO(field)

		// Special handling for oneOf fields
		if field.Type == "oneOf" {
			// Get the discriminator field value from data
			discriminatorValue := ""
			if options.DiscriminatorField != "" {
				if discVal, ok := data[options.DiscriminatorField]; ok {
					discriminatorValue = fmt.Sprint(discVal)
				}
			}

			// Validate oneOf with discriminator
			if value != nil {
				objValue, ok := value.(map[string]interface{})
				if !ok {
					result.Valid = false
					result.Errors = append(result.Errors, core.ValidationError{
						Field:   slug,
						Message: "must be an object",
						Code:    "invalid_type",
					})
					continue
				}

				oneOfResult := core.ValidateOneOfWithDiscriminator(objValue, discriminatorValue, options)
				if !oneOfResult.Valid {
					result.Valid = false
					for _, err := range oneOfResult.Errors {
						err.Field = slug
						result.Errors = append(result.Errors, err)
					}
				}
			} else if field.Required {
				result.Valid = false
				result.Errors = append(result.Errors, core.ValidationError{
					Field:   slug,
					Message: "this field is required",
					Code:    "required",
				})
			}
			continue
		}

		// Create validator for non-oneOf fields
		fieldValidator := core.NewFieldValidator(
			field.Name,
			core.FieldType(field.Type),
			field.Required,
			field.Unique,
			options,
		)

		// Validate field
		fieldResult := fieldValidator.Validate(value)
		if !fieldResult.Valid {
			result.Valid = false
			for _, err := range fieldResult.Errors {
				err.Field = slug
				result.Errors = append(result.Errors, err)
			}
		}
	}

	// Check for unknown fields (fields not in schema)
	for key := range data {
		if _, exists := v.fields[key]; !exists {
			// Unknown fields are allowed but logged
			// You could make this stricter if needed
		}
	}

	return result
}

// ValidateUniqueConstraints validates unique constraints (needs to be called separately with DB access)
func (v *EntryValidator) ValidateUniqueConstraints(
	data map[string]interface{},
	existingID *xid.ID,
	checkUnique func(field string, value interface{}, excludeID *xid.ID) (bool, error),
) *core.ValidationResult {
	result := &core.ValidationResult{Valid: true}

	for slug, field := range v.fields {
		if !field.Unique {
			continue
		}

		value := data[slug]
		if value == nil || value == "" {
			continue
		}

		exists, err := checkUnique(slug, value, existingID)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, core.ValidationError{
				Field:   slug,
				Message: "failed to check uniqueness: " + err.Error(),
				Code:    "unique_check_failed",
			})
			continue
		}

		if exists {
			result.Valid = false
			result.Errors = append(result.Errors, core.ValidationError{
				Field:   slug,
				Message: "value already exists",
				Code:    "unique",
			})
		}
	}

	return result
}

// ApplyDefaults applies default values to entry data
func (v *EntryValidator) ApplyDefaults(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = make(map[string]interface{})
	}

	for slug, field := range v.fields {
		// Skip if value already exists
		if _, exists := data[slug]; exists {
			continue
		}

		// Apply default value if available
		defaultVal, err := field.GetDefaultValue()
		if err == nil && defaultVal != nil {
			data[slug] = defaultVal
		}
	}

	return data
}

// SanitizeData removes hidden and read-only fields from input data
func (v *EntryValidator) SanitizeData(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for slug, value := range data {
		field, exists := v.fields[slug]
		if !exists {
			// Allow unknown fields to pass through
			sanitized[slug] = value
			continue
		}

		// Skip read-only fields on create/update
		if field.ReadOnly {
			continue
		}

		sanitized[slug] = value
	}

	return sanitized
}

// GetFieldsForDisplay returns fields that should be displayed (non-hidden)
func (v *EntryValidator) GetFieldsForDisplay() []*schema.ContentField {
	var visible []*schema.ContentField
	for _, field := range v.contentType.Fields {
		if !field.Hidden {
			visible = append(visible, field)
		}
	}
	return visible
}

// GetRequiredFields returns all required fields
func (v *EntryValidator) GetRequiredFields() []*schema.ContentField {
	var required []*schema.ContentField
	for _, field := range v.contentType.Fields {
		if field.Required {
			required = append(required, field)
		}
	}
	return required
}

// GetSearchableFields returns all fields that can be searched
func (v *EntryValidator) GetSearchableFields() []*schema.ContentField {
	var searchable []*schema.ContentField
	for _, field := range v.contentType.Fields {
		if field.IsSearchable() {
			searchable = append(searchable, field)
		}
	}
	return searchable
}

// GetRelationFields returns all relation fields
func (v *EntryValidator) GetRelationFields() []*schema.ContentField {
	var relations []*schema.ContentField
	for _, field := range v.contentType.Fields {
		if field.IsRelation() {
			relations = append(relations, field)
		}
	}
	return relations
}

// buildOptionsDTO builds a FieldOptionsDTO from schema.FieldOptions
func (v *EntryValidator) buildOptionsDTO(field *schema.ContentField) *core.FieldOptionsDTO {
	opts := &core.FieldOptionsDTO{
		MinLength:        field.Options.MinLength,
		MaxLength:        field.Options.MaxLength,
		Pattern:          field.Options.Pattern,
		Min:              field.Options.Min,
		Max:              field.Options.Max,
		Step:             field.Options.Step,
		Integer:          field.Options.Integer,
		RelatedType:      field.Options.RelatedType,
		RelationType:     field.Options.RelationType,
		OnDelete:         field.Options.OnDelete,
		InverseField:     field.Options.InverseField,
		AllowHTML:        field.Options.AllowHTML,
		MaxWords:         field.Options.MaxWords,
		AllowedMimeTypes: field.Options.AllowedMimeTypes,
		MaxFileSize:      field.Options.MaxFileSize,
		SourceField:      field.Options.SourceField,
		Schema:           field.Options.Schema,
		MinDate:          field.Options.MinDate,
		MaxDate:          field.Options.MaxDate,
		DateFormat:       field.Options.DateFormat,
	}

	// Convert choices
	if len(field.Options.Choices) > 0 {
		opts.Choices = make([]core.ChoiceDTO, len(field.Options.Choices))
		for i, choice := range field.Options.Choices {
			opts.Choices[i] = core.ChoiceDTO{
				Value:    choice.Value,
				Label:    choice.Label,
				Icon:     choice.Icon,
				Color:    choice.Color,
				Disabled: choice.Disabled,
			}
		}
	}

	return opts
}

// ValidationResultToMap converts validation result to error map for API responses
func ValidationResultToMap(result *core.ValidationResult) map[string]string {
	if result.Valid {
		return nil
	}

	errors := make(map[string]string)
	for _, err := range result.Errors {
		// Use first error for each field
		if _, exists := errors[err.Field]; !exists {
			errors[err.Field] = err.Message
		}
	}
	return errors
}

