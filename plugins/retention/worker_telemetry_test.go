package retention

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
)

// claimSpy records whether each ClaimDue ran under a span.
type claimSpy struct {
	Store
	mu      sync.Mutex
	valid   []bool
	claimed chan struct{}
}

func (c *claimSpy) ClaimDue(ctx context.Context, limit int, lease time.Duration, now time.Time) ([]*Job, error) {
	c.mu.Lock()
	c.valid = append(c.valid, trace.SpanContextFromContext(ctx).IsValid())
	c.mu.Unlock()
	defer func() { c.claimed <- struct{}{} }()
	return c.Store.ClaimDue(ctx, limit, lease, now)
}

// An idle tick claims nothing: its query runs outside any trace, and it
// starts no span, so a quiet outbox mints no traces every 30 seconds.
func TestWorkerIdleTickStartsNoTrace(t *testing.T) {
	rec := telemetrytest.Spans(t)
	spy := &claimSpy{Store: NewMemoryStore(), claimed: make(chan struct{}, 1)}
	w := newTestWorker(t, spy, &fakeProvider{caps: CapContacts})
	tick := make(chan time.Time)
	w.startWith(tick)
	t.Cleanup(w.stop)

	tick <- time.Now()
	select {
	case <-spy.claimed:
	case <-time.After(5 * time.Second):
		t.Fatal("the tick never claimed")
	}
	w.stop()

	spy.mu.Lock()
	defer spy.mu.Unlock()
	assert.Equal(t, []bool{false}, spy.valid)
	assert.Empty(t, telemetrytest.Authsome(rec))
}

// A tick that claims jobs is one root span with a child per job, each
// carrying its outcome, and each outcome is counted by provider kind.
func TestWorkerTracesAClaimedBatchAndCountsOutcomes(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	ctx := context.Background()
	s := NewMemoryStore()
	p := &fakeProvider{
		caps:      CapContacts | CapActivities,
		upsertErr: []error{nil, errors.New("400 invalid email for alice@example.com")},
	}
	enqueued(t, s, KindContactUpsert, "t1")
	enqueued(t, s, KindContactUpsert, "t2")

	w := newTestWorker(t, s, p)
	w.deps.Tel = telemetry.NewMetrics(reg)
	w.runOnce(ctx)

	spans := telemetrytest.Authsome(rec)
	roots := telemetrytest.Named(spans, spanDeliverJobs)
	require.Len(t, roots, 1)
	assert.False(t, roots[0].Parent().IsValid())
	assert.Equal(t, "2", telemetrytest.Attrs(roots[0])["authsome.retention.jobs"])

	jobs := telemetrytest.Named(spans, spanDeliverJob)
	require.Len(t, jobs, 2)
	outcomes := map[string]int{}
	for _, j := range jobs {
		assert.Equal(t, roots[0].SpanContext().SpanID(), j.Parent().SpanID())
		attrs := telemetrytest.Attrs(j)
		assert.Equal(t, KindContactUpsert, attrs["authsome.retention.job_kind"])
		assert.Equal(t, "other", attrs["authsome.retention.provider"], "the operator's name for the provider is never recorded")
		outcomes[attrs["authsome.retention.outcome"]]++
		assert.NotContains(t, telemetrytest.Text(j), "alice@example.com")
		assert.NotContains(t, telemetrytest.Text(j), "invalid email")
		if attrs["authsome.retention.outcome"] == telemetry.JobDead {
			assert.Equal(t, codes.Error, j.Status().Code)
		}
	}
	assert.Equal(t, map[string]int{telemetry.JobDone: 1, telemetry.JobDead: 1}, outcomes)

	body := scrape()
	assert.Contains(t, body, `forge_authsome_retention_jobs_processed_total{outcome="done",provider="other"} 1`)
	assert.Contains(t, body, `forge_authsome_retention_jobs_processed_total{outcome="dead",provider="other"} 1`)
}

