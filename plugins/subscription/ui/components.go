// Package ui provides Pine UI components for the subscription plugin dashboard
package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

// SelectOption represents an option in a select field
type SelectOption struct {
	Value string
	Label string
}

// Common CSS classes using Pine UI patterns
const (
	// Cards
	cardClass       = "bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700"
	cardHeaderClass = "px-6 py-4 border-b border-gray-200 dark:border-gray-700"
	cardBodyClass   = "px-6 py-4"

	// Buttons
	btnPrimaryClass   = "inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
	btnSecondaryClass = "inline-flex items-center px-4 py-2 border border-gray-300 dark:border-gray-600 text-sm font-medium rounded-md text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
	btnDangerClass    = "inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-red-600 hover:bg-red-700"

	// Badges
	badgeSuccessClass = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
	badgeWarningClass = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200"
	badgeDangerClass  = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200"
	badgeInfoClass    = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200"
	badgeGrayClass    = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200"

	// Tables
	tableClass      = "min-w-full divide-y divide-gray-200 dark:divide-gray-700"
	tableHeadClass  = "bg-gray-50 dark:bg-gray-800"
	tableBodyClass  = "bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700"
	tableCellClass  = "px-6 py-4 whitespace-nowrap text-sm"
	tableHeaderCell = "px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider"
)

// Card renders a Pine UI card component
func Card(title string, children ...g.Node) g.Node {
	return html.Div(
		html.Class(cardClass),
		g.If(title != "", html.Div(
			html.Class(cardHeaderClass),
			html.H3(
				html.Class("text-lg font-medium text-gray-900 dark:text-white"),
				g.Text(title),
			),
		)),
		html.Div(
			html.Class(cardBodyClass),
			g.Group(children),
		),
	)
}

// CardWithActions renders a card with header actions
func CardWithActions(title string, actions g.Node, children ...g.Node) g.Node {
	return html.Div(
		html.Class(cardClass),
		html.Div(
			html.Class(cardHeaderClass+" flex items-center justify-between"),
			html.H3(
				html.Class("text-lg font-medium text-gray-900 dark:text-white"),
				g.Text(title),
			),
			actions,
		),
		html.Div(
			html.Class(cardBodyClass),
			g.Group(children),
		),
	)
}

// StatCard renders a statistics card
func StatCard(title, value, change string, positive bool, icon g.Node) g.Node {
	changeClass := "text-green-600 dark:text-green-400"
	if !positive {
		changeClass = "text-red-600 dark:text-red-400"
	}

	return html.Div(
		html.Class(cardClass+" p-6"),
		html.Div(
			html.Class("flex items-center"),
			html.Div(
				html.Class("flex-shrink-0 p-3 bg-indigo-50 dark:bg-indigo-900/20 rounded-lg"),
				icon,
			),
			html.Div(
				html.Class("ml-5 w-0 flex-1"),
				html.Dt(
					html.Class("text-sm font-medium text-gray-500 dark:text-gray-400 truncate"),
					g.Text(title),
				),
				html.Dd(
					html.Class("flex items-baseline"),
					html.Span(
						html.Class("text-2xl font-semibold text-gray-900 dark:text-white"),
						g.Text(value),
					),
					g.If(change != "", html.Span(
						html.Class("ml-2 text-sm font-medium "+changeClass),
						g.Text(change),
					)),
				),
			),
		),
	)
}

// Button renders a Pine UI button
func Button(text string, variant string, attrs ...g.Node) g.Node {
	class := btnPrimaryClass
	switch variant {
	case "secondary":
		class = btnSecondaryClass
	case "danger":
		class = btnDangerClass
	}

	allNodes := append([]g.Node{html.Class(class), g.Text(text)}, attrs...)
	return html.Button(allNodes...)
}

// LinkButton renders a button as a link
func LinkButton(text, href, variant string) g.Node {
	class := btnPrimaryClass
	switch variant {
	case "secondary":
		class = btnSecondaryClass
	case "danger":
		class = btnDangerClass
	}

	return html.A(
		html.Href(href),
		html.Class(class),
		g.Text(text),
	)
}

// Badge renders a Pine UI badge
func Badge(text, variant string) g.Node {
	class := badgeGrayClass
	switch variant {
	case "success":
		class = badgeSuccessClass
	case "warning":
		class = badgeWarningClass
	case "danger":
		class = badgeDangerClass
	case "info":
		class = badgeInfoClass
	}

	return html.Span(
		html.Class(class),
		g.Text(text),
	)
}

