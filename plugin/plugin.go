// Package plugin defines the plugin system for AuthSome v0.5.0.
// Plugins implement the base Plugin interface and optionally implement
// any combination of lifecycle and event hook interfaces.
//
// The registry type-caches plugins at registration time so emit calls
// iterate only over plugins implementing the relevant hook.
package plugin

import (
	"context"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/strategy"
	"github.com/xraph/authsome/user"

	"github.com/xraph/grove/migrate"
)

// Plugin is the base interface that all plugins must implement.
type Plugin interface {
	Name() string
}

// ──────────────────────────────────────────────────
// Lifecycle hooks
// ──────────────────────────────────────────────────

// OnInit is called during engine initialization. The engine parameter
// is intentionally typed as any to avoid import cycles; use the engine
// accessor methods to interact with it.
type OnInit interface {
	OnInit(ctx context.Context, engine any) error
}

// OnShutdown is called during engine shutdown.
type OnShutdown interface {
	OnShutdown(ctx context.Context) error
}

// ──────────────────────────────────────────────────
// Auth event hooks (signup / signin / signout)
// ──────────────────────────────────────────────────

// BeforeSignUp is called before a new account is created.
type BeforeSignUp interface {
	OnBeforeSignUp(ctx context.Context, req *account.SignUpRequest) error
}

// AfterSignUp is called after a new account is created.
type AfterSignUp interface {
	OnAfterSignUp(ctx context.Context, u *user.User, s *session.Session) error
}

// BeforeSignIn is called before authentication.
type BeforeSignIn interface {
	OnBeforeSignIn(ctx context.Context, req *account.SignInRequest) error
}

// AfterSignIn is called after successful authentication.
type AfterSignIn interface {
	OnAfterSignIn(ctx context.Context, u *user.User, s *session.Session) error
}

// BeforeSignOut is called before session termination.
type BeforeSignOut interface {
	OnBeforeSignOut(ctx context.Context, sessionID id.SessionID) error
}

// AfterSignOut is called after session termination.
type AfterSignOut interface {
	OnAfterSignOut(ctx context.Context, sessionID id.SessionID) error
}

// ──────────────────────────────────────────────────
// User lifecycle hooks
// ──────────────────────────────────────────────────

// BeforeUserCreate is called before a user is created.
type BeforeUserCreate interface {
	OnBeforeUserCreate(ctx context.Context, u *user.User) error
}

// AfterUserCreate is called after a user is created.
type AfterUserCreate interface {
	OnAfterUserCreate(ctx context.Context, u *user.User) error
}

// BeforeUserUpdate is called before a user is updated.
type BeforeUserUpdate interface {
	OnBeforeUserUpdate(ctx context.Context, u *user.User) error
}

// AfterUserUpdate is called after a user is updated.
type AfterUserUpdate interface {
	OnAfterUserUpdate(ctx context.Context, u *user.User) error
}

// BeforeUserDelete is called before a user is deleted.
type BeforeUserDelete interface {
	OnBeforeUserDelete(ctx context.Context, userID id.UserID) error
}

// AfterUserDelete is called after a user is deleted.
type AfterUserDelete interface {
	OnAfterUserDelete(ctx context.Context, userID id.UserID) error
}

// ──────────────────────────────────────────────────
// Session lifecycle hooks
// ──────────────────────────────────────────────────

// BeforeSessionCreate is called before a session is created.
type BeforeSessionCreate interface {
	OnBeforeSessionCreate(ctx context.Context, s *session.Session) error
}

// AfterSessionCreate is called after a session is created.
type AfterSessionCreate interface {
	OnAfterSessionCreate(ctx context.Context, s *session.Session) error
}

// AfterSessionRefresh is called after a session token is refreshed.
type AfterSessionRefresh interface {
	OnAfterSessionRefresh(ctx context.Context, s *session.Session) error
}

// AfterSessionRevoke is called after a session is revoked.
type AfterSessionRevoke interface {
	OnAfterSessionRevoke(ctx context.Context, sessionID id.SessionID) error
}

