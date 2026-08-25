package authsome

import (
	"context"

	log "github.com/xraph/go-utils/log"

	"github.com/xraph/authsome/dpop"
	"github.com/xraph/authsome/id"
	"github.com/xraph/authsome/settings"
)

// dpopNonceInfo domain-separates the DPoP nonce key from every other use of
// the engine's HMAC material, the same way the dashboard nonce does.
const dpopNonceInfo = "authsome:dpop:nonce-v1"

// initDPoP builds the validator and, when a secret is derivable, the nonce
// signer. Call it during engine construction after settings and the JWT
// formats are in place.
//
// The validator is always constructed, even when every app has DPoP off. A
// validator that exists but is never consulted costs nothing, and it means no
// call site has to nil-check before enforcing.
func (e *Engine) initDPoP() {
	if e.dpopReplayCache == nil {
		e.dpopReplayCache = dpop.NewMemoryReplayCache(0)
	}

	// Built without a Nonce verifier for now. Leaving the field unset keeps
	// it a true nil interface; assigning a still-nil *dpop.NonceSigner here
	// instead would store a non-nil interface wrapping a nil pointer, and a
	// caller that later checked cfg.Nonce == nil would get the wrong answer
	// and panic on the first Verify call.
	e.dpopValidator = dpop.NewValidator(dpop.Config{
		Replay: e.dpopReplayCache,
	})

	secret := e.dpopNonceSecret()
	if len(secret) == 0 {
		// Leave the signer nil and carry on. Startup cannot know which apps
		// will turn nonces on later, so returning an error here would refuse
		// to start every deployment that has no HMAC JWT key, which is most of
		// them and none of which asked for nonces.
		//
		// The setting is where this becomes a decision, and
		// DPoPNonceRequiredForApp fails closed when it finds nonces switched on
		// with nothing to mint them from. This line is a heads-up, not the
		// control.
		e.logger.Warn("authsome: no DPoP nonce secret available; an app that turns dpop.nonce_required on will refuse its bound requests until an HMAC JWT key or AUTHSOME_DASHBOARD_NONCE_SECRET is configured")
		return
	}

	signer, err := dpop.NewNonceSigner(secret, dpop.DefaultNonceTTL)
	if err != nil {
		e.logger.Warn("authsome: DPoP nonce signer unavailable",
			log.String("error", err.Error()),
		)
		return
	}
	e.dpopNonceSigner = signer

	// Rebuild the validator so it holds the signer. Cheap, and it keeps the
	// ordering obvious rather than relying on a pointer set after the fact.
	e.dpopValidator = dpop.NewValidator(dpop.Config{
		Replay: e.dpopReplayCache,
		Nonce:  signer,
	})
}

// dpopNonceSecret derives the DPoP nonce key from the same material the
// dashboard nonce uses, under a different info string.
func (e *Engine) dpopNonceSecret() []byte {
	base := e.NonceSecret()
	if len(base) == 0 {
		return nil
	}
	return hkdfLike(base, dpopNonceInfo)
}

// DPoPValidator returns the engine's proof validator. Never nil after
// construction.
func (e *Engine) DPoPValidator() *dpop.Validator { return e.dpopValidator }

// DPoPNonceSigner returns the nonce signer, or nil when no secret could be
// derived.
func (e *Engine) DPoPNonceSigner() *dpop.NonceSigner { return e.dpopNonceSigner }

// DPoPModeForApp resolves the effective mode for an app, before any per-client
// value is folded in with dpop.MaxMode.
func (e *Engine) DPoPModeForApp(ctx context.Context, appID id.AppID) dpop.Mode {
	mgr := e.Settings()
	if mgr == nil {
		return dpop.ModeOff
	}
	opts := settings.ResolveOpts{}
	if appID.Prefix() != "" {
		opts.AppID = appID.String()
	}
	raw, _ := settings.Get(ctx, mgr, SettingDPoPMode, opts) //nolint:errcheck // Get falls back to the registered default
	return dpop.ParseMode(raw)
}

// DPoPNonceRequiredForApp reports whether an app demands a nonce.
//
// It used to answer false whenever the signer was nil, on the reasoning that
// demanding a nonce we cannot mint locks every client out of the app. True,
// and it is also what an operator asked for. Answering false turned an
// explicit security control off and told nobody, which is the one outcome a
// security control must never have: one Warn line at startup, no enforcement,
// and a dashboard switch that reads as on.
//
// So the misconfiguration fails closed. Nonces stay required, the requests
// that would have gone unprotected are refused instead, and the reason goes in
// the log at error level with the app named. Locking clients out is loud and
// fixable in one config change. Silently not enforcing is neither.
//
// Nothing here fires unless somebody set dpop.nonce_required, which defaults
// to false.
func (e *Engine) DPoPNonceRequiredForApp(ctx context.Context, appID id.AppID) bool {
	mgr := e.Settings()
	if mgr == nil {
		return false
	}
	opts := settings.ResolveOpts{}
	if appID.Prefix() != "" {
		opts.AppID = appID.String()
	}
	required, _ := settings.Get(ctx, mgr, SettingDPoPNonceRequired, opts) //nolint:errcheck // best effort
	if !required {
		return false
	}

	if e.dpopNonceSigner == nil {
		e.logger.Error("authsome: dpop.nonce_required is on for this app but no nonce secret is derivable, so every DPoP request for it will be refused; configure an HMAC JWT key or set AUTHSOME_DASHBOARD_NONCE_SECRET",
			log.String("app_id", appID.String()),
		)
	}
	return true
}
