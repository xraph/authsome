package middleware

import (
	"context"
	"testing"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/forge"
)

// mockStrategy is a test authentication strategy
type mockStrategy struct {
	id            string
	priority      int
	shouldExtract bool
	shouldAuth    bool
	extractValue  interface{}
}

func (m *mockStrategy) ID() string {
	return m.id
}

func (m *mockStrategy) Priority() int {
	return m.priority
}

func (m *mockStrategy) Extract(c forge.Context) (interface{}, bool) {
	if m.shouldExtract {
		return m.extractValue, true
	}
	return nil, false
}

func (m *mockStrategy) Authenticate(ctx context.Context, credentials interface{}) (*contexts.AuthContext, error) {
	if m.shouldAuth {
		return &contexts.AuthContext{
			Method:          contexts.AuthMethodSession,
			IsAuthenticated: true,
		}, nil
	}
	return nil, &AuthStrategyError{Strategy: m.id, Message: "auth failed"}
}

func TestAuthStrategyRegistry_Register(t *testing.T) {
	registry := NewAuthStrategyRegistry()

	strategy1 := &mockStrategy{id: "strategy1", priority: 10}
	strategy2 := &mockStrategy{id: "strategy2", priority: 20}

	// Test successful registration
	err := registry.Register(strategy1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	err = registry.Register(strategy2)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test duplicate registration
	err = registry.Register(strategy1)
	if err == nil {
		t.Fatal("Expected error for duplicate registration")
	}

	// Verify registry contains strategies
	strategies := registry.List()
	if len(strategies) != 2 {
		t.Fatalf("Expected 2 strategies, got %d", len(strategies))
	}
}

func TestAuthStrategyRegistry_Get(t *testing.T) {
	registry := NewAuthStrategyRegistry()

	strategy := &mockStrategy{id: "test-strategy", priority: 10}
	_ = registry.Register(strategy)

	// Test successful retrieval
	found, ok := registry.Get("test-strategy")
	if !ok {
		t.Fatal("Expected to find strategy")
	}
	if found.ID() != "test-strategy" {
		t.Fatalf("Expected strategy ID 'test-strategy', got %s", found.ID())
	}

	// Test non-existent strategy
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Fatal("Expected not to find nonexistent strategy")
	}
}

func TestAuthStrategyRegistry_PrioritySorting(t *testing.T) {
	registry := NewAuthStrategyRegistry()

	// Register in reverse priority order
	_ = registry.Register(&mockStrategy{id: "low", priority: 30})
	_ = registry.Register(&mockStrategy{id: "high", priority: 10})
	_ = registry.Register(&mockStrategy{id: "medium", priority: 20})

	// Verify sorted by priority (low to high)
	strategies := registry.List()
	if len(strategies) != 3 {
		t.Fatalf("Expected 3 strategies, got %d", len(strategies))
	}

	expectedOrder := []string{"high", "medium", "low"}
	for i, expected := range expectedOrder {
		if strategies[i].ID() != expected {
			t.Errorf("Expected strategy %d to be %s, got %s", i, expected, strategies[i].ID())
		}
	}
}

func TestAuthStrategyError(t *testing.T) {
	// Test error without wrapped error
	err := &AuthStrategyError{
		Strategy: "test",
		Message:  "test message",
	}
	expected := "test: test message"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}

	// Test error with wrapped error
	innerErr := &AuthStrategyError{Strategy: "inner", Message: "inner error"}
	err = &AuthStrategyError{
		Strategy: "outer",
		Message:  "outer error",
		Err:      innerErr,
	}
	if err.Unwrap() != innerErr {
		t.Error("Expected Unwrap to return inner error")
	}
}
