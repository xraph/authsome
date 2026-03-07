package ipreputation

import (
	"context"
	"strconv"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/plugins/ipreputation/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin          = (*Plugin)(nil)
	_ dashboard.DashboardPageContributor = (*Plugin)(nil)
)

// DashboardWidgets returns no widgets for the ipreputation plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns a settings panel showing ipreputation configuration.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	providerConfigured := p.config.Provider != nil
	return dashui.SettingsPanel(
		strconv.Itoa(p.config.BlockThreshold),
		strconv.Itoa(p.config.WarnThreshold),
		p.config.CacheTTL.String(),
		providerConfigured,
	)
}

// DashboardPages returns nil — pages are handled via DashboardPageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// DashboardPageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the IP reputation page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "IP Reputation",
			Path:     "/ip-reputation",
			Icon:     "shield-alert",
			Group:    "Security",
			Priority: 60,
		},
	}
}

// DashboardRenderPage renders the IP reputation configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/ip-reputation" {
		return nil, contributor.ErrPageNotFound
	}
	providerConfigured := p.config.Provider != nil
	return dashui.ConfigPage(
		strconv.Itoa(p.config.BlockThreshold),
		strconv.Itoa(p.config.WarnThreshold),
		p.config.CacheTTL.String(),
		providerConfigured,
	), nil
}
