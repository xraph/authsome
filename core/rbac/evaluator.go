package rbac

import (
	"strings"
)

// ConditionEvaluator evaluates a condition string against a Context.
// Returning true means the condition passes.
type ConditionEvaluator func(condition string, ctx *Context) bool

// Evaluator checks if a policy allows the given context.
type Evaluator struct {
	EvaluateCondition ConditionEvaluator
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		EvaluateCondition: defaultConditionEvaluator,
	}
}

// Evaluate returns true if the policy allows the action on the resource for the subject.
func (e *Evaluator) Evaluate(policy *Policy, ctx *Context) bool {
	// subject must match exactly (simple model)
	if !equal(policy.Subject, ctx.Subject) {
		return false
	}

	// action must be included
	if !contains(policy.Actions, ctx.Action) {
		return false
	}

	// resource supports wildcard suffix "*"
	if !resourceMatches(policy.Resource, ctx.Resource) {
		return false
	}

	// evaluate condition if provided
	if strings.TrimSpace(policy.Condition) != "" {
		return e.EvaluateCondition(policy.Condition, ctx)
	}

	return true
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if equal(item, v) {
			return true
		}
	}

	return false
}

func equal(a, b string) bool { return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b)) }

func resourceMatches(pattern, value string) bool {
	p := strings.TrimSpace(pattern)
	v := strings.TrimSpace(value)

	if p == "*" {
		return true
	}

	if before, ok := strings.CutSuffix(p, ":*"); ok {
		prefix := before

		return strings.HasPrefix(v, prefix+":")
	}

	return equal(p, v)
}

// defaultConditionEvaluator supports simple "key = value" checks using ctx.Vars.
func defaultConditionEvaluator(condition string, ctx *Context) bool {
	parts := strings.Split(condition, "=")
	if len(parts) != 2 {
		// unsupported condition syntax -> deny
		return false
	}

	key := strings.TrimSpace(parts[0])
	val := strings.TrimSpace(parts[1])

	if ctx == nil || ctx.Vars == nil {
		return false
	}

	ctxVal, ok := ctx.Vars[key]
	if !ok {
		return false
	}

	return equal(ctxVal, val)
}
