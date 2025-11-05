package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Notifications Tab Content
func notificationsTabContent(data SettingsPageData) g.Node {
	return Div(
		g.Attr("x-show", "activeTab === 'notifications'"),
		g.Attr("x-cloak", ""),
		Div(Class("space-y-6"),
			// Header with Create Button
			Div(Class("flex items-center justify-between"),
				Div(
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Notification Templates"),
					),
					P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
						g.Text("Manage email and SMS notification templates"),
					),
				),
				Button(
					Type("button"),
					g.Attr("@click", fmt.Sprintf("window.location.href='%s/dashboard/settings/notifications/new'", data.BasePath)),
					Class("inline-flex items-center gap-2 rounded-lg border border-transparent bg-violet-600 px-4 py-2 text-sm leading-5 font-semibold text-white hover:bg-violet-700 transition-colors"),
					lucide.Plus(Class("h-4 w-4")),
					g.Text("Create Template"),
				),
			),

			// Templates List or Empty State
			g.If(len(data.NotificationTemplates) > 0,
				notificationTemplatesList(data.NotificationTemplates, data.BasePath),
			),
			g.If(len(data.NotificationTemplates) == 0,
				emptyState(
					lucide.Mail(Class("h-8 w-8 text-slate-400 dark:text-gray-500")),
					"No Templates",
					"Create your first notification template",
					"Templates can be customized per organization in SaaS mode",
				),
			),
		),
	)
}

func notificationTemplatesList(templates []NotificationTemplate, basePath string) g.Node {
	return Div(Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden"),
		Div(Class("p-5"),
			Div(Class("min-w-full overflow-x-auto rounded-sm"),
				Table(Class("min-w-full align-middle text-sm"),
					THead(
						Tr(Class("border-b-2 border-slate-100 dark:border-gray-800"),
							Th(Class("min-w-[200px] py-3 pe-3 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Name"),
							),
							Th(Class("min-w-[100px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Type"),
							),
							Th(Class("min-w-[80px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Language"),
							),
							Th(Class("min-w-[140px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Updated"),
							),
							Th(Class("min-w-[150px] py-2 ps-3 text-end text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Actions"),
							),
						),
					),
					TBody(
						g.Group(notificationTemplateRows(templates, basePath)),
					),
				),
			),
		),
	)
}

func notificationTemplateRows(templates []NotificationTemplate, basePath string) []g.Node {
	rows := make([]g.Node, 0, len(templates))
	for _, tmpl := range templates {
		rows = append(rows, notificationTemplateRow(tmpl, basePath))
	}
	return rows
}

