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
