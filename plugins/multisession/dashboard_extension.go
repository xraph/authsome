package multisession

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/multisession/pages"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements the ui.DashboardExtension interface
// This allows the multisession plugin to add its own screens to the dashboard.
type DashboardExtension struct {
	plugin     *Plugin
	baseUIPath string
}

// NewDashboardExtension creates a new dashboard extension for multisession.
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{
		plugin:     plugin,
		baseUIPath: "/api/identity/ui",
	}
}

// SetRegistry sets the extension registry reference (deprecated but kept for compatibility).
func (e *DashboardExtension) SetRegistry(registry any) {
	// No longer needed - layout handled by ForgeUI
}

// ExtensionID returns the unique identifier for this extension.
func (e *DashboardExtension) ExtensionID() string {
	return "multisession"
}

// NavigationItems returns navigation items to register.
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:    "multisession",
			Label: "Sessions",
			Icon: lucide.MonitorSmartphone(
				Class("size-4"),
			),
			Position: ui.NavPositionMain,
			Order:    40,
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp != nil {
					return basePath + "/app/" + currentApp.ID.String() + "/multisession"
				}

				return basePath + ""
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "multisession" || activePage == "session-detail"
			},
			RequiresPlugin: "multisession",
		},
	}
}

// Routes returns routes to register under /app/:appId/.
func (e *DashboardExtension) Routes() []ui.Route {
	return []ui.Route{
		{
			Method:       "GET",
			Path:         "/multisession",
			Handler:      e.ServeMultiSessionPage,
			Name:         "dashboard.multisession",
			Summary:      "Multi-session management",
			Description:  "View and manage multiple active sessions for users in the app",
			Tags:         []string{"Dashboard", "Multi-Session"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/multisession/session/:sessionId",
			Handler:      e.ServeSessionDetailPage,
			Name:         "dashboard.multisession.detail",
			Summary:      "Session detail",
			Description:  "View detailed information about a specific session",
			Tags:         []string{"Dashboard", "Multi-Session"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/multisession/user/:userId",
			Handler:      e.ServeUserSessionsPage,
			Name:         "dashboard.multisession.user",
			Summary:      "User sessions",
			Description:  "View all sessions for a specific user",
			Tags:         []string{"Dashboard", "Multi-Session"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/multisession/revoke/:sessionId",
			Handler:      e.RevokeSession,
			Name:         "dashboard.multisession.revoke",
			Summary:      "Revoke a session",
			Description:  "Revoke a specific session by ID",
			Tags:         []string{"Dashboard", "Multi-Session"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/multisession/revoke-all/:userId",
			Handler:      e.RevokeAllUserSessions,
			Name:         "dashboard.multisession.revoke-all",
			Summary:      "Revoke all user sessions",
			Description:  "Revoke all sessions for a specific user",
			Tags:         []string{"Dashboard", "Multi-Session"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "GET",
			Path:         "/settings/multisession",
			Handler:      e.ServeSettings,
			Name:         "dashboard.settings.multisession",
			Summary:      "Multi-session settings page",
			Description:  "View and configure multi-session settings",
			Tags:         []string{"Dashboard", "Settings", "Multi-Session"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
		{
			Method:       "POST",
			Path:         "/multisession/settings",
			Handler:      e.SaveSettings,
			Name:         "dashboard.multisession.settings",
			Summary:      "Save multisession settings",
			Description:  "Update multisession configuration",
			Tags:         []string{"Dashboard", "Multi-Session"},
			RequireAuth:  true,
			RequireAdmin: true,
		},
	}
}

// SettingsSections returns settings sections to add to the settings page
// Deprecated: Use SettingsPages() instead.
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return nil
}

// SettingsPages returns full settings pages for the new sidebar layout.
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "multisession",
			Label:         "Sessions",
			Description:   "Configure multi-session behavior and limits",
			Icon:          lucide.MonitorSmartphone(Class("h-5 w-5")),
			Category:      "security",
			Order:         20,
			Path:          "multisession",
			RequirePlugin: "multisession",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns widgets to show on the main dashboard.
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "multisession-stats",
			Title: "Active Sessions",
			Icon: lucide.MonitorSmartphone(
				Class("size-5"),
			),
			Order: 30,
			Size:  1,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderDashboardWidget(basePath, currentApp)
			},
		},
	}
}

// BridgeFunctions returns bridge functions for the multisession plugin.
func (e *DashboardExtension) BridgeFunctions() []ui.BridgeFunction {
	return e.getBridgeFunctions()
}

// Helper methods using dashboard handler

func (e *DashboardExtension) getUserFromContext(ctx *router.PageContext) any {
	reqCtx := ctx.Request.Context()
	if u, ok := reqCtx.Value("user").(any); ok {
		return u
	}

	return nil
}

func (e *DashboardExtension) extractAppFromURL(ctx *router.PageContext) (*app.App, error) {
	appIDStr := ctx.Param("appId")
	if appIDStr == "" {
		return nil, errs.RequiredField("appId")
	}

	appID, err := xid.FromString(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid app ID format: %w", err)
	}

	return &app.App{ID: appID}, nil
}

func (e *DashboardExtension) getBasePath() string {
	return e.baseUIPath
}

// ServeMultiSessionPage renders the multi-session management page using v2 pages.
func (e *DashboardExtension) ServeMultiSessionPage(ctx *router.PageContext) (g.Node, error) {
	// Authentication is handled by route middleware (RequireAuth: true)
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	// Use v2 pages package with bridge pattern
	return pages.SessionsListPage(currentApp, basePath), nil
}

// ServeMultiSessionPageLegacy renders the multi-session management page with legacy layout
// Deprecated: Use ServeMultiSessionPage with v2 pages instead.
func (e *DashboardExtension) ServeMultiSessionPageLegacy(ctx *router.PageContext) (g.Node, error) {
	currentUser := e.getUserFromContext(ctx)
	if currentUser == nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, "/api/auth/login", http.StatusFound)

		return nil, nil
	}

	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	// Get filter parameters
	page := queryIntDefault(ctx, "page", 1)
	pageSize := queryIntDefault(ctx, "pageSize", 25)
	deviceFilter := ctx.Query("device")      // mobile, desktop, tablet
	statusFilter := ctx.Query("status")      // active, expiring, expired
	searchQuery := ctx.Query("search")       // user ID search
	view := ctx.QueryDefault("view", "grid") // grid or list

	content := e.renderPageContent(ctx, currentApp, basePath, page, pageSize, deviceFilter, statusFilter, searchQuery, view)

	return content, nil
}

