package scim

import (
	"context"
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// Dashboard Widgets

// Widget rendering implementations for dashboard home page

// RenderRecentActivityWidget renders the recent activity widget.
func (e *DashboardExtension) RenderRecentActivityWidget(basePath string, currentApp *app.App) g.Node {
	if currentApp == nil {
		return Div(Class("text-gray-500"), g.Text("No app context"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch recent activity
	events, err := e.plugin.service.GetRecentActivity(ctx, currentApp.ID, nil, 5)
	if err != nil || len(events) == 0 {
		return Div(
			Class("text-center py-4"),
			P(Class("text-sm text-slate-500 dark:text-gray-400"),
				g.Text("No recent activity")),
		)
	}

	return Div(
		Class("space-y-2"),
		g.Group(e.renderRecentActivityItems(events)),
	)
}

// renderRecentActivityItems renders activity items.
func (e *DashboardExtension) renderRecentActivityItems(events []*SCIMSyncEvent) []g.Node {
	items := make([]g.Node, len(events))
	for i, event := range events {
		items[i] = Div(
			Class("flex items-start gap-2 text-sm"),
			statusIcon(event.Status),
			Div(
				Class("flex-1 min-w-0"),
				Div(Class("font-medium text-slate-900 dark:text-white truncate"), g.Text(event.EventType)),
				Div(Class("text-xs text-slate-500 dark:text-gray-400"), g.Text(formatRelativeTime(event.CreatedAt))),
			),
		)
	}

	return items
}

// RenderFailedOperationsWidget renders the failed operations widget.
func (e *DashboardExtension) RenderFailedOperationsWidget(basePath string, currentApp *app.App) g.Node {
	if currentApp == nil {
		return Div(Class("text-gray-500"), g.Text("No app context"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch failed operations count
	failedCount, err := e.plugin.service.GetFailedOperationsCount(ctx, currentApp.ID, nil)
	if err != nil {
		return Div(Class("text-gray-500"), g.Text("Error loading data"))
	}

	statusClass := "text-green-600 dark:text-green-400"
	statusText := "All Good"

	if failedCount > 0 {
		statusClass = "text-red-600 dark:text-red-400"
		statusText = "Needs Attention"
	}

	return Div(
		Class("text-center"),
		Div(
			Class("flex items-center justify-center gap-2 mb-2"),
			g.If(failedCount == 0,
				lucide.Check(Class("size-5 text-green-500")),
			),
			g.If(failedCount > 0,
				lucide.Info(Class("size-5 text-red-500")),
			),
			Div(
				Class("text-2xl font-bold "+statusClass),
				g.Textf("%d", failedCount),
			),
		),
		Div(
			Class("text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Failed Operations"),
		),
		Div(
			Class(fmt.Sprintf("text-xs %s mt-1", statusClass)),
			g.Text(statusText),
		),
	)
}

// Additional widget helper functions

// renderWidgetCard wraps content in a widget card.
func renderWidgetCard(title string, icon g.Node, content g.Node) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center gap-2 mb-3"),
			icon,
			H3(Class("text-sm font-semibold text-slate-900 dark:text-white"), g.Text(title)),
		),
		content,
	)
}

// renderMetricValue renders a metric value with optional change indicator.
func renderMetricValue(value string, change string, positive bool) g.Node {
	changeClass := "text-green-600 dark:text-green-400"
	changeIcon := lucide.TrendingUp(Class("size-3"))

	if !positive {
		changeClass = "text-red-600 dark:text-red-400"
		changeIcon = lucide.TrendingDown(Class("size-3"))
	}

	return Div(
		Div(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text(value)),
		g.If(change != "",
			Div(
				Class("flex items-center gap-1 text-xs "+changeClass),
				changeIcon,
				g.Text(change),
			),
		),
	)
}
