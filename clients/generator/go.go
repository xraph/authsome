package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/clients/manifest"
)

// GoGenerator generates Go client code
type GoGenerator struct {
	outputDir   string
	manifests   []*manifest.Manifest
	packageName string
}

// NewGoGenerator creates a new Go generator
func NewGoGenerator(outputDir string, manifests []*manifest.Manifest) *GoGenerator {
	return &GoGenerator{
		outputDir:   outputDir,
		manifests:   manifests,
		packageName: "authsome",
	}
}

// Generate generates Go client code
func (g *GoGenerator) Generate() error {
	if err := g.createDirectories(); err != nil {
		return err
	}

	if err := g.generateGoMod(); err != nil {
		return err
	}

	if err := g.generateTypes(); err != nil {
		return err
	}

	if err := g.generateErrors(); err != nil {
		return err
	}

	if err := g.generatePlugin(); err != nil {
		return err
	}

	if err := g.generateClient(); err != nil {
		return err
	}

	if err := g.generatePlugins(); err != nil {
		return err
	}

	return nil
}

func (g *GoGenerator) createDirectories() error {
	dirs := []string{
		g.outputDir,
		filepath.Join(g.outputDir, "plugins"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (g *GoGenerator) generateGoMod() error {
	content := `module github.com/xraph/authsome-client

go 1.21

require ()
`
	return g.writeFile("go.mod", content)
}

func (g *GoGenerator) generateTypes() error {
	var sb strings.Builder

	sb.WriteString("package authsome\n\n")
	sb.WriteString("import \"time\"\n\n")
	sb.WriteString("// Auto-generated types\n\n")

	// Collect all types from all manifests
	typeMap := make(map[string]*manifest.TypeDef)
	for _, m := range g.manifests {
		for _, t := range m.Types {
			if _, exists := typeMap[t.Name]; !exists {
				typeMap[t.Name] = &t
			}
		}
	}

	// Generate type definitions
	for _, t := range typeMap {
		if t.Description != "" {
			sb.WriteString(fmt.Sprintf("// %s represents %s\n", t.Name, t.Description))
		}
		sb.WriteString(fmt.Sprintf("type %s struct {\n", t.Name))

		for name, typeStr := range t.Fields {
			field := manifest.ParseField(name, typeStr)
			goType := g.mapTypeToGo(field.Type)

			if field.Array {
				goType = "[]" + goType
			}

			if !field.Required && !field.Array {
				goType = "*" + goType
			}

			jsonTag := field.Name
			if !field.Required {
				jsonTag += ",omitempty"
			}

			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n",
				g.exportedName(field.Name), goType, jsonTag))
		}
		sb.WriteString("}\n\n")
	}

	return g.writeFile("types.go", sb.String())
}

func (g *GoGenerator) generateErrors() error {
	content := `package authsome

import "fmt"

// Auto-generated error types

// Error represents an API error
type Error struct {
	Message    string
	StatusCode int
	Code       string
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s (status: %d)", e.Code, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("%s (status: %d)", e.Message, e.StatusCode)
}

// Specific error types
var (
	ErrUnauthorized = &Error{Message: "Unauthorized", StatusCode: 401, Code: "UNAUTHORIZED"}
	ErrForbidden    = &Error{Message: "Forbidden", StatusCode: 403, Code: "FORBIDDEN"}
	ErrNotFound     = &Error{Message: "Not found", StatusCode: 404, Code: "NOT_FOUND"}
	ErrConflict     = &Error{Message: "Conflict", StatusCode: 409, Code: "CONFLICT"}
	ErrRateLimit    = &Error{Message: "Rate limit exceeded", StatusCode: 429, Code: "RATE_LIMIT"}
	ErrServer       = &Error{Message: "Internal server error", StatusCode: 500, Code: "SERVER_ERROR"}
)

// NewError creates an error from a status code and message
func NewError(statusCode int, message string) *Error {
	code := ""
	switch statusCode {
	case 400:
		code = "VALIDATION_ERROR"
	case 401:
		code = "UNAUTHORIZED"
	case 403:
		code = "FORBIDDEN"
	case 404:
		code = "NOT_FOUND"
	case 409:
		code = "CONFLICT"
	case 429:
		code = "RATE_LIMIT"
	case 500:
		code = "SERVER_ERROR"
	}
	
	return &Error{
		Message:    message,
		StatusCode: statusCode,
		Code:       code,
	}
}
`
	return g.writeFile("errors.go", content)
}

func (g *GoGenerator) generatePlugin() error {
	content := `package authsome

// Auto-generated plugin interface

// Plugin defines the interface for client plugins
type Plugin interface {
	// ID returns the unique plugin identifier
	ID() string
	
	// Init initializes the plugin with the client
	Init(client *Client) error
}
`
	return g.writeFile("plugin.go", content)
}

func (g *GoGenerator) generateClient() error {
	var sb strings.Builder

	sb.WriteString("package authsome\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"bytes\"\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"encoding/json\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"io\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\t\"net/url\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("// Auto-generated AuthSome client\n\n")

	// Find core manifest
	var coreManifest *manifest.Manifest
	for _, m := range g.manifests {
		if m.PluginID == "core" {
			coreManifest = m
			break
		}
	}

	sb.WriteString("// Client is the main AuthSome client\n")
	sb.WriteString("type Client struct {\n")
	sb.WriteString("\tbaseURL    string\n")
	sb.WriteString("\thttpClient *http.Client\n")
	sb.WriteString("\ttoken      string\n")
	sb.WriteString("\theaders    map[string]string\n")
	sb.WriteString("\tplugins    map[string]Plugin\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// Option is a functional option for configuring the client\n")
	sb.WriteString("type Option func(*Client)\n\n")

	sb.WriteString("// WithHTTPClient sets a custom HTTP client\n")
	sb.WriteString("func WithHTTPClient(client *http.Client) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tc.httpClient = client\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// WithToken sets the authentication token\n")
	sb.WriteString("func WithToken(token string) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tc.token = token\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// WithHeaders sets custom headers\n")
	sb.WriteString("func WithHeaders(headers map[string]string) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tc.headers = headers\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// WithPlugins adds plugins to the client\n")
	sb.WriteString("func WithPlugins(plugins ...Plugin) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tfor _, p := range plugins {\n")
	sb.WriteString("\t\t\tc.plugins[p.ID()] = p\n")
	sb.WriteString("\t\t\tp.Init(c)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// NewClient creates a new AuthSome client\n")
	sb.WriteString("func NewClient(baseURL string, opts ...Option) *Client {\n")
	sb.WriteString("\tc := &Client{\n")
	sb.WriteString("\t\tbaseURL:    baseURL,\n")
	sb.WriteString("\t\thttpClient: http.DefaultClient,\n")
	sb.WriteString("\t\theaders:    make(map[string]string),\n")
	sb.WriteString("\t\tplugins:    make(map[string]Plugin),\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\tfor _, opt := range opts {\n")
	sb.WriteString("\t\topt(c)\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\treturn c\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// SetToken sets the authentication token\n")
	sb.WriteString("func (c *Client) SetToken(token string) {\n")
	sb.WriteString("\tc.token = token\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// GetPlugin returns a plugin by ID\n")
	sb.WriteString("func (c *Client) GetPlugin(id string) (Plugin, bool) {\n")
	sb.WriteString("\tp, ok := c.plugins[id]\n")
	sb.WriteString("\treturn p, ok\n")
	sb.WriteString("}\n\n")

	// Generate request helper
	sb.WriteString("func (c *Client) request(ctx context.Context, method, path string, body interface{}, result interface{}, auth bool) error {\n")
	sb.WriteString("\tvar bodyReader io.Reader\n")
	sb.WriteString("\tif body != nil {\n")
	sb.WriteString("\t\tdata, err := json.Marshal(body)\n")
	sb.WriteString("\t\tif err != nil {\n")
	sb.WriteString("\t\t\treturn fmt.Errorf(\"failed to marshal request: %w\", err)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tbodyReader = bytes.NewReader(data)\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\treq, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)\n")
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString("\t\treturn fmt.Errorf(\"failed to create request: %w\", err)\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\treq.Header.Set(\"Content-Type\", \"application/json\")\n")
	sb.WriteString("\tfor k, v := range c.headers {\n")
	sb.WriteString("\t\treq.Header.Set(k, v)\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\tif auth && c.token != \"\" {\n")
	sb.WriteString("\t\treq.Header.Set(\"Authorization\", \"Bearer \"+c.token)\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\tresp, err := c.httpClient.Do(req)\n")
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString("\t\treturn fmt.Errorf(\"request failed: %w\", err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tdefer resp.Body.Close()\n\n")
	sb.WriteString("\tif resp.StatusCode >= 400 {\n")
	sb.WriteString("\t\tvar errResp struct {\n")
	sb.WriteString("\t\t\tError   string `json:\"error\"`\n")
	sb.WriteString("\t\t\tMessage string `json:\"message\"`\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tjson.NewDecoder(resp.Body).Decode(&errResp)\n")
	sb.WriteString("\t\tmessage := errResp.Error\n")
	sb.WriteString("\t\tif message == \"\" {\n")
	sb.WriteString("\t\t\tmessage = errResp.Message\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tif message == \"\" {\n")
	sb.WriteString("\t\t\tmessage = resp.Status\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\treturn NewError(resp.StatusCode, message)\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\tif result != nil {\n")
	sb.WriteString("\t\tif err := json.NewDecoder(resp.Body).Decode(result); err != nil {\n")
	sb.WriteString("\t\t\treturn fmt.Errorf(\"failed to decode response: %w\", err)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\treturn nil\n")
	sb.WriteString("}\n\n")

	// Generate core methods
	if coreManifest != nil {
		for _, route := range coreManifest.Routes {
			g.generateGoMethod(&sb, coreManifest, &route)
		}
	}

	return g.writeFile("client.go", sb.String())
}

func (g *GoGenerator) generateGoMethod(sb *strings.Builder, m *manifest.Manifest, route *manifest.Route) {
	methodName := route.Name

	// Generate request struct if needed
	if len(route.Request) > 0 {
		sb.WriteString(fmt.Sprintf("// %sRequest is the request for %s\n", methodName, methodName))
		sb.WriteString(fmt.Sprintf("type %sRequest struct {\n", methodName))
		for name, typeStr := range route.Request {
			field := manifest.ParseField(name, typeStr)
			goType := g.mapTypeToGo(field.Type)
			if field.Array {
				goType = "[]" + goType
			}
			if !field.Required {
				goType = "*" + goType
			}
			jsonTag := field.Name
			if !field.Required {
				jsonTag += ",omitempty"
			}
			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", g.exportedName(field.Name), goType, jsonTag))
		}
		sb.WriteString("}\n\n")
	}

	// Generate response struct if needed
	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("// %sResponse is the response for %s\n", methodName, methodName))
		sb.WriteString(fmt.Sprintf("type %sResponse struct {\n", methodName))
		for name, typeStr := range route.Response {
			field := manifest.ParseField(name, typeStr)
			goType := g.mapTypeToGo(field.Type)
			if field.Array {
				goType = "[]" + goType
			}
			jsonTag := field.Name
			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", g.exportedName(field.Name), goType, jsonTag))
		}
		sb.WriteString("}\n\n")
	}

	// Generate method
	if route.Description != "" {
		sb.WriteString(fmt.Sprintf("// %s %s\n", methodName, route.Description))
	}
	sb.WriteString(fmt.Sprintf("func (c *Client) %s(ctx context.Context", methodName))

	if len(route.Request) > 0 {
		sb.WriteString(fmt.Sprintf(", req *%sRequest", methodName))
	}
	if len(route.Params) > 0 {
		for paramName, typeStr := range route.Params {
			field := manifest.ParseField(paramName, typeStr)
			sb.WriteString(fmt.Sprintf(", %s %s", field.Name, g.mapTypeToGo(field.Type)))
		}
	}

	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf(") (*%sResponse, error) {\n", methodName))
	} else {
		sb.WriteString(") error {\n")
	}

	// Build path
	path := m.BasePath + route.Path
	if len(route.Params) > 0 {
		for paramName := range route.Params {
			path = strings.ReplaceAll(path, "{"+paramName+"}", `" + url.PathEscape(`+paramName+`) + "`)
		}
		sb.WriteString(fmt.Sprintf("\tpath := \"%s\"\n", path))
	} else {
		sb.WriteString(fmt.Sprintf("\tpath := \"%s\"\n", path))
	}

	// Make request
	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("\tvar result %sResponse\n", methodName))
	}

	sb.WriteString("\terr := c.request(ctx, \"")
	sb.WriteString(strings.ToUpper(route.Method))
	sb.WriteString("\", path, ")

	if len(route.Request) > 0 {
		sb.WriteString("req")
	} else {
		sb.WriteString("nil")
	}
	sb.WriteString(", ")

	if len(route.Response) > 0 {
		sb.WriteString("&result")
	} else {
		sb.WriteString("nil")
	}
	sb.WriteString(", ")

	if route.Auth {
		sb.WriteString("true")
	} else {
		sb.WriteString("false")
	}
	sb.WriteString(")\n")

	if len(route.Response) > 0 {
		sb.WriteString("\tif err != nil {\n")
		sb.WriteString("\t\treturn nil, err\n")
		sb.WriteString("\t}\n")
		sb.WriteString("\treturn &result, nil\n")
	} else {
		sb.WriteString("\treturn err\n")
	}

	sb.WriteString("}\n\n")
}

