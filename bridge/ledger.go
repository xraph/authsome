package bridge

import (
	"context"
	"errors"
)

// ErrLedgerNotAvailable is returned when the ledger bridge is not configured.
var ErrLedgerNotAvailable = errors.New("bridge: ledger not available (standalone mode)")

// Ledger is a local billing/metering interface. Implementations record usage
// events and check feature entitlements via the ledger extension.
type Ledger interface {
	// RecordUsage records a usage event for a metered feature.
	RecordUsage(ctx context.Context, featureKey string, quantity int64) error

	// CheckEntitlement checks whether a tenant is entitled to use a feature.
	// Returns allowed=true if the feature is accessible, false otherwise.
	CheckEntitlement(ctx context.Context, featureKey string) (allowed bool, err error)
}
