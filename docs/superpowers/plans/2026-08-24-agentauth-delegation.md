# Agent delegation (agentauth) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a user authorize an AI agent to act on their behalf with a scoped, expiring, revocable grant, instead of handing it a static API key or a full session.

**Architecture:** An agent is an OAuth client registered in `oauth2provider`. A grant is a new `AgentGrant` row joining that client to a delegating user, an org, a scope set and an expiry. Agent sessions carry `PrincipalKind = "agent"` while keeping the delegating human in `Session.UserID`. Authorization at request time is the conjunction of a scope gate and the existing Warden permission check on that user.

**Tech Stack:** Go, forge router, grove migrations, Warden for RBAC, testify for assertions. Stores: postgres, sqlite, mongo, memory.

**Spec:** `docs/superpowers/specs/2026-08-24-agentauth-delegation-design.md`

## Global Constraints

- Authorization is strict intersection, always. An agent's effective permission is the granted scopes intersected with what the delegating user can do. No autonomous path.
- `AgentGrant.UserID` is never the zero value. `AgentGrant.ExpiresAt` is a value, not a pointer.
- The agent path fails closed. If no `plugin.PermissionChecker` is available, deny. This deliberately inverts `plugin.PermissionGuard`, which returns nil to degrade to session-only.
- Check the scope gate before the user gate, always. The reverse order leaks the owner's permission set to the agent.
- Do not modify `plugins/oauth2provider` schema, models, or store interfaces. The only permitted change to that package is the additive `ConsentGate` in Task 6.
- Do not modify `rbac/warden_store.go`. The service-account TODO at line 326 is out of scope.
- Agent session issuance must emit `BeforeSessionCreate` and `AfterSignIn`. Never hand-build a `session.Session` the way `plugins/apikey/plugin.go:567` does.
- Test command is `go test ./... ` scoped to the package under test. Lint with `make lint` before each commit.
- Migration versions are fixed: `20260824000060` for the core `authsome` group change to `authsome_sessions` (Task 5), and `20260824000061` for the `authsome-agentauth` group (Task 13). Do not renumber them. Six sibling branches dated today register migrations, and `20260824000001` through `20260824000003`, `20260824000030`, `20260824000040` and `20260824000050` through `20260824000052` are already claimed. Two migrations sharing a version inside one group fails at startup.

## File structure

| File | Responsibility |
|---|---|
| `id/id.go` | Add `AgentID` / `AgentGrantID` types, prefixes, constructors, parsers |
| `plugins/agentauth/agentauth.go` | Domain types: `Agent`, `AgentGrant`, `OrgAgentPolicy`, `Store` |
| `plugins/agentauth/scope.go` | Scope registry and the scope-to-Warden mapping |
| `plugins/agentauth/plugin.go` | Plugin lifecycle, `OnInit`, route registration |
| `plugins/agentauth/grant.go` | Consent evaluation, TTL clamp, grant creation and revocation |
| `plugins/agentauth/middleware.go` | Intersection enforcement |
| `plugins/agentauth/cache.go` | In-process grant cache |
| `plugins/agentauth/lifecycle.go` | Offboarding hooks |
| `plugins/agentauth/handlers.go` | User and admin HTTP surfaces |
| `plugins/agentauth/store_memory.go` | Memory store |
| `plugins/agentauth/store_postgres.go` / `_sqlite.go` / `_mongo.go` | Persistent stores |
| `plugins/agentauth/migrations.go` | Grove migration groups |
| `session/session.go` | Add `AgentID` and `GrantID` fields |
| `plugins/oauth2provider/consent_gate.go` | The one additive interface this plugin needs |

---

### Task 1: Agent identity types

**Files:**
- Modify: `id/id.go`
- Test: `id/id_agent_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `id.AgentID`, `id.AgentGrantID` (both aliases of `id.ID`), `id.NewAgentID() ID`, `id.NewAgentGrantID() ID`, `id.ParseAgentID(string) (ID, error)`, `id.ParseAgentGrantID(string) (ID, error)`, `id.PrefixAgent`, `id.PrefixAgentGrant`.

- [ ] **Step 1: Write the failing test**

Create `id/id_agent_test.go`:

```go
package id_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestNewAgentID_HasAgentPrefix(t *testing.T) {
	a := id.NewAgentID()
	assert.Equal(t, id.PrefixAgent, a.Prefix())
}

func TestParseAgentID_RejectsForeignPrefix(t *testing.T) {
	grant := id.NewAgentGrantID()

	_, err := id.ParseAgentID(grant.String())

	require.Error(t, err, "an agent grant id must not parse as an agent id")
}

