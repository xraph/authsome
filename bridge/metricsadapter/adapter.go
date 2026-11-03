// Package metricsadapter implements bridge.MetricsCollector over a go-utils
// metrics registry, such as a Forge app's.
//
// You only need it if you embed the engine and want the bridge signals
// (hook-bus events and the operational counters IncrementCounter carries)
// in a registry. In a Forge app the extension passes the app's registry to
// authsome.WithMetricsSink instead, which records authsome's own metric
// families and needs no adapter.
//
// Every family this adapter writes keeps the same label keys on every
// series, and none is labelled by tenant: the tenant the engine passes is
// an app ID, which is unbounded.
package metricsadapter

import (
	"strings"
	"time"

	"github.com/xraph/forge"
	"github.com/xraph/go-utils/metrics"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/internal/telemetry"
)

// Adapter implements bridge.MetricsCollector over a metrics registry.
type Adapter struct {
	m metrics.Metrics
}

// New returns an adapter recording into m. A nil registry yields a nil
// adapter, which callers treat as "no metrics".
func New(m forge.Metrics) *Adapter {
	if m == nil {
		return nil
	}
	return &Adapter{m: m}
}

// metricName turns an authsome signal name into a registry name: dots,
// dashes and spaces become underscores under the authsome_ prefix.
func metricName(name string) string {
	r := strings.NewReplacer(".", "_", "-", "_", " ", "_")
	return "authsome_" + r.Replace(name)
}

// counterName is metricName with the _total suffix a counter carries.
func counterName(name string) string {
	n := metricName(name)
	if strings.HasSuffix(n, "_total") {
		return n
	}
	return n + "_total"
}

// eventLabels always sets all three keys, empty or not, so every series
// under one name has the same label set. The values go through
// telemetry.Bounded, so a caller that passes free text can't grow the
// series without bound.
func eventLabels(action, resource, outcome string) metrics.MetricOption {
	return metrics.WithLabels(map[string]string{
		"action":   telemetry.Bounded(action),
		"resource": telemetry.Bounded(resource),
		"outcome":  telemetry.Bounded(outcome),
	})
}

// RecordEvent implements bridge.MetricsCollector: authsome_events_total by
// action, resource and outcome and, when a duration is given, a histogram
// of it in seconds under the same labels. The tenant is ignored.
func (a *Adapter) RecordEvent(action, resource, outcome, _ string, duration time.Duration) {
	if a == nil {
		return
	}
	l := eventLabels(action, resource, outcome)
	a.m.Counter("authsome_events_total", l).Inc()
	if duration > 0 {
		a.m.Histogram("authsome_event_duration_seconds", l, metrics.WithUnit("s")).Observe(duration.Seconds())
	}
}

// IncrementGauge implements bridge.MetricsCollector. The tenant is
// ignored.
func (a *Adapter) IncrementGauge(name, _ string, delta int) {
	if a == nil {
		return
	}
	a.m.Gauge(metricName(name)).Add(float64(delta))
}

// IncrementCounter implements bridge.MetricsCollector:
// IncrementCounter("auth.lockout", ...) adds one to
// authsome_auth_lockout_total. The tenant is ignored.
func (a *Adapter) IncrementCounter(name, _ string) {
	if a == nil {
		return
	}
	a.m.Counter(counterName(name)).Inc()
}

var _ bridge.MetricsCollector = (*Adapter)(nil)
