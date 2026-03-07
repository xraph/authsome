package geofence

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/plugins/geofence/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin          = (*Plugin)(nil)
	_ dashboard.DashboardPageContributor = (*Plugin)(nil)
)

// DashboardWidgets returns no widgets for the geofence plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns a settings panel showing geofence configuration.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return dashui.SettingsPanel(
		p.config.DefaultPolicy,
		p.config.AllowedCountries,
		p.config.BlockedCountries,
		p.config.BlockMessage,
	)
}

// DashboardPages returns nil — pages are handled via DashboardPageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// DashboardPageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the geofence page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Geofence",
			Path:     "/geofence",
			Icon:     "map-pin",
			Group:    "Security",
			Priority: 30,
		},
	}
}

// DashboardRenderPage renders the geofence configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/geofence" {
		return nil, contributor.ErrPageNotFound
	}
	return dashui.ConfigPage(
		p.config.DefaultPolicy,
		p.config.AllowedCountries,
		p.config.BlockedCountries,
		p.config.BlockMessage,
	), nil
}
