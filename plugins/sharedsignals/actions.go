package sharedsignals

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/plugins/sharedsignals/caep"
)

// Sanity bounds for an event's forensic timestamp. A SET's event_timestamp is
// attacker-influenced, and our own seconds-vs-millis heuristic can turn a
// hostile value into a wildly implausible time (e.g. 99999999999 reads as
// seconds and lands around the year 5138). Nothing consumes EventAt today,
// but it is a field an investigator will read later, so an implausible value
// is clamped to now rather than stored as-is.
const (
	maxEventTimestampFuture = 24 * time.Hour
	maxEventTimestampPast   = 20 * 365 * 24 * time.Hour
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
//
// The caller MUST have already called checkCircuitBreaker for this stream
// and confirmed it returned true before calling applyEvent. applyEvent has
// no way to see how many other events for this stream are in flight or how
// many actions the stream has already taken this hour, so it cannot bound
// its own blast radius -- that is entirely the breaker's job. Skipping the
// breaker check defeats it.
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
		p.audit(ctx, s, ev, res, bridge.SeverityWarning, "would_"+action, nil)
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
		// subject.go guarantees res.SessionID belongs to res.UserID, but it
		// never checks which app the session lives in. Load it and check
		// that here -- the exact same rule the revoke-all loop below
		// enforces -- and refuse rather than falling back to a wider
		// action: a stream that names a session outside its own app gets
		// nothing revoked, not everything.
		sess, err := p.authStore.GetSession(ctx, res.SessionID)
		if err != nil {
			return "", fmt.Errorf("sharedsignals: load targeted session: %w", err)
		}
		if sess.AppID != s.AppID {
			return "", fmt.Errorf(
				"sharedsignals: targeted session %s belongs to a different app than stream %s, refusing to revoke",
				res.SessionID, s.ID)
		}
		if err := p.revoker.RevokeSession(ctx, res.SessionID); err != nil {
			return "", err
		}

	case ActionRevokeAll:
		sessions, err := p.authStore.ListUserSessions(ctx, res.UserID)
		if err != nil {
			return "", err
		}
		var revoked, inScope int
		var failures []error
		for _, sess := range sessions {
			// Stay inside this stream's app. A stream never reaches a
			// session it was not scoped to.
			if sess.AppID != s.AppID {
				continue
			}
			inScope++
			if err := p.revoker.RevokeSession(ctx, sess.ID); err != nil {
				failures = append(failures, fmt.Errorf("session %s: %w", sess.ID, err))
				continue
			}
			revoked++
		}
		if len(failures) > 0 {
			// Revoking as many sessions as possible is the safer direction
			// for a compromise event, so one failed session does not abort
			// the rest. This audit record matters more than usual: the
			// duplicate-jti replay guard means a retransmitted SET answers
			// 202 without ever running this code again, so this is the only
			// trace a partial failure leaves for an operator to find.
			p.audit(ctx, s, ev, res, bridge.SeverityCritical, ActionRevokeAll+"_partial", map[string]string{
				"revoked": fmt.Sprintf("%d", revoked),
				"failed":  fmt.Sprintf("%d", len(failures)),
				"total":   fmt.Sprintf("%d", inScope),
			})
			return ActionRevokeAll, fmt.Errorf(
				"sharedsignals: revoke_all revoked %d/%d in-scope sessions: %w",
				revoked, inScope, errors.Join(failures...))
		}
	}

	p.audit(ctx, s, ev, res, bridge.SeverityCritical, action, nil)
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
		// A hostile or malformed timestamp can still land far outside any
		// plausible range even after the millis/seconds guess above (e.g.
		// 99999999999 is read as seconds and lands around the year 5138).
		// EventAt is a forensic field an investigator will read, so refuse
		// to store nonsense and fall back to now instead.
		if eventAt.After(now.Add(maxEventTimestampFuture)) || eventAt.Before(now.Add(-maxEventTimestampPast)) {
			eventAt = now
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
// string constants; bridge does not export a dedicated type for it. extra is
// merged into the metadata map (e.g. partial-revocation counts) and may be
// nil.
func (p *Plugin) audit(ctx context.Context, s *InboundStream, ev caep.Event,
	res Resolution, severity string, action string, extra map[string]string) {
	if p.chronicle == nil {
		return
	}
	metadata := map[string]string{
		"stream_id":  s.ID.String(),
		"issuer":     s.Issuer,
		"event_type": ev.Type,
		"action":     action,
	}
	for k, v := range extra {
		metadata[k] = v
	}
	_ = p.chronicle.Record(ctx, &bridge.AuditEvent{ //nolint:errcheck // best-effort audit
		Action:   "ssf_event_applied",
		Resource: "session",
		ActorID:  res.UserID.String(),
		Tenant:   s.AppID.String(),
		Outcome:  bridge.OutcomeSuccess,
		Severity: severity,
		Metadata: metadata,
	})
}
