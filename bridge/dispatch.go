package bridge

import (
	"context"
	"errors"
	"time"
)

// ErrDispatchNotAvailable is returned when the dispatch bridge is not configured.
var ErrDispatchNotAvailable = errors.New("bridge: dispatch not available (standalone mode)")

// Dispatcher is a local job/task queue interface. Implementations enqueue
// background jobs and schedule future work via the dispatch extension.
type Dispatcher interface {
	// Enqueue submits a background job for immediate processing.
	Enqueue(ctx context.Context, jobName string, payload []byte) error

	// Schedule submits a background job for processing at a future time.
	Schedule(ctx context.Context, jobName string, payload []byte, runAt time.Time) error
}