// renderPageContent renders the main content for the multisession page.
func (e *DashboardExtension) renderPageContent(ctx *router.PageContext, currentApp *app.App, basePath string, page, pageSize int, deviceFilter, statusFilter, searchQuery, view string) g.Node {
	reqCtx := ctx.Request.Context()

	// Get all sessions for the app
	sessionsResp, err := e.plugin.service.sessionSvc.ListSessions(reqCtx, &session.ListSessionsFilter{
		AppID: currentApp.ID,
		PaginationParams: pagination.PaginationParams{
			Limit:  pageSize,
			Offset: (page - 1) * pageSize,
		},
	})

	var (
		sessions      []*session.Session
		totalSessions int64
	)

	if err == nil && sessionsResp != nil {
		sessions = sessionsResp.Data
		if sessionsResp.Pagination != nil {
			totalSessions = sessionsResp.Pagination.Total
		}
	}

	// Apply filters
	filteredSessions := e.filterSessions(sessions, deviceFilter, statusFilter, searchQuery)

	// Calculate stats from all sessions (before filtering)
	stats := e.calculateSessionStats(sessions)

	totalPages := int((totalSessions + int64(pageSize) - 1) / int64(pageSize))

	return Div(
		Class("space-y-2"),

		// Page header with gradient accent
		Div(
			Class("relative overflow-hidden rounded-xl bg-gradient-to-r from-indigo-600 via-purple-600 to-pink-500 p-6 text-white shadow-lg"),
			// Background pattern
			Div(
				Class("absolute inset-0 opacity-10"),
				StyleAttr("background-image: url(\"data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='1'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E\")"),
			),
			Div(
				Class("relative flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold"),
						g.Text("Session Management")),
					P(Class("mt-2 text-white/80"),
						g.Text("Monitor and manage active sessions across all devices")),
				),
				// Action buttons
				Div(
					Class("flex items-center gap-3"),
					Button(
						Type("button"),
						Class("inline-flex items-center gap-2 rounded-lg bg-white/20 px-4 py-2 text-sm font-medium text-white backdrop-blur-sm hover:bg-white/30 transition-colors"),
						g.Attr("onclick", "window.location.reload()"),
						lucide.RefreshCw(Class("size-4")),
						g.Text("Refresh"),
					),
				),
			),
		),

		// Stats cards row
		Div(
			Class("grid gap-4 sm:grid-cols-2 lg:grid-cols-4"),
			e.renderStatCard("Total Sessions", strconv.FormatInt(totalSessions, 10), lucide.Layers(Class("size-5")), "bg-blue-500", ""),
			e.renderStatCard("Active Sessions", strconv.Itoa(stats.Active), lucide.Activity(Class("size-5")), "bg-emerald-500", ""),
			e.renderStatCard("Mobile Devices", strconv.Itoa(stats.Mobile), lucide.Smartphone(Class("size-5")), "bg-violet-500", ""),
			e.renderStatCard("Unique Users", strconv.Itoa(stats.UniqueUsers), lucide.Users(Class("size-5")), "bg-amber-500", ""),
		),

		// Filter and search bar
		Div(
			Class("flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between"),
			// Filter tabs
			Div(
				Class("flex flex-wrap items-center gap-2 rounded-lg bg-slate-100 p-1 dark:bg-gray-800"),
				e.renderFilterTab("All", "", statusFilter, view, currentApp, basePath, totalSessions),
				e.renderFilterTab("Active", "active", statusFilter, view, currentApp, basePath, int64(stats.Active)),
				e.renderFilterTab("Expiring", "expiring", statusFilter, view, currentApp, basePath, int64(stats.ExpiringSoon)),
			),
			// Right side controls
			Div(
				Class("flex items-center gap-3"),

				// View Switcher
				Div(
					Class("flex items-center rounded-lg border border-slate-200 bg-white p-1 dark:border-gray-700 dark:bg-gray-800"),
					A(
						Href(fmt.Sprintf("%s/app/%s/multisession?view=grid&device=%s&status=%s&search=%s", basePath, currentApp.ID.String(), deviceFilter, statusFilter, searchQuery)),
						Class("rounded p-1.5 transition-colors "+func() string {
							if view == "grid" {
								return "bg-slate-100 text-slate-900 dark:bg-gray-700 dark:text-white"
							}

							return "text-slate-400 hover:text-slate-600 dark:hover:text-gray-300"
						}()),
						lucide.LayoutGrid(Class("size-4")),
					),
					A(
						Href(fmt.Sprintf("%s/app/%s/multisession?view=list&device=%s&status=%s&search=%s", basePath, currentApp.ID.String(), deviceFilter, statusFilter, searchQuery)),
						Class("rounded p-1.5 transition-colors "+func() string {
							if view == "list" {
								return "bg-slate-100 text-slate-900 dark:bg-gray-700 dark:text-white"
							}

							return "text-slate-400 hover:text-slate-600 dark:hover:text-gray-300"
						}()),
						lucide.List(Class("size-4")),
					),
				),

				// Device filter dropdown
				Div(
					Class("relative"),
					Select(
						Name("device"),
						Class("appearance-none rounded-lg border border-slate-200 bg-white pl-3 pr-8 py-2 text-sm font-medium text-slate-700 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300"),
						g.Attr("onchange", fmt.Sprintf("window.location.href='%s/app/%s/multisession?device='+this.value+'&status=%s&view=%s'", basePath, currentApp.ID.String(), statusFilter, view)),
						Option(Value(""), g.If(deviceFilter == "", g.Attr("selected", "")), g.Text("All Devices")),
						Option(Value("desktop"), g.If(deviceFilter == "desktop", g.Attr("selected", "")), g.Text("Desktop")),
						Option(Value("mobile"), g.If(deviceFilter == "mobile", g.Attr("selected", "")), g.Text("Mobile")),
						Option(Value("tablet"), g.If(deviceFilter == "tablet", g.Attr("selected", "")), g.Text("Tablet")),
					),
					lucide.ChevronDown(Class("pointer-events-none absolute right-2.5 top-1/2 size-4 -translate-y-1/2 text-slate-400")),
				),
				// Search input
				Div(
					Class("relative"),
					lucide.Search(Class("pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400")),
					Input(
						Type("search"),
						Name("search"),
						Placeholder("Search by user ID..."),
						Value(searchQuery),
						Class("w-64 rounded-lg border border-slate-200 bg-white pl-10 pr-4 py-2 text-sm text-slate-900 shadow-sm placeholder:text-slate-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
				),
			),
		),

		// Sessions content
		g.If(len(filteredSessions) == 0,
			e.renderEmptyState(currentApp, basePath),
		),
		g.If(len(filteredSessions) > 0,
			Div(
				Class("space-y-4"),
				// Sessions grid/list
				g.If(view == "list", e.renderSessionsList(filteredSessions, currentApp, basePath)),
				g.If(view != "list", e.renderSessionsGrid(filteredSessions, currentApp, basePath)),

				// Pagination
				g.If(totalPages > 1,
					e.renderPagination(page, totalPages, currentApp, basePath, deviceFilter, statusFilter, view),
				),
			),
		),
	)
}

// Session statistics structure.
type sessionStats struct {
	Active       int
	ExpiringSoon int
	Expired      int
	Mobile       int
	Desktop      int
	Tablet       int
	UniqueUsers  int
}

func (e *DashboardExtension) calculateSessionStats(sessions []*session.Session) sessionStats {
	stats := sessionStats{}
	userMap := make(map[string]bool)
	now := time.Now()
	soonThreshold := 24 * time.Hour

	for _, sess := range sessions {
		// Count unique users
		if !sess.UserID.IsNil() {
			userMap[sess.UserID.String()] = true
		}

		// Count by status
		if sess.ExpiresAt.After(now) {
			stats.Active++
			if sess.ExpiresAt.Sub(now) < soonThreshold {
				stats.ExpiringSoon++
			}
		} else {
			stats.Expired++
		}

		// Count by device type
		device := ParseUserAgent(sess.UserAgent)
		switch {
		case device.IsMobile:
			stats.Mobile++
		case device.IsTablet:
			stats.Tablet++
		default:
			stats.Desktop++
		}
	}

	stats.UniqueUsers = len(userMap)

	return stats
}

