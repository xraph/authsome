package middleware_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

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