// ──────────────────────────────────────────────────
// Organization lifecycle hooks
// ──────────────────────────────────────────────────

// AfterOrgCreate is called after an organization is created.
type AfterOrgCreate interface {
	OnAfterOrgCreate(ctx context.Context, o *organization.Organization) error
}

// AfterOrgUpdate is called after an organization is updated.
type AfterOrgUpdate interface {
	OnAfterOrgUpdate(ctx context.Context, o *organization.Organization) error
}

// AfterOrgDelete is called after an organization is deleted.
type AfterOrgDelete interface {
	OnAfterOrgDelete(ctx context.Context, orgID id.OrgID) error
}

// AfterMemberAdd is called after a member is added to an organization.
type AfterMemberAdd interface {
	OnAfterMemberAdd(ctx context.Context, m *organization.Member) error
}

// AfterMemberRemove is called after a member is removed from an organization.
type AfterMemberRemove interface {
	OnAfterMemberRemove(ctx context.Context, memberID id.MemberID) error
}

// AfterMemberRoleChange is called after a member's role is changed.
type AfterMemberRoleChange interface {
	OnAfterMemberRoleChange(ctx context.Context, m *organization.Member) error
}

// ──────────────────────────────────────────────────
// Account linking
// ──────────────────────────────────────────────────

// AuthMethod describes a single authentication method linked to a user account.
type AuthMethod struct {
	Type     string `json:"type"`     // e.g. "password", "social:google", "passkey", "phone"
	Provider string `json:"provider"` // e.g. "google", "github", "password", "phone"
	Label    string `json:"label"`    // Human-readable label, e.g. "Google (user@gmail.com)"
	LinkedAt string `json:"linked_at,omitempty"`
}

// AuthMethodContributor is implemented by plugins that can report which
// authentication methods are linked to a user. The engine aggregates these
// to provide a unified "list auth methods" API.
type AuthMethodContributor interface {
	Plugin
	ListUserAuthMethods(ctx context.Context, userID id.UserID) ([]*AuthMethod, error)
}

// AuthMethodUnlinker is optionally implemented by plugins that support
// unlinking an auth method from a user account.
type AuthMethodUnlinker interface {
	Plugin
	UnlinkAuthMethod(ctx context.Context, userID id.UserID, provider string) error
	CanUnlink(ctx context.Context, userID id.UserID, provider string) bool
}

// ──────────────────────────────────────────────────
// Strategy and provider hooks
// ──────────────────────────────────────────────────

// RouteProvider allows a plugin to register additional HTTP routes.
// The router parameter is expected to be a forge.Router; typed as any to
// avoid a circular dependency between the plugin and forge packages.
type RouteProvider interface {
	RegisterRoutes(router any) error
}

// MigrationProvider allows a plugin to register its own grove migration
// groups. The engine collects these groups and passes them to Store.Migrate()
// so plugin tables are created alongside the core schema. The driverName
// parameter ("pg", "sqlite", "mongo") lets the plugin return driver-specific
// migration groups.
type MigrationProvider interface {
	MigrationGroups(driverName string) []*migrate.Group
}

// Extensible allows a plugin to accept sub-plugins.
type Extensible interface {
	RegisterSubPlugin(sub Plugin) error
}

// DataExportContributor allows a plugin to contribute data to the GDPR
// user export. The returned key names the data section (e.g. "organizations")
// and data is the payload that will be included in the export.
type DataExportContributor interface {
	ExportUserData(ctx context.Context, userID id.UserID) (key string, data any, err error)
}

// SettingsProvider is implemented by plugins that declare configurable
// settings via the dynamic settings system. The engine calls DeclareSettings
// during initialization so the settings are registered before use.
type SettingsProvider interface {
	DeclareSettings(m *settings.Manager) error
}

// StrategyProvider is implemented by plugins that contribute an authentication
// strategy to the strategy registry. The engine auto-registers these strategies
// during Start() so they participate in layered auth middleware evaluation.
type StrategyProvider interface {
	Plugin
	Strategy() strategy.Strategy
	StrategyPriority() int
}
