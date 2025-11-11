package multisession

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/xid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/xraph/authsome/schema"
	"github.com/xraph/forge"
)

// routerController is a local type alias to work around router.Controller being from internal package
// This is a workaround since we can't import github.com/xraph/forge/internal/router directly
type routerController any

// testRouter implements forge.Router for testing
type testRouter struct {
	mux  *http.ServeMux
	base string
}

func (r *testRouter) Group(basePath string, opts ...forge.GroupOption) forge.Router {
	newBase := r.base
	if !strings.HasPrefix(basePath, "/") {
		newBase += "/"
	}
	newBase += basePath
	return &testRouter{mux: r.mux, base: newBase}
}

func (r *testRouter) AsyncAPISpec() *forge.AsyncAPISpec {
	return nil // No-op for testing
}

func (r *testRouter) OpenAPISpec() *forge.OpenAPISpec {
	return nil // No-op for testing
}

func (r *testRouter) EnableWebTransport(config forge.WebTransportConfig) error {
	return nil // No-op for testing
}

func (r *testRouter) EventStream(path string, handler forge.SSEHandler, opts ...forge.RouteOption) error {
	return nil // No-op for testing
}

func (r *testRouter) Handler() http.Handler {
	return r.mux // Return the underlying mux as the handler
}

// RegisterController accepts router.Controller
// Using type alias as workaround since router.Controller is from internal package
func (r *testRouter) RegisterController(controller routerController) error {
	// No-op for testing - controller registration not needed for these tests
	_ = controller
	return nil
}

func (r *testRouter) HEAD(path string, handler any, opts ...forge.RouteOption) error {
	if h, ok := handler.(func(forge.Context) error); ok {
		r.handle("HEAD", path, h)
	}
	return nil
}

func (r *testRouter) GET(path string, handler any, opts ...forge.RouteOption) error {
	if h, ok := handler.(func(forge.Context) error); ok {
		r.handle("GET", path, h)
	}
	return nil
}

func (r *testRouter) POST(path string, handler any, opts ...forge.RouteOption) error {
	if h, ok := handler.(func(forge.Context) error); ok {
		r.handle("POST", path, h)
	}
	return nil
}

func (r *testRouter) PUT(path string, handler any, opts ...forge.RouteOption) error {
	if h, ok := handler.(func(forge.Context) error); ok {
		r.handle("PUT", path, h)
	}
	return nil
}

func (r *testRouter) PATCH(path string, handler any, opts ...forge.RouteOption) error {
	if h, ok := handler.(func(forge.Context) error); ok {
		r.handle("PATCH", path, h)
	}
	return nil
}

func (r *testRouter) DELETE(path string, handler any, opts ...forge.RouteOption) error {
	if h, ok := handler.(func(forge.Context) error); ok {
		r.handle("DELETE", path, h)
	}
	return nil
}

func (r *testRouter) OPTIONS(path string, handler any, opts ...forge.RouteOption) error {
	if h, ok := handler.(func(forge.Context) error); ok {
		r.handle("OPTIONS", path, h)
	}
	return nil
}

func (r *testRouter) handle(method, path string, h func(forge.Context) error) {
	fullPath := r.base + path
	r.mux.HandleFunc(fullPath, func(w http.ResponseWriter, req *http.Request) {
		if req.Method != method {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Create a minimal forge.Context for testing
		ctx := &testContext{w: w, r: req}
		if err := h(ctx); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		}
	})
}

// testContext implements forge.Context for testing
type testContext struct {
	w http.ResponseWriter
	r *http.Request
}

// NoContent implements shared.Context.
func (c *testContext) NoContent(code int) error {
	panic("unimplemented")
}

// ParamBool implements shared.Context.
func (c *testContext) ParamBool(name string) (bool, error) {
	panic("unimplemented")
}

// ParamBoolDefault implements shared.Context.
func (c *testContext) ParamBoolDefault(name string, defaultValue bool) bool {
	panic("unimplemented")
}

// ParamFloat64 implements shared.Context.
func (c *testContext) ParamFloat64(name string) (float64, error) {
	panic("unimplemented")
}

