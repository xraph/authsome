package pages

import (
	"fmt"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/table"
)

// InvitationsPage renders the organization invitations management page.
func InvitationsPage(currentApp *app.App, orgID, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())
	baseURL := fmt.Sprintf("%s/organizations/%s", appBase, orgID)

	return Div(
		Class("space-y-6"),

		// Alpine.js data
		Div(
			g.Attr("x-data", invitationsPageData(currentApp.ID.String(), orgID)),
			g.Attr("x-init", "loadInvitations()"),

			// Back link
			BackLink(baseURL, "Back to Organization"),

			// Page header
			PageHeader(
				"Invitations",
				"Manage pending and past invitations to join this organization",
			),

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

				// Filter tabs
				Div(
					Class("flex gap-2 border-b border-border"),
					Button(
						Type("button"),
						Class("px-4 py-2 text-sm font-medium transition-colors"),
						g.Attr(":class", "filters.status === 'all' ? 'border-b-2 border-primary text-primary' : 'text-muted-foreground hover:text-foreground'"),
						g.Attr("@click", "filters.status = 'all'; filters.page = 1; loadInvitations()"),
						g.Text("All"),
					),
					Button(
						Type("button"),
						Class("px-4 py-2 text-sm font-medium transition-colors"),
						g.Attr(":class", "filters.status === 'pending' ? 'border-b-2 border-primary text-primary' : 'text-muted-foreground hover:text-foreground'"),
						g.Attr("@click", "filters.status = 'pending'; filters.page = 1; loadInvitations()"),
						g.Text("Pending"),
					),
					Button(
						Type("button"),
						Class("px-4 py-2 text-sm font-medium transition-colors"),
						g.Attr(":class", "filters.status === 'accepted' ? 'border-b-2 border-primary text-primary' : 'text-muted-foreground hover:text-foreground'"),
						g.Attr("@click", "filters.status = 'accepted'; filters.page = 1; loadInvitations()"),
						g.Text("Accepted"),
					),
					Button(
						Type("button"),
						Class("px-4 py-2 text-sm font-medium transition-colors"),
						g.Attr(":class", "filters.status === 'declined' ? 'border-b-2 border-primary text-primary' : 'text-muted-foreground hover:text-foreground'"),
						g.Attr("@click", "filters.status = 'declined'; filters.page = 1; loadInvitations()"),
						g.Text("Declined"),
					),
					Button(
						Type("button"),
						Class("px-4 py-2 text-sm font-medium transition-colors"),
						g.Attr(":class", "filters.status === 'expired' ? 'border-b-2 border-primary text-primary' : 'text-muted-foreground hover:text-foreground'"),
						g.Attr("@click", "filters.status = 'expired'; filters.page = 1; loadInvitations()"),
						g.Text("Expired"),
					),
				),

				// Invitations table
				invitationsTable(),

				// Pagination
				Div(
					g.Attr("x-show", "pagination.totalPages > 1"),
					Class("flex items-center justify-center gap-2 mt-6"),
					button.Button(
						Div(
							lucide.ChevronLeft(Class("size-4")),
							g.Text("Previous"),
						),
						button.WithVariant("outline"),
						button.WithSize("sm"),
						button.WithAttrs(
							g.Attr("x-show", "pagination.currentPage > 1"),
							g.Attr("@click", "filters.page--; loadInvitations()"),
						),
					),
					Span(
						Class("text-sm text-muted-foreground"),
						g.Attr("x-text", "`Page ${pagination.currentPage} of ${pagination.totalPages}`"),
					),
					button.Button(
						Div(
							g.Text("Next"),
							lucide.ChevronRight(Class("size-4")),
						),
						button.WithVariant("outline"),
						button.WithSize("sm"),
						button.WithAttrs(
							g.Attr("x-show", "pagination.currentPage < pagination.totalPages"),
							g.Attr("@click", "filters.page++; loadInvitations()"),
						),
					),
				),
			),

			// Cancel invitation confirmation
			cancelInvitationModal(),
		),
	)
}

// invitationsPageData returns the Alpine.js data object.
func invitationsPageData(appID, orgID string) string {
	return fmt.Sprintf(`{
		invitations: [],
		pagination: {
			currentPage: 1,
			pageSize: 20,
			totalItems: 0,
			totalPages: 0
		},
		filters: {
			status: 'all',
			page: 1,
			limit: 20
		},
		loading: true,
		error: null,
		showCancelModal: false,
		selectedInvitation: null,
		
		async loadInvitations() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('organization.getInvitations', {
					appId: '%s',
					orgId: '%s',
					status: this.filters.status,
					page: this.filters.page,
					limit: this.filters.limit
				});
				
				this.invitations = result.data || [];
				this.pagination = result.pagination || { currentPage: 1, pageSize: 20, totalItems: 0, totalPages: 0 };
			} catch (err) {
				console.error('Failed to load invitations:', err);
				this.error = err.message || 'Failed to load invitations';
			} finally {
				this.loading = false;
			}
		},
		
		confirmCancelInvitation(invitation) {
			this.selectedInvitation = invitation;
			this.showCancelModal = true;
		},
		
		async cancelInvitation() {
			if (!this.selectedInvitation) return;
			
			try {
				await $bridge.call('organization.cancelInvitation', {
					appId: '%s',
					orgId: '%s',
					inviteId: this.selectedInvitation.id
				});
				
				this.showCancelModal = false;
				this.selectedInvitation = null;
				
				// Reload invitations
				await this.loadInvitations();
			} catch (err) {
				console.error('Failed to cancel invitation:', err);
				alert('Failed to cancel invitation: ' + (err.message || 'Unknown error'));
			}
		},
		
		formatDate(dateStr) {
			if (!dateStr) return 'N/A';
			const date = new Date(dateStr);
			return date.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
		},
		
		isExpired(expiresAt) {
			if (!expiresAt) return false;
			return new Date(expiresAt) < new Date();
		}
	}`, appID, orgID, appID, orgID)
}

