package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
)

// Manager is the central settings coordinator. It holds the definition
// registry, provides the resolution API, and emits change events.
type Manager struct {
	store  Store
	logger log.Logger

	mu          sync.RWMutex
	definitions map[string]*Definition
	namespaces  map[string][]string
	listeners   map[string][]ChangeListener
	enforcers   []EnforcementProvider
}

// NewManager creates a new settings manager.
func NewManager(store Store, logger log.Logger) *Manager {
	return &Manager{
		store:       store,
		logger:      logger,
		definitions: make(map[string]*Definition),
		namespaces:  make(map[string][]string),
		listeners:   make(map[string][]ChangeListener),
	}
}

// Register adds a setting definition to the registry.
func (m *Manager) Register(def Definition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.definitions[def.Key]; exists {
		return fmt.Errorf("%w: %q", ErrKeyAlreadyRegistered, def.Key)
	}

	m.definitions[def.Key] = &def
	m.namespaces[def.Namespace] = append(m.namespaces[def.Namespace], def.Key)
	return nil
}

// RegisterTyped registers a typed setting definition under the given namespace.
func RegisterTyped[T any](m *Manager, namespace string, def DefinitionTyped[T]) error {
	d := def.Def
	d.Namespace = namespace
	return m.Register(d)
}

// Resolve returns the effective value for a setting key, performing the
// full enforcement-aware cascade:
//
//	code default → global → app → org → user
//
// If any scope has enforced=true, the cascade stops there.
// EnforcementProviders are checked for programmatic rules.
func (m *Manager) Resolve(ctx context.Context, key string, opts ResolveOpts) (json.RawMessage, error) {
	m.mu.RLock()
	def, ok := m.definitions[key]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}

	// Check programmatic enforcement first (highest priority).
	if rule, found := m.findEnforcementRule(ctx, key, opts); found {
		return rule.Value, nil
	}

	// Fetch all scoped settings from store.
	settings, err := m.store.ResolveSettings(ctx, key, opts)
	if err != nil {
		return nil, fmt.Errorf("settings: resolve %q: %w", key, err)
	}

	// Walk through scopes in order (global → app → org → user).
	// If a scope is enforced, stop cascading.
	resolved := def.Default
	for _, s := range settings {
		resolved = s.Value
		if s.Enforced {
			break // enforced: stop cascade, lower scopes cannot override
		}
	}

	return resolved, nil
}

// Get is the type-safe generic resolver. It resolves the setting and
// unmarshals it into the Go type T declared in the Definition.
func Get[T any](ctx context.Context, m *Manager, def DefinitionTyped[T], opts ResolveOpts) (T, error) {
	var zero T
	raw, err := m.Resolve(ctx, def.Def.Key, opts)
	if err != nil {
		return zero, err
	}
	var val T
	if err := json.Unmarshal(raw, &val); err != nil {
		return zero, fmt.Errorf("settings: unmarshal %q into %T: %w", def.Def.Key, zero, err)
	}
	return val, nil
}

// Set writes a setting value at the given scope. It validates the value,
// checks scope permissions, checks enforcement, and emits a change event.
func (m *Manager) Set(ctx context.Context, key string, value json.RawMessage, scope Scope, scopeID, appID, orgID, updatedBy string) error {
	m.mu.RLock()
	def, ok := m.definitions[key]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}

	// Validate scope is allowed.
	if !def.HasScope(scope) {
		return fmt.Errorf("%w: key %q does not allow scope %q", ErrScopeNotAllowed, key, scope)
	}

	// Run validation function if defined.
	if def.Validate != nil {
		if err := def.Validate(value); err != nil {
			return fmt.Errorf("%w: %w", ErrValidation, err)
		}
	}

	// Check that no higher scope has this key enforced.
	enforced, _, err := m.isEnforcedAbove(ctx, key, scope, appID, orgID)
	if err != nil {
		return err
	}
	if enforced {
		return fmt.Errorf("%w: key %q", ErrEnforcedAtHigher, key)
	}

	// Fetch old value for the change event (best-effort).
	old, getErr := m.store.GetSetting(ctx, key, scope, scopeID)
	if getErr != nil {
		old = nil
	}

	now := timeNow()
	s := &Setting{
		ID:        id.NewSettingID(),
		Key:       key,
		Value:     value,
		Scope:     scope,
		ScopeID:   scopeID,
		AppID:     appID,
		OrgID:     orgID,
		Enforced:  false,
		Namespace: def.Namespace,
		Version:   1,
		UpdatedBy: updatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if old != nil {
		s.ID = old.ID
		s.Version = old.Version + 1
		s.CreatedAt = old.CreatedAt
		s.Enforced = old.Enforced // preserve enforcement flag
	}

	if err := m.store.SetSetting(ctx, s); err != nil {
		return fmt.Errorf("settings: set %q: %w", key, err)
	}

	// Emit change event.
	evt := ChangeEvent{
		Key:      key,
		Scope:    scope,
		ScopeID:  scopeID,
		AppID:    appID,
		OrgID:    orgID,
		Enforced: s.Enforced,
		NewValue: value,
	}
	if old != nil {
		evt.OldValue = old.Value
	}
	m.emitChange(evt)

	return nil
}

