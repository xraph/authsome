package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xraph/authsome/clients/manifest"
)

// RustGenerator generates Rust client code
type RustGenerator struct {
	outputDir string
	manifests []*manifest.Manifest
}

// NewRustGenerator creates a new Rust generator
func NewRustGenerator(outputDir string, manifests []*manifest.Manifest) *RustGenerator {
	return &RustGenerator{
		outputDir: outputDir,
		manifests: manifests,
	}
}

// Generate generates Rust client code
func (g *RustGenerator) Generate() error {
	if err := g.createDirectories(); err != nil {
		return err
	}

	if err := g.generateCargoToml(); err != nil {
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

	if err := g.generateLib(); err != nil {
		return err
	}

	return nil
}

func (g *RustGenerator) createDirectories() error {
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

func (g *RustGenerator) generateCargoToml() error {
	content := `[package]
name = "authsome-client"
version = "1.0.0"
edition = "2021"
description = "Rust client for AuthSome authentication"
license = "MIT"

[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
reqwest = { version = "0.11", features = ["json", "cookies"] }
tokio = { version = "1.0", features = ["full"] }
thiserror = "1.0"
url = "2.4"

[dev-dependencies]
tokio-test = "0.4"
`
	return g.writeFile("Cargo.toml", content)
}

func (g *RustGenerator) generateTypes() error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated Rust types\n\n")
	sb.WriteString("use serde::{Deserialize, Serialize};\n\n")

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
			sb.WriteString(fmt.Sprintf("/// %s\n", t.Description))
		}
		sb.WriteString("#[derive(Debug, Clone, Serialize, Deserialize)]\n")
		sb.WriteString(fmt.Sprintf("pub struct %s {\n", t.Name))

		for name, typeStr := range t.Fields {
			field := manifest.ParseField(name, typeStr)
			rustType := g.mapTypeToRust(field.Type)

			if field.Array {
				rustType = fmt.Sprintf("Vec<%s>", rustType)
			}

			if !field.Required {
				rustType = fmt.Sprintf("Option<%s>", rustType)
			}

			// Add serde attribute for JSON field mapping
			sb.WriteString(fmt.Sprintf("    #[serde(rename = \"%s\"", field.Name))
			if !field.Required {
				sb.WriteString(", skip_serializing_if = \"Option::is_none\"")
			}
			sb.WriteString(")]\n")

			sb.WriteString(fmt.Sprintf("    pub %s: %s,\n", g.snakeCase(field.Name), rustType))
		}
		sb.WriteString("}\n\n")
	}

	return g.writeFile("src/types.rs", sb.String())
}

func (g *RustGenerator) generateErrors() error {
	content := `// Auto-generated error types

use thiserror::Error;

#[derive(Debug, Error)]
pub enum AuthsomeError {
    #[error("Network error: {0}")]
    Network(String),
    
    #[error("Validation error: {0}")]
    Validation(String),
    
    #[error("Unauthorized: {0}")]
    Unauthorized(String),
    
    #[error("Forbidden: {0}")]
    Forbidden(String),
    
    #[error("Not found: {0}")]
    NotFound(String),
    
    #[error("Conflict: {0}")]
    Conflict(String),
    
    #[error("Rate limit exceeded: {0}")]
    RateLimit(String),
    
    #[error("Server error: {0}")]
    Server(String),
    
    #[error("API error (status {status}): {message}")]
    Api {
        status: u16,
        message: String,
    },
    
    #[error("Request error: {0}")]
    Request(#[from] reqwest::Error),
    
    #[error("JSON error: {0}")]
    Json(#[from] serde_json::Error),
}

impl AuthsomeError {
    pub fn from_status(status: u16, message: String) -> Self {
        match status {
            400 => Self::Validation(message),
            401 => Self::Unauthorized(message),
            403 => Self::Forbidden(message),
            404 => Self::NotFound(message),
            409 => Self::Conflict(message),
            429 => Self::RateLimit(message),
            500..=599 => Self::Server(message),
            _ => Self::Api { status, message },
        }
    }
}

pub type Result<T> = std::result::Result<T, AuthsomeError>;
`
	return g.writeFile("src/error.rs", content)
}

func (g *RustGenerator) generatePlugin() error {
	content := `// Auto-generated plugin trait

use crate::client::AuthsomeClient;
use crate::error::Result;

pub trait ClientPlugin: Send + Sync {
    /// Returns the unique plugin identifier
    fn id(&self) -> &str;
    
    /// Initialize plugin with base client
    fn init(&mut self, client: AuthsomeClient);
}
`
	return g.writeFile("src/plugin.rs", content)
}

