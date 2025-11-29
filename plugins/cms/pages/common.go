// Package pages provides gomponent-based page templates for the CMS dashboard.
package pages

import (
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// =============================================================================
// Common Components
// =============================================================================

// PageHeader renders a standard page header with title, description, and optional actions
func PageHeader(title, description string, actions ...g.Node) g.Node {
	return Div(
		Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between mb-6"),
		Div(
			H1(
				Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text(title),
			),
			g.If(description != "", func() g.Node {
				return P(
					Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text(description),
				)
			}()),
		),
		g.If(len(actions) > 0, func() g.Node {
			return Div(
				Class("flex items-center gap-2"),
				g.Group(actions),
			)
		}()),
	)
}

// PrimaryButton creates a primary action button
func PrimaryButton(href, text string, icon g.Node) g.Node {
	return A(
		Href(href),
		Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg hover:bg-violet-700 transition-colors"),
		g.If(icon != nil, icon),
		g.Text(text),
	)
}

// SecondaryButton creates a secondary action button
func SecondaryButton(href, text string, icon g.Node) g.Node {
	return A(
		Href(href),
		Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700 transition-colors"),
		g.If(icon != nil, icon),
		g.Text(text),
	)
}

// DangerButton creates a danger/delete action button
func DangerButton(href, text string, icon g.Node) g.Node {
	return A(
		Href(href),
		Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"),
		g.If(icon != nil, icon),
		g.Text(text),
	)
}

// Card creates a basic card container
func Card(children ...g.Node) g.Node {
	return Div(
		Class("bg-white rounded-lg border border-slate-200 shadow-sm dark:bg-gray-900 dark:border-gray-800"),
		g.Group(children),
	)
}

// CardWithHeader creates a card with a header section
func CardWithHeader(headerTitle string, headerActions []g.Node, body ...g.Node) g.Node {
	return Card(
		Div(
			Class("flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-gray-800"),
			H2(
				Class("text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text(headerTitle),
			),
			g.If(len(headerActions) > 0, func() g.Node {
				return Div(
					Class("flex items-center gap-2"),
					g.Group(headerActions),
				)
			}()),
		),
		Div(
			Class("px-6 py-4"),
			g.Group(body),
		),
	)
}

// StatCard creates a statistics card
func StatCard(title, value string, icon g.Node, colorClass string) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-4 dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm font-medium text-slate-600 dark:text-gray-400"), g.Text(title)),
				Div(Class("text-2xl font-bold text-slate-900 dark:text-white mt-1"), g.Text(value)),
			),
			Div(
				Class("rounded-full bg-slate-100 p-2 dark:bg-gray-800 "+colorClass),
				icon,
			),
		),
	)
}

// Badge creates a status badge
func Badge(text, colorClass string) g.Node {
	return Span(
		Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium "+colorClass),
		g.Text(text),
	)
}

// StatusBadge creates a status-specific badge
func StatusBadge(status string) g.Node {
	switch status {
	case "published":
		return Badge("Published", "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400")
	case "draft":
		return Badge("Draft", "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400")
	case "archived":
		return Badge("Archived", "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400")
	case "scheduled":
		return Badge("Scheduled", "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400")
	default:
		return Badge(status, "bg-slate-100 text-slate-800 dark:bg-slate-900/30 dark:text-slate-400")
	}
}

// EmptyState creates an empty state message
func EmptyState(icon g.Node, title, description, actionText, actionHref string) g.Node {
	return Div(
		Class("flex flex-col items-center justify-center py-12 text-center"),
		Div(
			Class("rounded-full bg-slate-100 p-4 mb-4 dark:bg-gray-800"),
			icon,
		),
		H3(
			Class("text-lg font-medium text-slate-900 dark:text-white mb-1"),
			g.Text(title),
		),
		P(
			Class("text-sm text-slate-600 dark:text-gray-400 mb-4 max-w-sm"),
			g.Text(description),
		),
		g.If(actionText != "" && actionHref != "", func() g.Node {
			return PrimaryButton(actionHref, actionText, lucide.Plus(Class("size-4")))
		}()),
	)
}

// Breadcrumbs creates a breadcrumb navigation
func Breadcrumbs(items ...BreadcrumbItem) g.Node {
	nodes := make([]g.Node, 0)
	for i, item := range items {
		if i > 0 {
			nodes = append(nodes, Span(
				Class("text-slate-400 dark:text-gray-600 mx-2"),
				g.Text("/"),
			))
		}
		if item.Href != "" {
			nodes = append(nodes, A(
				Href(item.Href),
				Class("text-slate-600 hover:text-violet-600 dark:text-gray-400 dark:hover:text-violet-400 transition-colors"),
				g.Text(item.Label),
			))
		} else {
			nodes = append(nodes, Span(
				Class("text-slate-900 dark:text-white font-medium"),
				g.Text(item.Label),
			))
		}
	}
	return Nav(
		Class("flex items-center text-sm mb-4"),
		g.Group(nodes),
	)
}

