package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// SettingsGeneralPage shows general settings
func (p *PagesManager) SettingsGeneralPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold"), g.Text("General Settings")),
					P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Configure basic application settings")),
				),
			),

			Div(
				g.Attr("x-data", `{
					settings: {
						appName: '',
						description: '',
						timezone: 'UTC',
						language: 'en',
						dateFormat: 'YYYY-MM-DD',
						supportEmail: '',
						websiteUrl: ''
					},
					errors: {},
					loading: true,
					saving: false,
					async loadSettings() {
						this.loading = true;
						try {
							const result = await $bridge.call('getGeneralSettings', {
								appId: '`+appID+`'
							});
							this.settings = { ...this.settings, ...result };
						} catch (err) {
							console.error('Failed to load settings:', err);
						} finally {
							this.loading = false;
						}
					},
					async saveSettings() {
						this.saving = true;
						this.errors = {};
						try {
							await $bridge.call('updateGeneralSettings', {
								appId: '`+appID+`',
								...this.settings
							});
							$toast.success('Settings saved successfully');
						} catch (err) {
							$toast.error('Failed to save settings: ' + (err.message || err));
						} finally {
							this.saving = false;
						}
					}
				}`),
				g.Attr("x-init", "loadSettings()"),

				// Loading state
				g.El("template", g.Attr("x-if", "loading"),
					Div(Class("flex items-center justify-center py-12"),
						icons.Loader(icons.WithClass("animate-spin h-8 w-8")),
					),
				),

				// Settings form
				g.El("template", g.Attr("x-if", "!loading"),
					Div(Class("space-y-6"),
						// Application Info Card
						card.Card(
							card.Header(
								card.Title("Application Information"),
								card.Description("Basic information about your application"),
							),
							card.Content(
								Div(
									Class("space-y-4"),
									// App Name
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Application Name"), Span(Class("text-red-500 ml-1"), g.Text("*"))),
										input.Input(
											input.WithPlaceholder("My Application"),
											input.WithAttrs(g.Attr("x-model", "settings.appName")),
										),
										Div(g.Attr("x-show", "errors.appName"), Class("text-sm text-red-500 mt-1"), g.Attr("x-text", "errors.appName")),
									),
									// Description
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Description")),
										Textarea(
											Class("flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"),
											g.Attr("x-model", "settings.description"),
											Placeholder("A brief description of your application"),
											g.Attr("rows", "3"),
										),
									),
								),
							),
						),

						// Regional Settings Card
						card.Card(
							card.Header(
								card.Title("Regional Settings"),
								card.Description("Configure timezone, language, and date format preferences"),
							),
							card.Content(
								Div(
									Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
									// Timezone
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Default Timezone")),
										Select(
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model", "settings.timezone"),
											Option(Value("UTC"), g.Text("UTC")),
											OptGroup(g.Attr("label", "Americas"),
												Option(Value("America/New_York"), g.Text("Eastern Time (US)")),
												Option(Value("America/Chicago"), g.Text("Central Time (US)")),
												Option(Value("America/Denver"), g.Text("Mountain Time (US)")),
												Option(Value("America/Los_Angeles"), g.Text("Pacific Time (US)")),
											),
											OptGroup(g.Attr("label", "Europe"),
												Option(Value("Europe/London"), g.Text("London")),
												Option(Value("Europe/Paris"), g.Text("Paris")),
												Option(Value("Europe/Berlin"), g.Text("Berlin")),
											),
											OptGroup(g.Attr("label", "Asia"),
												Option(Value("Asia/Tokyo"), g.Text("Tokyo")),
												Option(Value("Asia/Singapore"), g.Text("Singapore")),
												Option(Value("Asia/Shanghai"), g.Text("Shanghai")),
											),
										),
									),
									// Language
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Default Language")),
										Select(
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model", "settings.language"),
											Option(Value("en"), g.Text("English")),
											Option(Value("es"), g.Text("Español")),
											Option(Value("fr"), g.Text("Français")),
											Option(Value("de"), g.Text("Deutsch")),
											Option(Value("zh"), g.Text("中文")),
											Option(Value("ja"), g.Text("日本語")),
										),
									),
									// Date Format
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Date Format")),
										Select(
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model", "settings.dateFormat"),
											Option(Value("YYYY-MM-DD"), g.Text("2024-01-15 (ISO)")),
											Option(Value("MM/DD/YYYY"), g.Text("01/15/2024 (US)")),
											Option(Value("DD/MM/YYYY"), g.Text("15/01/2024 (EU)")),
										),
									),
								),
							),
						),

						// Contact Info Card
						card.Card(
							card.Header(
								card.Title("Contact Information"),
								card.Description("Public contact information for your application"),
							),
							card.Content(
								Div(
									Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
									// Support Email
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Support Email")),
										input.Input(
											input.WithType("email"),
											input.WithPlaceholder("support@example.com"),
											input.WithAttrs(g.Attr("x-model", "settings.supportEmail")),
										),
									),
									// Website URL
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Website URL")),
										input.Input(
											input.WithType("url"),
											input.WithPlaceholder("https://example.com"),
											input.WithAttrs(g.Attr("x-model", "settings.websiteUrl")),
										),
									),
								),
							),
						),

						// Save Button
						Div(
							Class("flex justify-end"),
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									g.El("template", g.Attr("x-if", "saving"),
										icons.Loader(icons.WithClass("animate-spin"), icons.WithSize(16)),
									),
									Span(g.Text("Save Changes")),
								),
								button.WithAttrs(g.Attr("@click", "saveSettings()")),
								button.WithAttrs(g.Attr(":disabled", "saving")),
							),
						),
					),
				),
			),
		),
	), nil
}

