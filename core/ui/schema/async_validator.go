package schema

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// AsyncValidator is the interface for asynchronous field validators
// These are used for validations that require I/O operations like
// database lookups, API calls, or other async operations.
type AsyncValidator interface {
	// ValidateAsync validates the value asynchronously
	ValidateAsync(ctx context.Context, fieldID string, value interface{}) error
	// Name returns the validator name for identification
	Name() string
}

// AsyncValidatorFunc is a function type that implements AsyncValidator
type AsyncValidatorFunc struct {
	name string
	fn   func(ctx context.Context, fieldID string, value interface{}) error
}

// ValidateAsync implements the AsyncValidator interface
func (v AsyncValidatorFunc) ValidateAsync(ctx context.Context, fieldID string, value interface{}) error {
	return v.fn(ctx, fieldID, value)
}

// Name returns the validator name
func (v AsyncValidatorFunc) Name() string {
	return v.name
}

// NewAsyncValidator creates a new async validator from a function
func NewAsyncValidator(name string, fn func(ctx context.Context, fieldID string, value interface{}) error) AsyncValidator {
	return AsyncValidatorFunc{name: name, fn: fn}
}

// UniqueCheckFunc is a function that checks if a value is unique
type UniqueCheckFunc func(ctx context.Context, value interface{}) (bool, error)

// UniqueValidator returns an async validator that checks for uniqueness
func UniqueValidator(checkFn UniqueCheckFunc) AsyncValidator {
	return NewAsyncValidator("unique", func(ctx context.Context, fieldID string, value interface{}) error {
		isUnique, err := checkFn(ctx, value)
		if err != nil {
			return fmt.Errorf("failed to check uniqueness: %w", err)
		}
		if !isUnique {
			return fmt.Errorf("this value is already taken")
		}
		return nil
	})
}

// UniqueValidatorWithMessage returns a unique validator with a custom message
func UniqueValidatorWithMessage(checkFn UniqueCheckFunc, message string) AsyncValidator {
	return NewAsyncValidator("unique", func(ctx context.Context, fieldID string, value interface{}) error {
		isUnique, err := checkFn(ctx, value)
		if err != nil {
			return fmt.Errorf("failed to check uniqueness: %w", err)
		}
		if !isUnique {
			return fmt.Errorf("%s", message)
		}
		return nil
	})
}

// ExternalAPIValidator returns an async validator that calls an external API
func ExternalAPIValidator(checkURL string, timeout time.Duration) AsyncValidator {
	client := &http.Client{Timeout: timeout}

	return NewAsyncValidator("external_api", func(ctx context.Context, fieldID string, value interface{}) error {
		// Build request URL with the value
		reqURL := fmt.Sprintf("%s?field=%s&value=%v", checkURL, fieldID, value)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("external validation failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("validation failed")
		}

		return nil
	})
}

// ExistsValidator returns an async validator that checks if a resource exists
func ExistsValidator(checkFn func(ctx context.Context, value interface{}) (bool, error), entityName string) AsyncValidator {
	return NewAsyncValidator("exists", func(ctx context.Context, fieldID string, value interface{}) error {
		exists, err := checkFn(ctx, value)
		if err != nil {
			return fmt.Errorf("failed to check existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("%s not found", entityName)
		}
		return nil
	})
}

// NotExistsValidator returns an async validator that checks if a resource does NOT exist
func NotExistsValidator(checkFn func(ctx context.Context, value interface{}) (bool, error), message string) AsyncValidator {
	return NewAsyncValidator("not_exists", func(ctx context.Context, fieldID string, value interface{}) error {
		exists, err := checkFn(ctx, value)
		if err != nil {
			return fmt.Errorf("failed to check: %w", err)
		}
		if exists {
			return fmt.Errorf("%s", message)
		}
		return nil
	})
}

// ConditionalAsyncValidator runs an async validator only when a condition is met
func ConditionalAsyncValidator(condition func(ctx context.Context, fieldID string, value interface{}) bool, validator AsyncValidator) AsyncValidator {
	return NewAsyncValidator("conditional_"+validator.Name(), func(ctx context.Context, fieldID string, value interface{}) error {
		if !condition(ctx, fieldID, value) {
			return nil
		}
		return validator.ValidateAsync(ctx, fieldID, value)
	})
}

// RateLimitedValidator wraps an async validator with rate limiting
func RateLimitedValidator(validator AsyncValidator, maxRequests int, window time.Duration) AsyncValidator {
	// Simple token bucket implementation
	tokens := maxRequests
	lastRefill := time.Now()

	return NewAsyncValidator("rate_limited_"+validator.Name(), func(ctx context.Context, fieldID string, value interface{}) error {
		// Refill tokens if window has passed
		now := time.Now()
		if now.Sub(lastRefill) >= window {
			tokens = maxRequests
			lastRefill = now
		}

		if tokens <= 0 {
			return fmt.Errorf("validation rate limit exceeded, please try again later")
		}

		tokens--
		return validator.ValidateAsync(ctx, fieldID, value)
	})
}

// TimeoutValidator wraps an async validator with a specific timeout
func TimeoutValidator(validator AsyncValidator, timeout time.Duration) AsyncValidator {
	return NewAsyncValidator("timeout_"+validator.Name(), func(ctx context.Context, fieldID string, value interface{}) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- validator.ValidateAsync(ctx, fieldID, value)
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return fmt.Errorf("validation timed out")
		}
	})
}

