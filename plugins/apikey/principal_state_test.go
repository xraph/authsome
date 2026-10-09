package apikey_test

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/principal"
)

type principalStateEngine struct {
	*mockEngine
	resolved *principal.Principal
	err      error
}

func (e principalStateEngine) ResolvePrincipal(context.Context, principal.Ref) (*principal.Principal, error) {
	return e.resolved, e.err
}

func TestStrategy_ServiceAccountPrincipalState(t *testing.T) {
	expired := time.Now().Add(-time.Hour)
	for _, tc := range []struct {
		name     string
		unwired  bool
		resolved *principal.Principal
		err      error
	}{
		{name: "absent resolver", unwired: true},
		{name: "resolver outage", err: errors.New("database unavailable: internal connection details")},
		{name: "nil result"},
		{name: "not found", err: principal.ErrNotFound},
		{name: "disabled", resolved: &principal.Principal{Disabled: true}},
		{name: "expired", resolved: &principal.Principal{ExpiresAt: &expired}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, gateEnabled := range []bool{false, true} {
				if tc.unwired && gateEnabled {
					continue // OnInit installs both resolver and gate.
				}
				name := "without gate"
				if gateEnabled {
					name = "permitting gate"
				}
				t.Run(name, func(t *testing.T) {
					p, store := newTestPlugin()
					gate := &fakeGate{}
					eng := principalStateEngine{
						mockEngine: &mockEngine{logger: log.NewNoopLogger(), store: store},
						resolved:   tc.resolved, err: tc.err,
					}
					if gateEnabled {
						eng.gate = gate
					}
					// Before OnInit, the public strategy has no resolver or gate.
					if !tc.unwired {
						require.NoError(t, p.OnInit(context.Background(), eng))
					}
					appID := id.NewAppID()
					raw := mintKey(t, store, appID, id.Nil, id.NewServiceAccountID())
					req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
					req.Header.Set("Authorization", "Bearer "+raw)
					req.Header.Set("X-App-ID", appID.String())

					result, err := p.Strategy().Authenticate(context.Background(), req)
					assert.Nil(t, result, "refusal must not create a session")
					assert.EqualError(t, err, "apikey: service account is not active")
					assert.Zero(t, gate.authorizeN, "principal resolution must precede the optional gate")
					assert.Zero(t, gate.observeN, "refusal must not be observed as success")
				})
			}
		})
	}
}
