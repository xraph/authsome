package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/forgeui/primitives"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// HistoryListPage renders the notification history/logs page
func HistoryListPage(currentApp *app.App, basePath string) g.Node {
	return primitives.Container(
		Div(
			Class("space-y-6"),

			// Page header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-2xl font-bold"), g.Text("Notification History")),
					P(Class("mt-1 text-sm text-muted-foreground"),
						g.Text("View all sent emails and SMS messages")),
				),
			),

			// Main content with Alpine.js data
			Div(
				g.Attr("x-data", historyListData(basePath, currentApp.ID.String())),
				g.Attr("x-init", "await loadNotifications()"),
				Class("space-y-6"),

				// Filters section
				Div(
					Class("rounded-lg border bg-card p-4"),
					Div(
						Class("grid gap-4 md:grid-cols-4"),

						// Type filter
						Div(
							Label(Class("text-sm font-medium"), g.Text("Type")),
							Select(
								g.Attr("x-model", "filters.type"),
								g.Attr("@change", "currentPage = 1; await loadNotifications()"),
								Class("mt-1.5 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
								Option(Value(""), g.Text("All Types")),
								Option(Value("email"), g.Text("Email")),
								Option(Value("sms"), g.Text("SMS")),
								Option(Value("push"), g.Text("Push")),
							),
						),

						// Status filter
						Div(
							Label(Class("text-sm font-medium"), g.Text("Status")),
							Select(
								g.Attr("x-model", "filters.status"),
								g.Attr("@change", "currentPage = 1; await loadNotifications()"),
								Class("mt-1.5 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
								Option(Value(""), g.Text("All Status")),
								Option(Value("pending"), g.Text("Pending")),
								Option(Value("sent"), g.Text("Sent")),
								Option(Value("delivered"), g.Text("Delivered")),
								Option(Value("failed"), g.Text("Failed")),
								Option(Value("bounced"), g.Text("Bounced")),
							),
						),

						// Recipient search
						Div(
							Label(Class("text-sm font-medium"), g.Text("Recipient")),
							Input(
								Type("text"),
								g.Attr("x-model", "filters.recipient"),
								g.Attr("@input.debounce.500ms", "currentPage = 1; await loadNotifications()"),
								Placeholder("Search by email or phone..."),
								Class("mt-1.5 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
							),
						),

						// Clear filters button
						Div(
							Class("flex items-end"),
							Button(
								Type("button"),
								g.Attr("@click", "clearFilters()"),
								Class("w-full rounded-md border border-input bg-background px-4 py-2 text-sm font-medium hover:bg-accent"),
								g.Text("Clear Filters"),
							),
						),
					),
				),

				// Loading state
				Div(
					g.Attr("x-show", "loading"),
					LoadingSpinner(),
				),

				// Error message
				ErrorMessage("error && !loading"),

				// Notifications table
				Div(
					g.Attr("x-show", "!loading && !error"),
					Class("rounded-lg border bg-card"),
					Div(
						Class("overflow-x-auto"),
						Table(
							Class("w-full text-sm"),
							THead(
								Class("border-b bg-muted/50"),
								Tr(
									Th(Class("px-4 py-3 text-left font-medium"), g.Text("Type")),
									Th(Class("px-4 py-3 text-left font-medium"), g.Text("Recipient")),
									Th(Class("px-4 py-3 text-left font-medium"), g.Text("Subject")),
									Th(Class("px-4 py-3 text-left font-medium"), g.Text("Status")),
									Th(Class("px-4 py-3 text-left font-medium"), g.Text("Sent At")),
									Th(Class("px-4 py-3 text-left font-medium"), g.Text("Actions")),
								),
							),
							TBody(
								g.Attr("x-show", "notifications.length > 0"),
								Template(
									g.Attr("x-for", "notif in notifications"),
									g.Attr(":key", "notif.id"),
									Tr(
										Class("border-b hover:bg-muted/50 cursor-pointer transition-colors"),
										g.Attr("@click", "viewDetail(notif)"),

										// Type
										Td(
											Class("px-4 py-3"),
											Div(
												Class("flex items-center gap-2"),
												Span(
													g.Attr("x-show", "notif.type === 'email'"),
													lucide.Mail(Class("h-4 w-4 text-blue-600")),
												),
												Span(
													g.Attr("x-show", "notif.type === 'sms'"),
													lucide.MessageSquare(Class("h-4 w-4 text-green-600")),
												),
												Span(
													g.Attr("x-show", "notif.type === 'push'"),
													lucide.Bell(Class("h-4 w-4 text-purple-600")),
												),
												Span(
													Class("capitalize"),
													g.Attr("x-text", "notif.type"),
												),
											),
										),

										// Recipient
										Td(
											Class("px-4 py-3"),
											Div(
												Class("max-w-xs truncate"),
												g.Attr("x-text", "notif.recipient"),
												g.Attr("title", "notif.recipient"),
											),
										),

										// Subject
										Td(
											Class("px-4 py-3"),
											Div(
												Class("max-w-xs truncate text-muted-foreground"),
												g.Attr("x-text", "notif.subject || '(No subject)'"),
												g.Attr("title", "notif.subject"),
											),
										),

										// Status
										Td(
											Class("px-4 py-3"),
											Span(
												Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"),
												g.Attr(":class", `{
													'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400': notif.status === 'delivered',
													'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400': notif.status === 'sent',
													'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400': notif.status === 'failed' || notif.status === 'bounced',
													'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400': notif.status === 'pending'
												}`),
												Span(Class("capitalize"), g.Attr("x-text", "notif.status")),
											),
										),

										// Sent At
										Td(
											Class("px-4 py-3 text-muted-foreground"),
											g.Attr("x-text", "formatDate(notif.sentAt || notif.createdAt)"),
										),

										// Actions
										Td(
											Class("px-4 py-3"),
											Button(
												Type("button"),
												g.Attr("@click.stop", "viewDetail(notif)"),
												Class("text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"),
												lucide.Eye(Class("h-4 w-4")),
											),
										),
									),
								),
							),
						),

						// Empty state
						Div(
							g.Attr("x-show", "!loading && !error && notifications.length === 0"),
							Class("py-12 text-center"),
							Div(
								Class("mx-auto flex h-16 w-16 items-center justify-center rounded-full bg-muted"),
								lucide.Inbox(Class("h-8 w-8 text-muted-foreground")),
							),
							H3(Class("mt-4 text-lg font-semibold"), g.Text("No notifications found")),
							P(Class("mt-2 text-sm text-muted-foreground"),
								g.Text("Try adjusting your filters or send some notifications")),
						),
					),
				),

				// Pagination
				Div(
					g.Attr("x-show", "!loading && !error && notifications.length > 0"),
					Class("flex items-center justify-between border-t bg-card px-4 py-3 sm:px-6"),
					Div(
						Class("flex flex-1 justify-between sm:hidden"),
						Button(
							Type("button"),
							g.Attr("@click", "previousPage()"),
							g.Attr(":disabled", "currentPage === 1"),
							g.Attr(":class", "currentPage === 1 ? 'opacity-50 cursor-not-allowed' : ''"),
							Class("relative inline-flex items-center rounded-md border border-input bg-background px-4 py-2 text-sm font-medium"),
							g.Text("Previous"),
						),
						Button(
							Type("button"),
							g.Attr("@click", "nextPage()"),
							g.Attr(":disabled", "!pagination.hasNext"),
							g.Attr(":class", "!pagination.hasNext ? 'opacity-50 cursor-not-allowed' : ''"),
							Class("relative ml-3 inline-flex items-center rounded-md border border-input bg-background px-4 py-2 text-sm font-medium"),
							g.Text("Next"),
						),
					),
					Div(
						Class("hidden sm:flex sm:flex-1 sm:items-center sm:justify-between"),
						Div(
							Class("text-sm text-muted-foreground"),
							Span(g.Text("Showing ")),
							Span(Class("font-medium"), g.Attr("x-text", "((currentPage - 1) * pagination.pageSize + 1)")),
							Span(g.Text(" to ")),
							Span(Class("font-medium"), g.Attr("x-text", "Math.min(currentPage * pagination.pageSize, pagination.totalCount)")),
							Span(g.Text(" of ")),
							Span(Class("font-medium"), g.Attr("x-text", "pagination.totalCount")),
							Span(g.Text(" results")),
						),
						Div(
							Class("flex gap-2"),
							Button(
								Type("button"),
								g.Attr("@click", "previousPage()"),
								g.Attr(":disabled", "currentPage === 1"),
								g.Attr(":class", "currentPage === 1 ? 'opacity-50 cursor-not-allowed' : ''"),
								Class("relative inline-flex items-center rounded-md border border-input bg-background px-3 py-2 text-sm font-medium hover:bg-accent"),
								lucide.ChevronLeft(Class("h-4 w-4")),
							),
							Button(
								Type("button"),
								g.Attr("@click", "nextPage()"),
								g.Attr(":disabled", "!pagination.hasNext"),
								g.Attr(":class", "!pagination.hasNext ? 'opacity-50 cursor-not-allowed' : ''"),
								Class("relative inline-flex items-center rounded-md border border-input bg-background px-3 py-2 text-sm font-medium hover:bg-accent"),
								lucide.ChevronRight(Class("h-4 w-4")),
							),
						),
					),
				),

				// Detail modal (will be embedded inline)
				g.Group([]g.Node{
					notificationDetailModal(basePath, currentApp.ID.String()),
				}),
			),
		),
	)
}

func historyListData(basePath string, appID string) string {
	return fmt.Sprintf(`{
		notifications: [],
		pagination: {
			currentPage: 1,
			totalPages: 1,
			totalCount: 0,
			pageSize: 20,
			hasNext: false
		},
		currentPage: 1,
		filters: {
			type: '',
			status: '',
			recipient: ''
		},
		loading: false,
		error: null,
		selectedNotification: null,
		showDetailModal: false,

		async loadNotifications() {
			this.loading = true;
			this.error = null;
			try {
				const input = {
					page: this.currentPage,
					limit: 20
				};

				if (this.filters.type) input.type = this.filters.type;
				if (this.filters.status) input.status = this.filters.status;
				if (this.filters.recipient) input.recipient = this.filters.recipient;

				const result = await $bridge.call('notification.listNotificationsHistory', input);
				if (result && result.notifications) {
					this.notifications = result.notifications || [];
					this.pagination = result.pagination || this.pagination;
				} else {
					this.error = 'Invalid response format';
				}
			} catch (err) {
				console.error('Failed to load notifications:', err);
				this.error = err.message || 'Failed to load notification history';
			} finally {
				this.loading = false;
			}
		},

		clearFilters() {
			this.filters = {
				type: '',
				status: '',
				recipient: ''
			};
			this.currentPage = 1;
			this.loadNotifications();
		},

		async nextPage() {
			if (this.pagination.hasNext) {
				this.currentPage++;
				await this.loadNotifications();
			}
		},

		async previousPage() {
			if (this.currentPage > 1) {
				this.currentPage--;
				await this.loadNotifications();
			}
		},

		viewDetail(notification) {
			this.selectedNotification = notification;
			this.showDetailModal = true;
		},

		closeDetail() {
			this.showDetailModal = false;
			this.selectedNotification = null;
		},

		formatDate(dateStr) {
			if (!dateStr) return 'N/A';
			const date = new Date(dateStr);
			const now = new Date();
			const diff = Math.floor((now - date) / 1000); // seconds

			if (diff < 60) return 'Just now';
			if (diff < 3600) return Math.floor(diff / 60) + ' minutes ago';
			if (diff < 86400) return Math.floor(diff / 3600) + ' hours ago';
			if (diff < 604800) return Math.floor(diff / 86400) + ' days ago';

			return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
		}
	}`)
}
