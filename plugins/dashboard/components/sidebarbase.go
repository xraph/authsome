package components

import (
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/environment"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// BaseSidebarLayout renders a sidebar-based layout inspired by Preline CMS template
func BaseSidebarLayout(data PageData, content g.Node) g.Node {
	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			Class("relative min-h-full"),
			g.Attr("x-data", "themeData()"),
			g.Attr("x-init", "initTheme()"),
			g.Attr(":class", "{ 'dark': isDark }"),

			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text(data.Title+" - AuthSome Dashboard")),

				Link(Rel("stylesheet"), Href("https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap")),

				Script(g.Raw(`
					const html = document.querySelector('html');
					const isLightOrAuto = localStorage.getItem('hs_theme') === 'light' || (localStorage.getItem('hs_theme') === 'auto' && !window.matchMedia('(prefers-color-scheme: dark)').matches);
					const isDarkOrAuto = localStorage.getItem('hs_theme') === 'dark' || (localStorage.getItem('hs_theme') === 'auto' && window.matchMedia('(prefers-color-scheme: dark)').matches);
					if (isLightOrAuto && html.classList.contains('dark')) html.classList.remove('dark');
					else if (isDarkOrAuto && html.classList.contains('light')) html.classList.remove('light');
					else if (isDarkOrAuto && !html.classList.contains('dark')) html.classList.add('dark');
					else if (isLightOrAuto && !html.classList.contains('light')) html.classList.add('light');
				`)),

				Link(Rel("stylesheet"), Href("https://cdn.jsdelivr.net/npm/apexcharts/dist/apexcharts.css")),
				StyleEl(g.Raw(`
					.apexcharts-tooltip.apexcharts-theme-light
					{
						background-color: transparent !important;
						border: none !important;
						box-shadow: none !important;
					}
				`)),

				// Compiled Tailwind CSS + Preline UI styles
				Link(Rel("stylesheet"), Href(data.BasePath+"/static/css/dashboard.css")),

				// Alpine.js x-cloak style
				StyleEl(g.Raw(`[x-cloak] { display: none !important; }`)),

				// Load Pines Components and Dashboard JS BEFORE Alpine.js
				Script(Src(data.BasePath+"/static/js/pines-components.js")),
				Script(Src(data.BasePath+"/static/js/dashboard.js")),

				// Bundled JavaScript (Preline UI) - loads before Alpine.js
				Script(Src(data.BasePath+"/static/js/bundle.js")),

				// Alpine.js - Load LAST (components must be defined before Alpine initializes)
				Script(Defer(), Src("https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js")),

				Link(Rel("stylesheet"), Href("https://preline.co/assets/css/main.css?v=3.0.1")),

				// Link(Rel("stylesheet"), Type ("text/css"), Href("https://cdn.jsdelivr.net/npm/daisyui@5")),
				Link(Rel("stylesheet"), Type("text/css"), Href("https://cdn.jsdelivr.net/combine/npm/daisyui@5/base/rootscrollgutter.css,npm/daisyui@5/base/scrollbar.css,npm/daisyui@5/base/properties.css,npm/daisyui@5/components/drawer.css,npm/daisyui@5/components/modal.css,npm/daisyui@5/theme/light.css")),
				// Script(Src("https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4")),
				// Link(Rel("stylesheet"), Href("https://cdn.jsdelivr.net/npm/daisyui@5/themes.css")),
			),

			Body(
				Class("hs-overlay-body-open overflow-hidden bg-gray-100 dark:bg-neutral-900"),
				g.Attr("x-data", `{ 
					sidebarOpen: true, 
					userDropdownOpen: false,
					appDropdownOpen: false,
					envDropdownOpen: false
				}`),

				// Global notification container
				notificationContainer(),

				// Fixed Header
				cmsHeader(data),

				// Main Content with Sidebar
				Main(
					Class("lg:hs-overlay-layout-open:ps-60 transition-all duration-300 lg:fixed lg:inset-0 pt-13 px-3 pb-3"),

					// Sidebar
					cmsSidebar(data),

					// Content Container
					Div(
						Class("h-[calc(100dvh-62px)] lg:h-full overflow-hidden flex flex-col bg-white border border-gray-200 shadow-xs rounded-lg dark:bg-neutral-800 dark:border-neutral-700"),

						// Content Header
						cmsContentHeader(data),

						// Content Body
						Div(
							Class("flex-1 container mx-auto p-2 lg:p-4 flex flex-col overflow-hidden overflow-y-auto [&::-webkit-scrollbar]:w-2 [&::-webkit-scrollbar-track]:bg-gray-100 [&::-webkit-scrollbar-thumb]:bg-gray-300 dark:[&::-webkit-scrollbar-track]:bg-neutral-700 dark:[&::-webkit-scrollbar-thumb]:bg-neutral-500"),
							g.Attr("style", "scrollbar-width: thin;"),

							// Flash Messages
							flashMessages(data),

							// Content
							content,
						),
					),
				),
			),
		),
	)
}