func (g *RustGenerator) generateClient() error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated AuthSome client\n\n")
	sb.WriteString("use reqwest::{Client as HttpClient, Method, RequestBuilder};\n")
	sb.WriteString("use serde::{de::DeserializeOwned, Serialize};\n")
	sb.WriteString("use std::collections::HashMap;\n")
	sb.WriteString("use std::sync::Arc;\n\n")
	sb.WriteString("use crate::error::{AuthsomeError, Result};\n")
	sb.WriteString("use crate::plugin::ClientPlugin;\n")
	sb.WriteString("use crate::types::*;\n\n")

	// Find core manifest
	var coreManifest *manifest.Manifest
	for _, m := range g.manifests {
		if m.PluginID == "core" {
			coreManifest = m
			break
		}
	}

	sb.WriteString("#[derive(Clone)]\n")
	sb.WriteString("pub struct AuthsomeClient {\n")
	sb.WriteString("    base_url: String,\n")
	sb.WriteString("    http_client: HttpClient,\n")
	sb.WriteString("    token: Option<String>,\n")
	sb.WriteString("    headers: HashMap<String, String>,\n")
	sb.WriteString("}\n\n")

	sb.WriteString("impl AuthsomeClient {\n")
	sb.WriteString("    pub fn builder() -> AuthsomeClientBuilder {\n")
	sb.WriteString("        AuthsomeClientBuilder::default()\n")
	sb.WriteString("    }\n\n")

	sb.WriteString("    pub fn new(base_url: impl Into<String>) -> Self {\n")
	sb.WriteString("        Self {\n")
	sb.WriteString("            base_url: base_url.into(),\n")
	sb.WriteString("            http_client: HttpClient::new(),\n")
	sb.WriteString("            token: None,\n")
	sb.WriteString("            headers: HashMap::new(),\n")
	sb.WriteString("        }\n")
	sb.WriteString("    }\n\n")

	sb.WriteString("    pub fn set_token(&mut self, token: String) {\n")
	sb.WriteString("        self.token = Some(token);\n")
	sb.WriteString("    }\n\n")

	// Generate request helper
	sb.WriteString("    async fn request<T: DeserializeOwned>(\n")
	sb.WriteString("        &self,\n")
	sb.WriteString("        method: Method,\n")
	sb.WriteString("        path: &str,\n")
	sb.WriteString("        body: Option<impl Serialize>,\n")
	sb.WriteString("        auth: bool,\n")
	sb.WriteString("    ) -> Result<T> {\n")
	sb.WriteString("        let url = format!(\"{}{}\", self.base_url, path);\n")
	sb.WriteString("        let mut req = self.http_client.request(method, &url);\n\n")
	sb.WriteString("        req = req.header(\"Content-Type\", \"application/json\");\n\n")
	sb.WriteString("        for (key, value) in &self.headers {\n")
	sb.WriteString("            req = req.header(key, value);\n")
	sb.WriteString("        }\n\n")
	sb.WriteString("        if auth {\n")
	sb.WriteString("            if let Some(token) = &self.token {\n")
	sb.WriteString("                req = req.bearer_auth(token);\n")
	sb.WriteString("            }\n")
	sb.WriteString("        }\n\n")
	sb.WriteString("        if let Some(body) = body {\n")
	sb.WriteString("            req = req.json(&body);\n")
	sb.WriteString("        }\n\n")
	sb.WriteString("        let resp = req.send().await?;\n")
	sb.WriteString("        let status = resp.status();\n\n")
	sb.WriteString("        if !status.is_success() {\n")
	sb.WriteString("            let error_body: serde_json::Value = resp.json().await.unwrap_or_default();\n")
	sb.WriteString("            let message = error_body[\"error\"].as_str()\n")
	sb.WriteString("                .or_else(|| error_body[\"message\"].as_str())\n")
	sb.WriteString("                .unwrap_or(\"Request failed\")\n")
	sb.WriteString("                .to_string();\n")
	sb.WriteString("            return Err(AuthsomeError::from_status(status.as_u16(), message));\n")
	sb.WriteString("        }\n\n")
	sb.WriteString("        Ok(resp.json().await?)\n")
	sb.WriteString("    }\n\n")

	// Generate core methods
	if coreManifest != nil {
		for _, route := range coreManifest.Routes {
			g.generateRustMethod(&sb, coreManifest, &route)
		}
	}

	sb.WriteString("}\n\n")

	// Generate builder
	sb.WriteString("#[derive(Default)]\n")
	sb.WriteString("pub struct AuthsomeClientBuilder {\n")
	sb.WriteString("    base_url: Option<String>,\n")
	sb.WriteString("    http_client: Option<HttpClient>,\n")
	sb.WriteString("    token: Option<String>,\n")
	sb.WriteString("    headers: HashMap<String, String>,\n")
	sb.WriteString("}\n\n")

	sb.WriteString("impl AuthsomeClientBuilder {\n")
	sb.WriteString("    pub fn base_url(mut self, url: impl Into<String>) -> Self {\n")
	sb.WriteString("        self.base_url = Some(url.into());\n")
	sb.WriteString("        self\n")
	sb.WriteString("    }\n\n")

	sb.WriteString("    pub fn http_client(mut self, client: HttpClient) -> Self {\n")
	sb.WriteString("        self.http_client = Some(client);\n")
	sb.WriteString("        self\n")
	sb.WriteString("    }\n\n")

	sb.WriteString("    pub fn token(mut self, token: impl Into<String>) -> Self {\n")
	sb.WriteString("        self.token = Some(token.into());\n")
	sb.WriteString("        self\n")
	sb.WriteString("    }\n\n")

	sb.WriteString("    pub fn header(mut self, key: impl Into<String>, value: impl Into<String>) -> Self {\n")
	sb.WriteString("        self.headers.insert(key.into(), value.into());\n")
	sb.WriteString("        self\n")
	sb.WriteString("    }\n\n")

	sb.WriteString("    pub fn build(self) -> Result<AuthsomeClient> {\n")
	sb.WriteString("        let base_url = self.base_url.ok_or_else(|| {\n")
	sb.WriteString("            AuthsomeError::Validation(\"base_url is required\".to_string())\n")
	sb.WriteString("        })?;\n\n")
	sb.WriteString("        Ok(AuthsomeClient {\n")
	sb.WriteString("            base_url,\n")
	sb.WriteString("            http_client: self.http_client.unwrap_or_else(HttpClient::new),\n")
	sb.WriteString("            token: self.token,\n")
	sb.WriteString("            headers: self.headers,\n")
	sb.WriteString("        })\n")
	sb.WriteString("    }\n")
	sb.WriteString("}\n")

	return g.writeFile("src/client.rs", sb.String())
}