func (e *DashboardExtension) filterSessions(sessions []*session.Session, deviceFilter, statusFilter, searchQuery string) []*session.Session {
	if deviceFilter == "" && statusFilter == "" && searchQuery == "" {
		return sessions
	}

	now := time.Now()
	soonThreshold := 24 * time.Hour

	var filtered []*session.Session

	for _, sess := range sessions {
		// Device filter
		if deviceFilter != "" {
			device := ParseUserAgent(sess.UserAgent)

			switch deviceFilter {
			case "mobile":
				if !device.IsMobile {
					continue
				}
			case "desktop":
				if !device.IsDesktop {
					continue
				}
			case "tablet":
				if !device.IsTablet {
					continue
				}
			}
		}

		// Status filter
		if statusFilter != "" {
			switch statusFilter {
			case "active":
				if !sess.ExpiresAt.After(now) {
					continue
				}
			case "expiring":
				if !sess.ExpiresAt.After(now) || sess.ExpiresAt.Sub(now) >= soonThreshold {
					continue
				}
			case "expired":
				if sess.ExpiresAt.After(now) {
					continue
				}
			}
		}

		// Search filter (by user ID)
		if searchQuery != "" {
			if !strings.Contains(strings.ToLower(sess.UserID.String()), strings.ToLower(searchQuery)) {
				continue
			}
		}

		filtered = append(filtered, sess)
	}

	return filtered
}

// renderStatCard renders a stat card with icon and gradient.
func (e *DashboardExtension) renderStatCard(label, value string, icon g.Node, bgColor, change string) g.Node {
	return Div(
		Class("group relative overflow-hidden rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition-all hover:shadow-md dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm font-medium text-slate-500 dark:text-gray-400"),
					g.Text(label)),
				Div(Class("mt-2 text-3xl font-bold tracking-tight text-slate-900 dark:text-white"),
					g.Text(value)),
				g.If(change != "",
					Span(Class("mt-1 inline-flex items-baseline text-sm font-semibold text-emerald-600 dark:text-emerald-400"),
						g.Text(change)),
				),
			),
			Div(
				Class("rounded-xl p-3 "+bgColor+" text-white shadow-lg ring-1 ring-black/5"),
				icon,
			),
		),
	)
}

// renderFilterTab renders a filter tab button.
func (e *DashboardExtension) renderFilterTab(label, value, current, view string, currentApp *app.App, basePath string, count int64) g.Node {
	isActive := current == value
	classes := "inline-flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-all "

	// Badge styles
	badgeClasses := "ml-1.5 rounded-full px-2 py-0.5 text-xs font-semibold "

	if isActive {
		classes += "bg-white text-slate-900 shadow-sm dark:bg-gray-900 dark:text-white"
		badgeClasses += "bg-slate-100 text-slate-900 dark:bg-gray-800 dark:text-gray-200"
	} else {
		classes += "text-slate-600 hover:text-slate-900 hover:bg-white/50 dark:text-gray-400 dark:hover:text-white dark:hover:bg-gray-700/50"
		badgeClasses += "bg-slate-200/50 text-slate-600 dark:bg-gray-700 dark:text-gray-400"
	}

	href := fmt.Sprintf("%s/app/%s/multisession", basePath, currentApp.ID.String())

	queryParams := []string{}
	if value != "" {
		queryParams = append(queryParams, "status="+value)
	}

	if view != "" && view != "grid" {
		queryParams = append(queryParams, "view="+view)
	}

	if len(queryParams) > 0 {
		href += "?" + strings.Join(queryParams, "&")
	}

	return A(
		Href(href),
		Class(classes),
		g.Text(label),
		Span(Class(badgeClasses),
			g.Text(strconv.FormatInt(count, 10))),
	)
}

// renderSessionsList renders sessions in a list layout.
func (e *DashboardExtension) renderSessionsList(sessions []*session.Session, currentApp *app.App, basePath string) g.Node {
	return Div(
		Class("overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Table(
			Class("min-w-full divide-y divide-slate-200 dark:divide-gray-800"),
			g.El("thead",
				Class("bg-slate-50 dark:bg-gray-800/50"),
				Tr(
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"), g.Text("Device")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"), g.Text("User")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"), g.Text("Location")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"), g.Text("Status")),
					Th(Class("px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"), g.Text("Activity")),
					Th(Class("px-6 py-3 text-right text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-400"), g.Text("Actions")),
				),
			),
			g.El("tbody",
				Class("divide-y divide-slate-200 bg-white dark:divide-gray-800 dark:bg-gray-900"),
				g.Map(sessions, func(sess *session.Session) g.Node {
					return e.renderSessionListRow(sess, currentApp, basePath)
				}),
			),
		),
	)
}

// renderSessionListRow renders a single session row.
func (e *DashboardExtension) renderSessionListRow(sess *session.Session, currentApp *app.App, basePath string) g.Node {
	device := ParseUserAgent(sess.UserAgent)
	isActive := IsSessionActive(sess.ExpiresAt)
	isExpiringSoon := IsSessionExpiringSoon(sess.ExpiresAt, 24*time.Hour)

	// Status configuration
	var statusColor, statusBg, statusText string
	if !isActive {
		statusColor = "text-red-700 dark:text-red-400"
		statusBg = "bg-red-50 dark:bg-red-900/20"
		statusText = "Expired"
	} else if isExpiringSoon {
		statusColor = "text-amber-700 dark:text-amber-400"
		statusBg = "bg-amber-50 dark:bg-amber-900/20"
		statusText = "Expiring Soon"
	} else {
		statusColor = "text-emerald-700 dark:text-emerald-400"
		statusBg = "bg-emerald-50 dark:bg-emerald-900/20"
		statusText = "Active"
	}

	return Tr(
		Class("group transition-colors hover:bg-slate-50 dark:hover:bg-gray-800/50"),
		Td(
			Class("whitespace-nowrap px-6 py-4"),
			Div(
				Class("flex items-center gap-3"),
				Div(
					Class("flex h-10 w-10 items-center justify-center rounded-lg "+e.getDeviceBgColor(device)),
					e.getDeviceIcon(device),
				),
				Div(
					Div(Class("font-medium text-slate-900 dark:text-white"), g.Text(device.ShortDeviceInfo())),
					Div(Class("text-xs text-slate-500 dark:text-gray-400"), g.Text(device.OS)),
				),
			),
		),
		Td(
			Class("whitespace-nowrap px-6 py-4"),
			Div(
				Class("flex items-center gap-2"),
				Div(Class("flex h-6 w-6 items-center justify-center rounded-full bg-slate-100 text-xs font-bold text-slate-600 dark:bg-gray-800 dark:text-gray-300"), g.Text("ID")),
				Span(Class("font-mono text-sm text-slate-600 dark:text-gray-400"), g.Text(sess.UserID.String()[0:8]+"..."+sess.UserID.String()[len(sess.UserID.String())-4:])),
			),
		),
		Td(
			Class("whitespace-nowrap px-6 py-4 text-sm text-slate-600 dark:text-gray-400"),
			g.Text(sess.IPAddress),
		),
		Td(
			Class("whitespace-nowrap px-6 py-4"),
			Span(
				Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium "+statusBg+" "+statusColor),
				g.Text(statusText),
			),
		),
		Td(
			Class("whitespace-nowrap px-6 py-4"),
			Div(Class("text-sm text-slate-900 dark:text-white"), g.Text("Created "+FormatRelativeTime(sess.CreatedAt))),
			Div(Class("text-xs text-slate-500 dark:text-gray-400"),
				g.If(isActive, g.Text("Expires "+FormatExpiresIn(sess.ExpiresAt))),
				g.If(!isActive, g.Text("Expired "+FormatRelativeTime(sess.ExpiresAt))),
			),
		),
		Td(
			Class("whitespace-nowrap px-6 py-4 text-right"),
			Div(
				Class("flex items-center justify-end gap-2"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/multisession/session/"+sess.ID.String()),
					Class("rounded-lg p-2 text-slate-400 hover:bg-white hover:text-indigo-600 hover:shadow-sm dark:hover:bg-gray-800 dark:hover:text-indigo-400"),
					g.Attr("title", "View Details"),
					lucide.Eye(Class("size-4")),
				),
				g.If(isActive,
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/multisession/revoke/"+sess.ID.String()),
						g.Attr("onsubmit", "return confirm('Revoke this session?')"),
						Button(
							Type("submit"),
							Class("rounded-lg p-2 text-slate-400 hover:bg-white hover:text-red-600 hover:shadow-sm dark:hover:bg-gray-800 dark:hover:text-red-400"),
							g.Attr("title", "Revoke Session"),
							lucide.LogOut(Class("size-4")),
						),
					),
				),
			),
		),
	)
}

