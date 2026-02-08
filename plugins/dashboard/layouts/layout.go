package layouts

import (
	"fmt"
	"strings"

	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/ui"
	"github.com/xraph/authsome/core/user"
	"github.com/xraph/forgeui"
	"github.com/xraph/forgeui/alpine"
	"github.com/xraph/forgeui/assets"
	"github.com/xraph/forgeui/bridge"
	"github.com/xraph/forgeui/components/breadcrumb"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/separator"
	"github.com/xraph/forgeui/components/sidebar"
	"github.com/xraph/forgeui/components/tabs"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/layout"
	"github.com/xraph/forgeui/primitives"
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
	LayoutAuthless  = "authless"
)

type mainNavigationItem struct {
	Label  string
	Icon   g.Node
	URL    string
	Active bool
}

type LayoutManager struct {
	fuiApp            *forgeui.App
	baseUIPath        string
	isMultiAppMode    bool
	extensionRegistry ExtensionRegistryInterface
	enabledPlugins    map[string]bool

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

// ExtensionRegistryInterface defines the minimal interface needed from the extension registry
type ExtensionRegistryInterface interface {
	GetNavigationItems(position ui.NavigationPosition, enabledPlugins map[string]bool) []ui.NavigationItem
	List() []ui.DashboardExtension
}

func NewLayoutManager(fuiApp *forgeui.App, baseUIPath string, isMultiAppMode bool, enabledPlugins map[string]bool) *LayoutManager {
	return &LayoutManager{
		fuiApp:         fuiApp,
		baseUIPath:     baseUIPath,
		isMultiAppMode: isMultiAppMode,
		enabledPlugins: enabledPlugins,
	}
}

// SetExtensionRegistry sets the extension registry for dynamic plugin navigation
func (l *LayoutManager) SetExtensionRegistry(registry ExtensionRegistryInterface) {
	l.extensionRegistry = registry
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

	// Register authless layout - inherits from root, adds sidebar
	l.fuiApp.RegisterLayout(LayoutAuthless, l.AuthlessLayout, router.WithParentLayout(LayoutRoot))

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
			html.TitleEl(g.Text("Authsome - Dashboard")),

			// Alpine.js cloak CSS
			layout.Alpine(),

			// In production, use: app.Assets.StyleSheet("css/tailwind.css")
			html.Script(
				html.Src("https://cdn.tailwindcss.com"),
			),

			// Compiled Tailwind CSS v4 with all custom styles
			app.Assets.StyleSheet("css/output.css"),

			layout.Theme(l.fuiApp.LightTheme(), l.fuiApp.DarkTheme()),

			// Theme styles (without Tailwind config which requires Tailwind to be loaded)
			alpine.CloakCSS(),
			theme.StyleTag(theme.DefaultLight(), theme.DefaultDark()),

			// Additional custom stylesheets
			layout.Styles(
				app.Assets.StyleSheet("css/custom.css", assets.WithMedia("text/css")),
			),

			// Page-specific metadata
			g.Group(metaNodes),
		),

		layout.Body(
			layout.Class("min-h-screen bg-background text-foreground antialiased"),

			// Dark mode script
			layout.DarkModeScript(),

			// Alpine global store initialization
			html.Script(
				g.Raw(`
					// Initialize dark mode store before Alpine starts
					document.addEventListener('alpine:init', () => {
						Alpine.store('darkMode', {
							on: document.documentElement.classList.contains('dark'),
							
							toggle() {
								this.on = !this.on;
								document.documentElement.classList.toggle('dark', this.on);
								document.documentElement.setAttribute('data-theme', this.on ? 'dark' : 'light');
								localStorage.setItem('theme', this.on ? 'dark' : 'light');
							}
						});
					});
				`),
			),

			// Child content
			content,

			// Scripts at end of body
			// 1. Bridge client scripts
			g.If(app.HasBridge(),
				bridge.BridgeScripts(bridge.ScriptConfig{
					Endpoint:      l.baseUIPath + "/api/bridge",
					IncludeAlpine: false,
				}),
			),

			// 2. Bridge Alpine integration
			g.If(app.HasBridge(),
				html.Script(
					g.Attr("defer", ""),
					g.Raw(bridge.GetAlpineJS()),
				),
			),

			// ✅ 3. Configure router settings BEFORE Alpine loads
			html.Script(
				g.Raw(`
					// Configure Pinecone Router settings synchronously
					// This runs BEFORE Alpine starts
					document.addEventListener('DOMContentLoaded', () => {
						if (window.PineconeRouter) {
							window.PineconeRouter.settings({
								handleClicks: false  // Disable automatic click interception
							});
						}
					});
				`),
			),

			// 4. Load ALL Alpine plugins together (router + collapse)
			layout.Scripts(
				alpine.Scripts(alpine.PluginCollapse),
			),

			// 5. Register bridge plugin
			g.If(app.HasBridge(),
				html.Script(
					g.Attr("defer", ""),
					g.Raw(`
						document.addEventListener('alpine:init', () => {
							if (window.AlpineBridgePlugin && window.Alpine) {
								window.Alpine.plugin(window.AlpineBridgePlugin);
							}
						});
					`),
				),
			),

			// 6. Hot reload
			g.If(app.IsDev(), layout.HotReload()),
		),
	)
}

func (l *LayoutManager) DashboardLayout(ctx *router.PageContext, content g.Node) g.Node {
	path := ctx.Request.URL.Path
	pattern := strings.Split(path, "/settings")
	isSettingsPage := len(pattern) > 1

	insetHeaderOptions := []sidebar.SidebarInsetHeaderOption{}
	if isSettingsPage {
		insetHeaderOptions = append(insetHeaderOptions, sidebar.WithSidebarInsetHeaderClass("!border-t-0"))
	}

	return html.Div(
		html.Class("flex min-h-screen"),

		// Sidebar with all patterns
		l.dashboardSidebar(ctx),

		// Main content area
		sidebar.SidebarInset(
			// Header with breadcrumb
			sidebar.SidebarInsetHeaderWithOptions(insetHeaderOptions,
				breadcrumb.Breadcrumb(
					l.buildBreadcrumbs(ctx)...,
				),
				// Theme toggle in header
				html.Div(
					html.Class("ml-auto"),
					button.Button(
						g.Group([]g.Node{
							html.Span(g.Attr("x-show", "!$store.darkMode.on"), icons.Moon(icons.WithSize(16))),
							html.Span(g.Attr("x-show", "$store.darkMode.on"), icons.Sun(icons.WithSize(16))),
						}),
						button.WithVariant(forgeui.VariantGhost),
						button.WithSize(forgeui.SizeIcon),
						button.WithAttrs(alpine.XOn("click", "$store.darkMode.toggle()")),
					),
				),
			),

			// Main content
			html.Main(
				g.If(isSettingsPage, content),
				g.If(!isSettingsPage,
					html.Class("px-4 lg:px-8 py-6"),
				),
				g.If(!isSettingsPage,
					primitives.Container(content),
				),
			),
		),
	)
}

