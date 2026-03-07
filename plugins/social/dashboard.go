package social

import (
	"context"
	"fmt"
	"strings"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	socialdash "github.com/xraph/authsome/plugins/social/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin          = (*Plugin)(nil)
	_ dashboard.DashboardPageContributor = (*Plugin)(nil)
	_ dashboard.UserDetailContributor    = (*Plugin)(nil)
)

// supportedProviders is the full list of providers that can be configured.
var supportedProviders = []string{
	"google", "github", "apple", "microsoft",
	"facebook", "linkedin", "discord", "slack",
	"twitter", "spotify", "twitch", "gitlab",
	"bitbucket", "dropbox", "yahoo", "amazon",
	"zoom", "pinterest", "strava", "patreon",
	"instagram", "line",
}

// ──────────────────────────────────────────────────
// DashboardPlugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns social auth widgets.
func (p *Plugin) DashboardWidgets(ctx context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "social-connections",
			Title:      "Social Connections",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(wCtx context.Context) templ.Component {
				return socialdash.ConnectionsWidget(p.allProviderNames(wCtx))
			},
		},
	}
}

// DashboardSettingsPanel returns the social plugin settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return socialdash.SettingsPanel(p.providerNames())
}

// DashboardPages returns nil since pages are handled via DashboardPageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// DashboardPageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the social login pages.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Social Login",
			Path:     "/social-providers",
			Icon:     "users",
			Group:    "Authentication",
			Priority: 3,
		},
	}
}

// DashboardRenderPage renders a page for the given route with params.
func (p *Plugin) DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	switch route {
	case "/social-providers":
		return p.renderProvidersGrid(ctx, params)
	case "/social-providers/detail":
		return p.renderProviderDetail(ctx, params)
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// ──────────────────────────────────────────────────
// Grid page
// ──────────────────────────────────────────────────

func (p *Plugin) renderProvidersGrid(ctx context.Context, _ contributor.Params) (templ.Component, error) {
	var data socialdash.ProvidersGridData

	// Build maps for quick lookups.
	codeProviders := make(map[string]struct{})
	for name := range p.providers {
		codeProviders[name] = struct{}{}
	}

	dbProviders := p.loadDBProviderSettings(ctx)
	dbMap := make(map[string]SocialProviderSetting)
	for _, s := range dbProviders {
		dbMap[s.Name] = s
	}

	// Build the unified provider list.
	for _, name := range supportedProviders {
		prov := socialdash.AvailableProvider{Name: name}

		if _, isCode := codeProviders[name]; isCode {
			prov.Status = socialdash.StatusActive
			prov.Source = "code"
		} else if dbSetting, isDB := dbMap[name]; isDB {
			prov.Source = "dashboard"
			if dbSetting.Enabled {
				prov.Status = socialdash.StatusEnabled
			} else {
				prov.Status = socialdash.StatusDisabled
			}
		} else {
			prov.Status = socialdash.StatusNotConfigured
			prov.Source = ""
		}

		data.Providers = append(data.Providers, prov)
	}

	return socialdash.ProvidersPage(data), nil
}

// ──────────────────────────────────────────────────
// Detail page
// ──────────────────────────────────────────────────

func (p *Plugin) renderProviderDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	providerName := params.QueryParams["provider"]
	if providerName == "" {
		return nil, contributor.ErrPageNotFound
	}
	providerName = strings.ToLower(providerName)

	// Validate provider name is in supported list.
	if !isSupportedProvider(providerName) {
		return nil, contributor.ErrPageNotFound
	}

	var data socialdash.ProviderDetailData
	data.Name = providerName

	// Handle form actions (POST via HTMX).
	action := params.FormData["action"]
	switch action {
	case "add_provider":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			data.Error = p.handleAddProvider(ctx, providerName, params)
			if data.Error == "" {
				data.Success = "Provider configured successfully."
			}
		}
	case "update_provider":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			data.Error = p.handleUpdateProvider(ctx, providerName, params)
			if data.Error == "" {
				data.Success = "Provider updated successfully."
			}
		}
	case "toggle_provider":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			data.Error = p.handleToggleProvider(ctx, params)
			if data.Error == "" {
				data.Success = "Provider status updated."
			}
		}
	case "remove_provider":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			data.Error = p.handleRemoveProvider(ctx, params)
			if data.Error == "" {
				data.Success = "Provider removed."
			}
		}
	}

	// Generate a fresh nonce for the form.
	data.FormNonce = dashboard.GenerateNonce()

	// Determine current provider state.
	if _, isCode := p.providers[providerName]; isCode {
		data.Status = socialdash.StatusActive
		data.Source = "code"
		data.IsReadOnly = true
	} else {
		dbProviders := p.loadDBProviderSettings(ctx)
		for _, s := range dbProviders {
			if s.Name == providerName {
				data.Source = "dashboard"
				data.ClientID = maskSecret(s.ClientID)
				data.RedirectURL = s.RedirectURL
				data.Scopes = strings.Join(s.Scopes, ", ")
				if s.Enabled {
					data.Status = socialdash.StatusEnabled
				} else {
					data.Status = socialdash.StatusDisabled
				}
				break
			}
		}
		if data.Source == "" {
			data.Status = socialdash.StatusNotConfigured
		}
	}

	return socialdash.ProviderDetailPage(data), nil
}

