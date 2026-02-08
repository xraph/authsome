package sql

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/pkg/schema/definition"
	"github.com/xraph/authsome/pkg/schema/generator"
)

// Generator generates SQL migration files
type Generator struct {
	dialect Dialect
}

// Dialect represents a SQL dialect
type Dialect interface {
	Name() string
	MapType(fieldType definition.FieldType, length int, precision int, scale int) string
	QuoteIdentifier(name string) string
	AutoIncrement() string
	BooleanType() string
	DefaultValue(value interface{}, fieldType definition.FieldType) string
}

// NewGenerator creates a new SQL generator
func NewGenerator(dialectName string) (*Generator, error) {
	var dialect Dialect

	switch strings.ToLower(dialectName) {
	case "postgres", "postgresql":
		dialect = &PostgreSQLDialect{}
	case "mysql":
		dialect = &MySQLDialect{}
	case "sqlite", "sqlite3":
		dialect = &SQLiteDialect{}
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialectName)
	}

	return &Generator{dialect: dialect}, nil
}

// Name returns the generator name
func (g *Generator) Name() string {
	return "sql"
}

// Description returns the generator description
func (g *Generator) Description() string {
	return fmt.Sprintf("SQL migration generator (%s dialect)", g.dialect.Name())
}

// Generate generates SQL migration files
func (g *Generator) Generate(schema *definition.Schema, opts generator.Options) error {
	// Create output directory
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate up migration
	upSQL := g.generateUp(schema)
	upPath := filepath.Join(opts.OutputDir, "001_initial_up.sql")
	if err := os.WriteFile(upPath, []byte(upSQL), 0644); err != nil {
		return fmt.Errorf("failed to write up migration: %w", err)
	}

	// Generate down migration
	downSQL := g.generateDown(schema)
	downPath := filepath.Join(opts.OutputDir, "001_initial_down.sql")
	if err := os.WriteFile(downPath, []byte(downSQL), 0644); err != nil {
		return fmt.Errorf("failed to write down migration: %w", err)
	}

	if opts.Verbose {

	}

	return nil
}

// generateUp generates the up migration SQL
func (g *Generator) generateUp(schema *definition.Schema) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("-- AuthSome Database Schema v%s\n", schema.Version))
	b.WriteString(fmt.Sprintf("-- Generated for %s\n\n", g.dialect.Name()))

	// Generate CREATE TABLE statements
	for _, model := range schema.Models {
		b.WriteString(g.generateCreateTable(model))
		b.WriteString("\n\n")
	}

	// Generate indexes (separate from table creation for better control)
	for _, model := range schema.Models {
		for _, index := range model.Indexes {
			b.WriteString(g.generateCreateIndex(model, index))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// generateDown generates the down migration SQL
func (g *Generator) generateDown(schema *definition.Schema) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("-- Rollback AuthSome Database Schema v%s\n\n", schema.Version))

	// Drop tables in reverse order (to handle foreign keys)
	models := make([]definition.Model, 0, len(schema.Models))
	for _, model := range schema.Models {
		models = append(models, model)
	}

	// Simple reverse (in production, would need topological sort for FK dependencies)
	for i := len(models) - 1; i >= 0; i-- {
		model := models[i]
		b.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", g.dialect.QuoteIdentifier(model.Table)))
	}

	return b.String()
}

// generateCreateTable generates a CREATE TABLE statement
func (g *Generator) generateCreateTable(model definition.Model) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", g.dialect.QuoteIdentifier(model.Table)))

	// Generate columns
	columns := []string{}
	for _, field := range model.Fields {
		columns = append(columns, g.generateColumn(field))
	}

	// Add primary key constraint if composite
	primaryKeys := []string{}
	for _, field := range model.Fields {
		if field.Primary {
			primaryKeys = append(primaryKeys, g.dialect.QuoteIdentifier(field.Column))
		}
	}

	// Add foreign key constraints
	for _, field := range model.Fields {
		if field.References != nil {
			refModel := field.References.Model
			refField := field.References.Field

			fkConstraint := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s(%s)",
				g.dialect.QuoteIdentifier(field.Column),
				g.dialect.QuoteIdentifier(toTableName(refModel)),
				g.dialect.QuoteIdentifier(toColumnName(refField)))

			if field.References.OnDelete != "" {
				fkConstraint += fmt.Sprintf(" ON DELETE %s", field.References.OnDelete)
			}
			if field.References.OnUpdate != "" {
				fkConstraint += fmt.Sprintf(" ON UPDATE %s", field.References.OnUpdate)
			}

			columns = append(columns, fkConstraint)
		}
	}

	b.WriteString("  " + strings.Join(columns, ",\n  "))
	b.WriteString("\n);")

	return b.String()
}

// generateColumn generates a column definition
func (g *Generator) generateColumn(field definition.Field) string {
	parts := []string{
		g.dialect.QuoteIdentifier(field.Column),
		g.dialect.MapType(field.Type, field.Length, field.Precision, field.Scale),
	}

	// Primary key
	if field.Primary {
		parts = append(parts, "PRIMARY KEY")
	}

	// NOT NULL / NULL
	if field.Required && !field.Primary {
		parts = append(parts, "NOT NULL")
	} else if field.Nullable && !field.Primary {
		parts = append(parts, "NULL")
	}

	// UNIQUE
	if field.Unique && !field.Primary {
		parts = append(parts, "UNIQUE")
	}

	// DEFAULT
	if field.Default != nil {
		defaultVal := g.dialect.DefaultValue(field.Default, field.Type)
		if defaultVal != "" {
			parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultVal))
		}
	}

	return strings.Join(parts, " ")
}

// generateCreateIndex generates a CREATE INDEX statement
func (g *Generator) generateCreateIndex(model definition.Model, index definition.Index) string {
	uniqueStr := ""
	if index.Unique {
		uniqueStr = "UNIQUE "
	}

	columns := []string{}
	for _, col := range index.Columns {
		columns = append(columns, g.dialect.QuoteIdentifier(col))
	}

	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);",
		uniqueStr,
		g.dialect.QuoteIdentifier(index.Name),
		g.dialect.QuoteIdentifier(model.Table),
		strings.Join(columns, ", "))
}

// Helper functions

func toTableName(modelName string) string {
	// Simple conversion - in production would use more sophisticated logic
	snake := toSnakeCase(modelName)
	if !strings.HasSuffix(snake, "s") {
		snake += "s"
	}
	return snake
}

func toColumnName(fieldName string) string {
	return toSnakeCase(fieldName)
}

func toSnakeCase(s string) string {
	var result strings.Builder
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(c + 32)
		} else {
			result.WriteRune(c)
		}
	}
	return result.String()
}
