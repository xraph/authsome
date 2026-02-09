package engine

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/xraph/authsome/internal/errs"
	"github.com/xraph/authsome/plugins/permissions/language"
)

// Evaluator evaluates compiled policies to make authorization decisions.
type Evaluator struct {
	timeout             time.Duration
	parallelEvaluations int
	enableParallel      bool
	resolver            *AttributeResolver // Optional: for automatic attribute resolution
}

// EvaluatorConfig configures the evaluator.
type EvaluatorConfig struct {
	Timeout             time.Duration
	ParallelEvaluations int
	EnableParallel      bool
	AttributeResolver   *AttributeResolver // Optional: enables automatic attribute enrichment
}

// DefaultEvaluatorConfig returns default evaluator configuration.
func DefaultEvaluatorConfig() EvaluatorConfig {
	return EvaluatorConfig{
		Timeout:             10 * time.Millisecond,
		ParallelEvaluations: 4,
		EnableParallel:      true,
		AttributeResolver:   nil, // Optional, can be set later
	}
}

// NewEvaluator creates a new policy evaluator.
func NewEvaluator(config EvaluatorConfig) *Evaluator {
	return &Evaluator{
		timeout:             config.Timeout,
		parallelEvaluations: config.ParallelEvaluations,
		enableParallel:      config.EnableParallel,
		resolver:            config.AttributeResolver,
	}
}

// SetAttributeResolver sets the attribute resolver for automatic attribute enrichment.
func (e *Evaluator) SetAttributeResolver(resolver *AttributeResolver) {
	e.resolver = resolver
}

// EnrichEvaluationContext enriches the evaluation context with attributes from the resolver
// This is called automatically before evaluation if a resolver is configured.
func (e *Evaluator) EnrichEvaluationContext(ctx context.Context, evalCtx *EvaluationContext) error {
	if e.resolver == nil {
		// No resolver configured, skip enrichment
		return nil
	}

	// Enrich principal attributes if principal.id is set but principal data is minimal
	if evalCtx.Principal != nil {
		if principalID, ok := evalCtx.Principal["id"].(string); ok && principalID != "" {
			// Check if we need to fetch more attributes (e.g., roles is missing)
			if _, hasRoles := evalCtx.Principal["roles"]; !hasRoles {
				// Fetch user attributes
				userAttrs, err := e.resolver.Resolve(ctx, "user", principalID)
				if err == nil {
					// Merge user attributes into principal (don't overwrite existing)
					for k, v := range userAttrs {
						if _, exists := evalCtx.Principal[k]; !exists {
							evalCtx.Principal[k] = v
						}
					}
				}
				// Note: We don't fail on error, just log it in production
			}
		}
	}

	// Enrich resource attributes if resource.type and resource.id are set
	if evalCtx.Resource != nil {
		resourceType, hasType := evalCtx.Resource["type"].(string)
		resourceID, hasID := evalCtx.Resource["id"].(string)

		if hasType && hasID && resourceType != "" && resourceID != "" {
			// Check if we need to fetch more attributes
			if _, hasOwner := evalCtx.Resource["owner"]; !hasOwner {
				// Fetch resource attributes
				key := fmt.Sprintf("%s:%s", resourceType, resourceID)

				resourceAttrs, err := e.resolver.Resolve(ctx, "resource", key)
				if err == nil {
					// Merge resource attributes (don't overwrite existing)
					for k, v := range resourceAttrs {
						if _, exists := evalCtx.Resource[k]; !exists {
							evalCtx.Resource[k] = v
						}
					}
				}
			}
		}
	}

	// Add request context attributes if not present
	if evalCtx.Request == nil {
		evalCtx.Request = make(map[string]any)
	}

	// Only enrich request context if timestamp is missing
	if _, hasTimestamp := evalCtx.Request["timestamp"]; !hasTimestamp {
		// Fetch current request context attributes
		contextAttrs, err := e.resolver.Resolve(ctx, "context", "current")
		if err == nil {
			// Merge context attributes (don't overwrite existing)
			for k, v := range contextAttrs {
				if _, exists := evalCtx.Request[k]; !exists {
					evalCtx.Request[k] = v
				}
			}
		}
	}

	return nil
}