// Enforce sets a value AND marks it as enforced, preventing lower scopes
// from overriding it. Only allowed if the definition has Enforceable=true.
func (m *Manager) Enforce(ctx context.Context, key string, value json.RawMessage, scope Scope, scopeID, appID, orgID, updatedBy string) error {
	m.mu.RLock()
	def, ok := m.definitions[key]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}
	if !def.Enforceable {
		return fmt.Errorf("%w: key %q", ErrNotEnforceable, key)
	}
	if !def.HasScope(scope) {
		return fmt.Errorf("%w: key %q does not allow scope %q", ErrScopeNotAllowed, key, scope)
	}

	// Run validation function if defined.
	if def.Validate != nil {
		if err := def.Validate(value); err != nil {
			return fmt.Errorf("%w: %w", ErrValidation, err)
		}
	}

	old, getErr := m.store.GetSetting(ctx, key, scope, scopeID)
	if getErr != nil {
		old = nil
	}

	now := timeNow()
	s := &Setting{
		ID:        id.NewSettingID(),
		Key:       key,
		Value:     value,
		Scope:     scope,
		ScopeID:   scopeID,
		AppID:     appID,
		OrgID:     orgID,
		Enforced:  true,
		Namespace: def.Namespace,
		Version:   1,
		UpdatedBy: updatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if old != nil {
		s.ID = old.ID
		s.Version = old.Version + 1
		s.CreatedAt = old.CreatedAt
	}

	if err := m.store.SetSetting(ctx, s); err != nil {
		return fmt.Errorf("settings: enforce %q: %w", key, err)
	}

	evt := ChangeEvent{
		Key:      key,
		Scope:    scope,
		ScopeID:  scopeID,
		AppID:    appID,
		OrgID:    orgID,
		Enforced: true,
		NewValue: value,
	}
	if old != nil {
		evt.OldValue = old.Value
	}
	m.emitChange(evt)

	return nil
}

// Unenforce removes the enforcement flag from a setting without changing
// its value. Lower scopes can then override it again.
func (m *Manager) Unenforce(ctx context.Context, key string, scope Scope, scopeID string) error {
	s, err := m.store.GetSetting(ctx, key, scope, scopeID)
	if err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("%w: %q at scope %q", ErrNotFound, key, scope)
	}

	s.Enforced = false
	s.Version++
	s.UpdatedAt = timeNow()
	return m.store.SetSetting(ctx, s)
}

// IsEnforced checks if a setting is enforced at any scope in the resolution
// chain for the given context. Returns the enforcing setting if found.
func (m *Manager) IsEnforced(ctx context.Context, key string, opts ResolveOpts) (bool, *Setting, error) {
	settings, err := m.store.ResolveSettings(ctx, key, opts)
	if err != nil {
		return false, nil, err
	}
	for _, s := range settings {
		if s.Enforced {
			return true, s, nil
		}
	}
	return false, nil, nil
}

// Delete removes a setting override at a specific scope.
func (m *Manager) Delete(ctx context.Context, key string, scope Scope, scopeID string) error {
	return m.store.DeleteSetting(ctx, key, scope, scopeID)
}

