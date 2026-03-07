package memory

import (
	"context"
	"sort"

	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/store"
)

// settingsKey builds a composite key for the settings map: key|scope|scope_id.
func settingsKey(key string, scope settings.Scope, scopeID string) string {
	return key + "|" + string(scope) + "|" + scopeID
}

func (s *Store) GetSetting(_ context.Context, key string, scope settings.Scope, scopeID string) (*settings.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st, ok := s.settingsMap[settingsKey(key, scope, scopeID)]
	if !ok {
		return nil, store.ErrNotFound
	}

	return st, nil
}

func (s *Store) SetSetting(_ context.Context, st *settings.Setting) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.settingsMap[settingsKey(st.Key, st.Scope, st.ScopeID)] = st
	return nil
}

func (s *Store) DeleteSetting(_ context.Context, key string, scope settings.Scope, scopeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.settingsMap, settingsKey(key, scope, scopeID))
	return nil
}

func (s *Store) ListSettings(_ context.Context, opts settings.ListOpts) ([]*settings.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*settings.Setting

	for _, st := range s.settingsMap {
		if opts.Namespace != "" && st.Namespace != opts.Namespace {
			continue
		}
		if opts.Scope != "" && st.Scope != opts.Scope {
			continue
		}
		if opts.ScopeID != "" && st.ScopeID != opts.ScopeID {
			continue
		}
		if opts.AppID != "" && st.AppID != opts.AppID {
			continue
		}
		if opts.OrgID != "" && st.OrgID != opts.OrgID {
			continue
		}
		result = append(result, st)
	}

	// Sort by created_at descending.
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	// Apply offset and limit.
	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	} else if opts.Offset >= len(result) {
		return nil, nil
	}

	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, nil
}

func (s *Store) ResolveSettings(_ context.Context, key string, opts settings.ResolveOpts) ([]*settings.Setting, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*settings.Setting

	// Global scope.
	if st, ok := s.settingsMap[settingsKey(key, settings.ScopeGlobal, "")]; ok {
		result = append(result, st)
	}

	// App scope.
	if opts.AppID != "" {
		if st, ok := s.settingsMap[settingsKey(key, settings.ScopeApp, opts.AppID)]; ok {
			result = append(result, st)
		}
	}

	// Org scope.
	if opts.OrgID != "" {
		if st, ok := s.settingsMap[settingsKey(key, settings.ScopeOrg, opts.OrgID)]; ok {
			if st.AppID == "" || st.AppID == opts.AppID {
				result = append(result, st)
			}
		}
	}

	// User scope.
	if opts.UserID != "" {
		if st, ok := s.settingsMap[settingsKey(key, settings.ScopeUser, opts.UserID)]; ok {
			if (st.AppID == "" || st.AppID == opts.AppID) && (st.OrgID == "" || st.OrgID == opts.OrgID) {
				result = append(result, st)
			}
		}
	}

	return result, nil
}

func (s *Store) BatchResolve(ctx context.Context, keys []string, opts settings.ResolveOpts) (map[string][]*settings.Setting, error) {
	result := make(map[string][]*settings.Setting, len(keys))

	for _, key := range keys {
		resolved, err := s.ResolveSettings(ctx, key, opts)
		if err != nil {
			return nil, err
		}
		if len(resolved) > 0 {
			result[key] = resolved
		}
	}

	return result, nil
}

func (s *Store) DeleteSettingsByNamespace(_ context.Context, namespace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, st := range s.settingsMap {
		if st.Namespace == namespace {
			delete(s.settingsMap, k)
		}
	}

	return nil
}
