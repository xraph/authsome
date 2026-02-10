package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/components/checkbox"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// DeviceFlowMonitorPage shows active device authorization codes.
func (p *PagesManager) DeviceFlowMonitorPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				deviceCodes: [],
				loading: true,
				error: null,
				status: 'pending',
				autoRefresh: true,
				refreshInterval: null,
				
				async loadDeviceCodes() {
					this.loading = true;
					this.error = null;
					try {
						const result = await $bridge.call('oidcprovider.getDeviceCodes', {
							appId: '`+appID+`',
							status: this.status,
							page: 1,
							pageSize: 50
						});
						this.deviceCodes = result.data || [];
					} catch (err) {
						this.error = err.message || 'Failed to load device codes';
					} finally {
						this.loading = false;
					}
				},
				
				async revokeCode(userCode) {
					if (!confirm('Revoke this device code?')) return;
					try {
						await $bridge.call('oidcprovider.revokeDeviceCode', {
							userCode: userCode
						});
						await this.loadDeviceCodes();
					} catch (err) {
						alert('Failed to revoke: ' + err.message);
					}
				},
				
				async cleanupExpired() {
					try {
						const result = await $bridge.call('oidcprovider.cleanupExpiredDeviceCodes', {
							appId: '`+appID+`'
						});
						alert('Cleaned up ' + result.data.expiredCount + ' expired codes');
						await this.loadDeviceCodes();
					} catch (err) {
						alert('Cleanup failed: ' + err.message);
					}
				},
				
				startAutoRefresh() {
					if (this.autoRefresh && !this.refreshInterval) {
						this.refreshInterval = setInterval(() => this.loadDeviceCodes(), 5000);
					}
				},
				
				stopAutoRefresh() {
					if (this.refreshInterval) {
						clearInterval(this.refreshInterval);
						this.refreshInterval = null;
					}
				},
				
				toggleAutoRefresh() {
					this.autoRefresh = !this.autoRefresh;
					if (this.autoRefresh) {
						this.startAutoRefresh();
					} else {
						this.stopAutoRefresh();
					}
				},
				
				formatTime(seconds) {
					if (seconds <= 0) return 'Expired';
					const mins = Math.floor(seconds / 60);
					const secs = seconds % 60;
					return mins + 'm ' + secs + 's';
				}
			}`),
			g.Attr("x-init", "loadDeviceCodes(); startAutoRefresh()"),

			// Header
			Div(Class("flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold"), g.Text("Device Flow Monitor")),
					P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Monitor active device authorization flows (RFC 8628)")),
				),
				Div(Class("flex items-center gap-2"),
					button.Button(
						Div(Class("flex items-center gap-2"),
							icons.RefreshCw(icons.WithSize(16)),
							g.Text("Refresh"),
						),
						button.WithVariant("outline"),
						button.WithAttrs(g.Attr("@click", "loadDeviceCodes()")),
					),
					button.Button(
						Div(Class("flex items-center gap-2"),
							icons.Trash2(icons.WithSize(16)),
							g.Text("Cleanup Expired"),
						),
						button.WithVariant("outline"),
						button.WithAttrs(g.Attr("@click", "cleanupExpired()")),
					),
				),
			),

			// Filters and auto-refresh
			Div(Class("flex items-center gap-4"),
				Select(
					Class("flex h-10 rounded-md border border-input bg-background px-3 py-2 text-sm"),
					g.Attr("x-model", "status"),
					g.Attr("@change", "loadDeviceCodes()"),
					Option(Value("pending"), g.Text("Pending")),
					Option(Value("authorized"), g.Text("Authorized")),
					Option(Value("denied"), g.Text("Denied")),
					Option(Value("expired"), g.Text("Expired")),
					Option(Value("consumed"), g.Text("Consumed")),
				),
				Div(Class("flex items-center gap-2 text-sm"),
					checkbox.Checkbox(
						checkbox.WithID("autoRefresh"),
						checkbox.WithAttrs(
							g.Attr("x-model", "autoRefresh"),
							g.Attr("@change", "toggleAutoRefresh()"),
						),
					),
					Label(For("autoRefresh"), g.Text("Auto-refresh (5s)")),
				),
			),

			// Device codes list
			g.El("template", g.Attr("x-if", "loading"),
				Div(Class("flex justify-center py-12"),
					Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary")),
				),
			),

			g.El("template", g.Attr("x-if", "!loading && deviceCodes.length === 0"),
				card.Card(card.Content(
					Div(Class("text-center py-8 text-muted-foreground"),
						g.Text("No device codes found"),
					),
				)),
			),

			g.El("template", g.Attr("x-if", "!loading && deviceCodes.length > 0"),
				Div(Class("grid gap-4"),
					g.El("template", g.Attr("x-for", "code in deviceCodes"),
						card.Card(
							card.Content(
								Div(Class("flex items-start justify-between"),
									Div(Class("flex-1 space-y-2"),
										Div(Class("flex items-center gap-3"),
											Div(Class("text-2xl font-mono font-bold"), g.Attr("x-text", "code.userCode")),
											Span(
												Class("inline-flex items-center rounded-md px-2 py-1 text-xs font-medium"),
												g.Attr(":class", `{
														'bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300': code.status === 'pending',
														'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300': code.status === 'authorized',
														'bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300': code.status === 'denied',
														'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300': code.status === 'expired'
													}`),
												g.Attr("x-text", "code.status"),
											),
										),
										Div(Class("text-sm text-muted-foreground"),
											Span(g.Text("Client: ")),
											Span(Class("font-mono"), g.Attr("x-text", "code.clientName")),
										),
										Div(Class("text-sm text-muted-foreground"),
											Span(g.Text("Device Code: ")),
											Span(Class("font-mono"), g.Attr("x-text", "code.deviceCode")),
										),
										Div(Class("text-sm"),
											Span(Class("text-muted-foreground"), g.Text("Expires: ")),
											Span(Class("font-medium"), g.Attr("x-text", "formatTime(code.timeRemaining)")),
										),
										Div(Class("text-sm"),
											Span(Class("text-muted-foreground"), g.Text("Polls: ")),
											Span(g.Attr("x-text", "code.pollCount")),
										),
									),
									button.Button(
										Div(Class("flex items-center gap-1"),
											icons.X(icons.WithSize(16)),
											g.Text("Revoke"),
										),
										button.WithVariant("outline"),
										button.WithSize("sm"),
										button.WithAttrs(
											g.Attr("@click", "revokeCode(code.userCode)"),
											g.Attr("x-show", "code.status === 'pending'"),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	), nil
}
