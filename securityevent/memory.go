package securityevent

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// MemoryStore is an in-memory security event store for development and testing.
type MemoryStore struct {
	mu     sync.RWMutex
	events []*Event
}

// NewMemoryStore creates an in-memory security event store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) RecordSecurityEvent(_ context.Context, event *Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if event.ID == "" {
		event.ID = id.NewUserID().String() // reuse for unique ID generation
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	s.events = append(s.events, event)
	return nil
}

func (s *MemoryStore) QuerySecurityEvents(_ context.Context, q *Query) ([]*Event, string, error) {
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

	var results []*Event
	idx := 0
	for _, e := range s.events {
		// Filter by app ID
		if q.AppID.Prefix() != "" && e.AppID != q.AppID {
			continue
		}
		// Filter by user ID
		if q.UserID.Prefix() != "" && e.UserID != q.UserID {
			continue
		}
		// Filter by action
		if q.Action != "" && e.Action != q.Action {
			continue
		}
		// Filter by time range
		if !q.Since.IsZero() && e.CreatedAt.Before(q.Since) {
			continue
		}
		if !q.Until.IsZero() && e.CreatedAt.After(q.Until) {
			continue
		}

		if idx < startIdx {
			idx++
			continue
		}

		results = append(results, e)
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
