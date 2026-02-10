package pages

import (
	"fmt"

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/checkbox"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/components/modal"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// ClientCreatePage shows the create client wizard.
func (p *PagesManager) ClientCreatePage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-4 max-w-3xl"),
			g.Attr("x-data", `{
				step: 1,
				form: {
					clientName: '',
					applicationType: 'web',
					logoUri: '',
					redirectUris: [''],
					postLogoutRedirectUris: [''],
					grantTypes: ['authorization_code'],
					responseTypes: ['code'],
					allowedScopes: ['openid', 'profile', 'email'],
					tokenEndpointAuthMethod: 'client_secret_basic',
					requirePkce: false,
					requireConsent: true,
					trustedClient: false,
					policyUri: '',
					tosUri: '',
					contacts: ['']
				},
				loading: false,
				error: null,
				showSuccessModal: false,
				createdClient: null,
				
				addRedirectUri() {
					this.form.redirectUris.push('');
				},
				removeRedirectUri(index) {
					this.form.redirectUris.splice(index, 1);
				},
				
				copyToClipboard(text, type) {
					navigator.clipboard.writeText(text).then(() => {
						// Could add a toast notification here
						console.log(type + ' copied to clipboard');
					});
				},
				
				async createClient() {
					this.loading = true;
					this.error = null;
					try {
						const result = await $bridge.call('oidcprovider.createClient', {
							appId: '`+appID+`',
							...this.form,
							redirectUris: this.form.redirectUris.filter(uri => uri.trim() !== ''),
							postLogoutRedirectUris: this.form.postLogoutRedirectUris.filter(uri => uri.trim() !== ''),
							contacts: this.form.contacts.filter(c => c.trim() !== '')
						});
						
						this.createdClient = result.data;
						this.showSuccessModal = true;
					} catch (err) {
						this.error = err.message || 'Failed to create client';
					} finally {
						this.loading = false;
					}
				},
				
				navigateToClient() {
					window.location.href = '`+p.baseUIPath+`/app/`+appID+`/oauth/clients/' + this.createdClient.clientId;
				}
			}`),

			// Back link
			A(
				Href(p.baseUIPath+"/app/"+appID+"/oauth/clients"),
				Class("inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"),
				icons.ArrowLeft(icons.WithSize(16)),
				g.Text("Back to Clients"),
			),

			// Header
			Div(
				Class("pb-3 border-b"),
				H1(Class("text-2xl font-semibold tracking-tight"), g.Text("Create OAuth Client")),
				P(Class("text-sm text-muted-foreground mt-1"), g.Text("Register a new OAuth2/OIDC client application")),
			),

			// Error message
			g.El("template", g.Attr("x-if", "error"),
				Div(
					Class("flex items-start gap-3 p-4 rounded-lg border border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/50"),
					icons.AlertCircle(icons.WithSize(20), icons.WithClass("text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0")),
					Div(
						Class("flex-1"),
						P(Class("text-sm font-medium text-red-900 dark:text-red-200"), g.Text("Error")),
						P(Class("text-sm text-red-700 dark:text-red-300 mt-1"), g.Attr("x-text", "error")),
					),
				),
			),

			// Form
			card.Card(
				card.Content(
					Div(Class("space-y-2"),
						// Basic Information Section
						Div(
							Div(Class("flex items-center gap-2 mb-4"),
								icons.Info(icons.WithSize(16), icons.WithClass("text-muted-foreground")),
								H3(Class("text-sm font-semibold"), g.Text("Basic Information")),
							),
							Div(Class("space-y-4 pl-6"),
								Div(
									Label(For("clientName"), Class("text-sm font-medium text-foreground"), g.Text("Client Name *")),
									input.Input(
										input.WithType("text"),
										input.WithName("clientName"),
										input.WithAttrs(
											g.Attr("id", "clientName"),
											g.Attr("x-model", "form.clientName"),
											g.Attr("required", ""),
										),
										input.WithPlaceholder("My OAuth Client"),
										input.WithClass("mt-1.5"),
									),
									P(Class("text-xs text-muted-foreground mt-1.5"), g.Text("A descriptive name for your application")),
								),
								Div(
									Label(For("applicationType"), Class("text-sm font-medium text-foreground"), g.Text("Application Type *")),
									Select(
										ID("applicationType"),
										Class("mt-1.5 flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"),
										g.Attr("x-model", "form.applicationType"),
										Option(Value("web"), g.Text("Web Application")),
										Option(Value("spa"), g.Text("Single Page App")),
										Option(Value("native"), g.Text("Native/Mobile App")),
									),
									P(Class("text-xs text-muted-foreground mt-1.5"), g.Text("Choose the type that matches your application")),
								),
							),
						),

						// Divider
						Hr(Class("border-t")),

						// OAuth Configuration Section
						Div(
							Div(Class("flex items-center gap-2 mb-4"),
								icons.Link(icons.WithSize(16), icons.WithClass("text-muted-foreground")),
								H3(Class("text-sm font-semibold"), g.Text("OAuth Configuration")),
							),
							Div(Class("space-y-4 pl-6"),
								Div(
									Label(For("redirectUris"), Class("text-sm font-medium text-foreground"), g.Text("Redirect URIs *")),
									P(Class("text-xs text-muted-foreground mt-1.5 mb-3"), g.Text("Allowed callback URLs where users will be redirected after authentication")),
									Div(Class("space-y-2"),
										g.El("template", g.Attr("x-for", "(uri, index) in form.redirectUris"),
											Div(Class("flex gap-2 space-y-1"),
												input.Input(
													input.WithType("url"),
													input.WithAttrs(
														g.Attr("x-model", "form.redirectUris[index]"),
														g.Attr("id", "redirectUris"),
													),
													input.WithPlaceholder("https://example.com/callback"),
												),
												button.Button(
													icons.Trash2(icons.WithSize(16)),
													button.WithVariant("ghost"),
													button.WithSize("sm"),
													button.WithAttrs(
														g.Attr("@click", "removeRedirectUri(index)"),
														g.Attr("x-show", "form.redirectUris.length > 1"),
													),
												),
											),
										),
									),
									button.Button(
										Div(Class("flex items-center gap-1.5"),
											icons.Plus(icons.WithSize(14)),
											g.Text("Add Another URI"),
										),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(g.Attr("@click", "addRedirectUri()"), g.Attr("class", "mt-3")),
									),
								),

								Div(
									Label(Class("text-sm font-medium text-foreground mb-3 block"), g.Text("Grant Types")),
									Div(Class("space-y-2.5"),
										Div(Class("flex items-start gap-2.5"),
											checkbox.Checkbox(
												checkbox.WithID("grantAuthCode"),
												checkbox.WithValue("authorization_code"),
												checkbox.WithAttrs(g.Attr("x-model", "form.grantTypes"), g.Attr("class", "mt-0.5")),
											),
											Label(For("grantAuthCode"), Class("text-sm leading-relaxed cursor-pointer"), g.Text("Authorization Code")),
										),
										Div(Class("flex items-start gap-2.5"),
											checkbox.Checkbox(
												checkbox.WithID("grantRefreshToken"),
												checkbox.WithValue("refresh_token"),
												checkbox.WithAttrs(g.Attr("x-model", "form.grantTypes"), g.Attr("class", "mt-0.5")),
											),
											Label(For("grantRefreshToken"), Class("text-sm leading-relaxed cursor-pointer"), g.Text("Refresh Token")),
										),
									),
								),
							),
						),

						// Divider
						Hr(Class("border-t")),

						// Security Options Section
						Div(
							Div(Class("flex items-center gap-2 mb-4"),
								icons.Shield(icons.WithSize(16), icons.WithClass("text-muted-foreground")),
								H3(Class("text-sm font-semibold"), g.Text("Security Options")),
							),
							Div(Class("space-y-3 pl-6"),
								Div(Class("flex items-start gap-2.5 p-3 rounded-lg border bg-muted/30"),
									checkbox.Checkbox(
										checkbox.WithID("requirePkce"),
										checkbox.WithAttrs(g.Attr("x-model", "form.requirePkce"), g.Attr("class", "mt-0.5")),
									),
									Div(Class("flex-1"),
										Label(For("requirePkce"), Class("text-sm font-medium cursor-pointer"), g.Text("Require PKCE")),
										P(Class("text-xs text-muted-foreground mt-1"), g.Text("Recommended for Single Page Apps and Native/Mobile Apps")),
									),
								),
								Div(Class("flex items-start gap-2.5 p-3 rounded-lg border bg-muted/30"),
									checkbox.Checkbox(
										checkbox.WithID("requireConsent"),
										checkbox.WithAttrs(g.Attr("x-model", "form.requireConsent"), g.Attr("class", "mt-0.5")),
									),
									Div(Class("flex-1"),
										Label(For("requireConsent"), Class("text-sm font-medium cursor-pointer"), g.Text("Require User Consent")),
										P(Class("text-xs text-muted-foreground mt-1"), g.Text("Ask users to approve access to their data")),
									),
								),
								Div(Class("flex items-start gap-2.5 p-3 rounded-lg border bg-muted/30"),
									checkbox.Checkbox(
										checkbox.WithID("trustedClient"),
										checkbox.WithAttrs(g.Attr("x-model", "form.trustedClient"), g.Attr("class", "mt-0.5")),
									),
									Div(Class("flex-1"),
										Label(For("trustedClient"), Class("text-sm font-medium cursor-pointer"), g.Text("Trusted Client")),
										P(Class("text-xs text-muted-foreground mt-1"), g.Text("Skip consent screen for first-party applications")),
									),
								),
							),
						),

						// Divider
						Hr(Class("border-t")),

						// Actions
						Div(Class("flex items-center justify-between pt-2"),
							button.Button(
								g.Text("Cancel"),
								button.WithVariant("ghost"),
								button.WithAttrs(
									g.Attr("@click", fmt.Sprintf("window.location.href = '%s/app/%s/oauth/clients'", p.baseUIPath, appID)),
								),
							),
							button.Button(
								g.El("span", g.Attr("x-text", "loading ? 'Creating...' : 'Create Client'")),
								button.WithAttrs(
									g.Attr("@click", "createClient()"),
									g.Attr("x-bind:disabled", "loading || !form.clientName"),
								),
							),
						),
					),
				),
			),

			// Success Modal
			g.El("template", g.Attr("x-if", "showSuccessModal && createdClient"),
				Div(
					Class("fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50"),
					g.Attr("@click.self", "showSuccessModal = false"),
					modal.Dialog(
						modal.DialogContent(
							modal.DialogHeader(
								modal.DialogTitle("OAuth Client Created Successfully"),
								modal.DialogDescription("Save your client credentials now - the secret will not be shown again"),
							),
							modal.DialogBody(
								Div(Class("space-y-4"),
									// Success message with icon
									Div(
										Class("flex items-start gap-3 p-3 rounded-lg bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800"),
										icons.CheckCircle(icons.WithSize(20), icons.WithClass("text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5")),
										P(Class("text-sm text-green-700 dark:text-green-300"), g.Text("Your OAuth client has been created successfully!")),
									),

									// Client ID
									Div(
										Label(Class("text-sm font-medium text-foreground mb-2 block"), g.Text("Client ID")),
										Div(
											Class("flex gap-2"),
											Div(
												Class("flex-1 font-mono text-xs p-3 bg-muted rounded-lg border break-all select-all"),
												g.Attr("x-text", "createdClient.clientId"),
											),
											button.Button(
												icons.Copy(icons.WithSize(16)),
												button.WithVariant("outline"),
												button.WithSize("sm"),
												button.WithAttrs(
													g.Attr("@click", "copyToClipboard(createdClient.clientId, 'Client ID')"),
													g.Attr("title", "Copy Client ID"),
												),
											),
										),
									),

									// Client Secret
									Div(
										Label(Class("text-sm font-medium text-foreground mb-2 block"), g.Text("Client Secret")),
										Div(
											Class("flex gap-2"),
											Div(
												Class("flex-1 font-mono text-xs p-3 bg-muted rounded-lg border break-all select-all"),
												g.Attr("x-text", "createdClient.clientSecret"),
											),
											button.Button(
												icons.Copy(icons.WithSize(16)),
												button.WithVariant("outline"),
												button.WithSize("sm"),
												button.WithAttrs(
													g.Attr("@click", "copyToClipboard(createdClient.clientSecret, 'Client Secret')"),
													g.Attr("title", "Copy Client Secret"),
												),
											),
										),
									),

									// Warning message
									Div(
										Class("flex items-start gap-3 p-3 rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800"),
										icons.AlertCircle(icons.WithSize(20), icons.WithClass("text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5")),
										Div(
											P(Class("text-sm font-medium text-amber-900 dark:text-amber-100"), g.Text("Important: Save these credentials now")),
											P(Class("text-xs text-amber-700 dark:text-amber-300 mt-1"), g.Text("The client secret will not be displayed again. Store it securely.")),
										),
									),
								),
							),
							modal.DialogFooter(
								button.Button(
									g.Text("View Client Details"),
									button.WithAttrs(
										g.Attr("@click", "navigateToClient()"),
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
