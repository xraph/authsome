package notification

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// SimpleTemplateEngine implements TemplateEngine using Go's text/template.
type SimpleTemplateEngine struct{}

// NewSimpleTemplateEngine creates a new simple template engine.
func NewSimpleTemplateEngine() *SimpleTemplateEngine {
	return &SimpleTemplateEngine{}
}

// Render renders a template with variables.
func (e *SimpleTemplateEngine) Render(templateStr string, variables map[string]any) (string, error) {
	tmpl, err := template.New("notification").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// ValidateTemplate validates template syntax.
func (e *SimpleTemplateEngine) ValidateTemplate(templateStr string) error {
	_, err := template.New("validation").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("invalid template syntax: %w", err)
	}

	return nil
}

// ExtractVariables extracts variable names from template.
func (e *SimpleTemplateEngine) ExtractVariables(templateStr string) ([]string, error) {
	// Regular expression to match Go template variables: {{.Variable}} or {{.Variable.Field}}
	re := regexp.MustCompile(`\{\{\s*\.([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*)\s*\}\}`)
	matches := re.FindAllStringSubmatch(templateStr, -1)

	variableMap := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			// Extract the root variable name (before any dots)
			variable := strings.Split(match[1], ".")[0]
			variableMap[variable] = true
		}
	}

	// Convert map to slice
	variables := make([]string, 0, len(variableMap))
	for variable := range variableMap {
		variables = append(variables, variable)
	}

	return variables, nil
}

// Common template functions that can be used in templates.
var templateFuncs = template.FuncMap{
	"upper": strings.ToUpper,
	"lower": strings.ToLower,
	"title": strings.Title,
}

// RenderWithFuncs renders a template with variables and custom functions.
func (e *SimpleTemplateEngine) RenderWithFuncs(templateStr string, variables map[string]any) (string, error) {
	tmpl, err := template.New("notification").Funcs(templateFuncs).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
