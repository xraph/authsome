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
	baseURL       string
	httpClient    *http.Client
	token         string              // Session token (Bearer)
	apiKey        string              // API key (pk_/sk_/rk_)
	cookieJar     http.CookieJar      // For session cookies
	headers       map[string]string
	plugins       map[string]Plugin
	appID         string              // Current app context
	environmentID string              // Current environment context
}

// Option is a functional option for configuring the client
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

// WithToken sets the authentication token (session token)
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = token
	}
}

// WithAPIKey sets the API key for authentication
func WithAPIKey(apiKey string) Option {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

// WithCookieJar sets a cookie jar for session management
func WithCookieJar(jar http.CookieJar) Option {
	return func(c *Client) {
		c.cookieJar = jar
		if c.httpClient != nil {
			c.httpClient.Jar = jar
		}
	}
}

// WithHeaders sets custom headers
func WithHeaders(headers map[string]string) Option {
	return func(c *Client) {
		c.headers = headers
	}
}

// WithAppContext sets the app and environment context for requests
func WithAppContext(appID, envID string) Option {
	return func(c *Client) {
		c.appID = appID
		c.environmentID = envID
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

// SetAPIKey sets the API key
func (c *Client) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

// SetAppContext sets the app and environment context
func (c *Client) SetAppContext(appID, envID string) {
	c.appID = appID
	c.environmentID = envID
}

// GetAppContext returns the current app and environment IDs
func (c *Client) GetAppContext() (appID, envID string) {
	return c.appID, c.environmentID
}

// GetPlugin returns a plugin by ID
func (c *Client) GetPlugin(id string) (Plugin, bool) {
	p, ok := c.plugins[id]
	return p, ok
}

// Request makes an HTTP request - exposed for plugin use
func (c *Client) Request(ctx context.Context, method, path string, body interface{}, result interface{}, auth bool) error {
	return c.request(ctx, method, path, body, result, auth)
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
	// Set custom headers
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	// Auto-detect and set authentication
	if auth {
		// Priority 1: API key (if set)
		if c.apiKey != "" {
			req.Header.Set("Authorization", "ApiKey "+c.apiKey)
		// Priority 2: Session token
		} else if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		// Note: Cookies are automatically attached by httpClient if cookieJar is set
	}

	// Set app and environment context headers if available
	if c.appID != "" {
		req.Header.Set("X-App-ID", c.appID)
	}
	if c.environmentID != "" {
		req.Header.Set("X-Environment-ID", c.environmentID)
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

// RefreshSession Refresh access token using refresh token
func (c *Client) RefreshSession(ctx context.Context, req *RefreshSessionRequest) (*RefreshSessionResponse, error) {
	path := "/api/auth/refresh"
	var result RefreshSessionResponse
	err := c.request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RequestPasswordReset Request password reset link
func (c *Client) RequestPasswordReset(ctx context.Context, req *RequestPasswordResetRequest) (*RequestPasswordResetResponse, error) {
	path := "/api/auth/password/reset/request"
	var result RequestPasswordResetResponse
	err := c.request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ResetPassword Reset password using token
func (c *Client) ResetPassword(ctx context.Context, req *ResetPasswordRequest) (*ResetPasswordResponse, error) {
	path := "/api/auth/password/reset/confirm"
	var result ResetPasswordResponse
	err := c.request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ValidateResetToken Validate password reset token
func (c *Client) ValidateResetToken(ctx context.Context) (*ValidateResetTokenResponse, error) {
	path := "/api/auth/password/reset/validate"
	var result ValidateResetTokenResponse
	err := c.request(ctx, "GET", path, nil, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ChangePassword Change password for authenticated user
func (c *Client) ChangePassword(ctx context.Context, req *ChangePasswordRequest) (*ChangePasswordResponse, error) {
	path := "/api/auth/password/change"
	var result ChangePasswordResponse
	err := c.request(ctx, "POST", path, req, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// RequestEmailChange Request email address change
func (c *Client) RequestEmailChange(ctx context.Context, req *RequestEmailChangeRequest) (*RequestEmailChangeResponse, error) {
	path := "/api/auth/email/change/request"
	var result RequestEmailChangeResponse
	err := c.request(ctx, "POST", path, req, &result, true)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ConfirmEmailChange Confirm email change using token
func (c *Client) ConfirmEmailChange(ctx context.Context, req *ConfirmEmailChangeRequest) (*ConfirmEmailChangeResponse, error) {
	path := "/api/auth/email/change/confirm"
	var result ConfirmEmailChangeResponse
	err := c.request(ctx, "POST", path, req, &result, false)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

