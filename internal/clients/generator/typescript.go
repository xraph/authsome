package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/internal/clients/manifest"
)

// TypeScriptGenerator generates TypeScript client code.
type TypeScriptGenerator struct {
	outputDir string
	manifests []*manifest.Manifest
}

// NewTypeScriptGenerator creates a new TypeScript generator.
func NewTypeScriptGenerator(outputDir string, manifests []*manifest.Manifest) *TypeScriptGenerator {
	return &TypeScriptGenerator{
		outputDir: outputDir,
		manifests: manifests,
	}
}

// Generate generates TypeScript client code.
func (g *TypeScriptGenerator) Generate() error {
	if err := g.createDirectories(); err != nil {
		return err
	}

	if err := g.generatePackageJSON(); err != nil {
		return err
	}

	if err := g.generateTSConfig(); err != nil {
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

	if err := g.generateIndex(); err != nil {
		return err
	}

	return nil
}

func (g *TypeScriptGenerator) createDirectories() error {
	dirs := []string{
		g.outputDir,
		filepath.Join(g.outputDir, "src"),
		filepath.Join(g.outputDir, "src", "plugins"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (g *TypeScriptGenerator) generatePackageJSON() error {
	content := `{
  "name": "@authsome/client",
  "version": "1.0.0",
  "description": "TypeScript client for AuthSome authentication",
  "main": "dist/index.js",
  "types": "dist/index.d.ts",
  "scripts": {
    "build": "tsc",
    "prepublishOnly": "npm run build"
  },
  "keywords": ["authsome", "authentication", "client"],
  "author": "",
  "license": "MIT",
  "devDependencies": {
    "typescript": "^5.0.0"
  }
}
`

	return g.writeFile("package.json", content)
}

func (g *TypeScriptGenerator) generateTSConfig() error {
	content := `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "declaration": true,
    "outDir": "./dist",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "moduleResolution": "node",
    "resolveJsonModule": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
`

	return g.writeFile("tsconfig.json", content)
}

func (g *TypeScriptGenerator) generateTypes() error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated TypeScript types\n\n")

	// Collect all types from all manifests
	typeMap := make(map[string]*manifest.TypeDef)

	for _, m := range g.manifests {
		for _, t := range m.Types {
			if _, exists := typeMap[t.Name]; !exists {
				typeMap[t.Name] = &t
			}
		}
	}

	// Collect all referenced types from routes (request_type and response_type)
	referencedTypes := make(map[string]bool)

	for _, m := range g.manifests {
		for _, route := range m.Routes {
			if route.RequestType != "" {
				// Strip package qualifier
				typeName := route.RequestType
				if idx := strings.LastIndex(typeName, "."); idx != -1 {
					typeName = typeName[idx+1:]
				}

				referencedTypes[typeName] = true
			}

			if route.ResponseType != "" {
				// Strip package qualifier
				typeName := route.ResponseType
				if idx := strings.LastIndex(typeName, "."); idx != -1 {
					typeName = typeName[idx+1:]
				}

				referencedTypes[typeName] = true
			}
			// Also collect types from response fields (like *apikey.APIKey)
			for _, typeStr := range route.Response {
				if typeStr == "" || typeStr == "-" {
					continue
				}

				field := manifest.ParseField("", typeStr)
				// Extract type name from qualified types
				if strings.Contains(field.Type, ".") {
					parts := strings.Split(field.Type, ".")
					if len(parts) == 2 {
						typeName := parts[1]
						referencedTypes[typeName] = true
						// Create synthetic type if not already defined
						if _, exists := typeMap[typeName]; !exists {
							typeMap[typeName] = &manifest.TypeDef{
								Name:   typeName,
								Fields: make(map[string]string),
							}
						}
					}
				}
			}
		}

		// Also scan through all type definitions to find referenced types in fields
		for _, typeDef := range m.Types {
			for _, fieldType := range typeDef.Fields {
				if fieldType == "" || fieldType == "-" {
					continue
				}

				field := manifest.ParseField("", fieldType)
				// Extract type name from qualified types or plain custom types
				if strings.Contains(field.Type, ".") {
					parts := strings.Split(field.Type, ".")
					if len(parts) == 2 {
						typeName := parts[1]
						// Skip invalid type names (with special characters)
						if typeName != "" && !strings.ContainsAny(typeName, "[](){}*") {
							referencedTypes[typeName] = true
							// Create synthetic type if not already defined
							if _, exists := typeMap[typeName]; !exists {
								typeMap[typeName] = &manifest.TypeDef{
									Name:   typeName,
									Fields: make(map[string]string),
								}
							}
						}
					}
				} else {
					// Check if it's a custom type (not a primitive)
					// Skip invalid type names (with special characters, array notation, etc.)
					if !g.isPrimitiveType(field.Type) && field.Type != "" && !strings.ContainsAny(field.Type, "[](){}*") {
						// Skip lowercase types (Go internal types)
						if len(field.Type) > 0 && (field.Type[0] < 'a' || field.Type[0] > 'z') {
							referencedTypes[field.Type] = true
						}
					}
				}
			}
		}
	}

	// Create placeholder types for referenced but undefined types
	for typeName := range referencedTypes {
		if _, exists := typeMap[typeName]; !exists {
			// Create a minimal type definition
			typeMap[typeName] = &manifest.TypeDef{
				Name:   typeName,
				Fields: map[string]string{},
			}
		}
	}

	// Generate type definitions
	for _, t := range typeMap {
		// Skip types with invalid names (empty, array notation, special characters)
		if t.Name == "" || strings.HasPrefix(t.Name, "[") || strings.ContainsAny(t.Name, "[](){}*") {
			continue
		}

		// Skip lowercase types (these are usually Go internal types or errors in extraction)
		if len(t.Name) > 0 && t.Name[0] >= 'a' && t.Name[0] <= 'z' {
			continue
		}

		sb.WriteString(fmt.Sprintf("export interface %s {\n", t.Name))

		// Handle empty types (placeholders)
		if len(t.Fields) == 0 {
			sb.WriteString("  [key: string]: any;\n")
		} else {
			for name, typeStr := range t.Fields {
				// Skip fields with empty or invalid names (embedded fields)
				if name == "" || name == "-" {
					continue
				}

				field := manifest.ParseField(name, typeStr)
				tsType := g.mapTypeToTSForTypesFile(field.Type)

				if field.Array {
					tsType += "[]"
				}

				optional := ""
				if !field.Required {
					optional = "?"
				}

				sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", field.Name, optional, tsType))
			}
		}

		sb.WriteString("}\n\n")
	}

	return g.writeFile("src/types.ts", sb.String())
}

func (g *TypeScriptGenerator) generateErrors() error {
	content := `// Auto-generated error classes

export class AuthsomeError extends Error {
  constructor(
    message: string,
    public statusCode: number,
    public code?: string
  ) {
    super(message);
    this.name = 'AuthsomeError';
  }
}

export class NetworkError extends AuthsomeError {
  constructor(message: string) {
    super(message, 0, 'NETWORK_ERROR');
    this.name = 'NetworkError';
  }
}

export class ValidationError extends AuthsomeError {
  constructor(message: string, public fields?: Record<string, string>) {
    super(message, 400, 'VALIDATION_ERROR');
    this.name = 'ValidationError';
  }
}

export class UnauthorizedError extends AuthsomeError {
  constructor(message: string = 'Unauthorized') {
    super(message, 401, 'UNAUTHORIZED');
    this.name = 'UnauthorizedError';
  }
}

export class ForbiddenError extends AuthsomeError {
  constructor(message: string = 'Forbidden') {
    super(message, 403, 'FORBIDDEN');
    this.name = 'ForbiddenError';
  }
}

export class NotFoundError extends AuthsomeError {
  constructor(message: string = 'Not found') {
    super(message, 404, 'NOT_FOUND');
    this.name = 'NotFoundError';
  }
}

export class ConflictError extends AuthsomeError {
  constructor(message: string) {
    super(message, 409, 'CONFLICT');
    this.name = 'ConflictError';
  }
}

export class RateLimitError extends AuthsomeError {
  constructor(message: string = 'Rate limit exceeded') {
    super(message, 429, 'RATE_LIMIT_EXCEEDED');
    this.name = 'RateLimitError';
  }
}

export class ServerError extends AuthsomeError {
  constructor(message: string = 'Internal server error') {
    super(message, 500, 'SERVER_ERROR');
    this.name = 'ServerError';
  }
}

export function createErrorFromResponse(statusCode: number, message: string): AuthsomeError {
  switch (statusCode) {
    case 400:
      return new ValidationError(message);
    case 401:
      return new UnauthorizedError(message);
    case 403:
      return new ForbiddenError(message);
    case 404:
      return new NotFoundError(message);
    case 409:
      return new ConflictError(message);
    case 429:
      return new RateLimitError(message);
    case 500:
    case 502:
    case 503:
    case 504:
      return new ServerError(message);
    default:
      return new AuthsomeError(message, statusCode);
  }
}
`

	return g.writeFile("src/errors.ts", content)
}

func (g *TypeScriptGenerator) generatePlugin() error {
	content := `// Auto-generated plugin interface

import { AuthsomeClient } from './client';

export interface ClientPlugin {
  readonly id: string;
  
  // Initialize plugin with base client
  init(client: AuthsomeClient): void;
  
  // Optional: validate configuration
  validate?(): Promise<boolean>;
}
`

	return g.writeFile("src/plugin.ts", content)
}

func (g *TypeScriptGenerator) generateClient() error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated AuthSome client\n\n")
	sb.WriteString("import { ClientPlugin } from './plugin';\n")
	sb.WriteString("import { createErrorFromResponse } from './errors';\n")
	sb.WriteString("import * as types from './types';\n")

	// Import plugin classes for type-safe access
	for _, m := range g.manifests {
		if m.PluginID != "core" {
			pluginClassName := g.pascalCase(m.PluginID) + "Plugin"
			sb.WriteString(fmt.Sprintf("import { %s } from './plugins/%s';\n", pluginClassName, m.PluginID))
		}
	}

	sb.WriteString("\n")

	// Find core manifest
	var coreManifest *manifest.Manifest

	for _, m := range g.manifests {
		if m.PluginID == "core" {
			coreManifest = m

			break
		}
	}

	sb.WriteString("/**\n")
	sb.WriteString(" * AuthSome client configuration\n")
	sb.WriteString(" * Supports multiple authentication methods that can be used simultaneously:\n")
	sb.WriteString(" * - Cookies: Automatically sent with every request (session-based auth)\n")
	sb.WriteString(" * - Bearer Token: JWT tokens sent in Authorization header when auth: true\n")
	sb.WriteString(" * - API Key: Sent with every request for server-to-server auth\n")
	sb.WriteString(" *   - Publishable Key (pk_*): Safe for frontend, limited permissions\n")
	sb.WriteString(" *   - Secret Key (sk_*): Backend only, full admin access\n")
	sb.WriteString(" */\n")
	sb.WriteString("export interface AuthsomeClientConfig {\n")
	sb.WriteString("  /** Base URL of the AuthSome API */\n")
	sb.WriteString("  baseURL: string;\n")
	sb.WriteString("  \n")
	sb.WriteString("  /** Plugin instances to initialize */\n")
	sb.WriteString("  plugins?: ClientPlugin[];\n")
	sb.WriteString("  \n")
	sb.WriteString("  /** JWT/Bearer token for user authentication (sent only when auth: true) */\n")
	sb.WriteString("  token?: string;\n")
	sb.WriteString("  \n")
	sb.WriteString("  /** API key for server-to-server auth (pk_* or sk_*, sent with all requests) */\n")
	sb.WriteString("  apiKey?: string;\n")
	sb.WriteString("  \n")
	sb.WriteString("  /** Custom header name for API key (default: 'X-API-Key') */\n")
	sb.WriteString("  apiKeyHeader?: string;\n")
	sb.WriteString("  \n")
	sb.WriteString("  /** Custom headers to include with all requests */\n")
	sb.WriteString("  headers?: Record<string, string>;\n")
	sb.WriteString("  \n")
	sb.WriteString("  /** Base path prefix for all API routes (default: '') */\n")
	sb.WriteString("  basePath?: string;\n")
	sb.WriteString("}\n\n")

	sb.WriteString("export class AuthsomeClient {\n")
	sb.WriteString("  private baseURL: string;\n")
	sb.WriteString("  private basePath: string;\n")
	sb.WriteString("  private token?: string;\n")
	sb.WriteString("  private apiKey?: string;\n")
	sb.WriteString("  private apiKeyHeader: string;\n")
	sb.WriteString("  private headers: Record<string, string>;\n")
	sb.WriteString("  private plugins: Map<string, ClientPlugin>;\n\n")

	sb.WriteString("  constructor(config: AuthsomeClientConfig) {\n")
	sb.WriteString("    this.baseURL = config.baseURL;\n")
	sb.WriteString("    this.basePath = config.basePath || '';\n")
	sb.WriteString("    this.token = config.token;\n")
	sb.WriteString("    this.apiKey = config.apiKey;\n")
	sb.WriteString("    this.apiKeyHeader = config.apiKeyHeader || 'X-API-Key';\n")
	sb.WriteString("    this.headers = config.headers || {};\n")
	sb.WriteString("    this.plugins = new Map();\n\n")
	sb.WriteString("    if (config.plugins) {\n")
	sb.WriteString("      for (const plugin of config.plugins) {\n")
	sb.WriteString("        this.plugins.set(plugin.id, plugin);\n")
	sb.WriteString("        plugin.init(this);\n")
	sb.WriteString("      }\n")
	sb.WriteString("    }\n")
	sb.WriteString("  }\n\n")

	sb.WriteString("  setToken(token: string): void {\n")
	sb.WriteString("    this.token = token;\n")
	sb.WriteString("  }\n\n")

	sb.WriteString("  setApiKey(apiKey: string, header?: string): void {\n")
	sb.WriteString("    this.apiKey = apiKey;\n")
	sb.WriteString("    if (header) {\n")
	sb.WriteString("      this.apiKeyHeader = header;\n")
	sb.WriteString("    }\n")
	sb.WriteString("  }\n\n")

	// Add publishable key helper
	sb.WriteString("  /**\n")
	sb.WriteString("   * Set a publishable key (pk_*) - safe for frontend use\n")
	sb.WriteString("   * Publishable keys have limited permissions and can be exposed in client-side code\n")
	sb.WriteString("   * Typically used for: session creation, user verification, public data reads\n")
	sb.WriteString("   */\n")
	sb.WriteString("  setPublishableKey(publishableKey: string): void {\n")
	sb.WriteString("    if (!publishableKey.startsWith('pk_')) {\n")
	sb.WriteString("      console.warn('Warning: Publishable keys should start with pk_');\n")
	sb.WriteString("    }\n")
	sb.WriteString("    this.setApiKey(publishableKey);\n")
	sb.WriteString("  }\n\n")

	// Add secret key helper
	sb.WriteString("  /**\n")
	sb.WriteString("   * Set a secret key (sk_*) - MUST be kept secret on server-side only!\n")
	sb.WriteString("   * Secret keys have full administrative access to all operations\n")
	sb.WriteString("   * WARNING: Never expose secret keys in client-side code (browser, mobile apps)\n")
	sb.WriteString("   */\n")
	sb.WriteString("  setSecretKey(secretKey: string): void {\n")
	sb.WriteString("    if (!secretKey.startsWith('sk_')) {\n")
	sb.WriteString("      console.warn('Warning: Secret keys should start with sk_');\n")
	sb.WriteString("    }\n")
	sb.WriteString("    this.setApiKey(secretKey);\n")
	sb.WriteString("  }\n\n")

	sb.WriteString("  setBasePath(basePath: string): void {\n")
	sb.WriteString("    this.basePath = basePath;\n")
	sb.WriteString("  }\n\n")

	// Add toQueryParams helper
	sb.WriteString("  /**\n")
	sb.WriteString("   * Convert an object to query parameters, handling optional values and type conversion\n")
	sb.WriteString("   */\n")
	sb.WriteString("  public toQueryParams(obj?: Record<string, any>): Record<string, string> | undefined {\n")
	sb.WriteString("    if (!obj) return undefined;\n")
	sb.WriteString("    \n")
	sb.WriteString("    const params: Record<string, string> = {};\n")
	sb.WriteString("    for (const [key, value] of Object.entries(obj)) {\n")
	sb.WriteString("      if (value !== undefined && value !== null) {\n")
	sb.WriteString("        params[key] = String(value);\n")
	sb.WriteString("      }\n")
	sb.WriteString("    }\n")
	sb.WriteString("    return Object.keys(params).length > 0 ? params : undefined;\n")
	sb.WriteString("  }\n\n")

	// Add setGlobalHeaders method
	sb.WriteString("  /**\n")
	sb.WriteString("   * Set global headers for all requests\n")
	sb.WriteString("   * @param headers - Headers to set\n")
	sb.WriteString("   * @param replace - If true, replaces all existing headers. If false (default), merges with existing headers\n")
	sb.WriteString("   */\n")
	sb.WriteString("  setGlobalHeaders(headers: Record<string, string>, replace: boolean = false): void {\n")
	sb.WriteString("    if (replace) {\n")
	sb.WriteString("      this.headers = { ...headers };\n")
	sb.WriteString("    } else {\n")
	sb.WriteString("      this.headers = { ...this.headers, ...headers };\n")
	sb.WriteString("    }\n")
	sb.WriteString("  }\n\n")

	sb.WriteString("  getPlugin<T extends ClientPlugin>(id: string): T | undefined {\n")
	sb.WriteString("    return this.plugins.get(id) as T | undefined;\n")
	sb.WriteString("  }\n\n")

	// Generate type-safe plugin accessors
	sb.WriteString("  public readonly $plugins = {\n")

	for _, m := range g.manifests {
		if m.PluginID != "core" {
			pluginClassName := g.pascalCase(m.PluginID) + "Plugin"
			sb.WriteString(fmt.Sprintf("    %s: (): %s | undefined => this.getPlugin<%s>('%s'),\n",
				g.camelCase(m.PluginID), pluginClassName, pluginClassName, m.PluginID))
		}
	}

	sb.WriteString("  };\n\n")

	// Generate request helper
	sb.WriteString("  public async request<T>(\n")
	sb.WriteString("    method: string,\n")
	sb.WriteString("    path: string,\n")
	sb.WriteString("    options?: {\n")
	sb.WriteString("      body?: any;\n")
	sb.WriteString("      query?: Record<string, string>;\n")
	sb.WriteString("      auth?: boolean;\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ): Promise<T> {\n")
	sb.WriteString("    const url = new URL(this.basePath + path, this.baseURL);\n\n")
	sb.WriteString("    if (options?.query) {\n")
	sb.WriteString("      for (const [key, value] of Object.entries(options.query)) {\n")
	sb.WriteString("        url.searchParams.append(key, value);\n")
	sb.WriteString("      }\n")
	sb.WriteString("    }\n\n")
	sb.WriteString("    const headers: Record<string, string> = {\n")
	sb.WriteString("      'Content-Type': 'application/json',\n")
	sb.WriteString("      ...this.headers,\n")
	sb.WriteString("    };\n\n")
	sb.WriteString("    if (options?.auth && this.token) {\n")
	sb.WriteString("      headers['Authorization'] = `Bearer ${this.token}`;\n")
	sb.WriteString("    }\n\n")
	sb.WriteString("    if (this.apiKey) {\n")
	sb.WriteString("      headers[this.apiKeyHeader] = this.apiKey;\n")
	sb.WriteString("    }\n\n")
	sb.WriteString("    const response = await fetch(url.toString(), {\n")
	sb.WriteString("      method,\n")
	sb.WriteString("      headers,\n")
	sb.WriteString("      body: options?.body ? JSON.stringify(options.body) : undefined,\n")
	sb.WriteString("      credentials: 'include',\n")
	sb.WriteString("    });\n\n")
	sb.WriteString("    if (!response.ok) {\n")
	sb.WriteString("      const error = await response.json().catch(() => ({ error: response.statusText }));\n")
	sb.WriteString("      throw createErrorFromResponse(response.status, error.error || error.message || 'Request failed');\n")
	sb.WriteString("    }\n\n")
	sb.WriteString("    return response.json();\n")
	sb.WriteString("  }\n\n")

	// Generate core methods
	if coreManifest != nil {
		for _, route := range coreManifest.Routes {
			g.generateTSMethod(&sb, coreManifest, &route)
		}
	}

	sb.WriteString("}\n")

	return g.writeFile("src/client.ts", sb.String())
}

func (g *TypeScriptGenerator) generateTSMethod(sb *strings.Builder, m *manifest.Manifest, route *manifest.Route) {
	methodName := g.camelCase(route.Name)

	// Generate method signature
	fmt.Fprintf(sb, "  async %s(", methodName)

	// Build parameters in order: path params, request body, query params
	params := []string{}

	// 1. Path parameters - keep as params object
	if len(route.Params) > 0 {
		params = append(params, fmt.Sprintf("params: { %s }", g.generateTSParamsType(route)))
	}

	// 2. Request body
	if len(route.Request) > 0 {
		params = append(params, fmt.Sprintf("request: { %s }", g.generateTSRequestType(route)))
	}

	// 3. Query parameters (optional)
	if len(route.Query) > 0 {
		params = append(params, fmt.Sprintf("query?: { %s }", g.generateTSQueryType(route)))
	}

	sb.WriteString(strings.Join(params, ", "))
	fmt.Fprintf(sb, "): Promise<{ %s }> {\n", g.generateTSResponseType(route))

	// Build path with interpolated parameters
	path := route.Path
	if len(route.Params) > 0 {
		for paramName := range route.Params {
			// Replace both {paramName} and :paramName styles with params.paramName
			path = strings.ReplaceAll(path, "{"+paramName+"}", "${params."+paramName+"}")
			path = strings.ReplaceAll(path, ":"+paramName, "${params."+paramName+"}")
		}

		fmt.Fprintf(sb, "    const path = `%s`;\n", path)
	} else {
		fmt.Fprintf(sb, "    const path = '%s';\n", path)
	}

	// Make request
	sb.WriteString("    return this.request")

	if len(route.Response) > 0 {
		fmt.Fprintf(sb, "<{ %s }>", g.generateTSResponseType(route))
	}

	fmt.Fprintf(sb, "('%s', path", strings.ToUpper(route.Method))

	if len(route.Request) > 0 || len(route.Query) > 0 || route.Auth {
		sb.WriteString(", {\n")

		if len(route.Request) > 0 {
			sb.WriteString("      body: request,\n")
		}

		if len(route.Query) > 0 {
			sb.WriteString("      query,\n")
		}

		if route.Auth {
			sb.WriteString("      auth: true,\n")
		}

		sb.WriteString("    }")
	}

	sb.WriteString(");\n")
	sb.WriteString("  }\n\n")
}

func (g *TypeScriptGenerator) generateTSRequestType(route *manifest.Route) string {
	var parts []string

	for name, typeStr := range route.Request {
		// Skip fields with empty or invalid names (embedded fields)
		if name == "" || name == "-" {
			continue
		}

		field := manifest.ParseField(name, typeStr)

		tsType := g.mapTypeToTS(field.Type)
		if field.Array {
			tsType += "[]"
		}

		optional := ""
		if !field.Required {
			optional = "?"
		}

		parts = append(parts, fmt.Sprintf("%s%s: %s", field.Name, optional, tsType))
	}

	return strings.Join(parts, "; ")
}

func (g *TypeScriptGenerator) generateTSParamsType(route *manifest.Route) string {
	var parts []string

	for name, typeStr := range route.Params {
		// Skip fields with empty or invalid names (embedded fields)
		if name == "" || name == "-" {
			continue
		}

		field := manifest.ParseField(name, typeStr)
		tsType := g.mapTypeToTS(field.Type)
		parts = append(parts, fmt.Sprintf("%s: %s", field.Name, tsType))
	}

	return strings.Join(parts, "; ")
}

func (g *TypeScriptGenerator) generateTSQueryType(route *manifest.Route) string {
	var parts []string

	for name, typeStr := range route.Query {
		// Skip fields with empty or invalid names (embedded fields)
		if name == "" || name == "-" {
			continue
		}

		field := manifest.ParseField(name, typeStr)
		tsType := g.mapTypeToTS(field.Type)
		optional := "?"
		parts = append(parts, fmt.Sprintf("%s%s: %s", field.Name, optional, tsType))
	}

	return strings.Join(parts, "; ")
}

func (g *TypeScriptGenerator) generateTSResponseType(route *manifest.Route) string {
	var parts []string

	for name, typeStr := range route.Response {
		// Skip fields with empty or invalid names (embedded fields)
		if name == "" || name == "-" {
			continue
		}

		field := manifest.ParseField(name, typeStr)

		tsType := g.mapTypeToTS(field.Type)
		if field.Array {
			tsType += "[]"
		}

		parts = append(parts, fmt.Sprintf("%s: %s", field.Name, tsType))
	}

	return strings.Join(parts, "; ")
}

func (g *TypeScriptGenerator) generatePlugins() error {
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

func (g *TypeScriptGenerator) generatePluginFile(m *manifest.Manifest) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("// Auto-generated %s plugin\n\n", m.PluginID))
	sb.WriteString("import { ClientPlugin } from '../plugin';\n")
	sb.WriteString("import { AuthsomeClient } from '../client';\n")
	sb.WriteString("import * as types from '../types';\n\n")

	// Generate plugin class
	pluginName := g.pascalCase(m.PluginID) + "Plugin"
	sb.WriteString(fmt.Sprintf("export class %s implements ClientPlugin {\n", pluginName))
	sb.WriteString(fmt.Sprintf("  readonly id = '%s';\n", m.PluginID))
	sb.WriteString("  private client!: AuthsomeClient;\n\n")

	sb.WriteString("  init(client: AuthsomeClient): void {\n")
	sb.WriteString("    this.client = client;\n")
	sb.WriteString("  }\n\n")

	// Generate plugin methods
	generatedMethods := make(map[string]bool)

	for _, route := range m.Routes {
		methodName := g.camelCase(route.Name)
		// Skip duplicate method names (can happen if routes are extracted multiple times)
		if generatedMethods[methodName] {
			continue
		}

		generatedMethods[methodName] = true

		g.generatePluginTSMethod(&sb, m, &route)
	}

	sb.WriteString("}\n\n")

	// Generate factory function
	sb.WriteString(fmt.Sprintf("export function %sClient(): %s {\n", g.camelCase(m.PluginID), pluginName))
	sb.WriteString(fmt.Sprintf("  return new %s();\n", pluginName))
	sb.WriteString("}\n")

	return g.writeFile(fmt.Sprintf("src/plugins/%s.ts", m.PluginID), sb.String())
}

func (g *TypeScriptGenerator) generatePluginTSMethod(sb *strings.Builder, m *manifest.Manifest, route *manifest.Route) {
	methodName := g.camelCase(route.Name)

	fmt.Fprintf(sb, "  async %s(", methodName)

	// Build parameters in order: path params, request body, query params
	params := []string{}

	// 1. Path parameters - keep as params object
	if len(route.Params) > 0 {
		params = append(params, fmt.Sprintf("params: { %s }", g.generateTSParamsType(route)))
	}

	// 2. Request body or query params
	method := strings.ToUpper(route.Method)
	isGetLike := method == "GET" || method == "DELETE" || method == "HEAD"
	hasRequest := len(route.Request) > 0 || route.RequestType != ""

	if hasRequest {
		// Use named type if available, otherwise inline type
		var requestTypeStr string

		if route.RequestType != "" {
			// Strip package qualifier if present (e.g., "responses.CreateRequest" -> "CreateRequest")
			typeName := route.RequestType
			if idx := strings.LastIndex(typeName, "."); idx != -1 {
				typeName = typeName[idx+1:]
			}

			requestTypeStr = "types." + typeName
		} else if len(route.Request) > 0 {
			requestTypeStr = fmt.Sprintf("{ %s }", g.generateTSRequestType(route))
		}

		// For GET/DELETE/HEAD, make request optional (query params)
		// For POST/PUT/PATCH, make request required (body)
		if isGetLike {
			params = append(params, "request?: "+requestTypeStr)
		} else {
			params = append(params, "request: "+requestTypeStr)
		}
	}

	// 3. Explicit query parameters (optional, separate from request)
	if len(route.Query) > 0 && !hasRequest {
		params = append(params, fmt.Sprintf("query?: { %s }", g.generateTSQueryType(route)))
	}

	sb.WriteString(strings.Join(params, ", "))

	// Use named response type if available, otherwise inline type
	var responseType string

	if route.ResponseType != "" {
		// Strip package qualifier if present (e.g., "responses.StatusResponse" -> "StatusResponse")
		typeName := route.ResponseType
		if idx := strings.LastIndex(typeName, "."); idx != -1 {
			typeName = typeName[idx+1:]
		}

		responseType = "types." + typeName
	} else if len(route.Response) > 0 {
		responseType = fmt.Sprintf("{ %s }", g.generateTSResponseType(route))
	} else {
		responseType = "void"
	}

	fmt.Fprintf(sb, "): Promise<%s> {\n", responseType)

	// Build path with interpolated parameters
	path := route.Path
	// Prepend base path if defined in manifest
	if m.BasePath != "" {
		path = m.BasePath + path
	}

	if len(route.Params) > 0 {
		for paramName := range route.Params {
			// Replace both {paramName} and :paramName styles with params.paramName
			path = strings.ReplaceAll(path, "{"+paramName+"}", "${params."+paramName+"}")
			path = strings.ReplaceAll(path, ":"+paramName, "${params."+paramName+"}")
		}

		fmt.Fprintf(sb, "    const path = `%s`;\n", path)
	} else {
		fmt.Fprintf(sb, "    const path = '%s';\n", path)
	}

	fmt.Fprintf(sb, "    return this.client.request<%s>('%s', path", responseType, strings.ToUpper(route.Method))

	// For GET/DELETE methods, request goes as query params
	// For POST/PUT/PATCH methods, request goes as body
	// Reuse variables declared above
	hasExplicitQuery := len(route.Query) > 0
	hasRequestBody := hasRequest && !isGetLike
	hasQueryParams := hasExplicitQuery || (hasRequest && isGetLike)

	if hasRequestBody || hasQueryParams || route.Auth {
		sb.WriteString(", {\n")

		if hasRequestBody {
			sb.WriteString("      body: request,\n")
		}

		if hasQueryParams {
			// If there are explicit query params (not from request), use 'query' parameter
			// Otherwise convert request object to query params
			if hasExplicitQuery && !hasRequest {
				sb.WriteString("      query: this.client.toQueryParams(query),\n")
			} else {
				sb.WriteString("      query: this.client.toQueryParams(request),\n")
			}
		}

		if route.Auth {
			sb.WriteString("      auth: true,\n")
		}

		sb.WriteString("    }")
	}

	sb.WriteString(");\n")
	sb.WriteString("  }\n\n")
}

func (g *TypeScriptGenerator) generateIndex() error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated exports\n\n")
	sb.WriteString("export { AuthsomeClient, AuthsomeClientConfig } from './client';\n")
	sb.WriteString("export { ClientPlugin } from './plugin';\n")
	sb.WriteString("export * from './types';\n")
	sb.WriteString("export * from './errors';\n\n")

	// Export plugins with explicit class names for type safety
	sb.WriteString("// Plugin exports\n")

	for _, m := range g.manifests {
		if m.PluginID == "core" {
			continue
		}

		pluginClassName := g.pascalCase(m.PluginID) + "Plugin"
		factoryName := g.camelCase(m.PluginID) + "Client"
		sb.WriteString(fmt.Sprintf("export { %s, %s } from './plugins/%s';\n", pluginClassName, factoryName, m.PluginID))
	}

	return g.writeFile("src/index.ts", sb.String())
}

