package apikey

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/apikey"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/core/user"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// renderAPIKeysListContent renders the main API keys management page content.
func (e *DashboardExtension) renderAPIKeysListContent(req *http.Request, currentApp *app.App, currentUser *user.User) g.Node {
	ctx := req.Context()

	basePath := e.getBasePath()

	// Fetch API keys for the app
	filter := &apikey.ListAPIKeysFilter{
		PaginationParams: pagination.PaginationParams{
			Limit:  100,
			Offset: 0,
		},
		AppID: currentApp.ID,
	}

	keysResp, err := e.plugin.service.ListAPIKeys(ctx, filter)
	if err != nil {
		return e.renderError("Failed to load API keys", err.Error())
	}

	keys := keysResp.Data

	appIDStr := currentApp.ID.String()

	return Div(
		Class("space-y-2"),
		g.Attr("x-data", fmt.Sprintf(`{
			appId: '%s',
			showCreateModal: false,
			showViewKeyModal: false,
			showRevokeModal: false,
			newAPIKey: '',
			revokeKeyID: '',
			revokeKeyName: '',
			copied: false,
			creating: false,
			selectedKeyType: 'rk',
			selectedScopes: [],
			openRevokeModal(id, name) {
				this.revokeKeyID = id;
				this.revokeKeyName = name;
				this.showRevokeModal = true;
			},
			toggleScope(scope) {
				if (this.selectedScopes.includes(scope)) {
					this.selectedScopes = this.selectedScopes.filter(s => s !== scope);
				} else {
					this.selectedScopes.push(scope);
				}
				document.getElementById('scopes').value = this.selectedScopes.join(',');
			},
			isScopeSelected(scope) {
				return this.selectedScopes.includes(scope);
			},
			async createAPIKey(event) {
				if (this.creating) return;
				this.creating = true;
				try {
					const formData = new FormData(event.target);
					const result = await $bridge.call('createAPIKey', {
						appId: this.appId,
						name: formData.get('name'),
						keyType: formData.get('key_type') || 'rk',
						scopes: formData.get('scopes') || '',
						rateLimit: parseInt(formData.get('rate_limit') || '1000'),
						expiresIn: parseInt(formData.get('expires_in') || '0'),
						description: formData.get('description') || ''
					});
					if (result.success) {
						this.newAPIKey = result.key;
						this.showCreateModal = false;
						this.showViewKeyModal = true;
					} else {
						alert('Error: ' + (result.error || 'Failed to create API key'));
					}
				} catch (err) {
					alert('Error: ' + (err.message || 'Failed to create API key'));
				} finally {
					this.creating = false;
				}
			},
			async rotateAPIKey(keyId) {
				if (!confirm('Are you sure you want to rotate this key? The old key will stop working immediately.')) {
					return;
				}
				try {
					const result = await $bridge.call('rotateAPIKey', {
						appId: this.appId,
						keyId: keyId
					});
					if (result.success) {
						this.newAPIKey = result.key;
						this.showViewKeyModal = true;
					} else {
						alert('Error: ' + (result.error || 'Failed to rotate API key'));
					}
				} catch (err) {
					alert('Error: ' + (err.message || 'Failed to rotate API key'));
				}
			},
			async revokeAPIKey(event) {
				try {
					const result = await $bridge.call('revokeAPIKey', {
						appId: this.appId,
						keyId: this.revokeKeyID
					});
					if (result.success) {
						this.showRevokeModal = false;
						window.location.reload();
					} else {
						alert('Error: ' + (result.error || 'Failed to revoke API key'));
					}
				} catch (err) {
					alert('Error: ' + (err.message || 'Failed to revoke API key'));
				}
			},
			copyToClipboard(text) {
				navigator.clipboard.writeText(text).then(() => {
					this.copied = true;
					setTimeout(() => { this.copied = false; }, 2000);
				});
			}
		}`, appIDStr)),

		// Page header
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
					g.Text("API Keys")),
				P(Class("mt-2 text-slate-600 dark:text-gray-400"),
					g.Text("Manage API keys for programmatic access to your application")),
			),
			Button(
				Type("button"),
				Class("py-2 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg border border-transparent bg-violet-600 text-white hover:bg-violet-700 focus:outline-none focus:bg-violet-700 disabled:opacity-50 disabled:pointer-events-none"),
				g.Attr("@click", "showCreateModal = true"),
				lucide.Plus(Class("size-4")),
				g.Text("Create API Key"),
			),
		),

		// Keys list or empty state
		g.If(len(keys) > 0,
			e.renderKeysTable(keys, currentApp, basePath),
		),
		g.If(len(keys) == 0,
			e.renderEmptyState(),
		),

		// Modals
		e.renderCreateModal(currentApp, basePath),
		e.renderViewKeyModal(),
		e.renderRevokeModal(currentApp, basePath),
	)
}

// renderKeysTable renders the table of API keys.
func (e *DashboardExtension) renderKeysTable(keys []*apikey.APIKey, currentApp *app.App, basePath string) g.Node {
	return Div(
		Class("bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden"),
		Div(Class("overflow-x-auto"),
			Table(Class("w-full"),
				THead(Class("bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700"),
					Tr(
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Name")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Type & Prefix")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Scopes")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Usage")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Last Used")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
						Th(Class("px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
					),
				),
				TBody(Class("bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:border-gray-700"),
					g.Group(e.renderKeyRows(keys, currentApp, basePath)),
				),
			),
		),
	)
}

// renderKeyRows renders individual key rows.
func (e *DashboardExtension) renderKeyRows(keys []*apikey.APIKey, currentApp *app.App, basePath string) []g.Node {
	rows := make([]g.Node, len(keys))
	for i, key := range keys {
		rows[i] = e.renderKeyRow(key, currentApp, basePath)
	}

	return rows
}

