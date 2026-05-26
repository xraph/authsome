// handlers_settings.go: Phase C.10 + D.2.5 — Settings dashboard.
//
// The settings.* surface is what the dashboard's settings.tabs and
// settings.panel renderers consume. Three queries return progressively
// richer projections of the settings system:
//
//   - settings.namespaces: the list of plugin namespaces with display
//     names and counts. Drives the top-level tab strip.
//   - settings.namespace:  the full grouped tree for one namespace,
//     including Definition metadata, the effective value at the
//     selected scope, override / enforced flags, and UIMetadata.
//     Drives the per-tab settings.panel.
//   - settings.list:       (deprecated) flat list of definitions, kept
//     for one release for any callers still bound to it.
//   - settings.detail:     (deprecated) raw resolved value for one
//     key. Replaced by the richer settings.namespace projection.
//
// Three commands cover writes:
//
//   - settings.update:     scope-aware write of a single setting.
//   - settings.enforce:    lock a setting value at a scope so lower
//     scopes cannot override.
//   - settings.unenforce:  release a previously-enforced setting.
package contract

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/settings"

	"github.com/xraph/forge/extensions/dashboard/contract"
)

// ────────────────────────────────────────────────────────────────────
// Deprecated wire shapes (kept until callers migrate)
// ────────────────────────────────────────────────────────────────────

type SettingDefinition struct {
	Key         string          `json:"key"`
	Namespace   string          `json:"namespace,omitempty"`
	Type        string          `json:"type,omitempty"`
	Description string          `json:"description,omitempty"`
	Default     json.RawMessage `json:"default,omitempty"`
}

type SettingsListResponse struct {
	Settings []SettingDefinition `json:"settings"`
}

type GetSettingInput struct {
	Key string `json:"key"`
}

type GetSettingResponse struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

// ────────────────────────────────────────────────────────────────────
// New wire shapes (Phase D.2.5)
// ────────────────────────────────────────────────────────────────────

// NamespaceSummary is the row shape returned by settings.namespaces.
// Field names match the React shell's settings.tabs renderer — renames
// are wire breaks.
type NamespaceSummary struct {
	Name         string `json:"name"`
	DisplayName  string `json:"displayName,omitempty"`
	Description  string `json:"description,omitempty"`
	SettingCount int    `json:"settingCount"`
}

// NamespacesContext echoes the scope IDs the caller's principal has
// available, letting the React shell disable scope-picker choices that
// can't resolve to a concrete ID.
type NamespacesContext struct {
	AppID  string `json:"appId,omitempty"`
	OrgID  string `json:"orgId,omitempty"`
	UserID string `json:"userId,omitempty"`
}

// SettingsNamespacesResponse is the top-level shape settings.tabs reads.
type SettingsNamespacesResponse struct {
	Namespaces []NamespaceSummary `json:"namespaces"`
	Context    NamespacesContext  `json:"context"`
}

// NamespaceInput is the param shape for settings.namespace. Scope is
// required; the *ID fields are conditional on the scope.
type NamespaceInput struct {
	Namespace string `json:"namespace"`
	Scope     string `json:"scope"`
	AppID     string `json:"appId,omitempty"`
	OrgID     string `json:"orgId,omitempty"`
	UserID    string `json:"userId,omitempty"`
}

// SettingField is the per-field projection used inside SettingCategory.
// Mirrors the React shell's SettingField interface in settings.panel.tsx.
type SettingField struct {
	Key            string             `json:"key"`
	DisplayName    string             `json:"displayName"`
	Description    string             `json:"description,omitempty"`
	Type           string             `json:"type"`
	InputType      string             `json:"inputType,omitempty"`
	Default        json.RawMessage    `json:"default,omitempty"`
	EffectiveValue json.RawMessage    `json:"effectiveValue,omitempty"`
	IsOverridden   bool               `json:"isOverridden"`
	IsEnforced     bool               `json:"isEnforced"`
	CanOverride    bool               `json:"canOverride"`
	ReadOnly       bool               `json:"readOnly,omitempty"`
	Sensitive      bool               `json:"sensitive,omitempty"`
	Placeholder    string             `json:"placeholder,omitempty"`
	HelpText       string             `json:"helpText,omitempty"`
	Options        []SettingOption    `json:"options,omitempty"`
	Validation     *SettingValidation `json:"validation,omitempty"`
	Order          int                `json:"order"`
	Section        string             `json:"section,omitempty"`
	Scopes         []string           `json:"scopes,omitempty"`
}

type SettingOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type SettingValidation struct {
	Required bool   `json:"required,omitempty"`
	Min      *int   `json:"min,omitempty"`
	Max      *int   `json:"max,omitempty"`
	MinLen   int    `json:"minLen,omitempty"`
	MaxLen   int    `json:"maxLen,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
}

// SettingCategory groups SettingFields by their Definition.Category.
type SettingCategory struct {
	Name     string         `json:"name"`
	Settings []SettingField `json:"settings"`
}

// SettingsNamespaceResponse is the grouped tree settings.panel reads.
type SettingsNamespaceResponse struct {
	Namespace   string            `json:"namespace"`
	DisplayName string            `json:"displayName,omitempty"`
	Scope       string            `json:"scope"`
	Categories  []SettingCategory `json:"categories"`
}

// UpdateSettingInput extends the deprecated v1 shape with scope-aware
// fields. Scope is optional for back-compat — empty defaults to
// ScopeApp at the principal's app, matching the old behaviour. New
// callers (settings.panel) always pass an explicit scope.
type UpdateSettingInput struct {
	Key    string          `json:"key"`
	Value  json.RawMessage `json:"value"`
	Scope  string          `json:"scope,omitempty"`
	AppID  string          `json:"appId,omitempty"`
	OrgID  string          `json:"orgId,omitempty"`
	UserID string          `json:"userId,omitempty"`
}

// EnforceSettingInput targets a specific (scope, scopeID) and locks the
// setting there so lower scopes cannot override.
type EnforceSettingInput struct {
	Key    string          `json:"key"`
	Value  json.RawMessage `json:"value"`
	Scope  string          `json:"scope"`
	AppID  string          `json:"appId,omitempty"`
	OrgID  string          `json:"orgId,omitempty"`
	UserID string          `json:"userId,omitempty"`
}

// UnenforceSettingInput releases an enforced setting at a scope.
type UnenforceSettingInput struct {
	Key    string `json:"key"`
	Scope  string `json:"scope"`
	AppID  string `json:"appId,omitempty"`
	OrgID  string `json:"orgId,omitempty"`
	UserID string `json:"userId,omitempty"`
}

// ────────────────────────────────────────────────────────────────────
// New handlers (Phase D.2.5)
// ────────────────────────────────────────────────────────────────────

func settingsNamespacesHandler(deps Deps) func(ctx context.Context, _ struct{}, p contract.Principal) (SettingsNamespacesResponse, error) {
	return func(_ context.Context, _ struct{}, p contract.Principal) (SettingsNamespacesResponse, error) {
		if deps.Engine == nil {
			return SettingsNamespacesResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		mgr := deps.Engine.Settings()
		if mgr == nil {
			return SettingsNamespacesResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "settings manager not configured"}
		}
		names := mgr.Namespaces()
		sort.Strings(names)
		out := SettingsNamespacesResponse{
			Namespaces: make([]NamespaceSummary, 0, len(names)),
			Context: NamespacesContext{
				AppID:  AppIDFromPrincipal(p, deps.Engine).String(),
				UserID: principalSubjectID(p),
			},
		}
		for _, ns := range names {
			defs := mgr.DefinitionsForNamespace(ns)
			out.Namespaces = append(out.Namespaces, NamespaceSummary{
				Name:         ns,
				DisplayName:  namespaceDisplayName(ns),
				SettingCount: len(defs),
			})
		}
		return out, nil
	}
}

func settingsNamespaceHandler(deps Deps) func(ctx context.Context, in NamespaceInput, p contract.Principal) (SettingsNamespaceResponse, error) {
	return func(ctx context.Context, in NamespaceInput, p contract.Principal) (SettingsNamespaceResponse, error) {
		if deps.Engine == nil {
			return SettingsNamespaceResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		mgr := deps.Engine.Settings()
		if mgr == nil {
			return SettingsNamespaceResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "settings manager not configured"}
		}
		ns := strings.TrimSpace(in.Namespace)
		if ns == "" {
			return SettingsNamespaceResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "namespace is required"}
		}
		opts := resolveOptsFromInput(in, deps, p)
		resolved, err := mgr.ResolveAllForNamespace(ctx, ns, opts)
		if err != nil {
			return SettingsNamespaceResponse{}, mapEngineError(err)
		}
		out := SettingsNamespaceResponse{
			Namespace:   ns,
			DisplayName: namespaceDisplayName(ns),
			Scope:       strings.ToLower(in.Scope),
			Categories:  groupResolved(resolved),
		}
		return out, nil
	}
}

// settingsUpdateHandler is the scope-aware v2 writer. Empty Scope
// preserves the v1 behaviour of writing to the principal's app scope.
func settingsUpdateHandler(deps Deps) func(ctx context.Context, in UpdateSettingInput, p contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in UpdateSettingInput, p contract.Principal) (AckResponse, error) {
		key := strings.TrimSpace(in.Key)
		if key == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "key is required"}
		}
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		mgr := deps.Engine.Settings()
		if mgr == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "settings manager not configured"}
		}
		adminID, err := principalUserID(p)
		if err != nil {
			return AckResponse{}, err
		}
		scope, scopeID, appID, orgID := writeScopeFromInput(in.Scope, in.AppID, in.OrgID, in.UserID, deps, p)
		if err := mgr.Set(ctx, key, in.Value, scope, scopeID, appID, orgID, adminID.String()); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: key}, nil
	}
}

func settingsEnforceHandler(deps Deps) func(ctx context.Context, in EnforceSettingInput, p contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in EnforceSettingInput, p contract.Principal) (AckResponse, error) {
		key := strings.TrimSpace(in.Key)
		if key == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "key is required"}
		}
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		mgr := deps.Engine.Settings()
		if mgr == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "settings manager not configured"}
		}
		adminID, err := principalUserID(p)
		if err != nil {
			return AckResponse{}, err
		}
		scope, scopeID, appID, orgID := writeScopeFromInput(in.Scope, in.AppID, in.OrgID, in.UserID, deps, p)
		if err := mgr.Enforce(ctx, key, in.Value, scope, scopeID, appID, orgID, adminID.String()); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: key}, nil
	}
}

func settingsUnenforceHandler(deps Deps) func(ctx context.Context, in UnenforceSettingInput, p contract.Principal) (AckResponse, error) {
	return func(ctx context.Context, in UnenforceSettingInput, p contract.Principal) (AckResponse, error) {
		key := strings.TrimSpace(in.Key)
		if key == "" {
			return AckResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "key is required"}
		}
		if deps.Engine == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		mgr := deps.Engine.Settings()
		if mgr == nil {
			return AckResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "settings manager not configured"}
		}
		scope, scopeID, _, _ := writeScopeFromInput(in.Scope, in.AppID, in.OrgID, in.UserID, deps, p)
		if err := mgr.Unenforce(ctx, key, scope, scopeID); err != nil {
			return AckResponse{}, mapEngineError(err)
		}
		return AckResponse{OK: true, ID: key}, nil
	}
}

// ────────────────────────────────────────────────────────────────────
// Deprecated handlers (kept for one release)
// ────────────────────────────────────────────────────────────────────

func settingsListHandler(deps Deps) func(ctx context.Context, _ struct{}, _ contract.Principal) (SettingsListResponse, error) {
	return func(_ context.Context, _ struct{}, _ contract.Principal) (SettingsListResponse, error) {
		if deps.Engine == nil {
			return SettingsListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		mgr := deps.Engine.Settings()
		if mgr == nil {
			return SettingsListResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "settings manager not configured"}
		}
		defs := mgr.Definitions()
		out := SettingsListResponse{Settings: make([]SettingDefinition, 0, len(defs))}
		for _, def := range defs {
			out.Settings = append(out.Settings, projectSettingDef(def))
		}
		return out, nil
	}
}

func settingsDetailHandler(deps Deps) func(ctx context.Context, in GetSettingInput, _ contract.Principal) (GetSettingResponse, error) {
	return func(ctx context.Context, in GetSettingInput, _ contract.Principal) (GetSettingResponse, error) {
		if deps.Engine == nil {
			return GetSettingResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "auth engine not configured"}
		}
		mgr := deps.Engine.Settings()
		if mgr == nil {
			return GetSettingResponse{}, &contract.Error{Code: contract.CodeUnavailable, Message: "settings manager not configured"}
		}
		key := strings.TrimSpace(in.Key)
		if key == "" {
			return GetSettingResponse{}, &contract.Error{Code: contract.CodeBadRequest, Message: "key is required"}
		}
		val, err := mgr.Resolve(ctx, key, settings.ResolveOpts{AppID: defaultAppID(deps.Engine).String()})
		if err != nil {
			return GetSettingResponse{}, mapEngineError(err)
		}
		return GetSettingResponse{Key: key, Value: val}, nil
	}
}

func projectSettingDef(def *settings.Definition) SettingDefinition {
	if def == nil {
		return SettingDefinition{}
	}
	return SettingDefinition{
		Key:         def.Key,
		Namespace:   def.Namespace,
		Type:        string(def.Type),
		Description: def.Description,
		Default:     def.Default,
	}
}

// ────────────────────────────────────────────────────────────────────
// Projection + helpers
// ────────────────────────────────────────────────────────────────────

func projectSettingField(r *settings.ResolvedSetting) SettingField {
	if r == nil || r.Definition == nil {
		return SettingField{}
	}
	def := r.Definition
	out := SettingField{
		Key:            def.Key,
		DisplayName:    fallback(def.DisplayName, def.Key),
		Description:    def.Description,
		Type:           string(def.Type),
		Default:        def.Default,
		EffectiveValue: r.EffectiveValue,
		IsOverridden:   len(r.ScopeValues) > 0,
		IsEnforced:     r.EnforcedAt != nil,
		CanOverride:    r.CanOverride,
		Sensitive:      def.Sensitive,
		Scopes:         scopeNames(def.Scopes),
	}
	// Redact sensitive values before they cross the wire. The schema /
	// validation hints still flow so the form can still render an input
	// (typed by the user) and submit; only the read-back is hidden.
	if def.Sensitive && len(r.EffectiveValue) > 0 {
		out.EffectiveValue = json.RawMessage(`"***"`)
	}
	if def.UI != nil {
		ui := def.UI
		out.InputType = string(ui.InputType)
		out.Placeholder = ui.Placeholder
		out.HelpText = ui.HelpText
		out.Order = ui.Order
		out.ReadOnly = ui.ReadOnly
		out.Section = ui.Section
		if len(ui.Options) > 0 {
			out.Options = make([]SettingOption, 0, len(ui.Options))
			for _, o := range ui.Options {
				out.Options = append(out.Options, SettingOption{Label: o.Label, Value: o.Value})
			}
		}
		if ui.Validation != nil {
			out.Validation = projectValidation(*ui.Validation)
		}
	}
	return out
}

func projectValidation(v formconfig.Validation) *SettingValidation {
	return &SettingValidation{
		Required: v.Required,
		Min:      v.Min,
		Max:      v.Max,
		MinLen:   v.MinLen,
		MaxLen:   v.MaxLen,
		Pattern:  v.Pattern,
	}
}

// groupByCategory partitions a flat slice of SettingField into a stable
// list of SettingCategory. Categories are sorted alphabetically; fields
// within each category preserve the input ordering (which mgr.ResolveAllForNamespace
// already sorted by Order).
func groupByCategory(fields []SettingField) []SettingCategory {
	if len(fields) == 0 {
		return nil
	}
	byCat := make(map[string][]SettingField)
	order := make([]string, 0)
	for _, f := range fields {
		cat := categoryOf(f)
		if _, ok := byCat[cat]; !ok {
			order = append(order, cat)
		}
		byCat[cat] = append(byCat[cat], f)
	}
	sort.Strings(order)
	out := make([]SettingCategory, 0, len(order))
	for _, name := range order {
		out = append(out, SettingCategory{Name: name, Settings: byCat[name]})
	}
	return out
}

// categoryOf returns the Category attribute when present, falling back
// to "General" so uncategorised settings still render in one card
// instead of vanishing.
func categoryOf(f SettingField) string {
	// SettingField doesn't carry Category directly because the projection
	// already grouped by it upstream — but our caller projects flat, so
	// pull Category off the source Definition via the global lookup.
	// Realistically we want this at projection time; the field is left
	// off the wire to avoid duplicating it across every row. As a
	// pragmatic compromise we reuse Section (or fall back to "General")
	// when no category info is plumbed through.
	if f.Section != "" {
		return f.Section
	}
	return "General"
}

// projectSettingFieldWithCategory is a richer projector used by
// groupByCategory: instead of losing the Definition.Category at
// projection time, we capture both the Field and its category in one
// step. Used internally by namespace projection.
func projectSettingFieldWithCategory(r *settings.ResolvedSetting) (string, SettingField) {
	cat := "General"
	if r != nil && r.Definition != nil && r.Definition.Category != "" {
		cat = r.Definition.Category
	}
	return cat, projectSettingField(r)
}

// groupResolved walks the ResolvedSetting slice and partitions by
// Definition.Category, sorting categories alphabetically and preserving
// the per-category Order-sorted field order from ResolveAllForNamespace.
func groupResolved(resolved []*settings.ResolvedSetting) []SettingCategory {
	byCat := make(map[string][]SettingField)
	order := make([]string, 0)
	for _, r := range resolved {
		cat, field := projectSettingFieldWithCategory(r)
		if _, ok := byCat[cat]; !ok {
			order = append(order, cat)
		}
		byCat[cat] = append(byCat[cat], field)
	}
	sort.Strings(order)
	out := make([]SettingCategory, 0, len(order))
	for _, name := range order {
		out = append(out, SettingCategory{Name: name, Settings: byCat[name]})
	}
	return out
}

// resolveOptsFromInput builds a settings.ResolveOpts from a wire-side
// NamespaceInput. For "app" scope without an explicit AppID, the
// principal's app is used; for "global" the IDs are zeroed.
func resolveOptsFromInput(in NamespaceInput, deps Deps, p contract.Principal) settings.ResolveOpts {
	scope := strings.ToLower(in.Scope)
	out := settings.ResolveOpts{}
	switch scope {
	case "app", "":
		out.AppID = firstNonEmpty(in.AppID, AppIDFromPrincipal(p, deps.Engine).String())
	case "org":
		out.AppID = firstNonEmpty(in.AppID, AppIDFromPrincipal(p, deps.Engine).String())
		out.OrgID = in.OrgID
	case "user":
		out.AppID = firstNonEmpty(in.AppID, AppIDFromPrincipal(p, deps.Engine).String())
		out.OrgID = in.OrgID
		out.UserID = firstNonEmpty(in.UserID, principalSubjectID(p))
	case "global":
		// no IDs — global resolution
	}
	return out
}

// writeScopeFromInput translates a wire-side scope+IDs to the
// (scope, scopeID, appID, orgID) tuple expected by mgr.Set / Enforce.
// Empty scope defaults to ScopeApp at the principal's app — preserves
// the v1 settings.update handler's behaviour for legacy callers.
func writeScopeFromInput(scopeStr, appID, orgID, userID string, deps Deps, p contract.Principal) (settings.Scope, string, string, string) {
	scope := strings.ToLower(scopeStr)
	resolvedApp := firstNonEmpty(appID, AppIDFromPrincipal(p, deps.Engine).String())
	switch scope {
	case "global":
		return settings.ScopeGlobal, "", "", ""
	case "org":
		return settings.ScopeOrg, orgID, resolvedApp, orgID
	case "user":
		uid := firstNonEmpty(userID, principalSubjectID(p))
		return settings.ScopeUser, uid, resolvedApp, orgID
	case "app", "":
		return settings.ScopeApp, resolvedApp, resolvedApp, ""
	}
	return settings.ScopeApp, resolvedApp, resolvedApp, ""
}

func principalSubjectID(p contract.Principal) string {
	if p.User == nil {
		return ""
	}
	return p.User.Subject
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func fallback(a, b string) string { return firstNonEmpty(a, b) }

func scopeNames(scopes []settings.Scope) []string {
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		out = append(out, string(s))
	}
	return out
}

// namespaceDisplayName is a best-effort prettifier — first letter
// uppercased, hyphens turned into spaces. Plugins that want a richer
// display name should add a Namespace metadata channel in a future
// iteration; for now this gives readable tab labels.
func namespaceDisplayName(ns string) string {
	if ns == "" {
		return ns
	}
	runes := []rune(ns)
	first := runes[0]
	if first >= 'a' && first <= 'z' {
		runes[0] = first - 'a' + 'A'
	}
	out := string(runes)
	return strings.ReplaceAll(out, "-", " ")
}

// Override the unused-import sentinel guards if any types end up
// unreferenced — these names are surfaced for downstream linkers
// (handler_test.go).
var (
	_ = groupByCategory
	_ = projectSettingFieldWithCategory
)
