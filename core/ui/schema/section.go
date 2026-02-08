package schema

import (
	"context"
	"encoding/json"
	"sort"
)

// Section represents a logical grouping of fields
type Section struct {
	// ID is the unique identifier for this section
	ID string `json:"id"`
	// Title is the display title for the section
	Title string `json:"title"`
	// Description provides additional context
	Description string `json:"description,omitempty"`
	// Icon is an optional icon identifier (e.g., lucide icon name)
	Icon string `json:"icon,omitempty"`
	// Order determines the display order
	Order int `json:"order,omitempty"`
	// Fields are the form fields in this section
	Fields []*Field `json:"fields"`
	// Collapsible indicates if the section can be collapsed
	Collapsible bool `json:"collapsible,omitempty"`
	// DefaultCollapsed indicates if the section is collapsed by default
	DefaultCollapsed bool `json:"defaultCollapsed,omitempty"`
	// Permissions are the permissions required to view/edit this section
	Permissions []string `json:"permissions,omitempty"`
	// Metadata contains additional section-level configuration
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// ReadOnly indicates if the entire section is read-only
	ReadOnly bool `json:"readOnly,omitempty"`
	// HelpURL is an optional link to documentation
	HelpURL string `json:"helpUrl,omitempty"`
}

// NewSection creates a new section with the given ID and title
func NewSection(id, title string) *Section {
	return &Section{
		ID:       id,
		Title:    title,
		Fields:   make([]*Field, 0),
		Metadata: make(map[string]interface{}),
	}
}

// AddField adds a field to the section
func (s *Section) AddField(field *Field) *Section {
	s.Fields = append(s.Fields, field)
	return s
}

// AddFields adds multiple fields to the section
func (s *Section) AddFields(fields ...*Field) *Section {
	s.Fields = append(s.Fields, fields...)
	return s
}

// GetField returns a field by ID
func (s *Section) GetField(fieldID string) *Field {
	for _, field := range s.Fields {
		if field.ID == fieldID {
			return field
		}
	}
	return nil
}

// GetSortedFields returns fields sorted by order
func (s *Section) GetSortedFields() []*Field {
	sorted := make([]*Field, len(s.Fields))
	copy(sorted, s.Fields)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Order < sorted[j].Order
	})
	return sorted
}

// GetDefaults returns the default values for all fields in the section
func (s *Section) GetDefaults() map[string]interface{} {
	defaults := make(map[string]interface{})
	for _, field := range s.Fields {
		defaults[field.ID] = field.GetDefaultValue()
	}
	return defaults
}

// Validate validates data against the section's field definitions
func (s *Section) Validate(ctx context.Context, data map[string]interface{}) *ValidationResult {
	result := NewValidationResult()

	for _, field := range s.Fields {
		value, exists := data[field.ID]

		// Check conditional visibility/requirements
		isRequired := field.Required
		isVisible := !field.Hidden
		for _, cond := range field.Conditions {
			condValue, _ := data[cond.Field]
			if evaluateCondition(cond, condValue) {
				switch cond.Action {
				case ActionRequire:
					isRequired = true
				case ActionHide:
					isVisible = false
				case ActionShow:
					isVisible = true
				}
			}
		}

		// Skip validation for hidden fields
		if !isVisible {
			continue
		}

		// Check required
		if isRequired && (!exists || isEmpty(value)) {
			result.AddFieldError(field.ID, "required", "This field is required")
			continue
		}

		// Skip further validation if no value
		if !exists || value == nil {
			continue
		}

		// Run sync validators
		for _, v := range field.Validators {
			if err := v.Validate(value); err != nil {
				result.AddFieldError(field.ID, v.Name(), err.Error())
			}
		}

		// Run built-in validations based on field properties
		if err := s.validateFieldProperties(field, value); err != nil {
			result.AddFieldError(field.ID, "field_property", err.Error())
		}
	}

	return result
}

// ValidateAsync runs async validators on the section data
func (s *Section) ValidateAsync(ctx context.Context, data map[string]interface{}) *ValidationResult {
	result := NewValidationResult()

	for _, field := range s.Fields {
		value, exists := data[field.ID]
		if !exists || value == nil {
			continue
		}

		for _, v := range field.AsyncValidators {
			if err := v.ValidateAsync(ctx, field.ID, value); err != nil {
				result.AddFieldError(field.ID, v.Name(), err.Error())
			}
		}
	}

	return result
}

// ValidateFull runs both sync and async validation
func (s *Section) ValidateFull(ctx context.Context, data map[string]interface{}) *ValidationResult {
	result := s.Validate(ctx, data)
	if result.HasErrors() {
		return result
	}
	return s.ValidateAsync(ctx, data)
}

