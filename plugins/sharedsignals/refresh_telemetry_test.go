package sharedsignals

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/internal/telemetry/telemetrytest"
	"github.com/xraph/authsome/plugins/sharedsignals/jwksclient"
)

// A background round with nothing due starts no span. A round with a key
// set due is one root span, and the fetch under it is a client span with
// no host and no URL on it: the JWKS URI is the transmitter's.
func TestRefreshKeysTracesOnlyARoundWithWork(t *testing.T) {
	rec := telemetrytest.Spans(t)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	body, err := json.Marshal(map[string]any{"keys": []map[string]string{{
		"kty": "RSA", "kid": "kid-1",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}})
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	defer srv.Close()

	now := time.Now()
	p := &Plugin{
		logger: log.NewNoopLogger(),
		jwks: jwksclient.New(jwksclient.Options{
			HTTPClient: telemetry.InstrumentClient(srv.Client(),
				telemetry.Endpoint{Kind: telemetry.ProviderJWKS, HideHost: true}, nil),
			MinRefetchInterval: 5 * time.Minute,
			Now:                func() time.Time { return now },
			ValidateURI:        func(string) error { return nil },
		}),
	}
	jwksURI := srv.URL + "/stream/acme/jwks.json"

	// Cold cache, then a set fetched just now: neither round has work.
	p.refreshKeys(context.Background())
	_, err = p.jwks.Key(context.Background(), jwksURI, "kid-1")
	require.NoError(t, err)
	p.refreshKeys(context.Background())
	assert.Empty(t, telemetrytest.Authsome(rec), "an idle round mints no trace")

	now = now.Add(10 * time.Minute)
	p.refreshKeys(context.Background())

	spans := telemetrytest.Authsome(rec)
	rounds := telemetrytest.Named(spans, spanRefreshStreamKeys)
	require.Len(t, rounds, 1)
	assert.False(t, rounds[0].Parent().IsValid())
	assert.Equal(t, map[string]string{"authsome.jwks.sets": "1", "authsome.jwks.failed": "0"}, telemetrytest.Attrs(rounds[0]))

	fetches := telemetrytest.Named(spans, "GET")
	require.Len(t, fetches, 1)
	assert.Equal(t, trace.SpanKindClient, fetches[0].SpanKind())
	assert.Equal(t, rounds[0].SpanContext().SpanID(), fetches[0].Parent().SpanID())
	text := telemetrytest.Text(fetches[0])
	assert.NotContains(t, text, "jwks.json")
	assert.NotContains(t, text, strings.TrimPrefix(srv.URL, "http://"))
}
