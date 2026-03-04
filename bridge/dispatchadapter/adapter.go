// Package dispatchadapter adapts the dispatch extension's engine to the
// authsome bridge.Dispatcher interface.
package dispatchadapter

import (
	"context"
	"time"

	"github.com/xraph/authsome/bridge"

	dispatchengine "github.com/xraph/dispatch/engine"
	"github.com/xraph/dispatch/job"
)

// Adapter implements bridge.Dispatcher by delegating to the dispatch extension's engine.
type Adapter struct {
	eng *dispatchengine.Engine
}

// Compile-time check.
var _ bridge.Dispatcher = (*Adapter)(nil)

// New creates a new dispatch adapter.
func New(eng *dispatchengine.Engine) *Adapter {
	return &Adapter{eng: eng}
}

// Enqueue submits a background job for immediate processing.
func (a *Adapter) Enqueue(ctx context.Context, jobName string, payload []byte) error {
	_, err := a.eng.EnqueueRaw(ctx, jobName, payload)
	return err
}

// Schedule submits a background job for processing at a future time.
func (a *Adapter) Schedule(ctx context.Context, jobName string, payload []byte, runAt time.Time) error {
	_, err := a.eng.EnqueueRaw(ctx, jobName, payload, job.WithRunAt(runAt))
	return err
}
