package social

import (
	"context"
	"encoding/json"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
)

// memStore is a minimal in-memory settings.Store for tests.
type memStore struct {
	settings map[string]*settings.Setting
}

func newMemStore() *memStore {
	return &memStore{settings: make(map[string]*settings.Setting)}
}

func memKey(key string, scope settings.Scope, scopeID string) string {
	return key + "|" + string(scope) + "|" + scopeID
}

func (s *memStore) GetSetting(_ context.Context, key string, scope settings.Scope, scopeID string) (*settings.Setting, error) {
	st, ok := s.settings[memKey(key, scope, scopeID)]
	if !ok {
		return nil, settings.ErrNotFound
	}
	return st, nil
}

func (s *memStore) SetSetting(_ context.Context, st *settings.Setting) error {
	s.settings[memKey(st.Key, st.Scope, st.ScopeID)] = st
	return nil
}

func (s *memStore) DeleteSetting(_ context.Context, key string, scope settings.Scope, scopeID string) error {
	delete(s.settings, memKey(key, scope, scopeID))
	return nil
}

func (s *memStore) ListSettings(_ context.Context, _ settings.ListOpts) ([]*settings.Setting, error) {
	out := make([]*settings.Setting, 0, len(s.settings))
	for _, st := range s.settings {
		out = append(out, st)
	}
	return out, nil
}

func (s *memStore) ResolveSettings(_ context.Context, key string, opts settings.ResolveOpts) ([]*settings.Setting, error) {
	var result []*settings.Setting
	if st, ok := s.settings[memKey(key, settings.ScopeGlobal, "")]; ok {
		result = append(result, st)
	}
	if opts.AppID != "" {
		if st, ok := s.settings[memKey(key, settings.ScopeApp, opts.AppID)]; ok {
			result = append(result, st)
		}
	}
	if opts.OrgID != "" {
		if st, ok := s.settings[memKey(key, settings.ScopeOrg, opts.OrgID)]; ok {
			result = append(result, st)
		}
	}
	if opts.UserID != "" {
		if st, ok := s.settings[memKey(key, settings.ScopeUser, opts.UserID)]; ok {
			result = append(result, st)
		}
	}
	return result, nil
}

func (s *memStore) BatchResolve(ctx context.Context, keys []string, opts settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	out := make(map[string][]*settings.Setting, len(keys))
	for _, k := range keys {
		r, err := s.ResolveSettings(ctx, k, opts)
		if err != nil {
			return nil, err
		}
		if len(r) > 0 {
			out[k] = r
		}
	}
	return out, nil
}

func (s *memStore) DeleteSettingsByNamespace(_ context.Context, ns string) error {
	for k, st := range s.settings {
		if st.Namespace == ns {
			delete(s.settings, k)
		}
	}
	return nil
}

// newAllowlistManager builds a settings.Manager with the allowlist setting
// registered. If csv is non-empty it is written at ScopeApp for appID; if
// globalCSV is non-empty it is written at ScopeGlobal.
func newAllowlistManager(t *testing.T, appID id.AppID, csv, globalCSV string) *settings.Manager {
	t.Helper()
	store := newMemStore()
	mgr := settings.NewManager(store, log.NewNoopLogger())
	if err := settings.RegisterTyped(mgr, "social", SettingAllowedFrontendURLs); err != nil {
		t.Fatalf("register: %v", err)
	}
	if csv != "" {
		raw, _ := json.Marshal(csv)
		if err := mgr.Set(context.Background(), SettingAllowedFrontendURLs.Def.Key, raw,
			settings.ScopeApp, appID.String(), appID.String(), "", "test"); err != nil {
			t.Fatalf("set app csv: %v", err)
		}
	}
	if globalCSV != "" {
		raw, _ := json.Marshal(globalCSV)
		if err := mgr.Set(context.Background(), SettingAllowedFrontendURLs.Def.Key, raw,
			settings.ScopeGlobal, "", "", "", "test"); err != nil {
			t.Fatalf("set global csv: %v", err)
		}
	}
	return mgr
}

func TestIsAllowedOrigin_EmptyAllowlistRejects(t *testing.T) {
	t.Parallel()
	appID := id.NewAppID()
	mgr := newAllowlistManager(t, appID, "", "")
	for _, c := range []string{
		"https://app.example",
		"https://attacker.example",
		"http://localhost:3000",
	} {
		if isAllowedOrigin(context.Background(), mgr, appID, c) {
			t.Errorf("isAllowedOrigin(%q) = true, want false (empty allowlist)", c)
		}
	}
}

