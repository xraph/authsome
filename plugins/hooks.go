package plugins

import (
	"context"

	"github.com/xraph/authsome/core"
)

// HookFunc is a function that runs before/after an operation.
type HookFunc = core.HookFunc

// HookRegistry manages hooks for various operations.
type HookRegistry struct {
	beforeHooks map[string][]HookFunc
	afterHooks  map[string][]HookFunc
}

// NewHookRegistry creates a new hook registry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{
		beforeHooks: make(map[string][]HookFunc),
		afterHooks:  make(map[string][]HookFunc),
	}
}

// RegisterBefore registers a before hook.
func (r *HookRegistry) RegisterBefore(operation string, hook HookFunc) {
	r.beforeHooks[operation] = append(r.beforeHooks[operation], hook)
}

// RegisterAfter registers an after hook.
func (r *HookRegistry) RegisterAfter(operation string, hook HookFunc) {
	r.afterHooks[operation] = append(r.afterHooks[operation], hook)
}

// RunBefore runs all before hooks for an operation.
func (r *HookRegistry) RunBefore(ctx context.Context, operation string, data any) error {
	for _, hook := range r.beforeHooks[operation] {
		if err := hook(ctx, data); err != nil {
			return err
		}
	}

	return nil
}

// RunAfter runs all after hooks for an operation.
func (r *HookRegistry) RunAfter(ctx context.Context, operation string, data any) error {
	for _, hook := range r.afterHooks[operation] {
		if err := hook(ctx, data); err != nil {
			return err
		}
	}

	return nil
}
