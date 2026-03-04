// Package ledgeradapter adapts the ledger extension to the authsome
// bridge.Ledger interface.
package ledgeradapter

import (
	"context"

	"github.com/xraph/authsome/bridge"

	"github.com/xraph/ledger"
)

// Adapter implements bridge.Ledger by delegating to the ledger extension.
type Adapter struct {
	engine *ledger.Ledger
}

// Compile-time check.
var _ bridge.Ledger = (*Adapter)(nil)

// New creates a new ledger adapter.
func New(engine *ledger.Ledger) *Adapter {
	return &Adapter{engine: engine}
}

// RecordUsage records a usage event for a metered feature.
func (a *Adapter) RecordUsage(ctx context.Context, featureKey string, quantity int64) error {
	return a.engine.Meter(ctx, featureKey, quantity)
}

// CheckEntitlement checks whether a tenant is entitled to use a feature.
func (a *Adapter) CheckEntitlement(ctx context.Context, featureKey string) (bool, error) {
	result, err := a.engine.Entitled(ctx, featureKey)
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}
