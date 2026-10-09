package bridge

import (
	"context"
	"sync"
)

// MemoryChronicle records audit events in memory. It exists for tests and
// for the standalone examples; production wiring requires a real Chronicle.
type MemoryChronicle struct {
	mu     sync.Mutex
	events []*AuditEvent
	err    error
}

// NewMemoryChronicle returns an empty recorder.
func NewMemoryChronicle() *MemoryChronicle { return &MemoryChronicle{} }

// Record implements Chronicle.
func (m *MemoryChronicle) Record(_ context.Context, event *AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	cp := *event
	if event.Metadata != nil {
		cp.Metadata = make(map[string]string, len(event.Metadata))
		for k, v := range event.Metadata {
			cp.Metadata[k] = v
		}
	}
	m.events = append(m.events, &cp)
	return nil
}

// Events returns a copy of everything recorded so far.
func (m *MemoryChronicle) Events() []*AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*AuditEvent, len(m.events))
	copy(out, m.events)
	return out
}

// Reset forgets every recorded event.
func (m *MemoryChronicle) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = nil
}

// FailWith makes every later Record return err; nil restores success.
func (m *MemoryChronicle) FailWith(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

var _ Chronicle = (*MemoryChronicle)(nil)
