package authsome

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/xraph/go-utils/log"
	"go.opentelemetry.io/otel/trace"

	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/plugin"
	"github.com/xraph/authsome/store"
)

// spanSweepRetention is the span a sweep that holds the lease runs in.
const spanSweepRetention = "sweep retention"

// retentionLeaseKey is the authsome_kv key one replica takes before a sweep
// so that a fleet of replicas runs one sweep per interval, not one each.
const retentionLeaseKey = "authsome:retention:lease"

// retentionInterval is how often the sweeper runs: Session.CleanupInterval,
// an hour when unset, and off when negative.
func (e *Engine) retentionInterval() time.Duration {
	switch iv := e.config.Session.CleanupInterval; {
	case iv < 0:
		return 0
	case iv == 0:
		return time.Hour
	default:
		return iv
	}
}

// retentionTTLs is the retention window for each core kind, for a store
// whose database can expire rows on its own.
func (e *Engine) retentionTTLs() map[string]time.Duration {
	ttl := make(map[string]time.Duration, 4)
	for _, kind := range []string{
		store.RetentionSessions, store.RetentionVerifications,
		store.RetentionPasswordResets, store.RetentionRevokedRefreshTokens,
	} {
		ttl[kind] = e.config.Retention.Days(kind)
	}
	return ttl
}

// startRetentionSweeper runs SweepRetention every interval until Stop. The
// first run waits a full interval so boot is not slowed by a sweep, and the
// database's own expiry indexes, where it has them, are brought in line
// with the configured windows first.
//
// ctx must carry no span (Start passes it through telemetry.Detach): each
// sweep that wins the lease is its own trace, and the lease check that
// loses is no trace at all.
func (e *Engine) startRetentionSweeper(ctx context.Context) {
	interval := e.retentionInterval()
	if interval <= 0 {
		return
	}
	if idx, ok := e.store.(store.TTLIndexer); ok {
		if err := idx.EnsureTTLIndexes(ctx, e.retentionTTLs()); err != nil {
			e.logger.Warn("authsome: could not set database expiry indexes; the sweeper will delete instead",
				log.String("error", err.Error()))
		}
	}
	stop, done := make(chan struct{}), make(chan struct{})
	e.retentionStop, e.retentionDone = stop, done
	e.spawn(func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				if _, err := e.SweepRetention(ctx); err != nil {
					e.logger.Warn("authsome: retention sweep failed", log.String("error", err.Error()))
				}
			}
		}
	})
}

// stopRetentionSweeper ends the loop and waits for a sweep in progress to
// return, so Stop leaves no goroutine writing to a store that is closing.
func (e *Engine) stopRetentionSweeper() {
	if e.retentionStop == nil {
		return
	}
	close(e.retentionStop)
	<-e.retentionDone
	e.retentionStop, e.retentionDone = nil, nil
}

// SweepRetention removes rows that have outlived their retention window:
// expired sessions, verifications, password resets and revocation records
// from the store, expired ceremony state, and whatever each plugin that
// implements plugin.RetentionSweeper keeps. It returns how many rows went.
//
// One replica sweeps at a time: the sweep first takes a lease in the store
// for half the interval and returns zero, without error, when another
// replica holds it. Each kind is removed in batches of Retention.BatchSize
// until a batch comes back short. A failure in one kind does not stop the
// others; the errors are joined and returned together.
//
// A sweep that holds the lease runs in a sweep retention span: a root span
// when the background sweeper calls this, a child when you call it under a
// span of your own. Losing the lease starts no span.
func (e *Engine) SweepRetention(ctx context.Context) (int64, error) {
	lease := e.retentionInterval() / 2
	if lease <= 0 {
		lease = 30 * time.Minute
	}
	held, err := e.store.KVSetNX(ctx, retentionLeaseKey, []byte(time.Now().UTC().Format(time.RFC3339)), lease)
	if err != nil {
		e.tel.RetentionSweep(telemetry.SweepError)
		return 0, fmt.Errorf("authsome: retention lease: %w", err)
	}
	if !held {
		e.tel.RetentionSweep(telemetry.SweepSkipped)
		e.logger.Debug("authsome: retention sweep skipped; another replica holds the lease")
		return 0, nil
	}

	ctx, span := telemetry.Start(ctx, spanSweepRetention)
	defer span.End()
	// failed notes one kind that did not finish, by kind and error type
	// only: a store's message can quote the rows it was working on.
	failed := func(kind string, err error) {
		span.AddEvent("sweep failed", trace.WithAttributes(
			telemetry.AttrRetentionKind.String(kind),
			telemetry.AttrExceptionType.String(telemetry.TypeOf(err)),
		))
	}

	batch := e.config.Retention.BatchSize
	if batch <= 0 {
		batch = 1000
	}
	now := time.Now()
	cutoff := func(kind string) time.Time { return e.config.Retention.Cutoff(kind, now) }

	var total int64
	var errs []error
	sweeps := []struct {
		kind  string
		label string
		del   func(ctx context.Context, before time.Time, batch int) (int64, error)
	}{
		{store.RetentionSessions, telemetry.RetentionSessions, e.store.DeleteExpiredSessions},
		{store.RetentionVerifications, telemetry.RetentionVerifications, e.store.DeleteExpiredVerifications},
		{store.RetentionPasswordResets, telemetry.RetentionPasswordResets, e.store.DeleteExpiredPasswordResets},
		{store.RetentionRevokedRefreshTokens, telemetry.RetentionRevokedRefreshTokens, e.store.DeleteExpiredRevokedRefreshTokens},
	}
	for _, sw := range sweeps {
		before := cutoff(sw.kind)
		if before.IsZero() {
			continue
		}
		n, sweepErr := store.DrainExpired(ctx, batch, func(ctx context.Context, batch int) (int64, error) {
			return sw.del(ctx, before, batch)
		})
		total += n
		e.tel.RetentionRowsDeleted(sw.label, n)
		if sweepErr != nil {
			failed(sw.label, sweepErr)
			errs = append(errs, fmt.Errorf("authsome: sweep %s: %w", sw.kind, sweepErr))
		}
	}

	n, err := e.store.KVDeleteExpired(ctx, now)
	total += n
	e.tel.RetentionRowsDeleted(telemetry.RetentionCeremonyState, n)
	if err != nil {
		failed(telemetry.RetentionCeremonyState, err)
		errs = append(errs, fmt.Errorf("authsome: sweep ceremony state: %w", err))
	}

	// Plugin sweepers share one label: a custom plugin's name is not a
	// bounded value.
	for _, p := range e.plugins.Plugins() {
		sw, ok := p.(plugin.RetentionSweeper)
		if !ok {
			continue
		}
		n, sweepErr := sw.SweepRetention(ctx, cutoff, batch)
		total += n
		e.tel.RetentionRowsDeleted(telemetry.RetentionPlugins, n)
		if sweepErr != nil {
			failed(telemetry.RetentionPlugins, sweepErr)
			errs = append(errs, fmt.Errorf("authsome: sweep plugin %s: %w", p.Name(), sweepErr))
		}
	}

	span.SetAttributes(telemetry.AttrRetentionRemoved.Int64(total))
	result := telemetry.SweepOK
	if len(errs) > 0 {
		result = telemetry.SweepError
		telemetry.Fail(span, "sweep_incomplete", nil)
	}
	e.tel.RetentionSweep(result)

	if total > 0 {
		e.logger.Info("authsome: retention sweep removed rows", log.Int64("removed", total))
	}
	return total, errors.Join(errs...)
}