// renderSessionsGrid renders sessions in a modern card grid.
func (e *DashboardExtension) renderSessionsGrid(sessions []*session.Session, currentApp *app.App, basePath string) g.Node {
	cards := make([]g.Node, 0, len(sessions))

	for _, sess := range sessions {
		cards = append(cards, e.renderSessionCard(sess, currentApp, basePath))
	}

	return Div(
		Class("grid gap-4 sm:grid-cols-2 lg:grid-cols-3"),
		g.Group(cards),
	)
}

// renderSessionCard renders a single session card.
func (e *DashboardExtension) renderSessionCard(sess *session.Session, currentApp *app.App, basePath string) g.Node {
	device := ParseUserAgent(sess.UserAgent)
	isActive := IsSessionActive(sess.ExpiresAt)
	isExpiringSoon := IsSessionExpiringSoon(sess.ExpiresAt, 24*time.Hour)

	// Status configuration
	var statusColor, statusBg, statusText string
	if !isActive {
		statusColor = "text-red-700 dark:text-red-400"
		statusBg = "bg-red-50 dark:bg-red-900/20"
		statusText = "Expired"
	} else if isExpiringSoon {
		statusColor = "text-amber-700 dark:text-amber-400"
		statusBg = "bg-amber-50 dark:bg-amber-900/20"
		statusText = "Expiring Soon"
	} else {
		statusColor = "text-emerald-700 dark:text-emerald-400"
		statusBg = "bg-emerald-50 dark:bg-emerald-900/20"
		statusText = "Active"
	}

	// Device icon
	deviceIcon := e.getDeviceIcon(device)

	return Div(
		Class("group relative flex flex-col justify-between overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm transition-all hover:border-indigo-300 hover:shadow-md dark:border-gray-800 dark:bg-gray-900 dark:hover:border-indigo-700"),

		Div(
			Class("p-5"),
			// Header
			Div(
				Class("flex items-start justify-between"),
				Div(
					Class("flex items-center gap-4"),
					// Device icon with platform-specific background
					Div(
						Class("flex h-12 w-12 items-center justify-center rounded-xl "+e.getDeviceBgColor(device)),
						deviceIcon,
					),
					Div(
						H3(Class("font-semibold text-slate-900 dark:text-white"),
							g.Text(device.ShortDeviceInfo())),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text(device.OS)),
					),
				),
				// Status badge
				Span(
					Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium "+statusBg+" "+statusColor),
					g.Text(statusText),
				),
			),

			// Info Grid
			Div(
				Class("mt-5 grid grid-cols-2 gap-4 border-t border-slate-100 pt-4 dark:border-gray-800"),

				// User
				Div(
					Class("col-span-2"),
					P(Class("text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-500"),
						g.Text("User")),
					Div(
						Class("mt-1 flex items-center gap-2"),
						Div(Class("flex h-6 w-6 items-center justify-center rounded-full bg-slate-100 text-xs font-bold text-slate-600 dark:bg-gray-800 dark:text-gray-300"),
							g.Text("ID")),
						Span(Class("truncate font-mono text-sm text-slate-700 dark:text-gray-300"),
							g.Text(sess.UserID.String())),
					),
				),

				// Location/IP
				Div(
					P(Class("text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-500"),
						g.Text("IP Address")),
					P(Class("mt-1 text-sm font-medium text-slate-900 dark:text-white"),
						g.Text(sess.IPAddress)),
				),

				// Created
				Div(
					P(Class("text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-500"),
						g.Text("Created")),
					P(Class("mt-1 text-sm text-slate-700 dark:text-gray-300"),
						g.Text(FormatRelativeTime(sess.CreatedAt))),
				),
			),
		),

		// Actions Footer
		Div(
			Class("mt-auto flex items-center justify-between border-t border-slate-100 bg-slate-50/50 px-5 py-3 dark:border-gray-800 dark:bg-gray-800/50"),

			Span(Class("text-xs text-slate-500 dark:text-gray-400"),
				g.If(isActive, g.Text("Expires "+FormatExpiresIn(sess.ExpiresAt))),
				g.If(!isActive, g.Text("Expired "+FormatRelativeTime(sess.ExpiresAt))),
			),

			Div(
				Class("flex items-center gap-2"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/multisession/session/"+sess.ID.String()),
					Class("rounded-lg p-2 text-slate-600 hover:bg-white hover:text-indigo-600 hover:shadow-sm dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-indigo-400"),
					g.Attr("title", "View Details"),
					lucide.Eye(Class("size-4")),
				),
				g.If(isActive,
					Form(
						Method("POST"),
						Action(basePath+"/app/"+currentApp.ID.String()+"/multisession/revoke/"+sess.ID.String()),
						g.Attr("onsubmit", "return confirm('Revoke this session?')"),
						Button(
							Type("submit"),
							Class("rounded-lg p-2 text-slate-600 hover:bg-white hover:text-red-600 hover:shadow-sm dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-red-400"),
							g.Attr("title", "Revoke Session"),
							lucide.LogOut(Class("size-4")),
						),
					),
				),
			),
		),
	)
}

// getDeviceBgColor returns a background color class based on device type.
func (e *DashboardExtension) getDeviceBgColor(device *DeviceInfo) string {
	switch {
	case device.IsMobile:
		return "bg-purple-100 text-purple-600 dark:bg-purple-900/30 dark:text-purple-400"
	case device.IsTablet:
		return "bg-amber-100 text-amber-600 dark:bg-amber-900/30 dark:text-amber-400"
	case device.IsBot:
		return "bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400"
	default:
		return "bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400"
	}
}

// getDeviceIcon returns an appropriate icon based on device type.
func (e *DashboardExtension) getDeviceIcon(device *DeviceInfo) g.Node {
	iconClass := "size-6 text-slate-600 dark:text-gray-400"

	switch {
	case device.IsMobile:
		return lucide.Smartphone(Class(iconClass))
	case device.IsTablet:
		return lucide.Tablet(Class(iconClass))
	case device.IsBot:
		return lucide.Bot(Class(iconClass))
	default:
		return lucide.Monitor(Class(iconClass))
	}
}

// getStatusBadgeClasses returns classes for status badge.
func (e *DashboardExtension) getStatusBadgeClasses(status string) string {
	switch status {
	case "Active":
		return "bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400"
	case "Expiring Soon":
		return "bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-400"
	case "Expired":
		return "bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-400"
	default:
		return "bg-slate-50 text-slate-700 dark:bg-gray-800 dark:text-gray-400"
	}
}

