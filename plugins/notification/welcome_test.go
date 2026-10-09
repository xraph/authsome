package notification

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/bridge"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/user"
)

type recordingHerald struct {
	async       bool
	hadDeadline bool
}

func (*recordingHerald) Send(context.Context, *bridge.HeraldSendRequest) error { return nil }
func (h *recordingHerald) Notify(ctx context.Context, req *bridge.HeraldNotifyRequest) error {
	h.async = req.Async
	_, h.hadDeadline = ctx.Deadline()
	return nil
}

type fakeDispatcher struct{}

func (fakeDispatcher) Enqueue(context.Context, string, []byte) error             { return nil }
func (fakeDispatcher) Schedule(context.Context, string, []byte, time.Time) error { return nil }

// The welcome notification goes through the queue when a dispatcher is
// present, and inline under a deadline when there is none.
func TestWelcome_QueuedWithDispatcherInlineWithDeadlineWithout(t *testing.T) {
	h := &recordingHerald{}
	p := New()
	p.herald = h
	p.logger = log.NewNoopLogger()
	u := &user.User{ID: id.NewUserID(), Email: "new@example.test"}

	require.NoError(t, p.OnAfterSignUp(context.Background(), u, nil))
	assert.False(t, h.async, "no dispatcher: sent inline")
	assert.True(t, h.hadDeadline, "an inline send runs under a deadline")

	p.dispatcher = fakeDispatcher{}
	require.NoError(t, p.OnAfterSignUp(context.Background(), u, nil))
	assert.True(t, h.async, "a dispatcher means the queue")

	assert.True(t, sendAsync(true, nil), "the operator's own async setting still applies")
}
