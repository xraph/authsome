package passkey

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"
)

// ChallengeSession stores WebAuthn challenge data with timeout
type ChallengeSession struct {
	Challenge   []byte
	UserID      xid.ID
	SessionData interface{} // webauthn.SessionData
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// ChallengeStore defines the interface for challenge session storage
type ChallengeStore interface {
	// Store saves a challenge session with expiration
	Store(ctx context.Context, sessionID string, session *ChallengeSession) error

	// Get retrieves a challenge session
	Get(ctx context.Context, sessionID string) (*ChallengeSession, error)

	// Delete removes a challenge session
	Delete(ctx context.Context, sessionID string) error

	// CleanupExpired removes all expired challenge sessions
	CleanupExpired(ctx context.Context) error
}

// MemoryChallengeStore is an in-memory implementation of ChallengeStore
// For production with multiple instances, use RedisChallengeStore instead
type MemoryChallengeStore struct {
	mu       sync.RWMutex
	sessions map[string]*ChallengeSession
	timeout  time.Duration
}

// NewMemoryChallengeStore creates a new in-memory challenge store
func NewMemoryChallengeStore(timeout time.Duration) *MemoryChallengeStore {
	store := &MemoryChallengeStore{
		sessions: make(map[string]*ChallengeSession),
		timeout:  timeout,
	}

	// Start cleanup goroutine
	go store.startCleanupWorker()

	return store
}

// Store saves a challenge session
func (s *MemoryChallengeStore) Store(ctx context.Context, sessionID string, session *ChallengeSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set expiration if not already set
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(s.timeout)
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}

	s.sessions[sessionID] = session
	return nil
}

// Get retrieves a challenge session
func (s *MemoryChallengeStore) Get(ctx context.Context, sessionID string) (*ChallengeSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("challenge session not found")
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("challenge session expired")
	}

	return session, nil
}

// Delete removes a challenge session
func (s *MemoryChallengeStore) Delete(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// CleanupExpired removes all expired sessions
func (s *MemoryChallengeStore) CleanupExpired(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for sessionID, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, sessionID)
		}
	}
	return nil
}

// startCleanupWorker runs periodic cleanup of expired sessions
func (s *MemoryChallengeStore) startCleanupWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		_ = s.CleanupExpired(context.Background())
	}
}

// RedisChallengeStore is a Redis-backed implementation of ChallengeStore
// This is recommended for production environments with multiple instances
type RedisChallengeStore struct {
	// redis *redis.Client
	// prefix string
	// timeout time.Duration
	// Implementation deferred - use MemoryChallengeStore for now
	// TODO: Implement Redis backend when Redis integration is needed
}

// Note: Redis implementation would store sessions with automatic expiration:
// func (s *RedisChallengeStore) Store(ctx context.Context, sessionID string, session *ChallengeSession) error {
//     key := s.prefix + sessionID
//     data, err := json.Marshal(session)
//     if err != nil {
//         return err
//     }
//     return s.redis.Set(ctx, key, data, s.timeout).Err()
// }
