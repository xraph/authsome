package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins"
	"github.com/xraph/authsome/plugins/anonymous"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/authsome/plugins/emailotp"
	"github.com/xraph/authsome/plugins/magiclink"
	"github.com/xraph/authsome/plugins/multisession"
	"github.com/xraph/authsome/plugins/multitenancy"
	"github.com/xraph/authsome/plugins/oidcprovider"
	"github.com/xraph/authsome/plugins/passkey"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/authsome/plugins/sso"
	"github.com/xraph/authsome/plugins/twofa"
	"github.com/xraph/authsome/plugins/username"
	"github.com/xraph/forge"
)

// ComprehensiveApp demonstrates all AuthSome features
type ComprehensiveApp struct {
	db   *bun.DB
	app  forge.App
	auth *authsome.Auth
}

// setupViper configures Viper with default settings for AuthSome
func setupViper() *viper.Viper {
	v := viper.New()

	// Set default configuration values for multitenancy plugin
	v.SetDefault("auth.multitenancy.platformOrganizationId", "platform")
	v.SetDefault("auth.multitenancy.defaultOrganizationName", "Default Organization")
	v.SetDefault("auth.multitenancy.enableOrganizationCreation", true)
	v.SetDefault("auth.multitenancy.maxMembersPerOrganization", 100)
	v.SetDefault("auth.multitenancy.maxTeamsPerOrganization", 10)
	v.SetDefault("auth.multitenancy.requireInvitation", false)
	v.SetDefault("auth.multitenancy.invitationExpiryHours", 72)

	return v
}

// Config holds application configuration
type Config struct {
	Mode        authsome.Mode
	DatabaseURL string
	Port        string
	EnableDebug bool
}

func main() {
	log.Println("🚀 Starting AuthSome Comprehensive Example...")

	// Load configuration
	config := &Config{
		Mode:        authsome.ModeSaaS,
		DatabaseURL: getEnv("DATABASE_URL", "file:authsome_comprehensive.db?cache=shared&_fk=1"),
		Port:        getEnv("PORT", "8081"),
		EnableDebug: getEnv("DEBUG", "true") == "true",
	}

	// Initialize application
	app := &ComprehensiveApp{}

	// Initialize components
	if err := app.initDatabase(config); err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}

	if err := app.initHTTP(); err != nil {
		log.Fatalf("❌ Failed to initialize HTTP: %v", err)
	}

	if err := app.initAuthSome(config); err != nil {
		log.Fatalf("❌ Failed to initialize AuthSome: %v", err)
	}

	if err := app.setupRoutes(); err != nil {
		log.Fatalf("❌ Failed to setup routes: %v", err)
	}

	// Start server
	log.Printf("🌟 Server starting on port %s", config.Port)
	log.Printf("📱 Dashboard: http://localhost:%s/home", config.Port)
	log.Printf("🔐 Auth API: http://localhost:%s/api/auth", config.Port)
	log.Printf("📊 Status: http://localhost:%s/status", config.Port)

	if err := app.app.Run(); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}

