package vpndetect

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

// newTestPlugin creates a vpndetect Plugin with the given config and a geoip
// test plugin backed by the provided IP-to-location mapping.
func newTestPlugin(cfg Config, mapping map[string]*geoip.GeoLocation) *Plugin {
	p := New(cfg)
	p.geoIP = geoip.NewTestPlugin(mapping)
	p.logger = log.NewNoopLogger()
	return p
}

func TestPlugin_Name(t *testing.T) {
	p := New(Config{})
	assert.Equal(t, "vpndetect", p.Name())
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

func TestBlockVPN_Blocks(t *testing.T) {
	p := newTestPlugin(Config{BlockVPN: true}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", IsVPN: true},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vpndetect:")
	assert.Contains(t, err.Error(), "vpn")
}

func TestBlockVPN_AllowsWhenDisabled(t *testing.T) {
	p := newTestPlugin(Config{BlockVPN: false}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", IsVPN: true},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	assert.NoError(t, err)
}

func TestBlockProxy_Blocks(t *testing.T) {
	p := newTestPlugin(Config{BlockProxy: true}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", IsProxy: true},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vpndetect:")
	assert.Contains(t, err.Error(), "proxy")
}

func TestBlockProxy_AllowsWhenDisabled(t *testing.T) {
	p := newTestPlugin(Config{BlockProxy: false}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", IsProxy: true},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	assert.NoError(t, err)
}

func TestBlockTor_Blocks(t *testing.T) {
	p := newTestPlugin(Config{BlockTor: true}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "DE", IsTor: true},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vpndetect:")
	assert.Contains(t, err.Error(), "tor")
}

func TestBlockTor_AllowsWhenDisabled(t *testing.T) {
	p := newTestPlugin(Config{BlockTor: false}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "DE", IsTor: true},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	assert.NoError(t, err)
}

func TestCleanIP_Passes(t *testing.T) {
	p := newTestPlugin(Config{
		BlockVPN:   true,
		BlockProxy: true,
		BlockTor:   true,
	}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US"},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	assert.NoError(t, err)
}

func TestNoGeoIP_NoOp(t *testing.T) {
	p := New(Config{BlockVPN: true, BlockProxy: true, BlockTor: true})
	p.logger = log.NewNoopLogger()
	// geoIP is nil -- check should be a no-op and permit the request.

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	assert.NoError(t, err)
}

func TestCheck_EmptyIP(t *testing.T) {
	p := newTestPlugin(Config{BlockVPN: true}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", IsVPN: true},
	})

	err := p.check(context.Background(), "", "app-1")
	assert.NoError(t, err)
}

func TestCheck_UnresolvableIP(t *testing.T) {
	p := newTestPlugin(Config{BlockVPN: true}, map[string]*geoip.GeoLocation{})

	// IP not in mapping, Resolve returns nil -- should not block.
	err := p.check(context.Background(), "99.99.99.99", "app-1")
	assert.NoError(t, err)
}

func TestCustomBlockMessage(t *testing.T) {
	p := newTestPlugin(Config{
		BlockVPN:     true,
		BlockMessage: "VPN connections are forbidden",
	}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", IsVPN: true},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VPN connections are forbidden")
}

func TestOnBeforeSignIn_Blocks(t *testing.T) {
	p := newTestPlugin(Config{BlockVPN: true}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", IsVPN: true},
	})

	req := &account.SignInRequest{
		AppID:     id.NewAppID(),
		IPAddress: "8.8.8.8",
	}

	err := p.OnBeforeSignIn(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vpndetect:")
}

func TestOnBeforeSignUp_Blocks(t *testing.T) {
	p := newTestPlugin(Config{BlockTor: true}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "DE", IsTor: true},
	})

	req := &account.SignUpRequest{
		AppID:     id.NewAppID(),
		IPAddress: "8.8.8.8",
	}

	err := p.OnBeforeSignUp(context.Background(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vpndetect:")
}

func TestMultipleFlags_FirstMatchWins(t *testing.T) {
	// An IP that is both a VPN and a proxy. With both BlockVPN and BlockProxy
	// enabled, the switch/case evaluates VPN first.
	p := newTestPlugin(Config{BlockVPN: true, BlockProxy: true}, map[string]*geoip.GeoLocation{
		"8.8.8.8": {IP: "8.8.8.8", Country: "US", IsVPN: true, IsProxy: true},
	})

	err := p.check(context.Background(), "8.8.8.8", "app-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vpn")
}
