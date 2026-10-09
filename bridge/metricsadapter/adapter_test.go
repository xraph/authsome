package metricsadapter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xraph/go-utils/metrics"
)

func TestAdapterRecordsIntoRegistry(t *testing.T) {
	reg := metrics.NewMetricsCollector("test")
	a := New(reg)

	a.RecordEvent("auth.signin", "session", "success", "app-1", 30*time.Millisecond)
	a.RecordEvent("auth.signin", "session", "success", "app-1", 0)
	a.IncrementCounter("refresh.replay_detected", "app-1")
	a.IncrementCounter("refresh.replay_detected", "app-1")
	a.IncrementGauge("sessions.active", "app-1", 3)

	l := metrics.WithLabels(map[string]string{"action": "auth.signin", "resource": "session", "outcome": "success", "tenant": "app-1"})
	assert.EqualValues(t, 2, reg.Counter("authsome_events_total", l).Value())
	assert.EqualValues(t, 2, reg.Counter("authsome_refresh_replay_detected", metrics.WithLabels(map[string]string{"tenant": "app-1"})).Value())

	var nilAdapter *Adapter
	assert.NotPanics(t, func() { nilAdapter.IncrementCounter("x", "") }, "a nil adapter is a no-op")
}
