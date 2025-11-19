package manifest

import (
	"fmt"
	"strings"
)

// Manifest represents a complete API manifest for a plugin or core module
type Manifest struct {
	PluginID    string    `yaml:"plugin_id" json:"plugin_id"`
	Version     string    `yaml:"version" json:"version"`
	Description string    `yaml:"description,omitempty" json:"description,omitempty"`
	Routes      []Route   `yaml:"routes" json:"routes"`
	Types       []TypeDef `yaml:"types,omitempty" json:"types,omitempty"`
	BasePath    string    `yaml:"base_path,omitempty" json:"base_path,omitempty"`
}

// Route represents a single API endpoint
type Route struct {
	Name         string            `yaml:"name" json:"name"`
	Description  string            `yaml:"description,omitempty" json:"description,omitempty"`
	Method       string            `yaml:"method" json:"method"`
	Path         string            `yaml:"path" json:"path"`
	Request      map[string]string `yaml:"request,omitempty" json:"request,omitempty"`
	Response     map[string]string `yaml:"response,omitempty" json:"response,omitempty"`
	RequestType  string            `yaml:"request_type,omitempty" json:"request_type,omitempty"`   // Named type for request (e.g., "SignInRequest")
	ResponseType string            `yaml:"response_type,omitempty" json:"response_type,omitempty"` // Named type for response (e.g., "SignInResponse")
	Params       map[string]string `yaml:"params,omitempty" json:"params,omitempty"`
	Query        map[string]string `yaml:"query,omitempty" json:"query,omitempty"`
	Headers      map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Errors       []ErrorDef        `yaml:"errors,omitempty" json:"errors,omitempty"`
	Auth         bool              `yaml:"auth,omitempty" json:"auth,omitempty"` // Requires authentication
}

// ErrorDef represents an error response
type ErrorDef struct {
	Code        int    `yaml:"code" json:"code"`
	Description string `yaml:"description" json:"description"`
}

// TypeDef represents a custom type definition
type TypeDef struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Fields      map[string]string `yaml:"fields" json:"fields"`
}

// Field represents a parsed field with its type and requirements
type Field struct {
	Name     string
	Type     string
	Required bool
	Array    bool
	Optional bool
}

// ParseField parses a field type string (e.g., "string!", "User[]", "int")
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

// Validate validates the manifest
func (m *Manifest) Validate() error {
	if m.PluginID == "" {
		return fmt.Errorf("plugin_id is required")
	}

	if m.Version == "" {
		return fmt.Errorf("version is required")
	}

	if len(m.Routes) == 0 {
		return fmt.Errorf("at least one route is required")
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

// Validate validates a route
func (r *Route) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}

	if r.Method == "" {
		return fmt.Errorf("method is required")
	}

	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	}
	if !validMethods[strings.ToUpper(r.Method)] {
		return fmt.Errorf("invalid method: %s", r.Method)
	}

	if r.Path == "" {
		return fmt.Errorf("path is required")
	}

	return nil
}

// GetRequestFields returns parsed request fields
func (r *Route) GetRequestFields() []Field {
	var fields []Field
	for name, typeStr := range r.Request {
		fields = append(fields, ParseField(name, typeStr))
	}
	return fields
}

// GetResponseFields returns parsed response fields
func (r *Route) GetResponseFields() []Field {
	var fields []Field
	for name, typeStr := range r.Response {
		fields = append(fields, ParseField(name, typeStr))
	}
	return fields
}

// GetParamFields returns parsed path parameter fields
func (r *Route) GetParamFields() []Field {
	var fields []Field
	for name, typeStr := range r.Params {
		fields = append(fields, ParseField(name, typeStr))
	}
	return fields
}

// GetQueryFields returns parsed query parameter fields
func (r *Route) GetQueryFields() []Field {
	var fields []Field
	for name, typeStr := range r.Query {
		fields = append(fields, ParseField(name, typeStr))
	}
	return fields
}

// GetTypeFields returns parsed type definition fields
func (t *TypeDef) GetFields() []Field {
	var fields []Field
	for name, typeStr := range t.Fields {
		fields = append(fields, ParseField(name, typeStr))
	}
	return fields
}
