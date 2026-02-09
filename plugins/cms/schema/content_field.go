package schema

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
)

// FieldOptions holds type-specific options for a content field.
type FieldOptions struct {
	// Text fields
	MinLength int    `json:"minLength,omitempty"`
	MaxLength int    `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`

	// Number fields
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Step    *float64 `json:"step,omitempty"`
	Integer bool     `json:"integer,omitempty"`

	// Select fields
	Choices []Choice `json:"choices,omitempty"`

	// Relation fields
	RelatedType  string `json:"relatedType,omitempty"`
	RelationType string `json:"relationType,omitempty"`
	OnDelete     string `json:"onDelete,omitempty"`
	InverseField string `json:"inverseField,omitempty"`

	// Rich text fields
	AllowHTML bool `json:"allowHtml,omitempty"`
	MaxWords  int  `json:"maxWords,omitempty"`

	// Media fields
	AllowedMimeTypes []string `json:"allowedMimeTypes,omitempty"`
	MaxFileSize      int64    `json:"maxFileSize,omitempty"`

	// Slug fields
	SourceField string `json:"sourceField,omitempty"`

	// JSON fields
	Schema string `json:"schema,omitempty"`

	// Date fields
	MinDate    *time.Time `json:"minDate,omitempty"`
	MaxDate    *time.Time `json:"maxDate,omitempty"`
	DateFormat string     `json:"dateFormat,omitempty"`

	// Enumeration fields
	EnumValues []string `json:"enumValues,omitempty"`

	// Decimal fields
	Precision int `json:"precision,omitempty"`
	Scale     int `json:"scale,omitempty"`

	// Object/Array fields (nested structures)
	NestedFields    []NestedFieldDef `json:"nestedFields,omitempty"`    // Inline sub-field definitions
	ComponentRef    string           `json:"componentRef,omitempty"`    // Reference to ComponentSchema slug
	MinItems        *int             `json:"minItems,omitempty"`        // For array: minimum items
	MaxItems        *int             `json:"maxItems,omitempty"`        // For array: maximum items
	Collapsible     bool             `json:"collapsible,omitempty"`     // UI: collapsible in form
	DefaultExpanded bool             `json:"defaultExpanded,omitempty"` // UI: expanded by default

	// OneOf fields (discriminated union)
	DiscriminatorField         string                       `json:"discriminatorField,omitempty"`         // Field name to watch for schema selection
	Schemas                    map[string]OneOfSchemaOption `json:"schemas,omitempty"`                    // Value -> schema mapping
	ClearOnDiscriminatorChange bool                         `json:"clearOnDiscriminatorChange,omitempty"` // Clear data when discriminator changes

	// Conditional visibility
	ShowWhen        *FieldCondition `json:"showWhen,omitempty"`        // Show field when condition is met
	HideWhen        *FieldCondition `json:"hideWhen,omitempty"`        // Hide field when condition is met
	ClearWhenHidden bool            `json:"clearWhenHidden,omitempty"` // Clear value when hidden
}

// OneOfSchemaOption defines a schema option for oneOf fields.
type OneOfSchemaOption struct {
	ComponentRef string           `json:"componentRef,omitempty"` // Reference to ComponentSchema slug
	NestedFields []NestedFieldDef `json:"nestedFields,omitempty"` // Or inline field definitions
	Label        string           `json:"label,omitempty"`        // Display label for this option
}

// FieldCondition defines a condition for showing/hiding fields.
type FieldCondition struct {
	Field    string `json:"field"`           // Field name to watch
	Operator string `json:"operator"`        // eq, ne, in, notIn, exists, notExists
	Value    any    `json:"value,omitempty"` // Value(s) to compare
}

