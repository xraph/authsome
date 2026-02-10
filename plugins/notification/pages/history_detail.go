package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// notificationDetailModal renders a modal for viewing notification details.
func notificationDetailModal(basePath string, appID string) g.Node {
	return Div(
		g.Attr("x-show", "showDetailModal"),
		g.Attr("@keydown.escape.window", "closeDetail()"),
		g.Attr("x-transition:enter", "transition ease-out duration-300"),
		g.Attr("x-transition:enter-start", "opacity-0"),
		g.Attr("x-transition:enter-end", "opacity-100"),
		g.Attr("x-transition:leave", "transition ease-in duration-200"),
		g.Attr("x-transition:leave-start", "opacity-100"),
		g.Attr("x-transition:leave-end", "opacity-0"),
		Class("fixed inset-0 z-50 overflow-y-auto"),
		g.Attr("aria-labelledby", "modal-title"),
		g.Attr("role", "dialog"),
		g.Attr("aria-modal", "true"),

		// Backdrop
		Div(
			Class("fixed inset-0 bg-black/50 transition-opacity"),
			g.Attr("@click", "closeDetail()"),
		),

		// Modal panel
		Div(
			Class("flex min-h-full items-center justify-center p-4"),
			Div(
				g.Attr("@click.stop", ""),
				g.Attr("x-show", "selectedNotification"),
				Class("relative w-full max-w-3xl transform overflow-hidden rounded-lg bg-card shadow-xl transition-all"),

				// Modal header
				Div(
					Class("flex items-center justify-between border-b p-6"),
					Div(
						H3(
							ID("modal-title"),
							Class("text-lg font-semibold"),
							g.Text("Notification Details"),
						),
						P(
							Class("mt-1 text-sm text-muted-foreground"),
							g.Attr("x-show", "selectedNotification"),
							g.Attr("x-text", "'ID: ' + (selectedNotification?.id || '')"),
						),
					),
					Button(
						Type("button"),
						g.Attr("@click", "closeDetail()"),
						Class("rounded-md p-2 text-muted-foreground hover:bg-accent"),
						lucide.X(Class("h-5 w-5")),
					),
				),

				// Modal body
				Div(
					Class("max-h-[calc(100vh-200px)] overflow-y-auto p-6"),
					g.Attr("x-show", "selectedNotification"),

					// Notification info grid
					Div(
						Class("space-y-6"),

						// Type and Status row
						Div(
							Class("grid gap-6 md:grid-cols-2"),

							// Type
							Div(
								Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Type")),
								Div(
									Class("mt-1.5 flex items-center gap-2"),
									Span(
										g.Attr("x-show", "selectedNotification?.type === 'email'"),
										lucide.Mail(Class("h-5 w-5 text-blue-600")),
									),
									Span(
										g.Attr("x-show", "selectedNotification?.type === 'sms'"),
										lucide.MessageSquare(Class("h-5 w-5 text-green-600")),
									),
									Span(
										g.Attr("x-show", "selectedNotification?.type === 'push'"),
										lucide.Bell(Class("h-5 w-5 text-purple-600")),
									),
									Span(
										Class("capitalize text-base font-medium"),
										g.Attr("x-text", "selectedNotification?.type"),
									),
								),
							),

							// Status
							Div(
								Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Status")),
								Div(
									Class("mt-1.5"),
									Span(
										Class("inline-flex items-center rounded-full px-3 py-1 text-sm font-medium"),
										g.Attr(":class", `{
											'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400': selectedNotification?.status === 'delivered',
											'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400': selectedNotification?.status === 'sent',
											'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400': selectedNotification?.status === 'failed' || selectedNotification?.status === 'bounced',
											'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400': selectedNotification?.status === 'pending'
										}`),
										Span(Class("capitalize"), g.Attr("x-text", "selectedNotification?.status")),
									),
								),
							),
						),

						// Recipient
						Div(
							Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Recipient")),
							P(Class("mt-1.5 text-base font-medium"), g.Attr("x-text", "selectedNotification?.recipient")),
						),

						// Subject (for email)
						Div(
							g.Attr("x-show", "selectedNotification?.subject"),
							Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Subject")),
							P(Class("mt-1.5 text-base"), g.Attr("x-text", "selectedNotification?.subject")),
						),

						// Body
						Div(
							g.Attr("x-data", "{ showSource: false }"),
							Div(
								Class("flex items-center justify-between"),
								Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Content")),
								// Toggle between rendered and source view for emails
								Button(
									Type("button"),
									g.Attr("x-show", "selectedNotification?.type === 'email'"),
									g.Attr("@click", "showSource = !showSource"),
									Class("text-xs text-muted-foreground hover:text-foreground underline"),
									Span(g.Attr("x-text", "showSource ? 'View Rendered' : 'View Source'")),
								),
							),
							// Rendered HTML view (for emails)
							Div(
								g.Attr("x-show", "selectedNotification?.type === 'email' && !showSource"),
								Class("mt-1.5 rounded-md border bg-white dark:bg-gray-900"),
								g.El("iframe",
									g.Attr("x-ref", "emailFrame"),
									g.Attr("x-init", `
										$watch('selectedNotification', (value) => {
											if (value && value.type === 'email' && value.body && $refs.emailFrame) {
												const frame = $refs.emailFrame;
												const doc = frame.contentDocument || frame.contentWindow.document;
												doc.open();
												doc.write(value.body);
												doc.close();
											}
										})
									`),
									Class("w-full min-h-[400px] max-h-[600px] border-0"),
									g.Attr("sandbox", "allow-same-origin"),
								),
							),
							// Source view (for emails when toggled, or default for SMS/push)
							Div(
								g.Attr("x-show", "(selectedNotification?.type === 'email' && showSource) || selectedNotification?.type !== 'email'"),
								Class("mt-1.5 rounded-md border bg-muted/50 p-4"),
								Pre(
									Class("max-h-96 overflow-auto whitespace-pre-wrap text-sm"),
									g.Attr("x-text", "selectedNotification?.body"),
								),
							),
						),

						// Error (if failed)
						Div(
							g.Attr("x-show", "selectedNotification?.error"),
							Label(Class("text-sm font-medium text-red-600 dark:text-red-400"), g.Text("Error")),
							Div(
								Class("mt-1.5 rounded-md border border-red-200 bg-red-50 p-3 dark:border-red-900 dark:bg-red-900/20"),
								P(Class("text-sm text-red-800 dark:text-red-300"), g.Attr("x-text", "selectedNotification?.error")),
							),
						),

						// Timestamps
						Div(
							Class("grid gap-4 md:grid-cols-3"),
							Div(
								Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Created")),
								P(Class("mt-1.5 text-sm"), g.Attr("x-text", "formatDate(selectedNotification?.createdAt)")),
							),
							Div(
								g.Attr("x-show", "selectedNotification?.sentAt"),
								Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Sent")),
								P(Class("mt-1.5 text-sm"), g.Attr("x-text", "formatDate(selectedNotification?.sentAt)")),
							),
							Div(
								g.Attr("x-show", "selectedNotification?.deliveredAt"),
								Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Delivered")),
								P(Class("mt-1.5 text-sm"), g.Attr("x-text", "formatDate(selectedNotification?.deliveredAt)")),
							),
						),

						// Provider ID (if available)
						Div(
							g.Attr("x-show", "selectedNotification?.providerId"),
							Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Provider ID")),
							P(
								Class("mt-1.5 font-mono text-xs text-muted-foreground"),
								g.Attr("x-text", "selectedNotification?.providerId"),
							),
						),

						// Template ID (if used)
						Div(
							g.Attr("x-show", "selectedNotification?.templateId"),
							Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Template")),
							Div(
								Class("mt-1.5"),
								A(
									g.Attr(":href", "`"+basePath+"/app/"+appID+"/notifications/templates/${selectedNotification?.templateId}`"),
									Class("inline-flex items-center gap-1 text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"),
									lucide.FileText(Class("h-4 w-4")),
									Span(g.Text("View Template")),
								),
							),
						),

						// Metadata (if available)
						Div(
							g.Attr("x-show", "selectedNotification?.metadata && Object.keys(selectedNotification.metadata).length > 0"),
							Label(Class("text-sm font-medium text-muted-foreground"), g.Text("Metadata")),
							Div(
								Class("mt-1.5 rounded-md border bg-muted/50 p-3"),
								Pre(
									Class("max-h-48 overflow-auto whitespace-pre-wrap font-mono text-xs"),
									g.Attr("x-text", "JSON.stringify(selectedNotification?.metadata, null, 2)"),
								),
							),
						),
					),
				),

				// Modal footer
				Div(
					Class("flex items-center justify-end gap-3 border-t p-6"),
					Button(
						Type("button"),
						g.Attr("@click", "closeDetail()"),
						Class("rounded-md border border-input bg-background px-4 py-2 text-sm font-medium hover:bg-accent"),
						g.Text("Close"),
					),
				),
			),
		),
	)
}