// initDatabase initializes the database connection
func (app *ComprehensiveApp) initDatabase(config *Config) error {
	log.Println("🗄️  Initializing database...")

	sqldb, err := sql.Open(sqliteshim.ShimName, config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	app.db = bun.NewDB(sqldb, sqlitedialect.New())

	log.Println("✅ Database initialized")
	return nil
}

// initHTTP initializes the HTTP server
func (app *ComprehensiveApp) initHTTP() error {
	log.Println("🌐 Initializing HTTP server...")
	app.app = forge.NewApp(forge.AppConfig{
		Name:        "authsome-comprehensive",
		Version:     "1.0.0",
		Environment: "development",
		HTTPAddress: ":8081",
	})
	log.Println("✅ HTTP server initialized")
	return nil
}

// initAuthSome initializes AuthSome with configuration
func (app *ComprehensiveApp) initAuthSome(config *Config) error {
	log.Println("🔐 Initializing AuthSome...")

	app.auth = authsome.New(
		authsome.WithMode(config.Mode),
		authsome.WithDatabase(app.db),
		authsome.WithForgeApp(app.app),
	)

	// Register plugins before initialization
	if err := app.registerPlugins(); err != nil {
		return fmt.Errorf("failed to register plugins: %w", err)
	}

	// Initialize AuthSome (this will call Init on all registered plugins)
	if err := app.auth.Initialize(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize AuthSome: %w", err)
	}

	log.Println("✅ AuthSome initialized")
	return nil
}

// registerPlugins registers all available plugins
func (app *ComprehensiveApp) registerPlugins() error {
	log.Println("🔌 Registering plugins...")

	// List of all available plugins with proper function signature
	pluginRegistrations := []struct {
		name   string
		plugin func() plugins.Plugin
		emoji  string
	}{
		{"Dashboard", func() plugins.Plugin { return dashboard.NewPlugin() }, "📊"},
		{"Username", func() plugins.Plugin { return username.NewPlugin() }, "👤"},
		{"Two-Factor Auth", func() plugins.Plugin { return twofa.NewPlugin() }, "🔐"},
		{"Anonymous", func() plugins.Plugin { return anonymous.NewPlugin() }, "👻"},
		{"Multi-tenancy", func() plugins.Plugin { return multitenancy.NewPlugin() }, "🏢"},
		{"Email OTP", func() plugins.Plugin { return emailotp.NewPlugin() }, "📧"},
		{"Magic Link", func() plugins.Plugin { return magiclink.NewPlugin() }, "✨"},
		{"Phone", func() plugins.Plugin { return phone.NewPlugin() }, "📱"},
		{"Passkey", func() plugins.Plugin { return passkey.NewPlugin() }, "🔑"},
		{"SSO", func() plugins.Plugin { return sso.NewPlugin() }, "🏢"},
		{"Multi-session", func() plugins.Plugin { return multisession.NewPlugin() }, "🔄"},
		{"OIDC Provider", func() plugins.Plugin { return oidcprovider.NewPlugin() }, "🔐"},
	}

	// Register each plugin
	for _, p := range pluginRegistrations {
		plugin := p.plugin()
		if err := app.auth.RegisterPlugin(plugin); err != nil {
			log.Printf("⚠️  Failed to register %s plugin: %v", p.name, err)
		} else {
			log.Printf("  %s %s registered", p.emoji, p.name)
		}
	}

	log.Println("✅ Plugin registration completed")
	return nil
}

// setupRoutes configures all application routes
func (app *ComprehensiveApp) setupRoutes() error {
	log.Println("🛣️  Setting up routes...")

	// Add application routes first
	app.setupAppRoutes()

	// Mount AuthSome routes
	if err := app.auth.Mount(app.app.Router(), "/api/auth"); err != nil {
		return fmt.Errorf("failed to mount AuthSome: %w", err)
	}

	log.Println("✅ Routes configured")
	return nil
}

// setupAppRoutes adds application-specific routes
func (app *ComprehensiveApp) setupAppRoutes() {
	router := app.app.Router()
	
	// Health check
	router.GET("/health", func(c forge.Context) error {
		return c.JSON(200, map[string]string{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"version":   "1.0.0",
		})
	})

	// Status endpoint
	router.GET("/status", func(c forge.Context) error {
		return c.JSON(200, map[string]interface{}{
			"authsome":  "initialized",
			"database":  "connected",
			"plugins":   []string{"dashboard", "multitenancy", "username", "twofa", "emailotp", "magiclink", "phone", "passkey", "anonymous", "sso"},
		})
	})

	// Home page
	router.GET("/home", func(c forge.Context) error {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>AuthSome Comprehensive Example</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 3px solid #007acc; padding-bottom: 10px; }
        .feature { background: #f8f9fa; padding: 15px; margin: 10px 0; border-radius: 5px; border-left: 4px solid #007acc; }
        .endpoint { background: #e8f4fd; padding: 10px; margin: 5px 0; border-radius: 3px; font-family: monospace; }
        a { color: #007acc; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .status { color: #28a745; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🚀 AuthSome Comprehensive Example</h1>
        <p class="status">✅ All systems operational</p>
        
        <div class="feature">
            <h3>🔐 Authentication Endpoints</h3>
            <div class="endpoint">POST /api/auth/signup - User registration</div>
            <div class="endpoint">POST /api/auth/signin - User login</div>
            <div class="endpoint">POST /api/auth/signout - User logout</div>
            <div class="endpoint">GET /api/auth/session - Get current session</div>
        </div>

        <div class="feature">
            <h3>🔌 Available Plugins</h3>
            <ul>
                <li>Dashboard - Admin interface</li>
                <li>Multi-tenancy - Organization support</li>
                <li>Username - Username authentication</li>
                <li>2FA - Two-factor authentication</li>
                <li>Email OTP - Email-based verification</li>
                <li>Magic Link - Passwordless login</li>
                <li>Phone - SMS authentication</li>
                <li>Passkey - WebAuthn support</li>
                <li>Anonymous - Guest users</li>
                <li>SSO - Single sign-on</li>
            </ul>
        </div>

        <div class="feature">
            <h3>📊 System Status</h3>
            <p><a href="/status">View JSON status</a></p>
            <p><a href="/health">Health check</a></p>
        </div>

        <div class="feature">
            <h3>📚 Documentation</h3>
            <p>Visit the <a href="https://github.com/xraph/authsome">AuthSome GitHub repository</a> for complete documentation.</p>
        </div>
    </div>
</body>
</html>`
		c.Response().Header().Set("Content-Type", "text/html")
		return c.String(200, html)
	})

	// Test endpoints
	router.GET("/test/auth/signup", func(c forge.Context) error {
		return c.JSON(200, map[string]string{
			"endpoint":    "POST /api/auth/signup",
			"description": "User registration endpoint",
			"example":     `{"email": "user@example.com", "password": "password123"}`,
		})
	})

	router.GET("/test/auth/signin", func(c forge.Context) error {
		return c.JSON(200, map[string]string{
			"endpoint":    "POST /api/auth/signin",
			"description": "User login endpoint",
			"example":     `{"email": "user@example.com", "password": "password123"}`,
		})
	})
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
