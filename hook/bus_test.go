package hook_test

import (
	"context"
	"errors"
	log "github.com/xraph/go-utils/log"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/hook"
)

func newBus() *hook.Bus { return hook.NewBus(log.NewNoopLogger()) }

func TestBus_OnAndEmit(t *testing.T) {
	bus := newBus()
	var called int32

	bus.On("test", func(_ context.Context, e *hook.Event) error {
		atomic.AddInt32(&called, 1)
		return nil
	})

	bus.Emit(context.Background(), &hook.Event{
		Action:   hook.ActionSignUp,
		Resource: hook.ResourceUser,
	})

	assert.Equal(t, int32(1), atomic.LoadInt32(&called))
}

func TestBus_MultipleHandlers(t *testing.T) {
	bus := newBus()
	var count int32

	bus.On("a", func(_ context.Context, _ *hook.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})
	bus.On("b", func(_ context.Context, _ *hook.Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	bus.Emit(context.Background(), &hook.Event{Action: hook.ActionSignIn})
	assert.Equal(t, int32(2), atomic.LoadInt32(&count))
}

func TestBus_ErrorIsolation(t *testing.T) {
	bus := newBus()
	var secondCalled bool

	bus.On("failing", func(_ context.Context, _ *hook.Event) error {
		return errors.New("boom")
	})
	bus.On("succeeding", func(_ context.Context, _ *hook.Event) error {
		secondCalled = true
		return nil
	})

	bus.Emit(context.Background(), &hook.Event{Action: hook.ActionSignOut})
	assert.True(t, secondCalled, "second handler should still be called after first errors")
}

func TestBus_EmitSetsTimestamp(t *testing.T) {
	bus := newBus()
	var captured *hook.Event

	bus.On("cap", func(_ context.Context, e *hook.Event) error {
		captured = e
		return nil
	})

	bus.Emit(context.Background(), &hook.Event{Action: "test"})
	assert.False(t, captured.Timestamp.IsZero(), "timestamp should be auto-set")
}

func TestBus_EmptyHandlers(t *testing.T) {
	bus := newBus()
	// Should not panic with no handlers.
	bus.Emit(context.Background(), &hook.Event{Action: "noop"})
}