// Choice represents a choice option for select fields.
type Choice struct {
	Value    string `json:"value"`
	Label    string `json:"label"`
	Icon     string `json:"icon,omitempty"`
	Color    string `json:"color,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

// Value implements the driver.Valuer interface for database storage.
func (o FieldOptions) Value() (driver.Value, error) {
	b, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}

	return string(b), nil
}

// Scan implements the sql.Scanner interface for database retrieval.
func (o *FieldOptions) Scan(value any) error {
	if value == nil {
		*o = FieldOptions{}

		return nil
	}

	var bytes []byte

	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return nil
	}

	return json.Unmarshal(bytes, o)
}

// ContentField represents a field definition within a content type.
type ContentField struct {
	bun.BaseModel `bun:"table:cms_content_fields,alias:cf"`

	ID            xid.ID       `bun:"id,pk,type:varchar(20)"                       json:"id"`
	ContentTypeID xid.ID       `bun:"content_type_id,notnull,type:varchar(20)"     json:"contentTypeId"`
	Title         string       `bun:"title,notnull"                                json:"title"`
	Name          string       `bun:"name,notnull"                                 json:"name"`
	Description   string       `bun:"description,nullzero"                         json:"description"`
	Type          string       `bun:"type,notnull"                                 json:"type"`
	Required      bool         `bun:"required,notnull,default:false"               json:"required"`
	Unique        bool         `bun:"unique,notnull,default:false"                 json:"unique"`
	Indexed       bool         `bun:"indexed,notnull,default:false"                json:"indexed"`
	Localized     bool         `bun:"localized,notnull,default:false"              json:"localized"`
	DefaultValue  string       `bun:"default_value,nullzero"                       json:"-"` // JSON-encoded default value
	Options       FieldOptions `bun:"options,type:jsonb,notnull"                   json:"options"`
	Order         int          `bun:"\"order\",notnull,default:0"                  json:"order"`
	Hidden        bool         `bun:"hidden,notnull,default:false"                 json:"hidden"`
	ReadOnly      bool         `bun:"read_only,notnull,default:false"              json:"readOnly"`
	CreatedAt     time.Time    `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time    `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`

	// Relations
	ContentType *ContentType `bun:"rel:belongs-to,join:content_type_id=id" json:"contentType,omitempty"`
}

// TableName returns the table name for ContentField.
func (cf *ContentField) TableName() string {
	return "cms_content_fields"
}

// BeforeInsert sets default values before insert.
func (cf *ContentField) BeforeInsert() {
	if cf.ID.IsNil() {
		cf.ID = xid.New()
	}

	now := time.Now()
	cf.CreatedAt = now
	cf.UpdatedAt = now
}

// BeforeUpdate updates the UpdatedAt timestamp.
func (cf *ContentField) BeforeUpdate() {
	cf.UpdatedAt = time.Now()
}

// GetDefaultValue returns the parsed default value.
func (cf *ContentField) GetDefaultValue() (any, error) {
	if cf.DefaultValue == "" {
		return nil, nil
	}

	var value any
	if err := json.Unmarshal([]byte(cf.DefaultValue), &value); err != nil {
		return nil, err
	}

	return value, nil
}

// SetDefaultValue sets the default value as JSON.
func (cf *ContentField) SetDefaultValue(value any) error {
	if value == nil {
		cf.DefaultValue = ""

		return nil
	}

	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	cf.DefaultValue = string(bytes)

	return nil
}

// IsText returns true if the field is a text-based field.
func (cf *ContentField) IsText() bool {
	switch cf.Type {
	case "text", "richText", "textarea", "markdown", "email", "url", "slug", "phone", "color":
		return true
	default:
		return false
	}
}

// IsNumeric returns true if the field is numeric.
func (cf *ContentField) IsNumeric() bool {
	switch cf.Type {
	case "number", "integer", "float", "bigInteger", "decimal":
		return true
	default:
		return false
	}
}

// IsDate returns true if the field is date-related.
func (cf *ContentField) IsDate() bool {
	switch cf.Type {
	case "date", "datetime", "time":
		return true
	default:
		return false
	}
}

// IsSelectable returns true if the field has selectable options.
func (cf *ContentField) IsSelectable() bool {
	switch cf.Type {
	case "select", "multiSelect", "enumeration":
		return true
	default:
		return false
	}
}

// IsRelation returns true if the field is a relation.
func (cf *ContentField) IsRelation() bool {
	return cf.Type == "relation"
}

// IsSearchable returns true if the field can be included in full-text search.
func (cf *ContentField) IsSearchable() bool {
	switch cf.Type {
	case "text", "richText", "textarea", "markdown":
		return true
	default:
		return false
	}
}

// IsMultiValue returns true if the field can have multiple values.
func (cf *ContentField) IsMultiValue() bool {
	switch cf.Type {
	case "multiSelect":
		return true
	case "relation":
		return cf.Options.RelationType == "oneToMany" || cf.Options.RelationType == "manyToMany"
	default:
		return false
	}
}

// GetRelationType returns the relation type for relation fields.
func (cf *ContentField) GetRelationType() string {
	if !cf.IsRelation() {
		return ""
	}

	return cf.Options.RelationType
}

// GetRelatedType returns the related content type slug.
func (cf *ContentField) GetRelatedType() string {
	if !cf.IsRelation() {
		return ""
	}

	return cf.Options.RelatedType
}

// GetOnDeleteAction returns the on-delete action for relation fields.
func (cf *ContentField) GetOnDeleteAction() string {
	if !cf.IsRelation() || cf.Options.OnDelete == "" {
		return "setNull"
	}

	return cf.Options.OnDelete
}

// GetChoices returns the choices for selectable fields.
func (cf *ContentField) GetChoices() []Choice {
	if !cf.IsSelectable() {
		return nil
	}

	return cf.Options.Choices
}