// renderKeyRow renders a single API key row.
func (e *DashboardExtension) renderKeyRow(key *apikey.APIKey, currentApp *app.App, basePath string) g.Node {
	statusClass := "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
	statusText := "Active"

	if !key.Active {
		statusClass = "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
		statusText = "Revoked"
	}

	lastUsed := "Never"
	if key.LastUsedAt != nil {
		lastUsed = formatTimeAgo(*key.LastUsedAt)
	}

	return Tr(Class("hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"),
		// Name
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(Class("text-sm font-medium text-gray-900 dark:text-white"), g.Text(key.Name)),
			g.If(key.ExpiresAt != nil,
				Div(Class("text-xs text-gray-500 dark:text-gray-400 mt-1"),
					lucide.Clock(Class("inline h-3 w-3 mr-1")),
					g.Text("Expires "+formatTimeAgo(*key.ExpiresAt)),
				),
			),
		),

		// Key Prefix & Type
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(Class("flex items-center gap-2"),
				// Key type badge
				g.If(key.KeyType == apikey.KeyTypePublishable,
					Span(Class("px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-400 text-xs font-semibold rounded"),
						g.Text("PK"),
					),
				),
				g.If(key.KeyType == apikey.KeyTypeSecret,
					Span(Class("px-2 py-1 bg-purple-100 dark:bg-purple-900/30 text-purple-800 dark:text-purple-400 text-xs font-semibold rounded"),
						g.Text("SK"),
					),
				),
				g.If(key.KeyType == apikey.KeyTypeRestricted,
					Span(Class("px-2 py-1 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-400 text-xs font-semibold rounded"),
						g.Text("RK"),
					),
				),
				// Key prefix
				Code(Class("px-2 py-1 bg-gray-100 dark:bg-gray-900 text-sm text-gray-800 dark:text-gray-200 rounded font-mono"),
					g.Text(key.Prefix+"..."),
				),
			),
		),

		// Scopes
		Td(Class("px-6 py-4"),
			Div(Class("flex flex-wrap gap-1"),
				g.Group(e.renderScopeBadges(key.Scopes)),
			),
		),

		// Usage
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(Class("text-sm text-gray-900 dark:text-white"),
				g.Text(fmt.Sprintf("%d requests", key.UsageCount)),
			),
			g.If(key.RateLimit > 0,
				Div(Class("text-xs text-gray-500 dark:text-gray-400"),
					g.Text(fmt.Sprintf("Limit: %d/hr", key.RateLimit)),
				),
			),
		),

		// Last Used
		Td(Class("px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400"),
			g.Text(lastUsed),
		),

		// Status
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Span(Class("px-2 inline-flex text-xs leading-5 font-semibold rounded-full "+statusClass),
				g.Text(statusText),
			),
		),

		// Actions
		Td(Class("px-6 py-4 whitespace-nowrap text-right text-sm font-medium"),
			Div(Class("flex justify-end gap-2"),
				g.If(key.Active,
					Button(
						Type("button"),
						Class("text-blue-600 hover:text-blue-900 dark:text-blue-400 dark:hover:text-blue-300 p-1"),
						g.Attr("@click", fmt.Sprintf("rotateAPIKey('%s')", key.ID.String())),
						g.Attr("title", "Rotate Key"),
						lucide.RefreshCw(Class("h-4 w-4")),
					),
				),
				g.If(key.Active,
					Button(
						Type("button"),
						Class("text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300 p-1"),
						g.Attr("@click", fmt.Sprintf("openRevokeModal('%s', '%s')", key.ID.String(), key.Name)),
						g.Attr("title", "Revoke Key"),
						lucide.Ban(Class("h-4 w-4")),
					),
				),
			),
		),
	)
}

// renderScopeBadges renders scope badges.
func (e *DashboardExtension) renderScopeBadges(scopes []string) []g.Node {
	if len(scopes) == 0 {
		return []g.Node{
			Span(Class("px-2 py-1 bg-gray-100 dark:bg-gray-700 text-xs text-gray-600 dark:text-gray-400 rounded"),
				g.Text("No scopes")),
		}
	}

	nodes := make([]g.Node, len(scopes))
	for i, scope := range scopes {
		nodes[i] = Span(
			Class("px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-xs text-blue-800 dark:text-blue-400 rounded"),
			g.Text(scope),
		)
	}

	return nodes
}

// ScopeDefinition defines a scope with metadata.
type ScopeDefinition struct {
	Value       string
	Label       string
	Description string
	DangerLevel string // "safe", "moderate", "dangerous", "critical"
	Category    string // "read", "write", "admin", "special"
}

// getScopeDefinitions returns all available scopes organized by category.
func (e *DashboardExtension) getScopeDefinitions() map[string][]ScopeDefinition {
	return map[string][]ScopeDefinition{
		"Read Operations (Safe)": {
			{Value: "app:identify", Label: "App Identify", Description: "Identify which app is making the request", DangerLevel: "safe", Category: "read"},
			{Value: "users:read", Label: "Read Users", Description: "View user information and profiles", DangerLevel: "safe", Category: "read"},
			{Value: "users:verify", Label: "Verify Users", Description: "Verify user tokens and sessions", DangerLevel: "safe", Category: "read"},
			{Value: "sessions:read", Label: "Read Sessions", Description: "View active user sessions", DangerLevel: "safe", Category: "read"},
			{Value: "roles:read", Label: "Read Roles", Description: "View roles and permissions", DangerLevel: "safe", Category: "read"},
			{Value: "audit:read", Label: "Read Audit Logs", Description: "View audit logs and activity", DangerLevel: "safe", Category: "read"},
		},
		"Write Operations (Moderate)": {
			{Value: "sessions:create", Label: "Create Sessions", Description: "Create new user sessions (for auth flows)", DangerLevel: "moderate", Category: "write"},
			{Value: "users:create", Label: "Create Users", Description: "Register new users", DangerLevel: "moderate", Category: "write"},
			{Value: "users:update", Label: "Update Users", Description: "Modify user information", DangerLevel: "moderate", Category: "write"},
			{Value: "sessions:update", Label: "Update Sessions", Description: "Modify session properties", DangerLevel: "moderate", Category: "write"},
			{Value: "tokens:create", Label: "Create Tokens", Description: "Generate authentication tokens", DangerLevel: "moderate", Category: "write"},
		},
		"Delete Operations (Dangerous)": {
			{Value: "sessions:delete", Label: "Delete Sessions", Description: "Revoke and delete user sessions", DangerLevel: "dangerous", Category: "write"},
			{Value: "sessions:revoke", Label: "Revoke Sessions", Description: "Force logout users", DangerLevel: "dangerous", Category: "write"},
			{Value: "users:delete", Label: "Delete Users", Description: "Permanently delete user accounts", DangerLevel: "dangerous", Category: "write"},
			{Value: "tokens:revoke", Label: "Revoke Tokens", Description: "Invalidate authentication tokens", DangerLevel: "dangerous", Category: "write"},
		},
		"Administrative (Critical)": {
			{Value: "admin:full", Label: "Full Admin Access", Description: "⚠️ Unrestricted access to all operations", DangerLevel: "critical", Category: "admin"},
			{Value: "admin:users", Label: "User Admin", Description: "Full control over all users", DangerLevel: "critical", Category: "admin"},
			{Value: "admin:roles", Label: "Role Admin", Description: "Manage roles and permissions", DangerLevel: "critical", Category: "admin"},
			{Value: "admin:config", Label: "Config Admin", Description: "Modify system configuration", DangerLevel: "critical", Category: "admin"},
			{Value: "admin:keys", Label: "API Key Admin", Description: "Manage API keys", DangerLevel: "critical", Category: "admin"},
		},
		"Special Permissions": {
			{Value: "impersonate:users", Label: "Impersonate Users", Description: "Act as other users (dangerous)", DangerLevel: "critical", Category: "special"},
			{Value: "webhooks:manage", Label: "Manage Webhooks", Description: "Create and configure webhooks", DangerLevel: "moderate", Category: "special"},
			{Value: "export:data", Label: "Export Data", Description: "Export user data and reports", DangerLevel: "moderate", Category: "special"},
		},
	}
}

