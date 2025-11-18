package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/plugins/dashboard/components"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// GeneralSettingsPageData contains data for the general settings page
type GeneralSettingsPageData struct {
	Settings  GeneralSettings
	BasePath  string
	CSRFToken string
	AppID     string
}

// GeneralSettingsPage renders the general settings page
func GeneralSettingsPage(data GeneralSettingsPageData) g.Node {
	return Div(
		Class("space-y-6"),

		components.SettingsPageHeader("General Settings", "Configure basic dashboard settings and preferences"),

		components.SettingsSection("Dashboard Configuration", "", 
			FormEl(
				Method("POST"),
				Action(fmt.Sprintf("%s/dashboard/app/%s/settings/general", data.BasePath, data.AppID)),
				Class("space-y-6"),

				Input(Type("hidden"), Name("csrf_token"), Value(data.CSRFToken)),

				// Dashboard Name
				components.SettingsFormField(
					"Dashboard Name",
					"The display name for your dashboard",
					"dashboard_name",
					Input(
						Type("text"),
						ID("dashboard_name"),
						Name("dashboard_name"),
						Value(data.Settings.DashboardName),
						Class("mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
				),

				// Session Duration
				components.SettingsFormField(
					"Default Session Duration",
					"How long users stay logged in (in hours)",
					"session_duration",
					Input(
						Type("number"),
						ID("session_duration"),
						Name("session_duration"),
						Value(fmt.Sprintf("%d", data.Settings.SessionDuration)),
						g.Attr("min", "1"),
						g.Attr("max", "720"),
						Class("mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
				),

				// Max Login Attempts
				components.SettingsFormField(
					"Max Login Attempts",
					"Number of failed login attempts before account lockout",
					"max_login_attempts",
					Input(
						Type("number"),
						ID("max_login_attempts"),
						Name("max_login_attempts"),
						Value(fmt.Sprintf("%d", data.Settings.MaxLoginAttempts)),
						g.Attr("min", "1"),
						g.Attr("max", "20"),
						Class("mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"),
					),
				),

				// Require Email Verification
				components.SettingsFormField(
					"Email Verification",
					"Require users to verify their email address",
					"require_email_verification",
					Div(
						Class("flex items-center"),
						Input(
							Type("checkbox"),
							ID("require_email_verification"),
							Name("require_email_verification"),
							Value("true"),
							g.If(data.Settings.RequireEmailVerification, Checked()),
							Class("h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800"),
						),
						Label(
							For("require_email_verification"),
							Class("ml-2 text-sm text-gray-700 dark:text-gray-300"),
							g.Text("Enable email verification for new accounts"),
						),
					),
				),

				components.SettingsActions("Save Changes", ""),
			),
		),

		// Security Section
		components.SettingsSection("Security Settings", "Configure security and authentication policies", 
			Div(
				Class("space-y-4"),
				Div(
					Class("flex items-center justify-between"),
					Div(
						H4(Class("text-sm font-medium text-gray-900 dark:text-white"), g.Text("Two-Factor Authentication")),
						P(Class("text-sm text-gray-500 dark:text-gray-400"), g.Text("Require 2FA for admin users")),
					),
					Button(
						Type("button"),
						Class("rounded-md bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
						g.Text("Configure"),
					),
				),
				Div(
					Class("flex items-center justify-between border-t border-gray-200 dark:border-gray-800 pt-4"),
					Div(
						H4(Class("text-sm font-medium text-gray-900 dark:text-white"), g.Text("Session Management")),
						P(Class("text-sm text-gray-500 dark:text-gray-400"), g.Text("Control active user sessions")),
					),
					Button(
						Type("button"),
						Class("rounded-md bg-gray-100 px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"),
						g.Text("Manage"),
					),
				),
			),
		),

		// Help Section
		Div(
			Class("rounded-lg border border-blue-200 bg-blue-50 dark:border-blue-900 dark:bg-blue-900/20 p-4"),
			Div(
				Class("flex items-start gap-3"),
				lucide.Info(Class("h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5")),
				Div(
					H4(Class("text-sm font-medium text-blue-900 dark:text-blue-100"), g.Text("Need Help?")),
					P(Class("mt-1 text-sm text-blue-700 dark:text-blue-300"),
						g.Text("Changes to these settings will affect all users in your application. "),
						A(
							Href("https://docs.authsome.dev"),
							Class("font-medium underline"),
							g.Text("Learn more"),
						),
						g.Text(" about general settings."),
					),
				),
			),
		),
	)
}

