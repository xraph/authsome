package pages

import (
	"fmt"
	"sort"
	"time"

	lucide "github.com/eduardolat/gomponents-lucide"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// ConfigSourceMetadata represents metadata about a configuration source
type ConfigSourceMetadata struct {
	Name         string
	Type         string
	Priority     int
	LastLoaded   time.Time
	LastModified time.Time
	IsWatching   bool
	KeyCount     int
	ErrorCount   int
	LastError    string
}

// ConfigViewerPageData contains data for the config viewer page
type ConfigViewerPageData struct {
	ConfigYAML     string
	SourceMetadata []ConfigSourceMetadata
	BasePath       string
}

// ConfigViewerPage renders the configuration viewer page
func ConfigViewerPage(data ConfigViewerPageData) g.Node {
	return Div(Class("space-y-6"),
		// Page Header
		configViewerHeader(),

		// Source Metadata Section
		sourceMetadataSection(data.SourceMetadata),

		// Configuration YAML Section
		configYAMLSection(data.ConfigYAML),
	)
}

func configViewerHeader() g.Node {
	return Div(Class("flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between"),
		Div(
			H1(Class("text-2xl font-bold text-slate-900 dark:text-white"),
				g.Text("Configuration Viewer"),
			),
			P(Class("mt-1 text-sm text-slate-600 dark:text-gray-400"),
				g.Text("View all configuration values from Forge ConfigManager with source information"),
			),
		),
		Div(Class("flex items-center gap-2"),
			Span(
				Class("inline-flex items-center gap-1.5 rounded-full bg-amber-100 dark:bg-amber-900/30 px-3 py-1 text-xs font-medium text-amber-800 dark:text-amber-400"),
				lucide.Eye(Class("h-3.5 w-3.5")),
				g.Text("Read-Only"),
			),
		),
	)
}

func sourceMetadataSection(sources []ConfigSourceMetadata) g.Node {
	if len(sources) == 0 {
		return Div(
			Class("rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 p-6"),
			Div(Class("flex items-center gap-3"),
				lucide.Info(Class("h-5 w-5 text-slate-400")),
				P(Class("text-sm text-slate-600 dark:text-gray-400"),
					g.Text("No configuration sources available"),
				),
			),
		)
	}

	// Sort sources by priority (higher priority first)
	sortedSources := make([]ConfigSourceMetadata, len(sources))
	copy(sortedSources, sources)
	sort.Slice(sortedSources, func(i, j int) bool {
		return sortedSources[i].Priority > sortedSources[j].Priority
	})

	return Div(Class("space-y-4"),
		// Section Header
		Div(Class("flex items-center gap-2"),
			lucide.Database(Class("h-5 w-5 text-violet-500")),
			H2(Class("text-lg font-semibold text-slate-900 dark:text-white"),
				g.Text("Configuration Sources"),
			),
			Span(
				Class("ml-2 rounded-full bg-slate-100 dark:bg-gray-800 px-2 py-0.5 text-xs font-medium text-slate-600 dark:text-gray-400"),
				g.Textf("%d sources", len(sources)),
			),
		),

		// Sources Grid
		Div(Class("grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"),
			g.Group(sourceCards(sortedSources)),
		),
	)
}

func sourceCards(sources []ConfigSourceMetadata) []g.Node {
	cards := make([]g.Node, len(sources))
	for i, source := range sources {
		cards[i] = sourceCard(source)
	}
	return cards
}

func sourceCard(source ConfigSourceMetadata) g.Node {
	// Determine status badge
	var statusBadge g.Node
	if source.ErrorCount > 0 {
		statusBadge = Span(
			Class("inline-flex items-center gap-1 rounded-full bg-red-100 dark:bg-red-900/30 px-2 py-0.5 text-xs font-medium text-red-700 dark:text-red-400"),
			lucide.CircleAlert(Class("h-3 w-3")),
			g.Textf("%d errors", source.ErrorCount),
		)
	} else if source.IsWatching {
		statusBadge = Span(
			Class("inline-flex items-center gap-1 rounded-full bg-emerald-100 dark:bg-emerald-900/30 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:text-emerald-400"),
			lucide.Activity(Class("h-3 w-3")),
			g.Text("Watching"),
		)
	} else {
		statusBadge = Span(
			Class("inline-flex items-center gap-1 rounded-full bg-slate-100 dark:bg-slate-900/30 px-2 py-0.5 text-xs font-medium text-slate-600 dark:text-slate-400"),
			lucide.Check(Class("h-3 w-3")),
			g.Text("Loaded"),
		)
	}

	// Get icon based on source type
	icon := getSourceIcon(source.Type)

	return Div(
		Class("rounded-lg border border-slate-200 dark:border-gray-800 bg-white dark:bg-gray-900 overflow-hidden"),
		// Card Header
		Div(
			Class("px-4 py-3 border-b border-slate-100 dark:border-gray-800 bg-slate-50 dark:bg-gray-800/50"),
			Div(Class("flex items-center justify-between"),
				Div(Class("flex items-center gap-2"),
					icon,
					Span(Class("font-medium text-slate-900 dark:text-white truncate max-w-[150px]"),
						g.Attr("title", source.Name),
						g.Text(source.Name),
					),
				),
				statusBadge,
			),
		),

		// Card Body
		Div(
			Class("px-4 py-3 space-y-2"),
			// Type
			Div(Class("flex items-center justify-between text-sm"),
				Span(Class("text-slate-500 dark:text-gray-400"), g.Text("Type")),
				Span(Class("font-medium text-slate-700 dark:text-gray-300"), g.Text(source.Type)),
			),
			// Priority
			Div(Class("flex items-center justify-between text-sm"),
				Span(Class("text-slate-500 dark:text-gray-400"), g.Text("Priority")),
				Span(Class("font-medium text-slate-700 dark:text-gray-300"), g.Textf("%d", source.Priority)),
			),
			// Key Count
			Div(Class("flex items-center justify-between text-sm"),
				Span(Class("text-slate-500 dark:text-gray-400"), g.Text("Keys")),
				Span(Class("font-medium text-slate-700 dark:text-gray-300"), g.Textf("%d", source.KeyCount)),
			),
			// Last Loaded
			g.If(!source.LastLoaded.IsZero(),
				Div(Class("flex items-center justify-between text-sm"),
					Span(Class("text-slate-500 dark:text-gray-400"), g.Text("Last Loaded")),
					Span(Class("font-medium text-slate-700 dark:text-gray-300 text-xs"),
						g.Text(formatTime(source.LastLoaded)),
					),
				),
			),
		),

		// Error Display
		g.If(source.LastError != "",
			Div(
				Class("px-4 py-2 bg-red-50 dark:bg-red-900/20 border-t border-red-100 dark:border-red-900/30"),
				P(Class("text-xs text-red-600 dark:text-red-400 truncate"),
					g.Attr("title", source.LastError),
					g.Text(source.LastError),
				),
			),
		),
	)
}

func getSourceIcon(sourceType string) g.Node {
	iconClass := "h-4 w-4 text-slate-500 dark:text-gray-400"
	switch sourceType {
	case "file", "yaml", "json", "toml":
		return lucide.FileJson(Class(iconClass))
	case "env", "environment":
		return lucide.Terminal(Class(iconClass))
	case "remote", "http", "https":
		return lucide.Globe(Class(iconClass))
	case "vault", "secrets":
		return lucide.Lock(Class(iconClass))
	case "memory":
		return lucide.Cpu(Class(iconClass))
	default:
		return lucide.Settings(Class(iconClass))
	}
}

func configYAMLSection(configYAML string) g.Node {
	return Div(Class("space-y-4"),
		// Section Header
		Div(Class("flex items-center justify-between"),
			Div(Class("flex items-center gap-2"),
				lucide.FileCode(Class("h-5 w-5 text-violet-500")),
				H2(Class("text-lg font-semibold text-slate-900 dark:text-white"),
					g.Text("Configuration Values"),
				),
			),
			// Copy Button
			Button(
				Class("inline-flex items-center gap-1.5 rounded-lg border border-slate-200 dark:border-gray-700 bg-white dark:bg-gray-800 px-3 py-1.5 text-xs font-medium text-slate-700 dark:text-gray-300 hover:bg-slate-50 dark:hover:bg-gray-700 transition-colors"),
				g.Attr("onclick", "copyConfigToClipboard()"),
				lucide.Copy(Class("h-3.5 w-3.5")),
				g.Text("Copy"),
			),
		),

		// YAML Code Block
		Div(
			Class("rounded-lg border border-slate-200 dark:border-gray-800 bg-slate-900 dark:bg-gray-950 overflow-hidden"),
			// Code Header
			Div(
				Class("flex items-center justify-between px-4 py-2 border-b border-slate-700 bg-slate-800 dark:bg-gray-900"),
				Div(Class("flex items-center gap-2"),
					Span(Class("h-3 w-3 rounded-full bg-red-500")),
					Span(Class("h-3 w-3 rounded-full bg-yellow-500")),
					Span(Class("h-3 w-3 rounded-full bg-green-500")),
				),
				Span(Class("text-xs text-slate-400"), g.Text("config.yaml")),
			),
			// Code Content
			Div(
				Class("p-4 overflow-auto max-h-[600px]"),
				Pre(
					ID("config-yaml"),
					Class("text-sm font-mono text-slate-300 whitespace-pre"),
					Code(
						g.Text(configYAML),
					),
				),
			),
		),

		// Copy Script
		Script(g.Raw(`
			function copyConfigToClipboard() {
				const configText = document.getElementById('config-yaml').innerText;
				navigator.clipboard.writeText(configText).then(function() {
					// Show a brief notification (could be improved with a toast)
					const btn = event.target.closest('button');
					const originalText = btn.innerHTML;
					btn.innerHTML = '<svg class="h-3.5 w-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg> Copied!';
					setTimeout(() => {
						btn.innerHTML = originalText;
					}, 2000);
				}).catch(function(err) {
					console.error('Failed to copy: ', err);
				});
			}
		`)),
	)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	// If within the last 24 hours, show relative time
	since := time.Since(t)
	if since < time.Minute {
		return "just now"
	}
	if since < time.Hour {
		return fmt.Sprintf("%dm ago", int(since.Minutes()))
	}
	if since < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(since.Hours()))
	}
	return t.Format("Jan 2, 15:04")
}

