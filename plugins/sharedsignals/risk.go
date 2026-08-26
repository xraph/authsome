package sharedsignals

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/riskengine"
	"github.com/xraph/authsome/settings"
)

var _ riskengine.RiskContributor = (*Plugin)(nil)

// caepSignalWeight is how much more a CAEP event counts than an IP heuristic.
// An identity provider that watched an account get taken over is a far better
// source than a reputation list. It is the fallback for
// sharedsignals.risk_weight, which an operator can raise or lower per app.
const caepSignalWeight = 2.0

// EvaluateRisk replays stored signals at sign-in time. This is how an event
// that arrived at 02:00, when nobody was signing in, reaches the 08:00 login
// that cares about it. A high score crosses riskengine's medium threshold and
// its decision becomes "challenge", which is how step-up is expressed without
// inventing a second mechanism.
func (p *Plugin) EvaluateRisk(ctx context.Context, req *riskengine.RiskRequest) (*riskengine.RiskSignal, error) {
	none := &riskengine.RiskSignal{
		Source: "sharedsignals", Score: 0, Weight: caepSignalWeight,
	}

	if p.store == nil || req == nil || req.AppID == "" {
		return none, nil
	}
	appID, err := id.ParseAppID(req.AppID)
	if err != nil {
		return none, nil //nolint:nilerr // an unparseable app is not our failure
	}
	weight := p.riskWeight(ctx, appID)
	none.Weight = weight
	// EnvID is optional on the request; an empty/unparseable value resolves
	// to id.Nil, which the user lookups below treat as "any environment".
	var envID id.EnvironmentID
	if req.EnvID != "" {
		if parsed, perr := id.ParseEnvironmentID(req.EnvID); perr == nil {
			envID = parsed
		}
	}

	userID, resolvedEnvID, ok := p.resolveRiskUser(ctx, appID, envID, req)
	if !ok {
		return none, nil
	}

	now := time.Now()
	signals, err := p.store.ListActiveSignals(ctx, appID, resolvedEnvID, userID, now)
	if err != nil {
		return none, err
	}
	if len(signals) == 0 {
		return none, nil
	}

	best := 0
	reason := ""
	for _, s := range signals {
		score := decayedSeverity(s, now)
		if score > best {
			best = score
			reason = s.EventType
		}
	}

	// Cap what we hand the engine. The stored severity stays as it was, so
	// the audit trail and any later dashboard still see how serious the event
	// actually was; this only bounds how far one signal may move the
	// composite decision. Without it a session-revoked event scored 100,
	// crossed riskengine's block threshold on its own, and refused a sign-in
	// that should have been stepped up instead.
	if ceiling := p.maxRiskScore(ctx, appID); best > ceiling {
		best = ceiling
	}

	return &riskengine.RiskSignal{
		Source: "sharedsignals",
		Score:  best,
		Weight: weight,
		Reason: "shared signals event: " + reason,
	}, nil
}

// resolveRiskUser turns whatever identifies the sign-in attempt into a user.
// UserID is preferred when the caller already knows it. The returned
// environment is the user's actual EnvID, not necessarily the request's: a
// lookup by email or username is deliberately env-agnostic (an empty envID
// matches any environment, see the store), so the request's EnvID may be
// unset or wrong. Signals are recorded per environment, so ListActiveSignals
// must be queried with the environment the resolved user actually belongs
// to, not the one the request guessed at.
func (p *Plugin) resolveRiskUser(ctx context.Context, appID id.AppID, envID id.EnvironmentID,
	req *riskengine.RiskRequest) (resolvedUserID id.UserID, resolvedEnvID id.EnvironmentID, resolved bool) {
	if req.UserID != "" {
		if userID, err := id.ParseUserID(req.UserID); err == nil {
			return userID, envID, true
		}
	}
	if p.authStore == nil {
		return id.Nil, id.Nil, false
	}
	if req.Email != "" {
		if u, err := p.authStore.GetUserByEmail(ctx, appID, envID, req.Email); err == nil && u != nil {
			return u.ID, u.EnvID, true
		}
	}
	if req.Username != "" {
		if u, err := p.authStore.GetUserByUsername(ctx, appID, envID, req.Username); err == nil && u != nil {
			return u.ID, u.EnvID, true
		}
	}
	return id.Nil, id.Nil, false
}

// decayedSeverity fades a signal linearly from its full severity at
// EventAt to zero at ExpiresAt.
func decayedSeverity(s *Signal, now time.Time) int {
	total := s.ExpiresAt.Sub(s.EventAt)
	if total <= 0 {
		return 0
	}
	remaining := s.ExpiresAt.Sub(now)
	if remaining <= 0 {
		return 0
	}
	if remaining > total {
		remaining = total
	}
	return int(float64(s.Severity) * (float64(remaining) / float64(total)))
}

// maxRiskScore resolves the cap, preferring an operator's per-app setting
// over the compiled-in Config value.
func (p *Plugin) maxRiskScore(ctx context.Context, appID id.AppID) int {
	if p.settingsMgr == nil {
		return p.config.MaxRiskScore
	}
	v, err := settings.Get(ctx, p.settingsMgr, SettingMaxRiskScore,
		settings.ResolveOpts{AppID: appID.String()})
	if err != nil {
		p.logger.Warn("sharedsignals: could not resolve the max risk score setting",
			logString("error", err.Error()),
		)
		return p.config.MaxRiskScore
	}
	if v <= 0 {
		return p.config.MaxRiskScore
	}
	return v
}
