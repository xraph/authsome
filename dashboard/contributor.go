package dashboard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/app"
	"github.com/xraph/authsome/dashboard/auth"
	"github.com/xraph/authsome/dashboard/components"
	"github.com/xraph/authsome/dashboard/pages"
	"github.com/xraph/authsome/dashboard/settings"
	"github.com/xraph/authsome/dashboard/widgets"
	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/rbac"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store"
	"github.com/xraph/authsome/user"
)

// Ensure Contributor implements the required interfaces at compile time.
var (
	_ contributor.LocalContributor    = (*Contributor)(nil)
	_ contributor.AuthPageContributor = (*Contributor)(nil)
	_ contributor.ContextPreparer     = (*Contributor)(nil)
)

// Contributor implements the dashboard LocalContributor and AuthPageContributor
// interfaces for the authsome extension. It renders pages, widgets, and settings
// using templ components and ForgeUI, and supports plugin-contributed UI sections.
type Contributor struct {
	manifest   *contributor.Manifest
	engine     *authsome.Engine
	plugins    []plugin.Plugin
	pageRoutes map[string]bool // core + plugin page routes for URL parsing
}

// New creates a new authsome dashboard contributor.
func New(manifest *contributor.Manifest, engine *authsome.Engine, plugins []plugin.Plugin) *Contributor {
	c := &Contributor{
		manifest: manifest,
		engine:   engine,
		plugins:  plugins,
	}
	c.pageRoutes = c.buildPageRoutes()
	return c
}

// buildPageRoutes merges core knownPageRoutes with routes contributed by
// plugins (via DashboardPageContributor and DashboardPlugin). This ensures
// parseAppEnvRoute correctly identifies plugin routes as page routes rather
// than misinterpreting them as app/env slugs.
func (c *Contributor) buildPageRoutes() map[string]bool {
	routes := make(map[string]bool, len(knownPageRoutes))
	for k, v := range knownPageRoutes {
		routes[k] = v
	}

	for _, p := range c.plugins {
		// DashboardPageContributor plugins declare nav items with paths.
		if dpc, ok := p.(DashboardPageContributor); ok {
			for _, nav := range dpc.DashboardNavItems() {
				routes[nav.Path] = true
			}
		}

		// DashboardPlugin plugins declare pages with routes.
		if dp, ok := p.(DashboardPlugin); ok {
			for _, pp := range dp.DashboardPages() {
				routes[pp.Route] = true
			}
		}
	}

	return routes
}

// Manifest returns the contributor manifest.
func (c *Contributor) Manifest() *contributor.Manifest { return c.manifest }

// PrepareContext implements contributor.ContextPreparer. It parses app/env
// slugs from the route and enriches the context so layout components
// (app switcher, env switcher) can access them during rendering.
func (c *Contributor) PrepareContext(ctx context.Context, route string) context.Context {
	appSlug, envSlug, pageRoute := c.parseAppEnvRoute(route)
	if appSlug == "" || envSlug == "" {
		// Fall back to default app/env so layout components (switchers) render.
		defaultApp, defaultEnv := c.resolveDefaults(ctx)
		appSlug = defaultApp.Slug
		envSlug = defaultEnv.Slug
	}

	a, env, err := c.resolveAppEnv(ctx, appSlug, envSlug)
	if err != nil {
		return ctx
	}

	ctx = WithAppID(ctx, a.ID)
	ctx = WithEnvID(ctx, env.ID)
	ctx = WithAppSlug(ctx, appSlug)
	ctx = WithEnvSlug(ctx, envSlug)
	ctx = WithPageRoute(ctx, pageRoute)
	return ctx
}

