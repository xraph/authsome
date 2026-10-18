package metricsadapter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/go-utils/metrics"

	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
)

func TestAdapterRecordsIntoRegistry(t *testing.T) {
	reg := metrics.NewMetricsCollector("test")
	a := New(reg)

	a.RecordEvent("auth.signin", "session", "success", "aapp_01jf0000000000000000000000", 30*time.Millisecond)
	a.RecordEvent("auth.signin", "session", "success", "aapp_01jf0000000000000000000001", 0)
	a.IncrementCounter("refresh.replay_detected", "aapp_01jf0000000000000000000000")
	a.IncrementCounter("refresh.replay_detected", "")
	a.IncrementGauge("sessions.active", "aapp_01jf0000000000000000000000", 3)

	// Two apps, one series: the tenant is not a label.
	l := metrics.WithLabels(map[string]string{"action": "auth.signin", "resource": "session", "outcome": "success"})
	assert.EqualValues(t, 2, reg.Counter("authsome_events_total", l).Value())
	assert.EqualValues(t, 2, reg.Counter("authsome_refresh_replay_detected_total").Value())
	assert.EqualValues(t, 3, reg.Gauge("authsome_sessions_active").Value())

	var nilAdapter *Adapter
	assert.NotPanics(t, func() { nilAdapter.IncrementCounter("x", "") }, "a nil adapter is a no-op")
}

// An event with an empty field keeps all three label keys, so the family
// never mixes label sets, and free text is folded into "other". Checked on a
// real Forge scrape, since that is where a mixed label set would break.
func TestAdapterKeepsLabelKeysConsistent(t *testing.T) {
	reg, scrape := telemetrytest.Registry(t)
	a := New(reg)

	a.RecordEvent("auth.signin", "", "", "aapp_01jf0000000000000000000000", 0)
	a.RecordEvent("GET /v1/sessions/ases_01jf0000000000000000000000", "session", "success", "", 0)
	a.IncrementCounter("control.degraded", "aapp_01jf0000000000000000000000")

	assert.ElementsMatch(t, []string{
		`forge_authsome_events_total{action="auth.signin",outcome="",resource=""}`,
		`forge_authsome_events_total{action="other",outcome="success",resource="session"}`,
		`forge_authsome_control_degraded_total`,
	}, telemetrytest.Series(scrape(), "forge_authsome_"))
}

func TestCounterNameGetsTotalOnce(t *testing.T) {
	assert.Equal(t, "authsome_auth_lockout_total", counterName("auth.lockout"))
	assert.Equal(t, "authsome_ratelimit_rejected_total", counterName("ratelimit.rejected"))
	assert.Equal(t, "authsome_things_total", counterName("things_total"))
}