// BatchResolve resolves multiple settings in a single store call.
func (m *Manager) BatchResolve(ctx context.Context, keys []string, opts ResolveOpts) (map[string]json.RawMessage, error) {
	dbValues, err := m.store.BatchResolve(ctx, keys, opts)
	if err != nil {
		return nil, err
	}

	result := make(map[string]json.RawMessage, len(keys))
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, key := range keys {
		// Check programmatic enforcement.
		if rule, found := m.findEnforcementRule(ctx, key, opts); found {
			result[key] = rule.Value
			continue
		}

		// Walk the cascade with enforcement.
		if settings, ok := dbValues[key]; ok && len(settings) > 0 {
			var resolved json.RawMessage
			for _, s := range settings {
				resolved = s.Value
				if s.Enforced {
					break
				}
			}
			result[key] = resolved
		} else if def, ok := m.definitions[key]; ok {
			result[key] = def.Default
		}
	}
	return result, nil
}

// Definitions returns all registered definitions.
func (m *Manager) Definitions() []*Definition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	defs := make([]*Definition, 0, len(m.definitions))
	for _, d := range m.definitions {
		defs = append(defs, d)
	}
	return defs
}

// Definition returns a single definition by key, or nil if not found.
func (m *Manager) Definition(key string) *Definition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.definitions[key]
}

// DefinitionsForNamespace returns definitions registered by a specific plugin.
func (m *Manager) DefinitionsForNamespace(ns string) []*Definition {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := m.namespaces[ns]
	defs := make([]*Definition, 0, len(keys))
	for _, k := range keys {
		if d, ok := m.definitions[k]; ok {
			defs = append(defs, d)
		}
	}
	return defs
}

// Namespaces returns a sorted list of all registered namespace names.
func (m *Manager) Namespaces() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ns := make([]string, 0, len(m.namespaces))
	for n := range m.namespaces {
		ns = append(ns, n)
	}
	sort.Strings(ns)
	return ns
}

// ResolveWithDetails returns the full resolution result for a setting key,
// including the definition, effective value, the value at each scope that
// has an override, and the enforcement state. This provides all the
// context a UI needs to render a scope-aware settings form.
func (m *Manager) ResolveWithDetails(ctx context.Context, key string, opts ResolveOpts) (*ResolvedSetting, error) {
	m.mu.RLock()
	def, ok := m.definitions[key]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownKey, key)
	}

	result := &ResolvedSetting{
		Definition:     def,
		EffectiveValue: def.Default,
		CanOverride:    true,
	}

	// Check programmatic enforcement first (highest priority).
	if rule, found := m.findEnforcementRule(ctx, key, opts); found {
		result.EffectiveValue = rule.Value
		result.CanOverride = false
		scope := rule.Scope
		result.EnforcedAt = &scope
		return result, nil
	}

	// Fetch all scoped settings from store.
	settings, err := m.store.ResolveSettings(ctx, key, opts)
	if err != nil {
		return nil, fmt.Errorf("settings: resolve details %q: %w", key, err)
	}

	// Build scope values and walk the cascade.
	for _, s := range settings {
		sv := ScopeValue{
			Scope:     s.Scope,
			ScopeID:   s.ScopeID,
			Value:     s.Value,
			Enforced:  s.Enforced,
			Version:   s.Version,
			UpdatedBy: s.UpdatedBy,
			UpdatedAt: s.UpdatedAt,
		}
		result.ScopeValues = append(result.ScopeValues, sv)
		result.EffectiveValue = s.Value

		if s.Enforced {
			scope := s.Scope
			result.EnforcedAt = &scope
			result.CanOverride = false
			break // enforced: stop cascade
		}
	}

	return result, nil
}

