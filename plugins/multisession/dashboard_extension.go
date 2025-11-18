package multisession

import (
	"fmt"
	"net/http"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/session"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/authsome/plugins/dashboard"
	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/forge"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// DashboardExtension implements the ui.DashboardExtension interface
// This allows the multisession plugin to add its own screens to the dashboard
type DashboardExtension struct {
	plugin   *Plugin
	registry *dashboard.ExtensionRegistry
}

// NewDashboardExtension creates a new dashboard extension for multisession
func NewDashboardExtension(plugin *Plugin) *DashboardExtension {
	return &DashboardExtension{plugin: plugin}
}

// SetRegistry sets the extension registry reference (called by dashboard after registration)
func (e *DashboardExtension) SetRegistry(registry *dashboard.ExtensionRegistry) {
	e.registry = registry
}

// ExtensionID returns the unique identifier for this extension
func (e *DashboardExtension) ExtensionID() string {
	return "multisession"
}

// NavigationItems returns navigation items to register
func (e *DashboardExtension) NavigationItems() []ui.NavigationItem {
	return []ui.NavigationItem{
		{
			ID:    "multisession",
			Label: "Multi-Session",
			Icon: lucide.Smartphone(
				Class("size-4"),
			),
			Position: ui.NavPositionMain,
			Order:    50, // Place after Sessions (order 40) but before Plugins (order 90)
			URLBuilder: func(basePath string, currentApp *app.App) string {
				if currentApp != nil {
					return basePath + "/dashboard/app/" + currentApp.ID.String() + "/multisession"
				}
				return basePath + "/dashboard/"
			},
			ActiveChecker: func(activePage string) bool {
				return activePage == "multisession"
			},
			RequiresPlugin: "multisession",
		},
	}
}

// Routes returns routes to register under /dashboard/app/:appId/
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
// Deprecated: Use SettingsPages() instead
func (e *DashboardExtension) SettingsSections() []ui.SettingsSection {
	return []ui.SettingsSection{
		{
			ID:          "multisession-settings",
			Title:       "Multi-Session Configuration",
			Description: "Configure multi-session behavior and limits",
			Icon: lucide.Smartphone(
				Class("size-5"),
			),
			Order: 50,
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderSettingsSection(basePath, currentApp)
			},
		},
	}
}

// SettingsPages returns full settings pages for the new sidebar layout
func (e *DashboardExtension) SettingsPages() []ui.SettingsPage {
	return []ui.SettingsPage{
		{
			ID:            "multisession",
			Label:         "Multi-Session",
			Description:   "Configure multi-session behavior and limits",
			Icon:          lucide.Smartphone(Class("h-5 w-5")),
			Category:      "security",
			Order:         20,
			Path:          "multisession",
			RequirePlugin: "multisession",
			RequireAdmin:  true,
		},
	}
}

// DashboardWidgets returns widgets to show on the main dashboard
func (e *DashboardExtension) DashboardWidgets() []ui.DashboardWidget {
	return []ui.DashboardWidget{
		{
			ID:    "multisession-stats",
			Title: "Active Sessions",
			Icon: lucide.Smartphone(
				Class("size-5"),
			),
			Order: 30,
			Size:  1, // 1 column
			Renderer: func(basePath string, currentApp *app.App) g.Node {
				return e.RenderDashboardWidget(basePath, currentApp)
			},
		},
	}
}

// ServeMultiSessionPage renders the multi-session management page with dashboard layout
func (e *DashboardExtension) ServeMultiSessionPage(c forge.Context) error {
	// Get the dashboard handler for rendering
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	// Extract user from context (set by middleware)
	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, "/api/auth/dashboard/login")
	}

	// Extract app ID from URL
	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	// Build minimal PageData - RenderWithLayout will automatically prepopulate:
	// - UserApps and ShowAppSwitcher
	// - CurrentEnvironment and UserEnvironments
	// - ShowEnvSwitcher
	// - EnabledPlugins, Year, and extension items
	pageData := components.PageData{
		Title:      "Multi-Session Management",
		User:       currentUser,
		ActivePage: "multisession",
		BasePath:   handler.GetBasePath(),
		CurrentApp: currentApp,
	}

	// Render page content
	content := e.renderPageContent(c, currentApp)

	// Use dashboard's RenderWithLayout to wrap in dashboard chrome
	return handler.RenderWithLayout(c, pageData, content)
}

