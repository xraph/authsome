package settings

import (
	"context"
	"encoding/json"
)

// EnforcementRule represents a programmatic enforcement applied by a plugin
// (e.g., a compliance plugin). It forces a setting value at a given scope
// and prevents lower scopes from overriding it.
type EnforcementRule struct {
	// Key is the setting key to enforce.
	Key string `json:"key"`

	// Value is the enforced value (JSON-encoded).
	Value json.RawMessage `json:"value"`

	// Scope is the scope at which enforcement applies.
	Scope Scope `json:"scope"`

	// ScopeID is the target scope ID (empty for global).
	ScopeID string `json:"scope_id"`

	// Reason explains why this enforcement exists (e.g., "SOC2 compliance").
	Reason string `json:"reason"`

	// Source is the plugin that created this rule.
	Source string `json:"source"`
}

// EnforcementProvider is implemented by plugins that programmatically
// enforce setting values (e.g., compliance, security hardening).
type EnforcementProvider interface {
	// EnforcementRules returns the current enforcement rules.
	// Called during settings resolution to check for forced values.
	EnforcementRules(ctx context.Context) ([]EnforcementRule, error)
}