func TestIsAllowedOrigin_HostInListPasses(t *testing.T) {
	t.Parallel()
	appID := id.NewAppID()
	mgr := newAllowlistManager(t, appID, "https://app.example,https://staging.app.example", "")

	cases := []struct {
		in   string
		want bool
	}{
		{"https://app.example/path", true},
		{"https://staging.app.example", true},
		{"https://attacker.example", false},
	}
	for _, tc := range cases {
		got := isAllowedOrigin(context.Background(), mgr, appID, tc.in)
		if got != tc.want {
			t.Errorf("isAllowedOrigin(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsAllowedOrigin_CaseInsensitiveHost(t *testing.T) {
	t.Parallel()
	appID := id.NewAppID()
	mgr := newAllowlistManager(t, appID, "https://APP.example", "")
	if !isAllowedOrigin(context.Background(), mgr, appID, "https://app.example") {
		t.Error("expected case-insensitive host match")
	}
}

func TestIsAllowedOrigin_PortMatters(t *testing.T) {
	t.Parallel()
	appID := id.NewAppID()
	mgr := newAllowlistManager(t, appID, "https://app.example:443", "")
	if isAllowedOrigin(context.Background(), mgr, appID, "https://app.example:8080") {
		t.Error("expected port mismatch to reject")
	}
}

func TestIsAllowedOrigin_BadInput(t *testing.T) {
	t.Parallel()
	appID := id.NewAppID()
	mgr := newAllowlistManager(t, appID, "https://app.example", "")
	for _, c := range []string{
		"",
		"javascript:alert(1)",
		"ftp://app.example",
	} {
		if isAllowedOrigin(context.Background(), mgr, appID, c) {
			t.Errorf("isAllowedOrigin(%q) = true, want false", c)
		}
	}
}

func TestIsAllowedOrigin_WhitespaceTolerant(t *testing.T) {
	t.Parallel()
	appID := id.NewAppID()
	mgr := newAllowlistManager(t, appID, "  https://app.example  ,  https://staging.app.example  ", "")
	for _, c := range []string{
		"https://app.example",
		"https://staging.app.example",
	} {
		if !isAllowedOrigin(context.Background(), mgr, appID, c) {
			t.Errorf("isAllowedOrigin(%q) = false, want true (whitespace tolerance)", c)
		}
	}
}

func TestIsAllowedOrigin_GlobalScopeFallback(t *testing.T) {
	t.Parallel()
	appID := id.NewAppID()
	otherAppID := id.NewAppID()
	mgr := newAllowlistManager(t, appID, "", "https://global.example")

	if !isAllowedOrigin(context.Background(), mgr, appID, "https://global.example") {
		t.Error("expected global allowlist to apply for appID")
	}
	if !isAllowedOrigin(context.Background(), mgr, otherAppID, "https://global.example") {
		t.Error("expected global allowlist to apply for any appID")
	}
	if isAllowedOrigin(context.Background(), mgr, appID, "https://attacker.example") {
		t.Error("expected non-allowlisted host to be rejected")
	}
}

// TestAttack_FrontendURL_NotAllowlisted exercises the trusted-origin
// selection path that handleStart uses.
//
// Wiring a full handleStart through tests requires a forge.Context, an
// engine, providers, and a store — far too heavy for this unit test. So
// we factor the trust-authority selection into selectTrustedOrigin and
// drive it directly.
func TestAttack_FrontendURL_NotAllowlisted(t *testing.T) {
	t.Parallel()
	appID := id.NewAppID()
	mgr := newAllowlistManager(t, appID, "https://app.example", "")

	// Attacker page calls /v1/social/google with attacker-controlled
	// frontend_url and Origin/Referer. Neither is on the allowlist, so
	// selectTrustedOrigin must return "" — the same outcome as if no
	// frontend_url was supplied at all.
	got := selectTrustedOrigin(context.Background(), mgr, appID,
		"https://attacker.example",         // frontend_url
		"https://attacker.example",         // Origin header
		"https://attacker.example/landing", // Referer header
	)
	if got != "" {
		t.Errorf("selectTrustedOrigin with attacker inputs = %q, want \"\" (no allowlist match)", got)
	}

	// Sanity: an allowlisted frontend_url wins.
	got = selectTrustedOrigin(context.Background(), mgr, appID,
		"https://app.example", "", "")
	if got != "https://app.example" {
		t.Errorf("selectTrustedOrigin(allowlisted frontend_url) = %q, want \"https://app.example\"", got)
	}

	// Sanity: empty frontend_url with an allowlisted Origin header wins.
	got = selectTrustedOrigin(context.Background(), mgr, appID,
		"", "https://app.example", "")
	if got != "https://app.example" {
		t.Errorf("selectTrustedOrigin(allowlisted Origin) = %q, want \"https://app.example\"", got)
	}
}
