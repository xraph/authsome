package components

import (
	"fmt"
	"strings"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/environment"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Header renders the page header with navigation
func DashboardHeader(data PageData) g.Node {
	return Header(
		ID("page-header"),
		Class("z-10 w-full sticky -top-14 shadow-sm backdrop-blur-md"),
		Div(
			Class("z-10 flex flex-none items-center bg-white/50 dark:bg-gray-900/50 "),
			Div(
				Class("container mx-auto px-4 lg:px-8"),
				Div(
					Class("-mx-4 px-4 rounded-none lg:-mx-6 lg:px-6"),
					Div(
						Class("flex justify-between py-1.5 lg:py-2.5"),

						// Left Section - Logo and Desktop Nav
						Div(
							Class("flex items-center gap-2 lg:gap-6"),
							Logo(data.BasePath, data.CurrentApp),
						),

						// Right Section - App Switcher, Environment Switcher, Theme Toggle, User Dropdown, Mobile Nav Toggle
						Div(
							Class("flex items-center gap-2"),
							g.If(data.ShowAppSwitcher, AppSwitcher(data)),
							g.If(data.ShowEnvSwitcher, EnvironmentSwitcher(data)),
							ThemeToggle(),
							UserDropdown(data),
							MobileNavToggle(),
						),
					),

					// Mobile Navigation
					MobileNavigation(data),
				),
			),
		),
		g.If(data.ShowAppSwitcher,
			Div(
				Class("z-10 flex flex-none items-center border-b border-slate-200 dark:border-gray-800 bg-white/50 dark:bg-gray-900/50 "),
				Div(
					Class("container mx-auto px-4 lg:px-8"),
					Div(
						Class("-mx-4 px-4 rounded-none lg:-mx-6 lg:px-6"),
						Div(
							Class("flex justify-between py-1 lg:py-1.5"),

							// Left Section - Logo and Desktop Nav
							Div(
								Class("flex items-center gap-2 lg:gap-6"),
								DesktopNavigation(data),
							),
						),

						// Mobile Navigation
						MobileNavigation(data),
					),
				),
			),
		),
	)
}

// Footer renders the page footer
func DashboardFooter(data PageData) g.Node {
	return Footer(
		ID("page-footer"),
		Class("flex flex-none items-center py-5"),
		Div(
			Class("container mx-auto flex flex-col px-4 text-center text-sm md:flex-row md:justify-between md:text-start lg:px-8"),
			Div(
				Class("pt-4 pb-1 md:pb-4 text-slate-600 dark:text-gray-400"),
				Span(Class("font-medium"), g.Text("AuthSome")),
				g.Text(" © "),
				g.Text(time.Now().Format("2006")),
			),
			Div(
				Class("inline-flex items-center justify-center pt-1 pb-4 md:pt-4 text-slate-600 dark:text-gray-400"),
				Span(g.Text("Powered by")),
				heartIcon(),
				Span(g.Text("Forge Framework")),
			),
		),
	)
}

func Logo(basePath string, currentApp *app.App) g.Node {
	// Logo always links to dashboard index (app list or redirect)
	logoURL := basePath + "/dashboard/"

	return A(
		Href(logoURL),
		Class("group inline-flex items-center gap-1.5 text-lg font-bold tracking-wide text-slate-900 dark:text-white hover:text-violet-600 dark:hover:text-violet-400"),
		shieldCheckIcon(),
		Span(
			g.Text("Auth"),
			Span(Class("font-normal"), g.Text("Some")),
		),
	)
}

func DesktopNavigation(data PageData) g.Node {
	// Build URLs with appId if we have a current app
	var dashURL, usersURL, environmentsURL, sessionsURL, pluginsURL, settingsURL string

	if data.CurrentApp != nil {
		appIDStr := data.CurrentApp.ID.String()
		dashURL = data.BasePath + "/dashboard/app/" + appIDStr + "/"
		usersURL = data.BasePath + "/dashboard/app/" + appIDStr + "/users"
		// organizationsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/organizations"
		environmentsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/environments"
		sessionsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/sessions"
		pluginsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/plugins"
		settingsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/settings"
	} else {
		// Fallback to index if no app context (shouldn't happen in app-scoped pages)
		dashURL = data.BasePath + "/dashboard/"
		usersURL = dashURL
		// organizationsURL = dashURL
		environmentsURL = dashURL
		sessionsURL = dashURL
		pluginsURL = dashURL
		settingsURL = dashURL
	}

	// Core navigation items
	navItems := []g.Node{
		navLink("Dashboard", dashURL, data.ActivePage == "dashboard"),
		navLink("Users", usersURL, data.ActivePage == "users"),
		// g.If(data.EnabledPlugins["organization"], navLink("Organizations", organizationsURL, data.ActivePage == "organizations")),
		navLink("Environments", environmentsURL, data.ActivePage == "environments"),
		g.If(!data.EnabledPlugins["multisession"], navLink("Sessions", sessionsURL, data.ActivePage == "sessions")),
	}

	// Add extension navigation items for main position
	if data.ExtensionNavItems != nil {
		for _, item := range data.ExtensionNavItems {
			navItems = append(navItems, item)
		}
	}

	// Add core settings/plugins at the end
	navItems = append(navItems,
		navLink("Plugins", pluginsURL, data.ActivePage == "plugins"),
		navLink("Settings", settingsURL, data.ActivePage == "settings"),
	)

	return Nav(
		Class("hidden items-center gap-1.5 lg:flex"),
		g.Group(navItems),
	)
}

func MobileNavigation(data PageData) g.Node {
	// Build URLs with appId if we have a current app
	var dashURL, usersURL, organizationsURL, environmentsURL, appsManagementURL, sessionsURL, pluginsURL, settingsURL string

	if data.CurrentApp != nil {
		appIDStr := data.CurrentApp.ID.String()
		dashURL = data.BasePath + "/dashboard/app/" + appIDStr + "/"
		usersURL = data.BasePath + "/dashboard/app/" + appIDStr + "/users"
		organizationsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/organizations"
		environmentsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/environments"
		appsManagementURL = data.BasePath + "/dashboard/app/" + appIDStr + "/apps-management"
		sessionsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/sessions"
		pluginsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/plugins"
		settingsURL = data.BasePath + "/dashboard/app/" + appIDStr + "/settings"
	} else {
		// Fallback to index if no app context
		dashURL = data.BasePath + "/dashboard/"
		usersURL = dashURL
		organizationsURL = dashURL
		environmentsURL = dashURL
		appsManagementURL = dashURL
		sessionsURL = dashURL
		pluginsURL = dashURL
		settingsURL = dashURL
	}

	navItems := []g.Node{
		mobileNavLink("Dashboard", dashURL, data.ActivePage == "dashboard"),
		mobileNavLink("Users", usersURL, data.ActivePage == "users"),
		mobileNavLink("Organizations", organizationsURL, data.ActivePage == "organizations"),
		mobileNavLink("Environments", environmentsURL, data.ActivePage == "environments"),
		mobileNavLink("Apps", appsManagementURL, data.ActivePage == "apps-management"),
		mobileNavLink("Sessions", sessionsURL, data.ActivePage == "sessions"),
		mobileNavLink("Plugins", pluginsURL, data.ActivePage == "plugins"),
		mobileNavLink("Settings", settingsURL, data.ActivePage == "settings"),
	}

	return Div(
		g.Attr("x-cloak", ""),
		g.Attr("x-show", "mobileNavOpen"),
		Class("lg:hidden"),
		Nav(
			Class("flex flex-col gap-2 border-t border-slate-200 dark:border-gray-800 py-4"),
			g.Group(navItems),
		),
	)
}

func AppSwitcher(data PageData) g.Node {
	if data.CurrentApp == nil || len(data.UserApps) < 1 {
		return g.Text("") // Don't show if no app or only one app
	}

	currentAppName := data.CurrentApp.Name

	return Div(
		Class("relative inline-block"),
		g.Attr("x-data", "{ appDropdownOpen: false }"),
		Button(
			g.Attr("@click", "appDropdownOpen = !appDropdownOpen"),
			g.Attr(":aria-expanded", "appDropdownOpen"),
			Type("button"),
			Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm leading-5 font-semibold text-slate-800 dark:text-gray-300 hover:border-violet-300 dark:hover:border-violet-700 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
			g.Attr("aria-haspopup", "true"),
			g.Attr("title", "Switch App"),
			folderOpenIcon(),
			Span(Class("hidden lg:inline max-w-[150px] truncate"), g.Text(currentAppName)),
			chevronDownIcon(),
		),
		appSwitcherDropdown(data),
	)
}

func appSwitcherDropdown(data PageData) g.Node {
	appLinks := []g.Node{}
	for _, app := range data.UserApps {
		isCurrentApp := data.CurrentApp != nil && app.ID == data.CurrentApp.ID
		appLinks = append(appLinks, appSwitcherLink(app, data.BasePath, isCurrentApp))
	}

	return Div(
		g.Attr("x-cloak", ""),
		g.Attr("x-show", "appDropdownOpen"),
		g.Attr("x-transition:enter", "transition ease-out duration-100"),
		g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
		g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
		g.Attr("x-transition:leave", "transition ease-in duration-75"),
		g.Attr("x-transition:leave-start", "opacity-100 scale-100"),
		g.Attr("x-transition:leave-end", "opacity-0 scale-95"),
		g.Attr("@click.outside", "appDropdownOpen = false"),
		g.Attr("role", "menu"),
		Class("absolute start-0 z-50 mt-2 w-64 rounded-lg shadow-xl origin-top-left"),
		Div(
			Class("divide-y divide-slate-100 dark:divide-gray-700 rounded-lg bg-white dark:bg-gray-800 ring-1 ring-black/5 dark:ring-white/10 max-h-96 overflow-y-auto"),
			Div(
				Class("px-3 py-2 border-b border-slate-100 dark:border-gray-700"),
				P(Class("text-xs font-semibold text-slate-500 dark:text-gray-400 uppercase"), g.Text("Switch App")),
			),
			Div(
				Class("space-y-1 p-2"),
				g.Group(appLinks),
			),
		),
	)
}

func appSwitcherLink(app *app.App, basePath string, isActive bool) g.Node {
	href := basePath + "/dashboard/app/" + app.ID.String() + "/"

	activeClass := ""
	if isActive {
		activeClass = "bg-violet-50 dark:bg-violet-900/20 text-violet-600 dark:text-violet-400"
	}

	return A(
		g.Attr("role", "menuitem"),
		Href(href),
		Class("group flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors "+activeClass),
		Div(
			Class("flex-shrink-0 w-8 h-8 rounded bg-primary/10 dark:bg-primary/20 flex items-center justify-center"),
			Span(Class("text-xs font-bold text-primary"), g.Text(string(app.Name[0]))),
		),
		Div(
			Class("flex-1 min-w-0"),
			P(Class("font-semibold truncate"), g.Text(app.Name)),
			P(Class("text-xs text-slate-500 dark:text-gray-400 truncate"), g.Text("@"+app.Slug)),
		),
		g.If(isActive, checkCircleIcon()),
	)
}

func EnvironmentSwitcher(data PageData) g.Node {
	fmt.Println("EnvironmentSwitcher", data.CurrentEnvironment, data.UserEnvironments)
	if data.CurrentEnvironment == nil || len(data.UserEnvironments) < 1 {
		return g.Text("") // Don't show if no environment or only one environment
	}

	currentEnvName := data.CurrentEnvironment.Name
	currentEnvType := data.CurrentEnvironment.Type

	return Div(
		Class("relative inline-block"),
		g.Attr("x-data", "{ envDropdownOpen: false }"),
		Button(
			g.Attr("@click", "envDropdownOpen = !envDropdownOpen"),
			g.Attr(":aria-expanded", "envDropdownOpen"),
			Type("button"),
			Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm leading-5 font-semibold text-slate-800 dark:text-gray-300 hover:border-violet-300 dark:hover:border-violet-700 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
			g.Attr("aria-haspopup", "true"),
			g.Attr("title", "Switch Environment"),
			layersIcon(),
			Span(Class("hidden lg:inline max-w-[150px] truncate"), g.Text(currentEnvName)),
			environmentBadge(currentEnvType),
			chevronDownIcon(),
		),
		environmentSwitcherDropdown(data),
	)
}

func environmentSwitcherDropdown(data PageData) g.Node {
	envLinks := []g.Node{}
	for _, env := range data.UserEnvironments {
		isCurrentEnv := data.CurrentEnvironment != nil && env.ID == data.CurrentEnvironment.ID
		envLinks = append(envLinks, environmentSwitcherLink(env, data.BasePath, data.CurrentApp.ID.String(), isCurrentEnv))
	}

	return Div(
		g.Attr("x-cloak", ""),
		g.Attr("x-show", "envDropdownOpen"),
		g.Attr("x-transition:enter", "transition ease-out duration-100"),
		g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
		g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
		g.Attr("x-transition:leave", "transition ease-in duration-75"),
		g.Attr("x-transition:leave-start", "opacity-100 scale-100"),
		g.Attr("x-transition:leave-end", "opacity-0 scale-95"),
		g.Attr("@click.outside", "envDropdownOpen = false"),
		g.Attr("role", "menu"),
		Class("absolute start-0 z-50 mt-2 w-64 rounded-lg shadow-xl origin-top-left"),
		Div(
			Class("divide-y divide-slate-100 dark:divide-gray-700 rounded-lg bg-white dark:bg-gray-800 ring-1 ring-black/5 dark:ring-white/10 max-h-96 overflow-y-auto"),
			Div(
				Class("px-3 py-2 border-b border-slate-100 dark:border-gray-700"),
				P(Class("text-xs font-semibold text-slate-500 dark:text-gray-400 uppercase"), g.Text("Switch Environment")),
			),
			Div(
				Class("space-y-1 p-2"),
				g.Group(envLinks),
			),
		),
	)
}

func environmentSwitcherLink(env *environment.Environment, basePath string, appIDStr string, isActive bool) g.Node {
	// Use a form POST to switch environment
	formID := "env-switch-" + env.ID.String()

	activeClass := ""
	if isActive {
		activeClass = "bg-violet-50 dark:bg-violet-900/20 text-violet-600 dark:text-violet-400"
	}

	return Div(
		Form(
			ID(formID),
			Method("POST"),
			Action(basePath+"/dashboard/app/"+appIDStr+"/environment/switch"),
			Class("hidden"),
			Input(Type("hidden"), Name("env_id"), Value(env.ID.String())),
		),
		Button(
			Type("button"),
			g.Attr("onclick", "document.getElementById('"+formID+"').submit()"),
			g.Attr("role", "menuitem"),
			Class("w-full group flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors "+activeClass),
			Div(
				Class("flex-shrink-0 w-8 h-8 rounded bg-primary/10 dark:bg-primary/20 flex items-center justify-center"),
				Span(Class("text-xs font-bold text-primary"), g.Text(string(env.Name[0]))),
			),
			Div(
				Class("flex-1 min-w-0"),
				Div(Class("flex items-center gap-2"),
					P(Class("font-semibold truncate"), g.Text(env.Name)),
					environmentBadge(env.Type),
				),
				P(Class("text-xs text-slate-500 dark:text-gray-400 truncate"), g.Text(env.Slug)),
			),
			g.If(isActive, checkCircleIcon()),
		),
	)
}

func environmentBadge(envType string) g.Node {
	badgeColor := "bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
	switch strings.ToLower(envType) {
	case "production":
		badgeColor = "bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400"
	case "staging":
		badgeColor = "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400"
	case "development":
		badgeColor = "bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400"
	}

	return Span(
		Class("px-2 py-0.5 text-xs font-semibold rounded-full "+badgeColor),
		g.Text(envType),
	)
}

func ThemeToggle() g.Node {
	return Button(
		g.Attr("@click", "toggleTheme()"),
		Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm leading-5 font-semibold text-slate-800 dark:text-gray-300 hover:border-violet-300 dark:hover:border-violet-700 hover:text-violet-800 dark:hover:text-violet-400 active:border-slate-200 transition-colors btn btn-ghost btn-sm btn-circle"),
		g.Attr(":title", "isDark ? 'Switch to light mode' : 'Switch to dark mode'"),
		sunIcon(),
		moonIcon(),
	)
}

func UserDropdown(data PageData) g.Node {
	userName := "User"
	if data.User != nil && data.User.Name != "" {
		userName = data.User.Name
	}

	initial := "U"

	if len(userName) > 0 {
		names := strings.Split(userName, " ")
		initial = string(names[0][0])
		if len(names) > 1 {
			initial += string(names[1][0])
		}
		initial = strings.ToUpper(initial)
	}

	return Div(
		Class("relative inline-block"),
		Button(
			g.Attr("@click", "userDropdownOpen = !userDropdownOpen"),
			g.Attr(":aria-expanded", "userDropdownOpen"),
			Type("button"),
			Class("avatar avatar-placeholder"),
			g.Attr("aria-haspopup", "true"),
			// userCircleIcon(),
			// Span(Class("hidden sm:inline"), g.Text(userName)),
			// chevronDownIcon(),
			Div(
				Class("bg-neutral text-neutral-content w-8 rounded-full"),
				Span(Class("text-xs text-bold"), g.Text(initial)),
			),
		),
		userDropdownMenu(data),
	)
}

func userDropdownMenu(data PageData) g.Node {
	userEmail := ""
	userName := ""
	userID := ""
	if data.User != nil {
		userEmail = data.User.Email
		userName = data.User.Name
		userID = data.User.ID.String()
	}

	return Div(
		g.Attr("x-cloak", ""),
		g.Attr("x-show", "userDropdownOpen"),
		g.Attr("x-transition:enter", "transition ease-out duration-100"),
		g.Attr("x-transition:enter-start", "opacity-0"),
		g.Attr("x-transition:enter-end", "opacity-100"),
		g.Attr("x-transition:leave", "transition ease-in duration-100"),
		g.Attr("x-transition:leave-start", "opacity-100"),
		g.Attr("x-transition:leave-end", "opacity-0"),
		g.Attr("@click.outside", "userDropdownOpen = false"),
		g.Attr("role", "menu"),
		Class("absolute end-0 z-50 mt-2 w-48 rounded-lg shadow-xl origin-top-right"),
		Div(
			Class("divide-y divide-slate-100 dark:divide-gray-700 rounded-lg bg-white dark:bg-gray-800 ring-1 ring-black/5 dark:ring-white/10"),
			Div(
				Class("space-y-1 p-2.5"),
				g.If(data.User != nil,
					Div(
						Class("px-2.5 py-2 text-xs text-slate-500 dark:text-gray-400 border-b border-slate-100 dark:border-gray-700"),
						P(Class("font-semibold text-slate-900 dark:text-white"), g.Text(userName)),
						P(Class("truncate"), g.Text(userEmail)),
					),
				),
				A(
					g.Attr("role", "menuitem"),
					Href(data.BasePath+"/dashboard/users/"+userID),
					Class("group flex items-center justify-between gap-2 rounded-lg px-2.5 py-2 text-sm font-medium text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400"),
					userCircleIconSmall(),
					Span(Class("grow"), g.Text("My Profile")),
				),
			),
			Div(
				Class("space-y-1 p-2.5"),
				A(
					g.Attr("role", "menuitem"),
					Href(data.BasePath+"/dashboard/logout"),
					Class("group flex items-center justify-between gap-2 rounded-lg px-2.5 py-2 text-sm font-medium text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400"),
					lockIcon(),
					Span(Class("grow"), g.Text("Sign out")),
				),
			),
		),
	)
}

func MobileNavToggle() g.Node {
	return Div(
		Class("lg:hidden"),
		Button(
			g.Attr("@click", "mobileNavOpen = !mobileNavOpen"),
			Type("button"),
			Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm leading-5 font-semibold text-slate-800 dark:text-gray-300 hover:border-violet-300 dark:hover:border-violet-700 hover:text-violet-800 dark:hover:text-violet-400 active:border-slate-200"),
			menuIcon(),
		),
	)
}

func navLink(text, href string, active bool) g.Node {
	activeClass := "text-slate-800 font-normal dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-600 dark:hover:text-violet-400"
	if active {
		activeClass = "font-medium bg-violet-50 dark:bg-violet-900/20 text-violet-600 dark:text-violet-400"
	}

	return A(
		Href(href),
		Class("group flex items-center gap-2 tracking-wide rounded-lg px-2.5 py-1.5 text-xs transition-colors "+activeClass),
		Span(g.Text(text)),
	)
}

func mobileNavLink(text, href string, active bool) g.Node {
	activeClass := "text-slate-800 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-600 dark:hover:text-violet-400"
	if active {
		activeClass = "border border-violet-50 dark:border-violet-900/30 bg-violet-50 dark:bg-violet-900/20 text-violet-600 dark:text-violet-400"
	}

	return A(
		Href(href),
		Class("group flex items-center gap-2 rounded-lg px-2.5 py-1.5 text-sm font-medium "+activeClass),
		Span(g.Text(text)),
	)
}

// Icon components using Lucide
func shieldCheckIcon() g.Node {
	return lucide.ShieldCheck(
		Class("inline-block size-5 text-violet-600 dark:text-violet-400 transition group-hover:scale-110"),
	)
}

func sunIcon() g.Node {
	return g.El("div",
		g.Attr("x-show", "isDark"),
		g.Attr("x-cloak", ""),
		lucide.Sun(Class("h-5 w-5")),
	)
}

func moonIcon() g.Node {
	return g.El("div",
		g.Attr("x-show", "!isDark"),
		g.Attr("x-cloak", ""),
		lucide.Moon(Class("h-5 w-5")),
	)
}

func userCircleIcon() g.Node {
	return lucide.User(
		Class("inline-block size-5 sm:hidden"),
	)
}

func chevronDownIcon() g.Node {
	return lucide.ChevronDown(
		Class("hidden size-5 opacity-40 sm:inline-block transition-transform"),
		g.Attr(":class", "{ 'rotate-180': userDropdownOpen }"),
	)
}

func userCircleIconSmall() g.Node {
	return lucide.User(
		Class("inline-block size-5 flex-none opacity-25 group-hover:opacity-50"),
	)
}

func lockIcon() g.Node {
	return lucide.Lock(
		Class("inline-block size-5 flex-none opacity-25 group-hover:opacity-50"),
	)
}

func menuIcon() g.Node {
	return lucide.Menu(
		Class("inline-block size-5"),
	)
}

func heartIcon() g.Node {
	return lucide.Heart(
		Class("mx-1 inline-block size-4 text-red-600 dark:text-red-500"),
		g.Attr("fill", "currentColor"),
	)
}

func folderOpenIcon() g.Node {
	return lucide.FolderOpen(
		Class("inline-block size-5"),
	)
}

func checkCircleIcon() g.Node {
	return lucide.CircleCheck(
		Class("inline-block size-5 flex-none text-violet-600 dark:text-violet-400"),
	)
}

func layersIcon() g.Node {
	return lucide.Layers(
		Class("inline-block size-4 flex-none"),
	)
}

// func userCircleIconSmall() g.Node {
// 	return lucide.UserCircle(
// 		Class("inline-block size-4 flex-none opacity-50 group-hover:opacity-100"),
// 	)
// }
