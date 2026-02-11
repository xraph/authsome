package schema

import (
	"encoding/json"
)

// SelectOption represents an option for select/multiselect fields.
type SelectOption struct {
	// Value is the actual value stored
	Value any `json:"value"`
	// Label is the display text
	Label string `json:"label"`
	// Description is optional additional context
	Description string `json:"description,omitempty"`
	// Disabled indicates if this option is disabled
	Disabled bool `json:"disabled,omitempty"`
	// Icon is an optional icon identifier
	Icon string `json:"icon,omitempty"`
	// Group is used to group options in the dropdown
	Group string `json:"group,omitempty"`
}

// ConditionOperator defines the comparison operator for conditions.
type ConditionOperator string

const (
	// ConditionEquals checks if the field value equals the target.
	ConditionEquals ConditionOperator = "eq"
	// ConditionNotEquals checks if the field value does not equal the target.
	ConditionNotEquals ConditionOperator = "neq"
	// ConditionContains checks if the field value contains the target.
	ConditionContains ConditionOperator = "contains"
	// ConditionGreaterThan checks if the field value is greater than the target.
	ConditionGreaterThan ConditionOperator = "gt"
	// ConditionLessThan checks if the field value is less than the target.
	ConditionLessThan ConditionOperator = "lt"
	// ConditionEmpty checks if the field value is empty.
	ConditionEmpty ConditionOperator = "empty"
	// ConditionNotEmpty checks if the field value is not empty.
	ConditionNotEmpty ConditionOperator = "not_empty"
)

// Condition represents a conditional visibility/requirement rule.
type Condition struct {
	// Field is the ID of the field to check
	Field string `json:"field"`
	// Operator is the comparison operator
	Operator ConditionOperator `json:"operator"`
	// Value is the target value for comparison
	Value any `json:"value,omitempty"`
	// Action specifies what happens when condition is met
	Action ConditionAction `json:"action"`
}

// ConditionAction specifies the action to take when a condition is met.
type ConditionAction string

const (
	// ActionShow shows the field when condition is met.
	ActionShow ConditionAction = "show"
	// ActionHide hides the field when condition is met.
	ActionHide ConditionAction = "hide"
	// ActionRequire makes the field required when condition is met.
	ActionRequire ConditionAction = "require"
	// ActionDisable disables the field when condition is met.
	ActionDisable ConditionAction = "disable"
	// ActionEnable enables the field when condition is met.
	ActionEnable ConditionAction = "enable"
)

// Field represents a single form field with its configuration.
type Field struct {
	// ID is the unique identifier for this field within the section
	ID string `json:"id"`
	// Type is the field type
	Type FieldType `json:"type"`
	// Label is the display label for the field
	Label string `json:"label"`
	// Description provides additional context for the field
	Description string `json:"description,omitempty"`
	// Placeholder is the placeholder text for input fields
	Placeholder string `json:"placeholder,omitempty"`
	// DefaultValue is the initial value for the field
	DefaultValue any `json:"defaultValue,omitempty"`
	// Required indicates if the field is required
	Required bool `json:"required,omitempty"`
	// Disabled indicates if the field is disabled (read-only)
	Disabled bool `json:"disabled,omitempty"`
	// Hidden indicates if the field is hidden from view
	Hidden bool `json:"hidden,omitempty"`
	// ReadOnly indicates if the field is read-only but visible
	ReadOnly bool `json:"readOnly,omitempty"`
	// Validators are the synchronous validators for this field
	Validators []Validator `json:"-"`
	// ValidatorConfigs are serializable validator configurations
	ValidatorConfigs []ValidatorConfig `json:"validators,omitempty"`
	// AsyncValidators are async validators (uniqueness checks, etc.)
	AsyncValidators []AsyncValidator `json:"-"`
	// AsyncValidatorConfigs are serializable async validator configurations
	AsyncValidatorConfigs []AsyncValidatorConfig `json:"asyncValidators,omitempty"`
	// Options are the available options for select/multiselect fields
	Options []SelectOption `json:"options,omitempty"`
	// Conditions define conditional visibility/requirements
	Conditions []Condition `json:"conditions,omitempty"`
	// Metadata contains additional field-level configuration
	Metadata map[string]any `json:"metadata,omitempty"`
	// Order determines the display order within the section
	Order int `json:"order,omitempty"`
	// Width specifies the field width (e.g., "full", "half", "third")
	Width string `json:"width,omitempty"`
	// HelpText provides additional help information
	HelpText string `json:"helpText,omitempty"`
	// Prefix is an optional prefix for the input (e.g., "$", "https://")
	Prefix string `json:"prefix,omitempty"`
	// Suffix is an optional suffix for the input (e.g., "px", "%")
	Suffix string `json:"suffix,omitempty"`
	// Min is the minimum value for number/slider fields
	Min *float64 `json:"min,omitempty"`
	// Max is the maximum value for number/slider fields
	Max *float64 `json:"max,omitempty"`
	// Step is the step increment for number/slider fields
	Step *float64 `json:"step,omitempty"`
	// MinLength is the minimum string length
	MinLength *int `json:"minLength,omitempty"`
	// MaxLength is the maximum string length
	MaxLength *int `json:"maxLength,omitempty"`
	// Pattern is a regex pattern for validation
	Pattern string `json:"pattern,omitempty"`
	// PatternMessage is the error message for pattern validation failure
	PatternMessage string `json:"patternMessage,omitempty"`
}

