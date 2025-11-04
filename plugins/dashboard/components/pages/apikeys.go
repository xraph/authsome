package pages

import (
	"fmt"

	. "maragu.dev/gomponents/html"
	g "maragu.dev/gomponents"
	lucide "github.com/eduardolat/gomponents-lucide"
)

// APIKeyData represents an API key for display
type APIKeyData struct {
	ID          string
	Name        string
	Key         string // Only shown once after creation
	Prefix      string // Always shown (first 8 chars)
	Scopes      []string
	RateLimit   int
	ExpiresAt   string
	LastUsedAt  string
	CreatedAt   string
	IsActive    bool
}

// APIKeysTabPageData contains data for the API keys management tab
type APIKeysTabPageData struct {
	APIKeys        []APIKeyData
	Organizations  []OrganizationOption // For organization selector
	IsSaaSMode     bool
	CanCreateKeys  bool
	CSRFToken      string
}

// OrganizationOption represents an organization for selection
type OrganizationOption struct {
	ID   string
	Name string
}

// apiKeysTabContent renders the complete API keys management interface
func apiKeysTabContent(data APIKeysTabPageData) g.Node {
	return Div(Class("space-y-6"),
		// Header with actions
		Div(Class("flex justify-between items-center"),
			Div(
				H3(Class("text-lg font-semibold text-gray-900 dark:text-white"), g.Text("API Keys")),
				P(Class("text-sm text-gray-600 dark:text-gray-400 mt-1"),
					g.Text("Manage API keys for programmatic access to your application"),
				),
			),
			g.If(data.CanCreateKeys,
				Button(
					Type("button"),
					Class("inline-flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"),
					g.Attr("@click", "showCreateModal = true"),
					lucide.Plus(Class("h-4 w-4")),
					g.Text("Create API Key"),
				),
			),
		),

		// API Keys List
		g.If(len(data.APIKeys) > 0,
			apiKeysTable(data.APIKeys),
		),

		// Empty State
		g.If(len(data.APIKeys) == 0,
			apiKeysEmptyState(),
		),

		// Create Modal
		createAPIKeyModal(data),

		// View Key Modal (for newly created keys)
		viewKeyModal(),

		// Revoke Confirmation Modal
		revokeKeyModal(data.CSRFToken),
	)
}

// apiKeysTable renders the table of API keys
func apiKeysTable(keys []APIKeyData) g.Node {
	return Div(Class("bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden"),
		Div(Class("overflow-x-auto"),
			Table(Class("w-full"),
				THead(Class("bg-gray-50 dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700"),
					Tr(
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Name")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Key Prefix")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Scopes")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Rate Limit")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Last Used")),
						Th(Class("px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Status")),
						Th(Class("px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"), g.Text("Actions")),
					),
				),
				TBody(Class("bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:border-gray-700"),
					g.Group(apiKeysTableRows(keys)),
				),
			),
		),
	)
}

// apiKeysTableRows generates table rows for API keys
func apiKeysTableRows(keys []APIKeyData) []g.Node {
	rows := make([]g.Node, len(keys))
	for i, key := range keys {
		rows[i] = apiKeyRow(key)
	}
	return rows
}

// apiKeyRow renders a single API key row
func apiKeyRow(key APIKeyData) g.Node {
	statusClass := "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
	statusText := "Active"
	if !key.IsActive {
		statusClass = "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
		statusText = "Revoked"
	}

	return Tr(Class("hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"),
		// Name
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Div(Class("text-sm font-medium text-gray-900 dark:text-white"), g.Text(key.Name)),
			g.If(key.ExpiresAt != "",
				Div(Class("text-xs text-gray-500 dark:text-gray-400 mt-1"),
					lucide.Clock(Class("inline h-3 w-3 mr-1")),
					g.Text(fmt.Sprintf("Expires %s", key.ExpiresAt)),
				),
			),
		),

		// Key Prefix
		Td(Class("px-6 py-4 whitespace-nowrap"),
			Code(Class("px-2 py-1 bg-gray-100 dark:bg-gray-900 text-sm text-gray-800 dark:text-gray-200 rounded font-mono"),
				g.Text(key.Prefix+"..."),
			),
		),

		// Scopes
		Td(Class("px-6 py-4"),
			Div(Class("flex flex-wrap gap-1"),
				g.Group(renderScopes(key.Scopes)),
			),
		),

		// Rate Limit
		Td(Class("px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400"),
			g.If(key.RateLimit > 0,
				g.Text(fmt.Sprintf("%d req/hr", key.RateLimit)),
			),
			g.If(key.RateLimit == 0,
				g.Text("Unlimited"),
			),
		),

		// Last Used
		Td(Class("px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400"),
			g.If(key.LastUsedAt != "",
				g.Text(key.LastUsedAt),
			),
			g.If(key.LastUsedAt == "",
				g.Text("Never"),
			),
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
				g.If(key.IsActive,
					Button(
						Type("button"),
						Class("text-red-600 hover:text-red-900 dark:text-red-400 dark:hover:text-red-300"),
						g.Attr("@click", fmt.Sprintf("showRevokeModal('%s', '%s')", key.ID, key.Name)),
						g.Attr("title", "Revoke Key"),
						lucide.Ban(Class("h-4 w-4")),
					),
				),
			),
		),
	)
}

