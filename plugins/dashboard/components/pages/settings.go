package pages

import (
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// GeneralSettings contains general dashboard settings
type GeneralSettings struct {
	DashboardName            string
	SessionDuration          int
	MaxLoginAttempts         int
	RequireEmailVerification bool
}

// APIKey represents an API key
type APIKey struct {
	ID         string
	Name       string
	Key        string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// Webhook represents a webhook configuration
type Webhook struct {
	ID      string
	URL     string
	Events  []string
	Enabled bool
}

// NotificationTemplate represents an email/SMS template
type NotificationTemplate struct {
	ID       string
	Name     string
	Type     string // "email" or "sms"
	Subject  string
	Language string
	Updated  time.Time
}

// SocialProvider represents an OAuth provider configuration
type SocialProvider struct {
	ID          string
	Provider    string // "google", "github", etc.
	Name        string
	ClientID    string
	Enabled     bool
	RedirectURL string
}

// ImpersonationLog represents an impersonation audit entry
type ImpersonationLog struct {
	ID         string
	AdminID    string
	AdminEmail string
	UserID     string
	UserEmail  string
	Reason     string
	StartedAt  time.Time
	EndedAt    *time.Time
	Duration   string
}

// MFAMethod represents an MFA method configuration
type MFAMethod struct {
	ID      string
	Type    string // "totp", "sms", "email", "webauthn"
	Name    string
	Enabled bool
}

// SettingsPageData contains data for the settings page
type SettingsPageData struct {
	ActiveTab             string // "general", "apikeys", "webhooks", "notifications", "social", "impersonation", "mfa"
	General               GeneralSettings
	APIKeys               APIKeysTabPageData // Full API keys page data
	Webhooks              []Webhook
	NotificationTemplates []NotificationTemplate
	SocialProviders       []SocialProvider
	ImpersonationLogs     []ImpersonationLog
	MFAMethods            []MFAMethod
	IsSaaSMode            bool
	BasePath              string
	CSRFToken             string
	EnabledPlugins        map[string]bool
}

// SettingsPage renders the complete settings page with tabs
func SettingsPage(data SettingsPageData) g.Node {
	// Default to general tab if not specified
	activeTab := data.ActiveTab
	if activeTab == "" {
		activeTab = "general"
	}

	return Div(
		Class("space-y-6"),
		g.Attr("x-data", fmt.Sprintf("{ activeTab: '%s' }", activeTab)),

		// Page Header
		pageHeader(),

		// Tabs Navigation
		tabsNavigation(data.EnabledPlugins),

		// Tab Content: General Settings
		generalTabContent(data),

		// Tab Content: API Keys (conditional)
		g.If(data.EnabledPlugins["apikey"],
			Div(
				g.Attr("x-show", "activeTab === 'apikeys'"),
				g.Attr("x-cloak", ""),
				apiKeysTabContent(data.APIKeys),
			),
		),

		// Tab Content: Webhooks (conditional)
		g.If(data.EnabledPlugins["webhook"],
			webhooksTabContent(data),
		),

		// Tab Content: Notifications (conditional)
		g.If(data.EnabledPlugins["notification"],
			notificationsTabContent(data),
		),

		// Tab Content: Social Providers (conditional)
		g.If(data.EnabledPlugins["social"],
			socialProvidersTabContent(data),
		),

		// Tab Content: Impersonation Logs (conditional)
		g.If(data.EnabledPlugins["impersonation"],
			impersonationTabContent(data),
		),

		// Tab Content: MFA Settings (conditional)
		g.If(data.EnabledPlugins["mfa"],
			mfaTabContent(data),
		),
	)
}

func pageHeader() g.Node {
	return Div(
		H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
			g.Text("Settings"),
		),
		P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
			g.Text("Manage your dashboard configuration, API keys, and integrations"),
		),
	)
}

