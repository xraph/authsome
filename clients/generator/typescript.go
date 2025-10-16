package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/clients/manifest"
)

// TypeScriptGenerator generates TypeScript client code
type TypeScriptGenerator struct {
	outputDir string
	manifests []*manifest.Manifest
}

// NewTypeScriptGenerator creates a new TypeScript generator
func NewTypeScriptGenerator(outputDir string, manifests []*manifest.Manifest) *TypeScriptGenerator {
	return &TypeScriptGenerator{
		outputDir: outputDir,
		manifests: manifests,
	}
}

// Generate generates TypeScript client code
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

	// Generate type definitions
	for _, t := range typeMap {
		sb.WriteString(fmt.Sprintf("export interface %s {\n", t.Name))
		for name, typeStr := range t.Fields {
			field := manifest.ParseField(name, typeStr)
			tsType := g.mapTypeToTS(field.Type)

			if field.Array {
				tsType += "[]"
			}

			optional := ""
			if !field.Required {
				optional = "?"
			}

			sb.WriteString(fmt.Sprintf("  %s%s: %s;\n", field.Name, optional, tsType))
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
	sb.WriteString("import * as types from './types';\n\n")

	// Find core manifest
	var coreManifest *manifest.Manifest
	for _, m := range g.manifests {
		if m.PluginID == "core" {
			coreManifest = m
			break
		}
	}

	sb.WriteString("export interface AuthsomeClientConfig {\n")
	sb.WriteString("  baseURL: string;\n")
	sb.WriteString("  plugins?: ClientPlugin[];\n")
	sb.WriteString("  token?: string;\n")
	sb.WriteString("  headers?: Record<string, string>;\n")
	sb.WriteString("}\n\n")

	sb.WriteString("export class AuthsomeClient {\n")
	sb.WriteString("  private baseURL: string;\n")
	sb.WriteString("  private token?: string;\n")
	sb.WriteString("  private headers: Record<string, string>;\n")
	sb.WriteString("  private plugins: Map<string, ClientPlugin>;\n\n")

	sb.WriteString("  constructor(config: AuthsomeClientConfig) {\n")
	sb.WriteString("    this.baseURL = config.baseURL;\n")
	sb.WriteString("    this.token = config.token;\n")
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

	sb.WriteString("  getPlugin<T extends ClientPlugin>(id: string): T | undefined {\n")
	sb.WriteString("    return this.plugins.get(id) as T | undefined;\n")
	sb.WriteString("  }\n\n")

	// Generate request helper
	sb.WriteString("  private async request<T>(\n")
	sb.WriteString("    method: string,\n")
	sb.WriteString("    path: string,\n")
	sb.WriteString("    options?: {\n")
	sb.WriteString("      body?: any;\n")
	sb.WriteString("      query?: Record<string, string>;\n")
	sb.WriteString("      auth?: boolean;\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ): Promise<T> {\n")
	sb.WriteString("    const url = new URL(path, this.baseURL);\n\n")
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
	sb.WriteString(fmt.Sprintf("  async %s(", methodName))

	// Parameters
	params := []string{}
	if len(route.Request) > 0 {
		params = append(params, fmt.Sprintf("request: { %s }", g.generateTSRequestType(route)))
	}
	if len(route.Params) > 0 {
		params = append(params, fmt.Sprintf("params: { %s }", g.generateTSParamsType(route)))
	}
	if len(route.Query) > 0 {
		params = append(params, fmt.Sprintf("query?: { %s }", g.generateTSQueryType(route)))
	}

	sb.WriteString(strings.Join(params, ", "))
	sb.WriteString(fmt.Sprintf("): Promise<{ %s }> {\n", g.generateTSResponseType(route)))

	// Build path
	path := route.Path
	if len(route.Params) > 0 {
		for paramName := range route.Params {
			path = strings.ReplaceAll(path, "{"+paramName+"}", "${params."+paramName+"}")
		}
		sb.WriteString(fmt.Sprintf("    const path = `%s%s`;\n", m.BasePath, path))
	} else {
		sb.WriteString(fmt.Sprintf("    const path = '%s%s';\n", m.BasePath, path))
	}

	// Make request
	sb.WriteString("    return this.request")
	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("<{ %s }>", g.generateTSResponseType(route)))
	}
	sb.WriteString(fmt.Sprintf("('%s', path", strings.ToUpper(route.Method)))

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
		field := manifest.ParseField(name, typeStr)
		tsType := g.mapTypeToTS(field.Type)
		parts = append(parts, fmt.Sprintf("%s: %s", field.Name, tsType))
	}
	return strings.Join(parts, "; ")
}

func (g *TypeScriptGenerator) generateTSQueryType(route *manifest.Route) string {
	var parts []string
	for name, typeStr := range route.Query {
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
	for _, route := range m.Routes {
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

	sb.WriteString(fmt.Sprintf("  async %s(", methodName))

	params := []string{}
	if len(route.Request) > 0 {
		params = append(params, fmt.Sprintf("request: { %s }", g.generateTSRequestType(route)))
	}
	if len(route.Params) > 0 {
		params = append(params, fmt.Sprintf("params: { %s }", g.generateTSParamsType(route)))
	}
	if len(route.Query) > 0 {
		params = append(params, fmt.Sprintf("query?: { %s }", g.generateTSQueryType(route)))
	}

	sb.WriteString(strings.Join(params, ", "))
	sb.WriteString(fmt.Sprintf("): Promise<{ %s }> {\n", g.generateTSResponseType(route)))

	// Build path
	path := route.Path
	if len(route.Params) > 0 {
		for paramName := range route.Params {
			path = strings.ReplaceAll(path, "{"+paramName+"}", "${params."+paramName+"}")
		}
		sb.WriteString(fmt.Sprintf("    const path = `%s%s`;\n", m.BasePath, path))
	} else {
		sb.WriteString(fmt.Sprintf("    const path = '%s%s';\n", m.BasePath, path))
	}

	sb.WriteString("    return (this.client as any).request")
	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("<{ %s }>", g.generateTSResponseType(route)))
	}
	sb.WriteString(fmt.Sprintf("('%s', path", strings.ToUpper(route.Method)))

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

func (g *TypeScriptGenerator) generateIndex() error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated exports\n\n")
	sb.WriteString("export { AuthsomeClient, AuthsomeClientConfig } from './client';\n")
	sb.WriteString("export { ClientPlugin } from './plugin';\n")
	sb.WriteString("export * from './types';\n")
	sb.WriteString("export * from './errors';\n\n")

	// Export plugins
	for _, m := range g.manifests {
		if m.PluginID == "core" {
			continue
		}
		sb.WriteString(fmt.Sprintf("export * from './plugins/%s';\n", m.PluginID))
	}

	return g.writeFile("src/index.ts", sb.String())
}

func (g *TypeScriptGenerator) mapTypeToTS(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int32", "int64", "uint", "uint32", "uint64", "float32", "float64":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "object", "map":
		return "Record<string, any>"
	default:
		// Assume it's a custom type
		return goType
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
