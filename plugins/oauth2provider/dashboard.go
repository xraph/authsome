package oauth2provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/a-h/templ"

	log "github.com/xraph/go-utils/log"

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

	// Set when the page should render the edit form pre-filled for one
	// client: either the operator asked to edit, or an update was refused and
	// they need the form back to fix it.
	var editClientID string

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
		if err := p.handleDashboardDeleteClient(ctx, appID, params); err != nil {
			data.Error = "Failed to delete OAuth2 client: " + err.Error()
		} else if params.FormData["client_id"] != "" {
			data.Success = "OAuth2 client deleted successfully."
		}
	case "edit":
		// Render-only. The form is populated from editClientID further down,
		// once the client list has been loaded; nothing is written here.
		editClientID = params.FormData["client_id"]
	case "update":
		if errMsg := p.handleDashboardUpdateClient(ctx, appID, params); errMsg != "" {
			data.Error = errMsg
			// Keep the edit form open on the offending client so the
			// operator can correct the value instead of starting over.
			editClientID = params.FormData["client_id"]
		} else {
			data.Success = "OAuth2 client updated successfully."
		}
	}

	// Fetch all clients for the app.
	//
	// Never silently swallow a store error here: a hidden failure renders an
	// empty table that is indistinguishable from a genuine "no clients" state,
	// which masks real problems (DB errors, an app_id scoping mismatch between
	// the row and the resolved app). Log it with the resolved app_id and surface
	// it to the page so the failure is visible instead of looking like an empty
	// list.
	clients, err := p.oauth2Store.ListClients(ctx, appID)
	if err != nil {
		p.logger.Error("oauth2: dashboard failed to list clients",
			log.String("app_id", appID.String()),
			log.Error(err),
		)
		if data.Error == "" {
			data.Error = "Failed to load OAuth2 clients: " + err.Error()
		}
		clients = nil
	} else {
		// Logged so an empty list can be diagnosed: it shows the exact app_id the
		// list is scoped to, which can be compared against the stored rows' app_id.
		p.logger.Debug("oauth2: dashboard listed clients",
			log.String("app_id", appID.String()),
			log.Int("count", len(clients)),
		)
	}

	data.Clients = make([]o2dash.OAuth2ClientView, 0, len(clients))
	for _, c := range clients {
		data.Clients = append(data.Clients, o2dash.OAuth2ClientView{
			ID:           c.ID.String(),
			Name:         c.Name,
			ClientID:     c.ClientID,
			RedirectURIs: c.RedirectURIs,
			Scopes:       c.Scopes,
			Resources:    c.Resources,
			GrantTypes:   c.GrantTypes,
			Public:       c.Public,
			CreatedAt:    c.CreatedAt,
		})
	}

	// Pick the client the edit form should be populated from. Reading it back
	// out of the rendered list rather than the form data means the form always
	// shows what is actually stored, including any field the failed submission
	// did not carry.
	if editClientID != "" {
		for i := range data.Clients {
			if data.Clients[i].ID == editClientID {
				data.EditClient = &data.Clients[i]
				break
			}
		}
	}

	return o2dash.ClientsPage(data), nil
}

