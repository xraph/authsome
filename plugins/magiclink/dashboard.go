package magiclink

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	mldash "github.com/xraph/authsome/plugins/magiclink/dashui"
)

// Compile-time interface check.
var _ dashboard.DashboardPlugin = (*Plugin)(nil)

// DashboardWidgets returns no widgets for the magic link plugin.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return nil
}

// DashboardSettingsPanel returns the magic link plugin settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return mldash.SettingsPanel(
		p.config.TokenTTL.String(),
		p.config.SessionTokenTTL.String(),
		p.config.SessionRefreshTTL.String(),
	)
}

// DashboardPages returns no extra pages for the magic link plugin.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}