// renderPageContent renders the main content for the multisession page
func (e *DashboardExtension) renderPageContent(c forge.Context, currentApp *app.App) g.Node {
	// Fetch real session data
	ctx := c.Request().Context()

	// Get all sessions for the app
	sessionsResp, err := e.plugin.service.sessionSvc.ListSessions(ctx, &session.ListSessionsFilter{
		AppID: currentApp.ID,
		PaginationParams: pagination.PaginationParams{
			Limit:  100,
			Offset: 0,
		},
	})

	var sessions []*session.Session
	var totalSessions int64
	if err == nil && sessionsResp != nil {
		sessions = sessionsResp.Data
		if sessionsResp.Pagination != nil {
			totalSessions = sessionsResp.Pagination.Total
		}
	}

	// Calculate stats
	activeDevices := e.countUniqueDevices(sessions)
	avgSessionsPerUser := float64(0)
	if len(sessions) > 0 {
		uniqueUsers := e.countUniqueUsers(sessions)
		if uniqueUsers > 0 {
			avgSessionsPerUser = float64(len(sessions)) / float64(uniqueUsers)
		}
	}

	return Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("Multi-Session Management")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("View and manage multiple active sessions across devices")),
			),
			// Refresh button
			Button(
				Type("button"),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 focus:outline-none focus:ring-2 focus:ring-violet-500"),
				g.Attr("onclick", "window.location.reload()"),
				lucide.RefreshCw(Class("size-4")),
				g.Text("Refresh"),
			),
		),

		// Stats cards
		Div(
			Class("grid gap-6 md:grid-cols-3"),
			e.statsCard("Total Sessions", fmt.Sprintf("%d", totalSessions), ""),
			e.statsCard("Active Devices", fmt.Sprintf("%d", activeDevices), ""),
			e.statsCard("Avg Sessions/User", fmt.Sprintf("%.1f", avgSessionsPerUser), ""),
		),

		// Sessions table
		e.renderSessionsTable(sessions, currentApp),
	)
}

// Helper methods using dashboard handler

func (e *DashboardExtension) getUserFromContext(c forge.Context) *user.User {
	handler := e.registry.GetHandler()
	if handler == nil {
		return nil
	}
	return handler.GetUserFromContext(c)
}

func (e *DashboardExtension) extractAppFromURL(c forge.Context) (*app.App, error) {
	handler := e.registry.GetHandler()
	if handler == nil {
		return nil, forge.NewHTTPError(http.StatusInternalServerError, "handler not available")
	}

	currentApp, err := handler.GetCurrentApp(c)
	if err != nil {
		return nil, err
	}

	return currentApp, nil
}

