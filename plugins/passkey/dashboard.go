package passkey

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	pkdash "github.com/xraph/authsome/plugins/passkey/dashui"
	"github.com/xraph/authsome/settings"
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

// DashboardSettingsPanel returns the passkey settings summary panel. Values
// are resolved from the dynamic settings store (cascading global → app), with
// the static Config{} as fallback when the settings manager is unavailable.
func (p *Plugin) DashboardSettingsPanel(ctx context.Context) templ.Component {
	data := pkdash.SettingsPanelData{
		RPDisplayName:  p.config.RPDisplayName,
		RPID:           p.config.RPID,
		RPOrigins:      p.config.RPOrigins,
		SessionTimeout: int(p.config.SessionTimeout.Seconds()),
	}

	if p.settingsMgr != nil {
		opts := settings.ResolveOpts{}
		if v, err := settings.Get(ctx, p.settingsMgr, SettingRPDisplayName, opts); err == nil && v != "" {
			data.RPDisplayName = v
		}
		if v, err := settings.Get(ctx, p.settingsMgr, SettingRPID, opts); err == nil && v != "" {
			data.RPID = v
		}
		if v, err := settings.Get(ctx, p.settingsMgr, SettingRPOrigins, opts); err == nil && len(v) > 0 {
			data.RPOrigins = v
		}
		if v, err := settings.Get(ctx, p.settingsMgr, SettingSessionTimeoutSeconds, opts); err == nil && v > 0 {
			data.SessionTimeout = v
		}
	}

	return pkdash.SettingsPanel(data)
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
