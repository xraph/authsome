package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/input"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// TokensListPage renders the SCIM tokens management page.
func TokensListPage(currentApp *app.App, basePath string) g.Node {
	appID := currentApp.ID.String()

	return Div(
		Class("space-y-6"),

		// Page header
		Div(
			Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
			Div(
				H1(Class("text-2xl font-bold tracking-tight"), g.Text("SCIM Tokens")),
				P(Class("text-muted-foreground"), g.Text("Manage bearer tokens for IdP authentication")),
			),
			button.Button(
				Div(Class("flex items-center gap-2"), lucide.Plus(Class("size-4")), g.Text("Create Token")),
				button.WithVariant("default"),
				button.WithAttrs(g.Attr("@click", "showCreateModal = true")),
			),
		),

		// Alpine.js container
		Div(
			g.Attr("x-data", tokensListData(appID)),
			g.Attr("x-init", "loadTokens()"),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				Class("flex items-center justify-center py-12"),
				Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary")),
			),

			// Error state
			g.El("template",
				g.Attr("x-if", "error && !loading"),
				card.Card(
					card.Content(
						Class("flex items-center gap-3 text-destructive"),
						lucide.CircleAlert(Class("size-5")),
						Span(g.Attr("x-text", "error")),
					),
				),
			),

			// Content
			Div(
				g.Attr("x-show", "!loading && !error"),
				g.Attr("x-cloak", ""),
				Class("space-y-4"),

				// Empty state
				Div(
					g.Attr("x-show", "tokens.length === 0"),
					card.Card(
						card.Content(
							Class("text-center py-12"),
							lucide.Key(Class("size-16 mx-auto text-muted-foreground mb-4")),
							H3(Class("text-lg font-semibold mb-2"), g.Text("No tokens created")),
							P(Class("text-muted-foreground mb-4"), g.Text("Create a token to authenticate SCIM requests from your identity provider")),
							button.Button(
								Div(Class("flex items-center gap-2"), lucide.Plus(Class("size-4")), g.Text("Create Token")),
								button.WithVariant("default"),
								button.WithAttrs(g.Attr("@click", "showCreateModal = true")),
							),
						),
					),
				),

				// Tokens list
				Div(
					g.Attr("x-show", "tokens.length > 0"),
					Class("space-y-3"),
					g.El("template",
						g.Attr("x-for", "token in tokens"),
						g.Attr(":key", "token.id"),
						tokenCard(),
					),
				),
			),

			// Create token modal
			createTokenModal(appID),

			// New token display modal
			newTokenDisplayModal(),
		),
	)
}

// tokenCard renders a single token card.
func tokenCard() g.Node {
	return card.Card(
		card.Content(
			Class("flex items-center justify-between"),
			Div(
				Class("flex items-center gap-4"),
				Div(
					Class("p-3 rounded-lg"),
					g.Attr(":class", "token.status === 'active' ? 'bg-emerald-100 dark:bg-emerald-900/30' : token.status === 'revoked' ? 'bg-red-100 dark:bg-red-900/30' : 'bg-gray-100 dark:bg-gray-800'"),
					lucide.Key(
						Class("size-5"),
						g.Attr(":class", "token.status === 'active' ? 'text-emerald-600' : token.status === 'revoked' ? 'text-red-600' : 'text-gray-600'"),
					),
				),
				Div(
					Div(
						Class("flex items-center gap-2"),
						Span(Class("font-semibold"), g.Attr("x-text", "token.name")),
						Span(
							Class("inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium"),
							g.Attr(":class", "token.status === 'active' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' : token.status === 'revoked' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'"),
							g.Attr("x-text", "token.status"),
						),
					),
					Div(
						Class("text-sm text-muted-foreground"),
						Span(g.Attr("x-text", "'Prefix: ' + token.prefix + '...'")),
						Span(g.Text(" • ")),
						Span(g.Attr("x-text", "token.lastUsed ? 'Last used ' + formatRelativeTime(token.lastUsed) : 'Never used'")),
					),
					Div(
						Class("text-sm text-muted-foreground"),
						g.Attr("x-show", "token.expiresAt"),
						Span(g.Attr("x-text", "'Expires: ' + formatDate(token.expiresAt)")),
					),
				),
			),
			Div(
				Class("flex items-center gap-2"),
				button.Button(
					Div(Class("flex items-center gap-1"), lucide.RefreshCw(Class("size-4")), g.Text("Rotate")),
					button.WithVariant("outline"),
					button.WithAttrs(
						g.Attr("@click", "rotateToken(token.id)"),
						g.Attr(":disabled", "token.status !== 'active'"),
					),
				),
				button.Button(
					Div(Class("flex items-center gap-1"), lucide.Ban(Class("size-4")), g.Text("Revoke")),
					button.WithVariant("outline"),
					button.WithAttrs(
						Class("text-destructive hover:text-destructive"),
						g.Attr("@click", "revokeToken(token.id)"),
						g.Attr(":disabled", "token.status !== 'active'"),
					),
				),
			),
		),
	)
}

