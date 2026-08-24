package middleware

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

// trustedProxyNets holds the CIDR ranges treated as trusted reverse proxies.
// X-Forwarded-For / X-Real-IP are honored ONLY when the immediate peer
// (RemoteAddr) falls in one of these ranges — so a directly-connecting client
// cannot spoof its IP by sending those headers.
//
// The default set is loopback plus the private / link-level ranges, which is
// where a same-network load balancer or sidecar proxy lives. Operators whose
// proxy connects from a public address can override the set via the
// AUTHSOME_TRUSTED_PROXIES environment variable (comma-separated CIDRs).
var (
	trustedProxyMu   sync.RWMutex
	trustedProxyNets = defaultTrustedProxyNets()
)

func defaultTrustedProxyNets() []*net.IPNet {
	return parseCIDRs([]string{
		"127.0.0.0/8", "::1/128",
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
		"169.254.0.0/16", "fe80::/10", "fc00::/7",
	})
}

func init() {
	if env := os.Getenv("AUTHSOME_TRUSTED_PROXIES"); env != "" {
		parts := strings.Split(env, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if nets := parseCIDRs(parts); len(nets) > 0 {
			SetTrustedProxies(nets)
		}
	}
}

// SetTrustedProxies replaces the trusted-proxy CIDR set. Passing nil restores
// the built-in defaults. Primarily used for configuration and tests.
func SetTrustedProxies(nets []*net.IPNet) {
	trustedProxyMu.Lock()
	defer trustedProxyMu.Unlock()
	if nets == nil {
		trustedProxyNets = defaultTrustedProxyNets()
		return
	}
	trustedProxyNets = nets
}

func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	trustedProxyMu.RLock()
	defer trustedProxyMu.RUnlock()
	for _, n := range trustedProxyNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// peerIP extracts the immediate peer IP from a "host:port" RemoteAddr.
func peerIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	return net.ParseIP(strings.TrimSpace(host))
}

// ClientIP resolves the real client IP for a request. X-Forwarded-For and
// X-Real-IP are trusted ONLY when the immediate peer is a trusted proxy;
// otherwise the peer address is returned, so a direct client cannot forge its
// IP. When the peer is trusted, the X-Forwarded-For chain is walked
// right-to-left and the first non-proxy address (the real client) is returned.
func ClientIP(r *http.Request) string {
	peer := peerIP(r.RemoteAddr)
	if peer == nil {
		// Unparseable RemoteAddr: return it verbatim (minus any port) rather
		// than trusting spoofable headers.
		addr := r.RemoteAddr
		if i := strings.LastIndex(addr, ":"); i > 0 {
			addr = addr[:i]
		}
		return addr
	}

	// Direct or untrusted peer: never trust forwarding headers.
	if !isTrustedProxy(peer) {
		return peer.String()
	}

	// Trusted proxy: the real client is the rightmost X-Forwarded-For entry
	// that isn't itself a trusted proxy. Anything a malicious client prepends
	// sits to the left of the address our proxy appended, so it is ignored.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(parts[i]))
			if ip == nil {
				continue
			}
			if !isTrustedProxy(ip) {
				return ip.String()
			}
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return ip.String()
		}
	}
	return peer.String()
}

// RequestURL reconstructs the absolute URL the client called, for comparison
// against a DPoP proof's htu claim.
//
// Behind a load balancer r.Host and r.TLS describe the hop, not the client's
// URL, so X-Forwarded-Proto and X-Forwarded-Host are honoured only when the
// immediate peer is a trusted proxy. That is the same rule ClientIP applies
// and it reuses the same configuration, so there is one notion of trusted
// proxy in this codebase rather than two.
//
// Query and fragment are excluded: RFC 9449 compares htu without them.
func RequestURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host

	if peer := peerIP(r.RemoteAddr); peer != nil && isTrustedProxy(peer) {
		if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
			// Take the first entry: a chain of proxies appends.
			if i := strings.IndexByte(p, ','); i >= 0 {
				p = p[:i]
			}
			scheme = strings.ToLower(strings.TrimSpace(p))
		}
		if h := r.Header.Get("X-Forwarded-Host"); h != "" {
			if i := strings.IndexByte(h, ','); i >= 0 {
				h = h[:i]
			}
			host = strings.TrimSpace(h)
		}
	}

	return (&url.URL{Scheme: scheme, Host: host, Path: r.URL.Path}).String()
}
