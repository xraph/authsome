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

// SessionDetailPage renders the session detail page with dynamic data loading.
func SessionDetailPage(currentApp *app.App, sessionID, basePath string) g.Node {
	appID := currentApp.ID.String()
	appBase := fmt.Sprintf("%s/app/%s", basePath, appID)

	return Div(
		Class("space-y-6"),

		// Dynamic content with Alpine.js
		Div(
			g.Attr("x-data", sessionDetailData(appID, sessionID)),
			g.Attr("x-init", "loadSession()"),

			// Loading state
			Div(
				g.Attr("x-show", "loading"),
				LoadingSpinner(),
			),

			// Error state
			ErrorMessage("error && !loading"),

			// Content
			Div(
				g.Attr("x-show", "!loading && !error && session"),
				g.Attr("x-cloak", ""),
				Class("space-y-6"),

				// Back navigation and header
				Div(
					Class("flex items-center gap-4"),
					BackLink(appBase+"/multisession", "Back to Sessions"),
				),

				// Header card
				card.Card(
					card.Content(
						Class("p-6"),
						Div(
							Class("flex flex-col gap-6 sm:flex-row sm:items-start sm:justify-between"),
							Div(
								Class("flex items-center gap-5"),
								// Large device icon
								Div(
									g.Attr(":class", `{
										'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400': session.deviceType === 'mobile',
										'bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400': session.deviceType === 'tablet',
										'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400': session.deviceType !== 'mobile' && session.deviceType !== 'tablet'
									}`),
									Class("flex h-16 w-16 items-center justify-center rounded-2xl shadow-inner"),
									Div(g.Attr("x-show", "session.deviceType === 'mobile'"), lucide.Smartphone(Class("size-8"))),
									Div(g.Attr("x-show", "session.deviceType === 'tablet'"), lucide.Tablet(Class("size-8"))),
									Div(g.Attr("x-show", "session.deviceType !== 'mobile' && session.deviceType !== 'tablet'"), lucide.Monitor(Class("size-8"))),
								),
								Div(
									H1(Class("text-2xl font-bold tracking-tight"), g.Attr("x-text", "session.deviceInfo")),
									Div(
										Class("mt-2 flex flex-wrap items-center gap-3"),
										DynamicStatusBadge(),
										Span(Class("text-sm text-muted-foreground"),
											g.Text("Session ID: "),
											Span(g.Attr("x-text", "session.id"))),
									),
								),
							),
							// Revoke action
							g.El("template", g.Attr("x-if", "session.isActive"),
								button.Button(
									Div(
										Class("flex items-center gap-2"),
										lucide.LogOut(Class("size-4")),
										g.Text("Revoke Session"),
									),
									button.WithVariant("outline"),
									button.WithAttrs(
										g.Attr("@click", "revokeSession()"),
										Class("text-destructive hover:text-destructive"),
									),
								),
							),
						),
					),
				),

				// Details grid
				Div(
					Class("grid gap-6 lg:grid-cols-2"),

					// Device Information
					DetailCard("Device Information", lucide.MonitorSmartphone(Class("size-5")),
						Div(
							Class("space-y-5"),
							DetailRow("Device Type", "session.deviceType", lucide.Laptop(Class("size-4"))),
							DetailRow("Browser", "session.browser + ' ' + session.browserVersion", lucide.Globe(Class("size-4"))),
							DetailRow("Operating System", "session.os + ' ' + session.osVersion", lucide.Settings(Class("size-4"))),
							// User agent
							Div(
								Class("pt-5 border-t"),
								Label(Class("text-xs font-medium uppercase tracking-wider text-muted-foreground"), g.Text("User Agent String")),
								Div(Class("mt-2 rounded-lg bg-muted p-3 font-mono text-xs break-all"), g.Attr("x-text", "session.userAgent")),
							),
						),
					),

					// Session Activity
					DetailCard("Session Activity", lucide.Clock(Class("size-5")),
						Div(
							Class("space-y-5"),
							DetailRow("IP Address", "session.ipAddress", lucide.MapPin(Class("size-4"))),
							DetailRow("Created", "session.createdAtFormatted", lucide.Calendar(Class("size-4"))),
							DetailRow("Last Updated", "session.updatedAtFormatted", lucide.RefreshCw(Class("size-4"))),
							DetailRow("Expires", "session.expiresAtFormatted", lucide.Timer(Class("size-4"))),
							g.El("template", g.Attr("x-if", "session.lastRefreshedFormatted"),
								DetailRow("Last Refreshed", "session.lastRefreshedFormatted", lucide.RotateCw(Class("size-4"))),
							),
						),
					),

					// User Information
					DetailCard("User Information", lucide.User(Class("size-5")),
						Div(
							Class("space-y-5"),
							DetailRow("User ID", "session.userId", lucide.Hash(Class("size-4"))),
							Div(
								Class("pt-2"),
								A(
									g.Attr(":href", fmt.Sprintf("'%s/multisession/user/' + session.userId", appBase)),
									Class("inline-flex items-center gap-2 text-sm font-medium text-primary hover:underline"),
									g.Text("View all sessions for this user"),
									lucide.ArrowRight(Class("size-4")),
								),
							),
						),
					),

					// Application Context
					DetailCard("Application Context", lucide.Layers(Class("size-5")),
						Div(
							Class("space-y-5"),
							DetailRow("App ID", "session.appId", lucide.AppWindow(Class("size-4"))),
							g.El("template", g.Attr("x-if", "session.organizationId"),
								DetailRow("Organization ID", "session.organizationId", lucide.Building2(Class("size-4"))),
							),
							g.El("template", g.Attr("x-if", "session.environmentId"),
								DetailRow("Environment ID", "session.environmentId", lucide.Server(Class("size-4"))),
							),
						),
					),
				),
			),
		),
	)
}

// sessionDetailData returns the Alpine.js data object for session detail.
func sessionDetailData(appID, sessionID string) string {
	return fmt.Sprintf(`{
		session: null,
		loading: true,
		error: null,
		
		async loadSession() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('multisession.getSession', {
					appId: '%s',
					sessionId: '%s'
				});
				this.session = result.session;
			} catch (err) {
				console.error('Failed to load session:', err);
				this.error = err.message || 'Failed to load session';
			} finally {
				this.loading = false;
			}
		},
		
		async revokeSession() {
			if (!confirm('Are you sure you want to revoke this session? The user will be logged out.')) return;
			try {
				const result = await $bridge.call('multisession.revokeSession', { 
					appId: '%s',
					sessionId: '%s' 
				});
				if (result.message) {
					alert(result.message);
				}
				// Redirect back to sessions list
				window.location.href = '/api/identity/ui/app/%s/multisession';
			} catch (err) {
				alert(err.message || 'Failed to revoke session');
			}
		}
	}`, appID, sessionID, appID, sessionID, appID)
}
