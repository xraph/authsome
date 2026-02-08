package pages

import (
	"fmt"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/organization"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// OrganizationsData contains data for the organizations list page
type OrganizationsData struct {
	Organizations []*organization.Organization
	Page          int
	TotalPages    int
	Total         int
}

// OrganizationsPage renders the organizations list page
func OrganizationsPage(data OrganizationsData, appIDStr string) g.Node {
	return Div(
		Class("space-y-6"),

		// Header with Create button
		Div(
			Class("flex items-center justify-between"),
			Div(
				H1(
					Class("text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text("Organizations"),
				),
				P(
					Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					g.Text("Manage user organizations and their members"),
				),
			),
			A(
				Href(fmt.Sprintf("/app/%s/organizations/create", appIDStr)),
				Class("inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 active:bg-violet-800 transition-colors"),
				lucide.Plus(Class("h-4 w-4")),
				g.Text("Create Organization"),
			),
		),

		// Organizations Table
		organizationsTable(data, appIDStr),
	)
}

func organizationsTable(data OrganizationsData, appIDStr string) g.Node {
	return Div(
		Class("flex flex-col rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden"),
		g.If(len(data.Organizations) > 0,
			g.Group([]g.Node{
				Div(
					Class("p-5"),
					Div(
						Class("min-w-full overflow-x-auto rounded-sm"),
						Table(
							Class("min-w-full align-middle text-sm"),
							organizationsTableHead(),
							organizationsTableBody(data.Organizations, appIDStr),
						),
					),
				),
				g.If(data.TotalPages > 1,
					organizationsPagination(data.Page, data.TotalPages, appIDStr),
				),
			}),
		),
		g.If(len(data.Organizations) == 0,
			emptyOrganizationsState(appIDStr),
		),
	)
}

func organizationsTableHead() g.Node {
	return THead(
		Class("bg-slate-50 dark:bg-gray-800"),
		Tr(
			Th(Class("px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-600 dark:text-gray-400"), g.Text("Name")),
			Th(Class("px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-600 dark:text-gray-400"), g.Text("Slug")),
			Th(Class("px-4 py-3 text-left text-xs font-semibold uppercase tracking-wider text-slate-600 dark:text-gray-400"), g.Text("Created")),
			Th(Class("px-4 py-3 text-right text-xs font-semibold uppercase tracking-wider text-slate-600 dark:text-gray-400"), g.Text("Actions")),
		),
	)
}

func organizationsTableBody(orgs []*organization.Organization, appIDStr string) g.Node {
	rows := make([]g.Node, len(orgs))
	for i, org := range orgs {
		rows[i] = organizationTableRow(org, appIDStr)
	}
	return TBody(Class("divide-y divide-slate-200 dark:divide-gray-800"), g.Group(rows))
}

func organizationTableRow(org *organization.Organization, appIDStr string) g.Node {
	return Tr(
		Class("hover:bg-slate-50 dark:hover:bg-gray-800 transition-colors"),
		Td(
			Class("px-4 py-3"),
			A(
				Href(fmt.Sprintf("/app/%s/organizations/%s", appIDStr, org.ID.String())),
				Class("font-medium text-slate-900 dark:text-white hover:text-violet-600 dark:hover:text-violet-400"),
				g.Text(org.Name),
			),
		),
		Td(
			Class("px-4 py-3 text-slate-600 dark:text-gray-400"),
			Code(
				Class("rounded bg-slate-100 dark:bg-gray-800 px-2 py-1 text-xs font-mono"),
				g.Text(org.Slug),
			),
		),
		Td(
			Class("px-4 py-3 text-slate-600 dark:text-gray-400"),
			g.Text(formatOrgTime(org.CreatedAt)),
		),
		Td(
			Class("px-4 py-3 text-right"),
			Div(
				Class("flex items-center justify-end gap-2"),
				A(
					Href(fmt.Sprintf("/app/%s/organizations/%s", appIDStr, org.ID.String())),
					Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
					lucide.Eye(Class("h-3.5 w-3.5")),
					g.Text("View"),
				),
				A(
					Href(fmt.Sprintf("/app/%s/organizations/%s/edit", appIDStr, org.ID.String())),
					Class("inline-flex items-center gap-1 rounded-lg border border-slate-200 dark:border-gray-700 px-2.5 py-1.5 text-xs font-medium text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
					lucide.Pencil(Class("h-3.5 w-3.5")),
					g.Text("Edit"),
				),
			),
		),
	)
}

func emptyOrganizationsState(appIDStr string) g.Node {
	return Div(
		Class("p-12 text-center"),
		Div(
			Class("mx-auto max-w-sm"),
			lucide.Building2(Class("mx-auto h-12 w-12 text-slate-400 dark:text-gray-600")),
			H3(
				Class("mt-4 text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text("No organizations"),
			),
			P(
				Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
				g.Text("Get started by creating your first organization."),
			),
			A(
				Href(fmt.Sprintf("/app/%s/organizations/create", appIDStr)),
				Class("mt-4 inline-flex items-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 active:bg-violet-800 transition-colors"),
				lucide.Plus(Class("h-4 w-4")),
				g.Text("Create Organization"),
			),
		),
	)
}

// OrganizationDetailData contains data for the organization detail page
type OrganizationDetailData struct {
	Organization *organization.Organization
	MemberCount  int
}

// OrganizationDetailPage renders the organization detail page
func OrganizationDetailPage(data OrganizationDetailData, appIDStr string) g.Node {
	return Div(
		Class("space-y-6"),

		// Header
		Div(
			Class("flex items-center justify-between"),
			Div(
				A(
					Href(fmt.Sprintf("/app/%s/organizations", appIDStr)),
					Class("text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400"),
					g.Text("← Back to Organizations"),
				),
				H1(
					Class("mt-2 text-2xl font-bold text-slate-900 dark:text-white"),
					g.Text(data.Organization.Name),
				),
				P(
					Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
					Code(
						Class("rounded bg-slate-100 dark:bg-gray-800 px-2 py-1 text-xs font-mono"),
						g.Text(data.Organization.Slug),
					),
				),
			),
			Div(
				Class("flex items-center gap-2"),
				A(
					Href(fmt.Sprintf("/app/%s/organizations/%s/edit", appIDStr, data.Organization.ID.String())),
					Class("inline-flex items-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 px-4 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
					lucide.Pencil(Class("h-4 w-4")),
					g.Text("Edit"),
				),
			),
		),

		// Organization Info
		Div(
			Class("rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-6"),
			H2(
				Class("text-lg font-semibold text-slate-900 dark:text-white mb-4"),
				g.Text("Organization Details"),
			),
			Dl(
				Class("grid grid-cols-1 gap-4 sm:grid-cols-2"),
				organizationDetailItem("ID", data.Organization.ID.String()),
				organizationDetailItem("Name", data.Organization.Name),
				organizationDetailItem("Slug", data.Organization.Slug),
				organizationDetailItem("Members", fmt.Sprintf("%d", data.MemberCount)),
				organizationDetailItem("Created", formatOrgTime(data.Organization.CreatedAt)),
				organizationDetailItem("Updated", formatOrgTime(data.Organization.UpdatedAt)),
			),
		),
	)
}

func organizationDetailItem(label, value string) g.Node {
	return Div(
		Dt(Class("text-sm font-medium text-slate-600 dark:text-gray-400"), g.Text(label)),
		Dd(Class("mt-1 text-sm text-slate-900 dark:text-white"), g.Text(value)),
	)
}

// OrganizationCreatePage renders the organization creation form
func OrganizationCreatePage(appIDStr, csrfToken string) g.Node {
	return Div(
		Class("space-y-6"),
		Div(
			A(
				Href(fmt.Sprintf("/app/%s/organizations", appIDStr)),
				Class("text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400"),
				g.Text("← Back to Organizations"),
			),
			H1(
				Class("mt-2 text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Create Organization"),
			),
		),
		organizationForm(fmt.Sprintf("/app/%s/organizations/create", appIDStr), appIDStr, nil, csrfToken),
	)
}

// OrganizationEditData contains data for the organization edit page
type OrganizationEditData struct {
	Organization *organization.Organization
}

// OrganizationEditPage renders the organization edit form
func OrganizationEditPage(data OrganizationEditData, appIDStr, csrfToken string) g.Node {
	return Div(
		Class("space-y-6"),
		Div(
			A(
				Href(fmt.Sprintf("/app/%s/organizations/%s", appIDStr, data.Organization.ID.String())),
				Class("text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400"),
				g.Text("← Back to Organization"),
			),
			H1(
				Class("mt-2 text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Edit Organization"),
			),
		),
		organizationForm(fmt.Sprintf("/app/%s/organizations/%s/edit", appIDStr, data.Organization.ID.String()), appIDStr, data.Organization, csrfToken),
	)
}

func organizationForm(action string, appIDStr string, org *organization.Organization, csrfToken string) g.Node {
	name := ""
	slug := ""
	if org != nil {
		name = org.Name
		slug = org.Slug
	}

	return FormEl(
		Method("POST"),
		Action(action),
		Class("rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-6 space-y-6"),
		Input(
			Type("hidden"),
			Name("csrf_token"),
			Value(csrfToken),
		),
		Input(
			Type("hidden"),
			Name("redirect"),
			Value(action),
		),
		Div(
			Label(
				For("name"),
				Class("block text-sm font-medium text-slate-900 dark:text-white"),
				g.Text("Name"),
			),
			Input(
				Type("text"),
				ID("name"),
				Name("name"),
				Value(name),
				Required(),
				Class("mt-1 block w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-gray-500 focus:border-violet-300 dark:focus:border-violet-700 focus:ring-2 focus:ring-violet-400/20 dark:focus:ring-violet-400/20 sm:text-sm transition-colors"),
				g.Attr("placeholder", "My Organization"),
			),
		),
		g.If(org == nil,
			Div(
				Label(
					For("slug"),
					Class("block text-sm font-medium text-slate-900 dark:text-white"),
					g.Text("Slug"),
				),
				Input(
					Type("text"),
					ID("slug"),
					Name("slug"),
					Value(slug),
					Required(),
					Class("mt-1 block w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-2 text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-gray-500 focus:border-violet-300 dark:focus:border-violet-700 focus:ring-2 focus:ring-violet-400/20 dark:focus:ring-violet-400/20 sm:text-sm transition-colors"),
					g.Attr("placeholder", "my-org"),
				),
				P(
					Class("mt-1 text-xs text-slate-500 dark:text-gray-400"),
					g.Text("URL-friendly identifier (alphanumeric, hyphens, underscores)"),
				),
			),
		),
		Div(
			Class("flex items-center justify-end gap-3"),
			A(
				Href(func() string {
					if org != nil {
						return fmt.Sprintf("/app/%s/organizations/%s", appIDStr, org.ID.String())
					}
					return fmt.Sprintf("/app/%s/organizations", appIDStr)
				}()),
				Class("rounded-lg border border-slate-200 dark:border-gray-700 px-4 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-800 transition-colors"),
				g.Text("Cancel"),
			),
			Button(
				Type("submit"),
				Class("rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 active:bg-violet-800 transition-colors"),
				g.If(org == nil, g.Text("Create Organization")),
				g.If(org != nil, g.Text("Update Organization")),
			),
		),
	)
}

func formatOrgTime(t time.Time) string {
	return t.Format("Jan 2, 2006 3:04 PM")
}

func organizationsPagination(page, totalPages int, appIDStr string) g.Node {
	if totalPages <= 1 {
		return g.Text("")
	}

	baseURL := fmt.Sprintf("/app/%s/organizations", appIDStr)

	var pages []g.Node
	for i := 1; i <= totalPages; i++ {
		if i == page {
			pages = append(pages, Span(
				Class("inline-flex items-center rounded-lg border border-violet-200 dark:border-violet-800 bg-violet-50 dark:bg-violet-900/20 px-3 py-2 text-sm font-semibold text-violet-800 dark:text-violet-400"),
				g.Text(fmt.Sprintf("%d", i)),
			))
		} else {
			pages = append(pages, A(
				Href(fmt.Sprintf("%s?page=%d", baseURL, i)),
				Class("inline-flex items-center rounded-lg border border-slate-200 dark:border-gray-700 px-3 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-violet-50 dark:hover:bg-violet-900/20 hover:text-violet-800 dark:hover:text-violet-400 transition-colors"),
				g.Text(fmt.Sprintf("%d", i)),
			))
		}
	}

	return Div(
		Class("flex items-center justify-center gap-2 border-t border-slate-200 dark:border-gray-800 bg-slate-50 dark:bg-gray-800 px-5 py-4"),
		g.Group(pages),
	)
}
