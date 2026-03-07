package settings

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/xraph/authsome/formconfig"
)

// ValueType describes the JSON-compatible type of a setting value.
type ValueType string

const (
	TypeString ValueType = "string"
	TypeInt    ValueType = "int"
	TypeFloat  ValueType = "float"
	TypeBool   ValueType = "bool"
	TypeObject ValueType = "object"
	TypeArray  ValueType = "array"
)

// UIMetadata describes how a setting should be rendered in a form UI.
// It reuses types from the formconfig package for consistency with
// other authsome UI schema definitions.
type UIMetadata struct {
	// InputType is the form field type (text, number, switch, select, etc.).
	InputType formconfig.FieldType `json:"input_type"`

	// Placeholder is placeholder text for the input field.
	Placeholder string `json:"placeholder,omitempty"`

	// HelpText is descriptive text shown below the input field.
	HelpText string `json:"help_text,omitempty"`

	// Options lists choices for select/radio/checkbox fields.
	Options []formconfig.SelectOption `json:"options,omitempty"`

	// Validation defines client-side validation hints.
	Validation *formconfig.Validation `json:"validation,omitempty"`

	// Order controls display ordering within a category (lower = first).
	Order int `json:"order"`

	// ReadOnly marks the field as non-editable in the UI.
	ReadOnly bool `json:"read_only,omitempty"`

	// Condition defines conditional visibility based on another setting.
	Condition *VisibilityCondition `json:"condition,omitempty"`

	// Section is an optional sub-group label within a category.
	Section string `json:"section,omitempty"`
}

// VisibilityCondition controls when a setting field is visible in the UI.
// The field is shown only when the referenced setting matches the condition.
type VisibilityCondition struct {
	// Key is the setting key to evaluate.
	Key string `json:"key"`

	// Value is the expected value (JSON-encoded).
	Value json.RawMessage `json:"value"`

	// Operator is the comparison operator: "eq" (default), "ne", "in".
	Operator string `json:"operator,omitempty"`
}

// Definition describes a single setting that can be configured.
// Plugins register definitions at init time to declare their configurable knobs.
type Definition struct {
	// Key is the unique dot-separated key, e.g. "password.min_length".
	Key string `json:"key"`

	// DisplayName is the human-readable label for UI surfaces.
	DisplayName string `json:"display_name"`

	// Description explains what this setting controls.
	Description string `json:"description"`

	// Type is the JSON schema type of the value.
	Type ValueType `json:"type"`

	// Default is the JSON-encoded default value (the "code default" in the cascade).
	Default json.RawMessage `json:"default"`

	// Scopes lists which scopes this setting can be overridden at.
	// An empty slice means only global.
	Scopes []Scope `json:"scopes"`

	// Enforceable marks this setting as lockable at higher scopes.
	// When enforced, lower scopes cannot override the value.
	Enforceable bool `json:"enforceable"`

	// Sensitive marks values that should be redacted in API responses.
	Sensitive bool `json:"sensitive"`

	// Validate is an optional validation function that receives the raw JSON
	// value and returns an error if the value is invalid. Called at write time.
	Validate func(value json.RawMessage) error `json:"-"`

	// Category groups related settings in UI displays.
	Category string `json:"category"`

	// Namespace is the owning plugin name (auto-set at registration).
	Namespace string `json:"namespace"`

	// UI holds rendering metadata for auto-generated settings forms.
	// When non-nil, the dashboard can auto-render this setting.
	UI *UIMetadata `json:"ui,omitempty"`
}

