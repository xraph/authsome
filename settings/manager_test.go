package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	log "github.com/xraph/go-utils/log"
)

// memStore is a minimal in-memory Store for unit tests.
type memStore struct {
	mu       sync.RWMutex
	settings map[string]*Setting // keyed by key|scope|scopeID
}

func newMemStore() *memStore {
	return &memStore{settings: make(map[string]*Setting)}
}

func memKey(key string, scope Scope, scopeID string) string {
	return key + "|" + string(scope) + "|" + scopeID
}

func (s *memStore) GetSetting(_ context.Context, key string, scope Scope, scopeID string) (*Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.settings[memKey(key, scope, scopeID)]
	if !ok {
		return nil, ErrNotFound
	}
	return st, nil
}

func (s *memStore) SetSetting(_ context.Context, st *Setting) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[memKey(st.Key, st.Scope, st.ScopeID)] = st
	return nil
}

func (s *memStore) DeleteSetting(_ context.Context, key string, scope Scope, scopeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.settings, memKey(key, scope, scopeID))
	return nil
}

func (s *memStore) ListSettings(_ context.Context, _ ListOpts) ([]*Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Setting, 0, len(s.settings))
	for _, st := range s.settings {
		result = append(result, st)
	}
	return result, nil
}

func (s *memStore) ResolveSettings(_ context.Context, key string, opts ResolveOpts) ([]*Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Setting
	if st, ok := s.settings[memKey(key, ScopeGlobal, "")]; ok {
		result = append(result, st)
	}
	if opts.AppID != "" {
		if st, ok := s.settings[memKey(key, ScopeApp, opts.AppID)]; ok {
			result = append(result, st)
		}
	}
	if opts.OrgID != "" {
		if st, ok := s.settings[memKey(key, ScopeOrg, opts.OrgID)]; ok {
			result = append(result, st)
		}
	}
	if opts.UserID != "" {
		if st, ok := s.settings[memKey(key, ScopeUser, opts.UserID)]; ok {
			result = append(result, st)
		}
	}
	return result, nil
}

func (s *memStore) BatchResolve(ctx context.Context, keys []string, opts ResolveOpts) (map[string][]*Setting, error) {
	result := make(map[string][]*Setting, len(keys))
	for _, key := range keys {
		resolved, err := s.ResolveSettings(ctx, key, opts)
		if err != nil {
			return nil, err
		}
		if len(resolved) > 0 {
			result[key] = resolved
		}
	}
	return result, nil
}

func (s *memStore) DeleteSettingsByNamespace(_ context.Context, ns string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, st := range s.settings {
		if st.Namespace == ns {
			delete(s.settings, k)
		}
	}
	return nil
}

// ──────────────────────────────────────────────────
// Test helpers
// ──────────────────────────────────────────────────

func newTestManager() *Manager {
	// Fix the clock for deterministic tests.
	timeNow = func() time.Time { return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) }
	store := newMemStore()
	return NewManager(store, log.NewNoopLogger())
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// ──────────────────────────────────────────────────
// Tests
// ──────────────────────────────────────────────────

func TestRegister(t *testing.T) {
	m := newTestManager()

	def := Define("test.key", 42, WithScopes(ScopeGlobal, ScopeApp))
	err := RegisterTyped(m, "test", def)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Duplicate registration should fail.
	err = RegisterTyped(m, "test", def)
	if err == nil {
		t.Fatal("expected duplicate key error")
	}
}

