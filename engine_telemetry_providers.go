package authsome

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/internal/telemetry"
)

// Span names for outgoing messages. Both are SpanKindInternal: the
// provider's own HTTP call, where there is one, is the client span under
// them.
const (
	spanSendEmail = "send email"
	spanSendSMS   = "send sms"
)

// isBreached asks HIBP whether password is in a known breach, through a
// client that measures the call and traces it as a child of the span in
// ctx. The lookup runs on context.WithoutCancel(ctx): the request's values
// travel with it (its span, so the call shows in the request's trace) but
// not its cancellation or deadline, which is exactly how it behaved on
// context.Background(). The client's own timeout is still the only limit.
// The request path carries the password hash prefix, and the client span
// never records a path.
func (e *Engine) isBreached(ctx context.Context, password string) (bool, error) {
	client := telemetry.InstrumentClient(&http.Client{Timeout: account.BreachCheckTimeout},
		telemetry.Endpoint{Kind: telemetry.ProviderBreach}, e.tel)
	return account.NewBreachCheckerWithClient(client).IsBreachedContext(context.WithoutCancel(ctx), password)
}

// ──────────────────────────────────────────────────
// Messaging
// ──────────────────────────────────────────────────

// passwordBreached is isBreached for a path that fails open: a lookup that
// could not run reports false and is counted as a degraded breach_check
// control, so you can see the check was skipped rather than passed.
func (e *Engine) passwordBreached(ctx context.Context, tenant, password string) bool {
	breached, err := e.isBreached(ctx, password)
	if err != nil {
		e.controlDegraded(telemetry.ControlBreachCheck, tenant)
	}
	return breached
}

// tracedMailer measures and traces every email the engine's plugins send,
// whichever Mailer is configured.
type tracedMailer struct {
	next bridge.Mailer
	tel  *telemetry.Metrics
}

func (m tracedMailer) SendEmail(ctx context.Context, msg *bridge.EmailMessage) error {
	ctx, span := telemetry.Start(ctx, spanSendEmail,
		trace.WithAttributes(telemetry.AttrMessageChannel.String(telemetry.ProviderEmail)))
	defer span.End()
	start := time.Now()
	err := m.next.SendEmail(ctx, msg)
	m.tel.ProviderRequest(telemetry.ProviderEmail, messageResult(ctx, span, err), time.Since(start))
	return err
}

// tracedSMSSender is tracedMailer for SMS.
type tracedSMSSender struct {
	next bridge.SMSSender
	tel  *telemetry.Metrics
}

func (s tracedSMSSender) SendSMS(ctx context.Context, msg *bridge.SMSMessage) error {
	ctx, span := telemetry.Start(ctx, spanSendSMS,
		trace.WithAttributes(telemetry.AttrMessageChannel.String(telemetry.ProviderSMS)))
	defer span.End()
	start := time.Now()
	err := s.next.SendSMS(ctx, msg)
	s.tel.ProviderRequest(telemetry.ProviderSMS, messageResult(ctx, span, err), time.Since(start))
	return err
}

// messageResult marks span failed when err is set and returns the provider
// result label for it. A provider's error text quotes addresses and codes,
// so only its type reaches the span.
func messageResult(ctx context.Context, span trace.Span, err error) string {
	if err == nil {
		return telemetry.ProviderOK
	}
	switch telemetry.CallClass(ctx, err) {
	case telemetry.ClassTimeout:
		telemetry.Fail(span, telemetry.ClassTimeout, err)
		return telemetry.ProviderTimeout
	case telemetry.ClassCanceled:
		telemetry.Fail(span, telemetry.ClassCanceled, err)
		return telemetry.ProviderCanceled
	default:
		telemetry.FailErr(span, err)
		return telemetry.ProviderError
	}
}