func TestParseAgentGrantID_RoundTrips(t *testing.T) {
	original := id.NewAgentGrantID()

	parsed, err := id.ParseAgentGrantID(original.String())

	require.NoError(t, err)
	assert.Equal(t, original.String(), parsed.String())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./id/ -run TestNewAgentID -v`
Expected: FAIL, compile error `undefined: id.PrefixAgent`.

- [ ] **Step 3: Write minimal implementation**

In `id/id.go`, add to the `Prefix` const block alongside `PrefixConsent`:

```go
	PrefixAgent      Prefix = "aagt"
	PrefixAgentGrant Prefix = "aagr"
```

Add the type aliases near `ConsentID`:

```go
// AgentID is a type-safe identifier for agents (prefix: "aagt").
type AgentID = ID

// AgentGrantID is a type-safe identifier for agent grants (prefix: "aagr").
type AgentGrantID = ID
```

Add the constructors near `NewConsentID`:

```go
// NewAgentID generates a new unique agent ID.
func NewAgentID() ID { return New(PrefixAgent) }

// NewAgentGrantID generates a new unique agent grant ID.
func NewAgentGrantID() ID { return New(PrefixAgentGrant) }
```

Add the parsers near `ParseConsentID`:

```go
// ParseAgentID parses a string and validates the "aagt" prefix.
func ParseAgentID(s string) (ID, error) { return ParseWithPrefix(s, PrefixAgent) }

// ParseAgentGrantID parses a string and validates the "aagr" prefix.
func ParseAgentGrantID(s string) (ID, error) { return ParseWithPrefix(s, PrefixAgentGrant) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./id/ -run "TestNewAgentID|TestParseAgent" -v`
Expected: PASS, three tests.

- [ ] **Step 5: Commit**

```bash
git add id/id.go id/id_agent_test.go
git commit -m "feat(id): add agent and agent grant identity types"
```

---

### Task 2: Domain types and the memory store

**Files:**
- Create: `plugins/agentauth/agentauth.go`
- Create: `plugins/agentauth/store_memory.go`
- Test: `plugins/agentauth/store_memory_test.go`

**Interfaces:**
- Consumes: `id.AgentID`, `id.AgentGrantID` from Task 1.
- Produces: `agentauth.Agent`, `agentauth.AgentGrant`, `agentauth.OrgAgentPolicy`, `agentauth.AgentOrigin`, `agentauth.AgentStatus`, `agentauth.PolicyMode`, the `agentauth.Store` interface, `agentauth.NewMemoryStore() *MemoryStore`, `agentauth.ErrNotFound`, and `(*AgentGrant).IsActive(now time.Time) bool`.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/store_memory_test.go`:

```go
package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
)

func newGrant(t *testing.T, userID id.UserID, orgID id.OrgID, expires time.Time) *agentauth.AgentGrant {
	t.Helper()
	return &agentauth.AgentGrant{
		ID:        id.NewAgentGrantID(),
		AppID:     id.NewAppID(),
		AgentID:   id.NewAgentID(),
		UserID:    userID,
		OrgID:     orgID,
		Scopes:    []string{"invoices:read"},
		ExpiresAt: expires,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestMemoryStore_GetAgentGrant_RoundTrips(t *testing.T) {
	s := agentauth.NewMemoryStore()
	g := newGrant(t, id.NewUserID(), id.NewOrgID(), time.Now().Add(time.Hour))

	require.NoError(t, s.CreateAgentGrant(context.Background(), g))

	got, err := s.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.Equal(t, g.UserID.String(), got.UserID.String())
	assert.Equal(t, []string{"invoices:read"}, got.Scopes)
}

func TestMemoryStore_GetAgentGrant_NotFound(t *testing.T) {
	s := agentauth.NewMemoryStore()

	_, err := s.GetAgentGrant(context.Background(), id.NewAgentGrantID())

	require.ErrorIs(t, err, agentauth.ErrNotFound)
}

// Offboarding revokes by user, so the store has to be able to find every
// grant a person issued regardless of which agent holds it.
func TestMemoryStore_RevokeGrantsByUser(t *testing.T) {
	s := agentauth.NewMemoryStore()
	victim, bystander := id.NewUserID(), id.NewUserID()
	org := id.NewOrgID()
	g1 := newGrant(t, victim, org, time.Now().Add(time.Hour))
	g2 := newGrant(t, victim, org, time.Now().Add(time.Hour))
	g3 := newGrant(t, bystander, org, time.Now().Add(time.Hour))
	for _, g := range []*agentauth.AgentGrant{g1, g2, g3} {
		require.NoError(t, s.CreateAgentGrant(context.Background(), g))
	}

	require.NoError(t, s.RevokeGrantsByUser(context.Background(), victim))

	for _, gid := range []id.AgentGrantID{g1.ID, g2.ID} {
		got, err := s.GetAgentGrant(context.Background(), gid)
		require.NoError(t, err)
		assert.NotNil(t, got.RevokedAt, "the victim's grants must be revoked")
	}
	survivor, err := s.GetAgentGrant(context.Background(), g3.ID)
	require.NoError(t, err)
	assert.Nil(t, survivor.RevokedAt, "another user's grant must survive")
}

// Removing someone from one org must not disarm their agents everywhere else.
func TestMemoryStore_RevokeGrantsByUserOrg_ScopedToThatOrg(t *testing.T) {
	s := agentauth.NewMemoryStore()
	user := id.NewUserID()
	leaving, staying := id.NewOrgID(), id.NewOrgID()
	gone := newGrant(t, user, leaving, time.Now().Add(time.Hour))
	kept := newGrant(t, user, staying, time.Now().Add(time.Hour))
	require.NoError(t, s.CreateAgentGrant(context.Background(), gone))
	require.NoError(t, s.CreateAgentGrant(context.Background(), kept))

	require.NoError(t, s.RevokeGrantsByUserOrg(context.Background(), user, leaving))

	g, err := s.GetAgentGrant(context.Background(), gone.ID)
	require.NoError(t, err)
	assert.NotNil(t, g.RevokedAt)
	k, err := s.GetAgentGrant(context.Background(), kept.ID)
	require.NoError(t, err)
	assert.Nil(t, k.RevokedAt, "grants in the user's other orgs must survive")
}

func TestAgentGrant_IsActive(t *testing.T) {
	now := time.Now()
	revoked := now.Add(-time.Minute)

	tests := []struct {
		name  string
		grant agentauth.AgentGrant
		want  bool
	}{
		{"live", agentauth.AgentGrant{ExpiresAt: now.Add(time.Hour)}, true},
		{"expired", agentauth.AgentGrant{ExpiresAt: now.Add(-time.Hour)}, false},
		{"revoked", agentauth.AgentGrant{ExpiresAt: now.Add(time.Hour), RevokedAt: &revoked}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.grant.IsActive(now))
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -v`
Expected: FAIL, `no Go files in .../plugins/agentauth`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/agentauth/agentauth.go`:

```go
// Package agentauth adds delegated agent identity to authsome. An agent is a
// non-human principal that always acts on behalf of a human, which is what
// separates it from a serviceaccount.ServiceAccount.
package agentauth

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/id"
)

// ErrNotFound is returned when an agent or grant cannot be found.
var ErrNotFound = errors.New("agentauth: not found")

// AgentOrigin records how an agent came to exist.
type AgentOrigin string

const (
	OriginSelfRegistered AgentOrigin = "self_registered"
	OriginOrgRegistered  AgentOrigin = "org_registered"
	OriginFirstParty     AgentOrigin = "first_party"
)

// AgentStatus is an agent's approval state.
type AgentStatus string

const (
	StatusPending  AgentStatus = "pending"
	StatusApproved AgentStatus = "approved"
	StatusBlocked  AgentStatus = "blocked"
)

// PolicyMode is an org's stance on agent delegation.
type PolicyMode string

const (
	ModeOpen      PolicyMode = "open"
	ModeAllowlist PolicyMode = "allowlist"
	ModeBlocked   PolicyMode = "blocked"
)

// Agent is a non-human principal that always acts for a human. It is keyed to
// an oauth2provider client by ClientID rather than embedded in it, so this
// package never migrates that plugin's schema.
type Agent struct {
	ID          id.AgentID  `json:"id"`
	AppID       id.AppID    `json:"app_id"`
	OrgID       id.OrgID    `json:"org_id,omitempty"`
	ClientID    string      `json:"client_id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	LogoURI     string      `json:"logo_uri,omitempty"`
	Origin      AgentOrigin `json:"origin"`
	Status      AgentStatus `json:"status"`
	CreatedBy   id.UserID   `json:"created_by,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// AgentGrant is one user's delegation to one agent. UserID is never the zero
// value and ExpiresAt is a value rather than a pointer, so both invariants
// this package depends on are carried by the type.
type AgentGrant struct {
	ID         id.AgentGrantID `json:"id"`
	AppID      id.AppID        `json:"app_id"`
	AgentID    id.AgentID      `json:"agent_id"`
	UserID     id.UserID       `json:"user_id"`
	OrgID      id.OrgID        `json:"org_id,omitempty"`
	Scopes     []string        `json:"scopes"`
	ConsentID  id.ConsentID    `json:"consent_id,omitempty"`
	ExpiresAt  time.Time       `json:"expires_at"`
	LastUsedAt *time.Time      `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time      `json:"revoked_at,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// IsActive reports whether the grant may authorize a request at now.
func (g *AgentGrant) IsActive(now time.Time) bool {
	if g.RevokedAt != nil {
		return false
	}
	return now.Before(g.ExpiresAt)
}

// OrgAgentPolicy is an org's gate on agent delegation.
type OrgAgentPolicy struct {
	OrgID         id.OrgID      `json:"org_id"`
	Mode          PolicyMode    `json:"mode"`
	MaxGrantTTL   time.Duration `json:"max_grant_ttl"`
	AllowedScopes []string      `json:"allowed_scopes,omitempty"`
}

// Store persists agents, grants and org policy.
type Store interface {
	CreateAgent(ctx context.Context, a *Agent) error
	GetAgent(ctx context.Context, agentID id.AgentID) (*Agent, error)
	GetAgentByClientID(ctx context.Context, clientID string) (*Agent, error)
	UpdateAgent(ctx context.Context, a *Agent) error
	ListAgents(ctx context.Context, appID id.AppID, orgID id.OrgID) ([]*Agent, error)

	CreateAgentGrant(ctx context.Context, g *AgentGrant) error
	GetAgentGrant(ctx context.Context, grantID id.AgentGrantID) (*AgentGrant, error)
	GetActiveGrant(ctx context.Context, agentID id.AgentID, userID id.UserID, orgID id.OrgID) (*AgentGrant, error)
	ListGrantsByUser(ctx context.Context, userID id.UserID) ([]*AgentGrant, error)
	UpdateAgentGrant(ctx context.Context, g *AgentGrant) error
	RevokeAgentGrant(ctx context.Context, grantID id.AgentGrantID) error
	RevokeGrantsByUser(ctx context.Context, userID id.UserID) error
	RevokeGrantsByUserOrg(ctx context.Context, userID id.UserID, orgID id.OrgID) error
	RevokeGrantsByAgent(ctx context.Context, agentID id.AgentID, orgID id.OrgID) error

	GetOrgPolicy(ctx context.Context, orgID id.OrgID) (*OrgAgentPolicy, error)
	PutOrgPolicy(ctx context.Context, p *OrgAgentPolicy) error
}
```

Create `plugins/agentauth/store_memory.go`:

```go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -v`
Expected: PASS, all tests including the four `TestAgentGrant_IsActive` subtests.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/
git commit -m "feat(agentauth): add agent and grant domain types with memory store"
```

---

### Task 3: Scope registry and Warden mapping

**Files:**
- Create: `plugins/agentauth/scope.go`
- Test: `plugins/agentauth/scope_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `agentauth.Permission` struct with `Action` and `Resource` string fields, `agentauth.Grants(action, resource string) Permission`, `agentauth.ScopeRegistry` with `Register(scope string, p Permission)`, `Lookup(scope string) (Permission, bool)`, `Known(scope string) bool`, and `Covers(scopes []string, action, resource string) bool`. Also `agentauth.NewScopeRegistry() *ScopeRegistry`.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/scope_test.go`:

```go
package agentauth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/plugins/agentauth"
)

func testRegistry() *agentauth.ScopeRegistry {
	r := agentauth.NewScopeRegistry()
	r.Register("invoices:read", agentauth.Grants("read", "invoice"))
	r.Register("invoices:write", agentauth.Grants("write", "invoice"))
	return r
}

func TestScopeRegistry_Covers(t *testing.T) {
	r := testRegistry()

	tests := []struct {
		name     string
		scopes   []string
		action   string
		resource string
		want     bool
	}{
		{"granted scope covers its permission", []string{"invoices:read"}, "read", "invoice", true},
		{"read does not cover write", []string{"invoices:read"}, "write", "invoice", false},
		{"scope does not cover another resource", []string{"invoices:read"}, "read", "payment", false},
		{"empty grant covers nothing", nil, "read", "invoice", false},
		{"unregistered scope covers nothing", []string{"invoices:delete"}, "delete", "invoice", false},
		{"any one matching scope is enough", []string{"invoices:read", "invoices:write"}, "write", "invoice", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, r.Covers(tt.scopes, tt.action, tt.resource))
		})
	}
}

// A scope with no mapping must be rejected at consent time, so the registry
// has to be able to answer this question before a grant is written.
func TestScopeRegistry_Known(t *testing.T) {
	r := testRegistry()

	assert.True(t, r.Known("invoices:read"))
	assert.False(t, r.Known("invoices:delete"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run TestScopeRegistry -v`
Expected: FAIL, compile error `undefined: agentauth.NewScopeRegistry`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/agentauth/scope.go`:

```go
package agentauth

import "sync"

// Permission is the Warden action and resource a delegation scope maps to.
type Permission struct {
	Action   string
	Resource string
}

// Grants builds the Permission a scope confers. It reads as
// Grants("read", "invoice") at the call site.
func Grants(action, resource string) Permission {
	return Permission{Action: action, Resource: resource}
}

// ScopeRegistry holds the host app's delegation vocabulary. Scopes are
// type-level: "invoices:read" means invoices, not one customer's invoices.
// Instance narrowing comes from the user gate, where Warden's ReBAC has
// already decided which invoices the delegating user may read.
type ScopeRegistry struct {
	mu     sync.RWMutex
	scopes map[string]Permission
}

// NewScopeRegistry returns an empty registry.
func NewScopeRegistry() *ScopeRegistry {
	return &ScopeRegistry{scopes: make(map[string]Permission)}
}

// Register maps a delegation scope onto a Warden permission. Re-registering a
// scope replaces its mapping.
func (r *ScopeRegistry) Register(scope string, p Permission) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scopes[scope] = p
}

// Lookup returns the permission a scope confers.
func (r *ScopeRegistry) Lookup(scope string) (Permission, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.scopes[scope]
	return p, ok
}

// Known reports whether a scope has a registered mapping. Consent uses this to
// reject unmapped scopes before a grant is written, so a stored grant can
// never carry a scope that means nothing.
func (r *ScopeRegistry) Known(scope string) bool {
	_, ok := r.Lookup(scope)
	return ok
}

// Covers reports whether any of the granted scopes confers the given
// permission. An unregistered scope confers nothing.
func (r *ScopeRegistry) Covers(scopes []string, action, resource string) bool {
	for _, s := range scopes {
		p, ok := r.Lookup(s)
		if ok && p.Action == action && p.Resource == resource {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -run TestScopeRegistry -v`
Expected: PASS, six `Covers` subtests plus `Known`.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/scope.go plugins/agentauth/scope_test.go
git commit -m "feat(agentauth): add delegation scope registry and warden mapping"
```

---

### Task 4: Plugin skeleton and options

**Files:**
- Create: `plugins/agentauth/plugin.go`
- Test: `plugins/agentauth/plugin_test.go`

**Interfaces:**
- Consumes: `Store`, `NewMemoryStore` (Task 2), `ScopeRegistry`, `Permission`, `Grants` (Task 3).
- Produces: `agentauth.Plugin`, `agentauth.New(opts ...Option) *Plugin`, `agentauth.Option`, `agentauth.WithScope(scope string, p Permission) Option`, `agentauth.WithStore(s Store) Option`, `agentauth.WithDefaultGrantTTL(d time.Duration) Option`, and the accessors `(*Plugin).Name() string`, `(*Plugin).Scopes() *ScopeRegistry`, `(*Plugin).Store() Store`, `(*Plugin).DefaultGrantTTL() time.Duration`.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/plugin_test.go`:

```go
package agentauth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/agentauth"
)

func TestPlugin_SatisfiesPluginInterface(t *testing.T) {
	var p plugin.Plugin = agentauth.New()
	assert.Equal(t, "agentauth", p.Name())
}

func TestPlugin_WithScope_RegistersMapping(t *testing.T) {
	p := agentauth.New(
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)

	perm, ok := p.Scopes().Lookup("invoices:read")

	require.True(t, ok)
	assert.Equal(t, "read", perm.Action)
	assert.Equal(t, "invoice", perm.Resource)
}

func TestPlugin_DefaultsToMemoryStore(t *testing.T) {
	p := agentauth.New()
	assert.NotNil(t, p.Store(), "a plugin with no store option must still be usable")
}

func TestPlugin_DefaultGrantTTL(t *testing.T) {
	assert.Equal(t, 90*24*time.Hour, agentauth.New().DefaultGrantTTL())

	p := agentauth.New(agentauth.WithDefaultGrantTTL(7 * 24 * time.Hour))
	assert.Equal(t, 7*24*time.Hour, p.DefaultGrantTTL())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run TestPlugin -v`
Expected: FAIL, compile error `undefined: agentauth.New`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/agentauth/plugin.go`:

```go
package agentauth

import (
	"context"
	"time"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/plugin"
)

// defaultGrantTTL caps how long a delegation lives before the user has to
// consent again.
const defaultGrantTTL = 90 * 24 * time.Hour

// Compile-time interface checks.
var (
	_ plugin.Plugin = (*Plugin)(nil)
	_ plugin.OnInit = (*Plugin)(nil)
)

// Plugin is the delegated agent identity plugin.
type Plugin struct {
	engine      plugin.Engine
	store       Store
	scopes      *ScopeRegistry
	hooks       *hook.Bus
	chronicle   bridge.Chronicle
	logger      log.Logger
	permChecker plugin.PermissionChecker
	grantTTL    time.Duration
	basePath    string
}

// Option configures the plugin at construction.
type Option func(*Plugin)

// WithScope maps a delegation scope onto a Warden permission. A scope with no
// mapping is rejected at consent time.
func WithScope(scope string, p Permission) Option {
	return func(pl *Plugin) { pl.scopes.Register(scope, p) }
}

// WithStore injects a persistent store. Without it the plugin uses an
// in-memory store.
func WithStore(s Store) Option {
	return func(pl *Plugin) { pl.store = s }
}

// WithDefaultGrantTTL sets the fallback cap on grant lifetime. An org policy
// with a shorter MaxGrantTTL still wins.
func WithDefaultGrantTTL(d time.Duration) Option {
	return func(pl *Plugin) { pl.grantTTL = d }
}

// New creates the agentauth plugin.
func New(opts ...Option) *Plugin {
	p := &Plugin{
		store:    NewMemoryStore(),
		scopes:   NewScopeRegistry(),
		grantTTL: defaultGrantTTL,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "agentauth" }

// Scopes returns the delegation scope registry.
func (p *Plugin) Scopes() *ScopeRegistry { return p.scopes }

// Store returns the plugin's store.
func (p *Plugin) Store() Store { return p.store }

// DefaultGrantTTL returns the fallback cap on grant lifetime.
func (p *Plugin) DefaultGrantTTL() time.Duration { return p.grantTTL }

// OnInit captures engine capabilities.
func (p *Plugin) OnInit(_ context.Context, engine plugin.Engine) error {
	p.engine = engine
	p.hooks = engine.Hooks()
	p.chronicle = engine.Chronicle()
	p.logger = engine.Logger()
	p.basePath = engine.BasePath()
	if p.basePath == "" {
		p.basePath = "/v1"
	}
	if pc, ok := engine.(plugin.PermissionChecker); ok {
		p.permChecker = pc
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -run TestPlugin -v`
Expected: PASS, four tests.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/plugin.go plugins/agentauth/plugin_test.go
git commit -m "feat(agentauth): add plugin skeleton with scope and store options"
```

---

### Task 5: Agent principal on sessions

This task is where the plan touches core. Read the whole task before starting: the postgres CHECK constraint added by migration `20260620000002` actively rejects an agent session, so the struct change alone will pass tests against memory and fail against postgres.

**Files:**
- Modify: `session/session.go`
- Modify: `store/postgres/models.go:208` area, `store/sqlite/models.go:208` area, `store/mongo/models.go`
- Modify: `store/postgres/migrations.go`, `store/sqlite/migrations.go`
- Test: `store/storetest/session_agent_test.go`

**Interfaces:**
- Consumes: `id.AgentID`, `id.AgentGrantID` (Task 1).
- Produces: `session.PrincipalKindAgent` constant with value `"agent"`, and the fields `Session.AgentID id.AgentID`, `Session.GrantID id.AgentGrantID`.

- [ ] **Step 1: Write the failing test**

Create `store/storetest/session_agent_test.go`. This suite is written against the shared store test harness so every backend runs it:

```go
package storetest_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/store/memory"
)

// An agent session keeps the delegating human in UserID. That is what lets
// DeleteUserSessions, audit records and org resolution keep working with no
// new branches, and it is the whole reason agentauth does not copy the
// service-account shape.
func TestSession_AgentPrincipal_RoundTrips(t *testing.T) {
	st := memory.New()
	ctx := context.Background()
	userID := id.NewUserID()
	agentID := id.NewAgentID()
	grantID := id.NewAgentGrantID()

	s := &session.Session{
		ID:            id.NewSessionID(),
		AppID:         id.NewAppID(),
		UserID:        userID,
		PrincipalKind: session.PrincipalKindAgent,
		AgentID:       agentID,
		GrantID:       grantID,
		Token:         "tok_agent_roundtrip",
		ExpiresAt:     time.Now().Add(time.Hour),
		CreatedAt:     time.Now(),
	}
	require.NoError(t, st.CreateSession(ctx, s))

	got, err := st.GetSessionByToken(ctx, "tok_agent_roundtrip")
	require.NoError(t, err)
	assert.Equal(t, session.PrincipalKindAgent, got.PrincipalKind)
	assert.Equal(t, userID.String(), got.UserID.String(), "the delegating human must survive the round trip")
	assert.Equal(t, agentID.String(), got.AgentID.String())
	assert.Equal(t, grantID.String(), got.GrantID.String())
}

// Offboarding leans entirely on this. DeleteUserSessions already exists and is
// already called on user deletion, so an agent session carrying the delegating
// user's id is swept up with no change to that code path.
func TestSession_DeleteUserSessions_SweepsAgentSessions(t *testing.T) {
	st := memory.New()
	ctx := context.Background()
	userID := id.NewUserID()

	human := &session.Session{
		ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: userID,
		Token: "tok_human", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}
	agent := &session.Session{
		ID: id.NewSessionID(), AppID: id.NewAppID(), UserID: userID,
		PrincipalKind: session.PrincipalKindAgent,
		AgentID:       id.NewAgentID(), GrantID: id.NewAgentGrantID(),
		Token: "tok_agent", ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
	}
	require.NoError(t, st.CreateSession(ctx, human))
	require.NoError(t, st.CreateSession(ctx, agent))

	require.NoError(t, st.DeleteUserSessions(ctx, userID))

	_, err := st.GetSessionByToken(ctx, "tok_agent")
	require.Error(t, err, "an agent session must die with its delegating user")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./store/storetest/ -run TestSession_Agent -v`
Expected: FAIL, compile error `undefined: session.PrincipalKindAgent`.

- [ ] **Step 3: Write minimal implementation**

In `session/session.go`, add the constant above the `Session` struct:

```go
// Principal kinds a session may carry. An empty PrincipalKind means
// PrincipalKindUser, for rows written before the column existed.
const (
	PrincipalKindUser           = "user"
	PrincipalKindServiceAccount = "service_account"
	// PrincipalKindAgent marks a session issued to a delegated agent. Unlike
	// a service-account session, UserID stays populated with the delegating
	// human, so every consumer that resolves a session's user keeps working.
	PrincipalKindAgent = "agent"
)
```

Add the fields to `Session`, below `ServiceAccountID`:

```go
	// AgentID is set when PrincipalKind is "agent". UserID remains the
	// delegating human.
	AgentID id.AgentID `json:"agent_id,omitempty"`
	// GrantID names the AgentGrant that authorized this session, so revoking
	// that grant can find and delete the sessions it issued.
	GrantID id.AgentGrantID `json:"grant_id,omitempty"`
```

In `store/postgres/models.go` and `store/sqlite/models.go`, add to the session model struct next to `PrincipalKind`:

```go
	AgentID string `grove:"agent_id"`
	GrantID string `grove:"grant_id"`
```

Map both in that file's to-domain and from-domain conversion functions, following exactly how `ServiceAccountID` is mapped there.

In `store/mongo/models.go`, add the same two fields with bson tags `agent_id` and `grant_id`, mapped the same way as `service_account_id`.

Add a postgres migration in `store/postgres/migrations.go`. The constraint rewrite is the point of it:

```go
		&migrate.Migration{
			Name:    "add_session_agent_principal",
			Version: "20260824000060",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				// The principal CHECK added in 20260620000002 admits only a
				// service-account session (no user) or a user session (no
				// service account). An agent session is neither: it names an
				// agent AND the human who delegated to it, so every insert
				// would fail with SQLSTATE 23514 until this widens.
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions
    ADD COLUMN IF NOT EXISTS agent_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS grant_id TEXT NOT NULL DEFAULT '';

ALTER TABLE authsome_sessions
    DROP CONSTRAINT IF EXISTS authsome_sessions_principal_check;
ALTER TABLE authsome_sessions
    ADD CONSTRAINT authsome_sessions_principal_check CHECK (
        (principal_kind = 'service_account'
             AND service_account_id <> '' AND user_id = '')
        OR (principal_kind = 'agent'
             AND agent_id <> '' AND user_id <> '' AND service_account_id = '')
        OR (principal_kind IN ('', 'user')
             AND user_id <> '' AND service_account_id = '' AND agent_id = '')
    );

CREATE INDEX IF NOT EXISTS idx_authsome_sessions_grant_id
    ON authsome_sessions (grant_id)
    WHERE grant_id <> '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				// Agent sessions cannot satisfy the narrower constraint, so
				// they go before it is restored.
				_, err := exec.Exec(ctx, `
DELETE FROM authsome_sessions WHERE principal_kind = 'agent';
DROP INDEX IF EXISTS idx_authsome_sessions_grant_id;
ALTER TABLE authsome_sessions
    DROP CONSTRAINT IF EXISTS authsome_sessions_principal_check;
ALTER TABLE authsome_sessions
    ADD CONSTRAINT authsome_sessions_principal_check CHECK (
        (principal_kind = 'service_account'
             AND service_account_id <> '' AND user_id = '')
        OR (principal_kind IN ('', 'user')
             AND user_id <> '' AND service_account_id = '')
    );
ALTER TABLE authsome_sessions DROP COLUMN IF EXISTS agent_id;
ALTER TABLE authsome_sessions DROP COLUMN IF EXISTS grant_id;
`)
				return err
			},
		},
```

Add the sqlite counterpart in `store/sqlite/migrations.go`, matching the style at line 1026. SQLite has no constraint to rewrite here, so it is columns only:

```go
		&migrate.Migration{
			Name:    "add_session_agent_principal",
			Version: "20260824000060",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions ADD COLUMN agent_id TEXT NOT NULL DEFAULT '';
ALTER TABLE authsome_sessions ADD COLUMN grant_id TEXT NOT NULL DEFAULT '';
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
ALTER TABLE authsome_sessions DROP COLUMN agent_id;
ALTER TABLE authsome_sessions DROP COLUMN grant_id;
`)
				return err
			},
		},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./store/... -run TestSession -v`
Expected: PASS. Then run the full store suite, `go test ./store/...`, and confirm no existing session test regressed on the constraint change.

- [ ] **Step 5: Commit**

```bash
git add session/session.go store/
git commit -m "feat(session): add agent principal kind with delegating user retained"
```

---

### Task 6: The oauth2provider consent gate

This is the only change permitted to `plugins/oauth2provider`. Keep it additive. Four other branches are editing that package.

**Files:**
- Create: `plugins/oauth2provider/consent_gate.go`
- Modify: the authorize handler in `plugins/oauth2provider/plugin.go`, at the point where an authorization code is about to be issued
- Test: `plugins/oauth2provider/consent_gate_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `oauth2provider.ConsentGate` interface with the single method `Evaluate(ctx context.Context, clientID string, userID id.UserID, orgID id.OrgID, scopes []string) error`; a `ConsentGate ConsentGate` field on the existing `oauth2provider.Config`; and the setter `(*Plugin).SetConsentGate(g ConsentGate)`.

Note on idiom: `oauth2provider.New` takes `cfg ...Config`, not functional options, and this package has no `Option` type. Do not invent one. Configuration goes on `Config`, and post-construction wiring uses a `SetX` method, matching `SetConsentStore` in `plugins/consent/plugin.go`.

- [ ] **Step 1: Write the failing test**

Create `plugins/oauth2provider/consent_gate_test.go`:

```go
package oauth2provider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

type recordingGate struct {
	called   bool
	clientID string
	scopes   []string
	err      error
}

func (g *recordingGate) Evaluate(_ context.Context, clientID string, _ id.UserID, _ id.OrgID, scopes []string) error {
	g.called = true
	g.clientID = clientID
	g.scopes = scopes
	return g.err
}

func TestConsentGate_GateIsConsulted(t *testing.T) {
	gate := &recordingGate{}
	p := oauth2provider.New(oauth2provider.Config{ConsentGate: gate})

	err := p.EvaluateConsent(context.Background(), "client_abc", id.NewUserID(), id.NewOrgID(), []string{"invoices:read"})

	require.NoError(t, err)
	assert.True(t, gate.called)
	assert.Equal(t, "client_abc", gate.clientID)
	assert.Equal(t, []string{"invoices:read"}, gate.scopes)
}

func TestConsentGate_RefusalPropagates(t *testing.T) {
	denied := errors.New("org policy blocks this agent")
	p := oauth2provider.New(oauth2provider.Config{ConsentGate: &recordingGate{err: denied}})

	err := p.EvaluateConsent(context.Background(), "client_abc", id.NewUserID(), id.NewOrgID(), nil)

	require.ErrorIs(t, err, denied)
}

// Without a gate the provider must behave exactly as it does today.
func TestConsentGate_SetterWiresGate(t *testing.T) {
	gate := &recordingGate{}
	p := oauth2provider.New()
	p.SetConsentGate(gate)

	require.NoError(t, p.EvaluateConsent(context.Background(), "client_abc", id.NewUserID(), id.NewOrgID(), nil))
	assert.True(t, gate.called, "the setter must wire the gate as effectively as Config does")
}

func TestEvaluateConsent_NoGateAllows(t *testing.T) {
	p := oauth2provider.New()

	err := p.EvaluateConsent(context.Background(), "client_abc", id.NewUserID(), id.NewOrgID(), []string{"anything"})

	require.NoError(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/oauth2provider/ -run ConsentGate -v`
Expected: FAIL, compile error `unknown field ConsentGate in struct literal`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/oauth2provider/consent_gate.go`:

```go
package oauth2provider

import (
	"context"

	"github.com/xraph/authsome/id"
)

// ConsentGate is an optional hook another plugin implements to veto an
// authorization before a code is issued. agentauth registers itself as the
// gate so an org's policy on delegated agents is enforced at the moment a
// user consents, which is the only point where "may this agent touch our
// data" is a well-formed question.
//
// A nil gate allows everything, so the provider's behavior is unchanged when
// no plugin registers one.
type ConsentGate interface {
	// Evaluate returns a non-nil error to refuse the authorization. The error
	// is surfaced to the caller, so it should be a forge HTTP error.
	Evaluate(ctx context.Context, clientID string, userID id.UserID, orgID id.OrgID, scopes []string) error
}

// SetConsentGate registers a gate consulted before every authorization code is
// issued. Use it when the gate is only available after the provider has been
// constructed; otherwise set Config.ConsentGate.
func (p *Plugin) SetConsentGate(g ConsentGate) { p.consentGate = g }

// EvaluateConsent runs the registered gate, if any.
func (p *Plugin) EvaluateConsent(ctx context.Context, clientID string, userID id.UserID, orgID id.OrgID, scopes []string) error {
	if p.consentGate == nil {
		return nil
	}
	return p.consentGate.Evaluate(ctx, clientID, userID, orgID, scopes)
}
```

Add the field `consentGate ConsentGate` to the `Plugin` struct in `plugins/oauth2provider/plugin.go`, add `ConsentGate ConsentGate` to the `Config` struct there, and carry it across in `New` alongside the existing default-filling (`p.consentGate = c.ConsentGate`).

In the authorize handler, immediately before the authorization code is created, insert:

```go
	if err := p.EvaluateConsent(ctx, req.ClientID, userID, orgID, scopes); err != nil {
		return nil, err
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/oauth2provider/ -v`
Expected: PASS, three new tests and no regressions in the existing authorize tests.

- [ ] **Step 5: Commit**

```bash
git add plugins/oauth2provider/consent_gate.go plugins/oauth2provider/consent_gate_test.go plugins/oauth2provider/plugin.go
git commit -m "feat(oauth2provider): add optional consent gate hook"
```

---

### Task 7: Consent evaluation, TTL clamp and grant creation

**Files:**
- Create: `plugins/agentauth/grant.go`
- Test: `plugins/agentauth/grant_test.go`

**Interfaces:**
- Consumes: `Store`, `AgentGrant`, `OrgAgentPolicy`, `PolicyMode`, `AgentStatus`, `ErrNotFound` (Task 2); `ScopeRegistry` (Task 3); `Plugin` (Task 4); `oauth2provider.ConsentGate` (Task 6).
- Produces: `(*Plugin).Evaluate(ctx, clientID string, userID id.UserID, orgID id.OrgID, scopes []string) error` satisfying `oauth2provider.ConsentGate`; `(*Plugin).clampTTL(policy *OrgAgentPolicy, requested time.Duration) time.Duration`; `(*Plugin).CreateGrant(ctx context.Context, in CreateGrantInput) (*AgentGrant, error)`; the `agentauth.CreateGrantInput` struct with fields `AppID id.AppID`, `AgentID id.AgentID`, `UserID id.UserID`, `OrgID id.OrgID`, `Scopes []string`, `RequestedTTL time.Duration`.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/grant_test.go`:

```go
package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
)

func approvedAgent(t *testing.T, s agentauth.Store, orgID id.OrgID, clientID string) *agentauth.Agent {
	t.Helper()
	a := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: id.NewAppID(), OrgID: orgID,
		ClientID: clientID, Name: "Test Agent",
		Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateAgent(context.Background(), a))
	return a
}

func TestEvaluate_BlockedOrgRefusesEvenApprovedAgent(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_blocked")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeBlocked,
	}))

	err := p.Evaluate(context.Background(), "client_blocked", id.NewUserID(), org, []string{"invoices:read"})

	require.Error(t, err, "a blocked org must refuse consent even for an approved agent")
}

func TestEvaluate_AllowlistRefusesPendingAgent(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	a := approvedAgent(t, store, org, "client_pending")
	a.Status = agentauth.StatusPending
	require.NoError(t, store.UpdateAgent(context.Background(), a))
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeAllowlist,
	}))

	err := p.Evaluate(context.Background(), "client_pending", id.NewUserID(), org, []string{"invoices:read"})

	require.Error(t, err)
}