// invitationsTable renders the invitations table.
func invitationsTable() g.Node {
	return table.Table()(
		table.TableHeader()(
			table.TableRow()(
				table.TableHeaderCell()(g.Text("Email")),
				table.TableHeaderCell()(g.Text("Role")),
				table.TableHeaderCell()(g.Text("Status")),
				table.TableHeaderCell()(g.Text("Invited By")),
				table.TableHeaderCell()(g.Text("Expires")),
				table.TableHeaderCell()(g.Text("Sent")),
				table.TableHeaderCell(table.WithAlign(table.AlignRight))(g.Text("Actions")),
			),
		),
		table.TableBody()(
			// Empty state
			Tr(
				g.Attr("x-show", "invitations.length === 0"),
				Td(
					g.Attr("colspan", "7"),
					Class("text-center py-8"),
					Div(
						Class("text-muted-foreground"),
						lucide.Mail(Class("size-12 mx-auto mb-2 opacity-50")),
						P(g.Text("No invitations found")),
					),
				),
			),
			// Template for each invitation
			Template(
				g.Attr("x-for", "invitation in invitations"),
				g.Attr(":key", "invitation.id"),
				table.TableRow()(
					table.TableCell()(
						Div(
							Class("font-medium"),
							g.Attr("x-text", "invitation.email"),
						),
					),
					table.TableCell()(
						Span(
							Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize"),
							g.Attr(":class", `{
								'bg-primary text-primary-foreground': invitation.role === 'owner',
								'bg-secondary text-secondary-foreground': invitation.role === 'admin',
								'bg-muted text-muted-foreground': invitation.role === 'member'
							}`),
							g.Attr("x-text", "invitation.role"),
						),
					),
					table.TableCell()(
						Span(
							Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"),
							g.Attr(":class", `{
								'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400': invitation.status === 'pending',
								'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400': invitation.status === 'accepted',
								'bg-slate-100 text-slate-700 dark:bg-slate-900/30 dark:text-slate-400': invitation.status === 'declined',
								'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400': invitation.status === 'expired'
							}`),
							g.Attr("x-text", "invitation.status"),
						),
					),
					table.TableCell()(
						Span(g.Attr("x-text", "invitation.inviterName || 'Unknown'")),
					),
					table.TableCell()(
						Span(
							g.Attr("x-text", "formatDate(invitation.expiresAt)"),
							g.Attr(":class", "isExpired(invitation.expiresAt) ? 'text-destructive' : ''"),
						),
					),
					table.TableCell()(
						Span(g.Attr("x-text", "formatDate(invitation.createdAt)")),
					),
					table.TableCell(table.WithAlign(table.AlignRight))(
						button.Button(
							Div(
								lucide.X(Class("size-3")),
								g.Text("Cancel"),
							),
							button.WithVariant("destructive"),
							button.WithSize("sm"),
							button.WithAttrs(
								g.Attr("x-show", "invitation.status === 'pending'"),
								g.Attr("@click", "confirmCancelInvitation(invitation)"),
							),
						),
					),
				),
			),
		),
	)
}

// cancelInvitationModal renders the cancel invitation confirmation modal.
func cancelInvitationModal() g.Node {
	return Div(
		g.Attr("x-show", "showCancelModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		g.Attr("@click.self", "showCancelModal = false"),
		Div(
			Class("max-w-md w-full mx-4 rounded-lg border bg-card shadow-lg"),
			Div(
				Class("p-6 space-y-4"),
				Div(
					Class("flex items-start gap-4"),
					Div(
						Class("rounded-full bg-destructive/10 p-3"),
						lucide.X(Class("size-6 text-destructive")),
					),
					Div(
						Class("flex-1"),
						H3(Class("text-lg font-semibold"), g.Text("Cancel Invitation")),
						P(
							Class("text-sm text-muted-foreground mt-2"),
							g.Text("Are you sure you want to cancel the invitation for "),
							Span(Class("font-medium"), g.Attr("x-text", "selectedInvitation?.email")),
							g.Text("?"),
						),
					),
				),
				Div(
					Class("flex justify-end gap-2"),
					button.Button(
						g.Text("Keep Invitation"),
						button.WithVariant("outline"),
						button.WithAttrs(
							g.Attr("@click", "showCancelModal = false"),
						),
					),
					button.Button(
						g.Text("Cancel Invitation"),
						button.WithVariant("destructive"),
						button.WithAttrs(
							g.Attr("@click", "cancelInvitation()"),
						),
					),
				),
			),
		),
	)
}
