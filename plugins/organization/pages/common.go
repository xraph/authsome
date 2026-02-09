// Package pages provides ForgeUI-based page templates for the organization plugin dashboard.
package pages

import (
	"fmt"
	"strconv"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui"
	"github.com/xraph/forgeui/components/badge"
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/emptystate"
	"github.com/xraph/forgeui/components/input"
	"github.com/xraph/forgeui/components/table"
	"github.com/xraph/forgeui/primitives"
)

// =============================================================================
// Common Components
// =============================================================================

// PageHeader renders a standard page header with title, description, and optional actions.
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

// PrimaryButton creates a primary action button using ForgeUI.
func PrimaryButton(href, text string, icon g.Node) g.Node {
	content := Div(g.Text(text))
	if icon != nil {
		content = Div(icon, g.Text(text))
	}

	return button.Button(
		content,
		button.WithVariant("default"),
		button.WithAttrs(
			g.Attr("onclick", fmt.Sprintf("window.location.href='%s'", href)),
		),
	)
}

// SecondaryButton creates a secondary action button using ForgeUI.
func SecondaryButton(href, text string, icon g.Node) g.Node {
	content := Div(g.Text(text))
	if icon != nil {
		content = Div(icon, g.Text(text))
	}

	return button.Button(
		content,
		button.WithVariant("secondary"),
		button.WithAttrs(
			g.Attr("onclick", fmt.Sprintf("window.location.href='%s'", href)),
		),
	)
}

// DangerButton creates a danger/destructive action button using ForgeUI.
func DangerButton(onclick, text string, icon g.Node) g.Node {
	content := Div(g.Text(text))
	if icon != nil {
		content = Div(icon, g.Text(text))
	}

	return button.Button(
		content,
		button.WithVariant("destructive"),
		button.WithAttrs(
			Type("button"),
			g.Attr("onclick", onclick),
		),
	)
}

// Card creates a basic card container.
func Card(children ...g.Node) g.Node {
	return card.Card(children...)
}

// CardWithHeader creates a card with a header section.
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

// StatsCard renders a statistics card with icon and value.
func StatsCard(label, xDataValue, iconColor string) g.Node {
	return card.Card(
		Class("hover:shadow-md transition-shadow"),
		card.Content(
			Class("p-6"),
			primitives.VStack("gap-2",
				P(Class("text-sm font-medium text-muted-foreground"), g.Text(label)),
				P(
					Class("text-2xl font-bold"),
					g.Attr("x-text", xDataValue),
					g.Text("0"), // Fallback
				),
			),
		),
	)
}

// RoleBadge renders a badge for organization roles.
func RoleBadge(role string) g.Node {
	var badgeClass string

	switch role {
	case "owner":
		badgeClass = "bg-primary text-primary-foreground capitalize"
	case "admin":
		badgeClass = "bg-secondary text-secondary-foreground capitalize"
	case "member":
		badgeClass = "border border-border capitalize"
	default:
		badgeClass = "border border-border capitalize"
	}

	return badge.Badge(role, badge.WithClass(badgeClass))
}

// StatusBadge renders a badge for status (active, pending, etc.)
func StatusBadge(status string) g.Node {
	var (
		badgeClass  string
		displayText string
	)

	switch status {
	case "active":
		badgeClass = "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400 border-transparent"
		displayText = "Active"
	case "pending":
		badgeClass = "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400 border-transparent"
		displayText = "Pending"
	case "suspended":
		badgeClass = "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400 border-transparent"
		displayText = "Suspended"
	case "accepted":
		badgeClass = "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400 border-transparent"
		displayText = "Accepted"
	case "declined":
		badgeClass = "border border-border"
		displayText = "Declined"
	case "expired":
		badgeClass = "border border-border"
		displayText = "Expired"
	default:
		badgeClass = "border border-border"
		displayText = status
	}

	return badge.Badge(displayText, badge.WithClass(badgeClass))
}

// SearchInput renders a search input field using ForgeUI.
func SearchInput(placeholder, currentValue, formAction string) g.Node {
	return Form(
		Method("GET"),
		Action(formAction),
		Class("flex-1 max-w-sm"),
		input.InputGroup(nil,
			input.InputLeftAddon(nil, lucide.Search(Class("size-4"))),
			input.Input(
				input.WithType("search"),
				input.WithName("search"),
				input.WithValue(currentValue),
				input.WithPlaceholder(placeholder),
				input.WithAttrs(
					g.Attr("x-model", "filters.search"),
					g.Attr("@input.debounce.500ms", "loadData()"),
				),
			),
		),
	)
}

