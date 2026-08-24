// Package dart generates a typed Dart client SDK from an AuthSome OpenAPI spec.
package dart

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

// GeneratorConfig configures the Dart SDK generator.
type GeneratorConfig struct {
	// PackageName is the Dart package name (default: "authsome_core").
	PackageName string

	// PackageVersion is the Dart package version (default: "0.1.0").
	PackageVersion string

	// OutputMode controls what files are generated.
	// "standalone" (default) — full Dart package (types, client, pubspec).
	// "embedded" — only api_types.dart and api_client.dart for embedding.
	OutputMode string

	// MethodOverrides maps operationID to a custom method name.
	MethodOverrides map[string]string
}

// Generator produces Dart SDK code from an OpenAPI spec.
type Generator struct {
	config GeneratorConfig
	spec   *openapi.Spec // Set during Generate() for $ref resolution.
}

// NewGenerator creates a Dart SDK generator.
func NewGenerator(cfg GeneratorConfig) *Generator {
	if cfg.PackageName == "" {
		cfg.PackageName = "authsome_core"
	}
	if cfg.PackageVersion == "" {
		cfg.PackageVersion = "0.1.0"
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

// Generate produces all Dart SDK files from the given spec.
func (g *Generator) Generate(spec *openapi.Spec) ([]GeneratedFile, error) {
	g.spec = spec // Store for $ref resolution.
	data := g.buildTemplateData(spec)

	files := []GeneratedFile{}

	if g.config.OutputMode == "embedded" {
		content, err := g.renderTemplate("templates/types.dart.tmpl", data)
		if err != nil {
			return nil, fmt.Errorf("render api_types.dart: %w", err)
		}
		files = append(files, GeneratedFile{Path: "api_types.dart", Content: content})

		content, err = g.renderTemplate("templates/client.dart.tmpl", data)
		if err != nil {
			return nil, fmt.Errorf("render api_client.dart: %w", err)
		}
		files = append(files, GeneratedFile{Path: "api_client.dart", Content: content})

		return files, nil
	}

	// Standalone mode: generate full package
	content, err := g.renderTemplate("templates/types.dart.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render types.dart: %w", err)
	}
	files = append(files, GeneratedFile{Path: "lib/src/types.dart", Content: content})

	content, err = g.renderTemplate("templates/client.dart.tmpl", data)
	if err != nil {
		return nil, fmt.Errorf("render client.dart: %w", err)
	}
	files = append(files, GeneratedFile{Path: "lib/src/client.dart", Content: content})

	return files, nil
}

// ── Template Data ───────────────────────────────────

// TemplateData holds all data passed to templates.
type TemplateData struct {
	PackageName    string
	PackageVersion string
	OutputMode     string
	Types          []TypeDef
	Operations     []OperationDef
}

// TypeDef represents a generated Dart class.
type TypeDef struct {
	Name   string
	Fields []FieldDef
}

// FieldDef represents a field in a Dart class.
type FieldDef struct {
	Name     string // Dart field name (camelCase)
	JSONName string // Original JSON key
	Type     string // Dart type
	Optional bool
}

// PathParamDef represents a path parameter in an operation.
type PathParamDef struct {
	Name     string // Original name from the spec
	DartName string // Dart parameter name (camelCase)
}

// QueryParamDef represents a query parameter in an operation.
type QueryParamDef struct {
	Name     string // Original name from the spec
	DartName string // Dart parameter name
	Type     string // Dart type
	Required bool
}

// OperationDef represents a generated client method.
type OperationDef struct {
	Name         string
	Method       string
	Path         string
	Summary      string
	HasBody      bool
	BodyType     string // Dart type of the request body
	ResponseType string // Dart type of the response
	AuthRequired bool
	PathParams   []PathParamDef
	QueryParams  []QueryParamDef
}

// ── Build Template Data ─────────────────────────────

func (g *Generator) buildTemplateData(spec *openapi.Spec) *TemplateData {
	data := &TemplateData{
		PackageName:    g.config.PackageName,
		PackageVersion: g.config.PackageVersion,
		OutputMode:     g.config.OutputMode,
	}

	// Names already claimed by a generated class. Request classes are named
	// after their operation and checked against this so a synthesized one never
	// collides with a component that got there first.
	declared := make(map[string]bool)

	// Build types from components/schemas
	if spec.Components != nil {
		schemaNames := make([]string, 0, len(spec.Components.Schemas))
		for name := range spec.Components.Schemas {
			schemaNames = append(schemaNames, name)
		}
		sort.Strings(schemaNames)

		for _, name := range schemaNames {
			schema := spec.Components.Schemas[name]

			// Skip primitive type aliases (e.g., ID = string) — they are
			// resolved inline by schemaToDartType and don't need a class.
			if g.isPrimitiveAlias(name) {
				continue
			}

			// Skip types whose names are Dart reserved words or conflict
			// with dart:core types (e.g., Object, Type, Function).
			if isDartReservedTypeName(name) {
				continue
			}

			td := TypeDef{Name: name, Fields: g.schemaToDartFields(schema)}
			data.Types = append(data.Types, td)
			declared[name] = true
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
			}

			// Determine if auth is required
			opDef.AuthRequired = g.operationRequiresAuth(pair.op)

			// Extract path and query parameters
			for _, param := range pair.op.Parameters {
				switch param.In {
				case "path":
					opDef.PathParams = append(opDef.PathParams, PathParamDef{
						Name:     param.Name,
						DartName: toCamelCase(param.Name),
					})
				case "query":
					dartType := "String"
					if param.Schema != nil {
						dartType = g.schemaToDartType(param.Schema)
					}
					opDef.QueryParams = append(opDef.QueryParams, QueryParamDef{
						Name:     param.Name,
						DartName: toCamelCase(param.Name),
						Type:     dartType,
						Required: param.Required,
					})
				}
			}

			// Determine body type
			if pair.op.RequestBody != nil {
				opDef.HasBody = true
				opDef.BodyType = g.bodyType(pair.op, data, declared)
			}

			// Determine response type
			opDef.ResponseType = g.responseType(pair.op)

			data.Operations = append(data.Operations, opDef)
		}
	}

	return data
}

