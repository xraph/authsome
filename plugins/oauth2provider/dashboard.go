package oauth2provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	o2dash "github.com/xraph/authsome/plugins/oauth2provider/dashui"

	"golang.org/x/crypto/bcrypt"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin          = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Plugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns OAuth2 provider widgets.
func (p *Plugin) DashboardWidgets(_ context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "oauth2-clients",
			Title:      "OAuth2 Clients",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(_ context.Context) templ.Component {
				return o2dash.ClientsWidget()
			},
		},
	}
}

// DashboardSettingsPanel returns the OAuth2 provider settings panel.
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return o2dash.SettingsPanel(
		p.config.Issuer,
		p.config.AuthCodeTTL.String(),
		p.config.AccessTokenTTL.String(),
	)
}

// DashboardPages returns nil — pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "OAuth2 Clients",
			Path:     "/oauth2-clients",
			Icon:     "key-round",
			Group:    "Developer",
			Priority: 10,
		},
	}
}

// DashboardRenderPage renders the OAuth2 clients management page.
func (p *Plugin) DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	if route != "/oauth2-clients" {
		return nil, contributor.ErrPageNotFound
	}
	return p.renderClientsPage(ctx, params)
}

// ──────────────────────────────────────────────────
// Dashboard render helpers
// ──────────────────────────────────────────────────

func (p *Plugin) renderClientsPage(ctx context.Context, params contributor.Params) (templ.Component, error) {
	if p.oauth2Store == nil {
		return o2dash.ClientsPage(o2dash.ClientsPageData{
			Error: "OAuth2 client store is not configured.",
		}), nil
	}

	appID, _ := dashboard.AppIDFromContext(ctx)

	var data o2dash.ClientsPageData

	// Handle form actions (POST).
	action := params.FormData["action"]
	switch action {
	case "create":
		created, errMsg := p.handleDashboardCreateClient(ctx, appID, params)
		if created != nil {
			data.CreatedClient = created
			data.Success = "OAuth2 client created successfully."
		} else {
			data.Error = errMsg
		}
	case "delete":
		if err := p.handleDashboardDeleteClient(ctx, params); err != nil {
			data.Error = "Failed to delete OAuth2 client: " + err.Error()
		} else if params.FormData["client_id"] != "" {
			data.Success = "OAuth2 client deleted successfully."
		}
	}

	// Fetch all clients for the app.
	clients, err := p.oauth2Store.ListClients(ctx, appID)
	if err != nil {
		clients = nil
	}

	data.Clients = make([]o2dash.OAuth2ClientView, 0, len(clients))
	for _, c := range clients {
		data.Clients = append(data.Clients, o2dash.OAuth2ClientView{
			ID:           c.ID.String(),
			Name:         c.Name,
			ClientID:     c.ClientID,
			RedirectURIs: c.RedirectURIs,
			Scopes:       c.Scopes,
			GrantTypes:   c.GrantTypes,
			Public:       c.Public,
			CreatedAt:    c.CreatedAt,
		})
	}

	return o2dash.ClientsPage(data), nil
}

// handleDashboardCreateClient creates a new OAuth2 client from form data.
// Returns the created client view and an empty error message on success, or nil
// and an error message on failure.
func (p *Plugin) handleDashboardCreateClient(ctx context.Context, appID id.AppID, params contributor.Params) (view *o2dash.CreatedClientView, errMsg string) {
	name := params.FormData["name"]
	if name == "" {
		return nil, "Client name is required."
	}

	// Parse redirect URIs from textarea (newline-separated).
	redirectURIsRaw := params.FormData["redirect_uris"]
	var redirectURIs []string
	for _, uri := range strings.Split(redirectURIsRaw, "\n") {
		uri = strings.TrimSpace(uri)
		if uri != "" {
			redirectURIs = append(redirectURIs, uri)
		}
	}

	// Parse scopes (comma or space separated).
	scopesRaw := params.FormData["scopes"]
	var scopes []string
	for _, s := range strings.FieldsFunc(scopesRaw, func(r rune) bool { return r == ',' || r == ' ' }) {
		s = strings.TrimSpace(s)
		if s != "" {
			scopes = append(scopes, s)
		}
	}
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}

	// Parse grant types from individual checkboxes.
	grantTypes := parseGrantTypeCheckboxes(params.FormData)
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}

	// Check if public client.
	isPublic := params.FormData["public"] == "on"

	// Generate client credentials.
	clientIDStr, err := generateSecureToken(16)
	if err != nil {
		return nil, "Failed to generate client ID."
	}

	var rawSecret string
	var hashedSecret string
	if !isPublic {
		rawSecret, err = generateSecureToken(32)
		if err != nil {
			return nil, "Failed to generate client secret."
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
		if err != nil {
			return nil, "Failed to hash client secret."
		}
		hashedSecret = string(hash)
	}

	now := time.Now()
	client := &OAuth2Client{
		ID:           id.NewOAuth2ClientID(),
		AppID:        appID,
		Name:         name,
		ClientID:     clientIDStr,
		ClientSecret: hashedSecret,
		RedirectURIs: redirectURIs,
		Scopes:       scopes,
		GrantTypes:   grantTypes,
		Public:       isPublic,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := p.oauth2Store.CreateClient(ctx, client); err != nil {
		return nil, "Failed to create OAuth2 client: " + err.Error()
	}

	return &o2dash.CreatedClientView{
		Name:         name,
		ClientID:     clientIDStr,
		ClientSecret: rawSecret,
		Public:       isPublic,
	}, ""
}

// parseGrantTypeCheckboxes reads individual grant type checkboxes from form data.
func parseGrantTypeCheckboxes(formData map[string]string) []string {
	var grantTypes []string
	if formData["grant_authorization_code"] == "on" {
		grantTypes = append(grantTypes, "authorization_code")
	}
	if formData["grant_client_credentials"] == "on" {
		grantTypes = append(grantTypes, "client_credentials")
	}
	if formData["grant_device_code"] == "on" {
		grantTypes = append(grantTypes, "urn:ietf:params:oauth:grant-type:device_code")
	}
	if formData["grant_refresh_token"] == "on" {
		grantTypes = append(grantTypes, "refresh_token")
	}
	return grantTypes
}

// handleDashboardDeleteClient deletes an OAuth2 client from form data.
func (p *Plugin) handleDashboardDeleteClient(ctx context.Context, params contributor.Params) error {
	clientIDStr := params.FormData["client_id"]
	if clientIDStr == "" {
		return nil
	}

	clientID, err := id.ParseOAuth2ClientID(clientIDStr)
	if err != nil {
		return fmt.Errorf("invalid client ID: %w", err)
	}

	return p.oauth2Store.DeleteClient(ctx, clientID)
}