func (l *LayoutManager) AuthlessLayout(ctx *router.PageContext, content g.Node) g.Node {
	return html.Div(
		html.Class("flex min-h-screen"),
		content,
	)
}

func (l *LayoutManager) SettingsLayout(ctx *router.PageContext, content g.Node) g.Node {
	return html.Div(
		html.Class("flex flex-col min-h-screen"),

		// Horizontal tab navigation
		l.settingsHorizontalNav(ctx),

		// Main content
		html.Main(
			html.Class("container mx-auto px-4 lg:px-8 py-6"),
			content,
		),
	)
}

func (l *LayoutManager) AppLayout(ctx *router.PageContext, content g.Node) g.Node {
	// Get app context if available
	appID := ctx.Param("appId")

	return html.Div(
		html.Class("min-h-screen bg-background"),

		// Sticky Top Navbar
		l.appNavbar(ctx, appID),

		// Main Content Area
		html.Main(
			html.Class("container mx-auto px-4 lg:px-8 py-6"),
			content,
		),
	)
}

func (l *LayoutManager) dashboardSidebar(ctx *router.PageContext) g.Node {

	// Build URLs with appId if we have a current app
	var dashURL, usersURL, environmentsURL, sessionsURL, pluginsURL, settingsURL string

	// Try to get current app from ForgeUI context first
	var currentApp *app.App
	if appRaw, ok := ctx.Get("currentApp"); ok {
		currentApp = appRaw.(*app.App)
	}

	// Build URLs based on current app
	if currentApp != nil {
		appIDStr := currentApp.ID.String()
		dashURL = l.baseUIPath + "/app/" + appIDStr + "/"
		usersURL = l.baseUIPath + "/app/" + appIDStr + "/users"
		// organizationsURL = l.baseUIPath + "/app/" + appIDStr + "/organizations"
		environmentsURL = l.baseUIPath + "/app/" + appIDStr + "/environments"
		sessionsURL = l.baseUIPath + "/app/" + appIDStr + "/sessions"
		pluginsURL = l.baseUIPath + "/app/" + appIDStr + "/plugins"
		settingsURL = l.baseUIPath + "/app/" + appIDStr + "/settings"
	} else {
		// Fallback to index if no app context
		dashURL = l.baseUIPath + "/"
		usersURL = dashURL
		// organizationsURL = dashURL
		environmentsURL = dashURL
		sessionsURL = dashURL
		pluginsURL = dashURL
		settingsURL = dashURL
	}

	mainNavigationItems := []mainNavigationItem{
		{Label: "Dashboard", Icon: icons.LayoutDashboard(icons.WithSize(20)), URL: dashURL},
		{Label: "Users", Icon: icons.Activity(icons.WithSize(20)), URL: usersURL},
		// {Label: "Organizations", Icon: icons.Users(icons.WithSize(20)), URL: organizationsURL},
		{Label: "Environments", Icon: icons.Box(icons.WithSize(20)), URL: environmentsURL},
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

	return sidebar.SidebarWithOptions(
		[]sidebar.SidebarOption{
			sidebar.WithDefaultCollapsed(false),
			sidebar.WithCollapsible(true),
			sidebar.WithCollapsibleMode(sidebar.CollapsibleIcon), // Use icon mode to show icons when collapsed
		},
		// Header with app switcher
		sidebar.SidebarHeader(
			l.buildAppSwitcher(currentApp),
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

			// Plugins section (collapsible) - dynamically populated from extensions
			l.renderPluginsSection(pluginsURL, currentApp),

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
				l.buildUserDropdown(ctx, currentApp),
			),
		),

		// Toggle button
		sidebar.SidebarToggle(),

		// Rail for better UX
		sidebar.SidebarRail(),
	)
}

