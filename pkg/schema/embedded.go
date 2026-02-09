package schema

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/xraph/authsome/pkg/schema/definition"
)

//go:embed authsome-schema.json
var embeddedSchema []byte

// GetEmbeddedSchema returns the embedded AuthSome schema.
func GetEmbeddedSchema() (*definition.Schema, error) {
	var schema definition.Schema
	if err := json.Unmarshal(embeddedSchema, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedded schema: %w", err)
	}

	return &schema, nil
}

// GetEmbeddedSchemaRaw returns the raw JSON bytes of the embedded schema.
func GetEmbeddedSchemaRaw() []byte {
	return embeddedSchema
}

// SchemaVersion returns the version of the embedded schema.
func SchemaVersion() (string, error) {
	schema, err := GetEmbeddedSchema()
	if err != nil {
		return "", err
	}

	return schema.Version, nil
}