// SettingsSecurityPage shows security settings
func (p *PagesManager) SettingsSecurityPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Header
			Div(
				H1(Class("text-3xl font-bold"), g.Text("Security Settings")),
				P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Configure password policies and security features")),
			),

			Div(
				g.Attr("x-data", `{
					settings: {
						passwordMinLength: 8,
						requireUppercase: true,
						requireLowercase: true,
						requireNumbers: true,
						requireSpecialChars: false,
						maxLoginAttempts: 5,
						lockoutDuration: 15,
						requireMFA: false
					},
					loading: true,
					saving: false,
					async loadSettings() {
						this.loading = true;
						try {
							const result = await $bridge.call('getSecuritySettings', {
								appId: '`+appID+`'
							});
							this.settings = { ...this.settings, ...result };
						} catch (err) {
							console.error('Failed to load settings:', err);
						} finally {
							this.loading = false;
						}
					},
					async saveSettings() {
						this.saving = true;
						try {
							await $bridge.call('updateSecuritySettings', {
								appId: '`+appID+`',
								...this.settings
							});
							$toast.success('Security settings saved successfully');
						} catch (err) {
							$toast.error('Failed to save settings: ' + (err.message || err));
						} finally {
							this.saving = false;
						}
					}
				}`),
				g.Attr("x-init", "loadSettings()"),

				// Loading state
				g.El("template", g.Attr("x-if", "loading"),
					Div(Class("flex items-center justify-center py-12"),
						icons.Loader(icons.WithClass("animate-spin h-8 w-8")),
					),
				),

				// Settings form
				g.El("template", g.Attr("x-if", "!loading"),
					Div(Class("space-y-6"),
						// Password Policy Card
						card.Card(
							card.Header(
								card.Title("Password Policy"),
								card.Description("Define password requirements for user accounts"),
							),
							card.Content(
								Div(
									Class("space-y-4"),
									// Min Length
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Minimum Password Length")),
										Input(
											Type("number"),
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model.number", "settings.passwordMinLength"),
											g.Attr("min", "6"),
											g.Attr("max", "128"),
										),
										P(Class("text-xs text-gray-500 mt-1"), g.Text("Minimum: 6, Maximum: 128")),
									),
									// Checkboxes
									Div(
										Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
										toggleField("requireUppercase", "Require Uppercase", "At least one uppercase letter (A-Z)"),
										toggleField("requireLowercase", "Require Lowercase", "At least one lowercase letter (a-z)"),
										toggleField("requireNumbers", "Require Numbers", "At least one number (0-9)"),
										toggleField("requireSpecialChars", "Require Special Characters", "At least one special character (!@#$%...)"),
									),
								),
							),
						),

						// Account Protection Card
						card.Card(
							card.Header(
								card.Title("Account Protection"),
								card.Description("Configure account lockout and multi-factor authentication"),
							),
							card.Content(
								Div(
									Class("space-y-4"),
									Div(
										Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
										// Max Login Attempts
										Div(
											Label(Class("block text-sm font-medium mb-2"), g.Text("Max Login Attempts")),
											Input(
												Type("number"),
												Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
												g.Attr("x-model.number", "settings.maxLoginAttempts"),
												g.Attr("min", "0"),
												g.Attr("max", "100"),
											),
											P(Class("text-xs text-gray-500 mt-1"), g.Text("0 = disabled")),
										),
										// Lockout Duration
										Div(
											Label(Class("block text-sm font-medium mb-2"), g.Text("Lockout Duration (minutes)")),
											Input(
												Type("number"),
												Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
												g.Attr("x-model.number", "settings.lockoutDuration"),
												g.Attr("min", "1"),
												g.Attr("max", "1440"),
												g.Attr(":disabled", "settings.maxLoginAttempts === 0"),
											),
										),
									),
									// Require MFA
									toggleField("requireMFA", "Require Multi-Factor Authentication", "Require all users to set up MFA"),
								),
							),
						),

						// Save Button
						Div(
							Class("flex justify-end"),
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									g.El("template", g.Attr("x-if", "saving"),
										icons.Loader(icons.WithClass("animate-spin"), icons.WithSize(16)),
									),
									Span(g.Text("Save Changes")),
								),
								button.WithAttrs(g.Attr("@click", "saveSettings()")),
								button.WithAttrs(g.Attr(":disabled", "saving")),
							),
						),
					),
				),
			),
		),
	), nil
}

