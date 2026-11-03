package authsome

import "github.com/xraph/authsome/internal/telemetry"

// Telemetry returns the engine's metric instruments, for authsome's own
// plugins to record into. It is nil when the engine was built without
// WithMetricsSink, and every method on it is safe to call on nil.
func (e *Engine) Telemetry() *telemetry.Metrics { return e.tel }

// ──────────────────────────────────────────────────
// Operational counters
// ──────────────────────────────────────────────────

// controlDegraded counts a security control that could not run and let the
// request through: once on the bridge collector, under its historical name,
// and once in the controls_degraded family.
func (e *Engine) controlDegraded(control, tenant string) {
	e.count("control.degraded", tenant)
	e.tel.ControlDegraded(control)
}

// throttled counts a request a rate limiter refused.
func (e *Engine) throttled(limiter, tenant string) {
	name := "ratelimit.rejected"
	if limiter == telemetry.LimiterIdentifier {
		name = "identifier_limit.rejected"
	}
	e.count(name, tenant)
	e.tel.Throttled(limiter)
}

// limiterError counts a rate limiter that could not answer.
func (e *Engine) limiterError(limiter, tenant string) {
	e.count("ratelimit.error", tenant)
	e.tel.RateLimiterError(limiter)
}
