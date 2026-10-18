package telemetry_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
)

// upstream records the headers each request arrived with.
type upstream struct {
	mu      sync.Mutex
	headers []http.Header
	status  int
}

func (u *upstream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u.mu.Lock()
	u.headers = append(u.headers, r.Header.Clone())
	u.mu.Unlock()
	w.WriteHeader(u.status)
}

// withParent starts a parent span carrying some inbound baggage, the way a
// request ctx does after forge's server middleware extracts it.
func withParent(t *testing.T) (context.Context, trace.Span) {
	t.Helper()
	member, err := baggage.NewMember("tenant", "acme")
	require.NoError(t, err)
	bag, err := baggage.New(member)
	require.NoError(t, err)
	ctx := baggage.ContextWithBaggage(context.Background(), bag)
	return telemetry.Start(ctx, "parent")
}

func TestProviderTransportRecordsAClientSpanWithoutTheURL(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	m := telemetry.NewMetrics(reg)

	up := &upstream{status: http.StatusOK}
	srv := httptest.NewServer(up)
	defer srv.Close()

	client := telemetry.InstrumentClient(&http.Client{Timeout: 5 * time.Second},
		telemetry.Endpoint{Kind: telemetry.ProviderBreach}, m)

	ctx, parent := withParent(t)
	// The path and query carry what a real breach-check or SMS call would:
	// a password hash prefix, an account SID, a token.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		srv.URL+"/range/5BAA6?token=sk-secret-query-value", http.NoBody)
	require.NoError(t, err)
	// A caller (or an instrumented layer) that set propagation headers
	// still sends none of them to the third party.
	req.Header.Set("traceparent", "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01")
	req.Header.Set("baggage", "tenant=acme")
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	parent.End()

	require.Len(t, up.headers, 1)
	for _, h := range []string{"Traceparent", "Tracestate", "Baggage"} {
		assert.Empty(t, up.headers[0].Get(h), "%s reached the third party", h)
	}
	assert.Equal(t, "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01", req.Header.Get("traceparent"),
		"the caller's own request is not modified")

	spans := telemetrytest.Authsome(rec)
	require.Len(t, telemetrytest.Named(spans, "GET"), 1)
	span := telemetrytest.Named(spans, "GET")[0]
	assert.Equal(t, trace.SpanKindClient, span.SpanKind())
	assert.Equal(t, parent.SpanContext().SpanID(), span.Parent().SpanID(), "the client span hangs off the caller's span")

	attrs := telemetrytest.Attrs(span)
	host, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"http.request.method":       "GET",
		"authsome.provider.kind":    "breach",
		"server.address":            host,
		"server.port":               port,
		"http.response.status_code": "200",
	}, attrs)
	text := telemetrytest.Text(span)
	for _, leaked := range []string{"5BAA6", "/range", "sk-secret-query-value", "token="} {
		assert.NotContains(t, text, leaked)
	}

	assert.Contains(t, scrape(), `forge_authsome_provider_request_duration_seconds_count{kind="breach",result="ok"} 1`)
}

func TestProviderTransportHidesACustomersHostAndMarksFailures(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	m := telemetry.NewMetrics(reg)

	up := &upstream{status: http.StatusServiceUnavailable}
	srv := httptest.NewServer(up)
	defer srv.Close()

	client := telemetry.InstrumentClient(nil, telemetry.Endpoint{Kind: telemetry.ProviderCRM, HideHost: true}, m)
	ctx, parent := withParent(t)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/hooks/contact", http.NoBody)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	parent.End()

	span := telemetrytest.Named(telemetrytest.Authsome(rec), "POST")[0]
	attrs := telemetrytest.Attrs(span)
	assert.NotContains(t, attrs, "server.address")
	assert.NotContains(t, attrs, "server.port")
	assert.Equal(t, "503", attrs["error.type"])
	assert.Equal(t, codes.Error, span.Status().Code)
	assert.Equal(t, "503", span.Status().Description)
	assert.Contains(t, scrape(), `forge_authsome_provider_request_duration_seconds_count{kind="crm",result="5xx"} 1`)
}

func TestProviderTransportStartsNoTraceWithoutAParent(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	m := telemetry.NewMetrics(reg)

	srv := httptest.NewServer(&upstream{status: http.StatusNotFound})
	defer srv.Close()

	client := telemetry.InstrumentClient(nil, telemetry.Endpoint{Kind: telemetry.ProviderSAMLMetadata, HideHost: true}, m)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, http.NoBody)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	assert.Empty(t, rec.Ended(), "a call made outside any request mints no trace")
	assert.Contains(t, scrape(), `forge_authsome_provider_request_duration_seconds_count{kind="saml_metadata",result="4xx"} 1`)
}

func TestProviderTransportClassifiesTransportFailures(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	m := telemetry.NewMetrics(reg)

	// A listener that is closed at once refuses the connection.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	client := telemetry.InstrumentClient(&http.Client{Timeout: 2 * time.Second},
		telemetry.Endpoint{Kind: telemetry.ProviderSocial}, m)
	ctx, parent := withParent(t)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/oauth/token?code=abc", http.NoBody)
	require.NoError(t, err)
	_, err = client.Do(req) //nolint:bodyclose // the request fails, there is no body
	require.Error(t, err)
	parent.End()

	span := telemetrytest.Named(telemetrytest.Authsome(rec), "GET")[0]
	attrs := telemetrytest.Attrs(span)
	assert.Equal(t, "connection", attrs["error.type"])
	assert.Equal(t, "connection", span.Status().Description)
	require.Len(t, span.Events(), 1)
	assert.Equal(t, "exception", span.Events()[0].Name)
	assert.NotContains(t, telemetrytest.Text(span), addr, "the transport's error text, which quotes the URL, stays off the span")
	assert.NotContains(t, telemetrytest.Text(span), "code=abc")
	assert.Contains(t, scrape(), `forge_authsome_provider_request_duration_seconds_count{kind="social",result="error"} 1`)
}

func TestCallClass(t *testing.T) {
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	canceled, cancel2 := context.WithCancel(context.Background())
	cancel2()

	assert.Equal(t, telemetry.ClassTimeout, telemetry.CallClass(expired, errors.New("x")))
	assert.Equal(t, telemetry.ClassCanceled, telemetry.CallClass(canceled, errors.New("x")))
	assert.Equal(t, telemetry.ClassDNS, telemetry.CallClass(context.Background(), &net.DNSError{Err: "no such host", Name: "x"}))
	assert.Equal(t, telemetry.ClassOther, telemetry.CallClass(context.Background(), errors.New("x")))
}