// renderScopeSelector renders the scope selection UI with categories.
func (e *DashboardExtension) renderScopeSelector() g.Node {
	scopesByCategory := e.getScopeDefinitions()

	return Div(
		Div(Class("space-y-4"),
			// Header
			Div(Class("flex items-center justify-between"),
				Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300"),
					g.Text("Permissions & Scopes")),
				Button(
					Type("button"),
					Class("text-xs text-violet-600 dark:text-violet-400 hover:text-violet-700 dark:hover:text-violet-300"),
					g.Attr("@click", "selectedScopes = []; document.getElementById('scopes').value = ''"),
					g.Text("Clear All"),
				),
			),
			P(Class("text-xs text-gray-500 dark:text-gray-400"),
				g.Text("Select the permissions this API key should have. Keys with no scopes will use defaults based on type.")),

			// Hidden input to store selected scopes
			Input(
				Type("hidden"),
				ID("scopes"),
				Name("scopes"),
				Value(""),
			),

			// Scope Categories
			Div(Class("space-y-4 max-h-96 overflow-y-auto pr-2"),
				g.Group(e.renderScopeCategories(scopesByCategory)),
			),

			// Selected Count
			Div(Class("pt-3 border-t border-gray-200 dark:border-gray-700"),
				P(Class("text-xs text-gray-600 dark:text-gray-400"),
					g.Attr("x-show", "selectedScopes.length > 0"),
					Span(Class("font-semibold"), g.Attr("x-text", "selectedScopes.length")),
					g.Text(" scope(s) selected"),
				),
			),
		),
	)
}

// renderScopeCategories renders all scope categories.
func (e *DashboardExtension) renderScopeCategories(scopesByCategory map[string][]ScopeDefinition) []g.Node {
	categories := []string{
		"Read Operations (Safe)",
		"Write Operations (Moderate)",
		"Delete Operations (Dangerous)",
		"Administrative (Critical)",
		"Special Permissions",
	}

	nodes := make([]g.Node, len(categories))
	for i, category := range categories {
		scopes := scopesByCategory[category]
		nodes[i] = e.renderScopeCategory(category, scopes)
	}

	return nodes
}

// renderScopeCategory renders a single category of scopes.
func (e *DashboardExtension) renderScopeCategory(categoryName string, scopes []ScopeDefinition) g.Node {
	return Div(Class("space-y-2"),
		// Category Header
		H4(Class("text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase tracking-wider"),
			g.Text(categoryName)),

		// Scope Items
		Div(Class("space-y-2"),
			g.Group(e.renderScopeItems(scopes)),
		),
	)
}

// renderScopeItems renders individual scope checkboxes.
func (e *DashboardExtension) renderScopeItems(scopes []ScopeDefinition) []g.Node {
	nodes := make([]g.Node, len(scopes))
	for i, scope := range scopes {
		nodes[i] = e.renderScopeItem(scope)
	}

	return nodes
}

// renderScopeItem renders a single scope checkbox with Pines-style design.
func (e *DashboardExtension) renderScopeItem(scope ScopeDefinition) g.Node {
	// Determine border color and badge based on danger level
	borderClass := "border-green-200 hover:border-green-300 dark:border-green-800 dark:hover:border-green-700"
	badgeClass := "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
	badgeIcon := lucide.Shield(Class("h-3 w-3"))
	badgeText := "Safe"

	switch scope.DangerLevel {
	case "moderate":
		borderClass = "border-yellow-200 hover:border-yellow-300 dark:border-yellow-800 dark:hover:border-yellow-700"
		badgeClass = "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
		badgeIcon = lucide.TriangleAlert(Class("h-3 w-3"))
		badgeText = "Moderate"
	case "dangerous":
		borderClass = "border-orange-200 hover:border-orange-300 dark:border-orange-800 dark:hover:border-orange-700"
		badgeClass = "bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400"
		badgeIcon = lucide.TriangleAlert(Class("h-3 w-3"))
		badgeText = "Dangerous"
	case "critical":
		borderClass = "border-red-200 hover:border-red-300 dark:border-red-800 dark:hover:border-red-700"
		badgeClass = "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
		badgeIcon = lucide.ShieldAlert(Class("h-3 w-3"))
		badgeText = "Critical"
	}

	return Label(
		g.Attr("@click", fmt.Sprintf("toggleScope('%s')", scope.Value)),
		g.Attr(":class", fmt.Sprintf("isScopeSelected('%s') ? 'bg-violet-50 dark:bg-violet-900/20 %s' : 'bg-white dark:bg-gray-800 %s'", scope.Value, borderClass, borderClass)),
		Class("flex items-start p-4 space-x-3 border rounded-lg shadow-sm cursor-pointer transition-all duration-200"),

		// Checkbox
		Input(
			Type("checkbox"),
			g.Attr(":checked", fmt.Sprintf("isScopeSelected('%s')", scope.Value)),
			Class("mt-0.5 text-violet-600 rounded focus:ring-violet-500 dark:bg-gray-700 dark:border-gray-600"),
		),

		// Content
		Span(Class("relative flex-1 flex flex-col space-y-1.5 leading-none"),
			// Title and Badge
			Span(Class("flex items-center justify-between gap-2"),
				Span(Class("font-semibold text-sm text-gray-900 dark:text-white"), g.Text(scope.Label)),
				Span(Class("inline-flex items-center gap-1 px-2 py-0.5 text-xs font-medium rounded "+badgeClass),
					badgeIcon,
					g.Text(badgeText),
				),
			),
			// Description
			Span(Class("text-xs text-gray-600 dark:text-gray-400"), g.Text(scope.Description)),
			// Scope Value
			Code(Class("text-xs text-gray-500 dark:text-gray-500 font-mono"), g.Text(scope.Value)),
		),
	)
}

// renderEmptyState renders the empty state when no keys exist.
func (e *DashboardExtension) renderEmptyState() g.Node {
	return Div(Class("text-center py-12 bg-white dark:bg-neutral-800 rounded-xl border border-gray-200 dark:border-neutral-700"),
		Div(Class("mx-auto size-16 text-gray-400 dark:text-neutral-600 mb-4"),
			lucide.Key(Class("size-16")),
		),
		H3(Class("text-lg font-medium text-gray-900 dark:text-white mb-2"), g.Text("No API Keys")),
		P(Class("text-gray-600 dark:text-neutral-400 mb-6 max-w-sm mx-auto"), g.Text("Create your first API key to enable programmatic access to your application")),
		Button(
			Type("button"),
			Class("py-2 px-3 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg border border-transparent bg-violet-600 text-white hover:bg-violet-700 focus:outline-none focus:bg-violet-700 disabled:opacity-50 disabled:pointer-events-none"),
			g.Attr("@click", "showCreateModal = true"),
			lucide.Plus(Class("size-4")),
			g.Text("Create API Key"),
		),
	)
}

