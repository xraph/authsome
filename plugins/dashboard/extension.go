package dashboard

import (
	"sort"
	"sync"

	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui"
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// ExtensionRegistry manages dashboard extensions from plugins
type ExtensionRegistry struct {
	mu         sync.RWMutex
	extensions map[string]ui.DashboardExtension
	handler    *Handler // Reference to handler for rendering helpers
}

// NewExtensionRegistry creates a new extension registry
func NewExtensionRegistry() *ExtensionRegistry {
	return &ExtensionRegistry{
		extensions: make(map[string]ui.DashboardExtension),
	}
}

// Register registers a dashboard extension
func (r *ExtensionRegistry) Register(ext ui.DashboardExtension) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := ext.ExtensionID()
	if _, exists := r.extensions[id]; exists {
		return ErrExtensionAlreadyRegistered
	}

	r.extensions[id] = ext

	// If extension has SetRegistry method, call it to provide registry reference
	if setter, ok := ext.(interface{ SetRegistry(*ExtensionRegistry) }); ok {
		setter.SetRegistry(r)
	}

	return nil
}

// Get retrieves an extension by ID
func (r *ExtensionRegistry) Get(id string) (ui.DashboardExtension, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ext, ok := r.extensions[id]
	return ext, ok
}

// List returns all registered extensions
func (r *ExtensionRegistry) List() []ui.DashboardExtension {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exts := make([]ui.DashboardExtension, 0, len(r.extensions))
	for _, ext := range r.extensions {
		exts = append(exts, ext)
	}
	return exts
}

// GetNavigationItems returns all navigation items for a specific position
func (r *ExtensionRegistry) GetNavigationItems(position ui.NavigationPosition, enabledPlugins map[string]bool) []ui.NavigationItem {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []ui.NavigationItem
	for _, ext := range r.extensions {
		for _, item := range ext.NavigationItems() {
			// Filter by position
			if item.Position != position {
				continue
			}

			// Check if required plugin is enabled
			if item.RequiresPlugin != "" && !enabledPlugins[item.RequiresPlugin] {
				continue
			}

			items = append(items, item)
		}
	}

	// Sort by order
	sort.Slice(items, func(i, j int) bool {
		return items[i].Order < items[j].Order
	})

	return items
}

// GetAllRoutes returns all routes from all extensions
func (r *ExtensionRegistry) GetAllRoutes() []ui.Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var routes []ui.Route
	for _, ext := range r.extensions {
		routes = append(routes, ext.Routes()...)
	}
	return routes
}

// GetSettingsSections returns all settings sections sorted by order
// Deprecated: Use GetSettingsPages for the new sidebar layout
func (r *ExtensionRegistry) GetSettingsSections() []ui.SettingsSection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sections []ui.SettingsSection
	for _, ext := range r.extensions {
		sections = append(sections, ext.SettingsSections()...)
	}

	// Sort by order
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Order < sections[j].Order
	})

	return sections
}

// GetSettingsPages returns all settings pages from extensions
func (r *ExtensionRegistry) GetSettingsPages(enabledPlugins map[string]bool) []ui.SettingsPage {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var pages []ui.SettingsPage
	for _, ext := range r.extensions {
		for _, page := range ext.SettingsPages() {
			// Check if required plugin is enabled
			if page.RequirePlugin != "" && !enabledPlugins[page.RequirePlugin] {
				continue
			}
			pages = append(pages, page)
		}
	}

	// Sort by category, then order
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Category != pages[j].Category {
			// Category order: general, security, communication, integrations, advanced
			catOrder := map[string]int{"general": 0, "security": 1, "communication": 2, "integrations": 3, "advanced": 4}
			iOrder, iExists := catOrder[pages[i].Category]
			jOrder, jExists := catOrder[pages[j].Category]
			if !iExists {
				iOrder = 999
			}
			if !jExists {
				jOrder = 999
			}
			return iOrder < jOrder
		}
		return pages[i].Order < pages[j].Order
	})

	return pages
}

// GetDashboardWidgets returns all dashboard widgets sorted by order
func (r *ExtensionRegistry) GetDashboardWidgets() []ui.DashboardWidget {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var widgets []ui.DashboardWidget
	for _, ext := range r.extensions {
		widgets = append(widgets, ext.DashboardWidgets()...)
	}

	// Sort by order
	sort.Slice(widgets, func(i, j int) bool {
		return widgets[i].Order < widgets[j].Order
	})

	return widgets
}

// RenderNavigationItems renders navigation items as gomponents nodes
func RenderNavigationItems(items []ui.NavigationItem, basePath string, currentApp *app.App, activePage string) []g.Node {
	nodes := make([]g.Node, 0, len(items))

	for _, item := range items {
		// Check if item is active
		isActive := false
		if item.ActiveChecker != nil {
			isActive = item.ActiveChecker(activePage)
		}

		// Build URL
		url := item.URLBuilder(basePath, currentApp)

		// Create navigation link
		nodes = append(nodes, navLinkWithIcon(item.Label, url, isActive, item.Icon))
	}

	return nodes
}

// navLinkWithIcon creates a navigation link with an optional icon
func navLinkWithIcon(label, href string, active bool, icon g.Node) g.Node {
	classes := "inline-flex items-center gap-2 rounded-lg px-2 py-1.5 text-xs transition "
	if active {
		classes += "font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"
	} else {
		classes += "text-slate-700 font-normal hover:bg-slate-100 dark:text-gray-300 dark:hover:bg-gray-800"
	}

	children := []g.Node{}
	if icon != nil {
		children = append(children, icon)
	}
	children = append(children, g.Text(label))

	return html.A(
		html.Href(href),
		html.Class(classes),
		g.Group(children),
	)
}

// GetHandler returns the handler instance for extensions to use
// Extensions can use this to access renderWithLayout and other helpers
func (r *ExtensionRegistry) GetHandler() *Handler {
	return r.handler
}

// SetHandler sets the handler instance (called by dashboard plugin during init)
func (r *ExtensionRegistry) SetHandler(h *Handler) {
	r.handler = h
}

var (
	// ErrExtensionAlreadyRegistered indicates an extension with the same ID is already registered
	ErrExtensionAlreadyRegistered = &DashboardError{
		Code:    "extension_already_registered",
		Message: "dashboard extension already registered",
	}
)

// DashboardError represents a dashboard-specific error
type DashboardError struct {
	Code    string
	Message string
}

func (e *DashboardError) Error() string {
	return e.Message
}