func notificationTemplateRow(tmpl NotificationTemplate, basePath string) g.Node {
	typeIcon := lucide.Mail(Class("h-4 w-4"))
	if tmpl.Type == "sms" {
		typeIcon = lucide.MessageSquare(Class("h-4 w-4"))
	}

	return Tr(Class("border-b border-slate-100 dark:border-gray-800 hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),
		Td(Class("py-3 pe-3 text-start"),
			Div(Class("flex items-center gap-2"),
				typeIcon,
				Div(
					Div(Class("font-medium text-slate-900 dark:text-white"),
						g.Text(tmpl.Name),
					),
					g.If(tmpl.Subject != "",
						Div(Class("text-xs text-slate-600 dark:text-gray-400"),
							g.Text(tmpl.Subject),
						),
					),
				),
			),
		),
		Td(Class("p-3"),
			Span(Class("inline-flex items-center rounded-full bg-violet-100 dark:bg-violet-500/20 px-2.5 py-0.5 text-xs font-semibold text-violet-800 dark:text-violet-400"),
				g.Text(tmpl.Type),
			),
		),
		Td(Class("p-3 text-slate-600 dark:text-gray-400 uppercase"),
			g.Text(tmpl.Language),
		),
		Td(Class("p-3 text-slate-600 dark:text-gray-400"),
			g.Text(tmpl.Updated.Format("Jan 2, 2006")),
		),
		Td(Class("py-3 ps-3 text-end font-medium flex justify-end gap-2"),
			Button(
				Type("button"),
				Class("text-violet-600 dark:text-violet-400 hover:text-violet-800 dark:hover:text-violet-300 transition-colors"),
				g.Text("Edit"),
			),
			Button(
				Type("button"),
				Class("text-rose-600 dark:text-rose-400 hover:text-rose-800 dark:hover:text-rose-300 transition-colors"),
				g.Text("Delete"),
			),
		),
	)
}

// Social Providers Tab Content
func socialProvidersTabContent(data SettingsPageData) g.Node {
	return Div(
		g.Attr("x-show", "activeTab === 'social'"),
		g.Attr("x-cloak", ""),
		Div(Class("space-y-6"),
			// Header
			Div(Class("flex items-center justify-between"),
				Div(
					H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text("Social Login Providers"),
					),
					P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
						g.Text("Configure OAuth providers for social authentication"),
					),
				),
				Button(
					Type("button"),
					g.Attr("@click", fmt.Sprintf("window.location.href='%s/dashboard/settings/social/new'", data.BasePath)),
					Class("inline-flex items-center gap-2 rounded-lg border border-transparent bg-violet-600 px-4 py-2 text-sm leading-5 font-semibold text-white hover:bg-violet-700 transition-colors"),
					lucide.Plus(Class("h-4 w-4")),
					g.Text("Add Provider"),
				),
			),

			// Providers Grid or Empty State
			g.If(len(data.SocialProviders) > 0,
				socialProvidersGrid(data.SocialProviders),
			),
			g.If(len(data.SocialProviders) == 0,
				emptyState(
					lucide.Users(Class("h-8 w-8 text-slate-400 dark:text-gray-500")),
					"No OAuth Providers",
					"Add your first OAuth provider to enable social login",
					"Supports Google, GitHub, Microsoft, Apple, and 13+ more providers",
				),
			),
		),
	)
}

func socialProvidersGrid(providers []SocialProvider) g.Node {
	items := make([]g.Node, 0, len(providers))
	for _, provider := range providers {
		items = append(items, socialProviderCard(provider))
	}

	return Div(Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"),
		g.Group(items),
	)
}

func socialProviderCard(provider SocialProvider) g.Node {
	return Div(Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-5 hover:shadow-md transition-shadow"),
		// Header with Status
		Div(Class("flex items-start justify-between mb-3"),
			H4(Class("text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text(provider.Name),
			),
			g.If(provider.Enabled,
				Span(Class("inline-flex items-center rounded-full bg-emerald-100 dark:bg-emerald-500/20 px-2.5 py-0.5 text-xs leading-4 font-semibold text-emerald-800 dark:text-emerald-400"),
					g.Text("Active"),
				),
			),
			g.If(!provider.Enabled,
				Span(Class("inline-flex items-center rounded-full bg-slate-100 dark:bg-gray-800 px-2.5 py-0.5 text-xs leading-4 font-semibold text-slate-800 dark:text-gray-400"),
					g.Text("Inactive"),
				),
			),
		),

		// Provider Details
		Div(Class("space-y-2 text-sm mb-4"),
			Div(
				Span(Class("text-slate-600 dark:text-gray-400"), g.Text("Provider: ")),
				Span(Class("font-mono text-slate-900 dark:text-white"), g.Text(provider.Provider)),
			),
			Div(
				Span(Class("text-slate-600 dark:text-gray-400"), g.Text("Client ID: ")),
				Span(Class("font-mono text-xs text-slate-900 dark:text-white"), g.Text(truncateClientID(provider.ClientID))),
			),
		),

		// Actions
		Div(Class("flex gap-2 mt-auto pt-4 border-t border-slate-100 dark:border-gray-800"),
			Button(
				Type("button"),
				Class("flex-1 inline-flex items-center justify-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-3 py-1.5 text-xs font-medium text-slate-800 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-800 transition-colors"),
				lucide.Settings(Class("h-3 w-3")),
				g.Text("Configure"),
			),
			Button(
				Type("button"),
				Class("inline-flex items-center justify-center gap-1 rounded-lg border border-rose-200 dark:border-rose-800 px-3 py-1.5 text-xs font-medium text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-900/20 transition-colors"),
				lucide.Trash2(Class("h-3 w-3")),
			),
		),
	)
}

func truncateClientID(clientID string) string {
	if len(clientID) > 20 {
		return clientID[:17] + "..."
	}
	return clientID
}

// Impersonation Logs Tab Content
func impersonationTabContent(data SettingsPageData) g.Node {
	return Div(
		g.Attr("x-show", "activeTab === 'impersonation'"),
		g.Attr("x-cloak", ""),
		Div(Class("space-y-6"),
			// Header
			Div(
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("Impersonation Audit Log"),
				),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Track all impersonation sessions for security and compliance"),
				),
			),

			// Logs Table or Empty State
			g.If(len(data.ImpersonationLogs) > 0,
				impersonationLogsTable(data.ImpersonationLogs),
			),
			g.If(len(data.ImpersonationLogs) == 0,
				emptyState(
					lucide.UserRound(Class("h-8 w-8 text-slate-400 dark:text-gray-500")),
					"No Impersonation Logs",
					"Impersonation sessions will appear here for audit purposes",
					"All sessions are automatically logged for compliance",
				),
			),
		),
	)
}