func (g *RustGenerator) generateRustMethod(sb *strings.Builder, m *manifest.Manifest, route *manifest.Route) {
	methodName := g.snakeCase(route.Name)

	// Generate request struct if needed
	if len(route.Request) > 0 {
		sb.WriteString(fmt.Sprintf("    /// Request for %s\n", methodName))
		sb.WriteString("    #[derive(Debug, Serialize)]\n")
		sb.WriteString(fmt.Sprintf("    pub struct %sRequest {\n", route.Name))
		for name, typeStr := range route.Request {
			field := manifest.ParseField(name, typeStr)
			rustType := g.mapTypeToRust(field.Type)
			if field.Array {
				rustType = fmt.Sprintf("Vec<%s>", rustType)
			}
			if !field.Required {
				rustType = fmt.Sprintf("Option<%s>", rustType)
			}
			sb.WriteString(fmt.Sprintf("        #[serde(rename = \"%s\"", field.Name))
			if !field.Required {
				sb.WriteString(", skip_serializing_if = \"Option::is_none\"")
			}
			sb.WriteString(")]\n")
			sb.WriteString(fmt.Sprintf("        pub %s: %s,\n", g.snakeCase(field.Name), rustType))
		}
		sb.WriteString("    }\n\n")
	}

	// Generate response struct if needed
	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("    /// Response for %s\n", methodName))
		sb.WriteString("    #[derive(Debug, Deserialize)]\n")
		sb.WriteString(fmt.Sprintf("    pub struct %sResponse {\n", route.Name))
		for name, typeStr := range route.Response {
			field := manifest.ParseField(name, typeStr)
			rustType := g.mapTypeToRust(field.Type)
			if field.Array {
				rustType = fmt.Sprintf("Vec<%s>", rustType)
			}
			sb.WriteString(fmt.Sprintf("        #[serde(rename = \"%s\")]\n", field.Name))
			sb.WriteString(fmt.Sprintf("        pub %s: %s,\n", g.snakeCase(field.Name), rustType))
		}
		sb.WriteString("    }\n\n")
	}

	// Generate method
	if route.Description != "" {
		sb.WriteString(fmt.Sprintf("    /// %s\n", route.Description))
	}
	sb.WriteString(fmt.Sprintf("    pub async fn %s(\n", methodName))
	sb.WriteString("        &self,\n")

	if len(route.Request) > 0 {
		sb.WriteString(fmt.Sprintf("        request: %sRequest,\n", route.Name))
	}
	if len(route.Params) > 0 {
		for paramName, typeStr := range route.Params {
			field := manifest.ParseField(paramName, typeStr)
			sb.WriteString(fmt.Sprintf("        %s: %s,\n", g.snakeCase(field.Name), g.mapTypeToRust(field.Type)))
		}
	}

	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("    ) -> Result<%sResponse> {\n", route.Name))
	} else {
		sb.WriteString("    ) -> Result<()> {\n")
	}

	// Build path
	path := m.BasePath + route.Path
	if len(route.Params) > 0 {
		sb.WriteString("        let path = format!(\"")
		for paramName := range route.Params {
			path = strings.ReplaceAll(path, "{"+paramName+"}", "{}")
		}
		sb.WriteString(path)
		sb.WriteString("\"")
		for paramName := range route.Params {
			sb.WriteString(fmt.Sprintf(", %s", g.snakeCase(paramName)))
		}
		sb.WriteString(");\n")
	} else {
		sb.WriteString(fmt.Sprintf("        let path = \"%s\";\n", path))
	}

	// Make request
	sb.WriteString("        self.request(\n")
	sb.WriteString(fmt.Sprintf("            Method::%s,\n", strings.ToUpper(route.Method)))
	sb.WriteString("            &path,\n")

	if len(route.Request) > 0 {
		sb.WriteString("            Some(request),\n")
	} else {
		sb.WriteString("            None::<()>,\n")
	}

	if route.Auth {
		sb.WriteString("            true,\n")
	} else {
		sb.WriteString("            false,\n")
	}

	sb.WriteString("        ).await\n")
	sb.WriteString("    }\n\n")
}

