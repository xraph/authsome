package mfa

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrEnrollmentNotFound is returned when an MFA enrollment is not found.
var ErrEnrollmentNotFound = errors.New("mfa: enrollment not found")

// MemoryStore is an in-memory Store for testing.
type MemoryStore struct {
	mu            sync.RWMutex
	enrollments   map[id.MFAID]*Enrollment
	recoveryCodes map[id.RecoveryCodeID]*RecoveryCode
}

// NewMemoryStore creates a new in-memory MFA enrollment store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		enrollments:   make(map[id.MFAID]*Enrollment),
		recoveryCodes: make(map[id.RecoveryCodeID]*RecoveryCode),
	}
}

var _ Store = (*MemoryStore)(nil)

// CreateEnrollment stores a new MFA enrollment.
func (s *MemoryStore) CreateEnrollment(_ context.Context, e *Enrollment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enrollments[e.ID] = cloneEnrollment(e)
	return nil
}

// GetEnrollment finds an enrollment by user ID and method.
func (s *MemoryStore) GetEnrollment(_ context.Context, userID id.UserID, method string) (*Enrollment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.enrollments {
		if e.UserID == userID && e.Method == method {
			return cloneEnrollment(e), nil
		}
	}
	return nil, ErrEnrollmentNotFound
}

// GetEnrollmentByID finds an enrollment by its ID.
func (s *MemoryStore) GetEnrollmentByID(_ context.Context, mfaID id.MFAID) (*Enrollment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.enrollments[mfaID]
	if !ok {
		return nil, ErrEnrollmentNotFound
	}
	return cloneEnrollment(e), nil
}

// UpdateEnrollment updates an existing enrollment.
func (s *MemoryStore) UpdateEnrollment(_ context.Context, e *Enrollment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.enrollments[e.ID]; !ok {
		return ErrEnrollmentNotFound
	}
	s.enrollments[e.ID] = cloneEnrollment(e)
	return nil
}

// DeleteEnrollment removes an enrollment by ID.
func (s *MemoryStore) DeleteEnrollment(_ context.Context, mfaID id.MFAID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.enrollments[mfaID]; !ok {
		return ErrEnrollmentNotFound
	}
	delete(s.enrollments, mfaID)
	return nil
}

// ListEnrollments returns all enrollments for a user.
func (s *MemoryStore) ListEnrollments(_ context.Context, userID id.UserID) ([]*Enrollment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Enrollment
	for _, e := range s.enrollments {
		if e.UserID == userID {
			result = append(result, cloneEnrollment(e))
		}
	}
	return result, nil
}

// ──────────────────────────────────────────────────
// Recovery code methods
// ──────────────────────────────────────────────────

// CreateRecoveryCodes stores a batch of recovery codes.
func (s *MemoryStore) CreateRecoveryCodes(_ context.Context, codes []*RecoveryCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range codes {
		s.recoveryCodes[c.ID] = cloneRecoveryCode(c)
	}
	return nil
}

// GetRecoveryCodes returns all recovery codes for a user.
func (s *MemoryStore) GetRecoveryCodes(_ context.Context, userID id.UserID) ([]*RecoveryCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*RecoveryCode
	for _, c := range s.recoveryCodes {
		if c.UserID == userID {
			result = append(result, cloneRecoveryCode(c))
		}
	}
	return result, nil
}

// ConsumeRecoveryCode marks a recovery code as used.
func (s *MemoryStore) ConsumeRecoveryCode(_ context.Context, codeID id.RecoveryCodeID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.recoveryCodes[codeID]
	if !ok {
		return ErrEnrollmentNotFound
	}
	// Consume only an unspent code. The SQL and mongo backends carry a
	// "used = false" predicate on the update; without the same guard here a
	// replayed code rewrites UsedAt and the audit trail records the wrong
	// moment. The code stays spent either way, so this is about accuracy
	// rather than access.
	if c.Used {
		return nil
	}
	c.Used = true
	now := time.Now()
	c.UsedAt = &now
	return nil
}

// DeleteRecoveryCodes removes all recovery codes for a user.
func (s *MemoryStore) DeleteRecoveryCodes(_ context.Context, userID id.UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, c := range s.recoveryCodes {
		if c.UserID == userID {
			delete(s.recoveryCodes, k)
		}
	}
	return nil
}

// cloneEnrollment and cloneRecoveryCode keep the store's own records
// unreachable from outside it, in both directions.
//
// This store mutates records in place. ConsumeRecoveryCode writes Used and
// UsedAt on the stored code while holding the write lock, and
// GetRecoveryCodes used to return those very pointers. A caller reading Used
// off what it got back was reading a field another goroutine could be
// writing, with nothing ordering the two.
//
// Used is not an ordinary field to race on: it is what makes a recovery code
// single-use. Note the copy fixes the data race, not the check-then-consume
// window above it. A caller that reads a code as unused and only later calls
// ConsumeRecoveryCode still races another caller doing the same, because
// nothing holds a lock across both steps. Callers that need single-use to be
// airtight have to make consumption itself the test, rather than reading
// first and consuming after.
//
// Both structs are value types apart from UsedAt, which is a *time.Time whose
// pointer is replaced rather than written through, so a shallow copy is
// enough here. Add a slice or map field and it stops being enough, the way
// agentauth's cloneGrant has to copy Scopes.
func cloneEnrollment(e *Enrollment) *Enrollment {
	if e == nil {
		return nil
	}
	cp := *e
	return &cp
}

func cloneRecoveryCode(c *RecoveryCode) *RecoveryCode {
	if c == nil {
		return nil
	}
	cp := *c
	return &cp
}
