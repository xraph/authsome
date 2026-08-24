package agentauth

import "sync"

// Permission is the Warden action and resource a delegation scope maps to.
type Permission struct {
	Action   string
	Resource string
}

// Grants builds the Permission a scope confers. It reads as
// Grants("read", "invoice") at the call site.
func Grants(action, resource string) Permission {
	return Permission{Action: action, Resource: resource}
}

// ScopeRegistry holds the host app's delegation vocabulary. Scopes are
// type-level: "invoices:read" means invoices, not one customer's invoices.
// Instance narrowing comes from the user gate, where Warden's ReBAC has
// already decided which invoices the delegating user may read.
type ScopeRegistry struct {
	mu     sync.RWMutex
	scopes map[string]Permission
}

// NewScopeRegistry returns an empty registry.
func NewScopeRegistry() *ScopeRegistry {
	return &ScopeRegistry{scopes: make(map[string]Permission)}
}

// Register maps a delegation scope onto a Warden permission. Re-registering a
// scope replaces its mapping.
func (r *ScopeRegistry) Register(scope string, p Permission) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scopes[scope] = p
}

// Lookup returns the permission a scope confers.
func (r *ScopeRegistry) Lookup(scope string) (Permission, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.scopes[scope]
	return p, ok
}

// Known reports whether a scope has a registered mapping. Consent uses this to
// reject unmapped scopes before a grant is written, so a stored grant can
// never carry a scope that means nothing.
func (r *ScopeRegistry) Known(scope string) bool {
	_, ok := r.Lookup(scope)
	return ok
}

// Covers reports whether any of the granted scopes confers the given
// permission. An unregistered scope confers nothing.
func (r *ScopeRegistry) Covers(scopes []string, action, resource string) bool {
	for _, s := range scopes {
		p, ok := r.Lookup(s)
		if ok && p.Action == action && p.Resource == resource {
			return true
		}
	}
	return false
}