func (l *LayoutManager) appNavbar(ctx *router.PageContext, appID string) g.Node {
	// Try to get current app from ForgeUI context first
	var currentApp *app.App
	if appRaw, ok := ctx.Get("currentApp"); ok {
		currentApp = appRaw.(*app.App)
	}

	// Use appID from context if available, otherwise use parameter
	if currentApp != nil {
		appID = currentApp.ID.String()
	}

	// Build navigation URLs
	var dashURL, usersURL, organizationsURL, sessionsURL, settingsURL string
	if appID != "" {
		dashURL = l.baseUIPath + "/app/" + appID
		usersURL = l.baseUIPath + "/app/" + appID + "/users"
		organizationsURL = l.baseUIPath + "/app/" + appID + "/organizations"
		sessionsURL = l.baseUIPath + "/app/" + appID + "/sessions"
		settingsURL = l.baseUIPath + "/app/" + appID + "/settings"
	} else {
		dashURL = l.baseUIPath + "/"
		usersURL = dashURL
		organizationsURL = dashURL
		sessionsURL = dashURL
		settingsURL = dashURL
	}

	return html.Header(
		html.Class("sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60"),
		html.Div(
			html.Class("container mx-auto px-4 lg:px-8"),
			html.Div(
				html.Class("flex h-16 items-center justify-between"),

				// Left Section - Logo and Navigation
				html.Div(
					html.Class("flex items-center gap-6"),

					// Logo/Brand
					html.A(
						html.Href(l.baseUIPath+"/"),
						html.Class("flex items-center gap-2 font-semibold text-lg hover:opacity-80 transition-opacity"),
						html.Div(
							html.Class("flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground"),
							icons.Shield(icons.WithSize(20)),
						),
						html.Span(
							html.Class("hidden sm:inline-block"),
							g.Text("AuthSome"),
						),
					),

					// Desktop Navigation
					g.If(!l.isMultiAppMode,
						html.Nav(
							html.Class("hidden md:flex items-center gap-1"),
							l.navLink("Dashboard", dashURL, ctx.Request.URL.Path == dashURL),
							l.navLink("Users", usersURL, ctx.Request.URL.Path == usersURL),
							l.navLink("Organizations", organizationsURL, ctx.Request.URL.Path == organizationsURL),
							l.navLink("Sessions", sessionsURL, ctx.Request.URL.Path == sessionsURL),
						),
					),
				),

				// Right Section - Theme Toggle, User Menu, Mobile Toggle
				html.Div(
					html.Class("flex items-center gap-2"),

					// App Switcher (if multiapp mode) - only show if isMultiAppMode is true
					g.If(l.isMultiAppMode,
						html.Div(
							g.Attr("x-data", `{
								appMenuOpen: false,
								apps: [],
								currentAppId: '`+appID+`',
								async loadApps() {
									try {
										const result = await $go('getAppsList', {});
										this.apps = result.apps || [];
									} catch (err) {
										console.error('Failed to load apps:', err);
									}
								}
							}`),
							g.Attr("x-init", "loadApps()"),
							html.Div(
								html.Class("relative"),
								button.Button(
									g.Group([]g.Node{
										icons.LayoutGrid(icons.WithSize(16)),
										html.Span(html.Class("hidden sm:inline-block"), g.Text("Apps")),
										icons.ChevronDown(icons.WithSize(16)),
									}),
									button.WithVariant(forgeui.VariantGhost),
									button.WithSize(forgeui.SizeSM),
									button.WithClass("flex items-center gap-2"),
									button.WithAttrs(alpine.XOn("click", "appMenuOpen = !appMenuOpen")),
								),
								// Dropdown Menu
								html.Div(
									g.Attr("x-show", "appMenuOpen"),
									alpine.XOn("click.away", "appMenuOpen = false"),
									g.Attr("x-transition", ""),
									html.Class("absolute right-0 mt-2 w-64 rounded-md border bg-popover shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none max-h-96 overflow-y-auto"),
									html.Div(
										html.Class("p-1"),
										// Apps List
										g.El("template", g.Attr("x-for", "app in apps"),
											html.A(
												g.Attr(":href", "`"+l.baseUIPath+"/app/${app.id}`"),
												html.Class("flex items-center gap-3 px-3 py-2 rounded-sm hover:bg-accent transition-colors cursor-pointer"),
												g.Attr(":class", "app.id === currentAppId ? 'bg-accent' : ''"),
												html.Div(
													g.Attr(":class", `'flex-shrink-0 w-8 h-8 rounded-lg flex items-center justify-center text-sm font-bold text-white ' + getAppGradient(app.name)`),
													g.Attr("x-text", "app.name.charAt(0).toUpperCase()"),
												),
												html.Div(
													html.Class("flex-1 min-w-0"),
													html.Div(
														html.Class("font-medium text-sm truncate"),
														g.Attr("x-text", "app.name"),
													),
													html.Div(
														html.Class("text-xs text-muted-foreground truncate"),
														g.Attr("x-text", "(app.userCount || 0) + ' members'"),
													),
												),
											),
										),
										separator.Separator(separator.WithClass("my-1")),
										html.A(
											html.Href(l.baseUIPath+"/"),
											html.Class("flex items-center gap-2 px-3 py-2 text-sm rounded-sm hover:bg-accent hover:text-accent-foreground transition-colors cursor-pointer"),
											icons.LayoutGrid(icons.WithSize(16)),
											g.Text("View All Apps"),
										),
									),
								),
							),
							// Gradient function for app icons
							html.Script(
								g.Raw(`
									function getAppGradient(name) {
										const gradients = [
											'bg-gradient-to-br from-violet-500 to-purple-600',
											'bg-gradient-to-br from-blue-500 to-cyan-600',
											'bg-gradient-to-br from-emerald-500 to-teal-600',
											'bg-gradient-to-br from-orange-500 to-red-600',
											'bg-gradient-to-br from-pink-500 to-rose-600',
											'bg-gradient-to-br from-indigo-500 to-blue-600',
										];
										const hash = name.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
										return gradients[hash % gradients.length];
									}
								`),
							),
						),
					),

					// Theme toggle in header
					html.Div(
						html.Class("ml-auto"),
						button.Button(
							g.Group([]g.Node{
								html.Span(g.Attr("x-show", "!$store.darkMode.on"), icons.Moon(icons.WithSize(16))),
								html.Span(g.Attr("x-show", "$store.darkMode.on"), icons.Sun(icons.WithSize(16))),
							}),
							button.WithVariant(forgeui.VariantGhost),
							button.WithSize(forgeui.SizeIcon),
							button.WithAttrs(alpine.XOn("click", "$store.darkMode.toggle()")),
						),
					),

					// // Settings Link
					// html.A(
					// 	html.Href(settingsURL),
					// 	html.Class("hidden md:inline-flex"),
					// 	button.Button(
					// 		icons.Settings(icons.WithSize(16)),
					// 		button.WithVariant(forgeui.VariantGhost),
					// 		button.WithSize(forgeui.SizeIcon),
					// 	),
					// ),

					// User Menu
					l.buildNavbarUserDropdown(ctx, settingsURL),

					// Mobile Menu Toggle
					html.Div(
						html.Class("md:hidden"),
						button.Button(
							icons.Menu(icons.WithSize(20)),
							button.WithVariant(forgeui.VariantGhost),
							button.WithSize(forgeui.SizeIcon),
							button.WithAttrs(alpine.XOn("click", "mobileMenuOpen = !mobileMenuOpen")),
						),
					),
				),
			),

			// Mobile Navigation Menu
			html.Div(
				g.Attr("x-data", "{ mobileMenuOpen: false }"),
				html.Div(
					g.Attr("x-show", "mobileMenuOpen"),
					alpine.XOn("click.away", "mobileMenuOpen = false"),
					g.Attr("x-transition", ""),
					html.Class("md:hidden border-t"),
					html.Div(
						html.Class("py-2 space-y-1"),
						l.mobileNavLink("Dashboard", dashURL, ctx.Request.URL.Path == dashURL),
						l.mobileNavLink("Users", usersURL, ctx.Request.URL.Path == usersURL),
						l.mobileNavLink("Organizations", organizationsURL, ctx.Request.URL.Path == organizationsURL),
						l.mobileNavLink("Sessions", sessionsURL, ctx.Request.URL.Path == sessionsURL),
						separator.Separator(separator.WithClass("my-2")),
						l.mobileNavLink("Settings", settingsURL, ctx.Request.URL.Path == settingsURL),
					),
				),
			),
		),
	)
}

// navLink creates a navigation link for desktop
func (l *LayoutManager) navLink(label, href string, active bool) g.Node {
	activeClass := ""
	if active {
		activeClass = "bg-accent text-accent-foreground"
	}

	return html.A(
		html.Href(href),
		html.Class("inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring "+activeClass),
		g.Text(label),
	)
}