func TestEvaluate_UnmappedScopeRefusedAtConsent(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_open")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen,
	}))

	err := p.Evaluate(context.Background(), "client_open", id.NewUserID(), org, []string{"invoices:delete"})

	require.Error(t, err, "a scope with no warden mapping must never reach a stored grant")
}

func TestEvaluate_ScopeOutsideOrgCeilingRefused(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
		agentauth.WithScope("invoices:write", agentauth.Grants("write", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_ceiling")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen, AllowedScopes: []string{"invoices:read"},
	}))

	err := p.Evaluate(context.Background(), "client_ceiling", id.NewUserID(), org, []string{"invoices:write"})

	require.Error(t, err)
}

func TestEvaluate_OpenOrgAllowsMappedScope(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_ok")
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen,
	}))

	err := p.Evaluate(context.Background(), "client_ok", id.NewUserID(), org, []string{"invoices:read"})

	require.NoError(t, err)
}

// An org with no policy row falls back to open. Changing this default is a
// policy decision, not an implementation detail, so it gets its own test.
func TestEvaluate_MissingPolicyDefaultsToOpen(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
	)
	org := id.NewOrgID()
	approvedAgent(t, store, org, "client_nopolicy")

	err := p.Evaluate(context.Background(), "client_nopolicy", id.NewUserID(), org, []string{"invoices:read"})

	require.NoError(t, err)
}

