package organization

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugins/organization/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin          = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Plugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns widgets this plugin contributes.
func (p *Plugin) DashboardWidgets(ctx context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "org-count",
			Title:      "Organizations",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(wCtx context.Context) templ.Component {
				appID, ok := dashboard.AppIDFromContext(wCtx)
				if !ok {
					appID, _ = id.ParseAppID(p.defaultAppID)
				}
				orgs, err := p.AdminListOrganizations(wCtx, appID)
				if err != nil {
					orgs = nil
				}
				return dashui.OrgCountWidget(len(orgs))
			},
		},
	}
}

// DashboardSettingsPanel returns nil (no settings panel for org plugin).
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return nil
}

// DashboardPages returns empty since pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the organization pages.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Organizations",
			Path:     "/organizations",
			Icon:     "building-2",
			Group:    "Authentication",
			Priority: 2,
		},
	}
}

// DashboardRenderPage renders a page for the given route with params.
func (p *Plugin) DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	switch route {
	case "/organizations":
		return p.renderOrgList(ctx, params)
	case "/organizations/create":
		return p.renderOrgCreate(ctx, params)
	case "/organizations/detail":
		return p.renderOrgDetail(ctx, params)
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// ──────────────────────────────────────────────────
// Dashboard render helpers
// ──────────────────────────────────────────────────

func (p *Plugin) renderOrgList(ctx context.Context, _ contributor.Params) (templ.Component, error) {
	appID, ok := dashboard.AppIDFromContext(ctx)
	if !ok {
		appID, _ = id.ParseAppID(p.defaultAppID)
	}

	orgs, err := p.AdminListOrganizations(ctx, appID)
	if err != nil {
		orgs = nil
	}

	// Compute aggregate stats across all orgs.
	data := dashui.OrgsPageData{Orgs: orgs}
	for _, org := range orgs {
		if members, err := p.ListMembers(ctx, org.ID); err == nil {
			data.TotalMembers += len(members)
		}
		if teams, err := p.ListTeams(ctx, org.ID); err == nil {
			data.TotalTeams += len(teams)
		}
		if invitations, err := p.ListInvitations(ctx, org.ID); err == nil {
			data.TotalInvitations += len(invitations)
		}
	}

	return dashui.OrganizationsPage(data), nil
}

func (p *Plugin) renderOrgCreate(ctx context.Context, params contributor.Params) (templ.Component, error) {
	appID, ok := dashboard.AppIDFromContext(ctx)
	if !ok {
		appID, _ = id.ParseAppID(p.defaultAppID)
	}

	var data dashui.CreateOrgPageData

	// Handle form actions (POST).
	action := params.FormData["action"]
	if action == "create" {
		nonce := params.FormData["nonce"]
		if dashboard.ConsumeNonce(nonce) {
			data.CreatedOrg, data.Error = p.handleDashboardCreateOrg(ctx, appID, params)
		}
	}

	// Generate a fresh nonce for the next form render.
	data.FormNonce = dashboard.GenerateNonce()

	// Collect plugin-contributed form fields.
	data.PluginFields = p.collectOrgCreateFormFields(ctx)

	return dashui.CreateOrganizationPage(data), nil
}

// handleDashboardCreateOrg creates a new organization from form data.
func (p *Plugin) handleDashboardCreateOrg(ctx context.Context, appID id.AppID, params contributor.Params) (*organization.Organization, string) {
	name := strings.TrimSpace(params.FormData["name"])
	slug := strings.TrimSpace(params.FormData["slug"])
	logo := strings.TrimSpace(params.FormData["logo"])

	if name == "" || slug == "" {
		return nil, "Name and slug are required."
	}

	// Check slug availability.
	available, err := p.IsOrgSlugAvailable(ctx, appID, slug)
	if err != nil {
		return nil, fmt.Sprintf("Failed to check slug availability: %v", err)
	}
	if !available {
		return nil, fmt.Sprintf("Slug %q is already in use. Please choose a different one.", slug)
	}

	org := &organization.Organization{
		ID:    id.NewOrgID(),
		AppID: appID,
		Name:  name,
		Slug:  slug,
		Logo:  logo,
	}

	if err := p.CreateOrganization(ctx, org); err != nil {
		return nil, fmt.Sprintf("Failed to create organization: %v", err)
	}

	return org, ""
}

func (p *Plugin) renderOrgDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	orgIDStr := params.PathParams["id"]
	if orgIDStr == "" {
		orgIDStr = params.QueryParams["id"]
	}
	if orgIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}

	orgID, err := id.ParseOrgID(orgIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}

	org, err := p.GetOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("organization dashboard: resolve organization: %w", err)
	}

	members, err := p.ListMembers(ctx, orgID)
	if err != nil {
		members = nil
	}

	teams, err := p.ListTeams(ctx, orgID)
	if err != nil {
		teams = nil
	}

	invitations, err := p.ListInvitations(ctx, orgID)
	if err != nil {
		invitations = nil
	}

	// Collect legacy plugin-contributed sections (rendered in Overview tab).
	pluginSections := p.collectOrgDetailSections(ctx, orgID)

	// Collect plugin-contributed tabs.
	pluginTabs := p.collectOrgDetailTabs(ctx, orgID)

	// Determine active tab from query param.
	activeTab := params.QueryParams["tab"]
	if activeTab == "" {
		activeTab = "overview"
	}

	data := dashui.OrgDetailPageData{
		Org:            org,
		Members:        members,
		Teams:          teams,
		Invitations:    invitations,
		PluginSections: pluginSections,
		PluginTabs:     pluginTabs,
		ActiveTab:      activeTab,
	}

	return dashui.OrgDetailPage(data), nil
}

