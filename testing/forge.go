package testing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/xraph/authsome/core/contexts"
	"github.com/xraph/authsome/schema"
)

// MockForgeContext is a mock implementation of forge.Context for testing.
type MockForgeContext struct {
	request  *http.Request
	response http.ResponseWriter
	params   map[string]string
	status   int
	body     any
}

// NewMockForgeContext creates a basic mock forge context for testing.
func (m *Mock) NewMockForgeContext(req *http.Request) *MockForgeContext {
	return &MockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}
}

// MockAuthenticatedForgeContext creates an authenticated mock Forge context
// This is useful for testing HTTP handlers that require authentication.
func (m *Mock) MockAuthenticatedForgeContext(
	req *http.Request,
	user *schema.User,
	app *schema.App,
	env *schema.Environment,
	org *schema.Organization,
	session *schema.Session,
) *MockForgeContext {
	// Set context values using core/contexts
	reqCtx := req.Context()
	reqCtx = contexts.SetAppID(reqCtx, app.ID)
	reqCtx = contexts.SetEnvironmentID(reqCtx, env.ID)
	reqCtx = contexts.SetOrganizationID(reqCtx, org.ID)
	reqCtx = contexts.SetUserID(reqCtx, user.ID)
	reqCtx = context.WithValue(reqCtx, sessionContextKey, session)

	// Update request with new context
	*req = *req.WithContext(reqCtx)

	return &MockForgeContext{
		request:  req,
		response: httptest.NewRecorder(),
		params:   make(map[string]string),
	}
}

// QuickAuthenticatedForgeContext creates authenticated Forge context with defaults
// This is the simplest way to create an authenticated context for testing.
func (m *Mock) QuickAuthenticatedForgeContext(method, path string) *MockForgeContext {
	user := m.CreateUser("test@example.com", "Test User")
	session := m.CreateSession(user.ID, m.defaultOrg.ID)

	req := httptest.NewRequest(method, path, nil)

	return m.MockAuthenticatedForgeContext(req, user, m.defaultApp, m.defaultEnv, m.defaultOrg, session)
}

// QuickAuthenticatedForgeContextWithUser creates authenticated Forge context for specific user.
func (m *Mock) QuickAuthenticatedForgeContextWithUser(method, path string, user *schema.User) *MockForgeContext {
	session := m.CreateSession(user.ID, m.defaultOrg.ID)

	req := httptest.NewRequest(method, path, nil)

	return m.MockAuthenticatedForgeContext(req, user, m.defaultApp, m.defaultEnv, m.defaultOrg, session)
}

// QuickForgeContext creates a basic unauthenticated Forge context.
func (m *Mock) QuickForgeContext(method, path string) *MockForgeContext {
	req := httptest.NewRequest(method, path, nil)

	return m.NewMockForgeContext(req)
}

// MockForgeContext methods to implement forge.Context interface

func (c *MockForgeContext) Request() *http.Request {
	return c.request
}

func (c *MockForgeContext) Response() http.ResponseWriter {
	return c.response
}

func (c *MockForgeContext) Param(name string) string {
	return c.params[name]
}

func (c *MockForgeContext) SetParam(name, value string) {
	c.params[name] = value
}

func (c *MockForgeContext) JSON(status int, data any) error {
	c.status = status
	c.body = data

	return nil
}

func (c *MockForgeContext) GetStatus() int {
	return c.status
}

func (c *MockForgeContext) GetBody() any {
	return c.body
}

func (c *MockForgeContext) String(status int, s string) error {
	c.status = status
	c.body = s
	_, err := c.response.Write([]byte(s))

	return err
}

func (c *MockForgeContext) HTML(status int, html string) error {
	c.response.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.status = status
	c.body = html
	_, err := c.response.Write([]byte(html))

	return err
}

func (c *MockForgeContext) SetHeader(key, value string) {
	c.response.Header().Set(key, value)
}

func (c *MockForgeContext) Header() http.Header {
	return c.response.Header()
}

func (c *MockForgeContext) Query(key string) string {
	return c.request.URL.Query().Get(key)
}

func (c *MockForgeContext) BindJSON(v any) error {
	return json.NewDecoder(c.request.Body).Decode(v)
}

func (c *MockForgeContext) Cookie(name string) (string, error) {
	ck, err := c.request.Cookie(name)
	if err != nil {
		return "", err
	}

	return ck.Value, nil
}

func (c *MockForgeContext) Redirect(status int, url string) error {
	http.Redirect(c.response, c.request, url, status)

	return nil
}
