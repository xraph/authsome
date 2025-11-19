package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xraph/authsome/internal/clients/introspector"
	"gopkg.in/yaml.v3"
)

func init() {
	generateCmd.AddCommand(introspectCmd)

	introspectCmd.Flags().StringP("plugin", "p", "", "Plugin ID to introspect (or 'all' for all plugins)")
	introspectCmd.Flags().StringP("output", "o", "./internal/clients/manifest/data", "Output directory for manifests")
	introspectCmd.Flags().Bool("dry-run", false, "Print manifests without writing files")
	introspectCmd.Flags().Bool("core", false, "Introspect core handlers")
}

var introspectCmd = &cobra.Command{
	Use:   "introspect",
	Short: "Auto-generate manifests by introspecting Go code",
	Long: `Introspect Go source code to automatically generate API manifests.

This command analyzes:
  - Handler functions to extract request/response types
  - Route registrations to get HTTP methods and paths
  - Struct definitions to extract type information
  - Plugin metadata from plugin.go files

Examples:
  # Introspect a specific plugin
  authsome generate introspect --plugin social

  # Introspect all plugins
  authsome generate introspect --plugin all

  # Introspect core handlers
  authsome generate introspect --core

  # Dry run (print without writing)
  authsome generate introspect --plugin social --dry-run`,
	RunE: runIntrospect,
}

func runIntrospect(cmd *cobra.Command, args []string) error {
	pluginID, _ := cmd.Flags().GetString("plugin")
	outputDir, _ := cmd.Flags().GetString("output")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	core, _ := cmd.Flags().GetBool("core")

	// Get project root (current directory)
	projectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	intro := introspector.NewIntrospector(projectRoot)

	if core {
		return introspectCore(intro, outputDir, dryRun)
	}

	if pluginID == "" {
		return fmt.Errorf("--plugin flag is required (or use --core for core handlers)")
	}

	if pluginID == "all" {
		return introspectAllPlugins(intro, outputDir, dryRun)
	}

	return introspectPlugin(intro, pluginID, outputDir, dryRun)
}

func introspectPlugin(intro *introspector.Introspector, pluginID, outputDir string, dryRun bool) error {
	fmt.Printf("Introspecting plugin: %s\n", pluginID)

	manifest, err := intro.GenerateManifest(pluginID)
	if err != nil {
		return fmt.Errorf("failed to introspect plugin %s: %w", pluginID, err)
	}

	// Check if manifest has routes - skip if none (middleware/service only plugins)
	if len(manifest.Routes) == 0 {
		fmt.Printf("⚠ Skipped: Plugin has no HTTP routes (middleware/service only)\n")
		return nil
	}

	// Validate manifest
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("generated manifest is invalid: %w", err)
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if dryRun {
		fmt.Printf("\n--- %s.yaml ---\n", pluginID)
		fmt.Println(string(yamlData))
		return nil
	}

	// Write to file
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, pluginID+".yaml")
	if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	fmt.Printf("✓ Generated manifest: %s\n", outputPath)
	fmt.Printf("  - %d routes\n", len(manifest.Routes))
	fmt.Printf("  - %d types\n", len(manifest.Types))

	return nil
}

func introspectCore(intro *introspector.Introspector, outputDir string, dryRun bool) error {
	fmt.Println("Introspecting core handlers...")

	// Introspect handlers directory
	handlersPath := filepath.Join(".", "handlers")
	routeInfo, err := intro.IntrospectHandlers(handlersPath)
	if err != nil {
		return fmt.Errorf("failed to introspect handlers: %w", err)
	}

	// Introspect routes
	routesPath := filepath.Join(".", "routes")
	registrations, err := intro.IntrospectRoutes(routesPath)
	if err != nil {
		return fmt.Errorf("failed to introspect routes: %w", err)
	}

	fmt.Printf("Found %d handler methods\n", len(routeInfo.Routes))
	fmt.Printf("Found %d route registrations\n", len(registrations))
	fmt.Printf("Found %d type definitions\n", len(routeInfo.Types))

	// Match and display
	for _, route := range routeInfo.Routes {
		for _, reg := range registrations {
			if reg.HandlerName == route.Name {
				fmt.Printf("  ✓ %s %s → %s\n", reg.Method, reg.Path, route.Name)
				if route.RequestType != "" {
					fmt.Printf("     Request: %s\n", route.RequestType)
				}
				if route.ResponseType != "" {
					fmt.Printf("     Response: %s\n", route.ResponseType)
				}
			}
		}
	}

	// TODO: Generate core.yaml manifest
	fmt.Println("\nNote: Core manifest generation is not yet fully implemented")
	fmt.Println("      This requires more complex analysis of the handlers package")

	return nil
}

