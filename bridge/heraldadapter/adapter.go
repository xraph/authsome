// Package heraldadapter bridges AuthSome notification requests to the Herald extension.
package heraldadapter

import (
	"context"

	"github.com/xraph/herald"

	"github.com/xraph/authsome/bridge"
)

// Adapter translates AuthSome notification requests to Herald engine calls.
type Adapter struct {
	h *herald.Herald
}

// New creates a Herald bridge adapter.
func New(h *herald.Herald) *Adapter {
	return &Adapter{h: h}
}

// Send implements bridge.Herald.
func (a *Adapter) Send(ctx context.Context, req *bridge.HeraldSendRequest) error {
	_, err := a.h.Send(ctx, &herald.SendRequest{
		AppID:    req.AppID,
		EnvID:    req.EnvID,
		OrgID:    req.OrgID,
		UserID:   req.UserID,
		Channel:  req.Channel,
		Template: req.Template,
		Locale:   req.Locale,
		To:       req.To,
		Data:     req.Data,
		Async:    req.Async,
		Metadata: req.Metadata,
	})
	return err
}

// Notify implements bridge.Herald.
func (a *Adapter) Notify(ctx context.Context, req *bridge.HeraldNotifyRequest) error {
	_, err := a.h.Notify(ctx, &herald.NotifyRequest{
		AppID:    req.AppID,
		EnvID:    req.EnvID,
		OrgID:    req.OrgID,
		UserID:   req.UserID,
		Template: req.Template,
		Locale:   req.Locale,
		To:       req.To,
		Data:     req.Data,
		Channels: req.Channels,
		Async:    req.Async,
		Metadata: req.Metadata,
	})
	return err
}

// Compile-time check.
var _ bridge.Herald = (*Adapter)(nil)
