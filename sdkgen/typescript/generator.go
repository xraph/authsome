// Package typescript generates a typed TypeScript client SDK from an AuthSome OpenAPI spec.
package typescript

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/xraph/authsome/sdkgen/openapi"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

// GeneratorConfig configures the TypeScript SDK generator.
type GeneratorConfig struct {
	// PackageName is the npm package name (default: "@authsome/client").
	PackageName string

	// PackageVersion is the npm package version (default: "0.5.0").
	PackageVersion string

	// OutputMode controls what files are generated.
	// "standalone" (default) — full npm package (types, client, index, package.json, tsconfig).
	// "embedded" — only api-types.ts and api-client.ts for embedding in another package.
	OutputMode string

	// MethodOverrides maps operationID to a custom method name.
	// e.g., {"refreshSession": "refresh", "deleteAccount": "deleteMe"}
	MethodOverrides map[string]string
}

// Generator produces TypeScript SDK code from an OpenAPI spec.
type Generator struct {
	config GeneratorConfig
}

// NewGenerator creates a TypeScript SDK generator.
func NewGenerator(cfg GeneratorConfig) *Generator {
	if cfg.PackageName == "" {
		cfg.PackageName = "@authsome/client"
	}
	if cfg.PackageVersion == "" {
		cfg.PackageVersion = "0.5.0"
	}
	if cfg.OutputMode == "" {
		cfg.OutputMode = "standalone"
	}
	if cfg.MethodOverrides == nil {
		cfg.MethodOverrides = make(map[string]string)
	}
	return &Generator{config: cfg}
}

// GeneratedFile represents a single generated file.
type GeneratedFile struct {
	Path    string
	Content string
}

// Generate produces all TypeScript SDK files from the given spec.
func (g *Generator) Generate(spec *openapi.Spec) ([]GeneratedFile, error) {
	data := g.buildTemplateData(spec)

	files := []GeneratedFile{}

	if g.config.OutputMode == "embedded" {
		// Embedded mode: only generate api-types.ts and api-client.ts
		content, err := g.renderTemplate("templates/types.ts.tmpl", data)
		if err != nil {
			return nil, fmt.Errorf("render api-types.ts: %w", err)
		}
		files = append(files, GeneratedFile{Path: "api-types.ts", Content: content})

		content, err = g.renderTemplate("templates/client.ts.tmpl", data)
		if err != nil {
			return nil, fmt.Errorf("render api-client.ts: %w", err)
		}
		files = append(files, GeneratedFile{Path: "api-client.ts", Content: content})

		return files, nil
	}

	// Standalone mode: generate full package
	content, err := g.renderTemplate("templates/types.ts.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render types.ts: %w", err)
	}
	files = append(files, GeneratedFile{Path: "src/types.ts", Content: content})

	content, err = g.renderTemplate("templates/client.ts.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render client.ts: %w", err)
	}
	files = append(files, GeneratedFile{Path: "src/client.ts", Content: content})

	content, err = g.renderTemplate("templates/index.ts.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render index.ts: %w", err)
	}
	files = append(files, GeneratedFile{Path: "src/index.ts", Content: content})

	content, err = g.renderTemplate("templates/package.json.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render package.json: %w", err)
	}
	files = append(files, GeneratedFile{Path: "package.json", Content: content})

	content, err = g.renderTemplate("templates/tsconfig.json.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render tsconfig.json: %w", err)
	}
	files = append(files, GeneratedFile{Path: "tsconfig.json", Content: content})

	return files, nil
}

// TemplateData holds all data passed to templates.
type TemplateData struct {
	PackageName        string
	PackageVersion     string
	OutputMode         string
	Types              []TypeDef
	Operations         []OperationDef
	RequestTypeImports []string // Additional request type names to import (not already in Types)
	HasSocial          bool
	HasMagicLink       bool
	HasMFA             bool
}

// TypeDef represents a generated TypeScript interface.
type TypeDef struct {
	Name       string
	Fields     []FieldDef
	IsResponse bool
}

// FieldDef represents a field in a TypeScript interface.
type FieldDef struct {
	Name     string
	Type     string
	Optional bool
}

// PathParamDef represents a path parameter in an operation.
type PathParamDef struct {
	Name   string // Original name from the spec (e.g., "orgId")
	TSName string // TypeScript parameter name (e.g., "orgId")
}

