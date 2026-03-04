package strategy

import (
	"context"
	"errors"
	"net/http"

	log "github.com/xraph/go-utils/log"
)

// Registry holds registered auth strategies ordered by priority.
// Strategies are evaluated in registration order during authentication.
type Registry struct {
	strategies []entry
	logger     log.Logger
}

type entry struct {
	name     string
	strategy Strategy
	priority int
}

// NewRegistry creates a strategy registry with the given logger.
func NewRegistry(logger log.Logger) *Registry {
	return &Registry{logger: logger}
}

// Register adds a strategy with a given priority. Lower priority values
// are evaluated first.
func (r *Registry) Register(s Strategy, priority int) {
	e := entry{
		name:     s.Name(),
		strategy: s,
		priority: priority,
	}

	// Insert in priority order.
	idx := len(r.strategies)
	for i, existing := range r.strategies {
		if priority < existing.priority {
			idx = i
			break
		}
	}
	r.strategies = append(r.strategies, entry{})
	copy(r.strategies[idx+1:], r.strategies[idx:])
	r.strategies[idx] = e
}

// Authenticate evaluates all registered strategies in priority order.
// Returns the result from the first applicable strategy that succeeds.
func (r *Registry) Authenticate(ctx context.Context, req *http.Request) (*Result, error) {
	for _, e := range r.strategies {
		result, err := e.strategy.Authenticate(ctx, req)
		if err != nil {
			var notApplicable ErrStrategyNotApplicable
			if errors.As(err, &notApplicable) {
				continue
			}
			r.logger.Debug("strategy authentication failed",
				log.String("strategy", e.name),
				log.String("error", err.Error()),
			)
			return nil, err
		}
		return result, nil
	}
	return nil, errors.New("strategy: no applicable authentication strategy found")
}

// Strategies returns the names of all registered strategies in priority order.
func (r *Registry) Strategies() []string {
	names := make([]string, len(r.strategies))
	for i, e := range r.strategies {
		names[i] = e.name
	}
	return names
}

// Get returns a strategy by name.
func (r *Registry) Get(name string) (Strategy, bool) {
	for _, e := range r.strategies {
		if e.name == name {
			return e.strategy, true
		}
	}
	return nil, false
}
