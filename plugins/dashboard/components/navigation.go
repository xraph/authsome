package components

import (
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Header renders the page header with navigation
func DashboardHeader(data PageData) g.Node {
	return Header(
		ID("page-header"),
		Class("z-10 flex flex-none items-center pt-5"),
		Div(
			Class("container mx-auto px-4 lg:px-8 xl:max-w-7xl"),
			Div(
				Class("-mx-4 border-y border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 px-4 shadow-sm sm:rounded-lg sm:border lg:-mx-6 lg:px-6"),
				Div(
					Class("flex justify-between py-2.5 lg:py-3.5"),

					// Left Section - Logo and Desktop Nav
					Div(
						Class("flex items-center gap-2 lg:gap-6"),
						Logo(data.BasePath),
						DesktopNavigation(data),
					),

					// Right Section - Theme Toggle, User Dropdown, Mobile Nav Toggle
					Div(
						Class("flex items-center gap-2"),
						ThemeToggle(),
						UserDropdown(data),
						MobileNavToggle(),
					),
				),

				// Mobile Navigation
				MobileNavigation(data),
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
			Class("container mx-auto flex flex-col px-4 text-center text-sm md:flex-row md:justify-between md:text-start lg:px-8 xl:max-w-7xl"),
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

func Logo(basePath string) g.Node {
	return A(
		Href(basePath+"/dashboard/"),
		Class("group inline-flex items-center gap-1.5 text-lg font-bold tracking-wide text-slate-900 dark:text-white hover:text-violet-600 dark:hover:text-violet-400"),
		shieldCheckIcon(),
		Span(
			g.Text("Auth"),
			Span(Class("font-normal"), g.Text("Some")),
		),
	)
}

func DesktopNavigation(data PageData) g.Node {
	navItems := []g.Node{
		navLink("Dashboard", data.BasePath+"/dashboard/", data.ActivePage == "dashboard"),
		navLink("Users", data.BasePath+"/dashboard/users", data.ActivePage == "users"),
		navLink("Sessions", data.BasePath+"/dashboard/sessions", data.ActivePage == "sessions"),
	}

	// Add Organizations link if in SaaS mode
	if data.IsSaaSMode {
		navItems = append(navItems, navLink("Organizations", data.BasePath+"/dashboard/organizations", data.ActivePage == "organizations"))
	}

	navItems = append(navItems, navLink("Settings", data.BasePath+"/dashboard/settings", data.ActivePage == "settings"))

	return Nav(
		Class("hidden items-center gap-1.5 lg:flex"),
		g.Group(navItems),
	)
}

func MobileNavigation(data PageData) g.Node {
	navItems := []g.Node{
		mobileNavLink("Dashboard", data.BasePath+"/dashboard/", data.ActivePage == "dashboard"),
		mobileNavLink("Users", data.BasePath+"/dashboard/users", data.ActivePage == "users"),
		mobileNavLink("Sessions", data.BasePath+"/dashboard/sessions", data.ActivePage == "sessions"),
	}

	// Add Organizations link if in SaaS mode
	if data.IsSaaSMode {
		navItems = append(navItems, mobileNavLink("Organizations", data.BasePath+"/dashboard/organizations", data.ActivePage == "organizations"))
	}

	navItems = append(navItems, mobileNavLink("Settings", data.BasePath+"/dashboard/settings", data.ActivePage == "settings"))

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

func ThemeToggle() g.Node {
	return Button(
		g.Attr("@click", "toggleTheme()"),
		Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm leading-5 font-semibold text-slate-800 dark:text-gray-300 hover:border-violet-300 dark:hover:border-violet-700 hover:text-violet-800 dark:hover:text-violet-400 active:border-slate-200 transition-colors"),
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

	return Div(
		Class("relative inline-block"),
		Button(
			g.Attr("@click", "userDropdownOpen = !userDropdownOpen"),
			g.Attr(":aria-expanded", "userDropdownOpen"),
			Type("button"),
			Class("inline-flex items-center justify-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-sm leading-5 font-semibold text-slate-800 dark:text-gray-300 hover:border-violet-300 dark:hover:border-violet-700 hover:text-violet-800 dark:hover:text-violet-400 active:border-slate-200 transition-colors"),
			g.Attr("aria-haspopup", "true"),
			userCircleIcon(),
			Span(Class("hidden sm:inline"), g.Text(userName)),
			chevronDownIcon(),
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
	activeClass := "text-slate-800 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-600 dark:hover:text-violet-400"
	if active {
		activeClass = "bg-violet-50 dark:bg-violet-900/20 text-violet-600 dark:text-violet-400"
	}

	return A(
		Href(href),
		Class("group flex items-center gap-2 rounded-lg px-2.5 py-1.5 text-sm font-medium transition-colors "+activeClass),
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
