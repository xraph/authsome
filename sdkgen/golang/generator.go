// Package golang generates a typed Go client SDK from an AuthSome OpenAPI spec.
package golang

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/xraph/authsome/sdkgen/openapi"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// GeneratorConfig configures the Go SDK generator.
type GeneratorConfig struct {
	// PackageName is the Go package name (default: "authclient").
	PackageName string

	// ModulePath is the Go module path (default: "github.com/xraph/authsome/client").
	ModulePath string
}

// Generator produces Go SDK code from an OpenAPI spec.
type Generator struct {
	config GeneratorConfig
}

// NewGenerator creates a Go SDK generator.
func NewGenerator(cfg GeneratorConfig) *Generator {
	if cfg.PackageName == "" {
		cfg.PackageName = "authclient"
	}
	if cfg.ModulePath == "" {
		cfg.ModulePath = "github.com/xraph/authsome/client"
	}
	return &Generator{config: cfg}
}

// GeneratedFile represents a single generated file.
type GeneratedFile struct {
	Path    string
	Content string
}

// Generate produces all Go SDK files from the given spec.
func (g *Generator) Generate(spec *openapi.Spec) ([]GeneratedFile, error) {
	data := g.buildTemplateData(spec)

	var files []GeneratedFile

	// types.go
	content, err := g.renderTemplate("templates/types.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render types.go: %w", err)
	}
	files = append(files, GeneratedFile{Path: "types.go", Content: content})

	// client.go
	content, err = g.renderTemplate("templates/client.go.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render client.go: %w", err)
	}
	files = append(files, GeneratedFile{Path: "client.go", Content: content})

	return files, nil
}

// TemplateData holds all data passed to templates.
type TemplateData struct {
	PackageName  string
	ModulePath   string
	Types        []TypeDef
	Operations   []OperationDef
	HasSocial    bool
	HasMagicLink bool
	HasMFA       bool
}

// TypeDef represents a generated Go struct.
type TypeDef struct {
	Name   string
	Fields []FieldDef
}

// FieldDef represents a field in a Go struct.
type FieldDef struct {
	Name     string
	Type     string
	JSONTag  string
	Optional bool
}

// OperationDef represents a generated client method.
type OperationDef struct {
	Name         string
	Method       string
	Path         string
	Summary      string
	HasBody      bool
	BodyFields   []FieldDef
	ResponseType string
	AuthRequired bool
	Tags         []string
}

func (g *Generator) buildTemplateData(spec *openapi.Spec) *TemplateData {
	data := &TemplateData{
		PackageName: g.config.PackageName,
		ModulePath:  g.config.ModulePath,
	}

	// Build types from components/schemas
	if spec.Components != nil {
		schemaNames := make([]string, 0, len(spec.Components.Schemas))
		for name := range spec.Components.Schemas {
			schemaNames = append(schemaNames, name)
		}
		sort.Strings(schemaNames)

		for _, name := range schemaNames {
			schema := spec.Components.Schemas[name]
			td := TypeDef{Name: name}
			if schema.Properties != nil {
				fieldNames := make([]string, 0, len(schema.Properties))
				for fn := range schema.Properties {
					fieldNames = append(fieldNames, fn)
				}
				sort.Strings(fieldNames)

				requiredSet := make(map[string]bool)
				for _, r := range schema.Required {
					requiredSet[r] = true
				}

				for _, fn := range fieldNames {
					fs := schema.Properties[fn]
					goType := g.schemaToGoType(fs)
					optional := !requiredSet[fn]
					omitempty := ""
					if optional {
						omitempty = ",omitempty"
					}
					td.Fields = append(td.Fields, FieldDef{
						Name:     exportedName(fn),
						Type:     goType,
						JSONTag:  fn + omitempty,
						Optional: optional,
					})
				}
			}
			data.Types = append(data.Types, td)
		}
	}

	// Build operations from paths
	paths := make([]string, 0, len(spec.Paths))
	for p := range spec.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		pathItem := spec.Paths[path]
		for _, pair := range []struct {
			method string
			op     *openapi.Operation
		}{
			{"GET", pathItem.Get},
			{"POST", pathItem.Post},
			{"PUT", pathItem.Put},
			{"PATCH", pathItem.Patch},
			{"DELETE", pathItem.Delete},
		} {
			if pair.op == nil || pair.op.OperationID == "" {
				continue
			}

			opDef := OperationDef{
				Name:    exportedName(pair.op.OperationID),
				Method:  pair.method,
				Path:    path,
				Summary: pair.op.Summary,
				Tags:    pair.op.Tags,
			}

			// Auth required?
			opDef.AuthRequired = g.operationRequiresAuth(pair.op)

			// Body fields
			if pair.op.RequestBody != nil {
				opDef.HasBody = true
				if ct, ok := pair.op.RequestBody.Content["application/json"]; ok && ct.Schema != nil {
					opDef.BodyFields = g.schemaToGoFields(ct.Schema)
				}
			}

			// Response type
			opDef.ResponseType = g.responseGoType(pair.op)

			data.Operations = append(data.Operations, opDef)
		}
	}

	// Detect plugins
	for _, op := range data.Operations {
		for _, tag := range op.Tags {
			switch tag {
			case "Social":
				data.HasSocial = true
			case "Magic Link":
				data.HasMagicLink = true
			case "MFA":
				data.HasMFA = true
			}
		}
	}

	return data
}

