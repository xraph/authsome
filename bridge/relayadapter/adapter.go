// Package relayadapter bridges AuthSome webhook events to the Relay extension.
package relayadapter

import (
	"context"
	"errors"
	"fmt"

	"github.com/xraph/relay"
	"github.com/xraph/relay/catalog"
	"github.com/xraph/relay/endpoint"
	"github.com/xraph/relay/event"
	"github.com/xraph/relay/id"

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

// ──────────────────────────────────────────────────
// Endpoint management
// ──────────────────────────────────────────────────

func toInput(in bridge.EndpointInput) endpoint.Input {
	return endpoint.Input{
		TenantID:    in.TenantID,
		URL:         in.URL,
		Description: in.Description,
		Secret:      in.Secret,
		EventTypes:  in.EventTypes,
		Metadata:    in.Metadata,
	}
}

// CreateEndpoint implements bridge.EndpointRelay.
func (a *Adapter) CreateEndpoint(ctx context.Context, in bridge.EndpointInput) (string, error) {
	ep, err := a.r.Endpoints().Create(ctx, toInput(in))
	if err != nil {
		return "", fmt.Errorf("relay: create endpoint: %w", err)
	}
	return ep.ID.String(), nil
}

// UpdateEndpoint implements bridge.EndpointRelay.
func (a *Adapter) UpdateEndpoint(ctx context.Context, endpointID string, in bridge.EndpointInput) error {
	epID, err := id.Parse(endpointID)
	if err != nil {
		return fmt.Errorf("relay: endpoint id: %w", err)
	}
	if _, err := a.r.Endpoints().Update(ctx, epID, toInput(in)); err != nil {
		return fmt.Errorf("relay: update endpoint: %w", err)
	}
	return nil
}

// SetEndpointEnabled implements bridge.EndpointRelay.
func (a *Adapter) SetEndpointEnabled(ctx context.Context, endpointID string, enabled bool) error {
	epID, err := id.Parse(endpointID)
	if err != nil {
		return fmt.Errorf("relay: endpoint id: %w", err)
	}
	if err := a.r.Endpoints().SetEnabled(ctx, epID, enabled); err != nil {
		return fmt.Errorf("relay: set endpoint enabled: %w", err)
	}
	return nil
}

// DeleteEndpoint implements bridge.EndpointRelay.
func (a *Adapter) DeleteEndpoint(ctx context.Context, endpointID string) error {
	epID, err := id.Parse(endpointID)
	if err != nil {
		return fmt.Errorf("relay: endpoint id: %w", err)
	}
	if err := a.r.Endpoints().Delete(ctx, epID); err != nil {
		if errors.Is(err, relay.ErrEndpointNotFound) {
			return nil
		}
		return fmt.Errorf("relay: delete endpoint: %w", err)
	}
	return nil
}

// RotateEndpointSecret implements bridge.EndpointRelay.
func (a *Adapter) RotateEndpointSecret(ctx context.Context, endpointID string) (string, error) {
	epID, err := id.Parse(endpointID)
	if err != nil {
		return "", fmt.Errorf("relay: endpoint id: %w", err)
	}
	secret, err := a.r.Endpoints().RotateSecret(ctx, epID)
	if err != nil {
		return "", fmt.Errorf("relay: rotate endpoint secret: %w", err)
	}
	return secret, nil
}

var _ bridge.EndpointRelay = (*Adapter)(nil)