func TestCreateGrant_ClampsTTLToOrgCeiling(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store), agentauth.WithDefaultGrantTTL(90*24*time.Hour))
	org := id.NewOrgID()
	require.NoError(t, store.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
		OrgID: org, Mode: agentauth.ModeOpen, MaxGrantTTL: 24 * time.Hour,
	}))

	g, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: id.NewAppID(), AgentID: id.NewAgentID(), UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"}, RequestedTTL: 365 * 24 * time.Hour,
	})

	require.NoError(t, err)
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), g.ExpiresAt, time.Minute,
		"the org ceiling must win over both the request and the plugin default")
}

func TestCreateGrant_RejectsZeroUser(t *testing.T) {
	p := agentauth.New()

	_, err := p.CreateGrant(context.Background(), agentauth.CreateGrantInput{
		AppID: id.NewAppID(), AgentID: id.NewAgentID(), OrgID: id.NewOrgID(),
		Scopes: []string{"invoices:read"},
	})

	require.Error(t, err, "a grant with no delegating human must be impossible to create")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run "TestEvaluate|TestCreateGrant" -v`
Expected: FAIL, compile error `p.Evaluate undefined`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/agentauth/grant.go`:

```go
package agentauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/oauth2provider"
)

// Compile-time check: the plugin is the provider's consent gate.
var _ oauth2provider.ConsentGate = (*Plugin)(nil)

// CreateGrantInput describes a delegation a user is about to authorize.
type CreateGrantInput struct {
	AppID        id.AppID
	AgentID      id.AgentID
	UserID       id.UserID
	OrgID        id.OrgID
	Scopes       []string
	ConsentID    id.ConsentID
	RequestedTTL time.Duration
}

// Evaluate implements oauth2provider.ConsentGate. It runs at the moment a
// user consents, which is the only point where an org's stance on a given
// agent is a well-formed question. Registration is app-global.
func (p *Plugin) Evaluate(ctx context.Context, clientID string, _ id.UserID, orgID id.OrgID, scopes []string) error {
	agent, err := p.store.GetAgentByClientID(ctx, clientID)
	if errors.Is(err, ErrNotFound) {
		// Not an agent, just an ordinary OAuth client. Not this gate's business.
		return nil
	}
	if err != nil {
		return fmt.Errorf("agentauth: load agent: %w", err)
	}

	if agent.Status == StatusBlocked {
		return forge.Forbidden("agent is blocked")
	}

	policy, err := p.policyFor(ctx, orgID)
	if err != nil {
		return err
	}

	switch policy.Mode {
	case ModeBlocked:
		return forge.Forbidden("this organization does not allow agent delegation")
	case ModeAllowlist:
		if agent.Status != StatusApproved {
			return forge.Forbidden("agent is not approved for this organization")
		}
	case ModeOpen:
		// Any non-blocked agent may be authorized.
	}

	for _, s := range scopes {
		if !p.scopes.Known(s) {
			return forge.BadRequest(fmt.Sprintf("unknown delegation scope %q", s))
		}
		if !scopeAllowed(policy.AllowedScopes, s) {
			return forge.Forbidden(fmt.Sprintf("scope %q is not permitted by organization policy", s))
		}
	}
	return nil
}

// CreateGrant writes the delegation. It refuses a grant with no delegating
// human, which is the invariant the whole authorization model rests on.
func (p *Plugin) CreateGrant(ctx context.Context, in CreateGrantInput) (*AgentGrant, error) {
	if in.UserID.IsNil() {
		return nil, errors.New("agentauth: a grant requires a delegating user")
	}

	policy, err := p.policyFor(ctx, in.OrgID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	g := &AgentGrant{
		ID:        id.NewAgentGrantID(),
		AppID:     in.AppID,
		AgentID:   in.AgentID,
		UserID:    in.UserID,
		OrgID:     in.OrgID,
		Scopes:    in.Scopes,
		ConsentID: in.ConsentID,
		ExpiresAt: now.Add(p.clampTTL(policy, in.RequestedTTL)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := p.store.CreateAgentGrant(ctx, g); err != nil {
		return nil, fmt.Errorf("agentauth: create grant: %w", err)
	}
	return g, nil
}

// clampTTL takes the shortest of the request, the org ceiling and the plugin
// default. A zero request means "use the default".
func (p *Plugin) clampTTL(policy *OrgAgentPolicy, requested time.Duration) time.Duration {
	ttl := p.grantTTL
	if requested > 0 && requested < ttl {
		ttl = requested
	}
	if policy != nil && policy.MaxGrantTTL > 0 && policy.MaxGrantTTL < ttl {
		ttl = policy.MaxGrantTTL
	}
	return ttl
}

// policyFor returns the org's policy, defaulting to open when no row exists.
func (p *Plugin) policyFor(ctx context.Context, orgID id.OrgID) (*OrgAgentPolicy, error) {
	policy, err := p.store.GetOrgPolicy(ctx, orgID)
	if errors.Is(err, ErrNotFound) {
		return &OrgAgentPolicy{OrgID: orgID, Mode: ModeOpen}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agentauth: load org policy: %w", err)
	}
	return policy, nil
}

// scopeAllowed reports whether a scope sits inside the org's ceiling. An empty
// ceiling means no restriction beyond the registry.
func scopeAllowed(ceiling []string, scope string) bool {
	if len(ceiling) == 0 {
		return true
	}
	for _, s := range ceiling {
		if s == scope {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -run "TestEvaluate|TestCreateGrant" -v`
Expected: PASS, eight tests.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/grant.go plugins/agentauth/grant_test.go
git commit -m "feat(agentauth): add consent evaluation, ttl clamp and grant creation"
```

---

### Task 8: Intersection enforcement

The security core. Scope gate first, then user gate, and deny when no permission checker exists.

**Files:**
- Create: `plugins/agentauth/middleware.go`
- Test: `plugins/agentauth/middleware_test.go`

**Interfaces:**
- Consumes: `Store`, `AgentGrant` (Task 2); `ScopeRegistry` (Task 3); `Plugin`, `permChecker` (Task 4); `session.PrincipalKindAgent` (Task 5).
- Produces: `(*Plugin).Authorize(ctx context.Context, sess *session.Session, action, resource string) error`, plus the sentinel errors `agentauth.ErrInsufficientScope` and `agentauth.ErrGrantInactive`.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/middleware_test.go`:

```go
package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/session"
)

// stubChecker stands in for the engine's Warden-backed PermissionChecker.
type stubChecker struct {
	allow map[string]bool
}

func (c *stubChecker) HasPermission(_ context.Context, userID id.UserID, action, resource string) (bool, error) {
	return c.allow[userID.String()+"|"+action+"|"+resource], nil
}

func agentSetup(t *testing.T, scopes []string, expires time.Time) (*agentauth.Plugin, *agentauth.MemoryStore, *session.Session, id.UserID) {
	t.Helper()
	store := agentauth.NewMemoryStore()
	p := agentauth.New(
		agentauth.WithStore(store),
		agentauth.WithScope("invoices:read", agentauth.Grants("read", "invoice")),
		agentauth.WithScope("invoices:write", agentauth.Grants("write", "invoice")),
	)
	userID := id.NewUserID()
	agentID := id.NewAgentID()
	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: agentID,
		UserID: userID, OrgID: id.NewOrgID(), Scopes: scopes,
		ExpiresAt: expires, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), g))
	sess := &session.Session{
		ID: id.NewSessionID(), AppID: g.AppID, UserID: userID,
		PrincipalKind: session.PrincipalKindAgent, AgentID: agentID, GrantID: g.ID,
	}
	return p, store, sess, userID
}

// The intersection property. If this ever passes, the feature is broken.
func TestAuthorize_AgentCannotExceedItsOwner(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:write"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true, // owner may read, but not write
	}})

	err := p.Authorize(context.Background(), sess, "write", "invoice")

	require.Error(t, err, "an agent granted write must still be refused when its owner cannot write")
}

func TestAuthorize_AllowsWhenScopeAndOwnerBothPermit(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})

	require.NoError(t, p.Authorize(context.Background(), sess, "read", "invoice"))
}

// Revoking a permission from the owner narrows every agent acting for them on
// the very next request. This is what proves agent authorization never rides
// the stamped-roles fast path on the session.
func TestAuthorize_OwnerLosingPermissionNarrowsAgentImmediately(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	checker := &stubChecker{allow: map[string]bool{userID.String() + "|read|invoice": true}}
	p.SetPermissionChecker(checker)
	require.NoError(t, p.Authorize(context.Background(), sess, "read", "invoice"))

	checker.allow[userID.String()+"|read|invoice"] = false

	require.Error(t, p.Authorize(context.Background(), sess, "read", "invoice"))
}

func TestAuthorize_MissingScopeIsInsufficientScope(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|write|invoice": true,
	}})

	err := p.Authorize(context.Background(), sess, "write", "invoice")

	require.ErrorIs(t, err, agentauth.ErrInsufficientScope)
}

func TestAuthorize_ExpiredGrantIsRefused(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(-time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})

	err := p.Authorize(context.Background(), sess, "read", "invoice")

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

func TestAuthorize_RevokedGrantIsRefused(t *testing.T) {
	p, store, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})
	require.NoError(t, store.RevokeAgentGrant(context.Background(), sess.GrantID))

	err := p.Authorize(context.Background(), sess, "read", "invoice")

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

// The agent path fails closed. plugin.PermissionGuard deliberately degrades to
// session-only when no checker exists; for agents that would be a hole,
// because the user gate is the entire security model.
func TestAuthorize_NoPermissionCheckerDenies(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))

	err := p.Authorize(context.Background(), sess, "read", "invoice")

	require.Error(t, err, "no permission checker must deny, never degrade")
}

// A human session is not this plugin's business and must pass straight through.
func TestAuthorize_NonAgentSessionPassesThrough(t *testing.T) {
	p := agentauth.New()
	sess := &session.Session{ID: id.NewSessionID(), UserID: id.NewUserID()}

	assert.NoError(t, p.Authorize(context.Background(), sess, "read", "invoice"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run TestAuthorize -v`
Expected: FAIL, compile error `p.Authorize undefined`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/agentauth/middleware.go`:

```go
package agentauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
)

