// Package pages provides gomponent-based page templates for the CMS dashboard.
package pages

import (
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui"
	"github.com/xraph/forgeui/components/badge"
	"github.com/xraph/forgeui/components/breadcrumb"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/emptystate"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/components/table"
	"github.com/xraph/forgeui/primitives"
)

// =============================================================================
// Common Components
// =============================================================================

// PageHeader renders a standard page header with title, description, and optional actions
func PageHeader(title, description string, actions ...g.Node) g.Node {
	titleSection := primitives.VStack("gap-1",
		H1(Class("text-2xl font-bold"), g.Text(title)),
		g.If(description != "", func() g.Node {
			return P(Class("text-sm text-muted-foreground"), g.Text(description))
		}()),
	)

	if len(actions) > 0 {
		return Div(
			Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between mb-6"),
			titleSection,
			primitives.HStack("gap-2", actions...),
		)
	}
	return Div(Class("mb-6"), titleSection)
}

// PrimaryButton creates a primary action button
func PrimaryButton(href, text string, icon g.Node) g.Node {
	content := []g.Node{g.Text(text)}
	if icon != nil {
		content = []g.Node{icon, g.Text(text)}
	}
	return A(
		Href(href),
		Class("inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium font-semibold transition-all duration-200 outline-none ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 hover:shadow-md h-9 px-4 py-2"),
		g.Group(content),
	)
}

// SecondaryButton creates a secondary action button
func SecondaryButton(href, text string, icon g.Node) g.Node {
	content := []g.Node{g.Text(text)}
	if icon != nil {
		content = []g.Node{icon, g.Text(text)}
	}
	return A(
		Href(href),
		Class("inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium font-semibold transition-all duration-200 outline-none ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-secondary text-secondary-foreground shadow-sm hover:bg-secondary/80 h-9 px-4 py-2"),
		g.Group(content),
	)
}

// DangerButton creates a danger/delete action button
func DangerButton(href, text string, icon g.Node) g.Node {
	content := []g.Node{g.Text(text)}
	if icon != nil {
		content = []g.Node{icon, g.Text(text)}
	}
	return A(
		Href(href),
		Class("inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium font-semibold transition-all duration-200 outline-none ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 bg-destructive text-destructive-foreground shadow-sm hover:bg-destructive/90 hover:shadow-md h-9 px-4 py-2"),
		g.Group(content),
	)
}

// Card creates a basic card container
func Card(children ...g.Node) g.Node {
	return card.Card(children...)
}

// CardWithHeader creates a card with a header section
func CardWithHeader(headerTitle string, headerActions []g.Node, body ...g.Node) g.Node {
	headerContent := []g.Node{card.Title(headerTitle)}
	if len(headerActions) > 0 {
		headerContent = append(headerContent, 
			Div(
				g.Attr("data-slot", "card-action"),
				primitives.HStack("gap-2", headerActions...),
			),
		)
	}
	return card.Card(
		card.Header(headerContent...),
		card.Content(g.Group(body)),
	)
}

// StatCard creates a statistics card
func StatCard(title, value string, icon g.Node, colorClass string) g.Node {
	return card.Card(
		card.Content(
			primitives.HStack("justify-between items-start",
				primitives.VStack("gap-1",
					Div(Class("text-sm font-medium text-muted-foreground"), g.Text(title)),
					Div(Class("text-2xl font-bold"), g.Text(value)),
				),
				Div(Class("rounded-full bg-muted p-2 "+colorClass), icon),
			),
		),
	)
}

// Badge creates a status badge with custom color classes
func Badge(text, colorClass string) g.Node {
	return badge.Badge(text, badge.WithClass(colorClass))
}

// StatusBadge creates a status-specific badge
func StatusBadge(status string) g.Node {
	switch status {
	case "published":
		return badge.Badge("Published", badge.WithClass("bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 border-transparent"))
	case "draft":
		return badge.Badge("Draft", badge.WithClass("bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400 border-transparent"))
	case "archived":
		return badge.Badge("Archived", badge.WithVariant(forgeui.VariantSecondary))
	case "scheduled":
		return badge.Badge("Scheduled", badge.WithClass("bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400 border-transparent"))
	default:
		return badge.Badge(status, badge.WithVariant(forgeui.VariantOutline))
	}
}

