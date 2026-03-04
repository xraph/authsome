package consent

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	consentdash "github.com/xraph/authsome/plugins/consent/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin       = (*Plugin)(nil)
	_ dashboard.UserDetailContributor = (*Plugin)(nil)
)

// DashboardWidgets returns consent-related widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "consent-overview",
			Title:      "Consent",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(ctx context.Context) templ.Component {
				return consentdash.OverviewWidget()
			},
		},
	}
}

// DashboardSettingsPanel returns the consent plugin settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return consentdash.SettingsPanel()
}

// DashboardPages returns extra page routes for consent management.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return []dashboard.PluginPage{
		{
			Route: "/consents",
			Label: "Consent",
			Icon:  "shield-check",
			Render: func(ctx context.Context) templ.Component {
				return consentdash.ConsentsPage()
			},
		},
	}
}

// DashboardUserDetailSection returns the user-specific consent section.
func (p *Plugin) DashboardUserDetailSection(ctx context.Context, userID id.UserID) templ.Component {
	if p.store == nil {
		return nil
	}
	// Use app ID from dashboard context to scope the query.
	appID, _ := dashboard.AppIDFromContext(ctx)
	consents, _, err := p.store.ListConsents(ctx, &Query{
		UserID: userID,
		AppID:  appID,
		Limit:  100,
	})
	if err != nil {
		consents = nil
	}

	views := make([]consentdash.ConsentView, len(consents))
	for i, c := range consents {
		views[i] = consentdash.ConsentView{
			Purpose:   c.Purpose,
			Granted:   c.Granted,
			Version:   c.Version,
			GrantedAt: c.GrantedAt,
			RevokedAt: c.RevokedAt,
		}
	}
	return consentdash.UserSection(views)
}
