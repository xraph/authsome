package pages

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/plugins/secrets/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// SecretsListPage renders the secrets list page.
func SecretsListPage(
	currentApp *app.App,
	basePath string,
	secrets []*core.SecretDTO,
	pag *pagination.Pagination,
	query *core.ListSecretsQuery,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Build stats node only if pagination is available
	var statsNode g.Node
	if pag != nil {
		statsNode = statsCards(secrets, pag.TotalItems)
	}

	return Div(
		Class("space-y-6"),

		// Header
		Div(
			Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
			Div(
				H1(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text("Secrets Manager"),
				),
				P(
					Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Manage encrypted secrets and configuration for your application"),
				),
			),
			A(
				Href(appBase+"/secrets/create"),
				Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
				lucide.Plus(Class("size-4")),
				g.Text("Create Secret"),
			),
		),

		// Search and filters
		searchAndFilters(appBase, query),

		// Stats cards
		statsNode,

		// Secrets table/tree
		secretsTable(currentApp, basePath, secrets, pag, query),
	)
}

// searchAndFilters renders the search and filter controls.
func searchAndFilters(appBase string, query *core.ListSecretsQuery) g.Node {
	return Div(
		Class("flex flex-wrap gap-3"),
		// Search input
		FormEl(
			Method("GET"),
			Action(appBase+"/secrets"),
			Class("flex-1 min-w-[200px]"),
			Div(
				Class("relative"),
				Div(
					Class("absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none"),
					lucide.Search(Class("size-4 text-slate-400")),
				),
				Input(
					Type("text"),
					Name("search"),
					Value(query.Search),
					Placeholder("Search secrets by path or description..."),
					Class("block w-full pl-10 pr-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
				),
			),
		),

		// Prefix filter
		Div(
			Class("min-w-[150px]"),
			Select(
				Name("prefix"),
				Class("block w-full py-2 px-3 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
				Aria("label", "Filter by prefix"),
				Option(
					Value(""),
					g.If(query.Prefix == "", Selected()),
					g.Text("All paths"),
				),
				Option(
					Value("database"),
					g.If(query.Prefix == "database", Selected()),
					g.Text("Database"),
				),
				Option(
					Value("api"),
					g.If(query.Prefix == "api", Selected()),
					g.Text("API Keys"),
				),
				Option(
					Value("config"),
					g.If(query.Prefix == "config", Selected()),
					g.Text("Configuration"),
				),
			),
		),

		// View toggle (list/tree)
		Div(
			Class("flex rounded-lg border border-slate-300 dark:border-gray-700 overflow-hidden"),
			Button(
				Type("button"),
				Class("px-3 py-2 text-sm bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"),
				Aria("label", "List view"),
				lucide.List(Class("size-4")),
			),
			Button(
				Type("button"),
				Class("px-3 py-2 text-sm text-slate-600 hover:bg-slate-100 dark:text-gray-400 dark:hover:bg-gray-800"),
				Aria("label", "Tree view"),
				lucide.FolderTree(Class("size-4")),
			),
		),
	)
}

// statsCards renders statistics cards.
func statsCards(secrets []*core.SecretDTO, total int) g.Node {
	// Count by type
	typeCounts := make(map[string]int)
	expiringCount := 0
	now := time.Now()
	thirtyDays := now.Add(30 * 24 * time.Hour)

	for _, s := range secrets {
		typeCounts[s.ValueType]++
		if s.ExpiresAt != nil && s.ExpiresAt.Before(thirtyDays) && s.ExpiresAt.After(now) {
			expiringCount++
		}
	}

	return Div(
		Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
		statCard("Total Secrets", strconv.Itoa(total), lucide.KeyRound(Class("size-5 text-violet-600"))),
		statCard("Plain Text", strconv.Itoa(typeCounts["plain"]), lucide.Type(Class("size-5 text-blue-600"))),
		statCard("Structured", strconv.Itoa(typeCounts["json"]+typeCounts["yaml"]), lucide.Braces(Class("size-5 text-green-600"))),
		statCard("Expiring Soon", strconv.Itoa(expiringCount), lucide.Clock(Class("size-5 text-orange-600"))),
	)
}

func statCard(title, value string, icon g.Node) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm font-medium text-slate-600 dark:text-gray-400"), g.Text(title)),
				Div(Class("text-2xl font-bold text-slate-900 dark:text-white mt-1"), g.Text(value)),
			),
			Div(
				Class("rounded-full bg-slate-100 p-2 dark:bg-gray-800"),
				icon,
			),
		),
	)
}