// cmsHeader renders the fixed top header (Preline CMS style)
func cmsHeader(data PageData) g.Node {
	return Header(
		Class("fixed top-0 inset-x-0 flex flex-wrap md:justify-start md:flex-nowrap z-48 lg:z-61 w-full bg-zinc-100 text-sm py-2.5 dark:bg-neutral-900"),
		Nav(
			Class("px-4 sm:px-5.5 flex basis-full items-center w-full mx-auto"),
			Div(
				Class("w-full flex justify-between items-center gap-x-1.5"),

				// Left side - Logo, Sidebar Toggle, App Switcher
				Ul(
					Class("flex items-center gap-1.5"),

					// Logo
					Li(
						Class("inline-flex items-center relative pe-1.5 last:pe-0 last:after:hidden after:absolute after:top-1/2 after:end-0 after:inline-block after:w-px after:h-3.5 after:bg-gray-300 after:rounded-full after:-translate-y-1/2 after:rotate-12 dark:after:bg-neutral-700"),
						A(
							Href(data.BasePath+"/"),
							Class("shrink-0 inline-flex justify-center items-center bg-violet-600 size-8 rounded-md text-xl font-semibold focus:outline-none focus:opacity-80"),
							g.Attr("aria-label", "AuthSome"),
							shieldCheckIcon(),
						),

						// Sidebar Toggle (desktop)
						Button(
							Type("button"),
							Class("p-1.5 size-7.5 inline-flex items-center gap-x-1 text-xs rounded-md border border-transparent text-gray-500 hover:text-gray-800 disabled:opacity-50 disabled:pointer-events-none focus:outline-hidden focus:text-gray-800 dark:text-neutral-500 dark:hover:text-neutral-400 dark:focus:text-neutral-400"),
							g.Attr("@click", "sidebarOpen = !sidebarOpen"),
							g.Attr("aria-haspopup", "dialog"),
							g.Attr("aria-expanded", "false"),
							g.Attr("aria-controls", "hs-pro-sidebar"),
							g.Attr("data-hs-overlay", "#hs-pro-sidebar"),
							g.Attr("aria-label", "Toggle Sidebar"),
							lucide.PanelLeft(Class("shrink-0 size-4")),
						),
					),

					// App Switcher
					g.If(data.ShowAppSwitcher && data.CurrentApp != nil,
						Li(
							Class("inline-flex items-center relative pe-1.5 last:pe-0 last:after:hidden after:absolute after:top-1/2 after:end-0 after:inline-block after:w-px after:h-3.5 after:bg-gray-300 after:rounded-full after:-translate-y-1/2 after:rotate-12 dark:after:bg-neutral-700"),
							cmsAppDropdown(data),
						),
					),

					// Environment Switcher
					g.If(data.ShowEnvSwitcher && data.CurrentEnvironment != nil,
						Li(
							Class("inline-flex items-center relative pe-1.5 last:pe-0 last:after:hidden after:absolute after:top-1/2 after:end-0 after:inline-block after:w-px after:h-3.5 after:bg-gray-300 after:rounded-full after:-translate-y-1/2 after:rotate-12 dark:after:bg-neutral-700"),
							cmsEnvDropdown(data),
						),
					),
				),

				// Right side - Theme Toggle, User Profile
				Ul(
					Class("flex flex-row justify-between items-center gap-x-3 ms-auto"),

					Li(
						Class("inline-flex items-center gap-1.5 relative pe-1.5 last:pe-0 last:after:hidden after:absolute after:top-1/2 after:end-0 after:inline-block after:w-px after:h-3.5 after:bg-gray-300 after:rounded-full after:-translate-y-1/2 after:rotate-12 dark:after:bg-neutral-700"),
						Button(
							Type("button"),
							Class("hidden lg:inline-flex items-center gap-1.5 relative text-gray-500 pe-3 last:pe-0 last:after:hidden after:absolute after:top-1/2 after:end-0 after:inline-block after:w-px after:h-3.5 after:bg-gray-300 after:rounded-full after:-translate-y-1/2 after:rotate-12 dark:text-neutral-200 dark:after:bg-neutral-700"),
							lucide.Brain(Class("shrink-0 size-4")),
							Span(Class("sr-only"), g.Text("Ask AI")),
						),

						A(
							Class("flex items-center gap-x-1.5 py-1.5 px-2 text-sm text-gray-800 rounded-lg hover:bg-gray-200 focus:outline-hidden focus:bg-gray-200 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800 dark:text-neutral-200"),
							Href(data.BasePath+"/ai"),
							lucide.Brain(Class("shrink-0 size-4")),
							Span(Class("sr-only"), g.Text("Docs")),
						),

						A(
							Class("flex items-center gap-x-1.5 py-1.5 px-2 text-sm text-gray-800 rounded-lg hover:bg-gray-200 focus:outline-hidden focus:bg-gray-200 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800 dark:text-neutral-200"),
							Href(data.BasePath+"/api"),
							lucide.Code(Class("shrink-0 size-4")),
							Span(Class("sr-only"), g.Text("API")),
						),
					),

					Li(
						Class("inline-flex items-center gap-1.5 relative text-gray-500 pe-3 last:pe-0 last:after:hidden after:absolute after:top-1/2 after:end-0 after:inline-block after:w-px after:h-3.5 after:bg-gray-300 after:rounded-full after:-translate-y-1/2 after:rotate-12 dark:text-neutral-200 dark:after:bg-neutral-700"),

						// Theme Toggle
						cmsThemeToggle(),

						// User Profile
						cmsUserDropdown(data),
					),
				),
			),
		),
	)
}

