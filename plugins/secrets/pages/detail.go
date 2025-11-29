package pages

import (
	"fmt"
	"strings"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/plugins/secrets/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// SecretDetailPage renders the secret detail page
func SecretDetailPage(
	currentApp *app.App,
	basePath string,
	secret *core.SecretDTO,
	versions []*core.SecretVersionDTO,
) g.Node {
	appBase := basePath + "/dashboard/app/" + currentApp.ID.String()

	return Div(
		Class("space-y-6"),

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
				Span(
					Class("text-slate-900 dark:text-white font-medium"),
					g.Text(secret.Path),
				),
			),

			// Title and actions
			Div(
				Class("flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between"),
				Div(
					// Path display
					Div(
						Class("flex items-center gap-3"),
						Div(
							Class("rounded-lg bg-violet-100 dark:bg-violet-900/30 p-2"),
							lucide.Key(Class("size-5 text-violet-600 dark:text-violet-400")),
						),
						Div(
							H1(
								Class("text-xl font-bold text-slate-900 dark:text-white font-mono"),
								g.Text(secret.Path),
							),
							g.If(secret.Description != "", func() g.Node {
								return P(
									Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
									g.Text(secret.Description),
								)
							}()),
						),
					),
				),

				// Action buttons
				Div(
					Class("flex items-center gap-2"),
					A(
						Href(appBase+"/secrets/"+secret.ID+"/edit"),
						Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700 transition-colors"),
						lucide.Pencil(Class("size-4")),
						g.Text("Edit"),
					),
					A(
						Href(appBase+"/secrets/"+secret.ID+"/history"),
						Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700 transition-colors"),
						lucide.History(Class("size-4")),
						g.Text("History"),
					),
					deleteButton(secret),
				),
			),
		),

		// Main content grid
		Div(
			Class("grid grid-cols-1 lg:grid-cols-3 gap-6"),

			// Left column - Value and metadata
			Div(
				Class("lg:col-span-2 space-y-6"),
				// Value section
				valueSection(appBase, secret),
				// Tags section
				g.If(len(secret.Tags) > 0, func() g.Node {
					return tagsSection(secret)
				}()),
			),

			// Right column - Info panel
			Div(
				Class("space-y-6"),
				infoPanel(secret),
				recentVersions(appBase, secret, versions),
			),
		),
	)
}

// valueSection renders the secret value with reveal functionality
func valueSection(appBase string, secret *core.SecretDTO) g.Node {
	return Div(
		Class("bg-white rounded-lg border border-slate-200 dark:bg-gray-900 dark:border-gray-800 overflow-hidden"),
		Div(
			Class("p-4 border-b border-slate-200 dark:border-gray-800"),
			Div(
				Class("flex items-center justify-between"),
				Div(
					Class("flex items-center gap-2"),
					H2(
						Class("text-sm font-medium text-slate-900 dark:text-white"),
						g.Text("Secret Value"),
					),
					valueTypeBadge(secret.ValueType),
				),
				Div(
					Class("flex items-center gap-2"),
					// Copy button
					Button(
						Type("button"),
						ID("copy-btn"),
						Class("inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-700 bg-slate-100 rounded-md hover:bg-slate-200 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700 transition-colors"),
						g.Attr("onclick", "copySecretValue()"),
						lucide.Copy(Class("size-3.5")),
						g.Text("Copy"),
					),
					// Reveal button
					Button(
						Type("button"),
						ID("reveal-btn"),
						Class("inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-violet-700 bg-violet-100 rounded-md hover:bg-violet-200 dark:bg-violet-900/30 dark:text-violet-400 dark:hover:bg-violet-900/50 transition-colors"),
						g.Attr("onclick", fmt.Sprintf("toggleReveal('%s')", appBase+"/secrets/"+secret.ID+"/reveal")),
						lucide.Eye(Class("size-3.5")),
						Span(ID("reveal-text"), g.Text("Reveal")),
					),
				),
			),
		),
		Div(
			Class("p-4"),
			// Masked value display
			Div(
				ID("secret-value"),
				Class("relative"),
				// Masked overlay
				Div(
					ID("masked-overlay"),
					Class("absolute inset-0 flex items-center justify-center bg-slate-50 dark:bg-gray-800 rounded-lg"),
					Div(
						Class("text-center"),
						lucide.EyeOff(Class("size-6 text-slate-400 mx-auto mb-2")),
						P(
							Class("text-sm text-slate-500 dark:text-gray-400"),
							g.Text("Click \"Reveal\" to show the secret value"),
						),
					),
				),
				// Actual value (hidden by default)
				Pre(
					ID("actual-value"),
					Class("hidden p-4 bg-slate-900 dark:bg-gray-950 rounded-lg text-sm font-mono text-green-400 overflow-x-auto"),
					Code(
						g.Text("********"),
					),
				),
			),
			// Value info
			Div(
				Class("mt-3 flex items-center gap-4 text-xs text-slate-500 dark:text-gray-400"),
				Div(
					Class("flex items-center gap-1"),
					lucide.FileType(Class("size-3.5")),
					g.Textf("Type: %s", secret.ValueType),
				),
				g.If(secret.HasSchema, func() g.Node {
					return Div(
						Class("flex items-center gap-1"),
						lucide.CircleCheck(Class("size-3.5 text-green-500")),
						g.Text("Schema validated"),
					)
				}()),
			),
		),

		// JavaScript for reveal functionality
		Script(g.Raw(`
			let revealed = false;
			let secretValue = '';
			
			async function toggleReveal(url) {
				const btn = document.getElementById('reveal-btn');
				const text = document.getElementById('reveal-text');
				const masked = document.getElementById('masked-overlay');
				const actual = document.getElementById('actual-value');
				
				if (revealed) {
					// Hide
					masked.classList.remove('hidden');
					actual.classList.add('hidden');
					text.textContent = 'Reveal';
					revealed = false;
				} else {
					// Show - fetch if needed
					if (!secretValue) {
						btn.disabled = true;
						text.textContent = 'Loading...';
						try {
							const response = await fetch(url, { method: 'POST' });
							const data = await response.json();
							secretValue = typeof data.value === 'object' 
								? JSON.stringify(data.value, null, 2) 
								: String(data.value);
							actual.querySelector('code').textContent = secretValue;
						} catch (err) {
							console.error('Failed to reveal:', err);
							text.textContent = 'Error';
							btn.disabled = false;
							return;
						}
						btn.disabled = false;
					}
					masked.classList.add('hidden');
					actual.classList.remove('hidden');
					text.textContent = 'Hide';
					revealed = true;
					
					// Auto-hide after 30 seconds
					setTimeout(() => {
						if (revealed) toggleReveal(url);
					}, 30000);
				}
			}
			
			function copySecretValue() {
				if (!secretValue) {
					alert('Please reveal the secret first');
					return;
				}
				navigator.clipboard.writeText(secretValue).then(() => {
					const btn = document.getElementById('copy-btn');
					btn.textContent = 'Copied!';
					setTimeout(() => {
						btn.innerHTML = '<svg class="size-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg> Copy';
					}, 2000);
				});
			}
		`)),
	)
}

