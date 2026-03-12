package deviceverify

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/plugins/deviceverify/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin          = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// DashboardWidgets returns no widgets for the deviceverify plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns a settings panel showing deviceverify configuration.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return dashui.SettingsPanel(
		p.config.NotifyOnNewDevice,
		p.config.ChallengeTTL.String(),
	)
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the device verification page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Device Verification",
			Path:     "/device-verification",
			Icon:     "monitor-smartphone",
			Group:    "Security",
			Priority: 20,
		},
	}
}

// DashboardRenderPage renders the device verification configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/device-verification" {
		return nil, contributor.ErrPageNotFound
	}
	return dashui.ConfigPage(
		p.config.NotifyOnNewDevice,
		p.config.ChallengeTTL.String(),
	), nil
}