// createTokenModal renders the create token modal.
func createTokenModal(appID string) g.Node {
	return g.El("template",
		g.Attr("x-if", "showCreateModal"),
		Div(
			Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
			g.Attr("@click.self", "showCreateModal = false"),
			card.Card(
				Class("w-full max-w-md mx-4"),
				card.Header(
					card.Title("Create SCIM Token"),
					card.Description("Generate a new bearer token for IdP authentication"),
				),
				card.Content(
					Class("space-y-4"),
					Div(
						Label(Class("text-sm font-medium block mb-2"), g.Text("Token Name")),
						input.Input(
							input.WithType("text"),
							input.WithPlaceholder("e.g., Okta Production"),
							input.WithAttrs(g.Attr("x-model", "newToken.name")),
						),
					),
					Div(
						Label(Class("text-sm font-medium block mb-2"), g.Text("Description (optional)")),
						input.Input(
							input.WithType("text"),
							input.WithPlaceholder("Describe the token's purpose"),
							input.WithAttrs(g.Attr("x-model", "newToken.description")),
						),
					),
					Div(
						Label(Class("text-sm font-medium block mb-2"), g.Text("Expiration")),
						Select(
							Class("w-full px-3 py-2 border rounded-md bg-background"),
							g.Attr("x-model", "newToken.expiresIn"),
							Option(Value("0"), g.Text("Never expires")),
							Option(Value("30"), g.Text("30 days")),
							Option(Value("90"), g.Text("90 days")),
							Option(Value("180"), g.Text("180 days")),
							Option(Value("365"), g.Text("1 year")),
						),
					),
					Div(
						Label(Class("text-sm font-medium block mb-2"), g.Text("Scopes")),
						Div(
							Class("space-y-2"),
							Label(
								Class("flex items-center gap-2 cursor-pointer"),
								Input(
									Type("checkbox"),
									Value("users:read"),
									g.Attr("x-model", "newToken.scopes"),
									Class("rounded"),
								),
								Span(Class("text-sm"), g.Text("Users: Read")),
							),
							Label(
								Class("flex items-center gap-2 cursor-pointer"),
								Input(
									Type("checkbox"),
									Value("users:write"),
									g.Attr("x-model", "newToken.scopes"),
									Class("rounded"),
								),
								Span(Class("text-sm"), g.Text("Users: Write")),
							),
							Label(
								Class("flex items-center gap-2 cursor-pointer"),
								Input(
									Type("checkbox"),
									Value("groups:read"),
									g.Attr("x-model", "newToken.scopes"),
									Class("rounded"),
								),
								Span(Class("text-sm"), g.Text("Groups: Read")),
							),
							Label(
								Class("flex items-center gap-2 cursor-pointer"),
								Input(
									Type("checkbox"),
									Value("groups:write"),
									g.Attr("x-model", "newToken.scopes"),
									Class("rounded"),
								),
								Span(Class("text-sm"), g.Text("Groups: Write")),
							),
						),
					),
				),
				card.Footer(
					Class("flex items-center justify-between"),
					button.Button(
						g.Text("Cancel"),
						button.WithVariant("outline"),
						button.WithAttrs(g.Attr("@click", "showCreateModal = false")),
					),
					button.Button(
						Div(
							Class("flex items-center gap-2"),
							lucide.Plus(Class("size-4")),
							Span(g.Attr("x-text", "creating ? 'Creating...' : 'Create Token'")),
						),
						button.WithVariant("default"),
						button.WithAttrs(
							g.Attr("@click", "createToken()"),
							g.Attr(":disabled", "creating"),
						),
					),
				),
			),
		),
	)
}

