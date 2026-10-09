package sso

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/environment"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

// The admin connection handlers must never read, change or create a
// connection outside the caller's own app. The tests mount the handlers on a
// bare router and stamp a session for callerApp onto every request, which is
// what the auth middleware does in production once the guard has passed.

type adminScopeFixture struct {
	p         *Plugin
	h         http.Handler
	callerApp id.AppID
	otherApp  id.AppID
}

func newAdminScopeFixture(t *testing.T) *adminScopeFixture {
	t.Helper()
	callerApp := id.NewAppID()
	otherApp := id.NewAppID()

	core := memory.New()
	require.NoError(t, core.CreateEnvironment(context.Background(), &environment.Environment{
		ID: id.NewEnvironmentID(), AppID: callerApp, Name: "Development", Slug: "development",
		Type: environment.TypeDevelopment, IsDefault: true,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	p := New()
	p.store = core
	p.ssoStore = NewMemoryStore()
	p.logger = log.NewNoopLogger()
	p.engine = stubEngineWithPlugin{}

	r := forge.NewRouter()
	g := r.Group("/t")
	require.NoError(t, g.POST("/connections", p.handleAdminCreateConnection))
	require.NoError(t, g.GET("/connections", p.handleAdminListConnections))
	require.NoError(t, g.GET("/connections/:connectionId", p.handleAdminGetConnection))
	require.NoError(t, g.PUT("/connections/:connectionId", p.handleAdminUpdateConnection))
	require.NoError(t, g.DELETE("/connections/:connectionId", p.handleAdminDeleteConnection))

	inner := r.Handler()
	sess := &session.Session{ID: id.NewSessionID(), AppID: callerApp, UserID: id.NewUserID()}
	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		inner.ServeHTTP(w, req.WithContext(middleware.WithSession(req.Context(), sess)))
	})
	return &adminScopeFixture{p: p, h: h, callerApp: callerApp, otherApp: otherApp}
}

func (f *adminScopeFixture) seed(t *testing.T, appID id.AppID, domain string) *Connection {
	t.Helper()
	conn := &Connection{
		ID: id.NewSSOConnectionID(), AppID: appID, EnvID: id.NewEnvironmentID().String(),
		Provider: "okta", Protocol: "oidc", Domain: domain, Issuer: "https://idp.example",
		ClientID: "cid", Active: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, f.p.ssoStore.CreateConnection(context.Background(), conn))
	return conn
}

func (f *adminScopeFixture) do(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(raw)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

func TestAdminCreateConnection_RefusesAnotherApp(t *testing.T) {
	f := newAdminScopeFixture(t)
	rec := f.do(t, http.MethodPost, "/t/connections", map[string]any{
		"app_id": f.otherApp.String(), "provider": "okta", "protocol": "oidc",
		"domain": "other.example", "issuer": "https://idp.example", "client_id": "cid",
	})
	assert.Equal(t, http.StatusForbidden, rec.Code, "body=%s", rec.Body.String())
	conns, err := f.p.ssoStore.ListConnections(context.Background(), f.otherApp)
	require.NoError(t, err)
	assert.Empty(t, conns, "nothing may be written into another app")
}

func TestAdminCreateConnection_LandsInCallerApp(t *testing.T) {
	f := newAdminScopeFixture(t)
	rec := f.do(t, http.MethodPost, "/t/connections", map[string]any{
		"provider": "okta", "protocol": "oidc", "domain": "mine.example",
		"issuer": "https://idp.example", "client_id": "cid",
	})
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var resp AdminCreateConnectionResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, f.callerApp.String(), resp.AppID, "the caller's app is the only target")
}

func TestAdminGetConnection_OtherAppIs404(t *testing.T) {
	f := newAdminScopeFixture(t)
	mine := f.seed(t, f.callerApp, "mine.example")
	theirs := f.seed(t, f.otherApp, "other.example")

	assert.Equal(t, http.StatusOK, f.do(t, http.MethodGet, "/t/connections/"+mine.ID.String(), nil).Code)
	assert.Equal(t, http.StatusNotFound, f.do(t, http.MethodGet, "/t/connections/"+theirs.ID.String(), nil).Code,
		"another app's connection must look absent")
}

func TestAdminUpdateConnection_OtherAppIs404(t *testing.T) {
	f := newAdminScopeFixture(t)
	theirs := f.seed(t, f.otherApp, "other.example")

	rec := f.do(t, http.MethodPut, "/t/connections/"+theirs.ID.String(), map[string]any{"domain": "stolen.example"})
	assert.Equal(t, http.StatusNotFound, rec.Code, "body=%s", rec.Body.String())

	got, err := f.p.ssoStore.GetConnection(context.Background(), theirs.ID)
	require.NoError(t, err)
	assert.Equal(t, "other.example", got.Domain, "the row must be untouched")
}

func TestAdminDeleteConnection_OtherAppIs404(t *testing.T) {
	f := newAdminScopeFixture(t)
	theirs := f.seed(t, f.otherApp, "other.example")

	assert.Equal(t, http.StatusNotFound, f.do(t, http.MethodDelete, "/t/connections/"+theirs.ID.String(), nil).Code)
	_, err := f.p.ssoStore.GetConnection(context.Background(), theirs.ID)
	assert.NoError(t, err, "the row must survive")
}

func TestAdminListConnections_ScopedToCaller(t *testing.T) {
	f := newAdminScopeFixture(t)
	f.seed(t, f.callerApp, "mine.example")
	f.seed(t, f.otherApp, "other.example")

	assert.Equal(t, http.StatusForbidden, f.do(t, http.MethodGet, "/t/connections?app_id="+f.otherApp.String(), nil).Code,
		"listing another app is refused")

	rec := f.do(t, http.MethodGet, "/t/connections", nil)
	require.Equal(t, http.StatusOK, rec.Code, "body=%s", rec.Body.String())
	var resp AdminListConnectionsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Connections, 1)
	assert.Equal(t, f.callerApp, resp.Connections[0].AppID)
}
