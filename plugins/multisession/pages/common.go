// Package pages provides ForgeUI-based page templates for the multisession plugin dashboard.
package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/emptystate"
	"github.com/xraph/forgeui/components/input"
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

// PageHeaderWithBack renders a page header with back navigation.
func PageHeaderWithBack(backHref, backText, title, description string, actions ...g.Node) g.Node {
	return Div(
		Class("space-y-4 mb-6"),
		BackLink(backHref, backText),
		PageHeader(title, description, actions...),
	)
}

// BackLink renders a back navigation link.
func BackLink(href, text string) g.Node {
	return A(
		Href(href),
		Class("inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"),
		lucide.ArrowLeft(Class("size-4")),
		g.Text(text),
	)
}

// PrimaryButton creates a primary action button using ForgeUI.
func PrimaryButton(href, text string, icon g.Node) g.Node {
	content := Div(g.Text(text))
	if icon != nil {
		content = Div(Class("flex items-center gap-2"), icon, g.Text(text))
	}

	return button.Button(
		content,
		button.WithVariant("default"),
		button.WithAttrs(
			g.Attr("onclick", fmt.Sprintf("window.location.href='%s'", href)),
		),
	)
}

// SecondaryButton creates a secondary action button.
func SecondaryButton(href, text string, icon g.Node) g.Node {
	content := Div(g.Text(text))
	if icon != nil {
		content = Div(Class("flex items-center gap-2"), icon, g.Text(text))
	}

	return button.Button(
		content,
		button.WithVariant("outline"),
		button.WithAttrs(
			g.Attr("onclick", fmt.Sprintf("window.location.href='%s'", href)),
		),
	)
}

// DangerButton creates a danger/destructive action button.
func DangerButton(onclick, text string, icon g.Node) g.Node {
	content := Div(g.Text(text))
	if icon != nil {
		content = Div(Class("flex items-center gap-2"), icon, g.Text(text))
	}

	return button.Button(
		content,
		button.WithVariant("destructive"),
		button.WithAttrs(
			Type("button"),
			g.Attr("@click", onclick),
		),
	)
}

// RefreshButton creates a refresh button with loading state.
func RefreshButton(loadFn string) g.Node {
	return button.Button(
		Div(
			Class("flex items-center gap-2"),
			lucide.RefreshCw(Class("size-4"), g.Attr(":class", "loading ? 'animate-spin' : ''")),
			g.Text("Refresh"),
		),
		button.WithVariant("outline"),
		button.WithAttrs(
			Type("button"),
			g.Attr("@click", loadFn),
			g.Attr(":disabled", "loading"),
		),
	)
}

// =============================================================================
// Card Components
// =============================================================================

// StatsCard renders a statistics card with dynamic value.
func StatsCard(label, xDataValue, colorClass string, icon g.Node) g.Node {
	iconBgClass := "bg-primary/10"
	iconTextClass := "text-primary"

	switch colorClass {
	case "blue":
		iconBgClass = "bg-blue-100 dark:bg-blue-900/30"
		iconTextClass = "text-blue-600 dark:text-blue-400"
	case "emerald", "green":
		iconBgClass = "bg-emerald-100 dark:bg-emerald-900/30"
		iconTextClass = "text-emerald-600 dark:text-emerald-400"
	case "violet", "purple":
		iconBgClass = "bg-violet-100 dark:bg-violet-900/30"
		iconTextClass = "text-violet-600 dark:text-violet-400"
	case "amber", "yellow":
		iconBgClass = "bg-amber-100 dark:bg-amber-900/30"
		iconTextClass = "text-amber-600 dark:text-amber-400"
	}

	return card.Card(
		Class("hover:shadow-md transition-shadow"),
		card.Content(
			Class("p-6"),
			Div(
				Class("flex items-center justify-between"),
				Div(
					P(Class("text-sm font-medium text-muted-foreground"), g.Text(label)),
					P(
						Class("text-3xl font-bold mt-2"),
						g.Attr("x-text", xDataValue),
						g.Text("—"),
					),
				),
				Div(
					Class("rounded-xl p-3 "+iconBgClass),
					Div(Class(iconTextClass), icon),
				),
			),
		),
	)
}

// DetailCard renders a card with a header icon and content.
func DetailCard(title string, icon g.Node, content ...g.Node) g.Node {
	return card.Card(
		card.Header(
			Div(
				Class("flex items-center gap-2"),
				Div(
					Class("rounded-lg bg-primary/10 p-1.5"),
					Div(Class("text-primary"), icon),
				),
				card.Title(title),
			),
		),
		card.Content(g.Group(content)),
	)
}