// ParamFloat64Default implements shared.Context.
func (c *testContext) ParamFloat64Default(name string, defaultValue float64) float64 {
	panic("unimplemented")
}

// ParamInt implements shared.Context.
func (c *testContext) ParamInt(name string) (int, error) {
	panic("unimplemented")
}

// ParamInt64 implements shared.Context.
func (c *testContext) ParamInt64(name string) (int64, error) {
	panic("unimplemented")
}

// ParamInt64Default implements shared.Context.
func (c *testContext) ParamInt64Default(name string, defaultValue int64) int64 {
	panic("unimplemented")
}

// ParamIntDefault implements shared.Context.
func (c *testContext) ParamIntDefault(name string, defaultValue int) int {
	panic("unimplemented")
}

// Params implements shared.Context.
func (c *testContext) Params() map[string]string {
	panic("unimplemented")
}

// ParseMultipartForm implements shared.Context.
func (c *testContext) ParseMultipartForm(maxMemory int64) error {
	panic("unimplemented")
}

// QueryDefault implements shared.Context.
func (c *testContext) QueryDefault(name string, defaultValue string) string {
	panic("unimplemented")
}

// Resolve implements shared.Context.
func (c *testContext) Resolve(name string) (any, error) {
	panic("unimplemented")
}

// SaveSession implements shared.Context.
func (c *testContext) SaveSession() error {
	panic("unimplemented")
}

// Scope implements shared.Context.
func (c *testContext) Scope() forge.Scope {
	panic("unimplemented")
}

// Session implements shared.Context.
func (c *testContext) Session() (forge.Session, error) {
	panic("unimplemented")
}

// SessionID implements shared.Context.
func (c *testContext) SessionID() string {
	panic("unimplemented")
}

// SetCookie implements shared.Context.
func (c *testContext) SetCookie(name string, value string, maxAge int) {
	panic("unimplemented")
}

// SetCookieWithOptions implements shared.Context.
func (c *testContext) SetCookieWithOptions(name string, value string, path string, domain string, maxAge int, secure bool, httpOnly bool) {
	panic("unimplemented")
}

// SetSession implements shared.Context.
func (c *testContext) SetSession(session forge.Session) {
	panic("unimplemented")
}

// SetSessionValue implements shared.Context.
func (c *testContext) SetSessionValue(key string, value any) {
	panic("unimplemented")
}

// Status implements shared.Context.
func (c *testContext) Status(code int) forge.ResponseBuilder {
	panic("unimplemented")
}

// WithContext implements shared.Context.
func (c *testContext) WithContext(ctx context.Context) {
	panic("unimplemented")
}

// XML implements shared.Context.
func (c *testContext) XML(code int, v any) error {
	panic("unimplemented")
}

