package pages

import (
	"github.com/xraph/authsome/plugins/dashboard/v2/layouts"
	"github.com/xraph/authsome/plugins/dashboard/v2/services"
	"github.com/xraph/forgeui"
)

type PagesManager struct {
	fuiApp     *forgeui.App
	baseUIPath string
	services   *services.Services
}

func NewPagesManager(fuiApp *forgeui.App, baseUIPath string) *PagesManager {
	return &PagesManager{
		fuiApp:     fuiApp,
		baseUIPath: baseUIPath,
	}
}

func (p *PagesManager) RegisterPages() error {
	if err := p.registerAuthPages(); err != nil {
		return err
	}

	if err := p.registerDashboardPages(); err != nil {
		return err
	}

	return nil
}

func (p *PagesManager) registerAuthPages() error {
	authGroup := p.fuiApp.Group("/auth").
		Middleware(p.services.AuthlessMiddleware).
		Layout(layouts.LayoutDashboard)

	authGroup.Page("/login").Handler(p.LoginPage).Register()
	authGroup.Page("/register").Handler(p.RegisterPage).Register()

	return nil
}

func (p *PagesManager) registerDashboardPages() error {
	dashboardGroup := p.fuiApp.Group("").
		Middleware(p.services.AuthMiddleware).
		Layout(layouts.LayoutDashboard)

	dashboardGroup.Page("/").Handler(p.IndexPage).Register()
	// dashboardGroup := p.fuiApp.Group("/dashboard").Layout(layouts.LayoutDashboard)

	// dashboardGroup.Page("/dashboard").Handler(p.DashboardPage)
	// dashboardGroup.Page("/settings").Handler(p.SettingsPage)
	// dashboardGroup.Page("/profile").Handler(p.ProfilePage)
	// dashboardGroup.Page("/help").Handler(p.HelpPage)
	// dashboardGroup.Page("/logout").Handler(p.LogoutPage)

	// dashboardGroup.Page("/dashboard/settings").Handler(p.DashboardSettingsPage)
	// dashboardGroup.Page("/dashboard/profile").Handler(p.DashboardProfilePage)
	// dashboardGroup.Page("/dashboard/help").Handler(p.DashboardHelpPage)
	// dashboardGroup.Page("/dashboard/logout").Handler(p.DashboardLogoutPage)
	return nil
}
