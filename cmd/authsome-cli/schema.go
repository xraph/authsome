package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xraph/authsome/pkg/schema/definition"
	"github.com/xraph/authsome/pkg/schema/extractor"
)

// schemaCmd represents the schema command
var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Schema management commands",
	Long:  `Commands for managing database schema definitions, extraction, and validation.`,
}

// schemaExtractCmd extracts schema from Go structs
var schemaExtractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extract schema from Go source files",
	Long: `Extract database schema definition from Go struct definitions with Bun tags.
This command reads Go source files containing Bun-tagged structs and generates
a JSON schema definition file that can be used to generate migrations for
different ORMs.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		inputDir, _ := cmd.Flags().GetString("input")
		outputFile, _ := cmd.Flags().GetString("output")

		fmt.Printf("Extracting schema from: %s\n", inputDir)

		// Create extractor
		ext := extractor.NewExtractor(inputDir)

		// Extract schema
		schema, err := ext.Extract()
		if err != nil {
			return fmt.Errorf("failed to extract schema: %w", err)
		}

		// Validate schema
		if err := schema.Validate(); err != nil {
			return fmt.Errorf("schema validation failed: %w", err)
		}

		// Save to file
		if err := schema.SaveToFile(outputFile); err != nil {
			return fmt.Errorf("failed to save schema: %w", err)
		}

		fmt.Printf("✓ Successfully extracted schema\n")
		fmt.Printf("  Models: %d\n", len(schema.Models))
		fmt.Printf("  Output: %s\n", outputFile)

		return nil
	},
}

// schemaValidateCmd validates a schema file
var schemaValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate a schema definition file",
	Long: `Validate a JSON schema definition file for correctness.
Checks for:
- Required fields
- Valid field types
- Valid references
- Index definitions
- Primary keys`,
	RunE: func(cmd *cobra.Command, args []string) error {
		schemaFile, _ := cmd.Flags().GetString("schema")

		fmt.Printf("Validating schema: %s\n", schemaFile)

		// Load schema
		schema, err := definition.LoadFromFile(schemaFile)
		if err != nil {
			return fmt.Errorf("failed to load schema: %w", err)
		}

		// Validate
		if err := schema.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}

		fmt.Printf("✓ Schema is valid\n")
		fmt.Printf("  Version: %s\n", schema.Version)
		fmt.Printf("  Models: %d\n", len(schema.Models))

		return nil
	},
}

// schemaInfoCmd shows information about a schema
var schemaInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show schema information",
	Long:  `Display detailed information about a schema definition file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		schemaFile, _ := cmd.Flags().GetString("schema")

		// Load schema
		schema, err := definition.LoadFromFile(schemaFile)
		if err != nil {
			return fmt.Errorf("failed to load schema: %w", err)
		}

		// Display info
		fmt.Printf("Schema Information\n")
		fmt.Printf("==================\n\n")
		fmt.Printf("Version: %s\n", schema.Version)
		if schema.Description != "" {
			fmt.Printf("Description: %s\n", schema.Description)
		}
		fmt.Printf("\nModels: %d\n\n", len(schema.Models))

		// List models
		for name, model := range schema.Models {
			fmt.Printf("  %s\n", name)
			fmt.Printf("    Table: %s\n", model.Table)
			fmt.Printf("    Fields: %d\n", len(model.Fields))
			fmt.Printf("    Indexes: %d\n", len(model.Indexes))
			fmt.Printf("    Relations: %d\n", len(model.Relations))
			fmt.Printf("\n")
		}

		return nil
	},
}

// schemaDiffCmd compares two schema files
var schemaDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare two schema files",
	Long:  `Compare two schema definition files and show differences.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fromFile, _ := cmd.Flags().GetString("from")
		toFile, _ := cmd.Flags().GetString("to")

		// Load schemas
		fromSchema, err := definition.LoadFromFile(fromFile)
		if err != nil {
			return fmt.Errorf("failed to load 'from' schema: %w", err)
		}

		toSchema, err := definition.LoadFromFile(toFile)
		if err != nil {
			return fmt.Errorf("failed to load 'to' schema: %w", err)
		}

		fmt.Printf("Comparing schemas...\n")
		fmt.Printf("  From: %s (version %s)\n", fromFile, fromSchema.Version)
		fmt.Printf("  To:   %s (version %s)\n\n", toFile, toSchema.Version)

		// Compare models
		added := []string{}
		removed := []string{}
		modified := []string{}

		for name := range toSchema.Models {
			if _, exists := fromSchema.Models[name]; !exists {
				added = append(added, name)
			}
		}

		for name := range fromSchema.Models {
			if _, exists := toSchema.Models[name]; !exists {
				removed = append(removed, name)
			} else {
				// Check if modified (simplified check)
				fromModel := fromSchema.Models[name]
				toModel := toSchema.Models[name]
				if len(fromModel.Fields) != len(toModel.Fields) {
					modified = append(modified, name)
				}
			}
		}

		// Display results
		if len(added) > 0 {
			fmt.Printf("Added models (%d):\n", len(added))
			for _, name := range added {
				fmt.Printf("  + %s\n", name)
			}
			fmt.Println()
		}

		if len(removed) > 0 {
			fmt.Printf("Removed models (%d):\n", len(removed))
			for _, name := range removed {
				fmt.Printf("  - %s\n", name)
			}
			fmt.Println()
		}

		if len(modified) > 0 {
			fmt.Printf("Modified models (%d):\n", len(modified))
			for _, name := range modified {
				fmt.Printf("  ~ %s\n", name)
			}
			fmt.Println()
		}

		if len(added) == 0 && len(removed) == 0 && len(modified) == 0 {
			fmt.Println("No differences found")
		}

		return nil
	},
}

func init() {
	// Add schema command
	rootCmd.AddCommand(schemaCmd)

	// Add subcommands
	schemaCmd.AddCommand(schemaExtractCmd)
	schemaCmd.AddCommand(schemaValidateCmd)
	schemaCmd.AddCommand(schemaInfoCmd)
	schemaCmd.AddCommand(schemaDiffCmd)

	// Extract command flags
	schemaExtractCmd.Flags().String("input", "./schema", "Input directory containing Go source files")
	schemaExtractCmd.Flags().String("output", "./authsome-schema.json", "Output JSON file")

	// Validate command flags
	schemaValidateCmd.Flags().String("schema", "./authsome-schema.json", "Schema file to validate")

	// Info command flags
	schemaInfoCmd.Flags().String("schema", "./authsome-schema.json", "Schema file to display")

	// Diff command flags
	schemaDiffCmd.Flags().String("from", "", "Original schema file")
	schemaDiffCmd.Flags().String("to", "", "New schema file")
	schemaDiffCmd.MarkFlagRequired("from")
	schemaDiffCmd.MarkFlagRequired("to")
}