// BreadcrumbItem represents a breadcrumb item
type BreadcrumbItem struct {
	Label string
	Href  string
}

// SearchInput creates a search input field
func SearchInput(placeholder, value, formAction string) g.Node {
	return FormEl(
		Method("GET"),
		Action(formAction),
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
				Value(value),
				Placeholder(placeholder),
				Class("block w-full pl-10 pr-4 py-2 text-sm border border-slate-300 rounded-lg bg-white dark:bg-gray-800 dark:border-gray-700 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"),
			),
		),
	)
}

// FormatTime formats a time for display
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("Jan 2, 2006 3:04 PM")
}

// FormatTimeAgo formats a time as relative time
func FormatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// Pagination renders pagination controls
func Pagination(currentPage, totalPages int, baseURL string) g.Node {
	if totalPages <= 1 {
		return nil
	}

	pages := make([]g.Node, 0)

	// Previous button
	if currentPage > 1 {
		pages = append(pages, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)),
			Class("px-3 py-2 text-sm font-medium text-slate-600 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700"),
			lucide.ChevronLeft(Class("size-4")),
		))
	}

	// Page numbers
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			pages = append(pages, Span(
				Class("px-3 py-2 text-sm font-medium text-white bg-violet-600 rounded-lg"),
				g.Text(fmt.Sprintf("%d", i)),
			))
		} else if i == 1 || i == totalPages || (i >= currentPage-1 && i <= currentPage+1) {
			pages = append(pages, A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
				Class("px-3 py-2 text-sm font-medium text-slate-600 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700"),
				g.Text(fmt.Sprintf("%d", i)),
			))
		} else if i == currentPage-2 || i == currentPage+2 {
			pages = append(pages, Span(
				Class("px-2 text-slate-400"),
				g.Text("..."),
			))
		}
	}

	// Next button
	if currentPage < totalPages {
		pages = append(pages, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)),
			Class("px-3 py-2 text-sm font-medium text-slate-600 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700"),
			lucide.ChevronRight(Class("size-4")),
		))
	}

	return Div(
		Class("flex items-center justify-center gap-2 mt-6"),
		g.Group(pages),
	)
}

// DataTable renders a data table
func DataTable(headers []string, rows []g.Node) g.Node {
	headerCells := make([]g.Node, len(headers))
	for i, h := range headers {
		headerCells[i] = Th(
			Class("px-4 py-3 text-left text-xs font-medium text-slate-500 dark:text-gray-400 uppercase tracking-wider"),
			g.Text(h),
		)
	}

	return Div(
		Class("overflow-x-auto"),
		Table(
			Class("min-w-full divide-y divide-slate-200 dark:divide-gray-800"),
			THead(
				Class("bg-slate-50 dark:bg-gray-800/50"),
				Tr(g.Group(headerCells)),
			),
			TBody(
				Class("bg-white divide-y divide-slate-200 dark:bg-gray-900 dark:divide-gray-800"),
				g.Group(rows),
			),
		),
	)
}

// TableRow creates a table row
func TableRow(cells ...g.Node) g.Node {
	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),
		g.Group(cells),
	)
}

// TableCell creates a table cell
func TableCell(content g.Node) g.Node {
	return Td(
		Class("px-4 py-4 whitespace-nowrap text-sm text-slate-900 dark:text-white"),
		content,
	)
}

// TableCellSecondary creates a secondary table cell with muted text
func TableCellSecondary(content g.Node) g.Node {
	return Td(
		Class("px-4 py-4 whitespace-nowrap text-sm text-slate-600 dark:text-gray-400"),
		content,
	)
}

// TableCellActions creates a table cell with action buttons
func TableCellActions(actions ...g.Node) g.Node {
	return Td(
		Class("px-4 py-4 whitespace-nowrap text-right text-sm"),
		Div(
			Class("flex items-center justify-end gap-2"),
			g.Group(actions),
		),
	)
}

// IconButton creates a small icon button
func IconButton(href string, icon g.Node, title, colorClass string) g.Node {
	return A(
		Href(href),
		Title(title),
		Class("p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-gray-800 transition-colors "+colorClass),
		icon,
	)
}

// ConfirmButton creates a button that requires confirmation
func ConfirmButton(formAction, method, text, confirmMessage, colorClass string, icon g.Node) g.Node {
	return FormEl(
		Action(formAction),
		Method(method),
		g.Attr("onsubmit", fmt.Sprintf("return confirm('%s')", confirmMessage)),
		Class("inline"),
		Button(
			Type("submit"),
			Class("inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium rounded-lg transition-colors "+colorClass),
			g.If(icon != nil, icon),
			g.Text(text),
		),
	)
}

