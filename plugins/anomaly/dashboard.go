package anomaly

import (
	"context"
	"strconv"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/plugins/anomaly/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin          = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// DashboardWidgets returns no widgets for the anomaly plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns a settings panel showing anomaly detection configuration.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return dashui.SettingsPanel(
		strconv.Itoa(p.config.MinLoginHistory),
		strconv.Itoa(p.config.RiskThreshold),
		p.config.EnableTimeAnomaly,
		p.config.EnableGeoAnomaly,
	)
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the anomaly detection page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Anomaly Detection",
			Path:     "/anomaly-detection",
			Icon:     "activity",
			Group:    "Security",
			Priority: 10,
		},
	}
}

// DashboardRenderPage renders the anomaly detection configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/anomaly-detection" {
		return nil, contributor.ErrPageNotFound
	}
	return dashui.ConfigPage(
		strconv.Itoa(p.config.MinLoginHistory),
		strconv.Itoa(p.config.RiskThreshold),
		p.config.EnableTimeAnomaly,
		p.config.EnableGeoAnomaly,
	), nil
}