func TestResolve_Default(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42, WithScopes(ScopeGlobal, ScopeApp))
	_ = RegisterTyped(m, "test", def)

	val, err := Get(context.Background(), m, def, ResolveOpts{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

func TestResolve_GlobalOverride(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42, WithScopes(ScopeGlobal, ScopeApp))
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()
	err := m.Set(ctx, "test.key", mustJSON(100), ScopeGlobal, "", "", "", "admin")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := Get(ctx, m, def, ResolveOpts{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100, got %d", val)
	}
}

func TestResolve_Cascade(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42,
		WithScopes(ScopeGlobal, ScopeApp, ScopeOrg, ScopeUser),
	)
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()

	// Set global = 100
	_ = m.Set(ctx, "test.key", mustJSON(100), ScopeGlobal, "", "", "", "admin")

	// Set app = 200
	_ = m.Set(ctx, "test.key", mustJSON(200), ScopeApp, "app1", "app1", "", "admin")

	// Set org = 300
	_ = m.Set(ctx, "test.key", mustJSON(300), ScopeOrg, "org1", "app1", "org1", "admin")

	// Resolve at user level — org override (300) should win.
	val, err := Get(ctx, m, def, ResolveOpts{AppID: "app1", OrgID: "org1", UserID: "user1"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 300 {
		t.Errorf("expected 300 (org override), got %d", val)
	}

	// Resolve at app level — app override (200) should win.
	val, err = Get(ctx, m, def, ResolveOpts{AppID: "app1"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 200 {
		t.Errorf("expected 200 (app override), got %d", val)
	}
}

func TestEnforce(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42,
		WithScopes(ScopeGlobal, ScopeApp, ScopeOrg),
		WithEnforceable(),
	)
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()

	// Enforce at app level = 99
	err := m.Enforce(ctx, "test.key", mustJSON(99), ScopeApp, "app1", "app1", "", "admin")
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}

	// Resolve should return enforced value regardless of org override.
	_ = m.Set(ctx, "test.key", mustJSON(999), ScopeGlobal, "", "", "", "admin")

	val, err := Get(ctx, m, def, ResolveOpts{AppID: "app1", OrgID: "org1"})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 99 {
		t.Errorf("expected enforced 99, got %d", val)
	}
}

func TestEnforce_BlocksLowerWrite(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42,
		WithScopes(ScopeGlobal, ScopeApp, ScopeOrg),
		WithEnforceable(),
	)
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()

	// Enforce at app level.
	_ = m.Enforce(ctx, "test.key", mustJSON(99), ScopeApp, "app1", "app1", "", "admin")

	// Attempt to set at org level should fail.
	err := m.Set(ctx, "test.key", mustJSON(200), ScopeOrg, "org1", "app1", "org1", "admin")
	if err == nil {
		t.Fatal("expected enforcement error for lower scope write")
	}
}

func TestEnforce_NotEnforceable(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42, WithScopes(ScopeGlobal, ScopeApp))
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()
	err := m.Enforce(ctx, "test.key", mustJSON(99), ScopeGlobal, "", "", "", "admin")
	if err == nil {
		t.Fatal("expected ErrNotEnforceable")
	}
}

func TestSet_ScopeNotAllowed(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()
	err := m.Set(ctx, "test.key", mustJSON(99), ScopeApp, "app1", "app1", "", "admin")
	if err == nil {
		t.Fatal("expected ErrScopeNotAllowed")
	}
}

func TestSet_Validation(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42,
		WithScopes(ScopeGlobal),
		WithValidation(func(v json.RawMessage) error {
			var n int
			_ = json.Unmarshal(v, &n)
			if n < 6 || n > 128 {
				return fmt.Errorf("must be 6-128")
			}
			return nil
		}),
	)
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()
	err := m.Set(ctx, "test.key", mustJSON(3), ScopeGlobal, "", "", "", "admin")
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestOnChange(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "test", def)

	var received []ChangeEvent
	m.OnChange("test.key", func(evt ChangeEvent) {
		received = append(received, evt)
	})

	ctx := context.Background()
	_ = m.Set(ctx, "test.key", mustJSON(100), ScopeGlobal, "", "", "", "admin")

	if len(received) != 1 {
		t.Fatalf("expected 1 change event, got %d", len(received))
	}
	if received[0].Key != "test.key" {
		t.Errorf("expected key test.key, got %s", received[0].Key)
	}
}

func TestOnChange_Wildcard(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "test", def)

	var received int
	m.OnChange("*", func(_ ChangeEvent) {
		received++
	})

	ctx := context.Background()
	_ = m.Set(ctx, "test.key", mustJSON(100), ScopeGlobal, "", "", "", "admin")

	if received != 1 {
		t.Fatalf("expected 1 wildcard event, got %d", received)
	}
}

