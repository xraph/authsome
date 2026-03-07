// Package geoip provides GeoIP location resolution for authentication events.
// It resolves IP addresses to geographic locations and stores the result in
// context for downstream plugins (geofence, impossible travel, etc.).
package geoip

import (
	"math"
	"net"
	"sync"
	"time"
)

// GeoLocation represents the geographic location resolved from an IP address.
type GeoLocation struct {
	IP          string  `json:"ip"`
	Country     string  `json:"country"`      // ISO 3166-1 alpha-2
	CountryName string  `json:"country_name"` // human-readable
	Region      string  `json:"region,omitempty"`
	City        string  `json:"city,omitempty"`
	Timezone    string  `json:"timezone,omitempty"`
	ISP         string  `json:"isp,omitempty"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	ASN         int     `json:"asn,omitempty"`
	IsProxy     bool    `json:"is_proxy,omitempty"`
	IsVPN       bool    `json:"is_vpn,omitempty"`
	IsTor       bool    `json:"is_tor,omitempty"`
}

// Provider resolves an IP address to a GeoLocation.
type Provider interface {
	Lookup(ip net.IP) (*GeoLocation, error)
	Close() error
}

// cachedEntry wraps a GeoLocation with an expiration time.
type cachedEntry struct {
	loc       *GeoLocation
	expiresAt time.Time
}

// Cache is a thread-safe TTL cache for GeoLocation lookups.
type Cache struct {
	mu      sync.RWMutex
	entries map[string]cachedEntry
	ttl     time.Duration
}

// NewCache creates a new GeoIP cache with the given TTL.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		entries: make(map[string]cachedEntry),
		ttl:     ttl,
	}
}

// Get returns a cached GeoLocation or nil if not found/expired.
func (c *Cache) Get(ip string) *GeoLocation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[ip]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return entry.loc
}

// Set stores a GeoLocation in the cache.
func (c *Cache) Set(ip string, loc *GeoLocation) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[ip] = cachedEntry{
		loc:       loc,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Haversine calculates the distance in kilometers between two geographic
// coordinates using the Haversine formula.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := degreesToRadians(lat2 - lat1)
	dLon := degreesToRadians(lon2 - lon1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func degreesToRadians(d float64) float64 {
	return d * math.Pi / 180
}
