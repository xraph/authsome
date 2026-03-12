package phone

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	phonedash "github.com/xraph/authsome/plugins/phone/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin          = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Plugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns no widgets for the phone plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns the phone plugin settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	autoCreate := true
	if p.config.AutoCreate != nil {
		autoCreate = *p.config.AutoCreate
	}
	return phonedash.SettingsPanel(p.config.CodeTTL.String(), autoCreate)
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the phone auth page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Phone Auth",
			Path:     "/phone-auth",
			Icon:     "phone",
			Group:    "Authentication",
			Priority: 60,
		},
	}
}

// DashboardRenderPage renders the phone auth configuration page.
func (p *Plugin) DashboardRenderPage(_ context.Context, route string, _ contributor.Params) (templ.Component, error) {
	if route != "/phone-auth" {
		return nil, contributor.ErrPageNotFound
	}
	autoCreate := true
	if p.config.AutoCreate != nil {
		autoCreate = *p.config.AutoCreate
	}
	return phonedash.ConfigPage(p.config.CodeTTL.String(), autoCreate), nil
}
