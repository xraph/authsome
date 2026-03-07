package geoip

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/plugins/geoip/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin          = (*Plugin)(nil)
	_ dashboard.DashboardPageContributor = (*Plugin)(nil)
)

// DashboardWidgets returns no widgets for the geoip plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns a settings panel showing geoip configuration.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	providerConfigured := p.provider != nil
	return dashui.SettingsPanel(p.config.DatabasePath, p.config.CacheTTL.String(), providerConfigured)
}

// DashboardPages returns nil — pages are handled via DashboardPageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// DashboardPageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the GeoIP page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "GeoIP",
			Path:     "/geoip",
			Icon:     "globe",
			Group:    "Security",
			Priority: 40,
		},
	}
}

// DashboardRenderPage renders the GeoIP configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/geoip" {
		return nil, contributor.ErrPageNotFound
	}
	providerConfigured := p.provider != nil
	return dashui.ConfigPage(
		p.config.DatabasePath,
		p.config.CacheTTL.String(),
		providerConfigured,
	), nil
}
