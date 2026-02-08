package notification

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// renderNotificationSettings renders the notification settings page
func (e *DashboardExtension) renderNotificationSettings(currentApp *app.App, basePath string, success bool) g.Node {
	cfg := e.plugin.config

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text("Notification Settings")),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"), g.Text("Configure notification behavior and auto-send rules")),
			),
		),

		// Success message
		g.If(success,
			Div(
				Class("bg-green-50 border border-green-200 rounded-lg p-4 dark:bg-green-900/20 dark:border-green-800"),
				Div(
					Class("flex"),
					Div(
						Class("flex-shrink-0"),
						lucide.Check(Class("h-5 w-5 text-green-400")),
					),
					Div(
						Class("ml-3"),
						P(Class("text-sm font-medium text-green-800 dark:text-green-200"), g.Text("Settings updated successfully")),
					),
				),
			),
		),

		// Settings form
		Form(
			Method("POST"),
			Action(basePath+"/app/"+currentApp.ID.String()+"/notifications/settings"),
			Class("space-y-6"),

			// General Settings
			Div(
				Class("bg-white dark:bg-gray-800 rounded-lg border border-slate-200 dark:border-gray-700 p-6"),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"), g.Text("General Settings")),

				Div(
					Class("space-y-4"),
					// App Name
					Div(
						Label(
							For("app_name"),
							Class("block text-sm font-medium text-slate-700 dark:text-gray-300 mb-2"),
							g.Text("Application Name"),
						),
						Input(
							Type("text"),
							Name("app_name"),
							ID("app_name"),
							Value(cfg.AppName),
							Placeholder(currentApp.Name),
							Class("w-full px-3 py-2 border border-slate-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"),
						),
						P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"), g.Text("Used in email notifications. Leave empty to use app name.")),
					),
				),
			),

			// Authentication Notifications
			e.renderSettingsSection("Authentication Notifications", "Automatic notifications for authentication events", []settingField{
				{Name: "auth_welcome", Label: "Welcome Email", Description: "Send welcome email on user signup", Checked: cfg.AutoSend.Auth.Welcome},
			}),

			// Organization Notifications
			e.renderSettingsSection("Organization Notifications", "Automatic notifications for organization events", []settingField{
				{Name: "org_invite", Label: "Organization Invite", Description: "Send invitation emails", Checked: cfg.AutoSend.Organization.Invite},
				{Name: "org_member_added", Label: "Member Added", Description: "Notify when member is added", Checked: cfg.AutoSend.Organization.MemberAdded},
				{Name: "org_member_removed", Label: "Member Removed", Description: "Notify when member is removed", Checked: cfg.AutoSend.Organization.MemberRemoved},
				{Name: "org_role_changed", Label: "Role Changed", Description: "Notify on role changes", Checked: cfg.AutoSend.Organization.RoleChanged},
				{Name: "org_transfer", Label: "Ownership Transfer", Description: "Notify on ownership transfer", Checked: cfg.AutoSend.Organization.Transfer},
				{Name: "org_deleted", Label: "Organization Deleted", Description: "Notify on organization deletion", Checked: cfg.AutoSend.Organization.Deleted},
				{Name: "org_member_left", Label: "Member Left", Description: "Notify when member leaves", Checked: cfg.AutoSend.Organization.MemberLeft},
			}),

			// Session/Device Security Notifications
			e.renderSettingsSection("Session & Device Security", "Automatic notifications for security events", []settingField{
				{Name: "session_new_device", Label: "New Device Login", Description: "Notify on new device login", Checked: cfg.AutoSend.Session.NewDevice},
				{Name: "session_new_location", Label: "New Location Login", Description: "Notify on new location login", Checked: cfg.AutoSend.Session.NewLocation},
				{Name: "session_suspicious_login", Label: "Suspicious Login", Description: "Notify on suspicious activity", Checked: cfg.AutoSend.Session.SuspiciousLogin},
				{Name: "session_device_removed", Label: "Device Removed", Description: "Notify when device removed", Checked: cfg.AutoSend.Session.DeviceRemoved},
				{Name: "session_all_revoked", Label: "All Sessions Revoked", Description: "Notify on mass signout", Checked: cfg.AutoSend.Session.AllRevoked},
			}),

			// Account Lifecycle Notifications
			e.renderSettingsSection("Account Lifecycle", "Automatic notifications for account changes", []settingField{
				{Name: "account_email_change_request", Label: "Email Change Request", Description: "Send confirmation for email change", Checked: cfg.AutoSend.Account.EmailChangeRequest},
				{Name: "account_email_changed", Label: "Email Changed", Description: "Notify on email change completion", Checked: cfg.AutoSend.Account.EmailChanged},
				{Name: "account_password_changed", Label: "Password Changed", Description: "Notify on password change", Checked: cfg.AutoSend.Account.PasswordChanged},
				{Name: "account_username_changed", Label: "Username Changed", Description: "Notify on username change", Checked: cfg.AutoSend.Account.UsernameChanged},
				{Name: "account_deleted", Label: "Account Deleted", Description: "Notify on account deletion", Checked: cfg.AutoSend.Account.Deleted},
				{Name: "account_suspended", Label: "Account Suspended", Description: "Notify on account suspension", Checked: cfg.AutoSend.Account.Suspended},
				{Name: "account_reactivated", Label: "Account Reactivated", Description: "Notify on account reactivation", Checked: cfg.AutoSend.Account.Reactivated},
			}),

			// Save button
			Div(
				Class("flex items-center justify-end gap-3 pt-6 border-t border-slate-200 dark:border-gray-700"),
				A(
					Href(basePath+"/app/"+currentApp.ID.String()+"/notifications"),
					Class("px-4 py-2 text-sm font-medium text-slate-700 dark:text-gray-300 hover:text-slate-900 dark:hover:text-white transition"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium rounded-lg transition"),
					g.Text("Save Settings"),
				),
			),
		),
	)
}

type settingField struct {
	Name        string
	Label       string
	Description string
	Checked     bool
}

func (e *DashboardExtension) renderSettingsSection(title, description string, fields []settingField) g.Node {
	var fieldNodes []g.Node

	for _, field := range fields {
		fieldNodes = append(fieldNodes, Div(
			Class("flex items-start"),
			Div(
				Class("flex items-center h-5"),
				Input(
					Type("checkbox"),
					Name(field.Name),
					ID(field.Name),
					g.If(field.Checked, Checked()),
					Class("w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500 dark:focus:ring-blue-600 dark:ring-offset-gray-800 focus:ring-2 dark:bg-gray-700 dark:border-gray-600"),
				),
			),
			Div(
				Class("ml-3"),
				Label(
					For(field.Name),
					Class("font-medium text-sm text-slate-900 dark:text-white"),
					g.Text(field.Label),
				),
				P(Class("text-xs text-slate-500 dark:text-gray-400"), g.Text(field.Description)),
			),
		))
	}

	return Div(
		Class("bg-white dark:bg-gray-800 rounded-lg border border-slate-200 dark:border-gray-700 p-6"),
		H2(Class("text-lg font-semibold text-slate-900 dark:text-white mb-2"), g.Text(title)),
		P(Class("text-sm text-slate-600 dark:text-gray-400 mb-4"), g.Text(description)),
		Div(Class("space-y-3"), g.Group(fieldNodes)),
	)
}