// ── Type Conversion ────────────────────────────────

// isPrimitiveAlias checks if a schema name refers to a type alias for a primitive.
func (g *Generator) isPrimitiveAlias(name string) bool {
	if g.spec == nil || g.spec.Components == nil {
		return false
	}
	schema, ok := g.spec.Components.Schemas[name]
	if !ok {
		return false
	}
	if len(schema.Properties) > 0 {
		return false
	}
	switch schema.Type {
	case "string", "integer", "number", "boolean":
		return true
	}
	return false
}

func (g *Generator) schemaToDartType(s *openapi.Schema) string {
	if s.Ref != "" {
		parts := strings.Split(s.Ref, "/")
		refName := parts[len(parts)-1]

		// Skip reserved type names — map to their Dart equivalents.
		if isDartReservedTypeName(refName) {
			if g.spec != nil && g.spec.Components != nil {
				if refSchema, ok := g.spec.Components.Schemas[refName]; ok {
					if refSchema.Type == "object" {
						return "Map<String, dynamic>"
					}
				}
			}
			return "dynamic"
		}

		// Resolve primitive type aliases (e.g., ID → String).
		if g.spec != nil && g.spec.Components != nil {
			if refSchema, ok := g.spec.Components.Schemas[refName]; ok {
				if len(refSchema.Properties) == 0 {
					switch refSchema.Type {
					case "string":
						return "String"
					case "integer":
						return "int"
					case "number":
						return "double"
					case "boolean":
						return "bool"
					case "object":
						return "Map<String, dynamic>"
					}
				}
			}
		}
		return refName
	}

	switch s.Type {
	case "string":
		return "String"
	case "integer":
		return "int"
	case "number":
		if s.Format == "int64" || s.Format == "int32" {
			return "int"
		}
		return "double"
	case "boolean":
		return "bool"
	case "array":
		if s.Items != nil {
			return "List<" + g.schemaToDartType(s.Items) + ">"
		}
		return "List<dynamic>"
	case "object":
		return "Map<String, dynamic>"
	default:
		return "dynamic"
	}
}

// requestBodySchema picks the schema describing a request body.
//
// JSON wins when it is offered, which is all but a handful of routes. The rest
// are form-encoded: the OAuth2 endpoints take application/x-www-form-urlencoded
// because RFC 6749 says so, and forge describes them that way rather than as
// JSON. Reading only the JSON entry threw away whatever name the spec gave
// those bodies and left the caller with an untyped map.
//
// Remaining content types are considered in sorted order, so a route offering
// several does not generate a different signature from one run to the next.
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

