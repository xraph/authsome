package organization

import (
	"fmt"
	"sort"
	"sync"

	"github.com/xraph/authsome/core/ui"
)

// OrganizationUIRegistry manages organization UI extensions from plugins
// It collects widgets, tabs, actions, and quick links from registered plugins
// and provides them in sorted, filtered order for rendering
type OrganizationUIRegistry struct {
	extensions map[string]ui.OrganizationUIExtension
	mu         sync.RWMutex
}

// NewOrganizationUIRegistry creates a new organization UI registry
func NewOrganizationUIRegistry() *OrganizationUIRegistry {
	return &OrganizationUIRegistry{
		extensions: make(map[string]ui.OrganizationUIExtension),
	}
}

// Register registers an organization UI extension
// Returns an error if an extension with the same ID is already registered
func (r *OrganizationUIRegistry) Register(ext ui.OrganizationUIExtension) error {
	if ext == nil {
		return fmt.Errorf("cannot register nil extension")
	}

	extID := ext.ExtensionID()
	if extID == "" {
		return fmt.Errorf("extension ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.extensions[extID]; exists {
		return fmt.Errorf("extension with ID %s already registered", extID)
	}

	r.extensions[extID] = ext
	return nil
}

// GetWidgets returns all registered widgets, sorted by order
// Filters out widgets that require admin if the user is not an admin
func (r *OrganizationUIRegistry) GetWidgets(ctx ui.OrgExtensionContext) []ui.OrganizationWidget {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var widgets []ui.OrganizationWidget
	seen := make(map[string]bool)

	for _, ext := range r.extensions {
		for _, widget := range ext.OrganizationWidgets() {
			// Skip if already seen (duplicate ID check)
			if seen[widget.ID] {
				continue
			}
			seen[widget.ID] = true

			// Filter by admin requirement
			if widget.RequireAdmin && !ctx.IsAdmin {
				continue
			}

			widgets = append(widgets, widget)
		}
	}

	// Sort by order
	sort.Slice(widgets, func(i, j int) bool {
		return widgets[i].Order < widgets[j].Order
	})

	return widgets
}

// GetTabs returns all registered tabs, sorted by order
// Filters out tabs that require admin if the user is not an admin
func (r *OrganizationUIRegistry) GetTabs(ctx ui.OrgExtensionContext) []ui.OrganizationTab {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tabs []ui.OrganizationTab
	seen := make(map[string]bool)

	for _, ext := range r.extensions {
		for _, tab := range ext.OrganizationTabs() {
			// Skip if already seen (duplicate ID check)
			if seen[tab.ID] {
				continue
			}
			seen[tab.ID] = true

			// Filter by admin requirement
			if tab.RequireAdmin && !ctx.IsAdmin {
				continue
			}

			tabs = append(tabs, tab)
		}
	}

	// Sort by order
	sort.Slice(tabs, func(i, j int) bool {
		return tabs[i].Order < tabs[j].Order
	})

	return tabs
}

// GetTabByPath returns a tab by its path, or nil if not found
func (r *OrganizationUIRegistry) GetTabByPath(ctx ui.OrgExtensionContext, path string) *ui.OrganizationTab {
	tabs := r.GetTabs(ctx)
	for _, tab := range tabs {
		if tab.Path == path {
			return &tab
		}
	}
	return nil
}

// GetActions returns all registered actions, sorted by order
// Filters out actions that require admin if the user is not an admin
func (r *OrganizationUIRegistry) GetActions(ctx ui.OrgExtensionContext) []ui.OrganizationAction {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var actions []ui.OrganizationAction
	seen := make(map[string]bool)

	for _, ext := range r.extensions {
		for _, action := range ext.OrganizationActions() {
			// Skip if already seen (duplicate ID check)
			if seen[action.ID] {
				continue
			}
			seen[action.ID] = true

			// Filter by admin requirement
			if action.RequireAdmin && !ctx.IsAdmin {
				continue
			}

			actions = append(actions, action)
		}
	}

	// Sort by order
	sort.Slice(actions, func(i, j int) bool {
		return actions[i].Order < actions[j].Order
	})

	return actions
}

// GetQuickLinks returns all registered quick links, sorted by order
// Filters out links that require admin if the user is not an admin
func (r *OrganizationUIRegistry) GetQuickLinks(ctx ui.OrgExtensionContext) []ui.OrganizationQuickLink {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var links []ui.OrganizationQuickLink
	seen := make(map[string]bool)

	for _, ext := range r.extensions {
		for _, link := range ext.OrganizationQuickLinks() {
			// Skip if already seen (duplicate ID check)
			if seen[link.ID] {
				continue
			}
			seen[link.ID] = true

			// Filter by admin requirement
			if link.RequireAdmin && !ctx.IsAdmin {
				continue
			}

			links = append(links, link)
		}
	}

	// Sort by order
	sort.Slice(links, func(i, j int) bool {
		return links[i].Order < links[j].Order
	})

	return links
}

// GetSettingsSections returns all registered settings sections, sorted by order
// Filters out sections that require admin if the user is not an admin
func (r *OrganizationUIRegistry) GetSettingsSections(ctx ui.OrgExtensionContext) []ui.OrganizationSettingsSection {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var sections []ui.OrganizationSettingsSection
	seen := make(map[string]bool)

	for _, ext := range r.extensions {
		for _, section := range ext.OrganizationSettingsSections() {
			// Skip if already seen (duplicate ID check)
			if seen[section.ID] {
				continue
			}
			seen[section.ID] = true

			// Filter by admin requirement
			if section.RequireAdmin && !ctx.IsAdmin {
				continue
			}

			sections = append(sections, section)
		}
	}

	// Sort by order
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Order < sections[j].Order
	})

	return sections
}

// ListExtensions returns the IDs of all registered extensions
func (r *OrganizationUIRegistry) ListExtensions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.extensions))
	for id := range r.extensions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// HasExtension checks if an extension with the given ID is registered
func (r *OrganizationUIRegistry) HasExtension(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.extensions[id]
	return exists
}
