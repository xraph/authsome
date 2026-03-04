// Package relayadapter bridges AuthSome webhook events to the Relay extension.
package relayadapter

import (
	"context"

	"github.com/xraph/relay"
	"github.com/xraph/relay/catalog"
	"github.com/xraph/relay/event"

	"github.com/xraph/authsome/bridge"
)

// Adapter translates AuthSome webhook events to Relay events.
type Adapter struct {
	r *relay.Relay
}

// New creates a Relay bridge adapter.
func New(r *relay.Relay) *Adapter {
	return &Adapter{r: r}
}

// Send implements bridge.EventRelay.
func (a *Adapter) Send(ctx context.Context, evt *bridge.WebhookEvent) error {
	return a.r.Send(ctx, &event.Event{
		Type:           evt.Type,
		TenantID:       evt.TenantID,
		Data:           evt.Data,
		IdempotencyKey: evt.IdempotencyKey,
	})
}

// RegisterEventTypes implements bridge.EventRelay.
func (a *Adapter) RegisterEventTypes(ctx context.Context, defs []bridge.WebhookDefinition) error {
	for _, def := range defs {
		_, err := a.r.RegisterEventType(ctx, catalog.WebhookDefinition{
			Name:        def.Name,
			Description: def.Description,
			Group:       def.Group,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Compile-time check.
var _ bridge.EventRelay = (*Adapter)(nil)
