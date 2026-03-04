package email

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	emaildash "github.com/xraph/authsome/plugins/email/dashui"
)

// Compile-time interface check.
var _ dashboard.DashboardPlugin = (*Plugin)(nil)

// DashboardWidgets returns no widgets for the email plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns the email plugin settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return emaildash.SettingsPanel(p.config.From, p.config.AppName, p.config.BaseURL)
}

// DashboardPages returns no extra pages for the email plugin.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}
