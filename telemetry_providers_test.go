package authsome_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
)

func TestMailerAndSMSAreTracedAndMeasured(t *testing.T) {
	rec := telemetrytest.Spans(t)
	reg, scrape := telemetrytest.Registry(t)
	smsErr := errors.New("twilio: 21211 invalid 'To' phone number +15550100")
	eng, _ := newTestEngine(t,
		authsome.WithMetricsSink(reg),
		authsome.WithMailer(bridge.MailerFunc(func(context.Context, *bridge.EmailMessage) error { return nil })),
		authsome.WithSMSSender(smsFunc(func(context.Context, *bridge.SMSMessage) error { return smsErr })),
	)
	ctx, parent := telemetry.Start(context.Background(), "request")
	require.NoError(t, eng.Mailer().SendEmail(ctx, &bridge.EmailMessage{To: []string{"alice@example.com"}, Subject: "code 492817"}))
	require.ErrorIs(t, eng.SMSSender().SendSMS(ctx, &bridge.SMSMessage{To: "+15550100", Body: "code 492817"}), smsErr)
	parent.End()

	spans := telemetrytest.Authsome(rec)
	email := telemetrytest.Named(spans, "send email")
	sms := telemetrytest.Named(spans, "send sms")
	require.Len(t, email, 1)
	require.Len(t, sms, 1)
	assert.Equal(t, parent.SpanContext().SpanID(), email[0].Parent().SpanID())
	assert.Equal(t, codes.Unset, email[0].Status().Code)
	assert.Equal(t, codes.Error, sms[0].Status().Code)
	for _, s := range append(email, sms...) {
		text := telemetrytest.Text(s)
		for _, leaked := range []string{"alice@example.com", "492817", "+15550100", "21211"} {
			assert.NotContains(t, text, leaked)
		}
	}
	body := scrape()
	assert.Contains(t, body, `forge_authsome_provider_request_duration_seconds_count{kind="email",result="ok"} 1`)
	assert.Contains(t, body, `forge_authsome_provider_request_duration_seconds_count{kind="sms",result="error"} 1`)

	// The no-op mailer is handed back as it is: nothing was sent, so
	// nothing is measured.
	noop, _ := newTestEngine(t, authsome.WithMailer(bridge.NewNoopMailer(nil)))
	_, isNoop := noop.Mailer().(*bridge.NoopMailer)
	assert.True(t, isNoop)
}

type smsFunc func(context.Context, *bridge.SMSMessage) error

func (f smsFunc) SendSMS(ctx context.Context, msg *bridge.SMSMessage) error { return f(ctx, msg) }
