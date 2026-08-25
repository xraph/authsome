package middleware_test

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/middleware"
)

// TestClientIP_TrustedProxyAware verifies that X-Forwarded-For / X-Real-IP are
// honored ONLY when the immediate peer is a trusted (loopback/private) proxy.
// A direct client cannot spoof its IP by setting these headers, which is the
// property rate limiting, session IP-binding, and IP allow/block lists depend
// on.
func TestClientIP_TrustedProxyAware(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		xff        string
		xRealIP    string
		want       string
	}{
		{
			name:       "direct client cannot spoof via XFF",
			remoteAddr: "203.0.113.9:44321",
			xff:        "1.2.3.4",
			want:       "203.0.113.9",
		},
		{
			name:       "direct client cannot spoof via X-Real-IP",
			remoteAddr: "203.0.113.9:44321",
			xRealIP:    "1.2.3.4",
			want:       "203.0.113.9",
		},
		{
			name:       "no forwarding headers, direct peer",
			remoteAddr: "198.51.100.7:5555",
			want:       "198.51.100.7",
		},
		{
			name:       "trusted loopback proxy honors XFF",
			remoteAddr: "127.0.0.1:8080",
			xff:        "203.0.113.20",
			want:       "203.0.113.20",
		},
		{
			name:       "trusted private proxy honors XFF",
			remoteAddr: "10.0.0.5:8080",
			xff:        "203.0.113.21",
			want:       "203.0.113.21",
		},
		{
			name:       "trusted proxy, client-spoofed left entry is ignored (rightmost-untrusted)",
			remoteAddr: "10.0.0.5:8080",
			xff:        "1.2.3.4, 203.0.113.30",
			want:       "203.0.113.30",
		},
		{
			name:       "trusted proxy falls back to X-Real-IP when XFF absent",
			remoteAddr: "127.0.0.1:8080",
			xRealIP:    "203.0.113.40",
			want:       "203.0.113.40",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}
			assert.Equal(t, tc.want, middleware.ClientIP(req))
		})
	}
}

// TestRequestURL_TrustedProxyAware pins the trust boundary RequestURL applies
// to X-Forwarded-Proto / X-Forwarded-Host. This matters beyond ClientIP: the
// URL RequestURL produces is compared directly against a DPoP proof's htu
// claim, so a wrong trust decision here either lets an untrusted client steer
// what a proof is checked against, or breaks legitimate proofs behind a real
// load balancer.
func TestRequestURL_TrustedProxyAware(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		host       string
		xfProto    string
		xfHost     string
		want       string
	}{
		{
			name:       "untrusted peer: forwarded headers are ignored",
			remoteAddr: "203.0.113.9:44321", // public, not a trusted proxy
			host:       "example.com",
			xfProto:    "https",
			xfHost:     "spoofed.example.com",
			want:       "http://example.com/test",
		},
		{
			name:       "trusted peer: forwarded headers are honored",
			remoteAddr: "127.0.0.1:8080", // loopback, trusted
			host:       "internal.local",
			xfProto:    "https",
			xfHost:     "public.example.com",
			want:       "https://public.example.com/test",
		},
		{
			name:       "trusted peer: comma-separated chain, first entry wins",
			remoteAddr: "10.0.0.5:8080", // private range, trusted
			host:       "internal.local",
			xfProto:    "https, http",
			xfHost:     "first.example.com, second.example.com",
			want:       "https://first.example.com/test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
			req.RemoteAddr = tc.remoteAddr
			req.Host = tc.host
			if tc.xfProto != "" {
				req.Header.Set("X-Forwarded-Proto", tc.xfProto)
			}
			if tc.xfHost != "" {
				req.Header.Set("X-Forwarded-Host", tc.xfHost)
			}
			assert.Equal(t, tc.want, middleware.RequestURL(req))
		})
	}
}

// TestRequestURL_EmptyHostFailsClosed: an empty r.Host produces a URL whose
// host component is also empty. dpop.Validator's htu comparison rejects any
// URL missing a host ("url %q is not absolute"), so this is not a passthrough
// bug — it is the request failing closed rather than being compared against
// something that happens to match.
func TestRequestURL_EmptyHostFailsClosed(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/test", nil)
	req.RemoteAddr = "203.0.113.9:44321"
	req.Host = ""

	got := middleware.RequestURL(req)

	parsed, err := url.Parse(got)
	require.NoError(t, err)
	assert.Empty(t, parsed.Host, "an empty request Host must produce a URL with no host, which normalizeHTU rejects rather than silently matching")
}
