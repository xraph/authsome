package telemetry

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

// Endpoint describes a third-party call for ProviderTransport. Nothing in
// it comes from the request URL, which is why a span can't leak the
// password hash prefix in a breach-check path, an account SID in an SMS
// path, or a token in a query string.
type Endpoint struct {
	// Kind is the provider kind, one of the Provider* constants. It is
	// recorded as authsome.provider.kind and as the kind label.
	Kind string

	// HideHost keeps server.address and server.port off the span. Set it
	// when a customer supplies the URL (an SSO issuer, a SAML metadata URL,
	// a webhook or a generic CRM endpoint): the host is then theirs, not a
	// vendor's.
	HideHost bool
}

// propagationHeaders are the headers a trace propagator writes. A request
// to a third party goes out without them.
var propagationHeaders = []string{"traceparent", "tracestate", "baggage"}

// ProviderTransport wraps base (http.DefaultTransport when nil) so every
// request it sends is measured, and traced under the span in the request's
// context.
//
// The client span is named after the HTTP method alone and carries
// http.request.method, authsome.provider.kind, server.address and
// server.port (unless ep.HideHost), and http.response.status_code. It never
// carries the URL, its path or its query, or any header. A 4xx or 5xx
// answer, or a transport failure, marks it failed with error.type set to
// the status code or to timeout, canceled, dns, tls, connection or other.
// The error's text never reaches it. With no span in the request's context
// it records the duration and starts no span, so a call made outside any
// request does not mint a trace of its own.
//
// The request goes out with exactly the headers its caller set: no
// traceparent, tracestate or baggage. Providers are third parties, and an
// inbound request's baggage is not theirs to see.
//
// m may be nil, in which case only the span is recorded.
func ProviderTransport(base http.RoundTripper, ep Endpoint, m *Metrics) http.RoundTripper {
	return &providerTransport{base: base, ep: ep, m: m}
}

// InstrumentClient returns a copy of c whose transport is c's own wrapped
// in ProviderTransport. A nil c stands for http.DefaultClient. The copy
// keeps c's timeout, redirect policy and cookie jar.
func InstrumentClient(c *http.Client, ep Endpoint, m *Metrics) *http.Client {
	if c == nil {
		c = http.DefaultClient
	}
	out := *c
	out.Transport = ProviderTransport(c.Transport, ep, m)
	return &out
}

type providerTransport struct {
	base http.RoundTripper
	ep   Endpoint
	m    *Metrics
}

// RoundTrip implements http.RoundTripper.
func (t *providerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	ctx := baggage.ContextWithoutBaggage(req.Context())

	// With no parent there is no span: SpanFromContext on a ctx without one
	// is the no-op span, and every call below is free on it.
	span := trace.SpanFromContext(context.Background())
	if trace.SpanContextFromContext(ctx).IsValid() {
		method := knownMethod(req.Method)
		attrs := []attribute.KeyValue{
			AttrHTTPMethod.String(method),
			AttrProviderKind.String(t.ep.Kind),
		}
		if !t.ep.HideHost && req.URL != nil {
			attrs = append(attrs, AttrServerAddress.String(req.URL.Hostname()))
			if p, err := strconv.Atoi(req.URL.Port()); err == nil {
				attrs = append(attrs, AttrServerPort.Int(p))
			}
		}
		ctx, span = Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient), trace.WithAttributes(attrs...))
	}
	defer span.End()

	out := req.WithContext(ctx)
	if hasPropagation(req.Header) {
		out = req.Clone(ctx)
		for _, h := range propagationHeaders {
			out.Header.Del(h)
		}
	}

	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(out)
	if err != nil {
		class := CallClass(ctx, err)
		Fail(span, class, err)
		t.m.ProviderRequest(t.ep.Kind, providerResult(class), time.Since(start))
		return nil, err
	}
	span.SetAttributes(AttrHTTPStatusCode.Int(resp.StatusCode))
	result := ProviderOK
	if resp.StatusCode >= http.StatusBadRequest {
		Fail(span, strconv.Itoa(resp.StatusCode), nil)
		result = Provider4xx
		if resp.StatusCode >= http.StatusInternalServerError {
			result = Provider5xx
		}
	}
	t.m.ProviderRequest(t.ep.Kind, result, time.Since(start))
	return resp, nil
}

func hasPropagation(h http.Header) bool {
	for _, k := range propagationHeaders {
		if h.Get(k) != "" {
			return true
		}
	}
	return false
}

// providerResult maps a transport failure class onto the result label.
func providerResult(class string) string {
	switch class {
	case ClassTimeout:
		return ProviderTimeout
	case ClassCanceled:
		return ProviderCanceled
	default:
		return ProviderError
	}
}

// knownMethod returns m when it is a standard HTTP method and "_OTHER"
// otherwise, as the HTTP semantic conventions ask.
func knownMethod(m string) string {
	switch m {
	case "":
		return http.MethodGet
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
		return m
	default:
		return "_OTHER"
	}
}

// Transport failure classes. These are the only values CallClass returns.
const (
	ClassTimeout    = "timeout"
	ClassCanceled   = "canceled"
	ClassDNS        = "dns"
	ClassTLS        = "tls"
	ClassConnection = "connection"
	ClassOther      = "other"
)

// CallClass sorts a transport failure under ctx into timeout, canceled,
// dns, tls, connection or other. A deadline that has passed or a canceled
// ctx decides the class whatever err says, since http.Client's own timeout
// can surface as an error that is neither a context error nor a net.Error.
func CallClass(ctx context.Context, err error) string {
	if dl, ok := ctx.Deadline(); ok && !time.Now().Before(dl) {
		return ClassTimeout
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return ClassCanceled
	}
	switch {
	case errors.Is(err, context.Canceled):
		return ClassCanceled
	case errors.Is(err, context.DeadlineExceeded):
		return ClassTimeout
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsTimeout {
			return ClassTimeout
		}
		return ClassDNS
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ClassTimeout
	}
	if isTLS(err) {
		return ClassTLS
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) ||
		errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return ClassConnection
	}
	return ClassOther
}

func isTLS(err error) bool {
	var (
		verifyErr   *tls.CertificateVerificationError
		recordErr   tls.RecordHeaderError
		alertErr    tls.AlertError
		unknownAuth x509.UnknownAuthorityError
		hostnameErr x509.HostnameError
		invalidCert x509.CertificateInvalidError
	)
	return errors.As(err, &verifyErr) || errors.As(err, &recordErr) || errors.As(err, &alertErr) ||
		errors.As(err, &unknownAuth) || errors.As(err, &hostnameErr) || errors.As(err, &invalidCert)
}
