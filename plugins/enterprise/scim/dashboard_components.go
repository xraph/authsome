package scim

import (
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/rs/xid"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// UI Components for SCIM Dashboard

// statusBadge renders a status badge with appropriate styling.
func statusBadge(status string) g.Node {
	var classes string

	switch status {
	case "active", "success", "healthy", "synced":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
	case "pending", "warning", "needs_attention":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400"
	case "failed", "error", "disconnected", "inactive":
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
	default:
		classes = "inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300"
	}

	return Span(Class(classes), g.Text(status))
}

// statusIcon returns an appropriate icon for a status.
func statusIcon(status string) g.Node {
	switch status {
	case "active", "success", "healthy", "synced":
		return lucide.Check(Class("size-4 text-green-500"))
	case "pending", "warning", "needs_attention":
		return lucide.Info(Class("size-4 text-yellow-500"))
	case "failed", "error", "disconnected":
		return lucide.X(Class("size-4 text-red-500"))
	case "inactive":
		return lucide.Circle(Class("size-4 text-gray-400"))
	default:
		return lucide.CircleDot(Class("size-4 text-gray-400"))
	}
}

// statsCard renders a statistics card.
func statsCard(title, value, subtitle string, icon g.Node) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Div(Class("text-sm font-medium text-slate-600 dark:text-gray-400"), g.Text(title)),
				Div(Class("mt-1 text-2xl font-bold text-slate-900 dark:text-white"), g.Text(value)),
				g.If(subtitle != "", Div(Class("mt-1 text-xs text-slate-500 dark:text-gray-500"), g.Text(subtitle))),
			),
			Div(
				Class("rounded-full bg-violet-100 p-3 dark:bg-violet-900/30"),
				icon,
			),
		),
	)
}

// tokenCard renders a SCIM token card.
func tokenCard(token *SCIMToken, basePath string, appID xid.ID, onRevoke, onRotate string) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-gray-800 dark:bg-gray-900"),
		Div(
			Class("flex items-start justify-between"),
			Div(
				Class("flex-1"),
				Div(
					Class("flex items-center gap-2"),
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text(token.Name)),
					statusBadge(func() string {
						if token.RevokedAt != nil {
							return "revoked"
						}

						if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
							return "expired"
						}

						return "active"
					}()),
				),
				g.If(token.Description != "", P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"), g.Text(token.Description))),
				Div(
					Class("mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-gray-500"),
					Div(Class("flex items-center gap-1"),
						lucide.Key(Class("size-3")),
						g.Text("ID: "+token.ID.String()),
					),
					Div(Class("flex items-center gap-1"),
						lucide.Calendar(Class("size-3")),
						g.Text("Created: "+token.CreatedAt.Format("2006-01-02")),
					),
					g.If(token.ExpiresAt != nil,
						Div(Class("flex items-center gap-1"),
							lucide.Clock(Class("size-3")),
							g.Text("Expires: "+token.ExpiresAt.Format("2006-01-02")),
						),
					),
					g.If(token.LastUsedAt != nil,
						Div(Class("flex items-center gap-1"),
							lucide.Activity(Class("size-3")),
							g.Text("Last used: "+formatRelativeTime(*token.LastUsedAt)),
						),
					),
				),
				g.If(len(token.Scopes) > 0,
					Div(
						Class("mt-2 flex flex-wrap gap-1"),
						g.Group(scopeBadges(token.Scopes)),
					),
				),
			),
			Div(
				Class("flex gap-2"),
				g.If(token.RevokedAt == nil,
					Button(
						Type("button"),
						Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
						g.Attr("onclick", onRotate),
						lucide.RotateCw(Class("size-4")),
					),
				),
				g.If(token.RevokedAt == nil,
					Button(
						Type("button"),
						Class("text-sm text-red-600 hover:text-red-700 dark:text-red-400"),
						g.Attr("onclick", onRevoke),
						lucide.Trash2(Class("size-4")),
					),
				),
			),
		),
	)
}

// scopeBadges renders scope badges.
func scopeBadges(scopes []string) []g.Node {
	badges := make([]g.Node, len(scopes))
	for i, scope := range scopes {
		badges[i] = Span(
			Class("inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700 dark:bg-gray-800 dark:text-gray-300"),
			g.Text(scope),
		)
	}

	return badges
}

