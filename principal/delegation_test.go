package principal_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
)

func TestDelegationIsActive(t *testing.T) {
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	past := at.Add(-time.Hour)
	future := at.Add(time.Hour)

	base := principal.Delegation{
		ID:        id.NewDelegationID(),
		Actor:     principal.Ref{Kind: principal.KindAgent, ID: "svc_1"},
		Subject:   principal.Ref{Kind: principal.KindUser, ID: "ausr_1"},
		GrantKind: principal.GrantDelegation,
	}

	assert.True(t, base.IsActive(at), "no expiry and no revocation means active")

	expired := base
	expired.ExpiresAt = &past
	assert.False(t, expired.IsActive(at))

	live := base
	live.ExpiresAt = &future
	assert.True(t, live.IsActive(at))

	// A revoked grant stays dead even when its expiry is still in the future.
	revoked := live
	revoked.RevokedAt = &past
	assert.False(t, revoked.IsActive(at), "revocation must beat a future expiry")
}

// An empty scope list on a grant means the grant places no scope restriction
// of its own. Narrowing then comes from the actor's own scopes alone. An
// empty list must not be read as "deny everything", which would make every
// grant created without an explicit scope useless.
func TestDelegationAllowsScope(t *testing.T) {
	unrestricted := principal.Delegation{}
	assert.True(t, unrestricted.AllowsScope("repo:write"))

	narrow := principal.Delegation{Scopes: []string{"repo:read", "issues:read"}}
	assert.True(t, narrow.AllowsScope("repo:read"))
	assert.False(t, narrow.AllowsScope("repo:write"))
}

func TestDelegationIDPrefix(t *testing.T) {
	d := id.NewDelegationID()
	parsed, err := id.ParseDelegationID(d.String())
	assert.NoError(t, err)
	assert.Equal(t, d.String(), parsed.String())

	_, err = id.ParseDelegationID(id.NewUserID().String())
	assert.Error(t, err, "a user id must not parse as a delegation id")
}

func TestPrincipalContextRoundTrip(t *testing.T) {
	ctx := context.Background()

	_, ok := principal.FromContext(ctx)
	assert.False(t, ok, "a bare context carries no principal")

	p := &principal.Principal{Ref: principal.Ref{Kind: principal.KindAgent, ID: "svc_1"}}
	ctx = principal.NewContext(ctx, p)
	got, ok := principal.FromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, p, got)

	chain := principal.Chain{{Kind: principal.KindAgent, ID: "svc_1"}}
	ctx = principal.NewActorsContext(ctx, chain)
	gotChain, ok := principal.ActorsFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, chain, gotChain)
}
