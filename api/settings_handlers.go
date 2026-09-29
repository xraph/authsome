package api

import (
	"errors"
	"fmt"
	"net/http"
	"sort"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/settings"
)

// ──────────────────────────────────────────────────
// Settings route registration
// ──────────────────────────────────────────────────

func (a *API) registerSettingsRoutes(router forge.Router) error {
	// All settings endpoints require admin authentication.
	g := router.Group("/v1/admin/settings",
		forge.WithGroupTags("admin", "settings"),
		forge.WithGroupMiddleware(
			middleware.RequireAuth(),
			middleware.RequirePermission(a.engine, "manage", "settings"),
		),
	)

	// Definition listing.
	if err := g.GET("/definitions", a.handleListDefinitions,
		forge.WithSummary("List all setting definitions"),
		forge.WithDescription("Returns all registered setting definitions grouped by namespace and category. Includes UI metadata for auto-generating settings forms."),
		forge.WithOperationID("listSettingsDefinitions"),
		forge.WithResponseSchema(http.StatusOK, "Setting definitions", ListDefinitionsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/definitions/:namespace", a.handleListNamespaceDefinitions,
		forge.WithSummary("List definitions for a namespace"),
		forge.WithDescription("Returns setting definitions for a specific plugin namespace."),
		forge.WithOperationID("listNamespaceSettingsDefinitions"),
		forge.WithResponseSchema(http.StatusOK, "Namespace definitions", ListDefinitionsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Resolve endpoints.
	if err := g.GET("/resolve", a.handleResolveSettings,
		forge.WithSummary("Resolve all settings at a scope"),
		forge.WithDescription("Resolves all registered settings for the given scope context. Returns effective values, scope cascade breakdown, and enforcement state."),
		forge.WithOperationID("resolveSettings"),
		forge.WithResponseSchema(http.StatusOK, "Resolved settings", ResolvedSettingsResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.GET("/resolve/:key", a.handleResolveSetting,
		forge.WithSummary("Resolve one setting with cascade details"),
		forge.WithDescription("Resolves a single setting with full cascade details including value at each scope, enforcement state, and whether the value can be overridden."),
		forge.WithOperationID("resolveSetting"),
		forge.WithResponseSchema(http.StatusOK, "Resolved setting", ResolvedSettingResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	// Write endpoints.
	if err := g.PUT("/values/:key", a.handleSetSetting,
		forge.WithSummary("Set a setting value at a scope"),
		forge.WithDescription("Sets a setting value at the specified scope. Fails if the setting is enforced at a higher scope."),
		forge.WithOperationID("setSetting"),
		forge.WithRequestSchema(SetSettingRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Setting updated", SettingValueResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.PUT("/enforce/:key", a.handleEnforceSetting,
		forge.WithSummary("Enforce a setting value at a scope"),
		forge.WithDescription("Sets a value AND marks it as enforced, preventing lower scopes from overriding it."),
		forge.WithOperationID("enforceSetting"),
		forge.WithRequestSchema(EnforceSettingRequest{}),
		forge.WithResponseSchema(http.StatusOK, "Setting enforced", SettingValueResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	if err := g.DELETE("/enforce/:key", a.handleUnenforceSetting,
		forge.WithSummary("Remove enforcement from a setting"),
		forge.WithDescription("Removes the enforcement flag from a setting without changing its value."),
		forge.WithOperationID("unenforceSetting"),
		forge.WithResponseSchema(http.StatusOK, "Enforcement removed", StatusResponse{}),
		forge.WithErrorResponses(),
	); err != nil {
		return err
	}

	return g.DELETE("/values/:key", a.handleDeleteSetting,
		forge.WithSummary("Delete a setting override at a scope"),
		forge.WithDescription("Removes a setting override at the specified scope, reverting to the next higher scope's value or the code default."),
		forge.WithOperationID("deleteSetting"),
		forge.WithResponseSchema(http.StatusOK, "Setting deleted", StatusResponse{}),
		forge.WithErrorResponses(),
	)
}

// ──────────────────────────────────────────────────
// Settings handlers
// ──────────────────────────────────────────────────

func (a *API) handleListDefinitions(_ forge.Context, req *ListSettingsDefinitionsRequest) (*ListDefinitionsResponse, error) {
	mgr := a.engine.Settings()
	if mgr == nil {
		return &ListDefinitionsResponse{}, nil
	}

	defs := mgr.Definitions()

	// Apply filters.
	filtered := make([]*settings.Definition, 0, len(defs))
	for _, d := range defs {
		if req.Namespace != "" && d.Namespace != req.Namespace {
			continue
		}
		if req.Category != "" && d.Category != req.Category {
			continue
		}
		filtered = append(filtered, d)
	}

	return buildDefinitionGroupsResponse(filtered), nil
}

func (a *API) handleListNamespaceDefinitions(_ forge.Context, req *ListNamespaceDefinitionsRequest) (*ListDefinitionsResponse, error) {
	mgr := a.engine.Settings()
	if mgr == nil {
		return &ListDefinitionsResponse{}, nil
	}

	defs := mgr.DefinitionsForNamespace(req.Namespace)
	return buildDefinitionGroupsResponse(defs), nil
}

func (a *API) handleResolveSettings(ctx forge.Context, req *ResolveSettingsRequest) (*ResolvedSettingsResponse, error) {
	mgr := a.engine.Settings()
	if mgr == nil {
		return &ResolvedSettingsResponse{}, nil
	}

	opts := settings.ResolveOpts{
		AppID:  req.AppID,
		OrgID:  req.OrgID,
		UserID: req.UserID,
	}

	// If namespace is specified, resolve only that namespace.
	if req.Namespace != "" {
		resolved, err := mgr.ResolveAllForNamespace(ctx.Context(), req.Namespace, opts)
		if err != nil {
			return nil, mapSettingsError(err)
		}
		return &ResolvedSettingsResponse{Settings: resolved}, nil
	}

	// Otherwise, resolve all namespaces.
	var allResolved []*settings.ResolvedSetting
	for _, ns := range mgr.Namespaces() {
		resolved, err := mgr.ResolveAllForNamespace(ctx.Context(), ns, opts)
		if err != nil {
			return nil, mapSettingsError(err)
		}
		allResolved = append(allResolved, resolved...)
	}

	return &ResolvedSettingsResponse{Settings: allResolved}, nil
}

func (a *API) handleResolveSetting(ctx forge.Context, req *ResolveSettingRequest) (*ResolvedSettingResponse, error) {
	mgr := a.engine.Settings()
	if mgr == nil {
		return nil, forge.NotFound("settings not enabled")
	}

	opts := settings.ResolveOpts{
		AppID:  req.AppID,
		OrgID:  req.OrgID,
		UserID: req.UserID,
	}

	rs, err := mgr.ResolveWithDetails(ctx.Context(), req.Key, opts)
	if err != nil {
		return nil, mapSettingsError(err)
	}

	return &ResolvedSettingResponse{Setting: rs}, nil
}

func (a *API) handleSetSetting(ctx forge.Context, req *SetSettingRequest) (*SettingValueResponse, error) {
	mgr := a.engine.Settings()
	if mgr == nil {
		return nil, forge.NotFound("settings not enabled")
	}

	scope := settings.Scope(req.Scope)
	if scope == "" {
		return nil, forge.BadRequest("scope is required")
	}

	// Determine updatedBy from the authenticated user.
	updatedBy := "admin"
	if uid, ok := middleware.UserIDFrom(ctx.Context()); ok {
		updatedBy = uid.String()
	}

	old := a.settingBefore(ctx, mgr, req.Key, req.AppID, req.OrgID)
	if err := a.auditSettingsWrite(ctx, hook.ActionSettingsUpdate, req.Key, scope, req.ScopeID, req.AppID, req.OrgID, old, string(req.Value)); err != nil {
		return nil, err
	}
	if err := mgr.Set(ctx.Context(), req.Key, req.Value, scope, req.ScopeID, req.AppID, req.OrgID, updatedBy); err != nil {
		return nil, mapSettingsError(err)
	}

	return &SettingValueResponse{
		Key:     req.Key,
		Value:   req.Value,
		Scope:   req.Scope,
		ScopeID: req.ScopeID,
		Status:  "updated",
	}, nil
}

// settingBefore returns the effective value of a key before a write so the
// audit record can show what changed. A missing value is reported as "".
func (a *API) settingBefore(ctx forge.Context, mgr *settings.Manager, key, appID, orgID string) string {
	raw, err := mgr.Resolve(ctx.Context(), key, settings.ResolveOpts{AppID: appID, OrgID: orgID})
	if err != nil {
		return ""
	}
	return string(raw)
}

// auditSettingsWrite records a settings change on the audit trail. A change
// at global scope affects every tenant, so it is recorded before the write
// and refused when the trail is unavailable; narrower scopes are recorded
// best-effort so a logging blip cannot block a tenant's own configuration.
func (a *API) auditSettingsWrite(ctx forge.Context, action, key string, scope settings.Scope, scopeID, appID, orgID, oldValue, newValue string) error {
	actor := ""
	if uid, ok := middleware.UserIDFrom(ctx.Context()); ok {
		actor = uid.String()
	}
	tenant := appID
	if tenant == "" {
		if callerApp, ok := a.callerAppID(ctx); ok {
			tenant = callerApp.String()
		}
	}
	ev := &hook.Event{
		Action:     action,
		Resource:   "setting",
		ResourceID: key,
		ActorID:    actor,
		Tenant:     tenant,
		OrgID:      orgID,
		Category:   "settings",
		Metadata: map[string]string{
			"key":      key,
			"scope":    string(scope),
			"scope_id": scopeID,
			"old":      oldValue,
			"new":      newValue,
		},
	}
	if scope == settings.ScopeGlobal {
		ev.Severity = hook.SeverityCritical
		if err := a.engine.Hooks().EmitCritical(ctx.Context(), ev); err != nil {
			return forge.InternalError(fmt.Errorf("audit trail unavailable: %w", err))
		}
		return nil
	}
	a.engine.Hooks().Emit(ctx.Context(), ev)
	return nil
}

func (a *API) handleEnforceSetting(ctx forge.Context, req *EnforceSettingRequest) (*SettingValueResponse, error) {
	mgr := a.engine.Settings()
	if mgr == nil {
		return nil, forge.NotFound("settings not enabled")
	}

	scope := settings.Scope(req.Scope)
	if scope == "" {
		return nil, forge.BadRequest("scope is required")
	}

	updatedBy := "admin"
	if uid, ok := middleware.UserIDFrom(ctx.Context()); ok {
		updatedBy = uid.String()
	}

	old := a.settingBefore(ctx, mgr, req.Key, req.AppID, req.OrgID)
	if err := a.auditSettingsWrite(ctx, hook.ActionSettingsEnforce, req.Key, scope, req.ScopeID, req.AppID, req.OrgID, old, string(req.Value)); err != nil {
		return nil, err
	}
	if err := mgr.Enforce(ctx.Context(), req.Key, req.Value, scope, req.ScopeID, req.AppID, req.OrgID, updatedBy); err != nil {
		return nil, mapSettingsError(err)
	}

	return &SettingValueResponse{
		Key:     req.Key,
		Value:   req.Value,
		Scope:   req.Scope,
		ScopeID: req.ScopeID,
		Status:  "enforced",
	}, nil
}

func (a *API) handleUnenforceSetting(ctx forge.Context, req *UnenforceSettingRequest) (*StatusResponse, error) {
	mgr := a.engine.Settings()
	if mgr == nil {
		return nil, forge.NotFound("settings not enabled")
	}

	scope := settings.Scope(req.Scope)
	if scope == "" {
		return nil, forge.BadRequest("scope is required")
	}

	old := a.settingBefore(ctx, mgr, req.Key, "", "")
	if err := a.auditSettingsWrite(ctx, hook.ActionSettingsUnenforce, req.Key, scope, req.ScopeID, "", "", old, ""); err != nil {
		return nil, err
	}
	if err := mgr.Unenforce(ctx.Context(), req.Key, scope, req.ScopeID); err != nil {
		return nil, mapSettingsError(err)
	}

	return &StatusResponse{Status: "unenforced"}, nil
}

func (a *API) handleDeleteSetting(ctx forge.Context, req *DeleteSettingRequest) (*StatusResponse, error) {
	mgr := a.engine.Settings()
	if mgr == nil {
		return nil, forge.NotFound("settings not enabled")
	}

	scope := settings.Scope(req.Scope)
	if scope == "" {
		return nil, forge.BadRequest("scope is required")
	}

	old := a.settingBefore(ctx, mgr, req.Key, "", "")
	if err := a.auditSettingsWrite(ctx, hook.ActionSettingsDelete, req.Key, scope, req.ScopeID, "", "", old, ""); err != nil {
		return nil, err
	}
	if err := mgr.Delete(ctx.Context(), req.Key, scope, req.ScopeID); err != nil {
		return nil, mapSettingsError(err)
	}

	return &StatusResponse{Status: "deleted"}, nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

// mapSettingsError converts settings package errors into Forge HTTP errors.
func mapSettingsError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, settings.ErrUnknownKey) {
		return forge.NotFound(err.Error())
	}
	if errors.Is(err, settings.ErrNotFound) {
		return forge.NotFound(err.Error())
	}
	if errors.Is(err, settings.ErrScopeNotAllowed) {
		return forge.BadRequest(err.Error())
	}
	if errors.Is(err, settings.ErrEnforcedAtHigher) {
		return forge.NewHTTPError(http.StatusConflict, err.Error())
	}
	if errors.Is(err, settings.ErrNotEnforceable) {
		return forge.BadRequest(err.Error())
	}
	if errors.Is(err, settings.ErrValidation) {
		return forge.BadRequest(err.Error())
	}
	return forge.InternalError(err)
}

// buildDefinitionGroupsResponse groups definitions by namespace+category.
func buildDefinitionGroupsResponse(defs []*settings.Definition) *ListDefinitionsResponse {
	type groupKey struct {
		Namespace string
		Category  string
	}
	groupMap := make(map[groupKey]*DefinitionGroup)
	var groupOrder []groupKey

	for _, d := range defs {
		k := groupKey{Namespace: d.Namespace, Category: d.Category}
		g, ok := groupMap[k]
		if !ok {
			g = &DefinitionGroup{
				Namespace: d.Namespace,
				Category:  d.Category,
			}
			groupMap[k] = g
			groupOrder = append(groupOrder, k)
		}
		g.Definitions = append(g.Definitions, d)
	}

	// Sort groups by namespace then category.
	sort.Slice(groupOrder, func(i, j int) bool {
		if groupOrder[i].Namespace != groupOrder[j].Namespace {
			return groupOrder[i].Namespace < groupOrder[j].Namespace
		}
		return groupOrder[i].Category < groupOrder[j].Category
	})

	groups := make([]DefinitionGroup, 0, len(groupOrder))
	for _, k := range groupOrder {
		g := groupMap[k]
		// Sort definitions within each group by UI order then key.
		sort.Slice(g.Definitions, func(i, j int) bool {
			oi, oj := 0, 0
			if g.Definitions[i].UI != nil {
				oi = g.Definitions[i].UI.Order
			}
			if g.Definitions[j].UI != nil {
				oj = g.Definitions[j].UI.Order
			}
			if oi != oj {
				return oi < oj
			}
			return g.Definitions[i].Key < g.Definitions[j].Key
		})
		groups = append(groups, *g)
	}

	return &ListDefinitionsResponse{
		Groups: groups,
		Total:  len(defs),
	}
}
