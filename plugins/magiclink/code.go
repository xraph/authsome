package magiclink

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xraph/forge"

	authsome "github.com/xraph/authsome"
	"github.com/xraph/authsome/account"
	"github.com/xraph/authsome/ceremony"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/internal/telemetry"
	"github.com/xraph/authsome/middleware"
	"github.com/xraph/authsome/store"
)

// codeLength is the number of digits in an emailed sign-in code.
const codeLength = 6

// MaxCodeAttempts caps the tries against one emailed code. Every try
// counts, and the code is discarded once they are spent: a six-digit code
// is a million guesses, and this keeps the odds of one landing at five in a
// million per email sent. The route limiter is the other defence, but it is
// absent when rate limiting is off, so the code bounds its own attempts the
// way an MFA ticket does.
const MaxCodeAttempts = 5

// errInvalidCode is the one answer /verify-code gives for an unknown
// address, a wrong code, a used code and an expired one, so the answer
// never says whether an account exists.
const errInvalidCode = "invalid or expired code"

// codeState is the ceremony entry for the live code of one user. It is tied
// to the magic link it was sent with by the link's verification id, and it
// carries what is needed to mark that link used without its plaintext.
type codeState struct {
	VerificationID string    `json:"verification_id"`
	LinkTokenHash  string    `json:"link_token_hash"`
	AppID          string    `json:"app_id"`
	EnvID          string    `json:"env_id"`
	UserID         string    `json:"user_id"`
	CodeHash       string    `json:"code_hash"`
	ExpiresAt      time.Time `json:"expires_at"`
	CreatedAt      time.Time `json:"created_at"`
}

func codeKey(appID id.AppID, userID id.UserID) string {
	return "magiclink_code:" + appID.String() + ":" + userID.String()
}

func codeAttemptsKey(appID id.AppID, userID id.UserID, verificationID string) string {
	return codeKey(appID, userID) + ":attempts:" + verificationID
}

func claimKey(verificationID id.VerificationID) string {
	return "magiclink_used:" + verificationID.String()
}

