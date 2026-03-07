package impossibletravel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/plugins/geoip"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/user"
)

func newTestPlugin(cfg Config, mapping map[string]*geoip.GeoLocation) *Plugin {
	p := New(cfg)
	p.geoIP = geoip.NewTestPlugin(mapping)
	p.logger = log.NewNoopLogger()
	return p
}

var defaultMapping = map[string]*geoip.GeoLocation{
	"1.1.1.1": {IP: "1.1.1.1", Country: "US", City: "New York", Latitude: 40.7128, Longitude: -74.0060},
	"2.2.2.2": {IP: "2.2.2.2", Country: "GB", City: "London", Latitude: 51.5074, Longitude: -0.1278},
}

func TestPlugin_Name(t *testing.T) {
	p := New(Config{})
	assert.Equal(t, "impossibletravel", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) {
	var p interface{} = New(Config{})
	_, ok := p.(plugin.Plugin)
	assert.True(t, ok)
	_, ok = p.(plugin.OnInit)
	assert.True(t, ok)
	_, ok = p.(plugin.AfterSignIn)
	assert.True(t, ok)
}

func TestFirstLogin_NoAlert(t *testing.T) {
	p := newTestPlugin(Config{}, defaultMapping)
	u := &user.User{ID: id.NewUserID()}
	s := &session.Session{IPAddress: "1.1.1.1"}
	err := p.OnAfterSignIn(context.Background(), u, s)
	assert.NoError(t, err)
}

func TestSameLocation_NoAlert(t *testing.T) {
	p := newTestPlugin(Config{}, defaultMapping)
	u := &user.User{ID: id.NewUserID()}
	s1 := &session.Session{IPAddress: "1.1.1.1"}
	s2 := &session.Session{IPAddress: "1.1.1.1"}
	require.NoError(t, p.OnAfterSignIn(context.Background(), u, s1))
	err := p.OnAfterSignIn(context.Background(), u, s2)
	assert.NoError(t, err)
	// No alert because same location = 0 distance < MinDistanceKm
}

func TestImpossibleSpeed_Alert(t *testing.T) {
	p := newTestPlugin(Config{MaxSpeedKmH: 900, MinDistanceKm: 100}, defaultMapping)
	userID := id.NewUserID()
	u := &user.User{ID: userID}

	// First login from NYC
	s1 := &session.Session{IPAddress: "1.1.1.1"}
	require.NoError(t, p.OnAfterSignIn(context.Background(), u, s1))

	// Manually set the last login time to 1 minute ago
	p.mu.Lock()
	p.lastLogins[userID.String()].LoginAt = time.Now().Add(-1 * time.Minute)
	p.mu.Unlock()

	// Second login from London — 5570 km in 1 minute = impossible
	s2 := &session.Session{IPAddress: "2.2.2.2"}
	// This should not error (action is "flag" by default, not "block")
	err := p.OnAfterSignIn(context.Background(), u, s2)
	assert.NoError(t, err)
}

func TestRealisticSpeed_NoAlert(t *testing.T) {
	p := newTestPlugin(Config{MaxSpeedKmH: 900, MinDistanceKm: 100}, defaultMapping)
	userID := id.NewUserID()
	u := &user.User{ID: userID}

	s1 := &session.Session{IPAddress: "1.1.1.1"}
	require.NoError(t, p.OnAfterSignIn(context.Background(), u, s1))

	// Set last login 8 hours ago
	p.mu.Lock()
	p.lastLogins[userID.String()].LoginAt = time.Now().Add(-8 * time.Hour)
	p.mu.Unlock()

	// NYC to London (5570km) in 8h = ~696 km/h < 900 km/h threshold
	s2 := &session.Session{IPAddress: "2.2.2.2"}
	err := p.OnAfterSignIn(context.Background(), u, s2)
	assert.NoError(t, err)
}

func TestBelowMinDistance_NoAlert(t *testing.T) {
	// MinDistanceKm very high, so even NYC->London is ignored
	p := newTestPlugin(Config{MinDistanceKm: 10000}, defaultMapping)
	userID := id.NewUserID()
	u := &user.User{ID: userID}

	s1 := &session.Session{IPAddress: "1.1.1.1"}
	require.NoError(t, p.OnAfterSignIn(context.Background(), u, s1))

	p.mu.Lock()
	p.lastLogins[userID.String()].LoginAt = time.Now().Add(-1 * time.Minute)
	p.mu.Unlock()

	s2 := &session.Session{IPAddress: "2.2.2.2"}
	err := p.OnAfterSignIn(context.Background(), u, s2)
	assert.NoError(t, err)
}

func TestRiskLevel_Critical(t *testing.T) {
	// Speed > 5x threshold = critical
	assert.Equal(t, "critical", riskLevel(5000, 900))
}

func TestRiskLevel_High(t *testing.T) {
	// Speed 2-5x threshold = high
	assert.Equal(t, "high", riskLevel(2000, 900))
}

func TestRiskLevel_Medium(t *testing.T) {
	assert.Equal(t, "medium", riskLevel(1000, 900))
}

func TestLookbackExpired(t *testing.T) {
	p := newTestPlugin(Config{LookbackWindow: 1 * time.Hour, MinDistanceKm: 100}, defaultMapping)
	userID := id.NewUserID()
	u := &user.User{ID: userID}

	s1 := &session.Session{IPAddress: "1.1.1.1"}
	require.NoError(t, p.OnAfterSignIn(context.Background(), u, s1))

	// Set last login beyond lookback window (2 hours ago, lookback is 1h)
	p.mu.Lock()
	p.lastLogins[userID.String()].LoginAt = time.Now().Add(-2 * time.Hour)
	p.mu.Unlock()

	s2 := &session.Session{IPAddress: "2.2.2.2"}
	err := p.OnAfterSignIn(context.Background(), u, s2)
	assert.NoError(t, err)
}
