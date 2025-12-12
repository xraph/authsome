package middleware

import (
	"context"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/forge"
)

// AuthStrategy defines the interface for pluggable authentication strategies
// Each strategy is responsible for:
// 1. Extracting credentials from the request
// 2. Validating those credentials
// 3. Building an AuthContext on success
//
// Strategies are tried in priority order (lower number = higher priority)
// until one successfully authenticates or all fail.
type AuthStrategy interface {
	// ID returns the unique strategy identifier (e.g., "bearer", "apikey", "cookie")
	ID() string

	// Priority determines the order strategies are tried
	// Lower values are tried first. Recommended ranges:
	//   0-9:   Reserved for system
	//   10-19: API keys and machine auth
	//   20-29: Bearer tokens
	//   30-39: Cookie sessions
	//   40-49: Basic auth
	//   50+:   Custom strategies
	Priority() int

	// Extract attempts to extract credentials from the request
	// Returns credentials (strategy-specific type) and whether they were found
	// If no credentials are present for this strategy, return (nil, false)
	Extract(c forge.Context) (credentials interface{}, found bool)

	// Authenticate validates the credentials and returns an AuthContext
	// This method is only called if Extract returned found=true
	// On success, return a fully populated AuthContext
	// On failure, return an error (auth will try next strategy)
	Authenticate(ctx context.Context, credentials interface{}) (*contexts.AuthContext, error)
}

// AuthStrategyRegistry manages registered authentication strategies
type AuthStrategyRegistry struct {
	strategies []AuthStrategy
}

// NewAuthStrategyRegistry creates a new strategy registry
func NewAuthStrategyRegistry() *AuthStrategyRegistry {
	return &AuthStrategyRegistry{
		strategies: make([]AuthStrategy, 0),
	}
}

// Register adds a new authentication strategy
// Strategies are automatically sorted by priority after registration
func (r *AuthStrategyRegistry) Register(strategy AuthStrategy) error {
	// Check for duplicate IDs
	for _, s := range r.strategies {
		if s.ID() == strategy.ID() {
			return &StrategyAlreadyRegisteredError{ID: strategy.ID()}
		}
	}

	r.strategies = append(r.strategies, strategy)
	r.sortByPriority()
	return nil
}

// Get retrieves a strategy by ID
func (r *AuthStrategyRegistry) Get(id string) (AuthStrategy, bool) {
	for _, s := range r.strategies {
		if s.ID() == id {
			return s, true
		}
	}
	return nil, false
}

// List returns all registered strategies in priority order
func (r *AuthStrategyRegistry) List() []AuthStrategy {
	return r.strategies
}

// sortByPriority sorts strategies by priority (lower first)
func (r *AuthStrategyRegistry) sortByPriority() {
	// Simple bubble sort - fine for small number of strategies
	n := len(r.strategies)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if r.strategies[j].Priority() > r.strategies[j+1].Priority() {
				r.strategies[j], r.strategies[j+1] = r.strategies[j+1], r.strategies[j]
			}
		}
	}
}

// StrategyAlreadyRegisteredError is returned when attempting to register a duplicate strategy
type StrategyAlreadyRegisteredError struct {
	ID string
}

func (e *StrategyAlreadyRegisteredError) Error() string {
	return "auth strategy already registered: " + e.ID
}