// QueryParamDef represents a query parameter in an operation.
type QueryParamDef struct {
	Name     string // Original name from the spec (e.g., "limit")
	TSName   string // TypeScript parameter name (e.g., "limit")
	Type     string // TypeScript type (e.g., "number")
	Required bool
}

// OperationDef represents a generated client method.
type OperationDef struct {
	Name              string
	Method            string
	Path              string
	Summary           string
	HasBody           bool
	BodyType          string
	ResponseType      string
	AuthRequired      bool
	Tags              []string
	PathParams        []PathParamDef
	QueryParams       []QueryParamDef
	NeedsRequestAlias bool // true when the request type alias differs from an existing schema type
}

func (g *Generator) buildTemplateData(spec *openapi.Spec) *TemplateData {
	data := &TemplateData{
		PackageName:    g.config.PackageName,
		PackageVersion: g.config.PackageVersion,
		OutputMode:     g.config.OutputMode,
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
			td := TypeDef{
				Name:       name,
				IsResponse: strings.HasSuffix(name, "Response"),
			}
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
					td.Fields = append(td.Fields, FieldDef{
						Name:     fn,
						Type:     g.schemaToTSType(fs),
						Optional: !requiredSet[fn],
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

			// Resolve method name (apply overrides)
			methodName := pair.op.OperationID
			if override, ok := g.config.MethodOverrides[methodName]; ok {
				methodName = override
			}

			opDef := OperationDef{
				Name:    methodName,
				Method:  pair.method,
				Path:    path,
				Summary: pair.op.Summary,
				Tags:    pair.op.Tags,
			}

			// Determine if auth is required
			opDef.AuthRequired = g.operationRequiresAuth(pair.op)

			// Extract path and query parameters
			for _, param := range pair.op.Parameters {
				switch param.In {
				case "path":
					opDef.PathParams = append(opDef.PathParams, PathParamDef{
						Name:   param.Name,
						TSName: param.Name,
					})
				case "query":
					tsType := "string"
					if param.Schema != nil {
						tsType = g.schemaToTSType(param.Schema)
					}
					opDef.QueryParams = append(opDef.QueryParams, QueryParamDef{
						Name:     param.Name,
						TSName:   param.Name,
						Type:     tsType,
						Required: param.Required,
					})
				}
			}

			// Determine body type
			if pair.op.RequestBody != nil {
				opDef.HasBody = true
				if ct, ok := pair.op.RequestBody.Content["application/json"]; ok && ct.Schema != nil {
					opDef.BodyType = g.schemaToTSInlineType(ct.Schema)
				} else {
					opDef.BodyType = "Record<string, unknown>"
				}
			}

			// Determine response type
			opDef.ResponseType = g.responseType(pair.op)

			data.Operations = append(data.Operations, opDef)
		}
	}

	// Build a set of schema type names to detect collisions with request type aliases.
	schemaNames := make(map[string]bool, len(data.Types))
	for _, t := range data.Types {
		schemaNames[t.Name] = true
	}

	// Compute NeedsRequestAlias for each operation and build deduplicated
	// RequestTypeImports list for the client template.
	seen := make(map[string]bool)
	for i := range data.Operations {
		op := &data.Operations[i]
		if op.HasBody {
			reqTypeName := pascalCase(op.Name) + "Request"
			// The alias is needed only when it would produce a different name
			// than an already-declared schema interface AND the body type is not
			// a self-reference (alias == body type).
			op.NeedsRequestAlias = !schemaNames[reqTypeName] || reqTypeName != op.BodyType
			// But if the alias name collides with a schema interface, we must
			// skip the alias entirely to avoid duplicate identifiers.
			if schemaNames[reqTypeName] {
				op.NeedsRequestAlias = false
			}
			// Import the request type only if it's not already a schema type.
			if !schemaNames[reqTypeName] && !seen[reqTypeName] {
				data.RequestTypeImports = append(data.RequestTypeImports, reqTypeName)
				seen[reqTypeName] = true
			}
		}

		// Detect plugins
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

func (g *Generator) schemaToTSType(s *openapi.Schema) string {
	if s.Ref != "" {
		// Extract type name from $ref
		parts := strings.Split(s.Ref, "/")
		return parts[len(parts)-1]
	}

	switch s.Type {
	case "string":
		return "string"
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		if s.Items != nil {
			return g.schemaToTSType(s.Items) + "[]"
		}
		return "unknown[]"
	case "object":
		return "Record<string, unknown>"
	default:
		return "unknown"
	}
}

func (g *Generator) schemaToTSInlineType(s *openapi.Schema) string {
	if s.Ref != "" {
		parts := strings.Split(s.Ref, "/")
		return parts[len(parts)-1]
	}

	if s.Type != "object" || s.Properties == nil {
		return "Record<string, unknown>"
	}

	// Build inline object type
	var fields []string
	fieldNames := make([]string, 0, len(s.Properties))
	for fn := range s.Properties {
		fieldNames = append(fieldNames, fn)
	}
	sort.Strings(fieldNames)

	requiredSet := make(map[string]bool)
	for _, r := range s.Required {
		requiredSet[r] = true
	}

	for _, fn := range fieldNames {
		fs := s.Properties[fn]
		opt := ""
		if !requiredSet[fn] {
			opt = "?"
		}
		fields = append(fields, fmt.Sprintf("%s%s: %s", fn, opt, g.schemaToTSType(fs)))
	}

	return "{ " + strings.Join(fields, "; ") + " }"
}

func (g *Generator) responseType(op *openapi.Operation) string {
	// Check 200 or 201 response
	for _, code := range []string{"200", "201"} {
		resp, ok := op.Responses[code]
		if !ok || resp == nil {
			continue
		}
		ct, ok := resp.Content["application/json"]
		if !ok || ct.Schema == nil {
			continue
		}
		return g.schemaToTSType(ct.Schema)
	}
	return "void"
}

func (g *Generator) renderTemplate(name string, data *TemplateData) (string, error) {
	funcMap := template.FuncMap{
		"lower":        strings.ToLower,
		"camelToKebab": camelToKebab,
		"pascalCase":   pascalCase,
		"hasTag": func(tags []string, tag string) bool {
			for _, t := range tags {
				if t == tag {
					return true
				}
			}
			return false
		},
		"hasPathParams": func(params []PathParamDef) bool {
			return len(params) > 0
		},
		"hasQueryParams": func(params []QueryParamDef) bool {
			return len(params) > 0
		},
		"buildPath": func(path string, params []PathParamDef) string {
			if len(params) == 0 {
				return `"` + path + `"`
			}
			result := path
			for _, p := range params {
				result = strings.ReplaceAll(result, "{"+p.Name+"}", "${"+p.TSName+"}")
			}
			return "`" + result + "`"
		},
		"buildParams": func(op OperationDef, outputMode string) string {
			var parts []string
			// 1. Path params (always required)
			for _, p := range op.PathParams {
				parts = append(parts, p.TSName+": string")
			}
			// 2. Body (always required when present)
			if op.HasBody {
				parts = append(parts, "body: "+pascalCase(op.Name)+"Request")
			}
			// 3. Required query params
			for _, q := range op.QueryParams {
				if q.Required {
					parts = append(parts, q.TSName+": "+q.Type)
				}
			}
			// 4. Token (required when auth needed in embedded mode)
			if outputMode == "embedded" && op.AuthRequired {
				parts = append(parts, "token: string")
			}
			// 5. Optional query params (must come last)
			for _, q := range op.QueryParams {
				if !q.Required {
					parts = append(parts, q.TSName+"?: "+q.Type)
				}
			}
			return strings.Join(parts, ", ")
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS, name)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	// Execute the named template (filename from the pattern)
	parts := strings.Split(name, "/")
	tmplName := parts[len(parts)-1]
	if err := tmpl.ExecuteTemplate(&buf, tmplName, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// camelToKebab converts camelCase to kebab-case.
func camelToKebab(s string) string {
	var result []byte
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result = append(result, '-')
			}
			result = append(result, byte(r-'A'+'a')) //nolint:gosec // G115: safe, r is in [A-Z]
		} else {
			result = append(result, byte(r)) //nolint:gosec // G115: safe cast, ASCII range
		}
	}
	return string(result)
}

// pascalCase converts a camelCase string to PascalCase.
func pascalCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
