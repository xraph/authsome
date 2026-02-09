package security

import (
	"context"
	"testing"
)

type noopRepo struct{}

func (noopRepo) Create(_ context.Context, _ *SecurityEvent) error { return nil }

func TestShouldTrustForwardedHeaders_Disabled(t *testing.T) {
	s := NewService(noopRepo{}, Config{TrustProxyHeaders: false})
	if s.ShouldTrustForwardedHeaders("203.0.113.10") {
		t.Fatalf("expected not to trust forwarded headers when disabled")
	}
}

func TestShouldTrustForwardedHeaders_AllTrusted(t *testing.T) {
	s := NewService(noopRepo{}, Config{TrustProxyHeaders: true})
	if !s.ShouldTrustForwardedHeaders("198.51.100.23") {
		t.Fatalf("expected to trust forwarded headers when enabled with no restrictions")
	}
}

func TestShouldTrustForwardedHeaders_WithTrustedProxies(t *testing.T) {
	s := NewService(noopRepo{}, Config{TrustProxyHeaders: true, TrustedProxies: []string{"10.0.0.0/8", "203.0.113.42"}})

	cases := []struct {
		ip   string
		want bool
	}{
		{"10.1.2.3", true},
		{"203.0.113.7", false},
		{"203.0.113.42", true},
	}
	for _, c := range cases {
		got := s.ShouldTrustForwardedHeaders(c.ip)
		if got != c.want {
			t.Fatalf("ip %s: expected %v, got %v", c.ip, c.want, got)
		}
	}
}
