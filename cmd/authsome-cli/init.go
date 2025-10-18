package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xraph/authsome/pkg/schema"
	"github.com/xraph/authsome/pkg/schema/generator"
	"github.com/xraph/authsome/pkg/schema/generator/bun"
	"github.com/xraph/authsome/pkg/schema/generator/diesel"
	"github.com/xraph/authsome/pkg/schema/generator/gorm"
	"github.com/xraph/authsome/pkg/schema/generator/prisma"
	"github.com/xraph/authsome/pkg/schema/generator/sql"
)

// initCmd initializes a new AuthSome project
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new AuthSome project",
	Long: `Initialize a new AuthSome project with the specified ORM.

This command:
- Creates authsome.yaml configuration file
- Extracts authsome-schema.json from embedded schema
- Generates migrations for your chosen ORM
- Sets up project structure

Examples:
  # Initialize with Bun (default)
  authsome init --orm=bun

  # Initialize with GORM
  authsome init --orm=gorm

  # Initialize with Prisma
  authsome init --orm=prisma

  # Initialize with SQL migrations (PostgreSQL)
  authsome init --orm=sql --dialect=postgres
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		orm, _ := cmd.Flags().GetString("orm")
		dialect, _ := cmd.Flags().GetString("dialect")
		outputDir, _ := cmd.Flags().GetString("output")
		configMode, _ := cmd.Flags().GetString("mode")

		fmt.Printf("🚀 Initializing AuthSome project with %s...\n\n", orm)

		// 1. Create project directory structure
		if err := createDirectories(outputDir); err != nil {
			return fmt.Errorf("failed to create directories: %w", err)
		}

		// 2. Write embedded schema to file
		schemaPath := filepath.Join(outputDir, "authsome-schema.json")
		embeddedSchema, err := schema.GetEmbeddedSchema()
		if err != nil {
			return fmt.Errorf("failed to get embedded schema: %w", err)
		}

		if err := embeddedSchema.SaveToFile(schemaPath); err != nil {
			return fmt.Errorf("failed to save schema: %w", err)
		}
		fmt.Printf("✓ Created authsome-schema.json (v%s)\n", embeddedSchema.Version)

		// 3. Generate sample configuration
		configPath := filepath.Join(outputDir, "authsome.yaml")
		var configContent string
		if configMode == "saas" {
			configContent = generateSaaSConfig()
		} else {
			configContent = generateStandaloneConfig()
		}

		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}
		fmt.Printf("✓ Created authsome.yaml (%s mode)\n", configMode)

		// 4. Generate migrations
		migrationsDir := filepath.Join(outputDir, "migrations")
		opts := generator.Options{
			OutputDir:   migrationsDir,
			Dialect:     dialect,
			PackageName: "migrations",
			Overwrite:   false,
			Verbose:     false,
		}

		var gen generator.Generator
		switch orm {
		case "sql":
			if dialect == "" {
				dialect = "postgres" // default
			}
			gen, err = sql.NewGenerator(dialect)
		case "bun":
			gen = bun.NewGenerator()
		case "gorm":
			gen = gorm.NewGenerator()
		case "prisma":
			gen = prisma.NewGenerator()
			migrationsDir = filepath.Join(outputDir, "prisma")
			opts.OutputDir = migrationsDir
		case "diesel":
			gen = diesel.NewGenerator()
		default:
			return fmt.Errorf("unsupported ORM: %s", orm)
		}

		if err != nil {
			return fmt.Errorf("failed to create generator: %w", err)
		}

		if err := gen.Generate(embeddedSchema, opts); err != nil {
			return fmt.Errorf("failed to generate migrations: %w", err)
		}
		fmt.Printf("✓ Generated %s migrations\n", orm)

		// 5. Create .authsome metadata directory
		metadataDir := filepath.Join(outputDir, ".authsome")
		if err := os.MkdirAll(metadataDir, 0755); err != nil {
			return fmt.Errorf("failed to create metadata directory: %w", err)
		}

		// Write metadata
		metadata := fmt.Sprintf("orm: %s\nversion: %s\ndialect: %s\n",
			orm, embeddedSchema.Version, dialect)
		metadataPath := filepath.Join(metadataDir, "config")
		if err := os.WriteFile(metadataPath, []byte(metadata), 0644); err != nil {
			return fmt.Errorf("failed to write metadata: %w", err)
		}

		// 6. Show next steps
		fmt.Printf("\n✅ AuthSome project initialized successfully!\n\n")
		fmt.Printf("Project structure:\n")
		fmt.Printf("  %s/\n", outputDir)
		fmt.Printf("    authsome.yaml          - Configuration\n")
		fmt.Printf("    authsome-schema.json   - Database schema\n")
		if orm == "prisma" {
			fmt.Printf("    prisma/                - Prisma schema\n")
		} else {
			fmt.Printf("    migrations/            - Database migrations\n")
		}
		fmt.Printf("    .authsome/             - Metadata\n")
		fmt.Printf("\n")

		// ORM-specific next steps
		fmt.Printf("Next steps:\n")
		switch orm {
		case "sql":
			fmt.Printf("  1. Review migrations in %s/\n", migrationsDir)
			fmt.Printf("  2. Apply migrations: psql -d yourdb < migrations/001_initial_up.sql\n")
			fmt.Printf("  3. Update authsome.yaml with your database connection\n")
		case "bun":
			fmt.Printf("  1. Review migrations in %s/\n", migrationsDir)
			fmt.Printf("  2. Import migrations in your Go app\n")
			fmt.Printf("  3. Run: migrator := migrate.NewMigrator(db); migrator.Run(ctx)\n")
		case "gorm":
			fmt.Printf("  1. Review models in %s/models.go\n", migrationsDir)
			fmt.Printf("  2. Import and run: migrations.AutoMigrate(db)\n")
		case "prisma":
			fmt.Printf("  1. cd %s\n", migrationsDir)
			fmt.Printf("  2. npx prisma migrate dev --name initial\n")
			fmt.Printf("  3. npx prisma generate\n")
		case "diesel":
			fmt.Printf("  1. diesel migration run\n")
			fmt.Printf("  2. diesel print-schema > src/schema.rs\n")
		}

		fmt.Printf("  4. Start building with AuthSome!\n")

		return nil
	},
}

func createDirectories(base string) error {
	dirs := []string{
		base,
		filepath.Join(base, "migrations"),
		filepath.Join(base, ".authsome"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().String("orm", "bun", "Target ORM (sql, bun, gorm, prisma, diesel)")
	initCmd.Flags().String("dialect", "postgres", "SQL dialect (postgres, mysql, sqlite) - for SQL ORM")
	initCmd.Flags().String("output", ".", "Output directory for project files")
	initCmd.Flags().String("mode", "standalone", "Configuration mode (standalone, saas)")
}
