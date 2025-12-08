package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/internal/clients/manifest"
)

// GoGenerator generates Go client code
type GoGenerator struct {
	outputDir   string
	manifests   []*manifest.Manifest
	packageName string
	moduleName  string
}

// NewGoGenerator creates a new Go generator
func NewGoGenerator(outputDir string, manifests []*manifest.Manifest, moduleName string) *GoGenerator {
	if moduleName == "" {
		moduleName = "github.com/xraph/authsome/clients/go"
	}
	return &GoGenerator{
		outputDir:   outputDir,
		manifests:   manifests,
		packageName: "authsome",
		moduleName:  moduleName,
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

	if err := g.generateContextHelpers(); err != nil {
		return err
	}

	if err := g.generateForgeMiddleware(); err != nil {
		return err
	}

	if err := g.generateHTTPMiddleware(); err != nil {
		return err
	}

	if err := g.generateContextFile(); err != nil {
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
	content := fmt.Sprintf(`module %s

go 1.21

require ()
`, g.moduleName)
	return g.writeFile("go.mod", content)
}

func (g *GoGenerator) generateTypes() error {
	var sb strings.Builder

	sb.WriteString("package authsome\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"time\"\n")
	sb.WriteString("\n")
	sb.WriteString("\t\"github.com/rs/xid\"\n")
	sb.WriteString(")\n\n")
	sb.WriteString("// Auto-generated types\n\n")

	// Placeholder type aliases for undefined types referenced in manifests
	sb.WriteString("// Placeholder type aliases for undefined enum/custom types\n")
	sb.WriteString("type (\n")
	sb.WriteString("\tRecoveryMethod       = string\n")
	sb.WriteString("\tRecoveryStatus       = string\n")
	sb.WriteString("\tComplianceStandard   = string\n")
	sb.WriteString("\tVerificationMethod   = string\n")
	sb.WriteString("\tFactorPriority       = string\n")
	sb.WriteString("\tFactorType           = string\n")
	sb.WriteString("\tFactorStatus         = string\n")
	sb.WriteString("\tRiskLevel            = string\n")
	sb.WriteString("\tSecurityLevel        = string\n")
	sb.WriteString("\tChallengeStatus      = string\n")
	sb.WriteString("\tJSONBMap             = map[string]interface{}\n")
	sb.WriteString(")\n\n")

	// Placeholder types as empty interfaces (to be used directly, not as package qualifiers)
	sb.WriteString("type (\n")
	sb.WriteString("\tschema               schemaPlaceholder\n")
	sb.WriteString("\tsession              sessionPlaceholder\n")
	sb.WriteString("\tuser                 userPlaceholder\n")
	sb.WriteString("\tproviders            providersPlaceholder\n")
	sb.WriteString("\tapikey               apikeyPlaceholder\n")
	sb.WriteString("\torganization         organizationPlaceholder\n")
	sb.WriteString(")\n\n")

	// Placeholder structs for package-qualified types
	sb.WriteString("// Placeholder structs for package-qualified types\n")
	sb.WriteString("type schemaPlaceholder struct {\n")
	sb.WriteString("\tIdentityVerificationSession interface{}\n")
	sb.WriteString("\tSocialAccount               interface{}\n")
	sb.WriteString("\tIdentityVerification        interface{}\n")
	sb.WriteString("\tUserVerificationStatus      interface{}\n")
	sb.WriteString("\tUser                        interface{}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("type providersPlaceholder struct {\n")
	sb.WriteString("\tEmailProvider interface{}\n")
	sb.WriteString("\tOAuthProvider interface{}\n")
	sb.WriteString("\tSAMLProvider  interface{}\n")
	sb.WriteString("\tSMSProvider   interface{}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("type sessionPlaceholder struct {\n")
	sb.WriteString("\tSession interface{}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("type userPlaceholder struct {\n")
	sb.WriteString("\tUser interface{}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("type apikeyPlaceholder struct {\n")
	sb.WriteString("\tAPIKey interface{}\n")
	sb.WriteString("\tRole   interface{}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("type organizationPlaceholder struct {\n")
	sb.WriteString("\tTeam       interface{}\n")
	sb.WriteString("\tInvitation interface{}\n")
	sb.WriteString("\tMember     interface{}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("type redisPlaceholder struct {\n")
	sb.WriteString("\tClient interface{}\n")
	sb.WriteString("}\n\n")

	// Make redis an alias to redisPlaceholder for backward compat
	sb.WriteString("var redis = redisPlaceholder{}\n\n")

	// Add placeholder empty structs for commonly missing types
	sb.WriteString("// Placeholder types for undefined/missing types\n")
	sb.WriteString("type (\n")
	sb.WriteString("\tTime                        = time.Time\n")
	sb.WriteString("\tID                          = xid.ID\n")
	sb.WriteString("\tIdentityVerification        struct {}\n")
	sb.WriteString("\tSocialAccount               struct {}\n")
	sb.WriteString("\tIdentityVerificationSession struct {}\n")
	sb.WriteString("\tNotificationType            = string\n")
	sb.WriteString("\tTeam                        struct {}\n")
	sb.WriteString("\tAPIKey                      struct {}\n")
	sb.WriteString("\tInvitation                  struct {}\n")
	sb.WriteString("\tUserVerificationStatus      = string\n")
	sb.WriteString("\tRole                        struct {}\n")
	sb.WriteString("\tProviderConfig              struct {}\n")
	sb.WriteString("\tMember                      struct {}\n")
	sb.WriteString(")\n\n")

	// Collect all types from all manifests
	// Deduplicate by type name, preferring core definitions
	typeMap := make(map[string]*manifest.TypeDef)

	// First pass: collect core types
	for _, m := range g.manifests {
		if m.PluginID == "core" {
			for _, t := range m.Types {
				td := t // Create a copy
				typeMap[t.Name] = &td
			}
			break
		}
	}

	// Second pass: collect plugin types (only if not already defined)
	for _, m := range g.manifests {
		if m.PluginID != "core" {
			for _, t := range m.Types {
				if _, exists := typeMap[t.Name]; !exists {
					td := t // Create a copy
					typeMap[t.Name] = &td
				}
			}
		}
	}

	// Third pass: collect request/response types from routes (to avoid inline redeclaration)
	for _, m := range g.manifests {
		for _, route := range m.Routes {
			// Collect request types
			if len(route.Request) > 0 {
				typeName := route.Name + "Request"
				if _, exists := typeMap[typeName]; !exists {
					typeMap[typeName] = &manifest.TypeDef{
						Name:   typeName,
						Fields: route.Request,
					}
				}
			}
			// Collect response types
			if len(route.Response) > 0 {
				typeName := route.Name + "Response"
				if _, exists := typeMap[typeName]; !exists {
					typeMap[typeName] = &manifest.TypeDef{
						Name:   typeName,
						Fields: route.Response,
					}
				}
			}
		}
	}

	// Generate type definitions
	for _, t := range typeMap {
		// Skip types that conflict with handwritten code
		if t.Name == "Plugin" {
			continue // Plugin is defined as an interface in plugin.go
		}

		if t.Description != "" {
			sb.WriteString(fmt.Sprintf("// %s represents %s\n", t.Name, t.Description))
		}
		sb.WriteString(fmt.Sprintf("type %s struct {\n", t.Name))

		for name, typeStr := range t.Fields {
			// Skip fields with empty names or "-" (JSON omit marker)
			if name == "" || name == "-" {
				continue
			}

			field := manifest.ParseField(name, typeStr)

			// If type is empty, use interface{} instead of trying to infer
			// (inference often leads to undefined type references)
			if field.Type == "" {
				field.Type = "interface{}"
			}

			goType := g.mapTypeToGo(field.Type)

			if field.Array {
				// Use pointer to element type for custom types
				if g.isCustomType(field.Type) {
					goType = "[]*" + goType
				} else {
					goType = "[]" + goType
				}
			}

			if !field.Required && !field.Array {
				// Don't add pointer if type already is a pointer (e.g., *redis.Client)
				if !g.isPointerType(field.Type) {
					goType = "*" + goType
				}
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
	sb.WriteString("\tbaseURL       string\n")
	sb.WriteString("\thttpClient    *http.Client\n")
	sb.WriteString("\ttoken         string              // Session token (Bearer)\n")
	sb.WriteString("\tapiKey        string              // API key (pk_/sk_/rk_)\n")
	sb.WriteString("\tcookieJar     http.CookieJar      // For session cookies\n")
	sb.WriteString("\theaders       map[string]string\n")
	sb.WriteString("\tplugins       map[string]Plugin\n")
	sb.WriteString("\tappID         string              // Current app context\n")
	sb.WriteString("\tenvironmentID string              // Current environment context\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// Option is a functional option for configuring the client\n")
	sb.WriteString("type Option func(*Client)\n\n")

	sb.WriteString("// WithHTTPClient sets a custom HTTP client\n")
	sb.WriteString("func WithHTTPClient(client *http.Client) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tc.httpClient = client\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// WithToken sets the authentication token (session token)\n")
	sb.WriteString("func WithToken(token string) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tc.token = token\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// WithAPIKey sets the API key for authentication\n")
	sb.WriteString("func WithAPIKey(apiKey string) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tc.apiKey = apiKey\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// WithCookieJar sets a cookie jar for session management\n")
	sb.WriteString("func WithCookieJar(jar http.CookieJar) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tc.cookieJar = jar\n")
	sb.WriteString("\t\tif c.httpClient != nil {\n")
	sb.WriteString("\t\t\tc.httpClient.Jar = jar\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// WithHeaders sets custom headers\n")
	sb.WriteString("func WithHeaders(headers map[string]string) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tc.headers = headers\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// WithAppContext sets the app and environment context for requests\n")
	sb.WriteString("func WithAppContext(appID, envID string) Option {\n")
	sb.WriteString("\treturn func(c *Client) {\n")
	sb.WriteString("\t\tc.appID = appID\n")
	sb.WriteString("\t\tc.environmentID = envID\n")
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

	sb.WriteString("// SetAPIKey sets the API key\n")
	sb.WriteString("func (c *Client) SetAPIKey(apiKey string) {\n")
	sb.WriteString("\tc.apiKey = apiKey\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// SetAppContext sets the app and environment context\n")
	sb.WriteString("func (c *Client) SetAppContext(appID, envID string) {\n")
	sb.WriteString("\tc.appID = appID\n")
	sb.WriteString("\tc.environmentID = envID\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// GetAppContext returns the current app and environment IDs\n")
	sb.WriteString("func (c *Client) GetAppContext() (appID, envID string) {\n")
	sb.WriteString("\treturn c.appID, c.environmentID\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// GetPlugin returns a plugin by ID\n")
	sb.WriteString("func (c *Client) GetPlugin(id string) (Plugin, bool) {\n")
	sb.WriteString("\tp, ok := c.plugins[id]\n")
	sb.WriteString("\treturn p, ok\n")
	sb.WriteString("}\n\n")

	// Generate public Request method for plugins
	sb.WriteString("// Request makes an HTTP request - exposed for plugin use\n")
	sb.WriteString("func (c *Client) Request(ctx context.Context, method, path string, body interface{}, result interface{}, auth bool) error {\n")
	sb.WriteString("\treturn c.request(ctx, method, path, body, result, auth)\n")
	sb.WriteString("}\n\n")

	// Generate request helper with auto-detection
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
	sb.WriteString("\t// Set custom headers\n")
	sb.WriteString("\tfor k, v := range c.headers {\n")
	sb.WriteString("\t\treq.Header.Set(k, v)\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\t// Auto-detect and set authentication\n")
	sb.WriteString("\tif auth {\n")
	sb.WriteString("\t\t// Priority 1: API key (if set)\n")
	sb.WriteString("\t\tif c.apiKey != \"\" {\n")
	sb.WriteString("\t\t\treq.Header.Set(\"Authorization\", \"ApiKey \"+c.apiKey)\n")
	sb.WriteString("\t\t// Priority 2: Session token\n")
	sb.WriteString("\t\t} else if c.token != \"\" {\n")
	sb.WriteString("\t\t\treq.Header.Set(\"Authorization\", \"Bearer \"+c.token)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\t// Note: Cookies are automatically attached by httpClient if cookieJar is set\n")
	sb.WriteString("\t}\n\n")
	sb.WriteString("\t// Set app and environment context headers if available\n")
	sb.WriteString("\tif c.appID != \"\" {\n")
	sb.WriteString("\t\treq.Header.Set(\"X-App-ID\", c.appID)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tif c.environmentID != \"\" {\n")
	sb.WriteString("\t\treq.Header.Set(\"X-Environment-ID\", c.environmentID)\n")
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

	// Request/Response types are now generated in types.go via deduplication
	// No need to generate inline types here

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

	// Check if we need net/url import (for path parameters)
	hasPathParams := false
	for _, route := range m.Routes {
		if len(route.Params) > 0 {
			hasPathParams = true
			break
		}
	}

	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	if hasPathParams {
		sb.WriteString("\t\"net/url\"\n")
	}
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("\t\"%s\"\n", g.moduleName))
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

	// Request/Response types are now generated in types.go via deduplication
	// No need to generate inline types here

	// Generate method
	if route.Description != "" {
		sb.WriteString(fmt.Sprintf("// %s %s\n", methodName, route.Description))
	}
	sb.WriteString(fmt.Sprintf("func (p *Plugin) %s(ctx context.Context", methodName))

	if len(route.Request) > 0 {
		sb.WriteString(fmt.Sprintf(", req *authsome.%sRequest", methodName))
	}
	if len(route.Params) > 0 {
		for paramName, typeStr := range route.Params {
			field := manifest.ParseField(paramName, typeStr)
			sb.WriteString(fmt.Sprintf(", %s %s", field.Name, g.mapTypeToGo(field.Type)))
		}
	}

	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf(") (*authsome.%sResponse, error) {\n", methodName))
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

	// Make request through client
	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("\tvar result authsome.%sResponse\n", methodName))
	}

	sb.WriteString("\terr := p.client.Request(ctx, \"")
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

func (g *GoGenerator) generateContextHelpers() error {
	var sb strings.Builder

	sb.WriteString("package authsome\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString(")\n\n")
	sb.WriteString("// Auto-generated context helper methods\n\n")

	// GetCurrentUser method
	sb.WriteString("// GetCurrentUser retrieves the current user from the session\n")
	sb.WriteString("func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {\n")
	sb.WriteString("\tsession, err := c.GetSession(ctx)\n")
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString("\t\treturn nil, err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn &session.User, nil\n")
	sb.WriteString("}\n\n")

	// GetCurrentSession method
	sb.WriteString("// GetCurrentSession retrieves the current session\n")
	sb.WriteString("func (c *Client) GetCurrentSession(ctx context.Context) (*Session, error) {\n")
	sb.WriteString("\tsession, err := c.GetSession(ctx)\n")
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString("\t\treturn nil, err\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn &session.Session, nil\n")
	sb.WriteString("}\n\n")

	return g.writeFile("context_helpers.go", sb.String())
}

func (g *GoGenerator) isPointerType(t string) bool {
	return strings.HasPrefix(t, "*")
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
	case "Time":
		return "time.Time"
	case "ID":
		return "xid.ID"
	case "Duration":
		return "time.Duration"
	case "NotificationType":
		return "string"
	default:
		// Strip package qualifiers for authsome internal types (e.g., "user.User" -> "User")
		// But preserve standard library and external package qualifiers (e.g., "xid.ID", "time.Time")
		if strings.Contains(t, ".") {
			parts := strings.Split(t, ".")
			pkg := parts[0]
			// Preserve qualifiers for standard library and known external packages
			if pkg == "time" || pkg == "xid" || pkg == "redis" || pkg == "context" {
				return t
			}
			// Strip qualifiers for authsome internal types
			return parts[len(parts)-1]
		}
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

func (g *GoGenerator) isCustomType(t string) bool {
	primitives := map[string]bool{
		"string": true, "int": true, "int32": true, "int64": true,
		"uint": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true,
		"bool": true, "boolean": true,
		"object": true, "map": true,
	}
	return !primitives[t]
}

// mapTypeToGoWithPackage maps a type to Go type with package qualifier for custom types
func (g *GoGenerator) mapTypeToGoWithPackage(t string) string {
	if g.isCustomType(t) {
		// Custom types need package qualifier (authsome.TypeName)
		return "authsome." + t
	}
	return g.mapTypeToGo(t)
}

func (g *GoGenerator) writeFile(path string, content string) error {
	fullPath := filepath.Join(g.outputDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}