// isPrimitiveType checks if a Go type is a primitive type.
func (g *TypeScriptGenerator) isPrimitiveType(goType string) bool {
	primitives := map[string]bool{
		"string": true, "int": true, "int32": true, "int64": true,
		"uint": true, "uint32": true, "uint64": true,
		"float32": true, "float64": true, "bool": true, "boolean": true,
		"byte": true, "object": true, "map": true, "error": true,
		"any": true, "interface{}": true, "Duration": true,
	}

	return primitives[goType]
}

// mapTypeToTSForTypesFile maps Go types to TypeScript without the types. prefix
// Used when generating the types.ts file itself.
func (g *TypeScriptGenerator) mapTypeToTSForTypesFile(goType string) string {
	// Handle empty types
	if goType == "" {
		return "any"
	}

	// Handle Go map syntax: map[keyType]valueType
	if strings.HasPrefix(goType, "map[") {
		// Extract key and value types
		// Simple extraction: map[string]interface{} -> Record<string, any>
		if strings.Contains(goType, "string]") {
			return "Record<string, any>"
		}

		return "Record<string, any>"
	}

	// Handle array notation (shouldn't happen after ParseField, but just in case)
	if after, ok := strings.CutPrefix(goType, "[]"); ok {
		innerType := g.mapTypeToTSForTypesFile(after)

		return innerType + "[]"
	}

	// Handle pointer notation
	if after, ok := strings.CutPrefix(goType, "*"); ok {
		innerType := g.mapTypeToTSForTypesFile(after)

		return innerType + " | undefined"
	}

	// Handle qualified types (e.g., xid.ID, time.Time, types.User, apikey.APIKey)
	if strings.Contains(goType, ".") {
		// For qualified types, just return string or appropriate mapping
		switch {
		case strings.HasSuffix(goType, ".ID"):
			return "string"
		case strings.HasSuffix(goType, ".Time"):
			return "string"
		case strings.HasPrefix(goType, "types."):
			// Already prefixed with types. - just use the type name without prefix
			return strings.TrimPrefix(goType, "types.")
		default:
			// For other package-qualified types (e.g., apikey.APIKey, user.User)
			// Extract just the type name (last part after the dot)
			parts := strings.Split(goType, ".")
			if len(parts) == 2 {
				return parts[1]
			}

			return "any"
		}
	}

	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64", "float32", "float64":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "byte":
		return "number"
	case "object", "map":
		return "Record<string, any>"
	case "error":
		return "Error"
	case "any", "interface{}":
		return "any"
	case "Duration":
		// time.Duration in Go is typically represented as a string in JSON (e.g., "5m", "1h")
		return "string"
	// Enums and special types that should be strings
	case "ComplianceStandard", "RecoveryMethod", "FactorType", "FactorPriority", "RecoveryStatus", "SecurityLevel", "ChallengeStatus", "FactorStatus", "VerificationMethod", "RiskLevel":
		return "string"
	// Special map types
	case "JSONBMap":
		return "Record<string, any>"
	default:
		// It's a custom type - NO prefix since we're in types.ts
		return goType
	}
}