// RenderSettingsSection renders the settings section for multi-session
func (e *DashboardExtension) RenderSettingsSection(basePath string, currentApp *app.App) g.Node {
	cfg := e.plugin.config

	return Form(
		g.Attr("method", "POST"),
		g.Attr("action", basePath+"/dashboard/app/"+currentApp.ID.String()+"/multisession/settings"),
		Class("space-y-4"),

		// Max sessions per user
		Div(
			Label(
				Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Attr("for", "maxSessionsPerUser"),
				g.Text("Max Sessions Per User"),
			),
			Input(
				Type("number"),
				Name("maxSessionsPerUser"),
				ID("maxSessionsPerUser"),
				Value(strconv.Itoa(cfg.MaxSessionsPerUser)),
				g.Attr("min", "1"),
				g.Attr("max", "100"),
				Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
				g.Text("Maximum number of concurrent sessions a user can have")),
		),

		// Session expiry
		Div(
			Label(
				Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Attr("for", "sessionExpiryHours"),
				g.Text("Session Expiry (Hours)"),
			),
			Input(
				Type("number"),
				Name("sessionExpiryHours"),
				ID("sessionExpiryHours"),
				Value(strconv.Itoa(cfg.SessionExpiryHours)),
				g.Attr("min", "1"),
				g.Attr("max", "8760"),
				Class("mt-1 block w-full rounded-md border-slate-300 shadow-sm focus:border-violet-500 focus:ring-violet-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"),
				g.Text("Session expiry time in hours")),
		),

		// Device tracking
		Div(
			Label(
				Class("flex items-center space-x-3"),
				Input(
					Type("checkbox"),
					Name("enableDeviceTracking"),
					ID("enableDeviceTracking"),
					Value("true"),
					g.If(cfg.EnableDeviceTracking, Checked()),
					Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700"),
				),
				Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Enable Device Tracking")),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400 ml-6"),
				g.Text("Track device information for each session")),
		),

		// Cross-platform
		Div(
			Label(
				Class("flex items-center space-x-3"),
				Input(
					Type("checkbox"),
					Name("allowCrossPlatform"),
					ID("allowCrossPlatform"),
					Value("true"),
					g.If(cfg.AllowCrossPlatform, Checked()),
					Class("rounded border-slate-300 text-violet-600 focus:ring-violet-500 dark:border-gray-700"),
				),
				Span(Class("text-sm font-medium text-slate-700 dark:text-gray-300"),
					g.Text("Allow Cross-Platform Sessions")),
			),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400 ml-6"),
				g.Text("Allow sessions across different platforms")),
		),

		// Save button
		Div(
			Class("flex justify-end"),
			Button(
				Type("submit"),
				Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700 focus:outline-none focus:ring-2 focus:ring-violet-500 focus:ring-offset-2"),
				g.Text("Save Settings"),
			),
		),
	)
}

// RenderDashboardWidget renders the dashboard widget showing session stats
func (e *DashboardExtension) RenderDashboardWidget(basePath string, currentApp *app.App) g.Node {
	// This is called during page render, so we can't easily inject context
	// We'll show a placeholder and let the full page show real data
	// In a real implementation, you'd pass context or make this async

	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),

		// Header
		Div(
			Class("flex items-center justify-between mb-4"),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text("Active Sessions")),
			lucide.Smartphone(
				Class("size-5 text-violet-600 dark:text-violet-400"),
			),
		),

		// Stats
		Div(
			Class("space-y-3"),
			Div(
				Div(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("—")),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Total active sessions")),
			),
			P(Class("text-xs text-slate-500 dark:text-gray-500"),
				g.Text("View detailed stats on the multi-session page")),
		),

		// View more link
		Div(
			Class("mt-4 pt-4 border-t border-slate-200 dark:border-gray-800"),
			A(
				Href(basePath+"/dashboard/app/"+currentApp.ID.String()+"/multisession"),
				Class("text-sm font-medium text-violet-600 hover:text-violet-700 dark:text-violet-400 dark:hover:text-violet-300"),
				g.Text("View all sessions →"),
			),
		),
	)
}

// statsCard renders a stats card component
func (e *DashboardExtension) statsCard(label, value, change string) g.Node {
	card := Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		P(Class("text-sm font-medium text-slate-600 dark:text-gray-400"),
			g.Text(label)),
		Div(
			Class("mt-2 flex items-baseline"),
			Span(Class("text-2xl font-semibold text-slate-900 dark:text-white"),
				g.Text(value)),
		),
	)

	if change != "" {
		card = Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			P(Class("text-sm font-medium text-slate-600 dark:text-gray-400"),
				g.Text(label)),
			Div(
				Class("mt-2 flex items-baseline"),
				Span(Class("text-2xl font-semibold text-slate-900 dark:text-white"),
					g.Text(value)),
				Span(Class("ml-2 text-sm font-medium text-green-600 dark:text-green-400"),
					g.Text(change)),
			),
		)
	}

	return card
}

