package pages

import (
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ClientTypeBadge renders a badge for client application type
func ClientTypeBadge(appType string) g.Node {
	return Span(
		Class("inline-flex items-center rounded-md bg-gray-100 dark:bg-gray-800 px-2 py-1 text-xs font-medium capitalize"),
		g.Text(appType),
	)
}

// GrantTypeBadges renders badges for grant types
func GrantTypeBadges(grantTypes []string) []g.Node {
	badges := make([]g.Node, len(grantTypes))
	for i, grant := range grantTypes {
		badges[i] = Span(
			Class("inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium"),
			g.Text(grant),
		)
	}
	return badges
}

// ScopeBadges renders badges for scopes
func ScopeBadges(scopes []string) []g.Node {
	badges := make([]g.Node, len(scopes))
	for i, scope := range scopes {
		badges[i] = Span(
			Class("inline-flex items-center rounded-md bg-blue-100 dark:bg-blue-900 px-2 py-1 text-xs font-medium text-blue-700 dark:text-blue-300"),
			g.Text(scope),
		)
	}
	return badges
}

// StatusBadge renders a status badge
func StatusBadge(status string) g.Node {
	var bgClass, textClass, text string
	
	switch status {
	case "pending":
		bgClass = "bg-yellow-100 dark:bg-yellow-900"
		textClass = "text-yellow-700 dark:text-yellow-300"
		text = "Pending"
	case "authorized":
		bgClass = "bg-green-100 dark:bg-green-900"
		textClass = "text-green-700 dark:text-green-300"
		text = "Authorized"
	case "denied":
		bgClass = "bg-red-100 dark:bg-red-900"
		textClass = "text-red-700 dark:text-red-300"
		text = "Denied"
	case "expired":
		bgClass = "bg-gray-100 dark:bg-gray-800"
		textClass = "text-gray-700 dark:text-gray-300"
		text = "Expired"
	case "consumed":
		bgClass = "bg-blue-100 dark:bg-blue-900"
		textClass = "text-blue-700 dark:text-blue-300"
		text = "Consumed"
	default:
		bgClass = "bg-gray-100 dark:bg-gray-800"
		textClass = "text-gray-700 dark:text-gray-300"
		text = status
	}
	
	return Span(
		Class("inline-flex items-center rounded-md px-2 py-1 text-xs font-medium "+bgClass+" "+textClass),
		g.Text(text),
	)
}
