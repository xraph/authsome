package dashboard

import (
	"context"

	"github.com/xraph/authsome/bridge"
)

// Auditor wraps a bridge.Chronicle so destructive dashboard actions can
// record a single line per call. nil-safe: a nil Auditor or nil chronicle
// silently does nothing (errors during dashboard rendering must never block
// the user; failures are best-effort by design).
type Auditor struct {
	chronicle bridge.Chronicle
}

// NewAuditor returns an Auditor backed by ch. ch may be nil — the resulting
// Auditor is inert.
func NewAuditor(ch bridge.Chronicle) *Auditor {
	return &Auditor{chronicle: ch}
}

// unknownActor is the ActorID written when the calling context has no
// authenticated user. The dashboard middleware should always populate a
// user; reaching this fallback indicates a misconfigured route or a bug,
// and we want the audit log to make that visible rather than silently
// recording an empty TypeID.
const unknownActor = "unknown"

// Record builds a bridge.AuditEvent from the supplied fields and forwards
// it to the chronicle. Best-effort: errors are swallowed.
//
// Outcome defaults to bridge.OutcomeSuccess. Audit calls in this codebase
// fire BEFORE the action runs (so an attempt is recorded even if the
// action itself fails), but the convention across the schema is that
// pre-action records use Success — the absence of a follow-up failure
// event is what indicates rollback. Use RecordWithOutcome to override.
//
// If actorID is empty, the audit log records "unknown" so a misconfigured
// route can't silently produce empty-actor events.
func (a *Auditor) Record(
	ctx context.Context,
	action string,
	severity string,
	actorID, resourceID string,
	metadata map[string]string,
) {
	a.RecordWithOutcome(ctx, action, severity, bridge.OutcomeSuccess, actorID, resourceID, metadata)
}

// RecordWithOutcome is the explicit-outcome variant of Record.
func (a *Auditor) RecordWithOutcome(
	ctx context.Context,
	action string,
	severity string,
	outcome string,
	actorID, resourceID string,
	metadata map[string]string,
) {
	if a == nil || a.chronicle == nil {
		return
	}
	if actorID == "" {
		actorID = unknownActor
	}
	_ = a.chronicle.Record(ctx, &bridge.AuditEvent{
		Action:     action,
		Severity:   severity,
		Outcome:    outcome,
		ActorID:    actorID,
		ResourceID: resourceID,
		Metadata:   metadata,
	})
}
