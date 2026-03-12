package plugin

import (
	"context"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"

	"github.com/xraph/grove/migrate"
)

// Named entry types pair a hook with the plugin name for logging.

type onInitEntry struct {
	name string
	hook OnInit
}
type onShutdownEntry struct {
	name string
	hook OnShutdown
}
type beforeSignUpEntry struct {
	name string
	hook BeforeSignUp
}
type afterSignUpEntry struct {
	name string
	hook AfterSignUp
}
type beforeSignInEntry struct {
	name string
	hook BeforeSignIn
}
type afterSignInEntry struct {
	name string
	hook AfterSignIn
}
type beforeSignOutEntry struct {
	name string
	hook BeforeSignOut
}
type afterSignOutEntry struct {
	name string
	hook AfterSignOut
}
type beforeUserCreateEntry struct {
	name string
	hook BeforeUserCreate
}
type afterUserCreateEntry struct {
	name string
	hook AfterUserCreate
}
type beforeUserUpdateEntry struct {
	name string
	hook BeforeUserUpdate
}
type afterUserUpdateEntry struct {
	name string
	hook AfterUserUpdate
}
type beforeUserDeleteEntry struct {
	name string
	hook BeforeUserDelete
}
type afterUserDeleteEntry struct {
	name string
	hook AfterUserDelete
}
type beforeSessionCreateEntry struct {
	name string
	hook BeforeSessionCreate
}
type afterSessionCreateEntry struct {
	name string
	hook AfterSessionCreate
}
type afterSessionRefreshEntry struct {
	name string
	hook AfterSessionRefresh
}
type afterSessionRevokeEntry struct {
	name string
	hook AfterSessionRevoke
}
type afterOrgCreateEntry struct {
	name string
	hook AfterOrgCreate
}
type afterOrgUpdateEntry struct {
	name string
	hook AfterOrgUpdate
}
type afterOrgDeleteEntry struct {
	name string
	hook AfterOrgDelete
}
type afterMemberAddEntry struct {
	name string
	hook AfterMemberAdd
}
type afterMemberRemoveEntry struct {
	name string
	hook AfterMemberRemove
}
type afterMemberRoleChangeEntry struct {
	name string
	hook AfterMemberRoleChange
}
type routeProviderEntry struct {
	name string
	hook RouteProvider
}
type migrationProviderEntry struct {
	name string
	hook MigrationProvider
}
type dataExportContributorEntry struct {
	name string
	hook DataExportContributor
}

// Registry holds registered plugins and dispatches lifecycle events.
// It type-caches plugins at registration time so emit calls iterate
// only over plugins implementing the relevant hook.
type Registry struct {
	plugins []Plugin
	logger  log.Logger

	onInit                 []onInitEntry
	onShutdown             []onShutdownEntry
	beforeSignUp           []beforeSignUpEntry
	afterSignUp            []afterSignUpEntry
	beforeSignIn           []beforeSignInEntry
	afterSignIn            []afterSignInEntry
	beforeSignOut          []beforeSignOutEntry
	afterSignOut           []afterSignOutEntry
	beforeUserCreate       []beforeUserCreateEntry
	afterUserCreate        []afterUserCreateEntry
	beforeUserUpdate       []beforeUserUpdateEntry
	afterUserUpdate        []afterUserUpdateEntry
	beforeUserDelete       []beforeUserDeleteEntry
	afterUserDelete        []afterUserDeleteEntry
	beforeSessionCreate    []beforeSessionCreateEntry
	afterSessionCreate     []afterSessionCreateEntry
	afterSessionRefresh    []afterSessionRefreshEntry
	afterSessionRevoke     []afterSessionRevokeEntry
	afterOrgCreate         []afterOrgCreateEntry
	afterOrgUpdate         []afterOrgUpdateEntry
	afterOrgDelete         []afterOrgDeleteEntry
	afterMemberAdd         []afterMemberAddEntry
	afterMemberRemove      []afterMemberRemoveEntry
	afterMemberRoleChange  []afterMemberRoleChangeEntry
	routeProviders         []routeProviderEntry
	migrationProviders     []migrationProviderEntry
	dataExportContributors []dataExportContributorEntry
}

