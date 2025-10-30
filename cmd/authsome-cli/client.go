package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xraph/authsome/internal/clients/generator"
)

func init() {
	generateCmd.AddCommand(generateClientCmd)

	// Flags for generate client command
	generateClientCmd.Flags().StringP("lang", "l", "", "Target language: go, typescript, rust, or all")
	generateClientCmd.Flags().StringP("output", "o", "./clients", "Output directory")
	generateClientCmd.Flags().StringSliceP("plugins", "p", []string{}, "Plugin IDs to include (empty = all)")
	generateClientCmd.Flags().StringP("manifest-dir", "m", "./internal/clients/manifest/data", "Manifest directory")
	generateClientCmd.Flags().StringP("module-name", "", "github.com/xraph/authsome/clients/go", "Go module name (Go only)")
	generateClientCmd.Flags().Bool("validate", false, "Only validate manifests without generating")
	generateClientCmd.Flags().Bool("list", false, "List available plugins")
}

var generateClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Generate client libraries for Go, TypeScript, and Rust",
	Long: `Generate type-safe client libraries from API manifests.

Examples:
  # Generate all clients
  authsome generate client --lang all

  # Generate TypeScript client only
  authsome generate client --lang typescript

  # Generate Go client with custom module name
  authsome generate client --lang go --module-name github.com/myorg/myproject/authclient

  # Generate with specific plugins
  authsome generate client --lang go --plugins core,social,twofa

  # Validate manifests without generating
  authsome generate client --validate

  # List available plugins
  authsome generate client --list

  # Custom output directory
  authsome generate client --lang typescript --output ./frontend/lib/authsome`,
	RunE: runGenerateClient,
}

func runGenerateClient(cmd *cobra.Command, args []string) error {
	lang, _ := cmd.Flags().GetString("lang")
	outputDir, _ := cmd.Flags().GetString("output")
	plugins, _ := cmd.Flags().GetStringSlice("plugins")
	manifestDir, _ := cmd.Flags().GetString("manifest-dir")
	moduleName, _ := cmd.Flags().GetString("module-name")
	validate, _ := cmd.Flags().GetBool("validate")
	list, _ := cmd.Flags().GetBool("list")

	// Make paths absolute
	if !filepath.IsAbs(outputDir) {
		outputDir, _ = filepath.Abs(outputDir)
	}
	if !filepath.IsAbs(manifestDir) {
		manifestDir, _ = filepath.Abs(manifestDir)
	}

	// Create generator
	gen, err := generator.NewGenerator(manifestDir, outputDir, moduleName)
	if err != nil {
		return fmt.Errorf("failed to create generator: %w", err)
	}

	// List plugins if requested
	if list {
		plugins := gen.ListPlugins()
		fmt.Println("Available plugins:")
		for _, plugin := range plugins {
			fmt.Printf("  - %s\n", plugin)
		}
		return nil
	}

	// Validate manifests if requested
	if validate {
		if err := gen.ValidateManifests(); err != nil {
			return fmt.Errorf("manifest validation failed: %w", err)
		}
		fmt.Println("✓ All manifests are valid")
		return nil
	}

	// Require language flag
	if lang == "" {
		return fmt.Errorf("--lang flag is required (go, typescript, rust, or all)")
	}

	// Generate clients
	lang = strings.ToLower(lang)

	if lang == "all" {
		fmt.Println("Generating clients for all languages...")

		languages := []string{"go", "typescript", "rust"}
		for _, l := range languages {
			fmt.Printf("\nGenerating %s client...\n", l)
			if err := gen.Generate(generator.Language(l), plugins); err != nil {
				return fmt.Errorf("failed to generate %s client: %w", l, err)
			}
			fmt.Printf("✓ %s client generated at %s\n", l, filepath.Join(outputDir, l))
		}

		fmt.Println("\n✓ All clients generated successfully!")
		return nil
	}

	// Validate language
	validLanguages := map[string]generator.Language{
		"go":         generator.LanguageGo,
		"typescript": generator.LanguageTypeScript,
		"ts":         generator.LanguageTypeScript,
		"rust":       generator.LanguageRust,
		"rs":         generator.LanguageRust,
	}

	targetLang, ok := validLanguages[lang]
	if !ok {
		return fmt.Errorf("invalid language: %s (supported: go, typescript, rust)", lang)
	}

	// Generate client
	fmt.Printf("Generating %s client...\n", lang)

	if len(plugins) > 0 {
		fmt.Printf("Including plugins: %s\n", strings.Join(plugins, ", "))
	} else {
		fmt.Println("Including all available plugins")
	}

	if err := gen.Generate(targetLang, plugins); err != nil {
		return fmt.Errorf("failed to generate client: %w", err)
	}

	outputPath := filepath.Join(outputDir, string(targetLang))
	fmt.Printf("\n✓ Client generated successfully at %s\n", outputPath)

	// Print next steps based on language
	printNextSteps(targetLang, outputPath)

	return nil
}

func printNextSteps(lang generator.Language, outputPath string) {
	fmt.Println("\nNext steps:")

	switch lang {
	case generator.LanguageTypeScript:
		fmt.Println("  1. cd " + outputPath)
		fmt.Println("  2. npm install")
		fmt.Println("  3. npm run build")
		fmt.Println("\nUsage example:")
		fmt.Println("  import { AuthsomeClient } from '@authsome/client';")
		fmt.Println("  const client = new AuthsomeClient({ baseURL: 'https://api.example.com' });")

	case generator.LanguageGo:
		fmt.Println("  1. cd " + outputPath)
		fmt.Println("  2. go mod tidy")
		fmt.Println("\nUsage example:")
		fmt.Println("  import \"github.com/xraph/authsome-client\"")
		fmt.Println("  client := authsome.NewClient(\"https://api.example.com\")")

	case generator.LanguageRust:
		fmt.Println("  1. cd " + outputPath)
		fmt.Println("  2. cargo build")
		fmt.Println("\nUsage example:")
		fmt.Println("  use authsome_client::AuthsomeClient;")
		fmt.Println("  let client = AuthsomeClient::builder()")
		fmt.Println("      .base_url(\"https://api.example.com\")")
		fmt.Println("      .build()?;")
	}
}