func (g *RustGenerator) generatePlugins() error {
	var modContent strings.Builder
	modContent.WriteString("// Auto-generated plugin modules\n\n")

	for _, m := range g.manifests {
		if m.PluginID == "core" {
			continue
		}

		if err := g.generatePluginFile(m); err != nil {
			return err
		}

		modContent.WriteString(fmt.Sprintf("pub mod %s;\n", m.PluginID))
	}

	return g.writeFile("src/plugins/mod.rs", modContent.String())
}

func (g *RustGenerator) generatePluginFile(m *manifest.Manifest) error {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("// Auto-generated %s plugin\n\n", m.PluginID))
	sb.WriteString("use reqwest::Method;\n")
	sb.WriteString("use serde::{Deserialize, Serialize};\n\n")
	sb.WriteString("use crate::client::AuthsomeClient;\n")
	sb.WriteString("use crate::error::Result;\n")
	sb.WriteString("use crate::plugin::ClientPlugin;\n")
	sb.WriteString("use crate::types::*;\n\n")

	// Generate plugin struct
	sb.WriteString(fmt.Sprintf("pub struct %sPlugin {{\n", g.pascalCase(m.PluginID)))
	sb.WriteString("    client: Option<AuthsomeClient>,\n")
	sb.WriteString("}\n\n")

	sb.WriteString(fmt.Sprintf("impl %sPlugin {{\n", g.pascalCase(m.PluginID)))
	sb.WriteString("    pub fn new() -> Self {\n")
	sb.WriteString("        Self { client: None }\n")
	sb.WriteString("    }\n\n")

	// Generate plugin methods
	for _, route := range m.Routes {
		g.generatePluginRustMethod(&sb, m, &route)
	}

	sb.WriteString("}\n\n")

	// Implement ClientPlugin trait
	sb.WriteString(fmt.Sprintf("impl ClientPlugin for %sPlugin {{\n", g.pascalCase(m.PluginID)))
	sb.WriteString("    fn id(&self) -> &str {\n")
	sb.WriteString(fmt.Sprintf("        \"%s\"\n", m.PluginID))
	sb.WriteString("    }\n\n")
	sb.WriteString("    fn init(&mut self, client: AuthsomeClient) {\n")
	sb.WriteString("        self.client = Some(client);\n")
	sb.WriteString("    }\n")
	sb.WriteString("}\n")

	return g.writeFile(fmt.Sprintf("src/plugins/%s.rs", m.PluginID), sb.String())
}