// handleDashboardUpdateClient applies an edit to an existing OAuth2 client.
// It returns an empty string on success or a message to show the operator.
//
// This mirrors handleUpdateClient on the admin API, with one difference forced
// by the transport: an HTML form posts every field it renders, so there is no
// "absent means leave alone" to honour here. The form is pre-filled from the
// stored client, so what comes back is the full intended state.
//
// The fields the API refuses to change are simply never read from the form:
// client_id, app_id, public and the secret are not editable here either.
func (p *Plugin) handleDashboardUpdateClient(ctx context.Context, appID id.AppID, params contributor.Params) string {
	rawID := params.FormData["client_id"]
	if rawID == "" {
		return "Client ID is required."
	}
	clientPK, err := id.ParseOAuth2ClientID(rawID)
	if err != nil {
		return "Invalid client ID."
	}

	name := params.FormData["name"]
	if name == "" {
		return "Client name is required."
	}

	var redirectURIs []string
	for _, uri := range strings.Split(params.FormData["redirect_uris"], "\n") {
		if uri = strings.TrimSpace(uri); uri != "" {
			redirectURIs = append(redirectURIs, uri)
		}
	}

	var scopes []string
	for _, s := range strings.FieldsFunc(params.FormData["scopes"], func(r rune) bool { return r == ',' || r == ' ' }) {
		if s = strings.TrimSpace(s); s != "" {
			scopes = append(scopes, s)
		}
	}

	var resources []string
	for _, r := range strings.FieldsFunc(params.FormData["resources"], func(r rune) bool { return r == ',' || r == ' ' }) {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if msg := resourceURISyntaxError(r); msg != "" {
			return "Invalid resource: " + msg
		}
		resources = append(resources, r)
	}

	client, err := p.oauth2Store.GetClientByID(ctx, clientPK)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			return "OAuth2 client not found."
		}
		return "Failed to load OAuth2 client: " + err.Error()
	}
	// Same tenancy rule handleDashboardDeleteClient applies. A client owned by
	// another app is reported as missing rather than forbidden, so the form
	// cannot be used to confirm that some other app's client exists.
	//
	// This is the fourth app-scoping site in the package and the only one that
	// does not call plugin.AssertAppScope, because that wants a forge.Context
	// and the dashboard contributor hands us a plain context.Context. So a
	// grep for AssertAppScope returns three, not four. If you are auditing
	// this package for scoping coverage, count this one by hand.
	if client.AppID != appID {
		return "OAuth2 client not found."
	}

	// A confidential client with no redirect URI can never complete a code
	// flow, the same rule the create path and the admin API enforce.
	if len(redirectURIs) == 0 && !client.Public {
		return "At least one redirect URI is required for a confidential client."
	}

	grantTypes := parseGrantTypeCheckboxes(params.FormData)
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}

	client.Name = name
	client.RedirectURIs = redirectURIs
	client.Scopes = scopes
	client.Resources = resources
	client.GrantTypes = grantTypes

	if err := p.oauth2Store.UpdateClient(ctx, client); err != nil {
		return "Failed to update OAuth2 client: " + err.Error()
	}
	return ""
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
		if uri == "" {
			continue
		}
		// Matches the admin API and the update path: the dashboard must not
		// be the one door that lets a non-loopback http:// URI through.
		if uriErr := validateRedirectURI(uri); uriErr != nil {
			return nil, "Invalid redirect URI: " + registrationErrorDescription(uriErr)
		}
		redirectURIs = append(redirectURIs, uri)
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

	// Parse resources (comma or space separated). Unlike scopes, this has no
	// default: an omitted field stores an empty allowlist, same as the admin
	// API, since a client that never asks for a resource should never be
	// handed one just because the form was left blank.
	resourcesRaw := params.FormData["resources"]
	var resources []string
	for _, r := range strings.FieldsFunc(resourcesRaw, func(r rune) bool { return r == ',' || r == ' ' }) {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if msg := resourceURISyntaxError(r); msg != "" {
			return nil, "Invalid resource: " + msg
		}
		resources = append(resources, r)
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
		ID:                      id.NewOAuth2ClientID(),
		AppID:                   appID,
		Name:                    name,
		ClientID:                clientIDStr,
		ClientSecret:            hashedSecret,
		RedirectURIs:            redirectURIs,
		Scopes:                  scopes,
		Resources:               resources,
		GrantTypes:              grantTypes,
		Public:                  isPublic,
		TokenEndpointAuthMethod: authMethodForPublic(isPublic),
		CreatedAt:               now,
		UpdatedAt:               now,
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
//
// appID is the app the page resolved, and the client has to belong to it. The
// id arrives in a form field, so without this check a caller could post any
// client id at all and have it deleted out of whatever app owned it. Create
// and list on this page were already scoped; delete was the one that was not.
//
// A client owned by another app is reported as ErrClientNotFound, the same
// answer an id that exists nowhere gets, so the rendered error cannot be used
// to tell the two apart.
func (p *Plugin) handleDashboardDeleteClient(ctx context.Context, appID id.AppID, params contributor.Params) error {
	clientIDStr := params.FormData["client_id"]
	if clientIDStr == "" {
		return nil
	}

	clientID, err := id.ParseOAuth2ClientID(clientIDStr)
	if err != nil {
		return fmt.Errorf("invalid client ID: %w", err)
	}

	client, err := p.oauth2Store.GetClientByID(ctx, clientID)
	if err != nil {
		return err
	}
	if client.AppID != appID {
		return ErrClientNotFound
	}

	return p.oauth2Store.DeleteClient(ctx, clientID)
}
