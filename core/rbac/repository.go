package rbac

import "context"

// PolicyRepository provides access to stored policy expressions
type PolicyRepository interface {
	// ListAll returns all stored policy expressions
	ListAll(ctx context.Context) ([]string, error)
	// Create stores a new policy expression
	Create(ctx context.Context, expression string) error
}