func (g *Generator) operationRequiresAuth(op *openapi.Operation) bool {
	if len(op.Security) == 0 {
		return false
	}
	for _, sec := range op.Security {
		if _, ok := sec["bearerAuth"]; ok {
			return true
		}
	}
	return false
}

func (g *Generator) schemaToGoType(s *openapi.Schema) string {
	if s.Ref != "" {
		parts := strings.Split(s.Ref, "/")
		return "*" + parts[len(parts)-1]
	}

	switch s.Type {
	case "string":
		return "string"
	case "integer":
		return "int64"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		if s.Items != nil {
			return "[]" + g.schemaToGoType(s.Items)
		}
		return "[]any"
	case "object":
		return "map[string]any"
	default:
		return "any"
	}
}

func (g *Generator) schemaToGoFields(s *openapi.Schema) []FieldDef {
	if s.Properties == nil {
		return nil
	}

	fieldNames := make([]string, 0, len(s.Properties))
	for fn := range s.Properties {
		fieldNames = append(fieldNames, fn)
	}
	sort.Strings(fieldNames)

	requiredSet := make(map[string]bool)
	for _, r := range s.Required {
		requiredSet[r] = true
	}

	var fields []FieldDef
	for _, fn := range fieldNames {
		fs := s.Properties[fn]
		optional := !requiredSet[fn]
		omitempty := ""
		if optional {
			omitempty = ",omitempty"
		}
		fields = append(fields, FieldDef{
			Name:     exportedName(fn),
			Type:     g.schemaToGoType(fs),
			JSONTag:  fn + omitempty,
			Optional: optional,
		})
	}
	return fields
}

func (g *Generator) responseGoType(op *openapi.Operation) string {
	for _, code := range []string{"200", "201"} {
		resp, ok := op.Responses[code]
		if !ok || resp == nil {
			continue
		}
		ct, ok := resp.Content["application/json"]
		if !ok || ct.Schema == nil {
			continue
		}
		t := g.schemaToGoType(ct.Schema)
		// Remove leading * for response types since they're returned as values
		t = strings.TrimPrefix(t, "*")
		return t
	}
	return ""
}

func (g *Generator) renderTemplate(name string, data *TemplateData) (string, error) {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"unexport": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return strings.ToLower(s[:1]) + s[1:]
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, name)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	parts := strings.Split(name, "/")
	tmplName := parts[len(parts)-1]
	if err := tmpl.ExecuteTemplate(&buf, tmplName, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// exportedName converts a snake_case or camelCase name to Go exported PascalCase.
func exportedName(s string) string {
	// Split on underscores and hyphens
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})

	var result strings.Builder
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}
		// Known acronyms
		upper := strings.ToUpper(part)
		switch upper {
		case "ID", "URL", "URI", "API", "HTTP", "HTTPS", "JSON", "XML", "HTML", "CSS", "SQL", "MFA", "JWT", "IP":
			result.WriteString(upper)
		default:
			result.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return result.String()
}