// renderSessionsTable renders the sessions table
func (e *DashboardExtension) renderSessionsTable(sessions []*session.Session, currentApp *app.App) g.Node {
	if len(sessions) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			Div(
				Class("text-center py-12"),
				lucide.Smartphone(
					Class("mx-auto size-12 text-slate-400 dark:text-gray-600 mb-4"),
				),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"),
					g.Text("No Active Sessions")),
				P(Class("text-slate-600 dark:text-gray-400"),
					g.Text("There are no active sessions for this app yet.")),
			),
		)
	}

	return Div(
		Class("rounded-lg border border-slate-200 bg-white shadow-sm dark:border-gray-800 dark:bg-gray-900 overflow-hidden"),

		// Table header
		Div(
			Class("px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
			H2(Class("text-xl font-semibold text-slate-900 dark:text-white"),
				g.Text("Active Sessions")),
		),

		// Table
		Div(
			Class("overflow-x-auto"),
			Table(
				Class("w-full"),
				THead(
					Class("bg-slate-50 dark:bg-gray-800/50"),
					Tr(
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("User")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Device")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("IP Address")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Created")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Expires")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
							g.Text("Actions")),
					),
				),
				TBody(
					Class("bg-white dark:bg-gray-900 divide-y divide-slate-200 dark:divide-gray-800"),
					g.Group(e.renderSessionRows(sessions, currentApp)),
				),
			),
		),
	)
}

// renderSessionRows renders individual session table rows
func (e *DashboardExtension) renderSessionRows(sessions []*session.Session, currentApp *app.App) []g.Node {
	rows := make([]g.Node, 0, len(sessions))

	for _, sess := range sessions {
		rows = append(rows, e.renderSessionRow(sess, currentApp))
	}

	return rows
}

// renderSessionRow renders a single session table row
func (e *DashboardExtension) renderSessionRow(sess *session.Session, currentApp *app.App) g.Node {
	createdAt := sess.CreatedAt.Format("Jan 2, 2006 15:04")
	expiresAt := sess.ExpiresAt.Format("Jan 2, 2006 15:04")

	// Get device info if available
	deviceInfo := "Unknown"
	ipAddress := sess.IPAddress
	if ipAddress == "" {
		ipAddress = "N/A"
	}

	// Fetch user info (in production, you'd batch this)
	userEmail := sess.UserID.String()

	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800/50"),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("flex items-center"),
				Div(
					Class("text-sm font-medium text-slate-900 dark:text-white"),
					g.Text(userEmail),
				),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(deviceInfo),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(ipAddress),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(createdAt),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(expiresAt),
			),
		),
		Td(Class("px-6 py-4 whitespace-nowrap text-right text-sm font-medium"),
			Form(
				g.Attr("method", "POST"),
				g.Attr("action", "/api/auth/dashboard/app/"+currentApp.ID.String()+"/multisession/revoke/"+sess.ID.String()),
				g.Attr("onsubmit", "return confirm('Are you sure you want to revoke this session?')"),
				Class("inline"),
				Button(
					Type("submit"),
					Class("inline-flex items-center gap-1 text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"),
					lucide.Trash2(Class("size-4")),
					g.Text("Revoke"),
				),
			),
		),
	)
}

// Action handlers

// RevokeSession handles session revocation
func (e *DashboardExtension) RevokeSession(c forge.Context) error {
	sessionIDStr := c.Param("sessionId")
	if sessionIDStr == "" {
		return c.String(http.StatusBadRequest, "Session ID required")
	}

	sessionID, err := xid.FromString(sessionIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid session ID")
	}

	ctx := c.Request().Context()
	if err := e.plugin.service.sessionSvc.RevokeByID(ctx, sessionID); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to revoke session")
	}

	// Redirect back to multisession page
	currentApp, _ := e.extractAppFromURL(c)
	if currentApp != nil {
		return c.Redirect(http.StatusFound, "/api/auth/dashboard/app/"+currentApp.ID.String()+"/multisession")
	}

	return c.Redirect(http.StatusFound, "/api/auth/dashboard/")
}

