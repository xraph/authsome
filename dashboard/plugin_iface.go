package dashboard

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/id"
)

// Context keys for embedding app/env IDs and slugs in context.
type appIDContextKey struct{}
type envIDContextKey struct{}
type appSlugContextKey struct{}
type envSlugContextKey struct{}

// WithAppID returns a context with the resolved app ID embedded.
// Dashboard plugins can extract this via AppIDFromContext to scope
// their data queries to the correct application.
func WithAppID(ctx context.Context, appID id.AppID) context.Context {
	return context.WithValue(ctx, appIDContextKey{}, appID)
}

// AppIDFromContext extracts the app ID from context, if present.
func AppIDFromContext(ctx context.Context) (id.AppID, bool) {
	v, ok := ctx.Value(appIDContextKey{}).(id.AppID)
	return v, ok
}

// WithEnvID returns a context with the resolved environment ID embedded.
func WithEnvID(ctx context.Context, envID id.EnvironmentID) context.Context {
	return context.WithValue(ctx, envIDContextKey{}, envID)
}

// EnvIDFromContext extracts the environment ID from context, if present.
func EnvIDFromContext(ctx context.Context) (id.EnvironmentID, bool) {
	v, ok := ctx.Value(envIDContextKey{}).(id.EnvironmentID)
	return v, ok
}

// WithAppSlug returns a context with the app slug embedded.
func WithAppSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, appSlugContextKey{}, slug)
}

// AppSlugFromContext extracts the app slug from context.
func AppSlugFromContext(ctx context.Context) string {
	v, _ := ctx.Value(appSlugContextKey{}).(string)
	return v
}

// WithEnvSlug returns a context with the environment slug embedded.
func WithEnvSlug(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, envSlugContextKey{}, slug)
}

// EnvSlugFromContext extracts the environment slug from context.
func EnvSlugFromContext(ctx context.Context) string {
	v, _ := ctx.Value(envSlugContextKey{}).(string)
	return v
}

// PluginWidget describes a widget contributed by an authsome plugin.
type PluginWidget struct {
	ID         string
	Title      string
	Size       string // "sm", "md", "lg"
	RefreshSec int
	Render     func(ctx context.Context) templ.Component
}

// PluginPage describes an extra page route contributed by a plugin.
type PluginPage struct {
	Route  string // e.g. "/mfa", "/social-providers"
	Label  string // nav label
	Icon   string // lucide icon name
	Render func(ctx context.Context) templ.Component
}

// DashboardPlugin is optionally implemented by authsome plugins
// to contribute UI sections to the authsome dashboard contributor.
// When plugins implement this interface, their pages, widgets, and
// settings panels are automatically merged into the dashboard.
type DashboardPlugin interface {
	// DashboardWidgets returns widgets this plugin contributes.
	DashboardWidgets(ctx context.Context) []PluginWidget
	// DashboardSettingsPanel returns a settings templ component, or nil.
	DashboardSettingsPanel(ctx context.Context) templ.Component
	// DashboardPages returns extra page routes this plugin handles.
	DashboardPages() []PluginPage
}

// UserDetailContributor is optionally implemented by plugins that want to
// contribute a section to the user detail page. The contributor passes the
// user ID so the plugin can fetch relevant data from its own store.
type UserDetailContributor interface {
	DashboardUserDetailSection(ctx context.Context, userID id.UserID) templ.Component
}

// OrgDetailContributor is optionally implemented by plugins that want to
// contribute a section to the organization detail page. The contributor passes
// the org ID so the plugin can fetch org-specific data from its own store.
type OrgDetailContributor interface {
	DashboardOrgDetailSection(ctx context.Context, orgID id.OrgID) templ.Component
}

// DashboardPageContributor is an enhanced interface for plugins that need
// access to route parameters when rendering dashboard pages. Unlike the
// basic DashboardPlugin.DashboardPages() which only supports parameterless
// rendering, this interface receives the full route params, enabling
// detail pages that parse IDs from query/path parameters.
type DashboardPageContributor interface {
	// DashboardNavItems returns navigation items this plugin contributes.
	DashboardNavItems() []contributor.NavItem
	// DashboardRenderPage renders a page for the given route with params.
	// Returns (nil, ErrPageNotFound) if the route is not handled by this plugin.
	DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error)
}