// newTokenDisplayModal renders the modal showing the newly created token.
func newTokenDisplayModal() g.Node {
	return g.El("template",
		g.Attr("x-if", "createdTokenPlainText"),
		Div(
			Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
			card.Card(
				Class("w-full max-w-lg mx-4"),
				card.Header(
					Div(
						Class("flex items-center gap-2 text-emerald-600"),
						lucide.CircleCheck(Class("size-5")),
						Span(Class("font-semibold"), g.Text("Token Created Successfully")),
					),
					card.Description("Copy this token now. It will only be shown once."),
				),
				card.Content(
					Class("space-y-4"),
					Div(
						Class("relative"),
						Div(
							Class("p-4 bg-muted rounded-lg font-mono text-sm break-all pr-12"),
							g.Attr("x-text", "createdTokenPlainText"),
						),
						button.Button(
							lucide.Copy(Class("size-4")),
							button.WithVariant("ghost"),
							button.WithSize("icon"),
							button.WithAttrs(
								Class("absolute top-2 right-2"),
								g.Attr("@click", "copyToken()"),
							),
						),
					),
					Div(
						Class("flex items-center gap-2 p-3 bg-amber-100 dark:bg-amber-900/30 rounded-lg text-sm text-amber-800 dark:text-amber-200"),
						lucide.TriangleAlert(Class("size-4 flex-shrink-0")),
						P(g.Text("Store this token securely. You won't be able to see it again.")),
					),
				),
				card.Footer(
					button.Button(
						g.Text("Done"),
						button.WithVariant("default"),
						button.WithAttrs(
							Class("w-full"),
							g.Attr("@click", "createdTokenPlainText = null; loadTokens()"),
						),
					),
				),
			),
		),
	)
}

// tokensListData returns the Alpine.js data for the tokens list.
func tokensListData(appID string) string {
	return fmt.Sprintf(`{
		tokens: [],
		loading: true,
		error: null,
		showCreateModal: false,
		creating: false,
		createdTokenPlainText: null,
		newToken: {
			name: '',
			description: '',
			expiresIn: '90',
			scopes: ['users:read', 'users:write', 'groups:read', 'groups:write']
		},
		
		async loadTokens() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('scim.getTokens', {
					appId: '%s',
					page: 1,
					pageSize: 100
				});
				this.tokens = result.tokens || [];
			} catch (err) {
				console.error('Failed to load tokens:', err);
				this.error = err.message || 'Failed to load tokens';
			} finally {
				this.loading = false;
			}
		},
		
		async createToken() {
			if (!this.newToken.name) {
				alert('Please enter a token name');
				return;
			}
			
			this.creating = true;
			try {
				const result = await $bridge.call('scim.createToken', {
					appId: '%s',
					name: this.newToken.name,
					description: this.newToken.description,
					expiresIn: parseInt(this.newToken.expiresIn) || 0,
					scopes: this.newToken.scopes
				});
				this.showCreateModal = false;
				this.createdTokenPlainText = result.plainText;
				this.newToken = {
					name: '',
					description: '',
					expiresIn: '90',
					scopes: ['users:read', 'users:write', 'groups:read', 'groups:write']
				};
			} catch (err) {
				alert('Failed to create token: ' + (err.message || 'Unknown error'));
			} finally {
				this.creating = false;
			}
		},
		
		async rotateToken(tokenId) {
			if (!confirm('This will invalidate the current token and generate a new one. Continue?')) {
				return;
			}
			
			try {
				const result = await $bridge.call('scim.rotateToken', {
					appId: '%s',
					tokenId: tokenId
				});
				this.createdTokenPlainText = result.plainText;
			} catch (err) {
				alert('Failed to rotate token: ' + (err.message || 'Unknown error'));
			}
		},
		
		async revokeToken(tokenId) {
			if (!confirm('This will permanently revoke this token. This action cannot be undone. Continue?')) {
				return;
			}
			
			try {
				await $bridge.call('scim.revokeToken', {
					appId: '%s',
					tokenId: tokenId
				});
				await this.loadTokens();
			} catch (err) {
				alert('Failed to revoke token: ' + (err.message || 'Unknown error'));
			}
		},
		
		copyToken() {
			navigator.clipboard.writeText(this.createdTokenPlainText);
			alert('Token copied to clipboard');
		},
		
		formatRelativeTime(timestamp) {
			if (!timestamp) return 'Never';
			const date = new Date(timestamp);
			const now = new Date();
			const diff = Math.floor((now - date) / 1000);
			
			if (diff < 60) return 'just now';
			if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
			if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
			return Math.floor(diff / 86400) + 'd ago';
		},
		
		formatDate(timestamp) {
			if (!timestamp) return '';
			return new Date(timestamp).toLocaleDateString();
		}
	}`, appID, appID, appID, appID)
}