// mobileNavLink creates a navigation link for mobile menu
func (l *LayoutManager) mobileNavLink(label, href string, active bool) g.Node {
	activeClass := ""
	if active {
		activeClass = "bg-accent text-accent-foreground"
	}

	return html.A(
		html.Href(href),
		html.Class("block px-4 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground rounded-md mx-2 "+activeClass),
		g.Text(label),
	)
}

// dropdownMenuItem creates a dropdown menu item
func (l *LayoutManager) dropdownMenuItem(href, label string, icon g.Node) g.Node {
	return html.A(
		html.Href(href),
		html.Class("flex items-center gap-2 px-3 py-2 text-sm rounded-sm hover:bg-accent hover:text-accent-foreground transition-colors cursor-pointer"),
		icon,
		g.Text(label),
	)
}

func (l *LayoutManager) SetBaseUIPath(baseUIPath string) {
	l.baseUIPath = baseUIPath
}

// renderPluginsSection renders the plugins section with dynamic navigation items from extensions
func (l *LayoutManager) renderPluginsSection(pluginsURL string, currentApp *app.App) g.Node {
	// Default "Overview" menu item - always shown
	pluginMenuItems := []g.Node{
		sidebar.SidebarMenuItem(
			sidebar.SidebarMenuButton(
				"Overview",
				sidebar.WithMenuHref(pluginsURL),
				sidebar.WithMenuIcon(icons.Box(icons.WithSize(20))),
			),
		),
	}

	// Add dynamic plugin navigation items from extensions
	// Note: Extensions are registered during plugin Start phase, so they should be available here
	if l.extensionRegistry != nil {
		// Get navigation items for all extensions at once
		navItems := l.extensionRegistry.GetNavigationItems(ui.NavPositionMain, make(map[string]bool))

		// Group navigation items by plugin/extension
		// For now, just add them as individual menu items (not grouped)
		for _, item := range navItems {
			// Build URL for the item with currentApp
			itemURL := "#"
			if item.URLBuilder != nil {
				itemURL = item.URLBuilder(l.baseUIPath, currentApp)
			}

			pluginMenuItems = append(pluginMenuItems,
				sidebar.SidebarMenuItem(
					sidebar.SidebarMenuButton(
						item.Label,
						sidebar.WithMenuHref(itemURL),
						sidebar.WithMenuIcon(item.Icon),
					),
				),
			)
		}
	}

	return sidebar.SidebarGroupCollapsible(
		[]sidebar.SidebarGroupOption{
			sidebar.WithGroupKey("plugins"),
			sidebar.WithGroupDefaultOpen(true),
		},
		sidebar.SidebarGroupLabelCollapsible("plugins", "Plugins", icons.FolderKanban(icons.WithSize(16))),
		sidebar.SidebarGroupContent("plugins",
			sidebar.SidebarMenu(pluginMenuItems...),
		),
	)
}

