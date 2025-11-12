package engine

import (
	"fmt"
	"time"

	"github.com/xraph/authsome/plugins/permissions/core"
	"github.com/xraph/authsome/plugins/permissions/language"
)

// Compiler compiles policies from CEL expressions to executable programs
type Compiler struct {
	parser        *language.Parser
	maxComplexity int
}

// CompilerConfig configures the compiler
type CompilerConfig struct {
	MaxComplexity int // Maximum allowed expression complexity
}

// DefaultCompilerConfig returns default compiler configuration
func DefaultCompilerConfig() CompilerConfig {
	return CompilerConfig{
		MaxComplexity: 100,
	}
}

// NewCompiler creates a new policy compiler
func NewCompiler(config CompilerConfig) (*Compiler, error) {
	parser, err := language.NewParser()
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	return &Compiler{
		parser:        parser,
		maxComplexity: config.MaxComplexity,
	}, nil
}

// Compile converts a policy to a compiled, executable form
func (c *Compiler) Compile(policy *core.Policy) (*CompiledPolicy, error) {
	if policy == nil {
		return nil, fmt.Errorf("policy cannot be nil")
	}

	if policy.Expression == "" {
		return nil, fmt.Errorf("policy expression cannot be empty")
	}

	// Parse the CEL expression
	ast, err := c.parser.Parse(policy.Expression)
	if err != nil {
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	// Check expression complexity
	complexity := c.parser.ExpressionComplexity(ast)
	if complexity > c.maxComplexity {
		return nil, fmt.Errorf("expression complexity %d exceeds maximum %d", complexity, c.maxComplexity)
	}

	// Create executable program
	prg, err := c.parser.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program creation failed: %w", err)
	}

	// Build compiled policy
	compiled := &CompiledPolicy{
		PolicyID:           policy.ID,
		AppID:              policy.AppID,
		UserOrganizationID: policy.UserOrganizationID,
		NamespaceID:        policy.NamespaceID,
		Name:               policy.Name,
		Description:        policy.Description,
		Program:            prg,
		AST:                ast,
		ResourceType:       policy.ResourceType,
		Actions:            policy.Actions,
		Priority:           policy.Priority,
		Version:            policy.Version,
		CompiledAt:         time.Now(),
	}

	return compiled, nil
}

// CompileBatch compiles multiple policies in parallel
func (c *Compiler) CompileBatch(policies []*core.Policy) (map[string]*CompiledPolicy, error) {
	if len(policies) == 0 {
		return make(map[string]*CompiledPolicy), nil
	}

	type result struct {
		id       string
		compiled *CompiledPolicy
		err      error
	}

	results := make(chan result, len(policies))

	// Compile policies concurrently
	for _, policy := range policies {
		go func(p *core.Policy) {
			compiled, err := c.Compile(p)
			results <- result{
				id:       p.ID.String(),
				compiled: compiled,
				err:      err,
			}
		}(policy)
	}

	// Collect results
	compiled := make(map[string]*CompiledPolicy)
	var errors []error

	for i := 0; i < len(policies); i++ {
		res := <-results
		if res.err != nil {
			errors = append(errors, fmt.Errorf("policy %s: %w", res.id, res.err))
		} else {
			compiled[res.id] = res.compiled
		}
	}

	if len(errors) > 0 {
		// Return partial results with first error
		return compiled, errors[0]
	}

	return compiled, nil
}

// Validate checks if a policy expression is valid without fully compiling
func (c *Compiler) Validate(expression string) error {
	return c.parser.ValidateExpression(expression)
}

// EstimateComplexity returns the estimated complexity of an expression
func (c *Compiler) EstimateComplexity(expression string) (int, error) {
	ast, err := c.parser.Parse(expression)
	if err != nil {
		return 0, err
	}
	return c.parser.ExpressionComplexity(ast), nil
}

// GetMaxComplexity returns the maximum allowed complexity
func (c *Compiler) GetMaxComplexity() int {
	return c.maxComplexity
}