// validateFieldProperties validates against built-in field properties
func (s *Section) validateFieldProperties(field *Field, value interface{}) error {
	switch field.Type {
	case FieldTypeText, FieldTypeTextArea, FieldTypeEmail, FieldTypeURL, FieldTypePassword:
		str, ok := value.(string)
		if !ok {
			return nil
		}
		if field.MinLength != nil && len(str) < *field.MinLength {
			return NewValidationError(field.ID, "min_length", "Must be at least %d characters", *field.MinLength)
		}
		if field.MaxLength != nil && len(str) > *field.MaxLength {
			return NewValidationError(field.ID, "max_length", "Must be at most %d characters", *field.MaxLength)
		}

	case FieldTypeNumber, FieldTypeSlider:
		num, ok := toFloat64(value)
		if !ok {
			return nil
		}
		if field.Min != nil && num < *field.Min {
			return NewValidationError(field.ID, "min_value", "Must be at least %v", *field.Min)
		}
		if field.Max != nil && num > *field.Max {
			return NewValidationError(field.ID, "max_value", "Must be at most %v", *field.Max)
		}

	case FieldTypeSelect:
		if len(field.Options) > 0 {
			valid := false
			for _, opt := range field.Options {
				if opt.Value == value {
					valid = true
					break
				}
			}
			if !valid {
				return NewValidationError(field.ID, "invalid_option", "Invalid option selected")
			}
		}
	}

	return nil
}

// Patch merges patch data into existing data for this section
func (s *Section) Patch(existing, patch map[string]interface{}) (map[string]interface{}, error) {
	if existing == nil {
		existing = make(map[string]interface{})
	}

	result := make(map[string]interface{})
	// Copy existing values
	for k, v := range existing {
		result[k] = v
	}

	// Apply patch values (only for known fields)
	for _, field := range s.Fields {
		if value, ok := patch[field.ID]; ok {
			result[field.ID] = value
		}
	}

	return result, nil
}

// ExtractData extracts only the fields defined in this section from the data
func (s *Section) ExtractData(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, field := range s.Fields {
		if value, ok := data[field.ID]; ok {
			result[field.ID] = value
		}
	}
	return result
}

// Clone creates a deep copy of the section
func (s *Section) Clone() *Section {
	data, _ := json.Marshal(s)
	var cloned Section
	_ = json.Unmarshal(data, &cloned)
	return &cloned
}

// SectionBuilder provides a fluent API for building sections
type SectionBuilder struct {
	section *Section
}

// NewSectionBuilder creates a new section builder
func NewSectionBuilder(id, title string) *SectionBuilder {
	return &SectionBuilder{
		section: NewSection(id, title),
	}
}

// Description sets the section description
func (b *SectionBuilder) Description(desc string) *SectionBuilder {
	b.section.Description = desc
	return b
}

// Icon sets the section icon
func (b *SectionBuilder) Icon(icon string) *SectionBuilder {
	b.section.Icon = icon
	return b
}

// Order sets the display order
func (b *SectionBuilder) Order(order int) *SectionBuilder {
	b.section.Order = order
	return b
}

// Collapsible makes the section collapsible
func (b *SectionBuilder) Collapsible() *SectionBuilder {
	b.section.Collapsible = true
	return b
}

// DefaultCollapsed makes the section collapsed by default
func (b *SectionBuilder) DefaultCollapsed() *SectionBuilder {
	b.section.DefaultCollapsed = true
	b.section.Collapsible = true
	return b
}

// ReadOnly makes the entire section read-only
func (b *SectionBuilder) ReadOnly() *SectionBuilder {
	b.section.ReadOnly = true
	return b
}

// WithPermissions sets the required permissions
func (b *SectionBuilder) WithPermissions(perms ...string) *SectionBuilder {
	b.section.Permissions = perms
	return b
}

// HelpURL sets the help documentation URL
func (b *SectionBuilder) HelpURL(url string) *SectionBuilder {
	b.section.HelpURL = url
	return b
}

// WithMetadata adds metadata to the section
func (b *SectionBuilder) WithMetadata(key string, value interface{}) *SectionBuilder {
	b.section.Metadata[key] = value
	return b
}

// AddField adds a field to the section
func (b *SectionBuilder) AddField(field *Field) *SectionBuilder {
	b.section.AddField(field)
	return b
}

// AddFields adds multiple fields
func (b *SectionBuilder) AddFields(fields ...*Field) *SectionBuilder {
	b.section.AddFields(fields...)
	return b
}

// Build finalizes and returns the section
func (b *SectionBuilder) Build() *Section {
	return b.section
}

// Helper functions

func evaluateCondition(cond Condition, value interface{}) bool {
	switch cond.Operator {
	case ConditionEquals:
		return value == cond.Value
	case ConditionNotEquals:
		return value != cond.Value
	case ConditionEmpty:
		return isEmpty(value)
	case ConditionNotEmpty:
		return !isEmpty(value)
	case ConditionContains:
		if str, ok := value.(string); ok {
			if target, ok := cond.Value.(string); ok {
				return containsString(str, target)
			}
		}
		return false
	case ConditionGreaterThan:
		v, ok1 := toFloat64(value)
		t, ok2 := toFloat64(cond.Value)
		return ok1 && ok2 && v > t
	case ConditionLessThan:
		v, ok1 := toFloat64(value)
		t, ok2 := toFloat64(cond.Value)
		return ok1 && ok2 && v < t
	default:
		return false
	}
}

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr) >= 0))
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
