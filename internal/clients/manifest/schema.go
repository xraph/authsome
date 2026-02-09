package manifest

import (
	"fmt"
	"strings"

	"github.com/xraph/authsome/internal/errs"
)

// Manifest represents a complete API manifest for a plugin or core module.
type Manifest struct {
	PluginID    string    `json:"plugin_id"             yaml:"plugin_id"`
	Version     string    `json:"version"               yaml:"version"`
	Description string    `json:"description,omitempty" yaml:"description,omitempty"`
	Routes      []Route   `json:"routes"                yaml:"routes"`
	Types       []TypeDef `json:"types,omitempty"       yaml:"types,omitempty"`
	BasePath    string    `json:"base_path,omitempty"   yaml:"base_path,omitempty"`
}

// Route represents a single API endpoint.
type Route struct {
	Name         string            `json:"name"                    yaml:"name"`
	Description  string            `json:"description,omitempty"   yaml:"description,omitempty"`
	Method       string            `json:"method"                  yaml:"method"`
	Path         string            `json:"path"                    yaml:"path"`
	Request      map[string]string `json:"request,omitempty"       yaml:"request,omitempty"`
	Response     map[string]string `json:"response,omitempty"      yaml:"response,omitempty"`
	RequestType  string            `json:"request_type,omitempty"  yaml:"request_type,omitempty"`  // Named type for request (e.g., "SignInRequest")
	ResponseType string            `json:"response_type,omitempty" yaml:"response_type,omitempty"` // Named type for response (e.g., "SignInResponse")
	Params       map[string]string `json:"params,omitempty"        yaml:"params,omitempty"`
	Query        map[string]string `json:"query,omitempty"         yaml:"query,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"       yaml:"headers,omitempty"`
	Errors       []ErrorDef        `json:"errors,omitempty"        yaml:"errors,omitempty"`
	Auth         bool              `json:"auth,omitempty"          yaml:"auth,omitempty"` // Requires authentication
}

// ErrorDef represents an error response.
type ErrorDef struct {
	Code        int    `json:"code"        yaml:"code"`
	Description string `json:"description" yaml:"description"`
}

// TypeDef represents a custom type definition.
type TypeDef struct {
	Name        string            `json:"name"                  yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Fields      map[string]string `json:"fields"                yaml:"fields"`
}

// Field represents a parsed field with its type and requirements.
type Field struct {
	Name     string
	Type     string
	Required bool
	Array    bool
	Optional bool
}

// ParseField parses a field type string (e.g., "string!", "User[]", "int").
func ParseField(name, typeStr string) Field {
	field := Field{Name: name}

	// Check for required marker FIRST (before array check)
	if strings.HasSuffix(typeStr, "!") {
		field.Required = true
		typeStr = strings.TrimSuffix(typeStr, "!")
	}

	// Check for optional marker (default is optional unless marked required)
	if strings.HasSuffix(typeStr, "?") {
		field.Optional = true
		typeStr = strings.TrimSuffix(typeStr, "?")
	}

	// Check for array type AFTER removing markers
	if strings.HasSuffix(typeStr, "[]") {
		field.Array = true
		typeStr = strings.TrimSuffix(typeStr, "[]")
	}

	field.Type = typeStr

	return field
}

// Validate validates the manifest.
func (m *Manifest) Validate() error {
	if m.PluginID == "" {
		return errs.RequiredField("plugin_id")
	}

	if m.Version == "" {
		return errs.RequiredField("version")
	}

	if len(m.Routes) == 0 {
		return errs.RequiredField("routes")
	}

	// Validate routes
	for i, route := range m.Routes {
		if err := route.Validate(); err != nil {
			return fmt.Errorf("route %d (%s): %w", i, route.Name, err)
		}
	}

	// Validate types
	typeNames := make(map[string]bool)

	for i, typeDef := range m.Types {
		if typeDef.Name == "" {
			return fmt.Errorf("type %d: name is required", i)
		}

		if typeNames[typeDef.Name] {
			return fmt.Errorf("duplicate type name: %s", typeDef.Name)
		}

		typeNames[typeDef.Name] = true
	}

	return nil
}

// Validate validates a route.
func (r *Route) Validate() error {
	if r.Name == "" {
		return errs.RequiredField("name")
	}

	if r.Method == "" {
		return errs.RequiredField("method")
	}

	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	}
	if !validMethods[strings.ToUpper(r.Method)] {
		return fmt.Errorf("invalid method: %s", r.Method)
	}

	if r.Path == "" {
		return errs.RequiredField("path")
	}

	return nil
}

// GetRequestFields returns parsed request fields.
func (r *Route) GetRequestFields() []Field {
	var fields []Field
	for name, typeStr := range r.Request {
		fields = append(fields, ParseField(name, typeStr))
	}

	return fields
}

// GetResponseFields returns parsed response fields.
func (r *Route) GetResponseFields() []Field {
	var fields []Field
	for name, typeStr := range r.Response {
		fields = append(fields, ParseField(name, typeStr))
	}

	return fields
}

// GetParamFields returns parsed path parameter fields.
func (r *Route) GetParamFields() []Field {
	var fields []Field
	for name, typeStr := range r.Params {
		fields = append(fields, ParseField(name, typeStr))
	}

	return fields
}

// GetQueryFields returns parsed query parameter fields.
func (r *Route) GetQueryFields() []Field {
	var fields []Field
	for name, typeStr := range r.Query {
		fields = append(fields, ParseField(name, typeStr))
	}

	return fields
}

// GetTypeFields returns parsed type definition fields.
func (t *TypeDef) GetFields() []Field {
	var fields []Field
	for name, typeStr := range t.Fields {
		fields = append(fields, ParseField(name, typeStr))
	}

	return fields
}
