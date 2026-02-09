package gorm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/pkg/schema/definition"
	"github.com/xraph/authsome/pkg/schema/generator"
)

// Generator generates GORM migration files.
type Generator struct{}

// NewGenerator creates a new GORM generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// Name returns the generator name.
func (g *Generator) Name() string {
	return "gorm"
}

// Description returns the generator description.
func (g *Generator) Description() string {
	return "GORM migration generator (Go)"
}

// Generate generates GORM migration files.
func (g *Generator) Generate(schema *definition.Schema, opts generator.Options) error {
	// Create output directory
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate models file
	modelsCode := g.generateModels(schema, opts)
	modelsPath := filepath.Join(opts.OutputDir, "models.go")

	if err := os.WriteFile(modelsPath, []byte(modelsCode), 0644); err != nil {
		return fmt.Errorf("failed to write models: %w", err)
	}

	// Generate migration file
	migrationCode := g.generateMigration(schema, opts)
	migrationPath := filepath.Join(opts.OutputDir, "migrate.go")

	if err := os.WriteFile(migrationPath, []byte(migrationCode), 0644); err != nil {
		return fmt.Errorf("failed to write migration: %w", err)
	}

	if opts.Verbose {

	}

	return nil
}

// generateModels generates GORM model structs.
func (g *Generator) generateModels(schema *definition.Schema, opts generator.Options) string {
	var b strings.Builder

	pkgName := opts.PackageName
	if pkgName == "" {
		pkgName = "models"
	}

	b.WriteString(fmt.Sprintf("package %s\n\n", pkgName))
	b.WriteString("import \"time\"\n\n")

	// Generate each model
	for _, model := range schema.Models {
		b.WriteString(fmt.Sprintf("// %s represents the %s table\n", model.Name, model.Table))
		b.WriteString(fmt.Sprintf("type %s struct {\n", model.Name))

		for _, field := range model.Fields {
			gormTag := g.buildGORMTag(field)
			goType := g.mapToGoType(field)

			b.WriteString(fmt.Sprintf("\t%s %s `gorm:\"%s\" json:\"%s\"`\n",
				field.Name, goType, gormTag, toJSONName(field.Name)))
		}

		b.WriteString("}\n\n")
		b.WriteString(fmt.Sprintf("// TableName specifies the table name for %s\n", model.Name))
		b.WriteString(fmt.Sprintf("func (%s) TableName() string {\n", model.Name))
		b.WriteString(fmt.Sprintf("\treturn \"%s\"\n", model.Table))
		b.WriteString("}\n\n")
	}

	return b.String()
}

// generateMigration generates the GORM AutoMigrate code.
func (g *Generator) generateMigration(schema *definition.Schema, opts generator.Options) string {
	var b strings.Builder

	pkgName := opts.PackageName
	if pkgName == "" {
		pkgName = "models"
	}

	b.WriteString(fmt.Sprintf("package %s\n\n", pkgName))
	b.WriteString("import \"gorm.io/gorm\"\n\n")
	b.WriteString("// AutoMigrate runs automatic migrations for all models\n")
	b.WriteString("func AutoMigrate(db *gorm.DB) error {\n")
	b.WriteString("\treturn db.AutoMigrate(\n")

	for _, model := range schema.Models {
		b.WriteString(fmt.Sprintf("\t\t&%s{},\n", model.Name))
	}

	b.WriteString("\t)\n")
	b.WriteString("}\n")

	return b.String()
}

// buildGORMTag builds a GORM struct tag.
func (g *Generator) buildGORMTag(field definition.Field) string {
	parts := []string{"column:" + field.Column}

	if field.Primary {
		parts = append(parts, "primaryKey")
	}

	// Type specification
	if field.Type == definition.FieldTypeString && field.Length > 0 {
		parts = append(parts, fmt.Sprintf("type:varchar(%d)", field.Length))
	}

	if field.Required && !field.Primary {
		parts = append(parts, "not null")
	}

	if field.Unique && !field.Primary {
		parts = append(parts, "uniqueIndex")
	}

	if field.Default != nil {
		defaultVal := fmt.Sprintf("%v", field.Default)
		if defaultVal == "current_timestamp" {
			parts = append(parts, "autoCreateTime")
		} else if field.Type == definition.FieldTypeBoolean {
			parts = append(parts, "default:"+defaultVal)
		} else if field.Type != definition.FieldTypeString {
			parts = append(parts, "default:"+defaultVal)
		}
	}

	return strings.Join(parts, ";")
}

// mapToGoType maps a field type to a Go type.
func (g *Generator) mapToGoType(field definition.Field) string {
	var goType string

	switch field.Type {
	case definition.FieldTypeString, definition.FieldTypeText:
		goType = "string"
	case definition.FieldTypeInteger:
		goType = "int"
	case definition.FieldTypeBigInt:
		goType = "int64"
	case definition.FieldTypeFloat:
		goType = "float64"
	case definition.FieldTypeBoolean:
		goType = "bool"
	case definition.FieldTypeTimestamp, definition.FieldTypeDate:
		goType = "time.Time"
	default:
		goType = "string"
	}

	if field.Nullable {
		return "*" + goType
	}

	return goType
}

func toJSONName(name string) string {
	if len(name) == 0 {
		return name
	}

	return strings.ToLower(name[:1]) + name[1:]
}
