package principal_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/principal"
)

func TestRefStringRoundTrips(t *testing.T) {
	r := principal.Ref{Kind: principal.KindAgent, ID: "svc_01h2xcejqtf2nbrexx3vqjhp41"}
	assert.Equal(t, "agent:svc_01h2xcejqtf2nbrexx3vqjhp41", r.String())

	got, err := principal.ParseRef(r.String())
	require.NoError(t, err)
	assert.Equal(t, r, got)
}

func TestParseRefRejectsMalformed(t *testing.T) {
	for _, in := range []string{"", "agent", "agent:", ":svc_1", "nosuchkind:svc_1"} {
		_, err := principal.ParseRef(in)
		assert.Error(t, err, "ParseRef(%q) must fail", in)
	}
}

// An ID containing a colon must not be truncated. Refs are compared to make
// authorization decisions, so a split on the wrong colon would silently
// address a different principal.
func TestParseRefSplitsOnFirstColonOnly(t *testing.T) {
	got, err := principal.ParseRef("workload:svc_1:extra")
	require.NoError(t, err)
	assert.Equal(t, principal.KindWorkload, got.Kind)
	assert.Equal(t, "svc_1:extra", got.ID)
}

func TestZeroRef(t *testing.T) {
	assert.True(t, principal.Ref{}.IsZero())
	assert.False(t, principal.Ref{Kind: principal.KindUser, ID: "ausr_1"}.IsZero())
}

func TestPrincipalExpiry(t *testing.T) {
	at := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	past := at.Add(-time.Hour)
	future := at.Add(time.Hour)

	durable := principal.Principal{Ref: principal.Ref{Kind: principal.KindWorkload, ID: "svc_1"}}
	assert.False(t, durable.IsExpired(at), "a nil ExpiresAt means durable")
	assert.True(t, durable.IsActive(at))

	lapsed := principal.Principal{
		Ref:       principal.Ref{Kind: principal.KindAgent, ID: "svc_2"},
		ExpiresAt: &past,
	}
	assert.True(t, lapsed.IsExpired(at))
	assert.False(t, lapsed.IsActive(at))

	live := principal.Principal{
		Ref:       principal.Ref{Kind: principal.KindAgent, ID: "svc_3"},
		ExpiresAt: &future,
	}
	assert.False(t, live.IsExpired(at))
	assert.True(t, live.IsActive(at))

	disabled := principal.Principal{
		Ref:      principal.Ref{Kind: principal.KindAgent, ID: "svc_4"},
		Disabled: true,
	}
	assert.False(t, disabled.IsActive(at), "a disabled principal is never active")
}
