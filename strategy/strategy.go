// Package strategy defines the authentication strategy system.
// Strategies implement different authentication mechanisms (password,
// social OAuth, magic link, passkey, etc.) and are registered with
// priorities for ordered evaluation.
package strategy

import (
	"context"
	"net/http"

	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// Strategy is the interface that authentication strategies must implement.
type Strategy interface {
	// Name returns the unique identifier for this strategy (e.g., "password", "google", "magic-link").
	Name() string

	// Authenticate attempts to authenticate a request using this strategy.
	// Returns the authenticated user and a new session, or an error if authentication fails.
	// If this strategy does not apply to the request, return ErrStrategyNotApplicable.
	Authenticate(ctx context.Context, r *http.Request) (*Result, error)
}

// Result represents the outcome of a successful authentication.
type Result struct {
	User    *user.User       `json:"user"`
	Session *session.Session `json:"session"`
	New     bool             `json:"new"` // True if user was created (signup)
}

// ErrStrategyNotApplicable is returned when a strategy does not apply to the request.
type ErrStrategyNotApplicable struct{}

func (e ErrStrategyNotApplicable) Error() string {
	return "strategy: not applicable to this request"
}
