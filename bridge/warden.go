package bridge

import "context"

// Authorizer is a local authorization interface. Implementations perform
// access control checks (e.g., via warden).
type Authorizer interface {
	Check(ctx context.Context, req *AuthzRequest) (*AuthzResult, error)
}

// AuthzRequest represents an authorization check.
type AuthzRequest struct {
	Subject  string `json:"subject"`
	Action   string `json:"action"`
	Resource string `json:"resource"`
	Tenant   string `json:"tenant,omitempty"`
}

// AuthzResult represents the outcome of an authorization check.
type AuthzResult struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason,omitempty"`
}

// AuthorizerFunc is an adapter to use a plain function as an Authorizer.
type AuthorizerFunc func(ctx context.Context, req *AuthzRequest) (*AuthzResult, error)

// Check implements Authorizer.
func (f AuthorizerFunc) Check(ctx context.Context, req *AuthzRequest) (*AuthzResult, error) {
	return f(ctx, req)
}
