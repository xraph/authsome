package rbac

import (
    "context"
    "strings"
    "sync"
)

// Service provides in-memory management of RBAC policies.
// Storage-backed repositories can be added later via repository interfaces.
type Service struct {
    mu       sync.RWMutex
    policies []*Policy
    eval     *Evaluator
}

func NewService() *Service {
    return &Service{eval: NewEvaluator()}
}

func (s *Service) AddPolicy(p *Policy) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.policies = append(s.policies, p)
}

func (s *Service) AddExpression(expression string) error {
    parser := NewParser()
    p, err := parser.Parse(expression)
    if err != nil {
        return err
    }
    s.AddPolicy(p)
    return nil
}

// Allowed checks whether any registered policy allows the context.
func (s *Service) Allowed(ctx *Context) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for _, p := range s.policies {
        if s.eval.Evaluate(p, ctx) {
            return true
        }
    }
    return false
}

// AllowedWithRoles checks policies against a subject plus assigned roles.
// If a policy subject is of form "role:<name>", it will be evaluated when
// that role is present in the provided roles slice.
func (s *Service) AllowedWithRoles(ctx *Context, roles []string) bool {
    s.mu.RLock()
    defer s.mu.RUnlock()
    for _, p := range s.policies {
        // Direct subject match
        if s.eval.Evaluate(p, ctx) {
            return true
        }
        // Role-based subject: evaluate using role subject when user has role
        if strings.HasPrefix(strings.ToLower(strings.TrimSpace(p.Subject)), "role:") {
            roleName := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(p.Subject), "role:"))
            for _, r := range roles {
                if strings.EqualFold(strings.TrimSpace(r), roleName) {
                    // clone context with role subject
                    rc := *ctx
                    rc.Subject = p.Subject
                    if s.eval.Evaluate(p, &rc) {
                        return true
                    }
                }
            }
        }
    }
    return false
}

// LoadPolicies loads and parses all stored policy expressions from a repository
func (s *Service) LoadPolicies(ctx context.Context, repo PolicyRepository) error {
    exprs, err := repo.ListAll(ctx)
    if err != nil {
        return err
    }
    parser := NewParser()
    for _, ex := range exprs {
        p, err := parser.Parse(ex)
        if err != nil {
            // skip invalid entries
            continue
        }
        s.AddPolicy(p)
    }
    return nil
}