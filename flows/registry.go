package flows

import (
	"fmt"
	"sync"

	"github.com/xraph/authsome/internal/errs"
)

// Registry manages all available flows in the system.
type Registry struct {
	flows map[string]Flow
	mutex sync.RWMutex
}

// NewRegistry creates a new flow registry.
func NewRegistry() *Registry {
	return &Registry{
		flows: make(map[string]Flow),
	}
}

// Register registers a new flow in the registry.
func (r *Registry) Register(flow Flow) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if flow == nil {
		return errs.BadRequest("flow cannot be nil")
	}

	if flow.ID() == "" {
		return errs.RequiredField("flow ID")
	}

	if _, exists := r.flows[flow.ID()]; exists {
		return fmt.Errorf("flow with ID %s already exists", flow.ID())
	}

	r.flows[flow.ID()] = flow

	return nil
}

// Get retrieves a flow by its ID.
func (r *Registry) Get(id string) (Flow, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	flow, exists := r.flows[id]
	if !exists {
		return nil, fmt.Errorf("flow with ID %s not found", id)
	}

	return flow, nil
}

// List returns all registered flows.
func (r *Registry) List() []Flow {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	flows := make([]Flow, 0, len(r.flows))
	for _, flow := range r.flows {
		flows = append(flows, flow)
	}

	return flows
}

// ListIDs returns all registered flow IDs.
func (r *Registry) ListIDs() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	ids := make([]string, 0, len(r.flows))
	for id := range r.flows {
		ids = append(ids, id)
	}

	return ids
}

// Unregister removes a flow from the registry.
func (r *Registry) Unregister(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.flows[id]; !exists {
		return fmt.Errorf("flow with ID %s not found", id)
	}

	delete(r.flows, id)

	return nil
}

// Clear removes all flows from the registry.
func (r *Registry) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.flows = make(map[string]Flow)
}

// Count returns the number of registered flows.
func (r *Registry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.flows)
}

// Exists checks if a flow with the given ID exists.
func (r *Registry) Exists(id string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.flows[id]

	return exists
}
