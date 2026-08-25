package mongo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
)

// TestSessionModelToSessionImpersonationSurvivesLegacyColumnMix pins a
// document shape the conformance suite cannot produce, because toSessionModel
// always writes actors/actor_grant and impersonated_by together: a document
// written by an older binary during a rolling deploy, or one the
// add_principal_fields_and_backfill_chain migration has not reached yet,
// carries impersonated_by set with actor_grant still empty.
//
// fromSessionModel must resolve that shape as an impersonation. Before this
// test was added, an unconditional `s.ActorGrant =
// principal.GrantKind(m.ActorGrant)` placed after the SetImpersonatedBy call
// would reset the grant back to empty, so the document read back as an
// ordinary user session: no impersonation banner, no admin-severity audit
// record.
func TestSessionModelToSessionImpersonationSurvivesLegacyColumnMix(t *testing.T) {
	sessID := id.NewSessionID()
	appID := id.NewAppID()
	envID := id.NewEnvironmentID()
	userID := id.NewUserID()
	adminID := id.NewUserID()

	m := &sessionModel{
		ID:     sessID.String(),
		AppID:  appID.String(),
		EnvID:  envID.String(),
		UserID: userID.String(),
		// The legacy shape: impersonated_by is set, but actors and
		// actor_grant are left at their zero value, exactly as a document
		// written before those fields existed would be.
		ImpersonatedBy: adminID.String(),
	}

	got, err := fromSessionModel(m)
	require.NoError(t, err)
	assert.Equal(t, adminID.String(), got.ImpersonatedBy().String(),
		"a legacy impersonated_by field must still resolve to an impersonation, not be silently cleared by an empty actor_grant")
	assert.False(t, got.ImpersonatedBy().IsNil())
}

// TestSessionModelToSessionActorGrantWinsOverLegacyImpersonatedBy pins the
// other half of the ordering fix: when a document already carries a real
// actor chain (the authoritative shape written by toSessionModel going
// forward), a stray legacy impersonated_by value must never override it.
func TestSessionModelToSessionActorGrantWinsOverLegacyImpersonatedBy(t *testing.T) {
	sessID := id.NewSessionID()
	appID := id.NewAppID()
	envID := id.NewEnvironmentID()
	userID := id.NewUserID()
	agentID := id.NewServiceAccountID()
	stale := id.NewUserID()

	m := &sessionModel{
		ID:         sessID.String(),
		AppID:      appID.String(),
		EnvID:      envID.String(),
		UserID:     userID.String(),
		Actors:     []actorModel{{Kind: "agent", ID: agentID.String()}},
		ActorGrant: "delegation",
		// A stale legacy value that must not win now that actor_grant is set.
		ImpersonatedBy: stale.String(),
	}

	got, err := fromSessionModel(m)
	require.NoError(t, err)
	assert.True(t, got.ImpersonatedBy().IsNil(),
		"a session with a real actor_grant must not read back as an impersonation from a stale legacy field")
	require.Len(t, got.Actors, 1)
	assert.Equal(t, agentID.String(), got.Actors[0].ID)
}
