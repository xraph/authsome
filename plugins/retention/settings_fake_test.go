package retention

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/settings"
)

// fakeSettingsStore is a minimal settings.Store for driving newConsentPolicy
// against a real *settings.Manager. settings.Manager.Resolve only ever calls
// ResolveSettings, so that is the only method with real behaviour here; the
// rest are unused stubs satisfying the interface.
type fakeSettingsStore struct {
	// perApp holds one stored app-scoped override per (key, appID). Nothing
	// in this task exercises a global-scope override, only per-app, so that
	// is the only scope this fake supports.
	perApp map[string]map[string]*settings.Setting

	// err, when set, is returned by every ResolveSettings call, simulating a
	// settings-store outage.
	err error
}

var _ settings.Store = (*fakeSettingsStore)(nil)

func newFakeSettingsStore() *fakeSettingsStore {
	return &fakeSettingsStore{perApp: make(map[string]map[string]*settings.Setting)}
}

// set stores an app-scoped override for key, JSON-encoding value the same
// way settings.Get expects to unmarshal it.
func (s *fakeSettingsStore) set(t *testing.T, key, appID string, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	if s.perApp[key] == nil {
		s.perApp[key] = make(map[string]*settings.Setting)
	}
	s.perApp[key][appID] = &settings.Setting{
		Key: key, Value: raw, Scope: settings.ScopeApp, ScopeID: appID, AppID: appID,
	}
}

func (s *fakeSettingsStore) ResolveSettings(_ context.Context, key string, opts settings.ResolveOpts) ([]*settings.Setting, error) {
	if s.err != nil {
		return nil, s.err
	}
	if byApp, ok := s.perApp[key]; ok {
		if st, ok := byApp[opts.AppID]; ok {
			return []*settings.Setting{st}, nil
		}
	}
	return nil, nil
}

func (s *fakeSettingsStore) GetSetting(context.Context, string, settings.Scope, string) (*settings.Setting, error) {
	return nil, settings.ErrNotFound
}

func (s *fakeSettingsStore) SetSetting(context.Context, *settings.Setting) error { return nil }

func (s *fakeSettingsStore) DeleteSetting(context.Context, string, settings.Scope, string) error {
	return nil
}

func (s *fakeSettingsStore) ListSettings(context.Context, settings.ListOpts) ([]*settings.Setting, error) {
	return nil, nil
}

func (s *fakeSettingsStore) BatchResolve(context.Context, []string, settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	return nil, nil
}

func (s *fakeSettingsStore) DeleteSettingsByNamespace(context.Context, string) error { return nil }
