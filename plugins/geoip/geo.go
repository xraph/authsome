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
//
// The returned value is a copy. Handing out the stored pointer would make
// this type thread-safe only for callers that never write: one caller
// mutating what it got back would corrupt the entry every other caller sees
// for the rest of the ttl, and that write races every concurrent Get, since
// it happens outside the lock. Plugin.Resolve returns this value straight
// through to anomaly, geofence, impossibletravel and vpndetect, so the
// pointer would be shared across plugins as well as goroutines.
func (c *Cache) Get(ip string) *GeoLocation {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[ip]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil
	}
	return cloneLocation(entry.loc)
}

// Set stores a GeoLocation in the cache. The value is copied on the way in
// for the mirror-image reason Get copies on the way out: otherwise a caller
// that mutates loc after Set returns rewrites the cached entry behind the
// lock's back.
func (c *Cache) Set(ip string, loc *GeoLocation) {
	if loc == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[ip] = cachedEntry{
		loc:       cloneLocation(loc),
		expiresAt: time.Now().Add(c.ttl),
	}
}

// cloneLocation copies loc. Every GeoLocation field is a value type today, so
// a shallow struct copy is a deep copy. Add a slice or map field and this
// must copy it explicitly, the way agentauth's cloneGrant copies Scopes.
func cloneLocation(loc *GeoLocation) *GeoLocation {
	if loc == nil {
		return nil
	}
	cp := *loc
	return &cp
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
