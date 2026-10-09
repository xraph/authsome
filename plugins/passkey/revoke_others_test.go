package passkey

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
)

type recordingRevoker struct {
	calls int
	user  id.UserID
	keep  id.SessionID
}

func (r *recordingRevoker) RevokeOtherUserSessions(_ context.Context, userID id.UserID, keep id.SessionID) error {
	r.calls++
	r.user = userID
	r.keep = keep
	return nil
}

func TestRevokeOtherSessions_KeepsTheRequestSession(t *testing.T) {
	r := &recordingRevoker{}
	p := &Plugin{revoker: r}
	uid, sid := id.NewUserID(), id.NewSessionID()

	p.revokeOtherSessions(middleware.WithSessionID(context.Background(), sid), uid)

	assert.Equal(t, 1, r.calls)
	assert.Equal(t, uid, r.user)
	assert.Equal(t, sid, r.keep)
}

func TestRevokeOtherSessions_NoEngineIsNoop(_ *testing.T) {
	p := &Plugin{}
	p.revokeOtherSessions(context.Background(), id.NewUserID())
}
