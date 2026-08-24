// Package golang generates a typed Go client SDK from an AuthSome OpenAPI spec.
package golang

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
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

	for i := range files {
		formatted, fmtErr := gofmt(files[i].Content)
		if fmtErr != nil {
			return nil, fmt.Errorf("format %s: %w", files[i].Path, fmtErr)
		}

		files[i].Content = formatted
	}

	return files, nil
}

// gofmt runs the output through go/format, which is the same pass `gofmt`
// makes.
//
// A template cannot align a struct field against fields it has not rendered
// yet, and it writes whatever the surrounding text says even where Go has a
// shorter spelling for it: a lone return type comes out as `(error)` because
// the parentheses are in the template. CI checks the tree with goimports and
// fails on both. Formatting here means generated code is formatted the moment
// it is written, and nobody has to remember to run gofmt over sdk/ afterwards.
//
// A parse failure is returned, not swallowed. Unformattable output means the
// template produced something that is not Go, and writing it out to be
// discovered at build time helps nobody.
func gofmt(src string) (string, error) {
	out, err := format.Source([]byte(src))
	if err != nil {
		return "", err
	}

	return string(out), nil
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
	// schemaNames tracks type names emitted from schemas so templates can
	// skip duplicate request-type generation.
	schemaNames map[string]bool
}

