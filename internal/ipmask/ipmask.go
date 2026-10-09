// Package ipmask reduces a client address to its network so a record can
// keep where a request came from without keeping who.
package ipmask

import (
	"net/netip"
	"strings"
)

// Network returns the /24 (IPv4) or /64 (IPv6) network an address belongs
// to, in CIDR form, or "" when the input is not an address. A record that
// held the address keeps enough to show the request's origin for a dispute
// and no longer identifies the device.
func Network(ip string) string {
	ip = strings.TrimSpace(ip)
	// A value that is already a network is left as it is, so masking is
	// idempotent and a second erasure pass changes nothing.
	if prefix, err := netip.ParsePrefix(ip); err == nil {
		return prefix.Masked().String()
	}
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return ""
	}
	bits := 64
	if addr.Is4() || addr.Is4In6() {
		addr = addr.Unmap()
		bits = 24
	}
	return netip.PrefixFrom(addr, bits).Masked().String()
}