func TestEnforcementProvider(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42,
		WithScopes(ScopeGlobal, ScopeApp),
		WithEnforceable(),
	)
	_ = RegisterTyped(m, "test", def)

	// Register a programmatic enforcement provider.
	m.AddEnforcementProvider(&mockEnforcer{
		rules: []EnforcementRule{
			{
				Key:     "test.key",
				Value:   mustJSON(12),
				Scope:   ScopeGlobal,
				ScopeID: "",
				Reason:  "SOC2 compliance",
				Source:  "compliance",
			},
		},
	})

	ctx := context.Background()

	// Set a different value — the enforcer should override it.
	_ = m.Set(ctx, "test.key", mustJSON(999), ScopeGlobal, "", "", "", "admin")

	val, err := Get(ctx, m, def, ResolveOpts{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// Programmatic enforcement has highest priority.
	if val != 12 {
		t.Errorf("expected enforced 12, got %d", val)
	}
}

func TestUnenforce(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42,
		WithScopes(ScopeGlobal, ScopeApp),
		WithEnforceable(),
	)
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()

	// Enforce, then unenforce.
	_ = m.Enforce(ctx, "test.key", mustJSON(99), ScopeGlobal, "", "", "", "admin")
	err := m.Unenforce(ctx, "test.key", ScopeGlobal, "")
	if err != nil {
		t.Fatalf("Unenforce: %v", err)
	}

	// Setting should still be there but no longer enforced.
	enforced, _, err := m.IsEnforced(ctx, "test.key", ResolveOpts{})
	if err != nil {
		t.Fatalf("IsEnforced: %v", err)
	}
	if enforced {
		t.Error("expected unenforced")
	}
}

