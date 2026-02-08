package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
)

// UserSessionsPage renders all sessions for a specific user
func UserSessionsPage(currentApp *app.App, userID, basePath string) g.Node {
	appID := currentApp.ID.String()
	appBase := fmt.Sprintf("%s/app/%s", basePath, appID)

	return Div(
		Class("space-y-6"),

		// Dynamic content with Alpine.js
		Div(
			g.Attr("x-data", userSessionsData(appID, userID)),
			g.Attr("x-init", "loadUserSessions()"),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				LoadingSpinner(),
			),

			// Error state
			ErrorMessage("error && !loading"),

			// Content
			Div(
				g.Attr("x-show", "!loading && !error"),
				g.Attr("x-cloak", ""),
				Class("space-y-6"),

				// Back navigation
				Div(
					Class("flex items-center gap-4"),
					BackLink(appBase+"/multisession", "Back to Sessions"),
				),

				// Header card
				card.Card(
					card.Content(
						Class("p-6"),
						Div(
							Class("flex items-center justify-between"),
							Div(
								Class("flex items-center gap-4"),
								// User avatar
								Div(
									Class("flex h-14 w-14 items-center justify-center rounded-full bg-gradient-to-br from-primary to-primary/60 text-primary-foreground shadow-lg"),
									lucide.User(Class("size-7")),
								),
								Div(
									H1(Class("text-xl font-bold"), g.Text("User Sessions")),
									Div(Class("mt-1 text-sm text-muted-foreground font-mono"),
										g.Attr("x-text", "userId")),
								),
							),
							// Stats and actions
							Div(
								Class("flex items-center gap-4"),
								// Stats
								Div(
									Class("text-right"),
									Div(Class("text-2xl font-bold"), g.Attr("x-text", "totalCount")),
									Div(Class("text-sm text-muted-foreground"), g.Text("Total Sessions")),
								),
								Div(
									Class("text-right"),
									Div(Class("text-2xl font-bold text-emerald-600 dark:text-emerald-400"), g.Attr("x-text", "activeCount")),
									Div(Class("text-sm text-muted-foreground"), g.Text("Active")),
								),
								// Revoke all button
								g.El("template", g.Attr("x-if", "activeCount > 0"),
									button.Button(
										Div(
											Class("flex items-center gap-2"),
											lucide.LogOut(Class("size-4")),
											g.Text("Revoke All"),
										),
										button.WithVariant("destructive"),
										button.WithAttrs(
											g.Attr("@click", "revokeAllSessions()"),
										),
									),
								),
							),
						),
					),
				),

				// Sessions grid
				Div(
					// Empty state
					Div(
						g.Attr("x-show", "sessions.length === 0"),
						card.Card(
							Class("border-dashed"),
							card.Content(
								Class("py-12 text-center"),
								lucide.MonitorSmartphone(Class("mx-auto size-12 text-muted-foreground mb-4")),
								H3(Class("text-lg font-semibold"), g.Text("No Sessions")),
								P(Class("mt-2 text-muted-foreground"), g.Text("This user has no active sessions.")),
							),
						),
					),

					// Sessions grid
					Div(
						g.Attr("x-show", "sessions.length > 0"),
						Class("grid gap-4 sm:grid-cols-2 lg:grid-cols-3"),
						g.El("template", g.Attr("x-for", "session in sessions"), g.Attr(":key", "session.id"),
							userSessionCard(appBase),
						),
					),
				),

				// Pagination
				Pagination("goToPage"),
			),
		),
	)
}

