package consent

import (
	"fmt"

	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/icons"
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
	Nonce               string
	Branding            BrandingConfig
}

// ScopeInfo represents a scope with its description
type ScopeInfo struct {
	Scope       string
	Description string
}

// OAuthConsentPage renders the OAuth consent page using ForgeUI
func OAuthConsentPage(data ConsentPageData) g.Node {
	// Generate custom CSS for brand colors
	customCSS := g.Raw(fmt.Sprintf(`
		:root {
			--brand-primary: %s;
			--brand-bg: %s;
			--brand-card: %s;
			--brand-text: %s;
		}
		body {
			background-color: var(--brand-bg);
		}
		.btn-brand {
			background-color: var(--brand-primary);
		}
		.btn-brand:hover {
			filter: brightness(0.9);
		}
		.icon-brand {
			color: var(--brand-primary);
		}
		[x-cloak] { 
			display: none !important; 
		}
	`, data.Branding.PrimaryColor, data.Branding.BackgroundColor, data.Branding.CardBackground, data.Branding.TextColor))

	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			g.Attr("x-data", "themeData()"),
			g.Attr("x-init", "initTheme()"),
			g.Attr(":class", "{ 'dark': isDark }"),

			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text("Authorization Required - "+data.Branding.AppName)),

				// Tailwind CSS CDN
				Script(Src("https://cdn.tailwindcss.com?plugins=forms,typography")),
				Script(g.Raw(`tailwind.config = { darkMode: 'class' }`)),

				// Alpine.js
				Script(Defer(), Src("https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js")),

				// Custom brand styling
				StyleEl(customCSS),

				// Theme script
				themeScript(),
			),

			Body(
				Class("min-h-screen flex items-center justify-center p-4"),

				Div(Class("max-w-lg w-full"),
					card.Card(
						card.Header(
							// Logo
							g.If(data.LogoURI != "",
								Div(Class("flex justify-center mb-6"),
									Img(Src(data.LogoURI), Alt(data.ClientName),
										Class("w-24 h-24 rounded-2xl object-cover shadow-md")),
								),
							),
							g.If(data.LogoURI == "",
								Div(Class("flex justify-center mb-6"),
									Div(Class("w-20 h-20 rounded-2xl flex items-center justify-center"),
										StyleEl(g.Raw(fmt.Sprintf("background-color: %s; opacity: 0.1;", data.Branding.PrimaryColor))),
										Div(Class("absolute"),
											icons.Lock(icons.WithSize(40), icons.WithClass("icon-brand")),
										),
									),
								),
							),
							card.Title("Authorize "+data.Branding.AppName),
							card.Description(data.ClientName+" wants to access your account"),
						),

						card.Content(
							Div(Class("space-y-6"),
								// Permissions section
								Div(Class("space-y-3"),
									H3(Class("text-sm font-semibold text-gray-700 dark:text-gray-200"),
										g.Text(data.ClientName+" will be able to:")),
									Div(Class("space-y-2"),
										g.Group(scopeItems(data.Scopes)),
									),
								),

								// Consent form
								FormEl(
									Method("POST"),
									Action("/oauth2/consent"),
									Class("space-y-6"),

									// Hidden fields
									Input(Type("hidden"), Name("client_id"), Value(data.ClientID)),
									Input(Type("hidden"), Name("redirect_uri"), Value(data.RedirectURI)),
									Input(Type("hidden"), Name("response_type"), Value(data.ResponseType)),
									Input(Type("hidden"), Name("scope"), Value(scopesToString(data.Scopes))),
									Input(Type("hidden"), Name("state"), Value(data.State)),
									Input(Type("hidden"), Name("code_challenge"), Value(data.CodeChallenge)),
									Input(Type("hidden"), Name("code_challenge_method"), Value(data.CodeChallengeMethod)),
									Input(Type("hidden"), Name("nonce"), Value(data.Nonce)),

									// Action buttons
									Div(Class("flex gap-3 pt-2"),
										Button(
											Type("submit"),
											Name("action"),
											Value("deny"),
											Class("flex-1 px-6 py-4 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200 rounded-xl font-semibold hover:bg-gray-200 dark:hover:bg-gray-600 transition-all"),
											g.Text("Deny"),
										),
										Button(
											Type("submit"),
											Name("action"),
											Value("allow"),
											Class("btn-brand flex-1 px-6 py-4 text-white rounded-xl font-semibold transition-all shadow-lg transform active:scale-95"),
											g.Text("Allow Access"),
										),
									),
								),

								// Security notice
								Div(Class("flex items-start gap-3 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-xl"),
									icons.Shield(icons.WithSize(20), icons.WithClass("text-blue-600 dark:text-blue-400 mt-0.5")),
									Div(Class("text-sm"),
										P(Class("font-semibold text-blue-900 dark:text-blue-200"), g.Text("Your data is secure")),
										P(Class("text-blue-700 dark:text-blue-300 mt-1"),
											g.Text("You can revoke access at any time from your account settings.")),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}

func scopeItems(scopes []ScopeInfo) []g.Node {
	items := make([]g.Node, len(scopes))
	for i, scope := range scopes {
		items[i] = Div(
			Class("flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"),
			icons.Check(icons.WithSize(20), icons.WithClass("text-green-600 dark:text-green-400")),
			Span(Class("text-sm text-gray-700 dark:text-gray-200"), g.Text(scope.Description)),
		)
	}
	return items
}

func scopesToString(scopes []ScopeInfo) string {
	var result string
	for i, s := range scopes {
		if i > 0 {
			result += " "
		}
		result += s.Scope
	}
	return result
}

func themeScript() g.Node {
	return Script(g.Raw(`
		function themeData() {
			return {
				isDark: false,
				initTheme() {
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
