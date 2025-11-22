package apikey

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/authsome/core/ui"
)

func TestDashboardExtension_Interface(t *testing.T) {
	plugin := NewPlugin()

	// Ensure the plugin returns a dashboard extension
	ext := plugin.DashboardExtension()
	assert.NotNil(t, ext, "DashboardExtension should not be nil")

	// Verify it implements the interface (compile-time check via return type)
	var _ ui.DashboardExtension = ext
}

func TestDashboardExtension_ExtensionID(t *testing.T) {
	plugin := NewPlugin()
	ext := plugin.DashboardExtension()

	extensionID := ext.ExtensionID()
	assert.Equal(t, "apikey", extensionID, "Extension ID should be 'apikey'")
}

func TestDashboardExtension_NavigationItems(t *testing.T) {
	plugin := NewPlugin()
	ext := plugin.DashboardExtension()

	navItems := ext.NavigationItems()
	assert.NotNil(t, navItems, "NavigationItems should not be nil")
	assert.Equal(t, 0, len(navItems), "NavigationItems should be empty (settings-only plugin)")
}

func TestDashboardExtension_Routes(t *testing.T) {
	plugin := NewPlugin()
	ext := plugin.DashboardExtension()

	routes := ext.Routes()
	assert.NotNil(t, routes, "Routes should not be nil")
	assert.Greater(t, len(routes), 0, "Routes should not be empty")

	// Verify we have the expected routes
	routePaths := make(map[string]bool)
	for _, route := range routes {
		routePaths[route.Path] = true
	}

	// Check for key routes
	assert.True(t, routePaths["/settings/api-keys"], "Should have main API keys settings route")
	assert.True(t, routePaths["/settings/api-keys-config"], "Should have configuration route")
	assert.True(t, routePaths["/settings/api-keys-security"], "Should have security route")
	assert.True(t, routePaths["/settings/api-keys/create"], "Should have create route")
}

func TestDashboardExtension_SettingsPages(t *testing.T) {
	plugin := NewPlugin()
	ext := plugin.DashboardExtension()

	settingsPages := ext.SettingsPages()
	assert.NotNil(t, settingsPages, "SettingsPages should not be nil")
	assert.Equal(t, 3, len(settingsPages), "Should have 3 settings pages")

	// Verify page IDs
	pageIDs := make(map[string]bool)
	for _, page := range settingsPages {
		pageIDs[page.ID] = true
	}

	assert.True(t, pageIDs["api-keys"], "Should have api-keys page")
	assert.True(t, pageIDs["api-keys-config"], "Should have api-keys-config page")
	assert.True(t, pageIDs["api-keys-security"], "Should have api-keys-security page")

	// Verify categories
	for _, page := range settingsPages {
		if page.ID == "api-keys" || page.ID == "api-keys-config" {
			assert.Equal(t, "integrations", page.Category, "API keys and config pages should be in integrations category")
		} else if page.ID == "api-keys-security" {
			assert.Equal(t, "security", page.Category, "Security page should be in security category")
		}

		// All pages should require the apikey plugin
		assert.Equal(t, "apikey", page.RequirePlugin, "All pages should require apikey plugin")

		// All pages should require admin
		assert.True(t, page.RequireAdmin, "All pages should require admin")
	}
}

func TestDashboardExtension_DashboardWidgets(t *testing.T) {
	plugin := NewPlugin()
	ext := plugin.DashboardExtension()

	widgets := ext.DashboardWidgets()
	assert.NotNil(t, widgets, "DashboardWidgets should not be nil")
	assert.Equal(t, 1, len(widgets), "Should have 1 dashboard widget")

	// Verify widget properties
	widget := widgets[0]
	assert.Equal(t, "apikey-stats", widget.ID, "Widget ID should be apikey-stats")
	assert.Equal(t, "API Keys", widget.Title, "Widget title should be API Keys")
	assert.Equal(t, 40, widget.Order, "Widget order should be 40")
	assert.Equal(t, 1, widget.Size, "Widget size should be 1 column")
	assert.NotNil(t, widget.Renderer, "Widget renderer should not be nil")
}

func TestDashboardExtension_SettingsSections(t *testing.T) {
	plugin := NewPlugin()
	ext := plugin.DashboardExtension()

	sections := ext.SettingsSections()
	assert.NotNil(t, sections, "SettingsSections should not be nil")
	assert.Equal(t, 0, len(sections), "SettingsSections should be empty (deprecated, using SettingsPages)")
}
