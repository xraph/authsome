package social

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	socialdash "github.com/xraph/authsome/plugins/social/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin       = (*Plugin)(nil)
	_ dashboard.UserDetailContributor = (*Plugin)(nil)
)

// DashboardWidgets returns social auth widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "social-connections",
			Title:      "Social Connections",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(ctx context.Context) templ.Component {
				return socialdash.ConnectionsWidget(p.providerNames())
			},
		},
	}
}

// DashboardSettingsPanel returns the social plugin settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return socialdash.SettingsPanel(p.providerNames())
}

// DashboardPages returns extra page routes for social auth.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return []dashboard.PluginPage{
		{
			Route: "/social-providers",
			Label: "Social Login",
			Icon:  "users",
			Render: func(ctx context.Context) templ.Component {
				return socialdash.ProvidersPage(p.providerNames())
			},
		},
	}
}

// DashboardUserDetailSection returns the user-specific social connections section.
func (p *Plugin) DashboardUserDetailSection(ctx context.Context, userID id.UserID) templ.Component {
	if p.oauthStore == nil {
		return nil
	}
	connections, err := p.oauthStore.GetOAuthConnectionsByUserID(ctx, userID)
	if err != nil {
		connections = nil
	}

	// Convert to view models.
	views := make([]socialdash.ConnectionView, len(connections))
	for i, c := range connections {
		views[i] = socialdash.ConnectionView{
			Provider:  c.Provider,
			Email:     c.Email,
			CreatedAt: c.CreatedAt,
		}
	}
	return socialdash.UserSection(views)
}

// providerNames returns a sorted list of configured provider names.
func (p *Plugin) providerNames() []string {
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	return names
}
