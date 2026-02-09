package security

import "context"

// GeoIPProvider resolves an IP address to a country code (ISO 3166-1 alpha-2 suggested).
type GeoIPProvider interface {
	CountryForIP(ctx context.Context, ip string) (string, error)
}

// equalCountry compares country codes case-insensitively.
func equalCountry(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	// simple case-insensitive compare without allocations
	for i := range len(a) {
		ca := a[i]
		cb := b[i]

		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}

		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}

		if ca != cb {
			return false
		}
	}

	return true
}