// tagsSection renders the tags for a secret
func tagsSection(secret *core.SecretDTO) g.Node {
	tags := make([]g.Node, len(secret.Tags))
	for i, tag := range secret.Tags {
		tags[i] = Span(
			Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-700 dark:bg-gray-800 dark:text-gray-300"),
			lucide.Tag(Class("size-3 mr-1")),
			g.Text(tag),
		)
	}

	return Div(
		Class("bg-white rounded-lg border border-slate-200 dark:bg-gray-900 dark:border-gray-800 p-4"),
		H3(
			Class("text-sm font-medium text-slate-900 dark:text-white mb-3"),
			g.Text("Tags"),
		),
		Div(
			Class("flex flex-wrap gap-2"),
			g.Group(tags),
		),
	)
}

// infoPanel renders the secret metadata panel
func infoPanel(secret *core.SecretDTO) g.Node {
	// Build expiry node only if applicable
	var expiryNode g.Node
	if secret.HasExpiry && secret.ExpiresAt != nil {
		expiring := secret.ExpiresAt.Before(time.Now().Add(30 * 24 * time.Hour))
		expired := secret.ExpiresAt.Before(time.Now())
		var statusClass string
		if expired {
			statusClass = "text-red-600 dark:text-red-400"
		} else if expiring {
			statusClass = "text-orange-600 dark:text-orange-400"
		}
		expiryNode = Div(
			Class("px-4 py-3 flex justify-between"),
			Dt(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text("Expires")),
			Dd(
				Class("text-sm font-medium "+statusClass),
				g.Text(secret.ExpiresAt.Format("Jan 2, 2006")),
			),
		)
	}

	return Div(
		Class("bg-white rounded-lg border border-slate-200 dark:bg-gray-900 dark:border-gray-800 overflow-hidden"),
		Div(
			Class("p-4 border-b border-slate-200 dark:border-gray-800"),
			H3(
				Class("text-sm font-medium text-slate-900 dark:text-white"),
				g.Text("Details"),
			),
		),
		Dl(
			Class("divide-y divide-slate-100 dark:divide-gray-800"),
			infoItem("ID", secret.ID, true),
			infoItem("Version", fmt.Sprintf("v%d", secret.Version), false),
			infoItem("Value Type", secret.ValueType, false),
			infoItem("Created", secret.CreatedAt.Format("Jan 2, 2006"), false),
			infoItem("Updated", secret.UpdatedAt.Format("Jan 2, 2006"), false),
			expiryNode,
		),
	)
}

func infoItem(label, value string, mono bool) g.Node {
	valueClass := "text-sm font-medium text-slate-900 dark:text-white"
	if mono {
		valueClass += " font-mono text-xs"
	}

	displayValue := value
	if mono && len(value) > 20 {
		displayValue = value[:8] + "..." + value[len(value)-8:]
	}

	return Div(
		Class("px-4 py-3 flex justify-between"),
		Dt(Class("text-sm text-slate-500 dark:text-gray-400"), g.Text(label)),
		Dd(
			Class(valueClass),
			Title(value),
			g.Text(displayValue),
		),
	)
}

