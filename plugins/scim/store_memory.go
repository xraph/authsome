package scim

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/xraph/authsome/id"
)

// MemoryStore is a thread-safe in-memory implementation of the SCIM Store.
type MemoryStore struct {
	mu      sync.RWMutex
	configs map[string]*SCIMConfig
	tokens  map[string]*Token
	logs    map[string]*ProvisionLog
}

// NewMemoryStore creates a new in-memory SCIM store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		configs: make(map[string]*SCIMConfig),
		tokens:  make(map[string]*Token),
		logs:    make(map[string]*ProvisionLog),
	}
}

// ──────────────────────────────────────────────────
// Config CRUD
// ──────────────────────────────────────────────────

func (s *MemoryStore) CreateConfig(_ context.Context, c *SCIMConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := c.ID.String()
	if _, exists := s.configs[key]; exists {
		return fmt.Errorf("scim: config %s already exists", key)
	}
	cp := *c
	s.configs[key] = &cp
	return nil
}

func (s *MemoryStore) GetConfig(_ context.Context, configID id.SCIMConfigID) (*SCIMConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.configs[configID.String()]
	if !ok {
		return nil, fmt.Errorf("scim: config %s not found", configID)
	}
	cp := *c
	return &cp, nil
}

func (s *MemoryStore) UpdateConfig(_ context.Context, c *SCIMConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := c.ID.String()
	if _, exists := s.configs[key]; !exists {
		return fmt.Errorf("scim: config %s not found", key)
	}
	cp := *c
	s.configs[key] = &cp
	return nil
}

func (s *MemoryStore) DeleteConfig(_ context.Context, configID id.SCIMConfigID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.configs, configID.String())
	return nil
}

func (s *MemoryStore) ListConfigs(_ context.Context, appID string) ([]*SCIMConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*SCIMConfig
	for _, c := range s.configs {
		if c.AppID.String() == appID {
			cp := *c
			result = append(result, &cp)
		}
	}
	sortConfigsByCreated(result)
	return result, nil
}

func (s *MemoryStore) ListConfigsByOrg(_ context.Context, orgID id.OrgID) ([]*SCIMConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*SCIMConfig
	for _, c := range s.configs {
		if c.OrgID.String() == orgID.String() {
			cp := *c
			result = append(result, &cp)
		}
	}
	sortConfigsByCreated(result)
	return result, nil
}

// ──────────────────────────────────────────────────
// Token CRUD
// ──────────────────────────────────────────────────

func (s *MemoryStore) CreateToken(_ context.Context, t *Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := t.ID.String()
	if _, exists := s.tokens[key]; exists {
		return fmt.Errorf("scim: token %s already exists", key)
	}
	cp := *t
	s.tokens[key] = &cp
	return nil
}

func (s *MemoryStore) GetToken(_ context.Context, tokenID id.SCIMTokenID) (*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tokens[tokenID.String()]
	if !ok {
		return nil, fmt.Errorf("scim: token %s not found", tokenID)
	}
	cp := *t
	return &cp, nil
}

func (s *MemoryStore) ListTokens(_ context.Context, configID id.SCIMConfigID) ([]*Token, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Token
	cfgKey := configID.String()
	for _, t := range s.tokens {
		if t.ConfigID.String() == cfgKey {
			cp := *t
			result = append(result, &cp)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result, nil
}

func (s *MemoryStore) DeleteToken(_ context.Context, tokenID id.SCIMTokenID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, tokenID.String())
	return nil
}

func (s *MemoryStore) FindTokenByHash(_ context.Context, tokenHash string) (*Token, *SCIMConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tokens {
		if t.TokenHash != tokenHash {
			continue
		}
		cp := *t
		cfg, ok := s.configs[t.ConfigID.String()]
		if !ok {
			return nil, nil, fmt.Errorf("scim: config for token not found")
		}
		cfgCp := *cfg
		return &cp, &cfgCp, nil
	}
	return nil, nil, fmt.Errorf("scim: token not found")
}

// ──────────────────────────────────────────────────
// Provision logs
// ──────────────────────────────────────────────────

func (s *MemoryStore) CreateLog(_ context.Context, l *ProvisionLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *l
	s.logs[l.ID.String()] = &cp
	return nil
}

func (s *MemoryStore) ListLogs(_ context.Context, configID id.SCIMConfigID, limit int) ([]*ProvisionLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*ProvisionLog
	cfgKey := configID.String()
	for _, l := range s.logs {
		if l.ConfigID.String() == cfgKey {
			cp := *l
			result = append(result, &cp)
		}
	}
	sortLogsByCreated(result)
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *MemoryStore) ListAllLogs(_ context.Context, appID string, limit int) ([]*ProvisionLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Collect config IDs for this app.
	appConfigs := make(map[string]bool)
	for _, c := range s.configs {
		if c.AppID.String() == appID {
			appConfigs[c.ID.String()] = true
		}
	}

	var result []*ProvisionLog
	for _, l := range s.logs {
		if appConfigs[l.ConfigID.String()] {
			cp := *l
			result = append(result, &cp)
		}
	}
	sortLogsByCreated(result)
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *MemoryStore) CountLogsByStatus(_ context.Context, configID id.SCIMConfigID) (success, errors, skipped int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfgKey := configID.String()
	for _, l := range s.logs {
		if l.ConfigID.String() == cfgKey {
			switch l.Status {
			case LogStatusSuccess:
				success++
			case LogStatusError:
				errors++
			case LogStatusSkipped:
				skipped++
			}
		}
	}
	return success, errors, skipped, nil
}

func (s *MemoryStore) CountAllLogsByStatus(_ context.Context, appID string) (success, errors, skipped int, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	appConfigs := make(map[string]bool)
	for _, c := range s.configs {
		if c.AppID.String() == appID {
			appConfigs[c.ID.String()] = true
		}
	}

	for _, l := range s.logs {
		if appConfigs[l.ConfigID.String()] {
			switch l.Status {
			case LogStatusSuccess:
				success++
			case LogStatusError:
				errors++
			case LogStatusSkipped:
				skipped++
			}
		}
	}
	return success, errors, skipped, nil
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

func sortConfigsByCreated(configs []*SCIMConfig) {
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].CreatedAt.After(configs[j].CreatedAt)
	})
}

func sortLogsByCreated(logs []*ProvisionLog) {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].CreatedAt.After(logs[j].CreatedAt)
	})
}
