package language

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/xraph/authsome/internal/errs"
)

// Parser handles CEL expression parsing with AuthSome-specific context.
type Parser struct {
	env *cel.Env
}

// NewParser creates a new CEL parser with AuthSome context variables and functions.
func NewParser() (*Parser, error) {
	// Create CEL environment with AuthSome-specific context and custom library
	env, err := cel.NewEnv(
		// Context variables available in all policies
		cel.Variable("principal", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("resource", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("request", cel.MapType(cel.StringType, cel.DynType)),
		cel.Variable("action", cel.StringType),

		// Use AuthSome custom library for functions
		cel.Lib(authsomeLib{}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return &Parser{env: env}, nil
}

// Parse compiles a CEL expression and returns the AST.
func (p *Parser) Parse(expression string) (*cel.Ast, error) {
	if expression == "" {
		return nil, errs.RequiredField("expression")
	}

	ast, issues := p.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("parse error: %w", issues.Err())
	}

	// Verify the expression returns a boolean
	if ast.OutputType().String() != "bool" {
		return nil, fmt.Errorf("expression must return boolean, got %v", ast.OutputType())
	}

	return ast, nil
}

// Program creates an executable program from an AST.
func (p *Parser) Program(ast *cel.Ast, opts ...cel.ProgramOption) (cel.Program, error) {
	prg, err := p.env.Program(ast, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create program: %w", err)
	}

	return prg, nil
}

// ValidateExpression checks if an expression is valid without creating a full program.
func (p *Parser) ValidateExpression(expression string) error {
	_, err := p.Parse(expression)

	return err
}

// GetFunctionHelp returns documentation for available functions.
func (p *Parser) GetFunctionHelp() map[string]string {
	return map[string]string{
		"has_role(role)":            "Check if principal has a specific role",
		"has_any_role(roles)":       "Check if principal has any of the specified roles",
		"has_all_roles(roles)":      "Check if principal has all specified roles",
		"in_time_range(start, end)": "Check if current time is within range (HH:MM format)",
		"is_weekday()":              "Check if current day is Monday-Friday",
		"ip_in_range(cidrs)":        "Check if request IP is in CIDR ranges",
		"resource_matches(pattern)": "Match resource ID against wildcard pattern",
		"days_since(timestamp)":     "Calculate days since a timestamp",
		"hours_since(timestamp)":    "Calculate hours since a timestamp",
		"in_org(org_id)":            "Check if resource belongs to organization",
		"is_member_of(org_id)":      "Check if principal is member of organization",
	}
}

// ExpressionComplexity estimates the complexity of an expression (operation count).
func (p *Parser) ExpressionComplexity(ast *cel.Ast) int {
	// Simple heuristic: count nodes in the expression
	// Real implementation would walk the full AST
	checkedExpr, err := cel.AstToCheckedExpr(ast)
	if err != nil {
		// Fallback to parsed expr if checked expr fails
		parsedExpr, err := cel.AstToParsedExpr(ast)
		if err != nil {
			return 0
		}
		return estimateComplexityNode(parsedExpr.GetExpr(), 0)
	}

	return estimateComplexityNode(checkedExpr.GetExpr(), 0)
}

// estimateComplexityNode recursively counts operations in an expression.
func estimateComplexityNode(expr any, depth int) int {
	if expr == nil || depth > 50 {
		return 1
	}
	// Simple heuristic - just return a fixed estimate for now
	// Real implementation would walk the AST
	return 10
}

// ExampleExpressions expressions for testing.
var ExampleExpressions = map[string]string{
	"owner_only":      `resource.owner == principal.id`,
	"admin_or_owner":  `principal.roles.exists(r, r == "admin") || resource.owner == principal.id`,
	"business_hours":  `is_weekday()`, // Custom function that works
	"team_members":    `principal.team_id == resource.team_id`,
	"public_or_owner": `resource.visibility == "public" || resource.owner == principal.id`,
	"complex_abac":    `resource.confidentiality == "public" || (resource.confidentiality == "internal" && principal.org_id == resource.org_id) || resource.owner == principal.id`,
}