// recentVersions renders the recent version history
func recentVersions(appBase string, secret *core.SecretDTO, versions []*core.SecretVersionDTO) g.Node {
	if len(versions) == 0 {
		return g.Group(nil)
	}

	// Show only last 5
	displayVersions := versions
	if len(versions) > 5 {
		displayVersions = versions[:5]
	}

	items := make([]g.Node, len(displayVersions))
	for i, v := range displayVersions {
		items[i] = Div(
			Class("flex items-center justify-between py-2"),
			Div(
				Class("flex items-center gap-2"),
				g.If(v.Version == secret.Version,
					lucide.CircleCheck(Class("size-4 text-green-500")),
				),
				g.If(v.Version != secret.Version,
					lucide.Circle(Class("size-4 text-slate-300")),
				),
				Span(
					Class("text-sm text-slate-900 dark:text-white"),
					g.Textf("v%d", v.Version),
				),
			),
			Span(
				Class("text-xs text-slate-500 dark:text-gray-400"),
				g.Text(timeAgo(v.CreatedAt)),
			),
		)
	}

	return Div(
		Class("bg-white rounded-lg border border-slate-200 dark:bg-gray-900 dark:border-gray-800 overflow-hidden"),
		Div(
			Class("p-4 border-b border-slate-200 dark:border-gray-800 flex items-center justify-between"),
			H3(
				Class("text-sm font-medium text-slate-900 dark:text-white"),
				g.Text("Recent Versions"),
			),
			A(
				Href(appBase+"/secrets/"+secret.ID+"/history"),
				Class("text-xs text-violet-600 hover:text-violet-700 dark:text-violet-400"),
				g.Text("View all"),
			),
		),
		Div(
			Class("p-4"),
			Div(
				Class("divide-y divide-slate-100 dark:divide-gray-800"),
				g.Group(items),
			),
		),
	)
}

// deleteButton renders the delete confirmation button
func deleteButton(secret *core.SecretDTO) g.Node {
	return Div(
		Class("relative"),
		Button(
			Type("button"),
			Class("inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-red-700 bg-red-50 border border-red-200 rounded-lg hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400 dark:border-red-800 dark:hover:bg-red-900/30 transition-colors"),
			g.Attr("onclick", "document.getElementById('delete-modal').classList.remove('hidden')"),
			lucide.Trash2(Class("size-4")),
			g.Text("Delete"),
		),

		// Delete confirmation modal
		Div(
			ID("delete-modal"),
			Class("hidden fixed inset-0 z-50 flex items-center justify-center bg-black/50"),
			Div(
				Class("bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4 p-6"),
				// Icon
				Div(
					Class("mx-auto w-12 h-12 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center mb-4"),
					lucide.Trash2(Class("size-6 text-red-600 dark:text-red-400")),
				),
				// Title
				H3(
					Class("text-lg font-medium text-slate-900 dark:text-white text-center"),
					g.Text("Delete Secret"),
				),
				// Description
				P(
					Class("mt-2 text-sm text-slate-600 dark:text-gray-400 text-center"),
					g.Text("Are you sure you want to delete this secret? This action cannot be undone."),
				),
				// Path
				Div(
					Class("mt-4 p-3 rounded-lg bg-slate-100 dark:bg-gray-800 text-center"),
					Code(
						Class("text-sm font-mono text-slate-700 dark:text-gray-300"),
						g.Text(secret.Path),
					),
				),
				// Actions
				Div(
					Class("mt-6 flex items-center justify-end gap-3"),
					Button(
						Type("button"),
						Class("px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700 transition-colors"),
						g.Attr("onclick", "document.getElementById('delete-modal').classList.add('hidden')"),
						g.Text("Cancel"),
					),
					FormEl(
						Method("POST"),
						Action("?_method=DELETE"),
						Button(
							Type("submit"),
							Class("px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"),
							g.Text("Delete Secret"),
						),
					),
				),
			),
		),
	)
}

// PathBreadcrumb renders a breadcrumb from a secret path
func PathBreadcrumb(path string, appBase string) g.Node {
	parts := strings.Split(path, "/")
	items := make([]g.Node, 0, len(parts)*2)

	var currentPath string
	for i, part := range parts {
		if i > 0 {
			items = append(items, lucide.ChevronRight(Class("size-4 text-slate-400")))
			currentPath += "/"
		}
		currentPath += part

		isLast := i == len(parts)-1
		if isLast {
			items = append(items, Span(
				Class("text-slate-900 font-medium dark:text-white"),
				g.Text(part),
			))
		} else {
			items = append(items, A(
				Href(appBase+"/secrets?prefix="+currentPath),
				Class("text-slate-500 hover:text-violet-600 dark:text-gray-400 transition-colors"),
				g.Text(part),
			))
		}
	}

	return Nav(
		Class("flex items-center gap-2 text-sm"),
		g.Group(items),
	)
}
