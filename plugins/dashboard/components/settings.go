package components

import (
	"fmt"

	"github.com/xraph/authsome/core/app"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// SettingsPageHeader renders a standard settings page header with title and description
func SettingsPageHeader(title, description string) g.Node {
	return Div(
		Class("mb-6"),
		H1(Class("text-2xl font-semibold text-gray-900 dark:text-white"), g.Text(title)),
		P(Class("mt-1 text-sm text-gray-500 dark:text-gray-400"), g.Text(description)),
	)
}

// SettingsNavItem represents a navigation item in the settings sidebar
type SettingsNavItem struct {
	ID            string
	Label         string
	Icon          g.Node
	URL           string
	Category      string
	RequirePlugin string
}

// SettingsLayoutData contains data for the settings layout
type SettingsLayoutData struct {
	NavItems    []SettingsNavItem
	ActivePage  string
	BasePath    string
	CurrentApp  *app.App
	PageContent g.Node
}

// BuildSettingsURL builds a settings page URL
func BuildSettingsURL(basePath string, appID, page string) string {
	if appID == "" {
		return fmt.Sprintf("%s/settings/%s", basePath, page)
	}
	return fmt.Sprintf("%s/app/%s/settings/%s", basePath, appID, page)
}

// SettingsLayout renders a minimal settings page wrapper
func SettingsLayout(data SettingsLayoutData) g.Node {
	return Div(
		Class("settings-layout min-h-screen p-4"),
		data.PageContent,
	)
}