// renderCreateModal renders the create API key modal using Alpine.js/Pines dialog pattern.
func (e *DashboardExtension) renderCreateModal(currentApp *app.App, basePath string) g.Node {
	actionURL := fmt.Sprintf("%s/app/%s/settings/api-keys/create", basePath, currentApp.ID.String())

	return Div(
		g.Attr("x-show", "showCreateModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("@keydown.escape.window", "showCreateModal = false"),

		Div(Class("flex min-h-screen items-center justify-center p-4"),
			// Backdrop
			Div(
				g.Attr("x-show", "showCreateModal"),
				g.Attr("x-transition:enter", "ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0"),
				g.Attr("x-transition:enter-end", "opacity-100"),
				g.Attr("x-transition:leave", "ease-in duration-200"),
				g.Attr("x-transition:leave-start", "opacity-100"),
				g.Attr("x-transition:leave-end", "opacity-0"),
				Class("fixed inset-0 bg-black/50 dark:bg-black/70"),
				g.Attr("@click", "showCreateModal = false"),
			),

			// Modal Content
			Div(
				g.Attr("x-show", "showCreateModal"),
				g.Attr("x-transition:enter", "ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
				g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
				g.Attr("x-transition:leave", "ease-in duration-200"),
				g.Attr("x-transition:leave-start", "opacity-100 scale-100"),
				g.Attr("x-transition:leave-end", "opacity-0 scale-95"),
				Class("relative bg-white dark:bg-neutral-800 rounded-xl shadow-xl max-w-2xl w-full max-h-[70vh] flex flex-col"),

				// Modal Header
				Div(
					Class("flex justify-between items-center py-4 px-6 border-b border-gray-200 dark:border-neutral-700 shrink-0"),
					Div(
						Class("flex items-center gap-2"),
						lucide.Key(Class("size-5 text-violet-600 dark:text-violet-400")),
						H3(
							Class("font-bold text-gray-800 dark:text-white"),
							g.Text("Create API Key"),
						),
					),
					Button(
						Type("button"),
						Class("size-8 inline-flex justify-center items-center gap-x-2 rounded-full border border-transparent bg-gray-100 text-gray-800 hover:bg-gray-200 focus:outline-none focus:bg-gray-200 dark:bg-neutral-700 dark:hover:bg-neutral-600 dark:text-neutral-400 dark:focus:bg-neutral-600"),
						g.Attr("@click", "showCreateModal = false"),
						lucide.X(Class("shrink-0 size-4")),
					),
				),

				// Modal Body - Form
				FormEl(
					g.Attr("method", "POST"),
					g.Attr("action", actionURL),
					g.Attr("@submit.prevent", "createAPIKey($event)"),
					Class("flex flex-col overflow-hidden"),

					Div(Class("p-6 overflow-y-auto flex-1 space-y-5"),
						// Name Field
						Div(
							Label(
								Class("block text-sm font-medium mb-2 dark:text-white"),
								g.Attr("for", "key_name"),
								g.Text("Key Name"),
								Span(Class("text-red-500"), g.Text(" *")),
							),
							Input(
								Type("text"),
								ID("key_name"),
								Name("name"),
								Required(),
								Class("py-3 px-4 block w-full border border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-900 dark:border-neutral-700 dark:text-neutral-300 dark:placeholder-neutral-500"),
								g.Attr("placeholder", "e.g., Production API Key"),
							),
							P(Class("mt-1 text-xs text-gray-500 dark:text-neutral-500"),
								g.Text("A descriptive name to identify this key")),
						),

						// Key Type Field
						Div(
							Label(
								Class("block text-sm font-medium mb-2 dark:text-white"),
								g.Attr("for", "key_type"),
								g.Text("Key Type"),
								Span(Class("text-red-500"), g.Text(" *")),
							),
							Select(
								ID("key_type"),
								Name("key_type"),
								Required(),
								g.Attr("x-model", "selectedKeyType"),
								Class("py-3 px-4 pe-9 block w-full border border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-900 dark:border-neutral-700 dark:text-neutral-300"),
								Option(Value("pk"), g.Text("Publishable Key (pk) - Frontend-safe")),
								Option(Value("sk"), g.Text("Secret Key (sk) - Backend admin")),
								Option(Value("rk"), Selected(), g.Text("Restricted Key (rk) - Backend scoped")),
							),

							// Key type description boxes
							Div(
								Class("mt-3"),
								// Publishable Key Info
								Div(
									g.Attr("x-show", "selectedKeyType === 'pk'"),
									g.Attr("x-transition:enter", "transition ease-out duration-200"),
									Class("p-3 bg-blue-50 border border-blue-200 rounded-lg dark:bg-blue-900/20 dark:border-blue-800"),
									Div(
										Class("flex"),
										Div(
											Class("shrink-0"),
											lucide.Globe(Class("size-4 text-blue-600 mt-0.5 dark:text-blue-400")),
										),
										Div(
											Class("ms-3"),
											H3(Class("text-sm font-medium text-blue-800 dark:text-blue-200"),
												g.Text("Frontend-Safe Key")),
											P(Class("mt-1 text-xs text-blue-700 dark:text-blue-300"),
												g.Text("Safe to use in client-side applications. Permissions are automatically set to:")),
											Ul(Class("mt-2 text-xs text-blue-700 dark:text-blue-300 list-disc list-inside space-y-1"),
												Li(g.Text("app:identify - Identify your application")),
												Li(g.Text("sessions:create - Create user sessions")),
												Li(g.Text("users:verify - Verify user tokens")),
											),
										),
									),
								),

								// Secret Key Info
								Div(
									g.Attr("x-show", "selectedKeyType === 'sk'"),
									g.Attr("x-transition:enter", "transition ease-out duration-200"),
									Class("p-3 bg-amber-50 border border-amber-200 rounded-lg dark:bg-amber-900/20 dark:border-amber-800"),
									Div(
										Class("flex"),
										Div(
											Class("shrink-0"),
											lucide.ShieldAlert(Class("size-4 text-amber-600 mt-0.5 dark:text-amber-400")),
										),
										Div(
											Class("ms-3"),
											H3(Class("text-sm font-medium text-amber-800 dark:text-amber-200"),
												g.Text("⚠️ Full Admin Access")),
											P(Class("mt-1 text-xs text-amber-700 dark:text-amber-300"),
												g.Text("This key has unrestricted access to all operations. Keep it secret and never expose in frontend code. Permissions automatically include admin:full.")),
										),
									),
								),

								// Restricted Key Info
								Div(
									g.Attr("x-show", "selectedKeyType === 'rk'"),
									g.Attr("x-transition:enter", "transition ease-out duration-200"),
									Class("p-3 bg-violet-50 border border-violet-200 rounded-lg dark:bg-violet-900/20 dark:border-violet-800"),
									Div(
										Class("flex"),
										Div(
											Class("shrink-0"),
											lucide.Settings2(Class("size-4 text-violet-600 mt-0.5 dark:text-violet-400")),
										),
										Div(
											Class("ms-3"),
											H3(Class("text-sm font-medium text-violet-800 dark:text-violet-200"),
												g.Text("Custom Scoped Key")),
											P(Class("mt-1 text-xs text-violet-700 dark:text-violet-300"),
												g.Text("Backend key with fine-grained access control. Select specific permissions below.")),
										),
									),
								),
							),
						),

						// Scopes Selection - Only show for restricted keys (rk)
						Div(
							g.Attr("x-show", "selectedKeyType === 'rk'"),
							g.Attr("x-transition:enter", "transition ease-out duration-300"),
							g.Attr("x-transition:enter-start", "opacity-0 transform -translate-y-2"),
							g.Attr("x-transition:enter-end", "opacity-100 transform translate-y-0"),
							e.renderScopeSelector(),
						),

						// Rate Limit Field
						Div(
							Label(
								Class("block text-sm font-medium mb-2 dark:text-white"),
								g.Attr("for", "rate_limit"),
								g.Text("Rate Limit"),
							),
							Div(
								Class("relative"),
								Input(
									Type("number"),
									ID("rate_limit"),
									Name("rate_limit"),
									Value(strconv.Itoa(e.plugin.config.DefaultRateLimit)),
									g.Attr("min", "0"),
									Class("py-3 px-4 pe-20 block w-full border border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-900 dark:border-neutral-700 dark:text-neutral-300"),
								),
								Div(
									Class("absolute inset-y-0 end-0 flex items-center pointer-events-none pe-4"),
									Span(Class("text-gray-500 dark:text-neutral-500 text-sm"), g.Text("req/hr")),
								),
							),
							P(Class("mt-1 text-xs text-gray-500 dark:text-neutral-500"),
								g.Text("Set to 0 for unlimited (not recommended for production)")),
						),

						// Expiry Field
						Div(
							Label(
								Class("block text-sm font-medium mb-2 dark:text-white"),
								g.Attr("for", "expires_in"),
								g.Text("Expiration"),
							),
							Div(
								Class("relative"),
								Input(
									Type("number"),
									ID("expires_in"),
									Name("expires_in"),
									g.Attr("min", "1"),
									g.Attr("max", "365"),
									Class("py-3 px-4 pe-16 block w-full border border-gray-200 rounded-lg text-sm focus:border-violet-500 focus:ring-violet-500 dark:bg-neutral-900 dark:border-neutral-700 dark:text-neutral-300"),
									g.Attr("placeholder", "Leave empty for no expiration"),
								),
								Div(
									Class("absolute inset-y-0 end-0 flex items-center pointer-events-none pe-4"),
									Span(Class("text-gray-500 dark:text-neutral-500 text-sm"), g.Text("days")),
								),
							),
							P(Class("mt-1 text-xs text-gray-500 dark:text-neutral-500"),
								g.Text("Recommended: Set an expiration for better security")),
						),
					),

					// Modal Footer
					Div(
						Class("flex justify-end items-center gap-x-2 py-4 px-6 border-t border-gray-200 dark:border-neutral-700 shrink-0"),
						Button(
							Type("button"),
							Class("py-2 px-4 text-sm font-medium rounded-lg border border-gray-200 bg-white text-gray-800 shadow-sm hover:bg-gray-50 dark:bg-neutral-800 dark:border-neutral-700 dark:text-white dark:hover:bg-neutral-700 transition-colors"),
							g.Attr("@click", "showCreateModal = false"),
							g.Text("Cancel"),
						),
						Button(
							Type("submit"),
							Class("py-2 px-4 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors"),
							lucide.Plus(Class("size-4")),
							g.Text("Create Key"),
						),
					),
				),
			),
		),
	)
}

// renderViewKeyModal renders the modal to display newly created API key using Alpine.js/Pines dialog pattern.
func (e *DashboardExtension) renderViewKeyModal() g.Node {
	return Div(
		g.Attr("x-show", "showViewKeyModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 overflow-y-auto"),

		Div(Class("flex min-h-screen items-center justify-center p-4"),
			// Backdrop
			Div(
				g.Attr("x-show", "showViewKeyModal"),
				g.Attr("x-transition:enter", "ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0"),
				g.Attr("x-transition:enter-end", "opacity-100"),
				g.Attr("x-transition:leave", "ease-in duration-200"),
				g.Attr("x-transition:leave-start", "opacity-100"),
				g.Attr("x-transition:leave-end", "opacity-0"),
				Class("fixed inset-0 bg-black/50 dark:bg-black/70"),
			),

			// Modal Content
			Div(
				g.Attr("x-show", "showViewKeyModal"),
				g.Attr("x-transition:enter", "ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
				g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
				g.Attr("x-transition:leave", "ease-in duration-200"),
				g.Attr("x-transition:leave-start", "opacity-100 scale-100"),
				g.Attr("x-transition:leave-end", "opacity-0 scale-95"),
				Class("relative bg-white dark:bg-neutral-800 rounded-xl shadow-xl max-w-lg w-full max-h-[70vh] flex flex-col"),

				// Modal Header
				Div(
					Class("flex justify-between items-center py-4 px-6 border-b border-gray-200 dark:border-neutral-700 shrink-0"),
					Div(
						Class("flex items-center gap-2"),
						Div(
							Class("size-8 flex justify-center items-center rounded-full bg-teal-100 dark:bg-teal-900/30"),
							lucide.Check(Class("size-4 text-teal-600 dark:text-teal-400")),
						),
						H3(
							Class("font-bold text-gray-800 dark:text-white"),
							g.Text("API Key Created"),
						),
					),
					Button(
						Type("button"),
						Class("size-8 inline-flex justify-center items-center gap-x-2 rounded-full border border-transparent bg-gray-100 text-gray-800 hover:bg-gray-200 dark:bg-neutral-700 dark:hover:bg-neutral-600 dark:text-neutral-400"),
						g.Attr("@click", "showViewKeyModal = false; newAPIKey = ''; setTimeout(() => window.location.reload(), 100)"),
						lucide.X(Class("shrink-0 size-4")),
					),
				),

				// Modal Body
				Div(Class("p-6 space-y-4 overflow-y-auto flex-1"),
					// Warning Alert
					Div(
						Class("bg-amber-50 border border-amber-200 text-sm text-amber-800 rounded-lg p-4 dark:bg-amber-900/20 dark:border-amber-800 dark:text-amber-400"),
						Div(
							Class("flex"),
							Div(
								Class("shrink-0"),
								lucide.TriangleAlert(Class("size-4 mt-0.5")),
							),
							Div(
								Class("ms-3"),
								H3(Class("text-sm font-semibold"),
									g.Text("Important: Save Your Key")),
								Div(
									Class("mt-1 text-sm"),
									P(g.Text("This is the only time your API key will be displayed. Copy it now and store it securely.")),
								),
							),
						),
					),

					// API Key Display
					Div(
						Label(
							Class("block text-sm font-medium mb-2 dark:text-white"),
							g.Text("Your API Key"),
						),
						Div(
							Class("flex gap-2"),
							Input(
								Type("text"),
								g.Attr("x-model", "newAPIKey"),
								g.Attr("readonly", ""),
								Class("py-3 px-4 block w-full border border-gray-200 rounded-lg text-sm bg-gray-50 font-mono dark:bg-neutral-900 dark:border-neutral-700 dark:text-neutral-300"),
							),
							Button(
								Type("button"),
								g.Attr("@click", "copyToClipboard(newAPIKey)"),
								Class("py-3 px-4 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg border border-gray-200 bg-white text-gray-800 shadow-sm hover:bg-gray-50 dark:bg-neutral-800 dark:border-neutral-700 dark:text-white dark:hover:bg-neutral-700 transition-colors"),
								g.Attr("x-text", "copied ? 'Copied!' : 'Copy'"),
								g.Attr(":class", "copied ? 'bg-teal-50 text-teal-600 border-teal-200 dark:bg-teal-900/30 dark:text-teal-400 dark:border-teal-800' : ''"),
							),
						),
					),
				),

				// Modal Footer
				Div(
					Class("flex justify-end items-center gap-x-2 py-4 px-6 border-t border-gray-200 dark:border-neutral-700 shrink-0"),
					Button(
						Type("button"),
						Class("py-2 px-4 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg bg-violet-600 text-white hover:bg-violet-700 transition-colors"),
						g.Attr("@click", "showViewKeyModal = false; newAPIKey = ''; setTimeout(() => window.location.reload(), 100)"),
						lucide.Check(Class("size-4")),
						g.Text("I've Saved The Key"),
					),
				),
			),
		),
	)
}

// renderRevokeModal renders the confirmation modal for revoking an API key using Alpine.js/Pines dialog pattern.
func (e *DashboardExtension) renderRevokeModal(currentApp *app.App, basePath string) g.Node {
	actionURL := fmt.Sprintf("%s/app/%s/settings/api-keys/revoke", basePath, currentApp.ID.String())

	return Div(
		g.Attr("x-show", "showRevokeModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("@keydown.escape.window", "showRevokeModal = false"),

		Div(Class("flex min-h-screen items-center justify-center p-4"),
			// Backdrop
			Div(
				g.Attr("x-show", "showRevokeModal"),
				g.Attr("x-transition:enter", "ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0"),
				g.Attr("x-transition:enter-end", "opacity-100"),
				g.Attr("x-transition:leave", "ease-in duration-200"),
				g.Attr("x-transition:leave-start", "opacity-100"),
				g.Attr("x-transition:leave-end", "opacity-0"),
				Class("fixed inset-0 bg-black/50 dark:bg-black/70"),
				g.Attr("@click", "showRevokeModal = false"),
			),

			// Modal Content
			Div(
				g.Attr("x-show", "showRevokeModal"),
				g.Attr("x-transition:enter", "ease-out duration-300"),
				g.Attr("x-transition:enter-start", "opacity-0 scale-95"),
				g.Attr("x-transition:enter-end", "opacity-100 scale-100"),
				g.Attr("x-transition:leave", "ease-in duration-200"),
				g.Attr("x-transition:leave-start", "opacity-100 scale-100"),
				g.Attr("x-transition:leave-end", "opacity-0 scale-95"),
				Class("relative bg-white dark:bg-neutral-800 rounded-xl shadow-xl max-w-md w-full max-h-[70vh] flex flex-col"),

				// Modal Header
				Div(
					Class("flex justify-between items-center py-4 px-6 border-b border-gray-200 dark:border-neutral-700 shrink-0"),
					Div(
						Class("flex items-center gap-2"),
						Div(
							Class("size-8 flex justify-center items-center rounded-full bg-red-100 dark:bg-red-900/30"),
							lucide.Ban(Class("size-4 text-red-600 dark:text-red-400")),
						),
						H3(
							Class("font-bold text-gray-800 dark:text-white"),
							g.Text("Revoke API Key"),
						),
					),
					Button(
						Type("button"),
						Class("size-8 inline-flex justify-center items-center gap-x-2 rounded-full border border-transparent bg-gray-100 text-gray-800 hover:bg-gray-200 dark:bg-neutral-700 dark:hover:bg-neutral-600 dark:text-neutral-400"),
						g.Attr("@click", "showRevokeModal = false"),
						lucide.X(Class("shrink-0 size-4")),
					),
				),

				// Modal Body with Form
				FormEl(
					g.Attr("method", "POST"),
					g.Attr("action", actionURL),
					g.Attr("@submit.prevent", "revokeAPIKey($event)"),
					Class("flex flex-col overflow-hidden"),

					Input(Type("hidden"), Name("key_id"), g.Attr("x-model", "revokeKeyID")),

					Div(Class("p-6 space-y-4 overflow-y-auto flex-1"),
						// Danger Alert
						Div(
							Class("bg-red-50 border border-red-200 text-sm text-red-800 rounded-lg p-4 dark:bg-red-900/20 dark:border-red-800 dark:text-red-400"),
							Div(
								Class("flex"),
								Div(
									Class("shrink-0"),
									lucide.TriangleAlert(Class("size-4 mt-0.5")),
								),
								Div(
									Class("ms-3"),
									H3(Class("text-sm font-semibold"),
										g.Text("Warning: Irreversible Action")),
									Div(
										Class("mt-2 text-sm"),
										P(g.Text("Revoking this key will immediately break any applications using it. This action cannot be undone.")),
									),
								),
							),
						),

						// Key name display
						P(Class("text-sm text-gray-600 dark:text-neutral-400"),
							g.Text("You are about to revoke: "),
							Strong(
								Class("text-gray-900 dark:text-white"),
								g.Attr("x-text", "revokeKeyName"),
							),
						),
					),

					// Modal Footer
					Div(
						Class("flex justify-end items-center gap-x-2 py-4 px-6 border-t border-gray-200 dark:border-neutral-700 shrink-0"),
						Button(
							Type("button"),
							Class("py-2 px-4 text-sm font-medium rounded-lg border border-gray-200 bg-white text-gray-800 shadow-sm hover:bg-gray-50 dark:bg-neutral-800 dark:border-neutral-700 dark:text-white dark:hover:bg-neutral-700 transition-colors"),
							g.Attr("@click", "showRevokeModal = false"),
							g.Text("Cancel"),
						),
						Button(
							Type("submit"),
							Class("py-2 px-4 inline-flex items-center gap-x-2 text-sm font-medium rounded-lg bg-red-600 text-white hover:bg-red-700 transition-colors"),
							lucide.Ban(Class("size-4")),
							g.Text("Revoke Key"),
						),
					),
				),
			),
		),
	)
}

// renderConfigContent renders the configuration page content.
func (e *DashboardExtension) renderConfigContent(currentApp *app.App) g.Node {
	basePath := e.getBasePath()

	actionURL := fmt.Sprintf("%s/app/%s/settings/api-keys-config/update", basePath, currentApp.ID.String())

	return Div(
		Class("space-y-2"),
		g.Attr("x-data", `{
			async saveConfig(event) {
				const formData = new FormData(event.target);
				const response = await fetch(event.target.action, {
					method: 'POST',
					body: formData
				});
				const result = await response.json();
				if (result.success) {
					alert('Configuration saved successfully!');
				} else {
					alert('Error: ' + (result.error || 'Failed to save configuration'));
				}
			}
		}`),

		// Page header
		Div(
			H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
				g.Text("API Key Configuration")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Configure default settings and limits for API keys")),
		),

		// Configuration form
		FormEl(
			g.Attr("method", "POST"),
			g.Attr("action", actionURL),
			g.Attr("@submit.prevent", "saveConfig($event)"),

			Div(Class("bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 space-y-2"),
				// Rate Limiting Section
				Div(
					H3(Class("text-lg font-semibold text-gray-900 dark:text-white mb-4"),
						g.Text("Rate Limiting")),
					Div(Class("grid gap-4 md:grid-cols-2"),
						Div(
							Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
								g.Text("Default Rate Limit (req/hr)")),
							Input(
								Type("number"),
								Name("default_rate_limit"),
								Value(strconv.Itoa(e.plugin.config.DefaultRateLimit)),
								g.Attr("min", "0"),
								Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-violet-500"),
							),
						),
						Div(
							Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
								g.Text("Maximum Rate Limit (req/hr)")),
							Input(
								Type("number"),
								Name("max_rate_limit"),
								Value(strconv.Itoa(e.plugin.config.MaxRateLimit)),
								g.Attr("min", "0"),
								Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-violet-500"),
							),
						),
					),
				),

				// Key Limits Section
				Div(
					H3(Class("text-lg font-semibold text-gray-900 dark:text-white mb-4"),
						g.Text("Key Limits")),
					Div(Class("grid gap-4 md:grid-cols-3"),
						Div(
							Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
								g.Text("Max Keys Per User")),
							Input(
								Type("number"),
								Name("max_keys_per_user"),
								Value(strconv.Itoa(e.plugin.config.MaxKeysPerUser)),
								g.Attr("min", "1"),
								Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-violet-500"),
							),
						),
						Div(
							Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
								g.Text("Max Keys Per Organization")),
							Input(
								Type("number"),
								Name("max_keys_per_org"),
								Value(strconv.Itoa(e.plugin.config.MaxKeysPerOrg)),
								g.Attr("min", "1"),
								Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-violet-500"),
							),
						),
						Div(
							Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
								g.Text("Key Length (bytes)")),
							Input(
								Type("number"),
								Name("key_length"),
								Value(strconv.Itoa(e.plugin.config.KeyLength)),
								g.Attr("min", "16"),
								g.Attr("max", "64"),
								Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-violet-500"),
							),
						),
					),
				),

				// Auto Cleanup Section
				Div(
					H3(Class("text-lg font-semibold text-gray-900 dark:text-white mb-4"),
						g.Text("Auto Cleanup")),
					Div(Class("grid gap-4 md:grid-cols-2"),
						Div(
							Label(Class("flex items-center gap-2 cursor-pointer"),
								Input(
									Type("checkbox"),
									Name("auto_cleanup_enabled"),
									g.If(e.plugin.config.AutoCleanup.Enabled, Checked()),
									Class("rounded border-gray-300 dark:border-gray-600 text-violet-600 focus:ring-violet-500"),
								),
								Span(Class("text-sm font-medium text-gray-700 dark:text-gray-300"),
									g.Text("Enable Auto Cleanup")),
							),
							P(Class("text-xs text-gray-500 dark:text-gray-400 mt-1 ml-6"),
								g.Text("Automatically remove expired API keys")),
						),
						Div(
							Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"),
								g.Text("Cleanup Interval (hours)")),
							Input(
								Type("number"),
								Name("cleanup_interval_hours"),
								Value(fmt.Sprintf("%.0f", e.plugin.config.AutoCleanup.Interval.Hours())),
								g.Attr("min", "1"),
								Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-violet-500"),
							),
						),
					),
				),
			),

			// Save button
			Div(Class("flex justify-end"),
				Button(
					Type("submit"),
					Class("px-6 py-2 bg-violet-600 hover:bg-violet-700 text-white rounded-lg transition-colors"),
					g.Text("Save Configuration"),
				),
			),
		),
	)
}