func tabsNavigation(enabledPlugins map[string]bool) g.Node {
	return Div(Class("border-b border-slate-200 dark:border-gray-800"),
		Nav(Class("-mb-px flex space-x-8"),
			// General Tab
			tabButton("general", "General", lucide.Settings(Class("h-5 w-5"))),

			// API Keys Tab (conditional)
			g.If(enabledPlugins["apikey"],
				tabButton("apikeys", "API Keys", lucide.Key(Class("h-5 w-5"))),
			),

			// Webhooks Tab (conditional)
			g.If(enabledPlugins["webhook"],
				tabButton("webhooks", "Webhooks", lucide.Link(Class("h-5 w-5"))),
			),

			// Notifications Tab (conditional)
			g.If(enabledPlugins["notification"],
				tabButton("notifications", "Notifications", lucide.Mail(Class("h-5 w-5"))),
			),

			// Social Providers Tab (conditional)
			g.If(enabledPlugins["social"],
				tabButton("social", "Social Login", lucide.Users(Class("h-5 w-5"))),
			),

			// Impersonation Tab (conditional)
			g.If(enabledPlugins["impersonation"],
				tabButton("impersonation", "Impersonation", lucide.UserRound(Class("h-5 w-5"))),
			),

			// MFA Tab (conditional)
			g.If(enabledPlugins["mfa"],
				tabButton("mfa", "MFA", lucide.ShieldCheck(Class("h-5 w-5"))),
			),
		),
	)
}

func tabButton(tabName, label string, icon g.Node) g.Node {
	return Button(
		Type("button"),
		g.Attr("@click", fmt.Sprintf("activeTab = '%s'", tabName)),
		g.Attr(":class", fmt.Sprintf("activeTab === '%s' ? 'border-violet-500 text-violet-600 dark:text-violet-400' : 'border-transparent text-slate-500 dark:text-gray-400 hover:border-slate-300 dark:hover:border-gray-600 hover:text-slate-700 dark:hover:text-gray-300'", tabName)),
		Class("whitespace-nowrap border-b-2 py-4 px-1 text-sm font-medium transition-colors"),
		Div(Class("flex items-center gap-2"),
			icon,
			g.Text(label),
		),
	)
}

// General Settings Tab
func generalTabContent(data SettingsPageData) g.Node {
	return Div(
		g.Attr("x-show", "activeTab === 'general'"),
		g.Attr("x-cloak", ""),
		Div(Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900"),
			// Card Header
			Div(Class("p-6 border-b border-slate-100 dark:border-gray-800"),
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("General Settings"),
				),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Configure basic dashboard settings and preferences"),
				),
			),

			// Form
			Div(Class("p-6"),
				FormEl(
					Method("POST"),
					Action(data.BasePath+"/dashboard/settings/general"),
					Class("space-y-6"),

					Input(Type("hidden"), Name("csrf_token"), Value(data.CSRFToken)),

					// Dashboard Name
					formField(
						"dashboard_name",
						"Dashboard Name",
						"text",
						data.General.DashboardName,
						"",
						false,
					),

					// Session Duration
					formField(
						"session_duration",
						"Default Session Duration (hours)",
						"number",
						fmt.Sprintf("%d", data.General.SessionDuration),
						"How long users stay logged in",
						false,
						g.Attr("min", "1"),
						g.Attr("max", "720"),
					),

					// Max Login Attempts
					formField(
						"max_login_attempts",
						"Max Login Attempts",
						"number",
						fmt.Sprintf("%d", data.General.MaxLoginAttempts),
						"Before account lockout",
						false,
						g.Attr("min", "1"),
						g.Attr("max", "20"),
					),

					// Require Email Verification
					Div(Class("flex items-center"),
						Input(
							Type("checkbox"),
							ID("require_email_verification"),
							Name("require_email_verification"),
							g.If(data.General.RequireEmailVerification, Checked()),
							Class("h-4 w-4 rounded border-slate-300 dark:border-gray-600 text-violet-600 focus:ring-violet-500 dark:bg-gray-700"),
						),
						Label(
							For("require_email_verification"),
							Class("ml-2 block text-sm text-slate-900 dark:text-gray-300"),
							g.Text("Require email verification for new accounts"),
						),
					),

					// Save Button
					Div(Class("flex justify-end pt-4 border-t border-slate-100 dark:border-gray-800"),
						Button(
							Type("submit"),
							Class("inline-flex items-center justify-center gap-2 rounded-lg border border-transparent bg-violet-600 px-4 py-2 text-sm leading-5 font-semibold text-white hover:bg-violet-700 focus:ring-3 focus:ring-violet-400/50 active:bg-violet-700 transition-colors"),
							lucide.Check(Class("h-4 w-4")),
							g.Text("Save Changes"),
						),
					),
				),
			),
		),
	)
}

func formField(name, label, inputType, value, helpText string, required bool, extraAttrs ...g.Node) g.Node {
	attrs := []g.Node{
		Type(inputType),
		ID(name),
		Name(name),
		Value(value),
		Class("mt-1 block w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-gray-500 focus:border-violet-300 dark:focus:border-violet-700 focus:ring-2 focus:ring-violet-400/20 sm:text-sm"),
	}

	if required {
		attrs = append(attrs, Required())
	}

	attrs = append(attrs, extraAttrs...)

	return Div(
		Label(
			For(name),
			Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
			g.Text(label),
		),
		Input(attrs...),
		g.If(helpText != "",
			P(Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
				g.Text(helpText),
			),
		),
	)
}