func TestDelete(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()
	_ = m.Set(ctx, "test.key", mustJSON(100), ScopeGlobal, "", "", "", "admin")

	err := m.Delete(ctx, "test.key", ScopeGlobal, "")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should fall back to default.
	val, err := Get(ctx, m, def, ResolveOpts{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 42 {
		t.Errorf("expected default 42, got %d", val)
	}
}

func TestBatchResolve(t *testing.T) {
	m := newTestManager()
	def1 := Define("test.key1", 10, WithScopes(ScopeGlobal))
	def2 := Define("test.key2", 20, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "test", def1)
	_ = RegisterTyped(m, "test", def2)

	ctx := context.Background()
	_ = m.Set(ctx, "test.key1", mustJSON(11), ScopeGlobal, "", "", "", "admin")

	result, err := m.BatchResolve(ctx, []string{"test.key1", "test.key2"}, ResolveOpts{})
	if err != nil {
		t.Fatalf("BatchResolve: %v", err)
	}

	// key1 has a store override.
	var v1 int
	_ = json.Unmarshal(result["test.key1"], &v1)
	if v1 != 11 {
		t.Errorf("expected 11 for key1, got %d", v1)
	}

	// key2 has no store override → returns default.
	var v2 int
	_ = json.Unmarshal(result["test.key2"], &v2)
	if v2 != 20 {
		t.Errorf("expected 20 for key2, got %d", v2)
	}
}

func TestDefinitions(t *testing.T) {
	m := newTestManager()
	def := Define("test.key", 42, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "test", def)

	defs := m.Definitions()
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Key != "test.key" {
		t.Errorf("expected key test.key, got %s", defs[0].Key)
	}

	nsDefs := m.DefinitionsForNamespace("test")
	if len(nsDefs) != 1 {
		t.Fatalf("expected 1 namespace definition, got %d", len(nsDefs))
	}
}

func TestTypedDefine_String(t *testing.T) {
	m := newTestManager()
	def := Define("test.greeting", "hello", WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "test", def)

	val, err := Get(context.Background(), m, def, ResolveOpts{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "hello" {
		t.Errorf("expected hello, got %s", val)
	}
}

func TestTypedDefine_Bool(t *testing.T) {
	m := newTestManager()
	def := Define("test.flag", false, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()
	_ = m.Set(ctx, "test.flag", mustJSON(true), ScopeGlobal, "", "", "", "admin")

	val, err := Get(ctx, m, def, ResolveOpts{})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !val {
		t.Error("expected true")
	}
}

// ──────────────────────────────────────────────────
// New method tests (Phase 2)
// ──────────────────────────────────────────────────

func TestNamespaces(t *testing.T) {
	m := newTestManager()

	// Register definitions in multiple namespaces.
	def1 := Define("ns1.key", 1, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "zulu", def1)

	def2 := Define("ns2.key", 2, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "alpha", def2)

	def3 := Define("ns3.key", 3, WithScopes(ScopeGlobal))
	_ = RegisterTyped(m, "mike", def3)

	ns := m.Namespaces()
	if len(ns) != 3 {
		t.Fatalf("expected 3 namespaces, got %d", len(ns))
	}
	// Should be sorted.
	if ns[0] != "alpha" || ns[1] != "mike" || ns[2] != "zulu" {
		t.Errorf("expected sorted [alpha mike zulu], got %v", ns)
	}
}

func TestResolveWithDetails_Default(t *testing.T) {
	m := newTestManager()
	def := Define("test.detail", 42, WithScopes(ScopeGlobal, ScopeApp))
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()
	rs, err := m.ResolveWithDetails(ctx, "test.detail", ResolveOpts{})
	if err != nil {
		t.Fatalf("ResolveWithDetails: %v", err)
	}

	if rs.Definition.Key != "test.detail" {
		t.Errorf("expected key test.detail, got %s", rs.Definition.Key)
	}

	var val int
	_ = json.Unmarshal(rs.EffectiveValue, &val)
	if val != 42 {
		t.Errorf("expected effective value 42, got %d", val)
	}
	if len(rs.ScopeValues) != 0 {
		t.Errorf("expected 0 scope values (no overrides), got %d", len(rs.ScopeValues))
	}
	if !rs.CanOverride {
		t.Error("expected CanOverride=true")
	}
	if rs.EnforcedAt != nil {
		t.Error("expected EnforcedAt=nil")
	}
}

func TestResolveWithDetails_WithOverrides(t *testing.T) {
	m := newTestManager()
	def := Define("test.detail", 42,
		WithScopes(ScopeGlobal, ScopeApp, ScopeOrg),
	)
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()
	_ = m.Set(ctx, "test.detail", mustJSON(100), ScopeGlobal, "", "", "", "admin")
	_ = m.Set(ctx, "test.detail", mustJSON(200), ScopeApp, "app1", "app1", "", "admin")

	rs, err := m.ResolveWithDetails(ctx, "test.detail", ResolveOpts{AppID: "app1"})
	if err != nil {
		t.Fatalf("ResolveWithDetails: %v", err)
	}

	// Should have 2 scope values.
	if len(rs.ScopeValues) != 2 {
		t.Fatalf("expected 2 scope values, got %d", len(rs.ScopeValues))
	}
	if rs.ScopeValues[0].Scope != ScopeGlobal {
		t.Errorf("expected first scope global, got %s", rs.ScopeValues[0].Scope)
	}
	if rs.ScopeValues[1].Scope != ScopeApp {
		t.Errorf("expected second scope app, got %s", rs.ScopeValues[1].Scope)
	}

	// Effective value should be 200 (app override wins).
	var val int
	_ = json.Unmarshal(rs.EffectiveValue, &val)
	if val != 200 {
		t.Errorf("expected effective value 200, got %d", val)
	}
	if !rs.CanOverride {
		t.Error("expected CanOverride=true")
	}
}

func TestResolveWithDetails_Enforced(t *testing.T) {
	m := newTestManager()
	def := Define("test.detail", 42,
		WithScopes(ScopeGlobal, ScopeApp, ScopeOrg),
		WithEnforceable(),
	)
	_ = RegisterTyped(m, "test", def)

	ctx := context.Background()
	// Enforce at global level.
	_ = m.Enforce(ctx, "test.detail", mustJSON(99), ScopeGlobal, "", "", "", "admin")

	rs, err := m.ResolveWithDetails(ctx, "test.detail", ResolveOpts{AppID: "app1"})
	if err != nil {
		t.Fatalf("ResolveWithDetails: %v", err)
	}

	var val int
	_ = json.Unmarshal(rs.EffectiveValue, &val)
	if val != 99 {
		t.Errorf("expected enforced value 99, got %d", val)
	}
	if rs.CanOverride {
		t.Error("expected CanOverride=false when enforced")
	}
	if rs.EnforcedAt == nil || *rs.EnforcedAt != ScopeGlobal {
		t.Error("expected EnforcedAt=global")
	}
}

func TestResolveWithDetails_ProgrammaticEnforcement(t *testing.T) {
	m := newTestManager()
	def := Define("test.detail", 42,
		WithScopes(ScopeGlobal, ScopeApp),
		WithEnforceable(),
	)
	_ = RegisterTyped(m, "test", def)

	m.AddEnforcementProvider(&mockEnforcer{
		rules: []EnforcementRule{
			{Key: "test.detail", Value: mustJSON(77), Scope: ScopeGlobal, Reason: "compliance"},
		},
	})

	ctx := context.Background()
	rs, err := m.ResolveWithDetails(ctx, "test.detail", ResolveOpts{})
	if err != nil {
		t.Fatalf("ResolveWithDetails: %v", err)
	}

	var val int
	_ = json.Unmarshal(rs.EffectiveValue, &val)
	if val != 77 {
		t.Errorf("expected programmatic enforced value 77, got %d", val)
	}
	if rs.CanOverride {
		t.Error("expected CanOverride=false for programmatic enforcement")
	}
	if rs.EnforcedAt == nil || *rs.EnforcedAt != ScopeGlobal {
		t.Error("expected EnforcedAt=global")
	}
}

func TestResolveAllForNamespace(t *testing.T) {
	m := newTestManager()
	def1 := Define("ns.key1", 10,
		WithScopes(ScopeGlobal),
		WithOrder(20),
	)
	def2 := Define("ns.key2", 20,
		WithScopes(ScopeGlobal, ScopeApp),
		WithOrder(10),
	)
	_ = RegisterTyped(m, "mynamespace", def1)
	_ = RegisterTyped(m, "mynamespace", def2)

	ctx := context.Background()
	_ = m.Set(ctx, "ns.key2", mustJSON(30), ScopeGlobal, "", "", "", "admin")

	results, err := m.ResolveAllForNamespace(ctx, "mynamespace", ResolveOpts{})
	if err != nil {
		t.Fatalf("ResolveAllForNamespace: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Should be sorted by UI order: key2 (order=10) before key1 (order=20).
	if results[0].Definition.Key != "ns.key2" {
		t.Errorf("expected first result to be ns.key2 (order=10), got %s", results[0].Definition.Key)
	}
	if results[1].Definition.Key != "ns.key1" {
		t.Errorf("expected second result to be ns.key1 (order=20), got %s", results[1].Definition.Key)
	}

	// key2 should have the global override.
	var val int
	_ = json.Unmarshal(results[0].EffectiveValue, &val)
	if val != 30 {
		t.Errorf("expected key2 effective value 30, got %d", val)
	}

	// key1 should have default.
	_ = json.Unmarshal(results[1].EffectiveValue, &val)
	if val != 10 {
		t.Errorf("expected key1 effective value 10, got %d", val)
	}
}

func TestResolveAllForNamespace_Empty(t *testing.T) {
	m := newTestManager()
	results, err := m.ResolveAllForNamespace(context.Background(), "nonexistent", ResolveOpts{})
	if err != nil {
		t.Fatalf("ResolveAllForNamespace: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for nonexistent namespace, got %d", len(results))
	}
}

// mockEnforcer implements EnforcementProvider for testing.
type mockEnforcer struct {
	rules []EnforcementRule
}

func (e *mockEnforcer) EnforcementRules(_ context.Context) ([]EnforcementRule, error) {
	return e.rules, nil
}