// userSessionsData returns the Alpine.js data object for user sessions
func userSessionsData(appID, userID string) string {
	return fmt.Sprintf(`{
		sessions: [],
		userId: '%s',
		totalCount: 0,
		activeCount: 0,
		pagination: {
			currentPage: 1,
			pageSize: 100,
			totalItems: 0,
			totalPages: 0
		},
		loading: true,
		error: null,
		
		get visiblePages() {
			const current = this.pagination.currentPage;
			const total = this.pagination.totalPages;
			const range = [];
			const start = Math.max(1, current - 2);
			const end = Math.min(total, current + 2);
			for (let i = start; i <= end; i++) {
				range.push(i);
			}
			return range;
		},
		
		async loadUserSessions() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('multisession.getUserSessions', {
					appId: '%s',
					userId: '%s',
					page: this.pagination.currentPage,
					pageSize: this.pagination.pageSize
				});
				
				this.sessions = result.sessions || [];
				this.totalCount = result.totalCount || 0;
				this.activeCount = result.activeCount || 0;
				this.pagination = result.pagination || { currentPage: 1, pageSize: 100, totalItems: 0, totalPages: 0 };
			} catch (err) {
				console.error('Failed to load user sessions:', err);
				this.error = err.message || 'Failed to load user sessions';
			} finally {
				this.loading = false;
			}
		},
		
		goToPage(page) {
			if (page >= 1 && page <= this.pagination.totalPages) {
				this.pagination.currentPage = page;
				this.loadUserSessions();
			}
		},
		
		async revokeSession(sessionId) {
			if (!confirm('Are you sure you want to revoke this session?')) return;
			try {
				const result = await $bridge.call('multisession.revokeSession', { 
					appId: '%s',
					sessionId: sessionId 
				});
				if (result.message) {
					alert(result.message);
				}
				await this.loadUserSessions();
			} catch (err) {
				alert(err.message || 'Failed to revoke session');
			}
		},
		
		async revokeAllSessions() {
			if (!confirm('Are you sure you want to revoke all sessions for this user? They will be logged out from all devices.')) return;
			try {
				const result = await $bridge.call('multisession.revokeAllUserSessions', { 
					appId: '%s',
					userId: '%s'
				});
				if (result.message) {
					alert(result.message);
				}
				await this.loadUserSessions();
			} catch (err) {
				alert(err.message || 'Failed to revoke sessions');
			}
		}
	}`, userID, appID, userID, appID, appID, userID)
}

// userSessionCard renders a session card in the user sessions page
func userSessionCard(appBase string) g.Node {
	return card.Card(
		Class("hover:shadow-md transition-shadow hover:border-primary/50"),
		card.Content(
			Class("p-5"),

			// Header
			Div(
				Class("flex items-start justify-between"),
				Div(
					Class("flex items-center gap-4"),
					// Device icon
					Div(
						g.Attr(":class", `{
							'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400': session.deviceType === 'mobile',
							'bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400': session.deviceType === 'tablet',
							'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400': session.deviceType !== 'mobile' && session.deviceType !== 'tablet'
						}`),
						Class("flex h-12 w-12 items-center justify-center rounded-xl"),
						Div(g.Attr("x-show", "session.deviceType === 'mobile'"), lucide.Smartphone(Class("size-6"))),
						Div(g.Attr("x-show", "session.deviceType === 'tablet'"), lucide.Tablet(Class("size-6"))),
						Div(g.Attr("x-show", "session.deviceType !== 'mobile' && session.deviceType !== 'tablet'"), lucide.Monitor(Class("size-6"))),
					),
					Div(
						H3(Class("font-semibold"), g.Attr("x-text", "session.deviceInfo")),
						P(Class("text-sm text-muted-foreground"), g.Attr("x-text", "session.os")),
					),
				),
				// Status badge
				DynamicStatusBadge(),
			),

			// Info
			Div(
				Class("mt-5 grid grid-cols-2 gap-4 border-t pt-4"),

				// IP Address
				Div(
					P(Class("text-xs font-medium uppercase tracking-wider text-muted-foreground"), g.Text("IP Address")),
					P(Class("mt-1 text-sm font-medium"), g.Attr("x-text", "session.ipAddress")),
				),

				// Created
				Div(
					P(Class("text-xs font-medium uppercase tracking-wider text-muted-foreground"), g.Text("Created")),
					P(Class("mt-1 text-sm"), g.Attr("x-text", "session.lastUsed")),
				),
			),
		),

		// Footer
		card.Footer(
			Class("flex items-center justify-between bg-muted/50 px-5 py-3"),
			Span(Class("text-xs text-muted-foreground"),
				g.El("template", g.Attr("x-if", "session.isActive"),
					Span(g.Text("Expires "), Span(g.Attr("x-text", "session.expiresIn"))),
				),
				g.El("template", g.Attr("x-if", "!session.isActive"),
					Span(g.Text("Expired")),
				),
			),
			Div(
				Class("flex items-center gap-2"),
				// View details
				A(
					g.Attr(":href", fmt.Sprintf("'%s/multisession/session/' + session.id", appBase)),
					Class("rounded-lg p-2 text-muted-foreground hover:bg-background hover:text-primary transition-colors"),
					Title("View Details"),
					lucide.Eye(Class("size-4")),
				),
				// Revoke button
				g.El("template", g.Attr("x-if", "session.isActive"),
					Button(
						Type("button"),
						g.Attr("@click", "revokeSession(session.id)"),
						Class("rounded-lg p-2 text-muted-foreground hover:bg-background hover:text-destructive transition-colors"),
						Title("Revoke Session"),
						lucide.LogOut(Class("size-4")),
					),
				),
			),
		),
	)
}