// LoadingSpinner renders a loading spinner.
func LoadingSpinner() g.Node {
	return Div(
		Class("flex items-center justify-center py-12"),
		Div(
			Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary"),
		),
	)
}

// ErrorMessage renders an error message.
func ErrorMessage(xShowCondition string) g.Node {
	return Div(
		g.Attr("x-show", xShowCondition),
		g.Attr("x-cloak", ""),
		Class("bg-destructive/10 border border-destructive/20 rounded-lg p-4"),
		Div(
			Class("flex items-center gap-2 text-destructive"),
			lucide.TriangleAlert(Class("size-5")),
			Span(g.Attr("x-text", "error"), g.Text("An error occurred")),
		),
	)
}

// EmptyState renders an empty state message using ForgeUI.
func EmptyState(icon g.Node, title, description string) g.Node {
	return emptystate.EmptyState(
		emptystate.WithIcon(icon),
		emptystate.WithTitle(title),
		emptystate.WithDescription(description),
	)
}

// QuickLinkCard renders a quick access card.
func QuickLinkCard(title, description, href string, icon g.Node) g.Node {
	return A(
		Href(href),
		Class("group"),
		card.Card(
			Class("transition-all hover:shadow-md hover:border-primary/50"),
			card.Content(
				Class("p-4"),
				Div(
					Class("flex items-start gap-3"),
					g.If(icon != nil,
						Div(
							Class("rounded-lg bg-primary/10 p-3 group-hover:bg-primary/20 transition-colors"),
							icon,
						),
					),
					Div(
						Class("flex-1 min-w-0"),
						H3(
							Class("text-sm font-semibold group-hover:text-primary transition-colors"),
							g.Text(title),
						),
						P(
							Class("mt-1 text-xs text-muted-foreground line-clamp-2"),
							g.Text(description),
						),
					),
					lucide.ChevronRight(Class("size-5 text-muted-foreground transition-transform group-hover:translate-x-1")),
				),
			),
		),
	)
}

// BackLink renders a back navigation link.
func BackLink(href, text string) g.Node {
	return A(
		Href(href),
		Class("inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors mb-4"),
		lucide.ArrowLeft(Class("size-4")),
		g.Text(text),
	)
}

// FormField renders a form field with label and input using ForgeUI.
func FormField(id, label, fieldType, name, placeholder string, required bool, helpText string) g.Node {
	return Div(
		Class("space-y-2"),
		Label(
			For(id),
			Class("text-sm font-medium"),
			g.Text(label),
			g.If(required, Span(Class("text-destructive"), g.Text("*"))),
		),
		input.Input(
			input.WithType(fieldType),
			input.WithID(id),
			input.WithName(name),
			input.WithPlaceholder(placeholder),
			input.WithAttrs(
				g.If(required, Required()),
			),
		),
		g.If(helpText != "", func() g.Node {
			return P(Class("text-xs text-muted-foreground"), g.Text(helpText))
		}()),
	)
}

// TextareaField renders a textarea form field.
func TextareaField(id, label, name, placeholder string, rows int, required bool, helpText string) g.Node {
	return Div(
		Class("space-y-2"),
		Label(
			For(id),
			Class("text-sm font-medium"),
			g.Text(label),
			g.If(required, Span(Class("text-destructive"), g.Text("*"))),
		),
		Textarea(
			ID(id),
			Name(name),
			Placeholder(placeholder),
			Rows(strconv.Itoa(rows)),
			Class("flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"),
			g.If(required, g.Attr("required", "")),
		),
		g.If(helpText != "", func() g.Node {
			return P(Class("text-xs text-muted-foreground"), g.Text(helpText))
		}()),
	)
}

// SelectField renders a select dropdown field.
func SelectField(id, label, name string, required bool, options []SelectOption, helpText string) g.Node {
	optionNodes := make([]g.Node, len(options))
	for i, opt := range options {
		optionNodes[i] = Option(
			Value(opt.Value),
			g.If(opt.Selected, Selected()),
			g.Text(opt.Label),
		)
	}

	return Div(
		Class("space-y-2"),
		Label(
			For(id),
			Class("text-sm font-medium"),
			g.Text(label),
			g.If(required, Span(Class("text-destructive"), g.Text("*"))),
		),
		Select(
			ID(id),
			Name(name),
			Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"),
			g.If(required, g.Attr("required", "")),
			g.Group(optionNodes),
		),
		g.If(helpText != "", func() g.Node {
			return P(Class("text-xs text-muted-foreground"), g.Text(helpText))
		}()),
	)
}