// Sentinel errors so callers can map to the right HTTP response without
// string matching.
var (
	// ErrInsufficientScope means the grant does not confer the permission the
	// route requires. Safe to report to the agent: it names only what the
	// agent was granted, which the agent already knows.
	ErrInsufficientScope = errors.New("agentauth: insufficient scope")
	// ErrGrantInactive means the grant is missing, revoked or expired.
	ErrGrantInactive = errors.New("agentauth: grant is not active")
	// ErrNoPermissionChecker means the engine exposes no RBAC, so the user
	// gate cannot run and the request is denied.
	ErrNoPermissionChecker = errors.New("agentauth: no permission checker available")
)

// SetPermissionChecker injects the RBAC checker. OnInit does this from the
// engine; tests use it directly.
func (p *Plugin) SetPermissionChecker(pc plugin.PermissionChecker) { p.permChecker = pc }

// Authorize enforces the intersection: an agent may do something only if its
// grant confers it AND the delegating user can do it. A non-agent session
// passes straight through, since this plugin has no opinion on human traffic.
//
// The scope gate runs first. It is cheaper, a map lookup ahead of a Warden
// call, and it is the safe order: reporting a user-gate failure first would
// let an agent enumerate its owner's permissions one probe at a time.
func (p *Plugin) Authorize(ctx context.Context, sess *session.Session, action, resource string) error {
	if sess == nil || sess.PrincipalKind != session.PrincipalKindAgent {
		return nil
	}

	grant, err := p.store.GetAgentGrant(ctx, sess.GrantID)
	if errors.Is(err, ErrNotFound) {
		return ErrGrantInactive
	}
	if err != nil {
		return fmt.Errorf("agentauth: load grant: %w", err)
	}
	if !grant.IsActive(time.Now()) {
		return ErrGrantInactive
	}

	if !p.scopes.Covers(grant.Scopes, action, resource) {
		return ErrInsufficientScope
	}

	// Fail closed. plugin.PermissionGuard returns nil here on purpose so an
	// RBAC-less engine still enforces sessions. For an agent that fallback is
	// a hole, because the user gate is the security model.
	if p.permChecker == nil {
		return ErrNoPermissionChecker
	}

	allowed, err := p.permChecker.HasPermission(ctx, grant.UserID, action, resource)
	if err != nil {
		return fmt.Errorf("agentauth: permission check: %w", err)
	}
	if !allowed {
		return fmt.Errorf("agentauth: delegating user lacks %s on %s", action, resource)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -run TestAuthorize -v`
Expected: PASS, eight tests.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/middleware.go plugins/agentauth/middleware_test.go
git commit -m "feat(agentauth): enforce intersection of granted scope and owner permission"
```

---

### Task 9: Session issuance through the hook path

This is the task that keeps agent traffic visible to the risk plugins. Do not hand-build a `session.Session`.

**Files:**
- Create: `plugins/agentauth/issue.go`
- Test: `plugins/agentauth/issue_test.go`

**Interfaces:**
- Consumes: `AgentGrant` (Task 2); `Plugin` (Task 4); `session.PrincipalKindAgent` (Task 5).
- Produces: `(*Plugin).IssueAgentSession(ctx context.Context, grant *AgentGrant) (*session.Session, error)`.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/issue_test.go`:

```go
package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// recordingHooks is a plugin that records which lifecycle hooks fired during
// issuance. It is registered on a plugin.Registry, which is what carries the
// Emit* methods.
type recordingHooks struct {
	beforeSessionCreate int
	afterSignIn         int
}

func (h *recordingHooks) Name() string { return "recording-hooks" }

func (h *recordingHooks) OnBeforeSessionCreate(_ context.Context, _ *session.Session) error {
	h.beforeSessionCreate++
	return nil
}

func (h *recordingHooks) OnAfterSignIn(_ context.Context, _ *user.User, _ *session.Session) error {
	h.afterSignIn++
	return nil
}

// The risk plugins subscribe to BeforeSessionCreate and AfterSignIn. The API
// key plugin hand-builds a synthetic session at plugins/apikey/plugin.go:567
// and therefore fires neither, which is exactly why API key traffic is
// invisible to riskengine, impossibletravel and the rest today. This test
// stops somebody optimizing agent issuance into that same shape later and
// silently taking the risk plugins offline.
func TestIssueAgentSession_FiresRiskHooks(t *testing.T) {
	hooks := &recordingHooks{}
	p, grant := issuanceSetup(t, hooks)

	_, err := p.IssueAgentSession(context.Background(), grant)

	require.NoError(t, err)
	assert.Equal(t, 1, hooks.beforeSessionCreate, "BeforeSessionCreate must fire so riskengine can score agent traffic")
	assert.Equal(t, 1, hooks.afterSignIn, "AfterSignIn must fire so impossibletravel records the agent's location")
}

func TestIssueAgentSession_StampsAgentPrincipalAndKeepsUser(t *testing.T) {
	p, grant := issuanceSetup(t, &recordingHooks{})

	sess, err := p.IssueAgentSession(context.Background(), grant)

	require.NoError(t, err)
	assert.Equal(t, session.PrincipalKindAgent, sess.PrincipalKind)
	assert.Equal(t, grant.UserID.String(), sess.UserID.String(), "the delegating human stays on the session")
	assert.Equal(t, grant.AgentID.String(), sess.AgentID.String())
	assert.Equal(t, grant.ID.String(), sess.GrantID.String())
	assert.True(t, sess.ServiceAccountID.IsNil(), "an agent session is not a service account session")
}

// The session must never outlive the grant that authorized it.
func TestIssueAgentSession_NeverOutlivesGrant(t *testing.T) {
	p, grant := issuanceSetup(t, &recordingHooks{})
	grant.ExpiresAt = time.Now().Add(5 * time.Minute)

	sess, err := p.IssueAgentSession(context.Background(), grant)

	require.NoError(t, err)
	assert.False(t, sess.ExpiresAt.After(grant.ExpiresAt),
		"a session outliving its grant would resurrect a revoked delegation")
}

func TestIssueAgentSession_RefusesInactiveGrant(t *testing.T) {
	p, grant := issuanceSetup(t, &recordingHooks{})
	grant.ExpiresAt = time.Now().Add(-time.Minute)

	_, err := p.IssueAgentSession(context.Background(), grant)

	require.ErrorIs(t, err, agentauth.ErrGrantInactive)
}

// issuanceSetup is implemented in Step 3, below. It builds a plugin wired to
// a stub plugin.Engine whose Plugins() returns a registry with the recording
// plugin on it, and whose Store() returns a memory store.
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run TestIssueAgentSession -v`
Expected: FAIL, compile error `undefined: issuanceSetup`.

- [ ] **Step 3: Write minimal implementation**

First add the test harness at the bottom of `plugins/agentauth/issue_test.go`. It builds a `plugin.Engine` whose `Plugins()` returns a registry carrying the recording plugin, and whose `Store()` returns `memory.New()`:

```go
func issuanceSetup(t *testing.T, hooks *recordingHooks) (*agentauth.Plugin, *agentauth.AgentGrant) {
	t.Helper()
	reg := plugin.NewRegistry(log.NewNoopLogger())
	reg.Register(hooks)

	eng := &stubEngine{store: memory.New(), registry: reg, bus: hook.NewBus(log.NewNoopLogger())}
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	require.NoError(t, p.OnInit(context.Background(), eng))

	userID := id.NewUserID()
	require.NoError(t, eng.store.CreateUser(context.Background(), &user.User{
		ID: userID, AppID: id.NewAppID(), Email: "owner@example.com", CreatedAt: time.Now(),
	}))

	grant := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
		UserID: userID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), grant))
	return p, grant
}
```

Implement `stubEngine` in the same test file satisfying `plugin.Engine`. Return real values from `Store()`, `Plugins()`, `Hooks()`, `Logger()` and `GetUser()`, and zero values from the rest. Copy the method set from `plugin/plugin.go:52`. `Plugins()` is the one that matters here: it must return the registry built above, or no hook fires and the test fails for the wrong reason.

Now create `plugins/agentauth/issue.go`:

```go
package agentauth

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
)

// agentSessionTTL bounds an agent access token. Short by design: the refresh
// path re-checks the grant, so a short session is what makes revocation and
// expiry take effect quickly.
const agentSessionTTL = 15 * time.Minute

// IssueAgentSession mints a session for an agent acting under grant.
//
// Issuance deliberately goes through the engine's normal store and hook path
// so BeforeSessionCreate and AfterSignIn fire. That is what puts agent traffic
// in front of riskengine, impossibletravel, ipreputation and the rest. The API
// key plugin builds a synthetic session by hand and fires neither, which is
// why its traffic is invisible to all of them. Do not copy that shape here.
func (p *Plugin) IssueAgentSession(ctx context.Context, grant *AgentGrant) (*session.Session, error) {
	now := time.Now()
	if !grant.IsActive(now) {
		return nil, ErrGrantInactive
	}

	expires := now.Add(agentSessionTTL)
	if expires.After(grant.ExpiresAt) {
		// A session must never outlive the grant that authorized it.
		expires = grant.ExpiresAt
	}

	sess := &session.Session{
		ID:            id.NewSessionID(),
		AppID:         grant.AppID,
		UserID:        grant.UserID,
		OrgID:         grant.OrgID,
		PrincipalKind: session.PrincipalKindAgent,
		AgentID:       grant.AgentID,
		GrantID:       grant.ID,
		ExpiresAt:     expires,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// The Emit* methods live on *plugin.Registry, reached through
	// engine.Plugins(). engine.Hooks() returns a *hook.Bus, which is a
	// different type and has no Emit methods on it.
	reg := p.engine.Plugins()

	// BeforeSessionCreate returns an error and can veto. riskengine uses it to
	// block a session outright when the score is high enough.
	if err := reg.EmitBeforeSessionCreate(ctx, sess); err != nil {
		return nil, fmt.Errorf("agentauth: before session create: %w", err)
	}
	if err := p.engine.Store().CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("agentauth: create session: %w", err)
	}

	u, err := p.engine.GetUser(ctx, grant.UserID)
	if err != nil {
		return nil, fmt.Errorf("agentauth: resolve delegating user: %w", err)
	}
	// EmitAfterSignIn returns nothing. Notification hooks are fire-and-forget
	// by design, so there is no error to handle here.
	reg.EmitAfterSignIn(ctx, u, sess)

	stamp := now
	grant.LastUsedAt = &stamp
	if err := p.store.UpdateAgentGrant(ctx, grant); err != nil {
		p.logger.Warn("agentauth: could not stamp grant last-used", log.Error(err))
	}

	return sess, nil
}
```

Note the asymmetry, since it is easy to get wrong: `EmitBeforeSessionCreate` returns an `error` (`plugin/registry.go:441`) and `EmitAfterSignIn` returns nothing (`plugin/registry.go:348`). Writing `if err := reg.EmitAfterSignIn(...)` will not compile.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -run TestIssueAgentSession -v`
Expected: PASS, four tests.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/issue.go plugins/agentauth/issue_test.go
git commit -m "feat(agentauth): issue agent sessions through the hook path"
```

---

### Task 10: Grant cache

**Files:**
- Create: `plugins/agentauth/cache.go`
- Modify: `plugins/agentauth/middleware.go` (route the grant read through the cache)
- Test: `plugins/agentauth/cache_test.go`

**Interfaces:**
- Consumes: `AgentGrant`, `Store` (Task 2).
- Produces: `agentauth.grantCache` (unexported) with `newGrantCache(ttl time.Duration) *grantCache`, `get(grantID id.AgentGrantID) (*AgentGrant, bool)`, `put(g *AgentGrant)`, `invalidate(grantID id.AgentGrantID)`.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/cache_test.go`. Note this is an internal test, package `agentauth`, because the cache is unexported:

```go
package agentauth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

func TestGrantCache_HitAndMiss(t *testing.T) {
	c := newGrantCache(time.Minute)
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}

	_, ok := c.get(g.ID)
	assert.False(t, ok, "an empty cache must miss")

	c.put(g)
	got, ok := c.get(g.ID)
	require.True(t, ok)
	assert.Equal(t, g.ID.String(), got.ID.String())
}

func TestGrantCache_EntryExpires(t *testing.T) {
	c := newGrantCache(10 * time.Millisecond)
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}
	c.put(g)

	time.Sleep(20 * time.Millisecond)

	_, ok := c.get(g.ID)
	assert.False(t, ok, "a cache entry past its ttl must miss")
}

func TestGrantCache_Invalidate(t *testing.T) {
	c := newGrantCache(time.Minute)
	g := &AgentGrant{ID: id.NewAgentGrantID(), ExpiresAt: time.Now().Add(time.Hour)}
	c.put(g)

	c.invalidate(g.ID)

	_, ok := c.get(g.ID)
	assert.False(t, ok)
}

// Revocation must be visible immediately, not after the ttl. Session deletion
// is the primary invalidation point, but an explicit invalidate on revoke is
// what makes single-node behavior exact.
func TestAuthorize_RevocationBeatsTheCache(t *testing.T) {
	store := NewMemoryStore()
	p := New(WithStore(store), WithScope("invoices:read", Grants("read", "invoice")))
	userID := id.NewUserID()
	g := &AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
		UserID: userID, OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(t.Context(), g))
	p.SetPermissionChecker(allowAll{})
	sess := agentSession(g)
	require.NoError(t, p.Authorize(t.Context(), sess, "read", "invoice")) // warms the cache

	require.NoError(t, p.RevokeGrant(t.Context(), g.ID))

	require.ErrorIs(t, p.Authorize(t.Context(), sess, "read", "invoice"), ErrGrantInactive)
}
```

