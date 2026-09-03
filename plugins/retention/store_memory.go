package retention

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// MemoryStore is a process-local Store, used in tests and when no database is
// configured. Nothing here survives a restart, which for this plugin means a
// pending backlog is lost rather than delayed.
type MemoryStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
	keys map[string]string // idempotency key -> job id
	refs map[string]*ContactRef
}

// NewMemoryStore builds an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		jobs: make(map[string]*Job),
		keys: make(map[string]string),
		refs: make(map[string]*ContactRef),
	}
}

var _ Store = (*MemoryStore)(nil)

func refKey(appID id.AppID, envID id.EnvironmentID, userID id.UserID, provider string) string {
	return appID.String() + "|" + envID.String() + "|" + userID.String() + "|" + provider
}

func cloneJob(j *Job) *Job {
	out := *j
	out.Payload = make(map[string]string, len(j.Payload))
	for k, v := range j.Payload {
		out.Payload[k] = v
	}
	return &out
}

func (s *MemoryStore) Enqueue(_ context.Context, j *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j.IdempotencyKey != "" {
		if _, dupe := s.keys[j.IdempotencyKey]; dupe {
			return nil
		}
		s.keys[j.IdempotencyKey] = j.ID.String()
	}
	if j.State == "" {
		j.State = StatePending
	}
	s.jobs[j.ID.String()] = cloneJob(j)
	return nil
}

func (s *MemoryStore) ClaimDue(_ context.Context, limit int, lease time.Duration, now time.Time) ([]*Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var due []*Job
	for _, j := range s.jobs {
		switch j.State {
		case StatePending:
			if !j.NextAttemptAt.After(now) {
				due = append(due, j)
			}
		case StateInFlight:
			if j.InFlightUntil.Before(now) {
				due = append(due, j)
			}
		}
	}
	sort.Slice(due, func(a, b int) bool {
		if due[a].NextAttemptAt.Equal(due[b].NextAttemptAt) {
			return due[a].CreatedAt.Before(due[b].CreatedAt)
		}
		return due[a].NextAttemptAt.Before(due[b].NextAttemptAt)
	})
	if limit > 0 && len(due) > limit {
		due = due[:limit]
	}

	out := make([]*Job, 0, len(due))
	for _, j := range due {
		j.State = StateInFlight
		j.InFlightUntil = now.Add(lease)
		out = append(out, cloneJob(j))
	}
	return out, nil
}

func (s *MemoryStore) set(jobID id.RetentionJobID, fn func(*Job)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[jobID.String()]
	if !ok {
		return ErrNotFound
	}
	fn(j)
	return nil
}

func (s *MemoryStore) MarkDone(_ context.Context, jobID id.RetentionJobID, _ time.Time) error {
	return s.set(jobID, func(j *Job) { j.State = StateDone; j.LastError = "" })
}

func (s *MemoryStore) MarkRetry(_ context.Context, jobID id.RetentionJobID, next time.Time, lastErr string) error {
	return s.set(jobID, func(j *Job) {
		j.State = StatePending
		j.Attempts++
		j.NextAttemptAt = next
		j.InFlightUntil = time.Time{}
		j.LastError = lastErr
	})
}

// MarkDeferred returns the job to pending at next without spending an
// attempt. See the Store interface for why that distinction matters.
func (s *MemoryStore) MarkDeferred(_ context.Context, jobID id.RetentionJobID, next time.Time, reason string) error {
	return s.set(jobID, func(j *Job) {
		j.State = StatePending
		j.NextAttemptAt = next
		j.InFlightUntil = time.Time{}
		j.LastError = reason
	})
}

func (s *MemoryStore) MarkDead(_ context.Context, jobID id.RetentionJobID, lastErr string) error {
	return s.set(jobID, func(j *Job) { j.State = StateDead; j.LastError = lastErr })
}

func (s *MemoryStore) MarkSuppressed(_ context.Context, jobID id.RetentionJobID, reason string) error {
	return s.set(jobID, func(j *Job) { j.State = StateSuppressed; j.LastError = reason })
}

func (s *MemoryStore) GetJob(_ context.Context, jobID id.RetentionJobID) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[jobID.String()]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneJob(j), nil
}

func (s *MemoryStore) ListDead(_ context.Context, appID id.AppID, limit int) ([]*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Job
	for _, j := range s.jobs {
		if j.State == StateDead && j.AppID.String() == appID.String() {
			out = append(out, cloneJob(j))
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].CreatedAt.After(out[b].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *MemoryStore) GetRef(_ context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) (*ContactRef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.refs[refKey(appID, envID, userID, provider)]
	if !ok {
		return nil, ErrNotFound
	}
	out := *r
	return &out, nil
}

func (s *MemoryStore) PutRef(_ context.Context, r *ContactRef) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := *r
	s.refs[refKey(r.AppID, r.EnvID, r.UserID, r.Provider)] = &out
	return nil
}

func (s *MemoryStore) DeleteRef(_ context.Context, appID id.AppID, envID id.EnvironmentID,
	userID id.UserID, provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.refs, refKey(appID, envID, userID, provider))
	return nil
}

// ListRefsForUser returns every ref held for the user across all apps and
// providers. Deliberately unscoped by app: a data-subject export covers the
// person, not one app's view of them.
func (s *MemoryStore) ListRefsForUser(_ context.Context, userID id.UserID) ([]*ContactRef, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*ContactRef, 0)
	for _, r := range s.refs {
		if r.UserID.String() == userID.String() {
			cp := *r
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Provider < out[b].Provider })
	return out, nil
}