// renderSecurityContent renders the security settings page content.
func (e *DashboardExtension) renderSecurityContent(currentApp *app.App) g.Node {
	basePath := e.getBasePath()

	actionURL := fmt.Sprintf("%s/app/%s/settings/api-keys-security/update", basePath, currentApp.ID.String())

	return Div(
		Class("space-y-2"),
		g.Attr("x-data", `{
			async saveSecurity(event) {
				const formData = new FormData(event.target);
				const response = await fetch(event.target.action, {
					method: 'POST',
					body: formData
				});
				const result = await response.json();
				if (result.success) {
					alert('Security settings saved successfully!');
				} else {
					alert('Error: ' + (result.error || 'Failed to save security settings'));
				}
			}
		}`),

		// Page header
		Div(
			H1(Class("text-3xl font-bold text-slate-900 dark:text-white"),
				g.Text("API Key Security")),
			P(Class("mt-2 text-slate-600 dark:text-gray-400"),
				g.Text("Configure security settings for API key authentication")),
		),

		// Security form
		FormEl(
			g.Attr("method", "POST"),
			g.Attr("action", actionURL),
			g.Attr("@submit.prevent", "saveSecurity($event)"),

			Div(Class("bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 space-y-2"),
				// Authentication Methods
				Div(
					H3(Class("text-lg font-semibold text-gray-900 dark:text-white mb-4"),
						g.Text("Authentication Methods")),
					Div(Class("space-y-3"),
						Div(
							Label(Class("flex items-start gap-2 cursor-pointer"),
								Input(
									Type("checkbox"),
									Name("allow_query_param"),
									g.If(e.plugin.config.AllowQueryParam, Checked()),
									Class("mt-1 rounded border-gray-300 dark:border-gray-600 text-violet-600 focus:ring-violet-500"),
								),
								Div(
									Span(Class("text-sm font-medium text-gray-700 dark:text-gray-300"),
										g.Text("Allow Query Parameter")),
									P(Class("text-xs text-gray-500 dark:text-gray-400"),
										g.Text("Allow API keys in URL query parameters (?api_key=xxx). Not recommended for production.")),
								),
							),
						),
					),
				),

				// Rate Limiting
				Div(
					H3(Class("text-lg font-semibold text-gray-900 dark:text-white mb-4"),
						g.Text("Rate Limiting")),
					Div(Class("space-y-3"),
						Div(
							Label(Class("flex items-start gap-2 cursor-pointer"),
								Input(
									Type("checkbox"),
									Name("rate_limiting_enabled"),
									g.If(e.plugin.config.RateLimiting.Enabled, Checked()),
									Class("mt-1 rounded border-gray-300 dark:border-gray-600 text-violet-600 focus:ring-violet-500"),
								),
								Div(
									Span(Class("text-sm font-medium text-gray-700 dark:text-gray-300"),
										g.Text("Enable Rate Limiting")),
									P(Class("text-xs text-gray-500 dark:text-gray-400"),
										g.Text("Enforce per-key rate limits to prevent abuse")),
								),
							),
						),
					),
				),

				// IP Whitelisting
				Div(
					H3(Class("text-lg font-semibold text-gray-900 dark:text-white mb-4"),
						g.Text("IP Whitelisting")),
					Div(Class("space-y-3"),
						Div(
							Label(Class("flex items-start gap-2 cursor-pointer"),
								Input(
									Type("checkbox"),
									Name("ip_whitelisting_enabled"),
									g.If(e.plugin.config.IPWhitelisting.Enabled, Checked()),
									Class("mt-1 rounded border-gray-300 dark:border-gray-600 text-violet-600 focus:ring-violet-500"),
								),
								Div(
									Span(Class("text-sm font-medium text-gray-700 dark:text-gray-300"),
										g.Text("Enable IP Whitelisting")),
									P(Class("text-xs text-gray-500 dark:text-gray-400"),
										g.Text("Restrict API key usage to specific IP addresses")),
								),
							),
						),
					),
				),

				// Security Best Practices
				Div(Class("p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg"),
					Div(Class("flex items-start gap-2"),
						lucide.Info(Class("h-5 w-5 text-blue-600 dark:text-blue-500 mt-0.5")),
						Div(
							P(Class("text-sm font-medium text-blue-800 dark:text-blue-200 mb-2"),
								g.Text("Security Best Practices")),
							Ul(Class("text-sm text-blue-700 dark:text-blue-300 space-y-1 list-disc list-inside"),
								Li(g.Text("Always use HTTPS in production")),
								Li(g.Text("Prefer header-based authentication over query parameters")),
								Li(g.Text("Set appropriate rate limits for your use case")),
								Li(g.Text("Enable IP whitelisting for server-to-server communication")),
								Li(g.Text("Rotate keys regularly and after suspected compromise")),
								Li(g.Text("Monitor API key usage and audit logs")),
							),
						),
					),
				),
			),

			// Save button
			Div(Class("flex justify-end"),
				Button(
					Type("submit"),
					Class("px-6 py-2 bg-violet-600 hover:bg-violet-700 text-white rounded-lg transition-colors"),
					g.Text("Save Security Settings"),
				),
			),
		),
	)
}