// StatusBadge renders a status badge based on subscription status
func StatusBadge(status string) g.Node {
	switch status {
	case "active":
		return Badge("Active", "success")
	case "trialing":
		return Badge("Trial", "info")
	case "past_due":
		return Badge("Past Due", "warning")
	case "canceled":
		return Badge("Canceled", "danger")
	case "paused":
		return Badge("Paused", "warning")
	case "incomplete":
		return Badge("Incomplete", "warning")
	default:
		return Badge(status, "gray")
	}
}

// Table renders a Pine UI table
func Table(headers []string, rows ...g.Node) g.Node {
	headerCells := make([]g.Node, len(headers))
	for i, h := range headers {
		headerCells[i] = html.Th(
			html.Class(tableHeaderCell),
			g.Attr("scope", "col"),
			g.Text(h),
		)
	}

	return html.Div(
		html.Class("overflow-x-auto"),
		html.Table(
			html.Class(tableClass),
			html.THead(
				html.Class(tableHeadClass),
				html.Tr(headerCells...),
			),
			html.TBody(
				html.Class(tableBodyClass),
				g.Group(rows),
			),
		),
	)
}

// TableRow renders a table row
func TableRow(cells ...g.Node) g.Node {
	return html.Tr(
		html.Class("hover:bg-gray-50 dark:hover:bg-gray-800"),
		g.Group(cells),
	)
}

// TableCell renders a table cell
func TableCell(content g.Node) g.Node {
	return html.Td(
		html.Class(tableCellClass+" text-gray-900 dark:text-white"),
		content,
	)
}

// TableCellMuted renders a muted table cell
func TableCellMuted(content g.Node) g.Node {
	return html.Td(
		html.Class(tableCellClass+" text-gray-500 dark:text-gray-400"),
		content,
	)
}

// EmptyState renders an empty state component
func EmptyState(title, description string, action g.Node) g.Node {
	return html.Div(
		html.Class("text-center py-12"),
		html.Div(
			html.Class("mx-auto h-12 w-12 text-gray-400"),
			// Icon placeholder
			html.Span(g.Text("📋")),
		),
		html.H3(
			html.Class("mt-2 text-sm font-medium text-gray-900 dark:text-white"),
			g.Text(title),
		),
		html.P(
			html.Class("mt-1 text-sm text-gray-500 dark:text-gray-400"),
			g.Text(description),
		),
		g.If(action != nil, html.Div(
			html.Class("mt-6"),
			action,
		)),
	)
}

// Pagination renders pagination controls
func Pagination(currentPage, totalPages int, baseURL string) g.Node {
	if totalPages <= 1 {
		return nil
	}

	prevDisabled := currentPage <= 1
	nextDisabled := currentPage >= totalPages

	return html.Nav(
		html.Class("flex items-center justify-between border-t border-gray-200 dark:border-gray-700 px-4 py-3"),
		g.Attr("aria-label", "Pagination"),
		html.Div(
			html.Class("hidden sm:block"),
			html.P(
				html.Class("text-sm text-gray-700 dark:text-gray-300"),
				g.Textf("Page %d of %d", currentPage, totalPages),
			),
		),
		html.Div(
			html.Class("flex-1 flex justify-between sm:justify-end space-x-2"),
			g.If(prevDisabled,
				html.Span(
					html.Class(btnSecondaryClass+" opacity-50 cursor-not-allowed"),
					g.Text("Previous"),
				),
			),
			g.If(!prevDisabled,
				html.A(
					html.Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)),
					html.Class(btnSecondaryClass),
					g.Text("Previous"),
				),
			),
			g.If(nextDisabled,
				html.Span(
					html.Class(btnSecondaryClass+" opacity-50 cursor-not-allowed"),
					g.Text("Next"),
				),
			),
			g.If(!nextDisabled,
				html.A(
					html.Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)),
					html.Class(btnSecondaryClass),
					g.Text("Next"),
				),
			),
		),
	)
}

// Alert renders an alert component
func Alert(message, variant string) g.Node {
	baseClass := "rounded-md p-4"
	var bgClass, textClass, iconClass string

	switch variant {
	case "success":
		bgClass = "bg-green-50 dark:bg-green-900/20"
		textClass = "text-green-800 dark:text-green-200"
		iconClass = "text-green-400"
	case "warning":
		bgClass = "bg-yellow-50 dark:bg-yellow-900/20"
		textClass = "text-yellow-800 dark:text-yellow-200"
		iconClass = "text-yellow-400"
	case "error":
		bgClass = "bg-red-50 dark:bg-red-900/20"
		textClass = "text-red-800 dark:text-red-200"
		iconClass = "text-red-400"
	default:
		bgClass = "bg-blue-50 dark:bg-blue-900/20"
		textClass = "text-blue-800 dark:text-blue-200"
		iconClass = "text-blue-400"
	}

	_ = iconClass // Placeholder for icon usage

	return html.Div(
		html.Class(baseClass+" "+bgClass),
		html.Div(
			html.Class("flex"),
			html.Div(
				html.Class("ml-3"),
				html.P(
					html.Class("text-sm font-medium "+textClass),
					g.Text(message),
				),
			),
		),
	)
}