// NewRegistry creates a plugin registry with the given logger.
func NewRegistry(logger log.Logger) *Registry {
	return &Registry{logger: logger}
}

// Register adds a plugin and type-asserts it into all applicable
// hook caches. Plugins are notified in registration order.
func (r *Registry) Register(p Plugin) {
	r.plugins = append(r.plugins, p)
	name := p.Name()

	if h, ok := p.(OnInit); ok {
		r.onInit = append(r.onInit, onInitEntry{name, h})
	}
	if h, ok := p.(OnShutdown); ok {
		r.onShutdown = append(r.onShutdown, onShutdownEntry{name, h})
	}
	if h, ok := p.(BeforeSignUp); ok {
		r.beforeSignUp = append(r.beforeSignUp, beforeSignUpEntry{name, h})
	}
	if h, ok := p.(AfterSignUp); ok {
		r.afterSignUp = append(r.afterSignUp, afterSignUpEntry{name, h})
	}
	if h, ok := p.(BeforeSignIn); ok {
		r.beforeSignIn = append(r.beforeSignIn, beforeSignInEntry{name, h})
	}
	if h, ok := p.(AfterSignIn); ok {
		r.afterSignIn = append(r.afterSignIn, afterSignInEntry{name, h})
	}
	if h, ok := p.(BeforeSignOut); ok {
		r.beforeSignOut = append(r.beforeSignOut, beforeSignOutEntry{name, h})
	}
	if h, ok := p.(AfterSignOut); ok {
		r.afterSignOut = append(r.afterSignOut, afterSignOutEntry{name, h})
	}
	if h, ok := p.(BeforeUserCreate); ok {
		r.beforeUserCreate = append(r.beforeUserCreate, beforeUserCreateEntry{name, h})
	}
	if h, ok := p.(AfterUserCreate); ok {
		r.afterUserCreate = append(r.afterUserCreate, afterUserCreateEntry{name, h})
	}
	if h, ok := p.(BeforeUserUpdate); ok {
		r.beforeUserUpdate = append(r.beforeUserUpdate, beforeUserUpdateEntry{name, h})
	}
	if h, ok := p.(AfterUserUpdate); ok {
		r.afterUserUpdate = append(r.afterUserUpdate, afterUserUpdateEntry{name, h})
	}
	if h, ok := p.(BeforeUserDelete); ok {
		r.beforeUserDelete = append(r.beforeUserDelete, beforeUserDeleteEntry{name, h})
	}
	if h, ok := p.(AfterUserDelete); ok {
		r.afterUserDelete = append(r.afterUserDelete, afterUserDeleteEntry{name, h})
	}
	if h, ok := p.(BeforeSessionCreate); ok {
		r.beforeSessionCreate = append(r.beforeSessionCreate, beforeSessionCreateEntry{name, h})
	}
	if h, ok := p.(AfterSessionCreate); ok {
		r.afterSessionCreate = append(r.afterSessionCreate, afterSessionCreateEntry{name, h})
	}
	if h, ok := p.(AfterSessionRefresh); ok {
		r.afterSessionRefresh = append(r.afterSessionRefresh, afterSessionRefreshEntry{name, h})
	}
	if h, ok := p.(AfterSessionRevoke); ok {
		r.afterSessionRevoke = append(r.afterSessionRevoke, afterSessionRevokeEntry{name, h})
	}
	if h, ok := p.(AfterOrgCreate); ok {
		r.afterOrgCreate = append(r.afterOrgCreate, afterOrgCreateEntry{name, h})
	}
	if h, ok := p.(AfterOrgUpdate); ok {
		r.afterOrgUpdate = append(r.afterOrgUpdate, afterOrgUpdateEntry{name, h})
	}
	if h, ok := p.(AfterOrgDelete); ok {
		r.afterOrgDelete = append(r.afterOrgDelete, afterOrgDeleteEntry{name, h})
	}
	if h, ok := p.(AfterMemberAdd); ok {
		r.afterMemberAdd = append(r.afterMemberAdd, afterMemberAddEntry{name, h})
	}
	if h, ok := p.(AfterMemberRemove); ok {
		r.afterMemberRemove = append(r.afterMemberRemove, afterMemberRemoveEntry{name, h})
	}
	if h, ok := p.(AfterMemberRoleChange); ok {
		r.afterMemberRoleChange = append(r.afterMemberRoleChange, afterMemberRoleChangeEntry{name, h})
	}
	if h, ok := p.(RouteProvider); ok {
		r.routeProviders = append(r.routeProviders, routeProviderEntry{name, h})
	}
	if h, ok := p.(MigrationProvider); ok {
		r.migrationProviders = append(r.migrationProviders, migrationProviderEntry{name, h})
	}
	if h, ok := p.(DataExportContributor); ok {
		r.dataExportContributors = append(r.dataExportContributors, dataExportContributorEntry{name, h})
	}
}

