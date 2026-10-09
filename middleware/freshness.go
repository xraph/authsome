package middleware

import (
	"context"
	"time"

	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/session"
)

// DefaultStepUpWindow is how recently a caller must have signed in for a
// sensitive change (enrolling a second factor, registering a passkey) to
// proceed on session auth alone.
const DefaultStepUpWindow = 5 * time.Minute

// SessionLookup loads a session by id when the request carries only the id
// (a JWT, for instance) and not the row itself.
type SessionLookup func(ctx context.Context, sessionID id.SessionID) (*session.Session, error)

// SessionIsFresh reports whether the caller signed in within window: the
// session on the context, or failing that the one lookup returns for the
// context's session id, was created no longer than window ago. A missing
// session is never fresh, so a caller that cannot show when they signed in
// is asked to sign in again.
func SessionIsFresh(ctx context.Context, now time.Time, window time.Duration, lookup SessionLookup) bool {
	if window <= 0 {
		window = DefaultStepUpWindow
	}
	sess, ok := SessionFrom(ctx)
	if (!ok || sess == nil) && lookup != nil {
		if sid, hasID := SessionIDFrom(ctx); hasID && !sid.IsNil() {
			if loaded, err := lookup(ctx, sid); err == nil {
				sess = loaded
			}
		}
	}
	if sess == nil || sess.CreatedAt.IsZero() {
		return false
	}
	return !now.After(sess.CreatedAt.Add(window))
}
