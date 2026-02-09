package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/input"
)

// SettingsPage renders the multisession settings page.
func SettingsPage(currentApp *app.App, basePath string) g.Node {
	appID := currentApp.ID.String()

	return Div(
		Class("space-y-6"),

		// Page header
		PageHeader(
			"Session Settings",
			"Configure multi-session behavior, limits, and security settings",
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
				Div(
					g.Attr("x-show", "successMessage"),
					g.Attr("x-transition", ""),
					Class("bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800 rounded-lg p-4"),
					Div(
						Class("flex items-center gap-2 text-emerald-700 dark:text-emerald-400"),
						lucide.CircleCheck(Class("size-5")),
						Span(g.Attr("x-text", "successMessage")),
					),
				),

				// Session Limits section
				card.Card(
					card.Header(
						Div(
							Class("flex items-center gap-3"),
							Div(
								Class("rounded-lg bg-primary/10 p-2"),
								lucide.Layers(Class("size-5 text-primary")),
							),
							Div(
								card.Title("Session Limits"),
								card.Description("Control how many sessions users can have"),
							),
						),
					),
					card.Content(
						Div(
							Class("grid gap-6 md:grid-cols-2"),

							// Max sessions per user
							Div(
								Class("space-y-2"),
								Label(
									For("maxSessionsPerUser"),
									Class("text-sm font-medium"),
									g.Text("Max Sessions Per User"),
								),
								input.Input(
									input.WithType("number"),
									input.WithName("maxSessionsPerUser"),
									input.WithID("maxSessionsPerUser"),
									input.WithAttrs(
										g.Attr("x-model.number", "settings.maxSessionsPerUser"),
										Min("1"),
										Max("100"),
									),
								),
								P(Class("text-sm text-muted-foreground"),
									g.Text("Maximum number of concurrent sessions a user can have active")),
							),

							// Session expiry hours
							Div(
								Class("space-y-2"),
								Label(
									For("sessionExpiryHours"),
									Class("text-sm font-medium"),
									g.Text("Session Expiry (Hours)"),
								),
								input.Input(
									input.WithType("number"),
									input.WithName("sessionExpiryHours"),
									input.WithID("sessionExpiryHours"),
									input.WithAttrs(
										g.Attr("x-model.number", "settings.sessionExpiryHours"),
										Min("1"),
										Max("8760"),
									),
								),
								P(Class("text-sm text-muted-foreground"),
									g.Text("How long sessions remain valid before expiring (max: 8760 = 1 year)")),
							),
						),
					),
				),

				// Device Tracking section
				card.Card(
					card.Header(
						Div(
							Class("flex items-center gap-3"),
							Div(
								Class("rounded-lg bg-emerald-100 dark:bg-emerald-900/30 p-2"),
								lucide.MonitorSmartphone(Class("size-5 text-emerald-600 dark:text-emerald-400")),
							),
							Div(
								card.Title("Device & Platform"),
								card.Description("Configure device tracking and cross-platform settings"),
							),
						),
					),
					card.Content(
						Div(
							Class("space-y-4"),

							// Device tracking toggle
							toggleSetting(
								"enableDeviceTracking",
								"Enable Device Tracking",
								"Track and identify devices for each session",
								lucide.Fingerprint(Class("size-5 text-muted-foreground")),
							),

							// Cross-platform toggle
							toggleSetting(
								"allowCrossPlatform",
								"Allow Cross-Platform Sessions",
								"Allow users to have sessions on different platforms simultaneously",
								lucide.Globe(Class("size-5 text-muted-foreground")),
							),
						),
					),
				),

				// Save button
				Div(
					Class("flex justify-end"),
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
	)
}

// settingsData returns the Alpine.js data object for settings.
func settingsData(appID string) string {
	return fmt.Sprintf(`{
		settings: {
			maxSessionsPerUser: 10,
			sessionExpiryHours: 720,
			enableDeviceTracking: true,
			allowCrossPlatform: true
		},
		loading: true,
		saving: false,
		error: null,
		successMessage: null,
		
		async loadSettings() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('multisession.getSettings', {
					appId: '%s'
				});
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
				const result = await $bridge.call('multisession.updateSettings', {
					appId: '%s',
					maxSessionsPerUser: this.settings.maxSessionsPerUser,
					sessionExpiryHours: this.settings.sessionExpiryHours,
					enableDeviceTracking: this.settings.enableDeviceTracking,
					allowCrossPlatform: this.settings.allowCrossPlatform
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
	}`, appID, appID)
}

// toggleSetting renders a toggle setting row.
func toggleSetting(name, label, description string, icon g.Node) g.Node {
	return Div(
		Class("flex items-center justify-between rounded-lg border bg-muted/50 p-4"),
		Div(
			Class("flex items-center gap-3"),
			icon,
			Div(
				Div(Class("font-medium"), g.Text(label)),
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