// mapTypeToTS maps Go types to TypeScript with the types. prefix for use in plugin files.
func (g *TypeScriptGenerator) mapTypeToTS(goType string) string {
	// Handle empty types
	if goType == "" {
		return "any"
	}

	// Handle Go map syntax: map[keyType]valueType
	if strings.HasPrefix(goType, "map[") {
		// Extract key and value types
		// Simple extraction: map[string]interface{} -> Record<string, any>
		if strings.Contains(goType, "string]") {
			return "Record<string, any>"
		}

		return "Record<string, any>"
	}

	// Handle array notation (shouldn't happen after ParseField, but just in case)
	if after, ok := strings.CutPrefix(goType, "[]"); ok {
		innerType := g.mapTypeToTS(after)

		return innerType + "[]"
	}

	// Handle pointer notation
	if after, ok := strings.CutPrefix(goType, "*"); ok {
		innerType := g.mapTypeToTS(after)

		return innerType + " | undefined"
	}

	// Handle qualified types (e.g., xid.ID, time.Time)
	if strings.Contains(goType, ".") {
		// For qualified types, just return string or appropriate mapping
		switch {
		case strings.HasSuffix(goType, ".ID"):
			return "string"
		case strings.HasSuffix(goType, ".Time"):
			return "string"
		case strings.HasPrefix(goType, "types."):
			// Already has types. prefix
			return goType
		default:
			return "any"
		}
	}

	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64", "float32", "float64":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "byte":
		return "number"
	case "object", "map":
		return "Record<string, any>"
	case "error":
		return "Error"
	case "any", "interface{}":
		return "any"
	// Enums and special types that should be strings
	case "ComplianceStandard", "RecoveryMethod", "FactorType", "FactorPriority", "RecoveryStatus", "SecurityLevel", "ChallengeStatus", "FactorStatus", "VerificationMethod", "RiskLevel":
		return "string"
	// Special map types
	case "JSONBMap":
		return "Record<string, any>"
	default:
		// It's a custom type defined in types.ts - prefix with types.
		return "types." + goType
	}
}

func (g *TypeScriptGenerator) camelCase(s string) string {
	if len(s) == 0 {
		return s
	}

	return strings.ToLower(s[:1]) + s[1:]
}

func (g *TypeScriptGenerator) pascalCase(s string) string {
	if len(s) == 0 {
		return s
	}

	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *TypeScriptGenerator) writeFile(path string, content string) error {
	fullPath := filepath.Join(g.outputDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(fullPath, []byte(content), 0644)
}
