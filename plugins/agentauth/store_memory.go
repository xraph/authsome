package agentauth

import (
	"context"
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// MemoryStore is an in-process Store for tests and development.
type MemoryStore struct {
	mu       sync.RWMutex
	agents   map[string]*Agent
	grants   map[string]*AgentGrant
	policies map[string]*OrgAgentPolicy
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		agents:   make(map[string]*Agent),
		grants:   make(map[string]*AgentGrant),
		policies: make(map[string]*OrgAgentPolicy),
	}
}

func (s *MemoryStore) CreateAgent(_ context.Context, a *Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *a
	s.agents[a.ID.String()] = &cp
	return nil
}

func (s *MemoryStore) GetAgent(_ context.Context, agentID id.AgentID) (*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.agents[agentID.String()]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *a
	return &cp, nil
}

func (s *MemoryStore) GetAgentByClientID(_ context.Context, clientID string) (*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.agents {
		if a.ClientID == clientID {
			cp := *a
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) UpdateAgent(_ context.Context, a *Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.agents[a.ID.String()]; !ok {
		return ErrNotFound
	}
	cp := *a
	s.agents[a.ID.String()] = &cp
	return nil
}

func (s *MemoryStore) ListAgents(_ context.Context, appID id.AppID, orgID id.OrgID) ([]*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Agent
	for _, a := range s.agents {
		if a.AppID.String() != appID.String() {
			continue
		}
		if !orgID.IsNil() && a.OrgID.String() != orgID.String() {
			continue
		}
		cp := *a
		out = append(out, &cp)
	}
	return out, nil
}

func (s *MemoryStore) CreateAgentGrant(_ context.Context, g *AgentGrant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *g
	s.grants[g.ID.String()] = &cp
	return nil
}

func (s *MemoryStore) GetAgentGrant(_ context.Context, grantID id.AgentGrantID) (*AgentGrant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	g, ok := s.grants[grantID.String()]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *g
	return &cp, nil
}

func (s *MemoryStore) GetActiveGrant(_ context.Context, agentID id.AgentID, userID id.UserID, orgID id.OrgID) (*AgentGrant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	for _, g := range s.grants {
		if g.AgentID.String() != agentID.String() || g.UserID.String() != userID.String() {
			continue
		}
		if g.OrgID.String() != orgID.String() || !g.IsActive(now) {
			continue
		}
		cp := *g
		return &cp, nil
	}
	return nil, ErrNotFound
}

func (s *MemoryStore) ListGrantsByUser(_ context.Context, userID id.UserID) ([]*AgentGrant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*AgentGrant
	for _, g := range s.grants {
		if g.UserID.String() == userID.String() {
			cp := *g
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (s *MemoryStore) UpdateAgentGrant(_ context.Context, g *AgentGrant) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.grants[g.ID.String()]; !ok {
		return ErrNotFound
	}
	cp := *g
	s.grants[g.ID.String()] = &cp
	return nil
}

func (s *MemoryStore) RevokeAgentGrant(_ context.Context, grantID id.AgentGrantID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[grantID.String()]
	if !ok {
		return ErrNotFound
	}
	s.revokeLocked(g)
	return nil
}

func (s *MemoryStore) RevokeGrantsByUser(_ context.Context, userID id.UserID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, g := range s.grants {
		if g.UserID.String() == userID.String() {
			s.revokeLocked(g)
		}
	}
	return nil
}

func (s *MemoryStore) RevokeGrantsByUserOrg(_ context.Context, userID id.UserID, orgID id.OrgID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, g := range s.grants {
		if g.UserID.String() == userID.String() && g.OrgID.String() == orgID.String() {
			s.revokeLocked(g)
		}
	}
	return nil
}

func (s *MemoryStore) RevokeGrantsByAgent(_ context.Context, agentID id.AgentID, orgID id.OrgID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, g := range s.grants {
		if g.AgentID.String() != agentID.String() {
			continue
		}
		if !orgID.IsNil() && g.OrgID.String() != orgID.String() {
			continue
		}
		s.revokeLocked(g)
	}
	return nil
}

// revokeLocked stamps RevokedAt. The caller holds s.mu.
func (s *MemoryStore) revokeLocked(g *AgentGrant) {
	if g.RevokedAt != nil {
		return
	}
	now := time.Now()
	g.RevokedAt = &now
	g.UpdatedAt = now
}

func (s *MemoryStore) GetOrgPolicy(_ context.Context, orgID id.OrgID) (*OrgAgentPolicy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.policies[orgID.String()]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (s *MemoryStore) PutOrgPolicy(_ context.Context, p *OrgAgentPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	s.policies[p.OrgID.String()] = &cp
	return nil
}
