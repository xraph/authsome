package authclient

import (
	"context"
	"encoding/json"
	"fmt"
)

// Introspect validates a token via the authsome introspection endpoint.
// Returns the token's identity claims if active, or {Active: false} if the
// token is invalid or expired. This follows RFC 7662 semantics.
func (c *Client) Introspect(ctx context.Context, token string) (*IntrospectResponse, error) {
	body, err := json.Marshal(map[string]string{"token": token})
	if err != nil {
		return nil, fmt.Errorf("marshal introspect request: %w", err)
	}

	var result IntrospectResponse
	if err := c.do(ctx, "POST", "/v1/introspect", body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// BaseURL returns the client's base URL. Useful for creating request-scoped
// clients that inherit the same server address.
func (c *Client) BaseURL() string {
	return c.baseURL
}
