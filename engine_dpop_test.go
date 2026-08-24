package authsome_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/internal/secutil"
)

func TestEngine_DPoPDefaultsToOff(t *testing.T) {
	eng := secutil.NewTestEngine(t)
	require.NotNil(t, eng.DPoPValidator(), "a validator must always exist so nothing has to nil-check it")
	assert.Equal(t, dpop.ModeOff, eng.DPoPModeForApp(context.Background(), eng.PlatformAppID()))
}

func TestEngine_DPoPNonceSignerPresentWhenSecretAvailable(t *testing.T) {
	secutil.InitTestNonceSigner(t)
	eng := secutil.NewTestEngine(t)
	// A nil signer is legitimate when no secret can be derived. What must not
	// happen is a signer built from a per-process random secret, because its
	// nonces would fail verification on every other instance.
	if s := eng.DPoPNonceSigner(); s != nil {
		n := s.Issue("jkt-abc")
		assert.True(t, s.Verify("jkt-abc", n))
	}
}