// ValidatorConfig is a serializable validator configuration.
type ValidatorConfig struct {
	Type    string         `json:"type"`
	Params  map[string]any `json:"params,omitempty"`
	Message string         `json:"message,omitempty"`
}

// AsyncValidatorConfig is a serializable async validator configuration.
type AsyncValidatorConfig struct {
	Type    string         `json:"type"`
	Params  map[string]any `json:"params,omitempty"`
	Message string         `json:"message,omitempty"`
}

// Clone creates a deep copy of the field.
func (f *Field) Clone() *Field {
	data, err := json.Marshal(f)
	if err != nil {
		return nil
	}

	var cloned Field

	if err := json.Unmarshal(data, &cloned); err != nil {
		return nil
	}

	return &cloned
}

// FieldBuilder provides a fluent API for building fields.
type FieldBuilder struct {
	field *Field
}

// NewField creates a new field builder.
func NewField(id string, fieldType FieldType) *FieldBuilder {
	return &FieldBuilder{
		field: &Field{
			ID:       id,
			Type:     fieldType,
			Metadata: make(map[string]any),
		},
	}
}

// Label sets the field label.
func (b *FieldBuilder) Label(label string) *FieldBuilder {
	b.field.Label = label

	return b
}

// Description sets the field description.
func (b *FieldBuilder) Description(desc string) *FieldBuilder {
	b.field.Description = desc

	return b
}

// Placeholder sets the placeholder text.
func (b *FieldBuilder) Placeholder(placeholder string) *FieldBuilder {
	b.field.Placeholder = placeholder

	return b
}

// DefaultValue sets the default value.
func (b *FieldBuilder) DefaultValue(value any) *FieldBuilder {
	b.field.DefaultValue = value

	return b
}

// Required marks the field as required.
func (b *FieldBuilder) Required() *FieldBuilder {
	b.field.Required = true

	return b
}

// Disabled marks the field as disabled.
func (b *FieldBuilder) Disabled() *FieldBuilder {
	b.field.Disabled = true

	return b
}

// Hidden marks the field as hidden.
func (b *FieldBuilder) Hidden() *FieldBuilder {
	b.field.Hidden = true

	return b
}

// ReadOnly marks the field as read-only.
func (b *FieldBuilder) ReadOnly() *FieldBuilder {
	b.field.ReadOnly = true

	return b
}

// Order sets the display order.
func (b *FieldBuilder) Order(order int) *FieldBuilder {
	b.field.Order = order

	return b
}

// Width sets the field width.
func (b *FieldBuilder) Width(width string) *FieldBuilder {
	b.field.Width = width

	return b
}

// HelpText sets the help text.
func (b *FieldBuilder) HelpText(text string) *FieldBuilder {
	b.field.HelpText = text

	return b
}

// Prefix sets the input prefix.
func (b *FieldBuilder) Prefix(prefix string) *FieldBuilder {
	b.field.Prefix = prefix

	return b
}

// Suffix sets the input suffix.
func (b *FieldBuilder) Suffix(suffix string) *FieldBuilder {
	b.field.Suffix = suffix

	return b
}

// Min sets the minimum value for number fields.
func (b *FieldBuilder) Min(min float64) *FieldBuilder {
	b.field.Min = &min

	return b
}

// Max sets the maximum value for number fields.
func (b *FieldBuilder) Max(max float64) *FieldBuilder {
	b.field.Max = &max

	return b
}