// renderKeyStatsWidget renders the API key statistics widget.
func (e *DashboardExtension) renderKeyStatsWidget(stats KeyStats) g.Node {
	return Div(
		Class("bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6"),

		// Widget header
		Div(Class("flex items-center justify-between mb-4"),
			Div(Class("flex items-center gap-2"),
				lucide.Key(Class("h-5 w-5 text-violet-600")),
				H3(Class("text-lg font-semibold text-gray-900 dark:text-white"),
					g.Text("API Keys")),
			),
		),

		// Stats grid
		Div(Class("space-y-3"),
			e.statRow("Active Keys", strconv.Itoa(stats.TotalActive), "text-green-600 dark:text-green-400"),
			e.statRow("Used (24h)", strconv.Itoa(stats.UsedLast24h), "text-blue-600 dark:text-blue-400"),
			e.statRow("Avg Requests", fmt.Sprintf("%.0f", stats.AvgRequestRate), "text-gray-600 dark:text-gray-400"),
			g.If(stats.ExpiringSoon > 0,
				e.statRow("Expiring Soon", strconv.Itoa(stats.ExpiringSoon), "text-orange-600 dark:text-orange-400"),
			),
		),
	)
}

// statRow renders a single stat row in the widget.
func (e *DashboardExtension) statRow(label, value, colorClass string) g.Node {
	return Div(
		Class("flex items-center justify-between"),
		Span(Class("text-sm text-gray-600 dark:text-gray-400"), g.Text(label)),
		Span(Class("text-sm font-semibold "+colorClass), g.Text(value)),
	)
}

