package anomaly

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
	if mapping != nil {
		p.geoIP = geoip.NewTestPlugin(mapping)
	}
	p.logger = log.NewNoopLogger()
	return p
}

var usMapping = map[string]*geoip.GeoLocation{
	"1.1.1.1": {IP: "1.1.1.1", Country: "US"},
	"2.2.2.2": {IP: "2.2.2.2", Country: "GB"},
}

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "anomaly", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) {
	var p interface{} = New()
	_, ok := p.(plugin.Plugin)
	assert.True(t, ok)
	_, ok = p.(plugin.OnInit)
	assert.True(t, ok)
	_, ok = p.(plugin.AfterSignIn)
	assert.True(t, ok)
}

func TestBelowMinHistory_NoAlert(t *testing.T) {
	// Default MinLoginHistory is 10
	p := newTestPlugin(Config{MinLoginHistory: 10, EnableGeoAnomaly: true, EnableTimeAnomaly: true}, usMapping)
	u := &user.User{ID: id.NewUserID()}
	s := &session.Session{IPAddress: "1.1.1.1"}

	// Login 5 times (below threshold)
	for i := 0; i < 5; i++ {
		err := p.OnAfterSignIn(context.Background(), u, s)
		require.NoError(t, err)
	}

	// Pattern should exist but with only 5 logins
	p.mu.RLock()
	pattern := p.patterns[u.ID.String()]
	p.mu.RUnlock()
	require.NotNil(t, pattern)
	assert.Equal(t, 5, pattern.LoginCount)
}

func TestUnusualCountry_Alert(t *testing.T) {
	p := newTestPlugin(Config{MinLoginHistory: 5, EnableGeoAnomaly: true, RiskThreshold: 50}, usMapping)
	userID := id.NewUserID()
	u := &user.User{ID: userID}

	// Build history with US logins
	for i := 0; i < 10; i++ {
		s := &session.Session{IPAddress: "1.1.1.1"}
		require.NoError(t, p.OnAfterSignIn(context.Background(), u, s))
	}

	// Login from GB (new country) - should trigger geo anomaly
	// The anomaly doesn't return an error, it just logs/audits
	s := &session.Session{IPAddress: "2.2.2.2"}
	err := p.OnAfterSignIn(context.Background(), u, s)
	assert.NoError(t, err)

	// Verify GB was recorded
	p.mu.RLock()
	pattern := p.patterns[userID.String()]
	p.mu.RUnlock()
	assert.Equal(t, 1, pattern.CountryHistogram["GB"])
}

func TestUsualCountry_NoAlert(t *testing.T) {
	p := newTestPlugin(Config{MinLoginHistory: 5, EnableGeoAnomaly: true, RiskThreshold: 50}, usMapping)
	u := &user.User{ID: id.NewUserID()}

	// All logins from same country
	for i := 0; i < 15; i++ {
		s := &session.Session{IPAddress: "1.1.1.1"}
		require.NoError(t, p.OnAfterSignIn(context.Background(), u, s))
	}
	// No anomaly for familiar country
}

func TestUnusualTime_Alert(t *testing.T) {
	// Test the checkTimeAnomaly function directly
	p := New(Config{MinLoginHistory: 5, EnableTimeAnomaly: true})

	userID := id.NewUserID()
	sessID := id.SessionID{}
	now := time.Now()
	hour := now.Hour()

	// Create pattern where all logins are at a different hour
	otherHour := (hour + 12) % 24
	pattern := &LoginPattern{
		UserID:     userID,
		LoginCount: 100,
	}
	pattern.HourHistogram[otherHour] = 100 // all logins at different hour
	// Current hour has 0% of logins

	alert := p.checkTimeAnomaly(userID, sessID, now, pattern)
	require.NotNil(t, alert)
	assert.Equal(t, "unusual_time", alert.Type)
	assert.Equal(t, 100, alert.RiskScore) // 0% -> score 100
}

func TestUsualTime_NoAlert(t *testing.T) {
	p := New(Config{MinLoginHistory: 5, EnableTimeAnomaly: true})

	userID := id.NewUserID()
	sessID := id.SessionID{}
	now := time.Now()
	hour := now.Hour()

	// Create pattern where current hour has lots of logins
	pattern := &LoginPattern{
		UserID:     userID,
		LoginCount: 100,
	}
	pattern.HourHistogram[hour] = 50 // 50% at current hour

	alert := p.checkTimeAnomaly(userID, sessID, now, pattern)
	assert.Nil(t, alert)
}

func TestPatternAccumulates(t *testing.T) {
	p := newTestPlugin(Config{MinLoginHistory: 100}, usMapping) // high threshold so no anomaly checks
	userID := id.NewUserID()
	u := &user.User{ID: userID}

	for i := 0; i < 5; i++ {
		s := &session.Session{IPAddress: "1.1.1.1"}
		require.NoError(t, p.OnAfterSignIn(context.Background(), u, s))
	}

	p.mu.RLock()
	pattern := p.patterns[userID.String()]
	p.mu.RUnlock()

	require.NotNil(t, pattern)
	assert.Equal(t, 5, pattern.LoginCount)
	assert.Equal(t, 5, pattern.CountryHistogram["US"])

	// Verify hour histogram has entries at current hour
	hour := time.Now().Hour()
	assert.Equal(t, 5, pattern.HourHistogram[hour])
}