// providerCard renders a SCIM provider card.
func providerCard(provider *SCIMProvider, basePath string, appID xid.ID) g.Node {
	return Div(
		Class("rounded-lg border border-slate-200 bg-white p-4 shadow-sm hover:shadow-md transition-shadow dark:border-gray-800 dark:bg-gray-900"),
		A(
			Href(fmt.Sprintf("%s/dashboard/app/%s/settings/scim-providers/%s", basePath, appID.String(), provider.ID.String())),
			Class("block"),
			Div(
				Class("flex items-start justify-between"),
				Div(
					Class("flex-1"),
					Div(
						Class("flex items-center gap-2"),
						providerTypeIcon(provider.Type),
						H3(Class("text-lg font-semibold text-slate-900 dark:text-white"), g.Text(provider.Name)),
						statusBadge(provider.Status),
					),
					Div(
						Class("mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-gray-500"),
						Div(Class("flex items-center gap-1"),
							lucide.Tag(Class("size-3")),
							g.Text("Type: "+provider.Type),
						),
						Div(Class("flex items-center gap-1"),
							directionIcon(provider.Direction),
							g.Text("Direction: "+provider.Direction),
						),
						g.If(provider.LastSyncAt != nil,
							Div(Class("flex items-center gap-1"),
								lucide.RefreshCw(Class("size-3")),
								g.Text("Last sync: "+formatRelativeTime(*provider.LastSyncAt)),
							),
						),
					),
					g.If(provider.LastSyncStatus != "",
						Div(
							Class("mt-2"),
							Span(
								Class(fmt.Sprintf("text-xs %s", func() string {
									if provider.LastSyncStatus == "success" {
										return "text-green-600 dark:text-green-400"
									}

									return "text-red-600 dark:text-red-400"
								}())),
								g.Text("Last sync: "+provider.LastSyncStatus),
							),
						),
					),
				),
				Div(
					lucide.ChevronRight(Class("size-5 text-slate-400")),
				),
			),
		),
	)
}

// providerTypeIcon returns an icon for a provider type.
func providerTypeIcon(providerType string) g.Node {
	switch providerType {
	case "okta":
		return Div(
			Class("rounded-full bg-blue-100 p-2 dark:bg-blue-900/30"),
			lucide.Cloud(Class("size-4 text-blue-600 dark:text-blue-400")),
		)
	case "azure_ad", "azure":
		return Div(
			Class("rounded-full bg-cyan-100 p-2 dark:bg-cyan-900/30"),
			lucide.Cloud(Class("size-4 text-cyan-600 dark:text-cyan-400")),
		)
	case "onelogin":
		return Div(
			Class("rounded-full bg-orange-100 p-2 dark:bg-orange-900/30"),
			lucide.Cloud(Class("size-4 text-orange-600 dark:text-orange-400")),
		)
	case "google", "google_workspace":
		return Div(
			Class("rounded-full bg-red-100 p-2 dark:bg-red-900/30"),
			lucide.Cloud(Class("size-4 text-red-600 dark:text-red-400")),
		)
	default:
		return Div(
			Class("rounded-full bg-gray-100 p-2 dark:bg-gray-800"),
			lucide.Cloud(Class("size-4 text-gray-600 dark:text-gray-400")),
		)
	}
}

// directionIcon returns an icon for sync direction.
func directionIcon(direction string) g.Node {
	switch direction {
	case "inbound":
		return lucide.ArrowDownToLine(Class("size-3"))
	case "outbound":
		return lucide.ArrowUpFromLine(Class("size-3"))
	case "bidirectional":
		return lucide.ArrowLeftRight(Class("size-3"))
	default:
		return lucide.Circle(Class("size-3"))
	}
}

// syncEventRow renders a sync event table row.
func syncEventRow(event *SCIMSyncEvent) g.Node {
	return Tr(
		Class("border-b border-slate-200 dark:border-gray-800 hover:bg-slate-50 dark:hover:bg-gray-800/50"),
		Td(Class("px-4 py-3 text-sm"),
			Div(Class("flex items-center gap-2"),
				statusIcon(event.Status),
				g.Text(event.EventType),
			),
		),
		Td(Class("px-4 py-3 text-sm text-slate-600 dark:text-gray-400"),
			g.Text(event.ResourceType),
		),
		Td(Class("px-4 py-3 text-sm"),
			statusBadge(event.Status),
		),
		Td(Class("px-4 py-3 text-sm text-slate-600 dark:text-gray-400"),
			directionIcon(event.Direction),
			Span(Class("ml-1"), g.Text(event.Direction)),
		),
		Td(Class("px-4 py-3 text-sm text-slate-600 dark:text-gray-400"),
			g.If(event.Duration > 0,
				g.Textf("%dms", event.Duration),
			),
		),
		Td(Class("px-4 py-3 text-sm text-slate-500 dark:text-gray-500"),
			g.Text(formatRelativeTime(event.CreatedAt)),
		),
	)
}