// renderScopes renders scope badges
func renderScopes(scopes []string) []g.Node {
	if len(scopes) == 0 {
		return []g.Node{
			Span(Class("px-2 py-1 bg-gray-100 dark:bg-gray-700 text-xs text-gray-600 dark:text-gray-400 rounded"), g.Text("No scopes")),
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

// apiKeysEmptyState renders empty state
func apiKeysEmptyState() g.Node {
	return Div(Class("text-center py-12 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"),
		Div(Class("mx-auto h-16 w-16 text-gray-400 dark:text-gray-600 mb-4"),
			lucide.Key(Class("h-16 w-16")),
		),
		H3(Class("text-lg font-medium text-gray-900 dark:text-white mb-2"), g.Text("No API Keys")),
		P(Class("text-gray-600 dark:text-gray-400 mb-6"), g.Text("Create your first API key to enable programmatic access")),
		Button(
			Type("button"),
			Class("inline-flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"),
			g.Attr("@click", "showCreateModal = true"),
			lucide.Plus(Class("h-4 w-4")),
			g.Text("Create API Key"),
		),
	)
}

// createAPIKeyModal renders the create API key modal
func createAPIKeyModal(data APIKeysTabPageData) g.Node {
	return Div(
		g.Attr("x-show", "showCreateModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4"),
		g.Attr("@click.self", "showCreateModal = false"),

		Div(Class("bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full"),
			// Modal Header
			Div(Class("flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700"),
				H3(Class("text-lg font-semibold text-gray-900 dark:text-white"), g.Text("Create API Key")),
				Button(
					Type("button"),
					Class("text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"),
					g.Attr("@click", "showCreateModal = false"),
					lucide.X(Class("h-5 w-5")),
				),
			),

			// Modal Body
			FormEl(
				g.Attr("method", "POST"),
				g.Attr("action", "/dashboard/api-keys/create"),
				g.Attr("@submit.prevent", "createAPIKey($event)"),

				Div(Class("p-6 space-y-4"),
					// CSRF Token
					Input(Type("hidden"), Name("csrf_token"), Value(data.CSRFToken)),

					// Name
					Div(
						Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Key Name"), g.Attr("for", "key_name")),
						Input(
							Type("text"),
							ID("key_name"),
							Name("name"),
							Required(),
							Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"),
							g.Attr("placeholder", "My API Key"),
						),
					),

					// Organization (only in SaaS mode)
					g.If(data.IsSaaSMode && len(data.Organizations) > 0,
						Div(
							Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Organization"), g.Attr("for", "org_id")),
							Select(
								ID("org_id"),
								Name("org_id"),
								Required(),
								Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"),
								g.Group(organizationOptions(data.Organizations)),
							),
						),
					),

					// Scopes
					Div(
						Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Scopes (optional)")),
						Input(
							Type("text"),
							ID("scopes"),
							Name("scopes"),
							Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"),
							g.Attr("placeholder", "read, write, admin"),
						),
						P(Class("text-xs text-gray-500 dark:text-gray-400 mt-1"), g.Text("Comma-separated list of scopes")),
					),

					// Rate Limit
					Div(
						Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Rate Limit (requests/hour)"), g.Attr("for", "rate_limit")),
						Input(
							Type("number"),
							ID("rate_limit"),
							Name("rate_limit"),
							Value("1000"),
							g.Attr("min", "0"),
							Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"),
						),
						P(Class("text-xs text-gray-500 dark:text-gray-400 mt-1"), g.Text("0 for unlimited (not recommended)")),
					),

					// Expiry (optional)
					Div(
						Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Expires In (days, optional)"), g.Attr("for", "expires_in")),
						Input(
							Type("number"),
							ID("expires_in"),
							Name("expires_in"),
							g.Attr("min", "1"),
							g.Attr("max", "365"),
							Class("w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"),
							g.Attr("placeholder", "Leave empty for no expiration"),
						),
					),
				),

				// Modal Footer
				Div(Class("flex justify-end gap-3 px-6 py-4 bg-gray-50 dark:bg-gray-900 rounded-b-lg"),
					Button(
						Type("button"),
						Class("px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"),
						g.Attr("@click", "showCreateModal = false"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						Class("px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"),
						g.Text("Create Key"),
					),
				),
			),
		),
	)
}

// viewKeyModal renders the modal to display newly created API key
func viewKeyModal() g.Node {
	return Div(
		g.Attr("x-show", "showViewKeyModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4"),

		Div(Class("bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full"),
			// Modal Header
			Div(Class("flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700"),
				Div(Class("flex items-center gap-2"),
					lucide.Check(Class("h-6 w-6 text-green-600")),
					H3(Class("text-lg font-semibold text-gray-900 dark:text-white"), g.Text("API Key Created")),
				),
			),

			// Modal Body
			Div(Class("p-6 space-y-4"),
				Div(Class("p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg"),
					Div(Class("flex items-start gap-2"),
						lucide.Info(Class("h-5 w-5 text-yellow-600 dark:text-yellow-500 mt-0.5")),
						P(Class("text-sm text-yellow-800 dark:text-yellow-200"),
							g.Text("Save this key securely! For security reasons, we cannot show it again."),
						),
					),
				),

				Div(
					Label(Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2"), g.Text("Your API Key")),
					Div(Class("relative"),
						Input(
							Type("text"),
							g.Attr("x-model", "newAPIKey"),
							g.Attr("readonly", ""),
							Class("w-full px-4 py-2 pr-24 border border-gray-300 dark:border-gray-600 rounded-lg bg-gray-50 dark:bg-gray-900 text-gray-900 dark:text-white font-mono text-sm"),
						),
						Button(
							Type("button"),
							Class("absolute right-2 top-1/2 -translate-y-1/2 px-3 py-1 bg-blue-600 hover:bg-blue-700 text-white text-xs rounded transition-colors"),
							g.Attr("@click", "copyToClipboard(newAPIKey)"),
							g.Text("Copy"),
						),
					),
				),
			),

			// Modal Footer
			Div(Class("flex justify-end px-6 py-4 bg-gray-50 dark:bg-gray-900 rounded-b-lg"),
				Button(
					Type("button"),
					Class("px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"),
					g.Attr("@click", "showViewKeyModal = false; newAPIKey = ''"),
					g.Text("I've Saved The Key"),
				),
			),
		),
	)
}

// revokeKeyModal renders the confirmation modal for revoking an API key
func revokeKeyModal(csrfToken string) g.Node {
	return Div(
		g.Attr("x-show", "showRevokeModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4"),
		g.Attr("@click.self", "showRevokeModal = false"),

		Div(Class("bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full"),
			// Modal Header
			Div(Class("flex items-center justify-between p-6 border-b border-gray-200 dark:border-gray-700"),
				H3(Class("text-lg font-semibold text-gray-900 dark:text-white"), g.Text("Revoke API Key")),
				Button(
					Type("button"),
					Class("text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"),
					g.Attr("@click", "showRevokeModal = false"),
					lucide.X(Class("h-5 w-5")),
				),
			),

			// Modal Body
			Div(Class("p-6"),
				P(Class("text-gray-600 dark:text-gray-400"),
					g.Text("Are you sure you want to revoke "),
					Strong(g.Attr("x-text", "revokeKeyName")),
					g.Text("? This action cannot be undone and all applications using this key will lose access immediately."),
				),
			),

			// Modal Footer
			FormEl(
				g.Attr("method", "POST"),
				g.Attr("action", "/dashboard/api-keys/revoke"),
				g.Attr("@submit.prevent", "revokeAPIKey($event)"),

				Input(Type("hidden"), Name("csrf_token"), Value(csrfToken)),
				Input(Type("hidden"), Name("key_id"), g.Attr("x-model", "revokeKeyID")),

				Div(Class("flex justify-end gap-3 px-6 py-4 bg-gray-50 dark:bg-gray-900 rounded-b-lg"),
					Button(
						Type("button"),
						Class("px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-lg transition-colors"),
						g.Attr("@click", "showRevokeModal = false"),
						g.Text("Cancel"),
					),
					Button(
						Type("submit"),
						Class("px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors"),
						g.Text("Revoke Key"),
					),
				),
			),
		),
	)
}

// organizationOptions generates option elements for organization select
func organizationOptions(orgs []OrganizationOption) []g.Node {
	nodes := make([]g.Node, len(orgs))
	for i, org := range orgs {
		nodes[i] = Option(Value(org.ID), g.Text(org.Name))
	}
	return nodes
}