// =============================================================================
// Badge Components
// =============================================================================

// StatusBadge renders a session status badge (active, expiring, expired).
func StatusBadge(xShowCondition, status string) g.Node {
	var badgeClass, displayText string

	switch status {
	case "active":
		badgeClass = "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400 border-transparent"
		displayText = "Active"
	case "expiring":
		badgeClass = "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400 border-transparent"
		displayText = "Expiring Soon"
	case "expired":
		badgeClass = "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400 border-transparent"
		displayText = "Expired"
	default:
		badgeClass = "border border-border"
		displayText = status
	}

	return Span(
		g.Attr("x-show", xShowCondition),
		Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium "+badgeClass),
		g.Text(displayText),
	)
}

// DynamicStatusBadge renders a status badge that changes based on Alpine.js data.
func DynamicStatusBadge() g.Node {
	return Span(
		g.Attr("x-text", "session.status === 'active' ? 'Active' : (session.status === 'expiring' ? 'Expiring Soon' : 'Expired')"),
		g.Attr(":class", `{
			'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400': session.status === 'active',
			'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400': session.status === 'expiring',
			'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400': session.status === 'expired'
		}`),
		Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium border-transparent"),
	)
}

// DeviceBadge renders a device type badge.
func DeviceBadge(deviceType string) g.Node {
	var (
		badgeClass, displayText string
		icon                    g.Node
	)

	switch deviceType {
	case "mobile":
		badgeClass = "bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400"
		displayText = "Mobile"
		icon = lucide.Smartphone(Class("size-3 mr-1"))
	case "tablet":
		badgeClass = "bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400"
		displayText = "Tablet"
		icon = lucide.Tablet(Class("size-3 mr-1"))
	case "desktop":
		badgeClass = "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
		displayText = "Desktop"
		icon = lucide.Monitor(Class("size-3 mr-1"))
	default:
		badgeClass = "border border-border"
		displayText = deviceType
		icon = lucide.CircleHelp(Class("size-3 mr-1"))
	}

	// Note: Badge doesn't have WithIcon, so we wrap manually
	return Span(
		Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium "+badgeClass),
		icon,
		g.Text(displayText),
	)
}

// =============================================================================
// Form Components
// =============================================================================

// SearchInput renders a search input with Alpine.js binding.
func SearchInput(placeholder, xModel, debounceAction string) g.Node {
	return Div(
		Class("relative flex-1 max-w-sm"),
		Div(
			Class("absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none"),
			lucide.Search(Class("size-4 text-muted-foreground")),
		),
		input.Input(
			input.WithType("search"),
			input.WithPlaceholder(placeholder),
			input.WithAttrs(
				Class("pl-10"),
				g.Attr("x-model", xModel),
				g.Attr("@input.debounce.500ms", debounceAction),
			),
		),
	)
}

// FilterSelect renders a select dropdown for filtering.
func FilterSelect(label, xModel, onChange string, options []FilterOption) g.Node {
	optionNodes := make([]g.Node, len(options))
	for i, opt := range options {
		optionNodes[i] = Option(
			Value(opt.Value),
			g.Text(opt.Label),
		)
	}

	return Div(
		Class("flex items-center gap-2"),
		Label(Class("text-sm font-medium text-muted-foreground"), g.Text(label)),
		Select(
			Class("flex h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-ring"),
			g.Attr("x-model", xModel),
			g.Attr("@change", onChange),
			g.Group(optionNodes),
		),
	)
}

// FilterOption represents an option for a filter select.
type FilterOption struct {
	Value string
	Label string
}

// ViewToggle renders a grid/list view toggle.
func ViewToggle(xModel string) g.Node {
	return Div(
		Class("flex items-center rounded-lg border border-input bg-background p-1"),
		Button(
			Type("button"),
			g.Attr("@click", xModel+" = 'grid'"),
			g.Attr(":class", xModel+" === 'grid' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'"),
			Class("rounded p-1.5 transition-colors"),
			lucide.LayoutGrid(Class("size-4")),
		),
		Button(
			Type("button"),
			g.Attr("@click", xModel+" = 'list'"),
			g.Attr(":class", xModel+" === 'list' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'"),
			Class("rounded p-1.5 transition-colors"),
			lucide.List(Class("size-4")),
		),
	)
}

// =============================================================================
// State Components
// =============================================================================

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

// EmptyState renders an empty state message.
func EmptyState(icon g.Node, title, description string) g.Node {
	return emptystate.EmptyState(
		emptystate.WithIcon(icon),
		emptystate.WithTitle(title),
		emptystate.WithDescription(description),
	)
}

// =============================================================================
// Pagination Component
// =============================================================================

