package bun

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/pkg/schema/definition"
	"github.com/xraph/authsome/pkg/schema/generator"
)

// Generator generates Bun migration files
type Generator struct{}

// NewGenerator creates a new Bun generator
func NewGenerator() *Generator {
	return &Generator{}
}

// Name returns the generator name
func (g *Generator) Name() string {
	return "bun"
}

// Description returns the generator description
func (g *Generator) Description() string {
	return "Bun ORM migration generator (Go)"
}

// Generate generates Bun migration files
func (g *Generator) Generate(schema *definition.Schema, opts generator.Options) error {
	// Create output directory
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate migration file
	migrationCode := g.generateMigration(schema, opts)
	migrationPath := filepath.Join(opts.OutputDir, "001_initial.go")

	if err := os.WriteFile(migrationPath, []byte(migrationCode), 0644); err != nil {
		return fmt.Errorf("failed to write migration: %w", err)
	}

	if opts.Verbose {

	}

	return nil
}

// generateMigration generates the Bun migration Go code
func (g *Generator) generateMigration(schema *definition.Schema, opts generator.Options) string {
	var b strings.Builder

	pkgName := opts.PackageName
	if pkgName == "" {
		pkgName = "migrations"
	}

	// Package and imports
	b.WriteString(fmt.Sprintf("package %s\n\n", pkgName))
	b.WriteString(`import (
	"context"

	"github.com/uptrace/bun"
	"github.com/xraph/authsome/schema"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
`)

	// Create tables
	for _, model := range schema.Models {
		b.WriteString(fmt.Sprintf("\t\t// Create %s table\n", model.Table))
		b.WriteString(fmt.Sprintf("\t\t_, err := db.NewCreateTable().\n"))
		b.WriteString(fmt.Sprintf("\t\t\tModel((*schema.%s)(nil)).\n", model.Name))
		b.WriteString("\t\t\tIfNotExists().\n")
		b.WriteString("\t\t\tExec(ctx)\n")
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString("\t\t\treturn err\n")
		b.WriteString("\t\t}\n\n")
	}

	// Create indexes
	b.WriteString("\t\t// Create indexes\n")
	for _, model := range schema.Models {
		for _, index := range model.Indexes {
			b.WriteString(fmt.Sprintf("\t\t_, err = db.NewCreateIndex().\n"))
			b.WriteString(fmt.Sprintf("\t\t\tModel((*schema.%s)(nil)).\n", model.Name))
			b.WriteString(fmt.Sprintf("\t\t\tIndex(\"%s\").\n", index.Name))

			// Add columns
			for _, col := range index.Columns {
				b.WriteString(fmt.Sprintf("\t\t\tColumn(\"%s\").\n", col))
			}

			if index.Unique {
				b.WriteString("\t\t\tUnique().\n")
			}

			b.WriteString("\t\t\tIfNotExists().\n")
			b.WriteString("\t\t\tExec(ctx)\n")
			b.WriteString("\t\tif err != nil {\n")
			b.WriteString("\t\t\treturn err\n")
			b.WriteString("\t\t}\n\n")
		}
	}

	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}, func(ctx context.Context, db *bun.DB) error {\n")

	// Rollback - drop tables
	b.WriteString("\t\t// Rollback - drop all tables\n")
	b.WriteString("\t\ttables := []string{\n")
	for _, model := range schema.Models {
		b.WriteString(fmt.Sprintf("\t\t\t\"%s\",\n", model.Table))
	}
	b.WriteString("\t\t}\n\n")

	b.WriteString("\t\tfor _, table := range tables {\n")
	b.WriteString("\t\t\t_, err := db.NewDropTable().\n")
	b.WriteString("\t\t\t\tTable(table).\n")
	b.WriteString("\t\t\t\tIfExists().\n")
	b.WriteString("\t\t\t\tExec(ctx)\n")
	b.WriteString("\t\t\tif err != nil {\n")
	b.WriteString("\t\t\t\treturn err\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t}\n\n")

	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t})\n")
	b.WriteString("}\n")

	return b.String()
}
