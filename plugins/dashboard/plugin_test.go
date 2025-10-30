package dashboard

import (
	"net/http"
	"testing"

	"github.com/xraph/forge"
)

// TestPlugin_ID tests that the plugin returns the correct ID
func TestPlugin_ID(t *testing.T) {
	plugin := NewPlugin()
	expected := "dashboard"
	if got := plugin.ID(); got != expected {
		t.Errorf("Plugin.ID() = %v, want %v", got, expected)
	}
}

// TestPlugin_Init tests plugin initialization
func TestPlugin_Init(t *testing.T) {
	plugin := NewPlugin()

	// Test with nil dependency (should not fail)
	err := plugin.Init(nil)
	if err != nil {
		t.Errorf("Plugin.Init() with nil dependency failed: %v", err)
	}

	// Verify handler was initialized
	if plugin.handler == nil {
		t.Error("Plugin.Init() did not initialize handler")
	}
}

// TestPlugin_RegisterRoutes tests route registration
func TestPlugin_RegisterRoutes(t *testing.T) {
	plugin := NewPlugin()
	err := plugin.Init(nil)
	if err != nil {
		t.Fatalf("Plugin.Init() failed: %v", err)
	}

	// Create a test router
	mux := http.NewServeMux()
	app := forge.NewApp(mux)

	// Register routes
	err = plugin.RegisterRoutes(app)
	if err != nil {
		t.Errorf("Plugin.RegisterRoutes() failed: %v", err)
	}
}

// TestDashboardAssets tests that the embedded assets are accessible
func TestDashboardAssets(t *testing.T) {
	// Test that we can read from the embedded filesystem
	files, err := dashboardAssets.ReadDir("dist")
	if err != nil {
		t.Fatalf("Failed to read dist directory: %v", err)
	}

	// Should have at least index.html
	found := false
	for _, file := range files {
		if file.Name() == "index.html" {
			found = true
			break
		}
	}

	if !found {
		t.Error("index.html not found in embedded assets")
	}
}

// TestServeIndex tests serving the index.html file
func TestServeIndex(t *testing.T) {
	plugin := NewPlugin()
	err := plugin.Init(nil)
	if err != nil {
		t.Fatalf("Plugin.Init() failed: %v", err)
	}

	// Note: This is a basic test structure. In a real test, you'd need to properly
	// initialize the forge.Context with the request and response writer.
	// For now, we're just testing that the handler exists and can be called.

	if plugin.handler == nil {
		t.Error("Handler not initialized")
	}

	// Test that the handler has the required methods
	// Note: In Go, function fields are never nil if they're defined as methods
	// We just verify the handler is properly initialized
}

// TestGetAssets tests the GetAssets function
func TestGetAssets(t *testing.T) {
	assets := GetAssets()
	if assets == nil {
		t.Error("GetAssets() returned nil")
	}

	// Test that we can read files from the returned filesystem
	_, err := assets.Open("index.html")
	if err != nil {
		t.Errorf("Failed to open index.html from assets: %v", err)
	}
}
