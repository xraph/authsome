package impossibletravel

import (
	"context"
	"fmt"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/plugins/impossibletravel/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin          = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// DashboardWidgets returns no widgets for the impossibletravel plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns a settings panel showing impossibletravel configuration.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return dashui.SettingsPanel(
		fmt.Sprintf("%.0f km/h", p.config.MaxSpeedKmH),
		fmt.Sprintf("%.0f km", p.config.MinDistanceKm),
		p.config.LookbackWindow.String(),
		p.config.Action,
	)
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the impossible travel page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Impossible Travel",
			Path:     "/impossible-travel",
			Icon:     "plane",
			Group:    "Security",
			Priority: 50,
		},
	}
}

// DashboardRenderPage renders the impossible travel configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/impossible-travel" {
		return nil, contributor.ErrPageNotFound
	}
	return dashui.ConfigPage(
		fmt.Sprintf("%.0f km/h", p.config.MaxSpeedKmH),
		fmt.Sprintf("%.0f km", p.config.MinDistanceKm),
		p.config.LookbackWindow.String(),
		p.config.Action,
	), nil
}
