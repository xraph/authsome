package flows

import (
	"context"
	"fmt"
)

// Flow represents a customizable authentication flow
type Flow interface {
	// ID returns the unique identifier for this flow
	ID() string

	// Name returns the human-readable name for this flow
	Name() string

	// Execute runs the flow with the given context and data
	Execute(ctx context.Context, data map[string]interface{}) (*FlowResult, error)

	// Steps returns all steps in this flow
	Steps() []*Step

	// AddStep adds a step to the flow
	AddStep(step *Step) Flow

	// AddBeforeHook adds a hook to run before a specific step
	AddBeforeHook(stepID string, hook HookFunc) Flow

	// AddAfterHook adds a hook to run after a specific step
	AddAfterHook(stepID string, hook HookFunc) Flow
}

// Step represents a single step in an authentication flow
type Step struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Required    bool                   `json:"required"`
	Config      map[string]interface{} `json:"config"`
	BeforeHooks []HookFunc             `json:"-"`
	AfterHooks  []HookFunc             `json:"-"`
	Handler     StepHandler            `json:"-"`
}

// StepHandler defines the function signature for step handlers
type StepHandler func(ctx context.Context, step *Step, data map[string]interface{}) (*StepResult, error)

// HookFunc defines the function signature for before/after hooks
type HookFunc func(ctx context.Context, step *Step, data map[string]interface{}) error

// FlowResult represents the result of executing a flow
type FlowResult struct {
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data"`
	Errors    []string               `json:"errors,omitempty"`
	NextStep  string                 `json:"next_step,omitempty"`
	Completed bool                   `json:"completed"`
}

// StepResult represents the result of executing a single step
type StepResult struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error,omitempty"`
	Skip    bool                   `json:"skip"`
	Stop    bool                   `json:"stop"`
}

// BaseFlow provides a basic implementation of the Flow interface
type BaseFlow struct {
	id          string
	name        string
	steps       []*Step
	beforeHooks map[string][]HookFunc
	afterHooks  map[string][]HookFunc
}

// NewBaseFlow creates a new base flow
func NewBaseFlow(id, name string) *BaseFlow {
	return &BaseFlow{
		id:          id,
		name:        name,
		steps:       make([]*Step, 0),
		beforeHooks: make(map[string][]HookFunc),
		afterHooks:  make(map[string][]HookFunc),
	}
}

// ID returns the flow ID
func (f *BaseFlow) ID() string {
	return f.id
}

// Name returns the flow name
func (f *BaseFlow) Name() string {
	return f.name
}

// Steps returns all steps in the flow
func (f *BaseFlow) Steps() []*Step {
	return f.steps
}

// AddStep adds a step to the flow
func (f *BaseFlow) AddStep(step *Step) Flow {
	f.steps = append(f.steps, step)
	return f
}

// AddBeforeHook adds a hook to run before a specific step
func (f *BaseFlow) AddBeforeHook(stepID string, hook HookFunc) Flow {
	if f.beforeHooks[stepID] == nil {
		f.beforeHooks[stepID] = make([]HookFunc, 0)
	}
	f.beforeHooks[stepID] = append(f.beforeHooks[stepID], hook)
	return f
}

// AddAfterHook adds a hook to run after a specific step
func (f *BaseFlow) AddAfterHook(stepID string, hook HookFunc) Flow {
	if f.afterHooks[stepID] == nil {
		f.afterHooks[stepID] = make([]HookFunc, 0)
	}
	f.afterHooks[stepID] = append(f.afterHooks[stepID], hook)
	return f
}

// Execute runs the flow with the given context and data
func (f *BaseFlow) Execute(ctx context.Context, data map[string]interface{}) (*FlowResult, error) {
	result := &FlowResult{
		Success:   true,
		Data:      make(map[string]interface{}),
		Errors:    make([]string, 0),
		Completed: false,
	}

	// Copy input data to result
	for k, v := range data {
		result.Data[k] = v
	}

	// Execute each step
	for i, step := range f.steps {
		// Run before hooks
		if hooks, exists := f.beforeHooks[step.ID]; exists {
			for _, hook := range hooks {
				if err := hook(ctx, step, result.Data); err != nil {
					result.Success = false
					result.Errors = append(result.Errors, fmt.Sprintf("before hook error for step %s: %v", step.ID, err))
					return result, err
				}
			}
		}

		// Execute step handler if present
		if step.Handler != nil {
			stepResult, err := step.Handler(ctx, step, result.Data)
			if err != nil {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("step %s error: %v", step.ID, err))
				if step.Required {
					return result, err
				}
				continue
			}

			// Handle step result
			if stepResult != nil {
				if !stepResult.Success && step.Required {
					result.Success = false
					result.Errors = append(result.Errors, stepResult.Error)
					return result, fmt.Errorf("required step %s failed: %s", step.ID, stepResult.Error)
				}

				// Merge step data into result
				for k, v := range stepResult.Data {
					result.Data[k] = v
				}

				// Check if we should skip remaining steps
				if stepResult.Skip {
					continue
				}

				// Check if we should stop the flow
				if stepResult.Stop {
					break
				}
			}
		}

		// Run after hooks
		if hooks, exists := f.afterHooks[step.ID]; exists {
			for _, hook := range hooks {
				if err := hook(ctx, step, result.Data); err != nil {
					result.Success = false
					result.Errors = append(result.Errors, fmt.Sprintf("after hook error for step %s: %v", step.ID, err))
					return result, err
				}
			}
		}

		// Set next step if not the last one
		if i < len(f.steps)-1 {
			result.NextStep = f.steps[i+1].ID
		}
	}

	result.Completed = true
	return result, nil
}

// FlowBuilder provides a fluent interface for building flows
type FlowBuilder struct {
	flow *BaseFlow
}

// NewFlowBuilder creates a new flow builder
func NewFlowBuilder(id, name string) *FlowBuilder {
	return &FlowBuilder{
		flow: NewBaseFlow(id, name),
	}
}

// Step adds a step to the flow being built
func (b *FlowBuilder) Step(id, name, stepType string, required bool, config map[string]interface{}, handler StepHandler) *FlowBuilder {
	step := &Step{
		ID:       id,
		Name:     name,
		Type:     stepType,
		Required: required,
		Config:   config,
		Handler:  handler,
	}
	b.flow.AddStep(step)
	return b
}

// Before adds a before hook for the specified step
func (b *FlowBuilder) Before(stepID string, hook HookFunc) *FlowBuilder {
	b.flow.AddBeforeHook(stepID, hook)
	return b
}

// After adds an after hook for the specified step
func (b *FlowBuilder) After(stepID string, hook HookFunc) *FlowBuilder {
	b.flow.AddAfterHook(stepID, hook)
	return b
}

// Build returns the constructed flow
func (b *FlowBuilder) Build() Flow {
	return b.flow
}
