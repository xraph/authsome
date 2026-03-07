package geofence

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/geoip"
)

// newTestPlugin creates a geofence Plugin with the given config and a geoip
// test plugin backed by the provided IP-to-location mapping.
func newTestPlugin(cfg Config, mapping map[string]*geoip.GeoLocation) *Plugin {
	p := New(cfg)
	p.geoIP = geoip.NewTestPlugin(mapping)
	p.logger = log.NewNoopLogger()
	return p
}

func TestPlugin_Name(t *testing.T) {
	p := New(Config{})
	assert.Equal(t, "geofence", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) {
	var p interface{} = New(Config{})

	_, ok := p.(plugin.Plugin)
	assert.True(t, ok, "should implement plugin.Plugin")

	_, ok = p.(plugin.OnInit)
	assert.True(t, ok, "should implement plugin.OnInit")

	_, ok = p.(plugin.BeforeSignIn)
	assert.True(t, ok, "should implement plugin.BeforeSignIn")

	_, ok = p.(plugin.BeforeSignUp)
	assert.True(t, ok, "should implement plugin.BeforeSignUp")
}

func TestAllowedCountries_Permits(t *testing.T) {
	p := newTestPlugin(Config{
		AllowedCountries: []string{"US", "CA"},
	}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", City: "Mountain View"},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	assert.NoError(t, err)
}

func TestAllowedCountries_Blocks(t *testing.T) {
	p := newTestPlugin(Config{
		AllowedCountries: []string{"US", "CA"},
	}, map[string]*geoip.GeoLocation{
		"1.2.3.4": {IP: "1.2.3.4", Country: "RU", City: "Moscow"},
	})

	err := p.check(context.Background(), "1.2.3.4", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "geofence:")
}

func TestBlockedCountries_Blocks(t *testing.T) {
	p := newTestPlugin(Config{
		BlockedCountries: []string{"CN", "RU"},
	}, map[string]*geoip.GeoLocation{
		"1.2.3.4": {IP: "1.2.3.4", Country: "CN", City: "Beijing"},
	})

	err := p.check(context.Background(), "1.2.3.4", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "geofence:")
}

func TestBlockedCountries_Permits(t *testing.T) {
	p := newTestPlugin(Config{
		BlockedCountries: []string{"CN", "RU"},
	}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", City: "Mountain View"},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	assert.NoError(t, err)
}

func TestDenyAllDefault_BlocksUnresolved(t *testing.T) {
	p := newTestPlugin(Config{
		DefaultPolicy: "deny_all",
	}, map[string]*geoip.GeoLocation{})

	// The IP resolves to nil (not in mapping + private IPs return nil).
	// With deny_all, unresolvable IPs should be blocked.
	err := p.check(context.Background(), "99.99.99.99", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "geofence:")
}

func TestDenyAllDefault_BlocksKnownIP(t *testing.T) {
	p := newTestPlugin(Config{
		DefaultPolicy: "deny_all",
	}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US"},
	})

	// Even with a resolvable IP, deny_all with no allow/block lists blocks everything.
	err := p.check(context.Background(), "8.8.8.8", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "geofence:")
}

func TestAllowAllDefault_PermitsUnresolved(t *testing.T) {
	p := newTestPlugin(Config{
		DefaultPolicy: "allow_all",
	}, map[string]*geoip.GeoLocation{})

	err := p.check(context.Background(), "99.99.99.99", "app-1")
	assert.NoError(t, err)
}

func TestNoGeoIP_NoOp(t *testing.T) {
	p := New(Config{
		DefaultPolicy:    "deny_all",
		BlockedCountries: []string{"CN"},
	})
	p.logger = log.NewNoopLogger()
	// geoIP is nil -- check should be a no-op and permit the request.

	err := p.check(context.Background(), "1.2.3.4", "app-1")
	assert.NoError(t, err)
}

func TestCheck_EmptyIP(t *testing.T) {
	p := newTestPlugin(Config{
		DefaultPolicy: "deny_all",
	}, map[string]*geoip.GeoLocation{})

	err := p.check(context.Background(), "", "app-1")
	assert.NoError(t, err)
}

func TestCustomBlockMessage(t *testing.T) {
	p := newTestPlugin(Config{
		BlockedCountries: []string{"RU"},
		BlockMessage:     "your region is restricted",
	}, map[string]*geoip.GeoLocation{
		"1.2.3.4": {IP: "1.2.3.4", Country: "RU"},
	})

	err := p.check(context.Background(), "1.2.3.4", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "your region is restricted")
}

func TestOnBeforeSignIn_Blocks(t *testing.T) {
	p := newTestPlugin(Config{
		BlockedCountries: []string{"CN"},
	}, map[string]*geoip.GeoLocation{
		"1.2.3.4": {IP: "1.2.3.4", Country: "CN"},
	})

	req := &account.SignInRequest{
		AppID:     id.NewAppID(),
		IPAddress: "1.2.3.4",
	}

	err := p.OnBeforeSignIn(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "geofence:")
}

func TestOnBeforeSignUp_Blocks(t *testing.T) {
	p := newTestPlugin(Config{
		BlockedCountries: []string{"CN"},
	}, map[string]*geoip.GeoLocation{
		"1.2.3.4": {IP: "1.2.3.4", Country: "CN"},
	})

	req := &account.SignUpRequest{
		AppID:     id.NewAppID(),
		IPAddress: "1.2.3.4",
	}

	err := p.OnBeforeSignUp(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "geofence:")
}

func TestBlockedAndAllowed_BlockTakesPrecedence(t *testing.T) {
	// If a country appears in both blocked and allowed, blocked takes precedence
	// because check evaluates blocklist first.
	p := newTestPlugin(Config{
		AllowedCountries: []string{"CN", "US"},
		BlockedCountries: []string{"CN"},
	}, map[string]*geoip.GeoLocation{
		"1.2.3.4": {IP: "1.2.3.4", Country: "CN"},
	})

	err := p.check(context.Background(), "1.2.3.4", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "geofence:")
}