// settingsHorizontalNav renders horizontal tab navigation for settings pages
func (l *LayoutManager) settingsHorizontalNav(ctx *router.PageContext) g.Node {
	// Get current app from context
	var currentApp *app.App
	if appRaw, ok := ctx.Get("currentApp"); ok {
		currentApp = appRaw.(*app.App)
	}

	// Build navigation items
	var navItems []g.Node

	// Core "General" settings tab
	generalURL := l.buildSettingsURL(currentApp, "general")
	navItems = append(navItems, l.settingsTab("General", generalURL, l.isActiveSettingsTab(ctx, "general")))

	// Get navigation items from extension registry
	if l.extensionRegistry != nil {
		// Method 1: Get navigation items with NavPositionSettings
		items := l.extensionRegistry.GetNavigationItems(ui.NavPositionSettings, l.enabledPlugins)
		for _, item := range items {
			itemURL := "#"
			if item.URLBuilder != nil {
				itemURL = item.URLBuilder(l.baseUIPath, currentApp)
			}
			isActive := l.isActiveSettingsTab(ctx, item.ID)
			navItems = append(navItems, l.settingsTab(item.Label, itemURL, isActive))
		}

		// Method 2: Get settings pages (legacy support) and convert to tabs
		// Note: We use a type assertion to access GetSettingsPages if available
		type settingsPagesRegistry interface {
			GetSettingsPages(enabledPlugins map[string]bool) []ui.SettingsPage
		}
		if spRegistry, ok := interface{}(l.extensionRegistry).(settingsPagesRegistry); ok {
			settingsPages := spRegistry.GetSettingsPages(l.enabledPlugins)
			for _, page := range settingsPages {
				pageURL := l.buildSettingsURL(currentApp, page.Path)
				isActive := l.isActiveSettingsTab(ctx, page.ID)
				navItems = append(navItems, l.settingsTab(page.Label, pageURL, isActive))
			}
		}
	}

	return html.Div(
		html.Class("bg-background border-b border-border sticky top-12"),
		html.Div(
			html.Class("container mx-auto px-4 lg:px-8 pt-2"),
			html.Div(
				html.Class("relative overflow-hidden"),
				g.Attr("x-data", `{
					showLeft: false,
					showRight: false,
					checkScroll() {
						const container = this.$refs.scrollContainer;
						this.showLeft = container.scrollLeft > 0;
						this.showRight = container.scrollLeft < container.scrollWidth - container.clientWidth - 10;
					},
					scrollLeft() {
						this.$refs.scrollContainer.scrollBy({ left: -200, behavior: 'smooth' });
					},
					scrollRight() {
						this.$refs.scrollContainer.scrollBy({ left: 200, behavior: 'smooth' });
					}
				}`),
				g.Attr("x-init", "$nextTick(() => checkScroll())"),

				// Left scroll chevron
				html.Button(
					html.Type("button"),
					html.Class("absolute left-0 top-1 z-10 flex items-center justify-center w-8 h-8 rounded-full bg-background border border-border shadow-sm hover:bg-accent hover:border-accent-foreground/20 transition-all"),
					g.Attr("x-show", "showLeft"),
					g.Attr("x-cloak", ""),
					g.Attr("@click", "scrollLeft"),
					g.Attr("aria-label", "Scroll left"),
					g.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m15 18-6-6 6-6"/></svg>`),
				),

				// ForgeUI TabsList component with scrolling wrapper
				html.Div(
					g.Attr("x-ref", "scrollContainer"),
					g.Attr("@scroll.debounce.100ms", "checkScroll"),
					html.Class("overflow-x-auto scroll-smooth mx-10"),
					html.Style("scrollbar-width: none; -ms-overflow-style: none;"),

					tabs.TabsWithOptions([]tabs.Option{
						tabs.WithDefaultTab("General"),
					},
						// Use ForgeUI tabs.TabList component
						tabs.TabListWithOptions(
							[]tabs.TabListOption{
								tabs.WithScrollable(),
								tabs.WithTabListVariant(tabs.TabListVariantUnderline),
							},
							navItems...,
						)),
				),

				// Right scroll chevron
				html.Button(
					html.Type("button"),
					html.Class("absolute right-0 top-1 z-10 flex items-center justify-center w-8 h-8 rounded-full bg-background border border-border shadow-sm hover:bg-accent hover:border-accent-foreground/20 transition-all"),
					g.Attr("x-show", "showRight"),
					g.Attr("x-cloak", ""),
					g.Attr("@click", "scrollRight"),
					g.Attr("aria-label", "Scroll right"),
					g.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>`),
				),
			),
		),
	)
}

// settingsTab creates a link-based tab using ForgeUI tab styling
// Note: ForgeUI tabs.Tab() uses buttons for state switching, but we need links for navigation
// This creates an <a> tag with the exact same styling as ForgeUI tabs.Tab()
func (l *LayoutManager) settingsTab(label, href string, active bool) g.Node {
	// ForgeUI tab classes from tabs.Tab() - line 155 of tabs.go
	baseClasses := "inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1.5 text-sm font-medium ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm flex-1"

	// Determine state
	state := "inactive"
	if active {
		state = "active"
	}

	return tabs.Tab(
		label, g.Text(label),
		tabs.WithHref(href),
		tabs.WithTabClass(baseClasses),
		tabs.WithTabAttrs(g.Attr("role", "tab")),
		tabs.WithTabAttrs(g.Attr("aria-selected", fmt.Sprintf("%t", active))),
		tabs.WithTabAttrs(g.Attr("data-state", state)),
		tabs.WithActive(active),
		tabs.WithTabVariant(tabs.TabVariantUnderline),
		tabs.WithShrink(),
	)

	return html.A(
		html.Href(href),
		html.Class(baseClasses),
		g.Attr("role", "tab"),
		g.Attr("aria-selected", fmt.Sprintf("%t", active)),
		g.Attr("data-state", state),
		g.Text(label),
	)
}

// buildSettingsURL builds a settings page URL based on current app context
func (l *LayoutManager) buildSettingsURL(currentApp *app.App, page string) string {
	if currentApp != nil {
		return l.baseUIPath + "/app/" + currentApp.ID.String() + "/settings/" + page
	}
	return l.baseUIPath + "/settings/" + page
}

// buildBreadcrumbs generates breadcrumb items from URL path
func (l *LayoutManager) buildBreadcrumbs(ctx *router.PageContext) []g.Node {
	// Check for custom breadcrumbs override first
	if customCrumbs, ok := ctx.Get("breadcrumbs"); ok {
		if crumbs, ok := customCrumbs.([]g.Node); ok {
			return crumbs
		}
	}

	path := ctx.Request.URL.Path

	// Remove trailing slash
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// Remove base UI path prefix
	if strings.HasPrefix(path, l.baseUIPath) {
		path = strings.TrimPrefix(path, l.baseUIPath)
	}
	if path == "" {
		path = "/"
	}

	// Start with Dashboard as home
	items := []g.Node{
		breadcrumb.Item(l.baseUIPath+"/", g.Text("Dashboard")),
	}

	// If we're at the root, return just Dashboard
	if path == "/" {
		return items
	}

	// Split path into segments
	segments := strings.Split(strings.Trim(path, "/"), "/")

	// Get current app from context if available
	var currentApp *app.App
	if appRaw, ok := ctx.Get("currentApp"); ok {
		currentApp = appRaw.(*app.App)
	}

	// Build cumulative URL and breadcrumb items
	cumulativeURL := l.baseUIPath

	for i, segment := range segments {
		if segment == "" {
			continue
		}

		cumulativeURL += "/" + segment
		isLast := i == len(segments)-1

		// Check if this segment is an ID
		if isID(segment) {
			// Try to get entity name from context or use ID
			label := l.getEntityLabel(ctx, segments, i, currentApp)

			if isLast {
				// Last item should be a page (non-clickable)
				items = append(items, breadcrumb.Page(g.Text(label)))
			} else {
				// Clickable link
				items = append(items, breadcrumb.Item(cumulativeURL, g.Text(label)))
			}
		} else {
			// Humanize the segment name
			label := humanizeSegment(segment)

			if isLast {
				// Last item should be a page (non-clickable)
				items = append(items, breadcrumb.Page(g.Text(label)))
			} else {
				// Clickable link
				items = append(items, breadcrumb.Item(cumulativeURL, g.Text(label)))
			}
		}
	}

	// Limit breadcrumbs to reasonable length (keep first + last 4)
	if len(items) > 6 {
		// Keep Dashboard + last 4 items + ellipsis
		truncated := []g.Node{items[0]}
		truncated = append(truncated, breadcrumb.Item("", g.Text("...")))
		truncated = append(truncated, items[len(items)-4:]...)
		return truncated
	}

	return items
}

// getEntityLabel attempts to get a friendly label for an entity ID
func (l *LayoutManager) getEntityLabel(ctx *router.PageContext, segments []string, index int, currentApp *app.App) string {
	segment := segments[index]

	// Check what type of entity this is based on the previous segment
	if index > 0 {
		entityType := segments[index-1]

		switch entityType {
		case "app":
			// If we have currentApp and IDs match, use its name
			if currentApp != nil && currentApp.ID.String() == segment {
				return currentApp.Name
			}
			return "App"

		case "users":
			// Check if user data is in context
			if userData, ok := ctx.Get("currentUser"); ok {
				if user, ok := userData.(*struct{ Name string }); ok {
					return user.Name
				}
			}
			return "User"

		case "organizations":
			return "Organization"

		case "sessions":
			return "Session"

		case "environments":
			return "Environment"

		default:
			return humanizeSegment(entityType)
		}
	}

	return segment
}

// humanizeSegment converts URL segments to human-readable labels
func humanizeSegment(segment string) string {
	// Replace hyphens and underscores with spaces
	s := strings.ReplaceAll(segment, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	// Capitalize first letter of each word
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}

	return strings.Join(words, " ")
}

// isID checks if a segment looks like an ID (UUID/XID)
func isID(segment string) bool {
	// XID is 20 characters alphanumeric
	if len(segment) == 20 {
		for _, ch := range segment {
			if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
				return false
			}
		}
		return true
	}

	// UUID is 36 characters with hyphens (8-4-4-4-12)
	if len(segment) == 36 && segment[8] == '-' && segment[13] == '-' && segment[18] == '-' && segment[23] == '-' {
		return true
	}

	return false
}

// buildUserDropdown creates a rich user account dropdown
func (l *LayoutManager) buildUserDropdown(ctx *router.PageContext, currentApp *app.App) g.Node {
	// Get user from context
	userRaw, ok := ctx.Get("user")
	if !ok || userRaw == nil {
		// Return guest dropdown for unauthenticated state
		return l.buildGuestDropdown()
	}

	// Type assert to *user.User
	currentUser, ok := userRaw.(*user.User)
	if !ok || currentUser == nil {
		// Return guest dropdown if type assertion fails
		return l.buildGuestDropdown()
	}

	// Extract user details
	userName := currentUser.Name
	if userName == "" {
		userName = currentUser.Email // Fallback to email if name is empty
	}
	userEmail := currentUser.Email
	userImage := currentUser.Image

	// Build avatar node
	avatarNode := l.buildUserAvatar(userName, userImage)

	// Build dropdown menu URLs
	profileURL := l.baseUIPath + "/profile"
	accountSettingsURL := l.baseUIPath + "/account/settings"
	helpURL := l.baseUIPath + "/help"
	logoutURL := l.baseUIPath + "/auth/logout"

	return sidebar.SidebarMenuItem(
		html.Div(
			g.Attr("x-data", "{ userMenuOpen: false }"),
			html.Class("relative"),

			// Dropdown trigger button
			sidebar.SidebarMenuButton(
				userName,
				sidebar.WithMenuIcon(avatarNode),
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
				html.Class("absolute bottom-full left-0 mb-2 w-64 rounded-md border bg-popover p-1 shadow-md z-50"),

				// User info section
				html.Div(
					html.Class("px-3 py-2 border-b border-border"),
					html.Div(
						html.Class("flex items-center gap-3"),
						// Larger avatar for the dropdown header
						l.buildUserAvatarLarge(userName, userImage),
						html.Div(
							html.Class("flex-1 min-w-0"),
							html.Div(
								html.Class("font-semibold text-sm truncate"),
								g.Text(userName),
							),
							g.If(userEmail != "",
								html.Div(
									html.Class("text-xs text-muted-foreground truncate"),
									g.Text(userEmail),
								),
							),
						),
					),
				),

				// Menu items
				html.Div(
					html.Class("py-1"),
					l.dropdownMenuItem(profileURL, "Profile", icons.User(icons.WithSize(16))),
					l.dropdownMenuItem(accountSettingsURL, "Account Settings", icons.Settings(icons.WithSize(16))),
					l.dropdownMenuItem(helpURL, "Help & Support", icons.Info(icons.WithSize(16))),
				),

				separator.Separator(separator.WithClass("my-1")),

				// Logout
				html.Div(
					html.Class("py-1"),
					l.dropdownMenuItem(logoutURL, "Log out", icons.LogOut(icons.WithSize(16))),
				),
			),
		),
	)
}

// buildUserAvatar creates a small avatar for the sidebar button (8x8)
func (l *LayoutManager) buildUserAvatar(userName, userImage string) g.Node {
	if userImage != "" {
		// Return image avatar
		return html.Img(
			html.Src(userImage),
			html.Alt(userName),
			html.Class("h-8 w-8 rounded-full object-cover"),
		)
	}

	// Return initials avatar
	initials := getInitials(userName)
	return html.Div(
		html.Class("flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-semibold shrink-0"),
		g.Text(initials),
	)
}

// buildUserAvatarLarge creates a larger avatar for the dropdown (10x10)
func (l *LayoutManager) buildUserAvatarLarge(userName, userImage string) g.Node {
	if userImage != "" {
		// Return image avatar
		return html.Img(
			html.Src(userImage),
			html.Alt(userName),
			html.Class("h-10 w-10 rounded-full object-cover shrink-0"),
		)
	}

	// Return initials avatar
	initials := getInitials(userName)
	return html.Div(
		html.Class("flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground text-base font-semibold shrink-0"),
		g.Text(initials),
	)
}

// getInitials extracts initials from full name
func getInitials(name string) string {
	// Trim whitespace
	name = strings.TrimSpace(name)

	if name == "" {
		return "?"
	}

	// Split name into words
	words := strings.Fields(name)
	if len(words) == 0 {
		return "?"
	}

	// Get first letter of first two words
	initials := ""
	for i := 0; i < len(words) && i < 2; i++ {
		if len(words[i]) > 0 {
			initials += strings.ToUpper(string([]rune(words[i])[0]))
		}
	}

	if initials == "" {
		return "?"
	}

	return initials
}

// buildGuestDropdown creates dropdown for non-authenticated users
func (l *LayoutManager) buildGuestDropdown() g.Node {
	loginURL := l.baseUIPath + "/login"
	signupURL := l.baseUIPath + "/signup"

	return sidebar.SidebarMenuItem(
		html.Div(
			html.Class("flex items-center gap-2"),
			button.Button(
				g.Text("Login"),
				button.WithVariant(forgeui.VariantGhost),
				button.WithSize(forgeui.SizeSM),
				button.WithAttrs(g.Attr("onclick", fmt.Sprintf("window.location.href='%s'", loginURL))),
			),
			button.Button(
				g.Text("Sign Up"),
				button.WithVariant(forgeui.VariantDefault),
				button.WithSize(forgeui.SizeSM),
				button.WithAttrs(g.Attr("onclick", fmt.Sprintf("window.location.href='%s'", signupURL))),
			),
		),
	)
}

// buildNavbarUserDropdown creates the user dropdown for the top navbar
func (l *LayoutManager) buildNavbarUserDropdown(ctx *router.PageContext, settingsURL string) g.Node {
	// Get user from context
	userRaw, ok := ctx.Get("user")
	if !ok || userRaw == nil {
		// Return login button for unauthenticated state
		return html.Div(
			html.Class("flex items-center gap-2"),
			button.Button(
				g.Text("Login"),
				button.WithVariant(forgeui.VariantGhost),
				button.WithSize(forgeui.SizeSM),
				button.WithAttrs(g.Attr("onclick", fmt.Sprintf("window.location.href='%s'", l.baseUIPath+"/auth/login"))),
			),
		)
	}

	// Type assert to *user.User
	currentUser, ok := userRaw.(*user.User)
	if !ok || currentUser == nil {
		return html.Div(
			html.Class("flex items-center gap-2"),
			button.Button(
				g.Text("Login"),
				button.WithVariant(forgeui.VariantGhost),
				button.WithSize(forgeui.SizeSM),
				button.WithAttrs(g.Attr("onclick", fmt.Sprintf("window.location.href='%s'", l.baseUIPath+"/auth/login"))),
			),
		)
	}

	// Extract user details
	userName := currentUser.Name
	if userName == "" {
		userName = currentUser.Email
	}
	userEmail := currentUser.Email
	userImage := currentUser.Image

	// Build avatar node
	avatarNode := l.buildUserAvatar(userName, userImage)

	// Build dropdown menu URLs
	profileURL := l.baseUIPath + "/profile"
	logoutURL := l.baseUIPath + "/auth/logout"

	return html.Div(
		html.Class("relative"),
		g.Attr("x-data", "{ userMenuOpen: false }"),

		button.Button(
			g.Group([]g.Node{
				avatarNode,
				html.Span(
					html.Class("hidden sm:inline-block"),
					g.Text(userName),
				),
				icons.ChevronDown(icons.WithSize(16)),
			}),
			button.WithVariant(forgeui.VariantGhost),
			button.WithClass("flex items-center gap-2"),
			button.WithAttrs(alpine.XOn("click", "userMenuOpen = !userMenuOpen")),
		),

		// Dropdown Menu
		html.Div(
			g.Attr("x-show", "userMenuOpen"),
			alpine.XOn("click.away", "userMenuOpen = false"),
			g.Attr("x-transition", ""),
			html.Class("absolute right-0 mt-2 w-56 rounded-md border bg-popover shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none z-50"),
			html.Div(
				html.Class("p-1"),
				// User info
				html.Div(
					html.Class("px-3 py-2 text-sm"),
					html.Div(html.Class("font-medium"), g.Text(userName)),
					g.If(userEmail != "",
						html.Div(html.Class("text-muted-foreground text-xs"), g.Text(userEmail)),
					),
				),
				separator.Separator(separator.WithClass("my-1")),
				l.dropdownMenuItem(profileURL, "Profile", icons.User(icons.WithSize(16))),
				l.dropdownMenuItem(settingsURL, "Settings", icons.Settings(icons.WithSize(16))),
				separator.Separator(separator.WithClass("my-1")),
				l.dropdownMenuItem(logoutURL, "Log out", icons.LogOut(icons.WithSize(16))),
			),
		),
	)
}

// isActiveSettingsTab checks if a settings tab is currently active based on URL path
func (l *LayoutManager) isActiveSettingsTab(ctx *router.PageContext, tabID string) bool {
	path := ctx.Request.URL.Path

	// Check if the path ends with /settings/{tabID}
	// Supports both /settings/page and /app/:appId/settings/page patterns
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1] // Remove trailing slash
	}

	// Extract the last segment after /settings/
	settingsIndex := -1
	for i := len(path) - 1; i >= 0; i-- {
		if i >= 9 && path[i-8:i+1] == "/settings" {
			settingsIndex = i + 1
			break
		}
	}

	if settingsIndex > 0 && settingsIndex < len(path) {
		// Get the segment after /settings/
		remaining := path[settingsIndex:]
		if len(remaining) > 0 && remaining[0] == '/' {
			remaining = remaining[1:]
		}
		// Check if it matches the tabID
		return remaining == tabID
	}

	// If no specific page, check if we're on /settings (general is default)
	if tabID == "general" && (path == l.baseUIPath+"/settings" ||
		(len(path) >= 9 && path[len(path)-9:] == "/settings")) {
		return true
	}

	return false
}

// buildAppSwitcher creates a polished app switcher dropdown
func (l *LayoutManager) buildAppSwitcher(currentApp *app.App) g.Node {
	// Get current app name or use default
	appName := "Select App"
	if currentApp != nil {
		appName = currentApp.Name
	}

	// Get first letter for avatar
	appInitial := "A"
	if len(appName) > 0 {
		appInitial = strings.ToUpper(string([]rune(appName)[0]))
	}

	// Get current app ID as string
	currentAppID := ""
	if currentApp != nil {
		currentAppID = currentApp.ID.String()
	}

	return html.Div(
		html.Class("w-full relative"),
		g.Attr("x-data", fmt.Sprintf(`{
			open: false,
			apps: [],
			loading: true,
			search: '',
			currentAppId: '%s',
			currentApp: { name: '%s' },
			get filteredApps() {
				if (!this.search) return this.apps;
				const searchLower = this.search.toLowerCase();
				return this.apps.filter(app => app.name.toLowerCase().includes(searchLower));
			},
			async loadApps() {
				try {
					const result = await $go('getAppsList', {});
					this.apps = result.apps || [];
					if (this.currentAppId) {
						this.currentApp = this.apps.find(a => a.id === this.currentAppId) || { name: '%s' };
					}
				} catch (err) {
					console.error('Failed to load apps:', err);
				} finally {
					this.loading = false;
				}
			},
			selectApp(appId) {
				this.open = false;
				window.location.href = '%s/app/' + appId;
			},
			getGradientStyle(name) {
				const gradients = [
					'linear-gradient(135deg, #8b5cf6 0%%, #7c3aed 100%%)',
					'linear-gradient(135deg, #3b82f6 0%%, #0ea5e9 100%%)',
					'linear-gradient(135deg, #10b981 0%%, #14b8a6 100%%)',
					'linear-gradient(135deg, #f97316 0%%, #ef4444 100%%)',
					'linear-gradient(135deg, #ec4899 0%%, #f43f5e 100%%)',
					'linear-gradient(135deg, #6366f1 0%%, #3b82f6 100%%)'
				];
				const hash = (name || 'A').split('').reduce((acc, char) => acc + char.charCodeAt(0), 0);
				return gradients[hash %% gradients.length];
			}
		}`, currentAppID, appName, appName, l.baseUIPath)),
		g.Attr("x-init", "loadApps()"),
		alpine.XOn("click.outside", "open = false"),
		alpine.XOn("keydown.escape.window", "open = false"),

		// Trigger button
		html.Button(
			html.Type("button"),
			alpine.XOn("click", "open = !open"),
			html.Class("group flex items-center gap-3 w-full rounded-lg px-2 py-2 transition-all duration-200 hover:bg-accent"),
			g.Attr("aria-label", "Switch app"),
			g.Attr(":aria-expanded", "open"),

			// App icon with gradient
			html.Div(
				html.Class("flex h-9 w-9 items-center justify-center rounded-lg shrink-0 text-sm font-bold text-white shadow-sm"),
				g.Attr(":style", "'background: ' + getGradientStyle(currentApp.name)"),
				g.Attr("x-text", "currentApp.name ? currentApp.name.charAt(0).toUpperCase() : '"+appInitial+"'"),
			),

			// App name and label (hidden when sidebar collapsed)
			html.Div(
				html.Class("flex-1 text-left min-w-0"),
				alpine.XShow("$store.sidebar && (!$store.sidebar.collapsed || $store.sidebar.isMobile)"),
				html.Div(
					html.Class("font-semibold text-sm truncate"),
					g.Attr("x-text", "currentApp.name || '"+appName+"'"),
				),
				html.Div(
					html.Class("text-xs text-muted-foreground truncate"),
					g.Text("Workspace"),
				),
			),

			// Chevron icon (hidden when sidebar collapsed)
			html.Div(
				alpine.XShow("$store.sidebar && (!$store.sidebar.collapsed || $store.sidebar.isMobile)"),
				html.Class("shrink-0 text-muted-foreground transition-transform duration-200"),
				g.Attr(":class", "open ? 'rotate-180' : ''"),
				icons.ChevronDown(icons.WithSize(16)),
			),
		),

		// Dropdown panel
		html.Div(
			alpine.XShow("open"),
			g.Attr("x-transition", ""),
			html.Class("absolute left-0 top-full mt-2 w-[375px] z-50 rounded-xl border border-border bg-popover text-popover-foreground shadow-lg overflow-hidden"),

			// Header with search
			html.Div(
				html.Class("p-3 bg-muted/30"),
				html.Div(
					html.Class("relative"),
					html.Div(
						html.Class("absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground/70"),
						icons.Search(icons.WithSize(15)),
					),
					html.Input(
						html.Type("text"),
						html.Placeholder("Search apps..."),
						alpine.XModel("search"),
						g.Attr("x-ref", "searchInput"),
						html.Class("w-full h-9 pl-9 pr-3 text-sm bg-background border-0 rounded-lg ring-1 ring-border/50 focus:outline-none focus:ring-2 focus:ring-primary/50 placeholder:text-muted-foreground/60"),
					),
				),
			),

			// Apps list
			html.Div(
				html.Class("max-h-64 overflow-y-auto py-2"),

				// Loading state
				g.El("template",
					g.Attr("x-if", "loading"),
					html.Div(
						html.Class("flex items-center justify-center py-12"),
						html.Div(
							html.Class("flex flex-col items-center gap-2"),
							html.Div(
								html.Class("w-5 h-5 border-2 border-primary/30 border-t-primary rounded-full animate-spin"),
							),
							html.Span(
								html.Class("text-xs text-muted-foreground"),
								g.Text("Loading..."),
							),
						),
					),
				),

				// Empty state
				g.El("template",
					g.Attr("x-if", "!loading && filteredApps.length === 0"),
					html.Div(
						html.Class("flex flex-col items-center justify-center py-12 px-4"),
						html.Div(
							html.Class("w-12 h-12 rounded-full bg-muted flex items-center justify-center mb-3"),
							icons.Search(icons.WithSize(20), icons.WithClass("text-muted-foreground/50")),
						),
						html.Span(
							html.Class("text-sm font-medium text-foreground"),
							g.Text("No apps found"),
						),
						html.Span(
							html.Class("text-xs text-muted-foreground mt-1"),
							g.Text("Try a different search term"),
						),
					),
				),

				// Apps list items
				g.El("template",
					g.Attr("x-for", "app in filteredApps"),
					g.Attr(":key", "app.id"),
					html.Button(
						html.Type("button"),
						g.Attr("@click", "selectApp(app.id)"),
						html.Class("group w-full flex items-center gap-3 px-3 py-2.5 mx-2 rounded-lg transition-all duration-150 hover:bg-accent/80 text-left"),
						g.Attr(":class", "app.id === currentAppId ? 'bg-accent' : ''"),

						// App avatar with gradient
						html.Div(
							html.Class("flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-sm font-bold text-white shadow-sm ring-1 ring-white/10"),
							g.Attr(":style", "'background: ' + getGradientStyle(app.name)"),
							g.Attr("x-text", "app.name.charAt(0).toUpperCase()"),
						),

						// App info
						html.Div(
							html.Class("flex-1 min-w-0"),
							html.Div(
								html.Class("font-medium text-sm text-foreground truncate"),
								g.Attr("x-text", "app.name"),
							),
							html.Div(
								html.Class("text-xs text-muted-foreground mt-0.5 flex items-center gap-1"),
								icons.Users(icons.WithSize(11)),
								html.Span(
									g.Attr("x-text", "(app.userCount || 0) + ' users'"),
								),
							),
						),

						// Current indicator
						html.Div(
							g.Attr("x-show", "app.id === currentAppId"),
							html.Class("shrink-0 flex items-center justify-center w-5 h-5 rounded-full bg-primary"),
							icons.Check(icons.WithSize(12), icons.WithClass("text-primary-foreground")),
						),
					),
				),
			),

			// Footer actions
			html.Div(
				html.Class("border-t border-border/50 p-2 bg-muted/20"),
				html.A(
					html.Href(l.baseUIPath+"/"),
					html.Class("flex items-center gap-3 px-3 py-2 text-sm rounded-lg hover:bg-accent transition-colors w-full text-muted-foreground hover:text-foreground"),
					html.Div(
						html.Class("flex items-center justify-center w-8 h-8 rounded-md bg-muted"),
						icons.LayoutGrid(icons.WithSize(16)),
					),
					html.Span(g.Text("View all apps")),
				),
				g.If(l.isMultiAppMode,
					html.A(
						html.Href(l.baseUIPath+"/apps/new"),
						html.Class("flex items-center gap-3 px-3 py-2 text-sm rounded-lg hover:bg-accent transition-colors w-full text-muted-foreground hover:text-foreground mt-1"),
						html.Div(
							html.Class("flex items-center justify-center w-8 h-8 rounded-md bg-muted"),
							icons.Plus(icons.WithSize(16)),
						),
						html.Span(g.Text("Create new app")),
					),
				),
			),
		),
	)
}