// RenderPage renders a page for the given route.
// Routes are expected in the format /{appSlug}/{envSlug}/... (e.g. /platform/development/users).
// If the route has no app/env prefix, it redirects to the default app/env via HTMX.
func (c *Contributor) RenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error) {
	// Read from context (set by PrepareContext) or fall back to route parsing.
	appSlug := AppSlugFromContext(ctx)
	envSlug := EnvSlugFromContext(ctx)

	pageRoute := route
	if appSlug == "" || envSlug == "" {
		appSlug, envSlug, pageRoute = c.parseAppEnvRoute(route)
	} else {
		_, _, pageRoute = c.parseAppEnvRoute(route)
	}

	// If no app/env, redirect to default app/env URL.
	if appSlug == "" || envSlug == "" {
		defaultApp, defaultEnv := c.resolveDefaults(ctx)
		redirectURL := fmt.Sprintf("%s/ext/authsome/pages/%s/%s%s",
			params.BasePath, defaultApp.Slug, defaultEnv.Slug, route)
		return htmxRedirect(redirectURL), nil
	}

	// If context wasn't prepared (e.g. direct call), resolve and enrich now.
	if _, ok := AppIDFromContext(ctx); !ok {
		a, env, err := c.resolveAppEnv(ctx, appSlug, envSlug)
		if err != nil {
			return nil, contributor.ErrPageNotFound
		}
		ctx = WithAppID(ctx, a.ID)
		ctx = WithEnvID(ctx, env.ID)
		ctx = WithAppSlug(ctx, appSlug)
		ctx = WithEnvSlug(ctx, envSlug)
	}
	ctx = WithPageRoute(ctx, pageRoute)

	appID, _ := AppIDFromContext(ctx)

	// Render the page component, then wrap it with the context script.
	comp, err := c.renderPageRoute(ctx, pageRoute, appID, params)
	if err != nil {
		return nil, err
	}

	// Wrap the page component with the context script for HTMX nav link rewriting.
	return withContextScript(comp, appSlug, envSlug, c.knownRoutesCSV()), nil
}

// renderPageRoute dispatches to the correct page renderer based on the page route.
func (c *Contributor) renderPageRoute(ctx context.Context, pageRoute string, appID id.AppID, params contributor.Params) (templ.Component, error) {
	// Check plugin-contributed pages first (DashboardPageContributor for parameterized routes).
	// Plugins return (nil, ErrPageNotFound) for routes they don't handle; real errors
	// are propagated so the dashboard can surface them instead of silently swallowing.
	for _, p := range c.plugins {
		if dpc, ok := p.(DashboardPageContributor); ok {
			comp, err := dpc.DashboardRenderPage(ctx, pageRoute, params)
			if err != nil && !errors.Is(err, contributor.ErrPageNotFound) {
				return nil, err // propagate real errors
			}
			if comp != nil {
				return comp, nil
			}
		}
	}

	// Check plugin-contributed pages (DashboardPlugin for simple routes).
	for _, dp := range c.dashboardPlugins() {
		for _, pp := range dp.DashboardPages() {
			if pp.Route == pageRoute {
				return pp.Render(ctx), nil
			}
		}
	}

	switch pageRoute {
	case "/", "":
		return c.renderOverview(ctx, appID)
	case "/users":
		return c.renderUsers(ctx, appID, params)
	case "/users/detail":
		return c.renderUserDetail(ctx, params)
	case "/sessions":
		return c.renderSessions(ctx, params)
	case "/devices":
		return c.renderDevices(ctx, params)
	case "/roles":
		return c.renderRoles(ctx, appID)
	case "/webhooks":
		return c.renderWebhooks(ctx, appID)
	case "/environments":
		return c.renderEnvironments(ctx, appID)
	case "/environments/detail":
		return c.renderEnvironmentDetail(ctx, appID, params)
	case "/signup-forms":
		return c.renderSignupForms(ctx, appID)
	case "/signup-forms/edit":
		return c.renderSignupFormEditor(ctx, appID)
	case "/credentials":
		return c.renderCredentials(ctx, appID, params)
	case "/plugins":
		return c.renderPlugins(ctx)
	case "/settings":
		return c.renderSettingsPage(ctx)
	case "/settings/editor":
		return c.renderSettingsEditor(ctx, appID, params)
	case "/sessions/detail":
		return c.renderSessionDetail(ctx, params)
	case "/devices/detail":
		return c.renderDeviceDetail(ctx, params)
	case "/roles/detail":
		return c.renderRoleDetail(ctx, appID, params)
	case "/apps":
		return c.renderApps(ctx)
	case "/apps/create":
		return c.renderAppCreate(ctx, params)
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// withContextScript wraps a page component with the auth context script for HTMX nav rewriting.
func withContextScript(page templ.Component, appSlug, envSlug, knownRoutesCSV string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := components.ContextScript(appSlug, envSlug, knownRoutesCSV).Render(ctx, w); err != nil {
			return err
		}
		return page.Render(ctx, w)
	})
}

