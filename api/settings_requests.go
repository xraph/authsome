package api

import (
	"encoding/json"

	"github.com/xraph/authsome/settings"
)

// ---------------------------------------------------------------------------
// Settings definition requests
// ---------------------------------------------------------------------------

// ListSettingsDefinitionsRequest binds query params for GET /admin/settings/definitions.
type ListSettingsDefinitionsRequest struct {
	Namespace string `query:"namespace" description:"Filter by plugin namespace"`
	Category  string `query:"category" description:"Filter by category"`
}

// ListNamespaceDefinitionsRequest binds the path for GET /admin/settings/definitions/:namespace.
type ListNamespaceDefinitionsRequest struct {
	Namespace string `path:"namespace" description:"Plugin namespace"`
}

// ---------------------------------------------------------------------------
// Settings resolve requests
// ---------------------------------------------------------------------------

// ResolveSettingsRequest binds query params for GET /admin/settings/resolve.
type ResolveSettingsRequest struct {
	Namespace string `query:"namespace" description:"Filter by plugin namespace"`
	AppID     string `query:"app_id" description:"Application ID for scope context"`
	OrgID     string `query:"org_id" description:"Organization ID for scope context"`
	UserID    string `query:"user_id" description:"User ID for scope context"`
}

// ResolveSettingRequest binds the path + query for GET /admin/settings/resolve/:key.
type ResolveSettingRequest struct {
	Key    string `path:"key" description:"Setting key (e.g. password.min_length)"`
	AppID  string `query:"app_id" description:"Application ID for scope context"`
	OrgID  string `query:"org_id" description:"Organization ID for scope context"`
	UserID string `query:"user_id" description:"User ID for scope context"`
}

// ---------------------------------------------------------------------------
// Settings write requests
// ---------------------------------------------------------------------------

// SetSettingRequest binds the path + body for PUT /admin/settings/values/:key.
type SetSettingRequest struct {
	Key     string          `path:"key" description:"Setting key"`
	Value   json.RawMessage `json:"value" description:"Setting value (JSON)"`
	Scope   string          `json:"scope" description:"Target scope (global, app, org, user)"`
	ScopeID string          `json:"scope_id,omitempty" description:"Entity ID at the target scope"`
	AppID   string          `json:"app_id,omitempty" description:"Application ID"`
	OrgID   string          `json:"org_id,omitempty" description:"Organization ID"`
}

// EnforceSettingRequest binds the path + body for PUT /admin/settings/enforce/:key.
type EnforceSettingRequest struct {
	Key     string          `path:"key" description:"Setting key"`
	Value   json.RawMessage `json:"value" description:"Setting value (JSON)"`
	Scope   string          `json:"scope" description:"Target scope (global, app, org, user)"`
	ScopeID string          `json:"scope_id,omitempty" description:"Entity ID at the target scope"`
	AppID   string          `json:"app_id,omitempty" description:"Application ID"`
	OrgID   string          `json:"org_id,omitempty" description:"Organization ID"`
}

// UnenforceSettingRequest binds the path + query for DELETE /admin/settings/enforce/:key.
type UnenforceSettingRequest struct {
	Key     string `path:"key" description:"Setting key"`
	Scope   string `query:"scope" description:"Target scope (global, app, org, user)"`
	ScopeID string `query:"scope_id" description:"Entity ID at the target scope"`
}

// DeleteSettingRequest binds the path + query for DELETE /admin/settings/values/:key.
type DeleteSettingRequest struct {
	Key     string `path:"key" description:"Setting key"`
	Scope   string `query:"scope" description:"Target scope (global, app, org, user)"`
	ScopeID string `query:"scope_id" description:"Entity ID at the target scope"`
}

// ---------------------------------------------------------------------------
// Settings response types
// ---------------------------------------------------------------------------

// DefinitionGroup groups definitions by namespace and category for display.
type DefinitionGroup struct {
	Namespace   string                 `json:"namespace" description:"Plugin namespace"`
	Category    string                 `json:"category" description:"Settings category"`
	Definitions []*settings.Definition `json:"definitions" description:"Setting definitions in this group"`
}

// ListDefinitionsResponse wraps grouped definitions.
type ListDefinitionsResponse struct {
	Groups []DefinitionGroup `json:"groups" description:"Definitions grouped by namespace and category"`
	Total  int               `json:"total" description:"Total number of definitions"`
}

// ResolvedSettingsResponse wraps resolved settings with full cascade details.
type ResolvedSettingsResponse struct {
	Settings []*settings.ResolvedSetting `json:"settings" description:"Resolved settings with scope cascade details"`
}

// ResolvedSettingResponse wraps a single resolved setting.
type ResolvedSettingResponse struct {
	Setting *settings.ResolvedSetting `json:"setting" description:"Resolved setting with scope cascade details"`
}

// SettingValueResponse wraps a setting write result.
type SettingValueResponse struct {
	Key     string          `json:"key" description:"Setting key"`
	Value   json.RawMessage `json:"value" description:"Setting value"`
	Scope   string          `json:"scope" description:"Scope where the value was set"`
	ScopeID string          `json:"scope_id,omitempty" description:"Scope entity ID"`
	Status  string          `json:"status" description:"Operation status"`
}
