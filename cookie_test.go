package authsome_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/xraph/go-utils/log"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/settings"
)

// cookieMemStore is a minimal in-memory settings.Store for tests.
type cookieMemStore struct {
	mu       sync.Mutex
	settings map[string]*settings.Setting
}

func newCookieMemStore() *cookieMemStore {
	return &cookieMemStore{settings: make(map[string]*settings.Setting)}
}

func cookieMemKey(key string, scope settings.Scope, scopeID string) string {
	return key + "|" + string(scope) + "|" + scopeID
}

func (s *cookieMemStore) GetSetting(_ context.Context, key string, scope settings.Scope, scopeID string) (*settings.Setting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.settings[cookieMemKey(key, scope, scopeID)]
	if !ok {
		return nil, settings.ErrNotFound
	}
	return st, nil
}

func (s *cookieMemStore) SetSetting(_ context.Context, st *settings.Setting) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings[cookieMemKey(st.Key, st.Scope, st.ScopeID)] = st
	return nil
}

func (s *cookieMemStore) DeleteSetting(_ context.Context, key string, scope settings.Scope, scopeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.settings, cookieMemKey(key, scope, scopeID))
	return nil
}

func (s *cookieMemStore) ListSettings(_ context.Context, _ settings.ListOpts) ([]*settings.Setting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*settings.Setting, 0, len(s.settings))
	for _, st := range s.settings {
		out = append(out, st)
	}
	return out, nil
}

func (s *cookieMemStore) ResolveSettings(_ context.Context, key string, opts settings.ResolveOpts) ([]*settings.Setting, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []*settings.Setting
	if st, ok := s.settings[cookieMemKey(key, settings.ScopeGlobal, "")]; ok {
		result = append(result, st)
	}
	if opts.AppID != "" {
		if st, ok := s.settings[cookieMemKey(key, settings.ScopeApp, opts.AppID)]; ok {
			result = append(result, st)
		}
	}
	return result, nil
}

func (s *cookieMemStore) BatchResolve(ctx context.Context, keys []string, opts settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	out := make(map[string][]*settings.Setting, len(keys))
	for _, k := range keys {
		got, err := s.ResolveSettings(ctx, k, opts)
		if err != nil {
			return nil, err
		}
		out[k] = got
	}
	return out, nil
}

func (s *cookieMemStore) DeleteSettingsByNamespace(_ context.Context, _ string) error {
	return nil
}

// newCookieTestManager builds a settings.Manager pre-registered with the
// cookie settings.
func newCookieTestManager(t *testing.T) *settings.Manager {
	t.Helper()
	mgr := settings.NewManager(newCookieMemStore(), log.NewNoopLogger())
	require.NoError(t, settings.RegisterTyped(mgr, "session", authsome.SettingCookieName))
	require.NoError(t, settings.RegisterTyped(mgr, "session", authsome.SettingCookieDomain))
	require.NoError(t, settings.RegisterTyped(mgr, "session", authsome.SettingCookiePath))
	require.NoError(t, settings.RegisterTyped(mgr, "session", authsome.SettingCookieSecure))
	require.NoError(t, settings.RegisterTyped(mgr, "session", authsome.SettingCookieHTTPOnly))
	require.NoError(t, settings.RegisterTyped(mgr, "session", authsome.SettingCookieSameSite))
	require.NoError(t, settings.RegisterTyped(mgr, "session", authsome.SettingCookieUseHostPrefix))
	return mgr
}

// override sets a global override for a setting key in tests.
func override(t *testing.T, mgr *settings.Manager, key string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, mgr.Set(context.Background(), key, raw, settings.ScopeGlobal, "", "", "", "test"))
}

