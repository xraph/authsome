package components

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/user"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// PageData represents common data for all pages
type PageData struct {
	Title          string
	User           *user.User
	CSRFToken      string
	ActivePage     string
	BasePath       string
	Data           interface{}
	Error          string
	Success        string
	Year           int
	EnabledPlugins map[string]bool
	IsSaaSMode     bool // Whether multitenancy is enabled
}

// BaseLayout renders the main HTML structure
func BaseLayout(data PageData, content g.Node) g.Node {
	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			Class("h-full"),
			g.Attr("x-data", "themeData()"),
			g.Attr("x-init", "initTheme()"),
			g.Attr(":class", "{ 'dark': isDark }"),

			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text(data.Title+" - AuthSome Dashboard")),

				// Tailwind CSS CDN
				Script(Src("https://cdn.tailwindcss.com?plugins=forms,typography")),

				// Custom CSS
				Link(Rel("stylesheet"), Href(data.BasePath+"/dashboard/static/css/custom.css")),

				// Alpine.js x-cloak style
				StyleEl(g.Raw(`[x-cloak] { display: none !important; }`)),

				// Tailwind Configuration
				tailwindConfig(),

				// Load component functions BEFORE Alpine.js
				Script(Src(data.BasePath+"/dashboard/static/js/pines-components.js")),
				Script(Src(data.BasePath+"/dashboard/static/js/dashboard.js")),

				Link(Href("https://cdn.jsdelivr.net/npm/daisyui@5"), Rel("stylesheet"), Type("text/css")),
				Script(Src("https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4")),

				// Alpine.js - Load LAST
				Script(Defer(), Src("https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js")),
			),

			Body(
				Class("h-full bg-slate-50 dark:bg-gray-950"),
				g.Attr("x-data", "{ userDropdownOpen: false, mobileNavOpen: false }"),

				// Global notification container
				notificationContainer(),

				// Page container
				Div(
					ID("page-container"),
					Class("mx-auto flex min-h-screen w-full min-w-[320px] flex-col"),

					// Page Header
					DashboardHeader(data),

					// Page Content
					Main(
						ID("page-content"),
						Class("flex max-w-full flex-auto flex-col"),

						// Page Heading
						pageHeading(data),

						// Page Section
						Div(
							Class("container mx-auto p-4 lg:p-8 xl:max-w-7xl"),

							// Flash Messages
							flashMessages(data),

							// Content
							content,
						),
					),

					// Page Footer
					DashboardFooter(data),
				),
			),
		),
	)
}

func tailwindConfig() g.Node {
	return Script(g.Raw(`
        tailwind.config = {
            darkMode: 'class',
            theme: {
                extend: {
                    colors: {
                        primary: {
                            DEFAULT: 'rgb(124 58 237)',
                            50: 'rgb(250 245 255)',
                            100: 'rgb(243 232 255)',
                            200: 'rgb(233 213 255)',
                            300: 'rgb(216 180 254)',
                            400: 'rgb(192 132 252)',
                            500: 'rgb(168 85 247)',
                            600: 'rgb(147 51 234)',
                            700: 'rgb(126 34 206)',
                            800: 'rgb(107 33 168)',
                            900: 'rgb(88 28 135)',
                        }
                    }
                }
            }
        }
    `))
}

