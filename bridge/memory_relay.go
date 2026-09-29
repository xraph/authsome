package bridge

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrEndpointNotFound is returned by MemoryRelay for an unknown endpoint.
var ErrEndpointNotFound = errors.New("bridge: endpoint not found")

// MemoryEndpoint is one endpoint a MemoryRelay holds.
type MemoryEndpoint struct {
	Input   EndpointInput
	Enabled bool
}

// MemoryRelay is an EventRelay and EndpointRelay that keeps everything in
// memory: the events sent and the endpoints managed. It is for tests and
// for development without a Relay, where it lets webhook registration
// work end to end while delivering nothing.
type MemoryRelay struct {
	mu        sync.Mutex
	seq       int
	events    []*WebhookEvent
	endpoints map[string]*MemoryEndpoint
}

// NewMemoryRelay returns an empty MemoryRelay.
func NewMemoryRelay() *MemoryRelay {
	return &MemoryRelay{endpoints: make(map[string]*MemoryEndpoint)}
}

// Send implements EventRelay by recording the event.
func (m *MemoryRelay) Send(_ context.Context, event *WebhookEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

// RegisterEventTypes implements EventRelay as a no-op.
func (*MemoryRelay) RegisterEventTypes(context.Context, []WebhookDefinition) error { return nil }

// Events returns a copy of every event sent so far.
func (m *MemoryRelay) Events() []*WebhookEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*WebhookEvent, len(m.events))
	copy(out, m.events)
	return out
}

// CreateEndpoint implements EndpointRelay. An empty secret is replaced by
// a generated one, as Relay does.
func (m *MemoryRelay) CreateEndpoint(_ context.Context, in EndpointInput) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	epID := fmt.Sprintf("mep_%d", m.seq)
	if in.Secret == "" {
		in.Secret = fmt.Sprintf("whsec_memory_%d", m.seq)
	}
	m.endpoints[epID] = &MemoryEndpoint{Input: in, Enabled: true}
	return epID, nil
}

// UpdateEndpoint implements EndpointRelay. An empty secret keeps the
// current one.
func (m *MemoryRelay) UpdateEndpoint(_ context.Context, endpointID string, in EndpointInput) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ep, ok := m.endpoints[endpointID]
	if !ok {
		return ErrEndpointNotFound
	}
	if in.Secret == "" {
		in.Secret = ep.Input.Secret
	}
	ep.Input = in
	return nil
}

// SetEndpointEnabled implements EndpointRelay.
func (m *MemoryRelay) SetEndpointEnabled(_ context.Context, endpointID string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	ep, ok := m.endpoints[endpointID]
	if !ok {
		return ErrEndpointNotFound
	}
	ep.Enabled = enabled
	return nil
}

// DeleteEndpoint implements EndpointRelay; an unknown id is not an error.
func (m *MemoryRelay) DeleteEndpoint(_ context.Context, endpointID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.endpoints, endpointID)
	return nil
}

// RotateEndpointSecret implements EndpointRelay.
func (m *MemoryRelay) RotateEndpointSecret(_ context.Context, endpointID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ep, ok := m.endpoints[endpointID]
	if !ok {
		return "", ErrEndpointNotFound
	}
	m.seq++
	ep.Input.Secret = fmt.Sprintf("whsec_memory_rotated_%d", m.seq)
	return ep.Input.Secret, nil
}

// Endpoint returns a copy of the endpoint under id.
func (m *MemoryRelay) Endpoint(endpointID string) (MemoryEndpoint, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ep, ok := m.endpoints[endpointID]
	if !ok {
		return MemoryEndpoint{}, false
	}
	return *ep, true
}

// EndpointCount returns how many endpoints the relay holds.
func (m *MemoryRelay) EndpointCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.endpoints)
}

var (
	_ EventRelay    = (*MemoryRelay)(nil)
	_ EndpointRelay = (*MemoryRelay)(nil)
)
