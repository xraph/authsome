package plugins

import (
	"testing"

	"github.com/xraph/authsome/core"
	"github.com/xraph/authsome/core/hooks"
	"github.com/xraph/authsome/core/registry"
	"github.com/xraph/forge"
)

// Mock plugin for testing
type mockPlugin struct {
	id           string
	dependencies []string
}

func (m *mockPlugin) ID() string                                           { return m.id }
func (m *mockPlugin) Init(auth core.Authsome) error                        { return nil }
func (m *mockPlugin) RegisterRoutes(router forge.Router) error             { return nil }
func (m *mockPlugin) RegisterHooks(hooks *hooks.HookRegistry) error        { return nil }
func (m *mockPlugin) RegisterServiceDecorators(reg *registry.ServiceRegistry) error {
	return nil
}
func (m *mockPlugin) Migrate() error { return nil }

// Implement PluginWithDependencies if dependencies are provided
func (m *mockPlugin) Dependencies() []string {
	return m.dependencies
}

func TestListSorted_NoDependencies(t *testing.T) {
	registry := NewRegistry()

	// Register plugins without dependencies
	p1 := &mockPlugin{id: "plugin1"}
	p2 := &mockPlugin{id: "plugin2"}
	p3 := &mockPlugin{id: "plugin3"}

	if err := registry.Register(p1); err != nil {
		t.Fatalf("failed to register plugin1: %v", err)
	}
	if err := registry.Register(p2); err != nil {
		t.Fatalf("failed to register plugin2: %v", err)
	}
	if err := registry.Register(p3); err != nil {
		t.Fatalf("failed to register plugin3: %v", err)
	}

	// Sort should succeed
	sorted, err := registry.(*Registry).ListSorted()
	if err != nil {
		t.Fatalf("ListSorted failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(sorted))
	}
}

func TestListSorted_SimpleDependency(t *testing.T) {
	registry := NewRegistry()

	// plugin2 depends on plugin1
	p1 := &mockPlugin{id: "plugin1"}
	p2 := &mockPlugin{id: "plugin2", dependencies: []string{"plugin1"}}

	if err := registry.Register(p2); err != nil {
		t.Fatalf("failed to register plugin2: %v", err)
	}
	if err := registry.Register(p1); err != nil {
		t.Fatalf("failed to register plugin1: %v", err)
	}

	// Sort should succeed and plugin1 should come before plugin2
	sorted, err := registry.(*Registry).ListSorted()
	if err != nil {
		t.Fatalf("ListSorted failed: %v", err)
	}

	if len(sorted) != 2 {
		t.Fatalf("expected 2 plugins, got %d", len(sorted))
	}

	// Verify order
	if sorted[0].ID() != "plugin1" {
		t.Errorf("expected plugin1 first, got %s", sorted[0].ID())
	}
	if sorted[1].ID() != "plugin2" {
		t.Errorf("expected plugin2 second, got %s", sorted[1].ID())
	}
}

func TestListSorted_ChainDependency(t *testing.T) {
	registry := NewRegistry()

	// plugin3 -> plugin2 -> plugin1 (chain dependency)
	p1 := &mockPlugin{id: "plugin1"}
	p2 := &mockPlugin{id: "plugin2", dependencies: []string{"plugin1"}}
	p3 := &mockPlugin{id: "plugin3", dependencies: []string{"plugin2"}}

	// Register in reverse order
	if err := registry.Register(p3); err != nil {
		t.Fatalf("failed to register plugin3: %v", err)
	}
	if err := registry.Register(p2); err != nil {
		t.Fatalf("failed to register plugin2: %v", err)
	}
	if err := registry.Register(p1); err != nil {
		t.Fatalf("failed to register plugin1: %v", err)
	}

	// Sort should succeed with correct order
	sorted, err := registry.(*Registry).ListSorted()
	if err != nil {
		t.Fatalf("ListSorted failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(sorted))
	}

	// Verify order: plugin1, plugin2, plugin3
	if sorted[0].ID() != "plugin1" {
		t.Errorf("expected plugin1 first, got %s", sorted[0].ID())
	}
	if sorted[1].ID() != "plugin2" {
		t.Errorf("expected plugin2 second, got %s", sorted[1].ID())
	}
	if sorted[2].ID() != "plugin3" {
		t.Errorf("expected plugin3 third, got %s", sorted[2].ID())
	}
}

func TestListSorted_MultipleDependencies(t *testing.T) {
	registry := NewRegistry()

	// plugin3 depends on both plugin1 and plugin2
	p1 := &mockPlugin{id: "plugin1"}
	p2 := &mockPlugin{id: "plugin2"}
	p3 := &mockPlugin{id: "plugin3", dependencies: []string{"plugin1", "plugin2"}}

	if err := registry.Register(p3); err != nil {
		t.Fatalf("failed to register plugin3: %v", err)
	}
	if err := registry.Register(p1); err != nil {
		t.Fatalf("failed to register plugin1: %v", err)
	}
	if err := registry.Register(p2); err != nil {
		t.Fatalf("failed to register plugin2: %v", err)
	}

	sorted, err := registry.(*Registry).ListSorted()
	if err != nil {
		t.Fatalf("ListSorted failed: %v", err)
	}

	if len(sorted) != 3 {
		t.Fatalf("expected 3 plugins, got %d", len(sorted))
	}

	// plugin3 must be last
	if sorted[2].ID() != "plugin3" {
		t.Errorf("expected plugin3 last, got %s", sorted[2].ID())
	}

	// plugin1 and plugin2 must be before plugin3
	foundPlugin1 := false
	foundPlugin2 := false
	for i := 0; i < 2; i++ {
		if sorted[i].ID() == "plugin1" {
			foundPlugin1 = true
		}
		if sorted[i].ID() == "plugin2" {
			foundPlugin2 = true
		}
	}
	if !foundPlugin1 || !foundPlugin2 {
		t.Error("plugin1 and plugin2 must be before plugin3")
	}
}

func TestListSorted_CircularDependency(t *testing.T) {
	registry := NewRegistry()

	// Create circular dependency: plugin1 -> plugin2 -> plugin1
	p1 := &mockPlugin{id: "plugin1", dependencies: []string{"plugin2"}}
	p2 := &mockPlugin{id: "plugin2", dependencies: []string{"plugin1"}}

	if err := registry.Register(p1); err != nil {
		t.Fatalf("failed to register plugin1: %v", err)
	}
	if err := registry.Register(p2); err != nil {
		t.Fatalf("failed to register plugin2: %v", err)
	}

	// Should detect circular dependency
	_, err := registry.(*Registry).ListSorted()
	if err == nil {
		t.Fatal("expected circular dependency error, got nil")
	}

	if err.Error() != "circular dependency detected among plugins: plugin1, plugin2" &&
		err.Error() != "circular dependency detected among plugins: plugin2, plugin1" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestListSorted_MissingDependency(t *testing.T) {
	registry := NewRegistry()

	// plugin2 depends on plugin1, but plugin1 is not registered
	p2 := &mockPlugin{id: "plugin2", dependencies: []string{"plugin1"}}

	if err := registry.Register(p2); err != nil {
		t.Fatalf("failed to register plugin2: %v", err)
	}

	// Should detect missing dependency
	_, err := registry.(*Registry).ListSorted()
	if err == nil {
		t.Fatal("expected missing dependency error, got nil")
	}

	expectedError := "plugin 'plugin2' depends on 'plugin1' which is not registered"
	if err.Error() != expectedError {
		t.Errorf("expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestValidateDependencies_Success(t *testing.T) {
	registry := NewRegistry()

	p1 := &mockPlugin{id: "plugin1"}
	p2 := &mockPlugin{id: "plugin2", dependencies: []string{"plugin1"}}

	if err := registry.Register(p1); err != nil {
		t.Fatalf("failed to register plugin1: %v", err)
	}
	if err := registry.Register(p2); err != nil {
		t.Fatalf("failed to register plugin2: %v", err)
	}

	if err := registry.(*Registry).ValidateDependencies(); err != nil {
		t.Fatalf("ValidateDependencies failed: %v", err)
	}
}

func TestValidateDependencies_Failure(t *testing.T) {
	registry := NewRegistry()

	// Create circular dependency
	p1 := &mockPlugin{id: "plugin1", dependencies: []string{"plugin2"}}
	p2 := &mockPlugin{id: "plugin2", dependencies: []string{"plugin1"}}

	if err := registry.Register(p1); err != nil {
		t.Fatalf("failed to register plugin1: %v", err)
	}
	if err := registry.Register(p2); err != nil {
		t.Fatalf("failed to register plugin2: %v", err)
	}

	if err := registry.(*Registry).ValidateDependencies(); err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestListSorted_ComplexDAG(t *testing.T) {
	registry := NewRegistry()

	// Create a complex DAG:
	//     p1    p2
	//      \   /  \
	//       p3     p4
	//         \   /
	//          p5
	p1 := &mockPlugin{id: "p1"}
	p2 := &mockPlugin{id: "p2"}
	p3 := &mockPlugin{id: "p3", dependencies: []string{"p1", "p2"}}
	p4 := &mockPlugin{id: "p4", dependencies: []string{"p2"}}
	p5 := &mockPlugin{id: "p5", dependencies: []string{"p3", "p4"}}

	// Register in random order
	if err := registry.Register(p3); err != nil {
		t.Fatalf("failed to register p3: %v", err)
	}
	if err := registry.Register(p5); err != nil {
		t.Fatalf("failed to register p5: %v", err)
	}
	if err := registry.Register(p1); err != nil {
		t.Fatalf("failed to register p1: %v", err)
	}
	if err := registry.Register(p4); err != nil {
		t.Fatalf("failed to register p4: %v", err)
	}
	if err := registry.Register(p2); err != nil {
		t.Fatalf("failed to register p2: %v", err)
	}

	sorted, err := registry.(*Registry).ListSorted()
	if err != nil {
		t.Fatalf("ListSorted failed: %v", err)
	}

	if len(sorted) != 5 {
		t.Fatalf("expected 5 plugins, got %d", len(sorted))
	}

	// Create position map
	position := make(map[string]int)
	for i, p := range sorted {
		position[p.ID()] = i
	}

	// Verify dependencies: each plugin must come after its dependencies
	if position["p3"] <= position["p1"] {
		t.Error("p3 must come after p1")
	}
	if position["p3"] <= position["p2"] {
		t.Error("p3 must come after p2")
	}
	if position["p4"] <= position["p2"] {
		t.Error("p4 must come after p2")
	}
	if position["p5"] <= position["p3"] {
		t.Error("p5 must come after p3")
	}
	if position["p5"] <= position["p4"] {
		t.Error("p5 must come after p4")
	}
}