// SelectOption represents an option for a select field.
type SelectOption struct {
	Value    string
	Label    string
	Selected bool
}

// ConfirmDialog renders a confirmation dialog using Alpine.js.
func ConfirmDialog(message, confirmButtonText, xShowVar, onConfirm string) g.Node {
	return Div(
		g.Attr("x-show", xShowVar),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		g.Attr("@click.self", xShowVar+" = false"),
		card.Card(
			Class("max-w-md w-full mx-4"),
			card.Header(
				card.Title("Confirm Action"),
			),
			card.Content(
				P(Class("text-sm text-muted-foreground"), g.Text(message)),
			),
			card.Footer(
				Class("flex justify-end gap-2"),
				button.Button(
					g.Text("Cancel"),
					button.WithVariant("outline"),
					button.WithAttrs(
						Type("button"),
						g.Attr("@click", xShowVar+" = false"),
					),
				),
				button.Button(
					g.Text(confirmButtonText),
					button.WithVariant("destructive"),
					button.WithAttrs(
						Type("button"),
						g.Attr("@click", onConfirm),
					),
				),
			),
		),
	)
}

// FormatDate formats a time.Time to a readable string.
func FormatDate(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

// FormatDateTime formats a time.Time to a readable string with time.
func FormatDateTime(t time.Time) string {
	return t.Format("Jan 2, 2006 3:04 PM")
}

// FormatTimeAgo returns a human-readable "time ago" string.
func FormatTimeAgo(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)
	default:
		return FormatDate(t)
	}
}

// Pagination renders pagination controls.
func Pagination(currentPage, totalPages int, baseURL string) g.Node {
	if totalPages <= 1 {
		return nil
	}

	const (
		btnClass    = "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium transition-colors h-9 px-3 border border-input bg-background hover:bg-accent hover:text-accent-foreground"
		activeClass = "inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium h-9 px-3 bg-primary text-primary-foreground"
	)

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
				g.Text(strconv.Itoa(i)),
			))
		} else if i == 1 || i == totalPages || (i >= currentPage-1 && i <= currentPage+1) {
			pages = append(pages, A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
				Class(btnClass),
				g.Text(strconv.Itoa(i)),
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

// DataTable renders a data table using ForgeUI.
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

// TableRow creates a table row using ForgeUI.
func TableRow(cells ...g.Node) g.Node {
	return table.TableRow()(cells...)
}

// TableCell creates a table cell using ForgeUI.
func TableCell(content g.Node) g.Node {
	return table.TableCell()(content)
}

// TableCellSecondary creates a secondary table cell with muted text.
func TableCellSecondary(content g.Node) g.Node {
	return table.TableCell(table.WithCellClass("text-muted-foreground"))(content)
}

// TableCellActions creates a table cell with action buttons.
func TableCellActions(actions ...g.Node) g.Node {
	return table.TableCell(table.WithAlign(table.AlignRight))(
		primitives.HStack("gap-2 justify-end", actions...),
	)
}

// IconButton creates a small icon button.
func IconButton(href string, icon g.Node, title, colorClass string) g.Node {
	return A(
		Href(href),
		Title(title),
		Class("inline-flex items-center justify-center rounded-md outline-none ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 hover:bg-accent hover:text-accent-foreground size-9 "+colorClass),
		icon,
	)
}

// ConfirmButton creates a button that requires confirmation.
func ConfirmButton(formAction, method, text, confirmMessage, colorClass string, icon g.Node) g.Node {
	btnContent := g.Text(text)
	if icon != nil {
		btnContent = Div(icon, g.Text(text))
	}

	return Form(
		Action(formAction),
		Method(method),
		g.Attr("onsubmit", fmt.Sprintf("return confirm('%s')", confirmMessage)),
		Class("inline"),
		button.Button(
			btnContent,
			button.WithVariant(forgeui.Variant(colorClass)),
			button.WithSize("sm"),
			button.WithAttrs(Type("submit")),
		),
	)
}

// =============================================================================
// Helper Functions
// =============================================================================

// XIDToString converts xid.ID to string safely.
func XIDToString(id xid.ID) string {
	return id.String()
}

// StringPtr returns a pointer to a string.
func StringPtr(s string) *string {
	return &s
}