// renderEmptyState renders the empty state.
func (e *DashboardExtension) renderEmptyState(currentApp *app.App, basePath string) g.Node {
	return Div(
		Class("rounded-xl border border-dashed border-slate-300 bg-white p-12 text-center dark:border-gray-700 dark:bg-gray-900"),
		Div(
			Class("mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-slate-100 dark:bg-gray-800"),
			lucide.MonitorSmartphone(Class("size-8 text-slate-400 dark:text-gray-500")),
		),
		H3(Class("mt-4 text-lg font-semibold text-slate-900 dark:text-white"),
			g.Text("No Active Sessions")),
		P(Class("mt-2 text-slate-500 dark:text-gray-400 max-w-md mx-auto"),
			g.Text("There are no active sessions matching your filters. Sessions will appear here when users sign in to your application.")),
	)
}

// renderPagination renders pagination controls.
func (e *DashboardExtension) renderPagination(currentPage, totalPages int, currentApp *app.App, basePath, deviceFilter, statusFilter, view string) g.Node {
	if totalPages <= 1 {
		return nil
	}

	buildURL := func(page int) string {
		url := fmt.Sprintf("%s/app/%s/multisession?page=%d", basePath, currentApp.ID.String(), page)
		if deviceFilter != "" {
			url += "&device=" + deviceFilter
		}

		if statusFilter != "" {
			url += "&status=" + statusFilter
		}

		if view != "" && view != "grid" {
			url += "&view=" + view
		}

		return url
	}

	items := make([]g.Node, 0)

	// Previous button
	if currentPage > 1 {
		items = append(items, A(
			Href(buildURL(currentPage-1)),
			Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
			lucide.ChevronLeft(Class("size-4")),
			g.Text("Previous"),
		))
	}

	// Page numbers
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			items = append(items, Span(
				Class("inline-flex items-center justify-center rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white"),
				g.Text(strconv.Itoa(i)),
			))
		} else if i == 1 || i == totalPages || (i >= currentPage-1 && i <= currentPage+1) {
			items = append(items, A(
				Href(buildURL(i)),
				Class("inline-flex items-center justify-center rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
				g.Text(strconv.Itoa(i)),
			))
		} else if i == currentPage-2 || i == currentPage+2 {
			items = append(items, Span(
				Class("px-2 py-2 text-slate-400"),
				g.Text("..."),
			))
		}
	}

	// Next button
	if currentPage < totalPages {
		items = append(items, A(
			Href(buildURL(currentPage+1)),
			Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
			g.Text("Next"),
			lucide.ChevronRight(Class("size-4")),
		))
	}

	return Div(
		Class("flex items-center justify-center gap-2"),
		g.Group(items),
	)
}

// ServeSessionDetailPage renders detailed information about a single session
// ServeSessionDetailPage renders the session detail page using v2 pages.
func (e *DashboardExtension) ServeSessionDetailPage(ctx *router.PageContext) (g.Node, error) {
	// Authentication is handled by route middleware (RequireAuth: true)
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	sessionIDStr := ctx.Param("sessionId")
	if sessionIDStr == "" {
		return nil, errs.BadRequest("Session ID required")
	}

	basePath := e.getBasePath()

	// Use v2 pages package with bridge pattern
	return pages.SessionDetailPage(currentApp, sessionIDStr, basePath), nil
}

// ServeSessionDetailPageLegacy renders the session detail page with legacy layout
// Deprecated: Use ServeSessionDetailPage with v2 pages instead.
func (e *DashboardExtension) ServeSessionDetailPageLegacy(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	sessionIDStr := ctx.Param("sessionId")
	if sessionIDStr == "" {
		return nil, errs.BadRequest("Session ID required")
	}

	sessionID, err := xid.FromString(sessionIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid session ID")
	}

	reqCtx := ctx.Request.Context()

	sess, err := e.plugin.service.sessionSvc.FindByID(reqCtx, sessionID)
	if err != nil {
		return nil, errs.NotFound("Session not found")
	}

	basePath := e.getBasePath()

	content := e.renderSessionDetailContent(sess, currentApp, basePath)

	return content, nil
}