func (c *testContext) Request() *http.Request { return c.r }
func (c *testContext) Header(key string) string {
	return c.r.Header.Get(key) // For getting header values
}
func (c *testContext) Headers() http.Header {
	return c.w.Header() // For SetHeader and other header operations
}
func (c *testContext) JSON(status int, v interface{}) error {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(status)
	return json.NewEncoder(c.w).Encode(v)
}
func (c *testContext) Param(name string) string { return "" } // Not needed for these tests
func (c *testContext) Query(key string) string  { return c.r.URL.Query().Get(key) }
func (c *testContext) SetHeader(key, value string) {
	c.w.Header().Set(key, value)
}
func (c *testContext) Response() http.ResponseWriter { return c.w }
func (c *testContext) String(status int, s string) error {
	c.w.WriteHeader(status)
	_, err := c.w.Write([]byte(s))
	return err
}
func (c *testContext) Redirect(status int, url string) error {
	http.Redirect(c.w, c.r, url, status)
	return nil
}
func (c *testContext) Cookie(name string) (string, error) {
	ck, err := c.r.Cookie(name)
	if err != nil {
		return "", err
	}
	return ck.Value, nil
}
func (c *testContext) DeleteCookie(name string) {
	http.SetCookie(c.w, &http.Cookie{
		Name:   name,
		Value:  "",
		MaxAge: -1,
	})
}
func (c *testContext) DeleteSessionValue(key string) {
	// No-op for testing - session storage not needed for these tests
}
func (c *testContext) DestroySession() error {
	// No-op for testing - session destruction not needed for these tests
	return nil
}
func (c *testContext) FormFile(name string) (multipart.File, *multipart.FileHeader, error) {
	// File upload not needed for these tests
	return nil, nil, nil
}
func (c *testContext) FormFiles(name string) ([]*multipart.FileHeader, error) {
	// Multiple file upload not needed for these tests
	return nil, nil
}
func (c *testContext) FormValue(key string) string {
	return c.r.FormValue(key)
}
func (c *testContext) FormValues(key string) []string {
	return c.r.Form[key] // Return form values for the key
}
func (c *testContext) GetAllCookies() map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range c.r.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	return cookies
}
func (c *testContext) GetSessionValue(key string) (any, bool) {
	// No-op for testing - session storage not needed for these tests
	return nil, false
}
func (c *testContext) HasCookie(name string) bool {
	_, err := c.r.Cookie(name)
	return err == nil
}
func (c *testContext) Must(key string) any {
	// No-op for testing - Must is typically used for required values
	return nil
}
func (c *testContext) MustGet(key string) any {
	// No-op for testing - MustGet is typically used for required values
	return nil
}
func (c *testContext) HTML(status int, html string) error {
	c.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.w.WriteHeader(status)
	_, err := c.w.Write([]byte(html))
	return err
}
func (c *testContext) Get(key string) interface{}        { return nil }
func (c *testContext) Set(key string, value interface{}) {}
func (c *testContext) SetRequest(r *http.Request)        { c.r = r }
func (c *testContext) Bind(v interface{}) error {
	return json.NewDecoder(c.r.Body).Decode(v)
}
func (c *testContext) BindJSON(v interface{}) error {
	return json.NewDecoder(c.r.Body).Decode(v)
}
func (c *testContext) BindXML(v interface{}) error {
	// XML binding not needed for these tests
	return nil
}
func (c *testContext) Bytes(status int, data []byte) error {
	c.w.Header().Set("Content-Type", "application/octet-stream")
	c.w.WriteHeader(status)
	_, err := c.w.Write(data)
	return err
}
func (c *testContext) Container() forge.Container {
	return nil // No-op for testing
}
func (c *testContext) Context() context.Context {
	return c.r.Context() // Return the request context
}

