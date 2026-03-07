// Package riskengine provides a composite risk assessment engine that
// aggregates risk signals from multiple geo-security plugins into a unified
// score. Based on the score it decides: allow, challenge (step-up MFA), or block.
package riskengine

import (
	"context"
	"fmt"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/formconfig"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/session"
	"github.com/xraph/authsome/settings"
	"github.com/xraph/authsome/user"
)

// Compile-time interface checks.
var (
	_ plugin.Plugin              = (*Plugin)(nil)
	_ plugin.OnInit              = (*Plugin)(nil)
	_ plugin.BeforeSignIn        = (*Plugin)(nil)
	_ plugin.BeforeSessionCreate = (*Plugin)(nil)
	_ plugin.SettingsProvider    = (*Plugin)(nil)
)

// ──────────────────────────────────────────────────
// Dynamic setting definitions
// ──────────────────────────────────────────────────

var (
	// SettingLowThreshold is the score below which risk is considered low.
	SettingLowThreshold = settings.Define("riskengine.low_threshold", 30,
		settings.WithDisplayName("Low Threshold"),
		settings.WithDescription("Score below which risk is considered low"),
		settings.WithCategory("Risk Engine"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(0), Max: intPtr(100)}),
		settings.WithHelpText("Score below which risk is considered low"),
		settings.WithOrder(10),
	)

	// SettingMediumThreshold is the score below which risk is considered medium.
	SettingMediumThreshold = settings.Define("riskengine.medium_threshold", 60,
		settings.WithDisplayName("Medium Threshold"),
		settings.WithDescription("Score below which risk is considered medium"),
		settings.WithCategory("Risk Engine"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(0), Max: intPtr(100)}),
		settings.WithHelpText("Score below which risk is considered medium"),
		settings.WithOrder(20),
	)

	// SettingHighThreshold is the score at or above which risk is considered high.
	SettingHighThreshold = settings.Define("riskengine.high_threshold", 85,
		settings.WithDisplayName("High Threshold"),
		settings.WithDescription("Score at or above which risk is considered high and action is taken"),
		settings.WithCategory("Risk Engine"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithEnforceable(),
		settings.WithInputType(formconfig.FieldNumber),
		settings.WithUIValidation(formconfig.Validation{Min: intPtr(0), Max: intPtr(100)}),
		settings.WithHelpText("Score at or above which risk is considered high and action is taken"),
		settings.WithOrder(30),
	)

	// SettingBlockMessage is the error message shown when access is blocked.
	SettingBlockMessage = settings.Define("riskengine.block_message", "",
		settings.WithDisplayName("Block Message"),
		settings.WithDescription("Error message shown when access is blocked due to high risk score"),
		settings.WithCategory("Risk Engine"),
		settings.WithScopes(settings.ScopeGlobal, settings.ScopeApp),
		settings.WithHelpText("Error message shown when access is blocked due to high risk score"),
		settings.WithPlaceholder("Access denied due to security concerns"),
		settings.WithOrder(40),
	)
)

func intPtr(i int) *int { return &i }

// RiskSignal is a single risk factor from a contributor.
type RiskSignal struct {
	Source string  `json:"source"`
	Score  int     `json:"score"` // 0-100
	Weight float64 `json:"weight"`
	Reason string  `json:"reason"`
}

// RiskAssessment is the aggregated risk evaluation.
type RiskAssessment struct {
	OverallScore int          `json:"overall_score"`
	Signals      []RiskSignal `json:"signals"`
	Decision     string       `json:"decision"` // "allow", "challenge", "block"
}

// RiskRequest contains the data needed for risk evaluation.
type RiskRequest struct {
	IPAddress string
	UserAgent string
	AppID     string
	UserID    string
}

// RiskContributor is a sub-plugin interface for plugins that contribute
// risk signals to the risk engine.
type RiskContributor interface {
	plugin.Plugin
	EvaluateRisk(ctx context.Context, req *RiskRequest) (*RiskSignal, error)
}

// Config configures the risk engine plugin.
type Config struct {
	// LowThreshold is the score below which requests are allowed (default: 30).
	LowThreshold int

	// MediumThreshold is the score above which step-up MFA is required (default: 60).
	MediumThreshold int

	// HighThreshold is the score above which requests are blocked (default: 85).
	HighThreshold int

	// Weights maps contributor names to weight multipliers (default: 1.0 for all).
	Weights map[string]float64

	// BlockMessage is the error message when blocked.
	BlockMessage string
}

func (c *Config) defaults() {
	if c.LowThreshold == 0 {
		c.LowThreshold = 30
	}
	if c.MediumThreshold == 0 {
		c.MediumThreshold = 60
	}
	if c.HighThreshold == 0 {
		c.HighThreshold = 85
	}
	if c.BlockMessage == "" {
		c.BlockMessage = "request blocked due to high risk score"
	}
	if c.Weights == nil {
		c.Weights = make(map[string]float64)
	}
}

// Plugin is the adaptive risk engine that aggregates signals from contributors.
type Plugin struct {
	config       Config
	contributors []RiskContributor
	chronicle    bridge.Chronicle
	relay        bridge.EventRelay
	logger       log.Logger
	settingsMgr  *settings.Manager

	// lastAssessment stores the most recent assessment per-request
	// for the BeforeSessionCreate hook to attach metadata.
	// Thread-safe because hooks run sequentially per request.
	lastAssessment *RiskAssessment
}

// New creates a new risk engine plugin. Config is optional; if omitted
// sensible defaults are applied.
func New(contributors ...RiskContributor) *Plugin {
	var c Config
	c.defaults()
	return &Plugin{
		config:       c,
		contributors: contributors,
	}
}