// IsNewRequestType returns true if the operation's request type name
// doesn't collide with an existing schema type.
func (d *TemplateData) IsNewRequestType(opName string) bool {
	return !d.schemaNames[opName+"Request"]
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

// ParamDef represents a path or query parameter for a client method.
type ParamDef struct {
	Name   string // original name from spec ("orgId")
	GoName string // exported Go name ("OrgID")
	Type   string // Go type ("string")
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
	PathParams   []ParamDef
	QueryParams  []ParamDef
}

// HasPathParams returns true when the operation has path parameters.
func (o OperationDef) HasPathParams() bool { return len(o.PathParams) > 0 }

// HasQueryParams returns true when the operation has query parameters.
func (o OperationDef) HasQueryParams() bool { return len(o.QueryParams) > 0 }

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
						Name: exportedName(fn),
						Type: goType,
						// Normalize to snake_case: the Forge OpenAPI generator emits
						// PascalCase Go field names for request schemas instead of
						// reading json tags. The actual API uses snake_case.
						JSONTag:  normalizeJSONTag(fn) + omitempty,
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

			// Path and query parameters
			for _, param := range pair.op.Parameters {
				goType := "string"
				if param.Schema != nil {
					goType = g.schemaToGoType(param.Schema)
				}
				pd := ParamDef{
					Name:   param.Name,
					GoName: exportedName(param.Name),
					Type:   goType,
				}
				switch param.In {
				case "path":
					opDef.PathParams = append(opDef.PathParams, pd)
				case "query":
					opDef.QueryParams = append(opDef.QueryParams, pd)
				}
			}

			// Auth required?
			opDef.AuthRequired = g.operationRequiresAuth(pair.op)

			// Body fields
			if pair.op.RequestBody != nil {
				opDef.HasBody = true
				if schema := requestBodySchema(pair.op.RequestBody); schema != nil {
					opDef.BodyFields = g.schemaToGoFields(resolveSchemaRef(spec, schema))
				}
			}

			// Response type
			opDef.ResponseType = g.responseGoType(pair.op)

			data.Operations = append(data.Operations, opDef)
		}
	}

	// Build schema name index for dedup in templates.
	data.schemaNames = make(map[string]bool, len(data.Types))
	for _, t := range data.Types {
		data.schemaNames[t.Name] = true
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

// resolveSchemaRef follows a $ref to the component it names, so callers get a
// schema with properties on it rather than an empty shell.
//
// A request body may be written either way. Inline, its properties sit right
// there; as a $ref, the schema carries nothing but the reference. The request
// struct is named after the operation and the component is named after the Go
// type, so the two names usually differ (refreshTokens against RefreshRequest)
// and the struct cannot simply be dropped in favour of the component. Reading
// the fields through the reference is what lets the generated struct keep its
// own name and still have the fields in it.
//
// Anything that is not a component reference comes back untouched, including a
// dangling ref, which is left for the caller to notice rather than papered over
// with an empty struct.
func resolveSchemaRef(spec *openapi.Spec, s *openapi.Schema) *openapi.Schema {
	const prefix = "#/components/schemas/"

	seen := make(map[string]bool)

	for s != nil && s.Ref != "" {
		name, ok := strings.CutPrefix(s.Ref, prefix)
		if !ok || seen[name] || spec.Components == nil {
			return s
		}

		seen[name] = true

		target, found := spec.Components.Schemas[name]
		if !found || target == nil {
			return s
		}

		s = target
	}

	return s
}

// requestBodySchema picks the schema describing a request body.
//
// JSON wins when it is offered, which is all but a handful of routes. The rest
// are form-encoded: the OAuth2 endpoints take application/x-www-form-urlencoded
// because RFC 6749 says so, and forge describes them that way rather than as
// JSON. Reading only the JSON entry left those operations with a body the
// generator knew was there and no fields to put in it, so the request struct
// came out empty and the client sent "{}" over a body the server required.
//
// Remaining content types are considered in sorted order, so a route offering
// several does not generate a different struct from one run to the next.
func requestBodySchema(rb *openapi.RequestBody) *openapi.Schema {
	if ct, ok := rb.Content["application/json"]; ok && ct.Schema != nil {
		return ct.Schema
	}

	mediaTypes := make([]string, 0, len(rb.Content))
	for mt := range rb.Content {
		mediaTypes = append(mediaTypes, mt)
	}
	sort.Strings(mediaTypes)

	for _, mt := range mediaTypes {
		if ct := rb.Content[mt]; ct.Schema != nil {
			return ct.Schema
		}
	}

	return nil
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
			JSONTag:  normalizeJSONTag(fn) + omitempty,
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
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"unexport":  unexportedName,
		"snakecase": toSnakeCase,
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

// unexportedName returns a Go unexported (camelCase) version of a name.
// "OrgID" → "orgID", "MemberID" → "memberID", "Name" → "name"
func unexportedName(s string) string {
	if s == "" {
		return s
	}
	// Find the boundary between leading uppercase and the rest.
	// For "OrgID" → "orgID", for "Name" → "name", for "ID" → "id"
	runes := []rune(s)
	if len(runes) == 1 {
		return strings.ToLower(s)
	}
	// All-uppercase (like "ID") → all-lowercase
	allUpper := true
	for _, r := range runes {
		if r < 'A' || r > 'Z' {
			allUpper = false
			break
		}
	}
	if allUpper {
		return strings.ToLower(s)
	}
	// Leading uppercase followed by uppercase (acronym prefix): "OrgID" → "orgID"
	// Find where the lowercase starts
	i := 0
	for i < len(runes) && runes[i] >= 'A' && runes[i] <= 'Z' {
		i++
	}
	if i <= 1 {
		// Single leading uppercase: "Name" → "name"
		return strings.ToLower(string(runes[:1])) + string(runes[1:])
	}
	// Multi-uppercase prefix: lowercase all but the last uppercase char
	// "OrgID" (i=1 after "O", but runes[1]='r' is lower) — actually just lowercase first char
	return strings.ToLower(string(runes[:1])) + string(runes[1:])
}

// toSnakeCase converts PascalCase or camelCase to snake_case.
// "UserID" → "user_id", "AppID" → "app_id", "FirstName" → "first_name"
func toSnakeCase(s string) string {
	var result strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				// Insert underscore before uppercase if preceded by lowercase,
				// or if this starts a new word (uppercase followed by lowercase)
				prev := runes[i-1]
				if prev >= 'a' && prev <= 'z' {
					result.WriteRune('_')
				} else if i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z' && prev >= 'A' && prev <= 'Z' {
					result.WriteRune('_')
				}
			}
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// normalizeJSONTag ensures a JSON tag uses snake_case. If the name is already
// snake_case it passes through unchanged. PascalCase names (from Forge's
// OpenAPI generator reflecting Go field names) are converted.
func normalizeJSONTag(name string) string {
	// Already snake_case or lowercase — pass through
	hasUpper := false
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return name
	}
	return toSnakeCase(name)
}

// exportedName converts a snake_case or camelCase name to Go exported PascalCase.
func exportedName(s string) string {
	// Strip leading non-alpha characters (e.g. "$ref" → "ref")
	for s != "" && (s[0] < 'A' || (s[0] > 'Z' && s[0] < 'a') || s[0] > 'z') {
		s = s[1:]
	}
	if s == "" {
		return "Field"
	}

	// Split on underscores and hyphens
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})

	var result strings.Builder
	for _, part := range parts {
		if part == "" {
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