// RenderWidget renders a widget by ID.
func (c *Contributor) RenderWidget(ctx context.Context, widgetID string) (templ.Component, error) {
	appID := c.defaultAppID()

	// Inject app ID into context so plugin widgets can scope their queries.
	ctx = WithAppID(ctx, appID)

	// Check plugin-contributed widgets first.
	for _, dp := range c.dashboardPlugins() {
		for _, w := range dp.DashboardWidgets(ctx) {
			if w.ID == widgetID {
				return w.Render(ctx), nil
			}
		}
	}

	switch widgetID {
	case "authsome-stats":
		return c.renderStatsWidget(ctx, appID)
	case "authsome-recent-signups":
		return c.renderRecentSignupsWidget(ctx, appID)
	case "authsome-activity":
		return c.renderActivityWidget()
	default:
		return nil, contributor.ErrWidgetNotFound
	}
}

// RenderSettings renders a settings panel by ID.
func (c *Contributor) RenderSettings(ctx context.Context, settingID string) (templ.Component, error) {
	// Inject app ID into context so plugin settings can scope their queries.
	ctx = WithAppID(ctx, c.defaultAppID())

	// Check plugin-contributed settings panels.
	pluginSettings := c.collectPluginSettings(ctx)

	switch settingID {
	case "authsome-config":
		return c.renderSettings(ctx, pluginSettings)
	default:
		return nil, contributor.ErrSettingNotFound
	}
}

// ─── Auth Page Contributor ───────────────────────────────────────────────────

// RenderAuthPage renders an authentication page by type.
func (c *Contributor) RenderAuthPage(_ context.Context, pageType string, _ contributor.Params) (templ.Component, error) {
	switch pageType {
	case "login":
		return auth.LoginPage(auth.LoginPageLinks{RegisterPath: "./register"}), nil
	case "register":
		return auth.RegisterPage(auth.RegisterPageLinks{LoginPath: "./login"}), nil
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// HandleAuthAction handles form submissions for authentication pages.
// On success it returns a redirect URL; on failure it returns a re-rendered component.
func (c *Contributor) HandleAuthAction(ctx context.Context, pageType string, params contributor.Params) (string, templ.Component, error) {
	switch pageType {
	case "login":
		return c.handleLogin(ctx, params)
	case "register":
		return c.handleRegister(ctx, params)
	default:
		return "", nil, contributor.ErrPageNotFound
	}
}

// ─── Private Render Helpers ──────────────────────────────────────────────────

func (c *Contributor) renderOverview(ctx context.Context, appID id.AppID) (templ.Component, error) {
	totalUsers, err := fetchStats(ctx, c.engine, appID)
	if err != nil {
		totalUsers = 0
	}

	recentUsers, err := fetchUsers(ctx, c.engine, appID, "", 5)
	if err != nil {
		recentUsers = nil
	}

	var recent []*user.User
	if recentUsers != nil {
		recent = recentUsers.Users
	}

	stats := pages.OverviewStats{
		TotalUsers:    totalUsers,
		ActivePlugins: len(c.plugins),
	}

	// Collect plugin-contributed sections for the overview.
	pluginSections := c.collectPluginSections(ctx)

	return templ.ComponentFunc(func(tCtx context.Context, w io.Writer) error {
		childCtx := templ.WithChildren(tCtx, components.PluginSections(pluginSections))
		return pages.OverviewPage(stats, recent).Render(childCtx, w)
	}), nil
}

func (c *Contributor) renderUsers(ctx context.Context, appID id.AppID, params contributor.Params) (templ.Component, error) {
	cursor := params.QueryParams["cursor"]
	userList, err := fetchUsers(ctx, c.engine, appID, cursor, 25)
	if err != nil {
		return nil, fmt.Errorf("dashboard: render users: %w", err)
	}

	basePath := "./users"
	return pages.UsersPage(userList, cursor, basePath), nil
}

func (c *Contributor) renderUserDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	userIDStr := params.PathParams["id"]
	if userIDStr == "" {
		userIDStr = params.QueryParams["id"]
	}
	if userIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}

	// Validate the ID format before resolving.
	userID, err := id.ParseUserID(userIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}

	u, err := c.engine.ResolveUser(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("dashboard: resolve user: %w", err)
	}

	data := pages.UserDetailPageData{
		User: u,
	}

	// Fetch sessions for this user.
	if sessions, err := c.engine.ListSessions(ctx, userID); err == nil {
		data.Sessions = sessions
	}

	// Fetch devices for this user.
	if devices, err := c.engine.ListUserDevices(ctx, userID); err == nil {
		data.Devices = devices
	}

	// Fetch role slugs for this user.
	if roleSlugs, err := c.engine.ListUserRoleSlugs(ctx, userID); err == nil {
		data.Roles = roleSlugs
	}

	// Collect plugin-contributed sections for user detail.
	pluginSections := c.collectUserDetailSections(ctx, userID)

	return templ.ComponentFunc(func(tCtx context.Context, w io.Writer) error {
		childCtx := templ.WithChildren(tCtx, components.PluginSections(pluginSections))
		return pages.UserDetailPage(data).Render(childCtx, w)
	}), nil
}