// RevokeAllUserSessions handles revoking all sessions for a user
func (e *DashboardExtension) RevokeAllUserSessions(c forge.Context) error {
	userIDStr := c.Param("userId")
	if userIDStr == "" {
		return c.String(http.StatusBadRequest, "User ID required")
	}

	userID, err := xid.FromString(userIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user ID")
	}

	ctx := c.Request().Context()

	// Get all user sessions
	sessionsResp, err := e.plugin.service.sessionSvc.ListSessions(ctx, &session.ListSessionsFilter{
		UserID: &userID,
		PaginationParams: pagination.PaginationParams{
			Limit:  1000,
			Offset: 0,
		},
	})

	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to fetch user sessions")
	}

	// Revoke all sessions
	if sessionsResp != nil {
		for _, sess := range sessionsResp.Data {
			_ = e.plugin.service.sessionSvc.RevokeByID(ctx, sess.ID)
		}
	}

	// Redirect back to multisession page
	currentApp, _ := e.extractAppFromURL(c)
	if currentApp != nil {
		return c.Redirect(http.StatusFound, "/api/auth/dashboard/app/"+currentApp.ID.String()+"/multisession")
	}

	return c.Redirect(http.StatusFound, "/api/auth/dashboard/")
}

// ServeSettings renders the multisession settings page
func (e *DashboardExtension) ServeSettings(c forge.Context) error {
	handler := e.registry.GetHandler()
	if handler == nil {
		return c.String(http.StatusInternalServerError, "Dashboard handler not available")
	}

	currentUser := e.getUserFromContext(c)
	if currentUser == nil {
		return c.Redirect(http.StatusFound, handler.GetBasePath()+"/dashboard/login")
	}

	currentApp, err := e.extractAppFromURL(c)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid app context")
	}

	content := e.renderSettingsContent(currentApp, handler.GetBasePath())

	// Use the settings layout with sidebar navigation
	return handler.RenderSettingsPage(c, "multisession", content)
}

// SaveSettings handles saving multisession settings
func (e *DashboardExtension) SaveSettings(c forge.Context) error {
	// Parse form data
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "Invalid form data")
	}

	form := c.Request().Form

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

	// Update plugin config (in-memory for now)
	// In production, you'd persist this to a database or config file
	e.plugin.config.MaxSessionsPerUser = maxSessions
	e.plugin.config.SessionExpiryHours = sessionExpiry
	e.plugin.config.EnableDeviceTracking = enableDeviceTracking
	e.plugin.config.AllowCrossPlatform = allowCrossPlatform

	// Redirect back to settings page
	currentApp, _ := e.extractAppFromURL(c)
	if currentApp != nil {
		return c.Redirect(http.StatusFound, "/api/auth/dashboard/app/"+currentApp.ID.String()+"/settings/multisession?saved=true")
	}

	return c.Redirect(http.StatusFound, "/api/auth/dashboard/settings/multisession?saved=true")
}

// renderSettingsContent renders the settings page content with header and form
func (e *DashboardExtension) renderSettingsContent(currentApp *app.App, basePath string) g.Node {
	return Div(
		Class("space-y-6"),

		// Header
		Div(
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Multi-Session Settings")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Configure multi-session behavior and session limits")),
		),

		// Settings form
		Div(
			Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
			e.RenderSettingsSection(basePath, currentApp),
		),
	)
}

// Helper methods for stats calculation

// countUniqueDevices counts unique user agents as a proxy for devices
// Since Session doesn't have DeviceID, we use UserAgent as an approximation
func (e *DashboardExtension) countUniqueDevices(sessions []*session.Session) int {
	deviceMap := make(map[string]bool)
	for _, sess := range sessions {
		if sess.UserAgent != "" {
			deviceMap[sess.UserAgent] = true
		}
	}
	return len(deviceMap)
}

// countUniqueUsers counts unique user IDs in sessions
func (e *DashboardExtension) countUniqueUsers(sessions []*session.Session) int {
	userMap := make(map[string]bool)
	for _, sess := range sessions {
		if !sess.UserID.IsNil() {
			userMap[sess.UserID.String()] = true
		}
	}
	return len(userMap)
}