// cmsAppDropdown renders the app/project dropdown in header
func cmsAppDropdown(data PageData) g.Node {
	if data.CurrentApp == nil {
		return g.Text("")
	}

	return Div(
		Class("inline-flex justify-center w-full"),
		Div(
			Class("relative inline-flex"),
			Button(
				ID("cms-app-dropdown"),
				Type("button"),
				Class("py-1 px-2 min-h-8 flex items-center gap-x-1 font-medium text-sm text-gray-800 rounded-lg hover:bg-gray-200 focus:outline-none focus:bg-gray-200 dark:text-neutral-200 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800"),
				g.Attr("@click", "appDropdownOpen = !appDropdownOpen"),
				g.Attr("aria-haspopup", "menu"),
				g.Attr(":aria-expanded", "appDropdownOpen"),

				// App icon
				Div(
					Class("shrink-0 size-6 flex items-center justify-center rounded-full bg-violet-600 text-white text-xs font-semibold me-1"),
					g.Text(getFirstChar(data.CurrentApp.Name, "A")),
				),
				g.Text(data.CurrentApp.Name),
				lucide.ChevronsUpDown(Class("shrink-0 size-3.5 ms-1")),
			),

			// Dropdown
			Div(
				g.Attr("x-show", "appDropdownOpen"),
				g.Attr("x-cloak", ""),
				g.Attr("@click.outside", "appDropdownOpen = false"),
				g.Attr("x-transition:enter", "transition ease-out duration-100"),
				g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
				g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
				g.Attr("x-transition:leave", "transition ease-in duration-75"),
				g.Attr("x-transition:leave-start", "opacity-100 scale-100"),
				g.Attr("x-transition:leave-end", "opacity-0 scale-95"),
				Class("absolute top-full mt-2 start-0 w-64 z-20 bg-white border border-gray-200 rounded-xl shadow-xl dark:bg-neutral-900 dark:border-neutral-700"),
				g.Attr("role", "menu"),

				Div(
					Class("p-1"),
					Span(Class("block pt-2 pb-2 ps-2.5 text-xs text-gray-500 dark:text-neutral-500"),
						g.Textf("Apps (%d)", len(data.UserApps)),
					),
					Div(
						Class("flex flex-col gap-y-1 max-h-60 overflow-y-auto"),
						g.Group(func() []g.Node {
							var items []g.Node
							for _, appItem := range data.UserApps {
								isActive := data.CurrentApp != nil && appItem.ID == data.CurrentApp.ID
								items = append(items, cmsAppDropdownItem(appItem, data.BasePath, isActive))
							}
							return items
						}()),
					),
				),
			),
		),
	)
}

