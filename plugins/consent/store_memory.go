package consent

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when a consent record is not found.
var ErrNotFound = errors.New("consent: not found")

// MemoryStore is an in-memory consent store for development and testing.
type MemoryStore struct {
	mu       sync.RWMutex
	consents []*Consent
}

// NewMemoryStore creates an in-memory consent store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) GrantConsent(_ context.Context, c *Consent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for existing consent with same user+app+purpose
	for i, existing := range s.consents {
		if existing.UserID == c.UserID && existing.AppID == c.AppID && existing.Purpose == c.Purpose {
			s.consents[i] = cloneConsent(c)
			return nil
		}
	}

	if c.ID.IsNil() {
		c.ID = id.NewConsentID()
	}
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now

	s.consents = append(s.consents, cloneConsent(c))
	return nil
}

func (s *MemoryStore) RevokeConsent(_ context.Context, userID id.UserID, appID id.AppID, purpose string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range s.consents {
		if c.UserID != userID || c.AppID != appID || c.Purpose != purpose {
			continue
		}
		now := time.Now()
		c.Granted = false
		c.RevokedAt = &now
		c.UpdatedAt = now
		return nil
	}

	return ErrNotFound
}

func (s *MemoryStore) GetConsent(_ context.Context, userID id.UserID, appID id.AppID, purpose string) (*Consent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.consents {
		if c.UserID == userID && c.AppID == appID && c.Purpose == purpose {
			return cloneConsent(c), nil
		}
	}

	return nil, ErrNotFound
}

func (s *MemoryStore) ListConsents(_ context.Context, q *Query) ([]*Consent, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}

	startIdx := 0
	if q.Cursor != "" {
		if v, err := strconv.Atoi(q.Cursor); err == nil {
			startIdx = v
		}
	}

	var results []*Consent
	idx := 0
	for _, c := range s.consents {
		if q.UserID.Prefix() != "" && c.UserID != q.UserID {
			continue
		}
		if q.AppID.Prefix() != "" && c.AppID != q.AppID {
			continue
		}
		if q.Purpose != "" && c.Purpose != q.Purpose {
			continue
		}

		if idx < startIdx {
			idx++
			continue
		}

		results = append(results, cloneConsent(c))
		idx++

		if len(results) >= limit {
			break
		}
	}

	var cursor string
	if len(results) == limit {
		cursor = strconv.Itoa(idx)
	}

	return results, cursor, nil
}

// cloneConsent copies c so the store never hands out, or retains, a pointer
// into its own storage.
//
// This is not a nicety here. RevokeConsent mutates a stored *Consent in place
// (Granted, RevokedAt, UpdatedAt) while holding the write lock, and GetConsent
// and ListConsents used to return that same pointer. A caller reading Granted
// off the value it got back was reading a field another goroutine's
// RevokeConsent could be writing, with no synchronization between the two,
// which is a data race and also lets a revocation appear or disappear
// mid-decision.
//
// RevokedAt is a *time.Time, so the copy shares that pointer. That is safe as
// written, because every write replaces the pointer rather than writing
// through it, but a caller must still treat what it gets back as read-only.
func cloneConsent(c *Consent) *Consent {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}
