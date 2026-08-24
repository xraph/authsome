package sharedsignals

import (
	"context"
	"fmt"
	"time"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
)

// Actions an event can produce.
const (
	ActionRevokeAll     = "revoke_all"
	ActionRevokeSession = "revoke_session"
	ActionSignal        = "signal"
	ActionLog           = "log"
	ActionNone          = "none"
)

// actionFor decides what an event does. A stream override always wins over
// the default, so an operator can quiet a noisy transmitter per event type
// without turning the whole stream off.
func (p *Plugin) actionFor(s *InboundStream, ev caep.Event) string {
	if override, ok := s.ActionOverrides[ev.Type]; ok && override != "" {
		return override
	}

	switch ev.Type {
	case caep.EventSessionRevoked:
		return ActionRevokeAll

	case caep.EventTokenClaimsChange:
		// session.Roles is stamped at issue time and never re-resolved, so a
		// claims change cannot reach a live session. Ending it is the only
		// way to pick up the new claims.
		return ActionRevokeAll

	case caep.EventCredentialChange:
		if ev.ChangeType == "revoke" || ev.ChangeType == "delete" {
			return ActionRevokeAll
		}
		return ActionSignal

	case caep.EventAssuranceLevelChange,
		caep.EventDeviceComplianceChange,
		caep.EventRiskLevelChange:
		return ActionSignal

	case caep.EventVerification:
		return ActionNone

	default:
		return ActionSignal
	}
}

// severityFor scores an event 0 to 100 for the risk engine.
func (p *Plugin) severityFor(ev caep.Event) int {
	switch ev.Type {
	case caep.EventSessionRevoked:
		return 100

	case caep.EventTokenClaimsChange:
		return 50

	case caep.EventCredentialChange:
		if ev.ChangeType == "revoke" || ev.ChangeType == "delete" {
			return 90
		}
		return 20

	case caep.EventAssuranceLevelChange:
		if ev.ChangeDirection == "decrease" {
			return 70
		}
		return 10

	case caep.EventDeviceComplianceChange:
		if ev.CurrentStatus == "not-compliant" {
			return 60
		}
		return 10

	case caep.EventRiskLevelChange:
		switch ev.CurrentLevel {
		case "HIGH":
			return 80
		case "MEDIUM":
			return 50
		default:
			return 10
		}

	default:
		return 30
	}
}

// applyEvent runs the matrix for one resolved event and returns the action it
// actually took. A signal is always recorded first, so even an action that is
// skipped or fails still leaves the next sign-in something to score.
func (p *Plugin) applyEvent(ctx context.Context, s *InboundStream,
	ev caep.Event, res Resolution) (string, error) {
	if ev.Type == caep.EventVerification {
		return "", p.completeVerification(ctx, s, ev)
	}

	action := p.actionFor(s, ev)

	if err := p.recordSignal(ctx, s, ev, res); err != nil {
		return "", err
	}

	if action == ActionSignal || action == ActionNone || action == ActionLog {
		return "", nil
	}

	// Observe mode records everything and changes nothing.
	if s.EnforcementMode == EnforcementObserve {
		p.audit(ctx, s, ev, res, bridge.SeverityWarning, "would_"+action)
		return ActionLog, nil
	}

	if p.revoker == nil {
		return "", fmt.Errorf("sharedsignals: cannot revoke, engine does not support it")
	}

	// A session member in the subject narrows this to one session.
	if !res.SessionID.IsNil() && action == ActionRevokeAll {
		action = ActionRevokeSession
	}

	switch action {
	case ActionRevokeSession:
		if err := p.revoker.RevokeSession(ctx, res.SessionID); err != nil {
			return "", err
		}
	case ActionRevokeAll:
		sessions, err := p.authStore.ListUserSessions(ctx, res.UserID)
		if err != nil {
			return "", err
		}
		for _, sess := range sessions {
			// Stay inside this stream's app. A stream never reaches a
			// session it was not scoped to.
			if sess.AppID != s.AppID {
				continue
			}
			if err := p.revoker.RevokeSession(ctx, sess.ID); err != nil {
				return "", err
			}
		}
	}

	p.audit(ctx, s, ev, res, bridge.SeverityCritical, action)
	return action, nil
}

