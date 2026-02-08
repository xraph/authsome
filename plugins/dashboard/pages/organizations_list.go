package pages

import (
	"github.com/xraph/forgeui/components/button"
	"github.com/xraph/forgeui/components/card"
	"github.com/xraph/forgeui/icons"
	"github.com/xraph/forgeui/primitives"
	"github.com/xraph/forgeui/router"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// OrganizationsListPage shows list of organizations
func (p *PagesManager) OrganizationsListPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")

	return primitives.Container(
		Div(
			Class("space-y-2"),

			// Header
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(Class("text-3xl font-bold"), g.Text("Organizations")),
					P(Class("text-gray-600 dark:text-gray-400 mt-1"), g.Text("Manage user organizations")),
				),
				button.Button(
					Div(
						Class("flex items-center gap-2"),
						icons.Plus(icons.WithSize(16)),
						Span(g.Text("New Organization")),
					),
				),
			),

			// Organizations grid
			Div(
				g.Attr("x-data", `{
					organizations: [],
					loading: true,
					error: null,
					async loadOrganizations() {
						this.loading = true;
						this.error = null;
						try {
							const result = await $bridge.call('organization.getOrganizations', {
								appId: '`+appID+`'
							});
							console.log('Organizations result:', result);
							this.organizations = result.data || [];
						} catch (err) {
							console.error('Failed to load organizations:', err);
							this.error = err.message || 'Failed to load organizations';
						} finally {
							this.loading = false;
						}
					}
				}`),
				g.Attr("x-init", "loadOrganizations()"),

				// Loading state
				g.El("template", g.Attr("x-if", "loading"),
					Div(
						Class("flex items-center justify-center py-12"),
						Div(
							Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary"),
						),
					),
				),

				// Error state
				g.El("template", g.Attr("x-if", "!loading && error"),
					card.Card(
						card.Content(
							Div(
								Class("text-center py-8"),
								icons.AlertCircle(icons.WithSize(48), icons.WithClass("mx-auto text-red-500 mb-4")),
								P(Class("text-red-600 dark:text-red-400 font-medium"), g.Attr("x-text", "error")),
								button.Button(
									g.Text("Retry"),
									button.WithVariant("outline"),
									button.WithAttrs(g.Attr("@click", "loadOrganizations()"), g.Attr("class", "mt-4")),
								),
							),
						),
					),
				),

				// Empty state
				g.El("template", g.Attr("x-if", "!loading && !error && organizations.length === 0"),
					card.Card(
						card.Content(
							Div(
								Class("text-center py-12"),
								icons.Building2(icons.WithSize(48), icons.WithClass("mx-auto text-gray-400 mb-4")),
								H3(Class("text-lg font-semibold text-gray-900 dark:text-gray-100"), g.Text("No organizations yet")),
								P(Class("text-gray-500 dark:text-gray-400 mt-1 mb-4"), g.Text("Create your first organization to get started.")),
								button.Button(
									Div(
										Class("flex items-center gap-2"),
										icons.Plus(icons.WithSize(16)),
										Span(g.Text("Create Organization")),
									),
								),
							),
						),
					),
				),

				// Organizations grid
				g.El("template", g.Attr("x-if", "!loading && !error && organizations.length > 0"),
					Div(
						Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6"),
						g.El("template", g.Attr("x-for", "org in organizations"),
							A(
								g.Attr(":href", "`"+p.baseUIPath+"/app/"+appID+"/organizations/${org.id}`"),
								card.Card(
									card.Header(
										card.Title("", card.WithAttrs(g.Attr("x-text", "org.name"))),
										card.Description("", card.WithAttrs(g.Attr("x-text", "org.slug || 'No slug'"))),
									),
									card.Content(
										Div(
											Class("text-sm text-gray-600 dark:text-gray-400"),
											Div(
												Span(g.Text("Members: ")),
												Span(g.Attr("x-text", "org.memberCount || 0")),
											),
											Div(
												Span(g.Text("Teams: ")),
												Span(g.Attr("x-text", "org.teamCount || 0")),
											),
											Div(
												Span(g.Text("Created: ")),
												Span(g.Attr("x-text", "new Date(org.createdAt).toLocaleDateString()")),
											),
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

// OrganizationDetailPage shows organization details with tabs, stats, and extension support
func (p *PagesManager) OrganizationDetailPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")
	orgID := ctx.Param("orgId")
	activeTab := ctx.Query("tab")
	if activeTab == "" {
		activeTab = "overview"
	}

	appBase := p.baseUIPath + "/app/" + appID

	return primitives.Container(
		Div(
			Class("space-y-2"),
			// Alpine.js data and state management
			g.Attr("x-data", p.organizationDetailData(appID, orgID, appBase)),
			g.Attr("x-init", "loadData()"),

			// Back link
			A(
				Href(appBase+"/organizations"),
				Class("inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"),
				icons.ArrowLeft(icons.WithSize(16)),
				Span(g.Text("Back to Organizations")),
			),

			// Loading state
			g.El("template", g.Attr("x-if", "loading"),
				Div(
					Class("flex items-center justify-center py-12"),
					Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary")),
				),
			),

			// Error state
			g.El("template", g.Attr("x-if", "!loading && error"),
				card.Card(
					card.Content(
						Div(
							Class("text-center py-8"),
							icons.AlertCircle(icons.WithSize(48), icons.WithClass("mx-auto text-red-500 mb-4")),
							H3(Class("text-lg font-semibold text-red-600"), g.Text("Error Loading Organization")),
							P(Class("text-muted-foreground mt-1"), g.Attr("x-text", "error")),
							button.Button(
								g.Text("Retry"),
								button.WithVariant("outline"),
								button.WithAttrs(g.Attr("@click", "loadData()"), g.Attr("class", "mt-4")),
							),
						),
					),
				),
			),

			// Not found state
			g.El("template", g.Attr("x-if", "!loading && !error && !organization"),
				card.Card(
					card.Content(
						Div(
							Class("text-center py-12"),
							icons.Building2(icons.WithSize(48), icons.WithClass("mx-auto text-gray-400 mb-4")),
							H3(Class("text-lg font-semibold"), g.Text("Organization Not Found")),
							P(Class("text-muted-foreground mt-1"), g.Text("The organization you're looking for doesn't exist or you don't have access.")),
							button.Button(
								Div(Class("flex items-center gap-2"),
									icons.ArrowLeft(icons.WithSize(16)),
									g.Text("Back to Organizations"),
								),
								button.WithVariant("outline"),
								button.WithAttrs(g.Attr("@click", "window.location.href = '"+appBase+"/organizations'"), g.Attr("class", "mt-4")),
							),
						),
					),
				),
			),

			// Main content
			g.El("template", g.Attr("x-if", "!loading && !error && organization"),
				Div(
					Class("space-y-2"),

					// Organization header
					p.organizationHeader(appBase, orgID),

					// Tab navigation
					p.organizationTabs(appBase, orgID, activeTab),

					// Tab content
					Div(
						// Overview tab
						g.El("template", g.Attr("x-if", "activeTab === 'overview'"),
							Div(
								Class("space-y-2"),
								// Stats cards
								Div(
									Class("grid gap-4 md:grid-cols-3"),
									p.statsCard("Members", "stats.memberCount", icons.Users(icons.WithSize(20))),
									p.statsCard("Teams", "stats.teamCount", icons.UsersRound(icons.WithSize(20))),
									p.statsCard("Pending Invitations", "stats.invitationCount", icons.Mail(icons.WithSize(20))),
								),

								// Quick links
								p.quickLinksSection(appBase, orgID),

								// Extension widgets
								p.extensionWidgetsSection(),
							),
						),

						// Members tab
						g.El("template", g.Attr("x-if", "activeTab === 'members'"),
							p.membersTabContent(appID, orgID, appBase),
						),

						// Teams tab
						g.El("template", g.Attr("x-if", "activeTab === 'teams'"),
							p.teamsTabContent(appID, orgID, appBase),
						),

						// Invitations tab
						g.El("template", g.Attr("x-if", "activeTab === 'invitations'"),
							p.invitationsTabContent(appID, orgID, appBase),
						),

						// Extension tabs (dynamic)
						g.El("template", g.Attr("x-for", "tab in extensionData.tabs"),
							g.El("template", g.Attr("x-if", "activeTab === tab.id"),
								Div(
									Class("space-y-4"),
									H2(Class("text-xl font-semibold"), g.Attr("x-text", "tab.label")),
									P(Class("text-muted-foreground"), g.Text("Extension tab content loaded dynamically.")),
								),
							),
						),
					),
				),
			),

			// Delete confirmation modal
			p.deleteConfirmationModal(appBase),
		),
	), nil
}

// organizationDetailData returns the Alpine.js data object for the organization detail page
func (p *PagesManager) organizationDetailData(appID, orgID, appBase string) string {
	return `{
		organization: null,
		userRole: '',
		stats: { memberCount: 0, teamCount: 0, invitationCount: 0 },
		extensionData: { widgets: [], tabs: [], actions: [], quickLinks: [] },
		loading: true,
		error: null,
		activeTab: '` + "overview" + `',
		showDeleteModal: false,
		deleting: false,
		
		// Members tab state
		members: [],
		membersLoading: false,
		showInviteModal: false,
		inviteEmail: '',
		inviteRole: 'member',
		inviting: false,
		
		// Teams tab state
		teams: [],
		teamsLoading: false,
		showCreateTeamModal: false,
		newTeamName: '',
		newTeamDescription: '',
		creatingTeam: false,
		
		// Invitations tab state
		invitations: [],
		invitationsLoading: false,
		
		get canDelete() { return this.userRole === 'owner'; },
		get canManage() { return this.userRole === 'owner' || this.userRole === 'admin'; },
		
		async loadData() {
			this.loading = true;
			this.error = null;
			try {
				const result = await $bridge.call('organization.getOrganization', {
					appId: '` + appID + `',
					orgId: '` + orgID + `'
				});
				this.organization = result.organization;
				this.userRole = result.userRole || '';
				this.stats = result.stats || { memberCount: 0, teamCount: 0, invitationCount: 0 };
				await this.loadExtensions();
			} catch (err) {
				console.error('Failed to load organization:', err);
				this.error = err.message || 'Failed to load organization';
			} finally {
				this.loading = false;
			}
		},
		
		async loadExtensions() {
			try {
				const result = await $bridge.call('organization.getExtensionData', {
					appId: '` + appID + `',
					orgId: '` + orgID + `'
				});
				this.extensionData = result || { widgets: [], tabs: [], actions: [], quickLinks: [] };
			} catch (err) {
				console.warn('Failed to load extension data:', err);
			}
		},
		
		async loadMembers() {
			this.membersLoading = true;
			try {
				const result = await $bridge.call('organization.getMembers', {
					appId: '` + appID + `',
					orgId: '` + orgID + `'
				});
				this.members = result.data || [];
			} catch (err) {
				console.error('Failed to load members:', err);
			} finally {
				this.membersLoading = false;
			}
		},
		
		async loadTeams() {
			this.teamsLoading = true;
			try {
				const result = await $bridge.call('organization.getTeams', {
					appId: '` + appID + `',
					orgId: '` + orgID + `'
				});
				this.teams = result.data || [];
			} catch (err) {
				console.error('Failed to load teams:', err);
			} finally {
				this.teamsLoading = false;
			}
		},
		
		async loadInvitations() {
			this.invitationsLoading = true;
			try {
				const result = await $bridge.call('organization.getInvitations', {
					appId: '` + appID + `',
					orgId: '` + orgID + `'
				});
				this.invitations = result.data || [];
			} catch (err) {
				console.error('Failed to load invitations:', err);
			} finally {
				this.invitationsLoading = false;
			}
		},
		
		async inviteMember() {
			if (!this.inviteEmail) return;
			this.inviting = true;
			try {
				await $bridge.call('organization.inviteMember', {
					appId: '` + appID + `',
					orgId: '` + orgID + `',
					email: this.inviteEmail,
					role: this.inviteRole
				});
				this.showInviteModal = false;
				this.inviteEmail = '';
				this.inviteRole = 'member';
				await this.loadMembers();
				await this.loadInvitations();
				this.stats.invitationCount++;
			} catch (err) {
				alert('Failed to invite member: ' + (err.message || 'Unknown error'));
			} finally {
				this.inviting = false;
			}
		},
		
		async createTeam() {
			if (!this.newTeamName) return;
			this.creatingTeam = true;
			try {
				await $bridge.call('organization.createTeam', {
					appId: '` + appID + `',
					orgId: '` + orgID + `',
					name: this.newTeamName,
					description: this.newTeamDescription
				});
				this.showCreateTeamModal = false;
				this.newTeamName = '';
				this.newTeamDescription = '';
				await this.loadTeams();
				this.stats.teamCount++;
			} catch (err) {
				alert('Failed to create team: ' + (err.message || 'Unknown error'));
			} finally {
				this.creatingTeam = false;
			}
		},
		
		async cancelInvitation(inviteId) {
			if (!confirm('Are you sure you want to cancel this invitation?')) return;
			try {
				await $bridge.call('organization.cancelInvitation', {
					appId: '` + appID + `',
					orgId: '` + orgID + `',
					inviteId: inviteId
				});
				await this.loadInvitations();
				this.stats.invitationCount--;
			} catch (err) {
				alert('Failed to cancel invitation: ' + (err.message || 'Unknown error'));
			}
		},
		
		async removeMember(memberId) {
			if (!confirm('Are you sure you want to remove this member?')) return;
			try {
				await $bridge.call('organization.removeMember', {
					appId: '` + appID + `',
					orgId: '` + orgID + `',
					memberId: memberId
				});
				await this.loadMembers();
				this.stats.memberCount--;
			} catch (err) {
				alert('Failed to remove member: ' + (err.message || 'Unknown error'));
			}
		},
		
		async deleteOrganization() {
			if (!this.canDelete) return;
			this.deleting = true;
			try {
				await $bridge.call('organization.deleteOrganization', {
					appId: '` + appID + `',
					orgId: '` + orgID + `'
				});
				window.location.href = '` + appBase + `/organizations';
			} catch (err) {
				alert('Failed to delete organization: ' + (err.message || 'Unknown error'));
			} finally {
				this.deleting = false;
				this.showDeleteModal = false;
			}
		},
		
		switchTab(tab) {
			this.activeTab = tab;
			if (tab === 'members' && this.members.length === 0) this.loadMembers();
			if (tab === 'teams' && this.teams.length === 0) this.loadTeams();
			if (tab === 'invitations' && this.invitations.length === 0) this.loadInvitations();
		}
	}`
}

// organizationHeader renders the organization header with logo, name, and actions
func (p *PagesManager) organizationHeader(appBase, orgID string) g.Node {
	return card.Card(
		card.Content(
			Div(
				Class("flex items-center justify-between py-2"),
				// Left: Logo and info
				Div(
					Class("flex items-center gap-4"),
					// Logo or placeholder
					Div(
						g.El("template", g.Attr("x-if", "organization?.logo"),
							Img(
								g.Attr(":src", "organization.logo"),
								g.Attr(":alt", "organization.name"),
								Class("size-16 rounded-lg object-cover"),
							),
						),
						g.El("template", g.Attr("x-if", "!organization?.logo"),
							Div(
								Class("size-16 rounded-lg bg-primary/10 flex items-center justify-center"),
								icons.Building2(icons.WithSize(32), icons.WithClass("text-primary")),
							),
						),
					),
					// Name and slug
					Div(
						H1(
							Class("text-2xl font-bold"),
							g.Attr("x-text", "organization?.name || ''"),
						),
						P(
							Class("text-sm text-muted-foreground"),
							g.Text("@"),
							Span(g.Attr("x-text", "organization?.slug || ''")),
						),
						Div(
							Class("mt-2"),
							Span(
								Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize"),
								g.Attr(":class", `{
									'bg-primary text-primary-foreground': userRole === 'owner',
									'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200': userRole === 'admin',
									'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200': userRole === 'member'
								}`),
								g.Attr("x-text", "userRole"),
							),
						),
					),
				),
				// Right: Actions
				Div(
					Class("flex items-center gap-2"),
					// Extension actions
					g.El("template", g.Attr("x-for", "action in extensionData.actions"),
						button.Button(
							Span(g.Attr("x-text", "action.label")),
							button.WithVariant("outline"),
							button.WithSize("sm"),
							button.WithAttrs(g.Attr("@click", "eval(action.action)")),
						),
					),
					// Edit button
					g.El("template", g.Attr("x-if", "canManage"),
						A(
							Href(appBase+"/organizations/"+orgID+"/edit"),
							button.Button(
								Div(Class("flex items-center gap-2"),
									icons.Pencil(icons.WithSize(16)),
									g.Text("Edit"),
								),
								button.WithVariant("outline"),
								button.WithSize("sm"),
							),
						),
					),
					// Delete button
					g.El("template", g.Attr("x-if", "canDelete"),
						button.Button(
							Div(Class("flex items-center gap-2"),
								icons.Trash2(icons.WithSize(16)),
								g.Text("Delete"),
							),
							button.WithVariant("destructive"),
							button.WithSize("sm"),
							button.WithAttrs(g.Attr("@click", "showDeleteModal = true")),
						),
					),
				),
			),
		),
	)
}

// organizationTabs renders the tab navigation
func (p *PagesManager) organizationTabs(appBase, orgID, activeTab string) g.Node {
	tabs := []struct {
		ID    string
		Label string
		Icon  g.Node
	}{
		{"overview", "Overview", icons.LayoutDashboard(icons.WithSize(16))},
		{"members", "Members", icons.Users(icons.WithSize(16))},
		{"teams", "Teams", icons.UsersRound(icons.WithSize(16))},
		{"invitations", "Invitations", icons.Mail(icons.WithSize(16))},
	}

	tabNodes := make([]g.Node, 0, len(tabs))
	for _, tab := range tabs {
		tabNodes = append(tabNodes, Button(
			Type("button"),
			Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-md transition-colors"),
			g.Attr(":class", `activeTab === '`+tab.ID+`' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground hover:bg-muted'`),
			g.Attr("@click", "switchTab('"+tab.ID+"')"),
			tab.Icon,
			Span(g.Text(tab.Label)),
		))
	}

	// Add extension tabs
	extensionTabsTemplate := g.El("template", g.Attr("x-for", "tab in extensionData.tabs"),
		Button(
			Type("button"),
			Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-md transition-colors"),
			g.Attr(":class", `activeTab === tab.id ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground hover:bg-muted'`),
			g.Attr("@click", "switchTab(tab.id)"),
			Span(g.Attr("x-text", "tab.label")),
		),
	)

	return Div(
		Class("flex flex-wrap gap-2 border-b pb-4"),
		g.Group(tabNodes),
		extensionTabsTemplate,
	)
}

// statsCard renders a stats card with icon
func (p *PagesManager) statsCard(label, valueExpr string, icon g.Node) g.Node {
	return card.Card(
		card.Content(
			Div(
				Class("flex items-center gap-4 py-2"),
				Div(
					Class("p-3 rounded-lg bg-primary/10"),
					icon,
				),
				Div(
					P(Class("text-sm text-muted-foreground"), g.Text(label)),
					P(Class("text-2xl font-bold"), g.Attr("x-text", valueExpr+" || 0")),
				),
			),
		),
	)
}

// quickLinksSection renders the quick links
func (p *PagesManager) quickLinksSection(appBase, orgID string) g.Node {
	return Div(
		Class("space-y-4"),
		H3(Class("text-lg font-semibold"), g.Text("Quick Actions")),
		Div(
			Class("grid gap-4 md:grid-cols-2 lg:grid-cols-4"),
			// Built-in quick links
			p.quickLinkCard("Invite Members", "Add new members to your organization", "@click", "showInviteModal = true; switchTab('members')", icons.UserPlus(icons.WithSize(20))),
			p.quickLinkCard("Create Team", "Organize members into teams", "@click", "showCreateTeamModal = true; switchTab('teams')", icons.Users(icons.WithSize(20))),
			p.quickLinkCard("Manage Roles", "Configure member permissions", "href", appBase+"/organizations/"+orgID+"/roles", icons.Shield(icons.WithSize(20))),
			p.quickLinkCard("View Activity", "See recent organization activity", "href", appBase+"/organizations/"+orgID+"/activity", icons.Activity(icons.WithSize(20))),
		),
		// Extension quick links
		g.El("template", g.Attr("x-if", "extensionData.quickLinks.length > 0"),
			Div(
				Class("grid gap-4 md:grid-cols-2 lg:grid-cols-4 mt-4"),
				g.El("template", g.Attr("x-for", "link in extensionData.quickLinks"),
					A(
						g.Attr(":href", "link.url"),
						card.Card(
							card.Content(
								Div(
									Class("flex items-start gap-3 py-2"),
									Div(
										Class("p-2 rounded-lg bg-muted"),
										Span(g.Attr("x-html", "link.icon")),
									),
									Div(
										P(Class("font-medium"), g.Attr("x-text", "link.title")),
										P(Class("text-sm text-muted-foreground"), g.Attr("x-text", "link.description")),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}

// quickLinkCard renders a quick link card
func (p *PagesManager) quickLinkCard(title, description, actionType, actionValue string, icon g.Node) g.Node {
	var actionAttr g.Node
	if actionType == "href" {
		actionAttr = A(
			Href(actionValue),
			card.Card(
				card.Content(
					Div(
						Class("flex items-start gap-3 py-2 cursor-pointer hover:bg-muted/50 transition-colors rounded-lg"),
						Div(Class("p-2 rounded-lg bg-muted"), icon),
						Div(
							P(Class("font-medium"), g.Text(title)),
							P(Class("text-sm text-muted-foreground"), g.Text(description)),
						),
					),
				),
			),
		)
	} else {
		actionAttr = Div(
			g.Attr(actionType, actionValue),
			card.Card(
				card.Content(
					Div(
						Class("flex items-start gap-3 py-2 cursor-pointer hover:bg-muted/50 transition-colors rounded-lg"),
						Div(Class("p-2 rounded-lg bg-muted"), icon),
						Div(
							P(Class("font-medium"), g.Text(title)),
							P(Class("text-sm text-muted-foreground"), g.Text(description)),
						),
					),
				),
			),
		)
	}
	return actionAttr
}

// extensionWidgetsSection renders extension widgets
func (p *PagesManager) extensionWidgetsSection() g.Node {
	return g.El("template", g.Attr("x-if", "extensionData.widgets.length > 0"),
		Div(
			Class("space-y-4"),
			H3(Class("text-lg font-semibold"), g.Text("Extensions")),
			Div(
				Class("grid gap-4 md:grid-cols-2"),
				g.El("template", g.Attr("x-for", "widget in extensionData.widgets"),
					card.Card(
						card.Header(
							card.Title("", card.WithAttrs(g.Attr("x-text", "widget.title"))),
						),
						card.Content(
							Div(g.Attr("x-html", "widget.content")),
						),
					),
				),
			),
		),
	)
}

// membersTabContent renders the members tab
func (p *PagesManager) membersTabContent(appID, orgID, appBase string) g.Node {
	return Div(
		Class("space-y-4"),
		g.Attr("x-init", "loadMembers()"),

		// Header with invite button
		Div(
			Class("flex items-center justify-between"),
			H2(Class("text-xl font-semibold"), g.Text("Members")),
			g.El("template", g.Attr("x-if", "canManage"),
				button.Button(
					Div(Class("flex items-center gap-2"),
						icons.UserPlus(icons.WithSize(16)),
						g.Text("Invite Member"),
					),
					button.WithAttrs(g.Attr("@click", "showInviteModal = true")),
				),
			),
		),

		// Loading
		g.El("template", g.Attr("x-if", "membersLoading"),
			Div(Class("flex justify-center py-8"),
				Div(Class("animate-spin rounded-full h-6 w-6 border-b-2 border-primary")),
			),
		),

		// Empty state
		g.El("template", g.Attr("x-if", "!membersLoading && members.length === 0"),
			card.Card(
				card.Content(
					Div(
						Class("text-center py-8"),
						icons.Users(icons.WithSize(48), icons.WithClass("mx-auto text-gray-400 mb-4")),
						P(Class("text-muted-foreground"), g.Text("No members yet. Invite someone to get started.")),
					),
				),
			),
		),

		// Members list
		g.El("template", g.Attr("x-if", "!membersLoading && members.length > 0"),
			card.Card(
				card.Content(
					Div(
						Class("divide-y"),
						g.El("template", g.Attr("x-for", "member in members"),
							Div(
								Class("flex items-center justify-between py-4"),
								Div(
									Class("flex items-center gap-3"),
									Div(
										Class("size-10 rounded-full bg-primary/10 flex items-center justify-center"),
										Span(Class("text-sm font-medium"), g.Attr("x-text", "(member.userName || member.userEmail || '?')[0].toUpperCase()")),
									),
									Div(
										P(Class("font-medium"), g.Attr("x-text", "member.userName || member.userEmail")),
										P(Class("text-sm text-muted-foreground"), g.Attr("x-text", "member.userEmail")),
									),
								),
								Div(
									Class("flex items-center gap-3"),
									Span(
										Class("inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize"),
										g.Attr(":class", `{
											'bg-primary text-primary-foreground': member.role === 'owner',
											'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200': member.role === 'admin',
											'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200': member.role === 'member'
										}`),
										g.Attr("x-text", "member.role"),
									),
									g.El("template", g.Attr("x-if", "canManage && member.role !== 'owner'"),
										button.Button(
											icons.Trash2(icons.WithSize(16)),
											button.WithVariant("ghost"),
											button.WithSize("sm"),
											button.WithAttrs(g.Attr("@click", "removeMember(member.id)")),
										),
									),
								),
							),
						),
					),
				),
			),
		),

		// Invite modal
		p.inviteMemberModal(),
	)
}

// teamsTabContent renders the teams tab
func (p *PagesManager) teamsTabContent(appID, orgID, appBase string) g.Node {
	return Div(
		Class("space-y-4"),
		g.Attr("x-init", "loadTeams()"),

		// Header
		Div(
			Class("flex items-center justify-between"),
			H2(Class("text-xl font-semibold"), g.Text("Teams")),
			g.El("template", g.Attr("x-if", "canManage"),
				button.Button(
					Div(Class("flex items-center gap-2"),
						icons.Plus(icons.WithSize(16)),
						g.Text("Create Team"),
					),
					button.WithAttrs(g.Attr("@click", "showCreateTeamModal = true")),
				),
			),
		),

		// Loading
		g.El("template", g.Attr("x-if", "teamsLoading"),
			Div(Class("flex justify-center py-8"),
				Div(Class("animate-spin rounded-full h-6 w-6 border-b-2 border-primary")),
			),
		),

		// Empty state
		g.El("template", g.Attr("x-if", "!teamsLoading && teams.length === 0"),
			card.Card(
				card.Content(
					Div(
						Class("text-center py-8"),
						icons.UsersRound(icons.WithSize(48), icons.WithClass("mx-auto text-gray-400 mb-4")),
						P(Class("text-muted-foreground"), g.Text("No teams yet. Create a team to organize members.")),
					),
				),
			),
		),

		// Teams grid
		g.El("template", g.Attr("x-if", "!teamsLoading && teams.length > 0"),
			Div(
				Class("grid gap-4 md:grid-cols-2 lg:grid-cols-3"),
				g.El("template", g.Attr("x-for", "team in teams"),
					card.Card(
						card.Header(
							card.Title("", card.WithAttrs(g.Attr("x-text", "team.name"))),
							card.Description("", card.WithAttrs(g.Attr("x-text", "team.description || 'No description'"))),
						),
						card.Content(
							Div(
								Class("flex items-center justify-between"),
								Span(Class("text-sm text-muted-foreground"),
									g.Attr("x-text", "(team.memberCount || 0) + ' members'"),
								),
								button.Button(
									g.Text("View"),
									button.WithVariant("outline"),
									button.WithSize("sm"),
								),
							),
						),
					),
				),
			),
		),

		// Create team modal
		p.createTeamModal(),
	)
}

// invitationsTabContent renders the invitations tab
func (p *PagesManager) invitationsTabContent(appID, orgID, appBase string) g.Node {
	return Div(
		Class("space-y-4"),
		g.Attr("x-init", "loadInvitations()"),

		H2(Class("text-xl font-semibold"), g.Text("Pending Invitations")),

		// Loading
		g.El("template", g.Attr("x-if", "invitationsLoading"),
			Div(Class("flex justify-center py-8"),
				Div(Class("animate-spin rounded-full h-6 w-6 border-b-2 border-primary")),
			),
		),

		// Empty state
		g.El("template", g.Attr("x-if", "!invitationsLoading && invitations.length === 0"),
			card.Card(
				card.Content(
					Div(
						Class("text-center py-8"),
						icons.Mail(icons.WithSize(48), icons.WithClass("mx-auto text-gray-400 mb-4")),
						P(Class("text-muted-foreground"), g.Text("No pending invitations.")),
					),
				),
			),
		),

		// Invitations list
		g.El("template", g.Attr("x-if", "!invitationsLoading && invitations.length > 0"),
			card.Card(
				card.Content(
					Div(
						Class("divide-y"),
						g.El("template", g.Attr("x-for", "invite in invitations"),
							Div(
								Class("flex items-center justify-between py-4"),
								Div(
									P(Class("font-medium"), g.Attr("x-text", "invite.email")),
									P(Class("text-sm text-muted-foreground"),
										g.Text("Invited as "),
										Span(Class("capitalize"), g.Attr("x-text", "invite.role")),
									),
								),
								Div(
									Class("flex items-center gap-3"),
									Span(
										Class("text-sm text-muted-foreground"),
										g.Attr("x-text", "'Expires ' + new Date(invite.expiresAt).toLocaleDateString()"),
									),
									button.Button(
										g.Text("Cancel"),
										button.WithVariant("destructive"),
										button.WithSize("sm"),
										button.WithAttrs(g.Attr("@click", "cancelInvitation(invite.id)")),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}

// inviteMemberModal renders the invite member modal
func (p *PagesManager) inviteMemberModal() g.Node {
	return Div(
		g.Attr("x-show", "showInviteModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center"),

		// Backdrop
		Div(
			Class("absolute inset-0 bg-black/50"),
			g.Attr("@click", "showInviteModal = false"),
		),

		// Modal
		Div(
			Class("relative bg-background rounded-lg shadow-lg w-full max-w-md p-6 space-y-4"),
			H3(Class("text-lg font-semibold"), g.Text("Invite Member")),

			Div(
				Class("space-y-4"),
				Div(
					Label(Class("text-sm font-medium"), g.Attr("for", "invite-email"), g.Text("Email Address")),
					Input(
						Type("email"),
						Class("mt-1 w-full px-3 py-2 border rounded-md"),
						g.Attr("x-model", "inviteEmail"),
						g.Attr("placeholder", "member@example.com"),
					),
				),
				Div(
					Label(Class("text-sm font-medium"), g.Attr("for", "invite-role"), g.Text("Role")),
					Select(
						Class("mt-1 w-full px-3 py-2 border rounded-md"),
						g.Attr("x-model", "inviteRole"),
						Option(Value("member"), g.Text("Member")),
						Option(Value("admin"), g.Text("Admin")),
					),
				),
			),

			Div(
				Class("flex justify-end gap-3"),
				button.Button(
					g.Text("Cancel"),
					button.WithVariant("outline"),
					button.WithAttrs(g.Attr("@click", "showInviteModal = false")),
				),
				button.Button(
					g.Text("Send Invitation"),
					button.WithAttrs(
						g.Attr("@click", "inviteMember()"),
						g.Attr(":disabled", "inviting || !inviteEmail"),
					),
				),
			),
		),
	)
}

// createTeamModal renders the create team modal
func (p *PagesManager) createTeamModal() g.Node {
	return Div(
		g.Attr("x-show", "showCreateTeamModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center"),

		// Backdrop
		Div(
			Class("absolute inset-0 bg-black/50"),
			g.Attr("@click", "showCreateTeamModal = false"),
		),

		// Modal
		Div(
			Class("relative bg-background rounded-lg shadow-lg w-full max-w-md p-6 space-y-4"),
			H3(Class("text-lg font-semibold"), g.Text("Create Team")),

			Div(
				Class("space-y-4"),
				Div(
					Label(Class("text-sm font-medium"), g.Attr("for", "team-name"), g.Text("Team Name")),
					Input(
						Type("text"),
						Class("mt-1 w-full px-3 py-2 border rounded-md"),
						g.Attr("x-model", "newTeamName"),
						g.Attr("placeholder", "Engineering"),
					),
				),
				Div(
					Label(Class("text-sm font-medium"), g.Attr("for", "team-description"), g.Text("Description (optional)")),
					g.El("textarea",
						Class("mt-1 w-full px-3 py-2 border rounded-md"),
						g.Attr("x-model", "newTeamDescription"),
						g.Attr("placeholder", "Team description..."),
						g.Attr("rows", "3"),
					),
				),
			),

			Div(
				Class("flex justify-end gap-3"),
				button.Button(
					g.Text("Cancel"),
					button.WithVariant("outline"),
					button.WithAttrs(g.Attr("@click", "showCreateTeamModal = false")),
				),
				button.Button(
					g.Text("Create Team"),
					button.WithAttrs(
						g.Attr("@click", "createTeam()"),
						g.Attr(":disabled", "creatingTeam || !newTeamName"),
					),
				),
			),
		),
	)
}

// deleteConfirmationModal renders the delete confirmation modal
func (p *PagesManager) deleteConfirmationModal(appBase string) g.Node {
	return Div(
		g.Attr("x-show", "showDeleteModal"),
		g.Attr("x-cloak", ""),
		Class("fixed inset-0 z-50 flex items-center justify-center"),

		// Backdrop
		Div(
			Class("absolute inset-0 bg-black/50"),
			g.Attr("@click", "showDeleteModal = false"),
		),

		// Modal
		Div(
			Class("relative bg-background rounded-lg shadow-lg w-full max-w-md p-6 space-y-4"),
			Div(
				Class("flex items-center gap-3 text-destructive"),
				icons.AlertCircle(icons.WithSize(24)),
				H3(Class("text-lg font-semibold"), g.Text("Delete Organization")),
			),
			P(Class("text-muted-foreground"),
				g.Text("Are you sure you want to delete "),
				Strong(g.Attr("x-text", "organization?.name")),
				g.Text("? This action cannot be undone. All members, teams, and data will be permanently removed."),
			),
			Div(
				Class("flex justify-end gap-3"),
				button.Button(
					g.Text("Cancel"),
					button.WithVariant("outline"),
					button.WithAttrs(g.Attr("@click", "showDeleteModal = false")),
				),
				button.Button(
					g.Text("Delete Organization"),
					button.WithVariant("destructive"),
					button.WithAttrs(
						g.Attr("@click", "deleteOrganization()"),
						g.Attr(":disabled", "deleting"),
					),
				),
			),
		),
	)
}

// OrganizationEditPage shows the organization edit form
func (p *PagesManager) OrganizationEditPage(ctx *router.PageContext) (g.Node, error) {
	appID := ctx.Param("appId")
	orgID := ctx.Param("orgId")
	appBase := p.baseUIPath + "/app/" + appID

	return primitives.Container(
		Div(
			Class("space-y-2"),
			g.Attr("x-data", `{
				organization: null,
				loading: true,
				saving: false,
				error: null,
				
				name: '',
				slug: '',
				logo: '',
				
				async loadData() {
					this.loading = true;
					this.error = null;
					try {
						const result = await $bridge.call('organization.getOrganization', {
							appId: '`+appID+`',
							orgId: '`+orgID+`'
						});
						this.organization = result.organization;
						this.name = result.organization?.name || '';
						this.slug = result.organization?.slug || '';
						this.logo = result.organization?.logo || '';
					} catch (err) {
						console.error('Failed to load organization:', err);
						this.error = err.message || 'Failed to load organization';
					} finally {
						this.loading = false;
					}
				},
				
				async saveOrganization() {
					if (!this.name) {
						alert('Name is required');
						return;
					}
					this.saving = true;
					try {
						await $bridge.call('organization.updateOrganization', {
							appId: '`+appID+`',
							orgId: '`+orgID+`',
							name: this.name,
							logo: this.logo
						});
						window.location.href = '`+appBase+`/organizations/`+orgID+`';
					} catch (err) {
						alert('Failed to save organization: ' + (err.message || 'Unknown error'));
					} finally {
						this.saving = false;
					}
				}
			}`),
			g.Attr("x-init", "loadData()"),

			// Back link
			A(
				Href(appBase+"/organizations/"+orgID),
				Class("inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground transition-colors"),
				icons.ArrowLeft(icons.WithSize(16)),
				Span(g.Text("Back to Organization")),
			),

			// Loading state
			g.El("template", g.Attr("x-if", "loading"),
				Div(
					Class("flex items-center justify-center py-12"),
					Div(Class("animate-spin rounded-full h-8 w-8 border-b-2 border-primary")),
				),
			),

			// Error state
			g.El("template", g.Attr("x-if", "!loading && error"),
				card.Card(
					card.Content(
						Div(
							Class("text-center py-8"),
							icons.AlertCircle(icons.WithSize(48), icons.WithClass("mx-auto text-red-500 mb-4")),
							P(Class("text-red-600"), g.Attr("x-text", "error")),
						),
					),
				),
			),

			// Edit form
			g.El("template", g.Attr("x-if", "!loading && !error && organization"),
				Div(
					Class("space-y-2"),
					H1(Class("text-2xl font-bold"), g.Text("Edit Organization")),

					card.Card(
						card.Content(
							Div(
								Class("space-y-2 py-4"),

								// Name field
								Div(
									Label(Class("text-sm font-medium"), g.Attr("for", "org-name"), g.Text("Organization Name")),
									Input(
										Type("text"),
										ID("org-name"),
										Class("mt-1 w-full px-3 py-2 border rounded-md"),
										g.Attr("x-model", "name"),
										g.Attr("placeholder", "Acme Inc."),
									),
								),

								// Slug field (read-only)
								Div(
									Label(Class("text-sm font-medium"), g.Attr("for", "org-slug"), g.Text("Slug")),
									Input(
										Type("text"),
										ID("org-slug"),
										Class("mt-1 w-full px-3 py-2 border rounded-md bg-muted"),
										g.Attr("x-model", "slug"),
										g.Attr("readonly", ""),
									),
									P(Class("text-xs text-muted-foreground mt-1"), g.Text("Slug cannot be changed after creation.")),
								),

								// Logo URL field
								Div(
									Label(Class("text-sm font-medium"), g.Attr("for", "org-logo"), g.Text("Logo URL")),
									Input(
										Type("url"),
										ID("org-logo"),
										Class("mt-1 w-full px-3 py-2 border rounded-md"),
										g.Attr("x-model", "logo"),
										g.Attr("placeholder", "https://example.com/logo.png"),
									),
								),

								// Logo preview
								g.El("template", g.Attr("x-if", "logo"),
									Div(
										Class("mt-2"),
										Label(Class("text-sm font-medium"), g.Text("Preview")),
										Img(
											g.Attr(":src", "logo"),
											Class("mt-1 size-16 rounded-lg object-cover"),
										),
									),
								),
							),
						),
					),

					// Actions
					Div(
						Class("flex justify-end gap-3"),
						A(
							Href(appBase+"/organizations/"+orgID),
							button.Button(
								g.Text("Cancel"),
								button.WithVariant("outline"),
							),
						),
						button.Button(
							Div(
								Class("flex items-center gap-2"),
								icons.Save(icons.WithSize(16)),
								g.Text("Save Changes"),
							),
							button.WithAttrs(
								g.Attr("@click", "saveOrganization()"),
								g.Attr(":disabled", "saving"),
							),
						),
					),
				),
			),
		),
	), nil
}
