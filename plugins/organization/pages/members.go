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

// MembersPage renders the organization members management page.
func MembersPage(currentApp *app.App, orgID, basePath string) g.Node {
	appBase := fmt.Sprintf("%s/app/%s", basePath, currentApp.ID.String())
	baseURL := fmt.Sprintf("%s/organizations/%s", appBase, orgID)

	return Div(
		Class("space-y-6"),

		// Alpine.js data
		Div(
			g.Attr("x-data", membersPageData(currentApp.ID.String(), orgID)),
			g.Attr("x-init", "loadMembers()"),

			// Back link
			BackLink(baseURL, "Back to Organization"),

			// Page header
			PageHeader(
				"Members",
				"Manage organization members and their roles",
				button.Button(
					Div(
						lucide.UserPlus(Class("size-4")),
						g.Text("Invite Member"),
					),
					button.WithVariant("default"),
					button.WithAttrs(
						g.Attr("x-show", "canManage"),
						g.Attr("@click", "showInviteModal = true"),
					),
				),
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

				// Search bar
				Div(
					Class("flex items-center gap-4"),
					SearchInput("Search members...", "", ""),
				),

				// Members table
				membersTable(),

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
							g.Attr("@click", "filters.page--; loadMembers()"),
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
							g.Attr("@click", "filters.page++; loadMembers()"),
						),
					),
				),
			),

			// Invite member modal
			inviteMemberModal(),

			// Remove member confirmation
			removeMemberModal(),
		),
	)
}

// membersPageData returns the Alpine.js data object.
func membersPageData(appID, orgID string) string {
	return fmt.Sprintf(`{
		members: [],
		pagination: {
			currentPage: 1,
			pageSize: 20,
			totalItems: 0,
			totalPages: 0
		},
		filters: {
			search: '',
			page: 1,
			limit: 20
		},
		canManage: false,
		loading: true,
		error: null,
		showInviteModal: false,
		showRemoveModal: false,
		selectedMember: null,
		inviteForm: {
			email: '',
			role: 'member',
			submitting: false,
			error: null
		},
		
		async loadMembers() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('organization.getMembers', {
					appId: '%s',
					orgId: '%s',
					search: this.filters.search,
					page: this.filters.page,
					limit: this.filters.limit
				});
				
				this.members = result.data || [];
				this.pagination = result.pagination || { currentPage: 1, pageSize: 20, totalItems: 0, totalPages: 0 };
				this.canManage = result.canManage || false;
			} catch (err) {
				console.error('Failed to load members:', err);
				this.error = err.message || 'Failed to load members';
			} finally {
				this.loading = false;
			}
		},
		
		async inviteMember() {
			this.inviteForm.submitting = true;
			this.inviteForm.error = null;
			try {
				await $bridge.call('organization.inviteMember', {
					appId: '%s',
					orgId: '%s',
					email: this.inviteForm.email,
					role: this.inviteForm.role
				});
				
				// Reset form and close modal
				this.inviteForm.email = '';
				this.inviteForm.role = 'member';
				this.showInviteModal = false;
				
				// Reload members
				await this.loadMembers();
			} catch (err) {
				console.error('Failed to invite member:', err);
				this.inviteForm.error = err.message || 'Failed to send invitation';
			} finally {
				this.inviteForm.submitting = false;
			}
		},
		
		async updateMemberRole(memberId, newRole) {
			try {
				await $bridge.call('organization.updateMemberRole', {
					appId: '%s',
					orgId: '%s',
					memberId: memberId,
					role: newRole
				});
				
				// Reload members
				await this.loadMembers();
			} catch (err) {
				console.error('Failed to update role:', err);
				alert('Failed to update member role: ' + (err.message || 'Unknown error'));
			}
		},
		
		confirmRemoveMember(member) {
			this.selectedMember = member;
			this.showRemoveModal = true;
		},
		
		async removeMember() {
			if (!this.selectedMember) return;
			
			try {
				await $bridge.call('organization.removeMember', {
					appId: '%s',
					orgId: '%s',
					memberId: this.selectedMember.id
				});
				
				this.showRemoveModal = false;
				this.selectedMember = null;
				
				// Reload members
				await this.loadMembers();
			} catch (err) {
				console.error('Failed to remove member:', err);
				alert('Failed to remove member: ' + (err.message || 'Unknown error'));
			}
		},
		
		formatDate(dateStr) {
			if (!dateStr) return 'N/A';
			const date = new Date(dateStr);
			return date.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
		}
	}`, appID, orgID, appID, orgID, appID, orgID, appID, orgID)
}

