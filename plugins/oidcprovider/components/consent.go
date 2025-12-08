package components

import (
	"strings"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ConsentPageData contains all data needed for the consent page
type ConsentPageData struct {
	ClientName          string
	ClientID            string
	LogoURI             string
	Scopes              []ScopeInfo
	RedirectURI         string
	ResponseType        string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

// ScopeInfo represents a scope with its description
type ScopeInfo struct {
	Scope       string
	Description string
}

// ConsentPage renders the OAuth consent page using gomponents
func ConsentPage(data ConsentPageData) g.Node {
	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			g.Attr("x-data", "themeData()"),
			g.Attr("x-init", "initTheme()"),
			g.Attr(":class", "{ 'dark': isDark }"),

			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text("Authorization Required - AuthSome")),

				// Tailwind CSS CDN
				Script(Src("https://cdn.tailwindcss.com?plugins=forms,typography")),
				Script(g.Raw(`tailwind.config = { darkMode: 'class' }`)),

				// Alpine.js
				Script(Defer(), Src("https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js")),

				// Alpine x-cloak
				StyleEl(g.Raw(`[x-cloak] { display: none !important; }`)),

				// Custom inline styles for gradient and theme toggle
				customStyles(),

				// Theme toggle script (from dashboard)
				themeScript(),
			),

			Body(
				Class("min-h-screen flex items-center justify-center p-4 transition-colors duration-200"),
				g.Attr("style", "background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);"),

				consentContainer(data),
			),
		),
	)
}

func consentContainer(data ConsentPageData) g.Node {
	return Div(
		Class("max-w-md w-full bg-white dark:bg-gray-800 rounded-2xl shadow-2xl p-8 space-y-6"),

		// Header with logo
		consentHeader(data),

		// Permissions section
		permissionsSection(data.Scopes),

		// Consent form
		consentForm(data),

		// Security notice
		securityNotice(),
	)
}

func consentHeader(data ConsentPageData) g.Node {
	return Div(
		Class("text-center space-y-4"),

		// Logo or icon
		Div(
			Class("flex justify-center"),
			g.If(data.LogoURI != "",
				Img(
					Src(data.LogoURI),
					Alt(data.ClientName),
					Class("w-16 h-16 rounded-xl object-cover"),
				),
			),
			g.If(data.LogoURI == "",
				Div(
					Class("w-16 h-16 bg-purple-100 dark:bg-purple-900 rounded-xl flex items-center justify-center"),
					lucide.Lock(Class("w-8 h-8 text-purple-600 dark:text-purple-400")),
				),
			),
		),

		H1(
			Class("text-2xl font-bold text-gray-900 dark:text-white"),
			g.Text("Authorization Required"),
		),

		P(
			Class("text-gray-600 dark:text-gray-300"),
			g.Text(data.ClientName+" wants to access your account"),
		),
	)
}

func permissionsSection(scopes []ScopeInfo) g.Node {
	return Div(
		Class("space-y-3"),

		H3(
			Class("text-sm font-semibold text-gray-700 dark:text-gray-200 uppercase tracking-wider"),
			g.Text("This application will be able to:"),
		),

		Div(
			Class("space-y-2"),
			g.Group(scopeItems(scopes)),
		),
	)
}

func scopeItems(scopes []ScopeInfo) []g.Node {
	items := make([]g.Node, len(scopes))
	for i, scope := range scopes {
		items[i] = Div(
			Class("flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg transition-colors"),

			Div(
				Class("flex-shrink-0"),
				lucide.Check(Class("w-5 h-5 text-green-600 dark:text-green-400")),
			),

			Span(
				Class("text-sm text-gray-700 dark:text-gray-200"),
				g.Text(scope.Description),
			),
		)
	}
	return items
}

func consentForm(data ConsentPageData) g.Node {
	return FormEl(
		Method("POST"),
		Action("/oauth2/consent"),
		Class("space-y-4"),

		// Hidden fields
		Input(Type("hidden"), Name("client_id"), Value(data.ClientID)),
		Input(Type("hidden"), Name("redirect_uri"), Value(data.RedirectURI)),
		Input(Type("hidden"), Name("response_type"), Value(data.ResponseType)),
		Input(Type("hidden"), Name("scope"), Value(scopesToString(data.Scopes))),
		Input(Type("hidden"), Name("state"), Value(data.State)),
		Input(Type("hidden"), Name("code_challenge"), Value(data.CodeChallenge)),
		Input(Type("hidden"), Name("code_challenge_method"), Value(data.CodeChallengeMethod)),

		// Action buttons
		Div(
			Class("flex gap-3"),

			Button(
				Type("submit"),
				Name("action"),
				Value("deny"),
				Class("flex-1 px-4 py-3 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-200 rounded-lg font-medium hover:bg-gray-300 dark:hover:bg-gray-600 transition-colors"),
				g.Text("Deny"),
			),

			Button(
				Type("submit"),
				Name("action"),
				Value("allow"),
				Class("flex-1 px-4 py-3 bg-purple-600 text-white rounded-lg font-medium hover:bg-purple-700 transition-colors shadow-lg"),
				g.Text("Allow Access"),
			),
		),
	)
}

func securityNotice() g.Node {
	return Div(
		Class("flex items-start gap-3 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg"),

		Div(
			Class("flex-shrink-0 mt-0.5"),
			lucide.Shield(Class("w-5 h-5 text-blue-600 dark:text-blue-400")),
		),

		Div(
			Class("text-sm space-y-1"),
			P(
				Class("font-medium text-blue-900 dark:text-blue-200"),
				g.Text("Your data is secure"),
			),
			P(
				Class("text-blue-700 dark:text-blue-300"),
				g.Text("You can revoke access at any time from your account settings."),
			),
		),
	)
}

func customStyles() g.Node {
	return StyleEl(g.Raw(`
        .gradient-bg {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
    `))
}

func themeScript() g.Node {
	return Script(g.Raw(`
        function themeData() {
            return {
                isDark: false,
                initTheme() {
                    // Check localStorage or system preference
                    const savedTheme = localStorage.getItem('theme');
                    if (savedTheme) {
                        this.isDark = savedTheme === 'dark';
                    } else {
                        this.isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
                    }
                },
                toggleTheme() {
                    this.isDark = !this.isDark;
                    localStorage.setItem('theme', this.isDark ? 'dark' : 'light');
                }
            }
        }
    `))
}

func scopesToString(scopes []ScopeInfo) string {
	scopeStrs := make([]string, len(scopes))
	for i, s := range scopes {
		scopeStrs[i] = s.Scope
	}
	return strings.Join(scopeStrs, " ")
}
