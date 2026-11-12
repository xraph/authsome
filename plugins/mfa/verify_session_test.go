package mfa

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

type recordingStamper struct {
	calls int
	sid   id.SessionID
	at    time.Time
}

func (r *recordingStamper) MarkSessionMFAVerified(_ context.Context, sid id.SessionID, at time.Time) error {
	r.calls++
	r.sid = sid
	r.at = at
	return nil
}

func TestMarkSessionVerified_StampsTheRequestSession(t *testing.T) {
	r := &recordingStamper{}
	p := &Plugin{stamper: r}
	sid := id.NewSessionID()

	require.NoError(t, p.markSessionVerified(middleware.WithSessionID(context.Background(), sid)))

	assert.Equal(t, 1, r.calls)
	assert.Equal(t, sid, r.sid)
	assert.WithinDuration(t, time.Now(), r.at, time.Minute)
}

func TestMarkSessionVerified_NoSessionOrNoEngineIsNoop(t *testing.T) {
	r := &recordingStamper{}
	require.NoError(t, (&Plugin{stamper: r}).markSessionVerified(context.Background()))
	assert.Zero(t, r.calls)
	require.NoError(t, (&Plugin{}).markSessionVerified(middleware.WithSessionID(context.Background(), id.NewSessionID())))
}