func (c *Contributor) renderSessions(ctx context.Context, params contributor.Params) (templ.Component, error) {
	userIDStr := params.QueryParams["user_id"]

	// When no user_id is provided, list all recent sessions.
	if userIDStr == "" {
		sessions, err := c.engine.ListAllSessions(ctx, 100)
		if err != nil {
			sessions = nil
		}
		return pages.SessionsPage(sessions), nil
	}

	userID, err := id.ParseUserID(userIDStr)
	if err != nil {
		return pages.SessionsPage(nil), nil
	}

	sessions, err := c.engine.ListSessions(ctx, userID)
	if err != nil {
		return pages.SessionsPage(nil), nil
	}

	return pages.SessionsPage(sessions), nil
}

func (c *Contributor) renderDevices(ctx context.Context, params contributor.Params) (templ.Component, error) {
	userIDStr := params.QueryParams["user_id"]

	// When no user_id is provided, list all recent devices.
	if userIDStr == "" {
		devices, err := c.engine.ListAllDevices(ctx, 100)
		if err != nil {
			devices = nil
		}
		return pages.DevicesPage(devices), nil
	}

	userID, err := id.ParseUserID(userIDStr)
	if err != nil {
		return pages.DevicesPage(nil), nil
	}

	devices, err := c.engine.ListUserDevices(ctx, userID)
	if err != nil {
		return pages.DevicesPage(nil), nil
	}

	return pages.DevicesPage(devices), nil
}

func (c *Contributor) renderRoles(ctx context.Context, appID id.AppID) (templ.Component, error) {
	roles, err := fetchRoles(ctx, c.engine, appID)
	if err != nil {
		roles = nil
	}

	return pages.RolesPage(roles), nil
}

func (c *Contributor) renderWebhooks(ctx context.Context, appID id.AppID) (templ.Component, error) {
	hooks, err := fetchWebhooks(ctx, c.engine, appID)
	if err != nil {
		hooks = nil
	}

	return pages.WebhooksPage(hooks), nil
}

func (c *Contributor) renderEnvironments(ctx context.Context, appID id.AppID) (templ.Component, error) {
	envs, err := fetchEnvironments(ctx, c.engine, appID)
	if err != nil {
		envs = nil
	}

	return pages.EnvironmentsPage(envs), nil
}

func (c *Contributor) renderEnvironmentDetail(ctx context.Context, appID id.AppID, params contributor.Params) (templ.Component, error) {
	envIDStr := params.PathParams["id"]
	if envIDStr == "" {
		envIDStr = params.QueryParams["id"]
	}
	if envIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}

	envID, err := id.ParseEnvironmentID(envIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}

	env, err := c.engine.GetEnvironment(ctx, envID)
	if err != nil {
		return nil, fmt.Errorf("dashboard: resolve environment: %w", err)
	}

	// Resolve effective settings: type defaults + per-environment overrides.
	effective := environment.MergeSettings(environment.DefaultSettingsForType(env.Type), env.Settings)
	displayEnv := *env
	displayEnv.Settings = effective

	return templ.ComponentFunc(func(tCtx context.Context, w io.Writer) error {
		childCtx := templ.WithChildren(tCtx, components.PluginSections(nil))
		return pages.EnvironmentDetailPage(&displayEnv).Render(childCtx, w)
	}), nil
}

func (c *Contributor) renderSignupForms(ctx context.Context, appID id.AppID) (templ.Component, error) {
	configs, err := c.engine.ListFormConfigs(ctx, appID)
	if err != nil {
		configs = nil
	}

	return pages.SignupFormsPage(configs), nil
}

func (c *Contributor) renderSignupFormEditor(ctx context.Context, appID id.AppID) (templ.Component, error) {
	// Load existing config if available.
	fc, err := c.engine.GetSignupFormConfig(ctx, appID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, fmt.Errorf("dashboard: render signup form editor: %w", err)
	}

	if fc == nil {
		fc = &formconfig.FormConfig{
			AppID:    appID,
			FormType: formconfig.FormTypeSignup,
			Active:   true,
		}
	}

	return pages.SignupFormEditorPage(fc), nil
}