// HasPattern returns true if the field has a validation pattern.
func (cf *ContentField) HasPattern() bool {
	return cf.Options.Pattern != ""
}

// HasLengthConstraint returns true if the field has length constraints.
func (cf *ContentField) HasLengthConstraint() bool {
	return cf.Options.MinLength > 0 || cf.Options.MaxLength > 0
}

// HasNumericConstraint returns true if the field has numeric constraints.
func (cf *ContentField) HasNumericConstraint() bool {
	return cf.Options.Min != nil || cf.Options.Max != nil
}

// IsObject returns true if the field is an object type.
func (cf *ContentField) IsObject() bool {
	return cf.Type == "object"
}

// IsArray returns true if the field is an array type.
func (cf *ContentField) IsArray() bool {
	return cf.Type == "array"
}

// IsNested returns true if the field is an object, array, or oneOf type.
func (cf *ContentField) IsNested() bool {
	return cf.Type == "object" || cf.Type == "array" || cf.Type == "oneOf"
}

// IsOneOf returns true if the field is a oneOf type.
func (cf *ContentField) IsOneOf() bool {
	return cf.Type == "oneOf"
}

// HasNestedFields returns true if the field has inline nested field definitions.
func (cf *ContentField) HasNestedFields() bool {
	return len(cf.Options.NestedFields) > 0
}

// HasComponentRef returns true if the field references a component schema.
func (cf *ContentField) HasComponentRef() bool {
	return cf.Options.ComponentRef != ""
}

// GetNestedFields returns the nested field definitions.
func (cf *ContentField) GetNestedFields() []NestedFieldDef {
	return cf.Options.NestedFields
}

// GetComponentRef returns the component schema reference slug.
func (cf *ContentField) GetComponentRef() string {
	return cf.Options.ComponentRef
}

// GetMinItems returns the minimum array items constraint.
func (cf *ContentField) GetMinItems() int {
	if cf.Options.MinItems != nil {
		return *cf.Options.MinItems
	}

	return 0
}

// GetMaxItems returns the maximum array items constraint (-1 for no limit).
func (cf *ContentField) GetMaxItems() int {
	if cf.Options.MaxItems != nil {
		return *cf.Options.MaxItems
	}

	return -1
}

// IsCollapsible returns true if the nested field should be collapsible in UI.
func (cf *ContentField) IsCollapsible() bool {
	return cf.Options.Collapsible
}

// IsDefaultExpanded returns true if the nested field should be expanded by default.
func (cf *ContentField) IsDefaultExpanded() bool {
	return cf.Options.DefaultExpanded
}

// HasDiscriminatorField returns true if the field has a discriminator field configured.
func (cf *ContentField) HasDiscriminatorField() bool {
	return cf.Options.DiscriminatorField != ""
}

// GetDiscriminatorField returns the discriminator field name.
func (cf *ContentField) GetDiscriminatorField() string {
	return cf.Options.DiscriminatorField
}

// GetSchemas returns the oneOf schema options.
func (cf *ContentField) GetSchemas() map[string]OneOfSchemaOption {
	return cf.Options.Schemas
}

// GetSchemaForValue returns the schema option for a discriminator value.
func (cf *ContentField) GetSchemaForValue(value string) *OneOfSchemaOption {
	if cf.Options.Schemas == nil {
		return nil
	}

	if schema, ok := cf.Options.Schemas[value]; ok {
		return &schema
	}

	return nil
}

// ShouldClearOnDiscriminatorChange returns true if data should be cleared when discriminator changes.
func (cf *ContentField) ShouldClearOnDiscriminatorChange() bool {
	return cf.Options.ClearOnDiscriminatorChange
}

// HasShowCondition returns true if the field has a show condition.
func (cf *ContentField) HasShowCondition() bool {
	return cf.Options.ShowWhen != nil
}

// HasHideCondition returns true if the field has a hide condition.
func (cf *ContentField) HasHideCondition() bool {
	return cf.Options.HideWhen != nil
}

// GetShowCondition returns the show condition.
func (cf *ContentField) GetShowCondition() *FieldCondition {
	return cf.Options.ShowWhen
}

// GetHideCondition returns the hide condition.
func (cf *ContentField) GetHideCondition() *FieldCondition {
	return cf.Options.HideWhen
}

// ShouldClearWhenHidden returns true if data should be cleared when field is hidden.
func (cf *ContentField) ShouldClearWhenHidden() bool {
	return cf.Options.ClearWhenHidden
}

// HasConditionalVisibility returns true if the field has any conditional visibility rules.
func (cf *ContentField) HasConditionalVisibility() bool {
	return cf.Options.ShowWhen != nil || cf.Options.HideWhen != nil
}