func (g *RustGenerator) generatePluginRustMethod(sb *strings.Builder, m *manifest.Manifest, route *manifest.Route) {
	methodName := g.snakeCase(route.Name)

	// Generate request/response structs similar to client methods
	if len(route.Request) > 0 {
		sb.WriteString("    #[derive(Debug, Serialize)]\n")
		sb.WriteString(fmt.Sprintf("    pub struct %sRequest {\n", route.Name))
		for name, typeStr := range route.Request {
			field := manifest.ParseField(name, typeStr)
			rustType := g.mapTypeToRust(field.Type)
			if field.Array {
				rustType = fmt.Sprintf("Vec<%s>", rustType)
			}
			if !field.Required {
				rustType = fmt.Sprintf("Option<%s>", rustType)
			}
			sb.WriteString(fmt.Sprintf("        #[serde(rename = \"%s\"", field.Name))
			if !field.Required {
				sb.WriteString(", skip_serializing_if = \"Option::is_none\"")
			}
			sb.WriteString(")]\n")
			sb.WriteString(fmt.Sprintf("        pub %s: %s,\n", g.snakeCase(field.Name), rustType))
		}
		sb.WriteString("    }\n\n")
	}

	if len(route.Response) > 0 {
		sb.WriteString("    #[derive(Debug, Deserialize)]\n")
		sb.WriteString(fmt.Sprintf("    pub struct %sResponse {\n", route.Name))
		for name, typeStr := range route.Response {
			field := manifest.ParseField(name, typeStr)
			rustType := g.mapTypeToRust(field.Type)
			if field.Array {
				rustType = fmt.Sprintf("Vec<%s>", rustType)
			}
			sb.WriteString(fmt.Sprintf("        #[serde(rename = \"%s\")]\n", field.Name))
			sb.WriteString(fmt.Sprintf("        pub %s: %s,\n", g.snakeCase(field.Name), rustType))
		}
		sb.WriteString("    }\n\n")
	}

	// Generate method (placeholder - would need access to client's private request method)
	if route.Description != "" {
		sb.WriteString(fmt.Sprintf("    /// %s\n", route.Description))
	}
	sb.WriteString(fmt.Sprintf("    pub async fn %s(\n", methodName))
	sb.WriteString("        &self,\n")

	if len(route.Request) > 0 {
		sb.WriteString(fmt.Sprintf("        _request: %sRequest,\n", route.Name))
	}

	if len(route.Response) > 0 {
		sb.WriteString(fmt.Sprintf("    ) -> Result<%sResponse> {{\n", route.Name))
	} else {
		sb.WriteString("    ) -> Result<()> {\n")
	}

	sb.WriteString("        // TODO: Implement plugin method\n")
	sb.WriteString("        unimplemented!(\"Plugin methods need client access\")\n")
	sb.WriteString("    }\n\n")
}

func (g *RustGenerator) generateLib() error {
	var sb strings.Builder

	sb.WriteString("// Auto-generated library exports\n\n")
	sb.WriteString("pub mod client;\n")
	sb.WriteString("pub mod error;\n")
	sb.WriteString("pub mod plugin;\n")
	sb.WriteString("pub mod types;\n")
	sb.WriteString("pub mod plugins;\n\n")
	sb.WriteString("pub use client::{AuthsomeClient, AuthsomeClientBuilder};\n")
	sb.WriteString("pub use error::{AuthsomeError, Result};\n")
	sb.WriteString("pub use plugin::ClientPlugin;\n")
	sb.WriteString("pub use types::*;\n")

	return g.writeFile("src/lib.rs", sb.String())
}

func (g *RustGenerator) mapTypeToRust(t string) string {
	switch t {
	case "string":
		return "String"
	case "int", "int32", "uint", "uint32":
		return "i32"
	case "int64", "uint64":
		return "i64"
	case "float32":
		return "f32"
	case "float64":
		return "f64"
	case "bool", "boolean":
		return "bool"
	case "object", "map":
		return "serde_json::Value"
	default:
		// Assume it's a custom type
		return t
	}
}

func (g *RustGenerator) snakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func (g *RustGenerator) pascalCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (g *RustGenerator) writeFile(path string, content string) error {
	fullPath := filepath.Join(g.outputDir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullPath, []byte(content), 0644)
}