// ──────────────────────────────────────────────────
// Plugin contribution collectors
// ──────────────────────────────────────────────────

// collectOrgDetailSections gathers org detail sections from plugins implementing OrgDetailContributor.
func (p *Plugin) collectOrgDetailSections(ctx context.Context, orgID id.OrgID) []templ.Component {
	if p.plugins == nil {
		return nil
	}
	var sections []templ.Component
	for _, pl := range p.plugins.Plugins() {
		if odc, ok := pl.(dashboard.OrgDetailContributor); ok {
			if section := odc.DashboardOrgDetailSection(ctx, orgID); section != nil {
				sections = append(sections, section)
			}
		}
	}
	return sections
}

// collectOrgDetailTabs gathers tabs from plugins implementing OrgDetailTabContributor.
func (p *Plugin) collectOrgDetailTabs(ctx context.Context, orgID id.OrgID) []dashui.OrgDetailTabView {
	if p.plugins == nil {
		return nil
	}
	var raw []dashboard.OrgDetailTab
	for _, pl := range p.plugins.Plugins() {
		if tc, ok := pl.(dashboard.OrgDetailTabContributor); ok {
			raw = append(raw, tc.DashboardOrgDetailTabs(ctx, orgID)...)
		}
	}
	if len(raw) == 0 {
		return nil
	}

	// Sort by priority.
	sort.Slice(raw, func(i, j int) bool {
		return raw[i].Priority < raw[j].Priority
	})

	// Pre-render tab content into views.
	views := make([]dashui.OrgDetailTabView, 0, len(raw))
	for _, tab := range raw {
		views = append(views, dashui.OrgDetailTabView{
			ID:      tab.ID,
			Label:   tab.Label,
			Icon:    tab.Icon,
			Content: tab.Render(ctx, orgID),
		})
	}
	return views
}

// collectOrgCreateFormFields gathers form fields from plugins implementing OrgCreateFormContributor.
func (p *Plugin) collectOrgCreateFormFields(ctx context.Context) []templ.Component {
	if p.plugins == nil {
		return nil
	}
	var fields []templ.Component
	for _, pl := range p.plugins.Plugins() {
		if fc, ok := pl.(dashboard.OrgCreateFormContributor); ok {
			if field := fc.DashboardOrgCreateFormFields(ctx); field != nil {
				fields = append(fields, field)
			}
		}
	}
	return fields
}