// FormField renders a form field with label
func FormField(label, name, fieldType, value, placeholder string, required bool) g.Node {
	return html.Div(
		html.Class("mb-4"),
		html.Label(
			html.For(name),
			html.Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
			g.Text(label),
			g.If(required, html.Span(html.Class("text-red-500"), g.Text(" *"))),
		),
		html.Input(
			html.Type(fieldType),
			html.Name(name),
			html.ID(name),
			html.Value(value),
			g.If(placeholder != "", html.Placeholder(placeholder)),
			g.If(required, g.Attr("required", "required")),
			html.Class("mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm dark:bg-gray-800 dark:text-white"),
		),
	)
}

// SelectField renders a select field
func SelectField(label, name string, options []SelectOption, selectedValue string, required bool) g.Node {
	optionNodes := make([]g.Node, len(options))
	for i, opt := range options {
		optionNodes[i] = html.Option(
			html.Value(opt.Value),
			g.If(opt.Value == selectedValue, g.Attr("selected", "selected")),
			g.Text(opt.Label),
		)
	}

	return html.Div(
		html.Class("mb-4"),
		html.Label(
			html.For(name),
			html.Class("block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"),
			g.Text(label),
			g.If(required, html.Span(html.Class("text-red-500"), g.Text(" *"))),
		),
		html.Select(
			html.Name(name),
			html.ID(name),
			g.If(required, g.Attr("required", "required")),
			html.Class("mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm dark:bg-gray-800 dark:text-white"),
			g.Group(optionNodes),
		),
	)
}

// Modal renders a modal wrapper (requires JS to show/hide)
func Modal(id, title string, content, footer g.Node) g.Node {
	return html.Div(
		html.ID(id),
		html.Class("fixed inset-0 z-50 hidden overflow-y-auto"),
		g.Attr("aria-modal", "true"),
		g.Attr("role", "dialog"),
		// Backdrop
		html.Div(
			html.Class("fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"),
		),
		// Modal panel
		html.Div(
			html.Class("fixed inset-0 z-10 overflow-y-auto"),
			html.Div(
				html.Class("flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0"),
				html.Div(
					html.Class("relative transform overflow-hidden rounded-lg bg-white dark:bg-gray-800 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-lg"),
					// Header
					html.Div(
						html.Class("bg-white dark:bg-gray-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4"),
						html.H3(
							html.Class("text-lg font-medium leading-6 text-gray-900 dark:text-white"),
							g.Text(title),
						),
						html.Div(
							html.Class("mt-4"),
							content,
						),
					),
					// Footer
					g.If(footer != nil, html.Div(
						html.Class("bg-gray-50 dark:bg-gray-700 px-4 py-3 sm:flex sm:flex-row-reverse sm:px-6"),
						footer,
					)),
				),
			),
		),
	)
}

// DescriptionList renders a description list
func DescriptionList(items map[string]g.Node) g.Node {
	nodes := make([]g.Node, 0, len(items)*2)
	for key, value := range items {
		nodes = append(nodes,
			html.Dt(
				html.Class("text-sm font-medium text-gray-500 dark:text-gray-400"),
				g.Text(key),
			),
			html.Dd(
				html.Class("mt-1 text-sm text-gray-900 dark:text-white"),
				value,
			),
		)
	}

	return html.Dl(
		html.Class("divide-y divide-gray-200 dark:divide-gray-700"),
		html.Div(
			html.Class("py-4 sm:grid sm:grid-cols-3 sm:gap-4"),
			g.Group(nodes),
		),
	)
}

// MoneyDisplay formats and displays a money amount
func MoneyDisplay(amount int64, currency string) g.Node {
	// Simple formatting - in production would use proper currency formatting
	formatted := fmt.Sprintf("%s%.2f", currencySymbol(currency), float64(amount)/100)
	return html.Span(
		html.Class("font-mono"),
		g.Text(formatted),
	)
}

func currencySymbol(code string) string {
	symbols := map[string]string{
		"USD": "$",
		"EUR": "€",
		"GBP": "£",
		"JPY": "¥",
		"CAD": "CA$",
		"AUD": "A$",
	}
	if s, ok := symbols[code]; ok {
		return s
	}
	return code + " "
}
