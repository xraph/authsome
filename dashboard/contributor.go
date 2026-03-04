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
	manifest *contributor.Manifest
	engine   *authsome.Engine
	plugins  []plugin.Plugin
}

// New creates a new authsome dashboard contributor.
func New(manifest *contributor.Manifest, engine *authsome.Engine, plugins []plugin.Plugin) *Contributor {
	return &Contributor{
		manifest: manifest,
		engine:   engine,
		plugins:  plugins,
	}
}

// Manifest returns the contributor manifest.
func (c *Contributor) Manifest() *contributor.Manifest { return c.manifest }

// PrepareContext implements contributor.ContextPreparer. It parses app/env
// slugs from the route and enriches the context so layout components
// (app switcher, env switcher) can access them during rendering.
func (c *Contributor) PrepareContext(ctx context.Context, route string) context.Context {
	appSlug, envSlug, _ := parseAppEnvRoute(route)
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
		appSlug, envSlug, pageRoute = parseAppEnvRoute(route)
	} else {
		_, _, pageRoute = parseAppEnvRoute(route)
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

	appID, _ := AppIDFromContext(ctx)

	// Render the page component, then wrap it with the context script.
	comp, err := c.renderPageRoute(ctx, pageRoute, appID, params)
	if err != nil {
		return nil, err
	}

	// Wrap the page component with the context script for HTMX nav link rewriting.
	return withContextScript(comp, appSlug, envSlug), nil
}

// renderPageRoute dispatches to the correct page renderer based on the page route.
func (c *Contributor) renderPageRoute(ctx context.Context, pageRoute string, appID id.AppID, params contributor.Params) (templ.Component, error) {
	// Check plugin-contributed pages first (DashboardPageContributor for parameterized routes).
	for _, p := range c.plugins {
		if dpc, ok := p.(DashboardPageContributor); ok {
			if comp, err := dpc.DashboardRenderPage(ctx, pageRoute, params); err == nil && comp != nil {
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
	default:
		return nil, contributor.ErrPageNotFound
	}
}

// withContextScript wraps a page component with the auth context script for HTMX nav rewriting.
func withContextScript(page templ.Component, appSlug, envSlug string) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		if err := components.ContextScript(appSlug, envSlug).Render(ctx, w); err != nil {
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

	// Collect plugin-contributed sections for the overview.
	pluginSections := c.collectPluginSections(ctx)

	return templ.ComponentFunc(func(tCtx context.Context, w io.Writer) error {
		childCtx := templ.WithChildren(tCtx, components.PluginSections(pluginSections))
		return pages.OverviewPage(totalUsers, recent).Render(childCtx, w)
	}), nil
}

func (c *Contributor) renderUsers(ctx context.Context, appID id.AppID, params contributor.Params) (templ.Component, error) {
	cursor := params.QueryParams["cursor"]
	userList, err := fetchUsers(ctx, c.engine, appID, cursor, 25)
	if err != nil {
		return nil, fmt.Errorf("dashboard: render users: %w", err)
	}

	basePath := "/users"
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

	// Collect plugin-contributed sections for user detail.
	pluginSections := c.collectUserDetailSections(ctx, userID)

	return templ.ComponentFunc(func(tCtx context.Context, w io.Writer) error {
		childCtx := templ.WithChildren(tCtx, components.PluginSections(pluginSections))
		return pages.UserDetailPage(u).Render(childCtx, w)
	}), nil
}

func (c *Contributor) renderSessions(ctx context.Context, params contributor.Params) (templ.Component, error) {
	// ListSessions requires a user ID. If provided via query param, use it;
	// otherwise show an empty state prompting the admin to select a user.
	userIDStr := params.QueryParams["user_id"]
	if userIDStr == "" {
		return pages.SessionsPage(nil), nil
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
	// ListUserDevices requires a user ID. Same pattern as sessions.
	userIDStr := params.QueryParams["user_id"]
	if userIDStr == "" {
		return pages.DevicesPage(nil), nil
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

// ─── App/Env Route Parsing ───────────────────────────────────────────────────

// knownPageRoutes is the set of top-level page routes that the dashboard handles.
// Used to distinguish page routes from app slug segments in URL parsing.
var knownPageRoutes = map[string]bool{
	"/":                  true,
	"/users":             true,
	"/users/detail":      true,
	"/sessions":          true,
	"/devices":           true,
	"/roles":             true,
	"/webhooks":          true,
	"/environments":      true,
	"/environments/detail": true,
	"/signup-forms":      true,
	"/signup-forms/edit": true,
}

// parseAppEnvRoute extracts app slug, env slug, and page route from a route string.
// Routes with app/env: "/{appSlug}/{envSlug}/users" → ("platform", "development", "/users")
// Bare routes: "/users" → ("", "", "/users")
func parseAppEnvRoute(route string) (appSlug, envSlug, pageRoute string) {
	trimmed := strings.TrimPrefix(route, "/")
	if trimmed == "" {
		return "", "", "/"
	}

	parts := strings.SplitN(trimmed, "/", 3)

	// If the full route is a known page route, it's a bare route.
	if knownPageRoutes[route] {
		return "", "", route
	}

	// If first segment starts with a known page route prefix, it's bare.
	if len(parts) >= 1 && knownPageRoutes["/"+parts[0]] {
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
