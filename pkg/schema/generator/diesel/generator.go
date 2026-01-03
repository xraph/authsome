package diesel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xraph/authsome/pkg/schema/definition"
	"github.com/xraph/authsome/pkg/schema/generator"
)

// Generator generates Diesel migration files
type Generator struct{}

// NewGenerator creates a new Diesel generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Name returns the generator name
func (g *Generator) Name() string {
	return "diesel"
}

// Description returns the generator description
func (g *Generator) Description() string {
	return "Diesel migration generator (Rust)"
}

// Generate generates Diesel migration files
func (g *Generator) Generate(schema *definition.Schema, opts generator.Options) error {
	// Create migrations directory with timestamp
	timestamp := time.Now().Format("2006-01-02-150405")
	migrationDir := filepath.Join(opts.OutputDir, fmt.Sprintf("%s_authsome_initial", timestamp))

	if err := os.MkdirAll(migrationDir, 0755); err != nil {
		return fmt.Errorf("failed to create migration directory: %w", err)
	}

	// Generate up.sql
	upSQL := g.generateUpSQL(schema)
	upPath := filepath.Join(migrationDir, "up.sql")
	if err := os.WriteFile(upPath, []byte(upSQL), 0644); err != nil {
		return fmt.Errorf("failed to write up.sql: %w", err)
	}

	// Generate down.sql
	downSQL := g.generateDownSQL(schema)
	downPath := filepath.Join(migrationDir, "down.sql")
	if err := os.WriteFile(downPath, []byte(downSQL), 0644); err != nil {
		return fmt.Errorf("failed to write down.sql: %w", err)
	}

	if opts.Verbose {


	}

	return nil
}

// generateUpSQL generates the up migration SQL (PostgreSQL syntax)
func (g *Generator) generateUpSQL(schema *definition.Schema) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("-- AuthSome Database Schema v%s\n", schema.Version))
	b.WriteString("-- Generated for Diesel ORM (PostgreSQL)\n\n")

	// Generate CREATE TABLE statements
	for _, model := range schema.Models {
		b.WriteString(g.generateCreateTable(model))
		b.WriteString("\n\n")
	}

	// Generate indexes
	for _, model := range schema.Models {
		for _, index := range model.Indexes {
			b.WriteString(g.generateCreateIndex(model, index))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// generateDownSQL generates the down migration SQL
func (g *Generator) generateDownSQL(schema *definition.Schema) string {
	var b strings.Builder

	b.WriteString("-- Rollback AuthSome Database Schema\n\n")

	// Drop tables in reverse order
	models := make([]definition.Model, 0, len(schema.Models))
	for _, model := range schema.Models {
		models = append(models, model)
	}

	for i := len(models) - 1; i >= 0; i-- {
		model := models[i]
		b.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", model.Table))
	}

	return b.String()
}

// generateCreateTable generates a CREATE TABLE statement (PostgreSQL syntax)
func (g *Generator) generateCreateTable(model definition.Model) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", model.Table))

	// Generate columns
	columns := []string{}
	for _, field := range model.Fields {
		columns = append(columns, g.generateColumn(field))
	}

	// Add foreign key constraints
	for _, field := range model.Fields {
		if field.References != nil {
			refModel := field.References.Model
			refField := field.References.Field

			fkConstraint := fmt.Sprintf("    FOREIGN KEY (%s) REFERENCES %s(%s)",
				field.Column,
				toTableName(refModel),
				toColumnName(refField))

			if field.References.OnDelete != "" {
				fkConstraint += fmt.Sprintf(" ON DELETE %s", field.References.OnDelete)
			}
			if field.References.OnUpdate != "" {
				fkConstraint += fmt.Sprintf(" ON UPDATE %s", field.References.OnUpdate)
			}

			columns = append(columns, fkConstraint)
		}
	}

	b.WriteString(strings.Join(columns, ",\n"))
	b.WriteString("\n);")

	return b.String()
}

// generateColumn generates a column definition (PostgreSQL syntax)
func (g *Generator) generateColumn(field definition.Field) string {
	parts := []string{
		"    " + field.Column,
		g.mapToPostgreSQLType(field),
	}

	// Primary key
	if field.Primary {
		parts = append(parts, "PRIMARY KEY")
	}

	// NOT NULL
	if field.Required && !field.Primary {
		parts = append(parts, "NOT NULL")
	}

	// UNIQUE
	if field.Unique && !field.Primary {
		parts = append(parts, "UNIQUE")
	}

	// DEFAULT
	if field.Default != nil {
		defaultVal := g.formatDefault(field.Default, field.Type)
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

	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);",
		uniqueStr,
		index.Name,
		model.Table,
		strings.Join(index.Columns, ", "))
}

// mapToPostgreSQLType maps a field type to PostgreSQL type
func (g *Generator) mapToPostgreSQLType(field definition.Field) string {
	switch field.Type {
	case definition.FieldTypeString:
		if field.Length > 0 {
			return fmt.Sprintf("VARCHAR(%d)", field.Length)
		}
		return "VARCHAR(255)"
	case definition.FieldTypeText:
		return "TEXT"
	case definition.FieldTypeInteger:
		return "INTEGER"
	case definition.FieldTypeBigInt:
		return "BIGINT"
	case definition.FieldTypeFloat:
		return "DOUBLE PRECISION"
	case definition.FieldTypeDecimal:
		return "DECIMAL"
	case definition.FieldTypeBoolean:
		return "BOOLEAN"
	case definition.FieldTypeTimestamp:
		return "TIMESTAMP"
	case definition.FieldTypeDate:
		return "DATE"
	case definition.FieldTypeTime:
		return "TIME"
	case definition.FieldTypeUUID:
		return "UUID"
	case definition.FieldTypeJSON:
		return "JSON"
	case definition.FieldTypeJSONB:
		return "JSONB"
	case definition.FieldTypeBinary:
		return "BYTEA"
	default:
		return "TEXT"
	}
}

// formatDefault formats a default value for PostgreSQL
func (g *Generator) formatDefault(value interface{}, fieldType definition.FieldType) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		if v == "current_timestamp" {
			return "CURRENT_TIMESTAMP"
		}
		if fieldType == definition.FieldTypeBoolean {
			if v == "true" {
				return "true"
			}
			if v == "false" {
				return "false"
			}
		}
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("'%v'", v)
	}
}

func toTableName(modelName string) string {
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