func impersonationLogsTable(logs []ImpersonationLog) g.Node {
	return Div(Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden"),
		Div(Class("p-5"),
			Div(Class("min-w-full overflow-x-auto rounded-sm"),
				Table(Class("min-w-full align-middle text-sm"),
					THead(
						Tr(Class("border-b-2 border-slate-100 dark:border-gray-800"),
							Th(Class("min-w-[180px] py-3 pe-3 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Admin"),
							),
							Th(Class("min-w-[180px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Impersonated User"),
							),
							Th(Class("min-w-[200px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Reason"),
							),
							Th(Class("min-w-[140px] px-3 py-2 text-start text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Started"),
							),
							Th(Class("min-w-[100px] py-2 ps-3 text-end text-sm font-semibold tracking-wider text-slate-700 dark:text-gray-300 uppercase"),
								g.Text("Duration"),
							),
						),
					),
					TBody(
						g.Group(impersonationLogRows(logs)),
					),
				),
			),
		),
	)
}

func impersonationLogRows(logs []ImpersonationLog) []g.Node {
	rows := make([]g.Node, 0, len(logs))
	for _, log := range logs {
		rows = append(rows, impersonationLogRow(log))
	}
	return rows
}

func impersonationLogRow(log ImpersonationLog) g.Node {
	statusColor := "bg-yellow-100 dark:bg-yellow-500/20 text-yellow-800 dark:text-yellow-400"
	statusText := "Active"
	if log.EndedAt != nil {
		statusColor = "bg-slate-100 dark:bg-gray-800 text-slate-800 dark:text-gray-400"
		statusText = "Ended"
	}

	return Tr(Class("border-b border-slate-100 dark:border-gray-800 hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),
		Td(Class("py-3 pe-3 text-start"),
			Div(Class("font-medium text-slate-900 dark:text-white"),
				g.Text(log.AdminEmail),
			),
			Div(Class("text-xs font-mono text-slate-600 dark:text-gray-400"),
				g.Text(log.AdminID),
			),
		),
		Td(Class("p-3"),
			Div(Class("font-medium text-slate-900 dark:text-white"),
				g.Text(log.UserEmail),
			),
			Div(Class("text-xs font-mono text-slate-600 dark:text-gray-400"),
				g.Text(log.UserID),
			),
		),
		Td(Class("p-3 text-slate-600 dark:text-gray-400"),
			g.Text(log.Reason),
		),
		Td(Class("p-3 text-slate-600 dark:text-gray-400"),
			g.Text(log.StartedAt.Format("Jan 2, 15:04")),
		),
		Td(Class("py-3 ps-3 text-end"),
			Span(Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold "+statusColor),
				g.Text(statusText),
			),
			Div(Class("text-xs text-slate-600 dark:text-gray-400 mt-1"),
				g.Text(log.Duration),
			),
		),
	)
}

// MFA Settings Tab Content
func mfaTabContent(data SettingsPageData) g.Node {
	return Div(
		g.Attr("x-show", "activeTab === 'mfa'"),
		g.Attr("x-cloak", ""),
		Div(Class("space-y-6"),
			// Header
			Div(
				H3(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("Multi-Factor Authentication"),
				),
				P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Configure available MFA methods for enhanced security"),
				),
			),

			// MFA Methods Grid
			g.If(len(data.MFAMethods) > 0,
				mfaMethodsGrid(data.MFAMethods),
			),
			g.If(len(data.MFAMethods) == 0,
				emptyState(
					lucide.ShieldCheck(Class("h-8 w-8 text-slate-400 dark:text-gray-500")),
					"No MFA Methods",
					"Configure MFA methods to enhance account security",
					"Supports TOTP, SMS, Email, and WebAuthn",
				),
			),
		),
	)
}