// Pagination renders pagination controls with Alpine.js.
func Pagination(goToPageFn string) g.Node {
	return Div(
		g.Attr("x-show", "pagination.totalPages > 1"),
		Class("flex items-center justify-between px-2 py-4 border-t border-border"),

		// Info
		Div(
			Class("text-sm text-muted-foreground"),
			Span(g.Text("Showing ")),
			Span(Class("font-medium"), g.Attr("x-text", "((pagination.currentPage - 1) * pagination.pageSize) + 1")),
			Span(g.Text("-")),
			Span(Class("font-medium"), g.Attr("x-text", "Math.min(pagination.currentPage * pagination.pageSize, pagination.totalItems)")),
			Span(g.Text(" of ")),
			Span(Class("font-medium"), g.Attr("x-text", "pagination.totalItems")),
			Span(g.Text(" sessions")),
		),

		// Controls
		Div(
			Class("flex items-center gap-2"),
			button.Button(
				Div(
					Class("flex items-center gap-1"),
					lucide.ChevronLeft(Class("size-4")),
					Span(g.Text("Previous")),
				),
				button.WithVariant("outline"),
				button.WithSize("sm"),
				button.WithAttrs(
					g.Attr("@click", goToPageFn+"(pagination.currentPage - 1)"),
					g.Attr(":disabled", "pagination.currentPage === 1"),
				),
			),

			// Page numbers
			g.El("template", g.Attr("x-for", "page in visiblePages"),
				button.Button(
					Span(g.Attr("x-text", "page")),
					button.WithSize("sm"),
					button.WithAttrs(
						g.Attr("@click", goToPageFn+"(page)"),
						g.Attr(":class", "page === pagination.currentPage ? 'bg-primary text-primary-foreground' : ''"),
					),
				),
			),

			button.Button(
				Div(
					Class("flex items-center gap-1"),
					Span(g.Text("Next")),
					lucide.ChevronRight(Class("size-4")),
				),
				button.WithVariant("outline"),
				button.WithSize("sm"),
				button.WithAttrs(
					g.Attr("@click", goToPageFn+"(pagination.currentPage + 1)"),
					g.Attr(":disabled", "pagination.currentPage >= pagination.totalPages"),
				),
			),
		),
	)
}

// =============================================================================
// Detail Row Component
// =============================================================================

// DetailRow renders a label-value pair with an icon.
func DetailRow(label, xTextValue string, icon g.Node) g.Node {
	return Div(
		Class("flex items-start gap-3"),
		Div(Class("text-muted-foreground mt-0.5"), icon),
		Div(
			Div(Class("text-xs font-medium text-muted-foreground uppercase tracking-wider"), g.Text(label)),
			Div(
				Class("mt-0.5 text-sm font-medium"),
				g.Attr("x-text", xTextValue),
				g.Text("—"),
			),
		),
	)
}

// StaticDetailRow renders a static label-value pair with an icon.
func StaticDetailRow(label, value string, icon g.Node) g.Node {
	return Div(
		Class("flex items-start gap-3"),
		Div(Class("text-muted-foreground mt-0.5"), icon),
		Div(
			Div(Class("text-xs font-medium text-muted-foreground uppercase tracking-wider"), g.Text(label)),
			Div(Class("mt-0.5 text-sm font-medium"), g.Text(value)),
		),
	)
}

// =============================================================================
// Device Icon Component
// =============================================================================

// DeviceIcon returns an icon based on device type.
func DeviceIcon(xDeviceType string) g.Node {
	return Div(
		// Mobile
		g.El("template", g.Attr("x-if", xDeviceType+" === 'mobile'"),
			lucide.Smartphone(Class("size-5")),
		),
		// Tablet
		g.El("template", g.Attr("x-if", xDeviceType+" === 'tablet'"),
			lucide.Tablet(Class("size-5")),
		),
		// Desktop (default)
		g.El("template", g.Attr("x-if", xDeviceType+" !== 'mobile' && "+xDeviceType+" !== 'tablet'"),
			lucide.Monitor(Class("size-5")),
		),
	)
}

// =============================================================================
// Confirmation Dialog
// =============================================================================

// ConfirmDialog renders a confirmation dialog with Alpine.js.
func ConfirmDialog(xShowVar, title, message, confirmText, onConfirm string) g.Node {
	return Div(
		g.Attr("x-show", xShowVar),
		g.Attr("x-cloak", ""),
		g.Attr("x-transition", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		g.Attr("@click.self", xShowVar+" = false"),
		card.Card(
			Class("max-w-md w-full mx-4"),
			card.Header(
				card.Title(title),
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
					g.Text(confirmText),
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
