package waitlist

import (
	"context"
	"fmt"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/dashboard"
	"github.com/xraph/authsome/id"
	wldash "github.com/xraph/authsome/plugins/waitlist/dashui"
)

// Compile-time interface checks.
var (
	_ dashboard.Plugin        = (*Plugin)(nil)
	_ dashboard.PageContributor = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Plugin implementation
// ──────────────────────────────────────────────────

// DashboardWidgets returns waitlist-related widgets.
func (p *Plugin) DashboardWidgets(ctx context.Context) []dashboard.PluginWidget {
	return []dashboard.PluginWidget{
		{
			ID:         "waitlist-pending",
			Title:      "Waitlist",
			Size:       "sm",
			RefreshSec: 60,
			Render: func(wCtx context.Context) templ.Component {
				pending := p.pendingCount(wCtx)
				return wldash.PendingCountWidget(pending)
			},
		},
	}
}

// DashboardSettingsPanel returns nil since the waitlist plugin has no
// dedicated settings panel (it is configured via code).
func (p *Plugin) DashboardSettingsPanel(_ context.Context) templ.Component {
	return nil
}

// DashboardPages returns nil since pages are handled via PageContributor.
func (p *Plugin) DashboardPages() []dashboard.PluginPage {
	return nil
}

// ──────────────────────────────────────────────────
// PageContributor implementation
// ──────────────────────────────────────────────────

// DashboardNavItems returns navigation items for the waitlist page.
func (p *Plugin) DashboardNavItems() []contributor.NavItem {
	return []contributor.NavItem{
		{
			Label:    "Waitlist",
			Path:     "/waitlist",
			Icon:     "clock",
			Group:    "Authentication",
			Priority: 5,
		},
	}
}

// DashboardRenderPage renders a page for the given route with params.
func (p *Plugin) DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	switch {
	case route == "/waitlist":
		return p.renderWaitlistPage(ctx, params)
	case isApproveRoute(route):
		return p.handleApproveAction(ctx, route, params)
	case isRejectRoute(route):
		return p.handleRejectAction(ctx, route, params)
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// ──────────────────────────────────────────────────
// Waitlist page
// ──────────────────────────────────────────────────

func (p *Plugin) renderWaitlistPage(ctx context.Context, _ contributor.Params) (templ.Component, error) {
	appID := p.resolveAppIDFromContext(ctx)

	var data wldash.WaitlistPageData

	// Fetch counts.
	pending, approved, rejected, err := p.store.CountByStatus(ctx, appID)
	if err != nil {
		data.Error = fmt.Sprintf("Failed to load stats: %v", err)
		return wldash.WaitlistPage(data), nil
	}
	data.PendingCount = pending
	data.ApprovedCount = approved
	data.RejectedCount = rejected

	// Fetch entries.
	list, err := p.store.ListEntries(ctx, &WaitlistQuery{
		AppID: appID,
		Limit: 100,
	})
	if err != nil {
		data.Error = fmt.Sprintf("Failed to load entries: %v", err)
		return wldash.WaitlistPage(data), nil
	}

	for _, e := range list.Entries {
		data.Entries = append(data.Entries, wldash.WaitlistEntryView{
			ID:        e.ID.String(),
			Email:     e.Email,
			Name:      e.Name,
			Status:    string(e.Status),
			CreatedAt: e.CreatedAt,
		})
	}

	return wldash.WaitlistPage(data), nil
}

// ──────────────────────────────────────────────────
// Approve / Reject actions
// ──────────────────────────────────────────────────

func (p *Plugin) handleApproveAction(ctx context.Context, route string, _ contributor.Params) (templ.Component, error) {
	entryIDStr := extractEntryID(route, "/waitlist/approve/")
	if entryIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}

	entryID, err := id.ParseWaitlistID(entryIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}

	if err := p.store.UpdateEntryStatus(ctx, entryID, StatusApproved, ""); err != nil {
		return p.renderWaitlistPageWithError(ctx, fmt.Sprintf("Failed to approve entry: %v", err))
	}

	return p.renderWaitlistPageWithSuccess(ctx, "Entry approved successfully.")
}

func (p *Plugin) handleRejectAction(ctx context.Context, route string, _ contributor.Params) (templ.Component, error) {
	entryIDStr := extractEntryID(route, "/waitlist/reject/")
	if entryIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}

	entryID, err := id.ParseWaitlistID(entryIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}

	if err := p.store.UpdateEntryStatus(ctx, entryID, StatusRejected, ""); err != nil {
		return p.renderWaitlistPageWithError(ctx, fmt.Sprintf("Failed to reject entry: %v", err))
	}

	return p.renderWaitlistPageWithSuccess(ctx, "Entry rejected successfully.")
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// renderWaitlistPageWithError renders the waitlist page with an error message.
func (p *Plugin) renderWaitlistPageWithError(ctx context.Context, errMsg string) (templ.Component, error) {
	appID := p.resolveAppIDFromContext(ctx)
	var data wldash.WaitlistPageData
	data.Error = errMsg

	pending, approved, rejected, err := p.store.CountByStatus(ctx, appID)
	if err == nil {
		data.PendingCount = pending
		data.ApprovedCount = approved
		data.RejectedCount = rejected
	}

	list, err := p.store.ListEntries(ctx, &WaitlistQuery{AppID: appID, Limit: 100})
	if err == nil {
		for _, e := range list.Entries {
			data.Entries = append(data.Entries, wldash.WaitlistEntryView{
				ID:        e.ID.String(),
				Email:     e.Email,
				Name:      e.Name,
				Status:    string(e.Status),
				CreatedAt: e.CreatedAt,
			})
		}
	}

	return wldash.WaitlistPage(data), nil
}

// renderWaitlistPageWithSuccess renders the waitlist page with a success message.
func (p *Plugin) renderWaitlistPageWithSuccess(ctx context.Context, msg string) (templ.Component, error) {
	appID := p.resolveAppIDFromContext(ctx)
	var data wldash.WaitlistPageData
	data.Success = msg

	pending, approved, rejected, err := p.store.CountByStatus(ctx, appID)
	if err == nil {
		data.PendingCount = pending
		data.ApprovedCount = approved
		data.RejectedCount = rejected
	}

	list, err := p.store.ListEntries(ctx, &WaitlistQuery{AppID: appID, Limit: 100})
	if err == nil {
		for _, e := range list.Entries {
			data.Entries = append(data.Entries, wldash.WaitlistEntryView{
				ID:        e.ID.String(),
				Email:     e.Email,
				Name:      e.Name,
				Status:    string(e.Status),
				CreatedAt: e.CreatedAt,
			})
		}
	}

	return wldash.WaitlistPage(data), nil
}

// pendingCount returns the number of pending waitlist entries, falling back to 0 on error.
func (p *Plugin) pendingCount(ctx context.Context) int {
	if p.store == nil {
		return 0
	}
	appID := p.resolveAppIDFromContext(ctx)
	pending, _, _, err := p.store.CountByStatus(ctx, appID)
	if err != nil {
		return 0
	}
	return pending
}

// resolveAppIDFromContext tries to extract the app ID from the dashboard context.
func (p *Plugin) resolveAppIDFromContext(ctx context.Context) id.AppID {
	if appID, ok := dashboard.AppIDFromContext(ctx); ok {
		return appID
	}
	if p.defaultAppID != "" {
		parsed, err := id.ParseAppID(p.defaultAppID)
		if err == nil {
			return parsed
		}
	}
	return id.AppID{}
}

// isApproveRoute checks if the route matches /waitlist/approve/:id.
func isApproveRoute(route string) bool {
	return len(route) > len("/waitlist/approve/") && route[:len("/waitlist/approve/")] == "/waitlist/approve/"
}

// isRejectRoute checks if the route matches /waitlist/reject/:id.
func isRejectRoute(route string) bool {
	return len(route) > len("/waitlist/reject/") && route[:len("/waitlist/reject/")] == "/waitlist/reject/"
}

// extractEntryID extracts the entry ID from a route with the given prefix.
func extractEntryID(route, prefix string) string {
	if len(route) <= len(prefix) {
		return ""
	}
	return route[len(prefix):]
}
