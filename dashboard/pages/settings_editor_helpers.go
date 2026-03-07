package pages

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/settings"
)

// SettingsEditorData holds all data for the settings editor page.
type SettingsEditorData struct {
	Namespaces []SettingsNamespace
	Scope      string // "global", "app", or "org"
	AppID      string // current app ID when scope=app
	OrgID      string // current org ID when scope=org
	BasePath   string // API base path for fetch calls
}

// SettingsNamespace groups settings by plugin namespace.
type SettingsNamespace struct {
	Name       string
	Categories []SettingsCategory
}

// SettingsCategory groups settings by category within a namespace.
type SettingsCategory struct {
	Name     string
	Settings []SettingsField
}

// SettingsField holds all data needed to render a single setting form field.
type SettingsField struct {
	Key            string
	DisplayName    string
	Description    string
	HelpText       string
	InputType      string // "text", "number", "switch", "select", "textarea"
	Placeholder    string
	Options        []formconfig.SelectOption
	Validation     *formconfig.Validation
	EffectiveValue string // JSON string of current value
	DefaultValue   string // JSON string of code default
	IsOverridden   bool   // true if current scope has a custom value
	IsEnforced     bool   // true if enforced at current or higher scope
	CanOverride    bool   // false if enforced at a higher scope
	ReadOnly       bool
	Enforceable    bool
	Scopes         []string
	ValueType      string // "string", "int", "float", "bool", "array", "object"
	Order          int    // UI display order
}

// BuildSettingsEditorData resolves all settings and builds the editor data structure.
func BuildSettingsEditorData(ctx context.Context, mgr *settings.Manager, scope, appID, orgID, basePath string) SettingsEditorData {
	data := SettingsEditorData{
		Scope:    scope,
		AppID:    appID,
		OrgID:    orgID,
		BasePath: basePath,
	}

	opts := settings.ResolveOpts{}
	switch scope {
	case "app":
		if appID != "" {
			opts.AppID = appID
		}
	case "org":
		if orgID != "" {
			opts.OrgID = orgID
		}
		if appID != "" {
			opts.AppID = appID
		}
	}

	namespaces := mgr.Namespaces()
	for _, ns := range namespaces {
		resolved, err := mgr.ResolveAllForNamespace(ctx, ns, opts)
		if err != nil || len(resolved) == 0 {
			continue
		}

		nsData := SettingsNamespace{Name: ns}

		// Group by category.
		catMap := make(map[string][]SettingsField)
		var catOrder []string

		for _, rs := range resolved {
			if rs.Definition == nil || rs.Definition.UI == nil {
				continue
			}

			field := resolvedSettingToField(rs, scope, appID)
			cat := rs.Definition.Category
			if cat == "" {
				cat = "General"
			}

			if _, ok := catMap[cat]; !ok {
				catOrder = append(catOrder, cat)
			}
			catMap[cat] = append(catMap[cat], field)
		}

		for _, cat := range catOrder {
			fields := catMap[cat]
			// Sort by UI order.
			sort.Slice(fields, func(i, j int) bool {
				return fields[i].uiOrder() < fields[j].uiOrder()
			})
			nsData.Categories = append(nsData.Categories, SettingsCategory{
				Name:     cat,
				Settings: fields,
			})
		}

		if len(nsData.Categories) > 0 {
			data.Namespaces = append(data.Namespaces, nsData)
		}
	}

	return data
}

// resolvedSettingToField converts a ResolvedSetting to a SettingsField for UI rendering.
func resolvedSettingToField(rs *settings.ResolvedSetting, scope, appID string) SettingsField {
	def := rs.Definition
	ui := def.UI

	field := SettingsField{
		Key:            def.Key,
		DisplayName:    def.DisplayName,
		Description:    def.Description,
		HelpText:       ui.HelpText,
		InputType:      string(ui.InputType),
		Placeholder:    ui.Placeholder,
		Options:        ui.Options,
		Validation:     ui.Validation,
		EffectiveValue: settingValueToString(rs.EffectiveValue, def.Type),
		DefaultValue:   settingValueToString(def.Default, def.Type),
		CanOverride:    rs.CanOverride,
		ReadOnly:       ui.ReadOnly,
		Enforceable:    def.Enforceable,
		ValueType:      string(def.Type),
		Order:          ui.Order,
	}

	// Build scopes list.
	for _, s := range def.Scopes {
		field.Scopes = append(field.Scopes, string(s))
	}

	// Determine if enforced.
	if rs.EnforcedAt != nil {
		field.IsEnforced = true
	}

	// Check if overridden at current scope.
	for _, sv := range rs.ScopeValues {
		if string(sv.Scope) == scope {
			field.IsOverridden = true
			break
		}
	}

	return field
}

// settingValueToString converts a JSON-encoded setting value to a display string.
func settingValueToString(val json.RawMessage, valueType settings.ValueType) string {
	if len(val) == 0 {
		return ""
	}

	switch valueType {
	case settings.TypeBool:
		var b bool
		if err := json.Unmarshal(val, &b); err == nil {
			return fmt.Sprintf("%v", b)
		}
	case settings.TypeInt:
		var n int64
		if err := json.Unmarshal(val, &n); err == nil {
			return fmt.Sprintf("%d", n)
		}
	case settings.TypeFloat:
		var f float64
		if err := json.Unmarshal(val, &f); err == nil {
			return fmt.Sprintf("%g", f)
		}
	case settings.TypeString:
		var s string
		if err := json.Unmarshal(val, &s); err == nil {
			return s
		}
	case settings.TypeArray:
		var arr []string
		if err := json.Unmarshal(val, &arr); err == nil {
			return strings.Join(arr, "\n")
		}
		// Fall through to raw JSON if not a string array.
	}

	return string(val)
}

// uiOrder returns the UI order for sorting.
func (f SettingsField) uiOrder() int {
	return f.Order
}

// NamespaceDisplayName returns a human-friendly display name for a namespace.
func NamespaceDisplayName(ns string) string {
	// Capitalize first letter and replace underscores with spaces.
	if ns == "" {
		return "General"
	}
	name := strings.ReplaceAll(ns, "_", " ")
	return strings.ToUpper(name[:1]) + name[1:]
}

// ScopeLabel returns a human-friendly label for a scope.
func ScopeLabel(scope string) string {
	switch scope {
	case "global":
		return "Global"
	case "app":
		return "Application"
	case "org":
		return "Organization"
	case "user":
		return "User"
	default:
		return scope
	}
}
