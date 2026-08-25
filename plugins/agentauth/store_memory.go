package agentauth

import (
	"context"
	"fmt"
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
	// A ClientID must resolve to at most one agent: GetAgentByClientID ranges
	// over this same map and returns the first match, in nondeterministic Go
	// map-iteration order, so two agents sharing a ClientID would make
	// whichever one Evaluate sees (approved or blocked) a coin flip. Checked
	// under the same lock as the write below, so two concurrent registrations
	// for the same ClientID cannot both pass the check and both land.
	if a.ClientID != "" {
		for _, existing := range s.agents {
			if existing.ClientID == a.ClientID {
				return ErrConflict
			}
		}
	}
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
	// Same ClientID collision rule CreateAgent enforces, and for the same
	// reason: GetAgentByClientID must resolve to at most one agent.
	// Replacing a record wholesale (which is what this does) can retarget
	// its ClientID onto one another agent already holds just as easily as
	// CreateAgent can — the collision doesn't care which method produced
	// it — so this checked here too, under the same lock, is what the SQL
	// backends' unique index enforces unconditionally on every write.
	if a.ClientID != "" {
		for otherID, existing := range s.agents {
			if otherID != a.ID.String() && existing.ClientID == a.ClientID {
				return ErrConflict
			}
		}
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
	cp.Scopes = append([]string(nil), g.Scopes...)
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
	cp.Scopes = append([]string(nil), g.Scopes...)
	return &cp, nil
}

// GetActiveGrant returns the most recently created matching grant when more
// than one exists. Duplicates are the normal state, not an edge case:
// CreateGrant always inserts a fresh grant rather than upserting, so an
// ordinary re-consent leaves an older active grant for the same
// agent+user+org triple lying around alongside the new one. Ranging over
// s.grants in Go's randomized map-iteration order and returning the first
// match would make this backend disagree, run to run, with the SQL/Mongo
// backends' deterministic "newest wins" — this scans every match instead of
// stopping at the first, to agree with them.
func (s *MemoryStore) GetActiveGrant(_ context.Context, agentID id.AgentID, userID id.UserID, orgID id.OrgID) (*AgentGrant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	var newest *AgentGrant
	for _, g := range s.grants {
		if g.AgentID.String() != agentID.String() || g.UserID.String() != userID.String() {
			continue
		}
		if g.OrgID.String() != orgID.String() || !g.IsActive(now) {
			continue
		}
		if newest == nil || g.CreatedAt.After(newest.CreatedAt) {
			newest = g
		}
	}
	if newest == nil {
		return nil, ErrNotFound
	}
	cp := *newest
	cp.Scopes = append([]string(nil), newest.Scopes...)
	return &cp, nil
}

func (s *MemoryStore) ListGrantsByUser(_ context.Context, userID id.UserID) ([]*AgentGrant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*AgentGrant
	for _, g := range s.grants {
		if g.UserID.String() == userID.String() {
			cp := *g
			cp.Scopes = append([]string(nil), g.Scopes...)
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
	cp.Scopes = append([]string(nil), g.Scopes...)
	s.grants[g.ID.String()] = &cp
	return nil
}

// StampLastUsed mutates only LastUsedAt/UpdatedAt on the stored grant,
// leaving RevokedAt and Scopes untouched — see the Store interface doc
// comment for why that matters.
func (s *MemoryStore) StampLastUsed(_ context.Context, grantID id.AgentGrantID, t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	g, ok := s.grants[grantID.String()]
	if !ok {
		return ErrNotFound
	}
	stamp := t
	g.LastUsedAt = &stamp
	g.UpdatedAt = t
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

func (s *MemoryStore) RevokeGrantsByUser(_ context.Context, userID id.UserID) ([]id.AgentGrantID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var revoked []id.AgentGrantID
	for _, g := range s.grants {
		if g.UserID.String() == userID.String() {
			s.revokeLocked(g)
			revoked = append(revoked, g.ID)
		}
	}
	return revoked, nil
}

func (s *MemoryStore) RevokeGrantsByUserOrg(_ context.Context, userID id.UserID, orgID id.OrgID) ([]id.AgentGrantID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var revoked []id.AgentGrantID
	for _, g := range s.grants {
		if g.UserID.String() == userID.String() && g.OrgID.String() == orgID.String() {
			s.revokeLocked(g)
			revoked = append(revoked, g.ID)
		}
	}
	return revoked, nil
}

// RevokeGrantsByOrg revokes every grant scoped to orgID, regardless of which
// user issued it. Deleting an organization has to disarm every agent acting
// under it, not just one member's — that is the whole org's authorization
// surface, not a single user-org pair the way RevokeGrantsByUserOrg is.
func (s *MemoryStore) RevokeGrantsByOrg(_ context.Context, orgID id.OrgID) ([]id.AgentGrantID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var revoked []id.AgentGrantID
	for _, g := range s.grants {
		if g.OrgID.String() == orgID.String() {
			s.revokeLocked(g)
			revoked = append(revoked, g.ID)
		}
	}
	return revoked, nil
}

func (s *MemoryStore) RevokeGrantsByAgent(_ context.Context, agentID id.AgentID, orgID id.OrgID) ([]id.AgentGrantID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var revoked []id.AgentGrantID
	for _, g := range s.grants {
		if g.AgentID.String() != agentID.String() {
			continue
		}
		if !orgID.IsNil() && g.OrgID.String() != orgID.String() {
			continue
		}
		s.revokeLocked(g)
		revoked = append(revoked, g.ID)
	}
	return revoked, nil
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
	cp.AllowedScopes = append([]string(nil), p.AllowedScopes...)
	return &cp, nil
}

func (s *MemoryStore) PutOrgPolicy(_ context.Context, p *OrgAgentPolicy) error {
	switch p.Mode {
	case ModeOpen, ModeAllowlist, ModeBlocked:
	default:
		// A policy with an unrecognized mode must never make it into the
		// store: Evaluate and CreateGrant treat that case as a deny, but
		// refusing it here means bad data can't exist to be misread in the
		// first place.
		return fmt.Errorf("agentauth: invalid policy mode %q", p.Mode)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *p
	cp.AllowedScopes = append([]string(nil), p.AllowedScopes...)
	s.policies[p.OrgID.String()] = &cp
	return nil
}