// renderSessionDetailContent renders the session detail page content.
func (e *DashboardExtension) renderSessionDetailContent(sess *session.Session, currentApp *app.App, basePath string) g.Node {
	device := ParseUserAgent(sess.UserAgent)
	isActive := IsSessionActive(sess.ExpiresAt)
	isExpiringSoon := IsSessionExpiringSoon(sess.ExpiresAt, 24*time.Hour)

	// Status configuration
	var statusColor, statusBg, statusText string
	if !isActive {
		statusColor = "text-red-700 dark:text-red-400"
		statusBg = "bg-red-50 dark:bg-red-900/20"
		statusText = "Expired"
	} else if isExpiringSoon {
		statusColor = "text-amber-700 dark:text-amber-400"
		statusBg = "bg-amber-50 dark:bg-amber-900/20"
		statusText = "Expiring Soon"
	} else {
		statusColor = "text-emerald-700 dark:text-emerald-400"
		statusBg = "bg-emerald-50 dark:bg-emerald-900/20"
		statusText = "Active"
	}

	return Div(
		Class("space-y-2"),

		// Breadcrumb and back button
		Div(
			Class("flex items-center gap-4"),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/multisession"),
				Class("group inline-flex items-center gap-2 text-sm font-medium text-slate-600 transition-colors hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
				lucide.ArrowLeft(Class("size-4 transition-transform group-hover:-translate-x-1")),
				g.Text("Back to Sessions"),
			),
		),

		// Header card
		Div(
			Class("relative overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("p-6"),
				Div(
					Class("flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between"),
					Div(
						Class("flex items-center gap-5"),
						// Large device icon
						Div(
							Class("flex h-16 w-16 items-center justify-center rounded-2xl "+e.getDeviceBgColor(device)+" shadow-inner"),
							e.getLargeDeviceIcon(device),
						),
						Div(
							H1(Class("text-2xl font-bold tracking-tight text-slate-900 dark:text-white"),
								g.Text(device.FormatDeviceInfo())),
							Div(
								Class("mt-2 flex flex-wrap items-center gap-3"),
								Span(
									Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium "+statusBg+" "+statusColor),
									g.Text(statusText),
								),
								Span(Class("text-sm text-slate-500 dark:text-gray-400"),
									g.Text("Session ID: "+sess.ID.String())),
							),
						),
					),
					// Actions
					g.If(isActive,
						Form(
							Method("POST"),
							Action(basePath+"/app/"+currentApp.ID.String()+"/multisession/revoke/"+sess.ID.String()),
							g.Attr("onsubmit", "return confirm('Are you sure you want to revoke this session? The user will be logged out.')"),
							Button(
								Type("submit"),
								Class("inline-flex w-full items-center justify-center gap-2 rounded-lg bg-white border border-slate-200 px-4 py-2 text-sm font-medium text-red-600 shadow-sm hover:bg-red-50 hover:border-red-200 hover:text-red-700 sm:w-auto dark:bg-gray-800 dark:border-gray-700 dark:text-red-400 dark:hover:bg-gray-700/50"),
								lucide.LogOut(Class("size-4")),
								g.Text("Revoke Session"),
							),
						),
					),
				),
			),
		),

		// Details grid
		Div(
			Class("grid gap-6 lg:grid-cols-2"),

			// Device Information
			Div(
				Class("rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				H2(Class("flex items-center gap-2 text-lg font-semibold text-slate-900 dark:text-white mb-6"),
					Div(Class("rounded-lg bg-indigo-50 p-1.5 dark:bg-indigo-900/20"),
						lucide.MonitorSmartphone(Class("size-5 text-indigo-600 dark:text-indigo-400"))),
					g.Text("Device Information"),
				),
				Div(
					Class("space-y-5"),
					e.renderDetailRow("Device Type", device.DeviceType, lucide.Laptop(Class("size-4 text-slate-400"))),
					e.renderDetailRow("Browser", device.Browser+" "+device.BrowserVer, lucide.Globe(Class("size-4 text-slate-400"))),
					e.renderDetailRow("Operating System", device.OS+" "+device.OSVersion, lucide.Settings(Class("size-4 text-slate-400"))),
					g.If(sess.UserAgent != "",
						Div(
							Class("pt-5 border-t border-slate-100 dark:border-gray-800"),
							Label(Class("text-xs font-medium uppercase tracking-wider text-slate-500 dark:text-gray-500"),
								g.Text("User Agent String")),
							Div(Class("mt-2 rounded-lg bg-slate-50 p-3 font-mono text-xs text-slate-600 dark:bg-gray-800 dark:text-gray-400 break-all"),
								g.Text(sess.UserAgent)),
						),
					),
				),
			),

			// Session Details
			Div(
				Class("rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				H2(Class("flex items-center gap-2 text-lg font-semibold text-slate-900 dark:text-white mb-6"),
					Div(Class("rounded-lg bg-indigo-50 p-1.5 dark:bg-indigo-900/20"),
						lucide.Clock(Class("size-5 text-indigo-600 dark:text-indigo-400"))),
					g.Text("Session Activity"),
				),
				Div(
					Class("space-y-5"),
					e.renderDetailRow("IP Address", sess.IPAddress, lucide.MapPin(Class("size-4 text-slate-400"))),
					e.renderDetailRow("Created", sess.CreatedAt.Format("Jan 2, 2006 at 3:04 PM"), lucide.Calendar(Class("size-4 text-slate-400"))),
					e.renderDetailRow("Last Updated", sess.UpdatedAt.Format("Jan 2, 2006 at 3:04 PM"), lucide.RefreshCw(Class("size-4 text-slate-400"))),
					e.renderDetailRow("Expires", sess.ExpiresAt.Format("Jan 2, 2006 at 3:04 PM"), lucide.Timer(Class("size-4 text-slate-400"))),
					e.renderTimeRow("Last Refreshed", sess.LastRefreshedAt, "Jan 2, 2006 at 3:04 PM", lucide.RotateCw(Class("size-4 text-slate-400"))),
				),
			),

			// User Information
			Div(
				Class("rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				H2(Class("flex items-center gap-2 text-lg font-semibold text-slate-900 dark:text-white mb-6"),
					Div(Class("rounded-lg bg-indigo-50 p-1.5 dark:bg-indigo-900/20"),
						lucide.User(Class("size-5 text-indigo-600 dark:text-indigo-400"))),
					g.Text("User Information"),
				),
				Div(
					Class("space-y-5"),
					e.renderDetailRow("User ID", sess.UserID.String(), lucide.Hash(Class("size-4 text-slate-400"))),
					Div(
						Class("pt-2"),
						A(
							Href(basePath+"/app/"+currentApp.ID.String()+"/multisession/user/"+sess.UserID.String()),
							Class("inline-flex items-center gap-2 text-sm font-medium text-indigo-600 hover:text-indigo-700 dark:text-indigo-400"),
							g.Text("View all sessions for this user"),
							lucide.ArrowRight(Class("size-4")),
						),
					),
				),
			),

			// Context Information
			Div(
				Class("rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				H2(Class("flex items-center gap-2 text-lg font-semibold text-slate-900 dark:text-white mb-6"),
					Div(Class("rounded-lg bg-indigo-50 p-1.5 dark:bg-indigo-900/20"),
						lucide.Layers(Class("size-5 text-indigo-600 dark:text-indigo-400"))),
					g.Text("Application Context"),
				),
				Div(
					Class("space-y-5"),
					e.renderDetailRow("App ID", sess.AppID.String(), lucide.AppWindow(Class("size-4 text-slate-400"))),
					e.renderOptionalIDRow("Organization ID", sess.OrganizationID, lucide.Building2(Class("size-4 text-slate-400"))),
					e.renderOptionalIDRow("Environment ID", sess.EnvironmentID, lucide.Server(Class("size-4 text-slate-400"))),
				),
			),
		),
	)
}

// renderOptionalIDRow renders a detail row for an optional ID.
func (e *DashboardExtension) renderOptionalIDRow(label string, id *xid.ID, icon g.Node) g.Node {
	if id == nil {
		return nil
	}

	return e.renderDetailRow(label, id.String(), icon)
}

// renderTimeRow renders a detail row for an optional time.
func (e *DashboardExtension) renderTimeRow(label string, t *time.Time, format string, icon g.Node) g.Node {
	if t == nil {
		return nil
	}

	return e.renderDetailRow(label, t.Format(format), icon)
}

// getLargeDeviceIcon returns a larger icon for the detail page.
func (e *DashboardExtension) getLargeDeviceIcon(device *DeviceInfo) g.Node {
	iconClass := "size-8"

	switch {
	case device.IsMobile:
		return lucide.Smartphone(Class(iconClass))
	case device.IsTablet:
		return lucide.Tablet(Class(iconClass))
	case device.IsBot:
		return lucide.Bot(Class(iconClass))
	default:
		return lucide.Monitor(Class(iconClass))
	}
}

