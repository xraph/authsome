package sso

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	ssodash "github.com/xraph/authsome/plugins/sso/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin      = (*Plugin)(nil)
	_ dashboard.OrgDetailContributor = (*Plugin)(nil)
)

// DashboardWidgets returns SSO-related widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "sso-connections",
			Title:      "SSO Connections",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(ctx context.Context) templ.Component {
				return ssodash.ConnectionsWidget(p.providerNames())
			},
		},
	}
}

// DashboardSettingsPanel returns the SSO settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return ssodash.SettingsPanel(p.providerNames())
}

// DashboardPages returns extra page routes for SSO.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return []dashboard.PluginPage{
		{
			Route: "/sso-providers",
			Label: "SSO",
			Icon:  "lock",
			Render: func(ctx context.Context) templ.Component {
				return ssodash.ProvidersPage(p.providerNames())
			},
		},
	}
}

// DashboardOrgDetailSection returns the org-specific SSO section.
func (p *Plugin) DashboardOrgDetailSection(ctx context.Context, orgID id.OrgID) templ.Component {
	if p.ssoStore == nil {
		return nil
	}
	// Use app ID from dashboard context, falling back to plugin config.
	appID, ok := dashboard.AppIDFromContext(ctx)
	if !ok {
		appID, _ = id.ParseAppID(p.appID)
	}
	connections, err := p.ssoStore.ListSSOConnections(ctx, appID)
	if err != nil {
		connections = nil
	}

	// Filter connections to only those belonging to this org and convert to view models.
	var views []ssodash.SSOConnectionView
	for _, c := range connections {
		if c.OrgID == orgID {
			views = append(views, ssodash.SSOConnectionView{
				Provider: c.Provider,
				Protocol: c.Protocol,
				Domain:   c.Domain,
				Active:   c.Active,
			})
		}
	}

	return ssodash.OrgSection(views)
}

// providerNames returns the list of configured SSO provider names.
func (p *Plugin) providerNames() []string {
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	return names
}