// Step sets the step value for number fields.
func (b *FieldBuilder) Step(step float64) *FieldBuilder {
	b.field.Step = &step

	return b
}

// MinLength sets the minimum string length.
func (b *FieldBuilder) MinLength(min int) *FieldBuilder {
	b.field.MinLength = &min

	return b
}

// MaxLength sets the maximum string length.
func (b *FieldBuilder) MaxLength(max int) *FieldBuilder {
	b.field.MaxLength = &max

	return b
}

// Pattern sets a regex validation pattern.
func (b *FieldBuilder) Pattern(pattern, message string) *FieldBuilder {
	b.field.Pattern = pattern
	b.field.PatternMessage = message

	return b
}

// Options sets the options for select fields.
func (b *FieldBuilder) Options(options ...SelectOption) *FieldBuilder {
	b.field.Options = options

	return b
}

// StringOptions creates options from string values.
func (b *FieldBuilder) StringOptions(values ...string) *FieldBuilder {
	options := make([]SelectOption, len(values))
	for i, v := range values {
		options[i] = SelectOption{Value: v, Label: v}
	}

	b.field.Options = options

	return b
}

// LabeledOptions creates options with value-label pairs.
func (b *FieldBuilder) LabeledOptions(pairs ...string) *FieldBuilder {
	if len(pairs)%2 != 0 {
		return b
	}

	options := make([]SelectOption, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		options[i/2] = SelectOption{Value: pairs[i], Label: pairs[i+1]}
	}

	b.field.Options = options

	return b
}

// WithValidator adds a validator.
func (b *FieldBuilder) WithValidator(v Validator) *FieldBuilder {
	b.field.Validators = append(b.field.Validators, v)

	return b
}

// WithAsyncValidator adds an async validator.
func (b *FieldBuilder) WithAsyncValidator(v AsyncValidator) *FieldBuilder {
	b.field.AsyncValidators = append(b.field.AsyncValidators, v)

	return b
}

// Condition adds a condition.
func (b *FieldBuilder) Condition(cond Condition) *FieldBuilder {
	b.field.Conditions = append(b.field.Conditions, cond)

	return b
}

// DependsOn adds a condition that shows the field when another field equals a value.
func (b *FieldBuilder) DependsOn(fieldID string, value any) *FieldBuilder {
	b.field.Conditions = append(b.field.Conditions, Condition{
		Field:    fieldID,
		Operator: ConditionEquals,
		Value:    value,
		Action:   ActionShow,
	})

	return b
}

// HideWhen adds a condition that hides the field when another field equals a value.
func (b *FieldBuilder) HideWhen(fieldID string, value any) *FieldBuilder {
	b.field.Conditions = append(b.field.Conditions, Condition{
		Field:    fieldID,
		Operator: ConditionEquals,
		Value:    value,
		Action:   ActionHide,
	})

	return b
}

// RequireWhen adds a condition that requires the field when another field equals a value.
func (b *FieldBuilder) RequireWhen(fieldID string, value any) *FieldBuilder {
	b.field.Conditions = append(b.field.Conditions, Condition{
		Field:    fieldID,
		Operator: ConditionEquals,
		Value:    value,
		Action:   ActionRequire,
	})

	return b
}

// WithMetadata adds metadata to the field.
func (b *FieldBuilder) WithMetadata(key string, value any) *FieldBuilder {
	b.field.Metadata[key] = value

	return b
}

// Build finalizes and returns the field.
func (b *FieldBuilder) Build() *Field {
	// Build validator configs from validators for serialization
	for _, v := range b.field.Validators {
		b.field.ValidatorConfigs = append(b.field.ValidatorConfigs, ValidatorConfig{
			Type:    v.Name(),
			Message: "",
		})
	}

	for _, v := range b.field.AsyncValidators {
		b.field.AsyncValidatorConfigs = append(b.field.AsyncValidatorConfigs, AsyncValidatorConfig{
			Type:    v.Name(),
			Message: "",
		})
	}

	return b.field
}

// GetDefaultValue returns the field's default value, considering the field type.
func (f *Field) GetDefaultValue() any {
	if f.DefaultValue != nil {
		return f.DefaultValue
	}

	// Return type-appropriate defaults
	switch f.Type {
	case FieldTypeBoolean:
		return false
	case FieldTypeNumber, FieldTypeSlider:
		if f.Min != nil {
			return *f.Min
		}

		return 0
	case FieldTypeMultiSelect, FieldTypeTags:
		return []any{}
	default:
		return ""
	}
}