// Note: API Keys tab content moved to apikeys.go for comprehensive implementation

// Webhooks Tab
func webhooksTabContent(data SettingsPageData) g.Node {
	return Div(
		g.Attr("x-show", "activeTab === 'webhooks'"),
		g.Attr("x-cloak", ""),
		Div(Class("space-y-6"),
			// Header with Create Button
			Div(Class("flex items-center justify-between"),
				Div(
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Webhooks"),
					),
					P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
						g.Text("Configure webhooks to receive real-time notifications"),
					),
				),
				Button(
					Type("button"),
					g.Attr("@click", fmt.Sprintf("window.location.href='%s/dashboard/settings/webhooks/new'", data.BasePath)),
					Class("inline-flex items-center gap-2 rounded-lg border border-transparent bg-violet-600 px-4 py-2 text-sm leading-5 font-semibold text-white hover:bg-violet-700 transition-colors"),
					lucide.Plus(Class("h-4 w-4")),
					g.Text("Create Webhook"),
				),
			),

			// Webhooks List or Empty State
			g.If(len(data.Webhooks) > 0,
				webhooksList(data.Webhooks),
			),
			g.If(len(data.Webhooks) == 0,
				emptyState(
					lucide.Link(Class("h-8 w-8 text-slate-400 dark:text-gray-500")),
					"No Webhooks Configured",
					"Create your first webhook to receive event notifications",
					"Available events: user.created, user.login, session.created, and more",
				),
			),
		),
	)
}

func webhooksList(webhooks []Webhook) g.Node {
	items := make([]g.Node, 0, len(webhooks))
	for _, webhook := range webhooks {
		items = append(items, webhookItem(webhook))
	}

	return Div(Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden"),
		Div(Class("divide-y divide-slate-100 dark:divide-gray-800"),
			g.Group(items),
		),
	)
}

func webhookItem(webhook Webhook) g.Node {
	// Format events list
	eventsText := ""
	for i, event := range webhook.Events {
		if i > 0 {
			eventsText += ", "
		}
		eventsText += event
	}

	return Div(Class("p-5 hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),
		Div(Class("flex items-start justify-between"),
			Div(Class("flex-1"),
				Div(Class("flex items-center gap-3"),
					H4(Class("font-semibold text-slate-900 dark:text-white"),
						g.Text(webhook.URL),
					),
					g.If(webhook.Enabled,
						Span(Class("inline-flex items-center rounded-full bg-emerald-100 dark:bg-emerald-500/20 px-2.5 py-0.5 text-xs leading-4 font-semibold text-emerald-800 dark:text-emerald-400"),
							g.Text("Active"),
						),
					),
					g.If(!webhook.Enabled,
						Span(Class("inline-flex items-center rounded-full bg-slate-100 dark:bg-gray-800 px-2.5 py-0.5 text-xs leading-4 font-semibold text-slate-800 dark:text-gray-400"),
							g.Text("Inactive"),
						),
					),
				),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Events: "+eventsText),
				),
			),
			Div(Class("flex items-center gap-2 ml-4"),
				Button(
					Type("button"),
					Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-2.5 py-1.5 text-sm font-medium text-slate-800 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-800 transition-colors"),
					g.Text("Test"),
				),
				Button(
					Type("button"),
					Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-2.5 py-1.5 text-sm font-medium text-slate-800 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-800 transition-colors"),
					g.Text("Edit"),
				),
				Button(
					Type("button"),
					Class("inline-flex items-center gap-1 rounded-lg border border-rose-200 dark:border-rose-800 px-2.5 py-1.5 text-sm font-medium text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-900/20 transition-colors"),
					g.Text("Delete"),
				),
			),
		),
	)
}

func emptyState(icon g.Node, title, message, subMessage string) g.Node {
	return Div(Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden"),
		Div(Class("p-12 text-center"),
			Div(Class("mx-auto h-16 w-16 rounded-full bg-slate-100 dark:bg-gray-800 flex items-center justify-center mb-4"),
				icon,
			),
			H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text(title),
			),
			P(Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
				g.Text(message),
			),
			g.If(subMessage != "",
				P(Class("mt-1 text-xs text-slate-500 dark:text-gray-500"),
					g.Text(subMessage),
				),
			),
		),
	)
}