func notificationContainer() g.Node {
	return Div(
		g.Attr("x-data", "notification()"),
		Class("fixed top-4 right-4 z-50 space-y-2"),
		StyleEl(g.Raw("max-width: 420px;")),

		g.Raw(`
            <template x-for="notif in notifications" :key="notif.id">
                <div x-show="true"
                     x-transition:enter="transition ease-out duration-300"
                     x-transition:enter-start="opacity-0 translate-x-4"
                     x-transition:enter-end="opacity-100 translate-x-0"
                     x-transition:leave="transition ease-in duration-200"
                     x-transition:leave-start="opacity-100 translate-x-0"
                     x-transition:leave-end="opacity-0 translate-x-4"
                     class="flex items-start gap-3 p-4 rounded-xl shadow-lg backdrop-blur-sm"
                     :class="{
                         'bg-green-50/90 dark:bg-green-900/30 border border-green-200 dark:border-green-800': notif.type === 'success',
                         'bg-red-50/90 dark:bg-red-900/30 border border-red-200 dark:border-red-800': notif.type === 'error',
                         'bg-yellow-50/90 dark:bg-yellow-900/30 border border-yellow-200 dark:border-yellow-800': notif.type === 'warning',
                         'bg-blue-50/90 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800': notif.type === 'info'
                     }">
                    <div class="flex-shrink-0">
                        <svg x-show="notif.type === 'success'" class="h-5 w-5 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        <svg x-show="notif.type === 'error'" class="h-5 w-5 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
                        </svg>
                        <svg x-show="notif.type === 'warning'" class="h-5 w-5 text-yellow-600 dark:text-yellow-400" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z" />
                        </svg>
                        <svg x-show="notif.type === 'info'" class="h-5 w-5 text-blue-600 dark:text-blue-400" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z" />
                        </svg>
                    </div>
                    <p class="flex-1 text-sm font-medium"
                       :class="{
                           'text-green-800 dark:text-green-200': notif.type === 'success',
                           'text-red-800 dark:text-red-200': notif.type === 'error',
                           'text-yellow-800 dark:text-yellow-200': notif.type === 'warning',
                           'text-blue-800 dark:text-blue-200': notif.type === 'info'
                       }"
                       x-text="notif.message"></p>
                    <button @click="remove(notif.id)" class="flex-shrink-0 ml-2">
                        <svg class="h-4 w-4"
                             :class="{
                                 'text-green-600 hover:text-green-800 dark:text-green-400 dark:hover:text-green-200': notif.type === 'success',
                                 'text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-200': notif.type === 'error',
                                 'text-yellow-600 hover:text-yellow-800 dark:text-yellow-400 dark:hover:text-yellow-200': notif.type === 'warning',
                                 'text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-200': notif.type === 'info'
                             }"
                             fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
            </template>
        `),
	)
}

func flashMessages(data PageData) g.Node {
	return g.Group([]g.Node{
		g.If(data.Error != "",
			Div(
				Class("mb-6 rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-4"),
				Div(
					Class("flex items-start gap-3"),
					errorIcon(),
					P(Class("text-sm font-medium text-red-800 dark:text-red-200"), g.Text(data.Error)),
				),
			),
		),
		g.If(data.Success != "",
			Div(
				Class("mb-6 rounded-lg border border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-900/20 p-4"),
				Div(
					Class("flex items-start gap-3"),
					successIcon(),
					P(Class("text-sm font-medium text-green-800 dark:text-green-200"), g.Text(data.Success)),
				),
			),
		),
	})
}

func pageHeading(data PageData) g.Node {
	if data.Title != "Users" && data.Title != "Sessions" {
		return Div(
			Class("mx-auto w-full"),
		)
	}

	return Div(
		Class("mx-auto w-full space-y-2"),
		Div(
			Div(
				Class("container mx-auto px-4 py-6 lg:px-8 lg:py-8 xl:max-w-7xl"),
				Div(
					Class("flex flex-col gap-2 text-center sm:flex-row sm:items-center sm:justify-between sm:text-start"),
					Div(
						Class("grow"),
						H1(Class("mb-1 text-xl font-bold text-slate-900 dark:text-white"), g.Text(data.Title)),
						// H2(
						// 	Class("text-sm font-medium text-slate-500 dark:text-gray-400"),
						// 	g.Text("Welcome to your AuthSome dashboard"),
						// ),
					),
				),
			),
		),
		Hr(Class("mt-6 border-slate-200 dark:border-gray-800 lg:mt-8")),
	)
}

func errorIcon() g.Node {
	return lucide.CircleAlert(
		Class("h-5 w-5 text-red-600 dark:text-red-400 mt-0.5"),
	)
}

func successIcon() g.Node {
	return lucide.CircleCheck(
		Class("h-5 w-5 text-green-600 dark:text-green-400 mt-0.5"),
	)
}
