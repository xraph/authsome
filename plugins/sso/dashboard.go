package sso

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	ssodash "github.com/xraph/authsome/plugins/sso/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin          = (*Plugin)(nil)
	_ dashboard.DashboardPageContributor = (*Plugin)(nil)
	_ dashboard.OrgDetailContributor     = (*Plugin)(nil)
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

// DashboardPages returns nil — pages are handled via DashboardPageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// DashboardPageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the SSO providers page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "SSO",
			Path:     "/sso-providers",
			Icon:     "lock",
			Group:    "Security",
			Priority: 0,
		},
	}
}

// DashboardRenderPage renders the SSO providers page with form handling.
func (p *Plugin) DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	if route != "/sso-providers" {
		return nil, contributor.ErrPageNotFound
	}
	return p.renderProvidersPage(ctx, params)
}

// ──────────────────────────────────────────────────
// Dashboard render helpers
// ──────────────────────────────────────────────────

func (p *Plugin) renderProvidersPage(ctx context.Context, params contributor.Params) (templ.Component, error) {
	appID, ok := dashboard.AppIDFromContext(ctx)
	if !ok {
		appID, _ = id.ParseAppID(p.appID)
	}

	var data ssodash.ProvidersPageData
	data.CodeProviders = p.providerNames()

	// Handle form actions (POST).
	action := params.FormData["action"]
	switch action {
	case "add":
		nonce := params.FormData["nonce"]
		if !dashboard.ConsumeNonce(nonce) {
			break
		}
		data.Error = p.handleAddConnection(ctx, appID, params)
		if data.Error == "" {
			data.Success = "SSO connection added successfully."
		}
	case "toggle":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			p.handleToggleConnection(ctx, params)
		}
	case "delete":
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			p.handleDeleteConnection(ctx, params)
			data.Success = "SSO connection deleted."
		}
	}

	// Generate a fresh nonce for the next form render.
	data.FormNonce = dashboard.GenerateNonce()

	// Load DB-managed connections.
	if p.ssoStore != nil {
		connections, err := p.ssoStore.ListSSOConnections(ctx, appID)
		if err != nil {
			connections = nil
		}
		data.DBConnections = make([]ssodash.DBConnectionView, 0, len(connections))
		for _, c := range connections {
			masked := c.ClientID
			if len(masked) > 8 {
				masked = masked[:4] + "..." + masked[len(masked)-4:]
			}
			data.DBConnections = append(data.DBConnections, ssodash.DBConnectionView{
				ID:          c.ID.String(),
				Provider:    c.Provider,
				Protocol:    c.Protocol,
				Domain:      c.Domain,
				ClientID:    masked,
				Issuer:      c.Issuer,
				MetadataURL: c.MetadataURL,
				Active:      c.Active,
			})
		}
	}

	return ssodash.ProvidersPage(data), nil
}

// handleAddConnection creates a new SSO connection from form data.
func (p *Plugin) handleAddConnection(ctx context.Context, appID id.AppID, params contributor.Params) string {
	if p.ssoStore == nil {
		return "SSO store is not configured."
	}

	protocol := strings.TrimSpace(params.FormData["protocol"])
	provider := strings.TrimSpace(params.FormData["provider"])
	domain := strings.TrimSpace(params.FormData["domain"])

	if protocol == "" || provider == "" || domain == "" {
		return "Protocol, provider name, and domain are required."
	}
	if protocol != "oidc" && protocol != "saml" {
		return "Protocol must be 'oidc' or 'saml'."
	}

	now := time.Now()
	conn := &SSOConnection{
		ID:        id.NewSSOConnectionID(),
		AppID:     appID,
		Provider:  provider,
		Protocol:  protocol,
		Domain:    domain,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if protocol == "oidc" {
		conn.ClientID = strings.TrimSpace(params.FormData["client_id"])
		conn.ClientSecret = strings.TrimSpace(params.FormData["client_secret"])
		conn.Issuer = strings.TrimSpace(params.FormData["issuer"])
		if conn.ClientID == "" || conn.Issuer == "" {
			return "OIDC connections require Client ID and Issuer."
		}
	} else {
		conn.MetadataURL = strings.TrimSpace(params.FormData["metadata_url"])
		if conn.MetadataURL == "" {
			return "SAML connections require a Metadata URL."
		}
	}

	if err := p.ssoStore.CreateSSOConnection(ctx, conn); err != nil {
		return fmt.Sprintf("Failed to create connection: %v", err)
	}

	return ""
}

// handleToggleConnection toggles a connection's Active status.
func (p *Plugin) handleToggleConnection(ctx context.Context, params contributor.Params) {
	if p.ssoStore == nil {
		return
	}
	connIDStr := params.FormData["conn_id"]
	if connIDStr == "" {
		return
	}
	connID, err := id.ParseSSOConnectionID(connIDStr)
	if err != nil {
		return
	}
	conn, err := p.ssoStore.GetSSOConnection(ctx, connID)
	if err != nil {
		return
	}
	conn.Active = !conn.Active
	conn.UpdatedAt = time.Now()
	_ = p.ssoStore.UpdateSSOConnection(ctx, conn)
}

// handleDeleteConnection deletes a connection.
func (p *Plugin) handleDeleteConnection(ctx context.Context, params contributor.Params) {
	if p.ssoStore == nil {
		return
	}
	connIDStr := params.FormData["conn_id"]
	if connIDStr == "" {
		return
	}
	connID, err := id.ParseSSOConnectionID(connIDStr)
	if err != nil {
		return
	}
	_ = p.ssoStore.DeleteSSOConnection(ctx, connID)
}

// ──────────────────────────────────────────────────
// OrgDetailContributor
// ──────────────────────────────────────────────────

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