// schemaToDartFields turns a schema's properties into class fields, sorted by
// property name so the same schema always produces the same class.
func (g *Generator) schemaToDartFields(schema *openapi.Schema) []FieldDef {
	if schema == nil || schema.Properties == nil {
		return nil
	}

	fieldNames := make([]string, 0, len(schema.Properties))
	for fn := range schema.Properties {
		fieldNames = append(fieldNames, fn)
	}
	sort.Strings(fieldNames)

	requiredSet := make(map[string]bool)
	for _, r := range schema.Required {
		requiredSet[r] = true
	}

	fields := make([]FieldDef, 0, len(fieldNames))
	for _, fn := range fieldNames {
		fs := schema.Properties[fn]
		fields = append(fields, FieldDef{
			Name:     safeDartFieldName(toCamelCase(fn)),
			JSONName: fn,
			Type:     g.schemaToDartType(fs),
			Optional: !requiredSet[fn],
		})
	}

	return fields
}

// bodyType names the Dart type a request body is passed as, generating a class
// for it when the schema is written inline.
//
// A $ref already names something, and the class for it comes from the
// components pass. An inline schema names nothing, and every field the endpoint
// takes used to be thrown away in favour of Map<String, dynamic>. The four
// OAuth2 routes are the whole population of that case: forge writes their
// bodies inline because they are form-encoded. So the schema is turned into a
// class of its own, named after the operation the way the Go and TypeScript
// generators name theirs.
//
// The name is derived from the operation ID rather than the configured method
// name, so an embedded build with method overrides still produces the same
// class names as a standalone one.
func (g *Generator) bodyType(op *openapi.Operation, data *TemplateData, declared map[string]bool) string {
	const fallback = "Map<String, dynamic>"

	schema := requestBodySchema(op.RequestBody)
	if schema == nil {
		return fallback
	}

	if schema.Ref != "" {
		return g.schemaToDartBodyType(schema)
	}

	fields := g.schemaToDartFields(schema)
	if len(fields) == 0 {
		return fallback
	}

	name := exportedDartName(op.OperationID) + "Request"
	if name == "Request" || isDartReservedTypeName(name) {
		return fallback
	}

	// A component already holding the name has its own class; use that rather
	// than emitting a second one under the same name.
	if declared[name] {
		return name
	}

	data.Types = append(data.Types, TypeDef{Name: name, Fields: fields})
	declared[name] = true

	return name
}

