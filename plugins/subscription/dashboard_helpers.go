package subscription

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// calculateDateRange converts "7d", "30d", "90d" to start/end dates.
func calculateDateRange(rangeStr string) (time.Time, time.Time) {
	end := time.Now()

	var start time.Time

	switch rangeStr {
	case "7d":
		start = end.AddDate(0, 0, -7)
	case "30d":
		start = end.AddDate(0, 0, -30)
	case "90d":
		start = end.AddDate(0, 0, -90)
	default:
		start = end.AddDate(0, 0, -30)
	}

	return start, end
}

// formatNumber formats large numbers with commas.
func formatNumber(n int64) string {
	if n < 0 {
		return "-" + formatNumber(-n)
	}

	if n < 1000 {
		return strconv.FormatInt(n, 10)
	}

	// Convert to string and add commas
	str := strconv.FormatInt(n, 10)

	var result strings.Builder

	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}

		result.WriteRune(c)
	}

	return result.String()
}

// formatLimit handles -1 (unlimited) special case.
func formatLimit(limit int64) string {
	if limit == -1 || limit == 0 {
		return "Unlimited"
	}

	return formatNumber(limit)
}

// calculatePercent calculates usage percentage.
func calculatePercent(usage, limit int64) float64 {
	if limit == -1 || limit == 0 {
		return 0
	}

	return float64(usage) / float64(limit) * 100
}

// renderProgressBar creates a visual progress bar.
func renderProgressBar(percent float64) g.Node {
	// Cap at 100%
	if percent > 100 {
		percent = 100
	}

	color := "bg-green-500"
	if percent > 75 {
		color = "bg-yellow-500"
	}

	if percent > 90 {
		color = "bg-red-500"
	}

	return Div(
		Class("flex items-center gap-2"),
		Div(
			Class("flex-1 bg-gray-200 rounded-full h-2 dark:bg-gray-700"),
			Div(
				Class(fmt.Sprintf("h-2 rounded-full %s transition-all duration-300", color)),
				StyleAttr(fmt.Sprintf("width: %.1f%%", percent)),
			),
		),
		Span(
			Class("text-sm font-medium text-gray-700 dark:text-gray-300 min-w-[3rem] text-right"),
			g.Text(fmt.Sprintf("%.1f%%", percent)),
		),
	)
}

// renderDateRangeFilter renders the date range filter buttons.
func renderDateRangeFilter(currentPath, activeRange string) g.Node {
	buttonClass := func(rangeStr string) string {
		base := "px-4 py-2 text-sm font-medium rounded-lg transition-colors"
		if rangeStr == activeRange {
			return base + " bg-violet-600 text-white"
		}

		return base + " bg-white text-gray-700 border border-gray-200 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700 dark:hover:bg-gray-700"
	}

	return Div(
		Class("flex items-center gap-2"),
		A(
			Href(currentPath+"?range=7d"),
			Class(buttonClass("7d")),
			g.Text("Last 7 days"),
		),
		A(
			Href(currentPath+"?range=30d"),
			Class(buttonClass("30d")),
			g.Text("Last 30 days"),
		),
		A(
			Href(currentPath+"?range=90d"),
			Class(buttonClass("90d")),
			g.Text("Last 90 days"),
		),
	)
}

// renderEmptyState renders an empty state message.
func renderEmptyState(icon g.Node, title, description string) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-12 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("text-center"),
			Div(Class("mx-auto mb-4"), icon),
			H3(
				Class("text-lg font-medium text-slate-900 dark:text-white mb-2"),
				g.Text(title),
			),
			P(
				Class("text-slate-500 dark:text-gray-400 max-w-md mx-auto"),
				g.Text(description),
			),
		),
	)
}

// renderStatusBadge renders a subscription status badge.
func renderStatusBadge(status string) g.Node {
	var (
		colorClass  string
		displayText string
	)

	switch status {
	case "active":
		colorClass = "bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400"
		displayText = "Active"
	case "trialing":
		colorClass = "bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400"
		displayText = "Trial"
	case "canceled", "cancelled":
		colorClass = "bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400"
		displayText = "Canceled"
	case "paused":
		colorClass = "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400"
		displayText = "Paused"
	case "expired":
		colorClass = "bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400"
		displayText = "Expired"
	default:
		colorClass = "bg-gray-100 text-gray-800 dark:bg-gray-900/20 dark:text-gray-400"
		displayText = status
	}

	return Span(
		Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium "+colorClass),
		g.Text(displayText),
	)
}

// renderSimpleLineChart renders a simple ASCII-style line chart or placeholder.
func renderSimpleLineChart(dataPoints []int64, labels []string) g.Node {
	if len(dataPoints) == 0 {
		return renderEmptyState(
			lucide.TrendingUp(Class("mx-auto h-16 w-16 text-slate-300 dark:text-gray-600")),
			"No Data Available",
			"Start using features to see trends appear here",
		)
	}

	// For now, render a simple placeholder with data attributes
	// In a real implementation, this would use a charting library like Chart.js
	return Div(
		Class("h-64 flex items-center justify-center bg-slate-50 dark:bg-gray-800 rounded-lg border border-slate-200 dark:border-gray-700"),
		Div(
			Class("text-center text-slate-400"),
			lucide.TrendingUp(Class("mx-auto h-12 w-12 mb-2")),
			P(g.Text("Chart visualization")),
			P(Class("text-xs"), g.Text(fmt.Sprintf("%d data points", len(dataPoints)))),
		),
	)
}

// renderMetricChange renders a metric change indicator with arrow.
func renderMetricChange(change float64) g.Node {
	if change == 0 {
		return Span(
			Class("text-sm text-gray-500 dark:text-gray-400"),
			g.Text("No change"),
		)
	}

	var (
		colorClass string
		icon       g.Node
		prefix     string
	)

	if change > 0 {
		colorClass = "text-green-600 dark:text-green-400"
		icon = lucide.TrendingUp(Class("size-4"))
		prefix = "+"
	} else {
		colorClass = "text-red-600 dark:text-red-400"
		icon = lucide.TrendingDown(Class("size-4"))
		prefix = ""
	}

	return Span(
		Class("inline-flex items-center gap-1 text-sm font-medium "+colorClass),
		icon,
		g.Text(fmt.Sprintf("%s%.1f%%", prefix, change)),
	)
}