// ──────────────────────────────────────────────────
// Form action handlers
// ──────────────────────────────────────────────────

// handleAddProvider adds a new provider to DB settings from the detail page.
func (p *Plugin) handleAddProvider(ctx context.Context, providerName string, params contributor.Params) string {
	clientID := strings.TrimSpace(params.FormData["client_id"])
	clientSecret := strings.TrimSpace(params.FormData["client_secret"])
	redirectURL := strings.TrimSpace(params.FormData["redirect_url"])
	scopesStr := strings.TrimSpace(params.FormData["scopes"])

	if clientID == "" || clientSecret == "" {
		return "Client ID and Client Secret are required."
	}

	// Reject if provider is code-configured.
	if _, isCode := p.providers[providerName]; isCode {
		return fmt.Sprintf("Provider %q is configured in code and cannot be managed from the dashboard.", providerName)
	}

	// Check for duplicates in DB settings.
	dbProviders := p.loadDBProviderSettings(ctx)
	for _, s := range dbProviders {
		if s.Name == providerName {
			return fmt.Sprintf("Provider %q is already configured.", providerName)
		}
	}

	var scopes []string
	if scopesStr != "" {
		for _, s := range strings.Split(scopesStr, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				scopes = append(scopes, s)
			}
		}
	}

	dbProviders = append(dbProviders, SocialProviderSetting{
		Name:         providerName,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Enabled:      true,
	})

	if err := p.saveDBProviderSettings(ctx, dbProviders); err != nil {
		return fmt.Sprintf("Failed to save provider: %v", err)
	}
	return ""
}

// handleUpdateProvider updates credentials for an existing DB provider.
func (p *Plugin) handleUpdateProvider(ctx context.Context, providerName string, params contributor.Params) string {
	clientID := strings.TrimSpace(params.FormData["client_id"])
	clientSecret := strings.TrimSpace(params.FormData["client_secret"])
	redirectURL := strings.TrimSpace(params.FormData["redirect_url"])
	scopesStr := strings.TrimSpace(params.FormData["scopes"])

	dbProviders := p.loadDBProviderSettings(ctx)
	found := false
	for i, s := range dbProviders {
		if s.Name == providerName {
			if clientID != "" {
				dbProviders[i].ClientID = clientID
			}
			if clientSecret != "" {
				dbProviders[i].ClientSecret = clientSecret
			}
			dbProviders[i].RedirectURL = redirectURL

			var scopes []string
			if scopesStr != "" {
				for _, sc := range strings.Split(scopesStr, ",") {
					sc = strings.TrimSpace(sc)
					if sc != "" {
						scopes = append(scopes, sc)
					}
				}
			}
			dbProviders[i].Scopes = scopes
			found = true
			break
		}
	}

	if !found {
		return fmt.Sprintf("Provider %q not found.", providerName)
	}

	if err := p.saveDBProviderSettings(ctx, dbProviders); err != nil {
		return fmt.Sprintf("Failed to update provider: %v", err)
	}
	return ""
}

// handleRemoveProvider removes a provider from DB settings.
func (p *Plugin) handleRemoveProvider(ctx context.Context, params contributor.Params) string {
	providerName := strings.TrimSpace(params.FormData["provider_name"])
	if providerName == "" {
		return "Provider name is required."
	}

	dbProviders := p.loadDBProviderSettings(ctx)
	filtered := make([]SocialProviderSetting, 0, len(dbProviders))
	found := false
	for _, s := range dbProviders {
		if s.Name == providerName {
			found = true
			continue
		}
		filtered = append(filtered, s)
	}

	if !found {
		return fmt.Sprintf("Provider %q not found in dashboard settings.", providerName)
	}

	if err := p.saveDBProviderSettings(ctx, filtered); err != nil {
		return fmt.Sprintf("Failed to remove provider: %v", err)
	}
	return ""
}

// handleToggleProvider toggles the Enabled flag on a DB provider.
func (p *Plugin) handleToggleProvider(ctx context.Context, params contributor.Params) string {
	providerName := strings.TrimSpace(params.FormData["provider_name"])
	if providerName == "" {
		return "Provider name is required."
	}

	dbProviders := p.loadDBProviderSettings(ctx)
	found := false
	for i, s := range dbProviders {
		if s.Name == providerName {
			dbProviders[i].Enabled = !s.Enabled
			found = true
			break
		}
	}

	if !found {
		return fmt.Sprintf("Provider %q not found in dashboard settings.", providerName)
	}

	if err := p.saveDBProviderSettings(ctx, dbProviders); err != nil {
		return fmt.Sprintf("Failed to update provider: %v", err)
	}
	return ""
}

// ──────────────────────────────────────────────────
// UserDetailContributor implementation
// ──────────────────────────────────────────────────

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

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// providerNames returns a sorted list of code-configured provider names.
func (p *Plugin) providerNames() []string {
	names := make([]string, 0, len(p.providers))
	for name := range p.providers {
		names = append(names, name)
	}
	return names
}

// maskSecret masks all but the first 4 characters of a secret.
func maskSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + strings.Repeat("*", len(s)-4)
}

// isSupportedProvider checks if a provider name is in the supported list.
func isSupportedProvider(name string) bool {
	for _, p := range supportedProviders {
		if p == name {
			return true
		}
	}
	return false
}