func TestProviderLabelNamesTheTypeNotTheInstance(t *testing.T) {
	hub, err := NewHubSpotProvider(ProviderConfig{Name: "acme-prod-hubspot", Token: "x"})
	require.NoError(t, err)
	gen, err := NewGenericProvider(ProviderConfig{Name: "acme-crm", ContactURL: "https://crm.acme.example/contacts"})
	require.NoError(t, err)
	w := newWorker(workerDeps{Providers: newProviderRegistry(map[string]Provider{
		"acme-prod-hubspot": hub, "acme-crm": gen, "fake": &fakeProvider{},
	})})

	assert.Equal(t, telemetry.CRMHubSpot, w.providerLabel(&Job{Provider: "acme-prod-hubspot"}))
	assert.Equal(t, telemetry.CRMGeneric, w.providerLabel(&Job{Provider: "acme-crm"}))
	assert.Equal(t, telemetry.CRMOther, w.providerLabel(&Job{Provider: "fake"}))
	assert.Equal(t, telemetry.CRMOther, w.providerLabel(&Job{Provider: "gone"}))
}

// HubSpot's own API host is a vendor's and goes on the client span. An
// overridden base URL can point anywhere, so its host doesn't.
func TestHubSpotHostIsHiddenWhenTheBaseURLIsOverridden(t *testing.T) {
	rec := telemetrytest.Spans(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	defer srv.Close()

	call := func(p *HubSpotProvider) map[string]string {
		t.Helper()
		instrumentProviders(map[string]Provider{"h": p}, nil)
		ctx, parent := telemetry.Start(context.Background(), "job")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, http.NoBody)
		require.NoError(t, err)
		resp, err := p.client.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		parent.End()
		got := telemetrytest.Named(telemetrytest.Authsome(rec), "GET")
		return telemetrytest.Attrs(got[len(got)-1])
	}

	vendor, err := NewHubSpotProvider(ProviderConfig{Name: "hs", Token: "x"})
	require.NoError(t, err)
	assert.Contains(t, call(vendor), "server.address", "the default HubSpot host is recorded")

	custom, err := NewHubSpotProvider(ProviderConfig{Name: "hs", Token: "x", BaseURL: "https://crm-proxy.acme.example/"})
	require.NoError(t, err)
	assert.NotContains(t, call(custom), "server.address", "an overridden base URL's host is not")
}

// fieldLogger keeps every Warn with its fields.
type fieldLogger struct {
	log.Logger
	mu    sync.Mutex
	warns []string
}

func (l *fieldLogger) Warn(msg string, fields ...log.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	line := msg
	for _, f := range fields {
		line += " " + f.Key() + "=" + fmt.Sprint(f.Value())
	}
	l.warns = append(l.warns, line)
}

// A dead-lettered job is logged by ID, error type and status class. The
// CRM's own error text, which can quote the contact's email back, stays in
// the job row only.
func TestDeadLetterLogLeavesTheCRMResponseInTheJobRow(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()
	crmBody := `{"message":"Contact alice@example.com already exists","status":"error"}`
	p := &fakeProvider{
		caps: CapContacts | CapActivities,
		upsertErr: []error{&ProviderError{
			Err:    fmt.Errorf("422 Unprocessable Entity: %s", crmBody),
			Status: http.StatusUnprocessableEntity,
		}},
	}
	j := enqueued(t, s, KindContactUpsert, "dl1")

	logger := &fieldLogger{Logger: log.NewNoopLogger()}
	w := newTestWorker(t, s, p)
	w.deps.Logger = logger
	w.runOnce(ctx)

	stored, err := s.GetJob(ctx, j.ID)
	require.NoError(t, err)
	require.Equal(t, StateDead, stored.State)
	assert.Contains(t, stored.LastError, "alice@example.com", "the full text is kept where an operator investigates the job")

	require.Len(t, logger.warns, 1)
	line := logger.warns[0]
	assert.Contains(t, line, "retention: job dead-lettered")
	assert.Contains(t, line, "job_id="+j.ID.String())
	assert.Contains(t, line, "status_class=4xx")
	assert.Contains(t, line, "error_type=")
	assert.NotContains(t, line, "alice@example.com")
	assert.NotContains(t, line, "already exists")
}
