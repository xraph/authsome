package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// TestSessionToSessionImpersonationSurvivesLegacyColumnMix pins a row shape
// the conformance suite cannot produce, because fromSession always writes
// actors/actor_grant and impersonated_by together: a row written by an older
// binary during a rolling deploy, or one the 20260824000052 backfill has not
// reached yet, carries impersonated_by set with actor_grant still empty.
//
// toSession must resolve that shape as an impersonation. Before this test
// was added, the unconditional `s.ActorGrant = principal.GrantKind(m.ActorGrant)`
// ran after SetImpersonatedBy and reset the grant back to empty, so the row
// read back as an ordinary user session: no impersonation banner, no
// admin-severity audit record.
func TestSessionToSessionImpersonationSurvivesLegacyColumnMix(t *testing.T) {
	sessID := id.NewSessionID()
	appID := id.NewAppID()
	envID := id.NewEnvironmentID()
	userID := id.NewUserID()
	adminID := id.NewUserID()

	m := &SessionModel{
		ID:     sessID.String(),
		AppID:  appID.String(),
		EnvID:  envID.String(),
		UserID: userID.String(),
		// The legacy shape: impersonated_by is set, but actors and
		// actor_grant are left at their zero value, exactly as a row
		// written before those columns existed would be.
		ImpersonatedBy: adminID.String(),
	}

	got, err := toSession(m)
	require.NoError(t, err)
	assert.Equal(t, adminID.String(), got.ImpersonatedBy().String(),
		"a legacy impersonated_by column must still resolve to an impersonation, not be silently cleared by an empty actor_grant")
	assert.False(t, got.ImpersonatedBy().IsNil())
}

// TestSessionToSessionActorGrantWinsOverLegacyImpersonatedBy pins the other
// half of the ordering fix: when a row already carries a real actor chain
// (the authoritative shape written by fromSession going forward), a stray
// legacy impersonated_by value must never override it.
func TestSessionToSessionActorGrantWinsOverLegacyImpersonatedBy(t *testing.T) {
	sessID := id.NewSessionID()
	appID := id.NewAppID()
	envID := id.NewEnvironmentID()
	userID := id.NewUserID()
	agentID := id.NewServiceAccountID()
	stale := id.NewUserID()

	m := &SessionModel{
		ID:         sessID.String(),
		AppID:      appID.String(),
		EnvID:      envID.String(),
		UserID:     userID.String(),
		Actors:     []byte(`[{"kind":"agent","id":"` + agentID.String() + `"}]`),
		ActorGrant: "delegation",
		// A stale legacy value that must not win now that actor_grant is set.
		ImpersonatedBy: stale.String(),
	}

	got, err := toSession(m)
	require.NoError(t, err)
	assert.True(t, got.ImpersonatedBy().IsNil(),
		"a session with a real actor_grant must not read back as an impersonation from a stale legacy column")
	require.Len(t, got.Actors, 1)
	assert.Equal(t, agentID.String(), got.Actors[0].ID)
}
