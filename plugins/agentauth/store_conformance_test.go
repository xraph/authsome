package agentauth_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/grove"
	"github.com/xraph/grove/drivers/sqlitedriver"
	_ "github.com/xraph/grove/drivers/sqlitedriver/sqlitemigrate"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/agentauth"
	sqlitestore "github.com/xraph/authsome/store/sqlite"
)

// runStoreConformance is the single suite every agentauth.Store backend must
// satisfy, memory included, so a backend can't silently drift from the
// others. newStore is called once per subtest below.
//
// Contract: newStore need not return a distinct physical store per call —
// the Postgres and Mongo variants share one store across every subtest, to
// avoid paying for a fresh container/database per subtest — but it MUST
// return a store whose visible state is isolated from every other call's,
// which subtests achieve by keying everything they write with fresh random
// ids (id.New*ID()) rather than fixed literals. A subtest that needs a
// human-readable literal (a ClientID, say) must still make it unique per
// call — see "dup-client-"+id.NewAgentID().String() below — precisely
// because the backing store may be shared and long-lived across an entire
// test binary run, not because ids are pretty.
func runStoreConformance(t *testing.T, newStore func(t *testing.T) agentauth.Store) {
	ctx := context.Background()

	t.Run("grant round trips", func(t *testing.T) {
		s := newStore(t)
		g := &agentauth.AgentGrant{
			ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
			UserID: id.NewUserID(), OrgID: id.NewOrgID(),
			Scopes:    []string{"invoices:read", "invoices:write"},
			ExpiresAt: time.Now().Add(time.Hour).UTC(),
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
		require.NoError(t, s.CreateAgentGrant(ctx, g))

		got, err := s.GetAgentGrant(ctx, g.ID)
		require.NoError(t, err)
		assert.Equal(t, g.UserID.String(), got.UserID.String())
		assert.Equal(t, g.AppID.String(), got.AppID.String())
		assert.Equal(t, g.AgentID.String(), got.AgentID.String())
		assert.Equal(t, g.OrgID.String(), got.OrgID.String())
		assert.ElementsMatch(t, g.Scopes, got.Scopes, "the scope list must survive serialization")
		assert.WithinDuration(t, g.ExpiresAt, got.ExpiresAt, time.Second)
	})

	t.Run("agent lookup by client id", func(t *testing.T) {
		s := newStore(t)
		clientID := "client_lookup_" + id.NewAgentID().String()
		a := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: id.NewAppID(), ClientID: clientID,
			Name: "Lookup", Origin: agentauth.OriginSelfRegistered,
			Status: agentauth.StatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, s.CreateAgent(ctx, a))

		got, err := s.GetAgentByClientID(ctx, clientID)
		require.NoError(t, err)
		assert.Equal(t, a.ID.String(), got.ID.String())
		assert.Equal(t, a.Name, got.Name)
		assert.Equal(t, agentauth.OriginSelfRegistered, got.Origin)
		assert.Equal(t, agentauth.StatusPending, got.Status)
	})

	// CreateAgent must reject a duplicate ClientID atomically — a store that
	// only checked-then-inserted would race two concurrent registrations for
	// the same client into both landing. This can't observe the race
	// directly in a unit test, but it does prove the constraint exists at
	// all: on a store backed by a real check-then-insert bug, this test
	// would pass anyway (the second call still loses), so what it actually
	// guards is that ErrConflict is the error a backend author would reach
	// for FIRST, via a real uniqueness constraint rather than an
	// application-level check the next refactor could silently drop.
	t.Run("duplicate client id is rejected", func(t *testing.T) {
		s := newStore(t)
		appID := id.NewAppID()
		clientID := "dup-client-" + id.NewAgentID().String()
		first := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: appID, ClientID: clientID,
			Name: "First", Origin: agentauth.OriginSelfRegistered, Status: agentauth.StatusPending,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, s.CreateAgent(ctx, first))

		second := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: appID, ClientID: clientID,
			Name: "Second", Origin: agentauth.OriginSelfRegistered, Status: agentauth.StatusPending,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		err := s.CreateAgent(ctx, second)
		require.ErrorIs(t, err, agentauth.ErrConflict)

		// The loser must not have landed under a different id either.
		_, err = s.GetAgent(ctx, second.ID)
		assert.ErrorIs(t, err, agentauth.ErrNotFound)
	})

	// MemoryStore.CreateAgent only checks for a collision when a.ClientID !=
	// "" (see its comment), so two agents that both leave ClientID unset
	// must NOT collide — a bare (non-partial) unique index on client_id
	// would make this fail on SQL/mongo while memory allows it.
	t.Run("agents with unset client id do not collide", func(t *testing.T) {
		s := newStore(t)
		appID := id.NewAppID()
		a1 := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: appID, Name: "A1",
			Origin: agentauth.OriginFirstParty, Status: agentauth.StatusApproved,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		a2 := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: appID, Name: "A2",
			Origin: agentauth.OriginFirstParty, Status: agentauth.StatusApproved,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, s.CreateAgent(ctx, a1))
		require.NoError(t, s.CreateAgent(ctx, a2), "two agents with an unset client_id must not conflict")
	})

	t.Run("update agent persists and rejects unknown id", func(t *testing.T) {
		s := newStore(t)
		a := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: id.NewAppID(), ClientID: "client_update_" + id.NewAgentID().String(),
			Name: "Before", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusPending,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, s.CreateAgent(ctx, a))

		a.Status = agentauth.StatusApproved
		a.Name = "After"
		require.NoError(t, s.UpdateAgent(ctx, a))

		got, err := s.GetAgent(ctx, a.ID)
		require.NoError(t, err)
		assert.Equal(t, agentauth.StatusApproved, got.Status)
		assert.Equal(t, "After", got.Name)

		unknown := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: id.NewAppID(), Name: "Ghost",
			Origin: agentauth.OriginFirstParty, Status: agentauth.StatusPending,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		err = s.UpdateAgent(ctx, unknown)
		assert.ErrorIs(t, err, agentauth.ErrNotFound)
	})

	// I4: UpdateAgent must enforce the same ClientID collision rule
	// CreateAgent does — MemoryStore originally didn't (a plain replace,
	// no check), while all three SQL/mongo backends' unique index rejected
	// it unconditionally. Real drift a shared suite exists to catch.
	t.Run("update agent rejects retargeting onto another agent's client id", func(t *testing.T) {
		s := newStore(t)
		appID := id.NewAppID()
		heldClientID := "held_" + id.NewAgentID().String()
		held := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: appID, ClientID: heldClientID,
			Name: "Holder", Origin: agentauth.OriginSelfRegistered, Status: agentauth.StatusPending,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, s.CreateAgent(ctx, held))

		mover := &agentauth.Agent{
			ID: id.NewAgentID(), AppID: appID, ClientID: "mover_" + id.NewAgentID().String(),
			Name: "Mover", Origin: agentauth.OriginSelfRegistered, Status: agentauth.StatusPending,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, s.CreateAgent(ctx, mover))

		mover.ClientID = heldClientID
		err := s.UpdateAgent(ctx, mover)
		require.ErrorIs(t, err, agentauth.ErrConflict)

		// The holder's own record must be unaffected by the rejected update.
		gotHeld, err := s.GetAgent(ctx, held.ID)
		require.NoError(t, err)
		assert.Equal(t, heldClientID, gotHeld.ClientID)
	})

	t.Run("list agents is scoped to app, nil org lists all orgs", func(t *testing.T) {
		s := newStore(t)
		appA, appB := id.NewAppID(), id.NewAppID()
		orgX, orgY := id.NewOrgID(), id.NewOrgID()
		mk := func(app id.AppID, org id.OrgID) *agentauth.Agent {
			a := &agentauth.Agent{
				ID: id.NewAgentID(), AppID: app, OrgID: org, Name: "A",
				Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusApproved,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}
			require.NoError(t, s.CreateAgent(ctx, a))
			return a
		}
		x := mk(appA, orgX)
		y := mk(appA, orgY)
		_ = mk(appB, orgX) // different app: must never show up for appA queries

		all, err := s.ListAgents(ctx, appA, id.OrgID{})
		require.NoError(t, err)
		assert.Len(t, all, 2, "a nil org must list every org under the app")

		onlyX, err := s.ListAgents(ctx, appA, orgX)
		require.NoError(t, err)
		require.Len(t, onlyX, 1)
		assert.Equal(t, x.ID.String(), onlyX[0].ID.String())

		onlyY, err := s.ListAgents(ctx, appA, orgY)
		require.NoError(t, err)
		require.Len(t, onlyY, 1)
		assert.Equal(t, y.ID.String(), onlyY[0].ID.String())
	})

	t.Run("get agent grant not found", func(t *testing.T) {
		s := newStore(t)
		_, err := s.GetAgentGrant(ctx, id.NewAgentGrantID())
		assert.ErrorIs(t, err, agentauth.ErrNotFound)
	})

	t.Run("update agent grant persists and rejects unknown id", func(t *testing.T) {
		s := newStore(t)
		g := newGrant(t, id.NewUserID(), id.NewOrgID(), time.Now().Add(time.Hour))
		require.NoError(t, s.CreateAgentGrant(ctx, g))

		used := time.Now().Add(-time.Minute).UTC()
		g.LastUsedAt = &used
		require.NoError(t, s.UpdateAgentGrant(ctx, g))

		got, err := s.GetAgentGrant(ctx, g.ID)
		require.NoError(t, err)
		require.NotNil(t, got.LastUsedAt)
		assert.WithinDuration(t, used, *got.LastUsedAt, time.Second)

		ghost := newGrant(t, id.NewUserID(), id.NewOrgID(), time.Now().Add(time.Hour))
		err = s.UpdateAgentGrant(ctx, ghost)
		assert.ErrorIs(t, err, agentauth.ErrNotFound)
	})

	// Final review item 4: issue.go used to stamp LastUsedAt by copying a
	// grant read earlier in the request and writing the WHOLE row back
	// through UpdateAgentGrant. A revoke landing between that read and the
	// stamp write got silently undone: the stale copy's RevokedAt (nil) and
	// Scopes (pre-revocation) were both written back over the real
	// revocation. StampLastUsed must touch only last_used_at/updated_at, so
	// this ordering — revoke, THEN stamp with a grant reference captured
	// before the revoke — must leave the grant revoked.
	t.Run("stamp last used never resurrects a grant revoked in between", func(t *testing.T) {
		s := newStore(t)
		g := newGrant(t, id.NewUserID(), id.NewOrgID(), time.Now().Add(time.Hour))
		require.NoError(t, s.CreateAgentGrant(ctx, g))

		require.NoError(t, s.RevokeAgentGrant(ctx, g.ID))

		stampedAt := time.Now().UTC()
		require.NoError(t, s.StampLastUsed(ctx, g.ID, stampedAt))

		got, err := s.GetAgentGrant(ctx, g.ID)
		require.NoError(t, err)
		require.NotNil(t, got.RevokedAt, "a stamp landing after a revoke must never clear RevokedAt")
		require.NotNil(t, got.LastUsedAt)
		assert.WithinDuration(t, stampedAt, *got.LastUsedAt, time.Second)
		assert.ElementsMatch(t, g.Scopes, got.Scopes, "the stamp must never touch Scopes either")

		err = s.StampLastUsed(ctx, id.NewAgentGrantID(), time.Now())
		assert.ErrorIs(t, err, agentauth.ErrNotFound)
	})

	t.Run("revoke agent grant is idempotent and rejects unknown id", func(t *testing.T) {
		s := newStore(t)
		g := newGrant(t, id.NewUserID(), id.NewOrgID(), time.Now().Add(time.Hour))
		require.NoError(t, s.CreateAgentGrant(ctx, g))

		require.NoError(t, s.RevokeAgentGrant(ctx, g.ID))
		got, err := s.GetAgentGrant(ctx, g.ID)
		require.NoError(t, err)
		require.NotNil(t, got.RevokedAt)
		firstRevokedAt := *got.RevokedAt

		// Revoking again must be a no-op, not an error.
		require.NoError(t, s.RevokeAgentGrant(ctx, g.ID))
		got2, err := s.GetAgentGrant(ctx, g.ID)
		require.NoError(t, err)
		require.NotNil(t, got2.RevokedAt)
		assert.WithinDuration(t, firstRevokedAt, *got2.RevokedAt, time.Second,
			"re-revoking must not stamp a new RevokedAt")

		err = s.RevokeAgentGrant(ctx, id.NewAgentGrantID())
		assert.ErrorIs(t, err, agentauth.ErrNotFound)
	})

	t.Run("get active grant excludes revoked and expired and matches org exactly", func(t *testing.T) {
		s := newStore(t)
		agent := id.NewAgentID()
		user := id.NewUserID()
		org := id.NewOrgID()

		active := newGrant(t, user, org, time.Now().Add(time.Hour))
		active.AgentID = agent
		require.NoError(t, s.CreateAgentGrant(ctx, active))

		revokedGrant := newGrant(t, user, org, time.Now().Add(time.Hour))
		revokedGrant.AgentID = agent
		revokedGrant.UserID = id.NewUserID() // distinct user so it can't shadow active
		require.NoError(t, s.CreateAgentGrant(ctx, revokedGrant))
		require.NoError(t, s.RevokeAgentGrant(ctx, revokedGrant.ID))

		expired := newGrant(t, user, org, time.Now().Add(-time.Hour))
		expired.AgentID = agent
		expired.UserID = id.NewUserID()
		require.NoError(t, s.CreateAgentGrant(ctx, expired))

		got, err := s.GetActiveGrant(ctx, agent, user, org)
		require.NoError(t, err)
		assert.Equal(t, active.ID.String(), got.ID.String())

		_, err = s.GetActiveGrant(ctx, agent, revokedGrant.UserID, org)
		assert.ErrorIs(t, err, agentauth.ErrNotFound, "a revoked grant must never be returned as active")

		_, err = s.GetActiveGrant(ctx, agent, expired.UserID, org)
		assert.ErrorIs(t, err, agentauth.ErrNotFound, "an expired grant must never be returned as active")

		// GetActiveGrant's org matching is a plain equality, not the
		// nil-means-any-org semantics RevokeGrantsByAgent has: a query for a
		// different org must not find the grant even though it's active.
		_, err = s.GetActiveGrant(ctx, agent, user, id.NewOrgID())
		assert.ErrorIs(t, err, agentauth.ErrNotFound, "org must match exactly, not fall back to any org")
	})

	// M6: two active grants for the same agent+user+org triple are the
	// normal state, not a corrupt one — CreateGrant always inserts a fresh
	// grant on every consent rather than upserting over an existing active
	// one. GetActiveGrant must deterministically prefer the newest, the
	// same way on every backend, rather than letting each backend's
	// "whichever row a plain query happens to return first" differ.
	t.Run("get active grant is deterministic under duplicates: newest wins", func(t *testing.T) {
		s := newStore(t)
		agent := id.NewAgentID()
		user := id.NewUserID()
		org := id.NewOrgID()

		older := newGrant(t, user, org, time.Now().Add(time.Hour))
		older.AgentID = agent
		older.CreatedAt = time.Now().Add(-time.Hour).UTC()
		require.NoError(t, s.CreateAgentGrant(ctx, older))

		newer := newGrant(t, user, org, time.Now().Add(time.Hour))
		newer.AgentID = agent
		newer.CreatedAt = time.Now().UTC()
		require.NoError(t, s.CreateAgentGrant(ctx, newer))

		got, err := s.GetActiveGrant(ctx, agent, user, org)
		require.NoError(t, err)
		assert.Equal(t, newer.ID.String(), got.ID.String(), "the most recently created active grant must win")
	})

	t.Run("scopes round trip nil and empty consistently", func(t *testing.T) {
		s := newStore(t)
		nilScoped := &agentauth.AgentGrant{
			ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
			UserID: id.NewUserID(), OrgID: id.NewOrgID(),
			Scopes:    nil,
			ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		emptyScoped := &agentauth.AgentGrant{
			ID: id.NewAgentGrantID(), AppID: id.NewAppID(), AgentID: id.NewAgentID(),
			UserID: id.NewUserID(), OrgID: id.NewOrgID(),
			Scopes:    []string{},
			ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		require.NoError(t, s.CreateAgentGrant(ctx, nilScoped))
		require.NoError(t, s.CreateAgentGrant(ctx, emptyScoped))

		gotNil, err := s.GetAgentGrant(ctx, nilScoped.ID)
		require.NoError(t, err)
		assert.Empty(t, gotNil.Scopes)

		gotEmpty, err := s.GetAgentGrant(ctx, emptyScoped.ID)
		require.NoError(t, err)
		assert.Empty(t, gotEmpty.Scopes)
	})

	t.Run("reads return copies", func(t *testing.T) {
		s := newStore(t)
		g := newGrant(t, id.NewUserID(), id.NewOrgID(), time.Now().Add(time.Hour))
		require.NoError(t, s.CreateAgentGrant(ctx, g))

		got1, err := s.GetAgentGrant(ctx, g.ID)
		require.NoError(t, err)
		require.NotEmpty(t, got1.Scopes)
		got1.Scopes[0] = "mutated"

		got2, err := s.GetAgentGrant(ctx, g.ID)
		require.NoError(t, err)
		assert.NotEqual(t, "mutated", got2.Scopes[0], "mutating a returned grant must not corrupt stored state")

		org := id.NewOrgID()
		require.NoError(t, s.PutOrgPolicy(ctx, &agentauth.OrgAgentPolicy{
			OrgID: org, Mode: agentauth.ModeAllowlist, AllowedScopes: []string{"invoices:read"},
		}))
		p1, err := s.GetOrgPolicy(ctx, org)
		require.NoError(t, err)
		require.NotEmpty(t, p1.AllowedScopes)
		p1.AllowedScopes[0] = "mutated"

		p2, err := s.GetOrgPolicy(ctx, org)
		require.NoError(t, err)
		assert.NotEqual(t, "mutated", p2.AllowedScopes[0])
	})

	t.Run("revoke by user org is scoped", func(t *testing.T) {
		s := newStore(t)
		user := id.NewUserID()
		leaving, staying := id.NewOrgID(), id.NewOrgID()
		mk := func(org id.OrgID) *agentauth.AgentGrant {
			g := newGrant(t, user, org, time.Now().Add(time.Hour))
			require.NoError(t, s.CreateAgentGrant(ctx, g))
			return g
		}
		gone, kept := mk(leaving), mk(staying)

		revoked, err := s.RevokeGrantsByUserOrg(ctx, user, leaving)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{gone.ID.String()}, idStrings(revoked))

		g1, err := s.GetAgentGrant(ctx, gone.ID)
		require.NoError(t, err)
		assert.NotNil(t, g1.RevokedAt)
		g2, err := s.GetAgentGrant(ctx, kept.ID)
		require.NoError(t, err)
		assert.Nil(t, g2.RevokedAt)
	})

	// The Store interface documents that every bulk-revoke method returns
	// the ids of every grant MATCHED, not just newly revoked ones — callers
	// sweep sessions by that id set, and under-reporting an already-revoked
	// grant would leave its sessions unswept.
	t.Run("revoke by user returns every matched id including already revoked", func(t *testing.T) {
		s := newStore(t)
		victim, bystander := id.NewUserID(), id.NewUserID()
		org := id.NewOrgID()
		g1 := newGrant(t, victim, org, time.Now().Add(time.Hour))
		g2 := newGrant(t, victim, org, time.Now().Add(time.Hour))
		g3 := newGrant(t, bystander, org, time.Now().Add(time.Hour))
		for _, g := range []*agentauth.AgentGrant{g1, g2, g3} {
			require.NoError(t, s.CreateAgentGrant(ctx, g))
		}
		// g1 is revoked ahead of time — it must still be reported.
		require.NoError(t, s.RevokeAgentGrant(ctx, g1.ID))

		revoked, err := s.RevokeGrantsByUser(ctx, victim)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{g1.ID.String(), g2.ID.String()}, idStrings(revoked))

		survivor, err := s.GetAgentGrant(ctx, g3.ID)
		require.NoError(t, err)
		assert.Nil(t, survivor.RevokedAt, "another user's grant must survive")
	})

	t.Run("revoke by org revokes regardless of user", func(t *testing.T) {
		s := newStore(t)
		orgIn, orgOut := id.NewOrgID(), id.NewOrgID()
		userA, userB := id.NewUserID(), id.NewUserID()
		g1 := newGrant(t, userA, orgIn, time.Now().Add(time.Hour))
		g2 := newGrant(t, userB, orgIn, time.Now().Add(time.Hour))
		g3 := newGrant(t, userA, orgOut, time.Now().Add(time.Hour))
		for _, g := range []*agentauth.AgentGrant{g1, g2, g3} {
			require.NoError(t, s.CreateAgentGrant(ctx, g))
		}

		revoked, err := s.RevokeGrantsByOrg(ctx, orgIn)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{g1.ID.String(), g2.ID.String()}, idStrings(revoked))

		survivor, err := s.GetAgentGrant(ctx, g3.ID)
		require.NoError(t, err)
		assert.Nil(t, survivor.RevokedAt, "a grant in a different org must survive")
	})

	// A nil orgID passed to RevokeGrantsByAgent means "every org", not "the
	// org whose id is empty" — the central semantic the brief calls out as
	// easy to get wrong in SQL, where an empty string is a real value that
	// would otherwise match rows whose org actually is unset.
	t.Run("revoke by agent: nil org means every org, non-nil org scopes to it", func(t *testing.T) {
		s := newStore(t)
		agent := id.NewAgentID()
		user := id.NewUserID()
		org1, org2 := id.NewOrgID(), id.NewOrgID()
		g1 := newGrant(t, user, org1, time.Now().Add(time.Hour))
		g1.AgentID = agent
		g2 := newGrant(t, user, org2, time.Now().Add(time.Hour))
		g2.AgentID = agent
		require.NoError(t, s.CreateAgentGrant(ctx, g1))
		require.NoError(t, s.CreateAgentGrant(ctx, g2))

		scoped, err := s.RevokeGrantsByAgent(ctx, agent, org1)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{g1.ID.String()}, idStrings(scoped))
		survivor, err := s.GetAgentGrant(ctx, g2.ID)
		require.NoError(t, err)
		assert.Nil(t, survivor.RevokedAt, "revoking with a specific org must not touch other orgs")

		all, err := s.RevokeGrantsByAgent(ctx, agent, id.OrgID{})
		require.NoError(t, err)
		// g1 is matched again (already revoked); g2 is newly revoked.
		assert.ElementsMatch(t, []string{g1.ID.String(), g2.ID.String()}, idStrings(all))
		g2Now, err := s.GetAgentGrant(ctx, g2.ID)
		require.NoError(t, err)
		assert.NotNil(t, g2Now.RevokedAt, "a nil org must revoke across every org")
	})

	t.Run("org policy round trips", func(t *testing.T) {
		s := newStore(t)
		org := id.NewOrgID()
		require.NoError(t, s.PutOrgPolicy(ctx, &agentauth.OrgAgentPolicy{
			OrgID: org, Mode: agentauth.ModeAllowlist,
			MaxGrantTTL: 48 * time.Hour, AllowedScopes: []string{"invoices:read"},
		}))

		got, err := s.GetOrgPolicy(ctx, org)
		require.NoError(t, err)
		assert.Equal(t, agentauth.ModeAllowlist, got.Mode)
		assert.Equal(t, 48*time.Hour, got.MaxGrantTTL, "a duration must survive the round trip")
		assert.Equal(t, []string{"invoices:read"}, got.AllowedScopes)

		// PutOrgPolicy is an upsert: a second call with a different mode and
		// TTL replaces the row rather than erroring or leaving stale fields.
		require.NoError(t, s.PutOrgPolicy(ctx, &agentauth.OrgAgentPolicy{
			OrgID: org, Mode: agentauth.ModeBlocked, MaxGrantTTL: time.Hour,
		}))
		got2, err := s.GetOrgPolicy(ctx, org)
		require.NoError(t, err)
		assert.Equal(t, agentauth.ModeBlocked, got2.Mode)
		assert.Equal(t, time.Hour, got2.MaxGrantTTL)
		assert.Empty(t, got2.AllowedScopes)
	})

	t.Run("get org policy not found", func(t *testing.T) {
		s := newStore(t)
		_, err := s.GetOrgPolicy(ctx, id.NewOrgID())
		assert.ErrorIs(t, err, agentauth.ErrNotFound)
	})

	// I1: the zero-value OrgID is a real, meaningful policy key — it's what
	// governingOrgs falls back to for the single-tenant / app-scoped case,
	// where neither the agent nor the session carries an org — not an
	// absent value. A converter that unconditionally parses the stored org
	// id string turns "" into a parse error, and policyFor only maps
	// ErrNotFound to the "no policy configured" default; any other error
	// becomes a 500. One write through the exported Store must not
	// permanently break every read of the single-tenant policy.
	t.Run("zero-org policy round trips", func(t *testing.T) {
		s := newStore(t)
		require.NoError(t, s.PutOrgPolicy(ctx, &agentauth.OrgAgentPolicy{
			OrgID: id.OrgID{}, Mode: agentauth.ModeBlocked, MaxGrantTTL: time.Hour,
		}))

		got, err := s.GetOrgPolicy(ctx, id.OrgID{})
		require.NoError(t, err, "a zero-org policy must read back cleanly, not 500")
		assert.True(t, got.OrgID.IsNil())
		assert.Equal(t, agentauth.ModeBlocked, got.Mode)
		assert.Equal(t, time.Hour, got.MaxGrantTTL)
	})

	// A policy with a mode nothing recognizes must never reach storage on
	// any backend: Evaluate and CreateGrant both treat an unrecognized mode
	// as a deny, so a backend that let it through would silently accept
	// writes another part of the system can never correctly interpret.
	t.Run("put org policy rejects an unrecognized mode", func(t *testing.T) {
		s := newStore(t)
		err := s.PutOrgPolicy(ctx, &agentauth.OrgAgentPolicy{
			OrgID: id.NewOrgID(), Mode: agentauth.PolicyMode("bogus"),
		})
		require.Error(t, err)

		err = s.PutOrgPolicy(ctx, &agentauth.OrgAgentPolicy{OrgID: id.NewOrgID()})
		require.Error(t, err, "the zero-value mode must be refused too")
	})

	t.Run("list grants by user", func(t *testing.T) {
		s := newStore(t)
		userA, userB := id.NewUserID(), id.NewUserID()
		g1 := newGrant(t, userA, id.NewOrgID(), time.Now().Add(time.Hour))
		g2 := newGrant(t, userA, id.NewOrgID(), time.Now().Add(time.Hour))
		g3 := newGrant(t, userB, id.NewOrgID(), time.Now().Add(time.Hour))
		for _, g := range []*agentauth.AgentGrant{g1, g2, g3} {
			require.NoError(t, s.CreateAgentGrant(ctx, g))
		}

		got, err := s.ListGrantsByUser(ctx, userA)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{g1.ID.String(), g2.ID.String()}, idStrings(grantIDs(got)))
	})
}

