package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins/dashboard"
)

// ServeMuxTestApp demonstrates AuthSome with pure http.ServeMux
type ServeMuxTestApp struct {
	db   *bun.DB
	mux  *http.ServeMux
	auth *authsome.Auth
}

func main() {
	log.Println("🚀 Starting AuthSome ServeMux Test...")

	app := &ServeMuxTestApp{}

	// Initialize components
	config := &Config{
		Mode:        authsome.ModeStandalone,
		DatabaseURL: getEnv("DATABASE_URL", "file:test.db?cache=shared&mode=rwc"),
		Port:        getEnv("PORT", "8082"),
		EnableDebug: getEnv("DEBUG", "false") == "true",
	}

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
	log.Printf("📊 Dashboard: http://localhost:%s/dashboard/", config.Port)
	log.Printf("🔐 Auth API: http://localhost:%s/api/auth", config.Port)

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: app.mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}

type Config struct {
	Mode        authsome.Mode
	DatabaseURL string
	Port        string
	EnableDebug bool
}

func (app *ServeMuxTestApp) initDatabase(config *Config) error {
	log.Println("🗄️  Initializing database...")

	sqldb, err := sql.Open(sqliteshim.ShimName, config.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	app.db = bun.NewDB(sqldb, sqlitedialect.New())
	log.Println("✅ Database initialized")
	return nil
}

func (app *ServeMuxTestApp) initHTTP() error {
	log.Println("🌐 Initializing HTTP server...")
	app.mux = http.NewServeMux()
	log.Println("✅ HTTP server initialized")
	return nil
}

func (app *ServeMuxTestApp) initAuthSome(config *Config) error {
	log.Println("🔐 Initializing AuthSome...")

	configManager := setupViper()

	app.auth = authsome.New(
		authsome.WithMode(config.Mode),
		authsome.WithDatabase(app.db),
		authsome.WithForgeConfig(configManager),
	)

	// Register only dashboard plugin for testing
	if err := app.auth.RegisterPlugin(dashboard.NewPlugin()); err != nil {
		return fmt.Errorf("failed to register dashboard plugin: %w", err)
	}
	log.Println("  📊 Dashboard registered")

	// Initialize AuthSome
	if err := app.auth.Initialize(context.Background()); err != nil {
		return fmt.Errorf("failed to initialize AuthSome: %w", err)
	}

	log.Println("✅ AuthSome initialized")
	return nil
}

func (app *ServeMuxTestApp) setupRoutes() error {
	log.Println("🛣️  Setting up routes...")

	// Mount AuthSome routes first
	if err := app.auth.Mount(app.mux, "/api/auth"); err != nil {
		return fmt.Errorf("failed to mount AuthSome: %w", err)
	}

	// Add a simple home route at /home to avoid conflict with dashboard
	app.mux.HandleFunc("/home", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>ServeMux Test</title>
</head>
<body>
    <h1>ServeMux Test App</h1>
    <p>This is a test app using pure http.ServeMux to test dashboard asset serving.</p>
    <ul>
        <li><a href="/dashboard/">Dashboard</a></li>
        <li><a href="/api/auth/status">Auth Status</a></li>
        <li><a href="/">Root (Dashboard)</a></li>
    </ul>
</body>
</html>
		`))
	})

	log.Println("✅ Routes configured")
	return nil
}

func setupViper() *viper.Viper {
	v := viper.New()
	
	// Set minimal configuration for testing
	v.SetDefault("auth.dashboard.enabled", true)
	
	return v
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}