// Evaluate makes an authorization decision based on compiled policies.
func (e *Evaluator) Evaluate(ctx context.Context, policies []*CompiledPolicy, evalCtx *EvaluationContext) (*Decision, error) {
	startTime := time.Now()

	// Enrich evaluation context with attributes if resolver is configured
	if err := e.EnrichEvaluationContext(ctx, evalCtx); err != nil {
		return nil, fmt.Errorf("failed to enrich evaluation context: %w", err)
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Build CEL evaluation context
	celCtx := map[string]any{
		"principal": evalCtx.Principal,
		"resource":  evalCtx.Resource,
		"request":   evalCtx.Request,
		"action":    evalCtx.Action,
	}

	// Add function bindings for custom functions
	functions := language.CreateFunctionBindings(celCtx)
	maps.Copy(celCtx, functions)

	// Sort policies by priority (higher priority first)
	sortedPolicies := sortPoliciesByPriority(policies)

	var matchedPolicies []string

	evaluatedCount := 0

	// Evaluate policies with early exit
	if e.enableParallel && len(sortedPolicies) > e.parallelEvaluations {
		// Parallel evaluation for large policy sets
		result, err := e.evaluateParallel(timeoutCtx, sortedPolicies, celCtx)
		if err != nil {
			return nil, err
		}

		matchedPolicies = result.MatchedPolicies
		evaluatedCount = result.EvaluatedPolicies
	} else {
		// Sequential evaluation
		for _, policy := range sortedPolicies {
			select {
			case <-timeoutCtx.Done():
				return &Decision{
					Allowed:           false,
					EvaluatedPolicies: evaluatedCount,
					EvaluationTime:    time.Since(startTime),
					Error:             "evaluation timeout",
				}, nil
			default:
			}

			evaluatedCount++

			// Execute policy program
			result, _, err := policy.Program.Eval(celCtx)
			if err != nil {
				// Skip invalid policies (don't fail entire evaluation)
				continue
			}

			// Check result
			if boolResult, ok := result.Value().(bool); ok && boolResult {
				matchedPolicies = append(matchedPolicies, policy.PolicyID.String())

				// Early exit on first allow
				return &Decision{
					Allowed:           true,
					MatchedPolicies:   matchedPolicies,
					EvaluatedPolicies: evaluatedCount,
					EvaluationTime:    time.Since(startTime),
				}, nil
			}
		}
	}

	// No policy allowed access
	return &Decision{
		Allowed:           false,
		EvaluatedPolicies: evaluatedCount,
		EvaluationTime:    time.Since(startTime),
	}, nil
}

// evaluateParallel evaluates policies concurrently with early exit.
func (e *Evaluator) evaluateParallel(ctx context.Context, policies []*CompiledPolicy, celCtx map[string]any) (*Decision, error) {
	type result struct {
		policyID string
		allowed  bool
		err      error
	}

	results := make(chan result, len(policies))
	done := make(chan bool, 1)

	// Start concurrent evaluations
	for _, policy := range policies {
		go func(p *CompiledPolicy) {
			select {
			case <-ctx.Done():
				results <- result{policyID: p.PolicyID.String(), err: ctx.Err()}

				return
			case <-done:
				// Early exit signaled
				return
			default:
			}

			// Execute policy
			res, _, err := p.Program.Eval(celCtx)
			if err != nil {
				results <- result{policyID: p.PolicyID.String(), err: err}

				return
			}

			if boolResult, ok := res.Value().(bool); ok {
				results <- result{policyID: p.PolicyID.String(), allowed: boolResult}
			} else {
				results <- result{policyID: p.PolicyID.String(), err: errs.InternalServerErrorWithMessage("invalid result type")}
			}
		}(policy)
	}

	// Collect results with early exit
	var matchedPolicies []string

	evaluatedCount := 0

	for range policies {
		select {
		case <-ctx.Done():
			return &Decision{
				Allowed:           false,
				EvaluatedPolicies: evaluatedCount,
				Error:             "evaluation timeout",
			}, nil
		case res := <-results:
			evaluatedCount++

			if res.err != nil {
				// Skip failed evaluations
				continue
			}

			if res.allowed {
				matchedPolicies = append(matchedPolicies, res.policyID)

				// Signal early exit to other goroutines
				close(done)

				return &Decision{
					Allowed:           true,
					MatchedPolicies:   matchedPolicies,
					EvaluatedPolicies: evaluatedCount,
				}, nil
			}
		}
	}

	return &Decision{
		Allowed:           false,
		EvaluatedPolicies: evaluatedCount,
	}, nil
}

// EvaluateBatch evaluates multiple authorization requests in batch.
func (e *Evaluator) EvaluateBatch(ctx context.Context, policies []*CompiledPolicy, requests []*EvaluationContext) ([]*Decision, error) {
	decisions := make([]*Decision, len(requests))

	for i, req := range requests {
		decision, err := e.Evaluate(ctx, policies, req)
		if err != nil {
			return nil, fmt.Errorf("request %d failed: %w", i, err)
		}

		decisions[i] = decision
	}

	return decisions, nil
}

// sortPoliciesByPriority sorts policies by priority (higher first).
func sortPoliciesByPriority(policies []*CompiledPolicy) []*CompiledPolicy {
	sorted := make([]*CompiledPolicy, len(policies))
	copy(sorted, policies)

	// Simple insertion sort (efficient for small arrays)
	for i := 1; i < len(sorted); i++ {
		key := sorted[i]
		j := i - 1

		// Higher priority comes first
		for j >= 0 && sorted[j].Priority < key.Priority {
			sorted[j+1] = sorted[j]
			j--
		}

		sorted[j+1] = key
	}

	return sorted
}

// GetTimeout returns the evaluation timeout.
func (e *Evaluator) GetTimeout() time.Duration {
	return e.timeout
}

// SetTimeout updates the evaluation timeout.
func (e *Evaluator) SetTimeout(timeout time.Duration) {
	e.timeout = timeout
}