func (c *Contributor) renderCredentials(ctx context.Context, appID id.AppID, params contributor.Params) (templ.Component, error) {
	a, err := c.engine.GetApp(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("dashboard: render credentials: %w", err)
	}

	envID, _ := EnvIDFromContext(ctx)
	var env *environment.Environment
	if !envID.IsNil() {
		env, _ = c.engine.GetEnvironment(ctx, envID)
	}

	data := pages.CredentialsPageData{
		App:         a,
		Environment: env,
		BasePath:    c.engine.Config().BasePath,
	}

	return pages.CredentialsPage(data), nil
}

func (c *Contributor) renderPlugins(_ context.Context) (templ.Component, error) {
	var infos []pages.PluginInfo
	for _, p := range c.plugins {
		info := pages.PluginInfo{
			Name: p.Name(),
		}

		if dp, ok := p.(DashboardPlugin); ok {
			for _, pp := range dp.DashboardPages() {
				info.Pages = append(info.Pages, pages.PluginPageInfo{
					Route: pp.Route,
					Label: pp.Label,
					Icon:  pp.Icon,
				})
			}
			for _, w := range dp.DashboardWidgets(context.Background()) {
				info.Widgets = append(info.Widgets, pages.PluginWidgetInfo{
					Title: w.Title,
					Size:  w.Size,
				})
			}
			if dp.DashboardSettingsPanel(context.Background()) != nil {
				info.HasSettingsPanel = true
			}
		}

		if _, ok := p.(DashboardPageContributor); ok {
			info.HasPageContributor = true
		}

		if _, ok := p.(UserDetailContributor); ok {
			info.HasUserDetailSection = true
		}

		infos = append(infos, info)
	}

	return pages.PluginsPage(infos), nil
}

func (c *Contributor) renderSettingsPage(ctx context.Context) (templ.Component, error) {
	cfg := c.engine.Config()

	pluginNames := make([]string, 0, len(c.plugins))
	for _, p := range c.plugins {
		pluginNames = append(pluginNames, p.Name())
	}

	data := pages.SettingsPageData{
		Config:      cfg,
		PluginNames: pluginNames,
	}

	pluginSettings := c.collectPluginSettings(ctx)

	return templ.ComponentFunc(func(tCtx context.Context, w io.Writer) error {
		childCtx := templ.WithChildren(tCtx, components.PluginSections(pluginSettings))
		return pages.SettingsPage(data).Render(childCtx, w)
	}), nil
}

func (c *Contributor) renderSettingsEditor(_ context.Context, appID id.AppID, params contributor.Params) (templ.Component, error) {
	mgr := c.engine.Settings()
	if mgr == nil {
		return pages.SettingsEditorEmpty(), nil
	}

	// Allow scope override via query param (global, app, org).
	scope := params.QueryParams["scope"]
	if scope == "" {
		scope = "global"
		if !appID.IsNil() {
			scope = "app"
		}
	}

	appIDStr := ""
	if !appID.IsNil() {
		appIDStr = appID.String()
	}

	orgIDStr := params.QueryParams["org_id"]

	data := pages.BuildSettingsEditorData(context.Background(), mgr, scope, appIDStr, orgIDStr, c.engine.Config().BasePath)

	return pages.SettingsEditorPage(data), nil
}

func (c *Contributor) renderSessionDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	sessionIDStr := params.PathParams["id"]
	if sessionIDStr == "" {
		sessionIDStr = params.QueryParams["id"]
	}
	if sessionIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}

	sessionID, err := id.ParseSessionID(sessionIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}

	sess, err := c.engine.Store().GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("dashboard: resolve session: %w", err)
	}

	data := pages.SessionDetailData{
		Session: sess,
	}

	// Resolve the user associated with this session.
	if !sess.UserID.IsNil() {
		if u, err := c.engine.ResolveUser(sess.UserID.String()); err == nil {
			data.User = u
		}
	}

	// Resolve the device if bound.
	if !sess.DeviceID.IsNil() {
		if d, err := c.engine.GetDevice(ctx, sess.DeviceID); err == nil {
			data.Device = d
		}
	}

	return pages.SessionDetailPage(data), nil
}

