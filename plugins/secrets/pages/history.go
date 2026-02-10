package pages

import (
	"fmt"
	"strconv"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/pagination"
	"github.com/xraph/authsome/plugins/secrets/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html" //nolint:staticcheck // dot import is intentional for UI library
)

// VersionHistoryPage renders the version history page for a secret.
func VersionHistoryPage(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	versions []*core.SecretVersionDTO,
	pag *pagination.Pagination,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	// Build pagination node only if needed
	var pagNode g.Node
	if pag != nil && pag.TotalPages > 1 {
		pagNode = historyPagination(appBase+"/secrets/"+secret.ID+"/history", pag.Page, pag.TotalPages)
	}

	return Div(
		Class("space-y-2"),

		// Header with breadcrumb
		Div(
			Class("space-y-4"),
			// Breadcrumb
			Nav(
				Class("flex items-center gap-2 text-sm"),
				A(
					Href(appBase+"/secrets"),
					Class("text-slate-500 hover:text-violet-600 dark:text-gray-400 transition-colors"),
					g.Text("Secrets"),
				),
				lucide.ChevronRight(Class("size-4 text-slate-400")),
				A(
					Href(appBase+"/secrets/"+secret.ID),
					Class("text-slate-500 hover:text-violet-600 dark:text-gray-400 transition-colors"),
					g.Text(secret.Path),
				),
				lucide.ChevronRight(Class("size-4 text-slate-400")),
				Span(
					Class("text-slate-900 dark:text-white font-medium"),
					g.Text("Version History"),
				),
			),

			// Title
			Div(
				Class("flex items-center justify-between"),
				Div(
					H1(
						Class("text-2xl font-bold text-slate-900 dark:text-white"),
						g.Text("Version History"),
					),
					P(
						Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
						g.Text("View and restore previous versions of this secret"),
					),
				),
				A(
					Href(appBase+"/secrets/"+secret.ID),
					Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700 transition-colors"),
					lucide.ArrowLeft(Class("size-4")),
					g.Text("Back to Secret"),
				),
			),
		),

		// Current version info
		currentVersionCard(secret),

		// Version timeline
		versionTimeline(currentApp, basePath, secret, versions),

		// Pagination
		pagNode,
	)
}

// currentVersionCard shows the current active version.
func currentVersionCard(secret *core.SecretDTO) g.Node {
	return Div(
		Class("bg-violet-50 dark:bg-violet-900/20 rounded-lg border border-violet-200 dark:border-violet-800 p-4"),
		Div(
			Class("flex items-center gap-3"),
			Div(
				Class("rounded-full bg-violet-100 dark:bg-violet-900/50 p-2"),
				lucide.CircleCheck(Class("size-5 text-violet-600 dark:text-violet-400")),
			),
			Div(
				Div(
					Class("text-sm font-medium text-violet-900 dark:text-violet-100"),
					g.Text(fmt.Sprintf("Current Version: v%d", secret.Version)),
				),
				Div(
					Class("text-xs text-violet-700 dark:text-violet-300"),
					g.Text("Last updated "+secret.UpdatedAt.Format("Jan 2, 2006 at 3:04 PM")),
				),
			),
		),
	)
}

// versionTimeline renders the timeline of versions.
func versionTimeline(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	versions []*core.SecretVersionDTO,
) g.Node {
	appBase := basePath + "/app/" + currentApp.ID.String()

	if len(versions) == 0 {
		return Div(
			Class("bg-white rounded-lg border border-slate-200 p-8 text-center dark:bg-gray-900 dark:border-gray-800"),
			lucide.History(Class("size-12 text-slate-300 mx-auto mb-4")),
			P(
				Class("text-sm text-slate-600 dark:text-gray-400"),
				g.Text("No version history available"),
			),
		)
	}

	items := make([]g.Node, len(versions))
	for i, version := range versions {
		isCurrent := version.Version == secret.Version
		items[i] = versionTimelineItem(appBase, secret, version, isCurrent, i == len(versions)-1)
	}

	return Div(
		Class("bg-white rounded-lg border border-slate-200 dark:bg-gray-900 dark:border-gray-800 overflow-hidden"),
		Div(
			Class("p-4 border-b border-slate-200 dark:border-gray-800"),
			H3(
				Class("text-sm font-medium text-slate-900 dark:text-white"),
				g.Text(fmt.Sprintf("Version Timeline (%d versions)", len(versions))),
			),
		),
		Div(
			Class("divide-y divide-slate-100 dark:divide-gray-800"),
			g.Group(items),
		),
	)
}

