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
