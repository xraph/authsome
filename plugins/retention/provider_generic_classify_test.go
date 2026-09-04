package retention

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file covers classifyHTTPError against every row of the "Retry
// classification" table in
// docs/superpowers/specs/2026-09-03-crm-retention-delivery-design.md, plus
// retryAfter's Retry-After parsing on its own. NewGenericProvider builds
// unconditionally now that the policy is decided, so the last two tests
// drive real non-2xx responses through the actual post path end to end.

// fakeResp builds a *http.Response carrying just what classifyHTTPError and
// retryAfter look at: status, status code, and headers. No network involved.
func fakeResp(status int, header http.Header) *http.Response {
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     header,
	}
}

func headerWithRetryAfter(v string) http.Header {
	h := http.Header{}
	h.Set("Retry-After", v)
	return h
}

func TestClassifyHTTPError_Table(t *testing.T) {
	transportErr := errors.New("dial tcp: connection refused")

	tests := []struct {
		name       string
		resp       *http.Response
		err        error
		retryable  bool
		retryAfter time.Duration
		dropRef    bool
	}{
		{
			name:      "no response (transport error)",
			resp:      nil,
			err:       transportErr,
			retryable: true,
		},
		{
			name:       "429 with Retry-After delta-seconds",
			resp:       fakeResp(http.StatusTooManyRequests, headerWithRetryAfter("120")),
			retryable:  true,
			retryAfter: 2 * time.Minute,
		},
		{
			name:       "429 with Retry-After clamped to 30m ceiling",
			resp:       fakeResp(http.StatusTooManyRequests, headerWithRetryAfter("99999")),
			retryable:  true,
			retryAfter: 30 * time.Minute,
		},
		{
			name:       "429 with Retry-After floored to 1s",
			resp:       fakeResp(http.StatusTooManyRequests, headerWithRetryAfter("0")),
			retryable:  true,
			retryAfter: time.Second,
		},
		{
			name:       "429 with garbage Retry-After falls back to backoff",
			resp:       fakeResp(http.StatusTooManyRequests, headerWithRetryAfter("banana")),
			retryable:  true,
			retryAfter: 0,
		},
		{
			name:       "429 with no Retry-After header",
			resp:       fakeResp(http.StatusTooManyRequests, nil),
			retryable:  true,
			retryAfter: 0,
		},
		{
			name:       "401 unauthorized",
			resp:       fakeResp(http.StatusUnauthorized, nil),
			retryable:  true,
			retryAfter: 2 * time.Minute,
		},
		{
			name:       "403 forbidden",
			resp:       fakeResp(http.StatusForbidden, nil),
			retryable:  true,
			retryAfter: 2 * time.Minute,
		},
		{
			name:      "404 drops the ref",
			resp:      fakeResp(http.StatusNotFound, nil),
			retryable: true,
			dropRef:   true,
		},
		{
			name:      "500 internal server error",
			resp:      fakeResp(http.StatusInternalServerError, nil),
			retryable: true,
		},
		{
			name:      "502 bad gateway",
			resp:      fakeResp(http.StatusBadGateway, nil),
			retryable: true,
		},
		{
			name:      "503 service unavailable",
			resp:      fakeResp(http.StatusServiceUnavailable, nil),
			retryable: true,
		},
		{
			name:      "504 gateway timeout",
			resp:      fakeResp(http.StatusGatewayTimeout, nil),
			retryable: true,
		},
		{
			name:      "501 not implemented is terminal",
			resp:      fakeResp(http.StatusNotImplemented, nil),
			retryable: false,
		},
		{
			name:      "400 bad request is terminal",
			resp:      fakeResp(http.StatusBadRequest, nil),
			retryable: false,
		},
		{
			name:      "422 unprocessable entity is terminal",
			resp:      fakeResp(http.StatusUnprocessableEntity, nil),
			retryable: false,
		},
		{
			name:      "413 payload too large is terminal",
			resp:      fakeResp(http.StatusRequestEntityTooLarge, nil),
			retryable: false,
		},
		{
			name:      "408 request timeout",
			resp:      fakeResp(http.StatusRequestTimeout, nil),
			retryable: true,
		},
		{
			name:      "409 conflict",
			resp:      fakeResp(http.StatusConflict, nil),
			retryable: true,
		},
		{
			name:      "418 unrecognised status is terminal",
			resp:      fakeResp(http.StatusTeapot, nil),
			retryable: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pe := classifyHTTPError(tc.resp, []byte("body"), tc.err)
			require.NotNil(t, pe)
			assert.Equal(t, tc.retryable, pe.Retryable, "Retryable")
			assert.Equal(t, tc.retryAfter, pe.RetryAfter, "RetryAfter")
			assert.Equal(t, tc.dropRef, pe.DropRef, "DropRef")
		})
	}
}

