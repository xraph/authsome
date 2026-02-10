package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// PageHeader renders a page header with title and description.
func PageHeader(title, description string) g.Node {
	return Div(
		Class("mb-6"),
		H1(Class("text-3xl font-bold text-foreground"), g.Text(title)),
		P(Class("text-muted-foreground mt-1"), g.Text(description)),
	)
}

// LoadingSpinner renders a loading spinner.
func LoadingSpinner() g.Node {
	return Div(
		Class("flex items-center justify-center py-12"),
		lucide.Loader(Class("animate-spin h-8 w-8 text-muted-foreground")),
	)
}

// ErrorMessage renders an error message with an x-show condition.
func ErrorMessage(condition string) g.Node {
	return Div(
		g.Attr("x-show", condition),
		g.Attr("x-cloak", ""),
		Class("bg-destructive/15 dark:bg-destructive/25 border border-destructive/50 rounded-lg p-4 mb-6"),
		Div(
			Class("flex items-center gap-2 text-destructive"),
			lucide.CircleX(Class("size-5")),
			Span(g.Attr("x-text", "error")),
		),
	)
}

// SuccessMessage renders a success message with an x-show condition.
func SuccessMessage(condition string) g.Node {
	return Div(
		g.Attr("x-show", condition),
		g.Attr("x-transition", ""),
		Class("bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800 rounded-lg p-4 mb-6"),
		Div(
			Class("flex items-center gap-2 text-emerald-700 dark:text-emerald-400"),
			lucide.CircleCheck(Class("size-5")),
			Span(g.Attr("x-text", condition)),
		),
	)
}
