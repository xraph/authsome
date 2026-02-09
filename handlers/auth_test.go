package handlers

import (
	"context"
	"net/http"
	"testing"

	sec "github.com/xraph/authsome/core/security"
)

type noopRepo struct{}

func (noopRepo) Create(_ context.Context, _ *sec.SecurityEvent) error { return nil }

func TestClientIPFromRequest_NoTrustUsesRemoteAddr(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.10:5678"
	s := sec.NewService(noopRepo{}, sec.Config{TrustProxyHeaders: false})

	ip := clientIPFromRequest(r, s)
	if ip != "203.0.113.10" {
		t.Fatalf("expected remote addr host, got %s", ip)
	}
}

func TestClientIPFromRequest_TrustsXFFWhenEnabled(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.1:5678"
	r.Header.Set("X-Forwarded-For", "198.51.100.23, 203.0.113.10")

	s := sec.NewService(noopRepo{}, sec.Config{TrustProxyHeaders: true})

	ip := clientIPFromRequest(r, s)
	if ip != "198.51.100.23" {
		t.Fatalf("expected first XFF ip, got %s", ip)
	}
}

func TestClientIPFromRequest_TrustedProxyCIDR(t *testing.T) {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.10:9999"
	r.Header.Set("X-Forwarded-For", "198.51.100.23")

	s := sec.NewService(noopRepo{}, sec.Config{TrustProxyHeaders: true, TrustedProxies: []string{"203.0.113.0/24"}})

	ip := clientIPFromRequest(r, s)
	if ip != "198.51.100.23" {
		t.Fatalf("expected XFF due to trusted proxy, got %s", ip)
	}
	// Now not trusted proxy
	r.RemoteAddr = "198.51.100.99:9999"

	ip = clientIPFromRequest(r, s)
	if ip != "198.51.100.99" {
		t.Fatalf("expected remote addr due to untrusted proxy, got %s", ip)
	}
}
