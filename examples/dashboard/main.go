// Package main demonstrates how to integrate the dashboard plugin with AuthSome
//
// This example shows the basic integration pattern. In a real application,
// you would use this with a proper Forge framework setup.
//
// Usage:
//
//	go run examples/dashboard/main.go
//
// The dashboard plugin provides:
//   - Static asset serving at /dashboard/*
//   - React-based admin interface
//   - User management UI
//   - Session monitoring
//   - Security settings
package main

import (
	"log"

	"github.com/xraph/authsome"
	"github.com/xraph/authsome/plugins/dashboard"
)

func main() {
	log.Println("AuthSome Dashboard Plugin Integration Example")
	log.Println("===========================================")

	// Create AuthSome instance
	auth := authsome.New()

	// Create and register the dashboard plugin
	dashboardPlugin := dashboard.NewPlugin()

	log.Printf("Registering dashboard plugin: %s", dashboardPlugin.ID())
	if err := auth.RegisterPlugin(dashboardPlugin); err != nil {
		log.Fatal("Failed to register dashboard plugin:", err)
	}

	// Initialize the plugin (this would normally be done by AuthSome.Initialize)
	log.Println("Initializing dashboard plugin...")
	if err := dashboardPlugin.Init(auth); err != nil {
		log.Fatal("Failed to initialize dashboard plugin:", err)
	}

	// Test asset access
	log.Println("Testing dashboard assets...")
	// Note: dashboard.GetAssets() is not publicly exported
	// The dashboard plugin will serve assets when mounted
	log.Println("✅ Dashboard plugin registered successfully!")
	log.Println("   Assets will be available when the plugin is mounted to a Forge app")

}
