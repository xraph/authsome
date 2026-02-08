package components

import (
	"github.com/xraph/authsome/core/app"
	"github.com/xraph/authsome/core/environment"
	"github.com/xraph/authsome/core/user"
	g "maragu.dev/gomponents"
)

// ExtensionNavItemData holds raw data for extension navigation items
type ExtensionNavItemData struct {
	Label    string
	Icon     g.Node
	URL      string
	IsActive bool
}

// PageData represents common data for all pages
type PageData struct {
	Title              string
	User               *user.User
	CSRFToken          string
	ActivePage         string
	BasePath           string
	Data               interface{}
	Error              string
	Success            string
	Year               int
	EnabledPlugins     map[string]bool
	IsMultiApp         bool
	CurrentApp         *app.App
	UserApps           []*app.App
	ShowAppSwitcher    bool
	CurrentEnvironment *environment.Environment
	UserEnvironments   []*environment.Environment
	ShowEnvSwitcher    bool
	ExtensionNavItems  []g.Node
	ExtensionNavData   []ExtensionNavItemData
	ExtensionWidgets   []g.Node
}
