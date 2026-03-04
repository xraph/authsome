package apikey

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	akdash "github.com/xraph/authsome/plugins/apikey/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin       = (*Plugin)(nil)
	_ dashboard.UserDetailContributor = (*Plugin)(nil)
)

// DashboardWidgets returns API key widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "apikey-count",
			Title:      "API Keys",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(ctx context.Context) templ.Component {
				return akdash.CountWidget()
			},
		},
	}
}

// DashboardSettingsPanel returns the API key settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	var expiryStr string
	if p.config.DefaultExpiry > 0 {
		expiryStr = p.config.DefaultExpiry.String()
	}
	return akdash.SettingsPanel(p.config.MaxKeysPerUser, expiryStr)
}

// DashboardPages returns extra page routes for API keys.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return []dashboard.PluginPage{
		{
			Route: "/api-keys",
			Label: "API Keys",
			Icon:  "key",
			Render: func(ctx context.Context) templ.Component {
				return akdash.KeysPage()
			},
		},
	}
}

// DashboardUserDetailSection returns the user-specific API keys section.
func (p *Plugin) DashboardUserDetailSection(ctx context.Context, userID id.UserID) templ.Component {
	if p.store == nil {
		return nil
	}
	// Use app ID from dashboard context to scope the query.
	appID, _ := dashboard.AppIDFromContext(ctx)
	keys, err := p.store.ListAPIKeysByUser(ctx, appID, userID)
	if err != nil {
		keys = nil
	}

	// Convert to view models.
	views := make([]akdash.APIKeyView, len(keys))
	for i, k := range keys {
		views[i] = akdash.APIKeyView{
			Name:      k.Name,
			KeyPrefix: k.KeyPrefix,
			Revoked:   k.Revoked,
			CreatedAt: k.CreatedAt,
		}
	}
	return akdash.UserSection(views)
}