// Plugins returns all registered plugins.
func (r *Registry) Plugins() []Plugin { return r.plugins }

// ──────────────────────────────────────────────────
// Lifecycle emitters
// ──────────────────────────────────────────────────

// EmitOnInit notifies all plugins that implement OnInit.
func (r *Registry) EmitOnInit(ctx context.Context, engine any) {
	for _, e := range r.onInit {
		if err := e.hook.OnInit(ctx, engine); err != nil {
			r.logHookError("OnInit", e.name, err)
		}
	}
}

// EmitOnShutdown notifies all plugins that implement OnShutdown.
func (r *Registry) EmitOnShutdown(ctx context.Context) {
	for _, e := range r.onShutdown {
		if err := e.hook.OnShutdown(ctx); err != nil {
			r.logHookError("OnShutdown", e.name, err)
		}
	}
}

// ──────────────────────────────────────────────────
// Auth event emitters
// ──────────────────────────────────────────────────

// EmitBeforeSignUp notifies all plugins that implement BeforeSignUp.
func (r *Registry) EmitBeforeSignUp(ctx context.Context, req *account.SignUpRequest) error {
	for _, e := range r.beforeSignUp {
		if err := e.hook.OnBeforeSignUp(ctx, req); err != nil {
			return err // Before hooks can abort the operation
		}
	}
	return nil
}

// EmitAfterSignUp notifies all plugins that implement AfterSignUp.
func (r *Registry) EmitAfterSignUp(ctx context.Context, u *user.User, s *session.Session) {
	for _, e := range r.afterSignUp {
		if err := e.hook.OnAfterSignUp(ctx, u, s); err != nil {
			r.logHookError("OnAfterSignUp", e.name, err)
		}
	}
}

// EmitBeforeSignIn notifies all plugins that implement BeforeSignIn.
func (r *Registry) EmitBeforeSignIn(ctx context.Context, req *account.SignInRequest) error {
	for _, e := range r.beforeSignIn {
		if err := e.hook.OnBeforeSignIn(ctx, req); err != nil {
			return err // Before hooks can abort the operation
		}
	}
	return nil
}

// EmitAfterSignIn notifies all plugins that implement AfterSignIn.
func (r *Registry) EmitAfterSignIn(ctx context.Context, u *user.User, s *session.Session) {
	for _, e := range r.afterSignIn {
		if err := e.hook.OnAfterSignIn(ctx, u, s); err != nil {
			r.logHookError("OnAfterSignIn", e.name, err)
		}
	}
}

// EmitBeforeSignOut notifies all plugins that implement BeforeSignOut.
func (r *Registry) EmitBeforeSignOut(ctx context.Context, sessionID id.SessionID) error {
	for _, e := range r.beforeSignOut {
		if err := e.hook.OnBeforeSignOut(ctx, sessionID); err != nil {
			return err
		}
	}
	return nil
}

// EmitAfterSignOut notifies all plugins that implement AfterSignOut.
func (r *Registry) EmitAfterSignOut(ctx context.Context, sessionID id.SessionID) {
	for _, e := range r.afterSignOut {
		if err := e.hook.OnAfterSignOut(ctx, sessionID); err != nil {
			r.logHookError("OnAfterSignOut", e.name, err)
		}
	}
}

// ──────────────────────────────────────────────────
// User lifecycle emitters
// ──────────────────────────────────────────────────