// cmsAppDropdownItem renders a single app item in the dropdown
func cmsAppDropdownItem(appEntity *app.App, basePath string, isActive bool) g.Node {
	href := basePath + "/app/" + appEntity.ID.String() + "/"

	return A(
		Href(href),
		Class("py-2 px-2.5 group flex justify-start items-center gap-x-3 rounded-lg text-[13px] text-gray-800 hover:bg-gray-100 focus:outline-none focus:bg-gray-100 dark:text-neutral-200 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800"),
		g.If(isActive,
			lucide.Check(Class("shrink-0 size-4 text-violet-600 dark:text-violet-400")),
		),
		g.If(!isActive,
			Span(Class("shrink-0 size-4")),
		),
		Span(
			Class("grow"),
			Span(Class("block text-sm font-medium text-gray-800 dark:text-neutral-200"), g.Text(appEntity.Name)),
		),
		Div(
			Class("shrink-0 size-5 flex items-center justify-center rounded-full bg-violet-100 dark:bg-violet-900/30 text-violet-600 dark:text-violet-400 text-xs font-semibold"),
			g.Text(getFirstChar(appEntity.Name, "A")),
		),
	)
}

// cmsEnvDropdown renders the environment dropdown in header
func cmsEnvDropdown(data PageData) g.Node {
	if data.CurrentEnvironment == nil {
		return g.Text("")
	}

	return Div(
		Class("inline-flex justify-center w-full"),
		Div(
			Class("relative inline-flex"),
			Button(
				ID("cms-env-dropdown"),
				Type("button"),
				Class("py-1 px-2 min-h-8 flex items-center gap-x-1 font-medium text-sm text-gray-800 rounded-lg hover:bg-gray-200 focus:outline-none focus:bg-gray-200 dark:text-neutral-200 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800"),
				g.Attr("@click", "envDropdownOpen = !envDropdownOpen"),
				g.Attr("aria-haspopup", "menu"),
				g.Attr(":aria-expanded", "envDropdownOpen"),

				g.Text(data.CurrentEnvironment.Name),
				environmentBadge(data.CurrentEnvironment.Type),
				lucide.ChevronsUpDown(Class("shrink-0 size-3.5 ms-1")),
			),

			// Dropdown
			Div(
				g.Attr("x-show", "envDropdownOpen"),
				g.Attr("x-cloak", ""),
				g.Attr("@click.outside", "envDropdownOpen = false"),
				g.Attr("x-transition:enter", "transition ease-out duration-100"),
				g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
				g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
				g.Attr("x-transition:leave", "transition ease-in duration-75"),
				g.Attr("x-transition:leave-start", "opacity-100 scale-100"),
				g.Attr("x-transition:leave-end", "opacity-0 scale-95"),
				Class("absolute top-full mt-2 start-0 w-56 z-20 bg-white border border-gray-200 rounded-xl shadow-xl dark:bg-neutral-900 dark:border-neutral-700"),
				g.Attr("role", "menu"),

				Div(
					Class("p-1"),
					Span(Class("block pt-2 pb-2 ps-2.5 text-xs text-gray-500 dark:text-neutral-500"),
						g.Textf("Environments (%d)", len(data.UserEnvironments)),
					),
					Div(
						Class("flex flex-col gap-y-1 max-h-48 overflow-y-auto"),
						g.Group(func() []g.Node {
							var items []g.Node
							for _, env := range data.UserEnvironments {
								isActive := data.CurrentEnvironment != nil && env.ID == data.CurrentEnvironment.ID
								items = append(items, cmsEnvDropdownItem(env, data.BasePath, data.CurrentApp.ID.String(), isActive))
							}
							return items
						}()),
					),
				),
			),
		),
	)
}

