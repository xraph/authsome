package retention

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel/trace"

	"github.com/xraph/authsome/internal/telemetry"
)

// Span names the outbox worker starts. "deliver retention jobs" and "purge
// retention outbox" are root spans, one per tick that found work; "deliver
// retention job" is a child per claimed job.
const (
	spanDeliverJobs = "deliver retention jobs"
	spanDeliverJob  = "deliver retention job"
	spanPurgeOutbox = "purge retention outbox"
)

// jobKindLabel clamps a job's kind to the two the plugin enqueues.
func jobKindLabel(kind string) string {
	switch kind {
	case KindContactUpsert, KindActivityLog:
		return kind
	default:
		return "other"
	}
}

// providerLabel names the kind of CRM a job is bound for: the provider's
// type, never the name an operator gave it.
func (w *worker) providerLabel(j *Job) string {
	if w.deps.Providers == nil {
		return telemetry.CRMOther
	}
	switch w.deps.Providers.Load()[j.Provider].(type) {
	case *HubSpotProvider:
		return telemetry.CRMHubSpot
	case *GenericProvider:
		return telemetry.CRMGeneric
	default:
		return telemetry.CRMOther
	}
}

// outcome records where a job ended up on its span and its counter.
func (w *worker) outcome(ctx context.Context, j *Job, outcome string) {
	trace.SpanFromContext(ctx).SetAttributes(telemetry.AttrRetentionOutcome.String(outcome))
	w.deps.Tel.RetentionJob(w.providerLabel(j), outcome)
}

// instrumentProviders wraps each built-in provider's HTTP client so every
// CRM call is a client span under its job and an observation in the
// provider latency histogram. HubSpot's own API host is a vendor's and is
// recorded. A HubSpot base URL someone overrode can point anywhere, and a
// generic provider's URL is the customer's, so neither host is.
func instrumentProviders(providers map[string]Provider, tel *telemetry.Metrics) {
	for _, p := range providers {
		switch prov := p.(type) {
		case *HubSpotProvider:
			prov.client = instrumentCRMClient(prov.client, prov.baseURL != hubspotDefaultBaseURL, tel)
		case *GenericProvider:
			prov.client = instrumentCRMClient(prov.client, true, tel)
		}
	}
}

func instrumentCRMClient(c *http.Client, hideHost bool, tel *telemetry.Metrics) *http.Client {
	return telemetry.InstrumentClient(c, telemetry.Endpoint{Kind: telemetry.ProviderCRM, HideHost: hideHost}, tel)
}