// EmptyState creates an empty state message
func EmptyState(icon g.Node, title, description, actionText, actionHref string) g.Node {
	opts := []emptystate.Option{
		emptystate.WithIcon(icon),
		emptystate.WithTitle(title),
		emptystate.WithDescription(description),
	}
	if actionText != "" && actionHref != "" {
		opts = append(opts, emptystate.WithAction(
			PrimaryButton(actionHref, actionText, lucide.Plus(Class("size-4"))),
		))
	}
	return emptystate.EmptyState(opts...)
}

// Breadcrumbs creates a breadcrumb navigation
func Breadcrumbs(items ...BreadcrumbItem) g.Node {
	children := make([]g.Node, len(items))
	for i, item := range items {
		if item.Href != "" && i < len(items)-1 {
			children[i] = breadcrumb.Item(item.Href, g.Text(item.Label))
		} else {
			children[i] = breadcrumb.Page(g.Text(item.Label))
		}
	}
	return breadcrumb.Breadcrumb(children...)
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
		input.InputGroup(nil,
			input.InputLeftAddon(nil, lucide.Search(Class("size-4"))),
			input.Input(
				input.WithType("text"),
				input.WithName("search"),
				input.WithValue(value),
				input.WithPlaceholder(placeholder),
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

	const btnClass = "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors h-9 px-3 border border-input bg-background hover:bg-accent hover:text-accent-foreground"
	const activeClass = "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium h-9 px-3 bg-primary text-primary-foreground"

	pages := make([]g.Node, 0)

	// Previous button
	if currentPage > 1 {
		pages = append(pages, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)),
			Class(btnClass),
			lucide.ChevronLeft(Class("size-4")),
		))
	}

	// Page numbers
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			pages = append(pages, Span(
				Class(activeClass),
				g.Text(fmt.Sprintf("%d", i)),
			))
		} else if i == 1 || i == totalPages || (i >= currentPage-1 && i <= currentPage+1) {
			pages = append(pages, A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
				Class(btnClass),
				g.Text(fmt.Sprintf("%d", i)),
			))
		} else if i == currentPage-2 || i == currentPage+2 {
			pages = append(pages, Span(
				Class("px-2 text-muted-foreground"),
				g.Text("..."),
			))
		}
	}

	// Next button
	if currentPage < totalPages {
		pages = append(pages, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)),
			Class(btnClass),
			lucide.ChevronRight(Class("size-4")),
		))
	}

	return primitives.HStack("gap-2 justify-center mt-6", pages...)
}

// DataTable renders a data table
func DataTable(headers []string, rows []g.Node) g.Node {
	headerCells := make([]g.Node, len(headers))
	for i, h := range headers {
		headerCells[i] = table.TableHeaderCell()(g.Text(h))
	}

	return Div(
		Class("overflow-x-auto"),
		table.Table()(
			table.TableHeader()(
				table.TableRow()(headerCells...),
			),
			table.TableBody()(g.Group(rows)),
		),
	)
}

// TableRow creates a table row
func TableRow(cells ...g.Node) g.Node {
	return table.TableRow()(cells...)
}

// TableCell creates a table cell
func TableCell(content g.Node) g.Node {
	return table.TableCell()(content)
}

// TableCellSecondary creates a secondary table cell with muted text
func TableCellSecondary(content g.Node) g.Node {
	return table.TableCell(table.WithCellClass("text-muted-foreground"))(content)
}

// TableCellActions creates a table cell with action buttons
func TableCellActions(actions ...g.Node) g.Node {
	return table.TableCell(table.WithAlign(table.AlignRight))(
		primitives.HStack("gap-2 justify-end", actions...),
	)
}

// IconButton creates a small icon button
func IconButton(href string, icon g.Node, title, colorClass string) g.Node {
	return A(
		Href(href),
		Title(title),
		Class("inline-flex items-center justify-center rounded-md outline-none ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 hover:bg-accent hover:text-accent-foreground size-9 "+colorClass),
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
