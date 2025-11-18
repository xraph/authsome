package ui

import (
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
)

// NavigationPosition defines where a navigation item should be placed
type NavigationPosition string

const (
	// NavPositionMain places the item in the main navigation bar
	NavPositionMain NavigationPosition = "main"
	// NavPositionSettings places the item in the settings section
	NavPositionSettings NavigationPosition = "settings"
	// NavPositionUserDropdown places the item in the user dropdown menu
	NavPositionUserDropdown NavigationPosition = "user_dropdown"
	// NavPositionFooter places the item in the footer
	NavPositionFooter NavigationPosition = "footer"
)

// NavigationItem represents a navigation link registered by a plugin
type NavigationItem struct {
	// ID is a unique identifier for this nav item (e.g., "multisession")
	ID string
	// Label is the display text for the link
	Label string
	// Icon is an optional icon component (lucide icon recommended)
	Icon g.Node
	// Position determines where the nav item appears
	Position NavigationPosition
	// Order determines the display order within the position (lower = first)
	Order int
	// URLBuilder builds the URL for this nav item given the current app
	// Example: func(basePath string, app *app.App) string {
	//   return basePath + "/dashboard/app/" + app.ID.String() + "/multisession"
	// }
	URLBuilder func(basePath string, currentApp *app.App) string
	// ActiveChecker returns true if this nav item is currently active
	// Example: func(activePage string) bool { return activePage == "multisession" }
	ActiveChecker func(activePage string) bool
	// RequiresPlugin optionally specifies a plugin ID that must be enabled
	RequiresPlugin string
	// PermissionRequired optionally specifies a permission required to see this item
	PermissionRequired string
}

// Route represents a route registered by a plugin extension
type Route struct {
	// Method is the HTTP method (GET, POST, etc.)
	Method string
	// Path is the route path (relative to /dashboard/app/:appId/)
	// Example: "/multisession" becomes "/dashboard/app/:appId/multisession"
	Path string
	// Handler is the route handler function
	// Should be: func(c forge.Context) error
	// Use PageRenderer helper to render with dashboard layout
	Handler interface{}
	// Name is the route name for identification
	Name string
	// Summary is a short description for OpenAPI
	Summary string
	// Description is a detailed description for OpenAPI
	Description string
	// Tags are OpenAPI tags
	Tags []string
	// RequireAuth indicates if the route requires authentication
	RequireAuth bool
	// RequireAdmin indicates if the route requires admin privileges
	RequireAdmin bool
	// PageRenderer is an optional helper that renders the page content
	// The dashboard will wrap this in the layout automatically
	PageRenderer PageRenderer
}

// PageRenderer is a function that renders page content for an extension
// It receives helpers for building PageData and returns the content node
// The dashboard automatically wraps this in the layout (header, nav, footer)
type PageRenderer func(helpers PageHelpers) (g.Node, error)

// SettingsSection represents a section in the settings page
// Deprecated: Use SettingsPage instead for full-page settings
type SettingsSection struct {
	// ID is a unique identifier for this section
	ID string
	// Title is the section title
	Title string
	// Description is the section description
	Description string
	// Icon is an optional icon
	Icon g.Node
	// Order determines display order (lower = first)
	Order int
	// Renderer renders the section content
	Renderer func(basePath string, currentApp *app.App) g.Node
}

// SettingsPage represents a full settings page in the sidebar navigation
type SettingsPage struct {
	// ID is a unique identifier for this page (e.g., "role-templates", "api-keys")
	ID string
	// Label is the display text in the sidebar
	Label string
	// Description is a brief description of what this page does
	Description string
	// Icon is an optional icon component (lucide icon recommended)
	Icon g.Node
	// Category groups pages in the sidebar ("general", "security", "communication", "integrations", "advanced")
	Category string
	// Order determines the display order within the category (lower = first)
	Order int
	// Path is the URL path relative to /settings/ (e.g., "roles", "api-keys")
	Path string
	// RequirePlugin optionally specifies a plugin ID that must be enabled
	RequirePlugin string
	// RequireAdmin indicates if admin privileges are required
	RequireAdmin bool
}

// DashboardWidget represents a widget on the dashboard home page
type DashboardWidget struct {
	// ID is a unique identifier for this widget
	ID string
	// Title is the widget title
	Title string
	// Icon is an optional icon
	Icon g.Node
	// Order determines display order (lower = first)
	Order int
	// Size determines the widget size (1 = 1 column, 2 = 2 columns, etc.)
	Size int
	// Renderer renders the widget content
	Renderer func(basePath string, currentApp *app.App) g.Node
}

// DashboardExtension is the interface that plugins implement to extend the dashboard
type DashboardExtension interface {
	// ExtensionID returns a unique identifier for this extension
	ExtensionID() string

	// NavigationItems returns navigation items to register
	NavigationItems() []NavigationItem

	// Routes returns routes to register under /dashboard/app/:appId/
	Routes() []Route

	// SettingsSections returns settings sections to add to the settings page
	// Deprecated: Use SettingsPages() instead for full-page settings
	// Returns a list of setting section renderers
	SettingsSections() []SettingsSection

	// SettingsPages returns full settings pages to add to the settings sidebar
	// These pages get their own routes and full-page layouts
	SettingsPages() []SettingsPage

	// DashboardWidgets returns widgets to show on the main dashboard
	// These appear as cards on the dashboard home page
	DashboardWidgets() []DashboardWidget
}

// PageHelpers provides helper functions for extension page renderers
type PageHelpers interface {
	// GetCurrentUser returns the currently authenticated user
	GetCurrentUser() interface{}

	// GetCurrentApp returns the current app from URL context
	GetCurrentApp() interface{}

	// GetUserApps returns all apps the current user has access to
	GetUserApps() []interface{}

	// GetBasePath returns the dashboard base path
	GetBasePath() string

	// GetEnabledPlugins returns map of enabled plugin IDs
	GetEnabledPlugins() map[string]bool

	// SetTitle sets the page title
	SetTitle(title string)

	// SetActivePage sets the active page identifier for navigation highlighting
	SetActivePage(page string)
}