// renderDetailRow renders a detail row with icon, label, and value.
func (e *DashboardExtension) renderDetailRow(label, value string, icon g.Node) g.Node {
	if value == "" {
		value = "N/A"
	}

	return Div(
		Class("flex items-start gap-3"),
		icon,
		Div(
			Div(Class("text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
				g.Text(label)),
			Div(Class("mt-0.5 text-sm text-slate-900 dark:text-white font-medium"),
				g.Text(value)),
		),
	)
}

// ServeUserSessionsPage renders all sessions for a specific user
// ServeUserSessionsPage renders the user sessions page using v2 pages.
func (e *DashboardExtension) ServeUserSessionsPage(ctx *router.PageContext) (g.Node, error) {
	// Authentication is handled by route middleware (RequireAuth: true)
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	userIDStr := ctx.Param("userId")
	if userIDStr == "" {
		return nil, errs.BadRequest("User ID required")
	}

	basePath := e.getBasePath()

	// Use v2 pages package with bridge pattern
	return pages.UserSessionsPage(currentApp, userIDStr, basePath), nil
}

// ServeUserSessionsPageLegacy renders the user sessions page with legacy layout
// Deprecated: Use ServeUserSessionsPage with v2 pages instead.
func (e *DashboardExtension) ServeUserSessionsPageLegacy(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	userIDStr := ctx.Param("userId")
	if userIDStr == "" {
		return nil, errs.BadRequest("User ID required")
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid user ID")
	}

	reqCtx := ctx.Request.Context()
	basePath := e.getBasePath()

	// Get all sessions for this user
	sessionsResp, err := e.plugin.service.sessionSvc.ListSessions(reqCtx, &session.ListSessionsFilter{
		UserID: &userID,
		AppID:  currentApp.ID,
		PaginationParams: pagination.PaginationParams{
			Limit:  100,
			Offset: 0,
		},
	})

	var sessions []*session.Session
	if err == nil && sessionsResp != nil {
		sessions = sessionsResp.Data
	}

	content := e.renderUserSessionsContent(userID, sessions, currentApp, basePath)

	return content, nil
}

// renderUserSessionsContent renders the user sessions page content.
func (e *DashboardExtension) renderUserSessionsContent(userID xid.ID, sessions []*session.Session, currentApp *app.App, basePath string) g.Node {
	activeCount := 0

	for _, sess := range sessions {
		if IsSessionActive(sess.ExpiresAt) {
			activeCount++
		}
	}

	return Div(
		Class("space-y-2"),

		// Breadcrumb
		Div(
			Class("flex items-center gap-4"),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/multisession"),
				Class("inline-flex items-center gap-2 text-sm text-slate-600 hover:text-slate-900 dark:text-gray-400 dark:hover:text-white"),
				lucide.ArrowLeft(Class("size-4")),
				g.Text("Back to Sessions"),
			),
		),

		// Header
		Div(
			Class("rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("flex items-center justify-between"),
				Div(
					Class("flex items-center gap-4"),
					Div(
						Class("flex h-14 w-14 items-center justify-center rounded-full bg-gradient-to-br from-indigo-500 to-purple-600 text-white shadow-lg"),
						lucide.User(Class("size-7")),
					),
					Div(
						H1(Class("text-xl font-bold text-slate-900 dark:text-white"),
							g.Text("User Sessions")),
						Div(Class("mt-1 text-sm text-slate-500 dark:text-gray-400 font-mono"),
							g.Text(userID.String())),
					),
				),
				Div(
					Class("flex items-center gap-4"),
					Div(
						Class("text-right"),
						Div(Class("text-2xl font-bold text-slate-900 dark:text-white"),
							g.Text(strconv.Itoa(len(sessions)))),
						Div(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Total Sessions")),
					),
					Div(
						Class("text-right"),
						Div(Class("text-2xl font-bold text-emerald-600 dark:text-emerald-400"),
							g.Text(strconv.Itoa(activeCount))),
						Div(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Active")),
					),
					g.If(activeCount > 0,
						Form(
							Method("POST"),
							Action(basePath+"/app/"+currentApp.ID.String()+"/multisession/revoke-all/"+userID.String()),
							g.Attr("onsubmit", "return confirm('Are you sure you want to revoke all sessions for this user? They will be logged out from all devices.')"),
							Button(
								Type("submit"),
								Class("inline-flex items-center gap-2 rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"),
								lucide.LogOut(Class("size-4")),
								g.Text("Revoke All"),
							),
						),
					),
				),
			),
		),

		// Sessions grid
		g.If(len(sessions) == 0,
			Div(
				Class("rounded-xl border border-dashed border-slate-300 bg-white p-12 text-center dark:border-gray-700 dark:bg-gray-900"),
				lucide.MonitorSmartphone(Class("mx-auto size-12 text-slate-400 dark:text-gray-500 mb-4")),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("No Sessions")),
				P(Class("mt-2 text-slate-500 dark:text-gray-400"),
					g.Text("This user has no active sessions.")),
			),
		),
		g.If(len(sessions) > 0,
			e.renderSessionsGrid(sessions, currentApp, basePath),
		),
	)
}

// RenderDashboardWidget renders the dashboard widget showing session stats.
func (e *DashboardExtension) RenderDashboardWidget(basePath string, currentApp *app.App) g.Node {
	return Div(
		Class("space-y-4"),

		// Stats
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("—")),
				P(Class("text-sm text-slate-500 dark:text-gray-400"),
					g.Text("Active sessions")),
			),
			Div(
				Class("rounded-full bg-indigo-100 p-3 dark:bg-indigo-900/30"),
				lucide.MonitorSmartphone(Class("size-6 text-indigo-600 dark:text-indigo-400")),
			),
		),

		// Quick stats
		Div(
			Class("grid grid-cols-2 gap-4"),
			Div(
				Class("rounded-lg bg-slate-50 p-3 dark:bg-gray-800"),
				Div(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("—")),
				Div(Class("text-xs text-slate-500 dark:text-gray-400"), g.Text("Mobile")),
			),
			Div(
				Class("rounded-lg bg-slate-50 p-3 dark:bg-gray-800"),
				Div(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text("—")),
				Div(Class("text-xs text-slate-500 dark:text-gray-400"), g.Text("Desktop")),
			),
		),

		// View more link
		Div(
			Class("pt-4 border-t border-slate-200 dark:border-gray-700"),
			A(
				Href(basePath+"/app/"+currentApp.ID.String()+"/multisession"),
				Class("inline-flex items-center gap-2 text-sm font-medium text-indigo-600 hover:text-indigo-700 dark:text-indigo-400"),
				g.Text("Manage sessions"),
				lucide.ArrowRight(Class("size-4")),
			),
		),
	)
}

// Action handlers

// RevokeSession handles session revocation.
func (e *DashboardExtension) RevokeSession(ctx *router.PageContext) (g.Node, error) {
	sessionIDStr := ctx.Param("sessionId")
	if sessionIDStr == "" {
		return nil, errs.BadRequest("Session ID required")
	}

	sessionID, err := xid.FromString(sessionIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid session ID")
	}

	reqCtx := ctx.Request.Context()
	if err := e.plugin.service.sessionSvc.RevokeByID(reqCtx, sessionID); err != nil {
		return nil, errs.InternalServerErrorWithMessage("Failed to revoke session")
	}

	currentApp, _ := e.extractAppFromURL(ctx)

	basePath := e.getBasePath()
	if currentApp != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/multisession", http.StatusFound)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/", http.StatusFound)

	return nil, nil
}

// RevokeAllUserSessions handles revoking all sessions for a user.
func (e *DashboardExtension) RevokeAllUserSessions(ctx *router.PageContext) (g.Node, error) {
	userIDStr := ctx.Param("userId")
	if userIDStr == "" {
		return nil, errs.BadRequest("User ID required")
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return nil, errs.BadRequest("Invalid user ID")
	}

	reqCtx := ctx.Request.Context()

	currentApp, _ := e.extractAppFromURL(ctx)

	// Get all user sessions
	sessionsResp, err := e.plugin.service.sessionSvc.ListSessions(reqCtx, &session.ListSessionsFilter{
		UserID: &userID,
		AppID:  currentApp.ID,
		PaginationParams: pagination.PaginationParams{
			Limit:  1000,
			Offset: 0,
		},
	})
	if err != nil {
		return nil, errs.InternalServerErrorWithMessage("Failed to fetch user sessions")
	}

	// Revoke all sessions
	if sessionsResp != nil {
		for _, sess := range sessionsResp.Data {
			_ = e.plugin.service.sessionSvc.RevokeByID(reqCtx, sess.ID)
		}
	}

	basePath := e.getBasePath()
	if currentApp != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/multisession", http.StatusFound)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/", http.StatusFound)

	return nil, nil
}

// ServeSettings renders the multisession settings page
// ServeSettings renders the settings page using v2 pages.
func (e *DashboardExtension) ServeSettings(ctx *router.PageContext) (g.Node, error) {
	// Authentication is handled by route middleware (RequireAuth: true, RequireAdmin: true)
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	basePath := e.getBasePath()

	// Use v2 pages package with bridge pattern
	return pages.SettingsPage(currentApp, basePath), nil
}

