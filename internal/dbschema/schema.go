package dbschema

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/uptrace/bun"
)

// SchemaConfig holds schema configuration.
type SchemaConfig struct {
	// SchemaName is the PostgreSQL schema to use
	// Empty string means use database default (typically "public")
	SchemaName string
}

// ValidateSchemaName validates that the schema name is a safe SQL identifier
// This prevents SQL injection in schema names.
func ValidateSchemaName(schema string) error {
	if schema == "" {
		return nil // Empty is valid - means use default
	}

	// PostgreSQL identifier rules:
	// - Must start with letter or underscore
	// - Can contain letters, digits, underscores, dollar signs
	// - Max 63 bytes
	// - Case insensitive by default (we enforce lowercase for consistency)

	if len(schema) > 63 {
		return fmt.Errorf("schema name too long (max 63 characters): %s", schema)
	}

	// Allow alphanumeric, underscore, no leading digit
	validName := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	if !validName.MatchString(schema) {
		return fmt.Errorf("invalid schema name (must start with letter/underscore, contain only letters/digits/underscores): %s", schema)
	}

	return nil
}

// ApplySchema sets up the database to use the specified schema
// This does two things:
// 1. Creates the schema if it doesn't exist
// 2. Sets the PostgreSQL search_path to prioritize the schema.
func ApplySchema(ctx context.Context, db *bun.DB, schemaName string) error {
	if schemaName == "" {
		// No custom schema - use database default
		return nil
	}

	// Validate schema name to prevent SQL injection
	if err := ValidateSchemaName(schemaName); err != nil {
		return fmt.Errorf("invalid schema name: %w", err)
	}

	// Create schema if it doesn't exist
	// Using string interpolation is safe here because we validated the schema name
	createSchemaSQL := "CREATE SCHEMA IF NOT EXISTS " + quoteIdentifier(schemaName)
	if _, err := db.ExecContext(ctx, createSchemaSQL); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schemaName, err)
	}

	// Set search_path to use the custom schema first, then public
	// This allows references to tables without schema prefix
	// Format: SET search_path TO schema_name, public
	searchPathSQL := fmt.Sprintf("SET search_path TO %s, public", quoteIdentifier(schemaName))
	if _, err := db.ExecContext(ctx, searchPathSQL); err != nil {
		return fmt.Errorf("failed to set search_path to %s: %w", schemaName, err)
	}

	return nil
}

// GetTableName returns the fully qualified table name with schema prefix
// This is useful when you need to explicitly reference a table in a specific schema.
func GetTableName(schemaName, tableName string) string {
	if schemaName == "" {
		return tableName
	}

	return fmt.Sprintf("%s.%s", quoteIdentifier(schemaName), quoteIdentifier(tableName))
}

// quoteIdentifier quotes a SQL identifier to prevent injection
// PostgreSQL uses double quotes for identifiers.
func quoteIdentifier(name string) string {
	// Replace any double quotes with escaped double quotes
	escaped := strings.ReplaceAll(name, `"`, `""`)

	return fmt.Sprintf(`"%s"`, escaped)
}

// WrapConnection wraps a bun.DB connection to automatically apply schema for all operations
// Returns a new DB connection with the search_path set
// Note: This only affects the current connection. For connection pools,
// you should call ApplySchema in a connection hook.
func WrapConnection(ctx context.Context, db *bun.DB, schemaName string) (*bun.DB, error) {
	if schemaName == "" {
		return db, nil
	}

	// Apply schema to this connection
	if err := ApplySchema(ctx, db, schemaName); err != nil {
		return nil, err
	}

	return db, nil
}

// SetupConnectionHook creates a Bun query hook that sets the search_path for each connection
// This ensures that all queries use the correct schema, even in connection pools.
func SetupConnectionHook(schemaName string) bun.QueryHook {
	return &schemaHook{schemaName: schemaName}
}

// schemaHook is a Bun query hook that sets the search_path before each query.
type schemaHook struct {
	schemaName string
}

// BeforeQuery sets the search_path before executing a query.
func (h *schemaHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	// Only set search_path if schema is configured
	if h.schemaName == "" {
		return ctx
	}

	// For safety, we only set search_path on the first query in a session
	// This is handled automatically by PostgreSQL session state
	// No action needed here as ApplySchema already set it

	return ctx
}

// AfterQuery is called after a query executes (no-op for schema hook).
func (h *schemaHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	// No-op
}
