package organization

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/organization/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.DashboardPlugin          = (*Plugin)(nil)
	_ dashboard.DashboardPageContributor = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// DashboardPlugin implementation
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

// DashboardPages returns empty since pages are handled via DashboardPageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// DashboardPageContributor implementation
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
		return p.renderOrgList(ctx)
	case "/organizations/detail":
		return p.renderOrgDetail(ctx, params)
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// ──────────────────────────────────────────────────
// Dashboard render helpers
// ──────────────────────────────────────────────────

func (p *Plugin) renderOrgList(ctx context.Context) (templ.Component, error) {
	appID, ok := dashboard.AppIDFromContext(ctx)
	if !ok {
		appID, _ = id.ParseAppID(p.defaultAppID)
	}
	orgs, err := p.AdminListOrganizations(ctx, appID)
	if err != nil {
		orgs = nil
	}

	return dashui.OrganizationsPage(orgs), nil
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

	// Collect plugin-contributed sections for org detail.
	pluginSections := p.collectOrgDetailSections(ctx, orgID)

	return templ.ComponentFunc(func(tCtx context.Context, w io.Writer) error {
		childCtx := templ.WithChildren(tCtx, renderPluginSections(pluginSections))
		return dashui.OrgDetailPage(org, members).Render(childCtx, w)
	}), nil
}

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

// renderPluginSections renders a list of templ components sequentially.
func renderPluginSections(sections []templ.Component) templ.Component {
	if len(sections) == 0 {
		return templ.NopComponent
	}
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		for _, s := range sections {
			if err := s.Render(ctx, w); err != nil {
				return err
			}
		}
		return nil
	})
}