Add the small internal helpers `allowAll` (a `plugin.PermissionChecker` returning true) and `agentSession(g *AgentGrant) *session.Session` at the bottom of this file.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run TestGrantCache -v`
Expected: FAIL, compile error `undefined: newGrantCache`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/agentauth/cache.go`:

```go
package agentauth

import (
	"sync"
	"time"

	"github.com/xraph/authsome/id"
)

// grantCacheTTL bounds how stale a cached grant may be.
//
// Revocation does not rely on this. Revoking a grant deletes the sessions it
// issued, and session resolution runs on every request before the grant is
// ever consulted, so a stale entry is unreachable. What the ttl actually
// covers is the slower stuff: a grant ageing past ExpiresAt, or an org
// flipping its policy to blocked. Multi-node deployments inherit the same
// bounded staleness.
const grantCacheTTL = 10 * time.Second

type cachedGrant struct {
	grant    *AgentGrant
	cachedAt time.Time
}

type grantCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]cachedGrant
}

func newGrantCache(ttl time.Duration) *grantCache {
	return &grantCache{ttl: ttl, entries: make(map[string]cachedGrant)}
}

func (c *grantCache) get(grantID id.AgentGrantID) (*AgentGrant, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[grantID.String()]
	if !ok || time.Since(e.cachedAt) > c.ttl {
		return nil, false
	}
	cp := *e.grant
	return &cp, true
}

func (c *grantCache) put(g *AgentGrant) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := *g
	c.entries[g.ID.String()] = cachedGrant{grant: &cp, cachedAt: time.Now()}
}

func (c *grantCache) invalidate(grantID id.AgentGrantID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, grantID.String())
}
```

Add `cache *grantCache` to the `Plugin` struct and initialize it in `New` with `newGrantCache(grantCacheTTL)`.

In `middleware.go`, replace the direct store read in `Authorize` with a cache-first read:

```go
	grant, ok := p.cache.get(sess.GrantID)
	if !ok {
		loaded, err := p.store.GetAgentGrant(ctx, sess.GrantID)
		if errors.Is(err, ErrNotFound) {
			return ErrGrantInactive
		}
		if err != nil {
			return fmt.Errorf("agentauth: load grant: %w", err)
		}
		p.cache.put(loaded)
		grant = loaded
	}
```

Add the revocation entry point to `grant.go`:

```go
// RevokeGrant revokes a delegation and drops it from the cache, so the next
// request sees the revocation without waiting out the cache ttl.
func (p *Plugin) RevokeGrant(ctx context.Context, grantID id.AgentGrantID) error {
	if err := p.store.RevokeAgentGrant(ctx, grantID); err != nil {
		return fmt.Errorf("agentauth: revoke grant: %w", err)
	}
	p.cache.invalidate(grantID)
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -v`
Expected: PASS, the whole package including every earlier task's tests.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/cache.go plugins/agentauth/cache_test.go plugins/agentauth/middleware.go plugins/agentauth/grant.go
git commit -m "feat(agentauth): cache grants with explicit invalidation on revoke"
```

---

### Task 11: Offboarding lifecycle

Read this task fully before starting. The member-removal hook as it exists today cannot support org-scoped revocation, so this task widens it first.

`OnAfterMemberRemove(ctx, memberID)` carries only a `MemberID`, and `plugins/organization/service.go:140` calls `DeleteMember` before emitting. By the time any plugin sees the hook, the member row is gone and there is no way to learn which user left which org. So this task adds a `BeforeMemberRemove` hook carrying the full member, matching the `BeforeUserDelete` precedent already in `plugin/plugin.go:246`.

**Files:**
- Modify: `plugin/plugin.go` (add the `BeforeMemberRemove` interface)
- Modify: `plugin/registry.go` (add `EmitBeforeMemberRemove`)
- Modify: `plugins/organization/service.go:139-145`
- Create: `plugins/agentauth/lifecycle.go`
- Test: `plugins/agentauth/lifecycle_test.go`

**Interfaces:**
- Consumes: `Store` revocation methods (Task 2); `Plugin` (Task 4); `RevokeGrant` (Task 10).
- Produces: `plugin.BeforeMemberRemove` with `OnBeforeMemberRemove(ctx context.Context, m *organization.Member) error`; `(*Registry).EmitBeforeMemberRemove(ctx context.Context, m *organization.Member) error`; and on the plugin, `OnAfterUserUpdate`, `OnAfterUserDelete`, `OnBeforeMemberRemove`.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/lifecycle_test.go`:

```go
package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/user"
)

func grantFor(t *testing.T, s agentauth.Store, userID id.UserID, orgID id.OrgID) *agentauth.AgentGrant {
	t.Helper()
	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
		UserID: userID, OrgID: orgID, Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, s.CreateAgentGrant(context.Background(), g))
	return g
}

func assertRevoked(t *testing.T, s agentauth.Store, gid id.AgentGrantID) {
	t.Helper()
	g, err := s.GetAgentGrant(context.Background(), gid)
	require.NoError(t, err)
	assert.NotNil(t, g.RevokedAt, "grant %s should be revoked", gid)
}

// Banning a user must disarm every agent acting for them. Killing sessions
// alone is not enough: the next refresh would mint a fresh one, because the
// grant is what keeps authorizing.
func TestOnAfterUserUpdate_BannedUserLosesGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	userID := id.NewUserID()
	g := grantFor(t, store, userID, id.NewOrgID())

	require.NoError(t, p.OnAfterUserUpdate(context.Background(), &user.User{ID: userID, Banned: true}))

	assertRevoked(t, store, g.ID)
}

func TestOnAfterUserUpdate_ActiveUserKeepsGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	userID := id.NewUserID()
	g := grantFor(t, store, userID, id.NewOrgID())

	require.NoError(t, p.OnAfterUserUpdate(context.Background(), &user.User{ID: userID, Banned: false}))

	got, err := store.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.Nil(t, got.RevokedAt, "an ordinary profile update must not disarm agents")
}

func TestOnAfterUserDelete_RevokesGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	userID := id.NewUserID()
	g := grantFor(t, store, userID, id.NewOrgID())

	require.NoError(t, p.OnAfterUserDelete(context.Background(), userID))

	assertRevoked(t, store, g.ID)
}

// Leaving one org must not disarm the user's agents in their other orgs.
func TestOnBeforeMemberRemove_RevokesOnlyThatOrg(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	userID := id.NewUserID()
	leaving, staying := id.NewOrgID(), id.NewOrgID()
	gone := grantFor(t, store, userID, leaving)
	kept := grantFor(t, store, userID, staying)

	require.NoError(t, p.OnBeforeMemberRemove(context.Background(), &organization.Member{
		ID: id.NewMemberID(), OrgID: leaving, UserID: userID,
	}))

	assertRevoked(t, store, gone.ID)
	survivor, err := store.GetAgentGrant(context.Background(), kept.ID)
	require.NoError(t, err)
	assert.Nil(t, survivor.RevokedAt, "grants in the user's other orgs must survive")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run "TestOnAfter|TestOnBefore" -v`
Expected: FAIL, compile error `p.OnAfterUserUpdate undefined`.

- [ ] **Step 3: Write minimal implementation**

In `plugin/plugin.go`, add next to `AfterMemberRemove`:

```go
// BeforeMemberRemove is called before a member is removed from an
// organization, while the member record can still be read. AfterMemberRemove
// carries only the id and fires after deletion, so a plugin that needs to know
// which user left which org has to use this hook.
type BeforeMemberRemove interface {
	OnBeforeMemberRemove(ctx context.Context, m *organization.Member) error
}
```

In `plugin/registry.go`, add `EmitBeforeMemberRemove` following the exact shape of the existing `EmitAfterMemberRemove`, including the same type-cache registration.

In `plugins/organization/service.go`, rewrite `RemoveMember` so the member is resolved before deletion:

```go
// RemoveMember removes a member from an organization.
func (p *Plugin) RemoveMember(ctx context.Context, memberID id.MemberID) error {
	// Resolved before the delete so BeforeMemberRemove subscribers can see who
	// is leaving which org. AfterMemberRemove carries only the id, which is
	// not enough to revoke anything scoped to the pair.
	member, err := p.store.GetMember(ctx, memberID)
	if err != nil {
		return fmt.Errorf("organization: remove member: %w", err)
	}
	if err := p.plugins.EmitBeforeMemberRemove(ctx, member); err != nil {
		return fmt.Errorf("organization: remove member: %w", err)
	}
	if err := p.store.DeleteMember(ctx, memberID); err != nil {
		return fmt.Errorf("organization: remove member: %w", err)
	}
	p.plugins.EmitAfterMemberRemove(ctx, memberID)
	return nil
}
```

Create `plugins/agentauth/lifecycle.go`:

```go
package agentauth

import (
	"context"
	"fmt"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/organization"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.AfterUserUpdate    = (*Plugin)(nil)
	_ plugin.AfterUserDelete    = (*Plugin)(nil)
	_ plugin.BeforeMemberRemove = (*Plugin)(nil)
)

// OnAfterUserUpdate revokes every grant a user issued once they are banned.
// Sessions die separately through DeleteUserSessions; the grant has to go too,
// or the next refresh mints a replacement session.
func (p *Plugin) OnAfterUserUpdate(ctx context.Context, u *user.User) error {
	if u == nil || !u.Banned {
		return nil
	}
	return p.revokeUserGrants(ctx, u.ID)
}

// OnAfterUserDelete revokes every grant a deleted user issued.
func (p *Plugin) OnAfterUserDelete(ctx context.Context, userID id.UserID) error {
	return p.revokeUserGrants(ctx, userID)
}

// OnBeforeMemberRemove revokes the leaving member's grants in that org only.
// Their grants in other orgs are untouched.
func (p *Plugin) OnBeforeMemberRemove(ctx context.Context, m *organization.Member) error {
	if m == nil {
		return nil
	}
	grants, err := p.store.ListGrantsByUser(ctx, m.UserID)
	if err != nil {
		return fmt.Errorf("agentauth: list grants: %w", err)
	}
	if err := p.store.RevokeGrantsByUserOrg(ctx, m.UserID, m.OrgID); err != nil {
		return fmt.Errorf("agentauth: revoke org grants: %w", err)
	}
	for _, g := range grants {
		if g.OrgID.String() == m.OrgID.String() {
			p.cache.invalidate(g.ID)
		}
	}
	return nil
}

// revokeUserGrants revokes all of a user's grants and clears them from cache.
func (p *Plugin) revokeUserGrants(ctx context.Context, userID id.UserID) error {
	grants, err := p.store.ListGrantsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("agentauth: list grants: %w", err)
	}
	if err := p.store.RevokeGrantsByUser(ctx, userID); err != nil {
		return fmt.Errorf("agentauth: revoke user grants: %w", err)
	}
	for _, g := range grants {
		p.cache.invalidate(g.ID)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ ./plugins/organization/ ./plugin/ -v`
Expected: PASS. The organization suite must show no regressions from the `RemoveMember` rewrite.

- [ ] **Step 5: Commit**

```bash
git add plugin/plugin.go plugin/registry.go plugins/organization/service.go plugins/agentauth/lifecycle.go plugins/agentauth/lifecycle_test.go
git commit -m "feat(agentauth): revoke grants on ban, delete and org departure"
```

---

### Task 12: User and admin HTTP surfaces

**Files:**
- Create: `plugins/agentauth/handlers.go`
- Modify: `plugins/agentauth/plugin.go` (add `RegisterRoutes`)
- Test: `plugins/agentauth/handlers_test.go`

**Interfaces:**
- Consumes: everything from Tasks 2, 4, 7, 10, 11.
- Produces: `(*Plugin).RegisterRoutes(router forge.Router) error` satisfying `plugin.RouteProvider`, and the response types `agentauth.GrantView` with fields `ID string`, `AgentID string`, `AgentName string`, `Scopes []string`, `ExpiresAt time.Time`, `LastUsedAt *time.Time`; plus `agentauth.ListGrantsResponse` with `Grants []GrantView`.

Routes:

