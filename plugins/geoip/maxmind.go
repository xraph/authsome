package geoip

import (
	"fmt"
	"net"

	"github.com/oschwald/maxminddb-golang"
)

// maxmindRecord maps to the GeoLite2-City MMDB structure.
type maxmindRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		TimeZone  string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
	Subdivisions []struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
	Traits struct {
		IsAnonymousProxy    bool `maxminddb:"is_anonymous_proxy"`
		IsSatelliteProvider bool `maxminddb:"is_satellite_provider"`
	} `maxminddb:"traits"`
}

// MaxMindProvider resolves IPs using a MaxMind GeoLite2 MMDB file.
type MaxMindProvider struct {
	db *maxminddb.Reader
}

// NewMaxMindProvider opens a MaxMind MMDB file.
func NewMaxMindProvider(dbPath string) (*MaxMindProvider, error) {
	db, err := maxminddb.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("geoip: open maxmind db: %w", err)
	}
	return &MaxMindProvider{db: db}, nil
}

// Lookup resolves an IP address to a GeoLocation.
func (p *MaxMindProvider) Lookup(ip net.IP) (*GeoLocation, error) {
	var rec maxmindRecord
	if err := p.db.Lookup(ip, &rec); err != nil {
		return nil, fmt.Errorf("geoip: maxmind lookup: %w", err)
	}

	loc := &GeoLocation{
		IP:          ip.String(),
		Country:     rec.Country.ISOCode,
		CountryName: rec.Country.Names["en"],
		Latitude:    rec.Location.Latitude,
		Longitude:   rec.Location.Longitude,
		Timezone:    rec.Location.TimeZone,
		IsProxy:     rec.Traits.IsAnonymousProxy,
	}

	if rec.City.Names != nil {
		loc.City = rec.City.Names["en"]
	}
	if len(rec.Subdivisions) > 0 {
		loc.Region = rec.Subdivisions[0].Names["en"]
	}

	return loc, nil
}

// Close closes the MMDB reader.
func (p *MaxMindProvider) Close() error {
	return p.db.Close()
}
