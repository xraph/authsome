package notification

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	notifydash "github.com/xraph/authsome/plugins/notification/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin          = (*Plugin)(nil)
	_ dashboard.DashboardPageContributor = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// DashboardPlugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns no widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns the notification settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return notifydash.SettingsPanel(p.config.AppName, p.config.BaseURL, p.config.DefaultLocale, p.config.Async)
}

// DashboardPages returns nil — pages are handled via DashboardPageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// DashboardPageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the notifications page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Notifications",
			Path:     "/notifications",
			Icon:     "bell",
			Group:    "Configuration",
			Priority: 20,
		},
	}
}

// DashboardRenderPage renders the notifications configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/notifications" {
		return nil, contributor.ErrPageNotFound
	}

	// Build template mapping names for display.
	mappingNames := make([]string, 0, len(p.mappings))
	for action := range p.mappings {
		mappingNames = append(mappingNames, action)
	}

	return notifydash.ConfigPage(p.config.AppName, p.config.BaseURL, p.config.DefaultLocale, p.config.Async, mappingNames), nil
}
