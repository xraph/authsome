package bridge_test

import (
	"context"
	log "github.com/xraph/go-utils/log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/bridge"
)

func TestSlogChronicle_Record(t *testing.T) {
	c := bridge.NewSlogChronicle(log.NewNoopLogger())

	err := c.Record(context.Background(), &bridge.AuditEvent{
		Action:     "signup",
		Resource:   "user",
		ResourceID: "auth_usr_test123",
		ActorID:    "auth_usr_test123",
		Tenant:     "auth_app_test",
		Outcome:    bridge.OutcomeSuccess,
		Severity:   bridge.SeverityInfo,
		Metadata:   map[string]string{"email": "test@example.com"},
	})

	assert.NoError(t, err)
}

func TestNoopAuthorizer_AlwaysAllows(t *testing.T) {
	a := bridge.NewNoopAuthorizer()

	result, err := a.Check(context.Background(), &bridge.AuthzRequest{
		Subject:  "user:123",
		Action:   "delete",
		Resource: "user:456",
		Tenant:   "app:test",
	})

	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Contains(t, result.Reason, "standalone mode")
}

func TestNoopKeyManager_CreateKey_ReturnsError(t *testing.T) {
	km := bridge.NewNoopKeyManager()

	_, err := km.CreateKey(context.Background(), &bridge.CreateKeyInput{
		Name:  "test-key",
		Owner: "user:123",
	})

	assert.ErrorIs(t, err, bridge.ErrKeyManagerNotAvailable)
}

func TestNoopKeyManager_ValidateKey_ReturnsError(t *testing.T) {
	km := bridge.NewNoopKeyManager()

	_, err := km.ValidateKey(context.Background(), "some-raw-key")
	assert.ErrorIs(t, err, bridge.ErrKeyManagerNotAvailable)
}

func TestNoopKeyManager_RevokeKey_ReturnsError(t *testing.T) {
	km := bridge.NewNoopKeyManager()

	err := km.RevokeKey(context.Background(), "key-id")
	assert.ErrorIs(t, err, bridge.ErrKeyManagerNotAvailable)
}

func TestNoopRelay_Send(t *testing.T) {
	r := bridge.NewNoopRelay(log.NewNoopLogger())

	err := r.Send(context.Background(), &bridge.WebhookEvent{
		Type:     "user.created",
		TenantID: "auth_app_test",
		Data:     map[string]string{"user_id": "auth_usr_123"},
	})

	assert.NoError(t, err)
}

func TestNoopRelay_RegisterEventTypes(t *testing.T) {
	r := bridge.NewNoopRelay(log.NewNoopLogger())

	err := r.RegisterEventTypes(context.Background(), bridge.WebhookEventCatalog())
	assert.NoError(t, err)
}

func TestWebhookEventCatalog(t *testing.T) {
	catalog := bridge.WebhookEventCatalog()

	assert.NotEmpty(t, catalog)

	// Verify key events exist
	names := make(map[string]bool)
	for _, def := range catalog {
		names[def.Name] = true
		assert.NotEmpty(t, def.Description)
		assert.NotEmpty(t, def.Group)
	}

	assert.True(t, names["user.created"])
	assert.True(t, names["auth.signin"])
	assert.True(t, names["auth.signout"])
	assert.True(t, names["session.created"])
	assert.True(t, names["org.created"])
}

func TestChronicleFunc(t *testing.T) {
	var recorded *bridge.AuditEvent
	fn := bridge.ChronicleFunc(func(_ context.Context, event *bridge.AuditEvent) error {
		recorded = event
		return nil
	})

	event := &bridge.AuditEvent{Action: "test-action", Severity: bridge.SeverityInfo}
	err := fn.Record(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, "test-action", recorded.Action)
}

func TestEventRelayFunc(t *testing.T) {
	var sentEvent *bridge.WebhookEvent
	fn := bridge.EventRelayFunc(func(_ context.Context, event *bridge.WebhookEvent) error {
		sentEvent = event
		return nil
	})

	event := &bridge.WebhookEvent{Type: "user.created", TenantID: "test"}
	err := fn.Send(context.Background(), event)

	require.NoError(t, err)
	assert.Equal(t, "user.created", sentEvent.Type)

	// RegisterEventTypes should be a no-op
	err = fn.RegisterEventTypes(context.Background(), nil)
	assert.NoError(t, err)
}

func TestSeverityConstants(t *testing.T) {
	assert.Equal(t, "info", bridge.SeverityInfo)
	assert.Equal(t, "warning", bridge.SeverityWarning)
	assert.Equal(t, "critical", bridge.SeverityCritical)
}

func TestOutcomeConstants(t *testing.T) {
	assert.Equal(t, "success", bridge.OutcomeSuccess)
	assert.Equal(t, "failure", bridge.OutcomeFailure)
}