func introspectAllPlugins(intro *introspector.Introspector, outputDir string, dryRun bool) error {
	pluginsDir := filepath.Join(".", "plugins")
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var successCount int
	var failCount int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginID := entry.Name()

		// Skip special directories
		if pluginID == "mcp" || pluginID == "dashboard" {
			continue
		}

		// Handle enterprise as a composite plugin with sub-plugins
		if pluginID == "enterprise" {
			fmt.Printf("\n")
			success, fail := introspectEnterprisePlugins(intro, outputDir, dryRun)
			successCount += success
			failCount += fail
			continue
		}

		fmt.Printf("\n")
		if err := introspectPlugin(intro, pluginID, outputDir, dryRun); err != nil {
			fmt.Printf("✗ Failed to introspect %s: %v\n", pluginID, err)
			failCount++
		} else {
			successCount++
		}
	}

	fmt.Printf("\n")
	fmt.Printf("Summary:\n")
	fmt.Printf("  ✓ Success: %d\n", successCount)
	if failCount > 0 {
		fmt.Printf("  ✗ Failed: %d\n", failCount)
	}

	return nil
}

func introspectEnterprisePlugins(intro *introspector.Introspector, outputDir string, dryRun bool) (int, int) {
	enterpriseDir := filepath.Join(".", "plugins", "enterprise")
	entries, err := os.ReadDir(enterpriseDir)
	if err != nil {
		fmt.Printf("✗ Failed to read enterprise directory: %v\n", err)
		return 0, 1
	}

	var successCount int
	var failCount int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		subPluginID := entry.Name()
		fullPluginID := "enterprise/" + subPluginID

		fmt.Printf("Introspecting enterprise sub-plugin: %s\n", subPluginID)

		// Generate manifest for sub-plugin
		manifest, err := intro.GenerateManifest(fullPluginID)
		if err != nil {
			fmt.Printf("✗ Failed to introspect %s: %v\n", fullPluginID, err)
			failCount++
			continue
		}

		// Check if manifest has routes - skip if none
		if len(manifest.Routes) == 0 {
			fmt.Printf("⚠ Skipped %s: No HTTP routes (middleware/service only)\n", subPluginID)
			continue
		}

		// Validate manifest
		if err := manifest.Validate(); err != nil {
			fmt.Printf("✗ Failed to introspect %s: generated manifest is invalid: %v\n", fullPluginID, err)
			failCount++
			continue
		}

		// Convert to YAML
		yamlData, err := yaml.Marshal(manifest)
		if err != nil {
			fmt.Printf("✗ Failed to marshal %s manifest: %v\n", fullPluginID, err)
			failCount++
			continue
		}

		if dryRun {
			fmt.Printf("\n--- %s.yaml ---\n", subPluginID)
			fmt.Println(string(yamlData))
			successCount++
			continue
		}

		// Write to file
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("✗ Failed to create output directory: %v\n", err)
			failCount++
			continue
		}

		outputPath := filepath.Join(outputDir, "enterprise-"+subPluginID+".yaml")
		if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
			fmt.Printf("✗ Failed to write %s manifest: %v\n", fullPluginID, err)
			failCount++
			continue
		}

		fmt.Printf("✓ Generated manifest: %s\n", outputPath)
		fmt.Printf("  - %d routes\n", len(manifest.Routes))
		fmt.Printf("  - %d types\n", len(manifest.Types))
		successCount++
	}

	return successCount, failCount
}
