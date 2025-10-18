package definition

import (
	"encoding/json"
	"fmt"
	"os"
)

// Schema represents the complete database schema
type Schema struct {
	Version     string           `json:"version"`
	Description string           `json:"description,omitempty"`
	Models      map[string]Model `json:"models"`
}

// Model represents a database table
type Model struct {
	Name        string     `json:"name"`
	Table       string     `json:"table"`
	Description string     `json:"description,omitempty"`
	Fields      []Field    `json:"fields"`
	Indexes     []Index    `json:"indexes,omitempty"`
	Relations   []Relation `json:"relations,omitempty"`
}

// Field represents a table column
type Field struct {
	Name        string      `json:"name"`
	Column      string      `json:"column"`
	Type        FieldType   `json:"type"`
	Description string      `json:"description,omitempty"`
	Primary     bool        `json:"primary,omitempty"`
	Unique      bool        `json:"unique,omitempty"`
	Required    bool        `json:"required,omitempty"`
	Nullable    bool        `json:"nullable,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Length      int         `json:"length,omitempty"`
	Precision   int         `json:"precision,omitempty"`
	Scale       int         `json:"scale,omitempty"`
	AutoGen     bool        `json:"autoGen,omitempty"` // Auto-generated (e.g., timestamps)
	References  *Reference  `json:"references,omitempty"`
}

// FieldType represents supported field types
type FieldType string

const (
	FieldTypeString    FieldType = "string"
	FieldTypeText      FieldType = "text"
	FieldTypeInteger   FieldType = "integer"
	FieldTypeBigInt    FieldType = "bigint"
	FieldTypeFloat     FieldType = "float"
	FieldTypeDecimal   FieldType = "decimal"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeTimestamp FieldType = "timestamp"
	FieldTypeDate      FieldType = "date"
	FieldTypeTime      FieldType = "time"
	FieldTypeUUID      FieldType = "uuid"
	FieldTypeJSON      FieldType = "json"
	FieldTypeJSONB     FieldType = "jsonb"
	FieldTypeBinary    FieldType = "binary"
	FieldTypeEnum      FieldType = "enum"
)

// Reference represents a foreign key reference
type Reference struct {
	Model    string `json:"model"`
	Field    string `json:"field"`
	OnDelete string `json:"onDelete,omitempty"` // CASCADE, SET NULL, RESTRICT, etc.
	OnUpdate string `json:"onUpdate,omitempty"`
}

// Index represents a database index
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique,omitempty"`
	Type    string   `json:"type,omitempty"` // btree, hash, gin, etc.
}

// Relation represents relationships between models
type Relation struct {
	Name       string       `json:"name"`
	Type       RelationType `json:"type"`
	Model      string       `json:"model"`
	ForeignKey string       `json:"foreignKey,omitempty"`
	References string       `json:"references,omitempty"`
	Through    string       `json:"through,omitempty"` // For many-to-many
}

// RelationType represents relationship types
type RelationType string

const (
	RelationBelongsTo  RelationType = "belongsTo"
	RelationHasOne     RelationType = "hasOne"
	RelationHasMany    RelationType = "hasMany"
	RelationManyToMany RelationType = "manyToMany"
)

// LoadFromFile loads a schema from a JSON file
func LoadFromFile(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	return &schema, nil
}

// SaveToFile saves a schema to a JSON file
func (s *Schema) SaveToFile(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write schema file: %w", err)
	}

	return nil
}

// GetModel returns a model by name
func (s *Schema) GetModel(name string) (Model, bool) {
	model, ok := s.Models[name]
	return model, ok
}

// AddModel adds a model to the schema
func (s *Schema) AddModel(model Model) {
	if s.Models == nil {
		s.Models = make(map[string]Model)
	}
	s.Models[model.Name] = model
}
