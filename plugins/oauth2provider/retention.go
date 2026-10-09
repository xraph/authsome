package oauth2provider

import (
	"context"
	"errors"
	"time"

	"github.com/xraph/authsome/store"
)

// SweepRetention implements plugin.RetentionSweeper: authorization codes
// and device codes that expired before their cutoff are removed in batches.
func (p *Plugin) SweepRetention(ctx context.Context, cutoff func(kind string) time.Time, batch int) (int64, error) {
	if p.oauth2Store == nil {
		return 0, nil
	}
	var total int64
	var errs []error
	if before := cutoff(store.RetentionAuthCodes); !before.IsZero() {
		n, err := store.DrainExpired(ctx, batch, func(ctx context.Context, batch int) (int64, error) {
			return p.oauth2Store.DeleteExpiredAuthCodes(ctx, before, batch)
		})
		total += n
		errs = append(errs, err)
	}
	if before := cutoff(store.RetentionDeviceCodes); !before.IsZero() {
		n, err := store.DrainExpired(ctx, batch, func(ctx context.Context, batch int) (int64, error) {
			return p.oauth2Store.DeleteExpiredDeviceCodesBefore(ctx, before, batch)
		})
		total += n
		errs = append(errs, err)
	}
	return total, errors.Join(errs...)
}
