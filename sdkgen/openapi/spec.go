// Package openapi generates OpenAPI 3.1 specifications from AuthSome engine metadata.
package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadSpecFromFile reads a JSON-encoded OpenAPI spec from disk. This is used
// by the sdkgen CLI when --from-spec is provided, allowing the spec to be
// produced externally (e.g., by the specgen tool) rather than by the
// hardcoded generator.
func LoadSpecFromFile(path string) (*Spec, error) {
	// #nosec G304 -- operator-supplied path.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec file: %w", err)
	}
	var spec Spec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse spec file: %w", err)
	}
	return &spec, nil
}

// Spec represents an OpenAPI 3.1.0 specification document.
type Spec struct {
	OpenAPI    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Servers    []Server              `json:"servers,omitempty"`
	Paths      map[string]*PathItem  `json:"paths"`
	Components *Components           `json:"components,omitempty"`
	Security   []SecurityRequirement `json:"security,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty"`
}

// Info provides metadata about the API.
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// Server represents a server URL.
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PathItem describes operations available on a single path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Security    []SecurityRequirement `json:"security,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]*Response  `json:"responses"`
}

// Parameter describes a single operation parameter.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // "query", "path", "header"
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// RequestBody describes a single request body.
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"`
	Content     map[string]MediaType `json:"content"`
}

// MediaType describes a media type with a schema.
type MediaType struct {
	Schema *Schema `json:"schema"`
}

// Response describes a single response from an API Operation.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// Schema is a simplified JSON Schema representation.
type Schema struct {
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
}

// Components holds a set of reusable objects.
type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme defines a security scheme.
type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Description  string `json:"description,omitempty"`
	Name         string `json:"name,omitempty"`
	In           string `json:"in,omitempty"` // "header", "query", "cookie"
}

// SecurityRequirement identifies the security schemes required.
type SecurityRequirement map[string][]string

// Tag adds metadata to a single tag.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// SecuritySchemes returns the declared security schemes, or nil when the spec
// carries no components block. Callers pass the result straight to
// OperationTakesToken, so it has to survive a spec with nothing declared.
func (s *Spec) SecuritySchemes() map[string]*SecurityScheme {
	if s == nil || s.Components == nil {
		return nil
	}
	return s.Components.SecuritySchemes
}

// OperationTakesToken reports whether generated client methods for op should
// carry an explicit bearer-token parameter.
//
// The three language generators (golang, typescript, dart) each used to answer
// this by matching the literal scheme name "bearerAuth". That silently dropped
// the token parameter from every operation secured with forge's
// WithGroupAuth("session", "session-cookie") — 87 of the 200 operations in the
// current spec — leaving them impossible to authenticate through any SDK.
// "session" is a bearer scheme: extractToken reads Authorization: Bearer first
// and only then falls back to the auth_token cookie.
//
// The rule is deliberately a property of the scheme rather than its name: a
// scheme carries a token when it is declared as HTTP bearer. Adding a new name
// to a route is then enough, so long as specgen declares it, and a name nobody
// declared contributes nothing rather than quietly deciding the answer.
//
// schemes is spec.Components.SecuritySchemes, which may be nil or may not
// declare every name an operation references.
func OperationTakesToken(op *Operation, schemes map[string]*SecurityScheme) bool {
	if op == nil || len(op.Security) == 0 {
		return false
	}
	for _, req := range op.Security {
		for name := range req {
			s := schemes[name]
			if s == nil {
				continue
			}
			if s.Type == "http" && strings.EqualFold(s.Scheme, "bearer") {
				return true
			}
		}
	}
	return false
}