func TestClassifyHTTPError_NilResponsePreservesTransportError(t *testing.T) {
	transportErr := errors.New("dial tcp: connection refused")
	pe := classifyHTTPError(nil, nil, transportErr)
	require.NotNil(t, pe)
	assert.ErrorIs(t, pe, transportErr)
}

func TestRetryAfter_HTTPDateForm(t *testing.T) {
	when := time.Now().Add(90 * time.Second).UTC()
	resp := fakeResp(http.StatusTooManyRequests, headerWithRetryAfter(when.Format(http.TimeFormat)))

	got := retryAfter(resp)
	assert.InDelta(t, 90*time.Second, got, float64(5*time.Second),
		"HTTP-date Retry-After must be parsed as time-until, allowing slack for test execution time")
}

func TestClassifyHTTPError_429HTTPDateRetryAfter(t *testing.T) {
	when := time.Now().Add(90 * time.Second).UTC()
	resp := fakeResp(http.StatusTooManyRequests, headerWithRetryAfter(when.Format(http.TimeFormat)))

	pe := classifyHTTPError(resp, nil, nil)
	require.NotNil(t, pe)
	assert.True(t, pe.Retryable)
	assert.InDelta(t, 90*time.Second, pe.RetryAfter, float64(5*time.Second))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", truncate([]byte("hello"), 10), "a body under the cap is untouched")

	big := make([]byte, 1000)
	for i := range big {
		big[i] = 'x'
	}
	got := truncate(big, 512)
	assert.Len(t, got, 512+len("... (truncated)"))
	assert.Contains(t, got, "truncated")
}

// ──────────────────────────────────────────────────
// End to end: the classification arrives intact through the real post path.
// ──────────────────────────────────────────────────

func TestGenericProvider_UpsertContact_404EndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"contact not found"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{Name: "acme", ContactURL: srv.URL})
	require.NoError(t, err)

	_, err = p.UpsertContact(t.Context(), &Contact{Email: "x@example.com"})
	require.Error(t, err)

	var pe *ProviderError
	require.ErrorAs(t, err, &pe)
	assert.True(t, pe.Retryable)
	assert.True(t, pe.DropRef)
	assert.Contains(t, pe.Error(), "contact not found")
}

func TestGenericProvider_UpsertContact_429WithRetryAfterEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "45")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`slow down`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{Name: "acme", ContactURL: srv.URL})
	require.NoError(t, err)

	_, err = p.UpsertContact(t.Context(), &Contact{Email: "x@example.com"})
	require.Error(t, err)

	var pe *ProviderError
	require.ErrorAs(t, err, &pe)
	assert.True(t, pe.Retryable)
	assert.Equal(t, 45*time.Second, pe.RetryAfter)
	assert.False(t, pe.DropRef)
}

func TestGenericProvider_UpsertContact_400IsTerminalEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid email"}`))
	}))
	defer srv.Close()

	p, err := NewGenericProvider(ProviderConfig{Name: "acme", ContactURL: srv.URL})
	require.NoError(t, err)

	_, err = p.UpsertContact(t.Context(), &Contact{Email: "not-an-email"})
	require.Error(t, err)

	ok, after := Retryable(err)
	assert.False(t, ok, "a 400 must not retry")
	assert.Zero(t, after)
}
