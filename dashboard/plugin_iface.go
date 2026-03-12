package dashboard

import (
	"context"

	"github.com/a-h/templ"

	"github.com/xraph/forge/extensions/dashboard/contributor"

	"github.com/xraph/authsome/id"
)

// Context keys for embedding app/env IDs, slugs, and page route in context.
type appIDContextKey struct{}
type envIDContextKey struct{}
type appSlugContextKey struct{}
type envSlugContextKey struct{}
type pageRouteContextKey struct{}

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

// WithPageRoute returns a context with the current page route embedded.
func WithPageRoute(ctx context.Context, route string) context.Context {
	return context.WithValue(ctx, pageRouteContextKey{}, route)
}

// PageRouteFromContext extracts the current page route from context.
func PageRouteFromContext(ctx context.Context) string {
	v, _ := ctx.Value(pageRouteContextKey{}).(string)
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

// Plugin is optionally implemented by authsome plugins
// to contribute UI sections to the authsome dashboard contributor.
// When plugins implement this interface, their pages, widgets, and
// settings panels are automatically merged into the dashboard.
type Plugin interface {
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

// OrgDetailTab describes a tab contributed by a plugin to the organization detail page.
type OrgDetailTab struct {
	ID       string                                                    // unique tab identifier (e.g., "billing", "scim")
	Label    string                                                    // display label (e.g., "Billing", "SCIM")
	Icon     string                                                    // lucide icon name (e.g., "credit-card", "key")
	Priority int                                                       // ordering priority (lower = earlier)
	Render   func(ctx context.Context, orgID id.OrgID) templ.Component // renders tab content
}

// OrgDetailTabContributor is optionally implemented by plugins that want to
// contribute a full tab to the organization detail page. Unlike OrgDetailContributor
// which adds a section to the Overview tab, this interface contributes a named,
// navigable tab with its own content panel.
type OrgDetailTabContributor interface {
	DashboardOrgDetailTabs(ctx context.Context, orgID id.OrgID) []OrgDetailTab
}

// OrgCreateFormContributor is optionally implemented by plugins that want to
// contribute additional form fields to the organization creation form.
type OrgCreateFormContributor interface {
	DashboardOrgCreateFormFields(ctx context.Context) templ.Component
}

// PageContributor is an enhanced interface for plugins that need
// access to route parameters when rendering dashboard pages. Unlike the
// basic Plugin.DashboardPages() which only supports parameterless
// rendering, this interface receives the full route params, enabling
// detail pages that parse IDs from query/path parameters.
type PageContributor interface {
	// DashboardNavItems returns navigation items this plugin contributes.
	DashboardNavItems() []contributor.NavItem
	// DashboardRenderPage renders a page for the given route with params.
	// Returns (nil, ErrPageNotFound) if the route is not handled by this plugin.
	DashboardRenderPage(ctx context.Context, route string, params contributor.Params) (templ.Component, error)
}
