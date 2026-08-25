package session_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
	"github.com/xraph/authsome/session"
)

// Impersonation is now one shape of actor chain rather than its own field.
// Setting it must produce a chain, and reading it back must find the admin.
func TestImpersonationRoundTripsThroughTheChain(t *testing.T) {
	admin := id.NewUserID()
	target := id.NewUserID()

	s := &session.Session{UserID: target}
	assert.True(t, s.ImpersonatedBy().IsNil(), "a plain session has no impersonator")

	s.SetImpersonatedBy(admin)
	assert.Equal(t, admin.String(), s.ImpersonatedBy().String())
	assert.Equal(t, principal.GrantImpersonation, s.ActorGrant)

	actor, ok := s.Actors.Actor()
	require.True(t, ok)
	assert.Equal(t, principal.Ref{Kind: principal.KindUser, ID: admin.String()}, actor)
}

// A delegation chain is not impersonation. An agent acting for a user must
// not surface as that user's impersonator, or every audit record and every
// admin banner reads the delegation as an admin takeover.
func TestDelegationChainIsNotImpersonation(t *testing.T) {
	s := &session.Session{
		UserID:     id.NewUserID(),
		ActorGrant: principal.GrantDelegation,
		Actors:     principal.Chain{{Kind: principal.KindAgent, ID: "svc_1"}},
	}
	assert.True(t, s.ImpersonatedBy().IsNil(), "a delegation must not read as impersonation")
}

// AuthzActors is the single deliberate security exception in the whole
// actor-chain design, so it gets its own tests rather than relying on
// coverage from the impersonation/delegation tests above.
//
// For impersonation, AuthzActors must return nil: the admin's own chain is
// withheld from Engine.Can on purpose. Impersonating a user is the request to
// be evaluated as that user, not as the admin acting on the user's behalf. If
// the admin's chain were intersected in, an admin impersonating a user could
// do LESS than that user can do on their own, which inverts what
// impersonation is for and defeats its entire purpose ("see exactly what
// this user sees"). The one-time admin authorization check for entering
// impersonation lives on Engine.Impersonate; it is deliberately not repeated
// on every subsequent permission check made while impersonating.
func TestAuthzActorsWithholdsTheAdminDuringImpersonation(t *testing.T) {
	admin := id.NewUserID()
	s := &session.Session{
		UserID:     id.NewUserID(),
		ActorGrant: principal.GrantImpersonation,
		Actors:     principal.Chain{{Kind: principal.KindUser, ID: admin.String()}},
	}

	assert.Nil(t, s.AuthzActors(),
		"impersonation must not intersect the admin's own permissions into the check")
}

// For delegation, AuthzActors must return the chain unchanged: every actor in
// it is a real party Engine.Can has to independently authorize, and dropping
// any of them would let delegation widen instead of only narrowing.
func TestAuthzActorsReturnsTheChainForDelegation(t *testing.T) {
	chain := principal.Chain{{Kind: principal.KindAgent, ID: "svc_1"}}
	s := &session.Session{
		UserID:     id.NewUserID(),
		ActorGrant: principal.GrantDelegation,
		Actors:     chain,
	}

	assert.Equal(t, chain, s.AuthzActors(),
		"delegation must hand every actor to the check, unmodified")
}

func TestSubjectDerivesFromPrincipalKind(t *testing.T) {
	uid := id.NewUserID()
	human := &session.Session{UserID: uid}
	assert.Equal(t, principal.Ref{Kind: principal.KindUser, ID: uid.String()}, human.Subject(),
		"an unset PrincipalKind means user")
	assert.True(t, human.IsHumanPrincipal())

	svcID := id.NewServiceAccountID()
	machine := &session.Session{PrincipalKind: principal.KindService, ServiceAccountID: svcID}
	assert.Equal(t, principal.Ref{Kind: principal.KindService, ID: svcID.String()}, machine.Subject())
	assert.False(t, machine.IsHumanPrincipal())

	agent := &session.Session{PrincipalKind: principal.KindAgent, ServiceAccountID: svcID}
	assert.Equal(t, principal.Ref{Kind: principal.KindAgent, ID: svcID.String()}, agent.Subject())
	assert.False(t, agent.IsHumanPrincipal())
}

// The wire format must not move. Removing the struct field would silently drop
// impersonated_by from every serialized session.
// The wire format must not move. Before this change ImpersonatedBy was an
// id.UserID, a struct, and encoding/json never omits a struct for
// ,omitempty, so the key was always emitted, holding "" when nobody was
// impersonating. Omitting the key now would still be a move: a consumer
// that distinguishes an absent key from an empty one would see a session
// that no longer reports impersonation state at all.
func TestImpersonatedByStaysOnTheWire(t *testing.T) {
	admin := id.NewUserID()
	s := &session.Session{UserID: id.NewUserID()}
	s.SetImpersonatedBy(admin)

	raw, err := json.Marshal(s)
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, admin.String(), decoded["impersonated_by"])

	plain, err := json.Marshal(&session.Session{UserID: id.NewUserID()})
	require.NoError(t, err)
	var plainDecoded map[string]any
	require.NoError(t, json.Unmarshal(plain, &plainDecoded))
	gotImp, present := plainDecoded["impersonated_by"]
	assert.True(t, present, "an unimpersonated session must still carry the key, as it always has")
	assert.Equal(t, "", gotImp, "an unimpersonated session's impersonated_by must be empty, not absent")
}

func TestSessionJSONRoundTrip(t *testing.T) {
	admin := id.NewUserID()
	s := &session.Session{UserID: id.NewUserID()}
	s.SetImpersonatedBy(admin)

	raw, err := json.Marshal(s)
	require.NoError(t, err)

	var back session.Session
	require.NoError(t, json.Unmarshal(raw, &back))
	assert.Equal(t, admin.String(), back.ImpersonatedBy().String())
}
