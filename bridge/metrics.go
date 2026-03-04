package bridge

import "time"

// MetricsCollector is an optional interface for collecting authentication
// metrics. Implementations may export to Prometheus, OpenTelemetry, or
// any other observability backend.
//
// The engine registers a hook.Handler that delegates events to the
// MetricsCollector when configured via WithMetrics().
type MetricsCollector interface {
	// RecordEvent records an authentication event metric.
	RecordEvent(action, resource, outcome, tenant string, duration time.Duration)

	// IncrementGauge adjusts a named gauge (e.g., active sessions).
	IncrementGauge(name, tenant string, delta int)
}
