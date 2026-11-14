// Package handoff hands a session minted at an identity provider callback to
// the frontend that started the sign-in, by a single-use one-time code.
//
// The callback is a top-level browser navigation landing on the API origin,
// so it cannot give tokens to an app that keeps them on its own origin. It
// stashes a reference to the session under a random code instead and
// redirects with `code=`. The app redeems the code once, with its
// publishable key, and gets a rotated token pair: the code names the session
// and carries no credential, so a copy of the ceremony store yields nothing a
// browser could present. The SSO and social plugins both use it.
package handoff

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/session"
)

// TTL is how long a code stays redeemable. The browser redeems it on the
// page the callback redirects to, so a minute is plenty.
const TTL = 60 * time.Second

// Payload is what a code stands for. It names the session and carries no
// token.
type Payload struct {
	User      json.RawMessage `json:"user,omitempty"`
	SessionID string          `json:"session_id"`
	Provider  string          `json:"provider,omitempty"`
	IsNewUser bool            `json:"is_new_user,omitempty"`
	AppID     string          `json:"app_id"`
}

// Mint stashes p under a fresh code in the namespace prefix (e.g.
// "sso:otc:") and returns the code.
func Mint(ctx context.Context, store ceremony.Store, prefix string, p *Payload) (string, error) {
	if store == nil {
		return "", errors.New("handoff: no ceremony store")
	}
	if p == nil || p.SessionID == "" {
		return "", errors.New("handoff: a one-time code needs a session to hand over")
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := hex.EncodeToString(b)
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	if err := store.Set(ctx, prefix+code, raw, TTL); err != nil {
		return "", err
	}
	return code, nil
}

// Redeem spends code once and rotates the session it names. The caller's
// publishable-key app, when there is one, must be the app the code was
// minted for. Every failure is a 400 that says no more than that the code is
// no good, so a guess learns nothing.
//
// Single use holds across replicas: a claim marker set with SetNX lets
// exactly one redeemer through even if two read the payload at once.
func Redeem(ctx forge.Context, eng *authsome.Engine, store ceremony.Store, prefix, code string) (*Payload, *session.Session, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, nil, forge.BadRequest("code is required")
	}
	if eng == nil || store == nil {
		return nil, nil, middleware.InternalError(ctx, errors.New("handoff: exchange needs the authsome engine"))
	}
	goCtx := ctx.Context()
	raw, err := store.Get(goCtx, prefix+code)
	if err != nil {
		return nil, nil, forge.BadRequest("invalid or expired code")
	}
	claimed, err := store.SetNX(goCtx, prefix+"used:"+code, []byte{1}, TTL)
	if err != nil {
		return nil, nil, middleware.InternalError(ctx, err)
	}
	_ = store.Delete(goCtx, prefix+code) //nolint:errcheck // single use is the claim marker's job
	if !claimed {
		return nil, nil, forge.BadRequest("invalid or expired code")
	}
	var pl Payload
	if json.Unmarshal(raw, &pl) != nil {
		return nil, nil, forge.BadRequest("invalid code")
	}
	if appID, ok := middleware.AppIDFrom(goCtx); ok && appID.String() != pl.AppID {
		return nil, nil, forge.BadRequest("code is not valid for this app")
	}
	sessionID, err := id.ParseSessionID(pl.SessionID)
	if err != nil {
		return nil, nil, forge.BadRequest("invalid code")
	}
	r := ctx.Request()
	sess, err := eng.RefreshBySessionID(goCtx, sessionID, authsome.RefreshOpts{
		IPAddress: middleware.ClientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		return nil, nil, forge.BadRequest("invalid or expired code")
	}
	return &pl, sess, nil
}