// secretsTable renders the secrets table.
func secretsTable(
	currentApp *app.App,
	basePath string,
	secrets []*core.SecretDTO,
	pag *pagination.Pagination,
	query *core.ListSecretsQuery,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	if len(secrets) == 0 {
		return emptyState(appBase)
	}

	rows := make([]g.Node, len(secrets))
	for i, secret := range secrets {
		rows[i] = secretRow(currentApp, basePath, secret)
	}

	// Build pagination node only if needed
	var pagNode g.Node
	if pag != nil && pag.TotalPages > 1 {
		pagNode = paginationControls(appBase+"/secrets", pag.Page, pag.TotalPages)
	}

	return Div(
		Class("bg-white rounded-lg border border-slate-200 shadow-sm overflow-hidden dark:bg-gray-900 dark:border-gray-800"),

		// Table
		Div(
			Class("overflow-x-auto"),
			Table(
				Class("min-w-full divide-y divide-slate-200 dark:divide-gray-800"),
				THead(
					Class("bg-slate-50 dark:bg-gray-800/50"),
					Tr(
						Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider dark:text-gray-400"), g.Text("Path")),
						Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider dark:text-gray-400"), g.Text("Type")),
						Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider dark:text-gray-400"), g.Text("Version")),
						Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider dark:text-gray-400"), g.Text("Updated")),
						Th(Class("px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider dark:text-gray-400"), g.Text("Status")),
						Th(Class("px-4 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider dark:text-gray-400"), g.Text("Actions")),
					),
				),
				TBody(
					Class("divide-y divide-slate-200 dark:divide-gray-800"),
					g.Group(rows),
				),
			),
		),

		// Pagination
		pagNode,
	)
}

func secretRow(currentApp *app.App, basePath string, secret *core.SecretDTO) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Determine status
	status := "active"

	if secret.HasExpiry && secret.ExpiresAt != nil {
		if secret.ExpiresAt.Before(time.Now()) {
			status = "expired"
		} else if secret.ExpiresAt.Before(time.Now().Add(30 * 24 * time.Hour)) {
			status = "expiring"
		}
	}

	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),

		// Path
		Td(
			Class("px-4 py-3"),
			A(
				Href(appBase+"/secrets/"+secret.ID),
				Class("group"),
				Div(
					Class("flex items-center gap-2"),
					lucide.Key(Class("size-4 text-slate-400 group-hover:text-violet-600 transition-colors")),
					Div(
						Div(
							Class("text-sm font-medium text-slate-900 dark:text-white group-hover:text-violet-600 transition-colors"),
							g.Text(secret.Path),
						),
						g.If(secret.Description != "", func() g.Node {
							desc := secret.Description
							if len(desc) > 50 {
								desc = desc[:50] + "..."
							}

							return Div(
								Class("text-xs text-slate-500 dark:text-gray-400"),
								g.Text(desc),
							)
						}()),
					),
				),
			),
		),

		// Type badge
		Td(
			Class("px-4 py-3"),
			valueTypeBadge(secret.ValueType),
		),

		// Version
		Td(
			Class("px-4 py-3"),
			Span(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text(fmt.Sprintf("v%d", secret.Version)),
			),
		),

		// Updated
		Td(
			Class("px-4 py-3"),
			Span(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				Title(secret.UpdatedAt.Format(time.RFC3339)),
				g.Text(timeAgo(secret.UpdatedAt)),
			),
		),

		// Status
		Td(
			Class("px-4 py-3"),
			statusBadge(status),
		),

		// Actions
		Td(
			Class("px-4 py-3 text-right"),
			Div(
				Class("flex items-center justify-end gap-2"),
				A(
					Href(appBase+"/secrets/"+secret.ID),
					Class("p-1.5 rounded-lg text-slate-400 hover:text-violet-600 hover:bg-violet-50 dark:hover:bg-violet-900/20 transition-colors"),
					Title("View details"),
					lucide.Eye(Class("size-4")),
				),
				A(
					Href(appBase+"/secrets/"+secret.ID+"/edit"),
					Class("p-1.5 rounded-lg text-slate-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 transition-colors"),
					Title("Edit"),
					lucide.Pencil(Class("size-4")),
				),
				A(
					Href(appBase+"/secrets/"+secret.ID+"/history"),
					Class("p-1.5 rounded-lg text-slate-400 hover:text-green-600 hover:bg-green-50 dark:hover:bg-green-900/20 transition-colors"),
					Title("Version history"),
					lucide.History(Class("size-4")),
				),
			),
		),
	)
}