// cmsEnvDropdownItem renders a single environment item in the dropdown
func cmsEnvDropdownItem(env *environment.Environment, basePath, appIDStr string, isActive bool) g.Node {
	formID := "cms-env-switch-" + env.ID.String()

	return Div(
		Form(
			ID(formID),
			Method("POST"),
			Action(basePath+"/app/"+appIDStr+"/environment/switch"),
			Class("hidden"),
			Input(Type("hidden"), Name("env_id"), Value(env.ID.String())),
		),
		Button(
			Type("button"),
			g.Attr("onclick", "document.getElementById('"+formID+"').submit()"),
			Class("w-full py-2 px-2.5 group flex justify-start items-center gap-x-3 rounded-lg text-[13px] text-gray-800 hover:bg-gray-100 focus:outline-none focus:bg-gray-100 dark:text-neutral-200 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800"),
			g.If(isActive,
				lucide.Check(Class("shrink-0 size-4 text-violet-600 dark:text-violet-400")),
			),
			g.If(!isActive,
				Span(Class("shrink-0 size-4")),
			),
			Span(
				Class("grow text-start"),
				Span(Class("block text-sm font-medium text-gray-800 dark:text-neutral-200"), g.Text(env.Name)),
			),
			environmentBadge(env.Type),
		),
	)
}

// cmsThemeToggle renders the theme toggle button
func cmsThemeToggle() g.Node {
	return Div(
		Class("flex items-center gap-x-0.5"),
		Button(
			Type("button"),
			Class("shrink-0 flex justify-center items-center gap-x-1 text-xs text-gray-500 hover:text-gray-800 focus:outline-none focus:text-gray-800 dark:text-neutral-400 dark:hover:text-neutral-200 dark:focus:text-neutral-200"),
			g.Attr("@click", "toggleTheme()"),
			g.Attr("x-show", "isDark"),
			g.Attr("x-cloak", ""),
			g.Attr("data-hs-theme-click-value", "light"),
			lucide.Sun(Class("shrink-0 size-4")),
			Span(Class("hidden sm:inline"), g.Text("Light")),
		),
		Button(
			Type("button"),
			Class("shrink-0 flex justify-center items-center gap-x-1 text-xs text-gray-500 hover:text-gray-800 focus:outline-none focus:text-gray-800 dark:text-neutral-400 dark:hover:text-neutral-200 dark:focus:text-neutral-200"),
			g.Attr("@click", "toggleTheme()"),
			g.Attr("x-show", "!isDark"),
			g.Attr("data-hs-theme-click-value", "dark"),
			lucide.Moon(Class("shrink-0 size-4")),
			Span(Class("hidden sm:inline"), g.Text("Dark")),
		),

		// Div(
		// 	Class("mb-2 flex items-center gap-x-0.5"),
		// 	Button(
		// 		Type("button"),
		// 		Class("hs-dark-mode hs-dark-mode-active:hidden flex shrink-0 justify-center items-center gap-x-1 text-xs text-gray-500 hover:text-gray-800 focus:outline-hidden focus:text-gray-800 dark:text-neutral-400 dark:hover:text-neutral-200 dark:focus:text-neutral-200"),
		// 		g.Attr("data-hs-theme-click-value", "dark"),
		// 		lucide.Moon(Class("shrink-0 size-3.5")),
		// 		g.Text("Switch to Dark"),
		// 	),
		// 	Button(
		// 		Type("button"),
		// 		Class("hs-dark-mode hs-dark-mode-active:flex hidden shrink-0 justify-center items-center gap-x-1 text-xs text-gray-500 hover:text-gray-800 focus:outline-hidden focus:text-gray-800 dark:text-neutral-400 dark:hover:text-neutral-200 dark:focus:text-neutral-200"),
		// 		g.Attr("data-hs-theme-click-value", "light"),
		// 		lucide.Sun(Class("shrink-0 size-3.5")),
		// 		g.Text("Switch to Light"),
		// 	),
		// ),
	)
}

