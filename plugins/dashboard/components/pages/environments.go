package pages

import (
	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/environment"
	p "github.com/xraph/authsome/core/pagination"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// EnvironmentsData holds data for environments list page
type EnvironmentsData struct {
	Environments []*environment.Environment
	Pagination   *p.PageMeta `json:"pagination,omitempty"`
}

// EnvironmentsPage renders the environments list page
func EnvironmentsPage(data EnvironmentsData, basePath string, appIDStr string) g.Node {
	return g.Group([]g.Node{
		// Page Actions Bar
		Div(
			Class("mb-6 flex items-center justify-between"),
			Div(
				H2(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text("Environments")),
				P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"), g.Text("Manage application environments")),
			),
			A(
				Href(basePath+"/dashboard/app/"+appIDStr+"/environments/create"),
				Class("inline-flex items-center justify-center gap-2 rounded-lg border border-transparent bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 transition-colors"),
				lucide.Plus(Class("h-5 w-5")),
				g.Text("Create Environment"),
			),
		),

		// Environments Grid
		g.If(len(data.Environments) > 0, environmentsGrid(data.Environments, basePath, appIDStr)),
		g.If(len(data.Environments) == 0, emptyEnvironmentsState(basePath, appIDStr)),
	})
}

func environmentsGrid(environments []*environment.Environment, basePath string, appIDStr string) g.Node {
	cards := []g.Node{}
	for _, env := range environments {
		cards = append(cards, environmentCard(env, basePath, appIDStr))
	}

	return Div(
		Class("grid gap-4 sm:grid-cols-2 lg:grid-cols-3"),
		g.Group(cards),
	)
}

func environmentCard(env *environment.Environment, basePath string, appIDStr string) g.Node {
	detailURL := basePath + "/dashboard/app/" + appIDStr + "/environments/" + env.ID.String()
	editURL := detailURL + "/edit"

	// Status badge
	statusBadge := Span(
		Class("px-2 py-1 text-xs font-semibold rounded-full bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400"),
		g.Text("Active"),
	)

	// Type badge
	typeBadge := environmentTypeBadge(env.Type)

	// Default badge
	defaultBadge := g.If(env.IsDefault, Span(
		Class("px-2 py-1 text-xs font-semibold rounded-full bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400"),
		g.Text("Default"),
	))

	return Div(
		Class("rounded-xl border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6 hover:shadow-lg transition-shadow"),
		Div(
			Class("space-y-4"),
			// Header
			Div(
				Class("flex items-start justify-between"),
				Div(
					Class("flex-1"),
					H3(
						Class("text-lg font-bold text-slate-900 dark:text-white"),
						g.Text(env.Name),
					),
					P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"), g.Text(env.Slug)),
				),
				Div(
					Class("flex items-center gap-2"),
					typeBadge,
				),
			),

			// Badges
			Div(
				Class("flex flex-wrap items-center gap-2"),
				statusBadge,
				defaultBadge,
			),

			// Actions
			Div(
				Class("flex items-center gap-2 pt-4 border-t border-slate-100 dark:border-gray-700"),
				A(
					Href(detailURL),
					Class("flex-1 inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 px-3 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-700 transition-colors"),
					lucide.Eye(Class("h-4 w-4")),
					g.Text("View"),
				),
				A(
					Href(editURL),
					Class("flex-1 inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 px-3 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-700 transition-colors"),
					lucide.Pencil(Class("h-4 w-4")),
					g.Text("Edit"),
				),
			),
		),
	)
}

func environmentTypeBadge(envType string) g.Node {
	badgeColor := "bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
	switch envType {
	case "production":
		badgeColor = "bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400"
	case "staging":
		badgeColor = "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400"
	case "development":
		badgeColor = "bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400"
	}

	return Span(
		Class("px-2 py-1 text-xs font-semibold rounded-full "+badgeColor),
		g.Text(envType),
	)
}

func emptyEnvironmentsState(basePath string, appIDStr string) g.Node {
	return Div(
		Class("text-center py-12"),
		Div(
			Class("mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-slate-100 dark:bg-gray-800"),
			lucide.Layers(Class("h-6 w-6 text-slate-400 dark:text-gray-500")),
		),
		H3(
			Class("mt-4 text-lg font-semibold text-slate-900 dark:text-white"),
			g.Text("No environments"),
		),
		P(
			Class("mt-2 text-sm text-slate-500 dark:text-gray-400"),
			g.Text("Get started by creating your first environment"),
		),
		A(
			Href(basePath+"/dashboard/app/"+appIDStr+"/environments/create"),
			Class("mt-6 inline-flex items-center justify-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 transition-colors"),
			lucide.Plus(Class("h-5 w-5")),
			g.Text("Create Environment"),
		),
	)
}

// EnvironmentDetailData holds data for environment detail page
type EnvironmentDetailData struct {
	Environment *environment.Environment
}

// EnvironmentDetailPage renders the environment detail page
func EnvironmentDetailPage(data EnvironmentDetailData, basePath string, appIDStr string) g.Node {
	env := data.Environment
	editURL := basePath + "/dashboard/app/" + appIDStr + "/environments/" + env.ID.String() + "/edit"
	listURL := basePath + "/dashboard/app/" + appIDStr + "/environments"

	return g.Group([]g.Node{
		// Back button
		Div(
			Class("mb-6"),
			A(
				Href(listURL),
				Class("inline-flex items-center gap-2 text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400"),
				lucide.ArrowLeft(Class("h-4 w-4")),
				g.Text("Back to Environments"),
			),
		),

		// Header
		Div(
			Class("mb-6 flex items-center justify-between"),
			Div(
				H2(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text(env.Name)),
				P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"), g.Text(env.Slug)),
			),
			A(
				Href(editURL),
				Class("inline-flex items-center justify-center gap-2 rounded-lg border border-transparent bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 transition-colors"),
				lucide.Pencil(Class("h-5 w-5")),
				g.Text("Edit Environment"),
			),
		),

		// Details Card
		Div(
			Class("rounded-xl border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6"),
			Div(
				Class("space-y-6"),
				detailRow("ID", env.ID.String()),
				detailRow("Name", env.Name),
				detailRow("Slug", env.Slug),
				detailRow("Type", env.Type),
				detailRow("Default", Div(g.If(env.IsDefault, g.Text("Yes")), g.If(!env.IsDefault, g.Text("No")))),
				detailRow("Created", env.CreatedAt.Format("Jan 2, 2006 at 3:04 PM")),
				detailRow("Updated", env.UpdatedAt.Format("Jan 2, 2006 at 3:04 PM")),
			),
		),
	})
}

func detailRow(label string, value interface{}) g.Node {
	var valueNode g.Node
	switch v := value.(type) {
	case string:
		valueNode = g.Text(v)
	case g.Node:
		valueNode = v
	default:
		valueNode = g.Text("")
	}

	return Div(
		Class("flex items-center justify-between py-3 border-b border-slate-100 dark:border-gray-700 last:border-0"),
		Dt(Class("text-sm font-medium text-slate-500 dark:text-gray-400"), g.Text(label)),
		Dd(Class("text-sm text-slate-900 dark:text-white font-semibold"), valueNode),
	)
}

// EnvironmentCreatePage renders the create environment page
func EnvironmentCreatePage(basePath string, appIDStr string, csrfToken string) g.Node {
	listURL := basePath + "/dashboard/app/" + appIDStr + "/environments"
	formAction := basePath + "/dashboard/app/" + appIDStr + "/environments/create"

	return g.Group([]g.Node{
		// Back button
		Div(
			Class("mb-6"),
			A(
				Href(listURL),
				Class("inline-flex items-center gap-2 text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400"),
				lucide.ArrowLeft(Class("h-4 w-4")),
				g.Text("Back to Environments"),
			),
		),

		// Header
		Div(
			Class("mb-6"),
			H2(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text("Create Environment")),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"), g.Text("Add a new environment to your application")),
		),

		// Form Card
		Div(
			Class("rounded-xl border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6"),
			Form(
				Method("POST"),
				Action(formAction),
				Input(Type("hidden"), Name("csrf_token"), Value(csrfToken)),
				Div(
					Class("space-y-6"),
					environmentFormFields(nil),
					// Actions
					Div(
						Class("flex items-center gap-3 pt-6 border-t border-slate-100 dark:border-gray-700"),
						Button(
							Type("submit"),
							Class("inline-flex items-center justify-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 transition-colors"),
							lucide.Check(Class("h-5 w-5")),
							g.Text("Create Environment"),
						),
						A(
							Href(listURL),
							Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 px-4 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-700 transition-colors"),
							g.Text("Cancel"),
						),
					),
				),
			),
		),
	})
}

// EnvironmentEditData holds data for environment edit page
type EnvironmentEditData struct {
	Environment *environment.Environment
}

// EnvironmentEditPage renders the edit environment page
func EnvironmentEditPage(data EnvironmentEditData, basePath string, appIDStr string, csrfToken string) g.Node {
	env := data.Environment
	detailURL := basePath + "/dashboard/app/" + appIDStr + "/environments/" + env.ID.String()
	formAction := detailURL + "/edit"
	deleteAction := detailURL + "/delete"

	return g.Group([]g.Node{
		// Back button
		Div(
			Class("mb-6"),
			A(
				Href(detailURL),
				Class("inline-flex items-center gap-2 text-sm text-slate-600 dark:text-gray-400 hover:text-violet-600 dark:hover:text-violet-400"),
				lucide.ArrowLeft(Class("h-4 w-4")),
				g.Text("Back to Environment"),
			),
		),

		// Header
		Div(
			Class("mb-6"),
			H2(Class("text-2xl font-bold text-slate-900 dark:text-white"), g.Text("Edit Environment")),
			P(Class("mt-1 text-sm text-slate-500 dark:text-gray-400"), g.Text("Update environment settings")),
		),

		// Form Card
		Div(
			Class("rounded-xl border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-6"),
			Form(
				Method("POST"),
				Action(formAction),
				Input(Type("hidden"), Name("csrf_token"), Value(csrfToken)),
				Div(
					Class("space-y-6"),
					environmentFormFields(env),
					// Actions
					Div(
						Class("flex items-center justify-between pt-6 border-t border-slate-100 dark:border-gray-700"),
						Div(
							Class("flex items-center gap-3"),
							Button(
								Type("submit"),
								Class("inline-flex items-center justify-center gap-2 rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 transition-colors"),
								lucide.Check(Class("h-5 w-5")),
								g.Text("Save Changes"),
							),
							A(
								Href(detailURL),
								Class("inline-flex items-center justify-center gap-2 rounded-lg border border-slate-200 dark:border-gray-700 px-4 py-2 text-sm font-semibold text-slate-700 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-700 transition-colors"),
								g.Text("Cancel"),
							),
						),
						// Delete button (only if not default)
						g.If(!env.IsDefault,
							Form(
								Method("POST"),
								Action(deleteAction),
								Class("inline"),
								g.Attr("onsubmit", "return confirm('Are you sure you want to delete this environment? This action cannot be undone.')"),
								Input(Type("hidden"), Name("csrf_token"), Value(csrfToken)),
								Button(
									Type("submit"),
									Class("inline-flex items-center justify-center gap-2 rounded-lg border border-red-200 dark:border-red-800 px-4 py-2 text-sm font-semibold text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"),
									lucide.Trash2(Class("h-5 w-5")),
									g.Text("Delete"),
								),
							),
						),
					),
				),
			),
		),
	})
}

func environmentFormFields(env *environment.Environment) g.Node {
	nameValue := ""
	slugValue := ""
	typeValue := "development"
	isDefault := false

	if env != nil {
		nameValue = env.Name
		slugValue = env.Slug
		typeValue = env.Type
		isDefault = env.IsDefault
	}

	return g.Group([]g.Node{
		// Name
		Div(
			Label(
				For("name"),
				Class("block text-sm font-medium text-slate-900 dark:text-white mb-2"),
				g.Text("Name"),
			),
			Input(
				Type("text"),
				ID("name"),
				Name("name"),
				Value(nameValue),
				Required(),
				Class("w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-4 py-2 text-sm text-slate-900 dark:text-white focus:border-violet-500 focus:ring-2 focus:ring-violet-500/20"),
				Placeholder("Production"),
			),
		),

		// Slug
		Div(
			Label(
				For("slug"),
				Class("block text-sm font-medium text-slate-900 dark:text-white mb-2"),
				g.Text("Slug"),
			),
			Input(
				Type("text"),
				ID("slug"),
				Name("slug"),
				Value(slugValue),
				Required(),
				Class("w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-4 py-2 text-sm text-slate-900 dark:text-white focus:border-violet-500 focus:ring-2 focus:ring-violet-500/20"),
				Placeholder("production"),
			),
			P(Class("mt-1 text-xs text-slate-500 dark:text-gray-400"), g.Text("URL-friendly identifier (lowercase, no spaces)")),
		),

		// Type
		Div(
			Label(
				For("type"),
				Class("block text-sm font-medium text-slate-900 dark:text-white mb-2"),
				g.Text("Type"),
			),
			Select(
				ID("type"),
				Name("type"),
				Required(),
				Class("w-full rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-900 px-4 py-2 text-sm text-slate-900 dark:text-white focus:border-violet-500 focus:ring-2 focus:ring-violet-500/20"),
				Option(Value("development"), g.If(typeValue == "development", Selected()), g.Text("Development")),
				Option(Value("staging"), g.If(typeValue == "staging", Selected()), g.Text("Staging")),
				Option(Value("production"), g.If(typeValue == "production", Selected()), g.Text("Production")),
			),
		),

		// Is Default (only for create, cannot change for existing)
		g.If(env == nil,
			Div(
				Class("flex items-center gap-2"),
				Input(
					Type("checkbox"),
					ID("is_default"),
					Name("is_default"),
					Value("true"),
					g.If(isDefault, Checked()),
					Class("h-4 w-4 rounded border-slate-300 dark:border-gray-600 text-violet-600 focus:ring-violet-500"),
				),
				Label(
					For("is_default"),
					Class("text-sm font-medium text-slate-900 dark:text-white"),
					g.Text("Set as default environment"),
				),
			),
		),
	})
}
