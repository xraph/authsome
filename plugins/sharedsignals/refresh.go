package sharedsignals

import (
	"context"
	"sync"
	"time"
)

// refresher runs one piece of periodic work on a ticker until it is stopped.
//
// It exists because the JWKS cache is otherwise refreshed only by traffic: a
// key rotation is noticed by whichever inbound token is unlucky enough to
// arrive after MaxKeyAge, and a quiet stream that receives nothing for hours
// drifts towards the hard limit with nobody trying to stop it. A ticker turns
// both into ordinary background work.
//
// The loop is split from the ticker so tests drive it with a channel they
// control rather than by sleeping.
type refresher struct {
	interval time.Duration
	round    func(context.Context)

	ticker   *time.Ticker
	cancel   context.CancelFunc
	stopOnce sync.Once
	// done is closed when the loop has left, so stop can wait for a round
	// already in flight instead of returning while it is still talking to
	// somebody else's endpoint.
	done chan struct{}
}

func newRefresher(interval time.Duration, round func(context.Context)) *refresher {
	return &refresher{interval: interval, round: round, done: make(chan struct{})}
}

// start begins the loop on a real ticker.
func (r *refresher) start() {
	r.ticker = time.NewTicker(r.interval)
	r.startWith(r.ticker.C)
}

// startWith begins the loop on a caller-supplied tick channel.
func (r *refresher) startWith(tick <-chan time.Time) {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	go r.run(ctx, tick)
}

// run is the loop. The context is the refresher's own, not the one OnInit
// was handed: init's context may be cancelled the moment startup finishes,
// which would leave every refresh round failing on a dead context.
func (r *refresher) run(ctx context.Context, tick <-chan time.Time) {
	defer close(r.done)
	for {
		// Checked before the select as well as inside it. After a round
		// returns, a tick that arrived while it was running is also ready,
		// and select would pick between the two at random -- so a stopped
		// refresher could run one more round.
		if ctx.Err() != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-tick:
			r.round(ctx)
		}
	}
}

// stop ends the loop and waits for it. Safe to call on a refresher that was
// never started, and safe to call more than once: OnShutdown may run on a
// plugin whose OnInit never completed, and nothing stops a host calling it
// twice.
func (r *refresher) stop() {
	r.stopOnce.Do(func() {
		if r.cancel == nil {
			// Never started, so nothing is going to close done for us.
			close(r.done)
			return
		}
		r.cancel()
	})
	<-r.done
	if r.ticker != nil {
		r.ticker.Stop()
	}
}
