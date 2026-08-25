package sharedsignals

import (
	"context"
	"sort"
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

// GetReceivedEvent loads one audit row by ID and confirms it belongs to
// appID. Events are keyed in storage by (stream_id, jti, event_type), not by
// ID, so this scans -- MemoryStore backs tests and standalone mode, never a
// production hot path.
func (m *MemoryStore) GetReceivedEvent(ctx context.Context, appID id.AppID,
	eventID id.SSFEventID) (*ReceivedEvent, error) {
	m.mu.RLock()
	var found *ReceivedEvent
	for _, e := range m.events {
		if e.ID == eventID {
			c := *e
			found = &c
			break
		}
	}
	m.mu.RUnlock()

	if found == nil {
		return nil, ErrNotFound
	}
	// Resolve the tenant outside the read lock: GetInboundStream takes the
	// same RWMutex, and Go's RLock is not safe to re-enter while a writer
	// is queued behind it.
	if err := streamOwnedBy(ctx, m, appID, found.StreamID); err != nil {
		return nil, err
	}
	return found, nil
}

// ListReceivedEvents returns one stream's audit rows newest first.
func (m *MemoryStore) ListReceivedEvents(ctx context.Context, appID id.AppID,
	f ReceivedEventFilter) ([]*ReceivedEvent, error) {
	// Ownership first, and before any row is read: a stream that is not
	// this caller's must answer ErrNotFound, not an empty list.
	if err := streamOwnedBy(ctx, m, appID, f.StreamID); err != nil {
		return nil, err
	}
	f = f.normalized()

	m.mu.RLock()
	out := make([]*ReceivedEvent, 0, len(m.events))
	for _, e := range m.events {
		if e.StreamID != f.StreamID {
			continue
		}
		if !f.Since.IsZero() && e.ReceivedAt.Before(f.Since) {
			continue
		}
		if !f.Until.IsZero() && !e.ReceivedAt.Before(f.Until) {
			continue
		}
		c := *e
		out = append(out, &c)
	}
	m.mu.RUnlock()

	// Map iteration is random, so the ordering the SQL backends get from
	// the index has to be applied explicitly here or the conformance suite
	// would pass on a backend nobody could actually page through. The ID
	// tie-break keeps rows written in the same instant stable, which the
	// second-granularity timestamps in tests routinely are.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].ReceivedAt.Equal(out[j].ReceivedAt) {
			return out[i].ReceivedAt.After(out[j].ReceivedAt)
		}
		return out[i].ID.String() > out[j].ID.String()
	})
	if len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}

func (m *MemoryStore) CountEventsSince(_ context.Context, streamID id.SSFStreamID,
	since time.Time) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Every recorded event counts, not just the ones that acted -- see
	// Store.CountEventsSince.
	count := 0
	for _, e := range m.events {
		if e.StreamID == streamID && e.ReceivedAt.After(since) {
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