// cmsUserDropdown renders the user profile dropdown
func cmsUserDropdown(data PageData) g.Node {
	userName := "User"
	userEmail := ""
	userID := ""
	if data.User != nil {
		if data.User.Name != "" {
			userName = data.User.Name
		}
		userEmail = data.User.Email
		userID = data.User.ID.String()
	}

	initial := getInitials(userName)

	return Div(
		Class("h-8"),
		Div(
			Class("relative inline-flex text-start"),
			Button(
				ID("cms-user-dropdown"),
				Type("button"),
				Class("p-0.5 inline-flex shrink-0 items-center gap-x-3 text-start rounded-full hover:bg-gray-200 focus:outline-none focus:bg-gray-200 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800"),
				g.Attr("@click", "userDropdownOpen = !userDropdownOpen"),
				g.Attr("aria-haspopup", "menu"),
				g.Attr(":aria-expanded", "userDropdownOpen"),

				Div(
					Class("shrink-0 size-7 flex items-center justify-center rounded-full bg-violet-600 text-white text-xs font-semibold"),
					g.Text(initial),
				),
			),

			// Dropdown
			Div(
				g.Attr("x-show", "userDropdownOpen"),
				g.Attr("x-cloak", ""),
				g.Attr("@click.outside", "userDropdownOpen = false"),
				g.Attr("x-transition:enter", "transition ease-out duration-100"),
				g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
				g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
				g.Attr("x-transition:leave", "transition ease-in duration-75"),
				g.Attr("x-transition:leave-start", "opacity-100 scale-100"),
				g.Attr("x-transition:leave-end", "opacity-0 scale-95"),
				Class("absolute top-full mt-2 end-0 w-60 z-20 bg-white border border-gray-200 rounded-xl shadow-xl dark:bg-neutral-900 dark:border-neutral-700"),
				g.Attr("role", "menu"),

				// User info
				Div(
					Class("py-2 px-3.5"),
					Span(Class("font-medium text-gray-800 dark:text-neutral-300"), g.Text(userName)),
					P(Class("text-sm text-gray-500 dark:text-neutral-500"), g.Text(userEmail)),
				),

				// Menu items
				Div(
					Class("p-1 border-t border-gray-200 dark:border-neutral-800"),
					A(
						Href(data.BasePath+"/users/"+userID),
						Class("flex items-center gap-x-3 py-2 px-3 rounded-lg text-sm text-gray-600 hover:bg-gray-100 disabled:opacity-50 disabled:pointer-events-none focus:outline-none focus:bg-gray-100 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800"),
						lucide.User(Class("shrink-0 size-4")),
						g.Text("Profile"),
					),
					A(
						Href(getSettingsURL(data)),
						Class("flex items-center gap-x-3 py-2 px-3 rounded-lg text-sm text-gray-600 hover:bg-gray-100 disabled:opacity-50 disabled:pointer-events-none focus:outline-none focus:bg-gray-100 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800"),
						lucide.Settings(Class("shrink-0 size-4")),
						g.Text("Settings"),
					),
					A(
						Href(data.BasePath+"/logout"),
						Class("flex items-center gap-x-3 py-2 px-3 rounded-lg text-sm text-gray-600 hover:bg-gray-100 disabled:opacity-50 disabled:pointer-events-none focus:outline-none focus:bg-gray-100 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:focus:bg-neutral-800"),
						lucide.LogOut(Class("shrink-0 size-4")),
						g.Text("Log out"),
					),
				),
			),
		),
	)
}