func emptyState(appBase string) g.Node {
	return Div(
		Class("bg-white rounded-lg border border-slate-200 p-12 text-center dark:bg-gray-900 dark:border-gray-800"),
		Div(
			Class("mx-auto w-16 h-16 rounded-full bg-slate-100 dark:bg-gray-800 flex items-center justify-center mb-4"),
			lucide.KeyRound(Class("size-8 text-slate-400")),
		),
		H3(
			Class("text-lg font-medium text-slate-900 dark:text-white"),
			g.Text("No secrets found"),
		),
		P(
			Class("mt-2 text-sm text-slate-600 dark:text-gray-400 max-w-sm mx-auto"),
			g.Text("Get started by creating your first encrypted secret to store sensitive configuration."),
		),
		A(
			Href(appBase+"/secrets/create"),
			Class("inline-flex items-center gap-2 mt-6 px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
			lucide.Plus(Class("size-4")),
			g.Text("Create Secret"),
		),
	)
}

func paginationControls(baseURL string, currentPage, totalPages int) g.Node {
	items := make([]g.Node, 0)

	// Previous
	if currentPage > 1 {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Previous"),
		))
	}

	// Page numbers
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			items = append(items, Span(
				Class("px-3 py-2 text-sm font-medium text-white bg-violet-600 border border-violet-600 rounded-md"),
				g.Text(strconv.Itoa(i)),
			))
		} else if i == 1 || i == totalPages || (i >= currentPage-1 && i <= currentPage+1) {
			items = append(items, A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
				Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
				g.Text(strconv.Itoa(i)),
			))
		} else if i == currentPage-2 || i == currentPage+2 {
			items = append(items, Span(Class("px-2 py-2 text-slate-400"), g.Text("...")))
		}
	}

	// Next
	if currentPage < totalPages {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Next"),
		))
	}

	return Div(
		Class("flex items-center justify-center gap-2 px-4 py-4 border-t border-slate-200 dark:border-gray-800"),
		g.Group(items),
	)
}

// Helper functions.
func valueTypeBadge(valueType string) g.Node {
	var classes, icon string

	switch valueType {
	case "json":
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
		icon = "{}"
	case "yaml":
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400"
		icon = "---"
	case "binary":
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400"
		icon = "01"
	default:
		classes = "inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700 dark:bg-gray-700 dark:text-gray-300"
		icon = "Aa"
	}

	return Span(
		Class(classes),
		Span(Class("font-mono"), g.Text(icon)),
		g.Text(valueType),
	)
}

func statusBadge(status string) g.Node {
	var classes string

	switch status {
	case "active":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
	case "expired":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
	case "expiring":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
	default:
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300"
	}

	return Span(Class(classes), g.Text(status))
}

func timeAgo(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())

		return fmt.Sprintf("%dm ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())

		return fmt.Sprintf("%dh ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)

		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// TreeNode represents a node in the secrets tree.
type TreeNode struct {
	Name     string
	Path     string
	IsSecret bool
	Secret   *core.SecretDTO
	Children []*TreeNode
}

// SecretsTreeView renders a tree view of secrets.
func SecretsTreeView(
	currentApp *app.App,
	basePath string,
	tree []*core.SecretTreeNode,
) g.Node {
	return Div(
		Class("bg-white rounded-lg border border-slate-200 dark:bg-gray-900 dark:border-gray-800 overflow-hidden"),
		Div(
			Class("p-4 border-b border-slate-200 dark:border-gray-800"),
			H3(
				Class("text-sm font-medium text-slate-900 dark:text-white"),
				g.Text("Secrets Tree"),
			),
		),
		Div(
			Class("p-4"),
			g.Group(renderTreeNodes(currentApp, basePath, tree, 0)),
		),
	)
}

func renderTreeNodes(currentApp *app.App, basePath string, nodes []*core.SecretTreeNode, depth int) []g.Node {
	result := make([]g.Node, 0, len(nodes))
	appBase := basePath + "/app/" + currentApp.ID.String()

	for _, node := range nodes {
		indent := fmt.Sprintf("padding-left: %dpx", depth*20)

		if node.IsSecret {
			result = append(result, Div(
				Class("group py-1"),
				StyleAttr(indent),
				A(
					Href(appBase+"/secrets/"+node.Secret.ID),
					Class("flex items-center gap-2 text-sm text-slate-700 hover:text-violet-600 dark:text-gray-300"),
					lucide.Key(Class("size-4 text-slate-400 group-hover:text-violet-600")),
					Span(g.Text(node.Name)),
					valueTypeBadge(node.Secret.ValueType),
				),
			))
		} else {
			result = append(result, Div(
				Class("py-1"),
				StyleAttr(indent),
				Div(
					Class("flex items-center gap-2 text-sm font-medium text-slate-900 dark:text-white"),
					lucide.Folder(Class("size-4 text-amber-500")),
					Span(g.Text(node.Name)),
				),
			))
			if len(node.Children) > 0 {
				result = append(result, g.Group(renderTreeNodes(currentApp, basePath, node.Children, depth+1)))
			}
		}
	}

	return result
}

// SecretsPath renders breadcrumb-style path navigation.
func SecretsPath(path string) g.Node {
	parts := strings.Split(path, "/")
	items := make([]g.Node, 0, len(parts)*2-1)

	for i, part := range parts {
		if i > 0 {
			items = append(items, Span(
				Class("mx-1 text-slate-400"),
				g.Text("/"),
			))
		}

		isLast := i == len(parts)-1

		classes := "text-slate-600 dark:text-gray-400"
		if isLast {
			classes = "text-slate-900 font-medium dark:text-white"
		}

		items = append(items, Span(
			Class(classes),
			g.Text(part),
		))
	}

	return Div(
		Class("flex items-center text-sm font-mono"),
		g.Group(items),
	)
}