func mfaMethodsGrid(methods []MFAMethod) g.Node {
	items := make([]g.Node, 0, len(methods))
	for _, method := range methods {
		items = append(items, mfaMethodCard(method))
	}

	return Div(Class("grid grid-cols-1 md:grid-cols-2 gap-4"),
		g.Group(items),
	)
}

func mfaMethodCard(method MFAMethod) g.Node {
	icon := getMFAIcon(method.Type)

	return Div(Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-6"),
		// Header
		Div(Class("flex items-start justify-between mb-4"),
			Div(Class("flex items-center gap-3"),
				Div(Class("h-12 w-12 rounded-xl bg-violet-50 dark:bg-violet-900/20 flex items-center justify-center"),
					icon,
				),
				Div(
					H4(Class("text-lg font-semibold text-slate-900 dark:text-white"),
						g.Text(method.Name),
					),
					P(Class("text-sm text-slate-600 dark:text-gray-400"),
						g.Text(getMFADescription(method.Type)),
					),
				),
			),
		),

		// Status Toggle
		Div(Class("flex items-center justify-between pt-4 border-t border-slate-100 dark:border-gray-800"),
			Div(Class("flex items-center gap-2"),
				Input(
					Type("checkbox"),
					ID("mfa_"+method.ID),
					g.If(method.Enabled, Checked()),
					Class("h-4 w-4 rounded border-slate-300 dark:border-gray-600 text-violet-600 focus:ring-violet-500"),
				),
				Label(
					For("mfa_"+method.ID),
					Class("text-sm font-medium text-slate-900 dark:text-white"),
					g.Text("Enable for users"),
				),
			),
			Button(
				Type("button"),
				Class("text-sm text-violet-600 dark:text-violet-400 hover:text-violet-700 dark:hover:text-violet-300 transition-colors"),
				g.Text("Configure →"),
			),
		),
	)
}

func getMFAIcon(mfaType string) g.Node {
	switch mfaType {
	case "totp":
		return lucide.Smartphone(Class("h-6 w-6 text-violet-600 dark:text-violet-400"))
	case "sms":
		return lucide.MessageSquare(Class("h-6 w-6 text-violet-600 dark:text-violet-400"))
	case "email":
		return lucide.Mail(Class("h-6 w-6 text-violet-600 dark:text-violet-400"))
	case "webauthn":
		return lucide.Fingerprint(Class("h-6 w-6 text-violet-600 dark:text-violet-400"))
	default:
		return lucide.ShieldCheck(Class("h-6 w-6 text-violet-600 dark:text-violet-400"))
	}
}

func getMFADescription(mfaType string) string {
	switch mfaType {
	case "totp":
		return "Time-based one-time passwords (Google Authenticator, Authy)"
	case "sms":
		return "SMS-based verification codes"
	case "email":
		return "Email-based verification codes"
	case "webauthn":
		return "Hardware keys and biometric authentication"
	default:
		return "Multi-factor authentication method"
	}
}