// SettingsSessionPage shows session settings
func (p *PagesManager) SettingsSessionPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Header
			Div(
				H1(Class("text-3xl font-bold"), g.Text("Session Settings")),
				P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Configure user session behavior")),
			),

			Div(
				g.Attr("x-data", `{
					settings: {
						sessionDuration: 24,
						refreshTokenDuration: 30,
						idleTimeout: 0,
						allowMultipleSessions: true,
						maxConcurrentSessions: 0,
						rememberMeEnabled: true,
						rememberMeDuration: 30,
						cookieSameSite: 'lax',
						cookieSecure: true
					},
					loading: true,
					saving: false,
					async loadSettings() {
						this.loading = true;
						try {
							const result = await $bridge.call('getSessionSettings', {
								appId: '`+appID+`'
							});
							this.settings = { ...this.settings, ...result };
						} catch (err) {
							console.error('Failed to load settings:', err);
						} finally {
							this.loading = false;
						}
					},
					async saveSettings() {
						this.saving = true;
						try {
							await $bridge.call('updateSessionSettings', {
								appId: '`+appID+`',
								...this.settings
							});
							$toast.success('Session settings saved successfully');
						} catch (err) {
							$toast.error('Failed to save settings: ' + (err.message || err));
						} finally {
							this.saving = false;
						}
					}
				}`),
				g.Attr("x-init", "loadSettings()"),

				// Loading state
				g.El("template", g.Attr("x-if", "loading"),
					Div(Class("flex items-center justify-center py-12"),
						icons.Loader(icons.WithClass("animate-spin h-8 w-8")),
					),
				),

				// Settings form
				g.El("template", g.Attr("x-if", "!loading"),
					Div(Class("space-y-6"),
						// Session Duration Card
						card.Card(
							card.Header(
								card.Title("Session Duration"),
								card.Description("Configure how long user sessions remain active"),
							),
							card.Content(
								Div(
									Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Session Duration (hours)")),
										Input(
											Type("number"),
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model.number", "settings.sessionDuration"),
											g.Attr("min", "1"),
											g.Attr("max", "720"),
										),
										P(Class("text-xs text-gray-500 mt-1"), g.Text("How long a session remains valid after login")),
									),
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Refresh Token Duration (days)")),
										Input(
											Type("number"),
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model.number", "settings.refreshTokenDuration"),
											g.Attr("min", "1"),
											g.Attr("max", "365"),
										),
									),
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("Idle Timeout (minutes)")),
										Input(
											Type("number"),
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model.number", "settings.idleTimeout"),
											g.Attr("min", "0"),
											g.Attr("max", "1440"),
										),
										P(Class("text-xs text-gray-500 mt-1"), g.Text("0 = no idle timeout")),
									),
								),
							),
						),

						// Multiple Sessions Card
						card.Card(
							card.Header(
								card.Title("Multiple Sessions"),
								card.Description("Control how many devices can be logged in simultaneously"),
							),
							card.Content(
								Div(
									Class("space-y-4"),
									toggleField("allowMultipleSessions", "Allow Multiple Sessions", "Allow users to be logged in from multiple devices"),
									Div(
										g.Attr("x-show", "settings.allowMultipleSessions"),
										Label(Class("block text-sm font-medium mb-2"), g.Text("Max Concurrent Sessions")),
										Input(
											Type("number"),
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model.number", "settings.maxConcurrentSessions"),
											g.Attr("min", "0"),
											g.Attr("max", "100"),
										),
										P(Class("text-xs text-gray-500 mt-1"), g.Text("0 = unlimited")),
									),
								),
							),
						),

						// Remember Me Card
						card.Card(
							card.Header(
								card.Title("Remember Me"),
								card.Description("Configure persistent login functionality"),
							),
							card.Content(
								Div(
									Class("space-y-4"),
									toggleField("rememberMeEnabled", "Enable 'Remember Me'", "Allow users to stay logged in across browser sessions"),
									Div(
										g.Attr("x-show", "settings.rememberMeEnabled"),
										Label(Class("block text-sm font-medium mb-2"), g.Text("Remember Me Duration (days)")),
										Input(
											Type("number"),
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model.number", "settings.rememberMeDuration"),
											g.Attr("min", "1"),
											g.Attr("max", "365"),
										),
									),
								),
							),
						),

						// Cookie Settings Card
						card.Card(
							card.Header(
								card.Title("Cookie Settings"),
								card.Description("Configure session cookie security settings"),
							),
							card.Content(
								Div(
									Class("space-y-4"),
									Div(
										Label(Class("block text-sm font-medium mb-2"), g.Text("SameSite Policy")),
										Select(
											Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
											g.Attr("x-model", "settings.cookieSameSite"),
											Option(Value("strict"), g.Text("Strict")),
											Option(Value("lax"), g.Text("Lax")),
											Option(Value("none"), g.Text("None")),
										),
										P(Class("text-xs text-gray-500 mt-1"), g.Text("Lax is recommended for most applications")),
									),
									toggleField("cookieSecure", "Secure Cookies Only", "Only send cookies over HTTPS connections (recommended for production)"),
								),
							),
						),

						// Save Button
						Div(
							Class("flex justify-end"),
							button.Button(
								Div(
									Class("flex items-center gap-2"),
									g.El("template", g.Attr("x-if", "saving"),
										icons.Loader(icons.WithClass("animate-spin"), icons.WithSize(16)),
									),
									Span(g.Text("Save Changes")),
								),
								button.WithAttrs(g.Attr("@click", "saveSettings()")),
								button.WithAttrs(g.Attr(":disabled", "saving")),
							),
						),
					),
				),
			),
		),
	), nil
}

