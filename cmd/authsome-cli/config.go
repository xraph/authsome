package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Commands for managing AuthSome configuration files.`,
}

// Config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate configuration file",
	Long:  `Validate the syntax and structure of a configuration file.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := "config.yaml"
		if len(args) > 0 {
			configFile = args[0]
		}
		return validateConfig(configFile)
	},
}

// Config show command
var configShowCmd = &cobra.Command{
	Use:   "show [config-file]",
	Short: "Show configuration",
	Long:  `Display the current configuration with resolved values.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configFile := "config.yaml"
		if len(args) > 0 {
			configFile = args[0]
		}
		return showConfig(configFile)
	},
}

// Config init command
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration",
	Long:  `Create a new configuration file with default values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mode, _ := cmd.Flags().GetString("mode")
		output, _ := cmd.Flags().GetString("output")
		force, _ := cmd.Flags().GetBool("force")

		return initConfig(mode, output, force)
	},
}

func init() {
	// Add subcommands
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)

	// Init flags
	configInitCmd.Flags().StringP("mode", "m", "standalone", "Configuration mode (standalone|saas)")
	configInitCmd.Flags().StringP("output", "o", "config.yaml", "Output file path")
	configInitCmd.Flags().BoolP("force", "f", false, "Overwrite existing file")
}

// validateConfig validates a configuration file
func validateConfig(configFile string) error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configFile)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Basic validation
	if err := validateConfigStructure(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	fmt.Printf("Configuration file %s is valid\n", configFile)
	return nil
}

// showConfig displays the configuration
func showConfig(configFile string) error {
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s", configFile)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	fmt.Printf("Configuration from %s:\n\n", configFile)
	fmt.Print(string(data))
	return nil
}

// initConfig creates a new configuration file
func initConfig(mode, output string, force bool) error {
	if _, err := os.Stat(output); err == nil && !force {
		return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", output)
	}

	var config map[string]interface{}

	switch mode {
	case "standalone":
		config = getStandaloneConfig()
	case "saas":
		config = getSaaSConfig()
	default:
		return fmt.Errorf("invalid mode: %s (must be 'standalone' or 'saas')", mode)
	}

	// Create directory if it doesn't exist
	if dir := filepath.Dir(output); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(output, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Configuration file created: %s (mode: %s)\n", output, mode)
	return nil
}

// validateConfigStructure performs basic validation of config structure
func validateConfigStructure(config map[string]interface{}) error {
	// Check for required top-level sections
	requiredSections := []string{"server", "database", "auth"}
	for _, section := range requiredSections {
		if _, exists := config[section]; !exists {
			return fmt.Errorf("missing required section: %s", section)
		}
	}

	// Validate server section
	if server, ok := config["server"].(map[string]interface{}); ok {
		if _, exists := server["port"]; !exists {
			return fmt.Errorf("missing server.port")
		}
	}

	// Validate database section
	if database, ok := config["database"].(map[string]interface{}); ok {
		if _, exists := database["url"]; !exists {
			return fmt.Errorf("missing database.url")
		}
	}

	return nil
}

// getStandaloneConfig returns default standalone configuration
func getStandaloneConfig() map[string]interface{} {
	return map[string]interface{}{
		"server": map[string]interface{}{
			"port":    8080,
			"host":    "localhost",
			"baseUrl": "http://localhost:8080",
		},
		"database": map[string]interface{}{
			"url": "authsome.db",
		},
		"auth": map[string]interface{}{
			"mode": "standalone",
			"session": map[string]interface{}{
				"cookieName":   "authsome-session",
				"cookieSecure": false,
				"cookieMaxAge": 86400,
			},
			"password": map[string]interface{}{
				"minLength":      8,
				"requireUpper":   true,
				"requireLower":   true,
				"requireNumbers": true,
				"requireSymbols": false,
			},
			"rateLimit": map[string]interface{}{
				"enabled":     true,
				"maxAttempts": 5,
				"windowMs":    900000, // 15 minutes
			},
		},
		"plugins": map[string]interface{}{
			"username": map[string]interface{}{
				"enabled": true,
			},
			"twofa": map[string]interface{}{
				"enabled": false,
			},
		},
		"email": map[string]interface{}{
			"provider": "smtp",
			"smtp": map[string]interface{}{
				"host":     "localhost",
				"port":     587,
				"username": "",
				"password": "",
				"from":     "noreply@example.com",
			},
		},
	}
}

// getSaaSConfig returns default SaaS configuration
func getSaaSConfig() map[string]interface{} {
	config := getStandaloneConfig()

	// Override for SaaS mode
	auth := config["auth"].(map[string]interface{})
	auth["mode"] = "saas"

	// Add organization settings
	config["organizations"] = map[string]interface{}{
		"allowCreation":   true,
		"requireApproval": false,
		"defaultPlan":     "free",
		"maxUsersPerOrg":  100,
	}

	// Add multi-tenancy settings
	config["multiTenant"] = map[string]interface{}{
		"enabled":          true,
		"subdomainRouting": false,
		"pathBasedRouting": true,
	}

	return config
}
