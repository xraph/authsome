package authsome

import (
	"net/http"
)

// Auto-generated http.RoundTripper middleware

// RoundTripperMiddleware wraps http.RoundTripper with auth injection
// Use this to automatically inject authentication headers and context
// into all HTTP requests made by a standard http.Client
type RoundTripperMiddleware struct {
	client    *Client
	transport http.RoundTripper
}

// RoundTripper returns an http.RoundTripper that injects authentication
// You can use this with any standard http.Client:
//
//   httpClient := &http.Client{
//       Transport: authClient.RoundTripper(),
//   }
//
func (c *Client) RoundTripper() http.RoundTripper {
	transport := c.httpClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &RoundTripperMiddleware{
		client:    c,
		transport: transport,
	}
}

// RoundTrip implements http.RoundTripper interface
func (m *RoundTripperMiddleware) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone request to avoid mutation of the original
	req = req.Clone(req.Context())

	// Inject API key if available
	if m.client.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+m.client.apiKey)
	} else if m.client.token != "" {
		// Otherwise inject session token
		req.Header.Set("Authorization", "Bearer "+m.client.token)
	}

	// Inject context headers if available
	if m.client.appID != "" {
		req.Header.Set("X-App-ID", m.client.appID)
	}
	if m.client.environmentID != "" {
		req.Header.Set("X-Environment-ID", m.client.environmentID)
	}

	// Inject custom headers
	for k, v := range m.client.headers {
		// Don't override existing headers
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	// Execute the request
	return m.transport.RoundTrip(req)
}

// NewHTTPClientWithAuth creates a new http.Client with automatic auth injection
// This is a convenience function for creating HTTP clients with AuthSome authentication
func (c *Client) NewHTTPClientWithAuth() *http.Client {
	return &http.Client{
		Transport: c.RoundTripper(),
		Jar:       c.cookieJar,
	}
}