func versionTimelineItem(appBase string, secret *core.SecretDTO, version *core.SecretVersionDTO, isCurrent, isLast bool) g.Node {
	return Div(
		Class("relative p-4 hover:bg-slate-50 dark:hover:bg-gray-800/50 transition-colors"),

		// Timeline line
		g.If(!isLast, func() g.Node {
			return Div(
				Class("absolute left-7 top-12 bottom-0 w-0.5 bg-slate-200 dark:bg-gray-700"),
			)
		}()),

		Div(
			Class("flex items-start gap-4"),

			// Timeline dot
			Div(
				g.If(isCurrent,
					Div(
						Class("relative z-10 rounded-full bg-violet-100 dark:bg-violet-900/50 p-2"),
						lucide.CircleCheck(Class("size-4 text-violet-600 dark:text-violet-400")),
					),
				),
				g.If(!isCurrent,
					Div(
						Class("relative z-10 rounded-full bg-slate-100 dark:bg-gray-800 p-2"),
						lucide.Circle(Class("size-4 text-slate-400")),
					),
				),
			),

			// Content
			Div(
				Class("flex-1 min-w-0"),
				Div(
					Class("flex items-center justify-between gap-4"),
					Div(
						Div(
							Class("flex items-center gap-2"),
							Span(
								Class("text-sm font-medium text-slate-900 dark:text-white"),
								g.Text(fmt.Sprintf("Version %d", version.Version)),
							),
							g.If(isCurrent, func() g.Node {
								return Span(
									Class("inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400"),
									g.Text("Current"),
								)
							}()),
						),
						Div(
							Class("text-xs text-slate-500 dark:text-gray-400 mt-0.5"),
							g.Text(version.CreatedAt.Format("Jan 2, 2006 at 3:04 PM")),
						),
					),

					// Rollback button (only for non-current versions)
					g.If(!isCurrent, func() g.Node {
						return FormEl(
							Method("POST"),
							Action(appBase+"/secrets/"+secret.ID+"/rollback/"+strconv.Itoa(version.Version)),
							Button(
								Type("submit"),
								Class("inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-orange-700 bg-orange-50 border border-orange-200 rounded-md hover:bg-orange-100 dark:text-orange-400 dark:bg-orange-900/20 dark:border-orange-800 dark:hover:bg-orange-900/30 transition-colors"),
								lucide.RotateCcw(Class("size-3")),
								g.Text("Rollback"),
							),
						)
					}()),
				),

				// Change reason
				g.If(version.ChangeReason != "", func() g.Node {
					return Div(
						Class("mt-2 text-sm text-slate-600 dark:text-gray-400"),
						Span(Class("font-medium"), g.Text("Reason: ")),
						g.Text(version.ChangeReason),
					)
				}()),

				// Changed by
				g.If(version.ChangedBy != "", func() g.Node {
					return Div(
						Class("mt-1 flex items-center gap-1 text-xs text-slate-500 dark:text-gray-500"),
						lucide.User(Class("size-3")),
						g.Text("Changed by "),
						Span(Class("font-medium"), g.Text(version.ChangedBy)),
					)
				}()),

				// Time ago
				Div(
					Class("mt-1 text-xs text-slate-400 dark:text-gray-500"),
					g.Text(timeAgo(version.CreatedAt)),
				),
			),
		),
	)
}

