package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/checkbox"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// SettingsPage shows OIDC provider settings.
func (p *PagesManager) SettingsPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				settings: null,
				loading: true,
				error: null,
				saving: false,
				
				async loadSettings() {
					this.loading = true;
					this.error = null;
					try {
						const result = await $bridge.call('oidcprovider.getSettings', {
							appId: '`+appID+`'
						});
						this.settings = result.data;
					} catch (err) {
						this.error = err.message || 'Failed to load settings';
					} finally {
						this.loading = false;
					}
				},
				
				async saveTokenSettings() {
					this.saving = true;
					try {
						await $bridge.call('oidcprovider.updateTokenSettings', {
							accessTokenExpiry: this.settings.tokenSettings.accessTokenExpiry,
							idTokenExpiry: this.settings.tokenSettings.idTokenExpiry,
							refreshTokenExpiry: this.settings.tokenSettings.refreshTokenExpiry
						});
						alert('Token settings updated successfully');
					} catch (err) {
						alert('Failed to save: ' + err.message);
					} finally {
						this.saving = false;
					}
				},
				
				async saveDeviceFlowSettings() {
					this.saving = true;
					try {
						await $bridge.call('oidcprovider.updateDeviceFlowSettings', {
							enabled: this.settings.deviceFlow.enabled,
							codeExpiry: this.settings.deviceFlow.codeExpiry,
							userCodeLength: parseInt(this.settings.deviceFlow.userCodeLength),
							userCodeFormat: this.settings.deviceFlow.userCodeFormat,
							pollingInterval: parseInt(this.settings.deviceFlow.pollingInterval),
							verificationUri: this.settings.deviceFlow.verificationUri,
							maxPollAttempts: parseInt(this.settings.deviceFlow.maxPollAttempts),
							cleanupInterval: this.settings.deviceFlow.cleanupInterval
						});
						alert('Device flow settings updated successfully');
					} catch (err) {
						alert('Failed to save: ' + err.message);
					} finally {
						this.saving = false;
					}
				},
				
				async rotateKeys() {
					if (!confirm('Rotate JWT signing keys? This will generate new keys.')) return;
					try {
						const result = await $bridge.call('oidcprovider.rotateKeys', {});
						alert('Keys rotated successfully. New key ID: ' + result.newKeyId);
						await this.loadSettings();
					} catch (err) {
						alert('Key rotation failed: ' + err.message);
					}
				}
			}`),
			g.Attr("x-init", "loadSettings()"),

			// Header
			H1(Class("text-3xl font-bold"), g.Text("OIDC Provider Settings")),
			P(Class("text-gray-600 dark:text-gray-400"), g.Text("Configure OAuth2 and OpenID Connect settings")),

			g.El("template", g.Attr("x-if", "loading"),
				Div(Class("flex justify-center py-12"),
					Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary")),
				),
			),

			g.El("template", g.Attr("x-if", "!loading && settings"),
				Div(Class("space-y-2"),
					// General Settings
					card.Card(
						card.Header(card.Title("General")),
						card.Content(
							Div(Class("space-y-4"),
								Div(
									Label(For("issuer"), Class("text-sm font-medium"), g.Text("Issuer URL")),
									input.Input(
										input.WithType("text"),
										input.WithID("issuer"),
										input.WithAttrs(
											g.Attr("x-model", "settings.issuer"),
											g.Attr("disabled", ""),
										),
										input.WithClass("mt-1 bg-gray-50 dark:bg-gray-900"),
									),
									P(Class("text-xs text-muted-foreground mt-1"), g.Text("Read-only: Configured in server settings")),
								),
								Div(
									Label(For("discoveryUrl"), Class("text-sm font-medium"), g.Text("Discovery URL")),
									input.Input(
										input.WithType("text"),
										input.WithID("discoveryUrl"),
										input.WithAttrs(
											g.Attr("x-model", "settings.discoveryUrl"),
											g.Attr("disabled", ""),
										),
										input.WithClass("mt-1 bg-gray-50 dark:bg-gray-900"),
									),
								),
							),
						),
					),

					// Token Settings
					card.Card(
						card.Header(card.Title("Token Lifetimes")),
						card.Content(
							Div(Class("space-y-4"),
								Div(
									Label(For("accessTokenExpiry"), Class("text-sm font-medium"), g.Text("Access Token Expiry")),
									input.Input(
										input.WithType("text"),
										input.WithID("accessTokenExpiry"),
										input.WithPlaceholder("1h"),
										input.WithAttrs(
											g.Attr("x-model", "settings.tokenSettings.accessTokenExpiry"),
										),
										input.WithClass("mt-1"),
									),
									P(Class("text-xs text-muted-foreground mt-1"), g.Text("Duration (e.g., 1h, 30m)")),
								),
								Div(
									Label(For("idTokenExpiry"), Class("text-sm font-medium"), g.Text("ID Token Expiry")),
									input.Input(
										input.WithType("text"),
										input.WithID("idTokenExpiry"),
										input.WithPlaceholder("1h"),
										input.WithAttrs(
											g.Attr("x-model", "settings.tokenSettings.idTokenExpiry"),
										),
										input.WithClass("mt-1"),
									),
								),
								Div(
									Label(For("refreshTokenExpiry"), Class("text-sm font-medium"), g.Text("Refresh Token Expiry")),
									input.Input(
										input.WithType("text"),
										input.WithID("refreshTokenExpiry"),
										input.WithPlaceholder("720h"),
										input.WithAttrs(
											g.Attr("x-model", "settings.tokenSettings.refreshTokenExpiry"),
										),
										input.WithClass("mt-1"),
									),
								),
								button.Button(
									g.Text("Save Token Settings"),
									button.WithAttrs(g.Attr("@click", "saveTokenSettings()"), g.Attr("x-bind:disabled", "saving")),
								),
							),
						),
					),

					// Key Settings
					card.Card(
						card.Header(card.Title("JWT Keys")),
						card.Content(
							Div(Class("space-y-4"),
								Div(
									Label(For("currentKeyId"), Class("text-sm font-medium"), g.Text("Current Key ID")),
									input.Input(
										input.WithType("text"),
										input.WithID("currentKeyId"),
										input.WithAttrs(
											g.Attr("x-model", "settings.keySettings.currentKeyId"),
											g.Attr("disabled", ""),
										),
										input.WithClass("mt-1 font-mono bg-gray-50 dark:bg-gray-900"),
									),
								),
								Div(
									Label(For("lastRotation"), Class("text-sm font-medium"), g.Text("Last Rotation")),
									input.Input(
										input.WithType("text"),
										input.WithID("lastRotation"),
										input.WithAttrs(
											g.Attr("x-model", "settings.keySettings.lastRotation"),
											g.Attr("disabled", ""),
										),
										input.WithClass("mt-1 bg-gray-50 dark:bg-gray-900"),
									),
								),
								button.Button(
									Div(Class("flex items-center gap-2"),
										icons.Key(icons.WithSize(16)),
										g.Text("Rotate Keys Now"),
									),
									button.WithVariant("outline"),
									button.WithAttrs(g.Attr("@click", "rotateKeys()")),
								),
							),
						),
					),

					// Device Flow Settings
					card.Card(
						card.Header(card.Title("Device Flow (RFC 8628)")),
						card.Content(
							Div(Class("space-y-4"),
								Div(Class("flex items-center gap-2"),
									checkbox.Checkbox(
										checkbox.WithID("enableDeviceFlow"),
										checkbox.WithAttrs(g.Attr("x-model", "settings.deviceFlow.enabled")),
									),
									Label(For("enableDeviceFlow"), Class("text-sm font-medium"), g.Text("Enable Device Flow")),
								),
								Div(
									Label(For("codeExpiry"), Class("text-sm font-medium"), g.Text("Code Expiry")),
									input.Input(
										input.WithType("text"),
										input.WithID("codeExpiry"),
										input.WithPlaceholder("10m"),
										input.WithAttrs(
											g.Attr("x-model", "settings.deviceFlow.codeExpiry"),
										),
										input.WithClass("mt-1"),
									),
								),
								Div(
									Label(For("userCodeLength"), Class("text-sm font-medium"), g.Text("User Code Length")),
									input.Input(
										input.WithType("number"),
										input.WithID("userCodeLength"),
										input.WithAttrs(
											g.Attr("x-model", "settings.deviceFlow.userCodeLength"),
										),
										input.WithClass("mt-1"),
									),
								),
								Div(
									Label(For("pollingInterval"), Class("text-sm font-medium"), g.Text("Polling Interval (seconds)")),
									input.Input(
										input.WithType("number"),
										input.WithID("pollingInterval"),
										input.WithAttrs(
											g.Attr("x-model", "settings.deviceFlow.pollingInterval"),
										),
										input.WithClass("mt-1"),
									),
								),
								button.Button(
									g.Text("Save Device Flow Settings"),
									button.WithAttrs(g.Attr("@click", "saveDeviceFlowSettings()"), g.Attr("x-bind:disabled", "saving")),
								),
							),
						),
					),
				),
			),
		),
	), nil
}