func (c *Contributor) renderDeviceDetail(ctx context.Context, params contributor.Params) (templ.Component, error) {
	deviceIDStr := params.PathParams["id"]
	if deviceIDStr == "" {
		deviceIDStr = params.QueryParams["id"]
	}
	if deviceIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}

	deviceID, err := id.ParseDeviceID(deviceIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}

	d, err := c.engine.GetDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("dashboard: resolve device: %w", err)
	}

	data := pages.DeviceDetailData{
		Device: d,
	}

	// Resolve the user who owns this device.
	if !d.UserID.IsNil() {
		if u, err := c.engine.ResolveUser(d.UserID.String()); err == nil {
			data.User = u
		}
		// List sessions for this device's user.
		if sessions, err := c.engine.ListSessions(ctx, d.UserID); err == nil {
			// Filter to sessions using this device.
			var deviceSessions []*session.Session
			for _, s := range sessions {
				if s.DeviceID == deviceID {
					deviceSessions = append(deviceSessions, s)
				}
			}
			data.Sessions = deviceSessions
		}
	}

	return pages.DeviceDetailPage(data), nil
}

func (c *Contributor) renderRoleDetail(ctx context.Context, _ id.AppID, params contributor.Params) (templ.Component, error) {
	roleIDStr := params.PathParams["id"]
	if roleIDStr == "" {
		roleIDStr = params.QueryParams["id"]
	}
	if roleIDStr == "" {
		return nil, contributor.ErrPageNotFound
	}

	roleID, err := id.ParseRoleID(roleIDStr)
	if err != nil {
		return nil, contributor.ErrPageNotFound
	}

	role, err := c.engine.GetRole(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("dashboard: resolve role: %w", err)
	}

	var permissions []*rbac.Permission
	if perms, err := c.engine.ListRolePermissions(ctx, roleID); err == nil {
		permissions = perms
	}

	data := pages.RoleDetailData{
		Role:        role,
		Permissions: permissions,
	}

	return pages.RoleDetailPage(data), nil
}

func (c *Contributor) renderApps(ctx context.Context) (templ.Component, error) {
	apps, err := c.engine.ListApps(ctx)
	if err != nil {
		apps = nil
	}

	return pages.AppsPage(apps), nil
}

func (c *Contributor) renderAppCreate(ctx context.Context, params contributor.Params) (templ.Component, error) {
	// App creation from the sidebar dialog is handled by the bridge API
	// (authsome.createApp). This full-page route is kept as a fallback.
	action := params.FormData["action"]
	if action == "create" {
		nonce := params.FormData["nonce"]
		if ConsumeNonce(nonce) {
			created, errMsg := c.handleCreateApp(ctx, params)
			var data pages.CreateAppPageData
			data.CreatedApp = created
			data.Error = errMsg
			data.FormNonce = GenerateNonce()
			return pages.CreateAppPage(data), nil
		}
	}

	var data pages.CreateAppPageData
	data.FormNonce = GenerateNonce()
	return pages.CreateAppPage(data), nil
}

func (c *Contributor) handleCreateApp(ctx context.Context, params contributor.Params) (*app.App, string) {
	name := strings.TrimSpace(params.FormData["name"])
	slug := strings.TrimSpace(params.FormData["slug"])
	logo := strings.TrimSpace(params.FormData["logo"])

	if name == "" || slug == "" {
		return nil, "Name and slug are required."
	}

	existing, err := c.engine.GetAppBySlug(ctx, slug)
	if err == nil && existing != nil {
		return nil, fmt.Sprintf("Slug %q is already in use.", slug)
	}

	a := &app.App{
		Name: name,
		Slug: slug,
		Logo: logo,
	}

	if err := c.engine.CreateApp(ctx, a); err != nil {
		return nil, fmt.Sprintf("Failed to create app: %v", err)
	}

	return a, ""
}

// ─── Widget Render Helpers ───────────────────────────────────────────────────

func (c *Contributor) renderStatsWidget(ctx context.Context, appID id.AppID) (templ.Component, error) {
	totalUsers, err := fetchStats(ctx, c.engine, appID)
	if err != nil {
		totalUsers = 0
	}

	return widgets.StatsWidget(totalUsers), nil
}

func (c *Contributor) renderRecentSignupsWidget(ctx context.Context, appID id.AppID) (templ.Component, error) {
	userList, err := fetchUsers(ctx, c.engine, appID, "", 5)
	if err != nil || userList == nil {
		return widgets.RecentSignupsWidget(nil), nil
	}

	return widgets.RecentSignupsWidget(userList.Users), nil
}