func historyPagination(baseURL string, currentPage, totalPages int) g.Node {
	items := make([]g.Node, 0)

	// Previous
	if currentPage > 1 {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage-1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			lucide.ChevronLeft(Class("size-4")),
		))
	}

	// Page info
	items = append(items, Span(
		Class("px-3 py-2 text-sm text-slate-600 dark:text-gray-400"),
		g.Text(fmt.Sprintf("Page %d of %d", currentPage, totalPages)),
	))

	// Next
	if currentPage < totalPages {
		items = append(items, A(
			Href(fmt.Sprintf("%s?page=%d", baseURL, currentPage+1)),
			Class("px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700"),
			lucide.ChevronRight(Class("size-4")),
		))
	}

	return Div(
		Class("flex items-center justify-center gap-2 mt-6"),
		g.Group(items),
	)
}

// RollbackConfirmationModal renders a confirmation modal for rollback.
func RollbackConfirmationModal(secret *core.SecretDTO, targetVersion int) g.Node {
	return Div(
		Class("fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
		Div(
			Class("bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4 p-6"),

			// Icon
			Div(
				Class("mx-auto w-12 h-12 rounded-full bg-orange-100 dark:bg-orange-900/30 flex items-center justify-center mb-4"),
				lucide.RotateCcw(Class("size-6 text-orange-600 dark:text-orange-400")),
			),

			// Title
			H3(
				Class("text-lg font-medium text-slate-900 dark:text-white text-center"),
				g.Text("Confirm Rollback"),
			),

			// Description
			P(
				Class("mt-2 text-sm text-slate-600 dark:text-gray-400 text-center"),
				g.Textf("Are you sure you want to rollback to version %d? The current version (v%d) will be preserved in history.", targetVersion, secret.Version),
			),

			// Warning
			Div(
				Class("mt-4 p-3 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800"),
				Div(
					Class("flex items-start gap-2"),
					lucide.CircleAlert(Class("size-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5")),
					P(
						Class("text-xs text-amber-700 dark:text-amber-300"),
						g.Text("This action will update the secret value to the selected version. Any applications using this secret will receive the rolled-back value."),
					),
				),
			),

			// Actions
			Div(
				Class("mt-6 flex items-center justify-end gap-3"),
				Button(
					Type("button"),
					Class("px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700 transition-colors"),
					g.Text("Cancel"),
				),
				Button(
					Type("submit"),
					Class("px-4 py-2 text-sm font-medium text-white bg-orange-600 rounded-lg hover:bg-orange-700 transition-colors"),
					g.Text("Confirm Rollback"),
				),
			),
		),
	)
}

// VersionDiff renders a comparison between two versions.
func VersionDiff(oldVersion, newVersion *core.SecretVersionDTO) g.Node {
	return Div(
		Class("bg-white rounded-lg border border-slate-200 dark:bg-gray-900 dark:border-gray-800 overflow-hidden"),
		Div(
			Class("p-4 border-b border-slate-200 dark:border-gray-800"),
			H3(
				Class("text-sm font-medium text-slate-900 dark:text-white"),
				g.Textf("Comparing Version %d → Version %d", oldVersion.Version, newVersion.Version),
			),
		),
		Div(
			Class("grid grid-cols-2 divide-x divide-slate-200 dark:divide-gray-800"),
			// Old version
			Div(
				Class("p-4"),
				Div(
					Class("text-xs font-medium text-red-600 dark:text-red-400 mb-2"),
					g.Textf("Version %d (Previous)", oldVersion.Version),
				),
				Div(
					Class("bg-red-50 dark:bg-red-900/20 rounded p-3 text-sm font-mono text-slate-600 dark:text-gray-400"),
					g.Text("[Value encrypted]"),
				),
			),
			// New version
			Div(
				Class("p-4"),
				Div(
					Class("text-xs font-medium text-green-600 dark:text-green-400 mb-2"),
					g.Textf("Version %d (Current)", newVersion.Version),
				),
				Div(
					Class("bg-green-50 dark:bg-green-900/20 rounded p-3 text-sm font-mono text-slate-600 dark:text-gray-400"),
					g.Text("[Value encrypted]"),
				),
			),
		),
	)
}
