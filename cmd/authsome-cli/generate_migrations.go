package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xraph/authsome/pkg/schema/definition"
	"github.com/xraph/authsome/pkg/schema/generator"
	"github.com/xraph/authsome/pkg/schema/generator/bun"
	"github.com/xraph/authsome/pkg/schema/generator/diesel"
	"github.com/xraph/authsome/pkg/schema/generator/gorm"
	"github.com/xraph/authsome/pkg/schema/generator/prisma"
	"github.com/xraph/authsome/pkg/schema/generator/sql"
)

// generateMigrationsCmd generates migrations for different ORMs
var generateMigrationsCmd = &cobra.Command{
	Use:   "migrations",
	Short: "Generate database migrations",
	Long: `Generate database migration files for different ORMs and databases.

Supported ORMs:
  - sql: Raw SQL migrations (supports postgres, mysql, sqlite)
  - bun: Bun ORM migrations (Go)
  - gorm: GORM migrations (Go)
  - prisma: Prisma schema (TypeScript)
  - diesel: Diesel migrations (Rust)

Examples:
  # Generate PostgreSQL migrations
  authsome generate migrations --orm=sql --dialect=postgres --output=./migrations/sql

  # Generate MySQL migrations
  authsome generate migrations --orm=sql --dialect=mysql --output=./migrations/sql

  # Generate SQLite migrations
  authsome generate migrations --orm=sql --dialect=sqlite --output=./migrations/sql
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		schemaFile, _ := cmd.Flags().GetString("schema")
		orm, _ := cmd.Flags().GetString("orm")
		dialect, _ := cmd.Flags().GetString("dialect")
		outputDir, _ := cmd.Flags().GetString("output")
		overwrite, _ := cmd.Flags().GetBool("overwrite")

		fmt.Printf("Generating %s migrations...\n", orm)

		// Load schema
		schema, err := definition.LoadFromFile(schemaFile)
		if err != nil {
			return fmt.Errorf("failed to load schema: %w", err)
		}

		// Validate schema
		if err := schema.Validate(); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}

		// Create generator options
		opts := generator.Options{
			OutputDir: outputDir,
			Dialect:   dialect,
			Overwrite: overwrite,
			Verbose:   verbose,
		}

		// Select generator based on ORM
		var gen generator.Generator

		switch orm {
		case "sql":
			if dialect == "" {
				return fmt.Errorf("--dialect is required for SQL generator (postgres, mysql, sqlite)")
			}
			gen, err = sql.NewGenerator(dialect)
			if err != nil {
				return fmt.Errorf("failed to create SQL generator: %w", err)
			}

		case "bun":
			gen = bun.NewGenerator()

		case "gorm":
			gen = gorm.NewGenerator()

		case "prisma":
			gen = prisma.NewGenerator()

		case "diesel":
			gen = diesel.NewGenerator()

		default:
			return fmt.Errorf("unsupported ORM: %s (supported: sql, bun, gorm, prisma, diesel)", orm)
		}

		// Generate migrations
		if err := gen.Generate(schema, opts); err != nil {
			return fmt.Errorf("failed to generate migrations: %w", err)
		}

		fmt.Printf("✓ Successfully generated %s migrations\n", orm)
		fmt.Printf("  Output directory: %s\n", outputDir)

		return nil
	},
}

func init() {
	// Add to generate command
	generateCmd.AddCommand(generateMigrationsCmd)

	// Flags
	generateMigrationsCmd.Flags().String("schema", "./authsome-schema.json", "Schema definition file")
	generateMigrationsCmd.Flags().String("orm", "sql", "Target ORM (sql, bun, gorm, prisma, diesel)")
	generateMigrationsCmd.Flags().String("dialect", "postgres", "SQL dialect (postgres, mysql, sqlite) - required for sql ORM")
	generateMigrationsCmd.Flags().String("output", "./migrations", "Output directory for generated migrations")
	generateMigrationsCmd.Flags().Bool("overwrite", false, "Overwrite existing files")
}
