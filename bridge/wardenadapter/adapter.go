// Package wardenadapter bridges AuthSome authorization to the Warden extension.
package wardenadapter

import (
	"context"

	"github.com/xraph/warden"

	"github.com/xraph/authsome/bridge"
)

// Adapter translates AuthSome authorization requests to Warden checks.
type Adapter struct {
	engine *warden.Engine
}

// New creates a Warden bridge adapter.
func New(engine *warden.Engine) *Adapter {
	return &Adapter{engine: engine}
}

// Check implements bridge.Authorizer.
func (a *Adapter) Check(ctx context.Context, req *bridge.AuthzRequest) (*bridge.AuthzResult, error) {
	result, err := a.engine.Check(ctx, &warden.CheckRequest{
		Subject: warden.Subject{
			Kind: warden.SubjectUser,
			ID:   req.Subject,
		},
		Action: warden.Action{
			Name: req.Action,
		},
		Resource: warden.Resource{
			Type: req.Resource,
		},
	})
	if err != nil {
		return nil, err
	}

	return &bridge.AuthzResult{
		Allowed: result.Allowed,
		Reason:  result.Reason,
	}, nil
}

// Compile-time check.
var _ bridge.Authorizer = (*Adapter)(nil)
