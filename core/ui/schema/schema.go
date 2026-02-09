// Package schema provides a dynamic UI schema system for building settings forms
// with validation, forgeui component rendering, and plugin extensibility.
package schema

import (
	"encoding/json"
)

// FieldType represents the type of a form field.
type FieldType string

const (
	// FieldTypeText represents a text input field.
	FieldTypeText FieldType = "text"
	// FieldTypeNumber represents a number input field.
	FieldTypeNumber FieldType = "number"
	// FieldTypeBoolean represents a boolean toggle/checkbox field.
	FieldTypeBoolean FieldType = "boolean"
	// FieldTypeSelect represents a single-select dropdown.
	FieldTypeSelect FieldType = "select"
	// FieldTypeMultiSelect represents a multi-select dropdown.
	FieldTypeMultiSelect FieldType = "multiselect"
	// FieldTypePassword represents a password input field.
	FieldTypePassword FieldType = "password"
	// FieldTypeEmail represents an email input field.
	FieldTypeEmail FieldType = "email"
	// FieldTypeURL represents a URL input field.
	FieldTypeURL FieldType = "url"
	// FieldTypeTextArea represents a multi-line text area.
	FieldTypeTextArea FieldType = "textarea"
	// FieldTypeJSON represents a JSON editor field.
	FieldTypeJSON FieldType = "json"
	// FieldTypeDate represents a date picker field.
	FieldTypeDate FieldType = "date"
	// FieldTypeDateTime represents a date-time picker field.
	FieldTypeDateTime FieldType = "datetime"
	// FieldTypeColor represents a color picker field.
	FieldTypeColor FieldType = "color"
	// FieldTypeFile represents a file upload field.
	FieldTypeFile FieldType = "file"
	// FieldTypeSlider represents a slider/range input.
	FieldTypeSlider FieldType = "slider"
	// FieldTypeTags represents a tags input field.
	FieldTypeTags FieldType = "tags"
)

// String returns the string representation of the field type.
func (ft FieldType) String() string {
	return string(ft)
}

// IsValid checks if the field type is a recognized type.
func (ft FieldType) IsValid() bool {
	switch ft {
	case FieldTypeText, FieldTypeNumber, FieldTypeBoolean, FieldTypeSelect,
		FieldTypeMultiSelect, FieldTypePassword, FieldTypeEmail, FieldTypeURL,
		FieldTypeTextArea, FieldTypeJSON, FieldTypeDate, FieldTypeDateTime,
		FieldTypeColor, FieldTypeFile, FieldTypeSlider, FieldTypeTags:
		return true
	default:
		return false
	}
}

// Schema represents a complete settings schema with multiple sections.
type Schema struct {
	// ID is the unique identifier for this schema
	ID string `json:"id"`
	// Name is the display name of the schema
	Name string `json:"name"`
	// Description provides additional context about the schema
	Description string `json:"description,omitempty"`
	// Sections are the logical groupings of fields
	Sections []*Section `json:"sections"`
	// Version is the schema version for migration support
	Version int `json:"version"`
	// Metadata contains additional schema-level configuration
	Metadata map[string]any `json:"metadata,omitempty"`
}

// NewSchema creates a new schema with the given ID and name.
func NewSchema(id, name string) *Schema {
	return &Schema{
		ID:       id,
		Name:     name,
		Sections: make([]*Section, 0),
		Version:  1,
		Metadata: make(map[string]any),
	}
}

// AddSection adds a section to the schema.
func (s *Schema) AddSection(section *Section) *Schema {
	s.Sections = append(s.Sections, section)

	return s
}

// GetSection returns a section by ID.
func (s *Schema) GetSection(sectionID string) *Section {
	for _, section := range s.Sections {
		if section.ID == sectionID {
			return section
		}
	}

	return nil
}

// GetField returns a field by section and field ID.
func (s *Schema) GetField(sectionID, fieldID string) *Field {
	section := s.GetSection(sectionID)
	if section == nil {
		return nil
	}

	return section.GetField(fieldID)
}

// ToJSON serializes the schema to JSON.
func (s *Schema) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// FromJSON deserializes a schema from JSON.
func FromJSON(data []byte) (*Schema, error) {
	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}

	return &schema, nil
}

// GetDefaults returns the default values for all fields in the schema.
func (s *Schema) GetDefaults() map[string]map[string]any {
	defaults := make(map[string]map[string]any)
	for _, section := range s.Sections {
		defaults[section.ID] = section.GetDefaults()
	}

	return defaults
}

// Clone creates a deep copy of the schema.
func (s *Schema) Clone() *Schema {
	data, _ := json.Marshal(s)

	var cloned Schema

	_ = json.Unmarshal(data, &cloned)

	return &cloned
}
