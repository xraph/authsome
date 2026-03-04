package password

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	pwdash "github.com/xraph/authsome/plugins/password/dashui"
)

// Compile-time interface check.
var _ dashboard.DashboardPlugin = (*Plugin)(nil)

// DashboardWidgets returns no widgets for the password plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns the password plugin settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return pwdash.SettingsPanel(p.config.MinLength, p.config.RequireSpecial, p.config.AllowedDomains)
}

// DashboardPages returns no extra pages for the password plugin.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}
