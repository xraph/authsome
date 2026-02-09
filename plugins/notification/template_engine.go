package notification

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// TemplateEngine provides template rendering functionality.
type TemplateEngine struct {
	funcMap template.FuncMap
}

// NewTemplateEngine creates a new template engine.
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		funcMap: template.FuncMap{
			"upper":    strings.ToUpper,
			"lower":    strings.ToLower,
			"title":    strings.Title,
			"trim":     strings.TrimSpace,
			"truncate": truncate,
			"default":  defaultValue,
		},
	}
}

// Render renders a template with the given variables.
func (e *TemplateEngine) Render(templateStr string, variables map[string]any) (string, error) {
	tmpl, err := template.New("notification").Funcs(e.funcMap).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// ValidateTemplate validates a template for syntax errors.
func (e *TemplateEngine) ValidateTemplate(templateStr string) error {
	_, err := template.New("validation").Funcs(e.funcMap).Parse(templateStr)

	return err
}

// ExtractVariables extracts variable names from a template.
func (e *TemplateEngine) ExtractVariables(templateStr string) ([]string, error) {
	// Simple regex to find {{.VarName}} patterns
	re := regexp.MustCompile(`\{\{\s*\.(\w+)\s*\}\}`)
	matches := re.FindAllStringSubmatch(templateStr, -1)

	varMap := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			varMap[match[1]] = true
		}
	}

	// Also check for function calls like {{upper .VarName}}
	reFn := regexp.MustCompile(`\{\{\s*\w+\s+\.(\w+)\s*\}\}`)
	matchesFn := reFn.FindAllStringSubmatch(templateStr, -1)

	for _, match := range matchesFn {
		if len(match) > 1 {
			varMap[match[1]] = true
		}
	}

	variables := make([]string, 0, len(varMap))
	for v := range varMap {
		variables = append(variables, v)
	}

	return variables, nil
}

// Helper functions for templates

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}

	return s[:length] + "..."
}

func defaultValue(defaultVal, val any) any {
	if val == nil || val == "" {
		return defaultVal
	}

	return val
}
