package waitlist

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// MemoryStore is an in-memory waitlist store for development and testing.
type MemoryStore struct {
	mu      sync.RWMutex
	entries []*WaitlistEntry
}

// NewMemoryStore creates an in-memory waitlist store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// Compile-time interface check.
var _ Store = (*MemoryStore)(nil)

func (s *MemoryStore) CreateEntry(_ context.Context, e *WaitlistEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate email in same app.
	for _, existing := range s.entries {
		if existing.AppID == e.AppID && strings.EqualFold(existing.Email, e.Email) {
			return ErrDuplicateEmail
		}
	}

	if e.ID.IsNil() {
		e.ID = id.NewWaitlistID()
	}
	now := time.Now()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
	e.UpdatedAt = now

	s.entries = append(s.entries, cloneEntry(e))
	return nil
}

func (s *MemoryStore) GetEntry(_ context.Context, entryID id.WaitlistID) (*WaitlistEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.entries {
		if e.ID == entryID {
			return cloneEntry(e), nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) GetEntryByEmail(_ context.Context, appID id.AppID, email string) (*WaitlistEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.entries {
		if e.AppID == appID && strings.EqualFold(e.Email, email) {
			return cloneEntry(e), nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) UpdateEntryStatus(_ context.Context, entryID id.WaitlistID, status WaitlistStatus, note string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.entries {
		if e.ID == entryID {
			e.Status = status
			e.Note = note
			e.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrNotFound
}

func (s *MemoryStore) ListEntries(_ context.Context, q *WaitlistQuery) (*WaitlistList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	var filtered []*WaitlistEntry
	pastCursor := q.Cursor == ""

	for _, e := range s.entries {
		if !pastCursor {
			if e.ID.String() == q.Cursor {
				pastCursor = true
			}
			continue
		}

		if q.AppID.Prefix() != "" && e.AppID != q.AppID {
			continue
		}
		if q.Email != "" && !strings.EqualFold(e.Email, q.Email) {
			continue
		}
		if q.Status != "" && e.Status != q.Status {
			continue
		}

		filtered = append(filtered, cloneEntry(e))
		if len(filtered) > limit {
			break
		}
	}

	var cursor string
	if len(filtered) > limit {
		cursor = filtered[limit-1].ID.String()
		filtered = filtered[:limit]
	}

	return &WaitlistList{
		Entries:    filtered,
		Total:      len(filtered),
		NextCursor: cursor,
	}, nil
}

func (s *MemoryStore) CountByStatus(_ context.Context, appID id.AppID) (pending, approved, rejected int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, e := range s.entries {
		if appID.Prefix() != "" && e.AppID != appID {
			continue
		}
		switch e.Status {
		case StatusPending:
			pending++
		case StatusApproved:
			approved++
		case StatusRejected:
			rejected++
		}
	}
	return pending, approved, rejected, nil
}

func (s *MemoryStore) DeleteEntry(_ context.Context, entryID id.WaitlistID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, e := range s.entries {
		if e.ID == entryID {
			s.entries = append(s.entries[:i], s.entries[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

// cloneEntry copies e so the store never hands out, or retains, a pointer
// into its own slice.
//
// UpdateEntryStatus writes Status, Note and UpdatedAt on a stored entry in
// place under the write lock, and GetEntry, GetEntryByEmail and ListEntries
// used to return that same pointer. Reading Status off what you got back was
// therefore a read of a field another goroutine could be writing, unordered
// against it. Status is the field that decides whether someone is through the
// waitlist, so a caller could also rewrite it, or a note, straight into the
// store just by assigning to the value it was handed.
//
// UserID is a *id.UserID whose pointer is replaced rather than written
// through, so a shallow copy suffices today. A slice or map field would need
// copying explicitly, as agentauth's cloneGrant does for Scopes.
func cloneEntry(e *WaitlistEntry) *WaitlistEntry {
	if e == nil {
		return nil
	}
	cp := *e
	return &cp
}
