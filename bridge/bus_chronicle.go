package bridge

import (
	"context"

	"github.com/xraph/authsome/hook"
)

// BusChronicle is a Chronicle that records through the engine's hook bus
// instead of writing to the audit sink directly. Plugins and middleware hold
// a Chronicle for their audit helpers; giving them this one means every
// record they make is enriched with request correlation and observed by
// metrics and notifications exactly like an engine event, and reaches the
// real sink through the engine's single subscriber.
type BusChronicle struct {
	bus *hook.Bus
}

// NewBusChronicle returns a Chronicle backed by bus. A nil bus yields an
// inert Chronicle so callers keep their nil-safe shape.
func NewBusChronicle(bus *hook.Bus) *BusChronicle {
	return &BusChronicle{bus: bus}
}

// Record implements Chronicle. The bus logs sink failures; Record itself
// never fails, matching the best-effort contract every plugin helper has.
func (c *BusChronicle) Record(ctx context.Context, event *AuditEvent) error {
	if c == nil || c.bus == nil || event == nil {
		return nil
	}
	c.bus.Emit(ctx, &hook.Event{
		Action:     event.Action,
		Resource:   event.Resource,
		ResourceID: event.ResourceID,
		ActorID:    event.ActorID,
		Tenant:     event.Tenant,
		OrgID:      event.OrgID,
		Outcome:    event.Outcome,
		Severity:   event.Severity,
		Category:   event.Category,
		Reason:     event.Reason,
		Metadata:   event.Metadata,
		IP:         event.IP,
		UserAgent:  event.UserAgent,
		RequestID:  event.RequestID,
		SessionID:  event.SessionID,
	})
	return nil
}

var _ Chronicle = (*BusChronicle)(nil)
