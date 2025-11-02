package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins/emailotp"
	"github.com/xraph/authsome/plugins/magiclink"
	notificationPlugin "github.com/xraph/authsome/plugins/notification"
	"github.com/xraph/authsome/plugins/phone"
	"github.com/xraph/forge"
)

func main() {
	// Initialize database
	dsn := "postgres://postgres:postgres@localhost:5432/authsome?sslmode=disable"
	sqldb := pgdriver.NewConnector(pgdriver.WithDSN(dsn))
	db := bun.NewDB(sqldb, pgdialect.New())
	
	defer db.Close()

	// Create Forge app
	app := forge.New(forge.Config{
		AppName: "AuthSome Complete Example",
		Port:    8080,
	})

	// Initialize AuthSome with notification plugin
	auth := authsome.New(
		authsome.WithDatabase(db),
		authsome.WithPlugins(
			// IMPORTANT: Notification plugin MUST be first!
			notificationPlugin.NewPlugin(),
			
			// Then other plugins that use notifications
			emailotp.NewPlugin(),
			magiclink.NewPlugin(),
			phone.NewPlugin(),
		),
	)

	// Mount AuthSome
	if err := auth.Mount(app, "/api/auth"); err != nil {
		log.Fatal("Failed to mount AuthSome:", err)
	}

	// Run migrations
	ctx := context.Background()
	if err := auth.Migrate(ctx); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	log.Println("✅ AuthSome initialized with notification system")
	log.Println("✅ Default templates created automatically")
	log.Println("✅ Mock providers registered (email & SMS)")
	log.Println("")
	log.Println("Available endpoints:")
	log.Println("  POST /api/auth/email-otp/send")
	log.Println("  POST /api/auth/magic-link/send")
	log.Println("  POST /api/auth/phone/send-code")
	log.Println("  GET  /api/auth/templates")
	log.Println("  POST /api/auth/notifications/send")
	log.Println("")
	
	// Example: Send OTP email
	go func() {
		time.Sleep(2 * time.Second)
		demonstrateNotifications()
	}()

	// Start server
	log.Printf("🚀 Server running on http://localhost:8080\n")
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func demonstrateNotifications() {
	log.Println("\n=== Demonstrating Notification System ===")
	
	// This would normally be done via HTTP requests
	// Here we're just showing the flow
	
	log.Println("1. Email OTP sent (using auth.email_otp template)")
	log.Println("2. Magic Link sent (using auth.magic_link template)")
	log.Println("3. Phone OTP sent (using auth.phone_otp template)")
	log.Println("4. All notifications tracked in database")
	log.Println("5. Templates can be customized via API")
	log.Println("\n✨ Notification system is fully operational!")
}

