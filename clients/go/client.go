package authsome

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Auto-generated AuthSome client

// Client is the main AuthSome client
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	headers    map[string]string
	plugins    map[string]Plugin
}

// Option is a functional option for configuring the client
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithToken sets the authentication token
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithHeaders sets custom headers
func WithHeaders(headers map[string]string) Option {
	return func(c *Client) {
		c.headers = headers
	}
}

// WithPlugins adds plugins to the client
func WithPlugins(plugins ...Plugin) Option {
	return func(c *Client) {
		for _, p := range plugins {
			c.plugins[p.ID()] = p
			p.Init(c)
		}
	}
}

// NewClient creates a new AuthSome client
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
		headers:    make(map[string]string),
		plugins:    make(map[string]Plugin),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// SetToken sets the authentication token
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetPlugin returns a plugin by ID
func (c *Client) GetPlugin(id string) (Plugin, bool) {
	p, ok := c.plugins[id]
	return p, ok
}

func (c *Client) request(ctx context.Context, method, path string, body interface{}, result interface{}, auth bool) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	if auth && c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		message := errResp.Error
		if message == "" {
			message = errResp.Message
		}
		if message == "" {
			message = resp.Status
		}
		return NewError(resp.StatusCode, message)
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// SignUpRequest is the request for SignUp
type SignUpRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
	Name *string `json:"name,omitempty"`
}

// SignUpResponse is the response for SignUp
type SignUpResponse struct {
	User User `json:"user"`
	Session Session `json:"session"`
}

// SignUp Create a new user account
func (c *Client) SignUp(ctx context.Context, req *SignUpRequest) (*SignUpResponse, error) {
	path := "/api/auth/signup"
	var result SignUpResponse
	err := c.request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SignInRequest is the request for SignIn
type SignInRequest struct {
	Password string `json:"password"`
	Email string `json:"email"`
}

// SignInResponse is the response for SignIn
type SignInResponse struct {
	User User `json:"user"`
	Session Session `json:"session"`
	RequiresTwoFactor bool `json:"requiresTwoFactor"`
}

// SignIn Sign in with email and password
func (c *Client) SignIn(ctx context.Context, req *SignInRequest) (*SignInResponse, error) {
	path := "/api/auth/signin"
	var result SignInResponse
	err := c.request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// SignOutResponse is the response for SignOut
type SignOutResponse struct {
	Success bool `json:"success"`
}

// SignOut Sign out and invalidate session
func (c *Client) SignOut(ctx context.Context) (*SignOutResponse, error) {
	path := "/api/auth/signout"
	var result SignOutResponse
	err := c.request(ctx, "POST", path, nil, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSessionResponse is the response for GetSession
type GetSessionResponse struct {
	User User `json:"user"`
	Session Session `json:"session"`
}

// GetSession Get current session information
func (c *Client) GetSession(ctx context.Context) (*GetSessionResponse, error) {
	path := "/api/auth/session"
	var result GetSessionResponse
	err := c.request(ctx, "GET", path, nil, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateUserRequest is the request for UpdateUser
type UpdateUserRequest struct {
	Email *string `json:"email,omitempty"`
	Name *string `json:"name,omitempty"`
}

// UpdateUserResponse is the response for UpdateUser
type UpdateUserResponse struct {
	User User `json:"user"`
}

// UpdateUser Update current user profile
func (c *Client) UpdateUser(ctx context.Context, req *UpdateUserRequest) (*UpdateUserResponse, error) {
	path := "/api/auth/user/update"
	var result UpdateUserResponse
	err := c.request(ctx, "POST", path, req, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListDevicesResponse is the response for ListDevices
type ListDevicesResponse struct {
	Devices []*Device `json:"devices"`
}

// ListDevices List user devices
func (c *Client) ListDevices(ctx context.Context) (*ListDevicesResponse, error) {
	path := "/api/auth/devices"
	var result ListDevicesResponse
	err := c.request(ctx, "GET", path, nil, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RevokeDeviceRequest is the request for RevokeDevice
type RevokeDeviceRequest struct {
	DeviceId string `json:"deviceId"`
}

// RevokeDeviceResponse is the response for RevokeDevice
type RevokeDeviceResponse struct {
	Success bool `json:"success"`
}

// RevokeDevice Revoke a device
func (c *Client) RevokeDevice(ctx context.Context, req *RevokeDeviceRequest) (*RevokeDeviceResponse, error) {
	path := "/api/auth/devices/revoke"
	var result RevokeDeviceResponse
	err := c.request(ctx, "POST", path, req, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

