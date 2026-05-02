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

// Record builds a bridge.AuditEvent from the supplied fields and forwards
// it to the chronicle. Best-effort: errors are swallowed.
func (a *Auditor) Record(
	ctx context.Context,
	action string,
	severity string,
	actorID, resourceID string,
	metadata map[string]string,
) {
	if a == nil || a.chronicle == nil {
		return
	}
	_ = a.chronicle.Record(ctx, &bridge.AuditEvent{
		Action:     action,
		Severity:   severity,
		ActorID:    actorID,
		ResourceID: resourceID,
		Metadata:   metadata,
	})
}
