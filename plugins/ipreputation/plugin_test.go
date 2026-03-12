package ipreputation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/plugin"
)

type mockProvider struct {
	results map[string]*IPReputation
	err     error
	calls   int
}

func (m *mockProvider) CheckIP(_ context.Context, ip string) (*IPReputation, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	rep, ok := m.results[ip]
	if !ok {
		return &IPReputation{IP: ip, Score: 0}, nil
	}
	return rep, nil
}

func newTestPlugin(provider Provider, blockThreshold, warnThreshold int) *Plugin { //nolint:unparam // test helper
	p := New(Config{
		Provider:       provider,
		BlockThreshold: blockThreshold,
		WarnThreshold:  warnThreshold,
		CacheTTL:       time.Hour,
	})
	p.logger = log.NewNoopLogger()
	return p
}

func TestPlugin_Name(t *testing.T) {
	p := New(Config{})
	assert.Equal(t, "ipreputation", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) {
	var p interface{} = New(Config{})
	_, ok := p.(plugin.Plugin)
	assert.True(t, ok)
	_, ok = p.(plugin.OnInit)
	assert.True(t, ok)
	_, ok = p.(plugin.BeforeSignIn)
	assert.True(t, ok)
	_, ok = p.(plugin.BeforeSignUp)
	assert.True(t, ok)
}

func TestHighScore_Blocks(t *testing.T) {
	mp := &mockProvider{results: map[string]*IPReputation{
		"5.5.5.5": {IP: "5.5.5.5", Score: 90},
	}}
	p := newTestPlugin(mp, 80, 50)
	err := p.check(context.Background(), "5.5.5.5", "app1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ipreputation:")
}

func TestMidScore_Warns(t *testing.T) {
	mp := &mockProvider{results: map[string]*IPReputation{
		"5.5.5.5": {IP: "5.5.5.5", Score: 60},
	}}
	p := newTestPlugin(mp, 80, 50)
	err := p.check(context.Background(), "5.5.5.5", "app1")
	assert.NoError(t, err) // warns but doesn't block
}

func TestLowScore_Passes(t *testing.T) {
	mp := &mockProvider{results: map[string]*IPReputation{
		"5.5.5.5": {IP: "5.5.5.5", Score: 10},
	}}
	p := newTestPlugin(mp, 80, 50)
	err := p.check(context.Background(), "5.5.5.5", "app1")
	assert.NoError(t, err)
}

func TestBlacklisted_Blocks(t *testing.T) {
	mp := &mockProvider{results: map[string]*IPReputation{
		"5.5.5.5": {IP: "5.5.5.5", Score: 10, IsBlacklisted: true},
	}}
	p := newTestPlugin(mp, 80, 50)
	err := p.check(context.Background(), "5.5.5.5", "app1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ipreputation:")
}

func TestNoProvider_NoOp(t *testing.T) {
	p := New(Config{}) // no provider
	p.logger = log.NewNoopLogger()
	err := p.check(context.Background(), "5.5.5.5", "app1")
	assert.NoError(t, err)
}

func TestProviderError_FailOpen(t *testing.T) {
	mp := &mockProvider{err: errors.New("network error")}
	p := newTestPlugin(mp, 80, 50)
	err := p.check(context.Background(), "5.5.5.5", "app1")
	assert.NoError(t, err) // fail open
}

func TestCache_HitsCache(t *testing.T) {
	mp := &mockProvider{results: map[string]*IPReputation{
		"5.5.5.5": {IP: "5.5.5.5", Score: 10},
	}}
	p := newTestPlugin(mp, 80, 50)

	require.NoError(t, p.check(context.Background(), "5.5.5.5", "app1"))
	require.NoError(t, p.check(context.Background(), "5.5.5.5", "app1"))
	assert.Equal(t, 1, mp.calls, "provider should only be called once due to cache")
}
