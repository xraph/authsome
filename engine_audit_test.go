package authsome_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
)

func TestHookEventsReachChronicleWithRequestInfo(t *testing.T) {
	mem := bridge.NewMemoryChronicle()
	eng, _ := newTestEngine(t, authsome.WithChronicle(mem))
	mem.Reset()

	ctx := hook.WithRequestInfo(context.Background(), hook.RequestInfo{
		IP: "203.0.113.9", UserAgent: "curl/8.0", RequestID: "req_1", SessionID: "sess_1",
	})
	eng.Hooks().Emit(ctx, &hook.Event{
		Action: "test.action", Resource: "thing", ResourceID: "id_1", ActorID: "user_1", Tenant: "app_1", OrgID: "org_1",
		Severity: hook.SeverityWarning, Category: "test", Metadata: map[string]string{"k": "v"},
		Private:  map[string]string{"token": "secret"},
	})

	events := mem.Events()
	require.Len(t, events, 1)
	ev := events[0]
	assert.Equal(t, "test.action", ev.Action)
	assert.Equal(t, "app_1", ev.Tenant)
	assert.Equal(t, "org_1", ev.OrgID)
	assert.Equal(t, "warning", ev.Severity)
	assert.Equal(t, "success", ev.Outcome)
	assert.Equal(t, "203.0.113.9", ev.IP)
	assert.Equal(t, "curl/8.0", ev.UserAgent)
	assert.Equal(t, "req_1", ev.RequestID)
	assert.Equal(t, "sess_1", ev.SessionID)
	assert.Equal(t, "v", ev.Metadata["k"])
	assert.NotContains(t, ev.Metadata, "token", "Private must never reach the trail")
}

func TestEmitCriticalFailsWhenChronicleRefuses(t *testing.T) {
	mem := bridge.NewMemoryChronicle()
	eng, _ := newTestEngine(t, authsome.WithChronicle(mem))
	mem.FailWith(errors.New("sink down"))

	err := eng.Hooks().EmitCritical(context.Background(), &hook.Event{Action: "admin.impersonate", Resource: "session", Tenant: "app_1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sink down")
}
