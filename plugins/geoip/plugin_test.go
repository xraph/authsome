package geoip

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

// countingProvider wraps a TestProvider and tracks how many Lookup calls are made.
type countingProvider struct {
	inner *TestProvider
	count int
}

func (cp *countingProvider) Lookup(ip net.IP) (*GeoLocation, error) {
	cp.count++
	return cp.inner.Lookup(ip)
}

func (cp *countingProvider) Close() error { return nil }

func TestPlugin_Name(t *testing.T) {
	p := New(Config{})
	assert.Equal(t, "geoip", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) {
	var p interface{} = New(Config{})

	_, ok := p.(plugin.Plugin)
	assert.True(t, ok, "should implement plugin.Plugin")

	_, ok = p.(plugin.OnInit)
	assert.True(t, ok, "should implement plugin.OnInit")

	_, ok = p.(plugin.OnShutdown)
	assert.True(t, ok, "should implement plugin.OnShutdown")

	_, ok = p.(plugin.BeforeSignIn)
	assert.True(t, ok, "should implement plugin.BeforeSignIn")

	_, ok = p.(plugin.AfterSignIn)
	assert.True(t, ok, "should implement plugin.AfterSignIn")

	_, ok = p.(plugin.BeforeSignUp)
	assert.True(t, ok, "should implement plugin.BeforeSignUp")
}

func TestCache_SetAndGet(t *testing.T) {
	c := NewCache(time.Hour)
	loc := &GeoLocation{IP: "1.2.3.4", Country: "US", City: "New York"}

	c.Set("1.2.3.4", loc)
	got := c.Get("1.2.3.4")

	require.NotNil(t, got)
	assert.Equal(t, "US", got.Country)
	assert.Equal(t, "New York", got.City)
}

func TestCache_Miss(t *testing.T) {
	c := NewCache(time.Hour)
	assert.Nil(t, c.Get("9.9.9.9"))
}

func TestCache_Expired(t *testing.T) {
	c := NewCache(time.Millisecond)
	c.Set("1.2.3.4", &GeoLocation{IP: "1.2.3.4", Country: "US"})

	time.Sleep(5 * time.Millisecond)

	assert.Nil(t, c.Get("1.2.3.4"))
}

func TestHaversine_NYCToLondon(t *testing.T) {
	dist := Haversine(40.7128, -74.0060, 51.5074, -0.1278)
	assert.InDelta(t, 5570, dist, 20, "NYC to London should be approximately 5570 km")
}

func TestHaversine_SamePoint(t *testing.T) {
	dist := Haversine(40.7128, -74.0060, 40.7128, -74.0060)
	assert.InDelta(t, 0, dist, 0.01)
}

func TestHaversine_Antipodal(t *testing.T) {
	dist := Haversine(90, 0, -90, 0)
	assert.InDelta(t, 20015, dist, 30, "pole to pole should be approximately 20015 km")
}

func TestResolve_NoProvider(t *testing.T) {
	p := New(Config{})
	p.cache = NewCache(time.Hour)

	assert.Nil(t, p.Resolve("1.2.3.4"))
}

func TestResolve_EmptyString(t *testing.T) {
	p := NewTestPlugin(map[string]*GeoLocation{})
	assert.Nil(t, p.Resolve(""))
}

func TestResolve_PrivateIPs(t *testing.T) {
	p := NewTestPlugin(map[string]*GeoLocation{})

	assert.Nil(t, p.Resolve("192.168.1.1"))
	assert.Nil(t, p.Resolve("127.0.0.1"))
	assert.Nil(t, p.Resolve("10.0.0.1"))
}

func TestResolve_PublicIP(t *testing.T) {
	p := NewTestPlugin(map[string]*GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", City: "Mountain View"},
	})

	loc := p.Resolve("8.8.8.8")
	require.NotNil(t, loc)
	assert.Equal(t, "US", loc.Country)
	assert.Equal(t, "Mountain View", loc.City)
}

func TestResolve_StripPort(t *testing.T) {
	p := NewTestPlugin(map[string]*GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US"},
	})

	loc := p.Resolve("8.8.8.8:8080")
	require.NotNil(t, loc)
	assert.Equal(t, "US", loc.Country)
}

func TestResolve_CachesResult(t *testing.T) {
	cp := &countingProvider{
		inner: NewTestProvider(map[string]*GeoLocation{
			"8.8.8.8": {IP: "8.8.8.8", Country: "US"},
		}),
	}
	p := New(Config{CacheTTL: time.Hour})
	p.provider = cp
	p.cache = NewCache(time.Hour)
	p.logger = log.NewNoopLogger()

	loc1 := p.Resolve("8.8.8.8")
	require.NotNil(t, loc1)
	assert.Equal(t, "US", loc1.Country)

	loc2 := p.Resolve("8.8.8.8")
	require.NotNil(t, loc2)

	assert.Equal(t, 1, cp.count, "provider should only be called once due to caching")
}

func TestResolve_UnknownIP(t *testing.T) {
	p := NewTestPlugin(map[string]*GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US"},
	})

	assert.Nil(t, p.Resolve("1.1.1.1"))
}

func TestTestProvider_Lookup(t *testing.T) {
	tp := NewTestProvider(map[string]*GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", City: "Mountain View"},
	})

	loc, err := tp.Lookup(net.ParseIP("8.8.8.8"))
	require.NoError(t, err)
	require.NotNil(t, loc)
	assert.Equal(t, "US", loc.Country)

	loc2, err := tp.Lookup(net.ParseIP("1.1.1.1"))
	require.NoError(t, err)
	assert.Nil(t, loc2)

	assert.NoError(t, tp.Close())
}

func TestOnAfterSignIn_NilLocation(t *testing.T) {
	p := NewTestPlugin(map[string]*GeoLocation{})

	u := &user.User{}
	s := &session.Session{IPAddress: "192.168.1.1"}

	err := p.OnAfterSignIn(context.Background(), u, s)
	assert.NoError(t, err)
}

func TestOnAfterSignIn_WithLocation(t *testing.T) {
	p := NewTestPlugin(map[string]*GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", City: "Mountain View"},
	})

	u := &user.User{}
	s := &session.Session{IPAddress: "8.8.8.8"}

	err := p.OnAfterSignIn(context.Background(), u, s)
	assert.NoError(t, err)
}

func TestOnShutdown_NilProvider(t *testing.T) {
	p := New(Config{})
	assert.NoError(t, p.OnShutdown(context.Background()))
}

func TestOnShutdown_WithProvider(t *testing.T) {
	p := NewTestPlugin(map[string]*GeoLocation{})
	assert.NoError(t, p.OnShutdown(context.Background()))
}
