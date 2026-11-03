package sso

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"

	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
)

// The callback's discovery, token exchange and userinfo calls are client
// spans under the request, with no host on them (the issuer is the
// customer's IdP) and nothing from the URL, the code or the verifier. The
// discovery fetch used to run on context.Background() and was invisible.
func TestOIDCCallbackCallsAreClientSpansUnderTheRequest(t *testing.T) {
	rec := telemetrytest.Spans(t)
	op := newFakeOP(t)
	op.nonceToUse = "nonce-ok"
	p := NewOIDCProvider(OIDCConfig{Name: "acme-okta", Issuer: op.srv.URL + "/" + t.Name(), ClientID: "client-acme", RedirectURL: "https://rp.example/cb", HTTPClient: op.srv.Client()})

	member, err := baggage.NewMember("tenant", "acme")
	require.NoError(t, err)
	bag, err := baggage.New(member)
	require.NoError(t, err)
	ctx, parent := telemetry.Start(baggage.ContextWithBaggage(context.Background(), bag), "request")
	_, err = p.HandleCallback(ctx, map[string]string{"code": "code-value-123", "nonce": "nonce-ok", "code_verifier": "verifier-value-456"})
	require.NoError(t, err)
	parent.End()

	var calls []string
	for _, s := range telemetrytest.Authsome(rec) {
		if s.SpanKind() != trace.SpanKindClient {
			continue
		}
		calls = append(calls, s.Name())
		assert.Equal(t, parent.SpanContext().SpanID(), s.Parent().SpanID())
		attrs := telemetrytest.Attrs(s)
		assert.Equal(t, "oidc", attrs["authsome.provider.kind"])
		assert.NotContains(t, attrs, "server.address")
		text := telemetrytest.Text(s)
		for _, leaked := range []string{"code-value-123", "verifier-value-456", "client-acme", "acme-okta", strings.TrimPrefix(op.srv.URL, "http://"), "/token", "/me", "well-known"} {
			assert.NotContains(t, text, leaked)
		}
	}
	assert.Equal(t, []string{"GET", "POST", "GET"}, calls, "discovery, token exchange, userinfo")
}