// cmsSidebar renders the sidebar navigation (Preline CMS style)
func cmsSidebar(data PageData) g.Node {
	return Div(
		ID("cms-sidebar"),
		Class(`hs-overlay [--body-scroll:true] lg:[--overlay-backdrop:false] [--is-layout-affect:true] [--opened:lg] [--auto-close:lg]
    hs-overlay-open:translate-x-0 lg:hs-overlay-layout-open:translate-x-0
    -translate-x-full transition-all duration-300 transform
    w-60
    hidden
    fixed inset-y-0 z-60 start-0
    bg-gray-100
    lg:block lg:-translate-x-full lg:end-auto lg:bottom-0
    dark:bg-neutral-900`),
		// g.Attr(":class", "{ 'translate-x-0': sidebarOpen, '-translate-x-full': !sidebarOpen }"),
		g.Attr("x-show", "true"),
		g.Attr("role", "dialog"),
		g.Attr("tabindex", "-1"),
		g.Attr("aria-label", "Sidebar"),

		Div(
			Class("lg:pt-13 relative flex flex-col h-full max-h-full"),

			// Sidebar Body
			Nav(
				Class("p-3 size-full flex flex-col overflow-y-auto [&::-webkit-scrollbar]:w-2 [&::-webkit-scrollbar-thumb]:rounded-full [&::-webkit-scrollbar-track]:bg-gray-200 [&::-webkit-scrollbar-thumb]:bg-gray-300 dark:[&::-webkit-scrollbar-track]:bg-neutral-700 dark:[&::-webkit-scrollbar-thumb]:bg-neutral-500"),
				// g.Attr("style", "scrollbar-width: thin;"),

				// Mobile close button
				Div(
					Class("lg:hidden mb-2 flex items-center justify-between"),
					Button(
						Type("button"),
						Class("p-1.5 size-7.5 inline-flex items-center gap-x-1 text-xs rounded-md text-gray-500 disabled:opacity-50 disabled:pointer-events-none focus:outline-hidden dark:text-neutral-500"),
						g.Attr("@click", "sidebarOpen = false"),
						g.Attr("aria-haspopup", "dialog"),
						g.Attr("aria-expanded", "false"),
						g.Attr("aria-controls", "hs-pro-sidebar"),
						g.Attr("data-hs-overlay", "#hs-pro-sidebar"),
						lucide.X(Class("shrink-0 size-4")),
						Span(Class("sr-only"), g.Text("Close sidebar")),
					),
				),

				// Navigation Sections
				cmsSidebarSection("Home", []cmsSidebarLink{
					{Label: "Dashboard", URL: getDashboardURL(data), Active: data.ActivePage == "dashboard", Icon: lucide.LayoutDashboard(Class("shrink-0 size-4"))},
				}),

				cmsSidebarPagesSection(data),

				// Extension navigation section
				g.If(len(data.ExtensionNavData) > 0,
					cmsSidebarSectionFromExtensions("Extensions", data.ExtensionNavData),
				),

				cmsSidebarSection("Configuration", []cmsSidebarLink{
					{Label: "Plugins", URL: getPluginsURL(data), Active: data.ActivePage == "plugins", Icon: lucide.Puzzle(Class("shrink-0 size-4"))},
					{Label: "Settings", URL: getSettingsURL(data), Active: data.ActivePage == "settings", Icon: lucide.Settings(Class("shrink-0 size-4"))},
				}),
			),

			// Sidebar Footer
			Footer(
				Class("mt-auto p-3 flex flex-col"),
				Ul(
					Class("flex flex-col gap-y-1"),
					Li(
						A(
							Href("#"),
							Class("w-full flex items-center gap-x-2 py-2 px-2.5 text-sm text-gray-500 rounded-lg hover:bg-gray-200 hover:text-gray-800 focus:outline-none focus:bg-gray-200 focus:text-gray-800 dark:hover:bg-neutral-800 dark:hover:text-neutral-200 dark:focus:bg-neutral-800 dark:text-neutral-500"),
							lucide.Flame(Class("shrink-0 size-4")),
							g.Text("What's new?"),
						),
					),
					Li(
						A(
							Href("#"),
							Class("w-full flex items-center gap-x-2 py-2 px-2.5 text-sm text-gray-500 rounded-lg hover:bg-gray-200 hover:text-gray-800 focus:outline-none focus:bg-gray-200 focus:text-gray-800 dark:hover:bg-neutral-800 dark:hover:text-neutral-200 dark:focus:bg-neutral-800 dark:text-neutral-500"),
							lucide.MessageCircle(Class("shrink-0 size-4")),
							g.Text("Help & support"),
						),
					),
				),
			),
		),
	)
}

// cmsSidebarLink represents a sidebar navigation link
type cmsSidebarLink struct {
	Label  string
	URL    string
	Active bool
	Icon   g.Node
}

// cmsSidebarPagesSection renders the Pages section with conditional sessions link
func cmsSidebarPagesSection(data PageData) g.Node {
	links := []cmsSidebarLink{
		{Label: "Users", URL: getUsersURL(data), Active: data.ActivePage == "users", Icon: lucide.Users(Class("shrink-0 size-4"))},
	}

	// Only show sessions if multisession plugin is not enabled
	if !data.EnabledPlugins["multisession"] {
		links = append(links, cmsSidebarLink{Label: "Sessions", URL: getSessionsURL(data), Active: data.ActivePage == "sessions", Icon: lucide.Key(Class("shrink-0 size-4"))})
	}

	links = append(links, cmsSidebarLink{Label: "Environments", URL: getEnvironmentsURL(data), Active: data.ActivePage == "environments", Icon: lucide.Layers(Class("shrink-0 size-4"))})

	return cmsSidebarSection("Pages", links)
}