// SettingsAPIKeysPage shows API keys management
func (p *PagesManager) SettingsAPIKeysPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold"), g.Text("API Keys")),
					P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Manage API keys for programmatic access")),
				),
				button.Button(
					Div(
						Class("flex items-center gap-2"),
						icons.Plus(icons.WithSize(16)),
						Span(g.Text("Create API Key")),
					),
					button.WithAttrs(g.Attr("@click", "showCreateModal = true")),
				),
			),

			// API Keys list
			Div(
				g.Attr("x-data", `{
					apiKeys: [],
					loading: true,
					showCreateModal: false,
					async loadAPIKeys() {
						this.loading = true;
						try {
							const result = await $bridge.call('getAPIKeys', {
								appId: '`+appID+`'
							});
							this.apiKeys = result.apiKeys || [];
						} catch (err) {
							console.error('Failed to load API keys:', err);
						} finally {
							this.loading = false;
						}
					},
					async revokeKey(keyId) {
						if (!confirm('Are you sure you want to revoke this API key? This action cannot be undone.')) return;
						try {
							await $bridge.call('revokeAPIKey', { keyId });
							$toast.success('API key revoked successfully');
							await this.loadAPIKeys();
						} catch (err) {
							$toast.error('Failed to revoke API key');
						}
					}
				}`),
				g.Attr("x-init", "loadAPIKeys()"),

				// Loading state
				g.El("template", g.Attr("x-if", "loading"),
					Div(Class("flex items-center justify-center py-12"),
						icons.Loader(icons.WithClass("animate-spin h-8 w-8")),
					),
				),

				// Empty state
				g.El("template", g.Attr("x-if", "!loading && apiKeys.length === 0"),
					card.Card(
						card.Content(
							Div(
								Class("text-center py-12"),
								icons.Key(icons.WithClass("mx-auto h-12 w-12 text-gray-400")),
								H3(Class("mt-4 text-lg font-medium"), g.Text("No API keys")),
								P(Class("mt-2 text-gray-500"), g.Text("Create an API key to enable programmatic access to your application.")),
							),
						),
					),
				),

				// Keys list
				g.El("template", g.Attr("x-if", "!loading && apiKeys.length > 0"),
					Div(
						Class("space-y-4"),
						g.El("template", g.Attr("x-for", "key in apiKeys"),
							card.Card(
								card.Content(
									Div(
										Class("flex items-center justify-between pt-6"),
										Div(
											H4(Class("font-semibold"), g.Attr("x-text", "key.name")),
											P(Class("text-sm text-gray-600 dark:text-gray-400 mt-1"),
												Span(g.Text("Prefix: ")),
												Code(Class("bg-gray-100 dark:bg-gray-800 px-1 rounded"), g.Attr("x-text", "key.prefix + '...'")),
											),
											P(Class("text-xs text-gray-500 mt-1"), g.Attr("x-text", "`Created: ${new Date(key.createdAt).toLocaleDateString()}`")),
										),
										button.Button(
											g.Text("Revoke"),
											button.WithVariant("destructive"),
											button.WithSize("sm"),
											button.WithAttrs(g.Attr("@click", "revokeKey(key.id)")),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	), nil
}

// toggleField creates a toggle/switch field component
func toggleField(modelPath, label, description string) g.Node {
	return Div(
		Class("flex items-start space-x-3 py-2"),
		Label(
			Class("relative inline-flex items-center cursor-pointer"),
			Input(
				Type("checkbox"),
				Class("sr-only peer"),
				g.Attr("x-model", "settings."+modelPath),
			),
			Div(
				Class("w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 dark:peer-focus:ring-blue-800 rounded-full peer dark:bg-gray-700 peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-gray-600 peer-checked:bg-blue-600"),
			),
		),
		Div(
			Class("ml-3"),
			Span(Class("text-sm font-medium text-gray-900 dark:text-gray-100"), g.Text(label)),
			P(Class("text-xs text-gray-500 dark:text-gray-400"), g.Text(description)),
		),
	)
}