// EmitBeforeUserCreate notifies all plugins that implement BeforeUserCreate.
func (r *Registry) EmitBeforeUserCreate(ctx context.Context, u *user.User) error {
	for _, e := range r.beforeUserCreate {
		if err := e.hook.OnBeforeUserCreate(ctx, u); err != nil {
			return err
		}
	}
	return nil
}

// EmitAfterUserCreate notifies all plugins that implement AfterUserCreate.
func (r *Registry) EmitAfterUserCreate(ctx context.Context, u *user.User) {
	for _, e := range r.afterUserCreate {
		if err := e.hook.OnAfterUserCreate(ctx, u); err != nil {
			r.logHookError("OnAfterUserCreate", e.name, err)
		}
	}
}

// EmitBeforeUserUpdate notifies all plugins that implement BeforeUserUpdate.
func (r *Registry) EmitBeforeUserUpdate(ctx context.Context, u *user.User) error {
	for _, e := range r.beforeUserUpdate {
		if err := e.hook.OnBeforeUserUpdate(ctx, u); err != nil {
			return err
		}
	}
	return nil
}

// EmitAfterUserUpdate notifies all plugins that implement AfterUserUpdate.
func (r *Registry) EmitAfterUserUpdate(ctx context.Context, u *user.User) {
	for _, e := range r.afterUserUpdate {
		if err := e.hook.OnAfterUserUpdate(ctx, u); err != nil {
			r.logHookError("OnAfterUserUpdate", e.name, err)
		}
	}
}