// setupTestApp initializes in-memory Bun DB, creates necessary tables, and mounts multisession routes
func setupTestAppMS(t *testing.T) (*bun.DB, *http.ServeMux, *Plugin) {
	t.Helper()
	sqldb, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db := bun.NewDB(sqldb, sqlitedialect.New())

	ctx := context.Background()
	if _, err := db.NewCreateTable().Model((*schema.User)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create users: %v", err)
	}
	if _, err := db.NewCreateTable().Model((*schema.Session)(nil)).IfNotExists().Exec(ctx); err != nil {
		t.Fatalf("create sessions: %v", err)
	}

	// Initialize plugin
	p := NewPlugin()
	if err := p.Init(db); err != nil {
		t.Fatalf("plugin init: %v", err)
	}
	if err := p.Migrate(); err != nil {
		t.Fatalf("plugin migrate: %v", err)
	}

	mux := http.NewServeMux()
	router := &testRouter{mux: mux, base: "/api/auth"}
	if err := p.RegisterRoutes(router); err != nil {
		t.Fatalf("register routes: %v", err)
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	return db, mux, p
}

func TestMultiSession_ListSetActiveDelete(t *testing.T) {
	db, mux, _ := setupTestAppMS(t)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Create user and sessions directly in database
	ctx := context.Background()

	// Create user
	systemID := xid.New() // System user ID for audit fields
	user := &schema.User{
		ID:           xid.New(),
		Email:        "ms.user@example.com",
		Name:         "MS User",
		PasswordHash: "$2a$10$example.hash", // dummy hash
	}
	user.CreatedBy = systemID
	user.UpdatedBy = systemID
	if _, err := db.NewInsert().Model(user).Exec(ctx); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create two sessions
	sess1 := &schema.Session{
		ID:        xid.New(),
		UserID:    user.ID,
		Token:     "test-token-1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "ua-1",
	}
	sess1.CreatedBy = systemID
	sess1.UpdatedBy = systemID
	sess2 := &schema.Session{
		ID:        xid.New(),
		UserID:    user.ID,
		Token:     "test-token-2",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "ua-2",
	}
	sess2.CreatedBy = systemID
	sess2.UpdatedBy = systemID

	if _, err := db.NewInsert().Model(sess1).Exec(ctx); err != nil {
		t.Fatalf("create session1: %v", err)
	}
	if _, err := db.NewInsert().Model(sess2).Exec(ctx); err != nil {
		t.Fatalf("create session2: %v", err)
	}

	// List sessions for user using cookie from first session
	reqList, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/multi-session/list", nil)
	reqList.AddCookie(&http.Cookie{Name: "session_token", Value: sess1.Token})
	respList, err := http.DefaultClient.Do(reqList)
	if err != nil {
		t.Fatalf("list request error: %v", err)
	}
	if respList.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", respList.StatusCode)
	}
	var listOut map[string]any
	if err := json.NewDecoder(respList.Body).Decode(&listOut); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	sessionsVal, ok := listOut["sessions"].([]any)
	if !ok || len(sessionsVal) != 2 {
		t.Fatalf("expected 2 sessions, got %v", len(sessionsVal))
	}

	// Set active to second session
	body := map[string]any{"id": sess2.ID.String()}
	buf, _ := json.Marshal(body)
	reqSet, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/multi-session/set-active", bytes.NewReader(buf))
	reqSet.Header.Set("Content-Type", "application/json")
	reqSet.AddCookie(&http.Cookie{Name: "session_token", Value: sess1.Token})
	respSet, err := http.DefaultClient.Do(reqSet)
	if err != nil {
		t.Fatalf("set-active request error: %v", err)
	}
	if respSet.StatusCode != 200 {
		t.Fatalf("expected 200 set-active, got %d", respSet.StatusCode)
	}
	// Expect Set-Cookie header to include new token
	sc := respSet.Header.Get("Set-Cookie")
	if sc == "" || !containsCookieWithToken(sc, sess2.Token) {
		t.Fatalf("expected Set-Cookie with new token, got %q", sc)
	}
	var setOut map[string]any
	if err := json.NewDecoder(respSet.Body).Decode(&setOut); err != nil {
		t.Fatalf("decode set-active: %v", err)
	}
	if tok, ok := setOut["token"].(string); !ok || tok != sess2.Token {
		t.Fatalf("expected token %s, got %v", sess2.Token, tok)
	}

	// Delete first session via path param
	reqDel, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/auth/multi-session/delete/"+sess1.ID.String(), nil)
	reqDel.AddCookie(&http.Cookie{Name: "session_token", Value: sess2.Token})
	respDel, err := http.DefaultClient.Do(reqDel)
	if err != nil {
		t.Fatalf("delete request error: %v", err)
	}
	if respDel.StatusCode != 200 {
		t.Fatalf("expected 200 delete, got %d", respDel.StatusCode)
	}

	// List again should show 1 session
	reqList2, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/auth/multi-session/list", nil)
	reqList2.AddCookie(&http.Cookie{Name: "session_token", Value: sess2.Token})
	respList2, err := http.DefaultClient.Do(reqList2)
	if err != nil {
		t.Fatalf("list2 request error: %v", err)
	}
	var listOut2 map[string]any
	if err := json.NewDecoder(respList2.Body).Decode(&listOut2); err != nil {
		t.Fatalf("decode list2: %v", err)
	}
	sessionsVal2, ok := listOut2["sessions"].([]any)
	if !ok || len(sessionsVal2) != 1 {
		t.Fatalf("expected 1 session after delete, got %v", len(sessionsVal2))
	}
}

// containsCookieWithToken checks if Set-Cookie header contains session_token=<token>
func containsCookieWithToken(sc, token string) bool {
	want := "session_token=" + token
	return sc != "" && (sc == want || (len(sc) > len(want) && sc[:len(want)] == want))
}
