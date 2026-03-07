package settings

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// Scope represents the specificity level of a setting value.
// Scopes are ordered from least specific (global) to most specific (user).
type Scope string

const (
	// ScopeGlobal applies to the entire AuthSome instance.
	ScopeGlobal Scope = "global"
	// ScopeApp applies to a specific application.
	ScopeApp Scope = "app"
	// ScopeOrg applies to a specific organization within an app.
	ScopeOrg Scope = "org"
	// ScopeUser applies to a specific user within an org/app.
	ScopeUser Scope = "user"
)

// ScopePriority returns the numeric priority of a scope (lower = less specific).
func ScopePriority(s Scope) int {
	switch s {
	case ScopeGlobal:
		return 1
	case ScopeApp:
		return 2
	case ScopeOrg:
		return 3
	case ScopeUser:
		return 4
	default:
		return 0
	}
}

// Setting is a persisted setting override stored in the database.
// Each row represents one (key, scope, scope_id) combination.
type Setting struct {
	ID        id.SettingID    `json:"id"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	Scope     Scope           `json:"scope"`
	ScopeID   string          `json:"scope_id"`
	AppID     string          `json:"app_id"`
	OrgID     string          `json:"org_id"`
	Enforced  bool            `json:"enforced"`
	Namespace string          `json:"namespace"`
	Version   int64           `json:"version"`
	UpdatedBy string          `json:"updated_by"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ScopeValue represents the value of a setting at a particular scope.
// Used in ResolvedSetting to show the full cascade breakdown to UIs.
type ScopeValue struct {
	// Scope is the specificity level (global, app, org, user).
	Scope Scope `json:"scope"`

	// ScopeID is the ID of the entity at this scope (app ID, org ID, user ID).
	ScopeID string `json:"scope_id,omitempty"`

	// Value is the JSON-encoded setting value at this scope.
	Value json.RawMessage `json:"value"`

	// Enforced indicates whether this scope has the value locked.
	Enforced bool `json:"enforced"`

	// Version is the setting version at this scope.
	Version int64 `json:"version,omitempty"`

	// UpdatedBy is who last changed the value at this scope.
	UpdatedBy string `json:"updated_by,omitempty"`

	// UpdatedAt is when the value was last changed at this scope.
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// ResolvedSetting is the full resolution result for a setting key,
// including the definition, effective value, value at each scope,
// and enforcement state. This enables UIs to render scope-aware
// settings forms with override indicators.
type ResolvedSetting struct {
	// Definition is the full setting definition including UI metadata.
	Definition *Definition `json:"definition"`

	// EffectiveValue is the final resolved value after cascade + enforcement.
	EffectiveValue json.RawMessage `json:"effective_value"`

	// ScopeValues lists the value at each scope that has an override.
	// Ordered from least specific (global) to most specific (user).
	ScopeValues []ScopeValue `json:"scope_values"`

	// CanOverride is true if the current scope context allows setting a value.
	// False when a higher scope has this setting enforced.
	CanOverride bool `json:"can_override"`

	// EnforcedAt is the scope that has this setting locked, if any.
	EnforcedAt *Scope `json:"enforced_at,omitempty"`
}

// ChangeEvent is emitted when a setting value changes.
type ChangeEvent struct {
	Key      string          `json:"key"`
	Scope    Scope           `json:"scope"`
	ScopeID  string          `json:"scope_id"`
	AppID    string          `json:"app_id"`
	OrgID    string          `json:"org_id"`
	Enforced bool            `json:"enforced"`
	OldValue json.RawMessage `json:"old_value,omitempty"`
	NewValue json.RawMessage `json:"new_value"`
}

// ChangeListener is called when a setting changes.
type ChangeListener func(event ChangeEvent)

// Sentinel errors.
var (
	ErrNotFound             = errors.New("settings: not found")
	ErrKeyAlreadyRegistered = errors.New("settings: key already registered")
	ErrUnknownKey           = errors.New("settings: unknown key")
	ErrScopeNotAllowed      = errors.New("settings: scope not allowed for this setting")
	ErrEnforcedAtHigher     = errors.New("settings: setting is enforced at a higher scope")
	ErrNotEnforceable       = errors.New("settings: setting is not marked as enforceable")
	ErrValidation           = errors.New("settings: validation failed")
)