// cmsSidebarSection renders a sidebar section with title and links
func cmsSidebarSection(title string, links []cmsSidebarLink) g.Node {
	linkNodes := make([]g.Node, 0)
	for _, link := range links {
		if link.URL == "" {
			continue
		}
		linkNodes = append(linkNodes, cmsSidebarItem(link.Label, link.URL, link.Active, link.Icon))
	}

	return Div(
		Class("pt-3 mt-3 flex flex-col border-t border-gray-200 first:border-t-0 first:pt-0 first:mt-0 dark:border-neutral-700"),
		Span(
			Class("block ps-2.5 mb-2 font-medium text-xs uppercase text-gray-800 dark:text-neutral-500"),
			g.Text(title),
		),
		Ul(
			Class("flex flex-col gap-y-1"),
			g.Group(linkNodes),
		),
	)
}

// cmsSidebarSectionFromExtensions renders a sidebar section from extension nav data
func cmsSidebarSectionFromExtensions(title string, items []ExtensionNavItemData) g.Node {
	linkNodes := make([]g.Node, 0)
	for _, item := range items {
		linkNodes = append(linkNodes, cmsSidebarItem(item.Label, item.URL, item.IsActive, item.Icon))
	}

	return Div(
		Class("pt-3 mt-3 flex flex-col border-t border-gray-200 first:border-t-0 first:pt-0 first:mt-0 dark:border-neutral-700"),
		Span(
			Class("block ps-2.5 mb-2 font-medium text-xs uppercase text-gray-800 dark:text-neutral-500"),
			g.Text(title),
		),
		Ul(
			Class("flex flex-col gap-y-1"),
			g.Group(linkNodes),
		),
	)
}

// cmsSidebarItem renders a single sidebar navigation item
func cmsSidebarItem(label, href string, active bool, icon g.Node) g.Node {
	activeClass := "text-gray-500 hover:bg-gray-200 hover:text-gray-800 focus:outline-none focus:bg-gray-200 focus:text-gray-800 dark:hover:bg-neutral-800 dark:hover:text-neutral-200 dark:focus:bg-neutral-800 dark:text-neutral-500"
	if active {
		activeClass = "bg-gray-200 text-gray-800 dark:bg-neutral-800 dark:text-neutral-200"
	}

	return Li(
		A(
			Href(href),
			Class("w-full flex items-center gap-x-2 py-2 px-2.5 text-sm rounded-lg "+activeClass),
			g.If(icon != nil, icon),
			g.Text(label),
		),
	)
}

// cmsContentHeader renders the content header with title
func cmsContentHeader(data PageData) g.Node {
	return Div(
		Class("py-3 px-4 flex flex-wrap justify-between items-center gap-2 bg-white border-b border-gray-200 dark:bg-neutral-800 dark:border-neutral-700"),
		Div(
			H1(
				Class("font-medium text-lg text-gray-800 dark:text-neutral-200"),
				g.Text(data.Title),
			),
		),
	)
}

// Helper functions for URL generation (sidebar-specific)
func getDashboardURL(data PageData) string {
	if data.CurrentApp != nil {
		return data.BasePath + "/app/" + data.CurrentApp.ID.String() + "/"
	}
	return data.BasePath + "/"
}

func getUsersURL(data PageData) string {
	if data.CurrentApp != nil {
		return data.BasePath + "/app/" + data.CurrentApp.ID.String() + "/users"
	}
	return data.BasePath + "/"
}

func getSessionsURL(data PageData) string {
	if data.CurrentApp != nil {
		return data.BasePath + "/app/" + data.CurrentApp.ID.String() + "/sessions"
	}
	return data.BasePath + "/"
}

func getEnvironmentsURL(data PageData) string {
	if data.CurrentApp != nil {
		return data.BasePath + "/app/" + data.CurrentApp.ID.String() + "/environments"
	}
	return data.BasePath + "/"
}

func getPluginsURL(data PageData) string {
	if data.CurrentApp != nil {
		return data.BasePath + "/app/" + data.CurrentApp.ID.String() + "/plugins"
	}
	return data.BasePath + "/"
}

func getSettingsURL(data PageData) string {
	if data.CurrentApp != nil {
		return data.BasePath + "/app/" + data.CurrentApp.ID.String() + "/settings"
	}
	return data.BasePath + "/"
}

func getInitials(name string) string {
	if name == "" {
		return "U"
	}
	names := strings.Split(name, " ")
	initial := string(names[0][0])
	if len(names) > 1 {
		initial += string(names[1][0])
	}
	return strings.ToUpper(initial)
}

// getFirstChar safely returns the first character of a string or a fallback
func getFirstChar(s string, fallback string) string {
	if len(s) == 0 {
		return fallback
	}
	return strings.ToUpper(string(s[0]))
}