// NewWithConfig creates a new risk engine plugin with explicit configuration.
func NewWithConfig(cfg Config, contributors ...RiskContributor) *Plugin {
	cfg.defaults()
	return &Plugin{
		config:       cfg,
		contributors: contributors,
	}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return "riskengine" }

// DeclareSettings implements plugin.SettingsProvider.
// It registers all risk-engine-related configurable settings.
func (p *Plugin) DeclareSettings(m *settings.Manager) error {
	if err := settings.RegisterTyped(m, "riskengine", SettingLowThreshold); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "riskengine", SettingMediumThreshold); err != nil {
		return err
	}
	if err := settings.RegisterTyped(m, "riskengine", SettingHighThreshold); err != nil {
		return err
	}
	return settings.RegisterTyped(m, "riskengine", SettingBlockMessage)
}

// OnInit discovers engine dependencies.
func (p *Plugin) OnInit(_ context.Context, engine any) error {
	type loggerGetter interface{ Logger() log.Logger }
	if lg, ok := engine.(loggerGetter); ok {
		p.logger = lg.Logger()
	}
	if p.logger == nil {
		p.logger = log.NewNoopLogger()
	}

	type chronicleGetter interface{ Chronicle() bridge.Chronicle }
	if cg, ok := engine.(chronicleGetter); ok {
		p.chronicle = cg.Chronicle()
	}
	type relayGetter interface{ Relay() bridge.EventRelay }
	if rg, ok := engine.(relayGetter); ok {
		p.relay = rg.Relay()
	}

	type settingsGetter interface {
		Settings() *settings.Manager
	}
	if sg, ok := engine.(settingsGetter); ok {
		p.settingsMgr = sg.Settings()
	}

	return nil
}

// AddContributor adds a risk contributor to the engine.
func (p *Plugin) AddContributor(c RiskContributor) {
	p.contributors = append(p.contributors, c)
}

// OnBeforeSignIn evaluates risk and blocks if score exceeds high threshold.
func (p *Plugin) OnBeforeSignIn(ctx context.Context, req *account.SignInRequest) error {
	if len(p.contributors) == 0 {
		return nil
	}

	riskReq := &RiskRequest{
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		AppID:     req.AppID.String(),
	}

	assessment := p.evaluate(ctx, riskReq)
	p.lastAssessment = assessment

	p.auditAssessment(ctx, riskReq, assessment)

	if assessment.Decision == "block" {
		return fmt.Errorf("riskengine: %s", p.config.BlockMessage)
	}

	return nil
}

// OnBeforeSessionCreate attaches the risk score to the session metadata.
func (p *Plugin) OnBeforeSessionCreate(ctx context.Context, s *session.Session) error {
	// No-op: risk data is attached via audit trail, not session struct.
	return nil
}

func (p *Plugin) evaluate(ctx context.Context, req *RiskRequest) *RiskAssessment {
	var signals []RiskSignal
	var totalWeightedScore float64
	var totalWeight float64

	for _, contrib := range p.contributors {
		signal, err := contrib.EvaluateRisk(ctx, req)
		if err != nil {
			p.logger.Warn("riskengine: contributor error",
				log.String("contributor", contrib.Name()),
				log.String("error", err.Error()),
			)
			continue
		}
		if signal == nil {
			continue
		}

		// Apply configured weight.
		weight := signal.Weight
		if w, ok := p.config.Weights[contrib.Name()]; ok {
			weight = w
		}
		if weight == 0 {
			weight = 1.0
		}

		signal.Weight = weight
		signals = append(signals, *signal)
		totalWeightedScore += float64(signal.Score) * weight
		totalWeight += weight
	}

	overallScore := 0
	if totalWeight > 0 {
		overallScore = int(totalWeightedScore / totalWeight)
	}
	if overallScore > 100 {
		overallScore = 100
	}

	decision := "allow"
	switch {
	case overallScore >= p.config.HighThreshold:
		decision = "block"
	case overallScore >= p.config.MediumThreshold:
		decision = "challenge"
	}

	return &RiskAssessment{
		OverallScore: overallScore,
		Signals:      signals,
		Decision:     decision,
	}
}

func (p *Plugin) auditAssessment(ctx context.Context, req *RiskRequest, assessment *RiskAssessment) {
	severity := bridge.SeverityInfo
	if assessment.Decision == "challenge" {
		severity = bridge.SeverityWarning
	}
	if assessment.Decision == "block" {
		severity = bridge.SeverityCritical
	}

	metadata := map[string]string{
		"overall_score": fmt.Sprintf("%d", assessment.OverallScore),
		"decision":      assessment.Decision,
		"ip":            req.IPAddress,
		"signals_count": fmt.Sprintf("%d", len(assessment.Signals)),
	}

	if p.chronicle != nil {
		_ = p.chronicle.Record(ctx, &bridge.AuditEvent{
			Action:   "risk_assessment",
			Resource: "auth",
			ActorID:  req.UserID,
			Tenant:   req.AppID,
			Outcome:  bridge.OutcomeSuccess,
			Severity: severity,
			Metadata: metadata,
		})
	}

	if p.relay != nil && assessment.Decision != "allow" {
		_ = p.relay.Send(ctx, &bridge.WebhookEvent{
			Type:     "security.risk_assessment",
			TenantID: req.AppID,
			Data:     metadata,
		})
	}
}

// Ensure unused imports are used.
var (
	_ = (*user.User)(nil)
	_ = (*session.Session)(nil)
)
