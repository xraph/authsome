package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/primitives"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// SettingsPage renders the notification settings page.
func SettingsPage(currentApp *app.App, basePath string) g.Node {
	appID := currentApp.ID.String()

	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Page header
			PageHeader(
				"Notification Settings",
				"Configure automatic notifications for authentication, organization, and account events",
			),

			// Dynamic content with Alpine.js
			Div(
				g.Attr("x-data", settingsData(appID)),
				g.Attr("x-init", "loadSettings()"),

				// Loading state
				Div(
					g.Attr("x-show", "loading"),
					LoadingSpinner(),
				),

				// Error state
				ErrorMessage("error && !loading"),

				// Content
				Div(
					g.Attr("x-show", "!loading && !error"),
					g.Attr("x-cloak", ""),
					Class("space-y-6"),

					// Success message
					SuccessMessage("successMessage"),

					// Application Settings
					card.Card(
						card.Header(
							Div(
								Class("flex items-center gap-3"),
								Div(
									Class("rounded-lg bg-primary/10 p-2"),
									lucide.Settings(Class("size-5 text-primary")),
								),
								Div(
									card.Title("Application Settings"),
									card.Description("General notification configuration"),
								),
							),
						),
						card.Content(
							Div(
								Class("space-y-4"),
								// App Name
								Div(
									Class("space-y-2"),
									Label(
										For("appName"),
										Class("text-sm font-medium"),
										g.Text("Application Name"),
									),
									input.Input(
										input.WithType("text"),
										input.WithName("appName"),
										input.WithID("appName"),
										input.WithPlaceholder("Used in notification emails"),
										input.WithAttrs(
											g.Attr("x-model", "settings.appName"),
										),
									),
									P(Class("text-sm text-muted-foreground"),
										g.Text("Name used in email notifications. Leave empty to use app name.")),
								),
							),
						),
					),

					// Authentication Notifications
					card.Card(
						card.Header(
							Div(
								Class("flex items-center gap-3"),
								Div(
									Class("rounded-lg bg-blue-100 dark:bg-blue-900/30 p-2"),
									lucide.Shield(Class("size-5 text-blue-600 dark:text-blue-400")),
								),
								Div(
									card.Title("Authentication Notifications"),
									card.Description("Automatic notifications for authentication events"),
								),
							),
						),
						card.Content(
							Div(
								Class("space-y-3"),
								toggleField(
									"auth.welcome",
									"Welcome Email",
									"Send welcome email on user signup",
									lucide.Mail(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"auth.verificationEmail",
									"Verification Email",
									"Send email verification link",
									lucide.MailCheck(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"auth.magicLink",
									"Magic Link",
									"Send passwordless login link",
									lucide.Link(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"auth.emailOtp",
									"Email OTP",
									"Send one-time password via email",
									lucide.KeyRound(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"auth.mfaCode",
									"MFA Code",
									"Send multi-factor authentication code",
									lucide.ShieldCheck(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"auth.passwordReset",
									"Password Reset",
									"Send password reset link",
									lucide.KeySquare(Class("size-5 text-muted-foreground")),
								),
							),
						),
					),

					// Organization Notifications
					card.Card(
						card.Header(
							Div(
								Class("flex items-center gap-3"),
								Div(
									Class("rounded-lg bg-purple-100 dark:bg-purple-900/30 p-2"),
									lucide.Users(Class("size-5 text-purple-600 dark:text-purple-400")),
								),
								Div(
									card.Title("Organization Notifications"),
									card.Description("Automatic notifications for organization events"),
								),
							),
						),
						card.Content(
							Div(
								Class("space-y-3"),
								toggleField(
									"organization.invite",
									"Organization Invite",
									"Send invitation emails to new members",
									lucide.UserPlus(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"organization.memberAdded",
									"Member Added",
									"Notify when a member is added to the organization",
									lucide.UserCheck(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"organization.memberRemoved",
									"Member Removed",
									"Notify when a member is removed",
									lucide.UserMinus(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"organization.roleChanged",
									"Role Changed",
									"Notify when member role is changed",
									lucide.UserCog(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"organization.transfer",
									"Ownership Transfer",
									"Notify on organization ownership transfer",
									lucide.ArrowRightLeft(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"organization.deleted",
									"Organization Deleted",
									"Notify members when organization is deleted",
									lucide.Trash2(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"organization.memberLeft",
									"Member Left",
									"Notify when a member leaves the organization",
									lucide.LogOut(Class("size-5 text-muted-foreground")),
								),
							),
						),
					),

					// Session & Security Notifications
					card.Card(
						card.Header(
							Div(
								Class("flex items-center gap-3"),
								Div(
									Class("rounded-lg bg-orange-100 dark:bg-orange-900/30 p-2"),
									lucide.Lock(Class("size-5 text-orange-600 dark:text-orange-400")),
								),
								Div(
									card.Title("Session & Security"),
									card.Description("Security-related notifications for suspicious activities"),
								),
							),
						),
						card.Content(
							Div(
								Class("space-y-3"),
								toggleField(
									"session.newDevice",
									"New Device Login",
									"Notify on login from a new device",
									lucide.Smartphone(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"session.newLocation",
									"New Location Login",
									"Notify on login from a new location",
									lucide.MapPin(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"session.suspiciousLogin",
									"Suspicious Login",
									"Notify on suspicious login activity",
									lucide.TriangleAlert(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"session.deviceRemoved",
									"Device Removed",
									"Notify when a trusted device is removed",
									lucide.Trash2(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"session.allRevoked",
									"All Sessions Revoked",
									"Notify on mass session signout",
									lucide.LogOut(Class("size-5 text-muted-foreground")),
								),
							),
						),
					),

					// Account Lifecycle Notifications
					card.Card(
						card.Header(
							Div(
								Class("flex items-center gap-3"),
								Div(
									Class("rounded-lg bg-emerald-100 dark:bg-emerald-900/30 p-2"),
									lucide.CircleUser(Class("size-5 text-emerald-600 dark:text-emerald-400")),
								),
								Div(
									card.Title("Account Lifecycle"),
									card.Description("Notifications for account changes and updates"),
								),
							),
						),
						card.Content(
							Div(
								Class("space-y-3"),
								toggleField(
									"account.emailChangeRequest",
									"Email Change Request",
									"Send confirmation for email change requests",
									lucide.Mail(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"account.emailChanged",
									"Email Changed",
									"Notify on email change completion",
									lucide.MailCheck(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"account.passwordChanged",
									"Password Changed",
									"Notify on password change",
									lucide.Key(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"account.usernameChanged",
									"Username Changed",
									"Notify on username change",
									lucide.AtSign(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"account.deleted",
									"Account Deleted",
									"Send final notification on account deletion",
									lucide.Trash2(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"account.suspended",
									"Account Suspended",
									"Notify when account is suspended",
									lucide.Ban(Class("size-5 text-muted-foreground")),
								),
								toggleField(
									"account.reactivated",
									"Account Reactivated",
									"Notify on account reactivation",
									lucide.CircleCheck(Class("size-5 text-muted-foreground")),
								),
							),
						),
					),

					// Save button
					Div(
						Class("flex justify-end pt-4"),
						button.Button(
							Div(
								Class("flex items-center gap-2"),
								lucide.Save(Class("size-4")),
								g.Text("Save Settings"),
							),
							button.WithAttrs(
								g.Attr("@click", "saveSettings()"),
								g.Attr(":disabled", "saving"),
							),
						),
					),
				),
			),
		),
	)
}

// settingsData returns the Alpine.js data object for settings.
func settingsData(appID string) string {
	return `{
		settings: {
			appName: '',
			auth: {
				welcome: true,
				verificationEmail: false,
				magicLink: false,
				emailOtp: false,
				mfaCode: false,
				passwordReset: false
			},
			organization: {
				invite: true,
				memberAdded: true,
				memberRemoved: true,
				roleChanged: true,
				transfer: true,
				deleted: true,
				memberLeft: true
			},
			session: {
				newDevice: true,
				newLocation: true,
				suspiciousLogin: true,
				deviceRemoved: true,
				allRevoked: true
			},
			account: {
				emailChangeRequest: true,
				emailChanged: true,
				passwordChanged: true,
				usernameChanged: true,
				deleted: true,
				suspended: true,
				reactivated: true
			}
		},
		loading: true,
		saving: false,
		error: null,
		successMessage: null,
		
		async loadSettings() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('notification.getSettings', {});
				this.settings = result.settings || this.settings;
			} catch (err) {
				console.error('Failed to load settings:', err);
				this.error = err.message || 'Failed to load settings';
			} finally {
				this.loading = false;
			}
		},
		
		async saveSettings() {
			this.saving = true;
			this.error = null;
			this.successMessage = null;
			try {
				const result = await $bridge.call('notification.updateSettings', {
					appName: this.settings.appName,
					auth: this.settings.auth,
					organization: this.settings.organization,
					session: this.settings.session,
					account: this.settings.account
				});
				
				if (result.settings) {
					this.settings = result.settings;
				}
				
				this.successMessage = result.message || 'Settings saved successfully';
				
				// Clear success message after 3 seconds
				setTimeout(() => {
					this.successMessage = null;
				}, 3000);
			} catch (err) {
				console.error('Failed to save settings:', err);
				this.error = err.message || 'Failed to save settings';
			} finally {
				this.saving = false;
			}
		}
	}`
}

// toggleField renders a toggle setting row.
func toggleField(name, label, description string, icon g.Node) g.Node {
	return Div(
		Class("flex items-center justify-between rounded-lg border bg-muted/50 p-4"),
		Div(
			Class("flex items-center gap-3"),
			icon,
			Div(
				Div(Class("font-medium text-sm"), g.Text(label)),
				Div(Class("text-sm text-muted-foreground"), g.Text(description)),
			),
		),
		// Toggle switch
		Label(
			Class("relative inline-flex cursor-pointer items-center"),
			Input(
				Type("checkbox"),
				g.Attr("x-model", "settings."+name),
				Class("peer sr-only"),
			),
			Span(
				Class("peer h-6 w-11 rounded-full bg-muted after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:border after:border-muted-foreground/20 after:bg-background after:transition-all after:content-[''] peer-checked:bg-primary peer-checked:after:translate-x-full peer-checked:after:border-primary-foreground peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/25"),
			),
		),
	)
}
