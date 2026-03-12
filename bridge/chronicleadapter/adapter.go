// Package chronicleadapter bridges AuthSome audit events to the Chronicle extension.
package chronicleadapter

import (
	"context"

	"github.com/xraph/chronicle"

	"github.com/xraph/authsome/bridge"
)

// Adapter translates AuthSome audit events to Chronicle events.
type Adapter struct {
	emitter chronicle.Emitter
}

// New creates a Chronicle bridge adapter.
func New(emitter chronicle.Emitter) *Adapter {
	return &Adapter{emitter: emitter}
}

// Record implements bridge.Chronicle.
func (a *Adapter) Record(ctx context.Context, event *bridge.AuditEvent) error {
	var builder *chronicle.EventBuilder
	switch event.Severity {
	case bridge.SeverityCritical:
		builder = a.emitter.Critical(ctx, event.Action, event.Resource, event.ResourceID)
	case bridge.SeverityWarning:
		builder = a.emitter.Warning(ctx, event.Action, event.Resource, event.ResourceID)
	default:
		builder = a.emitter.Info(ctx, event.Action, event.Resource, event.ResourceID)
	}

	if event.Category != "" {
		builder = builder.Category(event.Category)
	}
	if event.ActorID != "" {
		builder = builder.UserID(event.ActorID)
	}
	if event.Tenant != "" {
		builder = builder.AppID(event.Tenant)
	}
	if event.Outcome != "" {
		builder = builder.Outcome(event.Outcome)
	}
	if event.Reason != "" {
		builder = builder.Reason(event.Reason)
	}
	for k, v := range event.Metadata {
		builder = builder.Meta(k, v)
	}

	return builder.Record()
}

// Compile-time check.
var _ bridge.Chronicle = (*Adapter)(nil)