// EmitBeforeUserDelete notifies all plugins that implement BeforeUserDelete.
func (r *Registry) EmitBeforeUserDelete(ctx context.Context, userID id.UserID) error {
	for _, e := range r.beforeUserDelete {
		if err := e.hook.OnBeforeUserDelete(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

// EmitAfterUserDelete notifies all plugins that implement AfterUserDelete.
func (r *Registry) EmitAfterUserDelete(ctx context.Context, userID id.UserID) {
	for _, e := range r.afterUserDelete {
		if err := e.hook.OnAfterUserDelete(ctx, userID); err != nil {
			r.logHookError("OnAfterUserDelete", e.name, err)
		}
	}
}

// ──────────────────────────────────────────────────
// Session lifecycle emitters
// ──────────────────────────────────────────────────

// EmitBeforeSessionCreate notifies all plugins that implement BeforeSessionCreate.
func (r *Registry) EmitBeforeSessionCreate(ctx context.Context, s *session.Session) error {
	for _, e := range r.beforeSessionCreate {
		if err := e.hook.OnBeforeSessionCreate(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// EmitAfterSessionCreate notifies all plugins that implement AfterSessionCreate.
func (r *Registry) EmitAfterSessionCreate(ctx context.Context, s *session.Session) {
	for _, e := range r.afterSessionCreate {
		if err := e.hook.OnAfterSessionCreate(ctx, s); err != nil {
			r.logHookError("OnAfterSessionCreate", e.name, err)
		}
	}
}

// EmitAfterSessionRefresh notifies all plugins that implement AfterSessionRefresh.
func (r *Registry) EmitAfterSessionRefresh(ctx context.Context, s *session.Session) {
	for _, e := range r.afterSessionRefresh {
		if err := e.hook.OnAfterSessionRefresh(ctx, s); err != nil {
			r.logHookError("OnAfterSessionRefresh", e.name, err)
		}
	}
}

// EmitAfterSessionRevoke notifies all plugins that implement AfterSessionRevoke.
func (r *Registry) EmitAfterSessionRevoke(ctx context.Context, sessionID id.SessionID) {
	for _, e := range r.afterSessionRevoke {
		if err := e.hook.OnAfterSessionRevoke(ctx, sessionID); err != nil {
			r.logHookError("OnAfterSessionRevoke", e.name, err)
		}
	}
}

// ──────────────────────────────────────────────────
// Organization lifecycle emitters
// ──────────────────────────────────────────────────

// EmitAfterOrgCreate notifies all plugins that implement AfterOrgCreate.
func (r *Registry) EmitAfterOrgCreate(ctx context.Context, o *organization.Organization) {
	for _, e := range r.afterOrgCreate {
		if err := e.hook.OnAfterOrgCreate(ctx, o); err != nil {
			r.logHookError("OnAfterOrgCreate", e.name, err)
		}
	}
}

// EmitAfterOrgUpdate notifies all plugins that implement AfterOrgUpdate.
func (r *Registry) EmitAfterOrgUpdate(ctx context.Context, o *organization.Organization) {
	for _, e := range r.afterOrgUpdate {
		if err := e.hook.OnAfterOrgUpdate(ctx, o); err != nil {
			r.logHookError("OnAfterOrgUpdate", e.name, err)
		}
	}
}

// EmitAfterOrgDelete notifies all plugins that implement AfterOrgDelete.
func (r *Registry) EmitAfterOrgDelete(ctx context.Context, orgID id.OrgID) {
	for _, e := range r.afterOrgDelete {
		if err := e.hook.OnAfterOrgDelete(ctx, orgID); err != nil {
			r.logHookError("OnAfterOrgDelete", e.name, err)
		}
	}
}

// EmitAfterMemberAdd notifies all plugins that implement AfterMemberAdd.
func (r *Registry) EmitAfterMemberAdd(ctx context.Context, m *organization.Member) {
	for _, e := range r.afterMemberAdd {
		if err := e.hook.OnAfterMemberAdd(ctx, m); err != nil {
			r.logHookError("OnAfterMemberAdd", e.name, err)
		}
	}
}

// EmitAfterMemberRemove notifies all plugins that implement AfterMemberRemove.
func (r *Registry) EmitAfterMemberRemove(ctx context.Context, memberID id.MemberID) {
	for _, e := range r.afterMemberRemove {
		if err := e.hook.OnAfterMemberRemove(ctx, memberID); err != nil {
			r.logHookError("OnAfterMemberRemove", e.name, err)
		}
	}
}

// EmitAfterMemberRoleChange notifies all plugins that implement AfterMemberRoleChange.
func (r *Registry) EmitAfterMemberRoleChange(ctx context.Context, m *organization.Member) {
	for _, e := range r.afterMemberRoleChange {
		if err := e.hook.OnAfterMemberRoleChange(ctx, m); err != nil {
			r.logHookError("OnAfterMemberRoleChange", e.name, err)
		}
	}
}

// ──────────────────────────────────────────────────
// Provider emitters
// ──────────────────────────────────────────────────

// RouteProviders returns all plugins that provide routes.
func (r *Registry) RouteProviders() []RouteProvider {
	providers := make([]RouteProvider, len(r.routeProviders))
	for i, e := range r.routeProviders {
		providers[i] = e.hook
	}
	return providers
}

// CollectMigrationGroups gathers migration groups from all plugins that
// implement MigrationProvider for the given driver name.
func (r *Registry) CollectMigrationGroups(driverName string) []*migrate.Group {
	var groups []*migrate.Group
	for _, e := range r.migrationProviders {
		groups = append(groups, e.hook.MigrationGroups(driverName)...)
	}
	return groups
}

// ──────────────────────────────────────────────────
// Data export emitters
// ──────────────────────────────────────────────────

// CollectExportData gathers data from all plugins that implement
// DataExportContributor. The returned map is keyed by each plugin's
// export key (e.g. "organizations") and contains the exported data.
func (r *Registry) CollectExportData(ctx context.Context, userID id.UserID) map[string]any {
	if len(r.dataExportContributors) == 0 {
		return nil
	}
	result := make(map[string]any, len(r.dataExportContributors))
	for _, e := range r.dataExportContributors {
		key, data, err := e.hook.ExportUserData(ctx, userID)
		if err != nil {
			r.logHookError("ExportUserData", e.name, err)
			continue
		}
		if data != nil {
			result[key] = data
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// logHookError logs a warning when a lifecycle hook returns an error.
// Errors from after-hooks are never propagated — they must not block the pipeline.
func (r *Registry) logHookError(hook, pluginName string, err error) {
	r.logger.Warn("plugin hook error",
		log.String("hook", hook),
		log.String("plugin", pluginName),
		log.String("error", err.Error()),
	)
}
