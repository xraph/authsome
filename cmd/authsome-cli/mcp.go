package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP (Model Context Protocol) server",
	Long: `Start an MCP server to expose AuthSome data and operations to AI assistants.

The MCP server provides standardized access for AI tools to:
- Query users, sessions, and audit logs
- Check permissions and RBAC policies
- Get schema and configuration information
- Perform administrative tasks (in admin mode)`,
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server",
	Long: `Start the MCP (Model Context Protocol) server.

Transport modes:
  --stdio    Use stdin/stdout (default, for local CLI integration)
  --http     Use HTTP server (for remote access)

Operation modes:
  --mode=readonly      Read-only operations (default, safest)
  --mode=admin         Allow administrative operations
  --mode=development   Allow all operations including test data creation

Examples:
  # Start stdio server (for local AI assistant)
  authsome mcp serve --config=config.yaml

  # Start HTTP server with admin access
  authsome mcp serve --config=config.yaml --http --port=9090 --mode=admin

  # Development mode with test data creation
  authsome mcp serve --config=config.yaml --mode=development`,
	RunE: runMCPServe,
}

var (
	mcpMode      string
	mcpTransport string
	mcpHTTPPort  int
	mcpNoAuth    bool
)

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpServeCmd)

	mcpServeCmd.Flags().StringVar(&mcpMode, "mode", "readonly", "Operation mode: readonly, admin, or development")
	mcpServeCmd.Flags().StringVar(&mcpTransport, "transport", "stdio", "Transport: stdio or http")
	mcpServeCmd.Flags().IntVar(&mcpHTTPPort, "port", 9090, "HTTP port (only for --transport=http)")
	mcpServeCmd.Flags().BoolVar(&mcpNoAuth, "no-auth", false, "Disable API key requirement (INSECURE, development only)")
}

func runMCPServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize database
	db, err := connectDatabaseMulti()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	// Verify MCP mode
	var mode mcp.Mode
	switch mcpMode {
	case "readonly":
		mode = mcp.ModeReadOnly
	case "admin":
		mode = mcp.ModeAdmin
	case "development":
		mode = mcp.ModeDevelopment
	default:
		return fmt.Errorf("invalid mode: %s (must be readonly, admin, or development)", mcpMode)
	}

	// Verify transport
	var transport mcp.Transport
	switch mcpTransport {
	case "stdio":
		transport = mcp.TransportStdio
	case "http":
		transport = mcp.TransportHTTP
	default:
		return fmt.Errorf("invalid transport: %s (must be stdio or http)", mcpTransport)
	}

	// Create MCP plugin config
	mcpConfig := mcp.Config{
		Enabled:   true,
		Mode:      mode,
		Transport: transport,
		HTTPPort:  mcpHTTPPort,
		Authorization: mcp.AuthorizationConfig{
			RequireAPIKey: !mcpNoAuth,
			AllowedOperations: []string{
				"query_user",
				"list_sessions",
				"check_permission",
				"search_audit_logs",
				"explain_route",
				"validate_policy",
			},
			AdminOperations: []string{
				"revoke_session",
				"rotate_api_key",
			},
		},
		RateLimit: mcp.RateLimitConfig{
			RequestsPerMinute: 60,
		},
	}

	// Security warning for development mode
	if mode == mcp.ModeDevelopment || mcpNoAuth {
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: Running in %s mode with no-auth=%v\n", mode, mcpNoAuth)
		fmt.Fprintf(os.Stderr, "⚠️  This is INSECURE and should only be used in development!\n\n")
	}

	// Create a simple config manager from viper
	configManager := NewConfigManager()

	// Create AuthSome instance
	auth := authsome.New(
		authsome.WithDatabase(db),
		authsome.WithForgeConfig(configManager),
	)

	// Create and register MCP plugin
	mcpPlugin := mcp.NewPlugin(mcpConfig)
	if err := auth.RegisterPlugin(mcpPlugin); err != nil {
		return fmt.Errorf("failed to register MCP plugin: %w", err)
	}

	// Initialize auth (this initializes the plugin)
	if err := auth.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize auth: %w", err)
	}

	// Start MCP server
	fmt.Fprintf(os.Stderr, "🚀 Starting MCP server...\n")
	fmt.Fprintf(os.Stderr, "   Mode: %s\n", mode)
	fmt.Fprintf(os.Stderr, "   Transport: %s\n", transport)
	if transport == mcp.TransportHTTP {
		fmt.Fprintf(os.Stderr, "   Port: %d\n", mcpHTTPPort)
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "\n⏸️  Shutting down MCP server...\n")
		cancel()
	}()

	// Start server
	if err := mcpPlugin.Start(ctx); err != nil {
		if err == context.Canceled {
			fmt.Fprintf(os.Stderr, "✅ MCP server stopped gracefully\n")
			return nil
		}
		return fmt.Errorf("MCP server error: %w", err)
	}

	return nil
}
