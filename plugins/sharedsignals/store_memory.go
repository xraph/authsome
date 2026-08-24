package sharedsignals

import (
	"context"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// MemoryStore is an in-memory Store, used in tests and in standalone mode
// when no database is configured.
type MemoryStore struct {
	mu      sync.RWMutex
	streams map[string]*InboundStream
	links   map[string]*SubjectLink
	events  map[string]*ReceivedEvent
	signals map[string]*Signal
}

// NewMemoryStore builds an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		streams: make(map[string]*InboundStream),
		links:   make(map[string]*SubjectLink),
		events:  make(map[string]*ReceivedEvent),
		signals: make(map[string]*Signal),
	}
}

var _ Store = (*MemoryStore)(nil)

func linkKey(appID id.AppID, envID id.EnvironmentID, issuer, subject string) string {
	return appID.String() + "|" + envID.String() + "|" + issuer + "|" + subject
}

// eventKey is the dedupe identity: RFC 8417 keys a SET's events object by
// event type URI, so a single delivery carries at most one event of a given
// type under one jti but may legitimately carry several different types.
// The key must include event_type or the second event in a multi-event SET
// collides with the first on its very first delivery.
func eventKey(streamID id.SSFStreamID, jti, eventType string) string {
	return streamID.String() + "|" + jti + "|" + eventType
}

func cloneStream(s *InboundStream) *InboundStream {
	out := *s
	out.AllowedEventTypes = append([]string(nil), s.AllowedEventTypes...)
	out.AllowedSubjectFormats = append([]string(nil), s.AllowedSubjectFormats...)
	out.VerifiedDomains = append([]string(nil), s.VerifiedDomains...)
	if s.ActionOverrides != nil {
		out.ActionOverrides = make(map[string]string, len(s.ActionOverrides))
		for k, v := range s.ActionOverrides {
			out.ActionOverrides[k] = v
		}
	}
	if s.LastVerifiedAt != nil {
		t := *s.LastVerifiedAt
		out.LastVerifiedAt = &t
	}
	return &out
}

func (m *MemoryStore) CreateInboundStream(_ context.Context, s *InboundStream) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	m.streams[s.ID.String()] = cloneStream(s)
	return nil
}

func (m *MemoryStore) GetInboundStream(_ context.Context, streamID id.SSFStreamID) (*InboundStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.streams[streamID.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneStream(s), nil
}

func (m *MemoryStore) GetInboundStreamByPushPathHash(_ context.Context, hash string) (*InboundStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, s := range m.streams {
		if s.PushPathHash == hash {
			return cloneStream(s), nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) ListInboundStreams(_ context.Context, appID id.AppID) ([]*InboundStream, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*InboundStream, 0, len(m.streams))
	for _, s := range m.streams {
		if s.AppID == appID {
			out = append(out, cloneStream(s))
		}
	}
	return out, nil
}

func (m *MemoryStore) UpdateInboundStream(_ context.Context, s *InboundStream) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.streams[s.ID.String()]; !ok {
		return ErrNotFound
	}
	s.UpdatedAt = time.Now()
	m.streams[s.ID.String()] = cloneStream(s)
	return nil
}

func (m *MemoryStore) DeleteInboundStream(_ context.Context, streamID id.SSFStreamID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.streams[streamID.String()]; !ok {
		return ErrNotFound
	}
	delete(m.streams, streamID.String())
	return nil
}

func (m *MemoryStore) UpsertSubjectLink(_ context.Context, l *SubjectLink) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if l.CreatedAt.IsZero() {
		l.CreatedAt = now
	}
	l.LastSeenAt = now
	out := *l
	m.links[linkKey(l.AppID, l.EnvID, l.Issuer, l.Subject)] = &out
	return nil
}

func (m *MemoryStore) GetSubjectLink(_ context.Context, appID id.AppID,
	envID id.EnvironmentID, issuer, subject string) (*SubjectLink, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	l, ok := m.links[linkKey(appID, envID, issuer, subject)]
	if !ok {
		return nil, ErrNotFound
	}
	out := *l
	return &out, nil
}

func (m *MemoryStore) InsertReceivedEvent(_ context.Context, e *ReceivedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := eventKey(e.StreamID, e.JTI, e.EventType)
	if _, ok := m.events[k]; ok {
		return ErrDuplicateJTI
	}
	if e.ReceivedAt.IsZero() {
		e.ReceivedAt = time.Now()
	}
	out := *e
	m.events[k] = &out
	return nil
}

func (m *MemoryStore) UpdateReceivedEvent(_ context.Context, e *ReceivedEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := eventKey(e.StreamID, e.JTI, e.EventType)
	if _, ok := m.events[k]; !ok {
		return ErrNotFound
	}
	out := *e
	m.events[k] = &out
	return nil
}

// DeleteReceivedEvent removes a row by ID. Events are keyed in storage by
// (stream_id, jti, event_type), not by ID, so this scans -- MemoryStore
// backs tests and standalone mode, never a production hot path.
func (m *MemoryStore) DeleteReceivedEvent(_ context.Context, eventID id.SSFEventID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, e := range m.events {
		if e.ID == eventID {
			delete(m.events, k)
			return nil
		}
	}
	return ErrNotFound
}

func (m *MemoryStore) CountActionsSince(_ context.Context, streamID id.SSFStreamID,
	since time.Time) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, e := range m.events {
		if e.StreamID == streamID && e.ActionTaken != "" && e.ReceivedAt.After(since) {
			count++
		}
	}
	return count, nil
}

func (m *MemoryStore) CreateSignal(_ context.Context, s *Signal) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	out := *s
	m.signals[s.ID.String()] = &out
	return nil
}

func (m *MemoryStore) ListActiveSignals(_ context.Context, appID id.AppID,
	envID id.EnvironmentID, userID id.UserID, now time.Time) ([]*Signal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Signal, 0, len(m.signals))
	for _, s := range m.signals {
		if s.AppID == appID && s.EnvID == envID && s.UserID == userID && s.ExpiresAt.After(now) {
			c := *s
			out = append(out, &c)
		}
	}
	return out, nil
}
