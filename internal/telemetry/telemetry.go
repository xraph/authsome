// Package telemetry holds authsome's tracing and metric plumbing: the tracer
// every authsome span is started on, the helpers that mark a span failed
// without ever putting error text on it, the bounded values metric labels
// take, and the instruments recorded through a go-utils metrics sink.
//
// Nothing here reads a request body, a credential or an identifier. Callers
// pass enums and counts; the helpers refuse to take anything else.
package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ScopeName is the instrumentation scope on every span authsome starts.
const ScopeName = "github.com/xraph/authsome"

// Span attribute keys. Every value is an enum, a count or a boolean. None of
// them carries an ID, an address, a credential or text a caller supplied.
const (
	AttrResult            = attribute.Key("authsome.result")
	AttrAuthMethod        = attribute.Key("authsome.auth.method")
	AttrMFARequired       = attribute.Key("authsome.mfa.required")
	AttrMFAVerified       = attribute.Key("authsome.mfa.verified")
	AttrSessionFormat     = attribute.Key("authsome.session.format")
	AttrRefreshVia        = attribute.Key("authsome.refresh.via")
	AttrPasswordAlgorithm = attribute.Key("authsome.password.algorithm")
	AttrPasswordOperation = attribute.Key("authsome.password.operation")
	AttrPluginName        = attribute.Key("authsome.plugin.name")
	AttrPluginHook        = attribute.Key("authsome.plugin.hook")
	AttrProviderKind      = attribute.Key("authsome.provider.kind")
	AttrMessageChannel    = attribute.Key("authsome.message.channel")
	AttrRetentionRemoved  = attribute.Key("authsome.retention.removed")
	AttrRetentionKind     = attribute.Key("authsome.retention.kind")
	AttrRetentionJobs     = attribute.Key("authsome.retention.jobs")
	AttrRetentionJobKind  = attribute.Key("authsome.retention.job_kind")
	AttrRetentionProvider = attribute.Key("authsome.retention.provider")
	AttrRetentionOutcome  = attribute.Key("authsome.retention.outcome")
	AttrJWKSSets          = attribute.Key("authsome.jwks.sets")
	AttrJWKSFailed        = attribute.Key("authsome.jwks.failed")

	// The OTel semantic convention keys the client spans use.
	AttrErrorType      = attribute.Key("error.type")
	AttrExceptionType  = attribute.Key("exception.type")
	AttrHTTPMethod     = attribute.Key("http.request.method")
	AttrHTTPStatusCode = attribute.Key("http.response.status_code")
	AttrServerAddress  = attribute.Key("server.address")
	AttrServerPort     = attribute.Key("server.port")
)

// Start starts a span on the global tracer provider, which forge.New
// installs. The tracer is looked up on every call, so a provider installed
// after the engine was built still gets the spans.
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(ScopeName).Start(ctx, name, opts...)
}

// StartRoot starts the root span of one unit of background work. Pass it a
// ctx that went through Detach, so nothing the unit does before or after
// the span lands in a trace that ended long ago.
func StartRoot(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Start(ctx, name, append(opts, trace.WithNewRoot())...)
}

// Detach returns ctx without the span it carries, keeping its values and
// its cancellation.
//
// A goroutine that outlives the call that started it must not keep that
// call's span: forge runs every extension's Start inside a "start <ext>"
// span, and grove opens a db.client span under any valid span context,
// even an ended one. Without this every query an idle loop makes would land
// in a boot trace that finished at startup.
func Detach(ctx context.Context) context.Context {
	return trace.ContextWithSpanContext(ctx, trace.SpanContext{})
}

// Fail marks span failed with class as its error.type and status
// description, and adds an exception event carrying only the Go type of
// err's innermost error. err's text never reaches the span: store, driver
// and provider errors quote the values they were given.
func Fail(span trace.Span, class string, err error) {
	if !span.IsRecording() {
		return
	}
	span.SetAttributes(AttrErrorType.String(class))
	if err != nil {
		span.AddEvent("exception", trace.WithAttributes(AttrExceptionType.String(TypeOf(err))))
	}
	span.SetStatus(codes.Error, class)
}

// FailErr is Fail with the error's own type as its class.
func FailErr(span trace.Span, err error) {
	if err == nil {
		return
	}
	Fail(span, TypeOf(err), err)
}

// TypeOf names an error by class: a context error by its package name,
// anything else by the Go type of the innermost error in its chain. A
// joined error reports its own type.
func TypeOf(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded):
		return "context.DeadlineExceeded"
	case errors.Is(err, context.Canceled):
		return "context.Canceled"
	}
	for {
		next := errors.Unwrap(err)
		if next == nil {
			return fmt.Sprintf("%T", err)
		}
		err = next
	}
}