// renderError renders an error message.
func (e *DashboardExtension) renderError(title, message string) g.Node {
	return Div(Class("bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4"),
		Div(Class("flex items-start gap-2"),
			lucide.X(Class("h-5 w-5 text-red-600 dark:text-red-500 mt-0.5")),
			Div(
				H3(Class("text-sm font-medium text-red-800 dark:text-red-200"), g.Text(title)),
				P(Class("text-sm text-red-700 dark:text-red-300 mt-1"), g.Text(message)),
			),
		),
	)
}

// formatTimeAgo formats a time as "X ago" or "in X" for future times.
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	if duration < 0 {
		// Future time
		duration = -duration
		if duration < time.Minute {
			return "in a few seconds"
		} else if duration < time.Hour {
			mins := int(duration.Minutes())
			if mins == 1 {
				return "in 1 minute"
			}

			return fmt.Sprintf("in %d minutes", mins)
		} else if duration < 24*time.Hour {
			hours := int(duration.Hours())
			if hours == 1 {
				return "in 1 hour"
			}

			return fmt.Sprintf("in %d hours", hours)
		} else {
			days := int(duration.Hours() / 24)
			if days == 1 {
				return "in 1 day"
			}

			return fmt.Sprintf("in %d days", days)
		}
	}

	// Past time
	if duration < time.Minute {
		return "a few seconds ago"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)
	} else {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)
	}
}
