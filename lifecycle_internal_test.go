package authsome

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Stop's wait covers every goroutine started through spawn, and gives up
// at the caller's deadline rather than hanging on a job that never ends.
func TestWaitBackground(t *testing.T) {
	e := &Engine{}
	release := make(chan struct{})
	e.spawn(func() { <-release })

	short, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	assert.False(t, e.waitBackground(short), "a job still running at the deadline is reported")

	close(release)
	assert.True(t, e.waitBackground(context.Background()), "once the job returns the wait does too")
}