func (p *Plugin) recordSignal(ctx context.Context, s *InboundStream,
	ev caep.Event, res Resolution) error {
	now := time.Now()
	eventAt := now
	if ev.EventTimestamp > 0 {
		// CAEP timestamps are seconds in the spec but Okta sends
		// milliseconds, so treat anything implausibly large as millis.
		if ev.EventTimestamp > 1e11 {
			eventAt = time.UnixMilli(ev.EventTimestamp)
		} else {
			eventAt = time.Unix(ev.EventTimestamp, 0)
		}
	}

	reason := ""
	if ev.ReasonAdmin != nil {
		reason = ev.ReasonAdmin["en"]
	}

	return p.store.CreateSignal(ctx, &Signal{
		ID:        id.NewSSFSignalID(),
		AppID:     s.AppID,
		EnvID:     s.EnvID,
		UserID:    res.UserID,
		StreamID:  s.ID,
		EventType: ev.Type,
		Severity:  p.severityFor(ev),
		Reason:    reason,
		EventAt:   eventAt,
		ExpiresAt: now.Add(p.config.SignalTTL),
		CreatedAt: now,
	})
}

// completeVerification matches the echoed state against what we sent. A
// mismatch is not an error to the transmitter, it just does not mark the
// stream verified.
func (p *Plugin) completeVerification(ctx context.Context, s *InboundStream,
	ev caep.Event) error {
	if s.PendingVerifyState == "" || ev.State != s.PendingVerifyState {
		return nil
	}
	now := time.Now()
	s.LastVerifiedAt = &now
	s.PendingVerifyState = ""
	return p.store.UpdateInboundStream(ctx, s)
}

// checkCircuitBreaker reports whether the stream may still act. Crossing the
// limit pauses the stream and raises an alert, because a transmitter asking
// for thousands of revocations is either compromised or misconfigured and
// both want the same answer.
func (p *Plugin) checkCircuitBreaker(ctx context.Context, s *InboundStream) (bool, error) {
	limit := s.MaxActionsPerHour
	if limit <= 0 {
		limit = p.config.MaxActionsPerHour
	}

	count, err := p.store.CountActionsSince(ctx, s.ID, time.Now().Add(-time.Hour))
	if err != nil {
		return false, err
	}
	if count < limit {
		return true, nil
	}

	s.Status = StatusPaused
	if err := p.store.UpdateInboundStream(ctx, s); err != nil {
		return false, err
	}

	if p.chronicle != nil {
		_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
			Action:   "ssf_circuit_breaker_tripped",
			Resource: "sharedsignals_stream",
			Tenant:   s.AppID.String(),
			Outcome:  bridge.OutcomeFailure,
			Severity: bridge.SeverityCritical,
			Metadata: map[string]string{
				"stream_id": s.ID.String(),
				"issuer":    s.Issuer,
				"limit":     fmt.Sprintf("%d", limit),
				"count":     fmt.Sprintf("%d", count),
			},
		})
	}
	if p.relay != nil {
		_ = p.relay.Send(ctx, &bridge.WebhookEvent{ //nolint:errcheck // best-effort webhook
			Type:     "security.ssf.circuit_breaker_tripped",
			TenantID: s.AppID.String(),
			Data: map[string]string{
				"stream_id": s.ID.String(),
				"issuer":    s.Issuer,
			},
		})
	}

	if p.logger != nil {
		p.logger.Warn("sharedsignals: circuit breaker tripped, stream paused",
			logString("stream_id", s.ID.String()),
			logString("issuer", s.Issuer),
		)
	}
	return false, nil
}

// audit records what applyEvent did. severity is one of the bridge.Severity*
// string constants; bridge does not export a dedicated type for it.
func (p *Plugin) audit(ctx context.Context, s *InboundStream, ev caep.Event,
	res Resolution, severity string, action string) {
	if p.chronicle == nil {
		return
	}
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
		Action:   "ssf_event_applied",
		Resource: "session",
		ActorID:  res.UserID.String(),
		Tenant:   s.AppID.String(),
		Outcome:  bridge.OutcomeSuccess,
		Severity: severity,
		Metadata: map[string]string{
			"stream_id":  s.ID.String(),
			"issuer":     s.Issuer,
			"event_type": ev.Type,
			"action":     action,
		},
	})
}
