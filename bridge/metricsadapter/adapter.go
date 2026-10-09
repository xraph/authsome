// Package metricsadapter records authsome's metrics in a Forge metrics
// registry, so an operator who already scrapes the app sees sign-ins,
// refusals, lockouts, replay detections and request latency beside
// everything else, with no second exporter.
package metricsadapter

import (
	"strings"
	"time"

	"github.com/xraph/forge"
	"github.com/xraph/go-utils/metrics"

	"github.com/xraph/authsome/bridge"
)

// Adapter implements bridge.MetricsCollector over a Forge metrics registry.
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

// metricName turns an authsome signal name into a registry name: dots and
// dashes become underscores under the authsome_ prefix.
func metricName(name string) string {
	r := strings.NewReplacer(".", "_", "-", "_", " ", "_")
	return "authsome_" + r.Replace(name)
}

func labels(kv ...string) metrics.MetricOption {
	m := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] != "" {
			m[kv[i]] = kv[i+1]
		}
	}
	return metrics.WithLabels(m)
}

// RecordEvent implements bridge.MetricsCollector: one counter per
// (action, resource, outcome, tenant) and, when a duration is given, a
// histogram of it in seconds.
func (a *Adapter) RecordEvent(action, resource, outcome, tenant string, duration time.Duration) {
	if a == nil {
		return
	}
	l := labels("action", action, "resource", resource, "outcome", outcome, "tenant", tenant)
	a.m.Counter("authsome_events_total", l).Inc()
	if duration > 0 {
		a.m.Histogram("authsome_event_duration_seconds", l).Observe(duration.Seconds())
	}
}

// IncrementGauge implements bridge.MetricsCollector.
func (a *Adapter) IncrementGauge(name, tenant string, delta int) {
	if a == nil {
		return
	}
	a.m.Gauge(metricName(name), labels("tenant", tenant)).Add(float64(delta))
}

// IncrementCounter implements bridge.MetricsCollector.
func (a *Adapter) IncrementCounter(name, tenant string) {
	if a == nil {
		return
	}
	a.m.Counter(metricName(name), labels("tenant", tenant)).Inc()
}

var _ bridge.MetricsCollector = (*Adapter)(nil)
