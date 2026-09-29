// Package safeurl decides whether an operator-supplied URL may be called
// from inside the service. A webhook URL is a request the service makes on
// someone else's say-so, so it must not reach anything the network keeps
// private: loopback, private ranges, link-local addresses (where cloud
// metadata lives), or the machine itself. The check runs on the resolved
// addresses and again at dial time, so a name cannot change its answer
// between the two.
package safeurl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

var (
	// ErrScheme is returned for anything but https (or http under
	// AllowInsecure).
	ErrScheme = errors.New("safeurl: url must use https")
	// ErrHost is returned when the URL names no host.
	ErrHost = errors.New("safeurl: url has no host")
	// ErrUserinfo is returned when the URL carries credentials.
	ErrUserinfo = errors.New("safeurl: url must not carry credentials")
	// ErrNonPublic is returned when the host resolves to an address the
	// service must not call.
	ErrNonPublic = errors.New("safeurl: url points at a non-public address")
	// ErrResolve is returned when the host cannot be resolved at all.
	ErrResolve = errors.New("safeurl: host does not resolve")
)

// Resolver looks a host name up. net.DefaultResolver satisfies it.
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// Options tune a check.
type Options struct {
	// AllowInsecure permits http and non-public addresses. It is for
	// development against a receiver on the same machine and must never be
	// set in production.
	AllowInsecure bool
	// Resolver overrides name resolution; nil uses net.DefaultResolver.
	Resolver Resolver
}

func (o Options) resolver() Resolver {
	if o.Resolver != nil {
		return o.Resolver
	}
	return net.DefaultResolver
}

// Validate parses raw and checks it may be called: https, a host, no
// credentials, and every address the host resolves to is public. It
// returns the parsed URL on success.
func Validate(ctx context.Context, raw string, opts Options) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("safeurl: parse url: %w", err)
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !opts.AllowInsecure {
			return nil, ErrScheme
		}
	default:
		return nil, ErrScheme
	}
	if u.Hostname() == "" {
		return nil, ErrHost
	}
	if u.User != nil {
		return nil, ErrUserinfo
	}
	if opts.AllowInsecure {
		return u, nil
	}
	if _, err := resolvePublic(ctx, u.Hostname(), opts); err != nil {
		return nil, err
	}
	return u, nil
}

// resolvePublic resolves host and returns its addresses, or ErrNonPublic
// when any of them is one the service must not call. All of them must pass:
// a name that answers with one public and one private address is a name
// that will, sooner or later, be dialled at the private one.
func resolvePublic(ctx context.Context, host string, opts Options) ([]netip.Addr, error) {
	var addrs []netip.Addr
	if ip, err := netip.ParseAddr(host); err == nil {
		addrs = []netip.Addr{ip}
	} else {
		found, err := opts.resolver().LookupIPAddr(ctx, host)
		if err != nil || len(found) == 0 {
			return nil, ErrResolve
		}
		for _, f := range found {
			if a, ok := netip.AddrFromSlice(f.IP); ok {
				addrs = append(addrs, a)
			}
		}
	}
	for _, a := range addrs {
		if !IsPublic(a) {
			return nil, ErrNonPublic
		}
	}
	return addrs, nil
}

// reserved lists the ranges netip's own predicates do not cover: the
// "this network" block, carrier-grade NAT, the IETF protocol block,
// benchmarking, the reserved class E block, NAT64 and IPv6 documentation.
var reserved = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("2001:db8::/32"),
}

// IsPublic reports whether a is an address the service may call: not
// loopback, private, link-local (the cloud metadata address included),
// multicast, unspecified or reserved. IPv4-mapped IPv6 addresses are judged
// as the IPv4 address they carry.
func IsPublic(a netip.Addr) bool {
	a = a.Unmap()
	if !a.IsValid() || a.IsLoopback() || a.IsPrivate() || a.IsLinkLocalUnicast() ||
		a.IsLinkLocalMulticast() || a.IsInterfaceLocalMulticast() || a.IsMulticast() || a.IsUnspecified() {
		return false
	}
	for _, p := range reserved {
		if p.Contains(a) {
			return false
		}
	}
	return true
}

// Client returns an HTTP client for calling a validated URL. It resolves
// the host itself and dials only addresses that pass IsPublic, so a name
// whose answer changed since Validate is refused rather than followed; it
// never follows a redirect (the 3xx response is returned as is, for the
// caller to refuse); and it ignores proxy settings, which would otherwise
// carry the request somewhere the check never saw.
func Client(timeout time.Duration, opts Options) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			if opts.AllowInsecure {
				return dialer.DialContext(ctx, network, address)
			}
			addrs, err := resolvePublic(ctx, host, opts)
			if err != nil {
				return nil, err
			}
			var lastErr error
			for _, a := range addrs {
				conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(a.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			return nil, lastErr
		},
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		MaxIdleConns:          1,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