// HasScope returns true if the definition allows the given scope.
func (d *Definition) HasScope(scope Scope) bool {
	for _, s := range d.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// DefinitionTyped is a generic wrapper that pairs a Definition with its Go type.
// This enables type-safe reading of resolved values via [Get].
type DefinitionTyped[T any] struct {
	Def Definition
}

// Define creates a typed setting definition. The Go type T determines
// the default value type and is used by the generic [Get] accessor.
//
// If any UI-related DefOption is used (e.g. WithInputType, WithHelpText)
// and no explicit InputType is set, the input type is inferred from T:
//
//	bool   → switch
//	int    → number
//	float  → number
//	string → text
//	[]T    → textarea
func Define[T any](key string, defaultVal T, opts ...DefOption) DefinitionTyped[T] {
	raw, err := json.Marshal(defaultVal)
	if err != nil {
		panic(fmt.Sprintf("settings: marshal default for %q: %v", key, err))
	}

	def := Definition{
		Key:     key,
		Default: raw,
		Type:    inferType[T](),
		Scopes:  []Scope{ScopeGlobal},
	}
	for _, opt := range opts {
		opt(&def)
	}

	// Auto-infer InputType from the Go type when UI metadata exists
	// but no explicit InputType was set.
	if def.UI != nil && def.UI.InputType == "" {
		def.UI.InputType = inferInputType[T]()
	}

	return DefinitionTyped[T]{Def: def}
}

// DefOption configures a [Definition].
type DefOption func(*Definition)

// WithDisplayName sets the human-readable label.
func WithDisplayName(name string) DefOption {
	return func(d *Definition) { d.DisplayName = name }
}

// WithDescription sets the description.
func WithDescription(desc string) DefOption {
	return func(d *Definition) { d.Description = desc }
}

// WithScopes sets which scopes allow override.
func WithScopes(scopes ...Scope) DefOption {
	return func(d *Definition) { d.Scopes = scopes }
}

// WithEnforceable marks the setting as lockable at higher scopes.
func WithEnforceable() DefOption {
	return func(d *Definition) { d.Enforceable = true }
}

// WithSensitive marks the setting value as sensitive (redacted in API).
func WithSensitive() DefOption {
	return func(d *Definition) { d.Sensitive = true }
}

// WithValidation sets a validation function called at write time.
func WithValidation(fn func(json.RawMessage) error) DefOption {
	return func(d *Definition) { d.Validate = fn }
}

// WithCategory sets the UI grouping category.
func WithCategory(cat string) DefOption {
	return func(d *Definition) { d.Category = cat }
}

// ──────────────────────────────────────────────────
// UI metadata DefOptions
// ──────────────────────────────────────────────────

// ensureUI lazily initialises the UI field on a Definition.
func ensureUI(d *Definition) *UIMetadata {
	if d.UI == nil {
		d.UI = &UIMetadata{}
	}
	return d.UI
}

// WithUI sets the entire UIMetadata at once.
func WithUI(ui UIMetadata) DefOption {
	return func(d *Definition) { d.UI = &ui }
}

// WithInputType sets the form field type (text, number, switch, select, …).
func WithInputType(t formconfig.FieldType) DefOption {
	return func(d *Definition) { ensureUI(d).InputType = t }
}

// WithPlaceholder sets placeholder text for the input field.
func WithPlaceholder(p string) DefOption {
	return func(d *Definition) { ensureUI(d).Placeholder = p }
}

// WithHelpText sets descriptive text shown below the input field.
func WithHelpText(h string) DefOption {
	return func(d *Definition) { ensureUI(d).HelpText = h }
}

// WithOptions sets the choices for select/radio/checkbox fields.
func WithOptions(opts ...formconfig.SelectOption) DefOption {
	return func(d *Definition) { ensureUI(d).Options = opts }
}

// WithUIValidation sets client-side validation hints for the field.
func WithUIValidation(v formconfig.Validation) DefOption {
	return func(d *Definition) { ensureUI(d).Validation = &v }
}

// WithOrder sets the display order within a category (lower = first).
func WithOrder(o int) DefOption {
	return func(d *Definition) { ensureUI(d).Order = o }
}

// WithReadOnly marks the setting as non-editable in the UI.
func WithReadOnly() DefOption {
	return func(d *Definition) { ensureUI(d).ReadOnly = true }
}

// WithVisibleWhen sets a conditional visibility rule. The setting field is
// only shown when the referenced key's value matches. Supported operators
// are "eq" (default), "ne", and "in".
func WithVisibleWhen(key string, value any, operator ...string) DefOption {
	return func(d *Definition) {
		raw, err := json.Marshal(value)
		if err != nil {
			panic(fmt.Sprintf("settings: marshal visibility condition value: %v", err))
		}
		op := "eq"
		if len(operator) > 0 && operator[0] != "" {
			op = operator[0]
		}
		ensureUI(d).Condition = &VisibilityCondition{
			Key:      key,
			Value:    raw,
			Operator: op,
		}
	}
}

// WithSection sets a sub-group label within a category.
func WithSection(s string) DefOption {
	return func(d *Definition) { ensureUI(d).Section = s }
}

// inferType maps a Go type to a ValueType via reflection.
func inferType[T any]() ValueType {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return TypeObject
	}
	switch t.Kind() {
	case reflect.String:
		return TypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return TypeInt
	case reflect.Float32, reflect.Float64:
		return TypeFloat
	case reflect.Bool:
		return TypeBool
	case reflect.Slice, reflect.Array:
		return TypeArray
	default:
		return TypeObject
	}
}

// inferInputType maps a Go type to a formconfig.FieldType for UI auto-inference.
func inferInputType[T any]() formconfig.FieldType {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return formconfig.FieldText
	}
	switch t.Kind() {
	case reflect.Bool:
		return formconfig.FieldSwitch
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return formconfig.FieldNumber
	case reflect.String:
		return formconfig.FieldText
	case reflect.Slice, reflect.Array:
		return formconfig.FieldTextarea
	default:
		return formconfig.FieldText
	}
}
