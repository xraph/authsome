// Package telemetrytest holds the helpers authsome's telemetry tests share:
// a real Forge metrics registry with its Prometheus scrape, and a span
// recorder installed as the global tracer provider. Only tests import it.
package telemetrytest

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/xraph/forge"
	gometrics "github.com/xraph/go-utils/metrics"

	"github.com/xraph/authsome/internal/telemetry"
)

// Registry returns a Forge app's metrics registry and a func that scrapes
// it the way /_/metrics does.
//
// forge.New installs its own global tracer provider and propagator. This
// puts back whatever was there before, so a recorder from Spans keeps
// receiving spans whichever of the two a test calls first.
func Registry(t testing.TB) (reg gometrics.Metrics, scrape func() string) {
	t.Helper()
	prevTP, prevProp := otel.GetTracerProvider(), otel.GetTextMapPropagator()
	app := forge.New(forge.WithAppName("authsome-telemetry-test"), forge.WithAppLogger(forge.NewNoopLogger()))
	otel.SetTracerProvider(prevTP)
	otel.SetTextMapPropagator(prevProp)
	m := app.Metrics()
	scrape = func() string {
		t.Helper()
		p, ok := m.(interface{ PrometheusHandler() http.Handler })
		if !ok {
			t.Fatalf("app metrics %T have no Prometheus handler", m)
		}
		rr := httptest.NewRecorder()
		p.PrometheusHandler().ServeHTTP(rr, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/_/metrics", http.NoBody))
		body, err := io.ReadAll(rr.Body)
		if err != nil {
			t.Fatalf("read scrape: %v", err)
		}
		return string(body)
	}
	return m, scrape
}

// Series returns the sample lines in a scrape whose metric name starts with
// prefix, without their values and without histogram bucket lines.
func Series(body, prefix string) []string {
	var out []string
	for line := range strings.SplitSeq(body, "\n") {
		if !strings.HasPrefix(line, prefix) || strings.Contains(line, "_bucket{") {
			continue
		}
		if i := strings.LastIndex(line, " "); i > 0 {
			line = line[:i]
		}
		out = append(out, line)
	}
	return out
}

// Spans makes a SpanRecorder the global tracer provider for the rest of
// the test and puts the previous provider back afterwards. Tests that use
// it must not run in parallel.
func Spans(t testing.TB) *tracetest.SpanRecorder {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(prev)
		if err := tp.Shutdown(context.Background()); err != nil {
			t.Errorf("shut down tracer provider: %v", err)
		}
	})
	return rec
}

// Authsome returns the ended spans authsome started, in the order they
// ended.
func Authsome(rec *tracetest.SpanRecorder) []sdktrace.ReadOnlySpan {
	return OnlyAuthsome(rec.Ended())
}

// OnlyAuthsome keeps the spans authsome started.
func OnlyAuthsome(spans []sdktrace.ReadOnlySpan) []sdktrace.ReadOnlySpan {
	var out []sdktrace.ReadOnlySpan
	for _, s := range spans {
		if s.InstrumentationScope().Name == telemetry.ScopeName {
			out = append(out, s)
		}
	}
	return out
}

// Names returns the names of spans.
func Names(spans []sdktrace.ReadOnlySpan) []string {
	out := make([]string, len(spans))
	for i, s := range spans {
		out[i] = s.Name()
	}
	return out
}

// Named returns the spans called name.
func Named(spans []sdktrace.ReadOnlySpan, name string) []sdktrace.ReadOnlySpan {
	var out []sdktrace.ReadOnlySpan
	for _, s := range spans {
		if s.Name() == name {
			out = append(out, s)
		}
	}
	return out
}

// Attrs returns a span's attributes as strings keyed by attribute name.
func Attrs(s sdktrace.ReadOnlySpan) map[string]string {
	out := make(map[string]string, len(s.Attributes()))
	for _, kv := range s.Attributes() {
		out[string(kv.Key)] = kv.Value.String()
	}
	return out
}

// Text flattens everything a span carries that an exporter would ship
// (name, attributes, events and their attributes, status, links) into one
// string, for tests that assert a value appears nowhere on it.
func Text(s sdktrace.ReadOnlySpan) string {
	var b strings.Builder
	b.WriteString(s.Name())
	for _, kv := range s.Attributes() {
		b.WriteString("\n" + string(kv.Key) + "=" + kv.Value.String())
	}
	for _, ev := range s.Events() {
		b.WriteString("\nevent " + ev.Name)
		for _, kv := range ev.Attributes {
			b.WriteString(" " + string(kv.Key) + "=" + kv.Value.String())
		}
	}
	b.WriteString("\nstatus " + s.Status().Code.String() + " " + s.Status().Description)
	for _, l := range s.Links() {
		for _, kv := range l.Attributes {
			b.WriteString("\nlink " + string(kv.Key) + "=" + kv.Value.String())
		}
	}
	for _, kv := range s.Resource().Attributes() {
		b.WriteString("\nresource " + string(kv.Key) + "=" + kv.Value.String())
	}
	return b.String()
}