func (g *GoGenerator) generatePlugins() error {
	for _, m := range g.manifests {
		if m.PluginID == "core" {
			continue
		}

		if err := g.generatePluginFile(m); err != nil {
			return err
		}
	}
	return nil
}

func (g *GoGenerator) generatePluginFile(m *manifest.Manifest) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("package %s\n\n", m.PluginID))
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"net/url\"\n\n")
	sb.WriteString("\t\"github.com/xraph/authsome-client\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString(fmt.Sprintf("// Auto-generated %s plugin\n\n", m.PluginID))

	// Generate plugin struct
	pluginName := "Plugin"
	sb.WriteString(fmt.Sprintf("// %s implements the %s plugin\n", pluginName, m.PluginID))
	sb.WriteString(fmt.Sprintf("type %s struct {\n", pluginName))
	sb.WriteString("\tclient *authsome.Client\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("// NewPlugin creates a new %s plugin\n", m.PluginID))
	sb.WriteString(fmt.Sprintf("func NewPlugin() *%s {\n", pluginName))
	sb.WriteString(fmt.Sprintf("\treturn &%s{}\n", pluginName))
	sb.WriteString("}\n\n")

	sb.WriteString("// ID returns the plugin identifier\n")
	sb.WriteString(fmt.Sprintf("func (p *%s) ID() string {\n", pluginName))
	sb.WriteString(fmt.Sprintf("\treturn \"%s\"\n", m.PluginID))
	sb.WriteString("}\n\n")

	sb.WriteString("// Init initializes the plugin\n")
	sb.WriteString(fmt.Sprintf("func (p *%s) Init(client *authsome.Client) error {\n", pluginName))
	sb.WriteString("\tp.client = client\n")
	sb.WriteString("\treturn nil\n")
	sb.WriteString("}\n\n")

	// Generate plugin methods
	for _, route := range m.Routes {
		g.generatePluginGoMethod(&sb, m, &route)
	}

	return g.writeFile(fmt.Sprintf("plugins/%s/%s.go", m.PluginID, m.PluginID), sb.String())
}

func (g *GoGenerator) generatePluginGoMethod(sb *strings.Builder, m *manifest.Manifest, route *manifest.Route) {
	methodName := route.Name

	// Generate request struct if needed
	if len(route.Request) > 0 {
		sb.WriteString(fmt.Sprintf("// %sRequest is the request for %s\n", methodName, methodName))
		sb.WriteString(fmt.Sprintf("type %sRequest struct {\n", methodName))
		for name, typeStr := range route.Request {
			field := manifest.ParseField(name, typeStr)
			goType := g.mapTypeToGo(field.Type)
			if field.Array {
				goType = "[]" + goType
			}
			if !field.Required {
				goType = "*" + goType
			}
			jsonTag := field.Name
			if !field.Required {
				jsonTag += ",omitempty"
			}
			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", g.exportedName(field.Name), goType, jsonTag))
		}
		sb.WriteString("}\n\n")
	}

	// Generate response struct if needed
	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("// %sResponse is the response for %s\n", methodName, methodName))
		sb.WriteString(fmt.Sprintf("type %sResponse struct {\n", methodName))
		for name, typeStr := range route.Response {
			field := manifest.ParseField(name, typeStr)
			goType := g.mapTypeToGo(field.Type)
			if field.Array {
				goType = "[]" + goType
			}
			jsonTag := field.Name
			sb.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", g.exportedName(field.Name), goType, jsonTag))
		}
		sb.WriteString("}\n\n")
	}

	// Generate method
	if route.Description != "" {
		sb.WriteString(fmt.Sprintf("// %s %s\n", methodName, route.Description))
	}
	sb.WriteString(fmt.Sprintf("func (p *Plugin) %s(ctx context.Context", methodName))

	if len(route.Request) > 0 {
		sb.WriteString(fmt.Sprintf(", req *%sRequest", methodName))
	}
	if len(route.Params) > 0 {
		for paramName, typeStr := range route.Params {
			field := manifest.ParseField(paramName, typeStr)
			sb.WriteString(fmt.Sprintf(", %s %s", field.Name, g.mapTypeToGo(field.Type)))
		}
	}

	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf(") (*%sResponse, error) {\n", methodName))
	} else {
		sb.WriteString(") error {\n")
	}

	// Build path
	path := m.BasePath + route.Path
	if len(route.Params) > 0 {
		for paramName := range route.Params {
			path = strings.ReplaceAll(path, "{"+paramName+"}", `" + url.PathEscape(`+paramName+`) + "`)
		}
		sb.WriteString(fmt.Sprintf("\tpath := \"%s\"\n", path))
	} else {
		sb.WriteString(fmt.Sprintf("\tpath := \"%s\"\n", path))
	}

	// Use reflection to call client.request (private method)
	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("\tvar result %sResponse\n", methodName))
	}

	sb.WriteString("\t// Note: This requires exposing client.request or using a different approach\n")
	sb.WriteString("\t// For now, this is a placeholder\n")
	sb.WriteString("\t_ = path\n")

	if len(route.Response) > 0 {
		sb.WriteString("\treturn &result, nil\n")
	} else {
		sb.WriteString("\treturn nil\n")
	}

	sb.WriteString("}\n\n")
}

func (g *GoGenerator) mapTypeToGo(t string) string {
	switch t {
	case "string":
		return "string"
	case "int", "int32":
		return "int"
	case "int64":
		return "int64"
	case "uint", "uint32":
		return "uint"
	case "uint64":
		return "uint64"
	case "float32":
		return "float32"
	case "float64":
		return "float64"
	case "bool", "boolean":
		return "bool"
	case "object", "map":
		return "map[string]interface{}"
	default:
		// Assume it's a custom type
		return t
	}
}

func (g *GoGenerator) exportedName(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *GoGenerator) writeFile(path string, content string) error {
	fullPath := filepath.Join(g.outputDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}
