package prisma

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/pkg/schema/definition"
	"github.com/xraph/authsome/pkg/schema/generator"
)

// Generator generates Prisma schema files
type Generator struct{}

// NewGenerator creates a new Prisma generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Name returns the generator name
func (g *Generator) Name() string {
	return "prisma"
}

// Description returns the generator description
func (g *Generator) Description() string {
	return "Prisma schema generator (TypeScript)"
}

// Generate generates Prisma schema file
func (g *Generator) Generate(schema *definition.Schema, opts generator.Options) error {
	// Create output directory
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate schema.prisma
	prismaSchema := g.generateSchema(schema, opts)
	schemaPath := filepath.Join(opts.OutputDir, "schema.prisma")

	if err := os.WriteFile(schemaPath, []byte(prismaSchema), 0644); err != nil {
		return fmt.Errorf("failed to write schema: %w", err)
	}

	if opts.Verbose {

	}

	return nil
}

// generateSchema generates the Prisma schema
func (g *Generator) generateSchema(schema *definition.Schema, opts generator.Options) string {
	var b strings.Builder

	// Datasource configuration
	dialect := opts.Dialect
	if dialect == "" || dialect == "postgres" {
		dialect = "postgresql"
	}

	b.WriteString("// AuthSome Database Schema\n")
	b.WriteString(fmt.Sprintf("// Version: %s\n\n", schema.Version))

	b.WriteString("datasource db {\n")
	b.WriteString(fmt.Sprintf("  provider = \"%s\"\n", dialect))
	b.WriteString("  url      = env(\"DATABASE_URL\")\n")
	b.WriteString("}\n\n")

	b.WriteString("generator client {\n")
	b.WriteString("  provider = \"prisma-client-js\"\n")
	b.WriteString("}\n\n")

	// Generate models
	for _, model := range schema.Models {
		b.WriteString(g.generateModel(model))
		b.WriteString("\n")
	}

	return b.String()
}

// generateModel generates a Prisma model
func (g *Generator) generateModel(model definition.Model) string {
	var b strings.Builder

	if model.Description != "" {
		b.WriteString(fmt.Sprintf("/// %s\n", model.Description))
	}
	b.WriteString(fmt.Sprintf("model %s {\n", model.Name))

	// Generate fields
	for _, field := range model.Fields {
		b.WriteString("  ")
		b.WriteString(g.generateField(field))
		b.WriteString("\n")
	}

	// Generate indexes
	if len(model.Indexes) > 0 {
		b.WriteString("\n")
		for _, index := range model.Indexes {
			if index.Unique {
				b.WriteString(fmt.Sprintf("  @@unique([%s], name: \"%s\")\n",
					strings.Join(index.Columns, ", "), index.Name))
			} else {
				b.WriteString(fmt.Sprintf("  @@index([%s], name: \"%s\")\n",
					strings.Join(index.Columns, ", "), index.Name))
			}
		}
	}

	// Table mapping
	b.WriteString(fmt.Sprintf("\n  @@map(\"%s\")\n", model.Table))
	b.WriteString("}\n")

	return b.String()
}

// generateField generates a Prisma field
func (g *Generator) generateField(field definition.Field) string {
	prismaType := g.mapToPrismaType(field)
	parts := []string{field.Name, prismaType}

	// Attributes
	attrs := []string{}

	if field.Primary {
		attrs = append(attrs, "@id")
	}

	if field.Unique && !field.Primary {
		attrs = append(attrs, "@unique")
	}

	if field.Default != nil {
		defaultVal := fmt.Sprintf("%v", field.Default)
		if defaultVal == "current_timestamp" {
			attrs = append(attrs, "@default(now())")
		} else if field.Type == definition.FieldTypeBoolean {
			attrs = append(attrs, fmt.Sprintf("@default(%s)", defaultVal))
		} else if field.Type == definition.FieldTypeInteger {
			attrs = append(attrs, fmt.Sprintf("@default(%s)", defaultVal))
		} else {
			attrs = append(attrs, fmt.Sprintf("@default(\"%s\")", defaultVal))
		}
	}

	// Column mapping if different from field name
	if field.Column != toSnakeCase(field.Name) {
		attrs = append(attrs, fmt.Sprintf("@map(\"%s\")", field.Column))
	}

	// Type specification for strings
	if field.Type == definition.FieldTypeString && field.Length > 0 {
		attrs = append(attrs, fmt.Sprintf("@db.VarChar(%d)", field.Length))
	}

	if len(attrs) > 0 {
		parts = append(parts, strings.Join(attrs, " "))
	}

	return strings.Join(parts, " ")
}

// mapToPrismaType maps a field type to a Prisma type
func (g *Generator) mapToPrismaType(field definition.Field) string {
	var prismaType string

	switch field.Type {
	case definition.FieldTypeString, definition.FieldTypeText:
		prismaType = "String"
	case definition.FieldTypeInteger, definition.FieldTypeBigInt:
		prismaType = "Int"
	case definition.FieldTypeFloat, definition.FieldTypeDecimal:
		prismaType = "Float"
	case definition.FieldTypeBoolean:
		prismaType = "Boolean"
	case definition.FieldTypeTimestamp, definition.FieldTypeDate:
		prismaType = "DateTime"
	case definition.FieldTypeJSON, definition.FieldTypeJSONB:
		prismaType = "Json"
	default:
		prismaType = "String"
	}

	if field.Nullable {
		prismaType += "?"
	}

	return prismaType
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
