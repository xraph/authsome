package schema

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/rs/xid"
	"github.com/uptrace/bun"
	mainSchema "github.com/xraph/authsome/schema"
)

// NestedFieldDef defines a field within a nested object or component schema.
type NestedFieldDef struct {
	Title       string        `json:"title"`
	Name        string        `json:"name"`
	Type        string        `json:"type"`
	Required    bool          `json:"required,omitempty"`
	Description string        `json:"description,omitempty"`
	Options     *FieldOptions `json:"options,omitempty"` // Recursive for multi-level nesting
}

// NestedFieldDefs is a slice of NestedFieldDef for database storage.
type NestedFieldDefs []NestedFieldDef

// Value implements the driver.Valuer interface for database storage.
func (n NestedFieldDefs) Value() (driver.Value, error) {
	if n == nil {
		return "[]", nil
	}

	b, err := json.Marshal(n)
	if err != nil {
		return nil, err
	}

	return string(b), nil
}

// Scan implements the sql.Scanner interface for database retrieval.
func (n *NestedFieldDefs) Scan(value any) error {
	if value == nil {
		*n = NestedFieldDefs{}

		return nil
	}

	var bytes []byte

	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		*n = NestedFieldDefs{}

		return nil
	}

	return json.Unmarshal(bytes, n)
}

// ComponentSchema represents a reusable schema definition for nested objects.
type ComponentSchema struct {
	bun.BaseModel `bun:"table:cms_component_schemas,alias:cs"`

	ID            xid.ID          `bun:"id,pk,type:varchar(20)"                       json:"id"`
	AppID         xid.ID          `bun:"app_id,notnull,type:varchar(20)"              json:"appId"`
	EnvironmentID xid.ID          `bun:"environment_id,notnull,type:varchar(20)"      json:"environmentId"`
	Title         string          `bun:"title,notnull"                                json:"title"`
	Name          string          `bun:"name,notnull"                                 json:"name"`
	Description   string          `bun:"description,nullzero"                         json:"description"`
	Icon          string          `bun:"icon,nullzero"                                json:"icon"`
	Fields        NestedFieldDefs `bun:"fields,type:jsonb,notnull"                    json:"fields"`
	CreatedBy     xid.ID          `bun:"created_by,type:varchar(20)"                  json:"createdBy"`
	UpdatedBy     xid.ID          `bun:"updated_by,type:varchar(20)"                  json:"updatedBy"`
	CreatedAt     time.Time       `bun:"created_at,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time       `bun:"updated_at,notnull,default:current_timestamp" json:"updatedAt"`
	DeletedAt     *time.Time      `bun:"deleted_at,soft_delete,nullzero"              json:"-"`

	// Relations
	App         *mainSchema.App         `bun:"rel:belongs-to,join:app_id=id"         json:"app,omitempty"`
	Environment *mainSchema.Environment `bun:"rel:belongs-to,join:environment_id=id" json:"environment,omitempty"`
}

// TableName returns the table name for ComponentSchema.
func (cs *ComponentSchema) TableName() string {
	return "cms_component_schemas"
}

// BeforeInsert sets default values before insert.
func (cs *ComponentSchema) BeforeInsert() {
	if cs.ID.IsNil() {
		cs.ID = xid.New()
	}

	now := time.Now()
	cs.CreatedAt = now

	cs.UpdatedAt = now
	if cs.Fields == nil {
		cs.Fields = NestedFieldDefs{}
	}
}

// BeforeUpdate updates the UpdatedAt timestamp.
func (cs *ComponentSchema) BeforeUpdate() {
	cs.UpdatedAt = time.Now()
}

// GetFieldByName returns a nested field by its name.
func (cs *ComponentSchema) GetFieldByName(name string) *NestedFieldDef {
	for i := range cs.Fields {
		if cs.Fields[i].Name == name {
			return &cs.Fields[i]
		}
	}

	return nil
}

// GetRequiredFields returns all required nested fields.
func (cs *ComponentSchema) GetRequiredFields() []NestedFieldDef {
	var required []NestedFieldDef

	for _, f := range cs.Fields {
		if f.Required {
			required = append(required, f)
		}
	}

	return required
}

// HasNestedObjects returns true if any field is an object or array type.
func (cs *ComponentSchema) HasNestedObjects() bool {
	for _, f := range cs.Fields {
		if f.Type == "object" || f.Type == "array" {
			return true
		}
	}

	return false
}

// GetAllFieldNames returns all field names (including nested ones recursively).
func (cs *ComponentSchema) GetAllFieldNames() []string {
	var names []string
	for _, f := range cs.Fields {
		names = append(names, f.Name)
		names = append(names, getNestedNames(f, f.Name)...)
	}

	return names
}

// getNestedNames recursively collects nested field names.
func getNestedNames(field NestedFieldDef, prefix string) []string {
	var names []string

	if field.Options != nil && len(field.Options.NestedFields) > 0 {
		for _, nested := range field.Options.NestedFields {
			fullName := prefix + "." + nested.Name
			names = append(names, fullName)
			names = append(names, getNestedNames(nested, fullName)...)
		}
	}

	return names
}

// ValidateFields validates all nested field definitions.
func (cs *ComponentSchema) ValidateFields() error {
	seen := make(map[string]bool)

	for _, f := range cs.Fields {
		if f.Title == "" {
			return ErrFieldTitleRequired
		}

		if f.Name == "" {
			return ErrFieldNameRequired
		}

		if f.Type == "" {
			return ErrFieldTypeRequired
		}

		if seen[f.Name] {
			return ErrDuplicateFieldName
		}

		seen[f.Name] = true
	}

	return nil
}

// Component schema specific errors.
var (
	ErrFieldTitleRequired = &ComponentSchemaError{Message: "field title is required"}
	ErrFieldNameRequired  = &ComponentSchemaError{Message: "field name is required"}
	ErrFieldTypeRequired  = &ComponentSchemaError{Message: "field type is required"}
	ErrDuplicateFieldName = &ComponentSchemaError{Message: "duplicate field name"}
)

// ComponentSchemaError represents a component schema validation error.
type ComponentSchemaError struct {
	Message string
}

func (e *ComponentSchemaError) Error() string {
	return e.Message
}
