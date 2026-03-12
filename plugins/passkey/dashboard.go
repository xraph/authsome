package passkey

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	pkdash "github.com/xraph/authsome/plugins/passkey/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin                = (*Plugin)(nil)
	_ dashboard.UserDetailContributor = (*Plugin)(nil)
)

// DashboardWidgets returns passkey-related widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "passkey-adoption",
			Title:      "Passkey Adoption",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(_ context.Context) templ.Component {
				return pkdash.AdoptionWidget()
			},
		},
	}
}

// DashboardSettingsPanel returns the passkey settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return pkdash.SettingsPanel(
		p.config.RPDisplayName,
		p.config.RPID,
		p.config.RPOrigins,
		p.config.SessionTimeout.String(),
	)
}

// DashboardPages returns extra page routes for passkeys.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return []dashboard.PluginPage{
		{
			Route: "/passkeys",
			Label: "Passkeys",
			Icon:  "fingerprint",
			Render: func(_ context.Context) templ.Component {
				return pkdash.CredentialsPage()
			},
		},
	}
}

// DashboardUserDetailSection returns the user-specific passkey section.
func (p *Plugin) DashboardUserDetailSection(ctx context.Context, userID id.UserID) templ.Component {
	if p.store == nil {
		return nil
	}
	credentials, err := p.store.ListUserCredentials(ctx, userID)
	if err != nil {
		credentials = nil
	}

	// Convert to view models.
	views := make([]pkdash.CredentialView, len(credentials))
	for i, c := range credentials {
		views[i] = pkdash.CredentialView{
			DisplayName: c.DisplayName,
			Transport:   c.Transport,
			SignCount:   c.SignCount,
			CreatedAt:   c.CreatedAt,
		}
	}
	return pkdash.UserSection(views)
}
