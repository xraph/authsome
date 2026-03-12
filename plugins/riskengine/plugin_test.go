package riskengine

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugin"
)

type mockContributor struct {
	name   string
	score  int
	weight float64
	err    error
}

func (m *mockContributor) Name() string { return m.name }

func (m *mockContributor) EvaluateRisk(_ context.Context, _ *RiskRequest) (*RiskSignal, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &RiskSignal{Source: m.name, Score: m.score, Weight: m.weight, Reason: "test"}, nil
}

func newTestPlugin(cfg Config, contributors ...RiskContributor) *Plugin {
	p := NewWithConfig(cfg, contributors...)
	p.logger = log.NewNoopLogger()
	return p
}

func TestPlugin_Name(t *testing.T) {
	p := New()
	assert.Equal(t, "riskengine", p.Name())
}

func TestPlugin_ImplementsInterfaces(t *testing.T) {
	var p interface{} = New()

	_, ok := p.(plugin.Plugin)
	assert.True(t, ok)

	_, ok = p.(plugin.OnInit)
	assert.True(t, ok)

	_, ok = p.(plugin.BeforeSignIn)
	assert.True(t, ok)

	_, ok = p.(plugin.BeforeSessionCreate)
	assert.True(t, ok)
}

func TestNoContributors_Allow(t *testing.T) {
	p := newTestPlugin(Config{})

	appID, _ := id.ParseAppID("aapp_01jf0000000000000000000000")
	err := p.OnBeforeSignIn(context.Background(), &account.SignInRequest{AppID: appID})
	assert.NoError(t, err)
}

func TestLowScore_Allow(t *testing.T) {
	contrib := &mockContributor{name: "test", score: 20, weight: 1.0}
	p := newTestPlugin(Config{}, contrib)

	assessment := p.evaluate(context.Background(), &RiskRequest{IPAddress: "1.2.3.4"})
	assert.Equal(t, "allow", assessment.Decision)
	assert.Equal(t, 20, assessment.OverallScore)
}

func TestMediumScore_Challenge(t *testing.T) {
	contrib := &mockContributor{name: "test", score: 65, weight: 1.0}
	p := newTestPlugin(Config{}, contrib)

	assessment := p.evaluate(context.Background(), &RiskRequest{IPAddress: "1.2.3.4"})
	assert.Equal(t, "challenge", assessment.Decision)
	assert.Equal(t, 65, assessment.OverallScore)
}

func TestHighScore_Block(t *testing.T) {
	contrib := &mockContributor{name: "test", score: 90, weight: 1.0}
	p := newTestPlugin(Config{}, contrib)

	assessment := p.evaluate(context.Background(), &RiskRequest{IPAddress: "1.2.3.4"})
	assert.Equal(t, "block", assessment.Decision)
	assert.Equal(t, 90, assessment.OverallScore)

	// OnBeforeSignIn should return an error for a blocked request.
	appID, _ := id.ParseAppID("aapp_01jf0000000000000000000000")
	err := p.OnBeforeSignIn(context.Background(), &account.SignInRequest{
		AppID:     appID,
		IPAddress: "1.2.3.4",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "riskengine:")
}

func TestWeightedScoring(t *testing.T) {
	c1 := &mockContributor{name: "c1", score: 80, weight: 1.0}
	c2 := &mockContributor{name: "c2", score: 20, weight: 1.0}

	// With configured weights: c1 has weight 2.0, c2 has weight 1.0.
	// Weighted average: (80*2 + 20*1) / (2+1) = 180/3 = 60
	p := newTestPlugin(Config{
		Weights: map[string]float64{"c1": 2.0, "c2": 1.0},
	}, c1, c2)

	assessment := p.evaluate(context.Background(), &RiskRequest{})
	assert.Equal(t, 60, assessment.OverallScore)
}

func TestContributorError_Skipped(t *testing.T) {
	good := &mockContributor{name: "good", score: 20, weight: 1.0}
	bad := &mockContributor{name: "bad", err: errors.New("failed")}
	p := newTestPlugin(Config{}, good, bad)

	assessment := p.evaluate(context.Background(), &RiskRequest{})
	assert.Equal(t, 20, assessment.OverallScore)
	assert.Len(t, assessment.Signals, 1)
}

func TestAddContributor(t *testing.T) {
	p := newTestPlugin(Config{})
	assert.Empty(t, p.contributors)

	c := &mockContributor{name: "added", score: 50, weight: 1.0}
	p.AddContributor(c)
	assert.Len(t, p.contributors, 1)

	assessment := p.evaluate(context.Background(), &RiskRequest{})
	assert.Equal(t, 50, assessment.OverallScore)
}

func TestAuditAssessment(t *testing.T) { //nolint:revive // test function signature
	// Verify auditAssessment does not panic when chronicle is nil.
	contrib := &mockContributor{name: "test", score: 50, weight: 1.0}
	p := newTestPlugin(Config{}, contrib)

	req := &RiskRequest{IPAddress: "1.2.3.4", AppID: "app1"}
	assessment := p.evaluate(context.Background(), req)
	p.auditAssessment(context.Background(), req, assessment)
}