| Method | Path | Guard |
|---|---|---|
| `GET` | `/v1/me/agents` | `plugin.SessionGuard` |
| `DELETE` | `/v1/me/agents/:id` | `plugin.SessionGuard` |
| `GET` | `/v1/admin/agents` | `plugin.AdminGuard(engine, "read", "agent")` |
| `POST` | `/v1/admin/agents` | `plugin.AdminGuard(engine, "write", "agent")` |
| `PATCH` | `/v1/admin/agents/:id/status` | `plugin.AdminGuard(engine, "write", "agent")` |
| `PUT` | `/v1/admin/agents/policy` | `plugin.AdminGuard(engine, "write", "agent")` |

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/handlers_test.go`:

```go
package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
)

// A user must be able to see every agent acting for them, and only theirs.
func TestListMyGrants_ShowsOnlyTheCallersGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	me, someoneElse := id.NewUserID(), id.NewUserID()
	agent := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: id.NewAppID(), ClientID: "client_x", Name: "Invoice Reader",
		Origin: agentauth.OriginSelfRegistered, Status: agentauth.StatusApproved,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgent(context.Background(), agent))
	mine := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: agent.AppID, AgentID: agent.ID, UserID: me,
		OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	theirs := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: agent.AppID, AgentID: agent.ID, UserID: someoneElse,
		OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), mine))
	require.NoError(t, store.CreateAgentGrant(context.Background(), theirs))

	resp, err := p.ListMyGrants(context.Background(), me)

	require.NoError(t, err)
	require.Len(t, resp.Grants, 1)
	assert.Equal(t, mine.ID.String(), resp.Grants[0].ID)
	assert.Equal(t, "Invoice Reader", resp.Grants[0].AgentName)
}

// A user must not be able to revoke somebody else's delegation.
func TestRevokeMyGrant_RefusesAnotherUsersGrant(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	me, owner := id.NewUserID(), id.NewUserID()
	theirs := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(), UserID: owner,
		OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), theirs))

	err := p.RevokeMyGrant(context.Background(), me, theirs.ID)

	require.Error(t, err)
	got, gerr := store.GetAgentGrant(context.Background(), theirs.ID)
	require.NoError(t, gerr)
	assert.Nil(t, got.RevokedAt, "the grant must survive an unauthorized revoke attempt")
}

func TestRevokeMyGrant_RevokesOwnGrant(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	me := id.NewUserID()
	mine := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(), UserID: me,
		OrgID: id.NewOrgID(), Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), mine))

	require.NoError(t, p.RevokeMyGrant(context.Background(), me, mine.ID))

	got, err := store.GetAgentGrant(context.Background(), mine.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.RevokedAt)
}

// Blocking an agent must disarm it across the org in one action.
func TestBlockAgent_RevokesItsGrants(t *testing.T) {
	store := agentauth.NewMemoryStore()
	p := agentauth.New(agentauth.WithStore(store))
	org := id.NewOrgID()
	agent := &agentauth.Agent{
		ID: id.NewAgentID(), AppID: id.NewAppID(), OrgID: org, ClientID: "client_bad",
		Name: "Rogue", Origin: agentauth.OriginSelfRegistered, Status: agentauth.StatusApproved,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgent(context.Background(), agent))
	g := &agentauth.AgentGrant{
		ID: id.NewAgentGrantID(), AppID: agent.AppID, AgentID: agent.ID, UserID: id.NewUserID(),
		OrgID: org, Scopes: []string{"invoices:read"},
		ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, store.CreateAgentGrant(context.Background(), g))

	require.NoError(t, p.SetAgentStatus(context.Background(), agent.ID, org, agentauth.StatusBlocked))

	got, err := store.GetAgentGrant(context.Background(), g.ID)
	require.NoError(t, err)
	assert.NotNil(t, got.RevokedAt, "blocking an agent must revoke the grants it holds")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run "TestListMyGrants|TestRevokeMyGrant|TestBlockAgent" -v`
Expected: FAIL, compile error `p.ListMyGrants undefined`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/agentauth/handlers.go`:

```go
package agentauth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
)

// Compile-time interface check.
var _ plugin.RouteProvider = (*Plugin)(nil)

// GrantView is one delegation as the granting user sees it.
type GrantView struct {
	ID         string     `json:"id"`
	AgentID    string     `json:"agent_id"`
	AgentName  string     `json:"agent_name"`
	Scopes     []string   `json:"scopes"`
	ExpiresAt  time.Time  `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// ListGrantsResponse is the /v1/me/agents payload.
type ListGrantsResponse struct {
	Grants []GrantView `json:"grants"`
}

// ListMyGrants returns every active delegation the user has issued.
func (p *Plugin) ListMyGrants(ctx context.Context, userID id.UserID) (*ListGrantsResponse, error) {
	grants, err := p.store.ListGrantsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("agentauth: list grants: %w", err)
	}
	now := time.Now()
	out := &ListGrantsResponse{Grants: []GrantView{}}
	for _, g := range grants {
		if !g.IsActive(now) {
			continue
		}
		view := GrantView{
			ID: g.ID.String(), AgentID: g.AgentID.String(),
			Scopes: g.Scopes, ExpiresAt: g.ExpiresAt, LastUsedAt: g.LastUsedAt,
		}
		if a, aerr := p.store.GetAgent(ctx, g.AgentID); aerr == nil {
			view.AgentName = a.Name
		}
		out.Grants = append(out.Grants, view)
	}
	return out, nil
}

// RevokeMyGrant revokes one of the caller's own delegations. It refuses a
// grant belonging to anybody else, so a grant id is not an authorization.
func (p *Plugin) RevokeMyGrant(ctx context.Context, userID id.UserID, grantID id.AgentGrantID) error {
	g, err := p.store.GetAgentGrant(ctx, grantID)
	if errors.Is(err, ErrNotFound) {
		return forge.NotFound("grant not found")
	}
	if err != nil {
		return fmt.Errorf("agentauth: load grant: %w", err)
	}
	if g.UserID.String() != userID.String() {
		// Same response as a missing grant, so the endpoint does not confirm
		// that somebody else's grant id exists.
		return forge.NotFound("grant not found")
	}
	return p.RevokeGrant(ctx, grantID)
}

// SetAgentStatus changes an agent's approval state. Blocking also revokes
// every grant the agent holds in that org, since leaving them live would mean
// a blocked agent kept working until its sessions aged out.
func (p *Plugin) SetAgentStatus(ctx context.Context, agentID id.AgentID, orgID id.OrgID, status AgentStatus) error {
	a, err := p.store.GetAgent(ctx, agentID)
	if errors.Is(err, ErrNotFound) {
		return forge.NotFound("agent not found")
	}
	if err != nil {
		return fmt.Errorf("agentauth: load agent: %w", err)
	}

	a.Status = status
	a.UpdatedAt = time.Now()
	if err := p.store.UpdateAgent(ctx, a); err != nil {
		return fmt.Errorf("agentauth: update agent: %w", err)
	}

	if status != StatusBlocked {
		return nil
	}
	if err := p.store.RevokeGrantsByAgent(ctx, agentID, orgID); err != nil {
		return fmt.Errorf("agentauth: revoke agent grants: %w", err)
	}
	// The cache is keyed by grant id and the revoked grants are not enumerated
	// here, so clear it wholesale. Blocking an agent is a rare admin action,
	// so the cost of a cold cache does not matter.
	p.cache = newGrantCache(grantCacheTTL)
	return nil
}
```

Add `RegisterRoutes` to `plugins/agentauth/plugin.go`, following the group and guard pattern in `plugins/consent/plugin.go`:

```go
// RegisterRoutes registers the user and admin surfaces.
func (p *Plugin) RegisterRoutes(router forge.Router) error {
	me := router.Group(p.basePath+"/me/agents",
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.SessionGuard(p.engine)...),
	)
	me.GET("", p.handleListMyGrants, forge.WithSummary("List agents acting on my behalf"))
	me.DELETE("/:id", p.handleRevokeMyGrant, forge.WithSummary("Revoke an agent's delegation"))

	admin := router.Group(p.basePath+"/admin/agents",
		forge.WithGroupAuth("session"),
		forge.WithGroupMiddleware(plugin.AdminGuard(p.engine, "read", "agent")...),
	)
	admin.GET("", p.handleListAgents, forge.WithSummary("List registered agents"))
	admin.POST("", p.handleRegisterAgent,
		append([]forge.RouteOption{forge.WithSummary("Register an agent")},
			plugin.PermissionRouteOptions(p.engine, "write", "agent")...)...)
	admin.PATCH("/:id/status", p.handleSetAgentStatus,
		append([]forge.RouteOption{forge.WithSummary("Approve or block an agent")},
			plugin.PermissionRouteOptions(p.engine, "write", "agent")...)...)
	admin.PUT("/policy", p.handlePutOrgPolicy,
		append([]forge.RouteOption{forge.WithSummary("Set the org's agent delegation policy")},
			plugin.PermissionRouteOptions(p.engine, "write", "agent")...)...)
	return nil
}
```

Write the six thin `handleX` methods in `handlers.go`. Each resolves the caller from `middleware` context, parses path params with the `id.ParseAgentGrantID` / `id.ParseAgentID` helpers from Task 1, and delegates to the service methods above. Follow the handler signatures used in `plugins/consent/plugin.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -v`
Expected: PASS, whole package.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/handlers.go plugins/agentauth/handlers_test.go plugins/agentauth/plugin.go
git commit -m "feat(agentauth): add user grant listing and admin agent management"
```

---

### Task 13: Persistent stores and migrations

**Files:**
- Create: `plugins/agentauth/store_models.go`, `store_postgres.go`, `store_sqlite.go`, `store_mongo.go`, `migrations.go`
- Test: `plugins/agentauth/store_conformance_test.go`

**Interfaces:**
- Consumes: the `Store` interface and all domain types (Task 2).
- Produces: `agentauth.NewPostgresStore(db *grove.DB) *PostgresStore`, `agentauth.NewSqliteStore(db *grove.DB) *SqliteStore`, `agentauth.NewMongoStore(db *grove.DB) *MongoStore`, `agentauth.PostgresMigrations`, `agentauth.SqliteMigrations`, and `(*Plugin).MigrationGroups(driverName string) []*migrate.Group`.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/store_conformance_test.go`. One suite, run against every backend, so a backend cannot drift:

```go
package agentauth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
)

// runStoreConformance is the single suite every backend must satisfy.
func runStoreConformance(t *testing.T, newStore func(t *testing.T) agentauth.Store) {
	t.Run("grant round trips", func(t *testing.T) {
		s := newStore(t)
		g := &agentauth.AgentGrant{
			ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
			UserID: id.NewUserID(), OrgID: id.NewOrgID(),
			Scopes: []string{"invoices:read", "invoices:write"},
			ExpiresAt: time.Now().Add(time.Hour).UTC(),
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, s.CreateAgentGrant(context.Background(), g))

		got, err := s.GetAgentGrant(context.Background(), g.ID)
		require.NoError(t, err)
		assert.Equal(t, g.UserID.String(), got.UserID.String())
		assert.ElementsMatch(t, g.Scopes, got.Scopes, "the scope list must survive serialization")
		assert.WithinDuration(t, g.ExpiresAt, got.ExpiresAt, time.Second)
	})

	t.Run("agent lookup by client id", func(t *testing.T) {
		s := newStore(t)
		a := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: id.NewAppID(), ClientID: "client_lookup",
			Name: "Lookup", Origin: agentauth.OriginSelfRegistered,
			Status: agentauth.StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, s.CreateAgent(context.Background(), a))

		got, err := s.GetAgentByClientID(context.Background(), "client_lookup")
		require.NoError(t, err)
		assert.Equal(t, a.ID.String(), got.ID.String())
	})

	t.Run("revoke by user org is scoped", func(t *testing.T) {
		s := newStore(t)
		user := id.NewUserID()
		leaving, staying := id.NewOrgID(), id.NewOrgID()
		mk := func(org id.OrgID) *agentauth.AgentGrant {
			g := &agentauth.AgentGrant{
				ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
				UserID: user, OrgID: org, Scopes: []string{"invoices:read"},
				ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}
			require.NoError(t, s.CreateAgentGrant(context.Background(), g))
			return g
		}
		gone, kept := mk(leaving), mk(staying)

		require.NoError(t, s.RevokeGrantsByUserOrg(context.Background(), user, leaving))

		g1, err := s.GetAgentGrant(context.Background(), gone.ID)
		require.NoError(t, err)
		assert.NotNil(t, g1.RevokedAt)
		g2, err := s.GetAgentGrant(context.Background(), kept.ID)
		require.NoError(t, err)
		assert.Nil(t, g2.RevokedAt)
	})

	t.Run("org policy round trips", func(t *testing.T) {
		s := newStore(t)
		org := id.NewOrgID()
		require.NoError(t, s.PutOrgPolicy(context.Background(), &agentauth.OrgAgentPolicy{
			OrgID: org, Mode: agentauth.ModeAllowlist,
			MaxGrantTTL: 48 * time.Hour, AllowedScopes: []string{"invoices:read"},
		}))

		got, err := s.GetOrgPolicy(context.Background(), org)
		require.NoError(t, err)
		assert.Equal(t, agentauth.ModeAllowlist, got.Mode)
		assert.Equal(t, 48*time.Hour, got.MaxGrantTTL, "a duration must survive the round trip")
		assert.Equal(t, []string{"invoices:read"}, got.AllowedScopes)
	})
}