// grantIDs projects a slice of grants down to their ids, for reuse with
// idStrings/ElementsMatch the same way idStrings works on revoke results.
func grantIDs(grants []*agentauth.AgentGrant) []id.AgentGrantID {
	out := make([]id.AgentGrantID, len(grants))
	for i, g := range grants {
		out[i] = g.ID
	}
	return out
}

func newMemoryConformanceStore(_ *testing.T) agentauth.Store { return agentauth.NewMemoryStore() }

// The SQLite store shares the model/converter code with Postgres (see
// store_models.go), so running the conformance suite against embedded
// SQLite exercises the same SQL column mapping — including the JSON scope
// round-trip and the partial unique index on client_id — that Postgres
// uses, without needing Docker.
func newSQLiteConformanceStore(t *testing.T) agentauth.Store {
	t.Helper()
	ctx := context.Background()
	// No foreign_keys pragma, and deliberately so: these tables carry no
	// REFERENCES to authsome_apps/authsome_agents (see the comment in
	// migrations.go), so there's nothing for the pragma to enforce here.
	dsn := "file:" + filepath.Join(t.TempDir(), "agentauth-conformance.db") + "?cache=shared"
	sdb := sqlitedriver.New()
	require.NoError(t, sdb.Open(ctx, dsn))
	db, err := grove.Open(sdb)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	// Core migrations satisfy the agentauth group's DependsOn("authsome");
	// the agentauth group creates authsome_agents/authsome_agent_grants/
	// authsome_agent_policies with the full column set.
	require.NoError(t, sqlitestore.New(db).Migrate(ctx, agentauth.SqliteMigrations))
	return agentauth.NewSqliteStore(db)
}

func TestStoreConformance_Memory(t *testing.T) { runStoreConformance(t, newMemoryConformanceStore) }
func TestStoreConformance_SQLite(t *testing.T) { runStoreConformance(t, newSQLiteConformanceStore) }
