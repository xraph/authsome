package telemetry

import (
	"time"

	"github.com/xraph/forge"
	gometrics "github.com/xraph/go-utils/metrics"
)

// Metric names as authsome records them. Forge's Prometheus export puts its
// namespace in front, so authsome_signin_attempts_total scrapes as
// forge_authsome_signin_attempts_total.
const (
	MetricSignIns              = "authsome_signin_attempts_total"
	MetricMFAVerifications     = "authsome_mfa_verifications_total"
	MetricSessionsRefreshed    = "authsome_sessions_refreshed_total"
	MetricPasswordHashDuration = "authsome_password_hash_duration_seconds"
	MetricAPIKeyAuths          = "authsome_apikey_authentications_total"  // #nosec G101 -- a metric name, not a credential
	MetricOAuth2Tokens         = "authsome_oauth2_tokens_requested_total" // #nosec G101 -- a metric name, not a credential
	MetricProviderDuration     = "authsome_provider_request_duration_seconds"
	MetricRetentionRows        = "authsome_retention_rows_deleted_total"
	MetricRetentionSweeps      = "authsome_retention_sweeps_attempted_total"
	MetricRetentionJobs        = "authsome_retention_jobs_processed_total"
	MetricControlsDegraded     = "authsome_controls_degraded_total"
	MetricRequestsThrottled    = "authsome_requests_throttled_total"
	MetricRateLimiterErrors    = "authsome_ratelimiter_errors_total"
	MetricAuditRecordsFailed   = "authsome_audit_records_failed_total"
	MetricEvents               = "authsome_events_total"
)

// passwordHashBuckets span a bcrypt or Argon2id run: a few milliseconds at
// test cost, tens to a few hundred at production cost, and a tail for a
// host under memory pressure.
var passwordHashBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2}

// providerBuckets span a third-party HTTP call from a warm connection to
// the longest client timeout authsome sets (30 seconds for a CRM).
var providerBuckets = []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30}

// Metrics holds authsome's instruments. They are declared once, when the
// engine is built. A nil *Metrics records nothing, which is what an engine
// built without WithMetricsSink gets; every method is safe to call on it.
type Metrics struct {
	signIns          *forge.Vec[forge.Counter]
	mfa              *forge.Vec[forge.Counter]
	refreshes        *forge.Vec[forge.Counter]
	passwordHash     *forge.Vec[forge.Histogram]
	apiKeys          *forge.Vec[forge.Counter]
	oauth2Tokens     *forge.Vec[forge.Counter]
	provider         *forge.Vec[forge.Histogram]
	retentionRows    *forge.Vec[forge.Counter]
	retentionSweeps  *forge.Vec[forge.Counter]
	retentionJobs    *forge.Vec[forge.Counter]
	controlsDegraded *forge.Vec[forge.Counter]
	throttled        *forge.Vec[forge.Counter]
	limiterErrors    *forge.Vec[forge.Counter]
	auditFailed      forge.Counter
	events           *forge.Vec[forge.Counter]
}

// NewMetrics declares authsome's instruments in m. A nil m returns nil.
func NewMetrics(m gometrics.Metrics) *Metrics {
	if m == nil {
		return nil
	}
	return &Metrics{
		signIns:   forge.CounterVec(m, MetricSignIns, "method", "result"),
		mfa:       forge.CounterVec(m, MetricMFAVerifications, "flow", "method", "result"),
		refreshes: forge.CounterVec(m, MetricSessionsRefreshed, "result"),
		passwordHash: forge.HistogramVec(m, MetricPasswordHashDuration, forge.HistogramOpts{
			Buckets: passwordHashBuckets,
			Help:    "Time authsome spent hashing or verifying a password, by operation and algorithm.",
			Unit:    "s",
		}, "operation", "algorithm"),
		apiKeys:      forge.CounterVec(m, MetricAPIKeyAuths, "result"),
		oauth2Tokens: forge.CounterVec(m, MetricOAuth2Tokens, "grant_type", "result"),
		provider: forge.HistogramVec(m, MetricProviderDuration, forge.HistogramOpts{
			Buckets: providerBuckets,
			Help:    "Time a third-party call took to answer, by provider kind and result.",
			Unit:    "s",
		}, "kind", "result"),
		retentionRows:    forge.CounterVec(m, MetricRetentionRows, "kind"),
		retentionSweeps:  forge.CounterVec(m, MetricRetentionSweeps, "result"),
		retentionJobs:    forge.CounterVec(m, MetricRetentionJobs, "provider", "outcome"),
		controlsDegraded: forge.CounterVec(m, MetricControlsDegraded, "control"),
		throttled:        forge.CounterVec(m, MetricRequestsThrottled, "limiter"),
		limiterErrors:    forge.CounterVec(m, MetricRateLimiterErrors, "limiter"),
		auditFailed:      m.Counter(MetricAuditRecordsFailed),
		events:           forge.CounterVec(m, MetricEvents, "action", "resource", "outcome"),
	}
}