func TestMemoryStore_Conformance(t *testing.T) {
	runStoreConformance(t, func(t *testing.T) agentauth.Store {
		return agentauth.NewMemoryStore()
	})
}

func TestSqliteStore_Conformance(t *testing.T) {
	runStoreConformance(t, func(t *testing.T) agentauth.Store {
		return agentauth.NewSqliteStore(newTestSqliteDB(t))
	})
}
```

Implement `newTestSqliteDB(t *testing.T) *grove.DB` in this file, opening an in-memory sqlite database and running `agentauth.SqliteMigrations`. Copy the setup used by an existing plugin's sqlite test.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run Conformance -v`
Expected: FAIL, compile error `undefined: agentauth.NewSqliteStore`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/agentauth/migrations.go`, following `plugins/consent/migrations.go` exactly:

```go
package agentauth

import (
	"context"

	"github.com/xraph/grove/migrate"
)

// PostgresMigrations is the postgres migration group for the agentauth plugin.
var PostgresMigrations = migrate.NewGroup("authsome-agentauth", migrate.DependsOn("authsome"))

// SqliteMigrations is the SQLite migration group for the agentauth plugin.
var SqliteMigrations = migrate.NewGroup("authsome-agentauth", migrate.DependsOn("authsome"))

func init() {
	PostgresMigrations.MustRegister(
		&migrate.Migration{
			Name:    "create_agentauth_tables",
			Version: "20260824000061",
			Up: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
CREATE TABLE IF NOT EXISTS authsome_agents (
    id          TEXT PRIMARY KEY,
    app_id      TEXT NOT NULL REFERENCES authsome_apps(id),
    org_id      TEXT NOT NULL DEFAULT '',
    client_id   TEXT NOT NULL,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    logo_uri    TEXT NOT NULL DEFAULT '',
    origin      TEXT NOT NULL,
    status      TEXT NOT NULL,
    created_by  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_agents_client_id
    ON authsome_agents (client_id);

CREATE TABLE IF NOT EXISTS authsome_agent_grants (
    id           TEXT PRIMARY KEY,
    app_id       TEXT NOT NULL REFERENCES authsome_apps(id),
    agent_id     TEXT NOT NULL REFERENCES authsome_agents(id),
    user_id      TEXT NOT NULL,
    org_id       TEXT NOT NULL DEFAULT '',
    scopes       TEXT NOT NULL DEFAULT '[]',
    consent_id   TEXT NOT NULL DEFAULT '',
    expires_at   TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Every grant names the human who authorized it. The intersection model
    -- has no meaning without one, so this is enforced rather than assumed.
    CONSTRAINT authsome_agent_grants_user_required CHECK (user_id <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_authsome_agent_grants_active
    ON authsome_agent_grants (agent_id, user_id, org_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_authsome_agent_grants_user
    ON authsome_agent_grants (user_id);

CREATE TABLE IF NOT EXISTS authsome_agent_policies (
    org_id         TEXT PRIMARY KEY,
    mode           TEXT NOT NULL,
    max_grant_ttl  BIGINT NOT NULL DEFAULT 0,
    allowed_scopes TEXT NOT NULL DEFAULT '[]',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`)
				return err
			},
			Down: func(ctx context.Context, exec migrate.Executor) error {
				_, err := exec.Exec(ctx, `
DROP TABLE IF EXISTS authsome_agent_policies;
DROP TABLE IF EXISTS authsome_agent_grants;
DROP TABLE IF EXISTS authsome_agents;
`)
				return err
			},
		},
	)

	// The SQLite group mirrors the above with TEXT timestamps, INTEGER for
	// max_grant_ttl, and no partial-index WHERE clause on the active-grant
	// index. Register it here following the same shape.
}

// MigrationGroups returns the agentauth migration groups for the driver.
func (p *Plugin) MigrationGroups(driverName string) []*migrate.Group {
	switch driverName {
	case "pg", "postgres":
		return []*migrate.Group{PostgresMigrations}
	case "sqlite", "sqlite3":
		return []*migrate.Group{SqliteMigrations}
	default:
		return nil
	}
}
```

Add `_ plugin.MigrationProvider = (*Plugin)(nil)` to the compile-time checks in `plugin.go`.

Create `store_models.go` with the row structs. `Scopes` and `AllowedScopes` serialize as JSON text; `MaxGrantTTL` stores as an integer count of nanoseconds so the duration round-trips exactly. Then create `store_postgres.go`, `store_sqlite.go` and `store_mongo.go` implementing every `Store` method, following the query style in `plugins/consent/store_postgres.go`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -v`
Expected: PASS. Both conformance suites green, with all four subtests each.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/
git commit -m "feat(agentauth): add persistent stores and migrations"
```

---

---

### Task 14: HTTP error mapping

Task 8 produces sentinel errors. This turns them into the responses the spec's error table specifies, including the `WWW-Authenticate` headers an agent developer needs to tell "re-authorize" apart from "ask for more scope".

**Files:**
- Create: `plugins/agentauth/errors.go`
- Test: `plugins/agentauth/errors_test.go`

**Interfaces:**
- Consumes: `ErrInsufficientScope`, `ErrGrantInactive`, `ErrNoPermissionChecker` (Task 8); `ScopeRegistry` (Task 3).
- Produces: `(*Plugin).AuthorizeHTTP(ctx context.Context, sess *session.Session, action, resource string) error`, which wraps `Authorize` and returns a forge HTTP error carrying the right status and headers.

- [ ] **Step 1: Write the failing test**

Create `plugins/agentauth/errors_test.go`:

```go
package agentauth_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/plugins/agentauth"
)

// An expired or revoked grant is 401, not 403. The distinction is what tells
// an agent to re-authorize instead of giving up.
func TestAuthorizeHTTP_InactiveGrantIs401(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(-time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})

	err := p.AuthorizeHTTP(context.Background(), sess, "read", "invoice")

	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, agentauth.StatusOf(err))
	assert.Contains(t, agentauth.HeaderOf(err, "WWW-Authenticate"), `error="invalid_token"`)
}

// A missing scope is 403 and must name the scope the agent needs, so a
// developer can fix their authorization request without guesswork.
func TestAuthorizeHTTP_MissingScopeIs403WithScopeNamed(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|write|invoice": true,
	}})

	err := p.AuthorizeHTTP(context.Background(), sess, "write", "invoice")

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, agentauth.StatusOf(err))
	header := agentauth.HeaderOf(err, "WWW-Authenticate")
	assert.Contains(t, header, `error="insufficient_scope"`)
	assert.Contains(t, header, `scope="invoices:write"`, "the response must name the scope that would satisfy the route")
}

// A user-gate failure must NOT name a scope. Reporting it the same way as a
// scope failure would let an agent enumerate its owner's permissions.
func TestAuthorizeHTTP_UserGateFailureIsOpaque(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{}})

	err := p.AuthorizeHTTP(context.Background(), sess, "read", "invoice")

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, agentauth.StatusOf(err))
	assert.NotContains(t, agentauth.HeaderOf(err, "WWW-Authenticate"), "insufficient_scope",
		"a user-gate failure must not be reported as a scope problem")
}

func TestAuthorizeHTTP_NoPermissionCheckerIs403(t *testing.T) {
	p, _, sess, _ := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))

	err := p.AuthorizeHTTP(context.Background(), sess, "read", "invoice")

	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, agentauth.StatusOf(err))
}

func TestAuthorizeHTTP_AllowedReturnsNil(t *testing.T) {
	p, _, sess, userID := agentSetup(t, []string{"invoices:read"}, time.Now().Add(time.Hour))
	p.SetPermissionChecker(&stubChecker{allow: map[string]bool{
		userID.String() + "|read|invoice": true,
	}})

	require.NoError(t, p.AuthorizeHTTP(context.Background(), sess, "read", "invoice"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./plugins/agentauth/ -run TestAuthorizeHTTP -v`
Expected: FAIL, compile error `p.AuthorizeHTTP undefined`.

- [ ] **Step 3: Write minimal implementation**

Create `plugins/agentauth/errors.go`:

```go
package agentauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/xraph/forge"

	"github.com/xraph/authsome/session"
)

// AuthorizeHTTP wraps Authorize and maps its sentinels onto the responses an
// agent developer can act on.
//
// The scope case names the scope that would satisfy the route, which is safe
// because it describes the route rather than the owner. The user-gate case
// stays opaque on purpose: reporting it in the same shape would let an agent
// enumerate its owner's permissions one request at a time.
func (p *Plugin) AuthorizeHTTP(ctx context.Context, sess *session.Session, action, resource string) error {
	err := p.Authorize(ctx, sess, action, resource)
	switch {
	case err == nil:
		return nil

	case errors.Is(err, ErrGrantInactive):
		return forge.NewHTTPError(http.StatusUnauthorized, "agent grant is no longer valid").
			WithHeader("WWW-Authenticate",
				`Bearer error="invalid_token", error_description="agent grant revoked or expired"`)

	case errors.Is(err, ErrInsufficientScope):
		return forge.NewHTTPError(http.StatusForbidden, "insufficient scope").
			WithHeader("WWW-Authenticate", fmt.Sprintf(
				`Bearer error="insufficient_scope", scope=%q`, p.scopeFor(action, resource)))

	case errors.Is(err, ErrNoPermissionChecker):
		// Fail closed. Do not tell the caller why: an RBAC outage is not an
		// agent's business, and the answer is the same either way.
		return forge.Forbidden("insufficient permissions")

	default:
		// The user gate refused. Same body as any other RBAC refusal, matching
		// middleware/rbac.go:62.
		return forge.Forbidden("insufficient permissions")
	}
}

// scopeFor returns the registered scope that confers a permission, so the
// insufficient_scope response can name it. Falls back to "action:resource"
// when the host app registered no scope for it.
func (p *Plugin) scopeFor(action, resource string) string {
	p.scopes.mu.RLock()
	defer p.scopes.mu.RUnlock()
	for scope, perm := range p.scopes.scopes {
		if perm.Action == action && perm.Resource == resource {
			return scope
		}
	}
	return action + ":" + resource
}
```

Add the two test-facing helpers to the same file so the tests can assert without reaching into forge internals:

```go
// StatusOf returns the HTTP status an error carries, or 0.
func StatusOf(err error) int { ... }

// HeaderOf returns a response header the error carries, or "".
func HeaderOf(err error, name string) string { ... }
```

Implement both against forge's HTTP error type. Check the constructor and header API in `forge` before writing these: if `forge.NewHTTPError(...).WithHeader(...)` does not exist, use whatever forge provides for a status-plus-header error and adjust the two helpers to match. The behavior the tests assert is the contract; the constructor is not.

Finally, wire `AuthorizeHTTP` into the guard the plugin exposes to host apps, so a route declaring a permission gets agent enforcement automatically. Add to `middleware.go`:

```go
// Guard returns middleware enforcing the agent intersection for a route that
// requires action on resource. A human session passes through untouched.
func (p *Plugin) Guard(action, resource string) forge.Middleware { ... }
```

Follow the middleware shape in `middleware/rbac.go:77` for resolving the session from context.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./plugins/agentauth/ -v`
Expected: PASS, whole package.

- [ ] **Step 5: Commit**

```bash
git add plugins/agentauth/errors.go plugins/agentauth/errors_test.go plugins/agentauth/middleware.go
git commit -m "feat(agentauth): map authorization failures to scoped http responses"
```


## Wiring it up

After Task 13, a host app enables the plugin like this. Add this snippet to `README.md` under Plugins as part of Task 13's commit:

```go
agents := agentauth.New(
    agentauth.WithScope("invoices:read",  agentauth.Grants("read",  "invoice")),
    agentauth.WithScope("invoices:write", agentauth.Grants("write", "invoice")),
    agentauth.WithDefaultGrantTTL(90 * 24 * time.Hour),
)

engine, err := authsome.NewEngine(
    authsome.WithStore(store),
    authsome.WithWarden(wardenEngine),
    authsome.WithPlugin(oauth2provider.New(oauth2provider.Config{
        Issuer:       "https://auth.example.com",
        ConsentGate:  agents,
    })),
    authsome.WithPlugin(agents),
)
```

Build `agents` before the provider, since the provider's `Config` holds the gate. If your wiring makes that awkward, construct the provider first and call `provider.SetConsentGate(agents)` instead.

## Deferred, and why

- **Refresh honoring `grant.ExpiresAt`.** The spec requires it. Agent sessions are capped at the grant expiry in Task 9, so a session can never outlive its grant, but the refresh endpoint itself still needs the check. It lives in the core refresh path rather than this plugin, so it is a separate change against `service.go`. Do it immediately after Task 13 and do not ship without it.
- **Self-registration through RFC 7591.** Task 12 ships admin registration only. Self-registration depends on the dynamic client registration work happening on its own branch. Org-registered agents are a complete, usable v1 without it.
- **Dashboard contract surface.** Every other plugin ships one. Add it once the API stabilizes.