func TestSessionCookieTemplate_DefaultsArePreservedForBackCompat(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", true)

	assert.Equal(t, "authsome_session_token", c.Name, "default cookie name")
	assert.Equal(t, "/", c.Path)
	assert.Equal(t, "", c.Domain, "no domain by default")
	assert.True(t, c.Secure, "secure cookie when isHTTPS=true and SettingCookieSecure default true")
	assert.True(t, c.HttpOnly, "HttpOnly default true")
	assert.Equal(t, http.SameSiteLaxMode, c.SameSite, "SameSite default Lax")
	assert.False(t, strings.HasPrefix(c.Name, "__Host-"), "no __Host- prefix when opt-in is off")
}

func TestSessionCookieTemplate_HostPrefixIsApplied(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	override(t, mgr, "session.cookie_use_host_prefix", true)

	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", false)

	assert.Equal(t, "__Host-authsome_session_token", c.Name,
		"name must be __Host-prefixed when opt-in is true")
}

func TestSessionCookieTemplate_HostPrefixForcesSecure(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	override(t, mgr, "session.cookie_use_host_prefix", true)
	override(t, mgr, "session.cookie_secure", false)

	// isHTTPS=false would normally yield secure=false.
	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", false)
	assert.True(t, c.Secure,
		"__Host- prefix MUST force Secure=true even when SettingCookieSecure=false and isHTTPS=false (browser drops the cookie otherwise)")
}

func TestSessionCookieTemplate_HostPrefixForcesEmptyDomain(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	override(t, mgr, "session.cookie_use_host_prefix", true)
	override(t, mgr, "session.cookie_domain", "example.com")

	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", true)
	assert.Equal(t, "", c.Domain,
		"__Host- prefix MUST clear Domain even when set; browser rejects __Host- cookies with a Domain attribute")
}

func TestSessionCookieTemplate_HostPrefixForcesPathRoot(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	override(t, mgr, "session.cookie_use_host_prefix", true)
	override(t, mgr, "session.cookie_path", "/api")

	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", true)
	assert.Equal(t, "/", c.Path,
		"__Host- prefix MUST force Path=/ even when set; browser rejects otherwise")
}

func TestSessionCookieTemplate_HostPrefixIdempotent(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	override(t, mgr, "session.cookie_use_host_prefix", true)
	override(t, mgr, "session.cookie_name", "__Host-already_prefixed")

	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", true)
	assert.Equal(t, "__Host-already_prefixed", c.Name,
		"prefix must not be applied twice when the configured name already starts with __Host-")
}

func TestSessionCookieTemplate_SameSiteStrict(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	override(t, mgr, "session.cookie_same_site", "strict")

	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", true)
	assert.Equal(t, http.SameSiteStrictMode, c.SameSite)
}

func TestSessionCookieTemplate_SameSiteNone(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	override(t, mgr, "session.cookie_same_site", "none")

	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", true)
	assert.Equal(t, http.SameSiteNoneMode, c.SameSite)
}

func TestSessionCookieTemplate_SecureRequiresHTTPS(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	// Default SettingCookieSecure is true; but isHTTPS=false should still
	// yield Secure=false (so dev HTTP doesn't break).
	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", false)
	assert.False(t, c.Secure, "secure must require BOTH the setting AND isHTTPS — dev HTTP must yield Secure=false")
}

func TestSessionCookieTemplate_AppScopedOverride(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	// Global default name. App override.
	raw, err := json.Marshal("custom_app_cookie")
	require.NoError(t, err)
	appID := "aapp_01jf0000000000000000000000"
	require.NoError(t, mgr.Set(context.Background(), "session.cookie_name", raw, settings.ScopeApp, appID, appID, "", "test"))

	c := authsome.SessionCookieTemplate(context.Background(), mgr, appID, true)
	assert.Equal(t, "custom_app_cookie", c.Name, "app-scoped override must take precedence")
}

func TestSessionCookieTemplate_GlobalOnlyWhenAppEmpty(t *testing.T) {
	t.Parallel()
	mgr := newCookieTestManager(t)
	override(t, mgr, "session.cookie_name", "global_cookie")

	c := authsome.SessionCookieTemplate(context.Background(), mgr, "", true)
	assert.Equal(t, "global_cookie", c.Name)
}
