package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/primitives"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ProvidersPage renders the providers configuration page.
func ProvidersPage(currentApp *app.App, basePath string) g.Node {
	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Page header
			PageHeader(
				"Notification Providers",
				"Configure email and SMS delivery providers",
			),

			// Main content with Alpine.js
			Div(
				g.Attr("x-data", providersData(currentApp.ID.String())),
				g.Attr("x-init", "await loadProviders()"),
				Class("space-y-6"),

				// Loading state
				Div(
					g.Attr("x-show", "loading"),
					LoadingSpinner(),
				),

				// Error message
				ErrorMessage("error && !loading"),

				// Success message
				SuccessMessage("successMessage && !loading"),

				// Providers configuration
				Div(
					g.Attr("x-show", "!loading && !error && providers"),
					Class("grid gap-6 lg:grid-cols-2"),

					// Email Provider Card
					card.Card(
						card.Header(
							card.Title("Email Provider"),
							card.Description("Configure email delivery settings"),
						),
						card.Content(
							Class("space-y-4"),

							// Provider type
							Div(
								Label(Class("text-sm font-medium"), g.Text("Provider Type")),
								Select(
									g.Attr("x-model", "providers.emailProvider.type"),
									Class("mt-1.5 w-full rounded-md border border-input bg-background px-3 py-2"),
									Option(Value("smtp"), g.Text("SMTP")),
									Option(Value("sendgrid"), g.Text("SendGrid")),
									Option(Value("postmark"), g.Text("Postmark")),
									Option(Value("mailersend"), g.Text("MailerSend")),
									Option(Value("resend"), g.Text("Resend")),
								),
							),

							// From name
							Div(
								Label(Class("text-sm font-medium"), g.Text("From Name")),
								Input(
									Type("text"),
									Placeholder("Your App Name"),
									g.Attr("x-model", "providers.emailProvider.fromName"),
									Class("mt-1.5 w-full rounded-md border border-input bg-background px-3 py-2"),
								),
							),

							// From email
							Div(
								Label(Class("text-sm font-medium"), g.Text("From Email")),
								Input(
									Type("email"),
									Placeholder("noreply@example.com"),
									g.Attr("x-model", "providers.emailProvider.fromEmail"),
									Class("mt-1.5 w-full rounded-md border border-input bg-background px-3 py-2"),
								),
							),

							// Test button
							Button(
								Type("button"),
								g.Attr("@click", "await testProvider('email')"),
								g.Attr(":disabled", "testing"),
								Class("inline-flex items-center gap-2 rounded-md border border-input bg-background px-4 py-2 text-sm hover:bg-accent hover:text-accent-foreground disabled:opacity-50"),
								lucide.Send(Class("h-4 w-4")),
								Span(g.Attr("x-text", "testing ? 'Sending...' : 'Test Email'")),
							),
						),
					),

					// SMS Provider Card
					card.Card(
						card.Header(
							card.Title("SMS Provider"),
							card.Description("Configure SMS delivery settings"),
						),
						card.Content(
							Class("space-y-4"),

							// Provider type
							Div(
								Label(Class("text-sm font-medium"), g.Text("Provider Type")),
								Select(
									g.Attr("x-model", "providers.smsProvider.type"),
									Class("mt-1.5 w-full rounded-md border border-input bg-background px-3 py-2"),
									Option(Value(""), g.Text("Not configured")),
									Option(Value("twilio"), g.Text("Twilio")),
									Option(Value("vonage"), g.Text("Vonage")),
									Option(Value("aws-sns"), g.Text("AWS SNS")),
								),
							),

							// Enabled toggle
							Div(
								Class("flex items-center gap-2"),
								Input(
									Type("checkbox"),
									g.Attr("x-model", "providers.smsProvider.enabled"),
									Class("h-4 w-4 rounded border-gray-300"),
									ID("sms-enabled"),
								),
								Label(For("sms-enabled"), Class("text-sm font-medium"), g.Text("Enable SMS notifications")),
							),

							// Test button
							Button(
								Type("button"),
								g.Attr("@click", "await testProvider('sms')"),
								g.Attr(":disabled", "testing || !providers.smsProvider.enabled"),
								Class("inline-flex items-center gap-2 rounded-md border border-input bg-background px-4 py-2 text-sm hover:bg-accent hover:text-accent-foreground disabled:opacity-50"),
								lucide.Send(Class("h-4 w-4")),
								Span(g.Attr("x-text", "testing ? 'Sending...' : 'Test SMS'")),
							),
						),
					),
				),

				// Save button
				Div(
					g.Attr("x-show", "!loading && !error"),
					Class("flex justify-end"),
					Button(
						Type("button"),
						g.Attr("@click", "await saveProviders()"),
						g.Attr(":disabled", "saving"),
						Class("inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"),
						lucide.Save(Class("h-4 w-4")),
						Span(g.Attr("x-text", "saving ? 'Saving...' : 'Save Changes'")),
					),
				),
			),
		),
	)
}

func providersData(appID string) string {
	return `{
		providers: {
			emailProvider: {
				type: '',
				enabled: false,
				fromName: '',
				fromEmail: '',
				config: {}
			},
			smsProvider: {
				type: '',
				enabled: false,
				config: {}
			}
		},
		loading: true,
		saving: false,
		testing: false,
		error: null,
		successMessage: null,

		async loadProviders() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('notification.getProviders', {});
				if (result && result.providers) {
					this.providers = result.providers;
				} else {
					this.error = 'Invalid response format';
				}
			} catch (err) {
				console.error('Failed to load providers:', err);
				this.error = err.message || 'Failed to load providers';
			} finally {
				this.loading = false;
			}
		},

		async saveProviders() {
			this.saving = true;
			this.error = null;
			this.successMessage = null;
			try {
				const result = await $bridge.call('notification.updateProviders', {
					emailProvider: this.providers.emailProvider,
					smsProvider: this.providers.smsProvider
				});
				if (result && result.providers) {
					this.providers = result.providers;
				this.successMessage = result.message || 'Providers updated successfully';
					setTimeout(() => this.successMessage = null, 3000);
				} else {
					this.error = 'Invalid response format';
				}
			} catch (err) {
				console.error('Failed to save providers:', err);
				this.error = err.message || 'Failed to save providers';
			} finally {
				this.saving = false;
			}
		},

		async testProvider(type) {
			const recipient = prompt('Enter ' + (type === 'email' ? 'email address' : 'phone number') + ' to send test to:');
			if (!recipient) return;

			this.testing = true;
			this.error = null;
			try {
				const result = await $bridge.call('notification.testProvider', {
					providerType: type,
					recipient: recipient
				});
				alert(result.message || 'Test notification sent successfully!');
			} catch (err) {
				console.error('Test failed:', err);
				alert('Test failed: ' + (err.message || 'Unknown error'));
			} finally {
				this.testing = false;
			}
		}
	}`
}
