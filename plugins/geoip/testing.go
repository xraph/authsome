package geoip

import (
	"net"
	"time"

	log "github.com/xraph/go-utils/log"
)

// TestProvider is a mock GeoIP provider for use in tests.
// It maps IP string representations to pre-configured GeoLocation values.
type TestProvider struct {
	Mapping map[string]*GeoLocation
}

// Lookup returns the pre-configured GeoLocation for the given IP.
func (tp *TestProvider) Lookup(ip net.IP) (*GeoLocation, error) {
	loc, ok := tp.Mapping[ip.String()]
	if !ok {
		return nil, nil
	}
	return loc, nil
}

// Close is a no-op for the test provider.
func (tp *TestProvider) Close() error { return nil }

// NewTestProvider creates a TestProvider with the given IP-to-GeoLocation mapping.
func NewTestProvider(m map[string]*GeoLocation) *TestProvider {
	return &TestProvider{Mapping: m}
}

// NewTestPlugin creates a fully configured Plugin with a test provider and
// cache. This is intended for use in tests across geo-security plugin packages
// that depend on a working geoip.Plugin without a real MaxMind database.
func NewTestPlugin(mapping map[string]*GeoLocation) *Plugin {
	p := New(Config{CacheTTL: time.Hour})
	p.provider = NewTestProvider(mapping)
	p.cache = NewCache(time.Hour)
	p.logger = log.NewNoopLogger()
	return p
}