// configField renders a configuration field with override support.
func configField(label, value string, isOverridden bool, canOverride bool, overrideAction string) g.Node {
	return Div(
		Class("flex items-center justify-between py-3 border-b border-slate-200 dark:border-gray-800"),
		Div(
			Class("flex-1"),
			Label(
				Class("block text-sm font-medium text-slate-700 dark:text-gray-300"),
				g.Text(label),
			),
			Div(
				Class("mt-1 flex items-center gap-2"),
				g.If(isOverridden,
					Span(
						Class("inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"),
						g.Text("Overridden"),
					),
				),
				g.If(!isOverridden,
					Span(
						Class("inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-400"),
						g.Text("Inherited"),
					),
				),
				Span(Class("text-sm text-slate-900 dark:text-white"), g.Text(value)),
			),
		),
		g.If(canOverride,
			Div(
				g.If(isOverridden,
					Button(
						Type("button"),
						Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
						g.Attr("onclick", overrideAction),
						g.Text("Reset to Default"),
					),
				),
				g.If(!isOverridden,
					Button(
						Type("button"),
						Class("text-sm text-violet-600 hover:text-violet-700 dark:text-violet-400"),
						g.Attr("onclick", overrideAction),
						g.Text("Override"),
					),
				),
			),
		),
	)
}

// alertBox renders an alert box.
func alertBox(alertType, title, message string) g.Node {
	var bgClass, borderClass, textClass, iconNode string

	switch alertType {
	case "success":
		bgClass = "bg-green-50 dark:bg-green-900/20"
		borderClass = "border-green-200 dark:border-green-800"
		textClass = "text-green-800 dark:text-green-400"
		iconNode = "CheckCircle"
	case "warning":
		bgClass = "bg-yellow-50 dark:bg-yellow-900/20"
		borderClass = "border-yellow-200 dark:border-yellow-800"
		textClass = "text-yellow-800 dark:text-yellow-400"
		iconNode = "AlertTriangle"
	case "error":
		bgClass = "bg-red-50 dark:bg-red-900/20"
		borderClass = "border-red-200 dark:border-red-800"
		textClass = "text-red-800 dark:text-red-400"
		iconNode = "XCircle"
	default: // info
		bgClass = "bg-blue-50 dark:bg-blue-900/20"
		borderClass = "border-blue-200 dark:border-blue-800"
		textClass = "text-blue-800 dark:text-blue-400"
		iconNode = "Info"
	}

	var icon g.Node

	switch iconNode {
	case "CheckCircle":
		icon = lucide.Check(Class("size-5 " + textClass))
	case "AlertTriangle":
		icon = lucide.Info(Class("size-5 " + textClass))
	case "XCircle":
		icon = lucide.X(Class("size-5 " + textClass))
	default:
		icon = lucide.Info(Class("size-5 " + textClass))
	}

	return Div(
		Class(fmt.Sprintf("rounded-lg border p-4 %s %s", bgClass, borderClass)),
		Div(
			Class("flex gap-3"),
			icon,
			Div(
				g.If(title != "", Div(Class(fmt.Sprintf("font-medium %s", textClass)), g.Text(title))),
				Div(Class(fmt.Sprintf("text-sm %s", textClass)), g.Text(message)),
			),
		),
	)
}

// emptyState renders an empty state message.
func emptyState(icon g.Node, title, description, actionText, actionURL string) g.Node {
	return Div(
		Class("flex flex-col items-center justify-center py-12"),
		Div(
			Class("rounded-full bg-slate-100 p-4 dark:bg-gray-800"),
			icon,
		),
		H3(Class("mt-4 text-lg font-semibold text-slate-900 dark:text-white"), g.Text(title)),
		P(Class("mt-2 text-sm text-slate-600 dark:text-gray-400 text-center max-w-sm"), g.Text(description)),
		g.If(actionText != "" && actionURL != "",
			A(
				Href(actionURL),
				Class("mt-4 inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-medium text-white hover:bg-violet-700"),
				g.Text(actionText),
				lucide.ArrowRight(Class("size-4")),
			),
		),
	)
}

// formatRelativeTime formats a time as a relative string (e.g., "5 minutes ago").
func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}

// loadingSpinner renders a loading spinner.
func loadingSpinner() g.Node {
	return Div(
		Class("flex items-center justify-center py-8"),
		Div(
			Class("animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"),
		),
	)
}

// pagination renders pagination controls.
func pagination(currentPage, totalPages int, baseURL string) g.Node {
	if totalPages <= 1 {
		return nil
	}

	items := make([]g.Node, 0)

	// Previous button
	if currentPage > 1 {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Previous"),
		))
	}

	// Page numbers (simplified - show current and nearby)
	for i := 1; i <= totalPages; i++ {
		if i == currentPage {
			items = append(items, Span(
				Class("px-3 py-2 text-sm font-medium text-white bg-violet-600 border border-violet-600 rounded-md"),
				g.Textf("%d", i),
			))
		} else if i == 1 || i == totalPages || (i >= currentPage-1 && i <= currentPage+1) {
			items = append(items, A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
				Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
				g.Textf("%d", i),
			))
		} else if i == currentPage-2 || i == currentPage+2 {
			items = append(items, Span(
				Class("px-2 py-2 text-slate-400"),
				g.Text("..."),
			))
		}
	}

	// Next button
	if currentPage < totalPages {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			g.Text("Next"),
		))
	}

	return Div(
		Class("flex items-center justify-center gap-2 mt-6"),
		g.Group(items),
	)
}
