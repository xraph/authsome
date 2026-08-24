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
	_, present := plainDecoded["impersonated_by"]
	assert.False(t, present, "an unimpersonated session must omit the key, as it does today")
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