// hashCode digests a code together with the verification it belongs to, so
// the stored digest is only good for that one link.
func hashCode(verificationID, code string) string {
	return store.HashToken(verificationID + ":" + code)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ceremonyStore returns where code state lives: the engine's ceremony store
// (shared across replicas) when there is an engine, a local one otherwise.
func (p *Plugin) ceremonyStore() ceremony.Store {
	if eng, ok := p.engine.(*authsome.Engine); ok && eng != nil {
		if cs := eng.CeremonyStore(); cs != nil {
			return cs
		}
	}
	return p.ceremonies
}

// codeEnabled reports whether a send mints a code for the app. A code is
// only minted when the mailer can deliver it.
func (p *Plugin) codeEnabled(ctx context.Context, appID id.AppID) bool {
	if _, ok := p.config.Mailer.(CodeMailer); !ok {
		return false
	}
	return p.resolveBool(ctx, appID, SettingEmailCode, p.config.EmailCode)
}

// issueCode mints a code for the link v and records it as the user's live
// code. A later send replaces it, so only the newest code works.
func (p *Plugin) issueCode(ctx context.Context, v *account.Verification) (string, error) {
	code, err := account.GenerateOTPCode()
	if err != nil {
		return "", err
	}
	st := codeState{
		VerificationID: v.ID.String(),
		LinkTokenHash:  store.HashToken(v.Token),
		AppID:          v.AppID.String(),
		EnvID:          v.EnvID.String(),
		UserID:         v.UserID.String(),
		CodeHash:       hashCode(v.ID.String(), code),
		ExpiresAt:      v.ExpiresAt,
		CreatedAt:      v.CreatedAt,
	}
	raw, err := json.Marshal(st)
	if err != nil {
		return "", err
	}
	ttl := time.Until(v.ExpiresAt)
	if ttl <= 0 {
		return "", errors.New("magiclink: link expired before its code was stored")
	}
	if err := p.ceremonyStore().Set(ctx, codeKey(v.AppID, v.UserID), raw, ttl); err != nil {
		return "", err
	}
	return code, nil
}

// claim marks the link (and so its code) used. Exactly one caller wins,
// across replicas, whichever of the two it presents.
func (p *Plugin) claim(ctx context.Context, verificationID id.VerificationID, expiresAt time.Time) (bool, error) {
	// Held a little past the link's expiry; after that both are refused as
	// expired anyway.
	ttl := time.Until(expiresAt) + time.Minute
	if ttl <= time.Minute {
		ttl = time.Minute
	}
	return p.ceremonyStore().SetNX(ctx, claimKey(verificationID), []byte("1"), ttl)
}

// dropCode removes the user's live code if it belongs to the link that was
// just used. A newer code, sent with a newer link, is left alone.
func (p *Plugin) dropCode(ctx context.Context, appID id.AppID, userID id.UserID, verificationID id.VerificationID) {
	cs := p.ceremonyStore()
	key := codeKey(appID, userID)
	raw, err := cs.Get(ctx, key)
	if err != nil {
		return
	}
	var st codeState
	if json.Unmarshal(raw, &st) != nil || st.VerificationID == verificationID.String() {
		_ = cs.Delete(ctx, key) //nolint:errcheck // best-effort; the claim already refuses the code
	}
}

func (p *Plugin) handleVerifyCode(ctx forge.Context, req *VerifyCodeRequest) (_ *VerifyResponse, err error) {
	attempt := telemetry.From(p.engine).StartSignIn(telemetry.MethodMagicLink)
	defer func() { attempt.Done(err) }()

	email := normalizeEmail(req.Email)
	code := strings.TrimSpace(req.Code)
	if email == "" || code == "" {
		return nil, forge.BadRequest("email and code required")
	}

	appIDStr := req.AppID
	if appIDStr == "" {
		appIDStr = p.appID
	}
	appID, err := id.ParseAppID(appIDStr)
	if err != nil {
		return nil, forge.BadRequest("invalid app_id")
	}

	u, lookupErr := p.lookupUser(ctx.Context(), appID, email)
	if lookupErr != nil {
		return nil, forge.Unauthorized(errInvalidCode)
	}

	cs := p.ceremonyStore()
	key := codeKey(appID, u.ID)
	raw, getErr := cs.Get(ctx.Context(), key)
	if getErr != nil {
		return nil, forge.Unauthorized(errInvalidCode)
	}
	var st codeState
	if json.Unmarshal(raw, &st) != nil || st.UserID != u.ID.String() {
		_ = cs.Delete(ctx.Context(), key) //nolint:errcheck // unusable entry
		return nil, forge.Unauthorized(errInvalidCode)
	}
	remaining := time.Until(st.ExpiresAt)
	if remaining <= 0 {
		_ = cs.Delete(ctx.Context(), key) //nolint:errcheck // expired
		return nil, forge.Unauthorized(errInvalidCode)
	}

	// Count the try before comparing, atomically, so concurrent guesses
	// can't slip past the bound between a read and a write. The counter
	// keeps the code's own deadline.
	tries, incErr := cs.Increment(ctx.Context(), codeAttemptsKey(appID, u.ID, st.VerificationID), remaining)
	if incErr != nil {
		// A try that can't be counted isn't allowed.
		return nil, forge.Unauthorized(errInvalidCode)
	}
	if tries > MaxCodeAttempts {
		_ = cs.Delete(ctx.Context(), key) //nolint:errcheck // spent
		return nil, forge.Unauthorized(errInvalidCode)
	}
	if subtle.ConstantTimeCompare([]byte(st.CodeHash), []byte(hashCode(st.VerificationID, code))) != 1 {
		if tries >= MaxCodeAttempts {
			_ = cs.Delete(ctx.Context(), key) //nolint:errcheck // spent
		}
		return nil, forge.Unauthorized(errInvalidCode)
	}

	verificationID, parseErr := id.ParseVerificationID(st.VerificationID)
	if parseErr != nil {
		_ = cs.Delete(ctx.Context(), key) //nolint:errcheck // unusable entry
		return nil, forge.Unauthorized(errInvalidCode)
	}

	// The code and its link are one credential: claim it, so the link is
	// refused from now on and a link already used refuses this code.
	won, claimErr := p.claim(ctx.Context(), verificationID, st.ExpiresAt)
	if claimErr != nil {
		return nil, middleware.InternalError(ctx, fmt.Errorf("failed to verify code: %w", claimErr))
	}
	_ = cs.Delete(ctx.Context(), key) //nolint:errcheck // used either way
	if !won {
		return nil, forge.Unauthorized(errInvalidCode)
	}

	// Record the link as consumed too, so the verification row tells the
	// truth. The claim is what refuses the link; this is bookkeeping.
	envID, _ := id.ParseEnvironmentID(st.EnvID) //nolint:errcheck // empty env stays empty
	createdAt := st.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_ = p.store.UpdateVerification(ctx.Context(), &account.Verification{ //nolint:errcheck // best-effort; the claim holds
		ID:        verificationID,
		AppID:     appID,
		EnvID:     envID,
		UserID:    u.ID,
		TokenHash: st.LinkTokenHash,
		Type:      VerificationTypeMagicLink,
		ExpiresAt: st.ExpiresAt,
		Consumed:  true,
		CreatedAt: createdAt,
	})

	return p.signIn(ctx, attempt, u, appID)
}
