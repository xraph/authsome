package riskengine

import (
	"context"
	"fmt"
	"strconv"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/plugins/riskengine/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin          = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// DashboardWidgets returns no widgets for the riskengine plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns a settings panel showing risk engine configuration.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	// Build weight display strings.
	weights := make([]dashui.WeightEntry, 0, len(p.config.Weights))
	for name, w := range p.config.Weights {
		weights = append(weights, dashui.WeightEntry{
			Name:   name,
			Weight: fmt.Sprintf("%.1f", w),
		})
	}

	return dashui.SettingsPanel(
		strconv.Itoa(p.config.LowThreshold),
		strconv.Itoa(p.config.MediumThreshold),
		strconv.Itoa(p.config.HighThreshold),
		strconv.Itoa(len(p.contributors)),
		weights,
	)
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the risk engine page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Risk Engine",
			Path:     "/risk-engine",
			Icon:     "gauge",
			Group:    "Security",
			Priority: 70,
		},
	}
}

// DashboardRenderPage renders the risk engine configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/risk-engine" {
		return nil, contributor.ErrPageNotFound
	}

	weights := make([]dashui.WeightEntry, 0, len(p.config.Weights))
	for name, w := range p.config.Weights {
		weights = append(weights, dashui.WeightEntry{
			Name:   name,
			Weight: fmt.Sprintf("%.1f", w),
		})
	}

	return dashui.ConfigPage(
		strconv.Itoa(p.config.LowThreshold),
		strconv.Itoa(p.config.MediumThreshold),
		strconv.Itoa(p.config.HighThreshold),
		strconv.Itoa(len(p.contributors)),
		weights,
	), nil
}
