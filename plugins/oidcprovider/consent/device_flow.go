package consent

import (
	"fmt"

	"github.com/xraph/authsome/schema"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/icons"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// BrandingConfig contains branding configuration for consent pages
type BrandingConfig struct {
	PrimaryColor    string // Main brand color (e.g., "#4F46E5")
	BackgroundColor string // Page background color (e.g., "#F9FAFB")
	CardBackground  string // Card background color (e.g., "#FFFFFF")
	TextColor       string // Primary text color (e.g., "#111827")
	AppName         string // Display name (e.g., "Acme Inc")
}

// DefaultBranding returns the default branding configuration
func DefaultBranding() BrandingConfig {
	return BrandingConfig{
		PrimaryColor:    "#4F46E5", // Indigo-600
		BackgroundColor: "#F9FAFB", // Gray-50
		CardBackground:  "#FFFFFF", // White
		TextColor:       "#111827", // Gray-900
		AppName:         "AuthSome",
	}
}

// ExtractBranding extracts branding configuration from OAuth client or app metadata
// Priority: 1. Client metadata, 2. App metadata, 3. Defaults
func ExtractBranding(client *schema.OAuthClient, app *schema.App) BrandingConfig {
	branding := DefaultBranding()

	// Try client metadata first
	if client != nil && client.Metadata != nil {
		if brandingMap, ok := client.Metadata["branding"].(map[string]interface{}); ok {
			if primaryColor, ok := brandingMap["primaryColor"].(string); ok && primaryColor != "" {
				branding.PrimaryColor = primaryColor
			}
			if backgroundColor, ok := brandingMap["backgroundColor"].(string); ok && backgroundColor != "" {
				branding.BackgroundColor = backgroundColor
			}
			if cardBackground, ok := brandingMap["cardBackground"].(string); ok && cardBackground != "" {
				branding.CardBackground = cardBackground
			}
			if textColor, ok := brandingMap["textColor"].(string); ok && textColor != "" {
				branding.TextColor = textColor
			}
			if appName, ok := brandingMap["appName"].(string); ok && appName != "" {
				branding.AppName = appName
			}
		}
		// If client has a name but no custom app name in branding, use client name
		if branding.AppName == "AuthSome" && client.Name != "" {
			branding.AppName = client.Name
		}
	}

	// Fallback to app metadata
	if app != nil && app.Metadata != nil {
		if brandingMap, ok := app.Metadata["branding"].(map[string]interface{}); ok {
			if branding.PrimaryColor == "#4F46E5" {
				if primaryColor, ok := brandingMap["primaryColor"].(string); ok && primaryColor != "" {
					branding.PrimaryColor = primaryColor
				}
			}
			if branding.BackgroundColor == "#F9FAFB" {
				if backgroundColor, ok := brandingMap["backgroundColor"].(string); ok && backgroundColor != "" {
					branding.BackgroundColor = backgroundColor
				}
			}
			if branding.CardBackground == "#FFFFFF" {
				if cardBackground, ok := brandingMap["cardBackground"].(string); ok && cardBackground != "" {
					branding.CardBackground = cardBackground
				}
			}
			if branding.TextColor == "#111827" {
				if textColor, ok := brandingMap["textColor"].(string); ok && textColor != "" {
					branding.TextColor = textColor
				}
			}
			if branding.AppName == "AuthSome" {
				if appName, ok := brandingMap["appName"].(string); ok && appName != "" {
					branding.AppName = appName
				}
			}
		}
		// If app has a name but no custom app name in branding, use app name
		if branding.AppName == "AuthSome" && app.Name != "" {
			branding.AppName = app.Name
		}
	}

	return branding
}

// CodeEntryPageData contains data for the device code entry page
type CodeEntryPageData struct {
	UserCode    string
	ErrorMsg    string
	Branding    BrandingConfig
	BasePath    string // Base path for URLs (e.g., "/oauth2")
	RedirectURL string // Optional redirect URL after authorization
}

// DeviceCodeEntryPage renders the device code entry page using ForgeUI
func DeviceCodeEntryPage(data CodeEntryPageData) g.Node {
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
	`, data.Branding.PrimaryColor, data.Branding.BackgroundColor, data.Branding.CardBackground, data.Branding.TextColor))

	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text("Device Authorization - "+data.Branding.AppName)),

				// Tailwind CSS CDN
				Script(Src("https://cdn.tailwindcss.com?plugins=forms,typography")),
				Script(g.Raw(`tailwind.config = { darkMode: 'class' }`)),

				// Custom brand styling
				StyleEl(customCSS),
			),
			Body(
				Class("min-h-screen flex items-center justify-center p-4"),

				Div(Class("max-w-md w-full"),
					card.Card(
						card.Header(
							Div(Class("flex justify-center mb-6"),
								Div(Class("w-20 h-20 rounded-2xl flex items-center justify-center"),
									StyleEl(g.Raw(fmt.Sprintf("background-color: %s; opacity: 0.1;", data.Branding.PrimaryColor))),
									Div(Class("absolute"),
										icons.Lock(icons.WithSize(40), icons.WithClass("icon-brand")),
									),
								),
							),
							card.Title("Sign in to "+data.Branding.AppName),
							card.Description("Enter the verification code shown on your device"),
						),

						card.Content(
							Div(Class("space-y-6"),
								// Error message
								g.If(data.ErrorMsg != "",
									Div(Class("p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg"),
										Div(Class("flex items-start gap-3"),
											icons.AlertCircle(icons.WithSize(20), icons.WithClass("text-red-600 dark:text-red-400 mt-0.5")),
											Span(Class("text-sm font-medium text-red-600 dark:text-red-400"), g.Text(data.ErrorMsg)),
										),
									),
								),

								// Code entry form
								FormEl(
									Method("POST"),
									Action(data.BasePath+"/device/verify"),
									Class("space-y-6"),

									g.If(data.RedirectURL != "",
										Input(Type("hidden"), Name("redirect"), Value(data.RedirectURL)),
									),

									Div(
										Label(Class("block text-sm font-semibold text-gray-700 dark:text-gray-200 mb-2"),
											g.Text("Verification Code")),
										Input(
											Type("text"),
											Name("user_code"),
											Value(data.UserCode),
											Placeholder("XXXX-XXXX"),
											Class("flex h-14 w-full rounded-lg border-2 border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 px-4 py-3 text-2xl text-center tracking-widest uppercase font-mono focus:outline-none focus:ring-2 focus:ring-offset-2 transition-all"),
											StyleEl(g.Raw(fmt.Sprintf("focus:ring-color: %s; focus:border-color: %s;", data.Branding.PrimaryColor, data.Branding.PrimaryColor))),
											g.Attr("autocomplete", "off"),
											MaxLength("9"),
											Required(),
											AutoFocus(),
										),
										P(Class("text-xs text-gray-500 dark:text-gray-400 mt-2 text-center"),
											g.Text("Code is case-insensitive and may contain hyphens")),
									),

									Button(
										Type("submit"),
										Class("btn-brand w-full px-6 py-4 text-white rounded-lg font-semibold transition-all transform active:scale-95"),
										g.Text("Continue"),
									),
								),

								// Info footer
								Div(Class("pt-4 border-t border-gray-200 dark:border-gray-700"),
									Div(Class("flex items-center justify-center gap-2 text-xs text-gray-500 dark:text-gray-400"),
										icons.Shield(icons.WithSize(14)),
										g.Text("Your connection is secure"),
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

// VerificationPageData contains data for the device verification/consent page
type VerificationPageData struct {
	UserCode          string // Normalized code (stored in DB, used in form field)
	UserCodeFormatted string // Formatted code (displayed to user)
	ClientName        string
	LogoURI           string
	Scopes            []ScopeInfo
	Branding          BrandingConfig
	BasePath          string // Base path for URLs (e.g., "/oauth2")
	RedirectURL       string // Optional redirect URL after authorization
}

// DeviceVerificationPage renders the device verification and consent page using ForgeUI
func DeviceVerificationPage(data VerificationPageData) g.Node {
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
		.code-display {
			background-color: %s;
			opacity: 0.1;
		}
		.code-text {
			color: var(--brand-primary);
		}
	`, data.Branding.PrimaryColor, data.Branding.BackgroundColor, data.Branding.CardBackground, data.Branding.TextColor, data.Branding.PrimaryColor))

	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text("Authorize Device - "+data.Branding.AppName)),

				// Tailwind CSS CDN
				Script(Src("https://cdn.tailwindcss.com?plugins=forms,typography")),
				Script(g.Raw(`tailwind.config = { darkMode: 'class' }`)),

				// Custom brand styling
				StyleEl(customCSS),
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

							// User code display with brand color
							Div(Class("my-8 relative"),
								Div(Class("code-display absolute inset-0 rounded-2xl")),
								Div(Class("relative text-5xl font-mono font-bold tracking-wider text-center p-6 rounded-2xl code-text"),
									g.If(data.UserCodeFormatted != "",
										g.Text(data.UserCodeFormatted),
									),
									g.If(data.UserCodeFormatted == "",
										g.Text(data.UserCode),
									),
								),
							),

							card.Title("Authorize "+data.Branding.AppName),
							card.Description("Confirm this code matches your device"),
						),

						card.Content(
							Div(Class("space-y-6"),
								// Security warning
								Div(Class("p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-xl"),
									Div(Class("flex items-start gap-3"),
										icons.Shield(icons.WithSize(20), icons.WithClass("text-amber-600 dark:text-amber-400 mt-0.5")),
										Div(Class("text-sm"),
											P(Class("font-semibold text-amber-900 dark:text-amber-200"),
												g.Text("Verify this request")),
											P(Class("text-amber-700 dark:text-amber-300 mt-1"),
												g.Text("Only approve if the code above matches exactly what's shown on your device.")),
										),
									),
								),

								// Permissions section
								g.If(len(data.Scopes) > 0,
									Div(Class("space-y-3"),
										H3(Class("text-sm font-semibold text-gray-700 dark:text-gray-200"),
											g.Text(data.ClientName+" will be able to:")),
										Div(Class("space-y-2"),
											g.Group(scopeItems(data.Scopes)),
										),
									),
								),

								// Form
								FormEl(
									Method("POST"),
									Action(data.BasePath+"/device/authorize"),
									Class("space-y-4"),

									Input(Type("hidden"), Name("user_code"), Value(data.UserCode)),
									g.If(data.RedirectURL != "",
										Input(Type("hidden"), Name("redirect"), Value(data.RedirectURL)),
									),

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
											Value("approve"),
											Class("btn-brand flex-1 px-6 py-4 text-white rounded-xl font-semibold transition-all flex items-center justify-center gap-2 shadow-lg transform active:scale-95"),
											icons.Check(icons.WithSize(18)),
											g.Text("Approve"),
										),
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

// DeviceSuccessPage renders the authorization success/denial page using ForgeUI
func DeviceSuccessPage(approved bool, branding BrandingConfig) g.Node {
	var icon g.Node
	var title, message, colorClass, bgClass string

	if approved {
		icon = icons.CheckCircle(icons.WithSize(80), icons.WithClass("mx-auto text-green-600 dark:text-green-400"))
		title = "Authorization Successful"
		message = "You have successfully authorized " + branding.AppName + ". You can now return to your device and complete the setup."
		colorClass = "text-green-600 dark:text-green-400"
		bgClass = "bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800"
	} else {
		icon = icons.XCircle(icons.WithSize(80), icons.WithClass("mx-auto text-red-600 dark:text-red-400"))
		title = "Authorization Denied"
		message = "You have denied the device authorization request for " + branding.AppName + ". The device will not have access to your account."
		colorClass = "text-red-600 dark:text-red-400"
		bgClass = "bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800"
	}

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
		@keyframes fadeIn { 
			from { 
				opacity: 0; 
				transform: scale(0.8); 
			} 
			to { 
				opacity: 1; 
				transform: scale(1); 
			} 
		}
		@keyframes slideUp {
			from {
				opacity: 0;
				transform: translateY(20px);
			}
			to {
				opacity: 1;
				transform: translateY(0);
			}
		}
		.animate-fade-in { 
			animation: fadeIn 0.5s ease-out; 
		}
		.animate-slide-up {
			animation: slideUp 0.6s ease-out 0.2s backwards;
		}
	`, branding.PrimaryColor, branding.BackgroundColor, branding.CardBackground, branding.TextColor))

	return Doctype(
		HTML(
			g.Attr("lang", "en"),
			Head(
				Meta(Charset("UTF-8")),
				Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
				TitleEl(g.Text(title+" - "+branding.AppName)),

				// Tailwind CSS CDN
				Script(Src("https://cdn.tailwindcss.com?plugins=forms,typography")),
				Script(g.Raw(`tailwind.config = { darkMode: 'class' }`)),

				// Custom styling
				StyleEl(customCSS),
			),
			Body(
				Class("min-h-screen flex items-center justify-center p-4"),

				Div(Class("max-w-lg w-full"),
					card.Card(
						card.Content(
							Div(Class("text-center space-y-8 py-12 px-6"),
								// Animated icon
								Div(Class("animate-fade-in mb-6"), icon),

								// Title and message
								Div(Class("animate-slide-up space-y-4"),
									H1(Class("text-3xl font-bold "+colorClass), g.Text(title)),
									P(Class("text-gray-600 dark:text-gray-300 text-lg leading-relaxed max-w-md mx-auto"),
										g.Text(message)),
								),

								// Status indicator
								Div(Class("animate-slide-up p-5 "+bgClass+" border rounded-2xl"),
									Div(Class("flex items-center justify-center gap-3"),
										g.If(approved,
											g.Group([]g.Node{
												icons.Sparkles(icons.WithSize(20), icons.WithClass("text-green-600 dark:text-green-400")),
												Span(Class("text-sm font-semibold text-green-700 dark:text-green-300"),
													g.Text("Device connected successfully")),
											}),
										),
										g.If(!approved,
											g.Group([]g.Node{
												icons.Shield(icons.WithSize(20), icons.WithClass("text-red-600 dark:text-red-400")),
												Span(Class("text-sm font-semibold text-red-700 dark:text-red-300"),
													g.Text("Access was not granted")),
											}),
										),
									),
								),

								// Close window hint
								Div(Class("animate-slide-up pt-6 border-t border-gray-200 dark:border-gray-700"),
									Div(Class("flex items-center justify-center gap-2 text-sm text-gray-500 dark:text-gray-400"),
										icons.Info(icons.WithSize(16)),
										g.Text("You can safely close this window"),
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
