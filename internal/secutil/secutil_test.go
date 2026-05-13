package secutil_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/internal/secutil"
)

func TestNewTestEngine_Boots(t *testing.T) {
	eng := secutil.NewTestEngine(t)
	require.NotNil(t, eng)
	// Sanity: config is populated and engine is alive.
	assert.NotEmpty(t, eng.Config().BasePath)
}

func TestAttackRequest_HasNoOriginByDefault(t *testing.T) {
	req := secutil.AttackRequest(t, http.MethodPost, "/v1/something", nil)
	require.NotNil(t, req)
	assert.Empty(t, req.Header.Get("Origin"), "AttackRequest must not set Origin")
	assert.Empty(t, req.Header.Get("Referer"), "AttackRequest must not set Referer")
	assert.Empty(t, req.Header.Get("Cookie"), "AttackRequest must not set Cookie")
	assert.Equal(t, http.MethodPost, req.Method)
}

func TestAssertAuditEvent_PassesOnMatch(t *testing.T) {
	c := secutil.NewBufferedChronicle()
	require.NoError(t, c.Record(context.Background(), &bridge.AuditEvent{
		Action:   "user.signin",
		Outcome:  bridge.OutcomeSuccess,
		Severity: bridge.SeverityInfo,
	}))

	called := false
	secutil.AssertAuditEvent(t, c, "user.signin", func(ev *bridge.AuditEvent) {
		called = true
		assert.Equal(t, bridge.OutcomeSuccess, ev.Outcome)
	})
	assert.True(t, called, "inspect callback must run")
}

func TestAssertNoAuditEvent_PassesWhenAbsent(t *testing.T) {
	c := secutil.NewBufferedChronicle()
	require.NoError(t, c.Record(context.Background(), &bridge.AuditEvent{
		Action: "user.signin",
	}))
	secutil.AssertNoAuditEvent(t, c, "user.delete")
}
