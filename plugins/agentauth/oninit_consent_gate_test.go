package agentauth_test

import (
	"context"
	"testing"

	log "github.com/xraph/go-utils/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/agentauth"
	"github.com/xraph/authsome/plugins/oauth2provider"
	"github.com/xraph/authsome/store/memory"
)

// TestOnInit_RegistersAsOAuth2ProviderConsentGate proves agentauth wires
// itself as oauth2provider's consent gate during OnInit, per the design spec
// ("agentauth registers itself as the gate during OnInit") and per
// oauth2provider's own consent_gate.go doc comment, which already asserts
// this happens. Observed behaviorally rather than by reaching into either
// plugin's unexported fields: a blocked agent's client_id must be refused by
// the REAL oauth2provider.Plugin.EvaluateConsent, not merely by agentauth's
// own Evaluate in isolation — that is the only way to prove the wiring
// happened, since nothing exposes the gate once it's set.
func TestOnInit_RegistersAsOAuth2ProviderConsentGate(t *testing.T) {
	agentStore := agentauth.NewMemoryStore()
	clientID := "blocked-client-" + id.NewAgentID().String()
	appID := id.NewAppID()
	require.NoError(t, agentStore.CreateAgent(context.Background(), &agentauth.Agent{
		ID: id.NewAgentID(), AppID: appID, ClientID: clientID,
		Name: "blocked", Origin: agentauth.OriginOrgRegistered, Status: agentauth.StatusBlocked,
	}))
	p := agentauth.New(agentauth.WithStore(agentStore))

	op := oauth2provider.New()
	reg := plugin.NewRegistry(log.NewNoopLogger())
	reg.Register(op)

	eng := &stubEngine{
		store:    memory.New(),
		registry: reg,
		bus:      hook.NewBus(log.NewNoopLogger()),
		logger:   log.NewNoopLogger(),
	}
	require.NoError(t, p.OnInit(context.Background(), eng))

	err := op.EvaluateConsent(context.Background(), clientID, id.NewUserID(), id.OrgID{}, appID, nil)
	assert.Error(t, err, "OnInit must register agentauth as oauth2provider's consent gate")
}

// TestOnInit_NoOauth2ProviderRegisteredIsANoOp proves OnInit does not error
// or panic when no "oauth2provider" plugin is registered on the engine — a
// host that hasn't installed oauth2provider, or a test harness that never
// will, must not be broken by agentauth trying to wire a gate that isn't
// there.
func TestOnInit_NoOauth2ProviderRegisteredIsANoOp(t *testing.T) {
	p := agentauth.New()
	reg := plugin.NewRegistry(log.NewNoopLogger())

	eng := &stubEngine{
		store:    memory.New(),
		registry: reg,
		bus:      hook.NewBus(log.NewNoopLogger()),
		logger:   log.NewNoopLogger(),
	}

	require.NoError(t, p.OnInit(context.Background(), eng))
}

// TestOnInit_Oauth2ProviderWithoutConsentGateSetterIsANoOp proves OnInit
// degrades gracefully — rather than panicking on a failed type assertion —
// if something else is registered under the name "oauth2provider" that does
// not expose SetConsentGate.
type namedButNotAGate struct{}

func (namedButNotAGate) Name() string { return "oauth2provider" }

func TestOnInit_Oauth2ProviderWithoutConsentGateSetterIsANoOp(t *testing.T) {
	p := agentauth.New()
	reg := plugin.NewRegistry(log.NewNoopLogger())
	reg.Register(namedButNotAGate{})

	eng := &stubEngine{
		store:    memory.New(),
		registry: reg,
		bus:      hook.NewBus(log.NewNoopLogger()),
		logger:   log.NewNoopLogger(),
	}

	require.NoError(t, p.OnInit(context.Background(), eng))
}
