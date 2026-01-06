package layouts

import (
	"fmt"

	"github.com/xraph/authsome/plugins/dashboard/components"
	"github.com/xraph/forgeui"
	"github.com/xraph/forgeui/alpine"
	"github.com/xraph/forgeui/components/breadcrumb"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/separator"
	"github.com/xraph/forgeui/components/sidebar"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/layout"
	"github.com/xraph/forgeui/router"
	"github.com/xraph/forgeui/theme"
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

const (
	LayoutRoot      = "root"
	LayoutDashboard = "dashboard"
	LayoutSettings  = "settings"
	LayoutApp       = "app"
)

type mainNavigationItem struct {
	Label  string
	Icon   g.Node
	URL    string
	Active bool
}

type LayoutManager struct {
	fuiApp     *forgeui.App
	baseUIPath string

	Title       string
	Description string
	Keywords    []string
	Author      string
	Copyright   string
	Robots      string
	Canonical   string

	ogTitle               string
	ogDescription         string
	ogImage               string
	ogUrl                 string
	ogType                string
	ogLocale              string
	ogSiteName            string
	ogLocaleAlternate     string
	ogLocaleAlternateName string
	ogLocaleAlternateUrl  string
	ogLocaleAlternateType string
}

func NewLayoutManager(fuiApp *forgeui.App, baseUIPath string) *LayoutManager {
	return &LayoutManager{
		fuiApp:     fuiApp,
		baseUIPath: baseUIPath,
	}
}

func (l *LayoutManager) RegisterLayouts() error {
	// Register root layout - defines HTML structure for ALL pages
	l.fuiApp.RegisterLayout(LayoutRoot, l.RootLayout)

	// Register dashboard layout - inherits from root, adds sidebar
	l.fuiApp.RegisterLayout(LayoutDashboard, l.DashboardLayout, router.WithParentLayout(LayoutRoot))

	// Register settings layout - inherits from root, adds sidebar
	l.fuiApp.RegisterLayout(LayoutSettings, l.SettingsLayout, router.WithParentLayout(LayoutDashboard))

	// Register app layout - inherits from root, adds sidebar
	l.fuiApp.RegisterLayout(LayoutApp, l.AppLayout, router.WithParentLayout(LayoutRoot))

	return nil
}

// RootLayout - THE ONLY layout with full HTML structure
// This is the single source of truth for head/body configuration
func (l *LayoutManager) RootLayout(ctx *router.PageContext, content g.Node) g.Node {
	// Type assert to get app (interface{} to avoid circular dependency)
	app := ctx.App().(*forgeui.App)

	// Build metadata nodes conditionally
	var metaNodes []g.Node
	if ctx.Meta != nil {
		metaNodes = []g.Node{
			layout.Title(ctx.Meta.Title),
			layout.Description(ctx.Meta.Description),
		}
	}

	return layout.Build(
		layout.Head(
			layout.Meta("viewport", "width=device-width, initial-scale=1"),
			layout.Charset("utf-8"),

			theme.HeadContent(theme.DefaultLight(), theme.DefaultDark()),
			html.TitleEl(g.Text("ForgeUI - Dashboard")),

			// Single source of truth for theme
			layout.Theme(app.LightTheme(), app.DarkTheme()),

			// Alpine.js cloak CSS
			layout.Alpine(),

			// Tailwind CSS (using CDN for demo simplicity)
			// In production, use: app.Assets.StyleSheet("css/tailwind.css")
			html.Script(
				html.Src("https://cdn.tailwindcss.com"),
			),
			theme.TailwindConfigScript(),
			alpine.CloakCSS(),
			theme.StyleTag(theme.DefaultLight(), theme.DefaultDark()),
			html.StyleEl(g.Raw(`
				@layer base {
					* {
						@apply border-border;
					}
				}
			`)),

			// Custom stylesheets
			layout.Styles(
				app.Assets.StyleSheet("css/tailwind.css"),
			),

			// Page-specific metadata
			g.Group(metaNodes),
		),

		layout.Body(
			layout.Class("min-h-screen bg-background text-foreground antialiased"),
			g.Attr("x-data", ""), // Alpine initialization

			// Dark mode script
			layout.DarkModeScript(),

			// Child content rendered here (from child layouts or pages)
			content,

			// Scripts at end of body
			layout.Scripts(
				layout.AlpineScripts(alpine.PluginCollapse),
				// Alpine scripts with Collapse plugin
				alpine.Scripts(alpine.PluginCollapse),
			),

			// Auto-inject based on app configuration
			g.If(app.IsDev(), layout.HotReload()),
			g.If(app.HasBridge(), layout.BridgeClient()),
		),
	)
}

func (l *LayoutManager) DashboardLayout(ctx *router.PageContext, content g.Node) g.Node {
	return html.Div(
		html.Class("flex min-h-screen"),

		// Sidebar with all patterns
		l.dashboardSidebar(ctx),

		// Main content area
		sidebar.SidebarInset(
			// Header with breadcrumb
			sidebar.SidebarInsetHeader(
				sidebar.SidebarTriggerDesktop(),
				separator.Separator(separator.Vertical(), separator.WithClass("mr-2 h-4")),
				breadcrumb.Breadcrumb(
					breadcrumb.Item("/", g.Text("Dashboard")),
					breadcrumb.Item("/analytics", g.Text("Analytics")),
					breadcrumb.Page(g.Text("Overview")),
				),
				// Theme toggle in header
				html.Div(
					html.Class("ml-auto"),
					button.Button(
						g.Group([]g.Node{
							alpine.XOn("click", "darkMode = !darkMode"),
							alpine.XText("darkMode ? '☀️' : '🌙'"),
						}),
						button.WithVariant(forgeui.VariantGhost),
						button.WithSize(forgeui.SizeIcon),
					),
				),
			),

			// Main content
			html.Main(content),
		),
	)
}

func (l *LayoutManager) SettingsLayout(ctx *router.PageContext, content g.Node) g.Node {
	return html.Div(
		html.Class("flex min-h-screen"),
		content,
	)
}

func (l *LayoutManager) AppLayout(ctx *router.PageContext, content g.Node) g.Node {
	return html.Div(
		html.Class("flex min-h-screen"),
		content,
	)
}

func (l *LayoutManager) dashboardSidebar(ctx *router.PageContext) g.Node {

	// Build URLs with appId if we have a current app
	var dashURL, usersURL, organizationsURL, environmentsURL, appsManagementURL, sessionsURL, pluginsURL, settingsURL string
	// Fallback to index if no app context
	dashURL = l.baseUIPath + "/"
	usersURL = dashURL
	organizationsURL = dashURL
	environmentsURL = dashURL
	appsManagementURL = dashURL
	sessionsURL = dashURL
	pluginsURL = dashURL
	settingsURL = dashURL

	pageDataRaw, ok := ctx.Get("pageData")
	if ok {
		pageData := pageDataRaw.(components.PageData)
		if pageData.CurrentApp != nil {
			appIDStr := pageData.CurrentApp.ID.String()
			dashURL = l.baseUIPath + "/app/" + appIDStr + "/"
			usersURL = l.baseUIPath + "/app/" + appIDStr + "/users"
			organizationsURL = l.baseUIPath + "/app/" + appIDStr + "/organizations"
			environmentsURL = l.baseUIPath + "/app/" + appIDStr + "/environments"
			appsManagementURL = l.baseUIPath + "/app/" + appIDStr + "/apps-management"
			sessionsURL = l.baseUIPath + "/app/" + appIDStr + "/sessions"
			pluginsURL = l.baseUIPath + "/app/" + appIDStr + "/plugins"
			settingsURL = l.baseUIPath + "/app/" + appIDStr + "/settings"
		}
	}

	mainNavigationItems := []mainNavigationItem{
		{Label: "Dashboard", Icon: icons.LayoutDashboard(icons.WithSize(20)), URL: dashURL},
		{Label: "Users", Icon: icons.Activity(icons.WithSize(20)), URL: usersURL},
		{Label: "Organizations", Icon: icons.Users(icons.WithSize(20)), URL: organizationsURL},
		{Label: "Environments", Icon: icons.Box(icons.WithSize(20)), URL: environmentsURL},
		{Label: "Apps", Icon: icons.Box(icons.WithSize(20)), URL: appsManagementURL},
		{Label: "Sessions", Icon: icons.Box(icons.WithSize(20)), URL: sessionsURL},
	}

	settingsNavigationItems := []mainNavigationItem{
		// {Label: "Plugins", Icon: icons.Box(icons.WithSize(20)), URL: pluginsURL},
		{Label: "Settings", Icon: icons.Cog(icons.WithSize(20)), URL: settingsURL},
	}

	var mainNavs []g.Node
	for _, item := range mainNavigationItems {
		mainNavs = append(
			mainNavs,
			sidebar.SidebarMenuItem(
				sidebar.SidebarMenuButton(
					item.Label,
					sidebar.WithMenuHref(item.URL),
					sidebar.WithMenuIcon(item.Icon),
					// sidebar.WithActive(item.Active),
					sidebar.WithMenuSize(sidebar.MenuButtonSizeDefault),
				),
			),
		)
	}

	var settingsNavs []g.Node
	for _, item := range settingsNavigationItems {
		settingsNavs = append(
			settingsNavs,
			sidebar.SidebarMenuItem(
				sidebar.SidebarMenuButton(item.Label, sidebar.WithMenuHref(item.URL), sidebar.WithMenuIcon(item.Icon), sidebar.WithMenuSize(sidebar.MenuButtonSizeDefault)),
			),
		)
	}

	fmt.Println("pluginsURL", pluginsURL)

	return sidebar.SidebarWithOptions(
		[]sidebar.SidebarOption{
			sidebar.WithDefaultCollapsed(false),
			sidebar.WithCollapsible(true),
			sidebar.WithCollapsibleMode(sidebar.CollapsibleIcon), // Use icon mode to show icons when collapsed
		},
		// Header with logo - icon always visible, text hidden when collapsed
		sidebar.SidebarHeader(
			html.Div(
				html.Class("flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground shrink-0"),
				icons.LayoutDashboard(icons.WithSize(20)),
			),
			html.Span(
				html.Class("font-bold text-lg"),
				g.Attr("x-data", "{}"),
				g.Attr("x-show", "$store.sidebar && (!$store.sidebar.collapsed || $store.sidebar.isMobile)"),
				g.Text("Acme Inc."),
			),
		),

		// Main content
		sidebar.SidebarContent(
			// Platform section
			sidebar.SidebarGroup(
				sidebar.SidebarGroupLabel("Platform"),
				sidebar.SidebarMenu(
					mainNavs...,
				),
			),

			// Projects section (collapsible) d
			sidebar.SidebarGroupCollapsible(
				[]sidebar.SidebarGroupOption{
					sidebar.WithGroupKey("plugins"),
					sidebar.WithGroupDefaultOpen(true),
				},
				sidebar.SidebarGroupLabelCollapsible("plugins", "Plugins", icons.FolderKanban(icons.WithSize(16))),
				sidebar.SidebarGroupContent("plugins",
					sidebar.SidebarMenu(
						sidebar.SidebarMenuItem(
							sidebar.SidebarMenuButton(
								"Overview",
								sidebar.WithMenuHref(pluginsURL),
								sidebar.WithMenuIcon(icons.Box(icons.WithSize(20))),
							),
						),
						sidebar.SidebarMenuItem(
							sidebar.SidebarMenuButton(
								"Design Engineering",
								sidebar.WithMenuHref("/projects/design"),
								sidebar.WithMenuIcon(icons.FolderKanban(icons.WithSize(20))),
							),
							sidebar.SidebarMenuAction(
								icons.EllipsisVertical(icons.WithSize(16)),
								"More options",
							),
							// Submenu
							sidebar.SidebarMenuSub(
								sidebar.SidebarMenuSubItem(
									sidebar.SidebarMenuSubButton("Overview", "/projects/design/overview", false),
								),
								sidebar.SidebarMenuSubItem(
									sidebar.SidebarMenuSubButton("Tasks", "/projects/design/tasks", false),
								),
								sidebar.SidebarMenuSubItem(
									sidebar.SidebarMenuSubButton("Settings", "/projects/design/settings", false),
								),
							),
						),
					),
				),
			),

			// // Projects section (collapsible) d
			// sidebar.SidebarGroupCollapsible(
			// 	[]sidebar.SidebarGroupOption{
			// 		sidebar.WithGroupKey("projects"),
			// 		sidebar.WithGroupDefaultOpen(true),
			// 	},
			// 	sidebar.SidebarGroupLabelCollapsible("projects", "Projects", icons.FolderKanban(icons.WithSize(16))),
			// 	sidebar.SidebarGroupContent("projects",
			// 		sidebar.SidebarMenu(
			// 			sidebar.SidebarMenuItem(
			// 				sidebar.SidebarMenuButton(
			// 					"Design Engineering",
			// 					sidebar.WithMenuHref("/projects/design"),
			// 					sidebar.WithMenuIcon(icons.FolderKanban(icons.WithSize(20))),
			// 				),
			// 				sidebar.SidebarMenuAction(
			// 					icons.EllipsisVertical(icons.WithSize(16)),
			// 					"More options",
			// 				),
			// 				// Submenu
			// 				sidebar.SidebarMenuSub(
			// 					sidebar.SidebarMenuSubItem(
			// 						sidebar.SidebarMenuSubButton("Overview", "/projects/design/overview", false),
			// 					),
			// 					sidebar.SidebarMenuSubItem(
			// 						sidebar.SidebarMenuSubButton("Tasks", "/projects/design/tasks", false),
			// 					),
			// 					sidebar.SidebarMenuSubItem(
			// 						sidebar.SidebarMenuSubButton("Settings", "/projects/design/settings", false),
			// 					),
			// 				),
			// 			),
			// 			sidebar.SidebarMenuItem(
			// 				sidebar.SidebarMenuButton(
			// 					"Sales & Marketing",
			// 					sidebar.WithMenuHref("/projects/sales"),
			// 					sidebar.WithMenuIcon(icons.FolderKanban(icons.WithSize(20))),
			// 				),
			// 				sidebar.SidebarMenuAction(
			// 					icons.EllipsisVertical(icons.WithSize(16)),
			// 					"More options",
			// 				),
			// 			),
			// 			sidebar.SidebarMenuItem(
			// 				sidebar.SidebarMenuButton(
			// 					"Travel",
			// 					sidebar.WithMenuHref("/projects/travel"),
			// 					sidebar.WithMenuIcon(icons.FolderKanban(icons.WithSize(20))),
			// 				),
			// 				sidebar.SidebarMenuAction(
			// 					icons.EllipsisVertical(icons.WithSize(16)),
			// 					"More options",
			// 				),
			// 			),
			// 		),
			// 	),
			// ),

			// Settings section
			sidebar.SidebarGroup(
				sidebar.SidebarGroupLabel("Settings"),
				sidebar.SidebarMenu(
					settingsNavs...,
				),
			),
		),

		// Footer with user profile
		sidebar.SidebarFooter(
			sidebar.SidebarMenu(
				sidebar.SidebarMenuItem(
					// User dropdown
					html.Div(
						alpine.XData(map[string]any{"userMenuOpen": false}),
						html.Class("relative"),
						sidebar.SidebarMenuButton(
							"John Doe",
							sidebar.WithMenuIcon(
								html.Div(
									html.Class("flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground"),
									g.Text("JD"),
								),
							),
							sidebar.WithMenuAsButton(),
							sidebar.WithMenuAttrs(
								alpine.XOn("click", "userMenuOpen = !userMenuOpen"),
							),
						),
						// Dropdown menu
						html.Div(
							g.Attr("x-show", "userMenuOpen"),
							alpine.XOn("click.away", "userMenuOpen = false"),
							g.Attr("x-transition", ""),
							html.Class("absolute bottom-full left-0 mb-2 w-56 rounded-md border bg-popover p-1 shadow-md"),
							html.Div(
								html.Class("px-2 py-1.5 text-sm font-semibold"),
								g.Text("john@example.com"),
							),
							separator.Separator(separator.WithClass("my-1")),
							// dropdownMenuItem("/profile", "Profile", icons.User(icons.WithSize(16))),
							// dropdownMenuItem("/settings", "Settings", icons.Settings(icons.WithSize(16))),
							// dropdownMenuItem("/help", "Help", icons.Info(icons.WithSize(16))),
							separator.Separator(separator.WithClass("my-1")),
							// dropdownMenuItem("/logout", "Log out", icons.LogOut(icons.WithSize(16))),
						),
					),
				),
			),
		),

		// Toggle button
		sidebar.SidebarToggle(),

		// Rail for better UX
		sidebar.SidebarRail(),
	)
}
