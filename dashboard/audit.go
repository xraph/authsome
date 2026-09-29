package dashboard

import (
	"context"

	"github.com/xraph/authsome/hook"
)

// Auditor records destructive dashboard actions on the audit trail through
// the engine's hook bus, so the record carries the same request correlation
// and reaches the same sink as every engine event. nil-safe: a nil Auditor
// or nil bus silently does nothing, because errors during dashboard
// rendering must never block the user.
type Auditor struct {
	bus *hook.Bus
}

// NewAuditor returns an Auditor backed by bus. bus may be nil; the resulting
// Auditor is inert.
func NewAuditor(bus *hook.Bus) *Auditor {
	return &Auditor{bus: bus}
}

// unknownActor is the ActorID written when the calling context has no
// authenticated user. The dashboard middleware should always populate a
// user; reaching this fallback indicates a misconfigured route or a bug,
// and we want the audit log to make that visible rather than silently
// recording an empty TypeID.
const unknownActor = "unknown"

// Record writes a successful action. Call it after the engine call returned
// nil; use RecordWithOutcome when the call failed, so the trail never shows
// a deletion or ban that did not happen.
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
	a.RecordWithOutcome(ctx, action, severity, hook.OutcomeSuccess, actorID, resourceID, metadata)
}

// RecordWithOutcome is the explicit-outcome variant of Record. Pass the
// engine error's text as metadata["error"] on failure so the reason is kept.
func (a *Auditor) RecordWithOutcome(
	ctx context.Context,
	action string,
	severity string,
	outcome string,
	actorID, resourceID string,
	metadata map[string]string,
) {
	if a == nil || a.bus == nil {
		return
	}
	if actorID == "" {
		actorID = unknownActor
	}
	ev := &hook.Event{
		Action:     action,
		Severity:   severity,
		Outcome:    outcome,
		ActorID:    actorID,
		ResourceID: resourceID,
		Category:   "dashboard",
		Metadata:   metadata,
	}
	if metadata != nil {
		ev.Tenant = metadata["app_id"]
		ev.Reason = metadata["error"]
	}
	a.bus.Emit(ctx, ev)
}

// outcomeOf maps an engine result to an audit outcome.
func outcomeOf(err error) string {
	if err != nil {
		return hook.OutcomeFailure
	}
	return hook.OutcomeSuccess
}

// withError returns metadata carrying the error text when err is set.
func withError(metadata map[string]string, err error) map[string]string {
	if err == nil {
		return metadata
	}
	out := make(map[string]string, len(metadata)+1)
	for k, v := range metadata {
		out[k] = v
	}
	out["error"] = err.Error()
	return out
}
