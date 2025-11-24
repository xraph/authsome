package generator

import (
	"strings"
)

// generateHTTPMiddleware generates http.RoundTripper middleware for the Go client
func (g *GoGenerator) generateHTTPMiddleware() error {
	var sb strings.Builder

	sb.WriteString("package authsome\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString(")\n\n")
	sb.WriteString("// Auto-generated http.RoundTripper middleware\n\n")

	// RoundTripperMiddleware struct
	sb.WriteString("// RoundTripperMiddleware wraps http.RoundTripper with auth injection\n")
	sb.WriteString("// Use this to automatically inject authentication headers and context\n")
	sb.WriteString("// into all HTTP requests made by a standard http.Client\n")
	sb.WriteString("type RoundTripperMiddleware struct {\n")
	sb.WriteString("\tclient    *Client\n")
	sb.WriteString("\ttransport http.RoundTripper\n")
	sb.WriteString("}\n\n")

	// RoundTripper method
	sb.WriteString("// RoundTripper returns an http.RoundTripper that injects authentication\n")
	sb.WriteString("// You can use this with any standard http.Client:\n")
	sb.WriteString("//\n")
	sb.WriteString("//   httpClient := &http.Client{\n")
	sb.WriteString("//       Transport: authClient.RoundTripper(),\n")
	sb.WriteString("//   }\n")
	sb.WriteString("//\n")
	sb.WriteString("func (c *Client) RoundTripper() http.RoundTripper {\n")
	sb.WriteString("\ttransport := c.httpClient.Transport\n")
	sb.WriteString("\tif transport == nil {\n")
	sb.WriteString("\t\ttransport = http.DefaultTransport\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn &RoundTripperMiddleware{\n")
	sb.WriteString("\t\tclient:    c,\n")
	sb.WriteString("\t\ttransport: transport,\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	// RoundTrip method
	sb.WriteString("// RoundTrip implements http.RoundTripper interface\n")
	sb.WriteString("func (m *RoundTripperMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {\n")
	sb.WriteString("\t// Clone request to avoid mutation of the original\n")
	sb.WriteString("\treq = req.Clone(req.Context())\n\n")

	sb.WriteString("\t// Inject API key if available\n")
	sb.WriteString("\tif m.client.apiKey != \"\" {\n")
	sb.WriteString("\t\treq.Header.Set(\"Authorization\", \"ApiKey \"+m.client.apiKey)\n")
	sb.WriteString("\t} else if m.client.token != \"\" {\n")
	sb.WriteString("\t\t// Otherwise inject session token\n")
	sb.WriteString("\t\treq.Header.Set(\"Authorization\", \"Bearer \"+m.client.token)\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString("\t// Inject context headers if available\n")
	sb.WriteString("\tif m.client.appID != \"\" {\n")
	sb.WriteString("\t\treq.Header.Set(\"X-App-ID\", m.client.appID)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tif m.client.environmentID != \"\" {\n")
	sb.WriteString("\t\treq.Header.Set(\"X-Environment-ID\", m.client.environmentID)\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString("\t// Inject custom headers\n")
	sb.WriteString("\tfor k, v := range m.client.headers {\n")
	sb.WriteString("\t\t// Don't override existing headers\n")
	sb.WriteString("\t\tif req.Header.Get(k) == \"\" {\n")
	sb.WriteString("\t\t\treq.Header.Set(k, v)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n\n")

	sb.WriteString("\t// Execute the request\n")
	sb.WriteString("\treturn m.transport.RoundTrip(req)\n")
	sb.WriteString("}\n\n")

	// NewHTTPClientWithAuth helper
	sb.WriteString("// NewHTTPClientWithAuth creates a new http.Client with automatic auth injection\n")
	sb.WriteString("// This is a convenience function for creating HTTP clients with AuthSome authentication\n")
	sb.WriteString("func (c *Client) NewHTTPClientWithAuth() *http.Client {\n")
	sb.WriteString("\treturn &http.Client{\n")
	sb.WriteString("\t\tTransport: c.RoundTripper(),\n")
	sb.WriteString("\t\tJar:       c.cookieJar,\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n")

	return g.writeFile("middleware_http.go", sb.String())
}

