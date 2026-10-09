package authsome_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/hook"
	"github.com/xraph/authsome/user"
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
		Private: map[string]string{"token": "secret"},
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

func signUpAuditUser(t *testing.T, eng *authsome.Engine, email string) *user.User {
	t.Helper()
	u, _, err := eng.SignUp(context.Background(), &account.SignUpRequest{
		AppID:    testAppID(t),
		Email:    email,
		Password: "Password-12345",
	})
	require.NoError(t, err)
	return u
}

func TestSignInRecordsExactlyOnceWithActorAndTenant(t *testing.T) {
	mem := bridge.NewMemoryChronicle()
	eng, _ := newTestEngine(t, authsome.WithChronicle(mem))
	u := signUpAuditUser(t, eng, "once@example.com")
	mem.Reset()

	_, _, err := eng.SignIn(context.Background(), &account.SignInRequest{
		AppID: u.AppID, Email: "once@example.com", Password: "Password-12345",
	})
	require.NoError(t, err)

	var signInCount int
	for _, ev := range mem.Events() {
		require.NotEmpty(t, ev.Tenant, "every event needs a tenant: %+v", ev)
		if ev.Action == hook.ActionSignIn {
			signInCount++
			assert.Equal(t, u.ID.String(), ev.ActorID)
			assert.Equal(t, "password", ev.Metadata["auth_method"])
			assert.NotEmpty(t, ev.SessionID)
		}
	}
	assert.Equal(t, 1, signInCount, "sign-in must be recorded once, not once per legacy audit call")
}

func TestFailedSignInDoesNotPersistTheIdentifier(t *testing.T) {
	mem := bridge.NewMemoryChronicle()
	eng, _ := newTestEngine(t, authsome.WithChronicle(mem))
	u := signUpAuditUser(t, eng, "victim@example.com")
	mem.Reset()

	_, _, err := eng.SignIn(context.Background(), &account.SignInRequest{
		AppID: u.AppID, Email: "victim@example.com", Password: "wrong-password",
	})
	require.Error(t, err)

	events := mem.Events()
	require.NotEmpty(t, events)
	for _, ev := range events {
		assert.NotContains(t, ev.Metadata, "identifier")
		assert.NotEqual(t, "victim@example.com", ev.Metadata["email"])
		if ev.Action == hook.ActionSignIn {
			assert.NotEmpty(t, ev.Metadata["identifier_hash"])
			assert.Equal(t, "failure", ev.Outcome)
		}
	}
}

func TestResetTokenNeverReachesTheTrail(t *testing.T) {
	mem := bridge.NewMemoryChronicle()
	eng, _ := newTestEngine(t, authsome.WithChronicle(mem))
	u := signUpAuditUser(t, eng, "reset@example.com")
	mem.Reset()

	_, err := eng.ForgotPassword(context.Background(), u.AppID, "reset@example.com")
	require.NoError(t, err)

	events := mem.Events()
	require.NotEmpty(t, events)
	for _, ev := range events {
		assert.NotContains(t, ev.Metadata, "token")
		assert.NotContains(t, ev.Metadata, "code")
		assert.NotContains(t, ev.Metadata, "email")
	}
}

func TestEmitCriticalFailsWhenChronicleRefuses(t *testing.T) {
	mem := bridge.NewMemoryChronicle()
	eng, _ := newTestEngine(t, authsome.WithChronicle(mem))
	mem.FailWith(errors.New("sink down"))

	err := eng.Hooks().EmitCritical(context.Background(), &hook.Event{Action: "admin.impersonate", Resource: "session", Tenant: "app_1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sink down")
}