// membersTable renders the members table.
func membersTable() g.Node {
	return table.Table()(
		table.TableHeader()(
			table.TableRow()(
				table.TableHeaderCell()(g.Text("Member")),
				table.TableHeaderCell()(g.Text("Email")),
				table.TableHeaderCell()(g.Text("Role")),
				table.TableHeaderCell()(g.Text("Status")),
				table.TableHeaderCell()(g.Text("Joined")),
				table.TableHeaderCell(table.WithAlign(table.AlignRight))(g.Text("Actions")),
			),
		),
		table.TableBody()(
			// Empty state
			Tr(
				g.Attr("x-show", "members.length === 0"),
				Td(
					g.Attr("colspan", "6"),
					Class("text-center py-8"),
					Div(
						Class("text-muted-foreground"),
						lucide.Users(Class("size-12 mx-auto mb-2 opacity-50")),
						P(g.Text("No members found")),
					),
				),
			),
			// Template for each member
			Template(
				g.Attr("x-for", "member in members"),
				g.Attr(":key", "member.id"),
				table.TableRow()(
					table.TableCell()(
						Div(
							Class("font-medium"),
							g.Attr("x-text", "member.userName || 'N/A'"),
						),
					),
					table.TableCell()(
						Span(g.Attr("x-text", "member.userEmail")),
					),
					table.TableCell()(
						// Role selector for admin/owner
						Select(
							g.Attr("x-show", "canManage && member.role !== 'owner'"),
							g.Attr(":value", "member.role"),
							g.Attr("@change", "updateMemberRole(member.id, $event.target.value)"),
							Class("text-sm rounded-md border border-input bg-background px-2 py-1"),
							Option(Value("member"), g.Text("Member")),
							Option(Value("admin"), g.Text("Admin")),
						),
						// Display only for owner or non-managers
						Span(
							g.Attr("x-show", "!canManage || member.role === 'owner'"),
							Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize"),
							g.Attr(":class", `{
								'bg-primary text-primary-foreground': member.role === 'owner',
								'bg-secondary text-secondary-foreground': member.role === 'admin',
								'bg-muted text-muted-foreground': member.role === 'member'
							}`),
							g.Attr("x-text", "member.role"),
						),
					),
					table.TableCell()(
						Span(
							Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium"),
							g.Attr(":class", `{
								'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400': member.status === 'active',
								'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400': member.status === 'pending',
								'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400': member.status === 'suspended'
							}`),
							g.Attr("x-text", "member.status"),
						),
					),
					table.TableCell()(
						Span(g.Attr("x-text", "formatDate(member.joinedAt)")),
					),
					table.TableCell(table.WithAlign(table.AlignRight))(
						button.Button(
							Div(
								lucide.UserMinus(Class("size-3")),
								g.Text("Remove"),
							),
							button.WithVariant("destructive"),
							button.WithSize("sm"),
							button.WithAttrs(
								g.Attr("x-show", "canManage && member.role !== 'owner'"),
								g.Attr("@click", "confirmRemoveMember(member)"),
							),
						),
					),
				),
			),
		),
	)
}

// inviteMemberModal renders the invite member modal.
func inviteMemberModal() g.Node {
	return Div(
		g.Attr("x-show", "showInviteModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		g.Attr("@click.self", "showInviteModal = false"),
		Div(
			Class("max-w-md w-full mx-4 rounded-lg border bg-card shadow-lg"),
			Form(
				g.Attr("@submit.prevent", "inviteMember()"),
				Div(
					Class("p-6 space-y-4"),
					H3(Class("text-lg font-semibold"), g.Text("Invite Member")),
					P(Class("text-sm text-muted-foreground"), g.Text("Send an invitation to join this organization")),

					// Error message
					Div(
						g.Attr("x-show", "inviteForm.error"),
						Class("bg-destructive/10 border border-destructive/20 rounded-lg p-3"),
						P(Class("text-sm text-destructive"), g.Attr("x-text", "inviteForm.error")),
					),

					// Email field
					FormField("invite-email", "Email Address", "email", "email", "user@example.com", true, ""),

					// Role field
					Div(
						Class("space-y-2"),
						Label(For("invite-role"), Class("text-sm font-medium"), g.Text("Role")),
						Select(
							ID("invite-role"),
							g.Attr("x-model", "inviteForm.role"),
							Class("flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"),
							Option(Value("member"), g.Text("Member")),
							Option(Value("admin"), g.Text("Admin")),
						),
					),

					// Actions
					Div(
						Class("flex justify-end gap-2 pt-4"),
						button.Button(
							g.Text("Cancel"),
							button.WithVariant("outline"),
							button.WithType("button"),
							button.WithAttrs(
								g.Attr("@click", "showInviteModal = false"),
								g.Attr(":disabled", "inviteForm.submitting"),
							),
						),
						button.Button(
							Div(
								Span(
									g.Attr("x-show", "inviteForm.submitting"),
									Class("inline-flex items-center gap-2"),
									Div(Class("animate-spin rounded-full h-4 w-4 border-b-2 border-current")),
									g.Text("Sending..."),
								),
								Span(
									g.Attr("x-show", "!inviteForm.submitting"),
									g.Text("Send Invitation"),
								),
							),
							button.WithVariant("default"),
							button.WithType("submit"),
							button.WithAttrs(
								g.Attr(":disabled", "inviteForm.submitting"),
							),
						),
					),
				),
			),
		),
	)
}

// removeMemberModal renders the remove member confirmation modal.
func removeMemberModal() g.Node {
	return Div(
		g.Attr("x-show", "showRemoveModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		g.Attr("@click.self", "showRemoveModal = false"),
		Div(
			Class("max-w-md w-full mx-4 rounded-lg border bg-card shadow-lg"),
			Div(
				Class("p-6 space-y-4"),
				Div(
					Class("flex items-start gap-4"),
					Div(
						Class("rounded-full bg-destructive/10 p-3"),
						lucide.UserMinus(Class("size-6 text-destructive")),
					),
					Div(
						Class("flex-1"),
						H3(Class("text-lg font-semibold"), g.Text("Remove Member")),
						P(
							Class("text-sm text-muted-foreground mt-2"),
							g.Text("Are you sure you want to remove "),
							Span(Class("font-medium"), g.Attr("x-text", "selectedMember?.userName || selectedMember?.userEmail")),
							g.Text(" from this organization?"),
						),
					),
				),
				Div(
					Class("flex justify-end gap-2"),
					button.Button(
						g.Text("Cancel"),
						button.WithVariant("outline"),
						button.WithAttrs(
							g.Attr("@click", "showRemoveModal = false"),
						),
					),
					button.Button(
						g.Text("Remove Member"),
						button.WithVariant("destructive"),
						button.WithAttrs(
							g.Attr("@click", "removeMember()"),
						),
					),
				),
			),
		),
	)
}