// ResolveAllForNamespace resolves all settings in a namespace with full
// cascade details. This is the primary endpoint for rendering an entire
// plugin's settings page in the dashboard.
func (m *Manager) ResolveAllForNamespace(ctx context.Context, namespace string, opts ResolveOpts) ([]*ResolvedSetting, error) {
	m.mu.RLock()
	keys := m.namespaces[namespace]
	m.mu.RUnlock()

	results := make([]*ResolvedSetting, 0, len(keys))
	for _, key := range keys {
		rs, err := m.ResolveWithDetails(ctx, key, opts)
		if err != nil {
			return nil, err
		}
		results = append(results, rs)
	}

	// Sort by UI order if available, then by key.
	sort.Slice(results, func(i, j int) bool {
		oi, oj := 0, 0
		if results[i].Definition.UI != nil {
			oi = results[i].Definition.UI.Order
		}
		if results[j].Definition.UI != nil {
			oj = results[j].Definition.UI.Order
		}
		if oi != oj {
			return oi < oj
		}
		return results[i].Definition.Key < results[j].Definition.Key
	})

	return results, nil
}

// OnChange registers a listener for changes to a specific key.
// Pass "*" to listen for all setting changes.
func (m *Manager) OnChange(key string, fn ChangeListener) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeners[key] = append(m.listeners[key], fn)
}

// AddEnforcementProvider registers a plugin that provides programmatic
// enforcement rules (e.g., compliance plugin).
func (m *Manager) AddEnforcementProvider(p EnforcementProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enforcers = append(m.enforcers, p)
}

// Store returns the underlying settings store.
func (m *Manager) Store() Store { return m.store }

// isEnforcedAbove checks if any scope higher than the given scope has this
// key enforced. Used during writes to prevent overriding enforced values.
func (m *Manager) isEnforcedAbove(ctx context.Context, key string, scope Scope, appID, orgID string) (bool, *Setting, error) {
	targetPriority := ScopePriority(scope)

	settings, err := m.store.ResolveSettings(ctx, key, ResolveOpts{
		AppID: appID,
		OrgID: orgID,
	})
	if err != nil {
		return false, nil, err
	}

	for _, s := range settings {
		// Only check scopes above the target scope.
		if ScopePriority(s.Scope) < targetPriority && s.Enforced {
			return true, s, nil
		}
	}

	// Also check programmatic enforcers.
	if rule, found := m.findEnforcementRule(ctx, key, ResolveOpts{AppID: appID, OrgID: orgID}); found {
		if ScopePriority(rule.Scope) < targetPriority {
			return true, nil, nil
		}
	}

	return false, nil, nil
}

// findEnforcementRule checks all registered enforcement providers for a
// matching rule. Returns the most specific matching rule.
func (m *Manager) findEnforcementRule(ctx context.Context, key string, opts ResolveOpts) (EnforcementRule, bool) {
	m.mu.RLock()
	enforcers := m.enforcers
	m.mu.RUnlock()

	var bestRule EnforcementRule
	found := false

	for _, ep := range enforcers {
		rules, err := ep.EnforcementRules(ctx)
		if err != nil {
			m.logger.Warn("settings: enforcement provider error",
				log.String("error", err.Error()),
			)
			continue
		}

		for _, rule := range rules {
			if rule.Key != key {
				continue
			}
			// Check if rule matches the current scope context.
			if !ruleMatchesContext(rule, opts) {
				continue
			}
			// Prefer the most specific matching rule.
			if !found || ScopePriority(rule.Scope) > ScopePriority(bestRule.Scope) {
				bestRule = rule
				found = true
			}
		}
	}

	return bestRule, found
}

// ruleMatchesContext checks if an enforcement rule applies to the given resolve context.
func ruleMatchesContext(rule EnforcementRule, opts ResolveOpts) bool {
	switch rule.Scope {
	case ScopeGlobal:
		return true
	case ScopeApp:
		return rule.ScopeID == "" || rule.ScopeID == opts.AppID
	case ScopeOrg:
		return rule.ScopeID == "" || rule.ScopeID == opts.OrgID
	case ScopeUser:
		return rule.ScopeID == "" || rule.ScopeID == opts.UserID
	default:
		return false
	}
}

// emitChange dispatches a change event to registered listeners.
func (m *Manager) emitChange(evt ChangeEvent) {
	m.mu.RLock()
	specific := m.listeners[evt.Key]
	wildcard := m.listeners["*"]
	m.mu.RUnlock()

	for _, fn := range specific {
		fn(evt)
	}
	for _, fn := range wildcard {
		fn(evt)
	}
}

// timeNow is a package-level var to allow testing with a fixed clock.
var timeNow = time.Now