// exportedDartName upper-cases the first letter of an operation ID, which is
// already camelCase, giving the PascalCase a class name needs.
func exportedDartName(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func (g *Generator) schemaToDartBodyType(s *openapi.Schema) string {
	if s.Ref != "" {
		parts := strings.Split(s.Ref, "/")
		return parts[len(parts)-1]
	}
	return "Map<String, dynamic>"
}

func (g *Generator) responseType(op *openapi.Operation) string {
	for _, code := range []string{"200", "201"} {
		resp, ok := op.Responses[code]
		if !ok || resp == nil {
			continue
		}
		ct, ok := resp.Content["application/json"]
		if !ok || ct.Schema == nil {
			continue
		}
		return g.schemaToDartType(ct.Schema)
	}
	return "void"
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

// ── Template Rendering ─────────────────────────────

func (g *Generator) renderTemplate(name string, data *TemplateData) (string, error) {
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"dartType": func(t string, optional bool) string {
			if optional {
				return t + "?"
			}
			return t
		},
		"hasPathParams": func(params []PathParamDef) bool {
			return len(params) > 0
		},
		"hasQueryParams": func(params []QueryParamDef) bool {
			return len(params) > 0
		},
		"buildDartPath": func(path string, params []PathParamDef) string {
			if len(params) == 0 {
				return "'" + path + "'"
			}
			result := path
			for _, p := range params {
				result = strings.ReplaceAll(result, "{"+p.Name+"}", "$"+p.DartName)
			}
			return "'" + result + "'"
		},
		"buildDartParams": func(op OperationDef) string {
			var parts []string
			// Path params
			for _, p := range op.PathParams {
				parts = append(parts, "required String "+p.DartName)
			}
			// Body
			if op.HasBody {
				parts = append(parts, "required "+op.BodyType+" body")
			}
			// Required query params
			for _, q := range op.QueryParams {
				if q.Required {
					parts = append(parts, "required "+q.Type+" "+q.DartName)
				}
			}
			// Token
			if op.AuthRequired {
				parts = append(parts, "required String token")
			}
			// Optional query params
			for _, q := range op.QueryParams {
				if !q.Required {
					parts = append(parts, q.Type+"? "+q.DartName)
				}
			}
			if len(parts) == 0 {
				return ""
			}
			return "{" + strings.Join(parts, ", ") + "}"
		},
		"isVoid": func(t string) bool {
			return t == "void"
		},
		"isMapType": func(t string) bool {
			return t == "Map<String, dynamic>"
		},
		"isListType": func(t string) bool {
			return strings.HasPrefix(t, "List<")
		},
		"listItemType": func(t string) string {
			// Extract inner type from List<T>
			return t[5 : len(t)-1]
		},
		"isPrimitiveType": func(t string) bool {
			switch t {
			case "String", "int", "double", "bool", "dynamic":
				return true
			}
			return false
		},
		"fromJsonExpr":     buildFromJSONExpr,
		"toJsonExpr":       buildToJSONExpr,
		"responseFromJson": buildResponseFromJSON,
		"dartStringEscape": func(s string) string {
			return strings.ReplaceAll(s, "$", `\$`)
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

// ── Helper Functions ───────────────────────────────

// isDartReservedTypeName returns true if the given name conflicts with
// Dart built-in types or reserved words that cannot be used as class names.
func isDartReservedTypeName(name string) bool {
	reserved := map[string]bool{
		"Object":   true,
		"Type":     true,
		"Function": true,
		"Null":     true,
		"Future":   true,
		"Stream":   true,
		"Iterable": true,
		"Iterator": true,
		"Map":      true,
		"List":     true,
		"Set":      true,
		"String":   true,
		"int":      true,
		"double":   true,
		"bool":     true,
		"dynamic":  true,
		"void":     true,
		"num":      true,
		"Never":    true,
		"Enum":     true,
		"Record":   true,
	}
	return reserved[name]
}

// isDartReservedKeyword returns true if the given name is a Dart language
// keyword that cannot be used as a field or variable identifier.
func isDartReservedKeyword(name string) bool {
	keywords := map[string]bool{
		"abstract": true, "as": true, "assert": true, "async": true,
		"await": true, "break": true, "case": true, "catch": true,
		"class": true, "const": true, "continue": true, "covariant": true,
		"default": true, "deferred": true, "do": true, "dynamic": true,
		"else": true, "enum": true, "export": true, "extends": true,
		"extension": true, "external": true, "factory": true, "false": true,
		"final": true, "finally": true, "for": true, "get": true,
		"if": true, "implements": true, "import": true, "in": true,
		"interface": true, "is": true, "late": true, "library": true,
		"mixin": true, "new": true, "null": true, "on": true,
		"operator": true, "part": true, "required": true, "rethrow": true,
		"return": true, "set": true, "show": true, "static": true,
		"super": true, "switch": true, "sync": true, "this": true,
		"throw": true, "true": true, "try": true, "typedef": true,
		"var": true, "void": true, "while": true, "with": true,
		"yield": true,
	}
	return keywords[name]
}

// safeDartFieldName returns a safe Dart identifier for a field name,
// appending a "$" suffix if the name is a reserved keyword.
func safeDartFieldName(name string) string {
	if isDartReservedKeyword(name) {
		return name + "Value"
	}
	return name
}

// toCamelCase converts a string to camelCase.
// Handles snake_case (session_token → sessionToken) and PascalCase (Provider → provider).
func toCamelCase(s string) string {
	if s == "" {
		return s
	}

	// Handle snake_case
	if strings.Contains(s, "_") {
		parts := strings.Split(s, "_")
		result := strings.ToLower(parts[0])
		for _, part := range parts[1:] {
			if part == "" {
				continue
			}
			runes := []rune(strings.ToLower(part))
			runes[0] = unicode.ToUpper(runes[0])
			result += string(runes)
		}
		return result
	}

	// Handle PascalCase → camelCase (just lowercase first letter)
	runes := []rune(s)
	// Handle sequences of uppercase letters (e.g., SAMLResponse → samlResponse)
	i := 0
	for i < len(runes) && unicode.IsUpper(runes[i]) {
		i++
	}
	if i == 0 {
		return s // Already camelCase
	}
	switch {
	case i == 1:
		// Single uppercase letter at start
		runes[0] = unicode.ToLower(runes[0])
	case i == len(runes):
		// All uppercase — lowercase everything
		for j := range runes {
			runes[j] = unicode.ToLower(runes[j])
		}
	default:
		// Multiple uppercase — lowercase all but the last (which starts the next word)
		for j := 0; j < i-1; j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
	}
	return string(runes)
}

// buildFromJSONExpr generates a Dart expression to extract a field from a JSON map.
func buildFromJSONExpr(jsonKey, dartType string, optional bool) string {
	safeKey := strings.ReplaceAll(jsonKey, `$`, `\$`)
	accessor := "json['" + safeKey + "']"

	switch dartType {
	case "String":
		if optional {
			return accessor + " as String?"
		}
		return accessor + " as String"
	case "int":
		if optional {
			return accessor + " == null ? null : (" + accessor + " as num).toInt()"
		}
		return "(" + accessor + " as num).toInt()"
	case "double":
		if optional {
			return accessor + " == null ? null : (" + accessor + " as num).toDouble()"
		}
		return "(" + accessor + " as num).toDouble()"
	case "bool":
		if optional {
			return accessor + " as bool?"
		}
		return accessor + " as bool"
	case "dynamic":
		return accessor
	}

	if dartType == "Map<String, dynamic>" {
		if optional {
			return accessor + " == null ? null : Map<String, dynamic>.from(" + accessor + " as Map)"
		}
		return "Map<String, dynamic>.from(" + accessor + " as Map)"
	}

	if strings.HasPrefix(dartType, "List<") {
		innerType := dartType[5 : len(dartType)-1]
		if optional {
			return accessor + " == null ? null : (" + accessor + " as List).map((e) => " + itemFromJSON("e", innerType) + ").toList()"
		}
		return "(" + accessor + " as List).map((e) => " + itemFromJSON("e", innerType) + ").toList()"
	}

	// Complex type — use fromJson
	if optional {
		return accessor + " == null ? null : " + dartType + ".fromJson(Map<String, dynamic>.from(" + accessor + " as Map))"
	}
	return dartType + ".fromJson(Map<String, dynamic>.from(" + accessor + " as Map))"
}

func itemFromJSON(varName, dartType string) string {
	switch dartType {
	case "String":
		return varName + " as String"
	case "int":
		return "(" + varName + " as num).toInt()"
	case "double":
		return "(" + varName + " as num).toDouble()"
	case "bool":
		return varName + " as bool"
	case "dynamic":
		return varName
	case "Map<String, dynamic>":
		return "Map<String, dynamic>.from(" + varName + " as Map)"
	}
	return dartType + ".fromJson(Map<String, dynamic>.from(" + varName + " as Map))"
}

// buildToJSONExpr generates a Dart expression to convert a field to JSON.
func buildToJSONExpr(dartName, dartType string, optional bool) string {
	switch dartType {
	case "String", "int", "double", "bool", "dynamic":
		return dartName
	}

	if dartType == "Map<String, dynamic>" {
		return dartName
	}

	if strings.HasPrefix(dartType, "List<") {
		innerType := dartType[5 : len(dartType)-1]
		switch innerType {
		case "String", "int", "double", "bool", "dynamic", "Map<String, dynamic>":
			return dartName
		}
		if optional {
			return dartName + "?.map((e) => e.toJson()).toList()"
		}
		return dartName + ".map((e) => e.toJson()).toList()"
	}

	// Complex type — use toJson
	if optional {
		return dartName + "?.toJson()"
	}
	return dartName + ".toJson()"
}

// buildResponseFromJSON generates a Dart expression to parse a response body.
// The template stores the _request() return value in a variable named `res`.
func buildResponseFromJSON(respType string) string {
	switch respType {
	case "void":
		return ""
	case "Map<String, dynamic>":
		return "Map<String, dynamic>.from(res as Map)"
	}

	if strings.HasPrefix(respType, "List<") {
		innerType := respType[5 : len(respType)-1]
		return "(res as List).map((e) => " + itemFromJSON("e", innerType) + ").toList()"
	}

	return respType + ".fromJson(Map<String, dynamic>.from(res as Map))"
}