// ServeSettingsLegacy renders the settings page with legacy layout
// Deprecated: Use ServeSettings with v2 pages instead.
func (e *DashboardExtension) ServeSettingsLegacy(ctx *router.PageContext) (g.Node, error) {
	currentApp, err := e.extractAppFromURL(ctx)
	if err != nil {
		return nil, errs.BadRequest("Invalid app context")
	}

	content := e.renderSettingsContent(currentApp, e.getBasePath())

	return content, nil
}

// renderSettingsContent renders the settings page content.
func (e *DashboardExtension) renderSettingsContent(currentApp *app.App, basePath string) g.Node {
	cfg := e.plugin.config

	return Div(
		Class("space-y-2"),

		// Header
		Div(
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Session Settings")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Configure multi-session behavior, limits, and security settings")),
		),

		// Settings form
		Form(
			Method("POST"),
			Action(basePath+"/app/"+currentApp.ID.String()+"/multisession/settings"),
			Class("space-y-2"),

			// Session Limits section
			Div(
				Class("rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-6"),
					Div(
						Class("rounded-lg bg-indigo-100 p-2 dark:bg-indigo-900/30"),
						lucide.Layers(Class("size-5 text-indigo-600 dark:text-indigo-400")),
					),
					Div(
						H2(Class("text-lg font-semibold text-slate-900 dark:text-white"),
							g.Text("Session Limits")),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Control how many sessions users can have")),
					),
				),

				Div(
					Class("grid gap-6 md:grid-cols-2"),

					// Max sessions per user
					Div(
						Label(
							For("maxSessionsPerUser"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Max Sessions Per User"),
						),
						Input(
							Type("number"),
							Name("maxSessionsPerUser"),
							ID("maxSessionsPerUser"),
							Value(strconv.Itoa(cfg.MaxSessionsPerUser)),
							Min("1"),
							Max("100"),
							Class("mt-2 block w-full rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-2 text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Maximum number of concurrent sessions a user can have active")),
					),

					// Session expiry
					Div(
						Label(
							For("sessionExpiryHours"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
							g.Text("Session Expiry (Hours)"),
						),
						Input(
							Type("number"),
							Name("sessionExpiryHours"),
							ID("sessionExpiryHours"),
							Value(strconv.Itoa(cfg.SessionExpiryHours)),
							Min("1"),
							Max("8760"),
							Class("mt-2 block w-full rounded-lg border border-slate-200 bg-white px-4 py-2.5 text-slate-900 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
						),
						P(Class("mt-2 text-sm text-slate-500 dark:text-gray-400"),
							g.Text("How long sessions remain valid before expiring")),
					),
				),
			),

			// Device Tracking section
			Div(
				Class("rounded-xl border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
				Div(
					Class("flex items-center gap-3 mb-6"),
					Div(
						Class("rounded-lg bg-emerald-100 p-2 dark:bg-emerald-900/30"),
						lucide.MonitorSmartphone(Class("size-5 text-emerald-600 dark:text-emerald-400")),
					),
					Div(
						H2(Class("text-lg font-semibold text-slate-900 dark:text-white"),
							g.Text("Device & Platform")),
						P(Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Configure device tracking and cross-platform settings")),
					),
				),

				Div(
					Class("space-y-4"),

					// Device tracking toggle
					Div(
						Class("flex items-center justify-between rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-gray-700 dark:bg-gray-800"),
						Div(
							Class("flex items-center gap-3"),
							lucide.Fingerprint(Class("size-5 text-slate-500 dark:text-gray-400")),
							Div(
								Div(Class("font-medium text-slate-900 dark:text-white"),
									g.Text("Enable Device Tracking")),
								Div(Class("text-sm text-slate-500 dark:text-gray-400"),
									g.Text("Track and identify devices for each session")),
							),
						),
						Label(
							Class("relative inline-flex cursor-pointer items-center"),
							Input(
								Type("checkbox"),
								Name("enableDeviceTracking"),
								Value("true"),
								g.If(cfg.EnableDeviceTracking, Checked()),
								Class("peer sr-only"),
							),
							Span(
								Class("peer h-6 w-11 rounded-full bg-slate-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-slate-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-indigo-600 peer-checked:after:translate-x-full peer-checked:after:border-white peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-indigo-300 dark:bg-gray-700 dark:peer-focus:ring-indigo-800"),
							),
						),
					),

					// Cross-platform toggle
					Div(
						Class("flex items-center justify-between rounded-lg border border-slate-200 bg-slate-50 p-4 dark:border-gray-700 dark:bg-gray-800"),
						Div(
							Class("flex items-center gap-3"),
							lucide.Globe(Class("size-5 text-slate-500 dark:text-gray-400")),
							Div(
								Div(Class("font-medium text-slate-900 dark:text-white"),
									g.Text("Allow Cross-Platform Sessions")),
								Div(Class("text-sm text-slate-500 dark:text-gray-400"),
									g.Text("Allow users to have sessions on different platforms simultaneously")),
							),
						),
						Label(
							Class("relative inline-flex cursor-pointer items-center"),
							Input(
								Type("checkbox"),
								Name("allowCrossPlatform"),
								Value("true"),
								g.If(cfg.AllowCrossPlatform, Checked()),
								Class("peer sr-only"),
							),
							Span(
								Class("peer h-6 w-11 rounded-full bg-slate-200 after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-slate-300 after:bg-white after:transition-all after:content-[''] peer-checked:bg-indigo-600 peer-checked:after:translate-x-full peer-checked:after:border-white peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-indigo-300 dark:bg-gray-700 dark:peer-focus:ring-indigo-800"),
							),
						),
					),
				),
			),

			// Submit button
			Div(
				Class("flex justify-end"),
				Button(
					Type("submit"),
					Class("inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-6 py-2.5 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2 transition-colors"),
					lucide.Save(Class("size-4")),
					g.Text("Save Settings"),
				),
			),
		),
	)
}

// SaveSettings handles saving multisession settings.
func (e *DashboardExtension) SaveSettings(ctx *router.PageContext) (g.Node, error) {
	if err := ctx.Request.ParseForm(); err != nil {
		return nil, errs.BadRequest("Invalid form data")
	}

	form := ctx.Request.Form

	// Parse max sessions per user
	maxSessions := 10

	if val := form.Get("maxSessionsPerUser"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 && parsed <= 100 {
			maxSessions = parsed
		}
	}

	// Parse session expiry hours
	sessionExpiry := 720

	if val := form.Get("sessionExpiryHours"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 && parsed <= 8760 {
			sessionExpiry = parsed
		}
	}

	// Parse checkboxes
	enableDeviceTracking := form.Get("enableDeviceTracking") == "true"
	allowCrossPlatform := form.Get("allowCrossPlatform") == "true"

	// Update plugin config
	e.plugin.config.MaxSessionsPerUser = maxSessions
	e.plugin.config.SessionExpiryHours = sessionExpiry
	e.plugin.config.EnableDeviceTracking = enableDeviceTracking
	e.plugin.config.AllowCrossPlatform = allowCrossPlatform

	currentApp, _ := e.extractAppFromURL(ctx)

	basePath := e.getBasePath()
	if currentApp != nil {
		http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/app/"+currentApp.ID.String()+"/settings/multisession?saved=true", http.StatusFound)

		return nil, nil
	}

	http.Redirect(ctx.ResponseWriter, ctx.Request, basePath+"/settings/multisession?saved=true", http.StatusFound)

	return nil, nil
}

// Helper functions

func queryIntDefault(ctx *router.PageContext, name string, defaultValue int) int {
	str := ctx.QueryDefault(name, "")
	if str == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return defaultValue
	}

	return val
}
