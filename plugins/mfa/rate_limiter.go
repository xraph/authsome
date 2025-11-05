package mfa

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/xid"
	"github.com/xraph/authsome/repository"
	"github.com/xraph/authsome/schema"
)

// RateLimiter provides rate limiting for MFA operations
type RateLimiter struct {
	config *RateLimitConfig
	repo   *repository.MFARepository
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig, repo *repository.MFARepository) *RateLimiter {
	return &RateLimiter{
		config: config,
		repo:   repo,
	}
}

// LimitResult represents the result of a rate limit check
type LimitResult struct {
	Allowed      bool
	RetryAfter   *time.Duration
	AttemptsLeft int
	LockoutEnds  *time.Time
}

// CheckUserLimit checks if a user has exceeded rate limits
func (r *RateLimiter) CheckUserLimit(ctx context.Context, userID xid.ID) (*LimitResult, error) {
	if !r.config.Enabled {
		return &LimitResult{Allowed: true, AttemptsLeft: r.config.MaxAttempts}, nil
	}

	window := time.Duration(r.config.WindowMinutes) * time.Minute
	since := time.Now().Add(-window)

	// Count failed attempts in the window
	failedCount, err := r.repo.CountFailedAttempts(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to count attempts: %w", err)
	}

	// Check if user is locked out
	if failedCount >= r.config.MaxAttempts {
		// Calculate lockout end time
		attempts, err := r.repo.GetRecentAttempts(ctx, userID, since)
		if err != nil {
			return nil, err
		}

		if len(attempts) > 0 {
			// Find the first attempt that triggered lockout
			lockoutStart := attempts[0].CreatedAt
			lockoutDuration := time.Duration(r.config.LockoutMinutes) * time.Minute
			lockoutEnd := lockoutStart.Add(lockoutDuration)

			if time.Now().Before(lockoutEnd) {
				// Still locked out
				retryAfter := time.Until(lockoutEnd)
				return &LimitResult{
					Allowed:      false,
					RetryAfter:   &retryAfter,
					AttemptsLeft: 0,
					LockoutEnds:  &lockoutEnd,
				}, nil
			}
		}
	}

	// Calculate remaining attempts
	attemptsLeft := r.config.MaxAttempts - failedCount
	if attemptsLeft < 0 {
		attemptsLeft = 0
	}

	return &LimitResult{
		Allowed:      true,
		AttemptsLeft: attemptsLeft,
	}, nil
}

// CheckFactorLimit checks if a specific factor has exceeded rate limits
func (r *RateLimiter) CheckFactorLimit(ctx context.Context, userID xid.ID, factorType FactorType) (*LimitResult, error) {
	if !r.config.Enabled {
		return &LimitResult{Allowed: true, AttemptsLeft: r.config.MaxAttempts}, nil
	}

	window := time.Duration(r.config.WindowMinutes) * time.Minute
	since := time.Now().Add(-window)

	// Get recent attempts for this factor type
	attempts, err := r.repo.GetRecentAttempts(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get attempts: %w", err)
	}

	// Count failed attempts for this factor type
	failedCount := 0
	for _, attempt := range attempts {
		if !attempt.Success && attempt.Type == string(factorType) {
			failedCount++
		}
	}

	// Check if locked out
	if failedCount >= r.config.MaxAttempts {
		// Find the most recent failed attempt for this factor
		var lastFailedAttempt *time.Time
		for _, attempt := range attempts {
			if !attempt.Success && attempt.Type == string(factorType) {
				lastFailedAttempt = &attempt.CreatedAt
				break
			}
		}

		if lastFailedAttempt != nil {
			lockoutDuration := time.Duration(r.config.LockoutMinutes) * time.Minute
			lockoutEnd := lastFailedAttempt.Add(lockoutDuration)

			if time.Now().Before(lockoutEnd) {
				retryAfter := time.Until(lockoutEnd)
				return &LimitResult{
					Allowed:      false,
					RetryAfter:   &retryAfter,
					AttemptsLeft: 0,
					LockoutEnds:  &lockoutEnd,
				}, nil
			}
		}
	}

	attemptsLeft := r.config.MaxAttempts - failedCount
	if attemptsLeft < 0 {
		attemptsLeft = 0
	}

	return &LimitResult{
		Allowed:      true,
		AttemptsLeft: attemptsLeft,
	}, nil
}

// RecordAttempt records a verification attempt
func (r *RateLimiter) RecordAttempt(ctx context.Context, userID xid.ID, factorID *xid.ID, factorType FactorType, success bool, metadata map[string]string) error {
	attempt := &schema.MFAAttempt{
		ID:       xid.New(),
		UserID:   userID,
		FactorID: factorID,
		Type:     string(factorType),
		Success:  success,
		Metadata: make(map[string]interface{}),
	}

	// Copy metadata
	for k, v := range metadata {
		attempt.Metadata[k] = v
	}

	if !success {
		attempt.FailureReason = metadata["failure_reason"]
	}

	// Set audit fields
	attempt.AuditableModel.CreatedBy = userID
	attempt.AuditableModel.UpdatedBy = userID

	return r.repo.CreateAttempt(ctx, attempt)
}

// GetExponentialBackoff calculates exponential backoff duration
func (r *RateLimiter) GetExponentialBackoff(attemptNumber int) time.Duration {
	// Exponential backoff: 2^n seconds, capped at lockout duration
	if attemptNumber <= 0 {
		return 0
	}

	baseDelay := time.Second
	maxDelay := time.Duration(r.config.LockoutMinutes) * time.Minute

	// Calculate 2^attemptNumber seconds
	delay := baseDelay * (1 << uint(attemptNumber-1))

	if delay > maxDelay {
		delay = maxDelay
	}

	return delay
}

// IsLockedOut checks if a user is currently locked out
func (r *RateLimiter) IsLockedOut(ctx context.Context, userID xid.ID) (bool, *time.Time, error) {
	result, err := r.CheckUserLimit(ctx, userID)
	if err != nil {
		return false, nil, err
	}

	return !result.Allowed, result.LockoutEnds, nil
}

// ClearLockout clears the lockout for a user (admin function)
func (r *RateLimiter) ClearLockout(ctx context.Context, userID xid.ID) error {
	// This would require marking old attempts as cleared
	// For now, we rely on time-based expiry
	// In production, add a "cleared" flag to attempts
	return nil
}