// RetryValidator wraps an async validator with retry logic
func RetryValidator(validator AsyncValidator, maxRetries int, backoff time.Duration) AsyncValidator {
	return NewAsyncValidator("retry_"+validator.Name(), func(ctx context.Context, fieldID string, value interface{}) error {
		var lastErr error
		for i := 0; i < maxRetries; i++ {
			err := validator.ValidateAsync(ctx, fieldID, value)
			if err == nil {
				return nil
			}
			lastErr = err

			// Don't wait after the last attempt
			if i < maxRetries-1 {
				select {
				case <-time.After(backoff * time.Duration(i+1)):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		return lastErr
	})
}

// CompositeAsyncValidator combines multiple async validators
func CompositeAsyncValidator(validators ...AsyncValidator) AsyncValidator {
	return NewAsyncValidator("composite", func(ctx context.Context, fieldID string, value interface{}) error {
		for _, v := range validators {
			if err := v.ValidateAsync(ctx, fieldID, value); err != nil {
				return err
			}
		}
		return nil
	})
}

// ParallelAsyncValidator runs multiple async validators in parallel and returns all errors
func ParallelAsyncValidator(validators ...AsyncValidator) AsyncValidator {
	return NewAsyncValidator("parallel", func(ctx context.Context, fieldID string, value interface{}) error {
		if len(validators) == 0 {
			return nil
		}

		errChan := make(chan error, len(validators))

		for _, v := range validators {
			go func(validator AsyncValidator) {
				errChan <- validator.ValidateAsync(ctx, fieldID, value)
			}(v)
		}

		var errors []string
		for i := 0; i < len(validators); i++ {
			if err := <-errChan; err != nil {
				errors = append(errors, err.Error())
			}
		}

		if len(errors) > 0 {
			return fmt.Errorf("%s", errors[0]) // Return first error
		}
		return nil
	})
}

// CachedAsyncValidator caches validation results for identical values
type CachedAsyncValidator struct {
	validator AsyncValidator
	cache     map[string]cachedResult
	ttl       time.Duration
}

type cachedResult struct {
	err       error
	expiresAt time.Time
}

// NewCachedAsyncValidator creates a cached async validator
func NewCachedAsyncValidator(validator AsyncValidator, ttl time.Duration) *CachedAsyncValidator {
	return &CachedAsyncValidator{
		validator: validator,
		cache:     make(map[string]cachedResult),
		ttl:       ttl,
	}
}

// ValidateAsync implements AsyncValidator with caching
func (c *CachedAsyncValidator) ValidateAsync(ctx context.Context, fieldID string, value interface{}) error {
	key := fmt.Sprintf("%s:%v", fieldID, value)

	// Check cache
	if result, ok := c.cache[key]; ok {
		if time.Now().Before(result.expiresAt) {
			return result.err
		}
		delete(c.cache, key)
	}

	// Run validation
	err := c.validator.ValidateAsync(ctx, fieldID, value)

	// Cache result
	c.cache[key] = cachedResult{
		err:       err,
		expiresAt: time.Now().Add(c.ttl),
	}

	return err
}

// Name returns the validator name
func (c *CachedAsyncValidator) Name() string {
	return "cached_" + c.validator.Name()
}

// CustomAsyncValidator creates an async validator with a custom validation function
func CustomAsyncValidator(name string, message string, fn func(ctx context.Context, fieldID string, value interface{}) bool) AsyncValidator {
	return NewAsyncValidator(name, func(ctx context.Context, fieldID string, value interface{}) error {
		if !fn(ctx, fieldID, value) {
			return fmt.Errorf("%s", message)
		}
		return nil
	})
}
