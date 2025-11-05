package apikey

import (
	"net"
	"strings"
)

// extractClientIP extracts the real client IP from the request
// Handles X-Forwarded-For, X-Real-IP headers
func extractClientIP(remoteAddr string) string {
	// Remove port if present
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		remoteAddr = remoteAddr[:idx]
	}

	// Remove IPv6 brackets
	remoteAddr = strings.Trim(remoteAddr, "[]")

	return remoteAddr
}

// isIPAllowed checks if an IP address is allowed based on whitelist
// Supports exact IP matching and CIDR notation
func isIPAllowed(clientIP string, allowedIPs []string) bool {
	if len(allowedIPs) == 0 {
		return true // No whitelist = allow all
	}

	// Parse client IP
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false // Invalid IP
	}

	for _, allowed := range allowedIPs {
		// Try exact match first
		if allowed == clientIP {
			return true
		}

		// Try CIDR notation
		if strings.Contains(allowed, "/") {
			_, ipNet, err := net.ParseCIDR(allowed)
			if err != nil {
				continue // Invalid CIDR, skip
			}
			if ipNet.Contains(ip) {
				return true
			}
		} else {
			// Plain IP without CIDR
			allowedIP := net.ParseIP(allowed)
			if allowedIP != nil && allowedIP.Equal(ip) {
				return true
			}
		}
	}

	return false
}

// validateIPWhitelist validates IP whitelist entries
// Returns error if any entry is invalid
func validateIPWhitelist(ips []string) error {
	for _, ipStr := range ips {
		if strings.Contains(ipStr, "/") {
			// CIDR notation
			_, _, err := net.ParseCIDR(ipStr)
			if err != nil {
				return err
			}
		} else {
			// Plain IP
			if net.ParseIP(ipStr) == nil {
				return net.InvalidAddrError(ipStr)
			}
		}
	}
	return nil
}