func (c *Contributor) renderActivityWidget() (templ.Component, error) {
	return widgets.AuthActivityWidget(), nil
}

// ─── Settings Render Helper ──────────────────────────────────────────────────

func (c *Contributor) renderSettings(_ context.Context, pluginSettings []templ.Component) (templ.Component, error) {
	cfg := c.engine.Config()

	pluginNames := make([]string, 0, len(c.plugins))
	for _, p := range c.plugins {
		pluginNames = append(pluginNames, p.Name())
	}

	return templ.ComponentFunc(func(tCtx context.Context, w io.Writer) error {
		childCtx := templ.WithChildren(tCtx, components.PluginSections(pluginSettings))
		return settings.ConfigPanel(cfg, pluginNames).Render(childCtx, w)
	}), nil
}

// ─── Auth Action Handlers ────────────────────────────────────────────────────

func (c *Contributor) handleLogin(ctx context.Context, params contributor.Params) (string, templ.Component, error) {
	email := params.FormData["email"]
	password := params.FormData["password"]

	if email == "" || password == "" {
		return "", auth.LoginError("Email and password are required", auth.LoginPageLinks{RegisterPath: "./register"}), nil
	}

	_, _, err := c.engine.SignIn(ctx, &account.SignInRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return "", auth.LoginError("Invalid email or password", auth.LoginPageLinks{RegisterPath: "./register"}), nil
	}

	// Redirect to dashboard root on success.
	return "/", nil, nil
}

func (c *Contributor) handleRegister(ctx context.Context, params contributor.Params) (string, templ.Component, error) {
	firstName := params.FormData["first_name"]
	lastName := params.FormData["last_name"]
	email := params.FormData["email"]
	password := params.FormData["password"]

	// Collect meta_* fields into metadata map.
	metadata := make(map[string]string)
	for key, val := range params.FormData {
		if len(key) > 5 && key[:5] == "meta_" && val != "" {
			metadata[key[5:]] = val
		}
	}

	if email == "" || password == "" {
		return "", auth.RegisterError("Email and password are required", auth.RegisterPageLinks{LoginPath: "./login"}), nil
	}

	req := &account.SignUpRequest{
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
	}
	if len(metadata) > 0 {
		req.Metadata = metadata
	}

	_, _, err := c.engine.SignUp(ctx, req)
	if err != nil {
		return "", auth.RegisterError(err.Error(), auth.RegisterPageLinks{LoginPath: "./login"}), nil
	}

	// Redirect to dashboard root on success.
	return "/", nil, nil
}

// ─── Plugin Helpers ──────────────────────────────────────────────────────────

// dashboardPlugins returns all registered plugins that implement DashboardPlugin.
func (c *Contributor) dashboardPlugins() []DashboardPlugin {
	var dps []DashboardPlugin
	for _, p := range c.plugins {
		if dp, ok := p.(DashboardPlugin); ok {
			dps = append(dps, dp)
		}
	}
	return dps
}

// collectPluginSections gathers rendered templ components from all dashboard plugins.
func (c *Contributor) collectPluginSections(ctx context.Context) []templ.Component {
	var sections []templ.Component
	for _, dp := range c.dashboardPlugins() {
		for _, w := range dp.DashboardWidgets(ctx) {
			sections = append(sections, w.Render(ctx))
		}
	}
	return sections
}

// collectPluginSettings gathers settings panels from all dashboard plugins.
func (c *Contributor) collectPluginSettings(ctx context.Context) []templ.Component {
	var panels []templ.Component
	for _, dp := range c.dashboardPlugins() {
		if panel := dp.DashboardSettingsPanel(ctx); panel != nil {
			panels = append(panels, panel)
		}
	}
	return panels
}

// collectUserDetailSections gathers user detail sections from plugins implementing UserDetailContributor.
func (c *Contributor) collectUserDetailSections(ctx context.Context, userID id.UserID) []templ.Component {
	var sections []templ.Component
	for _, p := range c.plugins {
		if udc, ok := p.(UserDetailContributor); ok {
			if section := udc.DashboardUserDetailSection(ctx, userID); section != nil {
				sections = append(sections, section)
			}
		}
	}
	return sections
}

