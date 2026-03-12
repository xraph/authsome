package vpndetect

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/plugins/vpndetect/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin          = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// DashboardWidgets returns no widgets for the vpndetect plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns a settings panel showing vpndetect configuration.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return dashui.SettingsPanel(
		p.config.BlockVPN,
		p.config.BlockProxy,
		p.config.BlockTor,
		p.config.BlockMessage,
	)
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the VPN detection page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "VPN Detection",
			Path:     "/vpn-detection",
			Icon:     "wifi-off",
			Group:    "Security",
			Priority: 80,
		},
	}
}

// DashboardRenderPage renders the VPN detection configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/vpn-detection" {
		return nil, contributor.ErrPageNotFound
	}
	return dashui.ConfigPage(
		p.config.BlockVPN,
		p.config.BlockProxy,
		p.config.BlockTor,
		p.config.BlockMessage,
	), nil
}
