package dbschema

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func TestValidateSchemaName(t *testing.T) {
	tests := []struct {
		name      string
		schema    string
		wantError bool
	}{
		{
			name:      "empty schema is valid",
			schema:    "",
			wantError: false,
		},
		{
			name:      "simple lowercase schema",
			schema:    "auth",
			wantError: false,
		},
		{
			name:      "schema with underscore",
			schema:    "auth_system",
			wantError: false,
		},
		{
			name:      "schema with uppercase",
			schema:    "AuthSystem",
			wantError: false,
		},
		{
			name:      "schema starting with underscore",
			schema:    "_auth",
			wantError: false,
		},
		{
			name:      "schema with numbers",
			schema:    "auth2",
			wantError: false,
		},
		{
			name:      "invalid: starts with number",
			schema:    "2auth",
			wantError: true,
		},
		{
			name:      "invalid: contains hyphen",
			schema:    "auth-system",
			wantError: true,
		},
		{
			name:      "invalid: contains space",
			schema:    "auth system",
			wantError: true,
		},
		{
			name:      "invalid: contains special characters",
			schema:    "auth@system",
			wantError: true,
		},
		{
			name:      "invalid: too long",
			schema:    "this_is_a_very_long_schema_name_that_exceeds_the_postgresql_limit_of_sixtythree_characters",
			wantError: true,
		},
		{
			name:      "invalid: SQL injection attempt",
			schema:    "auth; DROP TABLE users; --",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSchemaName(tt.schema)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple identifier",
			input:    "auth",
			expected: `"auth"`,
		},
		{
			name:     "identifier with underscore",
			input:    "auth_system",
			expected: `"auth_system"`,
		},
		{
			name:     "identifier with uppercase",
			input:    "AuthSystem",
			expected: `"AuthSystem"`,
		},
		{
			name:     "identifier with quote",
			input:    `auth"system`,
			expected: `"auth""system"`, // Escaped quote
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetTableName(t *testing.T) {
	tests := []struct {
		name       string
		schemaName string
		tableName  string
		expected   string
	}{
		{
			name:       "with schema",
			schemaName: "auth",
			tableName:  "users",
			expected:   `"auth"."users"`,
		},
		{
			name:       "without schema",
			schemaName: "",
			tableName:  "users",
			expected:   "users",
		},
		{
			name:       "complex table name",
			schemaName: "auth",
			tableName:  "user_roles",
			expected:   `"auth"."user_roles"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTableName(tt.schemaName, tt.tableName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration tests - these require a real PostgreSQL database
// Skip these in CI unless DB is available

func getTestDB(t *testing.T) *bun.DB {
	t.Helper()

	// Try to connect to test database
	dsn := "postgres://postgres:postgres@localhost:5432/authsome_test?sslmode=disable"
	
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Test connection
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	return db
}

func TestApplySchema_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	schemaName := "test_auth_schema"

	// Clean up before test
	_, _ = db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+quoteIdentifier(schemaName)+" CASCADE")

	// Test applying schema
	err := ApplySchema(ctx, db, schemaName)
	require.NoError(t, err)

	// Verify schema was created
	var exists bool
	err = db.NewRaw("SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname = ?)", schemaName).
		Scan(ctx, &exists)
	require.NoError(t, err)
	assert.True(t, exists, "schema should exist")

	// Verify search_path was set
	var searchPath string
	err = db.NewRaw("SHOW search_path").Scan(ctx, &searchPath)
	require.NoError(t, err)
	assert.Contains(t, searchPath, schemaName, "search_path should contain custom schema")

	// Create a test table in the schema
	_, err = db.ExecContext(ctx, "CREATE TABLE test_table (id SERIAL PRIMARY KEY, name TEXT)")
	require.NoError(t, err)

	// Verify table was created in the custom schema
	var tableSchema string
	err = db.NewRaw(`
		SELECT schemaname 
		FROM pg_tables 
		WHERE tablename = 'test_table'
	`).Scan(ctx, &tableSchema)
	require.NoError(t, err)
	assert.Equal(t, schemaName, tableSchema, "table should be in custom schema")

	// Clean up
	_, _ = db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+quoteIdentifier(schemaName)+" CASCADE")
}

func TestApplySchema_InvalidName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test with invalid schema name
	err := ApplySchema(ctx, db, "invalid-schema-name")
	assert.Error(t, err, "should reject invalid schema name")
}

func TestApplySchema_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test with empty schema name (should be no-op)
	err := ApplySchema(ctx, db, "")
	assert.NoError(t, err, "empty schema should be allowed")
}

func TestWrapConnection_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer db.Close()

	ctx := context.Background()
	schemaName := "test_wrap_schema"

	// Clean up before test
	_, _ = db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+quoteIdentifier(schemaName)+" CASCADE")

	// Wrap connection with schema
	wrappedDB, err := WrapConnection(ctx, db, schemaName)
	require.NoError(t, err)
	require.NotNil(t, wrappedDB)

	// Verify schema was created and search_path set
	var searchPath string
	err = wrappedDB.NewRaw("SHOW search_path").Scan(ctx, &searchPath)
	require.NoError(t, err)
	assert.Contains(t, searchPath, schemaName)

	// Clean up
	_, _ = db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+quoteIdentifier(schemaName)+" CASCADE")
}

// Benchmark tests

func BenchmarkValidateSchemaName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ValidateSchemaName("auth_system")
	}
}

func BenchmarkQuoteIdentifier(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = quoteIdentifier("auth_system")
	}
}

func BenchmarkGetTableName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GetTableName("auth", "users")
	}
}