// SignIn counts one sign-in attempt. method goes through AuthMethod.
func (t *Metrics) SignIn(method, result string) {
	if t == nil {
		return
	}
	t.signIns.With(AuthMethod(method), result).Inc()
}

// MFAVerification counts one second-factor code check.
func (t *Metrics) MFAVerification(flow, method, result string) {
	if t == nil {
		return
	}
	t.mfa.With(flow, method, result).Inc()
}

// Refresh counts one session refresh.
func (t *Metrics) Refresh(result string) {
	if t == nil {
		return
	}
	t.refreshes.With(result).Inc()
}

// PasswordHash records how long one password hash or verify took.
func (t *Metrics) PasswordHash(operation, algorithm string, took time.Duration) {
	if t == nil {
		return
	}
	t.passwordHash.With(operation, algorithm).Observe(took.Seconds())
}

// APIKeyAuth counts one API key authentication.
func (t *Metrics) APIKeyAuth(result string) {
	if t == nil {
		return
	}
	t.apiKeys.With(result).Inc()
}

// OAuth2Token counts one request at the OAuth2 token endpoint.
func (t *Metrics) OAuth2Token(grantType, result string) {
	if t == nil {
		return
	}
	t.oauth2Tokens.With(grantType, result).Inc()
}

// ProviderRequest records how long one third-party call took.
func (t *Metrics) ProviderRequest(kind, result string, took time.Duration) {
	if t == nil {
		return
	}
	t.provider.With(kind, result).Observe(took.Seconds())
}

// RetentionRowsDeleted adds n to the rows the retention sweep removed.
func (t *Metrics) RetentionRowsDeleted(kind string, n int64) {
	if t == nil || n <= 0 {
		return
	}
	t.retentionRows.With(kind).Add(float64(n))
}

// RetentionSweep counts one retention sweep.
func (t *Metrics) RetentionSweep(result string) {
	if t == nil {
		return
	}
	t.retentionSweeps.With(result).Inc()
}

// RetentionJob counts one CRM outbox job reaching an outcome.
func (t *Metrics) RetentionJob(provider, outcome string) {
	if t == nil {
		return
	}
	t.retentionJobs.With(provider, outcome).Inc()
}

// ControlDegraded counts a security control that could not run and let
// the request through.
func (t *Metrics) ControlDegraded(control string) {
	if t == nil {
		return
	}
	t.controlsDegraded.With(control).Inc()
}

// Throttled counts a request a rate limiter refused.
func (t *Metrics) Throttled(limiter string) {
	if t == nil {
		return
	}
	t.throttled.With(limiter).Inc()
}

// RateLimiterError counts a rate limiter that could not answer.
func (t *Metrics) RateLimiterError(limiter string) {
	if t == nil {
		return
	}
	t.limiterErrors.With(limiter).Inc()
}

// AuditRecordFailed counts an audit event the trail refused.
func (t *Metrics) AuditRecordFailed() {
	if t == nil {
		return
	}
	t.auditFailed.Inc()
}

// Event counts one event on the hook bus. The values go through Bounded.
func (t *Metrics) Event(action, resource, outcome string) {
	if t == nil {
		return
	}
	t.events.With(Bounded(action), Bounded(resource), Bounded(outcome)).Inc()
}

// From returns the instruments of an engine handed over as a plugin.Engine,
// or nil when the engine records none (or is a test stub without the
// method). It is how authsome's own plugins reach the sink without the
// plugin.Engine interface growing a method every stub would need.
func From(engine any) *Metrics {
	if src, ok := engine.(interface{ Telemetry() *Metrics }); ok {
		return src.Telemetry()
	}
	return nil
}