// knownRoutesCSV returns a comma-separated list of unique top-level page route
// prefixes (e.g. "/users,/sessions,/social-providers") for use by the client-side
// HTMX interceptor which rewrites sidebar nav links with app/env slug prefixes.
func (c *Contributor) knownRoutesCSV() string {
	seen := make(map[string]bool)
	var routes []string
	for route := range c.pageRoutes {
		// Extract the top-level segment (e.g. "/users" from "/users/detail").
		seg := route
		if len(seg) > 1 {
			if i := strings.Index(seg[1:], "/"); i >= 0 {
				seg = seg[:i+1]
			}
		}
		if seg == "/" || seg == "" || seen[seg] {
			continue
		}
		seen[seg] = true
		routes = append(routes, seg)
	}
	return strings.Join(routes, ",")
}

// ─── App/Env Route Parsing ───────────────────────────────────────────────────

// knownPageRoutes is the set of top-level page routes that the dashboard handles.
// Used to distinguish page routes from app slug segments in URL parsing.
var knownPageRoutes = map[string]bool{
	"/":                    true,
	"/users":               true,
	"/users/detail":        true,
	"/sessions":            true,
	"/sessions/detail":     true,
	"/devices":             true,
	"/devices/detail":      true,
	"/roles":               true,
	"/roles/detail":        true,
	"/webhooks":            true,
	"/environments":        true,
	"/environments/detail": true,
	"/signup-forms":        true,
	"/signup-forms/edit":   true,
	"/credentials":         true,
	"/plugins":             true,
	"/settings":            true,
	"/settings/editor":     true,
	"/apps":                true,
	"/apps/create":         true,
}

// parseAppEnvRoute extracts app slug, env slug, and page route from a route string.
// Routes with app/env: "/{appSlug}/{envSlug}/users" → ("platform", "development", "/users")
// Bare routes: "/users" → ("", "", "/users")
//
// Uses the contributor's pageRoutes map (core + plugin routes) to distinguish
// page routes from app/env slug segments. Without this, plugin routes like
// "/social-providers/detail" would be misinterpreted as appSlug="social-providers",
// envSlug="detail".
func (c *Contributor) parseAppEnvRoute(route string) (appSlug, envSlug, pageRoute string) {
	trimmed := strings.TrimPrefix(route, "/")
	if trimmed == "" {
		return "", "", "/"
	}

	parts := strings.SplitN(trimmed, "/", 3)

	// If the full route is a known page route, it's a bare route.
	if c.pageRoutes[route] {
		return "", "", route
	}

	// If first segment matches a known page route prefix, it's a bare route.
	// This catches sub-routes like "/social-providers/detail" when
	// "/social-providers" is registered.
	if len(parts) >= 1 && c.pageRoutes["/"+parts[0]] {
		return "", "", route
	}

	// Need at least 2 segments for app/env.
	if len(parts) < 2 {
		return "", "", route
	}

	appSlug = parts[0]
	envSlug = parts[1]
	if len(parts) == 3 {
		pageRoute = "/" + parts[2]
	} else {
		pageRoute = "/"
	}
	return
}

// resolveDefaults returns the default app and environment for redirect.
func (c *Contributor) resolveDefaults(ctx context.Context) (*app.App, *environment.Environment) {
	appID := c.defaultAppID()
	a, err := c.engine.GetApp(ctx, appID)
	if err != nil {
		// Fallback: return a minimal app with slug from config.
		a = &app.App{ID: appID, Slug: "platform"}
	}
	env, err := c.engine.GetDefaultEnvironment(ctx, appID)
	if err != nil {
		env = &environment.Environment{Slug: "development"}
	}
	return a, env
}

// resolveAppEnv resolves an app and environment from their slugs.
func (c *Contributor) resolveAppEnv(ctx context.Context, appSlug, envSlug string) (*app.App, *environment.Environment, error) {
	a, err := c.engine.GetAppBySlug(ctx, appSlug)
	if err != nil {
		return nil, nil, err
	}
	env, err := c.engine.GetEnvironmentBySlug(ctx, a.ID, envSlug)
	if err != nil {
		return nil, nil, err
	}
	return a, env, nil
}

// htmxRedirect returns a templ component that triggers an HTMX client-side redirect.
func htmxRedirect(url string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, fmt.Sprintf(
			`<div hx-get="%s" hx-trigger="load" hx-target="#content" hx-swap="innerHTML" hx-push-url="true"></div>`,
			url))
		return err
	})
}

// defaultAppID returns the app ID from the engine config.
func (c *Contributor) defaultAppID() id.AppID {
	appID, _ := id.ParseAppID(c.engine.Config().AppID)
	return appID
}
